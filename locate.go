package archdoc

import (
	"os"
	"path/filepath"
	"sort"
)

// Where is what a bare `archdoc` documents, found from the folder it was started in (decisions,
// 18 Nov 2026).
type Where struct {
	// Kind is "repository" — one project, at Root; "folder" — a folder holding repositories, each
	// one in Repos; or "none" — nothing to document from here, so only the library opens.
	Kind string
	// Root is the repository, or the folder of them. Absolute.
	Root string
	// Repos are a folder's repositories, absolute and sorted.
	Repos []string
}

// Locate finds what to document from dir. Inside a git repository, it is the repository, wherever
// in it dir is. Outside one, a folder with repositories in it is a folder of them; any other folder
// is documented as it is — a folder of scripts has no git. The home folder and the filesystem's root
// are never documented whole: from there only the library opens.
func Locate(dir string) (Where, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Where{}, err
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		home = filepath.Clean(home)
	}
	if root, ok := gitRoot(abs, home); ok {
		return Where{Kind: "repository", Root: root}, nil
	}
	if abs == home || filepath.Dir(abs) == abs {
		return Where{Kind: "none", Root: abs}, nil
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return Where{}, err
	}
	var repos []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		child := filepath.Join(abs, e.Name())
		if isGit(child) {
			repos = append(repos, child)
		}
	}
	if len(repos) > 0 {
		sort.Strings(repos)
		return Where{Kind: "folder", Root: abs, Repos: repos}, nil
	}
	return Where{Kind: "repository", Root: abs}, nil
}

// gitRoot is the nearest folder at or above dir with a .git — a folder, or the file a worktree or
// submodule has. The home folder is never one, even when it is a repository of someone's dotfiles.
func gitRoot(dir, home string) (string, bool) {
	for d := dir; ; {
		if d == home {
			return "", false
		}
		if isGit(d) {
			return d, true
		}
		up := filepath.Dir(d)
		if up == d {
			return "", false
		}
		d = up
	}
}

func isGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}
