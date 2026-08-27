package archdoc

import "testing"

func TestBuildInfoString(t *testing.T) {
	tests := []struct {
		name string
		in   BuildInfo
		want string
	}{
		{
			name: "version only, no VCS information",
			in:   BuildInfo{Version: "dev"},
			want: "dev",
		},
		{
			name: "revision is shortened to seven characters",
			in:   BuildInfo{Version: "v0.1.0", Revision: "a1b2c3d4e5f6"},
			want: "v0.1.0 (a1b2c3d)",
		},
		{
			name: "a dirty tree is reported",
			in:   BuildInfo{Version: "dev", Revision: "a1b2c3d4e5f6", Modified: true},
			want: "dev (a1b2c3d, modified)",
		},
		{
			name: "a revision shorter than seven characters is left alone",
			in:   BuildInfo{Version: "dev", Revision: "abc"},
			want: "dev (abc)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestBuildDoesNotPanic covers the path where the toolchain embeds no VCS information — a
// plain "go run", for instance. Build must degrade rather than fail.
func TestBuildDoesNotPanic(t *testing.T) {
	b := Build()

	if b.Version == "" {
		t.Error("Build().Version is empty; it should always carry at least \"dev\"")
	}
}
