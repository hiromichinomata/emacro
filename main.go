package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	argsCount := len(os.Args) - 1
	if argsCount < 2 {
		fmt.Fprintf(os.Stderr, "[usage] %s macro filename\n", os.Args[0])
		os.Exit(1)
	}
	macro := os.Args[1]
	filename := os.Args[2]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", filename, err)
		os.Exit(1)
	}
	fileContent := string(b)
	fmt.Println(convert(macro, fileContent))
}
