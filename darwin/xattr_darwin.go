package darwin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"syscall"
	"unsafe"
)

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
		ref := AttrRef{}
		if err = binary.Read(r, binary.LittleEndian, &ref); err != nil {
			return results, fmt.Errorf("failed reading ATTR_CMN_NAME ref - %s", err)
		}
		offsetPos := pos()
		// move to the offset minus the size of AttrRef (8)
		if ref.Offset > 0 {
			if _, err = r.Seek(int64(ref.Offset)-8, io.SeekCurrent); err != nil {
				return results, fmt.Errorf("failed to skip to the common name - %s", err)
			}
		}
		if ref.Len > 0 {
			// len-1 because the string is null terminated
			name := make([]byte, ref.Len-1)
			r.Read(name)
			results.Name = string(name)
		}
		// move back to the original offset
		if _, err = r.Seek(offsetPos, io.SeekStart); err != nil {
			return results, fmt.Errorf("failed to skip back after reading the common name - %s", err)
		}
	}

	if mask.CommonAttr&ATTR_CMN_DEVID > 0 {
		if err = binary.Read(r, binary.LittleEndian, &results.DevID); err != nil {
			return results, fmt.Errorf("failed to read the cmd devid - %s", err)
		}
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
		// (read/write) 32 bytes of data for use by the Finder.  Equivalent to the concatenation
		// of a FileInfo structure and an ExtendedFileInfo structure (or, for
		// directories, a FolderInfo structure and an ExtendedFolderInfo structure).
		// These structures are defined in <CarbonCore/Finder.h>.

		// This attribute is not byte swapped by the file system.  The value of multi-byte multibyte
		// byte fields on disk is always big endian.  When running on a little endian
		// system (such as Darwin on x86), you must byte swap any multibyte fields.
		if results.IsFolder() {
			if err = binary.Read(r, binary.BigEndian, &results.FolderInfo); err != nil {
				return results, fmt.Errorf("failed reading finder folder information - %s", err)
			}
		} else {
			if err = binary.Read(r, binary.BigEndian, &results.FileInfo); err != nil {
				return results, fmt.Errorf("failed reading finder file information - %s", err)
			}
		}
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
		if err = binary.Read(r, binary.LittleEndian, &results.UUID); err != nil {
			return results, fmt.Errorf("failed to read uuid - %s", err)
		}
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
			return results, fmt.Errorf("failed reading ATTR_VOL_NAME ref - %s", err)
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

func setxattr(path string, name string, value *byte, size int, pos int, options int) error {
	if _, _, e1 := syscall.Syscall6(syscall.SYS_SETXATTR, uintptr(unsafe.Pointer(syscall.StringBytePtr(path))), uintptr(unsafe.Pointer(syscall.StringBytePtr(name))), uintptr(unsafe.Pointer(value)), uintptr(size), uintptr(pos), uintptr(options)); e1 != syscall.Errno(0) {
		return e1
	}
	return nil
}
