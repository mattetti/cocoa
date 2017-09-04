[![GoDoc](https://godoc.org/github.com/mattetti/cocoa?status.svg)](https://godoc.org/github.com/mattetti/cocoa)


# Cocoa
Pure Go reimplementation of some Cocoa specific features.

Work in progress, this code is NOT production ready. There is a hight risk that
it will erase all your data and might also make your pets sick!

## Bookmark

Cocoa has a concept of bookmarks/alias which work a little bit with har links but with more flexibility.
Unfortunately this feature isn't exposed outside of Cocoa and while we could use cgo to generate a bookmark.
A pure Go solution has its advantages. 
This implementation is based on many guesses and can be seen as a hack. Use at your own risks.

```go
if err := cocoa.Bookmark("source/path", "destination/path"); err != nil {
    panic(err)
}
```

Yes, the API doesn't use the usual Go's destination then source argument list but hey, I wrote the code so I get to decide :p

Check [GoDoc](https://godoc.org/github.com/mattetti/cocoa) for more information.
