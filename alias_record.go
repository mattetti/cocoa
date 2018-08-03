package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

var (
	aliasEpoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)

	aliasTagCarbonFolderName          uint16 = 0
	aliasTagCnidPath                  uint16 = 1
	aliasTagCarbonPath                uint16 = 2
	aliasTagAppleshareZone            uint16 = 3
	aliasTagAppleshareServerName      uint16 = 4
	aliasTagAppleshareUsername        uint16 = 5
	aliasTagDriverName                uint16 = 6
	aliasTagNetworkMountInfo          uint16 = 9
	aliasTagDialupInfo                uint16 = 10
	aliasTagUnicodeFilename           uint16 = 14
	aliasTagUnicodeVolumeName         uint16 = 15
	aliasTagHighResVolumeCreationDate uint16 = 16
	aliasTagHighResCreationDate       uint16 = 17
	aliasTagPosixPath                 uint16 = 18
	aliasTagPosixPathToMountpoint     uint16 = 19
	aliasTagRecursiveAliasOfDiskImage uint16 = 20
	aliasTagUserHomeLengthPrefix      uint16 = 21
)

const (
	AliasKindFile   = 0
	AliasKindFolder = 1
)

// AliasRecord format documented by Alastair Houghton
// http://mac-alias.readthedocs.io/en/latest/alias_fmt.html

// AliasRecord is an alias representation that can be shared in memory
// For file persistency, see the Alias with bookmark data.
type AliasRecord struct {
	Path      string
	CNIDPath  []uint32
	PathItems []string
	// Application specific four-character code
	AppCode [4]byte
	// Version (only 2 is supported)
	Version uint16
	// Alias kind (0 = file, 1 = folder)
	Kind uint16
	// Volume name (encoded as Pascal style)
	VolumeName string
	// Volume date (encoded as seconds since 1904-01-01 00:00:00 UTC)
	VolumeDate time.Time
	// Filesystem type (typically ‘H+’ for HFS+)
	FileSystem string
	// Disk type (0 = fixed, 1 = network, 2 = 400Kb, 3 = 800kb, 4 = 1.44MB, 5 = ejectable)
	DiskType uint16
	// CNID of containing folder
	FolderCNID uint32
	// Target name (encoded as Pascal-style string)
	TargetName string
	// Target CNID
	TargetCNID uint32
	// Target creation date (encoded as seconds since 1904-01-01 00:00:00 UTC)
	TargetCreation time.Time
	// Target creator code (four-character code)
	TargetCreator [4]byte
	// Target type code (four-character code)
	TargetType [4]byte
	// Number of directory levels from alias to root (or -1)
	DirsAliasToRoot int16
	// Number of directory levels from root to target (or -1)
	DirsRootToTarget int16
	// Volume attributes
	VolumeAttributes [4]byte
	// Volume filesystem ID
	VolumeID uint16
}

// Encode converts the AliasRecord into binary data and returns the byte data
func (a *AliasRecord) Encode() ([]byte, error) {
	coder := &aliasRecordEncoder{record: a}
	return coder.encode()
}

type aliasRecordEncoder struct {
	record *AliasRecord
	buf    *bytes.Buffer
	err    error
}

func (e *aliasRecordEncoder) encode() ([]byte, error) {
	if e == nil || e.record == nil {
		return nil, fmt.Errorf("nil alias record or encoder")
	}
	e.buf = &bytes.Buffer{}
	e.write(e.record.AppCode[:]) // 4 bytes
	// record size, will need to come back to that
	e.add(uint16(0))
	// version
	e.add(uint16(2))
	// alias kind
	e.add(e.record.Kind)

	e.write(e.pascalString(e.carbonize(e.record.VolumeName), 28))
	e.add(uint32(e.dateInSecs(e.record.VolumeDate)))

	e.write([]byte(e.record.FileSystem))
	e.add(e.record.DiskType)

	e.add(e.record.FolderCNID)

	e.add(e.pascalString(e.carbonize(e.record.TargetName), 64))
	e.add(e.record.TargetCNID)
	e.add(e.dateInSecs(e.record.TargetCreation))
	e.write(e.record.TargetCreator[:])
	e.write(e.record.TargetType[:])
	// Number of directory levels from alias to root
	e.add(int16(-1))
	// Number of directory levels from root to target (or -1)
	e.add(int16(-1))
	// attributes flags
	e.write(e.record.VolumeAttributes[:])
	e.add(e.record.VolumeID)
	e.write(make([]byte, 10))

	e.folderNameTag()
	e.dateTags()

	e.cnidPathTag()
	e.carbonPathTag()
	e.filenameTag()
	e.volumeNameTag()
	e.posixPathTag()
	e.add(int16(-1))
	e.add(uint16(0))

	// go back to the record size and set it up
	data := e.buf.Bytes()
	binary.BigEndian.PutUint16(data[4:], uint16(len(data)))
	return e.buf.Bytes(), e.err
}

func (e *aliasRecordEncoder) write(data []byte) {
	_, err := e.buf.Write(data)
	e.setError(err)
}

func (e *aliasRecordEncoder) add(src interface{}) {
	e.setError(binary.Write(e.buf, binary.BigEndian, src))
}

func (e *aliasRecordEncoder) pascalString(str string, size int) []byte {
	data := append([]byte{byte(uint8(len(str)))}, []byte(str)...)
	if extra := size - len(data); extra > 0 {
		data = append(data, make([]byte, extra)...)
	}
	return data
}

func (e *aliasRecordEncoder) dateInSecs(t time.Time) uint32 {
	return uint32(t.Sub(aliasEpoch).Seconds())
}

func (e *aliasRecordEncoder) folderName() string {
	return filepath.Base(filepath.Dir(e.record.Path))
}

func (e *aliasRecordEncoder) folderNameTag() {
	e.add(aliasTagCarbonFolderName)
	carbonFoldName := e.carbonize(e.folderName())
	length := uint16(len(carbonFoldName))
	e.add(uint16(length))
	e.write([]byte(carbonFoldName))
	// optional padding
	if length&1 > 0 {
		e.write([]byte{0})
	}
}

func (e *aliasRecordEncoder) carbonPathTag() {
	e.add(aliasTagCarbonPath)
	fullPath := filepath.Join(e.record.PathItems...)
	carbonPath := strings.Join([]string{e.carbonize(e.record.VolumeName), e.carbonize(fullPath)}, ":")
	length := uint16(len(carbonPath))
	e.add(uint16(length))
	e.write([]byte(carbonPath))
	// optional padding
	if length&1 > 0 {
		e.write([]byte{0})
	}
}

func (e *aliasRecordEncoder) posixPathTag() {
	e.add(aliasTagPosixPath)
	posixPath := strings.Join(e.record.PathItems, "/")
	length := uint16(len(posixPath))
	e.add(uint16(length))
	e.write([]byte(posixPath))
	// optional padding
	if length&1 > 0 {
		e.write([]byte{0})
	}
	//
	e.add(aliasTagPosixPathToMountpoint)
	e.add(uint16(1))
	e.write([]byte("/"))
	e.write([]byte{0})
}

func (e *aliasRecordEncoder) filenameTag() {
	e.add(aliasTagUnicodeFilename)
	utf16Filename := utf16.Encode([]rune(e.carbonize(e.record.TargetName)))
	e.add(uint16(len(utf16Filename)*2) + 2)
	e.add(uint16(len(utf16Filename)*2) / 2)
	e.add(utf16Filename)
}

func (e *aliasRecordEncoder) volumeNameTag() {
	e.add(aliasTagUnicodeVolumeName)
	utf16Filename := utf16.Encode([]rune(e.carbonize(e.record.VolumeName)))
	e.add(uint16(len(utf16Filename)*2) + 2)
	e.add(uint16(len(utf16Filename)*2) / 2)
	e.add(utf16Filename)
}

func (e *aliasRecordEncoder) dateTags() {
	e.add(aliasTagHighResVolumeCreationDate)
	e.add(uint16(8))
	e.add(uint64(e.dateInSecs(e.record.VolumeDate)) * 65536)
	e.add(aliasTagHighResCreationDate)
	e.add(uint16(8))
	e.add(uint64(e.dateInSecs(e.record.TargetCreation) * 65536))
}

func (e *aliasRecordEncoder) cnidPathTag() {
	e.add(aliasTagCnidPath)
	// length in bytes
	e.add(uint16(len(e.record.CNIDPath) * 4))
	for _, cnid := range e.record.CNIDPath {
		e.add(cnid)
	}
}

func (e *aliasRecordEncoder) carbonize(str string) string {
	return strings.Replace(str, "/", string([]byte{':', 0x0}), -1)
}

func (e *aliasRecordEncoder) setError(err error) error {
	if err == nil {
		return nil
	}
	if e.err != nil {
		e.err = fmt.Errorf("%v - %v", e.err, err)
	}
	return e.err
}
