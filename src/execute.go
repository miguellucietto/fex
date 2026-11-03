package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func expandLine(fields []string, variables map[string][]string) ([]string, error) {
	var expanded []string

	for _, arg := range fields {
		hasVariable := false

		for i := 0; i < len(arg); i++ {

			if arg[i] != '$' {
				continue
			}

			j := i + 1
			for j < len(arg) {
				c := arg[j]
				// Aceita apenas letras (a-z, A-Z), números (0-9) e sublinhado (_)
				if !((c >= 'a' && c <= 'z') ||
					(c >= 'A' && c <= 'Z') ||
					(c >= '0' && c <= '9') ||
					c == '_') {
					break // Encontrou um caractere delimitador, para a leitura do nome
				}
				j++
			}

			if j == i+1 {
				return nil, fmt.Errorf("invalid variable reference in %q", arg)
			}

			name := arg[i+1 : j]
			value, exists := variables[name]
			if !exists {
				return nil, fmt.Errorf("undefined variable %q", name)
			}

			hasVariable = true

			for _, k := range value {
				expanded = append(expanded, k)
			}

			i = j - 1
		}

		if !hasVariable {
			expanded = append(expanded, arg)
		}
	}

	return expanded, nil
}
func executeSection(
	sectionName string,
	commands []string,
	variables map[string][]string,
	dryRun bool,
	verbose bool,
	stdout, stderr *os.File,
) []error {
	var runErrors []error

	for lineIndex, line := range commands {
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}

		expanded, err := expandLine(args, variables)
		if err != nil {
			runErrors = append(runErrors, fmt.Errorf("section %q, command %d: %w", sectionName, lineIndex+1, err))
			continue
		}
		if len(expanded) == 0 {
			// A command consisting only of an empty variable expands to nothing.
			continue
		}

		if dryRun {
			fmt.Fprintln(stdout, "DRY:", strings.Join(expanded, " "))
			continue
		}

		if verbose {
			fmt.Fprintln(stdout, "CMD:", strings.Join(expanded, " "))
		}

		if err := execCmd(expanded, stdout, stderr); err != nil {
			runErrors = append(runErrors, fmt.Errorf("section %q, command %d (%q): %w", sectionName, lineIndex+1, expanded[0], err))
		}
	}

	return runErrors
}

func execCmd(command []string, stdout, stderr *os.File) error {
	if len(command) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}
