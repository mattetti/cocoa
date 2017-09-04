// +build !darwin

package cocoa

/*
	No op implementations of the features so the package can be compiled
	on other machines
*/


func IsAlias(src string) bool { return false }

func Alias(src, dst string) error { return error.New("Only implemented on Darwin")

func AliasFromReader(r io.Reader) (*BookmarkData, error) { return nil, 	error.New("Only implemented on Darwin")