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

	id, changed, err := s.Save(model("example"), nil, "abc123", "test")
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

	first, _, err := s.Save(model("example"), nil, "abc123", "test")
	if err != nil {
		t.Fatal(err)
	}

	for i := range 5 {
		id, changed, err := s.Save(model("example"), nil, "abc123", "test")
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

	if _, _, err := s.Save(model("example"), nil, "abc123", "test"); err != nil {
		t.Fatal(err)
	}

	grown := model("example")
	grown.Nodes = append(grown.Nodes, archdoc.Node{
		ID: "svc:db", Name: "db", Kind: archdoc.Datastore, Evidence: archdoc.Declared,
		Prov: archdoc.Provenance{File: "docker-compose.yml", Line: 9},
	})

	if _, changed, err := s.Save(grown, nil, "def456", "test"); err != nil || !changed {
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

	if _, _, err := s.Save(model("example"), nil, "abc123", "test"); err != nil {
		t.Fatal(err)
	}

	if _, changed, err := s.Save(model("example"), nil, "def456", "test"); err != nil || changed {
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
	if _, _, err := first.Save(model("example"), nil, "abc123", "test"); err != nil {
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

// F-50: a citation whose line moved is not a change of architecture — no version is recorded — but
// the stored version follows it, with the commit it is true at, so a reader's link lands on the
// right line. A layout from a newer engine is kept the same way (F-51).
func TestSameArchitectureRefreshesItsEvidence(t *testing.T) {
	s := history(t)
	first := model("example")
	id, _, err := s.Save(first, map[string]archdoc.Layout{"container": {Version: 1}}, "abc123", "test")
	if err != nil {
		t.Fatal(err)
	}

	moved := model("example")
	moved.Nodes[0].Prov.Line = 9
	again, changed, err := s.Save(moved, map[string]archdoc.Layout{"container": {Version: 2}}, "def456", "test")
	if err != nil || changed || again != id {
		t.Fatalf("a moved citation recorded a version: changed=%v id=%d err=%v", changed, again, err)
	}

	v, err := s.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if v.Model.Nodes[0].Prov.Line != 9 || v.Commit != "def456" {
		t.Errorf("the stored version kept stale evidence: line %d at %s", v.Model.Nodes[0].Prov.Line, v.Commit)
	}
	if v.Layouts["container"].Version != 2 {
		t.Errorf("the newer layout was not kept: %+v", v.Layouts)
	}

	// A save without layouts — a scan — keeps the stored ones.
	if _, _, err := s.Save(moved, nil, "def456", "test"); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Latest(); v.Layouts["container"].Version != 2 {
		t.Error("a save without layouts dropped the stored ones")
	}
}

// What is architecture does change the fingerprint; what only cites it does not.
func TestFingerprintIsTheArchitecture(t *testing.T) {
	a, _ := Fingerprint(model("example"))
	cited := model("example")
	cited.Nodes[0].Prov = archdoc.Provenance{File: "elsewhere.yml", Line: 40}
	b, _ := Fingerprint(cited)
	if a != b {
		t.Error("moving a citation changed the fingerprint")
	}
	described := model("example")
	described.Nodes[0].Description = "Serves the API"
	c, _ := Fingerprint(described)
	if a == c {
		t.Error("a new description did not change the fingerprint")
	}
}

// History written before the fingerprint changed — its column holds the old hash — does not gain a
// version just because archdoc was upgraded.
func TestAnUpgradeDoesNotMintAVersion(t *testing.T) {
	s := history(t)
	if _, _, err := s.Save(model("example"), nil, "abc123", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE versions SET fingerprint = 'from-an-older-archdoc'`); err != nil {
		t.Fatal(err)
	}
	if _, changed, err := s.Save(model("example"), nil, "abc123", "test"); err != nil || changed {
		t.Errorf("an upgrade recorded a version (err=%v)", err)
	}
}
