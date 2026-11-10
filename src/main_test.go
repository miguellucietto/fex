package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runInDirectory(t *testing.T, config string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	t.Logf("fixture working directory=%q; Fex contents=%q", workDir, config)
	if config != "" {
		if err := os.WriteFile(filepath.Join(workDir, "Fex"), []byte(config), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func TestRunHelpAndNoArguments(t *testing.T) {
	for _, args := range [][]string{{"-help"}, {}} {
		config := ""
		if len(args) == 0 {
			config = "empty:\nend\n"
		}
		runInDirectory(t, config)
		t.Logf("input CLI args=%#v; expected usage text on stderr and empty stdout", args)
		stdout, stderr, read := captureFiles(t)
		if err := run(args, stdout, stderr); err != nil {
			t.Fatalf("run(%v) error = %v", args, err)
		}
		out, errOut := read()
		if out != "" || !strings.Contains(errOut, "Usage: fex") {
			t.Errorf("run(%v) output = (%q, %q), want usage on stderr", args, out, errOut)
		}
	}
}

func TestRunListDryVerboseAndMultipleSections(t *testing.T) {
	config := "MESSAGE = hello\nfirst:\n    echo $MESSAGE\nend\nsecond:\n    echo second\nend\n"

	t.Run("list", func(t *testing.T) {
		runInDirectory(t, config)
		t.Logf("input CLI args=%#v; expected sections in declaration order: first, second", []string{"-list"})
		stdout, stderr, read := captureFiles(t)
		if err := run([]string{"-list"}, stdout, stderr); err != nil {
			t.Fatal(err)
		}
		out, _ := read()
		if !strings.Contains(out, "- first\n\t- second") {
			t.Errorf("list output = %q", out)
		}
	})

	t.Run("dry run", func(t *testing.T) {
		runInDirectory(t, config)
		t.Logf("input CLI args=%#v; expected expanded dry-run line: DRY: echo hello", []string{"-dry", "first"})
		stdout, stderr, read := captureFiles(t)
		if err := run([]string{"-dry", "first"}, stdout, stderr); err != nil {
			t.Fatal(err)
		}
		out, _ := read()
		if !strings.Contains(out, "DRY: echo hello") {
			t.Errorf("dry output = %q", out)
		}
	})

	t.Run("verbose execution and multiple sections", func(t *testing.T) {
		runInDirectory(t, config)
		t.Logf("input CLI args=%#v; expected both CMD lines, process outputs, and success summary", []string{"-v", "first", "second"})
		stdout, stderr, read := captureFiles(t)
		if err := run([]string{"-v", "first", "second"}, stdout, stderr); err != nil {
			t.Fatal(err)
		}
		out, _ := read()
		for _, want := range []string{"echo hello", "hello", "echo second", "No errors occurred"} {
			if !strings.Contains(out, want) {
				t.Errorf("verbose output %q does not contain %q", out, want)
			}
		}
	})
}

func TestRunReportsReadParseAndAggregateExecutionErrors(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		runInDirectory(t, "")
		t.Logf("input CLI args=%#v; Fex file is absent; expected read error", []string{"target"})
		stdout, stderr, _ := captureFiles(t)
		err := run([]string{"target"}, stdout, stderr)
		if err == nil || !strings.Contains(err.Error(), "could not read Fex") {
			t.Fatalf("run() error = %v, want read error", err)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		config := "broken:\n"
		runInDirectory(t, config)
		t.Logf("input CLI args=%#v; expected parse error containing %q", []string{"broken"}, "missing 'end'")
		stdout, stderr, _ := captureFiles(t)
		err := run([]string{"broken"}, stdout, stderr)
		if err == nil || !strings.Contains(err.Error(), "missing 'end'") {
			t.Fatalf("run() error = %v, want parse error", err)
		}
	})

	t.Run("unknown flags", func(t *testing.T) {
		runInDirectory(t, "")
		t.Logf("input CLI args=%#v; expected unknown-flag error", []string{"-unknown"})
		stdout, stderr, _ := captureFiles(t)
		err := run([]string{"-unknown"}, stdout, stderr)
		if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("run() error = %v, want flag error", err)
		}
	})

	t.Run("errors are joined while later sections still execute", func(t *testing.T) {
		config := "bad:\n    command-that-does-not-exist-fex\nend\ngood:\n    echo recovered\nend\n"
		runInDirectory(t, config)
		t.Logf("input CLI args=%#v; expected errors for unknown and bad plus execution of good", []string{"unknown", "bad", "good"})
		stdout, stderr, read := captureFiles(t)
		err := run([]string{"unknown", "bad", "good"}, stdout, stderr)
		if err == nil || !strings.Contains(err.Error(), `section "unknown" does not exist`) || !strings.Contains(err.Error(), `section "bad", command 1`) {
			t.Fatalf("run() error = %v, want joined errors", err)
		}
		out, _ := read()
		if !strings.Contains(out, "recovered") {
			t.Errorf("later requested section did not run; stdout = %q", out)
		}
	})
}
