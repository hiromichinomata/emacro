package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"search failure aborts macro", "^Sfoo^Dx", "abc", "abc"},
		{"search failure keeps earlier edits", "x^Sfoo^Dy", "abc", "xabc"},
		{"search from cursor", "^Sa^Sa^Dx", "aaab", "aaxb"},
		{"multibyte forward", "^Fx", "あいう", "あxいう"},
		{"multibyte delete", "^F^D", "あいう", "あう"},
		{"multibyte search", "^Sい^D", "あいうえ", "あいえ"},
		{"multibyte insert", "日本", "abc", "日本abc"},
		{"empty line", "^E^Dx", "", "x"},
		{"search ended by RET then insert", "^S,user^Mx", "a,user01", "a,userx01"},
		{"search word with literal caret", "^Sa^^b^M^Dx", "1a^bcd", "1a^bxd"},
		{"search then other op ends word", "^Sab^Bx", "zabc", "zaxbc"},
		{"search backward", "^E^R,^Dx", "a,b,c", "a,bxc"},
		{"search backward from cursor", "^E^R,^R,^Dx", "a,b,c", "axb,c"},
		{"search backward failure aborts", "^E^Rz^Dx", "abc", "abc"},
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
		{"kill is per line", "^Y", "abc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, err := parse(tt.macro)
			if err != nil {
				t.Fatalf("parse(%q) error: %v", tt.macro, err)
			}
			if got := apply(ops, tt.line); got != tt.want {
				t.Errorf("apply(%q, %q) = %q, want %q", tt.macro, tt.line, got, tt.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	for _, macro := range []string{"^", "abc^", "^Z", "^x", "^M", "x^M"} {
		if _, err := parse(macro); err == nil {
			t.Errorf("parse(%q) expected error", macro)
		}
	}
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
	ops, err := parse("^Suser^D^D")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := run(ops, strings.NewReader(tt.input), &buf); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunLongLine(t *testing.T) {
	long := strings.Repeat("a", 100000)
	ops, _ := parse("^Ex")
	var buf bytes.Buffer
	if err := run(ops, strings.NewReader(long+"\n"), &buf); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), long+"x\n"; got != want {
		t.Errorf("long line not handled: got len %d, want len %d", len(got), len(want))
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

	t.Run("stdin", func(t *testing.T) {
		out, _, code := runCLI(t, "user01\n", macro)
		if code != 0 || out != "user\n" {
			t.Errorf("got %q (code %d)", out, code)
		}
	})
	t.Run("dash is stdin", func(t *testing.T) {
		out, _, code := runCLI(t, "userXX\n", macro, a, "-", b)
		if code != 0 || out != "user\nuser\nuser\n" {
			t.Errorf("got %q (code %d)", out, code)
		}
	})
	t.Run("multiple files", func(t *testing.T) {
		out, _, code := runCLI(t, "", macro, a, b)
		if code != 0 || out != "user\nuser\n" {
			t.Errorf("got %q (code %d)", out, code)
		}
	})
	t.Run("missing file continues", func(t *testing.T) {
		out, errOut, code := runCLI(t, "", macro, a, filepath.Join(dir, "none"), b)
		if code != 1 || out != "user\nuser\n" || errOut == "" {
			t.Errorf("got %q, stderr %q (code %d)", out, errOut, code)
		}
	})
	t.Run("invalid macro", func(t *testing.T) {
		_, errOut, code := runCLI(t, "", "abc^", a)
		if code != 2 || errOut == "" {
			t.Errorf("stderr %q (code %d)", errOut, code)
		}
	})
	t.Run("no args", func(t *testing.T) {
		_, errOut, code := runCLI(t, "")
		if code != 2 || !strings.Contains(errOut, "Usage") {
			t.Errorf("stderr %q (code %d)", errOut, code)
		}
	})
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
	t.Run("unknown flag", func(t *testing.T) {
		_, _, code := runCLI(t, "", "-x")
		if code != 2 {
			t.Errorf("code %d", code)
		}
	})
}
