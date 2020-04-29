# emacro
Use Emacs macro like grep

## Usage

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

```

## Doc
Basic

* `^A` => `C-a`
* `^a` => `M-a`

Supported operations

* `C-a`
* `C-b`
* `C-d`
* `C-e`
* `C-f`
* `C-n`
* `C-s`

To input `^`, you can use `^^`

## TODO
* [ ] more operations
* [ ] how to install
* [ ] test
