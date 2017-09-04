// +build !darwin

package darwin

var (
	notDarwin = error.New("Only implemented on Darwin")
)

// SetAsAlias flags the destination file as an alias.
// This function doesn't verify that the file is actually an alias.
// Don't use on the wrong file!
func SetAsAlias(path string) error {
	return notDarwin
}

// GetAttrList returns attributes (that is, metadata) of file system objects. GetAttrList()
// works on the file system object named by path. You can think of getattrlist() as a
// seriously enhanced version of syscall.Stat.  The functions return attributes about
// the specified file system object into the buffer specified by attrBuf and
// attrBufSize.  The attrList parameter determines what attributes are returned.
//
// https://developer.apple.com/legacy/library/documentation/Darwin/Reference/ManPages/man2/getattrlist.2.html
func GetAttrList(path string, mask AttrListMask, attrBuf []byte, options uint32) (results *AttrList, err error) {
	return nil, notDarwin
}
