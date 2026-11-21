# emacro

Transform each line of a file with Emacs-style key sequences—useful for quick line-oriented edits in the same way you might use `grep` for line-oriented **filtering**.

The macro language supports movement, search, line/word editing, kill/yank, and literal insertion. Processing is **one line at a time**, and the cursor moves **one character (UTF-8) at a time**.

## Requirements

- [Go](https://go.dev/) 1.22 or later

## Install

```console
go install github.com/hiromichinomata/emacro@latest
```

From a clone of this repository:

```console
go build -o emacro .
```

## Usage

```console
emacro [options] [--] <macro> [file...]
```

- **Multiple files:** read in order; output is the concatenation of each file's transformed lines (no filename headers).
- **Standard input:** read when no file is given, or when `-` is given as a file name.
- **Line endings:** LF and CRLF are kept as they are.

### Options

| Option | Effect |
|--------|--------|
| `-h`, `--help` | Show help |
| `-v`, `--version` | Show version |
| `--strict` | Follow Emacs strictly (see [Strict mode](#strict-mode)) |
| `--` | End of options |

Options must come **before** the macro. Only arguments that exactly match an option are treated as options, so a macro starting with `-` works as is (e.g. `emacro '- ' notes.txt`). Use `--` when the macro itself is one of the options (e.g. `emacro -- -h file`).

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
$ cat test/sample.csv | emacro '^S,user^Mname_'
2020-04-01 01:23,username_01,male,17
2020-04-01 02:34,username_02,female,27
2020-04-01 03:45,username_03,male,37
```

## Macro reference

### Control (`^` + character)

| Sequence | Binding | Effect |
|----------|---------|--------|
| `^A` | `C-a` | Move to beginning of line |
| `^B` | `C-b` | Move backward one character |
| `^D` | `C-d` | Delete character under cursor |
| `^E` | `C-e` | Move to end of line |
| `^F` | `C-f` | Move forward one character |
| `^K` | `C-k` | Kill from cursor through end of line |
| `^Y` | `C-y` | Yank (insert the last killed text) |
| `^N` | `C-n` | No-op (each line is processed separately) |
| `^S` | `C-s` | Forward search: move cursor **past** the next match (see below) |
| `^R` | `C-r` | Backward search: move cursor to the **start** of the last match **before** the current position (see below) |
| `^M` | `RET` | End a search word (see below) |
| `^?` | `DEL` | Delete character before cursor |
| `^^` | — | Insert a literal `^` |

Any other character is **inserted** at the cursor; the cursor advances one character.

### Meta (`^m` + character)

| Sequence | Binding | Effect |
|----------|---------|--------|
| `^mf` | `M-f` | Forward one **word** |
| `^mb` | `M-b` | Backward one **word** |
| `^md` | `M-d` | Kill **forward** from cursor through end of next word |

A **word** is a maximal run of letters, digits, or `_` (including non-ASCII letters such as Japanese).

### Search terms (`^S` / `^R`)

The search string begins **after** `^S` or `^R` and ends at the **next `^`** in the macro, or at end of macro. To insert text right after a search, end the term with `^M` (`RET`), e.g. `^S,user^Mname_`. (`^N` also works as a terminator, as in v0.2.0.)

If the search **fails**, the search term is inserted at the cursor and the macro continues (v0.1.0 / v0.2.0 compatible). With `--strict`, the rest of the macro is skipped for that line (like Emacs).

### Kill and yank

- `^K` and `^md` kill text, and `^Y` yanks it. Consecutive kills are appended (like Emacs).
- Killed text is kept per line; it is not carried over to the next line.

### Strict mode

By default, emacro keeps compatibility with v0.2.0. `--strict` follows Emacs more strictly:

| | default | `--strict` |
|---|---|---|
| Failed search (`^S`, `^R`) | Insert the search term and continue | Skip the rest of the macro for the line |
| Unsupported sequence like `^Z` | Ignored | Error (exit status 2) |
| Trailing `^` | Ignored | Error (exit status 2) |

### Exit status

- `0`: success
- `1`: usage error, or a file could not be read
- `2`: invalid macro (only with `--strict`)

### Encoding

Input is processed as UTF-8. Bytes that are not valid UTF-8 (e.g. Shift_JIS text) are kept as they are, and each such byte is treated as one character.

## Development

```console
go test ./...
```
