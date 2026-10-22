package store

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Ahead counts the commits root's history has after from, up to to: how far a pinned revision is
// behind the one documented (platform-surface §4.6, version drift). Git, as for Tree: the history is
// in git's object store. Without git, or without either revision, there is no count — an error.
func Ahead(root, from, to string) (int, error) {
	for _, rev := range []string{from, to} {
		if rev == "" || strings.HasPrefix(rev, "-") || strings.Contains(rev, "..") {
			return 0, fmt.Errorf("%q is not a revision", rev)
		}
	}
	var problem strings.Builder
	cmd := exec.Command("git", "-C", root, "rev-list", "--count", from+".."+to)
	cmd.Stderr = &problem
	out, err := cmd.Output()
	if err != nil {
		var missing *exec.Error
		if errors.As(err, &missing) {
			return 0, errors.New("counting commits needs git, which was not found")
		}
		return 0, fmt.Errorf("git could not compare %s and %s: %s", from, to, strings.TrimSpace(problem.String()))
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}
