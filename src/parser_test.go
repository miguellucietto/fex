package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseData(t *testing.T) {
	t.Parallel()

	program, err := parseData("# comment\r\nCC = gcc -Wall\r\nbuild:\r\n  $CC main.c\r\n\r\n  # ignored\r\nend\r\nrun:\r\n  ./app\r\nend\r\n")
	if err != nil {
		t.Fatalf("parseData() error = %v", err)
	}
	if got, want := program.variables["CC"], []string{"gcc", "-Wall"}; !reflect.DeepEqual(got, want) {
		t.Errorf("variables[CC] = %#v, want %#v", got, want)
	}
	if got, want := program.sections["build"], []string{"$CC main.c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections[build] = %#v, want %#v", got, want)
	}
	if got, want := program.sectionOrder, []string{"build", "run"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sectionOrder = %#v, want %#v", got, want)
	}
}

func TestParseDataReportsMalformedSections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want string
	}{
		{"missing end", "build:\n echo hi\n", "missing 'end'"},
		{"duplicate section", "build:\nend\nbuild:\nend\n", "declared more than once"},
		{"end without section", "end\n", "without an open section"},
		{"empty section name", ":\nend\n", "section name cannot be empty"},
		{"invalid section name", "bad name:\nend\n", "invalid section name"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseData(tt.data)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseData() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestDirectoryModeMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode string
		want bool
	}{
		{"all files", "*", true},
		{"suffix", "*.go", true},
		{"prefix", "test*", true},
		{"prefix mismatch", "main*", false},
		{"both ends", "test*.go", true},
		{"both ends mismatch", "test*.txt", false},
		{"substring", "ain", true},
		{"empty mode", "", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := correspondsMode("test-main.go", tt.mode); got != tt.want {
				t.Errorf("correspondsMode(%q, %q) = %v, want %v", "test-main.go", tt.mode, got, tt.want)
			}
		})
	}
}

func TestGetFilesFiltersAndSortsEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"z.go", "a.go", "readme.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "folder.go"), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := getFiles(dir, "*.go")
	if err != nil {
		t.Fatalf("getFiles() error = %v", err)
	}
	want := []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "z.go")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getFiles() = %#v, want %#v", got, want)
	}
}
