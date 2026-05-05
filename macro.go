package main

import (
	"bufio"
	"strings"
)

// applyMacroToLine runs one Emacs-style macro on a single line. The macro
// string is interpreted per line so ^S search failure matches the original
// semantics (macro position advances by one byte instead of skipping the term).
func applyMacroToLine(macro, line string) string {
	index := 0
	for i := 0; i < len(macro); i++ {
		if string(macro[i]) != "^" {
			line = line[:index] + string(macro[i]) + line[index:]
			index++
		} else {
			if i+1 >= len(macro) {
				continue
			}
			switch macro[i : i+2] {
			case "^^":
				line = line[:index] + "^" + line[index:]
				i++
			case "^A":
				index = 0
				i++
			case "^B":
				index--
				if index < 0 {
					index = 0
				}
				i++
			case "^D":
				if index < len(line)-1 {
					line = line[:index] + line[index+1:]
				} else {
					line = line[:index]
				}
				i++
			case "^E":
				index = len(line)
				i++
			case "^F":
				if index < len(line) {
					index++
				}
				i++
			case "^N":
				i++
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
					i++
				}
			default:
				i++
			}
		}
	}
	return line
}

func convert(macro string, contents string) string {
	var b strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := applyMacroToLine(macro, scanner.Text())
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
