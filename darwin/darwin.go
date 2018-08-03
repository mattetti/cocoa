// daring is a package implementing low level features exposed in Cocoa but that
// can also be used directly in pure Go via syscalls.
package darwin

import (
	"encoding/hex"
	"fmt"
	"syscall"
	"time"
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
	FileInfo           FileInfo
	FolderInfo         FolderInfo
	UUID               [16]byte
	DevID              uint32
}

// StringVolUUID returns a string formatted version of the volume UUID
func (attr *AttrList) StringVolUUID() string {
	return toUUIDString(attr.VolUUID)
}

// IsFolder indicates if the attribute list is a folder.
// ATTR_CMN_OBJTYPE must have been ask as a common attribute to check this flag.
func (attr *AttrList) IsFolder() bool {
	return attr.ObjType == VDIR
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

type Point struct {
	X int16
	Y int16
}

type Rect struct {
	X int16
	Y int16
	W int16
	H int16
}

// FileInfo structure (32 bytes)
// See https://opensource.apple.com/source/CarbonHeaders/CarbonHeaders-9A581/Finder.h
type FileInfo struct {
	FileType            uint32
	FileCreator         uint32
	FinderFlags         uint16
	Location            Point
	ReservedField       uint16
	Reserved1           [4]int16
	ExtendedFinderFlags uint16
	Reserved2           int16
	PutAwayFolderID     int32
}

type FolderInfo struct {
	WindowBounds        Rect
	FinderFlags         uint16
	Location            Point
	ReservedField       uint16
	ScrollPosition      Point
	Reserved1           int32
	ExtendedFinderFlags uint16
	Reserved2           int16
	PutAwayFolderID     int32
}

type TimeSpec syscall.Timespec

func (ts TimeSpec) String() string {
	return fmt.Sprintf("Sec: %d, Nsec: %d", ts.Sec, ts.Nsec)
}

// Time returns a unix time representation of the time spec
func (ts TimeSpec) Time() time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}

func (ts TimeSpec) DarwinDuration() time.Duration {
	return ts.Time().Sub(Epoch)
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
