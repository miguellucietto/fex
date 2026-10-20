package main

import (
	"fmt"
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

		// TODO: add a way to get files from a directory, maybe like this: varName / dirName
		// and have variations like varName /.go dirName, the varName would have only the
		// files that has suffix .go in the directory. /(contains) | /*(suffix) | /(preffix)*

		return nil, fmt.Errorf("line %d: unexpected content %q", i+1, line)
	}

	return program, nil
}
