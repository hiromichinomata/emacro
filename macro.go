package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

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

// isWordChar reports whether c is a word constituent: a letter, a digit or
// '_' (as in v0.2.0, which used [A-Za-z0-9_]).
func isWordChar(c string) bool {
	r, size := utf8.DecodeRuneInString(c)
	if r == utf8.RuneError && size <= 1 {
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
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

// Control operations: '^' + one character.
var controlOps = map[string]opKind{
	"A": opBeginningOfLine,
	"B": opBackwardChar,
	"D": opDeleteChar,
	"?": opDeleteBackwardChar,
	"E": opEndOfLine,
	"F": opForwardChar,
	"K": opKillLine,
	"N": opNextLine,
	"Y": opYank,
}

// Meta operations: '^m' + one character.
var metaOps = map[string]opKind{
	"f": opForwardWord,
	"b": opBackwardWord,
	"d": opKillWord,
}

// parse converts a macro string into a list of operations.
//
// Without strict, it is compatible with v0.2.0:
//   - an unsupported operation '^X' is ignored
//   - for an unsupported meta operation '^mX', '^m' is ignored and 'X' is
//     handled as usual
//   - a trailing '^' (or '^m') is ignored
func parse(macro string, strict bool) ([]op, error) {
	cs := split(macro)
	var ops []op
	for i := 0; i < len(cs); i++ {
		if cs[i] != "^" {
			ops = append(ops, op{kind: opInsert, text: chars{cs[i]}})
			continue
		}
		if i+1 >= len(cs) {
			if strict {
				return nil, fmt.Errorf("invalid macro: trailing '^' (use '^^' to input '^')")
			}
			break
		}
		i++
		c := cs[i]
		if k, ok := controlOps[c]; ok {
			ops = append(ops, op{kind: k})
			continue
		}
		switch c {
		case "^":
			ops = append(ops, op{kind: opInsert, text: chars{"^"}})
		case "m":
			if i+1 < len(cs) {
				if k, ok := metaOps[cs[i+1]]; ok {
					ops = append(ops, op{kind: k})
					i++
					continue
				}
			}
			if strict {
				if i+1 < len(cs) {
					return nil, fmt.Errorf("invalid macro: unsupported operation '^m%s'", cs[i+1])
				}
				return nil, fmt.Errorf("invalid macro: trailing '^m'")
			}
			// v0.2.0 compatible: '^m' is skipped and the next character is
			// handled as usual.
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
			// v0.2.0 compatible: an unsupported operation is ignored.
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
		// v0.1.0 / v0.2.0 compatible: insert the search word and continue.
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

// convert applies macro to every line of contents in the default
// (v0.2.0 compatible) mode.
func convert(macro string, contents string) string {
	ops, _ := parse(macro, false) // never fails without strict
	m := &machine{ops: ops}
	var b strings.Builder
	_ = m.run(strings.NewReader(contents), &b)
	return b.String()
}
