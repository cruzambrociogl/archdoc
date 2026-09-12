package render

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Completeness is how far a human-owned section has come (OUT-09, SUR-15).
type Completeness string

const (
	// Missing means the file is not there — deleted, or never generated.
	Missing Completeness = "missing"
	// NotStarted means the file is still exactly the stub archdoc wrote.
	NotStarted Completeness = "not started"
	// Written means someone has changed it since archdoc created it.
	Written Completeness = "written"
	// MayBeStale means it was written, and the architecture has changed since.
	MayBeStale Completeness = "may be stale"
)

// SectionStatus is one human-owned section and its state.
type SectionStatus struct {
	Section  Section
	File     string // relative to the documentation directory
	State    Completeness
	Modified time.Time // zero when missing
}

// stubsFile records what each stub looked like when archdoc created it.
const stubsFile = ".archdoc/sections.json"

// stub is what archdoc knows about a file it created: its size and modification time at that
// moment. Nothing about its content.
type stub struct {
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// RecordStub remembers a stub archdoc has just written, so a later run can tell whether anyone
// has touched it.
//
// Size and modification time, not a hash of the content. OUT-03 says archdoc never reads a
// human-owned file, and hashing one is reading it. The filesystem already knows whether a file
// changed; asking it is enough, and it keeps the rule absolute rather than "never reads, except
// to hash".
func RecordStub(root, dir, name string) error {
	info, err := os.Stat(filepath.Join(root, dir, name))
	if err != nil {
		return err
	}
	stubs, err := loadStubs(root)
	if err != nil {
		return err
	}
	stubs[name] = stub{Size: info.Size(), ModTime: info.ModTime().UTC()}
	return saveStubs(root, stubs)
}

// SectionStates reports every human-owned section, by asking the filesystem and never opening a
// file. architectureChanged is when the model last changed; a section written before that may
// no longer describe the system.
func SectionStates(root, dir string, architectureChanged time.Time) ([]SectionStatus, error) {
	stubs, err := loadStubs(root)
	if err != nil {
		return nil, err
	}

	var out []SectionStatus
	for _, s := range Sections() {
		if s.Owner != Human {
			continue
		}
		st := SectionStatus{Section: s, File: s.File()}

		info, err := os.Stat(filepath.Join(root, dir, s.File()))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			st.State = Missing
		case err != nil:
			return nil, err
		default:
			st.Modified = info.ModTime().UTC()
			rec, known := stubs[s.File()]
			switch {
			case known && rec.Size == info.Size() && rec.ModTime.Equal(st.Modified):
				st.State = NotStarted
			case !architectureChanged.IsZero() && st.Modified.Before(architectureChanged):
				st.State = MayBeStale
			default:
				st.State = Written
			}
		}
		out = append(out, st)
	}
	return out, nil
}

func loadStubs(root string) (map[string]stub, error) {
	b, err := os.ReadFile(filepath.Join(root, stubsFile))
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]stub{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]stub{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func saveStubs(root string, stubs map[string]stub) error {
	b, err := json.MarshalIndent(stubs, "", "  ") // map keys marshal sorted: AC-7 holds
	if err != nil {
		return err
	}
	path := filepath.Join(root, stubsFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
