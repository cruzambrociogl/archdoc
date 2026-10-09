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
