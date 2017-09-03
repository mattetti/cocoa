// daring is a package implementing low level features exposed in Cocoa but that
// can also be used via syscalls.
package darwin

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const (
	attrBitMapCount      = 5
	dash            byte = '-'
)

var (
	// Epoch is the darwin epoch instead of unix'
	Epoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
)

type AttrList struct {
	Name               string
	FileID             uint32
	ReturnedAttributes *AttrSet
	CreationTime       *TimeSpec
	VolName            string
	VolSize            int64
	VolUUID            [16]byte
	ObjType            uint32
}

// StringVolUUID returns a string formatted version of the volume UUID
func (attr *AttrList) StringVolUUID() string {
	return toUUIDString(attr.VolUUID)
}

// AttrListMask is a structure defined in <sys/attr.h> and used by GetAttrList
// http://www.manpagez.com/man/2/getattrlist/
type AttrListMask struct {
	// number of attr. bit sets in list
	bitmapCount uint16
	// (to maintain 4-byte alignment)
	_ uint16
	// common attribute group. A bit set that specifies the common attributes
	// that you require. Common attributes relate to all types of file system
	// objects
	CommonAttr uint32
	// volume attribute group. A bit set that specifies the volume attributes
	// that you require.  Volume attributes relate to volumes (that is, mounted
	// file systems).  If you request volume attributes, path must reference the
	// root of a volume.  In addition, you can't request volume attributes if
	// you also request file or directory attributes.
	VolAttr uint32
	// directory attribute group. A bit set that specifies the directory
	// attributes that you require.
	DirAttr uint32
	// file attribute group. A bit set that specifies the file attributes that
	// you require.
	FileAttr uint32
	// fork attribute group. A bit set that specifies the fork attributes that
	// you require.  Fork attributes relate to the actual data in the file,
	// which can be held in multiple named contiguous ranges, or forks.
	ForkAttr uint32
}

type AttrSet struct {
	CommonAttr uint32
	VolAttr    uint32
	DirAttr    uint32
	FileAttr   uint32
	ForkAttr   uint32
}

type AttrRef struct {
	Offset int32
	Len    uint32
}

type TimeSpec struct {
	Sec  int64
	Nsec int64
}

func (ts TimeSpec) String() string {
	return fmt.Sprintf("Sec: %d, Nsec: %d", ts.Sec, ts.Nsec)
}

// Time returns a unix time representation of the time spec
func (ts TimeSpec) Time() time.Time {
	return time.Unix(ts.Sec, ts.Nsec)
}

func (ts TimeSpec) DarwinDuration() time.Duration {
	return ts.Time().Sub(Epoch)
}

// SetAlias flag the destination file as an alias/bookmark. Don't use on the wrong file!
func SetAlias(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%s can't be converted to an absolute path - %s", path, err)
	}
	aliasMagicFlag := []byte{0x61, 0x6c, 0x69, 0x73, 0x4d, 0x41, 0x43, 0x53, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	var dataval *byte = nil
	datalen := len(aliasMagicFlag)
	if datalen > 0 {
		dataval = &aliasMagicFlag[0]
	}
	return setxattr(filepath.Clean(absPath), "com.apple.FinderInfo", dataval, datalen, 0, 0)
}

// GetAttrList returns attributes (that is, metadata) of file system objects. GetAttrList()
// works on the file system object named by path. You can think of getattrlist() as a
// seriously enhanced version of syscall.Stat.  The functions return attributes about
// the specified file system object into the buffer specified by attrBuf and
// attrBufSize.  The attrList parameter determines what attributes are returned.
//
// https://developer.apple.com/legacy/library/documentation/Darwin/Reference/ManPages/man2/getattrlist.2.html
func GetAttrList(path string, mask AttrListMask, attrBuf []byte, options uint32) (results *AttrList, err error) {
	results = &AttrList{}
	if len(attrBuf) < 4 {
		return results, errors.New("attrBuf too small")
	}
	mask.bitmapCount = attrBitMapCount

	if mask.VolAttr > 0 {
		mask.VolAttr |= ATTR_VOL_INFO
	}
	options |= FSOPT_REPORT_FULLSIZE

	var _p0 *byte
	_p0, err = syscall.BytePtrFromString(path)
	if err != nil {
		return results, err
	}
	_, _, e1 := syscall.Syscall6(
		syscall.SYS_GETATTRLIST,
		uintptr(unsafe.Pointer(_p0)),
		uintptr(unsafe.Pointer(&mask)),
		uintptr(unsafe.Pointer(&attrBuf[0])),
		uintptr(len(attrBuf)),
		uintptr(options),
		0,
	)
	if e1 != 0 {
		return results, e1
	}

	// binary.LittleEndian.Uint32(attrBuf)
	size := *(*uint32)(unsafe.Pointer(&attrBuf[0]))
	// dat is the section of attrBuf that contains valid data,
	// without the 4 byte length header. All attribute offsets
	// are relative to dat.
	dat := attrBuf
	dat = dat[4:] // remove length prefix
	if int(size)+8 < len(attrBuf) {
		dat = dat[:size+8]
	}
	r := bytes.NewReader(dat)
	pos := func() int64 { return r.Size() - int64(r.Len()) }

	if mask.CommonAttr&ATTR_CMN_RETURNED_ATTRS > 0 {
		fmt.Println("ATTR_CMN_RETURNED_ATTRS not supported yet", pos())
	}

	if mask.CommonAttr&ATTR_CMN_NAME > 0 {
		fmt.Println("ATTR_CMN_NAME not supported yet", pos())
	}

	if mask.CommonAttr&ATTR_CMN_DEVID > 0 {
		fmt.Println("ATTR_CMN_DEVID not supported yet", pos())
	}

	if mask.CommonAttr&ATTR_CMN_FSID > 0 {
		fmt.Println("ATTR_CMN_FSID not supported yet", pos())
	}

	if mask.CommonAttr&ATTR_CMN_OBJTYPE > 0 {
		if err = binary.Read(r, binary.LittleEndian, &results.ObjType); err != nil {
			return results, fmt.Errorf("failed to read the object type - %s", err)
		}
	}

	if mask.CommonAttr&ATTR_CMN_OBJTAG > 0 {
		fmt.Println("ATTR_CMN_OBJTAG not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_OBJID > 0 {
		fmt.Println("ATTR_CMN_OBJID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_OBJPERMANENTID > 0 {
		fmt.Println("ATTR_CMN_OBJPERMANENTID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_PAROBJID > 0 {
		fmt.Println("ATTR_CMN_PAROBJID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_SCRIPT > 0 {
		fmt.Println("ATTR_CMN_SCRIPT not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_CRTIME > 0 {
		results.CreationTime = &TimeSpec{}
		if err = binary.Read(r, binary.LittleEndian, &results.CreationTime.Sec); err != nil {
			return results, fmt.Errorf("failed reading TTR_CMN_CRTIME sec - %s", err)
		}
		if err = binary.Read(r, binary.LittleEndian, &results.CreationTime.Nsec); err != nil {
			return results, fmt.Errorf("failed reading TTR_CMN_CRTIME nsec - %s", err)
		}
	}
	if mask.CommonAttr&ATTR_CMN_MODTIME > 0 {
		fmt.Println("ATTR_CMN_MODTIME not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_CHGTIME > 0 {
		fmt.Println("ATTR_CMN_CHGTIME not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_ACCTIME > 0 {
		fmt.Println("ATTR_CMN_ACCTIME not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_BKUPTIME > 0 {
		fmt.Println("ATTR_CMN_BKUPTIME not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_FNDRINFO > 0 {
		fmt.Println("ATTR_CMN_FNDRINFO not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_OWNERID > 0 {
		fmt.Println("ATTR_CMN_OWNERID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_GRPID > 0 {
		fmt.Println("ATTR_CMN_GRPID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_ACCESSMASK > 0 {
		fmt.Println("ATTR_CMN_ACCESSMASK not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_FLAGS > 0 {
		fmt.Println("ATTR_CMN_FLAGS not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_USERACCESS > 0 {
		fmt.Println("ATTR_CMN_USERACCESS not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_EXTENDED_SECURITY > 0 {
		fmt.Println("ATTR_CMN_EXTENDED_SECURITY not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_UUID > 0 {
		fmt.Println("ATTR_CMN_UUID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_GRPUUID > 0 {
		fmt.Println("ATTR_CMN_GRPUUID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_FILEID > 0 {
		if err = binary.Read(r, binary.LittleEndian, &results.FileID); err != nil {
			return results, fmt.Errorf("failed to read file ID - %s", err)
		}

	}
	if mask.CommonAttr&ATTR_CMN_PARENTID > 0 {
		fmt.Println("ATTR_CMN_PARENTID not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_FULLPATH > 0 {
		fmt.Println("ATTR_CMN_FULLPATH not supported yet", pos())
	}
	if mask.CommonAttr&ATTR_CMN_ADDEDTIME > 0 {
		fmt.Println("ATTR_CMN_ADDEDTIME not supported yet", pos())
	}

	// Volume attributes
	if mask.VolAttr&ATTR_VOL_FSTYPE > 0 {
		fmt.Println("ATTR_VOL_FSTYPE not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_SIGNATURE > 0 {
		fmt.Println("ATTR_VOL_SIGNATURE not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_SIZE > 0 {
		if err = binary.Read(r, binary.LittleEndian, &results.VolSize); err != nil {
			return results, fmt.Errorf("failed to read volume size - %s", err)
		}
	}
	if mask.VolAttr&ATTR_VOL_SPACEFREE > 0 {
		fmt.Println("ATTR_VOL_SPACEFREE not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_SPACEAVAIL > 0 {
		fmt.Println("ATTR_VOL_SPACEAVAIL not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_MINALLOCATION > 0 {
		fmt.Println("ATTR_VOL_MINALLOCATION not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_ALLOCATIONCLUMP > 0 {
		fmt.Println("ATTR_VOL_ALLOCATIONCLUMP not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_IOBLOCKSIZE > 0 {
		fmt.Println("ATTR_VOL_IOBLOCKSIZE not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_OBJCOUNT > 0 {
		fmt.Println("ATTR_VOL_OBJCOUNT not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_FILECOUNT > 0 {
		fmt.Println("ATTR_VOL_FILECOUNT not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_DIRCOUNT > 0 {
		fmt.Println("ATTR_VOL_DIRCOUNT not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_MAXOBJCOUNT > 0 {
		fmt.Println("ATTR_VOL_MAXOBJCOUNT not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_MOUNTPOINT > 0 {
		fmt.Println("ATTR_VOL_MOUNTPOINT not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_NAME > 0 {
		ref := AttrRef{}
		if err = binary.Read(r, binary.LittleEndian, &ref); err != nil {
			return results, fmt.Errorf("failed reading ATTR_CMN_NAME ref - %s", err)
		}
		offsetPos := pos()
		// move to the offset minus the size of AttrRef (8)
		if _, err = r.Seek(int64(ref.Offset)-8, io.SeekCurrent); err != nil {
			return results, fmt.Errorf("failed to skip to the volume name - %s", err)
		}
		if ref.Len > 0 {
			// len-1 because the string is null terminated
			name := make([]byte, ref.Len-1)
			r.Read(name)
			results.VolName = string(name)
		}
		// move back to the original offset
		if _, err = r.Seek(offsetPos, io.SeekStart); err != nil {
			return results, fmt.Errorf("failed to skip back after reading the volume name - %s", err)
		}

	}
	if mask.VolAttr&ATTR_VOL_MOUNTFLAGS > 0 {
		fmt.Println("ATTR_VOL_MOUNTFLAGS not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_MOUNTEDDEVICE > 0 {
		fmt.Println("ATTR_VOL_MOUNTEDDEVICE not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_ENCODINGSUSED > 0 {
		fmt.Println("ATTR_VOL_ENCODINGSUSED not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_CAPABILITIES > 0 {
		fmt.Println("ATTR_VOL_CAPABILITIES not supported yet", pos())
	}
	if mask.VolAttr&ATTR_VOL_UUID > 0 {
		if err = binary.Read(r, binary.LittleEndian, &results.VolUUID); err != nil {
			return results, fmt.Errorf("failed read the volume uuid - %s", err)
		}
	}
	if mask.VolAttr&ATTR_VOL_ATTRIBUTES > 0 {
		fmt.Println("ATTR_VOL_ATTRIBUTES not supported yet", pos())
	}

	// Directory
	if mask.DirAttr&ATTR_DIR_LINKCOUNT > 0 {
		fmt.Println("ATTR_DIR_LINKCOUNT not supported yet", pos())
	}
	if mask.DirAttr&ATTR_DIR_ENTRYCOUNT > 0 {
		fmt.Println("ATTR_DIR_ENTRYCOUNT not supported yet", pos())
	}
	if mask.DirAttr&ATTR_DIR_MOUNTSTATUS > 0 {
		fmt.Println("ATTR_DIR_MOUNTSTATUS not supported yet", pos())
	}

	// File
	if mask.FileAttr&ATTR_FILE_LINKCOUNT > 0 {
		fmt.Println("ATTR_FILE_LINKCOUNT not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_TOTALSIZE > 0 {
		fmt.Println("ATTR_FILE_TOTALSIZE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_ALLOCSIZE > 0 {
		fmt.Println("ATTR_FILE_ALLOCSIZE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_IOBLOCKSIZE > 0 {
		fmt.Println("ATTR_FILE_IOBLOCKSIZE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_CLUMPSIZE > 0 {
		fmt.Println("ATTR_FILE_CLUMPSIZE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_DEVTYPE > 0 {
		fmt.Println("ATTR_FILE_DEVTYPE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_FILETYPE > 0 {
		fmt.Println("ATTR_FILE_FILETYPE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_FORKCOUNT > 0 {
		fmt.Println("ATTR_FILE_FORKCOUNT not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_DATALENGTH > 0 {
		fmt.Println("ATTR_FILE_DATALENGTH not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_DATAALLOCSIZE > 0 {
		fmt.Println("ATTR_FILE_DATAALLOCSIZE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_DATAEXTENTS > 0 {
		fmt.Println("ATTR_FILE_DATAEXTENTS not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_RSRCLENGTH > 0 {
		fmt.Println("ATTR_FILE_RSRCLENGTH not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_RSRCALLOCSIZE > 0 {
		fmt.Println("ATTR_FILE_RSRCALLOCSIZE not supported yet", pos())
	}
	if mask.FileAttr&ATTR_FILE_RSRCEXTENTS > 0 {
		fmt.Println("ATTR_FILE_RSRCEXTENTS not supported yet", pos())
	}

	// fork
	if mask.ForkAttr&ATTR_FORK_TOTALSIZE > 0 {
		fmt.Println("ATTR_FORK_TOTALSIZE not supported yet", pos())
	}
	if mask.ForkAttr&ATTR_FORK_ALLOCSIZE > 0 {
		fmt.Println("ATTR_FORK_ALLOCSIZE not supported yet", pos())
	}

	return
}

func toUUIDString(uuid [16]byte) string {
	buf := make([]byte, 36)
	hex.Encode(buf[0:8], uuid[0:4])
	buf[8] = dash
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = dash
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = dash
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = dash
	hex.Encode(buf[24:], uuid[10:])

	return string(buf)
}

func setxattr(path string, name string, value *byte, size int, pos int, options int) error {
	if _, _, e1 := syscall.Syscall6(syscall.SYS_SETXATTR, uintptr(unsafe.Pointer(syscall.StringBytePtr(path))), uintptr(unsafe.Pointer(syscall.StringBytePtr(name))), uintptr(unsafe.Pointer(value)), uintptr(size), uintptr(pos), uintptr(options)); e1 != syscall.Errno(0) {
		return e1
	}
	return nil
}
