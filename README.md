# emacro
Use Emacs macro like grep

## Install

```
$ go install github.com/hiromichinomata/emacro@latest
```

## Usage

```
emacro [options] [--] MACRO [FILE...]
```

The macro is applied to each line independently. With no FILE, or when FILE is `-`, standard input is read.

```
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

$ cat test/sample.csv | emacro '^S,user^Mname_'
2020-04-01 01:23,username_01,male,17
2020-04-01 02:34,username_02,female,27
2020-04-01 03:45,username_03,male,37
```

Options

* `-h`, `--help`: show help
* `-v`, `--version`: show version
* `--strict`: follow Emacs strictly (see [Strict mode](#strict-mode))
* `--`: end of options

Options must come before MACRO. Only arguments that exactly match an option are treated as options, so a macro starting with `-` works as is (e.g. `emacro '- ' notes.txt`). Use `--` when the macro itself is one of the options (e.g. `emacro -- -h file`).

## Doc
Basic

* `^A` => `C-a`
* `^a` => `M-a`
* `^^` => `^` (literal)

Supported operations

| Macro | Key | Description |
| --- | --- | --- |
| `^A` | `C-a` | beginning of line |
| `^B` | `C-b` | backward char |
| `^D` | `C-d` | delete char |
| `^E` | `C-e` | end of line |
| `^F` | `C-f` | forward char |
| `^K` | `C-k` | kill to end of line |
| `^Y` | `C-y` | yank (paste the last killed text) |
| `^N` | `C-n` | next line (no-op, each line is processed independently) |
| `^R` | `C-r` | search backward |
| `^S` | `C-s` | search forward |
| `^M` | `RET` | end search word |
| `^?` | `DEL` | delete backward char |
| `^b` | `M-b` | backward word |
| `^d` | `M-d` | kill word |
| `^f` | `M-f` | forward word |

Search

* The search word continues until the next `^`. `^M` (`RET`) ends the search word; use it to insert text right after a search (e.g. `^S,user^Mname_`).
* When the search fails, the search word is inserted at the cursor and the macro continues (v0.1.0 compatible). With `--strict`, the rest of the macro is skipped for that line (like Emacs).

Kill and yank

* `^K` and `^d` kill text, and `^Y` yanks it. Consecutive kills are appended (like Emacs).
* Killed text is kept per line; it is not carried over to the next line.

Unsupported operations

* `^X` that is not a supported operation inserts `X` (v0.1.0 compatible). With `--strict`, it is reported as an error.

Strict mode

By default, emacro keeps compatibility with v0.1.0. `--strict` follows Emacs more strictly:

| | default | `--strict` |
| --- | --- | --- |
| failed search (`^S`, `^R`) | insert the search word and continue | skip the rest of the macro for the line |
| unsupported operation `^X` | insert `X` | error (exit status 2) |

Exit status

* `0`: success
* `1`: usage error, or a file could not be read
* `2`: invalid macro

Encoding

Input is processed as UTF-8. Bytes that are not valid UTF-8 are kept as they are, and each such byte is treated as one character.

## Test

```
$ go test ./...
```

## TODO
* [ ] more operations
* [x] how to install
* [x] test
