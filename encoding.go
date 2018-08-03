package cocoa

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/mattetti/cocoa/darwin"
)

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
