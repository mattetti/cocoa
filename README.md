[![GoDoc](https://godoc.org/github.com/mattetti/cocoa?status.svg)](https://godoc.org/github.com/mattetti/cocoa)


# Cocoa
Pure Go reimplementation of some Cocoa specific features.

Work in progress, this code is NOT production ready. There is a hight risk that
it will erase all your data and might also make your pets sick!

## Alias

Cocoa has a concept of alias which work a little bit like hard links but with more flexibility.
Unfortunately this feature isn't exposed outside of Cocoa and while we could use cgo to generate a aliases, a pure Go solution has its advantages.
This implementation is based on many guesses and can be seen as a hack. Use at your own risks.

```go
if err := cocoa.Alias("source/path", "destination/path"); err != nil {
    panic(err)
}
```


Check [GoDoc](https://godoc.org/github.com/mattetti/cocoa) for more information.
