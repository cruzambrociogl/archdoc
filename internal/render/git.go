package render

import (
	"os"
	"path/filepath"
	"strings"
)

// Commit reads the checked-out revision of a repository, for the stamp OUT-04 puts on every
// generated file.
//
// Read directly rather than by running git, for the same reason Compose is parsed in-process:
// archdoc must work where the tool is not installed, and a documentation generator that shells
// out acquires a dependency it does not need. Two files, no library.
//
// An empty result is normal — a repository may be a plain directory — and is not an error.
func Commit(root string) string {
	head, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(head))

	// A detached HEAD holds the hash directly.
	if !strings.HasPrefix(line, "ref:") {
		return short(line)
	}

	ref := strings.TrimSpace(strings.TrimPrefix(line, "ref:"))
	if b, err := os.ReadFile(filepath.Join(root, ".git", filepath.FromSlash(ref))); err == nil {
		return short(strings.TrimSpace(string(b)))
	}

	// A packed ref: the loose file is absent once git has packed it.
	packed, err := os.ReadFile(filepath.Join(root, ".git", "packed-refs"))
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(packed), "\n") {
		if hash, name, ok := strings.Cut(strings.TrimSpace(l), " "); ok && name == ref {
			return short(hash)
		}
	}
	return ""
}

func short(hash string) string {
	if len(hash) < 12 {
		return hash
	}
	return hash[:12]
}
