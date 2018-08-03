package cocoa

import (
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/mattetti/cocoa/darwin"
)

// NewAliasRecord returns th alias record representation of a path
func NewAliasRecord(path string) (*AliasRecord, error) {
	a := &AliasRecord{Path: path}

	srcPath, err := filepath.Abs(path)
	if err != nil {
		return a, fmt.Errorf("failed to read the path - %s", err)
	}
	srcPath = filepath.Clean(srcPath)
	a.Path = srcPath
	// read the attributes of the source.
	var stat syscall.Statfs_t

	err = syscall.Statfs(srcPath, &stat)
	if err != nil {
		return a, fmt.Errorf("failed to read the file stats - %s", err)
	}

	// Volume path
	volPathB := []byte{}
	for _, b := range stat.Mntonname {
		if b == 0x00 {
			break
		}
		volPathB = append(volPathB, byte(b))
	}
	volPath := string(volPathB)
	// volume attributes
	buf := make([]byte, 512)
	volumeAttrs, err := darwin.GetAttrList(volPath,
		darwin.AttrListMask{
			CommonAttr: darwin.ATTR_CMN_CRTIME,
			VolAttr:    darwin.ATTR_VOL_SIZE | darwin.ATTR_VOL_NAME | darwin.ATTR_VOL_UUID,
		},
		buf, 0|darwin.FSOPT_REPORT_FULLSIZE)
	if err != nil {
		return a, fmt.Errorf("failed to retrieve volume attribute list - %s", err)
	}

	// Volume information
	a.VolumeDate = volumeAttrs.CreationTime.Time()
	a.VolumeName = volumeAttrs.VolName
	a.VolumeID = uint16(volumeAttrs.FileID)
	a.FileSystem = "H+"

	fileAttrs, err := darwin.GetAttrList(srcPath,
		darwin.AttrListMask{
			CommonAttr: darwin.ATTR_CMN_OBJTYPE |
				darwin.ATTR_CMN_FNDRINFO |
				darwin.ATTR_CMN_CRTIME |
				darwin.ATTR_CMN_FILEID,
		},
		buf, darwin.FSOPT_NOFOLLOW) // maybe we should follow so we don't have issues with symlinks and aliases?
	if err != nil {
		return a, fmt.Errorf("failed to retrieve file attribute list - %s", err)
	}

	// TODO: decode the source alias and adjust the source instead of failing.
	// macOS UI lest you create an alias to an alias by reading the alias source
	// and creating another version of the alias.
	if fileAttrs.FileInfo.FinderFlags&darwin.FFKIsAlias > 0 {
		return a, fmt.Errorf("can't safely alias to an alias, choose another source")
	}

	// target attributes
	if fileAttrs.ObjType == darwin.VDIR {
		a.Kind = AliasKindFolder
	} else {
		a.Kind = AliasKindFile
	}
	a.TargetName = filepath.Base(path)
	a.TargetCNID = fileAttrs.FileID
	a.TargetCreation = fileAttrs.CreationTime.Time()
	a.DirsAliasToRoot = -1
	a.DirsRootToTarget = -1

	// getting data about each node of the path
	relPath, _ := filepath.Rel(string(volPath), srcPath)
	buf = make([]byte, 256)
	subPath := srcPath
	subPathAttrs, err := darwin.GetAttrList(subPath, darwin.AttrListMask{CommonAttr: darwin.ATTR_CMN_FILEID}, buf, 0)
	if err != nil {
		return a, fmt.Errorf("failed to retrieve file id for %s - %s", subPath, err)
	}
	a.CNIDPath = []uint32{subPathAttrs.FileID}
	a.PathItems = []string{filepath.Base(filepath.Dir(subPath)), filepath.Base(subPath)}

	// walk the path and extract the file id of each sub path
	dir := filepath.Dir(relPath)
	for dir != "" {
		dir, _ = filepath.Split(filepath.Clean(dir))
		if dir == "" {
			break
		}

		a.PathItems = append([]string{filepath.Base(dir)}, a.PathItems...)
		buf = make([]byte, 256)
		subPath = filepath.Join(string(volPath), dir)
		subPathAttrs, err = darwin.GetAttrList(subPath, darwin.AttrListMask{CommonAttr: darwin.ATTR_CMN_FILEID}, buf, 0)
		if err != nil {
			return a, fmt.Errorf("failed to retrieve file id for %s - %s", subPath, err)
		}
		a.CNIDPath = append([]uint32{subPathAttrs.FileID}, a.CNIDPath...)
	}
	folderIDX := len(a.CNIDPath) - 2
	a.FolderCNID = a.CNIDPath[folderIDX]

	return a, nil
}
