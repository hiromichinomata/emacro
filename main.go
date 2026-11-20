package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "[usage] %s macro filename\n", os.Args[0])
		os.Exit(1)
	}
	macro := os.Args[1]
	filename := os.Args[2]

	ops, err := parse(macro)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	w := bufio.NewWriter(os.Stdout)
	if err := run(ops, file, w); err != nil {
		w.Flush()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type opKind int

const (
	opInsert opKind = iota
	opBeginningOfLine
	opBackwardChar
	opDeleteChar
	opEndOfLine
	opForwardChar
	opNextLine
	opSearchForward
)

type op struct {
	kind opKind
	text []rune // for opInsert / opSearchForward
}

// parse converts a macro string into a list of operations.
func parse(macro string) ([]op, error) {
	rs := []rune(macro)
	var ops []op
	for i := 0; i < len(rs); i++ {
		if rs[i] != '^' {
			ops = append(ops, op{kind: opInsert, text: []rune{rs[i]}})
			continue
		}
		if i+1 >= len(rs) {
			return nil, fmt.Errorf("invalid macro: trailing '^' (use '^^' to input '^')")
		}
		i++
		switch rs[i] {
		case '^':
			ops = append(ops, op{kind: opInsert, text: []rune{'^'}})
		case 'A':
			ops = append(ops, op{kind: opBeginningOfLine})
		case 'B':
			ops = append(ops, op{kind: opBackwardChar})
		case 'D':
			ops = append(ops, op{kind: opDeleteChar})
		case 'E':
			ops = append(ops, op{kind: opEndOfLine})
		case 'F':
			ops = append(ops, op{kind: opForwardChar})
		case 'N':
			ops = append(ops, op{kind: opNextLine})
		case 'S':
			j := i + 1
			for j < len(rs) && rs[j] != '^' {
				j++
			}
			ops = append(ops, op{kind: opSearchForward, text: rs[i+1 : j]})
			i = j - 1
		default:
			return nil, fmt.Errorf("invalid macro: unsupported operation '^%c'", rs[i])
		}
	}
	return ops, nil
}

// apply runs the operations on a single line.
func apply(ops []op, line string) string {
	rs := []rune(line)
	index := 0
	for _, o := range ops {
		switch o.kind {
		case opInsert:
			rs = append(rs[:index], append(append([]rune{}, o.text...), rs[index:]...)...)
			index += len(o.text)
		case opBeginningOfLine:
			index = 0
		case opBackwardChar:
			if index > 0 {
				index--
			}
		case opDeleteChar:
			if index < len(rs) {
				rs = append(rs[:index], rs[index+1:]...)
			}
		case opEndOfLine:
			index = len(rs)
		case opForwardChar:
			if index < len(rs) {
				index++
			}
		case opNextLine:
			// Each line is processed independently, so this is a no-op.
		case opSearchForward:
			pos := strings.Index(string(rs[index:]), string(o.text))
			if pos == -1 {
				// Like Emacs, a failed search aborts the rest of the macro.
				return string(rs)
			}
			index += len([]rune(string(rs[index:])[:pos])) + len(o.text)
		}
	}
	return string(rs)
}

// run applies the operations to every line read from r and writes to w.
func run(ops []op, r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			eol := "\n"
			line = strings.TrimSuffix(line, "\n")
			if strings.HasSuffix(line, "\r") {
				line = strings.TrimSuffix(line, "\r")
				eol = "\r\n"
			}
			if _, werr := io.WriteString(w, apply(ops, line)+eol); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
