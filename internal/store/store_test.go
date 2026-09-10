package store

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func model(name string) archdoc.Model {
	return archdoc.Model{
		Name:   name,
		Source: "docker-compose.yml",
		Nodes: []archdoc.Node{{
			ID: "svc:api", Name: "api", Kind: archdoc.Application,
			Evidence: archdoc.Declared,
			Prov:     archdoc.Provenance{File: "docker-compose.yml", Line: 3},
		}},
	}
}

func history(t *testing.T) *Store {
	t.Helper()

	s, err := Memory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// MEM-01 — an immutable version per run, tagged with the commit it was taken from.
func TestSaveRecordsAVersion(t *testing.T) {
	s := history(t)

	id, changed, err := s.Save(model("example"), "abc123", "test")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !changed {
		t.Error("the first save reported no change")
	}

	got, err := s.Get(id)
	if err != nil || got == nil {
		t.Fatalf("get: %v", err)
	}
	if got.Commit != "abc123" {
		t.Errorf("commit is %q, want abc123", got.Commit)
	}
	// MEM-03 — a historical version has to come back whole, provenance included. A stored
	// model that lost its citations would be a picture again.
	if got.Model.Name != "example" || len(got.Model.Nodes) != 1 {
		t.Fatalf("the model did not survive the round trip: %+v", got.Model)
	}
	if got.Model.Nodes[0].Prov.Line != 3 {
		t.Error("provenance was lost in storage")
	}
}

// AC-7 guarantees byte-identical output across runs. Recording every run would therefore fill
// history with entries differing only in their timestamp, and a diff between two of them would
// be empty. A run that changes nothing must not create a version.
func TestUnchangedRunsDoNotCreateVersions(t *testing.T) {
	s := history(t)

	first, _, err := s.Save(model("example"), "abc123", "test")
	if err != nil {
		t.Fatal(err)
	}

	for i := range 5 {
		id, changed, err := s.Save(model("example"), "abc123", "test")
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Errorf("run %d recorded a version for an unchanged model", i+2)
		}
		if id != first {
			t.Errorf("run %d returned version %d, want the existing %d", i+2, id, first)
		}
	}

	vs, err := s.Versions(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Errorf("six identical runs produced %d versions, want 1", len(vs))
	}
}

// A model that differs — even by one node — is a new version, because that is exactly the moment
// a reader wants to know something moved.
func TestAChangedModelCreatesANewVersion(t *testing.T) {
	s := history(t)

	if _, _, err := s.Save(model("example"), "abc123", "test"); err != nil {
		t.Fatal(err)
	}

	grown := model("example")
	grown.Nodes = append(grown.Nodes, archdoc.Node{
		ID: "svc:db", Name: "db", Kind: archdoc.Datastore, Evidence: archdoc.Declared,
		Prov: archdoc.Provenance{File: "docker-compose.yml", Line: 9},
	})

	if _, changed, err := s.Save(grown, "def456", "test"); err != nil || !changed {
		t.Fatalf("adding a node did not record a version (err=%v)", err)
	}

	vs, err := s.Versions(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 {
		t.Fatalf("got %d versions, want 2", len(vs))
	}
	// Newest first: a listing is read from the top.
	if vs[0].Commit != "def456" {
		t.Errorf("history is not newest-first: %+v", vs)
	}
}

// The same model at a different commit is still the same architecture. Recording a version for
// it would report change where there was none — which is the noise MEM-04 exists to avoid.
func TestSameModelAtANewCommitIsNotAChange(t *testing.T) {
	s := history(t)

	if _, _, err := s.Save(model("example"), "abc123", "test"); err != nil {
		t.Fatal(err)
	}

	if _, changed, err := s.Save(model("example"), "def456", "test"); err != nil || changed {
		t.Errorf("a commit that changed no architecture recorded a version (err=%v)", err)
	}
}

func TestLatestOnEmptyHistoryIsNotAnError(t *testing.T) {
	v, err := history(t).Latest()
	if err != nil {
		t.Fatalf("an empty history was treated as an error: %v", err)
	}
	if v != nil {
		t.Errorf("got a version from an empty history: %+v", v)
	}
}

// Opening the same repository twice must find the history, not start a new one.
func TestHistorySurvivesReopening(t *testing.T) {
	root := t.TempDir()

	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := first.Save(model("example"), "abc123", "test"); err != nil {
		t.Fatal(err)
	}
	first.Close()

	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	v, err := second.Latest()
	if err != nil || v == nil {
		t.Fatalf("history did not survive reopening: %v", err)
	}
	if v.Model.Name != "example" {
		t.Errorf("got %q back", v.Model.Name)
	}
}
