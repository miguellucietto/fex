package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

type Section struct {
	name     string
	deps     []string
	commands []string
}

type Program struct {
	variables    map[string][]string
	sections     map[string]Section
	sectionOrder []string
}

func parseData(data string) (*Program, error) {
	program := &Program{
		variables: make(map[string][]string),
		sections:  make(map[string]Section),
	}

	lines := splitLines(data)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, ":") && !strings.Contains(line, "->") {
			l := strings.Split(line, ":")
			if len(strings.Fields(l[0])) > 1 {
				return nil, fmt.Errorf("line %d: invalid section name", i+1)
			}
			name := strings.TrimSpace(l[0])

			var deps []string
			if len(l) > 1 {
				deps = strings.Fields(l[1])
			}

			if name == "" {
				return nil, fmt.Errorf("line %d: section name cannot be empty", i+1)
			}
			if strings.ContainsAny(name, " \t") {
				return nil, fmt.Errorf("line %d: invalid section name %q", i+1, name)
			}
			if _, exists := program.sections[name]; exists {
				return nil, fmt.Errorf("line %d: section %q is declared more than once", i+1, name)
			}

			startLine := i + 1
			var commands []string
			endFound := false

			for j := i + 1; j < len(lines); j++ {
				commandLine := strings.TrimSpace(lines[j])

				if commandLine == "" || strings.HasPrefix(commandLine, "#") {
					continue
				}
				if commandLine == "end" {
					i = j
					endFound = true
					break
				}
				if strings.HasSuffix(commandLine, ":") {
					return nil, fmt.Errorf("line %d: section %q is missing 'end' before section %q", j+1, name, strings.TrimSuffix(commandLine, ":"))
				}
				commands = append(commands, commandLine)
			}

			if !endFound {
				return nil, fmt.Errorf("line %d: section %q is missing 'end'", startLine, name)
			}

			program.sections[name] = Section{
				commands: commands,
				deps:     deps,
				name:     name,
			}
			program.sectionOrder = append(program.sectionOrder, name)
			continue
		}

		if line == "end" {
			return nil, fmt.Errorf("line %d: 'end' without an open section", i+1)
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			name := strings.TrimSpace(parts[0])
			if name == "" {
				return nil, fmt.Errorf("line %d: variable name cannot be empty", i+1)
			}
			if strings.ContainsAny(name, " \t$/") {
				return nil, fmt.Errorf("line %d: invalid variable name %q", i+1, name)
			}
			program.variables[name] = strings.Fields(strings.TrimSpace(parts[1]))
			continue
		}

		if strings.Contains(line, "/") {

			parts := strings.SplitN(line, "/", 2)
			p := strings.SplitN(parts[1], " ", 2)
			parts = slices.Delete(parts, 1, 2)
			parts = append(parts, p...)

			if len(parts) != 3 {
				return nil, fmt.Errorf("line %d: not enough tokens in directory attribution, name /$mode dir", i+1)
			}

			name := strings.TrimSpace(parts[0])
			mode := strings.TrimSpace(parts[1])
			dir := strings.TrimSpace(parts[2])
			if name == "" || dir == "" || mode == "" {
				return nil, fmt.Errorf("line %d: variable name, directory name and mode cannot be empty", i+1)
			}

			m, err1 := expandLine([]string{mode}, program.variables)
			d, err2 := expandLine([]string{dir}, program.variables)
			if err1 != nil || err2 != nil {
				return nil, errors.Join(err1, err2)
			}

			mode = strings.Join(m, " ")
			dir = strings.Join(d, " ")

			files, err := getFiles(dir, mode)
			if err != nil {
				return nil, err
			}
			program.variables[name] = files
			continue
		}
		if strings.Contains(line, "->") {
			// SRC_FILES /*.c src
			// OBJ_FILES -> $SRC_FILES : *.c *.o
			parts := strings.SplitN(line, "->", 2)
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			if strings.ContainsAny(left, " $") {
				return nil, fmt.Errorf("line %d: variable name with spaces or '$' are not allowed", i+1)
			}
			name := left
			if !strings.Contains(right, ":") {
				return nil, fmt.Errorf("line %d: '->' attribution requires ':' (end of arguments) identifier", i+1)
			}
			parts = strings.SplitN(right, ":", 2)
			left = strings.TrimSpace(parts[0])
			right = strings.TrimSpace(parts[1])
			modes := strings.Fields(right)
			if len(modes) != 2 {
				return nil, fmt.Errorf("line %d: unexpected amount of modes [%v]", i+1, strings.Join(modes, ", "))
			}
			args, err := expandLine(strings.Fields(left), program.variables)
			if err != nil {
				return nil, err
			}
			var result []string
			for _, arg := range args {
				transformedString := transformString(arg, modes[0], modes[1])
				result = append(result, transformedString)
			}
			program.variables[name] = result
			continue
		}

		return nil, fmt.Errorf("line %d: unexpected content %q", i+1, line)
	}

	return program, nil
}

func getFiles(dir, mode string) ([]string, error) {
	dirEntry, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, file := range dirEntry {
		if !file.IsDir() && file.Type().IsRegular() {
			if correspondsMode(file.Name(), mode) {
				files = append(files, dir+"/"+file.Name())
			}
		}
	}

	return files, nil
}

func transformString(input, inputMode, outputMode string) string {
	if inputMode == outputMode || !correspondsMode(input, inputMode) {
		return input
	}

	if !strings.Contains(outputMode, "*") {
		return outputMode
	}

	// * || .
	if inputMode == "*" || !strings.Contains(inputMode, "*") {
		// *.sm
		if outputMode[0] == '*' {
			suffix := outputMode[1:]
			return input + suffix
		}

		// .sm*
		if outputMode[len(outputMode)-1] == '*' {
			preffix := outputMode[:len(outputMode)-2]
			return preffix + input
		}

		// .s*m
		parts := strings.SplitN(outputMode, "*", 2)
		left := parts[0]
		right := parts[1]
		return left + input + right
	}

	if inputMode[0] == '*' {
		before, found := strings.CutSuffix(input, inputMode[1:])
		if !found {
			return input
		}

		// *
		if outputMode == "*" {
			return before
		}

		// *.sm
		if outputMode[0] == '*' {
			return before + outputMode[1:]
		}

		// .sm*
		if outputMode[len(outputMode)-1] == '*' {
			return outputMode[:len(outputMode)-2] + before
		}

		// .s*m
		parts := strings.SplitN(outputMode, "*", 2)
		left := parts[0]
		right := parts[1]
		return left + before + right
	}

	after, found := strings.CutPrefix(input, inputMode[:len(inputMode)-2])
	if !found {
		return input
	}

	// *
	if outputMode == "*" {
		return after
	}

	// *.sm
	if outputMode[0] == '*' {
		return after + outputMode[1:]
	}

	// .sm*
	if outputMode[len(outputMode)-1] == '*' {
		return outputMode[:len(outputMode)-2] + after
	}

	parts := strings.SplitN(outputMode, "*", 2)
	left := parts[0]
	right := parts[1]
	return left + after + right
}

func correspondsMode(name, mode string) bool {
	// Valid modes
	// .sm | *.sm | .sm* | .s*m | *
	if mode == "" || strings.Count(mode, "*") > 1 {
		return false
	}

	// *
	if mode == "*" {
		return true
	}

	// *.sm
	if mode[0] == '*' {
		return strings.HasSuffix(name, mode[1:])
	}

	// .sm*
	if mode[len(mode)-1] == '*' {
		return strings.HasPrefix(name, mode[:len(mode)-1])
	}

	// .s*m
	if strings.Contains(mode, "*") {
		parts := strings.SplitN(mode, "*", 2)
		left := parts[0]
		right := parts[1]
		hasLeft := strings.HasPrefix(name, left)
		hasRight := strings.HasSuffix(name, right)

		return hasLeft && hasRight
	}

	return strings.Contains(name, mode)
}
