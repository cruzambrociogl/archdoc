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

// Remote is the web address of the repository's origin, when it is a host whose file links are
// known (GitHub, GitLab, Codeberg, Bitbucket), so a published site can open a citation at its line.
// Read from .git/config directly, like Commit. Empty when there is no origin or the host is
// unknown — the published site then shows paths as plain text rather than guessing a link.
func Remote(root string) string {
	b, err := os.ReadFile(filepath.Join(root, ".git", "config"))
	if err != nil {
		return ""
	}
	inOrigin := false
	for _, line := range strings.Split(string(b), "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "[") {
			inOrigin = l == `[remote "origin"]`
			continue
		}
		if !inOrigin {
			continue
		}
		if k, v, ok := strings.Cut(l, "="); ok && strings.TrimSpace(k) == "url" {
			return webURL(strings.TrimSpace(v))
		}
	}
	return ""
}

// webURL turns a clone address into the repository's web address, or "" for an unknown host.
func webURL(u string) string {
	u = strings.TrimSuffix(u, ".git")
	switch {
	case strings.HasPrefix(u, "git@"): // git@github.com:owner/repo
		host, path, ok := strings.Cut(strings.TrimPrefix(u, "git@"), ":")
		if !ok {
			return ""
		}
		u = "https://" + host + "/" + path
	case strings.HasPrefix(u, "ssh://git@"):
		u = "https://" + strings.TrimPrefix(u, "ssh://git@")
	case strings.HasPrefix(u, "http://"), strings.HasPrefix(u, "https://"):
		// Drop any credentials a clone URL may carry.
		_, rest, _ := strings.Cut(u, "://")
		if at := strings.LastIndex(strings.SplitN(rest, "/", 2)[0], "@"); at >= 0 {
			rest = rest[at+1:]
		}
		u = "https://" + rest
	default:
		return ""
	}
	for _, host := range []string{"github.com/", "gitlab.com/", "codeberg.org/", "bitbucket.org/"} {
		if strings.HasPrefix(strings.TrimPrefix(u, "https://"), host) {
			return u
		}
	}
	return ""
}
