package darwin

import (
	"fmt"
	"path/filepath"
)

// SetAsAlias flags the destination file as an alias.
// This function doesn't verify that the file is actually an alias.
// Don't use on the wrong file!
func SetAsAlias(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%s can't be converted to an absolute path - %s", path, err)
	}
	aliasMagicFlag := []byte{0x61, 0x6c, 0x69, 0x73, 0x4d, 0x41, 0x43, 0x53, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	var dataval *byte
	datalen := len(aliasMagicFlag)
	if datalen > 0 {
		dataval = &aliasMagicFlag[0]
	}
	return setxattr(filepath.Clean(absPath), "com.apple.FinderInfo", dataval, datalen, 0, 0)
}
