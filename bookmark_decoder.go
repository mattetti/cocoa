package cocoa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/mattetti/cocoa/darwin"
)

func newBookmarkDecoder(r io.Reader) (*bookmarkDecoder, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return &bookmarkDecoder{
		r: bytes.NewReader(data),
		b: &BookmarkData{},
	}, nil
}

type bookmarkDecoder struct {
	r          *bytes.Reader
	pos        int64
	err        error
	b          *BookmarkData
	headerSize uint32
	bodySize   uint32
	tocOffset  uint32
	oMap       offsetMap
}

// bookmark headers use a slightly different structure.
// TODO: add bookmarkHeader()
func (d *bookmarkDecoder) aliasHeader() error {
	buf := make([]byte, 4)
	d.read(&buf)
	if string(buf) != "book" {
		return fmt.Errorf("invalid bookmark file - bad header")
	}

	d.seek(4, io.SeekCurrent)
	d.read(&buf)
	if string(buf) != "mark" {
		return fmt.Errorf("invalid bookmark file - bad header")
	}
	d.seek(4, io.SeekCurrent)
	// size of the header
	d.read(&d.headerSize)
	d.seek(4, io.SeekCurrent) // another version of the size of the header
	d.read(&d.bodySize)
	d.seek(28, io.SeekCurrent)
	if d.pos != int64(d.headerSize) {
		return fmt.Errorf("header size didn't match expectations, at %d - %d", d.pos, d.headerSize)
	}
	return d.err
}

func (d *bookmarkDecoder) toc() error {
	// Size of TOC in bytes, minus 8
	var tocSize uint32
	d.read(&tocSize)
	// magic number
	tmp := make([]byte, 4)
	d.read(&tmp)
	if bytes.Compare(tmp, []byte{0xFE, 0xFF, 0xFF, 0xFF}) != 0 {
		return fmt.Errorf("bad TOC")
	}
	// skip
	d.seek(4+4, io.SeekCurrent)
	// identifier uint32(1)
	// Next TOC offset (or uint32(0) if none)
	// Number of entries in this TOC
	var nItems uint32
	d.read(&nItems)
	d.oMap = offsetMap{}
	var key uint32
	var offset uint32
	for i := uint32(0); i < nItems; i++ {
		// key uint32
		d.read(&key)
		// offset uint32
		d.read(&offset)
		// blank
		d.seek(4, io.SeekCurrent)
		d.oMap[key] = int(offset + d.headerSize) // set absolute position
	}

	return d.err
}

func (d *bookmarkDecoder) decodeStringSlice() ([]string, error) {
	var err error
	var size uint32
	var typeMask uint32
	d.read(&size)
	d.read(&typeMask)
	dType := typeMask & bmk_data_type_mask
	// dSubType := typeMask & bmk_data_subtype_mask

	if dType != bmk_array {
		return nil, fmt.Errorf("unexpected array type, expected %d got %d", bmk_array, dType)
	}

	nItems := size / 4
	offsets := make([]uint32, nItems)
	s := make([]string, nItems)
	for i := uint32(0); i < nItems; i++ {
		d.read(&offsets[i])
	}

	for i, offset := range offsets {
		d.seek(int64(d.headerSize+offset), io.SeekStart)
		s[i], err = d.decodeString()
		if err != nil {
			return s, err
		}
	}

	return s, nil
}

func (d *bookmarkDecoder) decodeUint32Slice() ([]uint32, error) {
	var size uint32
	var typeMask uint32
	d.read(&size)
	d.read(&typeMask)
	dType := typeMask & bmk_data_type_mask
	// dSubType := typeMask & bmk_data_subtype_mask

	if dType != bmk_array {
		return nil, fmt.Errorf("unexpected array type, expected %d got %d", bmk_array, dType)
	}

	nItems := size / 4
	items := make([]uint32, nItems)
	for i := uint32(0); i < nItems; i++ {
		d.read(&items[i])
	}
	return items, d.err
}

func (d *bookmarkDecoder) decodeUint32() (uint32, error) {
	var len uint32
	var typeMask uint32
	d.read(&len)
	d.read(&typeMask)
	dType := typeMask & bmk_data_type_mask
	// dSubType := typeMask & bmk_data_subtype_mask

	if dType != bmk_number {
		return 0, fmt.Errorf("unexpected number type, expected %d got %d", bmk_number, dType)
	}
	var n uint32
	d.read(&n)
	return n, d.err
}

func (d *bookmarkDecoder) decodeString() (string, error) {
	var len uint32
	var typeMask uint32
	d.read(&len)
	d.read(&typeMask)
	dType := typeMask & bmk_data_type_mask
	if dType != bmk_string {
		return "", fmt.Errorf("unexpected string type, expected %d got %d", bmk_string, dType)
	}
	strB := make([]byte, len)
	d.read(&strB)
	return string(strB), nil
}

func (d *bookmarkDecoder) decodeBytes() ([]byte, error) {
	var len uint32
	var typeMask uint32
	d.read(&len)
	d.read(&typeMask)
	dType := typeMask & bmk_data_type_mask
	if dType != bmk_data {
		return nil, fmt.Errorf("unexpected byte type, expected %d got %d", bmk_data, dType)
	}
	data := make([]byte, len)
	d.read(&data)
	return data, d.err
}

func (d *bookmarkDecoder) decodeTime() (time.Time, error) {
	var len uint32
	var typeMask uint32
	d.read(&len)
	d.read(&typeMask)
	dType := typeMask & bmk_data_type_mask
	if dType != bmk_date {
		return time.Time{}, fmt.Errorf("unexpected date type, expected %d got %d", bmk_date, dType)
	}
	var secs float64
	d.readBE(&secs)
	return darwin.Epoch.Add(time.Duration(int64(secs)) * time.Second), d.err
}

func (d *bookmarkDecoder) seek(offset int64, whence int) {
	var err error
	d.pos, err = d.r.Seek(offset, whence)
	d.setError(err)
}

func (d *bookmarkDecoder) read(dst interface{}) {
	if d.err != nil {
		return
	}
	d.pos += int64(binary.Size(dst))
	d.setError(binary.Read(d.r, binary.LittleEndian, dst))
}

func (d *bookmarkDecoder) readBE(dst interface{}) {
	if d.err != nil {
		return
	}
	d.pos += int64(binary.Size(dst))
	d.setError(binary.Read(d.r, binary.BigEndian, dst))
}

func (d *bookmarkDecoder) setError(e error) {
	if e == nil {
		return
	}

	if d.err == nil {
		d.err = e
		if d.err == io.EOF {
			d.err = io.ErrUnexpectedEOF
		}
	}
}
