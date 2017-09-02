package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/mattetti/cocoa/darwin"
)

/*
	Cocoa provides an API to persist NSURLs. Older versions of the OS were using
	alias records but that API was deprecated in favor of bookmarks. Cocoa does
	also support symlinks and hardlinks but those behave differently than
	bookmarks. Unfortunately, Apple doesn't document the Bookmark Data format.

	Here is some documentation on the usage of bookmarks:
	https://developer.apple.com/library/content/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/AccessingFilesandDirectories/AccessingFilesandDirectories.html#//apple_ref/doc/uid/TP40010672-CH3-SW10

	The format was partly reverse engineered and documented in a few place, here
	is the best summary I found:
	https://github.com/al45tair/mac_alias/blob/master/doc/bookmark_fmt.rst
	(documentation copied in doc/* in case the original repo disappears)
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
)

// Bookmark acts like os.Symlink but instead of creating a symlink, a bookmark is stored.
func Bookmark(src, dst string) error {
	srcPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get the path of the source - %s", err)
	}
	srcPath = filepath.Clean(srcPath)
	// read the attributes of the source.
	var stat syscall.Statfs_t

	w, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("failed to create the file at destination - %s", err)
	}
	defer w.Close()

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
			CommonAttr: darwin.ATTR_CMN_OBJTYPE | darwin.ATTR_CMN_OBJTYPE | darwin.ATTR_CMN_CRTIME | darwin.ATTR_CMN_FILEID,
		},
		buf, darwin.FSOPT_NOFOLLOW)
	if err != nil {
		return fmt.Errorf("failed to retrieve file attribute list - %s", err)
	}

	bookmark := &BookmarkData{
		FileCreationDate:   fileAttrs.CreationTime,
		VolumePath:         volPath,
		VolumeIsRoot:       volPath == "/",
		VolumeURL:          "file://" + volPath,
		VolumeName:         volumeAttrs.VolName,
		VolumeSize:         volumeAttrs.VolSize,
		VolumeCreationDate: volumeAttrs.CreationTime,
		VolumeUUID:         strings.ToUpper(volumeAttrs.StringVolUUID()),
		VolumeProperties:   []byte{},
		CreationOptions:    512,
		WasFileReference:   true,
		UserName:           "unknown",
		UID:                99,
	}

	// volume properties
	bb := &bytes.Buffer{}
	binary.Write(bb, binary.LittleEndian, 0x81|darwin.KCFURLVolumeSupportsPersistentIDs)
	binary.Write(bb, binary.LittleEndian, 0x13ef|darwin.KCFURLVolumeSupportsPersistentIDs)
	bookmark.VolumeProperties = append(bb.Bytes(), 0)

	// file properties
	bb = &bytes.Buffer{}
	switch fileAttrs.ObjType {
	case darwin.VREG:
		binary.Write(bb, binary.LittleEndian, darwin.KCFURLResourceIsRegularFile)
	case darwin.VDIR:
		binary.Write(bb, binary.LittleEndian, darwin.KCFURLResourceIsDirectory)
	case darwin.VLNK:
		binary.Write(bb, binary.LittleEndian, darwin.KCFURLResourceIsSymbolicLink)
	default:
		binary.Write(bb, binary.LittleEndian, darwin.KCFURLResourceIsRegularFile)
	}
	bookmark.FileProperties = append(bb.Bytes(), 0x0f, 0)

	// getting data about each node of the path
	relPath, _ := filepath.Rel(string(volPath), srcPath)
	buf = make([]byte, 256)
	subPath := srcPath
	subPathAttrs, err := darwin.GetAttrList(subPath, darwin.AttrListMask{CommonAttr: darwin.ATTR_CMN_FILEID}, buf, 0)
	if err != nil {
		return fmt.Errorf("failed to retrieve file id for %s - %s", subPath, err)
	}
	bookmark.CNIDPath = []uint64{subPathAttrs.FileID}
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
		bookmark.CNIDPath = append([]uint64{subPathAttrs.FileID}, bookmark.CNIDPath...)
	}

	bookmark.ContainingFolderIDX = uint32(len(bookmark.Path)) - 2

	bookmark.Write(w)

	// TODO: set attrx
	/*
		payload := make([]byte, 32)
		for i, b := range []byte{0x61, 0x6c, 0x69, 0x73, 0x4d, 0x41, 0x43, 0x53, 0x80} {
			payload[i] = b
		}
		err = xattr.Set("bookmark", "com.apple.FinderInfo", payload)
		if err != nil {
			panic(err)
		}
	*/
	return nil
}

type BookmarkData struct {
	Path                []string
	CNIDPath            []uint64
	FileCreationDate    *darwin.TimeSpec
	FileProperties      []byte
	ContainingFolderIDX uint32
	VolumePath          string
	VolumeIsRoot        bool
	VolumeURL           string // file://' + volPath
	VolumeName          string
	VolumeSize          int64
	VolumeCreationDate  *darwin.TimeSpec
	VolumeUUID          string // must be uppercase
	VolumeProperties    []byte
	CreationOptions     uint32 // 512
	WasFileReference    bool   // true
	UserName            string // unknown
	UID                 uint32 // 99
}

// Write converts the bookmark data into binary data and writes it to the passed writer.
// Note that the writes are buffered and written all at once.
func (b *BookmarkData) Write(w io.Writer) error {
	buf := &bytes.Buffer{}

	// bodyStart := buf.Len()
	oMap := offsetMap{}

	// Path
	oMap[darwin.KBookmarkPath] = buf.Len()
	// length of data (n items)
	binary.Write(buf, binary.LittleEndian, uint32(len(b.Path)*4))
	// type
	binary.Write(buf, binary.LittleEndian, uint32(bmk_array|bmk_st_one))
	// data
	sliceOffset := uint32(buf.Len() + len(b.Path)*4)
	sBuf := &bytes.Buffer{}
	for _, item := range b.Path {
		// offset FIX ME
		// write the offset start offset + x paths * offset (4 bytes) + len of encoded content
		data := encodedStringItem(item)
		binary.Write(buf, binary.LittleEndian, sliceOffset+uint32(sBuf.Len()))
		sBuf.Write(data)
	}

	// write the data
	buf.Write(sBuf.Bytes())

	// file properties
	oMap[darwin.KBookmarkFileProperties] = buf.Len()
	buf.Write(b.FileProperties)

	// KBookmarkFileCreationDate = 0x1040
	oMap[darwin.KBookmarkFileCreationDate] = buf.Len()
	// length of data
	binary.Write(buf, binary.LittleEndian, uint32(8))
	// type
	binary.Write(buf, binary.LittleEndian, uint32(bmk_date|bmk_st_zero))
	// data
	fmt.Println(b.FileCreationDate.DarwinDuration().Seconds())
	// timestamp
	binary.Write(buf, binary.BigEndian, float64(b.FileCreationDate.DarwinDuration().Seconds()))

	// KBookmarkVolumePath = 0x2002
	// KBookmarkVolumeURL = 0x2005
	// KBookmarkVolumeName = 0x2010

	// KBookmarkVolumeUUID = 0x2011
	// KBookmarkVolumeSize = 0x2012
	// KBookmarkVolumeCreationDate = 0x2013
	// KBookmarkVolumeProperties = 0x2020
	// KBookmarkVolumeIsRoot = 0x2030

	// KBookmarkContainingFolder = 0xc001
	// KBookmarkUserName = 0xc011
	// KBookmarkUID = 0xc012
	// KBookmarkWasFileReference = 0xd001

	// 0xd010
	// 0xf017
	// 0xf022

	// about to write the TOC header
	// tocHeaderPos := buf.Len()
	// TODO overwrite the value at tocOffsetCounterPos

	// header now that we have enough data
	hbuf := bytes.NewBufferString("book")
	hbuf.Write(make([]byte, 4))
	hbuf.Write([]byte("mark"))
	hbuf.Write(make([]byte, 4))
	// size of the header
	binary.Write(hbuf, binary.LittleEndian, uint32(56))
	// size of the header
	binary.Write(hbuf, binary.LittleEndian, uint32(56))

	toc := oMap.Bytes()

	// total size minus the header
	binary.Write(hbuf, binary.LittleEndian, 4+uint32(buf.Len()+len(toc)))
	// magic
	hbuf.Write([]byte{0x00, 0x00, 0x04, 0x10, 0x0, 0x0, 0x0, 0x0})

	// TODO: figure out those byte since they seem to set the icon
	hbuf.Write(make([]byte, 20))
	// offset to the TOC
	binary.Write(hbuf, binary.LittleEndian, 4+uint32(buf.Len()))
	// body
	hbuf.Write(buf.Bytes())
	// toc
	hbuf.Write(toc)

	_, err := w.Write(hbuf.Bytes())
	return err
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

	// TODO: sort keys
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
		binary.Write(buf, binary.LittleEndian, uint32(oMap[uint32(k)]))
		// reserved
		binary.Write(buf, binary.LittleEndian, uint32(0))
	}

	return buf.Bytes()
}

func encodedStringItem(str string) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf, uint32(len(str)))
	binary.LittleEndian.PutUint32(buf[4:], uint32(bmk_string|bmk_st_one))
	buf = append(buf, []byte(str)...)
	// pad if needed
	if diff := len(buf) & 3; diff > 0 {
		buf = append(buf, make([]byte, 4-diff)...)
	}

	return buf
}
