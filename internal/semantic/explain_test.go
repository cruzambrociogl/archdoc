package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func explainModel() archdoc.Model {
	p := func(f string, l int) archdoc.Provenance { return archdoc.Provenance{File: f, Line: l} }
	return archdoc.Model{Name: "x",
		Nodes: []archdoc.Node{
			{ID: "svc:server", Name: "server", Kind: archdoc.Application, Prov: p("compose.yml", 2)},
			{ID: "cmp:server/controllers", Name: "controllers", Kind: archdoc.Component, Parent: "svc:server", Dir: "server/src/controllers",
				Files: []string{"server/src/controllers/album.controller.ts"}, Lines: 40, Prov: p("server/src/controllers/album.controller.ts", 1)},
			{ID: "cmp:server/services", Name: "services", Kind: archdoc.Component, Parent: "svc:server", Dir: "server/src/services",
				Files: []string{"server/src/services/album.service.ts"}, Lines: 90, Prov: p("server/src/services/album.service.ts", 1)},
		},
		Edges: []archdoc.Edge{{From: "cmp:server/controllers", To: "cmp:server/services", Label: "uses", Weight: 3,
			Prov: []archdoc.Provenance{p("server/src/controllers/album.controller.ts", 2)}}},
		Entries: []archdoc.Entry{{ID: "route:server GET /api/albums", Kind: "http", Method: "GET", Path: "/api/albums",
			Handler: "AlbumController.getAll", Summary: "SECRET-SUMMARY", Container: "svc:server", Component: "cmp:server/controllers",
			Prov: p("server/src/controllers/album.controller.ts", 7)}},
	}
}

// fake answers each component with the sentences given, and records every prompt it was sent.
func fake(answers map[string][]string, prompts *[]string) Completer {
	return func(_ context.Context, _ string, turns []Turn) (Reply, error) {
		last := turns[len(turns)-1].Text
		*prompts = append(*prompts, last)
		for key, a := range answers {
			if strings.Contains(turns[0].Text, key) {
				if len(a) > 1 && len(turns) > 1 {
					return Reply{Text: a[1]}, nil // the corrected answer
				}
				return Reply{Text: a[0]}, nil
			}
		}
		return Reply{Text: `{"sentences":[]}`}, nil
	}
}

func TestExplanationsCiteTheirFacts(t *testing.T) {
	var prompts []string
	answers := map[string][]string{
		`component "controllers" inside`: {`{"sentences":[{"text":"Serves the album routes.","cites":["F4"]},{"text":"Hands work to services.","cites":["F3","F3"]}]}`},
	}
	m, mem, rep, err := Explain(context.Background(), fake(answers, &prompts), "test-model", explainModel(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Asked != 2 || len(m.Explanations) != 1 { // services was asked, and said nothing
		t.Fatalf("asked %d, explanations %+v", rep.Asked, m.Explanations)
	}
	x := m.Explanations[0]
	if x.Element != "cmp:server/controllers" || x.Prov.Origin != archdoc.Semantic || len(x.Claims) != 2 {
		t.Fatalf("explanation %+v", x)
	}
	if c := x.Claims[0]; c.Cites[0].Line != 7 || !strings.Contains(c.Facts[0], "GET /api/albums") {
		t.Errorf("a claim's citation does not resolve to the route's line: %+v", c)
	}
	if len(x.Claims[1].Cites) != 1 {
		t.Errorf("a fact cited twice is one citation: %+v", x.Claims[1])
	}
	if len(mem) != 1 {
		t.Errorf("remembered %d answers, want the one that said something", len(mem))
	}

	// Names only: nothing the code says in a string, and no line, leaves the machine.
	for _, p := range prompts {
		if strings.Contains(p, "SECRET-SUMMARY") || strings.Contains(p, ".ts:") {
			t.Errorf("the prompt carries a string from the code or a provenance:\n%s", p)
		}
	}
}

// A sentence that cites nothing, or a fact it was not given, goes back; refused again, it is left out.
func TestUncitedSentencesAreRefused(t *testing.T) {
	var prompts []string
	answers := map[string][]string{
		`component "services" inside`: {
			`{"sentences":[{"text":"Holds the business logic.","cites":[]},{"text":"Talks to the database.","cites":["F99"]},{"text":"Is used by controllers.","cites":["F3"]}]}`,
			`{"sentences":[{"text":"Holds the business logic.","cites":[]},{"text":"Is used by controllers.","cites":["F3"]}]}`,
		},
	}
	m, _, rep, err := Explain(context.Background(), fake(answers, &prompts), "test-model", explainModel(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var x archdoc.Explanation
	for _, e := range m.Explanations {
		if e.Element == "cmp:server/services" {
			x = e
		}
	}
	if len(x.Claims) != 1 || x.Claims[0].Text != "Is used by controllers." {
		t.Errorf("claims %+v", x.Claims)
	}
	if rep.Refused != 1 {
		t.Errorf("refused %d, want the one still uncited after the retries", rep.Refused)
	}
	found := false
	for _, p := range prompts {
		if strings.Contains(p, `"F99", which is not one of the facts given`) {
			found = true
		}
	}
	if !found {
		t.Error("the model was not told which citation did not resolve")
	}
}

// Remembered answers are reused while the facts are unchanged — with no model at all — and asked
// for again once they change.
func TestExplanationsAreRemembered(t *testing.T) {
	var prompts []string
	answers := map[string][]string{`component "controllers" inside`: {`{"sentences":[{"text":"Serves the album routes.","cites":["F4"]}]}`}}
	_, mem, _, _ := Explain(context.Background(), fake(answers, &prompts), "test-model", explainModel(), nil)

	again, _, rep, err := Explain(context.Background(), nil, "test-model", explainModel(), mem)
	if err != nil || rep.Remembered != 1 || rep.Asked != 0 || len(again.Explanations) != 1 {
		t.Fatalf("remembered %d, asked %d, %d explanations, %v", rep.Remembered, rep.Asked, len(again.Explanations), err)
	}

	changed := explainModel()
	changed.Entries[0].Path = "/api/v2/albums"
	after, _, rep, _ := Explain(context.Background(), nil, "test-model", changed, mem)
	if rep.Remembered != 0 || len(after.Explanations) != 0 {
		t.Errorf("an answer about different facts was reused: %+v", after.Explanations)
	}
}
