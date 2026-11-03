package main

import (
	"reflect"
	"strings"
	"testing"
)

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
