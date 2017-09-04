// +build darwin

package cocoa

import (
	"fmt"
	"io"
)

// AliasFromReader takes an io.reader pointing to an alias file
// decodes it and returns the contained bookkmak data.
func AliasFromReader(r io.Reader) (*BookmarkData, error) {
	d, err := newBookmarkDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read source - %s", err)
	}
	if err := d.aliasHeader(); err != nil {
		return nil, err
	}
	d.read(&d.tocOffset)
	// jump to toc
	d.seek(int64(d.tocOffset)-4, io.SeekCurrent)
	if err := d.toc(); err != nil {
		return nil, fmt.Errorf("failed to read the TOC - %s", err)
	}

	// we now need to use the oMap to extract the data
	// TODO: read all the keys
	for key, offset := range d.oMap {
		switch key {
		case KBookmarkPath:
			// path
			d.seek(int64(offset), io.SeekStart)
			d.b.Path, err = d.decodeStringSlice()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file path - %s", err)
				return nil, d.err
			}
		case KBookmarkCNIDPath:
			// CNID path
			d.seek(int64(offset), io.SeekStart)
			d.b.CNIDPath, err = d.decodeUint32Slice()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the CNID path - %s", err)
				return nil, d.err
			}
		case KBookmarkFileProperties:
			d.seek(int64(offset), io.SeekStart)
			d.b.FileProperties, err = d.decodeBytes()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file properties - %s", err)
				return nil, d.err
			}
		case KBookmarkFileCreationDate:
			d.seek(int64(offset), io.SeekStart)
			d.b.FileCreationDate, err = d.decodeTime()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file creation date - %s", err)
			}
		case KBookmarkFileID:
			d.seek(int64(offset), io.SeekStart)
			d.b.CNID, err = d.decodeUint32()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file CNID - %s", err)
			}
		default:
			fmt.Printf("%#x not parsed\n", key)
		}
	}

	return d.b, d.err
}
