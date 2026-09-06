package archdoc

import (
	"reflect"
	"testing"
)

// fixture is a hand-made model standing in for a small real stack: a browser reaching a
// gateway, the gateway routing to an API, the API talking to a database and to an object store
// it does not own.
//
// It is written by hand on purpose. Both tracks build against this shape before extraction
// produces it, which is what stops one waiting on the other.
func fixture() Model {
	at := func(line int) Provenance { return Provenance{File: "docker-compose.yml", Line: line} }

	return Model{
		Name:   "example",
		Source: "docker-compose.yml",
		Nodes: []Node{
			{ID: "actor:user", Name: "User", Kind: Actor, Evidence: Declared, Prov: at(20)},
			{ID: "svc:gateway", Name: "gateway", Kind: Proxy, Evidence: Declared, Prov: at(3)},
			{ID: "svc:api", Name: "api", Kind: Application, Evidence: Declared, Prov: at(10)},
			{ID: "svc:db", Name: "db", Kind: Datastore, Evidence: Declared, Prov: at(30)},
			{ID: "ext:s3.amazonaws.com", Name: "s3.amazonaws.com", Kind: External, Evidence: Referenced, Prov: at(14)},
		},
		Edges: []Edge{
			{From: "actor:user", To: "svc:gateway", Label: "visits", Prov: []Provenance{at(20)}},
			{From: "svc:gateway", To: "svc:api", Label: "routes to", Prov: []Provenance{at(5)}},
			{From: "svc:api", To: "svc:db", Label: "reads from", Technology: "postgres", Prov: []Provenance{at(11)}},
			{From: "svc:api", To: "ext:s3.amazonaws.com", Label: "stores in", Technology: "https", Prov: []Provenance{at(14)}},
		},
	}
}

func ids(m Model) []string {
	out := make([]string, 0, len(m.Nodes))
	for _, n := range m.Nodes {
		out = append(out, n.ID)
	}
	return out
}

func edgeKeys(m Model) []string {
	out := make([]string, 0, len(m.Edges))
	for _, e := range m.Edges {
		out = append(out, e.From+"->"+e.To)
	}
	return out
}

func TestContainerViewExcludesProxy(t *testing.T) {
	view := fixture().Container()

	want := []string{"actor:user", "svc:api", "svc:db", "ext:s3.amazonaws.com"}
	if got := ids(view); !reflect.DeepEqual(got, want) {
		t.Fatalf("nodes: got %v, want %v", got, want)
	}
}

// The gateway leaves the view, but the relationship it carried does not: the user still reaches
// the API, and the bridged edge cites both hops it was built from.
func TestContainerViewBridgesThroughProxy(t *testing.T) {
	view := fixture().Container()

	want := []string{"actor:user->svc:api", "svc:api->ext:s3.amazonaws.com", "svc:api->svc:db"}
	if got := edgeKeys(view); !reflect.DeepEqual(got, want) {
		t.Fatalf("edges: got %v, want %v", got, want)
	}

	for _, e := range view.Edges {
		if e.From != "actor:user" {
			continue
		}
		if len(e.Prov) != 2 {
			t.Fatalf("bridged edge should cite both hops, got %v", e.Prov)
		}
	}
}

// Context collapses everything declared into one box and keeps only what crosses the boundary.
// Nothing else needs deriving: the evidence kind already drew that line.
func TestContextViewCollapsesDeclared(t *testing.T) {
	view := fixture().Context()

	want := []string{"actor:user", "system", "ext:s3.amazonaws.com"}
	if got := ids(view); !reflect.DeepEqual(got, want) {
		t.Fatalf("nodes: got %v, want %v", got, want)
	}

	wantEdges := []string{"actor:user->system", "system->ext:s3.amazonaws.com"}
	if got := edgeKeys(view); !reflect.DeepEqual(got, wantEdges) {
		t.Fatalf("edges: got %v, want %v", got, wantEdges)
	}
}

// The collapsed system box still points at a real line. A node without provenance must not be
// emitted, and that applies to a derived one too.
func TestContextSystemNodeCarriesProvenance(t *testing.T) {
	view := fixture().Context()

	sys, ok := view.Node("system")
	if !ok {
		t.Fatal("no system node")
	}
	if !sys.Prov.Known() {
		t.Fatalf("system node has no provenance: %+v", sys.Prov)
	}
	if sys.Prov.Line != 3 {
		t.Fatalf("expected the earliest declared line, got %d", sys.Prov.Line)
	}
}

// AC-7 in miniature. Projection reads maps; unsorted output would break byte-identical runs.
func TestViewsAreDeterministic(t *testing.T) {
	for range 20 {
		if got, want := fixture().Container(), fixture().Container(); !reflect.DeepEqual(got, want) {
			t.Fatal("container view differs between runs")
		}
		if got, want := fixture().Context(), fixture().Context(); !reflect.DeepEqual(got, want) {
			t.Fatal("context view differs between runs")
		}
	}
}
