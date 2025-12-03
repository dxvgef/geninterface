package main

import (
	"flag"
	"fmt"

	"geninterface/pkg"
)

var Version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	filePath := flag.String("file", "", "struct file")
	structName := flag.String("struct", "", "struct name")
	genSetter := flag.Bool("setter", true, "generate setter method")
	flag.Parse()

	if *showVersion {
		fmt.Println(Version)
		return
	}

	if *filePath == "" {
		fmt.Println("error: please provide a Go file path using -file")
		return
	}
	if *structName == "" {
		fmt.Println("error: please provide a struct name using -struct")
		return
	}

	err := pkg.Generator(*filePath, *structName, *genSetter)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("done")
}
