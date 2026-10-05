package extract

import (
	"os"
	"path/filepath"
	"testing"
)

// F-53: the dotenv file interpolation was filled from is reported, and a sample is called one —
// Immich keeps example.env beside its compose file, and a reader should know the values filled
// from it are defaults, not the deployment's.
func TestInterpolationSourceIsReported(t *testing.T) {
	compose := "services:\n  api:\n    image: example/api:${TAG}\n"
	cases := []struct {
		files  []string
		want   string
		sample bool
	}{
		{[]string{"example.env"}, "example.env", true},
		{[]string{".env", "example.env"}, ".env", false},
		{nil, "", false},
	}
	for _, c := range cases {
		root := t.TempDir()
		os.WriteFile(filepath.Join(root, "docker-compose.yml"), []byte(compose), 0o644)
		for _, f := range c.files {
			os.WriteFile(filepath.Join(root, f), []byte("TAG=1.0\n"), 0o644)
		}
		fs, err := Scan(root)
		if err != nil {
			t.Fatal(err)
		}
		got, sample := "", false
		if fs.Interpolation != nil {
			got, sample = fs.Interpolation.File, fs.Interpolation.Sample
		}
		if got != c.want || sample != c.sample {
			t.Errorf("%v: got %q sample=%v, want %q sample=%v", c.files, got, sample, c.want, c.sample)
		}
	}
}
