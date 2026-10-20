# fex

A small command runner for `.fex` files.

## Usage

```sh
fex -help
fex -list
fex build
fex -v build test
fex -dry-run build
```

The program reads `.fex` from the current working directory.

## File format

```text
CC = gcc
FLAGS = -Wall -Wextra

build:
    $CC $FLAGS main.c -o app
end

run:
    ./app
end
```

- A section starts with `name:` and ends with `end`.
- Variables are declared as `NAME = value`.
- `$NAME` expands to zero or more command arguments.
- A variable reference must occupy an entire argument. For example, `-I$INCLUDE` is not expanded.
- Commands are split on whitespace. Quotes and shell syntax are not interpreted.
- Blank lines and lines starting with `#` are ignored.
- Commands are executed directly, without a shell.
- Command failures are collected and printed after the requested sections have been processed.
- A parse error prevents execution.
