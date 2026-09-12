package archdoc

import "testing"

// routed is a user reaching two services through a gateway, the Supabase shape: one published
// port, two routes, each route with its own line in the gateway's configuration.
func routed(withLabels bool) Model {
	at := func(file string, line int) Provenance { return Provenance{File: file, Line: line} }
	model := Provenance{Origin: Semantic, Note: "claude-opus-5"}

	m := Model{
		Name: "example",
		Nodes: []Node{
			{ID: "actor:user", Name: "User", Kind: Actor, Evidence: Declared, Prov: at("compose.yml", 5)},
			{ID: "svc:gw", Name: "gw", Kind: Proxy, Evidence: Declared, Prov: at("compose.yml", 3)},
			{ID: "svc:auth", Name: "auth", Kind: Application, Evidence: Declared, Prov: at("compose.yml", 10)},
			{ID: "svc:storage", Name: "storage", Kind: Application, Evidence: Declared, Prov: at("compose.yml", 20)},
		},
		Edges: []Edge{
			{From: "actor:user", To: "svc:gw", Label: "reaches", Traffic: true, Prov: []Provenance{at("compose.yml", 6)}},
			{From: "svc:gw", To: "svc:auth", Label: "routes to", Traffic: true, Prov: []Provenance{at("cds.yaml", 18)}},
			{From: "svc:gw", To: "svc:storage", Label: "routes to", Traffic: true, Prov: []Provenance{at("cds.yaml", 114)}},
		},
	}
	if withLabels {
		m.Edges[0].Label, m.Edges[0].LabelProv = "calls project APIs through", model
		m.Edges[1].Label, m.Edges[1].LabelProv = "forwards signup and token requests to", model
		m.Edges[2].Label, m.Edges[2].LabelProv = "forwards upload and download requests to", model
	}
	return m
}

func userLabels(m Model) map[string]Edge {
	out := map[string]Edge{}
	for _, e := range m.Container().Edges {
		if e.From == "actor:user" {
			out[e.To] = e
		}
	}
	return out
}

// Found on a live Supabase run: all seven user arrows read the same generic label, while the model
// had written a specific label for each gateway route and those were hidden with the gateway.
func TestBridgedArrowUsesTheRoutesWrittenLabel(t *testing.T) {
	got := userLabels(routed(true))

	if l := got["svc:auth"].Label; l != "forwards signup and token requests to" {
		t.Errorf("user → auth reads %q, want the route's own label", l)
	}
	if l := got["svc:storage"].Label; l != "forwards upload and download requests to" {
		t.Errorf("user → storage reads %q, want the route's own label", l)
	}
	if got["svc:auth"].Label == got["svc:storage"].Label {
		t.Error("two different routes still read the same")
	}
	// The label's citation travels with it, and never becomes evidence for the arrow.
	if !got["svc:auth"].LabelProv.Origin.Interpretation() {
		t.Error("the route label lost its model citation")
	}
	for _, p := range got["svc:auth"].Prov {
		if p.Origin == Semantic {
			t.Error("the model appears as evidence for a bridged arrow")
		}
	}
}

// With no model, the extracted labels are "reaches" and "routes to", and "reaches" is the one
// that reads right on a user's arrow. Unchanged behaviour.
func TestBridgedArrowKeepsTheFirstHopWithoutWrittenLabels(t *testing.T) {
	for to, e := range userLabels(routed(false)) {
		if e.Label != "reaches" {
			t.Errorf("user → %s reads %q, want %q", to, e.Label, "reaches")
		}
	}
}
