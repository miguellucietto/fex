package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func captureFiles(t *testing.T) (*os.File, *os.File, func() (string, string)) {
	t.Helper()
	dir := t.TempDir()
	stdout, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		stdout.Close()
		t.Fatal(err)
	}
	read := func() (string, string) {
		t.Helper()
		if err := stdout.Sync(); err != nil {
			t.Fatal(err)
		}
		if err := stderr.Sync(); err != nil {
			t.Fatal(err)
		}
		out, err := os.ReadFile(stdout.Name())
		if err != nil {
			t.Fatal(err)
		}
		errOut, err := os.ReadFile(stderr.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(out), string(errOut)
	}
	t.Cleanup(func() { _ = stdout.Close(); _ = stderr.Close() })
	return stdout, stderr, read
}

func TestExpandLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fields    []string
		variables map[string][]string
		want      []string
		wantError string
	}{
		{
			name:      "variable expands to multiple arguments",
			fields:    []string{"gcc", "$FLAGS", "main.c"},
			variables: map[string][]string{"FLAGS": {"-Wall", "-Wextra"}},
			want:      []string{"gcc", "-Wall", "-Wextra", "main.c"},
		},
		{
			name:      "empty variable removes its argument",
			fields:    []string{"echo", "$EMPTY"},
			variables: map[string][]string{"EMPTY": {}},
			want:      []string{"echo"},
		},
		{
			name:      "undefined variable",
			fields:    []string{"$MISSING"},
			variables: map[string][]string{},
			wantError: `undefined variable "MISSING"`,
		},
		{
			name:      "invalid variable reference",
			fields:    []string{"$"},
			variables: map[string][]string{},
			wantError: "invalid variable reference",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("input fields=%#v, variables=%#v; expected argv=%#v, expected error contains=%q", tt.fields, tt.variables, tt.want, tt.wantError)
			got, err := expandLine(tt.fields, tt.variables)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("expandLine() error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("expandLine() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expandLine() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestExecuteSectionDryRunExpandsAndDoesNotExecute(t *testing.T) {
	commands := []string{"echo $MESSAGE", "echo plain"}
	variables := map[string][]string{"MESSAGE": {"hello", "world"}}
	t.Logf("input section=%q, commands=%#v, variables=%#v, dryRun=true, verbose=false; expected output=%q and no command execution", "preview", commands, variables, "DRY: echo hello world\nDRY: echo plain\n")
	stdout, stderr, read := captureFiles(t)
	errs := executeSection("preview", commands, variables, true, false, stdout, stderr)
	if len(errs) != 0 {
		t.Fatalf("executeSection() errors = %v", errs)
	}
	out, errOut := read()
	if want := "DRY: echo hello world\nDRY: echo plain\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestExecuteSectionReportsErrorsAndContinues(t *testing.T) {
	commands := []string{"command-that-does-not-exist-fex", "echo after-error"}
	t.Logf("input section=%q, commands=%#v, variables=nil, dryRun=false, verbose=false; expected one command error and execution of the second command", "build", commands)
	stdout, stderr, read := captureFiles(t)
	errs := executeSection("build", commands, nil, false, false, stdout, stderr)
	if len(errs) != 1 {
		t.Fatalf("executeSection() returned %d errors, want 1: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), `section "build", command 1`) {
		t.Errorf("error lacks section/command context: %v", errs[0])
	}
	out, _ := read()
	if !strings.Contains(out, "after-error") {
		t.Errorf("later command did not run after failure; stdout = %q", out)
	}
}

func TestExecuteSectionVerboseOutput(t *testing.T) {
	commands := []string{"echo visible"}
	t.Logf("input section=%q, commands=%#v, variables=nil, dryRun=false, verbose=true; expected CMD line and process output", "show", commands)
	stdout, stderr, read := captureFiles(t)
	errs := executeSection("show", commands, nil, false, true, stdout, stderr)
	if len(errs) != 0 {
		t.Fatalf("executeSection() errors = %v", errs)
	}
	out, _ := read()
	if !strings.Contains(out, "CMD: echo visible\n") || !strings.Contains(out, "visible\n") {
		t.Errorf("verbose output = %q, want command and process output", out)
	}
}

func TestExecCmdRejectsEmptyCommand(t *testing.T) {
	t.Log("input command argv=nil; expected error=empty command")
	stdout, stderr, _ := captureFiles(t)
	err := execCmd(nil, stdout, stderr)
	if err == nil || err.Error() != "empty command" {
		t.Fatalf("execCmd(empty) error = %v, want empty command", err)
	}
}
