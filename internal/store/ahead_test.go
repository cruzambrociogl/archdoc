package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Three commits: the first is two behind the last, and the last none behind itself.
func TestAheadCountsTheCommitsSinceAPin(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	var commits []string
	for _, n := range []string{"a", "b", "c"} {
		os.WriteFile(filepath.Join(dir, n), []byte(n), 0o644)
		git("add", n)
		git("commit", "-q", "-m", n)
		commits = append(commits, git("rev-parse", "HEAD")[:12])
	}
	git("tag", "v1.0.0", commits[0])

	for _, c := range []struct {
		from string
		want int
	}{{commits[0], 2}, {"v1.0.0", 2}, {commits[2], 0}} {
		if n, err := Ahead(dir, c.from, commits[2]); err != nil || n != c.want {
			t.Errorf("from %s: %d (%v), want %d", c.from, n, err, c.want)
		}
	}
	for _, bad := range []string{"", "--all", "a..b"} {
		if _, err := Ahead(dir, bad, commits[2]); err == nil {
			t.Errorf("%q was taken for a revision", bad)
		}
	}
}
