package main

import (
	"bytes"
	"os"
	"path/filepath"
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
		{"caret ends search word", "^Sa^^b", "1a2", "1a^b2"},
		{"search then other op ends word", "^Sab^Bx", "zabc", "zaxbc"},
		{"search backward", "^E^R,^Dx", "a,b,c", "a,bxc"},
		{"search backward from cursor", "^E^R,^R,^Dx", "a,b,c", "axb,c"},
		{"search backward with RET", "^E^R,^Mx", "a,b", "ax,b"},
		{"kill line", "^F^K", "abc", "a"},
		{"kill line at end", "^E^K", "abc", "abc"},
		{"delete backward", "^E^?", "abc", "ab"},
		{"delete backward at beginning", "^?x", "ab", "xab"},
		{"delete backward multibyte", "^F^F^?", "あいう", "あう"},
		{"forward word", "^fx", "foo bar", "foox bar"},
		{"forward word skips separators", "^f^fx", "foo, bar baz", "foo, barx baz"},
		{"backward word", "^E^bx", "foo bar", "foo xbar"},
		{"backward word skips separators", "^E^bx", "foo bar, ", "foo xbar, "},
		{"kill word", "^d", "foo bar", " bar"},
		{"kill word from middle", "^F^d", "foo bar", "f bar"},
		{"kill word with separators", "^f^d", "foo, bar baz", "foo baz"},
		{"word multibyte", "^fx", "日本語 abc", "日本語x abc"},
		{"yank after kill line", "^S,^K^A^Y,", "a,b,c", "b,c,a,"},
		{"yank after kill word", "^d^E ^Y", "foo bar", " bar foo"},
		{"yank twice", "^d^Y^Y", "foo bar", "foofoo bar"},
		{"yank without kill", "^Yx", "abc", "xabc"},
		{"consecutive kills append", "^d^d^E^Y", "foo bar baz", " bazfoo bar"},
		{"non-consecutive kills replace", "^d^F^d^E^Y", "foo bar baz", "  bazbar"},
		{"yank multibyte", "^F^K^A^Y", "日本語", "本語日"},
		{"invalid utf-8 is kept", "^E^Bx", "a\xffb\xfe", "a\xffbx\xfe"},
		{"invalid utf-8 delete", "^F^D", "a\xff\xfeb", "a\xfeb"},
		{"invalid utf-8 search", "^S\xff^Dx", "a\xffbc", "a\xffxc"},
		{"invalid utf-8 is not a word char", "^fx", "ab\xffcd", "abx\xffcd"},
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

// Cases where v0.1.0 compatible mode and --strict differ.
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
	// v0.1.0 compatible: '^X' inserts 'X'.
	tests := []struct {
		macro, line, want string
	}{
		{"^Z", "abc", "Zabc"},
		{"^x^Fy", "abc", "xaybc"},
		{"^M", "abc", "Mabc"},
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

func TestParseError(t *testing.T) {
	for _, strict := range []bool{false, true} {
		for _, macro := range []string{"^", "abc^", "^Sa^"} {
			if _, err := parse(macro, strict); err == nil {
				t.Errorf("parse(%q, strict=%v) expected error", macro, strict)
			}
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

func runCLI(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestCLI(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(a, []byte("user01\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("user02\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	macro := "^Suser^D^D"

	tests := []struct {
		name       string
		stdin      string
		args       []string
		wantOut    string
		wantCode   int
		wantStderr bool
	}{
		{"stdin", "user01\n", []string{macro}, "user\n", 0, false},
		{"dash is stdin", "userXX\n", []string{macro, a, "-", b}, "user\nuser\nuser\n", 0, false},
		{"multiple files", "", []string{macro, a, b}, "user\nuser\n", 0, false},
		{"missing file continues", "", []string{macro, a, filepath.Join(dir, "none"), b}, "user\nuser\n", 1, true},
		{"invalid macro", "", []string{"abc^", a}, "", 2, true},
		{"no args", "", nil, "", 1, true},
		{"macro starting with dash", "abc\n", []string{"- "}, "- abc\n", 0, false},
		{"macro that looks like a flag", "abc\n", []string{"-x"}, "-xabc\n", 0, false},
		{"double dash", "abc\n", []string{"--", "-h"}, "-habc\n", 0, false},
		{"compat by default", "abc\n", []string{"^Sfoo^Dx"}, "fooxbc\n", 0, false},
		{"strict", "abc\n", []string{"--strict", "^Sfoo^Dx"}, "abc\n", 0, false},
		{"strict invalid macro", "abc\n", []string{"--strict", "^Z"}, "", 2, true},
		{"strict with double dash", "abc\n", []string{"--strict", "--", "-^Sz^Dx"}, "-abc\n", 0, false},
		{"options only before macro", "", []string{"x", "--strict"}, "", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := runCLI(t, tt.stdin, tt.args...)
			if out != tt.wantOut || code != tt.wantCode || (errOut != "") != tt.wantStderr {
				t.Errorf("got %q, stderr %q, code %d; want %q, code %d", out, errOut, code, tt.wantOut, tt.wantCode)
			}
		})
	}

	for _, f := range []string{"-h", "--help"} {
		t.Run(f, func(t *testing.T) {
			out, _, code := runCLI(t, "", f)
			if code != 0 || !strings.Contains(out, "Usage") {
				t.Errorf("got %q (code %d)", out, code)
			}
		})
	}
	for _, f := range []string{"-v", "--version"} {
		t.Run(f, func(t *testing.T) {
			out, _, code := runCLI(t, "", f)
			if code != 0 || !strings.HasPrefix(out, "emacro ") {
				t.Errorf("got %q (code %d)", out, code)
			}
		})
	}
}
