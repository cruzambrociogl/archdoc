package validate

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Found on the first live run: a model that only reworded an arrow's label was appended to the
// arrow's own citations, so the relationships table listed "model: claude-opus-5" first under
// "Declared at" — as though the model had declared the relationship. The label's citation belongs
// to the label; the relationship's citations must stay evidence that it exists.
func TestRelabellingNeverBecomesEvidenceForTheEdge(t *testing.T) {
	m := good()

	out, r := Apply(m, []archdoc.Op{{
		Kind: archdoc.SetEdgeLabel, Target: "svc:api", To: "svc:db", Value: "reads and writes orders",
		Origin: archdoc.Semantic, Prov: runProv(),
	}})
	if !r.OK() {
		t.Fatalf("relabel rejected:\n%s", r.Error())
	}

	for _, e := range out.Edges {
		if e.From != "svc:api" || e.To != "svc:db" {
			continue
		}
		if e.Label != "reads and writes orders" {
			t.Errorf("label not applied: %q", e.Label)
		}
		if e.LabelProv.Origin != archdoc.Semantic {
			t.Errorf("the label does not cite the model: %+v", e.LabelProv)
		}
		for _, p := range e.Prov {
			if p.Origin == archdoc.Semantic {
				t.Errorf("the model appears as evidence the relationship exists: %+v", e.Prov)
			}
		}
		return
	}
	t.Fatal("edge api → db missing")
}

// A renamed element must say so. Without it, "REST API" reads as the name the configuration
// declared, when the configuration said `rest`.
func TestRenameCarriesItsOwnCitation(t *testing.T) {
	out, r := Apply(good(), []archdoc.Op{{
		Kind: archdoc.SetName, Target: "svc:api", Value: "Public API",
		Origin: archdoc.Semantic, Prov: runProv(),
	}})
	if !r.OK() {
		t.Fatalf("rename rejected:\n%s", r.Error())
	}

	n, _ := out.Node("svc:api")
	if n.Name != "Public API" || n.NameProv.Origin != archdoc.Semantic {
		t.Errorf("rename not cited: name=%q prov=%+v", n.Name, n.NameProv)
	}
	if n.Prov.Origin == archdoc.Semantic {
		t.Error("the element's own evidence was replaced by the model")
	}
}
