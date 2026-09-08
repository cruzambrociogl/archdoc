package validate

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func at(line int) archdoc.Provenance {
	return archdoc.Provenance{File: "docker-compose.yml", Line: line}
}

// good is a small model that every rule should accept. It is deliberately *thin* — no
// descriptions, one edge without a protocol — because that is what R1.a produces before the
// semantic layer runs, and a validator that rejected it would be useless.
func good() archdoc.Model {
	return archdoc.Model{
		Name:   "example",
		Source: "docker-compose.yml",
		Nodes: []archdoc.Node{
			{ID: "actor:user", Name: "User", Kind: archdoc.Actor, Evidence: archdoc.Declared, Prov: at(5)},
			{ID: "svc:api", Name: "api", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at(10)},
			{
				ID: "svc:db", Name: "db", Kind: archdoc.Datastore,
				Technology: "PostgreSQL 16",
				TechProv:   archdoc.Provenance{Origin: archdoc.Catalog, Note: "postgres"},
				Evidence:   archdoc.Declared, Prov: at(20),
			},
		},
		Edges: []archdoc.Edge{
			{From: "actor:user", To: "svc:api", Label: "reaches", Traffic: true, Prov: []archdoc.Provenance{at(6)}},
			{From: "svc:api", To: "svc:db", Label: "connects to", Technology: "postgres", Traffic: true, Prov: []archdoc.Provenance{at(11)}},
		},
	}.Normalise()
}

func TestAThinButCorrectModelIsAccepted(t *testing.T) {
	r := Model(good())

	if !r.OK() {
		t.Fatalf("a correct model was rejected:\n%s", r.Error())
	}
	// It should still say what is missing. Incomplete and wrong are different answers.
	if len(r.Warnings()) == 0 {
		t.Error("no warnings for a model with no descriptions")
	}
}

// AC-4 — the validator rejects 100% of a fault-injection suite.
//
// Every fault here is a way the draw.io experiment failed, or a way the model could lie without
// a reader being able to tell. That is the standard for inclusion: if a person looking at the
// rendered diagram would not notice, it belongs in this table.
func TestFaultInjectionSuite(t *testing.T) {
	faults := []struct {
		name   string
		rule   string
		inject func(*archdoc.Model)
	}{
		{"edge to a node that was never placed", "VAL-02", func(m *archdoc.Model) {
			m.Edges[0].To = "svc:ghost"
		}},
		{"edge from a node that was never placed", "VAL-02", func(m *archdoc.Model) {
			m.Edges[0].From = "svc:ghost"
		}},
		{"edge pointing at itself", "VAL-02", func(m *archdoc.Model) {
			m.Edges[0].From = m.Edges[0].To
		}},
		{"node with no provenance", "VAL-05", func(m *archdoc.Model) {
			m.Nodes[1].Prov = archdoc.Provenance{}
		}},
		{"edge with no provenance", "VAL-05", func(m *archdoc.Model) {
			m.Edges[0].Prov = nil
		}},
		{"edge attested only by the model", "VAL-05", func(m *archdoc.Model) {
			m.Edges[0].Prov = []archdoc.Provenance{{Origin: archdoc.Semantic, Note: "run-1"}}
		}},
		{"technology with no provenance", "VAL-05", func(m *archdoc.Model) {
			m.Nodes[2].TechProv = archdoc.Provenance{}
		}},
		{"description with no provenance", "VAL-05", func(m *archdoc.Model) {
			m.Nodes[1].Description = "handles requests"
		}},
		{"duplicate node id", "VAL-01", func(m *archdoc.Model) {
			m.Nodes = append(m.Nodes, m.Nodes[1])
		}},
		{"node with no id", "VAL-01", func(m *archdoc.Model) {
			m.Nodes[1].ID = ""
		}},
		{"node with no name", "VAL-01", func(m *archdoc.Model) {
			m.Nodes[1].Name = ""
		}},
		{"node with an unknown kind", "VAL-01", func(m *archdoc.Model) {
			m.Nodes[1].Kind = archdoc.Kind("widget")
		}},
		{"node on neither side of the boundary", "VAL-04", func(m *archdoc.Model) {
			m.Nodes[1].Evidence = archdoc.EvidenceKind("maybe")
		}},
	}

	rejected := 0
	for _, f := range faults {
		t.Run(f.name, func(t *testing.T) {
			m := good()
			f.inject(&m)

			r := Model(m)
			if r.OK() {
				t.Errorf("accepted a model with: %s", f.name)
				return
			}
			rejected++

			found := false
			for _, e := range r.Errors() {
				found = found || e.Rule == f.rule
			}
			if !found {
				t.Errorf("rejected, but not by %s:\n%s", f.rule, r.Error())
			}
		})
	}

	if rejected != len(faults) {
		t.Errorf("AC-4: rejected %d of %d faults, threshold is all of them", rejected, len(faults))
	}
}

// VAL-08. A rejected set of operations must leave the caller holding exactly what they started
// with — not most of it.
func TestApplyIsAllOrNothing(t *testing.T) {
	m := good()
	before := Mermaidish(m)

	ops := []archdoc.Op{
		{Kind: archdoc.SetName, Target: "svc:api", Value: "API", Origin: archdoc.Rules, Prov: rulesAt(3)},
		{Kind: archdoc.SetName, Target: "svc:ghost", Value: "Ghost", Origin: archdoc.Rules, Prov: rulesAt(4)},
	}

	out, r := Apply(m, ops)

	if r.OK() {
		t.Fatal("an operation on a non-existent element was accepted")
	}
	if Mermaidish(out) != before {
		t.Error("a rejected batch left the model partially changed")
	}
	if !strings.Contains(r.Error(), "svc:ghost") {
		t.Errorf("the failure does not name the offending operation:\n%s", r.Error())
	}
}

func TestApplyAppliesEveryOperationWhenAllAreValid(t *testing.T) {
	m := good()

	ops := []archdoc.Op{
		{Kind: archdoc.SetName, Target: "svc:api", Value: "API", Origin: archdoc.Rules, Prov: rulesAt(3)},
		{Kind: archdoc.SetDescription, Target: "svc:api", Value: "Handles requests", Origin: archdoc.Semantic, Prov: runProv()},
		{Kind: archdoc.SetKind, Target: "svc:api", Value: string(archdoc.Proxy), Origin: archdoc.Rules, Prov: rulesAt(5)},
	}

	out, r := Apply(m, ops)
	if !r.OK() {
		t.Fatalf("valid operations were rejected:\n%s", r.Error())
	}

	n, _ := out.Node("svc:api")
	if n.Name != "API" || n.Kind != archdoc.Proxy || n.Description != "Handles requests" {
		t.Errorf("operations not applied: %+v", n)
	}
	// The description must be traceable to the run that wrote it — that is what PRV-05 renders.
	if n.DescProv.Origin != archdoc.Semantic {
		t.Errorf("description provenance is %+v, want the model run", n.DescProv)
	}
}

// Hard rule 4, at the gate rather than in the semantic package. The model may describe; it may
// never change what exists.
func TestStructuralOperationsFromTheModelAreRejected(t *testing.T) {
	m := good()

	for _, op := range []archdoc.Op{
		{Kind: archdoc.AddEdge, Target: "svc:api", To: "actor:user", Origin: archdoc.Semantic, Prov: runProv()},
		{Kind: archdoc.RemoveEdge, Target: "svc:api", To: "svc:db", Origin: archdoc.Semantic, Prov: runProv()},
		{Kind: archdoc.Exclude, Target: "svc:db", Origin: archdoc.Semantic, Prov: runProv()},
	} {
		out, r := Apply(m, []archdoc.Op{op})
		if r.OK() {
			t.Errorf("the model was allowed to %s", op.Kind)
		}
		if len(out.Nodes) != len(m.Nodes) || len(out.Edges) != len(m.Edges) {
			t.Errorf("%s changed the model despite being rejected", op.Kind)
		}
	}
}

// The same operations from a person are fine: a file records what they said and they are
// accountable for it.
func TestStructuralOperationsFromRulesAreAccepted(t *testing.T) {
	m := good()

	out, r := Apply(m, []archdoc.Op{
		{Kind: archdoc.Exclude, Target: "svc:db", Origin: archdoc.Rules, Prov: rulesAt(7)},
	})
	if !r.OK() {
		t.Fatalf("a rule was rejected:\n%s", r.Error())
	}
	if _, ok := out.Node("svc:db"); ok {
		t.Error("the excluded node is still present")
	}
	// Excluding a node must take its edges with it, or VAL-02 would reject the result.
	for _, e := range out.Edges {
		if e.To == "svc:db" {
			t.Error("an edge to the excluded node survived")
		}
	}
}

// Validation output reaches the run log and the report, so AC-7 applies to it too.
func TestFindingsAreDeterministic(t *testing.T) {
	first := Model(good()).Error() + renderFindings(Model(good()))

	for range 10 {
		if got := Model(good()).Error() + renderFindings(Model(good())); got != first {
			t.Fatal("findings differ between runs")
		}
	}
}

func renderFindings(r Result) string {
	var b strings.Builder
	for _, f := range r.Findings {
		b.WriteString(f.String())
		b.WriteString("\n")
	}
	return b.String()
}

func rulesAt(line int) archdoc.Provenance {
	return archdoc.Provenance{Origin: archdoc.Rules, File: "rules.yaml", Line: line}
}

func runProv() archdoc.Provenance {
	return archdoc.Provenance{Origin: archdoc.Semantic, Note: "run-1"}
}

// Mermaidish is a cheap structural fingerprint, used only to detect that a rejected batch left
// nothing changed.
func Mermaidish(m archdoc.Model) string {
	var b strings.Builder
	for _, n := range m.Nodes {
		b.WriteString(n.ID + "|" + n.Name + "|" + string(n.Kind) + "|" + n.Technology + "|" + n.Description + "\n")
	}
	for _, e := range m.Edges {
		b.WriteString(e.From + "->" + e.To + "|" + e.Label + "\n")
	}
	return b.String()
}
