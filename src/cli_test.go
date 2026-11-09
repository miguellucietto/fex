package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests exercise the real executable from a directory containing its own
// Fex file, just as users invoke fex in normal use.
func TestCLIIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("integration fixtures use Unix command names")
	}

	binDir := t.TempDir()
	bin := filepath.Join(binDir, "fex")
	build := exec.Command("go", "build", "-o", bin, ".")
	t.Logf("integration setup: build command=%#v; binary=%q", build.Args, bin)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fex: %v\n%s", err, output)
	}

	tests := []struct {
		name       string
		config     string
		args       []string
		wantOut    []string
		wantErrOut []string
		wantFail   bool
	}{
		{
			name:       "help works without a config file",
			args:       []string{"-help"},
			wantErrOut: []string{"Usage: fex", "-dry", "-list"},
		},
		{
			name:    "lists sections in declaration order",
			config:  "first:\nend\nsecond:\nend\n",
			args:    []string{"-list"},
			wantOut: []string{"Sections available:", "- first", "- second"},
		},
		{
			name:    "dry run expands variables without executing",
			config:  "MESSAGE = should-not-print\nshow:\n    echo $MESSAGE\nend\n",
			args:    []string{"-dry", "show"},
			wantOut: []string{"DRY: echo should-not-print"},
		},
		{
			name:    "runs requested section",
			config:  "show:\n    echo integration-ok\nend\n",
			args:    []string{"show"},
			wantOut: []string{"integration-ok"},
		},
		{
			name:       "reports missing section",
			config:     "known:\nend\n",
			args:       []string{"missing"},
			wantErrOut: []string{`section "missing" does not exist`},
			wantFail:   true,
		},
		{
			name:       "reports malformed config",
			config:     "broken:\n echo never-closed\n",
			args:       []string{"broken"},
			wantErrOut: []string{"missing 'end'"},
			wantFail:   true,
		},
		{
			name:       "reports command failure with section context",
			config:     "broken:\n    fex-command-that-does-not-exist\nend\n",
			args:       []string{"broken"},
			wantErrOut: []string{`section "broken", command 1`, "executable file not found"},
			wantFail:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			t.Logf("input CLI args=%#v; Fex contents=%q; expected stdout contains=%#v; expected stderr contains=%#v; expected failure=%t", tt.args, tt.config, tt.wantOut, tt.wantErrOut, tt.wantFail)
			if tt.config != "" {
				if err := os.WriteFile(filepath.Join(workDir, "Fex"), []byte(tt.config), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			cmd := exec.Command(bin, tt.args...)
			cmd.Dir = workDir
			stdout, stderr := new(strings.Builder), new(strings.Builder)
			cmd.Stdout, cmd.Stderr = stdout, stderr
			err := cmd.Run()
			if tt.wantFail && err == nil {
				t.Fatal("command succeeded, want failure")
			}
			if !tt.wantFail && err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
			}
			for _, want := range tt.wantOut {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("stdout %q does not contain %q", stdout.String(), want)
				}
			}
			for _, want := range tt.wantErrOut {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr %q does not contain %q", stderr.String(), want)
				}
			}
		})
	}
}
