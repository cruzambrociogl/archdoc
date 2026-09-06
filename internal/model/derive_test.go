package model

import (
	"reflect"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func at(line int) archdoc.Provenance {
	return archdoc.Provenance{File: "docker-compose.yml", Line: line}
}

// facts is the hand-made FactSet the delivery plan calls for: the contract between the two
// tracks, written by hand so neither waits on the other. It is small and deliberately awkward —
// a gateway in front of the app, a datastore, a queue, an external host the repository only
// references, and one relationship attested twice.
func facts() *archdoc.FactSet {
	return &archdoc.FactSet{
		Root:   "/repo",
		Name:   "example",
		Source: "docker-compose.yml",
		Services: []archdoc.Service{
			{
				Name: "gateway", Image: "nginx:1.27", Evidence: archdoc.Declared, Prov: at(3),
				Ports:     []archdoc.Port{{Published: "80", Target: 80, Prov: at(5)}},
				DependsOn: []archdoc.Dependency{{Service: "api", Prov: at(7)}},
				// The gateway is configured with an upstream, so there is evidence that
				// traffic flows through it and not merely that it starts second.
				Endpoints: []archdoc.Endpoint{
					{Var: "UPSTREAM_URL", Scheme: "http", Host: "api", Port: 8080, Prov: at(8)},
				},
			},
			{
				Name: "api", Image: "example/api:2.1", Evidence: archdoc.Declared, Prov: at(10),
				DependsOn: []archdoc.Dependency{{Service: "db", Prov: at(12)}},
				Endpoints: []archdoc.Endpoint{
					// The same relationship as the depends_on above, said a second way.
					{Var: "DATABASE_URL", Scheme: "postgres", Host: "db", Port: 5432, Prov: at(15)},
					{Var: "QUEUE_URL", Scheme: "amqp", Host: "broker", Prov: at(16)},
					{Var: "S3_ENDPOINT", Scheme: "https", Host: "s3.example.com", Prov: at(17)},
				},
			},
			{Name: "db", Image: "postgres:16-alpine", Evidence: archdoc.Declared, Prov: at(20)},
			{Name: "broker", Image: "rabbitmq:3.13", Evidence: archdoc.Declared, Prov: at(25)},
		},
	}
}

func TestKindsAndTechnologyComeFromTheImage(t *testing.T) {
	m := Derive(facts())

	want := map[string]struct {
		kind archdoc.Kind
		tech string
	}{
		"svc:gateway": {archdoc.Proxy, "nginx 1.27"},
		"svc:api":     {archdoc.Application, ""},
		"svc:db":      {archdoc.Datastore, "PostgreSQL 16"},
		"svc:broker":  {archdoc.Queue, "RabbitMQ 3.13"},
	}

	for id, w := range want {
		n, ok := m.Node(id)
		if !ok {
			t.Fatalf("%s missing", id)
		}
		if n.Kind != w.kind {
			t.Errorf("%s is %q, want %q", id, n.Kind, w.kind)
		}
		if n.Technology != w.tech {
			t.Errorf("%s technology is %q, want %q", id, n.Technology, w.tech)
		}
	}
}

// A custom-built image is an application with no technology. Empty says "not known from
// configuration", which is the honest answer and the one MDL-08 hands to the semantic layer.
func TestUnknownImageGetsNoInventedTechnology(t *testing.T) {
	if n, _ := Derive(facts()).Node("svc:api"); n.Technology != "" {
		t.Errorf("technology invented for a custom image: %q", n.Technology)
	}
}

// A host the repository names but does not define is outside the system. That is the evidence
// rule doing double duty — referenced is also where C4 draws the boundary.
func TestUndeclaredHostsBecomeReferencedExternals(t *testing.T) {
	m := Derive(facts())

	n, ok := m.Node("ext:s3.example.com")
	if !ok {
		t.Fatal("no external node for s3.example.com")
	}
	if n.Evidence != archdoc.Referenced {
		t.Errorf("evidence is %q, want referenced", n.Evidence)
	}
	if n.Kind != archdoc.External {
		t.Errorf("kind is %q, want external", n.Kind)
	}

	// broker *is* declared, so naming it in a URL must not also create an external twin.
	if _, ok := m.Node("ext:broker"); ok {
		t.Error("a declared service was duplicated as an external system")
	}
}

// One relationship, two attestations: depends_on says api needs db, and DATABASE_URL says how.
// That is one edge carrying both citations, not two edges.
func TestRepeatedAttestationsMergeIntoOneEdge(t *testing.T) {
	m := Derive(facts())

	count := 0
	var edge archdoc.Edge
	for _, e := range m.Edges {
		if e.From == "svc:api" && e.To == "svc:db" {
			count++
			edge = e
		}
	}

	if count != 1 {
		t.Fatalf("got %d edges api->db, want 1", count)
	}
	if len(edge.Prov) != 2 {
		t.Errorf("got %d citations, want 2: %+v", len(edge.Prov), edge.Prov)
	}
	if edge.Technology != "postgres" {
		t.Errorf("technology is %q, want postgres — the URL knows it and depends_on does not", edge.Technology)
	}
}

// A published port is the only evidence configuration offers that something outside reaches in.
func TestActorComesFromPublishedPorts(t *testing.T) {
	m := Derive(facts())

	a, ok := m.Node(actorID)
	if !ok {
		t.Fatal("no actor derived from the published port")
	}
	if a.Prov.Line != 5 {
		t.Errorf("actor cited at line %d, want 5 — the port mapping", a.Prov.Line)
	}

	for _, e := range m.Edges {
		if e.From == actorID && e.To == "svc:gateway" {
			return
		}
	}
	t.Error("no edge from the actor to the service publishing the port")
}

// A repository with no published port has no evidence of anyone outside, and archdoc must not
// draw a user it cannot cite.
func TestNoPortsMeansNoActor(t *testing.T) {
	f := facts()
	f.Services[0].Ports = nil

	if _, ok := Derive(f).Node(actorID); ok {
		t.Error("an actor was drawn with nothing to cite")
	}
}

// The container view applies the derivation rule: the gateway is infrastructure and leaves,
// but the user still reaches the API through it.
func TestContainerViewOverTheDerivedModel(t *testing.T) {
	view := Derive(facts()).Container()

	want := []string{"actor:user", "svc:api", "svc:db", "svc:broker", "ext:s3.example.com"}
	got := make([]string, 0, len(view.Nodes))
	for _, n := range view.Nodes {
		got = append(got, n.ID)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nodes: got %v, want %v", got, want)
	}

	for _, e := range view.Edges {
		if e.From == actorID && e.To == "svc:api" {
			return
		}
	}
	t.Error("excluding the gateway lost the user's route into the system")
}

// Context keeps only what crosses the boundary. Everything declared becomes one box.
func TestContextViewOverTheDerivedModel(t *testing.T) {
	view := Derive(facts()).Context()

	want := []string{"actor:user", "system", "ext:s3.example.com"}
	got := make([]string, 0, len(view.Nodes))
	for _, n := range view.Nodes {
		got = append(got, n.ID)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nodes: got %v, want %v", got, want)
	}

	if sys, _ := view.Node("system"); sys.Name != "example" {
		t.Errorf("system box is named %q, want the project name", sys.Name)
	}
}

func TestDerivationIsDeterministic(t *testing.T) {
	for range 20 {
		if !reflect.DeepEqual(Derive(facts()), Derive(facts())) {
			t.Fatal("derivation differs between runs")
		}
	}
}
