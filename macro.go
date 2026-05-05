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
			// Meta: ^mf ^mb ^md (M-f / M-b / M-d); word = ASCII letters, digits, underscore.
			if i+2 < len(macro) && macro[i+1] == 'm' {
				switch macro[i+2] {
				case 'f':
					index = forwardWordEnd(line, index)
					i += 2
				case 'b':
					index = backwardWordStart(line, index)
					i += 2
				case 'd':
					end := forwardWordEnd(line, index)
					line = line[:index] + line[end:]
					i += 2
				default:
					i++
				}
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
			case "^K":
				line = line[:index]
				i++
			case "^N":
				i++
			case "^R":
				searchWord := searchTermAfterCommand(macro, i)
				prefix := line[:index]
				last := strings.LastIndex(prefix, searchWord)
				if last != -1 {
					index = last
					i += len(searchWord) + 1
				} else {
					i++
				}
			case "^S":
				searchWord := searchTermAfterCommand(macro, i)
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

// searchTermAfterCommand reads the literal string after ^S or ^R up to the next '^' or EOF (same rule as Emacs-style forward search in this tool).
func searchTermAfterCommand(macro string, i int) string {
	var b strings.Builder
	for j := i + 2; j < len(macro); j++ {
		if macro[j] != '^' {
			b.WriteByte(macro[j])
		} else {
			break
		}
	}
	return b.String()
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

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func forwardWordEnd(line string, index int) int {
	j := index
	for j < len(line) && !isWordByte(line[j]) {
		j++
	}
	for j < len(line) && isWordByte(line[j]) {
		j++
	}
	return j
}

func backwardWordStart(line string, index int) int {
	j := index
	for j > 0 && !isWordByte(line[j-1]) {
		j--
	}
	for j > 0 && isWordByte(line[j-1]) {
		j--
	}
	return j
}
