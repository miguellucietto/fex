# fex

A small command runner for `.fex` files.

## Usage

`fex [flags] [sections]`

The flags are:
- `help` - Help -> print the help message
- `-v` - Verbose -> print more information
- `-list` - List -> print the available sections in your .fex file
- `-dry-run` - Dry Run -> print the commands of the section(s) without running them

The program reads `.fex` from the current working directory.

## File format

```text
CC = gcc
FLAGS = -Wall -Wextra
TARGET = bin/app

build:
    $CC $FLAGS main.c -o $TARGET
end

run:
    ./$TARGET
end
```

## Features

#### Variables

Variables can have a normal atribution (just 1 or more strings) and a directory atribution,
the ways are these:
* `normal atribution` -> `varName = value1 value2`
* `directory atribution` -> `varName /(mode) dirName` - The `modes` are:
  - `varName /* dirName` - get every file in the `dirName`
  - `varName /*sla dirName` - get every file with the suffix `sla` in the `dirName`
  - `varName /sla* dirName` - get every file with the preffix `sla` in the `dirName`
  - `varName /sla dirName` - get every file in `dirName` that contains `sla` in the name
  




