package archdoc

import (
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// History is what archdoc has recorded for a repository on this machine: every version of the
// architecture, and every run that used the network with what it sent. A local cache — git is the
// archive (decisions, 2026-09-10).
type History struct {
	// Root is the repository, as Read resolves it.
	Root string
	s    *store.Store
}

// OpenHistory opens a repository's history for reading. It never creates one: a repository with
// none has a History whose Recorded is false and whose lists are empty.
func OpenHistory(root string) (*History, error) {
	facts, err := extract.Scan(root)
	if err != nil {
		return nil, err
	}
	h := &History{Root: facts.Root}
	s, ok, err := openHistory(facts.Root)
	if err != nil {
		return nil, err
	}
	if ok {
		h.s = s
	}
	return h, nil
}

// Fingerprint is what decides whether a model is a new version, as history decides it: a hash of
// the architecture, not of the model's encoding. A citation whose line moved, or a field added to
// the schema, leaves it unchanged (F-50); two models with one fingerprint are one version. A caller
// keeping models elsewhere uses it to keep versions as archdoc does — and hashes the bytes itself
// when it needs to prove a stored file is the one written.
func Fingerprint(m Model) (string, error) { return store.Fingerprint(m) }

// Recorded reports whether the repository has a history at all.
func (h *History) Recorded() bool { return h.s != nil }

// Versions are the latest n versions, newest first.
func (h *History) Versions(n int) ([]Version, error) {
	if h.s == nil {
		return nil, nil
	}
	return h.s.Versions(n)
}

// Runs are the latest n runs that used the network, newest first.
func (h *History) Runs(n int) ([]Run, error) {
	if h.s == nil {
		return nil, nil
	}
	return h.s.Runs(n)
}

// Run is one run, with every request it sent and what came back; nil when there is none.
func (h *History) Run(id int64) (*Run, error) {
	if h.s == nil {
		return nil, nil
	}
	return h.s.Run(id)
}

// Close closes the history.
func (h *History) Close() error {
	if h.s == nil {
		return nil
	}
	return h.s.Close()
}
