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
