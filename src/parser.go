package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

type Program struct {
	variables    map[string][]string
	sections     map[string][]string
	sectionOrder []string
}

func parseData(data string) (*Program, error) {
	program := &Program{
		variables: make(map[string][]string),
		sections:  make(map[string][]string),
	}

	lines := splitLines(data)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasSuffix(line, ":") {
			name := strings.TrimSpace(strings.TrimSuffix(line, ":"))
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
					i = j + 1
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

			program.sections[name] = commands
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

func correspondsMode(name, mode string) bool {
	// Valid modes
	// .sm | *.sm | .sm* | .s*m | *
	if mode == "*" {
		return true
	}

	if mode[0] == '*' {
		return strings.HasSuffix(name, mode[1:])
	}

	if mode[len(mode)-1] == '*' {
		return strings.HasSuffix(name, mode[:len(mode)-1])
	}

	if strings.Contains(mode, "*") {
		parts := strings.SplitN(mode, "*", 2)
		left := parts[0]
		right := parts[1]
		hasLeft := strings.HasSuffix(name, left)
		hasRight := strings.HasPrefix(name, right)

		return hasLeft && hasRight
	}

	return strings.Contains(name, mode)
}
