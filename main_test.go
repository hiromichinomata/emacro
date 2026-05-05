package main

import "testing"

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
