# emacro

Use Emacs-style key sequences to transform each line of a file, similar in spirit to using `grep` for line-oriented workflows.

## Requirements

- [Go](https://go.dev/) 1.22 or later

## Install

From a clone of this repository:

```console
go build -o emacro .
```

To put the binary on your `GOPATH` / `GOBIN` (from the module root):

```console
go install
```

If you publish the module under a full path (for example `github.com/yourname/emacro`), you can install a tagged version with `go install github.com/yourname/emacro@latest` after updating `go.mod`’s `module` line.

## Usage

```console
emacro <macro> <file> [file...]
```

- **Multiple files:** each file is read in order; transformed output is printed in the same order (no filename headers).
- **Standard input:** use `-` as a file name to read from stdin.
- **Help:** `emacro -h` (or `--help`).

### Example

```console
$ cat test/sample.csv
2020-04-01 01:23,user01,male,17
2020-04-01 02:34,user02,female,27
2020-04-01 03:45,user03,male,37

$ grep female test/sample.csv
2020-04-01 02:34,user02,female,27

$ emacro '^S,user^D^D' test/sample.csv
2020-04-01 01:23,user,male,17
2020-04-01 02:34,user,female,27
2020-04-01 03:45,user,male,37
```

```console
$ cat test/sample.csv | emacro '^S,user^D^D' -
2020-04-01 01:23,user,male,17
2020-04-01 02:34,user,female,27
2020-04-01 03:45,user,male,37
```

## Macro syntax

Control characters are written with a leading `^` and an **uppercase** letter, matching the supported operations below.

- `^A` — `C-a` (beginning of line)
- `^B` — `C-b` (backward one character)
- `^D` — `C-d` (delete character under cursor)
- `^E` — `C-e` (end of line)
- `^F` — `C-f` (forward one character)
- `^N` — `C-n` (no-op in this tool; line breaks are handled by the line reader)
- `^S` — `C-s` (search forward for the following text; the search term runs up to the next `^` or end of macro)
- `^^` — a literal `^` (caret)

To type a single `^` in the output, use `^^` at the cursor.

Any other character in the macro is **inserted** at the current cursor position, and the cursor moves forward by one byte (same as the original implementation’s byte-oriented model).

## Development

```console
go test ./...
```

CI runs `go test` on push and pull request (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## TODO

- [ ] More operations (e.g. additional Emacs key bindings)
- [ ] Smoother install story once the module is published under a stable `go install` path
