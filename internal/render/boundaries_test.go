package render

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// netFixture is Mastodon's shape: every container on `backend`, only the externally reachable
// ones also on `frontend`, so the memberships nest. `orphan` is on neither.
func netFixture() archdoc.Model {
	n := func(id, name string, kind archdoc.Kind, nets ...string) archdoc.Node {
		return archdoc.Node{
			ID: id, Name: name, Kind: kind, Evidence: archdoc.Declared,
			Networks: nets, Prov: at(10),
		}
	}

	return archdoc.Model{
		Name:   "example",
		Source: "docker-compose.yml",
		Networks: []archdoc.Network{
			{Name: "backend", Internal: true, Prov: at(40)},
			{Name: "frontend", Prov: at(41)},
		},
		Nodes: []archdoc.Node{
			n("svc:web", "web", archdoc.Application, "backend", "frontend"),
			n("svc:worker", "worker", archdoc.Application, "backend", "frontend"),
			n("svc:db", "db", archdoc.Datastore, "backend"),
			n("svc:orphan", "orphan", archdoc.Application),
		},
	}.Normalise()
}

func TestNetworksNestWhenMembershipsNest(t *testing.T) {
	out := Mermaid(netFixture().Container(), true)

	backend := strings.Index(out, `subgraph net_backend`)
	frontend := strings.Index(out, `subgraph net_frontend`)
	db := strings.Index(out, "svc_db")

	if backend < 0 || frontend < 0 {
		t.Fatalf("no network boundaries drawn:\n%s", out)
	}
	// frontend ⊂ backend, so frontend must open inside backend, and db — which is only on
	// backend — must sit outside it.
	if !(backend < db && db < frontend) {
		t.Errorf("boundaries are not nested correctly:\n%s", out)
	}
}

// Compose's `internal: true` is the one reachability claim the file makes outright. It belongs
// on the diagram, because it is what tells a reader the data stores cannot be reached.
func TestInternalNetworkIsLabelled(t *testing.T) {
	out := Mermaid(netFixture().Container(), true)

	if !strings.Contains(out, "backend<br/>[no external connectivity]") {
		t.Errorf("internal network not marked:\n%s", out)
	}
	if strings.Contains(out, "frontend<br/>[no external") {
		t.Error("a network that declares nothing was marked internal")
	}
}

// A file may declare networks and still leave a service off all of them. That service belongs to
// the system but to no boundary within it.
func TestServiceOnNoNetworkSitsInTheSystemBoundary(t *testing.T) {
	out := Mermaid(netFixture().Container(), true)

	orphan := strings.Index(out, "svc_orphan")
	firstNet := strings.Index(out, "subgraph net_")

	if orphan < 0 {
		t.Fatal("orphan missing")
	}
	if orphan > firstNet {
		t.Errorf("a service on no network was drawn inside one:\n%s", out)
	}
}

// Two networks that share members without one containing the other cannot be drawn as nested
// boxes. Flattening them into one would erase the distinction the file drew, so no boundary is
// drawn at all — a wrong grouping claims more than no grouping.
func TestOverlappingNetworksFallBackToNoBoundary(t *testing.T) {
	m := netFixture()
	// worker leaves backend, so frontend {web, worker} and backend {web, db} overlap on web
	// while neither contains the other.
	for i := range m.Nodes {
		if m.Nodes[i].ID == "svc:worker" {
			m.Nodes[i].Networks = []string{"frontend"}
		}
	}

	out := Mermaid(m.Container(), true)

	if strings.Contains(out, "subgraph net_") {
		t.Errorf("overlapping networks were drawn as nested boxes:\n%s", out)
	}
	// Every container must still appear — falling back drops the grouping, never a node.
	for _, id := range []string{"svc_web", "svc_worker", "svc_db", "svc_orphan"} {
		if !strings.Contains(out, id) {
			t.Errorf("%s disappeared in the fallback", id)
		}
	}
}

// The boundary on the diagram has to be checkable like everything else on it.
func TestNetworkEvidenceIsCited(t *testing.T) {
	doc := Document(netFixture())

	if !strings.Contains(doc, "### Networks") {
		t.Fatal("no network evidence table")
	}
	for _, want := range []string{
		"| backend | **no external connectivity** | web, worker, db | `docker-compose.yml:40` |",
		"| frontend | reachable from outside | web, worker | `docker-compose.yml:41` |",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing row:\n  %s", want)
		}
	}
}

// A model with no networks must not grow an empty section or an empty box.
func TestNoNetworksMeansNoBoundarySection(t *testing.T) {
	doc := Document(fixture())

	if strings.Contains(doc, "### Networks") {
		t.Error("a network section appeared for a model that declares none")
	}
	if strings.Contains(Mermaid(fixture().Container(), true), "subgraph net_") {
		t.Error("a network boundary appeared for a model that declares none")
	}
}
