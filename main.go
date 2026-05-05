package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <macro> <file> [file...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Use \"-\" as a file to read from standard input.\n")
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	macro := args[0]
	var b strings.Builder
	for _, path := range args[1:] {
		var content string
		if path == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
				os.Exit(1)
			}
			content = string(data)
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "open %s: %v\n", path, err)
				os.Exit(1)
			}
			content = string(data)
		}
		b.WriteString(convert(macro, content))
	}
	fmt.Println(b.String())
}
