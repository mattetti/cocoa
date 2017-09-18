// The cocoa package reimplements some native Cocoa features in pure Go so Go
// programs running on Mac don't need to call into Cocoa. The goal of this
// project is not to replace or cover all Cocoa APIs but to facilitate the work
// of Gophers on Mac.
package cocoa

var (
	Debug bool
)

// bookmarks flags
const (
	bmk_data_type_mask    = 0xffffff00
	bmk_data_subtype_mask = 0x000000ff

	bmk_string  = 0x0100
	bmk_data    = 0x0200
	bmk_number  = 0x0300
	bmk_date    = 0x0400
	bmk_boolean = 0x0500
	bmk_array   = 0x0600
	bmk_dict    = 0x0700
	bmk_uuid    = 0x0800
	bmk_url     = 0x0900
	bmk_null    = 0x0a00

	bmk_st_zero = 0x0000
	bmk_st_one  = 0x0001

	bmk_boolean_st_false = 0x0000
	bmk_boolean_st_true  = 0x0001

	bmk_url_st_absolute = 0x0001
	bmk_url_st_relative = 0x0002

	// Bookmark keys
	//                           = 0x1003
	KBookmarkPath           = 0x1004 // Array of path components
	KBookmarkCNIDPath       = 0x1005 // Array of CNIDs
	KBookmarkFileProperties = 0x1010 // (CFURL rp flags,
	//  CFURL rp flags asked for,
	//  8 bytes NULL)
	KBookmarkFileName         = 0x1020
	KBookmarkFileID           = 0x1030
	KBookmarkFileCreationDate = 0x1040
	KBookmarkUnknown          = 0x1054 // always 1?
	KBookmarkUnknown1         = 0x1055 // point to value in 0x1054
	KBookmarkUnknown2         = 0x1056 // boolean, always true?

	//                           = 0x1101   // ?
	//                           = 0x1102   // ?
	KBookmarkTOCPath            = 0x2000 // A list of (TOC id, ?) pairs
	KBookmarkVolumePath         = 0x2002
	KBookmarkVolumeURL          = 0x2005
	KBookmarkVolumeName         = 0x2010
	KBookmarkVolumeUUID         = 0x2011 // Stored (perversely) as a string
	KBookmarkVolumeSize         = 0x2012
	KBookmarkVolumeCreationDate = 0x2013
	KBookmarkVolumeProperties   = 0x2020
	KBookmarkVolumeIsRoot       = 0x2030 // True if volume is FS root
	KBookmarkVolumeBookmark     = 0x2040 // Embedded bookmark for disk image (TOC id)
	KBookmarkVolumeMountPoint   = 0x2050 // A URL
	//                           = 0x2070
	KBookmarkContainingFolder  = 0xc001 // Index of containing folder in path
	KBookmarkUserName          = 0xc011 // User that created bookmark
	KBookmarkUID               = 0xc012 // UID that created bookmark
	KBookmarkWasFileReference  = 0xd001 // True if the URL was a file reference
	KBookmarkCreationOptions   = 0xd010
	KBookmarkURLLengths        = 0xe003 // See below
	KBookmarkFullFileName      = 0xf017
	KBookmarkFileType          = 0xf022 // -> 0x201 looks like some file reference with file extension
	KBookmarkSecurityExtension = 0xf080
	//                           = 0xf081
)
