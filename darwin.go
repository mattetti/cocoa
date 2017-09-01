package cocoa

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"syscall"
	"unsafe"
)

const (
	attrBitMapCount = 5

	// FSOPT_NOFOLLOW If this bit is set, getattrlist() will not follow a
	// symlink if it occurs as the last component of path.
	FSOPT_NOFOLLOW      = uint32(0x00000001)
	FSOPT_NOINMEMUPDATE = uint32(0x00000002)
	// FSOPT_REPORT_FULLSIZE: The size of the attributes reported (in the first
	// u_int32_t field in the attribute buffer) will be the size needed to hold
	// all the requested attributes; if not set, only the attributes actu- ally
	// returned will be reported.  This allows the caller to determine if any
	// truncation occurred.
	FSOPT_REPORT_FULLSIZE = uint32(0x00000004)
	// FSOPT_PACK_INVAL_ATTRS: If this is bit is set, then all requested
	// attributes, even ones that are not supported by the object or file
	// system, will be returned. Default values will be used for the invalid
	// ones. Requires that ATTR_CMN_RETURNED_ATTRS be requested.
	FSOPT_PACK_INVAL_ATTRS = uint32(0x00000008)
	// FSOPT_ATTR_CMN_EXTENDED: If this is bit is set, then ATTR_CMN_GEN_COUNT
	// and ATTR_CMN_DOCUMENT_ID can be requested. When this option is used,
	// callers must not reference forkattrs anywhere.
	FSOPT_ATTR_CMN_EXTENDED = uint32(0x00000020)

	ATTR_CMN_NAME              = uint32(0x00000001)
	ATTR_CMN_DEVID             = uint32(0x00000002)
	ATTR_CMN_FSID              = uint32(0x00000004)
	ATTR_CMN_OBJTYPE           = uint32(0x00000008)
	ATTR_CMN_OBJTAG            = uint32(0x00000010)
	ATTR_CMN_OBJID             = uint32(0x00000020)
	ATTR_CMN_OBJPERMANENTID    = uint32(0x00000040)
	ATTR_CMN_PAROBJID          = uint32(0x00000080)
	ATTR_CMN_SCRIPT            = uint32(0x00000100)
	ATTR_CMN_CRTIME            = uint32(0x00000200)
	ATTR_CMN_MODTIME           = uint32(0x00000400)
	ATTR_CMN_CHGTIME           = uint32(0x00000800)
	ATTR_CMN_ACCTIME           = uint32(0x00001000)
	ATTR_CMN_BKUPTIME          = uint32(0x00002000)
	ATTR_CMN_FNDRINFO          = uint32(0x00004000)
	ATTR_CMN_OWNERID           = uint32(0x00008000)
	ATTR_CMN_GRPID             = uint32(0x00010000)
	ATTR_CMN_ACCESSMASK        = uint32(0x00020000)
	ATTR_CMN_FLAGS             = uint32(0x00040000)
	ATTR_CMN_USERACCESS        = uint32(0x00200000)
	ATTR_CMN_EXTENDED_SECURITY = uint32(0x00400000)
	ATTR_CMN_UUID              = uint32(0x00800000)
	ATTR_CMN_GRPUUID           = uint32(0x01000000)
	ATTR_CMN_FILEID            = uint32(0x02000000)
	ATTR_CMN_PARENTID          = uint32(0x04000000)
	ATTR_CMN_FULLPATH          = uint32(0x08000000)
	ATTR_CMN_ADDEDTIME         = uint32(0x10000000)
	ATTR_CMN_RETURNED_ATTRS    = uint32(0x80000000)
	ATTR_CMN_ALL_ATTRS         = uint32(0x9fe7ffff)

	ATTR_VOL_FSTYPE          = uint32(0x00000001)
	ATTR_VOL_SIGNATURE       = uint32(0x00000002)
	ATTR_VOL_SIZE            = uint32(0x00000004)
	ATTR_VOL_SPACEFREE       = uint32(0x00000008)
	ATTR_VOL_SPACEAVAIL      = uint32(0x00000010)
	ATTR_VOL_MINALLOCATION   = uint32(0x00000020)
	ATTR_VOL_ALLOCATIONCLUMP = uint32(0x00000040)
	ATTR_VOL_IOBLOCKSIZE     = uint32(0x00000080)
	ATTR_VOL_OBJCOUNT        = uint32(0x00000100)
	ATTR_VOL_FILECOUNT       = uint32(0x00000200)
	ATTR_VOL_DIRCOUNT        = uint32(0x00000400)
	ATTR_VOL_MAXOBJCOUNT     = uint32(0x00000800)
	ATTR_VOL_MOUNTPOINT      = uint32(0x00001000)
	ATTR_VOL_NAME            = uint32(0x00002000)
	ATTR_VOL_MOUNTFLAGS      = uint32(0x00004000)
	ATTR_VOL_MOUNTEDDEVICE   = uint32(0x00008000)
	ATTR_VOL_ENCODINGSUSED   = uint32(0x00010000)
	ATTR_VOL_CAPABILITIES    = uint32(0x00020000)
	ATTR_VOL_UUID            = uint32(0x00040000)
	ATTR_VOL_ATTRIBUTES      = uint32(0x40000000)
	ATTR_VOL_INFO            = uint32(0x80000000)
	ATTR_VOL_ALL_ATTRS       = uint32(0xc007ffff)

	ATTR_DIR_LINKCOUNT     = uint32(0x00000001)
	ATTR_DIR_ENTRYCOUNT    = uint32(0x00000002)
	ATTR_DIR_MOUNTSTATUS   = uint32(0x00000004)
	DIR_MNTSTATUS_MNTPOINT = uint32(0x00000001)
	DIR_MNTSTATUS_TRIGGER  = uint32(0x00000002)
	ATTR_DIR_ALL_ATTRS     = uint32(0x00000007)

	ATTR_FILE_LINKCOUNT     = uint32(0x00000001)
	ATTR_FILE_TOTALSIZE     = uint32(0x00000002)
	ATTR_FILE_ALLOCSIZE     = uint32(0x00000004)
	ATTR_FILE_IOBLOCKSIZE   = uint32(0x00000008)
	ATTR_FILE_DEVTYPE       = uint32(0x00000020)
	ATTR_FILE_DATALENGTH    = uint32(0x00000200)
	ATTR_FILE_DATAALLOCSIZE = uint32(0x00000400)
	ATTR_FILE_RSRCLENGTH    = uint32(0x00001000)
	ATTR_FILE_RSRCALLOCSIZE = uint32(0x00002000)

	ATTR_FILE_ALL_ATTRS = uint32(0x0000362f)

	ATTR_FORK_TOTALSIZE = uint32(0x00000001)
	ATTR_FORK_ALLOCSIZE = uint32(0x00000002)
	ATTR_FORK_ALL_ATTRS = uint32(0x00000003)
)

type AttrList struct {
	ReturnedAttributes *AttrSet
	CreationTime       *TimeSpec
	VolumeSize         int32
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
	Forkattr uint32
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
	Sec  int32
	Nsec int32
}

func (ts TimeSpec) String() string {
	return fmt.Sprintf("Sec: %d, Nsec: %d", ts.Sec, ts.Nsec)
}

// GetAttrList returns attributes (that is, metadata) of file system objects. GetAttrList()
// works on the file system object named by path. You can think of getattrlist() as a
// seriously enhanced version of syscall.Stat.  The functions return attributes about
// the specified file system object into the buffer specified by attrBuf and
// attrBufSize.  The attrList parameter determines what attributes are returned.
//
// https://developer.apple.com/legacy/library/documentation/Darwin/Reference/ManPages/man2/getattrlist.2.html
func GetAttrList(path string, mask AttrListMask, attrBuf []byte, options uint32) (results *AttrList, err error) {
	if len(attrBuf) < 4 {
		return nil, errors.New("attrBuf too small")
	}
	mask.bitmapCount = attrBitMapCount

	if mask.VolAttr > 0 {
		mask.VolAttr |= ATTR_VOL_INFO
	}
	options |= FSOPT_REPORT_FULLSIZE

	var _p0 *byte
	_p0, err = syscall.BytePtrFromString(path)
	if err != nil {
		return nil, err
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
		return nil, e1
	}
	results = &AttrList{}

	// binary.LittleEndian.Uint32(attrBuf)
	size := *(*uint32)(unsafe.Pointer(&attrBuf[0]))
	// dat is the section of attrBuf that contains valid data,
	// without the 4 byte length header. All attribute offsets
	// are relative to dat.
	dat := attrBuf
	if int(size) < len(attrBuf) {
		// fmt.Println("truncating", size, len(attrBuf))
		dat = dat[:size]
	}
	dat = dat[4:] // remove length prefix
	r := bytes.NewReader(dat)

	if mask.CommonAttr&ATTR_CMN_RETURNED_ATTRS > 0 {
		fmt.Println("ATTR_CMN_RETURNED_ATTRS")
		if err = binary.Read(r, binary.LittleEndian, &results.ReturnedAttributes); err != nil {
			return nil, fmt.Errorf("failed reading ATTR_CMN_RETURNED_ATTRS - %s", err)
		}
	}

	if mask.CommonAttr&ATTR_CMN_NAME > 0 {
		fmt.Println("ATTR_CMN_NAME")
		ref := AttrRef{}
		if err = binary.Read(r, binary.LittleEndian, &ref); err != nil {
			return nil, fmt.Errorf("failed reading ATTR_CMN_NAME ref - %s", err)
		}
		r.Seek(io.SeekCurrent, int(ref.Offset))
		name := make([]byte, ref.Len)
		r.Read(name)
	}

	if mask.CommonAttr&ATTR_CMN_DEVID > 0 {
		fmt.Println("ATTR_CMN_DEVID")
	}

	if mask.CommonAttr&ATTR_CMN_FSID > 0 {
		fmt.Println("ATTR_CMN_FSID")
	}

	if mask.CommonAttr&ATTR_CMN_OBJTYPE > 0 {
		fmt.Println("TTR_CMN_OBJTYPE")
	}

	if mask.CommonAttr&ATTR_CMN_OBJTYPE > 0 {
		fmt.Println("ATTR_CMN_OBJTYPE")
	}
	if mask.CommonAttr&ATTR_CMN_OBJTAG > 0 {
		fmt.Println("ATTR_CMN_OBJTAG")
	}
	if mask.CommonAttr&ATTR_CMN_OBJID > 0 {
		fmt.Println("ATTR_CMN_OBJID")
	}
	if mask.CommonAttr&ATTR_CMN_OBJPERMANENTID > 0 {
		fmt.Println("ATTR_CMN_OBJPERMANENTID")
	}
	if mask.CommonAttr&ATTR_CMN_PAROBJID > 0 {
		fmt.Println("ATTR_CMN_PAROBJID")
	}
	if mask.CommonAttr&ATTR_CMN_SCRIPT > 0 {
		fmt.Println("ATTR_CMN_SCRIPT")
	}
	if mask.CommonAttr&ATTR_CMN_CRTIME > 0 {
		results.CreationTime = &TimeSpec{}
		if err = binary.Read(r, binary.LittleEndian, &results.CreationTime.Sec); err != nil {
			return nil, fmt.Errorf("failed reading TTR_CMN_CRTIME sec - %s", err)
		}
		if err = binary.Read(r, binary.LittleEndian, &results.CreationTime.Nsec); err != nil {
			return nil, fmt.Errorf("failed reading TTR_CMN_CRTIME nsec - %s", err)
		}
	}
	if mask.CommonAttr&ATTR_CMN_MODTIME > 0 {
		fmt.Println("ATTR_CMN_MODTIME")
	}
	if mask.CommonAttr&ATTR_CMN_CHGTIME > 0 {
		fmt.Println("ATTR_CMN_CHGTIME")
	}
	if mask.CommonAttr&ATTR_CMN_ACCTIME > 0 {
		fmt.Println("ATTR_CMN_ACCTIME")
	}
	if mask.CommonAttr&ATTR_CMN_BKUPTIME > 0 {
		fmt.Println("ATTR_CMN_BKUPTIME")
	}
	if mask.CommonAttr&ATTR_CMN_FNDRINFO > 0 {
		fmt.Println("ATTR_CMN_FNDRINFO")
	}
	if mask.CommonAttr&ATTR_CMN_OWNERID > 0 {
		fmt.Println("ATTR_CMN_OWNERID")
	}
	if mask.CommonAttr&ATTR_CMN_GRPID > 0 {
		fmt.Println("ATTR_CMN_GRPID")
	}
	if mask.CommonAttr&ATTR_CMN_ACCESSMASK > 0 {
		fmt.Println("ATTR_CMN_ACCESSMASK")
	}
	if mask.CommonAttr&ATTR_CMN_FLAGS > 0 {
		fmt.Println("ATTR_CMN_FLAGS")
	}
	if mask.CommonAttr&ATTR_CMN_USERACCESS > 0 {
		fmt.Println("ATTR_CMN_USERACCESS")
	}
	if mask.CommonAttr&ATTR_CMN_EXTENDED_SECURITY > 0 {
		fmt.Println("ATTR_CMN_EXTENDED_SECURITY")
	}

	// Volume attributes
	if mask.VolAttr&ATTR_VOL_FSTYPE > 0 {
		fmt.Println("ATTR_VOL_FSTYPE")
	}
	if mask.VolAttr&ATTR_VOL_SIGNATURE > 0 {
		fmt.Println("ATTR_VOL_SIGNATURE")
	}
	if mask.VolAttr&ATTR_VOL_SIZE > 0 {
		if err = binary.Read(r, binary.LittleEndian, &results.VolumeSize); err != nil {
			return results, fmt.Errorf("failed to read volume size - %s", err)
		}
	}
	if mask.VolAttr&ATTR_VOL_SPACEFREE > 0 {
		fmt.Println("ATTR_VOL_SPACEFREE")
	}
	if mask.VolAttr&ATTR_VOL_SPACEAVAIL > 0 {
		fmt.Println("ATTR_VOL_SPACEAVAIL")
	}
	if mask.VolAttr&ATTR_VOL_MINALLOCATION > 0 {
		fmt.Println("ATTR_VOL_MINALLOCATION")
	}
	if mask.VolAttr&ATTR_VOL_ALLOCATIONCLUMP > 0 {
		fmt.Println("ATTR_VOL_ALLOCATIONCLUMP")
	}
	if mask.VolAttr&ATTR_VOL_IOBLOCKSIZE > 0 {
		fmt.Println("ATTR_VOL_IOBLOCKSIZE")
	}
	if mask.VolAttr&ATTR_VOL_OBJCOUNT > 0 {
		fmt.Println("ATTR_VOL_OBJCOUNT")
	}
	if mask.VolAttr&ATTR_VOL_FILECOUNT > 0 {
		fmt.Println("ATTR_VOL_FILECOUNT")
	}
	if mask.VolAttr&ATTR_VOL_DIRCOUNT > 0 {
		fmt.Println("ATTR_VOL_DIRCOUNT")
	}
	if mask.VolAttr&ATTR_VOL_MAXOBJCOUNT > 0 {
		fmt.Println("ATTR_VOL_MAXOBJCOUNT")
	}
	if mask.VolAttr&ATTR_VOL_MOUNTPOINT > 0 {
		fmt.Println("ATTR_VOL_MOUNTPOINT")
	}
	if mask.VolAttr&ATTR_VOL_NAME > 0 {
		fmt.Println("ATTR_VOL_NAME")
	}
	if mask.VolAttr&ATTR_VOL_MOUNTFLAGS > 0 {
		fmt.Println("ATTR_VOL_MOUNTFLAGS")
	}
	if mask.VolAttr&ATTR_VOL_MOUNTEDDEVICE > 0 {
		fmt.Println("ATTR_VOL_MOUNTEDDEVICE")
	}
	if mask.VolAttr&ATTR_VOL_ENCODINGSUSED > 0 {
		fmt.Println("ATTR_VOL_ENCODINGSUSED")
	}
	if mask.VolAttr&ATTR_VOL_CAPABILITIES > 0 {
		fmt.Println("ATTR_VOL_CAPABILITIES")
	}
	if mask.VolAttr&ATTR_VOL_UUID > 0 {
		fmt.Println("ATTR_VOL_UUID")
	}
	if mask.VolAttr&ATTR_VOL_ATTRIBUTES > 0 {
		fmt.Println("ATTR_VOL_ATTRIBUTES")
	}

	return
}

/*

var (
	attr_info = (
    # Common attributes
    (0, ATTR_CMN_RETURNED_ATTRS, sizeof(attribute_set_t)),
    (0, ATTR_CMN_NAME, sizeof(attrreference_t) + NAME_MAX * 3 + 1),
    (0, ATTR_CMN_DEVID, sizeof(dev_t)),
    (0, ATTR_CMN_FSID, sizeof(fsid_t)),
    (0, ATTR_CMN_OBJTYPE, sizeof(fsobj_type_t)),
    (0, ATTR_CMN_OBJTAG, sizeof(fsobj_tag_t)),
    (0, ATTR_CMN_OBJPERMANENTID, sizeof(fsobj_id_t)),
    (0, ATTR_CMN_PAROBJID, sizeof(fsobj_id_t)),
    (0, ATTR_CMN_SCRIPT, sizeof(text_encoding_t)),
    (0, ATTR_CMN_CRTIME, sizeof(timespec)),
    (0, ATTR_CMN_MODTIME, sizeof(timespec)),
    (0, ATTR_CMN_CHGTIME, sizeof(timespec)),
    (0, ATTR_CMN_ACCTIME, sizeof(timespec)),
    (0, ATTR_CMN_BKUPTIME, sizeof(timespec)),
    (0, ATTR_CMN_FNDRINFO, sizeof(FinderInfo)),
    (0, ATTR_CMN_OWNERID, sizeof(uid_t)),
    (0, ATTR_CMN_GRPID, sizeof(gid_t)),
    (0, ATTR_CMN_ACCESSMASK, sizeof(uint32_t)),
    (0, ATTR_CMN_NAMEDATTRCOUNT, None),
    (0, ATTR_CMN_NAMEDATTRLIST, None),
    (0, ATTR_CMN_FLAGS, sizeof(uint32_t)),
    (0, ATTR_CMN_USERACCESS, sizeof(uint32_t)),
    (0, ATTR_CMN_EXTENDED_SECURITY, sizeof(attrreference_t) + sizeof(kauth_filesec)),
    (0, ATTR_CMN_UUID, sizeof(guid_t)),
    (0, ATTR_CMN_GRPUUID, sizeof(guid_t)),
    (0, ATTR_CMN_FILEID, sizeof(uint64_t)),
    (0, ATTR_CMN_PARENTID, sizeof(uint64_t)),
    (0, ATTR_CMN_FULLPATH, sizeof(attrreference_t) + PATH_MAX),
    (0, ATTR_CMN_ADDEDTIME, sizeof(timespec)),

    # Volume attributes
    (1, ATTR_VOL_FSTYPE, sizeof(uint32_t)),
    (1, ATTR_VOL_SIGNATURE, sizeof(uint32_t)),
    (1, ATTR_VOL_SIZE, sizeof(off_t)),
    (1, ATTR_VOL_SPACEFREE, sizeof(off_t)),
    (1, ATTR_VOL_SPACEAVAIL, sizeof(off_t)),
    (1, ATTR_VOL_MINALLOCATION, sizeof(off_t)),
    (1, ATTR_VOL_ALLOCATIONCLUMP, sizeof(off_t)),
    (1, ATTR_VOL_IOBLOCKSIZE, sizeof(uint32_t)),
    (1, ATTR_VOL_OBJCOUNT, sizeof(uint32_t)),
    (1, ATTR_VOL_FILECOUNT, sizeof(uint32_t)),
    (1, ATTR_VOL_DIRCOUNT, sizeof(uint32_t)),
    (1, ATTR_VOL_MAXOBJCOUNT, sizeof(uint32_t)),
    (1, ATTR_VOL_MOUNTPOINT, sizeof(attrreference_t) + PATH_MAX),
    (1, ATTR_VOL_NAME, sizeof(attrreference_t) + NAME_MAX + 1),
    (1, ATTR_VOL_MOUNTFLAGS, sizeof(uint32_t)),
    (1, ATTR_VOL_MOUNTEDDEVICE, sizeof(attrreference_t) + PATH_MAX),
    (1, ATTR_VOL_ENCODINGSUSED, sizeof(c_ulonglong)),
    (1, ATTR_VOL_CAPABILITIES, sizeof(vol_capabilities_attr_t)),
    (1, ATTR_VOL_UUID, sizeof(uuid_t)),
    (1, ATTR_VOL_ATTRIBUTES, sizeof(vol_attributes_attr_t)),

    # Directory attributes
    (2, ATTR_DIR_LINKCOUNT, sizeof(uint32_t)),
    (2, ATTR_DIR_ENTRYCOUNT, sizeof(uint32_t)),
    (2, ATTR_DIR_MOUNTSTATUS, sizeof(uint32_t)),

    # File attributes
    (3, ATTR_FILE_LINKCOUNT, sizeof(uint32_t)),
    (3, ATTR_FILE_TOTALSIZE, sizeof(off_t)),
    (3, ATTR_FILE_ALLOCSIZE, sizeof(off_t)),
    (3, ATTR_FILE_IOBLOCKSIZE, sizeof(uint32_t)),
    (3, ATTR_FILE_CLUMPSIZE, sizeof(uint32_t)),
    (3, ATTR_FILE_DEVTYPE, sizeof(uint32_t)),
    (3, ATTR_FILE_FILETYPE, sizeof(uint32_t)),
    (3, ATTR_FILE_FORKCOUNT, sizeof(uint32_t)),
    (3, ATTR_FILE_FORKLIST, None),
    (3, ATTR_FILE_DATALENGTH, sizeof(off_t)),
    (3, ATTR_FILE_DATAALLOCSIZE, sizeof(off_t)),
    (3, ATTR_FILE_DATAEXTENTS, sizeof(extentrecord)),
    (3, ATTR_FILE_RSRCLENGTH, sizeof(off_t)),
    (3, ATTR_FILE_RSRCALLOCSIZE, sizeof(off_t)),
    (3, ATTR_FILE_RSRCEXTENTS, sizeof(extentrecord)),

    # Fork attributes
    (4, ATTR_FORK_TOTALSIZE, sizeof(off_t)),
    (4, ATTR_FORK_ALLOCSIZE, sizeof(off_t))
	)
*/
