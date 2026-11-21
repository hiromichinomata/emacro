package main

import (
	"bytes"
	"strings"
	"testing"
)

func applyMacro(t *testing.T, macro, line string, strict bool) string {
	t.Helper()
	ops, err := parse(macro, strict)
	if err != nil {
		t.Fatalf("parse(%q) error: %v", macro, err)
	}
	m := &machine{ops: ops, strict: strict}
	return m.apply(line)
}

// Cases that behave the same with and without --strict.
func TestApply(t *testing.T) {
	tests := []struct {
		name  string
		macro string
		line  string
		want  string
	}{
		{"insert", "x", "abc", "xabc"},
		{"readme example", "^S,user^D^D", "2020-04-01 01:23,user01,male,17", "2020-04-01 01:23,user,male,17"},
		{"end and back", "^E^B^B^Dx", "a,17", "a,x7"},
		{"beginning of line", "^E^Ax", "abc", "xabc"},
		{"forward", "^F^Fx", "abc", "abxc"},
		{"forward at end", "^E^Fx", "ab", "abx"},
		{"backward at beginning", "^Bx", "ab", "xab"},
		{"delete at end is noop", "^E^D", "ab", "ab"},
		{"delete last char", "^E^B^D", "ab", "a"},
		{"next line is noop", "^Fx^N^Fy", "abc", "axbyc"},
		{"caret insert advances cursor", "^^x", "ab", "^xab"},
		{"search from cursor", "^Sa^Sa^Dx", "aaab", "aaxb"},
		{"empty search word", "^S^Dx", "abc", "xbc"},
		{"multibyte forward", "^Fx", "あいう", "あxいう"},
		{"multibyte delete", "^F^D", "あいう", "あう"},
		{"multibyte search", "^Sい^D", "あいうえ", "あいえ"},
		{"multibyte insert", "日本", "abc", "日本abc"},
		{"empty line", "^E^Dx", "", "x"},
		{"search ended by RET then insert", "^S,user^Mx", "a,user01", "a,userx01"},
		{"caret ends search word", "^Sa^^mb", "1a2", "1a^mb2"},
		{"search then other op ends word", "^Sab^Bx", "zabc", "zaxbc"},
		{"search backward", "^E^R,^Dx", "a,b,c", "a,bxc"},
		{"search backward from cursor", "^E^R,^R,^Dx", "a,b,c", "axb,c"},
		{"search backward with RET", "^E^R,^Mx", "a,b", "ax,b"},
		{"kill line", "^F^K", "abc", "a"},
		{"kill line at end", "^E^K", "abc", "abc"},
		{"delete backward", "^E^?", "abc", "ab"},
		{"delete backward at beginning", "^?x", "ab", "xab"},
		{"delete backward multibyte", "^F^F^?", "あいう", "あう"},
		{"forward word", "^mfx", "foo bar", "foox bar"},
		{"forward word skips separators", "^mf^mfx", "foo, bar baz", "foo, barx baz"},
		{"backward word", "^E^mbx", "foo bar", "foo xbar"},
		{"backward word skips separators", "^E^mbx", "foo bar, ", "foo xbar, "},
		{"kill word", "^md", "foo bar", " bar"},
		{"kill word from middle", "^F^md", "foo bar", "f bar"},
		{"kill word with separators", "^mf^md", "foo, bar baz", "foo baz"},
		{"word multibyte", "^mfx", "日本語 abc", "日本語x abc"},
		{"underscore is a word char", "^mfx", "foo_bar baz", "foo_barx baz"},
		{"meta at end of word", "^E^mb^md", "foo bar", "foo "},
		{"yank after kill line", "^S,^K^A^Y,", "a,b,c", "b,c,a,"},
		{"yank after kill word", "^md^E ^Y", "foo bar", " bar foo"},
		{"yank twice", "^md^Y^Y", "foo bar", "foofoo bar"},
		{"yank without kill", "^Yx", "abc", "xabc"},
		{"consecutive kills append", "^md^md^E^Y", "foo bar baz", " bazfoo bar"},
		{"non-consecutive kills replace", "^md^F^md^E^Y", "foo bar baz", "  bazbar"},
		{"yank multibyte", "^F^K^A^Y", "日本語", "本語日"},
		{"invalid utf-8 is kept", "^E^Bx", "a\xffb\xfe", "a\xffbx\xfe"},
		{"invalid utf-8 delete", "^F^D", "a\xff\xfeb", "a\xfeb"},
		{"invalid utf-8 search", "^S\xff^Dx", "a\xffbc", "a\xffxc"},
		{"invalid utf-8 is not a word char", "^mfx", "ab\xffcd", "abx\xffcd"},
		{"truncated multibyte is kept", "^Ex", "あ\xe3\x81", "あ\xe3\x81x"},
	}
	for _, tt := range tests {
		for _, strict := range []bool{false, true} {
			name := tt.name
			if strict {
				name += "/strict"
			}
			t.Run(name, func(t *testing.T) {
				if got := applyMacro(t, tt.macro, tt.line, strict); got != tt.want {
					t.Errorf("apply(%q, %q) = %q, want %q", tt.macro, tt.line, got, tt.want)
				}
			})
		}
	}
}

// Cases where the default (v0.2.0 compatible) mode and --strict differ.
func TestApplyCompatAndStrict(t *testing.T) {
	tests := []struct {
		name   string
		macro  string
		line   string
		compat string
		strict string
	}{
		{"search failure", "^Sfoo^Dx", "abc", "fooxbc", "abc"},
		{"search failure keeps earlier edits", "x^Sfoo^Dy", "abc", "xfooybc", "xabc"},
		{"search failure with RET", "^Sfoo^Mx", "abc", "fooxabc", "abc"},
		{"search backward failure", "^E^Rz^Dx", "abc", "abczx", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyMacro(t, tt.macro, tt.line, false); got != tt.compat {
				t.Errorf("compat: apply(%q, %q) = %q, want %q", tt.macro, tt.line, got, tt.compat)
			}
			if got := applyMacro(t, tt.macro, tt.line, true); got != tt.strict {
				t.Errorf("strict: apply(%q, %q) = %q, want %q", tt.macro, tt.line, got, tt.strict)
			}
		})
	}
}

func TestUnsupportedOperation(t *testing.T) {
	// v0.2.0 compatible: '^X' is ignored.
	tests := []struct {
		macro, line, want string
	}{
		{"^Z", "abc", "abc"},
		{"^x^Fy", "abc", "aybc"},
		{"^M", "abc", "abc"},
		{"^S,d^Zq", "abc,def", "abc,dqef"},
		{"^mz", "abc", "zabc"}, // v0.2.0: '^m' is skipped
		{"^m^Fx", "abc", "axbc"},
	}
	for _, tt := range tests {
		if got := applyMacro(t, tt.macro, tt.line, false); got != tt.want {
			t.Errorf("apply(%q, %q) = %q, want %q", tt.macro, tt.line, got, tt.want)
		}
		if _, err := parse(tt.macro, true); err == nil {
			t.Errorf("strict: parse(%q) expected error", tt.macro)
		}
	}
}

func TestTrailingCaret(t *testing.T) {
	// v0.2.0 compatible: a trailing '^' (or '^m') is ignored.
	tests := []struct {
		macro, line, want string
	}{
		{"^", "abc", "abc"},
		{"x^", "abc", "xabc"},
		{"^Sb^", "abc", "abc"},
		{"^m", "abc", "abc"},
	}
	for _, tt := range tests {
		if got := applyMacro(t, tt.macro, tt.line, false); got != tt.want {
			t.Errorf("apply(%q, %q) = %q, want %q", tt.macro, tt.line, got, tt.want)
		}
		if _, err := parse(tt.macro, true); err == nil {
			t.Errorf("strict: parse(%q) expected error", tt.macro)
		}
	}
}

func runMacro(t *testing.T, macro, input string) string {
	t.Helper()
	ops, err := parse(macro, false)
	if err != nil {
		t.Fatal(err)
	}
	m := &machine{ops: ops}
	var buf bytes.Buffer
	if err := m.run(strings.NewReader(input), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRun(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lines", "user01\nuser02\n", "user\nuser\n"},
		{"no trailing newline", "user01\nuser02", "user\nuser\n"},
		{"crlf", "user01\r\nuser02\r\n", "user\r\nuser\r\n"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runMacro(t, "^Suser^D^D", tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunKillIsPerLine(t *testing.T) {
	if got, want := runMacro(t, "^Y^K", "abc\ndef\n"), "\n\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := runMacro(t, "^Y^F^K", "abc\ndef\n"), "a\nd\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunLongLine(t *testing.T) {
	long := strings.Repeat("a", 100000)
	if got, want := runMacro(t, "^Ex", long+"\n"), long+"x\n"; got != want {
		t.Errorf("long line not handled: got len %d, want len %d", len(got), len(want))
	}
}

func TestRunInvalidUTF8(t *testing.T) {
	input := "\x82\xa0\x82\xa2\n" // "あい" in Shift_JIS
	if got, want := runMacro(t, "^E!", input), "\x82\xa0\x82\xa2!\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Tests from v0.2.0. They must keep passing for compatibility.

func Test_convert_readmeExample(t *testing.T) {
	sample := "" +
		"2020-04-01 01:23,user01,male,17\n" +
		"2020-04-01 02:34,user02,female,27\n" +
		"2020-04-01 03:45,user03,male,37\n"
	macro := "^S,user^D^D"
	want := "" +
		"2020-04-01 01:23,user,male,17\n" +
		"2020-04-01 02:34,user,female,27\n" +
		"2020-04-01 03:45,user,male,37\n"
	if got := convert(macro, sample); got != want {
		t.Errorf("convert(%q)\nwant:\n%q\ngot:\n%q", macro, want, got)
	}
}

func Test_convert_literalCaret(t *testing.T) {
	// ^^ inserts a single ^ at the cursor; cursor starts at 0
	if got := convert(`^^`, "ab"); got != "^ab\n" {
		t.Errorf("got %q want %q", got, "^ab\n")
	}
	// after moving past first char, ^^ inserts between a and b
	if got := convert(`^F^^`, "ab"); got != "a^b\n" {
		t.Errorf("got %q want %q", got, "a^b\n")
	}
}

func Test_convert_insertMovesCursor(t *testing.T) {
	// non-^ characters insert at cursor and advance
	if got := convert("X", "ab"); got != "Xab\n" {
		t.Errorf("got %q want %q", got, "Xab\n")
	}
}

func Test_convert_emptyContents(t *testing.T) {
	if got := convert("", ""); got != "" {
		t.Errorf("got %q want empty", got)
	}
}

func Test_convert_trailingCaret(t *testing.T) {
	// lone '^' at end of macro must not panic (invalid slice)
	if got := convert("^", "ab"); got != "ab\n" {
		t.Errorf("got %q want %q", got, "ab\n")
	}
}

func Test_convert_metaWord(t *testing.T) {
	// Word characters: [A-Za-z0-9_]
	t.Run("forwardWordThenInsert", func(t *testing.T) {
		got := convert("^mfX", "hello world")
		want := "helloX world\n"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("backwardWordFromEOL", func(t *testing.T) {
		got := convert("^E^mb", "hello world")
		want := "hello world\n"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("killWord", func(t *testing.T) {
		got := convert("^md", "hello world")
		want := " world\n"
		if got != want {
			t.Errorf("got %q want %q", got, " world\n")
		}
	})
}

func Test_convert_killLine(t *testing.T) {
	t.Run("fromStartClearsLine", func(t *testing.T) {
		if got := convert("^K", "hello"); got != "\n" {
			t.Errorf("got %q want newline-only line", got)
		}
	})
	t.Run("afterForwardDeletesTail", func(t *testing.T) {
		if got := convert("^F^F^K", "abcd"); got != "ab\n" {
			t.Errorf("got %q want %q", got, "ab\n")
		}
	})
	t.Run("atEndNoChange", func(t *testing.T) {
		if got := convert("^E^K", "hi"); got != "hi\n" {
			t.Errorf("got %q want %q", got, "hi\n")
		}
	})
}

func Test_convert_reverseSearch(t *testing.T) {
	t.Run("cursorAtMatchStartThenInsert", func(t *testing.T) {
		// Search term must end at the next '^' (^N is a no-op). ^E then ^R,user; point at
		// start of last ",user"; then X inserts there (no comma before X — would insert literally).
		got := convert("^E^R,user^NX", "a,user,b,user")
		want := "a,user,bX,user\n"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("lastOccurrenceBeforeCursor", func(t *testing.T) {
		got := convert("^E^Rabc", "xxabcyyabczz")
		// line unchanged; macro only moves cursor (no trailing edit)
		want := "xxabcyyabczz\n"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("noMatchThenContinuesMacro", func(t *testing.T) {
		// Term ",x" ends at ^A. On failure, i++ then the for-loop i++ skips past "^R" entirely
		// (same pattern as ^S), so the next macro byte is ',' — not a literal 'R'.
		got := convert("^R,x^A", "abc")
		want := ",xabc\n"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
