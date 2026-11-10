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

	input := "# comment\r\nCC = gcc -Wall\r\nbuild:\r\n  $CC main.c\r\n\r\n  # ignored\r\nend\r\nrun:\r\n  ./app\r\nend\r\n"
	t.Logf("input (Fex contents): %q; expected variables=%#v, sections=%#v, order=%#v", input, map[string][]string{"CC": {"gcc", "-Wall"}}, map[string][]string{"build": {"$CC main.c"}, "run": {"./app"}}, []string{"build", "run"})
	program, err := parseData(input)
	if err != nil {
		t.Fatalf("parseData() error = %v", err)
	}
	if got, want := program.variables["CC"], []string{"gcc", "-Wall"}; !reflect.DeepEqual(got, want) {
		t.Errorf("variables[CC] = %#v, want %#v", got, want)
	}
	if got, want := program.sections["build"].commands, []string{"$CC main.c"}; !reflect.DeepEqual(got, want) {
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
			t.Logf("input (Fex contents): %q; expected error contains: %q", tt.data, tt.want)
			_, err := parseData(tt.data)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseData() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestParseDataReportsMalformedAssignments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want string
	}{
		{"empty variable name", "= value\n", "variable name cannot be empty"},
		{"invalid variable name", "BAD NAME = value\n", "invalid variable name"},
		{"incomplete directory attribution", "FILES /\n", "not enough tokens"},
		{"directory does not exist", "FILES /*.go /directory/that/does/not/exist\n", "no such file or directory"},
		{"missing transform separator", "OBJECTS -> $SOURCES *.o\n", "requires ':'"},
		{"wrong number of transform modes", "OBJECTS -> $SOURCES : *.c\n", "unexpected amount of modes"},
		{"undefined transform input", "OBJECTS -> $SOURCES : *.c *.o\n", `undefined variable "SOURCES"`},
		{"unexpected content", "just text\n", "unexpected content"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("input (.fex contents): %q; expected error contains: %q", tt.data, tt.want)
			_, err := parseData(tt.data)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("parseData() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestParseDataDirectoryAndTransformAssignments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"alpha.c", "beta.c", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	data := "SOURCES /*.c " + dir + "\nOBJECTS -> $SOURCES : *.c *.o\n"
	t.Logf("input (.fex contents): %q; fixture directory entries: alpha.c, beta.c, notes.txt; expected SOURCES=[alpha.c beta.c], OBJECTS=[alpha.o beta.o]", data)
	program, err := parseData(data)
	if err != nil {
		t.Fatalf("parseData() error = %v", err)
	}
	if got, want := program.variables["SOURCES"], []string{filepath.Join(dir, "alpha.c"), filepath.Join(dir, "beta.c")}; !reflect.DeepEqual(got, want) {
		t.Errorf("SOURCES = %#v, want %#v", got, want)
	}
	if got, want := program.variables["OBJECTS"], []string{filepath.Join(dir, "alpha.o"), filepath.Join(dir, "beta.o")}; !reflect.DeepEqual(got, want) {
		t.Errorf("OBJECTS = %#v, want %#v", got, want)
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
			t.Logf("input filename=%q, mode=%q; expected match=%t", "test-main.go", tt.mode, tt.want)
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
	t.Logf("input directory=%q; entries=[z.go a.go readme.txt folder.go]; mode=%q; expected regular files=[a.go z.go]", dir, "*.go")

	got, err := getFiles(dir, "*.go")
	if err != nil {
		t.Fatalf("getFiles() error = %v", err)
	}
	want := []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "z.go")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getFiles() = %#v, want %#v", got, want)
	}
}

func TestTransformString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input, inputMode, outputMode, want string
	}{
		{"replace suffix", "alpha.c", "*.c", "*.o", "alpha.o"},
		{"remove suffix", "alpha.c", "*.c", "*", "alpha"},
		{"add prefix and suffix", "alpha.c", "*.c", "obj_*.o", "obj_alpha.o"},
		{"nonmatching input unchanged", "readme.txt", "*.c", "*.o", "readme.txt"},
		{"identical modes unchanged", "alpha.c", "*.c", "*.c", "alpha.c"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("input=%q, input mode=%q, output mode=%q; expected=%q", tt.input, tt.inputMode, tt.outputMode, tt.want)
			if got := transformString(tt.input, tt.inputMode, tt.outputMode); got != tt.want {
				t.Errorf("transformString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func FuzzParseData(f *testing.F) {
	for _, seed := range []string{
		"",
		"# comment\n",
		"build:\n go build ./...\nend\n",
		"FILES /*.go src\n",
		"OUT -> $IN : *.c *.o\n",
		"broken:\n",
		"= value\n",
	} {
		f.Logf("fuzz seed input: %q", seed)
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// Malformed user input must return an error rather than panic.
		_, _ = parseData(input)
	})
}
