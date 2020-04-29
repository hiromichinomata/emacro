package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	argsCount := len(os.Args) - 1
	if argsCount < 2 {
		fmt.Fprintln(os.Stderr, "[usage] %s macro filename", os.Args[0])
		os.Exit(1)
	}
	macro := os.Args[1]
	filename := os.Args[2]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "file not found")
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	fileContent := string(b)
	result := ""
	result = convert(macro, fileContent)
	fmt.Println(result)
}

func convert(macro string, contents string) string {
	result := ""
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		index := 0
		for i := 0; i < len(macro); i++ {
			if string(macro[i]) != "^" {
				line = line[:index] + string(macro[i]) + line[index:]
				index += 1
			}
		}
		result += line + "\n"
	}
	return result
}
