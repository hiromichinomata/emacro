package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"trailing caret is ignored", "", []string{"abc^", a}, "abcuser01\n", 0, false},
		{"strict trailing caret", "", []string{"--strict", "abc^", a}, "", 2, true},
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
