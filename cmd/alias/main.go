package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattetti/cocoa"
)

var (
	flagSrc   = flag.String("from", "", "Path of the file to link from")
	flagDest  = flag.String("to", "", "Path of the file to link to")
	flagParse = flag.String("parse", "", "debugging option")
)

func main() {
	flag.Parse()
	if *flagParse != "" {
		parse(*flagParse)
		return
	}
	if *flagSrc == "" {
		fmt.Println("You have to pass the source path: -src=<path> (file you want to create a bookmark for)")
		os.Exit(1)
	}
	if *flagDest == "" {
		fmt.Println("You have to define the destination path, where you want to save the bookmark: -dst=<dst>")
		os.Exit(1)
	}

	if cocoa.IsAlias(*flagSrc) {
		fmt.Println("let's not alias to an alias")
		os.Exit(1)
	}
	if err := cocoa.Alias(*flagSrc, *flagDest); err != nil {
		panic(err)
	}
}

func parse(src string) {
	f, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	b, err := cocoa.AliasFromReader(f)
	fmt.Printf("%#v\n", b)
	if err != nil {
		panic(err)
	}
	if len(b.Path) != len(b.CNIDPath) {
		fmt.Printf("The lenght of the path (%d) doesn't match the length of the CNID path (%d)\n", len(b.Path), len(b.CNIDPath))
	}
}
