package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

var (
	aliasEpoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
)

// AliasRecord format documented by Alastair Houghton
// http://mac-alias.readthedocs.io/en/latest/alias_fmt.html

// AliasRecord is an alias representation that can be shared in memory
// For file persistency, see the Alias with bookmark data.
type AliasRecord struct {
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
	VolumeID int16
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
	e.write(e.record.AppCode[:])
	// record size, will need to come back to that
	e.add(uint16(0))
	// version
	e.add(uint16(2))
	// alias kind
	e.add(e.record)
	e.write(e.pascalString(e.record.VolumeName))
	e.add(e.dateInSecs(e.record.VolumeDate))
	e.write([]byte(e.record.FileSystem))
	e.add(e.record.DiskType)
	e.add(e.record.FolderCNID)
	e.add(e.pascalString(e.record.TargetName))
	e.add(e.record.TargetCNID)
	e.add(e.dateInSecs(e.record.TargetCreation))
	e.write(e.record.TargetCreator[:])
	e.write(e.record.TargetType[:])
	// Number of directory levels from alias to root
	e.add(int32(-1))
	// Number of directory levels from root to target (or -1)
	e.add(int32(-1))
	e.write(e.record.VolumeAttributes[:])
	e.add(e.record.VolumeID)
	e.write(make([]byte, 10))

	return e.buf.Bytes(), e.err
}

func (e *aliasRecordEncoder) write(data []byte) {
	_, err := e.buf.Write(data)
	e.setError(err)
}

func (e *aliasRecordEncoder) add(src interface{}) {
	e.setError(binary.Write(e.buf, binary.BigEndian, src))
}

func (e *aliasRecordEncoder) pascalString(str string) []byte {
	return append([]byte{byte(uint8(len(str)))}, []byte(str)...)
}

func (e *aliasRecordEncoder) dateInSecs(t time.Time) uint32 {
	return uint32(t.Sub(aliasEpoch).Seconds())
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
