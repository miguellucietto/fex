package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type options struct {
	help    bool
	list    bool
	dryRun  bool
	verbose bool
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("fex", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts options
	fs.BoolVar(&opts.help, "help", false, "Print help message")
	fs.BoolVar(&opts.list, "list", false, "Print available sections")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "Print commands without executing them")
	fs.BoolVar(&opts.verbose, "v", false, "Print commands before executing them")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: fex [flags] [sections...]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if opts.help {
		fs.Usage()
		return nil
	}

	data, err := os.ReadFile(".fex")
	if err != nil {
		return fmt.Errorf("could not read .fex: %w", err)
	}

	program, err := parseData(string(data))
	if err != nil {
		return err
	}

	if opts.list {
		fmt.Fprintln(stdout, "Sections available:")
		for _, name := range program.sectionOrder {
			fmt.Fprintln(stdout, "\t-", name)
		}
		return nil
	}

	sectionNames := fs.Args()
	if len(sectionNames) == 0 {
		fs.Usage()
		return nil
	}

	var runErrors []error
	for _, name := range sectionNames {
		commands, exists := program.sections[name]
		if !exists {
			runErrors = append(runErrors, fmt.Errorf("section %q does not exist", name))
			continue
		}

		errs := executeSection(name, commands, program.variables, opts.dryRun, opts.verbose, stdout, stderr)
		runErrors = append(runErrors, errs...)
	}

	if len(runErrors) > 0 {
		// main prints the joined error once, after all requested sections finish.
		return errors.Join(runErrors...)
	}

	if opts.verbose && !opts.dryRun {
		fmt.Fprintln(stdout, "No errors occurred during command execution.")
	}
	return nil
}

// Keep the helper usable with standard streams while avoiding global flag state.
func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}
