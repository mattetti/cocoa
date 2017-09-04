package cocoa

/*
	No op implementations of the features so the package can be compiled
	on other machines and godoc can work fine.
*/

// IsAlias returns positively if the passed file path is an alias.
func IsAlias(src string) bool { return false }

// Alias acts like os.Symlink but instead of creating a symlink, a bookmark is stored.
func Alias(src, dst string) error { return error.New("Only implemented on Darwin")

// AliasFromReader takes an io.reader pointing to an alias file
// decodes it and returns the contained bookmark data.
func AliasFromReader(r io.Reader) (*BookmarkData, error) { return nil, 	error.New("Only implemented on Darwin")