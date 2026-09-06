package render

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func at(line int) archdoc.Provenance {
	return archdoc.Provenance{File: "docker-compose.yml", Line: line}
}

func fixture() archdoc.Model {
	return archdoc.Model{
		Name:   "example",
		Source: "docker-compose.yml",
		Nodes: []archdoc.Node{
			{ID: "actor:user", Name: "User", Kind: archdoc.Actor, Evidence: archdoc.Declared, Prov: at(5)},
			{ID: "svc:api", Name: "api", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at(10)},
			{ID: "svc:db", Name: "db", Kind: archdoc.Datastore, Technology: "PostgreSQL 16", Evidence: archdoc.Declared, Prov: at(20)},
			{ID: "svc:gateway", Name: "gateway", Kind: archdoc.Proxy, Technology: "nginx 1.27", Evidence: archdoc.Declared, Prov: at(3)},
			{ID: "ext:s3.amazonaws.com", Name: "s3.amazonaws.com", Kind: archdoc.External, Evidence: archdoc.Referenced, Prov: at(15)},
		},
		Edges: []archdoc.Edge{
			{From: "actor:user", To: "svc:api", Label: "reaches", Traffic: true, Prov: []archdoc.Provenance{at(5)}},
			{From: "svc:api", To: "svc:db", Label: "connects to", Technology: "postgres", Traffic: true, Prov: []archdoc.Provenance{at(12), at(13)}},
			{From: "svc:api", To: "ext:s3.amazonaws.com", Label: "connects to", Technology: "https", Traffic: true, Prov: []archdoc.Provenance{at(15)}},
		},
	}.Normalise()
}

// Model IDs carry punctuation Mermaid cannot take. If they reach the output the diagram does not
// render at all, which is the one failure a reader cannot work around.
func TestMermaidIdentifiersAreSafe(t *testing.T) {
	out := Mermaid(fixture().Container(), true)

	for _, line := range strings.Split(out, "\n") {
		decl, _, isNode := strings.Cut(line, "[")
		if !isNode {
			continue
		}
		if strings.ContainsAny(strings.TrimSpace(decl), ":.") {
			t.Errorf("unsafe identifier in %q", strings.TrimSpace(line))
		}
	}
}

func TestMermaidShapesFollowKind(t *testing.T) {
	out := Mermaid(fixture().Container(), true)

	for _, want := range []string{
		`actor_user(["<b>User</b><br/>[Person]"])`,
		`svc_db[("<b>db</b><br/>[Container: PostgreSQL 16]")]`,
		`ext_s3_amazonaws_com["<b>s3.amazonaws.com</b><br/>[External System]"]`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s\n\n%s", want, out)
		}
	}
}

// The boundary is what makes a container diagram a container diagram: it says which of these
// things the repository is, and which it merely talks to.
func TestContainerViewDrawsTheSystemBoundary(t *testing.T) {
	out := Mermaid(fixture().Container(), true)

	if !strings.Contains(out, `subgraph boundary["example"]`) {
		t.Fatalf("no system boundary:\n%s", out)
	}

	boundary := out[strings.Index(out, "subgraph"):strings.Index(out, "\n    end")]
	if strings.Contains(boundary, "actor_user") || strings.Contains(boundary, "ext_s3") {
		t.Error("an actor or external system was drawn inside the system boundary")
	}
}

// The context view has one box for the system, so there is nothing to group.
func TestContextViewHasNoBoundary(t *testing.T) {
	if out := Mermaid(fixture().Context(), false); strings.Contains(out, "subgraph") {
		t.Errorf("context view drew a boundary around a single box:\n%s", out)
	}
}

// Every element and every relationship has to be checkable, which is the entire claim the
// project makes. A table row without a citation is worse than a missing row.
func TestDocumentCitesEveryElementAndRelationship(t *testing.T) {
	doc := Document(fixture())

	for _, want := range []string{
		"| User | Person | — | declared | `docker-compose.yml:5` |",
		"| db | Container (data store) | PostgreSQL 16 | declared | `docker-compose.yml:20` |",
		"| s3.amazonaws.com | External system | — | referenced | `docker-compose.yml:15` |",
		"| api | db | connects to | postgres | `docker-compose.yml:12`, `docker-compose.yml:13` |",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing row:\n  %s", want)
		}
	}
}

// A diagram that silently drops elements is not traceable in the direction that matters most: a
// reader who knows the system looks for the gateway and must be told it was a decision.
func TestDocumentNamesWhatItExcluded(t *testing.T) {
	doc := Document(fixture())

	if !strings.Contains(doc, "### Not shown") {
		t.Fatal("the excluded gateway is not accounted for")
	}
	if !strings.Contains(doc, "| gateway | Infrastructure | `docker-compose.yml:3` |") {
		t.Error("the excluded element is not named with its line")
	}
}

// AC-7. The renderer reads maps — identifiers, style classes — and unsorted output would differ
// between runs without changing a single pixel of the diagram.
func TestRenderIsDeterministic(t *testing.T) {
	first := Document(fixture())

	for range 5 {
		if Document(fixture()) != first {
			t.Fatal("output is not byte-identical between runs")
		}
	}
}
