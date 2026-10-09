package render

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func dataFlowFixture() archdoc.Model {
	code := func(line int) archdoc.Provenance { return archdoc.Provenance{File: "api/src/repo.ts", Line: line} }
	m := fixture()
	m.Nodes = append(m.Nodes, archdoc.Node{ID: "app:web", Name: "web", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at(30)},
		archdoc.Node{ID: "tbl:api/album", Name: "album", Kind: archdoc.Table, Parent: "svc:api", Evidence: archdoc.Declared, Prov: code(1)},
		archdoc.Node{ID: "tbl:api/user", Name: "user", Kind: archdoc.Table, Parent: "svc:api", Evidence: archdoc.Declared, Prov: code(2)})
	m.Edges = append(m.Edges,
		archdoc.Edge{From: "actor:user", To: "app:web", Label: "reaches", Traffic: true, Prov: []archdoc.Provenance{at(30)}},
		archdoc.Edge{From: "app:web", To: "svc:api", Label: "calls", Traffic: true, Prov: []archdoc.Provenance{at(31)}})
	m.Access = []archdoc.Access{
		{Container: "svc:api", Table: "album", Element: "tbl:api/album", Op: "writes", Count: 3, Prov: []archdoc.Provenance{code(10)}},
		{Container: "svc:api", Table: "user", Element: "tbl:api/user", Op: "reads", Count: 1, Prov: []archdoc.Provenance{code(20)}},
		{Container: "svc:api", Table: "tmp", Op: "reads", Count: 1, Prov: []archdoc.Provenance{code(30)}}, // not declared: not counted
	}
	return m.Normalise()
}

// Each container sits in the stage its kind and its callers put it in, and the arrow into the store
// its tables live in says what the code writes and reads there, citing the queries.
func TestDataFlowStagesAndWhatMoves(t *testing.T) {
	v := dataFlowView(dataFlowFixture())
	want := map[string]string{"actor:user": "People", "app:web": "Clients", "svc:api": "Services", "svc:db": "Stores", "ext:s3.amazonaws.com": "Outside"}
	for _, n := range v.Model.Nodes {
		if want[n.ID] != n.Stage {
			t.Errorf("%s in %q, want %q", n.ID, n.Stage, want[n.ID])
		}
	}
	if got := strings.Join(v.Model.Stages, ","); got != "People,Clients,Services,Stores,Outside" {
		t.Errorf("stages %s", got)
	}
	e, ok := edgeOf(v.Model, "svc:api", "svc:db")
	if !ok || e.Label != "writes album table · reads 1 more" || e.LabelProv.Line != 10 || e.LabelProv.Origin.Interpretation() {
		t.Errorf("into the store: %q cited %v", e.Label, e.LabelProv)
	}
	if len(e.Prov) != 4 { // the two configuration lines, and the first query of each table
		t.Errorf("citations %v", e.Prov)
	}
}

// The stages are columns in the layout, left to right in their order, and the unseen arrows that
// keep them so are not drawn.
func TestDataFlowLaysOutStages(t *testing.T) {
	m := dataFlowFixture()
	v, ok := ViewOf(m, DataFlow)
	if !ok {
		t.Fatal("no data-flow view")
	}
	l, err := Layout(t.Context(), v.Model, v.Group)
	if err != nil {
		t.Fatal(err)
	}
	x := map[string]float64{}
	for _, g := range l.Groups {
		if !g.Stage {
			t.Errorf("boundary %q is not a stage", g.Name)
		}
		x[g.Name] = g.Rect.X
	}
	for i := 1; i < len(v.Model.Stages); i++ {
		if a, b := v.Model.Stages[i-1], v.Model.Stages[i]; x[a] >= x[b] {
			t.Errorf("%s at %v is not left of %s at %v", a, x[a], b, x[b])
		}
	}
	if len(l.Paths) != len(v.Model.Edges) {
		t.Errorf("%d paths drawn for %d arrows", len(l.Paths), len(v.Model.Edges))
	}
	if svg := SVG(v.Model, l); !strings.Contains(svg, ">STORES<") {
		t.Error("the SVG does not name the stores column")
	}
}
