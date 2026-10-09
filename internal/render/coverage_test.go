package render

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// AC-1, counted: an element a file states at a line, or one a named catalog entry supplies, is
// traced; one only a rule or a model vouches for is not, and neither is a relationship none of
// whose citations is a reading.
func TestTracedCountsWhatAFileOrTheCatalogStates(t *testing.T) {
	at := archdoc.Provenance{File: "compose.yml", Line: 3}
	m := archdoc.Model{
		Nodes: []archdoc.Node{
			{ID: "svc:api", Prov: at},
			{ID: "ext:stripe", Prov: archdoc.Provenance{Origin: archdoc.Catalog, Note: "the stripe package"}},
			{ID: "svc:added-by-rule", Prov: archdoc.Provenance{Origin: archdoc.Rules, File: ".archdoc/rules.yaml", Line: 2}},
			{ID: "svc:nowhere"},
		},
		Edges: []archdoc.Edge{
			{From: "svc:api", To: "ext:stripe", Prov: []archdoc.Provenance{{Origin: archdoc.Semantic, Note: "a model"}, at}},
			{From: "svc:api", To: "svc:nowhere", Prov: []archdoc.Provenance{{Origin: archdoc.Semantic, Note: "a model"}}},
		},
	}
	if traced, of := Traced(m); traced != 3 || of != 6 {
		t.Errorf("traced %d of %d, want 3 of 6", traced, of)
	}
	if got := percent(2499, 2500); got != "99.9%" {
		t.Errorf("2499 of 2500 is %s — it must not round up to 100%%", got)
	}
}

// A container with more components than a diagram shows has a main view: the ones that handle the
// most entries, then the largest, with only the uses among them — and a small container has none.
func TestMainViewKeepsTheComponentsThatHandleTheMost(t *testing.T) {
	at := archdoc.Provenance{File: "compose.yml", Line: 1}
	m := archdoc.Model{Name: "x", Nodes: []archdoc.Node{
		{ID: "svc:api", Name: "api", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
		{ID: "svc:small", Name: "small", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
		{ID: "cmp:small/only", Name: "only", Kind: archdoc.Component, Parent: "svc:small", Files: []string{"a.ts"}, Prov: at},
	}}
	for i := 0; i < MainParts+4; i++ {
		id := "cmp:api/c" + string(rune('a'+i))
		m.Nodes = append(m.Nodes, archdoc.Node{ID: id, Name: id[8:], Kind: archdoc.Component, Parent: "svc:api", Lines: 100 - i, Files: []string{id + ".ts"}, Prov: at})
	}
	last := "cmp:api/c" + string(rune('a'+MainParts+3)) // the smallest, and the only one that handles a route
	m.Entries = []archdoc.Entry{{ID: "route:api GET /x", Kind: "http", Method: "GET", Path: "/x", Container: "svc:api", Component: last, Prov: at}}
	m.Edges = []archdoc.Edge{
		{From: last, To: "cmp:api/ca", Label: "uses", Weight: 2, Prov: []archdoc.Provenance{at}},
		{From: "cmp:api/ca", To: "cmp:api/c" + string(rune('a'+MainParts+2)), Label: "uses", Weight: 1, Prov: []archdoc.Provenance{at}},
	}
	if got := Mains(m); len(got) != 1 || got[0] != "svc:api" {
		t.Fatalf("mains %v, want only the large container", got)
	}
	v, ok := ViewOf(m, MainPrefix+"svc:api")
	if !ok || len(v.Model.Nodes) != MainParts {
		t.Fatalf("main view: %v, %d components", ok, len(v.Model.Nodes))
	}
	if _, kept := v.Model.Node(last); !kept {
		t.Error("the component that handles a route was left out for being small")
	}
	if len(v.Model.Edges) != 1 || v.Model.Edges[0].From != last {
		t.Errorf("edges %+v, want only the use between two components shown", v.Model.Edges)
	}
	if _, ok := ViewOf(m, MainPrefix+"svc:small"); ok {
		t.Error("a container of one component has a main view")
	}
}
