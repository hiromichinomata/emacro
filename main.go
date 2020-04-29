package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	argsCount := len(os.Args) - 1
	if argsCount < 2 {
		fmt.Fprintln(os.Stderr, "[usage] %s macro filename", os.Args[0])
		os.Exit(1)
	}
	// macro := os.Args[1]
	filename := os.Args[2]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "file not found")
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	fileContent := string(b)
	fmt.Println(fileContent)

	// fmt.Println("macro: " + macro)
}
