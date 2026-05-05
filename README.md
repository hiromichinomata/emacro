# emacro

Transform each line of a file with Emacs-style key sequences—useful for quick line-oriented edits in the same way you might use `grep` for line-oriented **filtering**.

The macro language supports movement, search, line/word editing, and literal insertion. Processing is **one line at a time**; the cursor is **byte-based** (ASCII / UTF-8 multibyte lines are not fully “character-aware” for all operations).

## Requirements

- [Go](https://go.dev/) 1.22 or later

## Install

From a clone of this repository:

```console
go build -o emacro .
```

To install the binary onto your `PATH` (from the module root):

```console
go install
```

If the module is published under a full import path (for example `github.com/yourname/emacro`), you can also use:

```console
go install github.com/yourname/emacro@latest
```

(after setting the `module` line in `go.mod` to match that path).

## Usage

```console
emacro <macro> <file> [file...]
```

- **Multiple files:** read in order; output is the concatenation of each file’s transformed lines (no filename headers).
- **Standard input:** use `-` as a file name.
- **Help:** `emacro -h` or `emacro --help`.

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

## Macro reference

### Control (two bytes: `^` + letter)

| Sequence | Binding | Effect |
|----------|---------|--------|
| `^A` | `C-a` | Move to beginning of line |
| `^B` | `C-b` | Move backward one character |
| `^D` | `C-d` | Delete character under cursor (or one before end of line) |
| `^E` | `C-e` | Move to end of line |
| `^F` | `C-f` | Move forward one character |
| `^K` | `C-k` | Kill line: delete from cursor through end of line |
| `^N` | `C-n` | No-op (each line is processed separately) |
| `^S` | `C-s` | Forward search: move cursor **past** the next match (see below) |
| `^R` | `C-r` | Backward search: move cursor to the **start** of the last match **before** the current position (see below) |
| `^^` | — | Insert a literal `^` |

Any other character is **inserted** at the cursor; the cursor advances one byte.

### Search terms (`^S` / `^R`)

The search string begins **after** `^S` or `^R` and ends at the **next `^`** in the macro, or at end of macro. To avoid swallowing the rest of the macro (for example before inserting text), **terminate the term with a harmless command** such as `^N` before further letters—e.g. `^E^R,user^NX` if you need a literal `X` after a `^R,user` search.

If the search **fails**, the engine only advances the macro by one step (same idea as the original `^S` behavior), so the following bytes may be interpreted differently than after a successful search.

### Meta (three bytes: `^` + `m` + letter)

| Sequence | Binding | Effect |
|----------|---------|--------|
| `^mf` | `M-f` | Forward one **word** |
| `^mb` | `M-b` | Backward one **word** |
| `^md` | `M-d` | Kill **forward** from cursor through end of next word |

A **word** is a maximal run of ASCII letters, digits, or `_`.

### Notes

- Unknown two-byte sequences like `^Z` advance by one byte in the macro stream (the `^` is skipped in that step).
- Multibyte UTF-8 characters are easiest to reason about when lines stay ASCII; byte indexing can split Unicode code points.

## Development

```console
go test ./...
```

CI runs `go test` on push and pull request ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)).
