// Package system reads several repositories together (platform-surface §4.6): each one's latest
// stored version, and the links between them that their manifests prove — cited at both ends: the
// requirement in one, the package's own name in the other. A link is never guessed; a name nothing
// here publishes is simply not a link.
//
// Only code links are read today: one package requiring another's, by its exact name. Images, calls,
// shared contracts and distribution (§4.6) come later.
package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// System is a named group of repositories, and what links them.
type System struct {
	Name  string `json:"name"`
	Repos []Repo `json:"repos"`
	Links []Link `json:"links"`
}

// Repo is one repository of a system, as its latest version documents it.
type Repo struct {
	Name    string     `json:"name"`
	Root    string     `json:"root"`
	Version int64      `json:"version"`
	Commit  string     `json:"commit,omitempty"`
	At      time.Time  `json:"created_at"`
	Model   core.Model `json:"-"`
}

// Link is one repository's package requiring another's.
type Link struct {
	From string `json:"from"` // the repository that requires
	To   string `json:"to"`   // the repository whose package it is
	Kind string `json:"kind"` // "code"
	// Package is the name required, exactly as both manifests write it; Version as From requires it.
	Package string `json:"package"`
	Version string `json:"version,omitempty"`
	// Uses is the requirement's line in From; Declares the package's name in To.
	Uses     core.Provenance `json:"uses"`
	Declares core.Provenance `json:"declares"`
	Pin      *Pin            `json:"pin,omitempty"`
}

// Pin is which of To's code a requirement uses, where the version says: a commit, a tag, or the
// checkout beside it — and, for a commit or a tag, how far To's documented version has moved since.
type Pin struct {
	Kind string `json:"kind"` // "commit", "tag", "checkout"
	Ref  string `json:"ref"`
	// Ahead is how many commits To's documented version has after Ref; -1 when it cannot be counted.
	Ahead int `json:"ahead"`
}

// Ahead counts the commits a repository has after one revision, up to another (store.Ahead).
type Ahead func(root, from, to string) (int, error)

// Read finds the repositories directly inside dir that archdoc has documented, and composes them.
// A directory that is itself documented is a repository, not a system.
func Read(dir string) (System, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return System{}, err
	}
	if IsRepo(abs) {
		return System{}, fmt.Errorf("%s is a repository, not a folder of them", abs)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return System{}, err
	}
	var repos []Repo
	for _, e := range entries { // ReadDir sorts by name
		root := filepath.Join(abs, e.Name())
		if !e.IsDir() || !IsRepo(root) {
			continue
		}
		s, err := store.Open(root)
		if err != nil {
			return System{}, err
		}
		v, err := s.Latest()
		s.Close()
		if err != nil {
			return System{}, err
		}
		if v == nil {
			continue
		}
		repos = append(repos, Repo{Name: e.Name(), Root: root, Version: v.ID, Commit: v.Commit, At: v.CreatedAt, Model: v.Model})
	}
	if len(repos) < 2 {
		return System{}, errors.New("a system is two or more documented repositories, each in a folder directly inside it")
	}
	return Compose(filepath.Base(abs), repos, store.Ahead), nil
}

// IsRepo says whether archdoc has documented the repository at root.
func IsRepo(root string) bool {
	_, err := os.Stat(filepath.Join(root, store.File))
	return err == nil
}

// pseudo is a Go pseudo-version's tail: the commit time, then the commit's first twelve hex digits.
var pseudo = regexp.MustCompile(`-\d{14}-([0-9a-f]{12})$`)

// release is a version that names one release: v1.2.3, 1.2.3, with a pre-release after a dash.
var release = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// Compose links the repositories: every requirement in one that names a package another publishes.
func Compose(name string, repos []Repo, ahead Ahead) System {
	type published struct {
		repo     int
		manifest core.Manifest
	}
	byName := map[string][]published{}
	for i, r := range repos {
		for _, m := range r.Model.Manifests {
			byName[key(m.Package, m.Language)] = append(byName[key(m.Package, m.Language)], published{i, m})
		}
	}

	sys := System{Name: name, Repos: repos, Links: []Link{}}
	for i, r := range repos {
		for _, m := range r.Model.Manifests {
			for _, req := range m.Requires {
				for _, p := range byName[key(req.Name, m.Language)] {
					if p.repo == i {
						continue // a package of its own repository: not a link between two
					}
					to := repos[p.repo]
					sys.Links = append(sys.Links, Link{
						From: r.Name, To: to.Name, Kind: "code", Package: req.Name, Version: req.Version,
						Uses: req.Prov, Declares: p.manifest.Prov,
						Pin: pin(req.Version, filepath.Join(r.Root, m.Dir), filepath.Join(to.Root, p.manifest.Dir), to, ahead),
					})
				}
			}
		}
	}
	sort.SliceStable(sys.Links, func(i, j int) bool {
		a, b := sys.Links[i], sys.Links[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		return a.Uses.File < b.Uses.File
	})
	return sys
}

// key is how a requirement finds a package: exactly, except Python's, whose names ignore case and
// treat - and _ alike.
func key(name, language string) string {
	if language == "Python" {
		return "py:" + strings.ReplaceAll(strings.ToLower(name), "-", "_")
	}
	return name
}

// pin reads which of the package's code a version requirement uses. A version range names no one
// commit, and has no pin.
func pin(version, fromDir, toDir string, to Repo, ahead Ahead) *Pin {
	count := func(ref string) int {
		if to.Commit == "" || ahead == nil {
			return -1
		}
		n, err := ahead(to.Root, ref, to.Commit)
		if err != nil {
			return -1
		}
		return n
	}
	if m := pseudo.FindStringSubmatch(version); m != nil {
		return &Pin{Kind: "commit", Ref: m[1], Ahead: count(m[1])}
	}
	if release.MatchString(version) {
		return &Pin{Kind: "tag", Ref: version, Ahead: count(version)}
	}
	for _, scheme := range []string{"file:", "link:"} {
		if p, ok := strings.CutPrefix(version, scheme); ok && filepath.Clean(filepath.Join(fromDir, p)) == filepath.Clean(toDir) {
			return &Pin{Kind: "checkout", Ref: p, Ahead: 0}
		}
	}
	return nil
}

// Versions is each repository's latest version at a place: one repository, or every one of a
// system — what Home remembers as seen when the place is opened.
func Versions(path string) (map[string]int64, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	roots := []string{abs}
	if !IsRepo(abs) {
		sys, err := Read(abs)
		if err != nil {
			return nil, err
		}
		roots = roots[:0]
		for _, r := range sys.Repos {
			roots = append(roots, r.Root)
		}
	}
	out := map[string]int64{}
	for _, root := range roots {
		s, err := store.Open(root)
		if err != nil {
			return nil, err
		}
		v, err := s.Latest()
		s.Close()
		if err != nil {
			return nil, err
		}
		if v != nil {
			out[root] = v.ID
		}
	}
	return out, nil
}
