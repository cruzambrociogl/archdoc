// Package recent is this machine's list of the places archdoc has documented — repositories, and
// folders of them — which Home shows (platform-surface §5.3). It is one file in the user's
// configuration folder, never in a repository: what someone has looked at on their machine is
// theirs, and no repository's history.
//
// Each place remembers, per repository, the version it was at when last opened, so Home can say
// what changed since you last looked — "since you last opened it on this machine" (§4.5).
package recent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Most is how many places the list keeps; the least recent goes first.
const Most = 20

// Place is one documented folder: a repository, or a system of them.
type Place struct {
	Path string `json:"path"`
	// Opened is when it was last opened in the app; zero when it never was, only generated.
	Opened time.Time `json:"opened,omitzero"`
	// Touched is when archdoc last generated or served it: the list's order.
	Touched time.Time `json:"touched"`
	// Seen is each repository's version when the place was last opened, by its path.
	Seen map[string]int64 `json:"seen,omitempty"`
}

// ID names a place in an address: stable for its path, and saying nothing of it.
func ID(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])[:10]
}

// File is where the list is kept. A variable, so a test can keep it out of the user's own.
var File = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "archdoc", "recent.json"), nil
}

// List is every remembered place, the most recently touched first. No file is an empty list.
func List() ([]Place, error) {
	file, err := File()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out struct {
		Places []Place `json:"places"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out.Places, nil
}

// Touch remembers a place archdoc just generated or served, moving it to the top.
func Touch(path string, at time.Time) error {
	return change(path, func(p *Place) { p.Touched = at })
}

// Open records that a place was opened, and each repository's version then.
func Open(path string, at time.Time, seen map[string]int64) error {
	return change(path, func(p *Place) { p.Touched, p.Opened, p.Seen = at, at, seen })
}

func change(path string, f func(*Place)) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	places, err := List()
	if err != nil {
		places = nil // a list that cannot be read is started again, not left in the way
	}
	place := Place{Path: abs}
	kept := places[:0]
	for _, p := range places {
		if p.Path == abs {
			place = p
		} else {
			kept = append(kept, p)
		}
	}
	f(&place)
	places = append(kept, place)
	sort.SliceStable(places, func(i, j int) bool { return places[i].Touched.After(places[j].Touched) })
	if len(places) > Most {
		places = places[:Most]
	}
	return save(places)
}

// save writes the list whole, through a temporary file, so a reader never sees half of it.
func save(places []Place) error {
	file, err := File()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Places []Place `json:"places"`
	}{places}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), "recent-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), file)
}
