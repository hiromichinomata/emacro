# emacro
Use Emacs macro like grep

## Install

```
$ go install github.com/hiromichinomata/emacro@latest
```

## Usage

```
emacro [options] MACRO [FILE...]
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

* The search word continues until the next operation, or until `^M` (`RET`), which ends it. Use `^M` to insert text right after a search.
* `^^` in a search word matches a literal `^`.
* When the search fails, the rest of the macro is skipped for that line (like Emacs).

Kill and yank

* `^K` and `^d` kill text, and `^Y` yanks it. Consecutive kills are appended (like Emacs).
* Killed text is kept per line; it is not carried over to the next line.

Unsupported operations are reported as an error.

## Test

```
$ go test ./...
```

## TODO
* [ ] more operations
* [x] how to install
* [x] test
