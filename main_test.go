package main

import (
	"bytes"
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
	for _, macro := range []string{"^", "abc^", "^b", "^Z"} {
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
