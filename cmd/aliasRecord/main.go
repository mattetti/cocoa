package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattetti/cocoa"
)

var (
	flagSrc = flag.String("for", "", "Path of the file to create the alias record for")
)

func main() {
	flag.Parse()
	if *flagSrc == "" {
		fmt.Println("You have to pass the source path: -for=<path> (file you want to create an alias record for)")
		os.Exit(1)
	}

	r, err := cocoa.NewAliasRecord(*flagSrc)
	if err != nil {
		fmt.Printf("Failed to create an alias record for %s - %s\n", *flagSrc, err)
		os.Exit(1)
	}
	fmt.Printf("%#v\n", r)

	data, err := r.Encode()
	if err != nil {
		fmt.Println("Failed to encode the alias record", err)
	}
	f, err := os.Create("goout.hex")
	if err != nil {
		panic(err)
	}
	f.Write(data)
	f.Close()
	fmt.Println("goout.hex")
}
