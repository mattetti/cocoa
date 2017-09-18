package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BookmarkData represents the data structure holding the bookmark information
type BookmarkData struct {
	FileSystemType string
	Path           []string
	// CNIDPath in the case of an alias file is the offset to the path element (minus header size)
	CNIDPath            []uint64
	FileCreationDate    time.Time
	FileProperties      []byte
	TypeData            []byte // from 0xf022
	ContainingFolderIDX uint32
	VolumePath          string
	VolumeIsRoot        bool
	VolumeURL           string // file://' + volPath
	VolumeName          string
	VolumeSize          int64
	VolumeCreationDate  time.Time
	VolumeUUID          string // must be uppercase
	VolumeProperties    []byte
	CreationOptions     uint32 // 512
	WasFileReference    bool   // true
	UserName            string // unknown
	CNID                uint32
	UID                 uint32 // 99
	Filename            string
}

// TargetPath returns the full path to the current target url.
func (b *BookmarkData) TargetPath() string {
	return fmt.Sprintf("%s%s", b.VolumePath, filepath.Join(b.Path...))
}

// Write converts the bookmark data into binary data and writes it to the passed writer.
// Note that the writes are buffered and written all at once.
func (b *BookmarkData) Write(w io.Writer) error {
	// buffer for the body
	buf := &bytes.Buffer{}
	// track the offset within the body so we can build the TOC
	oMap := offsetMap{}

	oMap[KBookmarkCreationOptions] = buf.Len()
	buf.Write(encodedUint32(1024))

	slashPos := buf.Len()

	if !b.VolumeIsRoot {
		// HACK, not working for well
		buf.Write([]byte{'/', 0x0, 0x0, 0x0})
		buf.Write([]byte{0x0, 0xF0, 0x0, 0x0})
		buf.Write([]byte{0x48, 0x0, 0x0, 0x0})
	}

	var usernameOffset int
	var trueOffset int

	// write each path items one by one
	pathOffsets := make([]int, len(b.Path))
	for i, item := range b.Path {
		// track the starting offset of each item (append 4 for the body size value)
		// since we need those to create an array for the TOC
		pathOffsets[i] = 4 + buf.Len()
		// if part of the path matches the username, let's use that offset for the
		// username record.
		if item == b.UserName {
			usernameOffset = buf.Len()
		}
		// get the offset of the last item in the path
		if i == len(b.Path)-1 {
			oMap[KBookmarkFullFileName] = pathOffsets[i] - 4
		}
		buf.Write(encodedStringItem(item))
	}
	padBuf(buf)

	// offset to the start of path offsets
	// the TOC will point to here so we can find how many items are in the array
	// and access each item to rebuild the path.
	// 0x04 0x10
	oMap[KBookmarkPath] = buf.Len()
	// number of items
	binary.Write(buf, binary.LittleEndian, uint32(len(b.Path)*4))
	binary.Write(buf, binary.LittleEndian, uint32(bmk_array|bmk_st_one))
	// offset from after the header for each item
	for _, offset := range pathOffsets {
		binary.Write(buf, binary.LittleEndian, uint32(offset))
	}
	padBuf(buf)

	// each file ids for the path
	cnidOffsets := make([]int, len(b.Path))
	for i, cnid := range b.CNIDPath {
		cnidOffsets[i] = 4 + buf.Len()
		buf.Write(encodedUint64(cnid))
	}

	// 0x05 0x10
	oMap[KBookmarkCNIDPath] = buf.Len()
	binary.Write(buf, binary.LittleEndian, uint32(len(b.CNIDPath)*4))
	binary.Write(buf, binary.LittleEndian, uint32(bmk_array|bmk_st_one))
	for _, offset := range cnidOffsets {
		binary.Write(buf, binary.LittleEndian, uint32(offset))
	}
	padBuf(buf)

	// KBookmarkFileCreationDate 0x04 0x10
	oMap[KBookmarkFileCreationDate] = buf.Len()
	buf.Write(encodedTime(b.FileCreationDate))
	padBuf(buf)

	// file ID 0x30 0x10
	// if b.VolumeIsRoot {
	// 	oMap[KBookmarkFileID] = buf.Len()
	// 	buf.Write(encodedUint32(b.CNID))
	// 	padBuf(buf)
	// }

	// file properties
	// 0x10 0x10
	oMap[KBookmarkFileProperties] = buf.Len()
	buf.Write(encodedBytes(b.FileProperties))
	padBuf(buf)

	// KBookmarkWasFileReference 0x01 0xD0
	// oMap[KBookmarkWasFileReference] = buf.Len()
	// buf.Write(encodedBool(b.WasFileReference))
	// padBuf(buf)
	// if b.WasFileReference {
	// 	trueOffset = oMap[KBookmarkWasFileReference]
	// }

	// 0x54 0x10 unknown but seems to always be 1
	// 0x55 0x10 unknown, point to the same value
	// oMap[KBookmarkUnknown] = buf.Len()
	// oMap[KBookmarkUnknown1] = buf.Len()
	// buf.Write(encodedUint32(uint32(1)))
	// padBuf(buf)

	// KBookmarkContainingFolder 0x01 0xc0
	// TODO: only for root volumes?
	if b.VolumeIsRoot {
		oMap[KBookmarkContainingFolder] = buf.Len()
		buf.Write(encodedUint64(uint64(b.ContainingFolderIDX)))
		padBuf(buf)
	}

	// KBookmarkUID 0x12 0xc0
	if b.VolumeIsRoot {
		oMap[KBookmarkUID] = buf.Len()
		buf.Write(encodedUint32(b.UID))
		padBuf(buf)
	}

	// KBookmarkVolumeURL 0x05 0x20
	oMap[KBookmarkVolumeURL] = buf.Len()
	binary.Write(buf, binary.LittleEndian, uint32(len(b.VolumeURL)))
	// only support absolute path for now
	binary.Write(buf, binary.LittleEndian, uint32(bmk_url|bmk_url_st_absolute))
	buf.Write([]byte(b.VolumeURL))
	padBuf(buf)

	// KBookmarkVolumeName 0x10 0x20
	oMap[KBookmarkVolumeName] = buf.Len()
	buf.Write(encodedStringItem(b.VolumeName))
	padBuf(buf)

	// KBookmarkVolumeSize 0x12 0x20
	oMap[KBookmarkVolumeSize] = buf.Len()
	buf.Write(encodedUint64(uint64(b.VolumeSize)))
	padBuf(buf)

	// KBookmarkVolumeCreationDate 0x13 0x20
	oMap[KBookmarkVolumeCreationDate] = buf.Len()
	buf.Write(encodedTime(b.VolumeCreationDate))
	padBuf(buf)

	// KBookmarkVolumeUUID 0x11 0x20
	oMap[KBookmarkVolumeUUID] = buf.Len()
	buf.Write(encodedStringItem(b.VolumeUUID))
	padBuf(buf)

	// KBookmarkVolumeProperties 0x20 0x20
	oMap[KBookmarkVolumeProperties] = buf.Len()
	buf.Write(encodedBytes(b.VolumeProperties))
	padBuf(buf)

	// KBookmarkVolumePath 0x02 0x20
	oMap[KBookmarkVolumePath] = buf.Len()
	buf.Write(encodedStringItem(b.VolumePath))
	padBuf(buf)

	// KBookmarkFileType 0xf022
	oMap[KBookmarkFileType] = buf.Len()
	b.prepareTypeData()
	buf.Write(encodedBytes(b.TypeData))
	padBuf(buf)

	// 0x56 0x10 bool set to true
	// oMap[KBookmarkUnknown2] = trueOffset
	// if trueOffset < 1 {
	// 	oMap[KBookmarkUnknown2] = buf.Len()
	// 	buf.Write(encodedBool(true))
	// 	padBuf(buf)
	// }

	// KBookmarkTOCPath
	if !b.VolumeIsRoot {
		oMap[KBookmarkTOCPath] = buf.Len()
		// array of something
		// nbrBytesNeeded (nbr of elements * 4 bytes)

		offsets := []uint32{
			uint32(slashPos + 52),     // '/'
			uint32(slashPos + 52 + 4), // 00F00000
			uint32(slashPos + 52 + 8), // ?
		}
		for i := 0; i < len(b.Path)-2; i++ {
			offsets = append(offsets, uint32(slashPos+52+4))
		}

		binary.Write(buf, binary.LittleEndian, uint32(42))
		binary.Write(buf, binary.LittleEndian, uint32(bmk_array|bmk_st_one))
		for _, offs := range offsets {
			binary.Write(buf, binary.LittleEndian, uint32(offs))
		}

		padBuf(buf)
	}

	// KBookmarkVolumeIsRoot 0x30 0x20
	if b.VolumeIsRoot {
		if b.VolumeIsRoot && trueOffset > 0 {
			oMap[KBookmarkVolumeIsRoot] = trueOffset
		} else {
			oMap[KBookmarkVolumeIsRoot] = buf.Len()
			buf.Write(encodedBool(b.VolumeIsRoot))
			padBuf(buf)
		}
	}

	// KBookmarkUserName 0x11 0xc0
	if b.VolumeIsRoot {
		if usernameOffset > 0 {
			oMap[KBookmarkUserName] = usernameOffset
		} else {
			oMap[KBookmarkUserName] = buf.Len()
			buf.Write(encodedStringItem(b.UserName))
			padBuf(buf)
		}
	}

	// buffer the header now that we have enough data
	hbuf := bytes.NewBufferString("book")
	hbuf.Write(make([]byte, 4))
	hbuf.Write([]byte("mark"))
	hbuf.Write(make([]byte, 4))
	// size of the header
	binary.Write(hbuf, binary.LittleEndian, uint32(56))
	// size of the header
	binary.Write(hbuf, binary.LittleEndian, uint32(56))

	// convert the toc in bytes so we can calculate offsets
	toc := oMap.Bytes()

	// total size minus the header
	binary.Write(hbuf, binary.LittleEndian, 4+uint32(buf.Len()+len(toc)))
	// magic
	hbuf.Write([]byte{0x00, 0x00, 0x04, 0x10, 0x0, 0x0, 0x0, 0x0})
	// TODO: figure out those byte
	// 0x72, 0x73, 0x2F, 0x6D | 8 bytes that change
	hbuf.Write(make([]byte, 16))
	hbuf.Write([]byte{0x63, 0x65, 0x2F, 0x73})
	// end of header

	// offset to the TOC  (size of the body)
	binary.Write(hbuf, binary.LittleEndian, 4+uint32(buf.Len()))
	// body
	hbuf.Write(buf.Bytes())
	// toc
	hbuf.Write(toc)

	_, err := w.Write(hbuf.Bytes())
	return err
}

func (b *BookmarkData) prepareTypeData() {
	buf := &bytes.Buffer{}
	buf.Write([]byte{
		0x64, 0x6E, 0x69, 0x62, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	buf.Write(make([]byte, 12))
	// file extension
	ext := filepath.Ext(b.TargetPath())
	if strings.HasPrefix(ext, ".") {
		ext = ext[1:]
	}
	binary.Write(buf, binary.LittleEndian, uint32(len(ext)))
	buf.Write(make([]byte, 4))
	buf.Write([]byte(ext))
	buf.Write([]byte{0x3f, 0x3f, 0x3f, 0x3f, 0x1})
	buf.Write(make([]byte, 7))
	b.TypeData = buf.Bytes()
}

func (b *BookmarkData) String() string {
	out := fmt.Sprintf("Bookmark:\nSource Path: %s\n", filepath.Join(b.Path...))
	out += fmt.Sprintf("CNID path: %v\n", b.CNIDPath)
	out += fmt.Sprintf("CNID: %v\n", b.CNID)
	out += fmt.Sprintf("File creation date: %v\n", b.FileCreationDate)
	out += fmt.Sprintf("File Properties: %#v\n", b.FileProperties)
	return out
}

type offsetMap map[uint32]int

func (oMap offsetMap) Bytes() []byte {
	buf := &bytes.Buffer{}
	// Size of TOC in bytes, minus 8
	binary.Write(buf, binary.LittleEndian, uint32(3*4+len(oMap)*(3*4)))
	// magic number
	buf.Write([]byte{0xFE, 0xFF, 0xFF, 0xFF})
	// identifier
	binary.Write(buf, binary.LittleEndian, uint32(1))
	// Next TOC offset (or 0 if none)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	// Number of entries in this TOC
	binary.Write(buf, binary.LittleEndian, uint32(len(oMap)))

	// sort keys
	keys := make([]int, len(oMap))
	i := 0
	for k := range oMap {
		keys[i] = int(k)
		i++
	}
	sort.Ints(keys)

	for _, k := range keys {
		// key
		binary.Write(buf, binary.LittleEndian, uint32(k))
		// offset
		binary.Write(buf, binary.LittleEndian, uint32(oMap[uint32(k)])+4)
		// reserved
		binary.Write(buf, binary.LittleEndian, uint32(0))
	}

	return buf.Bytes()
}
