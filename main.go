package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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

func convert(macro string, contents string) string {
	var b strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		index := 0
		for i := 0; i < len(macro); i++ {
			if string(macro[i]) != "^" {
				line = line[:index] + string(macro[i]) + line[index:]
				index += 1
			} else {
				if i+1 >= len(macro) {
					continue
				}
				switch macro[i : i+2] {
				case "^^":
					line = line[:index] + "^" + line[index:]
					i += 1
				case "^A":
					index = 0
					i += 1
				case "^B":
					index -= 1
					if index < 0 {
						index = 0
					}
					i += 1
				case "^D":
					if index < len(line)-1 {
						line = line[:index] + line[index+1:]
					} else {
						line = line[:index]
					}
					i += 1
				case "^E":
					index = len(line)
					i += 1
				case "^F":
					if index < len(line) {
						index += 1
					}
					i += 1
				case "^N":
					i += 1
				case "^S":
					searchWord := ""
					for j := i + 2; j < len(macro); j++ {
						if string(macro[j]) != "^" {
							searchWord += string(macro[j])
						} else {
							break
						}
					}
					pos := strings.Index(line[index:], searchWord)
					if pos != -1 {
						index += pos + len(searchWord)
						i += len(searchWord) + 1
					} else {
						i += 1
					}
				default:
					i++
				}
			}
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
