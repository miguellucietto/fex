# fex

A small command runner for `.fex` files.

## Usage

`fex [flags] [sections]`

The flags are:
- `help` - Help -> print the help message
- `-v` - Verbose -> print more information
- `-list` - List -> print the available sections in your .fex file
- `-dry` - Dry Run -> print the commands of the section(s) without running them

The program reads `.fex` from the current working directory.

## File format

```text
CC = gcc
FLAGS = -Wall -Wextra
TARGET = bin/app
SRC_FILES /*.c src

# this is a comment

build:
    $CC $FLAGS $SRC_FILES -o $TARGET
end

run:
    ./$TARGET
end
```

## Features

#### Variables attributions

* `=`: normal way to assign a value to a variable
 - `VAR = something`

* `/`: get the files that match the mode in a spefic directory
 - `SRC_FILES /*.c src`

* `->`: every arg is renamed accordingly to the modes, `left mode` = what to change and `right mode` = which change
 - `OBJ_FILES -> $SRC_FILES : *.c *.o`

#### Comments

You can't make a comment in the end of a line that contains code, only full lines of comments

```
# this is ok
build: # this is not valid comment
    $CC $FLAGS $FILES -o $TARGET # this is not valid comment too
end
```

## Development

build the fex binary with `go build -o fex ./src`
now you can run the tests with `./fex test` and rebuild it with `./fex build` 
