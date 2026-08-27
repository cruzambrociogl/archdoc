package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		wantMatch string
	}{
		{
			name:      "no arguments prints usage",
			args:      nil,
			wantMatch: "usage:",
		},
		{
			name:      "help prints usage",
			args:      []string{"help"},
			wantMatch: "usage:",
		},
		{
			name:      "version prints the build",
			args:      []string{"version"},
			wantMatch: "dev",
		},
		{
			name:    "an unknown command is an error",
			args:    []string{"scan"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer

			err := run(tt.args, &out)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("run(%v) = nil, want an error", tt.args)
				}
				return
			}

			if err != nil {
				t.Fatalf("run(%v) = %v, want nil", tt.args, err)
			}

			if !strings.Contains(out.String(), tt.wantMatch) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantMatch)
			}
		})
	}
}
