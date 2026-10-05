package arrange

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Where the files live, inside archdoc's own directory — part of the closed write set.
const (
	Dir        = ".archdoc"
	LayoutFile = "layout.yaml"
	ViewsFile  = "views.yaml"
)

// ErrChanged is returned when a file changed on disk after the app loaded it (ANS-04): writing
// now could overwrite someone else's edit, so nothing is written.
var ErrChanged = errors.New("changed on disk since it was loaded — reload and try again")

// Hash identifies a file's content as loaded; "" means the file does not exist.
func Hash(b []byte) string {
	if b == nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// read returns a file's content and hash; a missing file is empty, not an error.
func read(root, name string) ([]byte, string, error) {
	b, err := os.ReadFile(filepath.Join(root, Dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return b, Hash(b), nil
}

// write replaces a file atomically, but only if it still holds what the caller loaded.
func write(root, name, expect string, content []byte) (string, error) {
	_, current, err := read(root, name)
	if err != nil {
		return "", err
	}
	if current != expect {
		return "", fmt.Errorf("%s/%s %w", Dir, name, ErrChanged)
	}
	dir := filepath.Join(root, Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, name)); err != nil {
		return "", err
	}
	return Hash(content), nil
}

// ——— layout.yaml ———

// LoadLayout reads .archdoc/layout.yaml. A missing file is an empty arrangement.
func LoadLayout(root string) (Arrangement, string, error) {
	b, hash, err := read(root, LayoutFile)
	if err != nil || b == nil {
		return Arrangement{}, hash, err
	}
	var raw map[string]map[string]struct {
		X float64 `yaml:"x"`
		Y float64 `yaml:"y"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, "", fmt.Errorf("%s/%s: %w", Dir, LayoutFile, err)
	}
	a := Arrangement{}
	for view, boxes := range raw {
		a[view] = map[string]archdoc.Point{}
		for id, p := range boxes {
			a[view][id] = archdoc.Point{X: p.X, Y: p.Y}
		}
	}
	return a, hash, nil
}

// SaveLayout writes the arrangement, refusing if the file changed since expect was loaded.
func SaveLayout(root string, a Arrangement, expect string) (string, error) {
	return write(root, LayoutFile, expect, FormatLayout(a))
}

// FormatLayout is the file's text: sorted, whole units, one box per line, so that it diffs
// cleanly in review and two saves of the same arrangement are byte-identical.
func FormatLayout(a Arrangement) []byte {
	var b strings.Builder
	b.WriteString("# Written by archdoc serve when a diagram is arranged, and re-applied on every run.\n")
	b.WriteString("# Position is presentation, not fact: nothing here can add, remove or rename an element.\n")
	b.WriteString("# Each entry is the top-left corner of an element's box. Delete a view to reset it.\n")
	for _, view := range sortedKeys(a) {
		if len(a[view]) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s:\n", view)
		for _, id := range sortedKeys(a[view]) {
			p := a[view][id]
			fmt.Fprintf(&b, "  %s: { x: %d, y: %d }\n", strconv.Quote(id), int(p.X+0.5), int(p.Y+0.5))
		}
	}
	return []byte(b.String())
}

// ——— views.yaml ———

// View is a named way of looking at the architecture: a level, what is selected, and how it is
// filtered — kept so a team opens the same view by name.
type View struct {
	Name  string `yaml:"name" json:"name"`
	Level string `yaml:"level" json:"level"`
	Focus string `yaml:"focus,omitempty" json:"focus,omitempty"`
	Find  string `yaml:"find,omitempty" json:"find,omitempty"`
	Dim   bool   `yaml:"dim,omitempty" json:"dim,omitempty"`
}

// LoadViews reads .archdoc/views.yaml. A missing file is no views.
func LoadViews(root string) ([]View, string, error) {
	b, hash, err := read(root, ViewsFile)
	if err != nil || b == nil {
		return []View{}, hash, err
	}
	var vs []View
	if err := yaml.Unmarshal(b, &vs); err != nil {
		return nil, "", fmt.Errorf("%s/%s: %w", Dir, ViewsFile, err)
	}
	if vs == nil {
		vs = []View{}
	}
	return vs, hash, nil
}

// SaveViews writes the views in the order given, refusing if the file changed since expect.
func SaveViews(root string, vs []View, expect string) (string, error) {
	body, err := yaml.Marshal(vs)
	if err != nil {
		return "", err
	}
	head := "# Saved views, written by archdoc serve. Commit this file and your team opens them by name.\n"
	return write(root, ViewsFile, expect, append([]byte(head), body...))
}
