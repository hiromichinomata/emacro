package main

import (
	"fmt"
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
	fmt.Printf("macro: " + macro + "\n")
	fmt.Printf("filename: " + filename + "\n")
}
