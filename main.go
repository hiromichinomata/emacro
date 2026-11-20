package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"unicode"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = ""

func getVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

const usage = `Usage: emacro [options] MACRO [FILE...]

Apply an Emacs keyboard macro to each line of FILEs (or stdin).
With no FILE, or when FILE is -, read standard input.

Options:
  -h, --help     show this help
  -v, --version  show version

Macro syntax:
  ^A  C-a  beginning of line        ^E  C-e  end of line
  ^B  C-b  backward char            ^F  C-f  forward char
  ^D  C-d  delete char              ^?  DEL  delete backward char
  ^K  C-k  kill to end of line      ^Y  C-y  yank (paste killed text)
  ^N  C-n  next line (no-op)
  ^S  C-s  search forward           ^R  C-r  search backward
  ^M  RET  end search word
  ^b  M-b  backward word            ^f  M-f  forward word
  ^d  M-d  kill word                ^^       insert '^'
`

func main() {
	os.Exit(cli(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func cli(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("emacro", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	var showHelp, showVersion bool
	fs.BoolVar(&showHelp, "h", false, "")
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if showHelp {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if showVersion {
		fmt.Fprintf(stdout, "emacro %s\n", getVersion())
		return 0
	}
	if fs.NArg() < 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	ops, err := parse(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "emacro: %v\n", err)
		return 2
	}

	files := fs.Args()[1:]
	if len(files) == 0 {
		files = []string{"-"}
	}

	w := bufio.NewWriter(stdout)
	defer w.Flush()
	status := 0
	for _, name := range files {
		if err := runFile(ops, name, stdin, w); err != nil {
			w.Flush()
			fmt.Fprintf(stderr, "emacro: %v\n", err)
			status = 1
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(stderr, "emacro: %v\n", err)
		return 1
	}
	return status
}

func runFile(ops []op, name string, stdin io.Reader, w io.Writer) error {
	if name == "-" {
		return run(ops, stdin, w)
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return run(ops, f, w)
}

type opKind int

const (
	opInsert opKind = iota
	opBeginningOfLine
	opBackwardChar
	opDeleteChar
	opDeleteBackwardChar
	opEndOfLine
	opForwardChar
	opKillLine
	opNextLine
	opSearchForward
	opSearchBackward
	opForwardWord
	opBackwardWord
	opKillWord
	opYank
)

type op struct {
	kind opKind
	text []rune // for opInsert / opSearchForward / opSearchBackward
}

var simpleOps = map[rune]opKind{
	'A': opBeginningOfLine,
	'B': opBackwardChar,
	'D': opDeleteChar,
	'?': opDeleteBackwardChar,
	'E': opEndOfLine,
	'F': opForwardChar,
	'K': opKillLine,
	'N': opNextLine,
	'f': opForwardWord,
	'b': opBackwardWord,
	'd': opKillWord,
	'Y': opYank,
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
		c := rs[i]
		if k, ok := simpleOps[c]; ok {
			ops = append(ops, op{kind: k})
			continue
		}
		switch c {
		case '^':
			ops = append(ops, op{kind: opInsert, text: []rune{'^'}})
		case 'S', 'R':
			word, next := parseSearchWord(rs, i+1)
			kind := opSearchForward
			if c == 'R' {
				kind = opSearchBackward
			}
			ops = append(ops, op{kind: kind, text: word})
			i = next - 1
		case 'M':
			return nil, fmt.Errorf("invalid macro: '^M' is only valid to end a search word")
		default:
			return nil, fmt.Errorf("invalid macro: unsupported operation '^%c'", c)
		}
	}
	return ops, nil
}

// parseSearchWord reads a search word starting at rs[start]. The word ends
// before the next operation, or at '^M' (RET), which is consumed. '^^' in the
// word means a literal '^'. It returns the word and the index to continue from.
func parseSearchWord(rs []rune, start int) ([]rune, int) {
	var word []rune
	i := start
	for i < len(rs) {
		if rs[i] != '^' {
			word = append(word, rs[i])
			i++
			continue
		}
		if i+1 < len(rs) && rs[i+1] == '^' {
			word = append(word, '^')
			i += 2
			continue
		}
		if i+1 < len(rs) && rs[i+1] == 'M' {
			return word, i + 2
		}
		break
	}
	return word, i
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func forwardWord(rs []rune, index int) int {
	for index < len(rs) && !isWordRune(rs[index]) {
		index++
	}
	for index < len(rs) && isWordRune(rs[index]) {
		index++
	}
	return index
}

func backwardWord(rs []rune, index int) int {
	for index > 0 && !isWordRune(rs[index-1]) {
		index--
	}
	for index > 0 && isWordRune(rs[index-1]) {
		index--
	}
	return index
}

// runeIndex finds sub in rs and returns the rune offset, or -1.
func runeIndex(rs, sub []rune, last bool) int {
	s := string(rs)
	var pos int
	if last {
		pos = strings.LastIndex(s, string(sub))
	} else {
		pos = strings.Index(s, string(sub))
	}
	if pos == -1 {
		return -1
	}
	return len([]rune(s[:pos]))
}

// apply runs the operations on a single line.
func apply(ops []op, line string) string {
	rs := []rune(line)
	index := 0
	// The killed text is kept per line. Consecutive kills are appended,
	// like Emacs.
	var killed []rune
	lastWasKill := false
	kill := func(from, to int) {
		if !lastWasKill {
			killed = nil
		}
		killed = append(killed, rs[from:to]...)
		rs = append(rs[:from], rs[to:]...)
	}
	for _, o := range ops {
		isKill := o.kind == opKillLine || o.kind == opKillWord
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
		case opDeleteBackwardChar:
			if index > 0 {
				rs = append(rs[:index-1], rs[index:]...)
				index--
			}
		case opEndOfLine:
			index = len(rs)
		case opForwardChar:
			if index < len(rs) {
				index++
			}
		case opKillLine:
			kill(index, len(rs))
		case opNextLine:
			// Each line is processed independently, so this is a no-op.
		case opSearchForward:
			pos := runeIndex(rs[index:], o.text, false)
			if pos == -1 {
				// Like Emacs, a failed search aborts the rest of the macro.
				return string(rs)
			}
			index += pos + len(o.text)
		case opSearchBackward:
			pos := runeIndex(rs[:index], o.text, true)
			if pos == -1 {
				return string(rs)
			}
			index = pos
		case opForwardWord:
			index = forwardWord(rs, index)
		case opBackwardWord:
			index = backwardWord(rs, index)
		case opKillWord:
			kill(index, forwardWord(rs, index))
		case opYank:
			rs = append(rs[:index], append(append([]rune{}, killed...), rs[index:]...)...)
			index += len(killed)
		}
		lastWasKill = isKill
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
