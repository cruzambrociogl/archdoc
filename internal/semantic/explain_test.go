package semantic

import (
	"context"
	"encoding/json"
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
				Files: []string{"server/src/services/album.service.ts", "server/src/services/base.service.ts"}, Lines: 90, Prov: p("server/src/services/album.service.ts", 1)},
		},
		Edges: []archdoc.Edge{{From: "cmp:server/controllers", To: "cmp:server/services", Label: "uses", Weight: 3,
			Prov: []archdoc.Provenance{p("server/src/controllers/album.controller.ts", 2)}}},
		Entries: []archdoc.Entry{{ID: "route:server GET /api/albums", Kind: "http", Method: "GET", Path: "/api/albums",
			Handler: "AlbumController.getAll", Summary: "List all albums", Container: "svc:server", Component: "cmp:server/controllers",
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

	// Names, and a route's own summary; never a provenance — no line leaves the machine.
	summary := false
	for _, p := range prompts {
		if strings.Contains(p, ".ts:") {
			t.Errorf("the prompt carries a provenance:\n%s", p)
		}
		summary = summary || strings.Contains(p, `described by the code as "List all albums"`)
	}
	if !summary {
		t.Error("the route's summary was not given to the model")
	}
}

// A sentence that cites nothing, or a fact it was not given, goes back; refused again, it is left out.
func TestUncitedSentencesAreRefused(t *testing.T) {
	var prompts []string
	answers := map[string][]string{
		`component "services" inside`: {
			`{"sentences":[{"text":"Holds the business logic.","cites":[]},{"text":"Talks to the database.","cites":["F99"]},{"text":"Is used by controllers.","cites":["F4"]}]}`,
			`{"sentences":[{"text":"Holds the business logic.","cites":[]},{"text":"Is used by controllers.","cites":["F4"]}]}`,
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
	// Once the facts change, the last answer still stands in — marked stale, not passed off as current.
	after, _, rep, _ := Explain(context.Background(), nil, "test-model", changed, mem)
	if rep.Remembered != 0 || rep.Stale != 1 || len(after.Explanations) != 1 || !after.Explanations[0].Stale {
		t.Errorf("remembered %d, stale %d, %+v", rep.Remembered, rep.Stale, after.Explanations)
	}

	// The first memory format — claims alone, keyed by the first wording's fingerprint — is still found.
	old := Memory{}
	if err := json.Unmarshal([]byte(`{"`+legacyFingerprint(explainModel(), "cmp:server/controllers")+`":[{"text":"Old answer.","facts":["f"],"cites":[{"file":"a.ts","line":1}]}]}`), &old); err != nil {
		t.Fatal(err)
	}
	legacy, _, rep, _ := Explain(context.Background(), nil, "test-model", explainModel(), old)
	if rep.Stale != 1 || len(legacy.Explanations) != 1 || legacy.Explanations[0].Claims[0].Text != "Old answer." {
		t.Errorf("a first-format memory was not found: stale %d, %+v", rep.Stale, legacy.Explanations)
	}
}

// A request that was paid for is never thrown away: an empty answer leaves that one component
// unexplained, and a failed request stops the asking — the answers before it are kept.
func TestABadAnswerDoesNotLoseTheOthers(t *testing.T) {
	calls := 0
	complete := func(_ context.Context, _ string, turns []Turn) (Reply, error) {
		calls++
		if strings.Contains(turns[0].Text, `component "controllers" inside`) {
			return Reply{Text: `{"sentences":[{"text":"Serves the album routes.","cites":["F4"]}]}`}, nil
		}
		return Reply{Text: "", Stop: "max_tokens"}, nil
	}
	m, mem, rep, err := Explain(context.Background(), complete, "test-model", explainModel(), nil)
	if err != nil {
		t.Fatalf("an empty answer failed the run: %v", err)
	}
	if len(m.Explanations) != 1 || len(mem) != 1 || len(rep.Unanswered) != 1 || !strings.Contains(rep.Unanswered[0], "max_tokens") {
		t.Errorf("explanations %d, remembered %d, unanswered %v", len(m.Explanations), len(mem), rep.Unanswered)
	}

	failing := func(_ context.Context, _ string, turns []Turn) (Reply, error) {
		if strings.Contains(turns[0].Text, `component "controllers" inside`) {
			return Reply{Text: `{"sentences":[{"text":"Serves the album routes.","cites":["F4"]}]}`}, nil
		}
		return Reply{}, context.DeadlineExceeded
	}
	m, mem, rep, err = Explain(context.Background(), failing, "test-model", explainModel(), nil)
	if err != nil || rep.Stopped == "" || len(m.Explanations) != 1 || len(mem) != 1 {
		t.Errorf("err %v, stopped %q, explanations %d, remembered %d", err, rep.Stopped, len(m.Explanations), len(mem))
	}
}

// A component of one file, with no route and no table, is not worth a request: its name is all
// there is to say.
func TestTrivialComponentsAreNotAskedAbout(t *testing.T) {
	m := explainModel()
	m.Nodes = append(m.Nodes, archdoc.Node{ID: "cmp:server/crypto", Name: "crypto", Kind: archdoc.Component, Parent: "svc:server",
		Files: []string{"server/src/repositories/crypto.repository.ts"}, Lines: 40, Prov: archdoc.Provenance{File: "server/src/repositories/crypto.repository.ts", Line: 1}})
	var prompts []string
	_, _, rep, err := Explain(context.Background(), fake(nil, &prompts), "test-model", m, nil)
	if err != nil || rep.Skipped != 1 || rep.Asked != 2 {
		t.Errorf("skipped %d, asked %d, %v", rep.Skipped, rep.Asked, err)
	}
	for _, p := range prompts {
		if strings.Contains(p, `component "crypto" inside`) {
			t.Error("a one-file component with no route or table was asked about")
		}
	}
}

// An answer in the agreed shape with nothing in it — seen live, three times in 83 — is asked again,
// and a component that still gets none is named in the report.
func TestAnEmptyAnswerIsAskedAgain(t *testing.T) {
	m := explainModel()
	ctl := `component "controllers" inside`
	svc := `component "services" inside`
	var prompts []string
	empty := `{"sentences":[{"cites":[],"text":""}]}`
	good := `{"sentences":[{"text":"It holds the services.","cites":["F1"]}]}`
	out, _, rep, err := Explain(context.Background(), fake(map[string][]string{ctl: {good}, svc: {empty, good}}, &prompts), "m", m, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Explanations) != 2 || len(rep.Unanswered) != 0 {
		t.Fatalf("explanations %d, unanswered %v — want the second answer kept", len(out.Explanations), rep.Unanswered)
	}
	if !strings.Contains(prompts[len(prompts)-1], "no sentence") {
		t.Errorf("the second request does not say what was wrong: %q", prompts[len(prompts)-1])
	}

	_, _, rep, err = Explain(context.Background(), fake(map[string][]string{ctl: {good}, svc: {empty, empty}}, &prompts), "m", m, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Unanswered) != 1 {
		t.Errorf("unanswered %v, want the component named", rep.Unanswered)
	}
}
