package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime/debug"
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
  ^A   C-a  beginning of line       ^E   C-e  end of line
  ^B   C-b  backward char           ^F   C-f  forward char
  ^D   C-d  delete char             ^?   DEL  delete backward char
  ^K   C-k  kill to end of line     ^Y   C-y  yank (paste killed text)
  ^N   C-n  next line (no-op)
  ^S   C-s  search forward          ^R   C-r  search backward
  ^M   RET  end search word
  ^mb  M-b  backward word           ^mf  M-f  forward word
  ^md  M-d  kill word               ^^        insert '^'
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
		if err := runFile(m, name, stdin, w); err != nil {
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

func runFile(m *machine, name string, stdin io.Reader, w io.Writer) error {
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
