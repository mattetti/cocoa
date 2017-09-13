package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	fsType := []byte{}
	for _, b := range stat.Fstypename {
		if b == 0 {
			break
		}
		fsType = append(fsType, byte(b))
	}
	fileSystemType := string(fsType)

	var volumeAttrs *darwin.AttrList
	buf := make([]byte, 512)
	switch fileSystemType {
	case "hfs":
		volumeAttrs, err = darwin.GetAttrList(volPath,
			darwin.AttrListMask{
				CommonAttr: darwin.ATTR_CMN_CRTIME,
				VolAttr: darwin.ATTR_VOL_SIZE |
					darwin.ATTR_VOL_NAME |
					darwin.ATTR_VOL_UUID,
			},
			buf, 0|darwin.FSOPT_REPORT_FULLSIZE)
		if err != nil {
			log.Printf("failed to retrieve volume attribute list (using blank values) - %s", err)
			volumeAttrs = &darwin.AttrList{
				CreationTime: &darwin.TimeSpec{},
			}
		}
		//we don't seem to be able to get the vol attributes for other formats such as "exFat"
	default:
		volumeAttrs = &darwin.AttrList{
			CreationTime: &darwin.TimeSpec{},
		}
		if st, err := os.Stat(volPath); err != nil {
			volumeAttrs.VolSize = st.Size()
		}
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
		FileSystemType:     fileSystemType,
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
	// TODO: look into using KCFURLVolumeIsEjectable if the drive isn't a volume root
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
	relPath, _ := filepath.Rel("/", srcPath)
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
		subPath = filepath.Join("/", dir)
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
	darwin.SetAsAlias(dst)

	return err
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

func nullTermStr(b []byte) string {
	return string(b[:clen(b)])
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}
