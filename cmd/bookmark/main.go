package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattetti/cocoa"
)

var (
	flagSrc  = flag.String("from", "", "Path of the file to link from")
	flagDest = flag.String("to", "", "Path of the file to link to")
)

func main() {
	flag.Parse()
	if *flagSrc == "" {
		fmt.Println("You have to pass the source path: -src=<path> (file you want to create a bookmark for)")
		os.Exit(1)
	}
	if *flagDest == "" {
		fmt.Println("You have to define the destination path, where you want to save the bookmark: -dst=<dst>")
		os.Exit(1)
	}

	if err := cocoa.Bookmark(*flagSrc, *flagDest); err != nil {
		panic(err)
	}
}
