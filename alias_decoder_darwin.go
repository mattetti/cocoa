package cocoa

import (
	"fmt"
	"io"
	"os"
)

// AliasFromReader takes an io.reader pointing to an alias file
// decodes it and returns the contained bookmark data.
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
			if Debug {
				fmt.Println("Parsing path at offset", offset)
			}
			// path
			d.seek(int64(offset), io.SeekStart)
			d.b.Path, err = d.decodeStringSlice()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file path - %s", err)
				return d.b, d.err
			}
		case KBookmarkCNIDPath:
			if Debug {
				fmt.Println("Parsing CNID path at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			offsets, err := d.decodeUint32Slice()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the CNID path offsets - %s", err)
				return d.b, d.err
			}
			d.b.CNIDPath = make([]uint64, len(offsets))
			var inode int64
			for i, offset := range offsets {
				d.seek(int64(d.headerSize+offset), io.SeekStart)
				inode, err = d.decodeInt64()
				if err != nil {
					return d.b, fmt.Errorf("failed to read the %d CNID path in array - %v", i, err)
				}
				d.b.CNIDPath[i] = uint64(inode)
			}

		case KBookmarkVolumeProperties:
			if Debug {
				fmt.Println("Parsing volume properties at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumeProperties, err = d.decodeBytes()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume properties - %s", err)
				return d.b, d.err
			}
		case KBookmarkFileProperties:
			if Debug {
				fmt.Println("Parsing file properties at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.FileProperties, err = d.decodeBytes()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file properties - %s", err)
				return d.b, d.err
			}
		case KBookmarkContainingFolder:
			if Debug {
				fmt.Println("Parsing containing folder index at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.ContainingFolderIDX, err = d.decodeUint32()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the containing folder IDX - %s", err)
				return d.b, d.err
			}
		case KBookmarkCreationOptions:
			if Debug {
				fmt.Println("Parsing creation options at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.CreationOptions, err = d.decodeUint32()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the creation options - %s", err)
				return d.b, d.err
			}
		case KBookmarkFileCreationDate:
			if Debug {
				fmt.Println("Parsing file creation date at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.FileCreationDate, err = d.decodeTime()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file creation date - %s", err)
				return d.b, d.err
			}
		case KBookmarkFileID:
			if Debug {
				fmt.Println("Parsing file id at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.CNID, err = d.decodeUint32()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file CNID - %s", err)
				return d.b, d.err
			}
		case KBookmarkVolumeURL:
			if Debug {
				fmt.Println("Parsing volume URL at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			var length uint32
			d.read(&length)
			// volume type flags
			d.seek(4, io.SeekCurrent)
			volPathB := make([]byte, length)
			d.read(&volPathB)
			if d.err != nil {
				d.err = fmt.Errorf("failed to decode the volume url - %s", err)
				continue
			}
			d.b.VolumeURL = string(volPathB)
		case KBookmarkVolumeName:
			if Debug {
				fmt.Println("Parsing volume name at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumeName, err = d.decodeString()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume name - %s", err)
				return d.b, d.err
			}
		case KBookmarkVolumePath:
			if Debug {
				fmt.Println("Parsing volume path at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumePath, err = d.decodeString()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume path - %s", err)
				return d.b, d.err
			}
		case KBookmarkFullFileName:
			if Debug {
				fmt.Println("Parsing filename at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.Filename, err = d.decodeString()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the full filename - %s", err)
				return d.b, d.err
			}
		case KBookmarkUserName:
			if Debug {
				fmt.Println("Parsing username at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.UserName, err = d.decodeString()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the user name - %s", err)
				return d.b, d.err
			}
		case KBookmarkVolumeSize:
			if Debug {
				fmt.Println("Parsing volume size at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumeSize, err = d.decodeInt64()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume size - %s", err)
				return d.b, d.err
			}
		case KBookmarkUID:
			if Debug {
				fmt.Println("Parsing UID at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.UID, err = d.decodeUint32()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the UID - %s", err)
				return d.b, d.err
			}
		case KBookmarkVolumeUUID:
			if Debug {
				fmt.Println("Parsing volume UUID at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumeUUID, err = d.decodeString()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume uuid - %s", err)
				return d.b, d.err
			}
		case KBookmarkVolumeCreationDate:
			if Debug {
				fmt.Println("Parsing creation date at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumeCreationDate, err = d.decodeTime()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume creation date - %s", err)
				return d.b, d.err
			}
		case KBookmarkVolumeIsRoot:
			if Debug {
				fmt.Println("Parsing volume root status at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.VolumeIsRoot, err = d.decodeBool()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the volume root status - %s", err)
				return d.b, d.err
			}
		case KBookmarkWasFileReference:
			if Debug {
				fmt.Println("Parsing file reference at offset", offset)
			}
			d.seek(int64(offset), io.SeekStart)
			d.b.WasFileReference, err = d.decodeBool()
			if err != nil {
				d.err = fmt.Errorf("failed to decode the file reference status - %s", err)
				return d.b, d.err
			}
		default:
			if Debug {
				fmt.Fprintf(os.Stderr, "%#x not parsed\n", key)
			}
		}
	}

	return d.b, d.err
}
