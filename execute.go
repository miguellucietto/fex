package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func expandCommand(args []string, variables map[string][]string) ([]string, error) {
	expanded := make([]string, 0, len(args))

	for _, arg := range args {
		if !strings.HasPrefix(arg, "$") {
			expanded = append(expanded, arg)
			continue
		}

		name := strings.TrimPrefix(arg, "$")
		if name == "" {
			return nil, fmt.Errorf("empty variable reference")
		}
		if strings.Contains(name, "$") {
			return nil, fmt.Errorf("invalid variable reference %q", arg)
		}

		value, exists := variables[name]
		if !exists {
			return nil, fmt.Errorf("undefined variable %q", name)
		}
		expanded = append(expanded, value...)
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

		expanded, err := expandCommand(args, variables)
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
