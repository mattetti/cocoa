package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"
)

// BookmarkData represents the data structure holding the bookmark information
type BookmarkData struct {
	Path                []string
	CNIDPath            []uint32
	FileCreationDate    time.Time
	FileProperties      []byte
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

	// write each path items one by one
	pathOffsets := make([]int, len(b.Path))
	for i, item := range b.Path {
		// track the starting offset of each item (append 4 for the body size value)
		pathOffsets[i] = 4 + buf.Len()
		// get the offset of the last item in the path
		if i == len(b.Path)-1 {
			oMap[KBookmarkFullFileName] = pathOffsets[i]
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

	// file ids for the path
	// 0x05 0x10
	oMap[KBookmarkCNIDPath] = buf.Len()
	binary.Write(buf, binary.LittleEndian, uint32(len(b.CNIDPath)*4))
	binary.Write(buf, binary.LittleEndian, uint32(bmk_array|bmk_st_one))
	for _, cnid := range b.CNIDPath {
		binary.Write(buf, binary.LittleEndian, uint32(cnid))
	}
	padBuf(buf)

	oMap[KBookmarkFileID] = buf.Len()
	buf.Write(encodedUint32(b.CNID))
	padBuf(buf)

	// file properties
	// 0x10 0x10
	oMap[KBookmarkFileProperties] = buf.Len()
	buf.Write(encodedBytes(b.FileProperties))
	padBuf(buf)

	// KBookmarkFileCreationDate 0x04 0x10
	oMap[KBookmarkFileCreationDate] = buf.Len()
	buf.Write(encodedTime(b.FileCreationDate))
	padBuf(buf)

	// 0x54 0x10 unknown but seems to always be 1
	// 0x55 0x10 unknown, point to the same value
	oMap[KBookmarkUnknown] = buf.Len()
	oMap[KBookmarkUnknown1] = buf.Len()
	buf.Write(encodedUint32(uint32(1)))
	padBuf(buf)

	// 0x56 0x10 bool set to true
	oMap[KBookmarkUnknown2] = buf.Len()
	buf.Write(encodedBool(true))
	padBuf(buf)

	// KBookmarkVolumePath 0x02 0x20
	oMap[KBookmarkVolumePath] = buf.Len()
	buf.Write(encodedStringItem(b.VolumePath))
	padBuf(buf)

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

	// KBookmarkVolumeUUID 0x11 0x20
	oMap[KBookmarkVolumeUUID] = buf.Len()
	buf.Write(encodedStringItem(b.VolumeUUID))
	padBuf(buf)

	// KBookmarkVolumeSize 0x12 0x20
	oMap[KBookmarkVolumeSize] = buf.Len()
	buf.Write(encodedUint64(uint64(b.VolumeSize)))
	padBuf(buf)

	// KBookmarkVolumeCreationDate 0x13 0x20
	oMap[KBookmarkVolumeCreationDate] = buf.Len()
	buf.Write(encodedTime(b.VolumeCreationDate))
	padBuf(buf)

	// KBookmarkVolumeProperties 0x20 0x20
	oMap[KBookmarkVolumeProperties] = buf.Len()
	buf.Write(encodedBytes(b.VolumeProperties))
	padBuf(buf)

	// KBookmarkVolumeIsRoot 0x30 20
	oMap[KBookmarkVolumeIsRoot] = buf.Len()
	buf.Write(encodedBool(b.VolumeIsRoot))
	padBuf(buf)

	// KBookmarkContainingFolder 0x01 0xc0
	oMap[KBookmarkContainingFolder] = buf.Len()
	buf.Write(encodedUint32(b.ContainingFolderIDX))
	padBuf(buf)

	// KBookmarkUserName 0x11 0xc0
	oMap[KBookmarkUserName] = buf.Len()
	buf.Write(encodedStringItem(b.UserName))
	padBuf(buf)

	// KBookmarkUID 0x12 0xc0
	oMap[KBookmarkUID] = buf.Len()
	buf.Write(encodedUint32(b.UID))
	padBuf(buf)

	// KBookmarkWasFileReference
	oMap[KBookmarkWasFileReference] = buf.Len()
	buf.Write(encodedBool(b.WasFileReference))
	padBuf(buf)

	// 0xf022

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
	toc := oMap.Bytes(4)

	// total size minus the header
	binary.Write(hbuf, binary.LittleEndian, 4+uint32(buf.Len()+len(toc)))
	// magic
	hbuf.Write([]byte{0x00, 0x00, 0x04, 0x10, 0x0, 0x0, 0x0, 0x0})
	// TODO: figure out those byte since they seem to set the icon
	hbuf.Write(make([]byte, 20))
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

func (b *BookmarkData) String() string {
	out := fmt.Sprintf("Bookmark:\nSource Path: %s\n", filepath.Join(b.Path...))
	out += fmt.Sprintf("CNID path: %v\n", b.CNIDPath)
	out += fmt.Sprintf("CNID: %v\n", b.CNID)
	out += fmt.Sprintf("File creation date: %v\n", b.FileCreationDate)
	out += fmt.Sprintf("File Properties: %#v\n", b.FileProperties)
	return out
}

type offsetMap map[uint32]int

func (oMap offsetMap) Bytes(additionalOffset int) []byte {
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
