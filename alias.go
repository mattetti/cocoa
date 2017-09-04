package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/mattetti/cocoa/darwin"
)

/*
	 Cocoa users can create virtual links to files using 3 ways:
	 symlinks, hard links and aliases. Symlinks point to a specific path,
	hard links to a specific files but require to be all deleted before deleting
	the original file and finally aliases which act like hard links but with more flexibility.

	Aliases wrap the BookmarkData format.
	Here is some documentation on the usage of bookmarks:
	https://developer.apple.com/library/content/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/AccessingFilesandDirectories/AccessingFilesandDirectories.html#//apple_ref/doc/uid/TP40010672-CH3-SW10

	The format was partly reverse engineered and documented in a few place, here
	is the best summary I found:
	https://github.com/al45tair/mac_alias/blob/master/doc/bookmark_fmt.rst
	and http://michaellynn.github.io/2015/10/24/apples-bookmarkdata-exposed/
	(documentation copied in doc/* in case the original repo disappears)
	Note that the documentation refers to the bookmark representation.
	The header for an alias is different but the body is the same.

	Another important point is that to become an alias file, the
	generated binary file must have a special finder extended attribute flag set.
*/

const (
	bmk_data_type_mask    = 0xffffff00
	bmk_data_subtype_mask = 0x000000ff

	bmk_string  = 0x0100
	bmk_data    = 0x0200
	bmk_number  = 0x0300
	bmk_date    = 0x0400
	bmk_boolean = 0x0500
	bmk_array   = 0x0600
	bmk_dict    = 0x0700
	bmk_uuid    = 0x0800
	bmk_url     = 0x0900
	bmk_null    = 0x0a00

	bmk_st_zero = 0x0000
	bmk_st_one  = 0x0001

	bmk_boolean_st_false = 0x0000
	bmk_boolean_st_true  = 0x0001

	bmk_url_st_absolute = 0x0001
	bmk_url_st_relative = 0x0002
)

// IsAlias returns positively if the passed file path is an alias.
func IsAlias(src string) bool {
	srcPath, err := filepath.Abs(src)
	if err != nil {
		return false
	}
	srcPath = filepath.Clean(srcPath)

	buf := make([]byte, 256)
	fileAttrs, err := darwin.GetAttrList(srcPath,
		darwin.AttrListMask{
			CommonAttr: darwin.ATTR_CMN_OBJTYPE | darwin.ATTR_CMN_FNDRINFO,
		},
		buf, darwin.FSOPT_NOFOLLOW)
	if err != nil {
		log.Printf("failed to retrieve file attribute list - %s", err)
		return false
	}

	return fileAttrs.FileInfo.FinderFlags&darwin.FFKIsAlias > 0
}

// Alias acts like os.Symlink but instead of creating a symlink, a bookmark is stored.
func Alias(src, dst string) error {
	srcPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get the path of the source - %s", err)
	}
	srcPath = filepath.Clean(srcPath)
	// read the attributes of the source.
	var stat syscall.Statfs_t

	err = syscall.Statfs(srcPath, &stat)
	if err != nil {
		return fmt.Errorf("failed to read the file stats - %s", err)
	}

	// Volume path
	volPathB := []byte{}
	for _, b := range stat.Mntonname {
		if b == 0x00 {
			break
		}
		volPathB = append(volPathB, byte(b))
	}
	volPath := string(volPathB)
	// volume attributes
	buf := make([]byte, 512)
	volumeAttrs, err := darwin.GetAttrList(volPath,
		darwin.AttrListMask{
			CommonAttr: darwin.ATTR_CMN_CRTIME,
			VolAttr:    darwin.ATTR_VOL_SIZE | darwin.ATTR_VOL_NAME | darwin.ATTR_VOL_UUID,
		},
		buf, 0|darwin.FSOPT_REPORT_FULLSIZE)
	if err != nil {
		return fmt.Errorf("failed to retrieve volume attribute list - %s", err)
	}

	// file attributes
	fileAttrs, err := darwin.GetAttrList(srcPath,
		darwin.AttrListMask{
			CommonAttr: darwin.ATTR_CMN_OBJTYPE |
				darwin.ATTR_CMN_FNDRINFO |
				darwin.ATTR_CMN_CRTIME |
				darwin.ATTR_CMN_FILEID,
		},
		buf, darwin.FSOPT_NOFOLLOW)
	if err != nil {
		return fmt.Errorf("failed to retrieve file attribute list - %s", err)
	}

	// TODO: decode the source alias and adjust the source instead of failing.
	// macOS UI lest you create an alias to an alias by reading the alias source
	// and creating another version of the alias.
	if fileAttrs.FileInfo.FinderFlags&darwin.FFKIsAlias > 0 {
		return fmt.Errorf("can't safely bookmark to a bookmark, choose another source")
	}

	w, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("failed to create the file at destination - %s", err)
	}
	defer w.Close()

	bookmark := &BookmarkData{
		FileCreationDate:   fileAttrs.CreationTime.Time(),
		VolumePath:         volPath,
		VolumeIsRoot:       volPath == "/",
		VolumeURL:          "file://" + volPath,
		VolumeName:         volumeAttrs.VolName,
		VolumeSize:         volumeAttrs.VolSize,
		VolumeCreationDate: volumeAttrs.CreationTime.Time(),
		VolumeUUID:         strings.ToUpper(volumeAttrs.StringVolUUID()),
		VolumeProperties:   []byte{},
		CreationOptions:    512,
		WasFileReference:   true,
		UserName:           "unknown",
		CNID:               uint32(fileAttrs.FileID),
		UID:                99,
	}

	// volume properties
	bb := &bytes.Buffer{}
	binary.Write(bb, binary.LittleEndian, uint64(0x81|darwin.KCFURLVolumeSupportsPersistentIDs))
	binary.Write(bb, binary.LittleEndian, uint64(0x13ef|darwin.KCFURLVolumeSupportsPersistentIDs))
	binary.Write(bb, binary.LittleEndian, uint64(0))
	bookmark.VolumeProperties = bb.Bytes()

	// file properties
	bb.Reset()
	switch fileAttrs.ObjType {
	// file
	case darwin.VREG:
		binary.Write(bb, binary.LittleEndian, uint64(darwin.KCFURLResourceIsRegularFile))
		// folder
	case darwin.VDIR:
		binary.Write(bb, binary.LittleEndian, uint64(darwin.KCFURLResourceIsDirectory))
		// symlink
	case darwin.VLNK:
		binary.Write(bb, binary.LittleEndian, uint64(darwin.KCFURLResourceIsSymbolicLink))
	default:
		binary.Write(bb, binary.LittleEndian, uint64(darwin.KCFURLResourceIsRegularFile))
	}
	binary.Write(bb, binary.LittleEndian, uint64(0x0f))
	binary.Write(bb, binary.LittleEndian, uint64(0))
	bookmark.FileProperties = bb.Bytes()

	// getting data about each node of the path
	relPath, _ := filepath.Rel(string(volPath), srcPath)
	buf = make([]byte, 256)
	subPath := srcPath
	subPathAttrs, err := darwin.GetAttrList(subPath, darwin.AttrListMask{CommonAttr: darwin.ATTR_CMN_FILEID}, buf, 0)
	if err != nil {
		return fmt.Errorf("failed to retrieve file id for %s - %s", subPath, err)
	}
	bookmark.CNIDPath = []uint32{subPathAttrs.FileID}
	bookmark.Path = []string{filepath.Base(subPath)}

	// walk the path and extract the file id of each sub path
	dir := filepath.Dir(relPath)
	for dir != "" {
		dir, _ = filepath.Split(filepath.Clean(dir))
		if dir == "" {
			break
		}

		bookmark.Path = append([]string{filepath.Base(dir)}, bookmark.Path...)
		buf = make([]byte, 256)
		subPath = filepath.Join(string(volPath), dir)
		subPathAttrs, err = darwin.GetAttrList(subPath, darwin.AttrListMask{CommonAttr: darwin.ATTR_CMN_FILEID}, buf, 0)
		if err != nil {
			return fmt.Errorf("failed to retrieve file id for %s - %s", subPath, err)
		}
		bookmark.CNIDPath = append([]uint32{subPathAttrs.FileID}, bookmark.CNIDPath...)
	}

	bookmark.ContainingFolderIDX = uint32(len(bookmark.Path)) - 2

	bookmark.Write(w)
	w.Close()
	// turn the file into an actual alias by setting the finder flags
	darwin.SetAlias(dst)

	return err
}

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

	oMap[darwin.KBookmarkCreationOptions] = buf.Len()
	buf.Write(encodedUint32(1024))

	// write each path items one by one
	pathOffsets := make([]int, len(b.Path))
	for i, item := range b.Path {
		// track the starting offset of each item (append 4 for the body size value)
		pathOffsets[i] = 4 + buf.Len()
		// get the offset of the last item in the path
		if i == len(b.Path)-1 {
			oMap[darwin.KBookmarkFullFileName] = pathOffsets[i]
		}
		buf.Write(encodedStringItem(item))
	}
	padBuf(buf)

	// offset to the start of path offsets
	// the TOC will point to here so we can find how many items are in the array
	// and access each item to rebuild the path.
	// 0x04 0x10
	oMap[darwin.KBookmarkPath] = buf.Len()
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
	oMap[darwin.KBookmarkCNIDPath] = buf.Len()
	binary.Write(buf, binary.LittleEndian, uint32(len(b.CNIDPath)*4))
	binary.Write(buf, binary.LittleEndian, uint32(bmk_array|bmk_st_one))
	for _, cnid := range b.CNIDPath {
		binary.Write(buf, binary.LittleEndian, uint32(cnid))
	}
	padBuf(buf)

	oMap[darwin.KBookmarkFileID] = buf.Len()
	buf.Write(encodedUint32(b.CNID))
	padBuf(buf)

	// file properties
	// 0x10 0x10
	oMap[darwin.KBookmarkFileProperties] = buf.Len()
	buf.Write(encodedBytes(b.FileProperties))
	padBuf(buf)

	// KBookmarkFileCreationDate 0x04 0x10
	oMap[darwin.KBookmarkFileCreationDate] = buf.Len()
	buf.Write(encodedTime(b.FileCreationDate))
	padBuf(buf)

	// 0x54 0x10 unknown but seems to always be 1
	// 0x55 0x10 unknown, point to the same value
	oMap[darwin.KBookmarkUnknown] = buf.Len()
	oMap[darwin.KBookmarkUnknown1] = buf.Len()
	buf.Write(encodedUint32(uint32(1)))
	padBuf(buf)

	// 0x56 0x10 bool set to true
	oMap[darwin.KBookmarkUnknown2] = buf.Len()
	buf.Write(encodedBool(true))
	padBuf(buf)

	// KBookmarkVolumePath 0x02 0x20
	oMap[darwin.KBookmarkVolumePath] = buf.Len()
	buf.Write(encodedStringItem(b.VolumePath))
	padBuf(buf)

	// KBookmarkVolumeURL 0x05 0x20
	oMap[darwin.KBookmarkVolumeURL] = buf.Len()
	binary.Write(buf, binary.LittleEndian, uint32(len(b.VolumeURL)))
	// only support absolute path for now
	binary.Write(buf, binary.LittleEndian, uint32(bmk_url|bmk_url_st_absolute))
	buf.Write([]byte(b.VolumeURL))
	padBuf(buf)

	// KBookmarkVolumeName 0x10 0x20
	oMap[darwin.KBookmarkVolumeName] = buf.Len()
	buf.Write(encodedStringItem(b.VolumeName))
	padBuf(buf)

	// KBookmarkVolumeUUID 0x11 0x20
	oMap[darwin.KBookmarkVolumeUUID] = buf.Len()
	buf.Write(encodedStringItem(b.VolumeUUID))
	padBuf(buf)

	// KBookmarkVolumeSize 0x12 0x20
	oMap[darwin.KBookmarkVolumeSize] = buf.Len()
	buf.Write(encodedUint64(uint64(b.VolumeSize)))
	padBuf(buf)

	// KBookmarkVolumeCreationDate 0x13 0x20
	oMap[darwin.KBookmarkVolumeCreationDate] = buf.Len()
	buf.Write(encodedTime(b.VolumeCreationDate))
	padBuf(buf)

	// KBookmarkVolumeProperties 0x20 0x20
	oMap[darwin.KBookmarkVolumeProperties] = buf.Len()
	buf.Write(encodedBytes(b.VolumeProperties))
	padBuf(buf)

	// KBookmarkVolumeIsRoot 0x30 20
	oMap[darwin.KBookmarkVolumeIsRoot] = buf.Len()
	buf.Write(encodedBool(b.VolumeIsRoot))
	padBuf(buf)

	// KBookmarkContainingFolder 0x01 0xc0
	oMap[darwin.KBookmarkContainingFolder] = buf.Len()
	buf.Write(encodedUint32(b.ContainingFolderIDX))
	padBuf(buf)

	// KBookmarkUserName 0x11 0xc0
	oMap[darwin.KBookmarkUserName] = buf.Len()
	buf.Write(encodedStringItem(b.UserName))
	padBuf(buf)

	// KBookmarkUID 0x12 0xc0
	oMap[darwin.KBookmarkUID] = buf.Len()
	buf.Write(encodedUint32(b.UID))
	padBuf(buf)

	// KBookmarkWasFileReference
	oMap[darwin.KBookmarkWasFileReference] = buf.Len()
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

// pad if needed
func padBuf(buf *bytes.Buffer) {
	offset := buf.Len()
	if diff := offset & 3; diff > 0 {
		buf.Write(make([]byte, 4-diff))
	}
}

func encodedBytes(b []byte) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf, uint32(len(b)))
	binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_data|bmk_st_one))
	buf = append(buf, b...)
	offset := len(buf)
	if diff := offset & 3; diff > 0 {
		buf = append(buf, make([]byte, 4-diff)...)
	}
	return buf
}

func encodedStringItem(str string) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf, uint32(len(str)))
	binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_string|bmk_st_one))
	buf = append(buf, []byte(str)...)
	offset := len(buf)
	if diff := offset & 3; diff > 0 {
		buf = append(buf, make([]byte, 4-diff)...)
	}
	return buf
}

func encodedTime(ts time.Time) []byte {
	buf := &bytes.Buffer{}
	// size
	binary.Write(buf, binary.LittleEndian, uint32(8))
	// type
	binary.Write(buf, binary.LittleEndian, uint32(bmk_date|bmk_st_zero))
	// data
	binary.Write(buf, binary.BigEndian, float64(ts.Sub(darwin.Epoch).Seconds()))
	return buf.Bytes()
}

func encodedBool(v bool) []byte {
	buf := make([]byte, 8)
	if v {
		binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_boolean|bmk_boolean_st_true))
	} else {
		binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_boolean|bmk_boolean_st_false))
	}
	return buf
}

func encodedUint32(n uint32) []byte {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf, uint32(4))
	binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_number|darwin.KCFNumberSInt32Type))
	binary.LittleEndian.PutUint32(buf[8:], n)
	return buf
}

func encodedUint64(n uint64) []byte {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint32(buf, uint32(8))
	binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_number|darwin.KCFNumberSInt64Type))
	binary.LittleEndian.PutUint64(buf[8:], n)
	return buf
}
