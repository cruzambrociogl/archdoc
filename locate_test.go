package archdoc

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// A bare archdoc documents the repository it is run in, from anywhere inside it; outside one, a
// folder of repositories, or the folder itself; and never the home folder, even when it is a git
// repository of someone's dotfiles.
func TestLocate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows' home
	mk := func(parts ...string) string {
		p := filepath.Join(append([]string{home}, parts...)...)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	mk(".git") // the dotfiles
	repo := mk("code", "shop")
	mk("code", "shop", ".git")
	deep := mk("code", "shop", "server", "src")
	mk("code", "api", ".git")
	scripts := mk("scripts")

	for _, tc := range []struct {
		name, dir string
		want      Where
	}{
		{"inside a repository", deep, Where{Kind: "repository", Root: repo}},
		{"at its root", repo, Where{Kind: "repository", Root: repo}},
		{"a folder of repositories", filepath.Join(home, "code"), Where{Kind: "folder", Root: filepath.Join(home, "code"),
			Repos: []string{filepath.Join(home, "code", "api"), repo}}},
		{"a folder with no git", scripts, Where{Kind: "repository", Root: scripts}},
		{"the home folder", home, Where{Kind: "none", Root: home}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Locate(tc.dir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Locate(%s) = %+v, want %+v", tc.dir, got, tc.want)
			}
		})
	}
}
