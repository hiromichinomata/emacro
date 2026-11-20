package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"unicode"
	"unicode/utf8"
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

const usage = `Usage: emacro [options] [--] MACRO [FILE...]

Apply an Emacs keyboard macro to each line of FILEs (or stdin).
With no FILE, or when FILE is -, read standard input.

Options (must come before MACRO):
  -h, --help     show this help
  -v, --version  show version
  --strict       follow Emacs strictly:
                   - unsupported operations are errors
                   - a failed search skips the rest of the macro for the line
  --             end of options (use it when MACRO is one of the options)

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

// Exit statuses.
const (
	exitOK           = 0
	exitError        = 1 // usage error, or a file could not be processed
	exitInvalidMacro = 2
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func cli(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	strict := false
	// Only arguments that exactly match an option are treated as options, so
	// macros starting with '-' (e.g. '- ') work as in v0.1.0.
	i := 0
options:
	for ; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprint(stdout, usage)
			return exitOK
		case "-v", "--version":
			fmt.Fprintf(stdout, "emacro %s\n", getVersion())
			return exitOK
		case "--strict":
			strict = true
		case "--":
			i++
			break options
		default:
			break options
		}
	}
	args = args[i:]
	if len(args) < 1 {
		fmt.Fprint(stderr, usage)
		return exitError
	}

	ops, err := parse(args[0], strict)
	if err != nil {
		fmt.Fprintf(stderr, "emacro: %v\n", err)
		return exitInvalidMacro
	}
	m := &machine{ops: ops, strict: strict}

	files := args[1:]
	if len(files) == 0 {
		files = []string{"-"}
	}

	w := bufio.NewWriter(stdout)
	defer w.Flush()
	status := exitOK
	for _, name := range files {
		if err := m.runFile(name, stdin, w); err != nil {
			w.Flush()
			fmt.Fprintf(stderr, "emacro: %v\n", err)
			status = exitError
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(stderr, "emacro: %v\n", err)
		return exitError
	}
	return status
}

// A line is handled as a list of characters. Each character is the bytes of
// one valid UTF-8 rune, or a single byte that is not valid UTF-8, so that
// such bytes are kept as they are.
type chars []string

func split(s string) chars {
	cs := make(chars, 0, len(s))
	for len(s) > 0 {
		_, size := utf8.DecodeRuneInString(s)
		cs = append(cs, s[:size])
		s = s[size:]
	}
	return cs
}

func (cs chars) String() string { return strings.Join(cs, "") }

func isWordChar(c string) bool {
	r, size := utf8.DecodeRuneInString(c)
	if r == utf8.RuneError && size <= 1 {
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r)
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
	text chars // for opInsert / opSearchForward / opSearchBackward
}

var simpleOps = map[string]opKind{
	"A": opBeginningOfLine,
	"B": opBackwardChar,
	"D": opDeleteChar,
	"?": opDeleteBackwardChar,
	"E": opEndOfLine,
	"F": opForwardChar,
	"K": opKillLine,
	"N": opNextLine,
	"Y": opYank,
	"f": opForwardWord,
	"b": opBackwardWord,
	"d": opKillWord,
}

// parse converts a macro string into a list of operations.
//
// Without strict, an unsupported operation '^X' inserts 'X' as in v0.1.0.
func parse(macro string, strict bool) ([]op, error) {
	cs := split(macro)
	var ops []op
	for i := 0; i < len(cs); i++ {
		if cs[i] != "^" {
			ops = append(ops, op{kind: opInsert, text: chars{cs[i]}})
			continue
		}
		if i+1 >= len(cs) {
			return nil, fmt.Errorf("invalid macro: trailing '^' (use '^^' to input '^')")
		}
		i++
		c := cs[i]
		if k, ok := simpleOps[c]; ok {
			ops = append(ops, op{kind: k})
			continue
		}
		switch c {
		case "^":
			ops = append(ops, op{kind: opInsert, text: chars{"^"}})
		case "S", "R":
			word, next := parseSearchWord(cs, i+1)
			kind := opSearchForward
			if c == "R" {
				kind = opSearchBackward
			}
			ops = append(ops, op{kind: kind, text: word})
			i = next - 1
		default:
			if strict {
				if c == "M" {
					return nil, fmt.Errorf("invalid macro: '^M' is only valid to end a search word")
				}
				return nil, fmt.Errorf("invalid macro: unsupported operation '^%s'", c)
			}
			ops = append(ops, op{kind: opInsert, text: chars{c}})
		}
	}
	return ops, nil
}

// parseSearchWord reads a search word starting at cs[start]. The word ends
// before the next '^'. If it is '^M' (RET), it is consumed. It returns the
// word and the index to continue from.
func parseSearchWord(cs chars, start int) (chars, int) {
	i := start
	for i < len(cs) && cs[i] != "^" {
		i++
	}
	word := cs[start:i]
	if i+1 < len(cs) && cs[i+1] == "M" {
		return word, i + 2
	}
	return word, i
}

func forwardWord(cs chars, index int) int {
	for index < len(cs) && !isWordChar(cs[index]) {
		index++
	}
	for index < len(cs) && isWordChar(cs[index]) {
		index++
	}
	return index
}

func backwardWord(cs chars, index int) int {
	for index > 0 && !isWordChar(cs[index-1]) {
		index--
	}
	for index > 0 && isWordChar(cs[index-1]) {
		index--
	}
	return index
}

func matchAt(cs, word chars, pos int) bool {
	if pos < 0 || pos+len(word) > len(cs) {
		return false
	}
	for j, c := range word {
		if cs[pos+j] != c {
			return false
		}
	}
	return true
}

// searchForward returns the index of the first match at or after from, or -1.
func searchForward(cs, word chars, from int) int {
	for pos := from; pos+len(word) <= len(cs); pos++ {
		if matchAt(cs, word, pos) {
			return pos
		}
	}
	return -1
}

// searchBackward returns the index of the last match ending at or before
// to, or -1.
func searchBackward(cs, word chars, to int) int {
	for pos := to - len(word); pos >= 0; pos-- {
		if matchAt(cs, word, pos) {
			return pos
		}
	}
	return -1
}

func insert(cs chars, index int, text chars) chars {
	return append(cs[:index], append(append(chars{}, text...), cs[index:]...)...)
}

type machine struct {
	ops    []op
	strict bool
}

// apply runs the operations on a single line.
func (m *machine) apply(line string) string {
	cs := split(line)
	index := 0
	// The killed text is kept per line. Consecutive kills are appended,
	// like Emacs.
	var killed chars
	lastWasKill := false
	kill := func(from, to int) {
		if !lastWasKill {
			killed = nil
		}
		killed = append(killed, cs[from:to]...)
		cs = append(cs[:from], cs[to:]...)
	}
	// searchFailed handles a failed search. It reports whether the rest of
	// the macro should be skipped.
	searchFailed := func(word chars) bool {
		if m.strict {
			// Like Emacs, a failed search aborts the rest of the macro.
			return true
		}
		// v0.1.0 compatible: insert the search word and continue.
		cs = insert(cs, index, word)
		index += len(word)
		return false
	}
	for _, o := range m.ops {
		isKill := o.kind == opKillLine || o.kind == opKillWord
		switch o.kind {
		case opInsert:
			cs = insert(cs, index, o.text)
			index += len(o.text)
		case opBeginningOfLine:
			index = 0
		case opBackwardChar:
			if index > 0 {
				index--
			}
		case opDeleteChar:
			if index < len(cs) {
				cs = append(cs[:index], cs[index+1:]...)
			}
		case opDeleteBackwardChar:
			if index > 0 {
				cs = append(cs[:index-1], cs[index:]...)
				index--
			}
		case opEndOfLine:
			index = len(cs)
		case opForwardChar:
			if index < len(cs) {
				index++
			}
		case opKillLine:
			kill(index, len(cs))
		case opNextLine:
			// Each line is processed independently, so this is a no-op.
		case opSearchForward:
			pos := searchForward(cs, o.text, index)
			if pos == -1 {
				if searchFailed(o.text) {
					return cs.String()
				}
				break
			}
			index = pos + len(o.text)
		case opSearchBackward:
			pos := searchBackward(cs, o.text, index)
			if pos == -1 {
				if searchFailed(o.text) {
					return cs.String()
				}
				break
			}
			index = pos
		case opForwardWord:
			index = forwardWord(cs, index)
		case opBackwardWord:
			index = backwardWord(cs, index)
		case opKillWord:
			kill(index, forwardWord(cs, index))
		case opYank:
			cs = insert(cs, index, killed)
			index += len(killed)
		}
		lastWasKill = isKill
	}
	return cs.String()
}

func (m *machine) runFile(name string, stdin io.Reader, w io.Writer) error {
	if name == "-" {
		return m.run(stdin, w)
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return m.run(f, w)
}

// run applies the operations to every line read from r and writes to w.
func (m *machine) run(r io.Reader, w io.Writer) error {
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
			if _, werr := io.WriteString(w, m.apply(line)+eol); werr != nil {
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
