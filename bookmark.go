package cocoa

import (
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/mattetti/cocoa/darwin"
)

/*
	Cocoa provides an API to persist NSURLs. Older versions of the OS were using
	alias records but that API was deprecated in favor of bookmarks. Cocoa does
	also support symlinks and hardlinks but those behave differently than
	bookmarks. Unfortunately, Apple doesn't document the Bookmark Data format.

	Here is some documentation on the usage of bookmarks:
	https://developer.apple.com/library/content/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/AccessingFilesandDirectories/AccessingFilesandDirectories.html#//apple_ref/doc/uid/TP40010672-CH3-SW10

	The format was partly reverse engineered and documented in a few place, here
	is the best summary I found:
	https://github.com/al45tair/mac_alias/blob/master/doc/bookmark_fmt.rst
	(documentation copied in doc/* in case the original repo disappears)
*/

// Bookmark acts like os.Symlink but instead of creating a symlink, a bookmark is stored.
func Bookmark(src, dst string) error {
	srcPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get the path of the source")
	}
	// read the attributes of the source.
	var stat syscall.Statfs_t

	err = syscall.Statfs(filepath.Clean(srcPath), &stat)
	if err != nil {
		return fmt.Errorf("failed to read the file stats - %s", err)
	}

	// Volume path
	volPath := []byte{}
	for _, b := range stat.Mntonname {
		if b == 0x00 {
			break
		}
		volPath = append(volPath, byte(b))
	}
	// fmt.Println(string(mntB))
	buf := make([]byte, 308)
	// attrs, err := GetAttrList(".", AttrList{CommonAttr: ATTR_CMN_FULLPATH}, buf, 0)
	attrs, err := darwin.GetAttrList(string(volPath),
		darwin.AttrListMask{
			CommonAttr: darwin.ATTR_CMN_CRTIME,
			VolAttr:    darwin.ATTR_VOL_SIZE | darwin.ATTR_VOL_NAME | darwin.ATTR_VOL_UUID,
		},
		buf, 0|darwin.FSOPT_REPORT_FULLSIZE)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to retrieve attribute list - %s", err))
	}
	fmt.Printf("%+v\n", attrs)
	fmt.Println("Volume UUID", attrs.StringVolUUID())

	// attributes
	return nil
}
