package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func TestIdenticalModelsHaveNoDiff(t *testing.T) {
	m := Derive(facts())
	if d := Compare(m, m); !d.Empty() {
		t.Errorf("a model differs from itself: %+v", d)
	}
}

// A service appearing is news about the system; it must be reported as structural.
func TestAddedAndRemovedElementsAreStructural(t *testing.T) {
	a := Derive(facts())
	b := Derive(facts())
	b.Nodes = append(b.Nodes, archdoc.Node{ID: "svc:cache", Name: "cache", Kind: archdoc.Datastore,
		Evidence: archdoc.Declared, Prov: at(40)})
	b = b.Normalise()

	d := Compare(a, b)
	if len(d.AddedNodes) != 1 || d.AddedNodes[0].ID != "svc:cache" {
		t.Errorf("added: %+v", d.AddedNodes)
	}
	if !d.Structural() {
		t.Error("an added element was not reported as structural")
	}

	back := Compare(b, a)
	if len(back.RemovedNodes) != 1 || back.RemovedNodes[0].ID != "svc:cache" {
		t.Errorf("removed: %+v", back.RemovedNodes)
	}
}

// A reworded description is news about the documentation, not the system. Kept separate so a
// reader scanning for structural change is not made to wade through wording.
func TestRewordingIsNotStructural(t *testing.T) {
	a := Derive(facts())
	b := Derive(facts())
	for i := range b.Nodes {
		if b.Nodes[i].ID == "svc:api" {
			b.Nodes[i].Description = "Serves the public API"
		}
	}

	d := Compare(a, b)
	if d.Structural() {
		t.Error("a reworded description was reported as a structural change")
	}
	if len(d.Changed) != 1 || d.Changed[0].Field != "description" || d.Changed[0].After != "Serves the public API" {
		t.Errorf("changes: %+v", d.Changed)
	}
}

// Identity is the element ID, so a rename is one changed field — not a removal and an addition.
func TestRenameIsAChangeNotAReplacement(t *testing.T) {
	a := Derive(facts())
	b := Derive(facts())
	for i := range b.Nodes {
		if b.Nodes[i].ID == "svc:db" {
			b.Nodes[i].Name = "Database"
		}
	}

	d := Compare(a, b)
	if d.Structural() {
		t.Errorf("a rename was reported as remove plus add: %+v", d)
	}
	if len(d.Changed) != 1 || d.Changed[0].Field != "name" {
		t.Errorf("changes: %+v", d.Changed)
	}
}

func TestDiffIsDeterministic(t *testing.T) {
	a := Derive(facts())
	b := Derive(facts())
	for i := range b.Nodes {
		b.Nodes[i].Description = "changed " + b.Nodes[i].ID
	}
	first := Compare(a, b)
	for range 10 {
		got := Compare(a, b)
		for i := range got.Changed {
			if got.Changed[i] != first.Changed[i] {
				t.Fatal("diff order differs between runs")
			}
		}
	}
}
