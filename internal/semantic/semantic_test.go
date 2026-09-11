package semantic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// These tests never touch the network. The model is replaced by a function returning canned
// answers, which is enough: everything worth testing here is what archdoc does with an answer,
// not what the model says.

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
			{ID: "svc:db", Name: "db", Kind: archdoc.Datastore, Evidence: archdoc.Declared, Prov: at(20)},
		},
		Edges: []archdoc.Edge{
			{From: "actor:user", To: "svc:api", Label: "reaches", Traffic: true, Prov: []archdoc.Provenance{at(6)}},
			{From: "svc:api", To: "svc:db", Label: "connects to", Traffic: true, Prov: []archdoc.Provenance{at(11)}},
		},
	}.Normalise()
}

type op struct{ Kind, Target, To, Value string }

func reply(ops ...op) string {
	type wire struct {
		Kind   string `json:"kind"`
		Target string `json:"target"`
		To     string `json:"to"`
		Value  string `json:"value"`
	}
	w := struct {
		Operations []wire `json:"operations"`
	}{}
	for _, o := range ops {
		w.Operations = append(w.Operations, wire(o))
	}
	b, _ := json.Marshal(w)
	return string(b)
}

// scripted answers each call with the next canned reply and records what it was sent.
func scripted(t *testing.T, replies ...Reply) (Completer, *[][]Turn) {
	t.Helper()
	var sent [][]Turn
	i := 0
	return func(_ context.Context, _ string, turns []Turn) (Reply, error) {
		sent = append(sent, append([]Turn(nil), turns...))
		if i >= len(replies) {
			t.Fatalf("the model was called %d times; only %d replies were scripted", i+1, len(replies))
		}
		r := replies[i]
		i++
		return r, nil
	}, &sent
}

func good() Reply {
	return Reply{Text: reply(
		op{"set_description", "svc:api", "", "Serves the public API"},
		op{"set_technology", "svc:api", "", "Go"},
		op{"set_edge_label", "svc:api", "svc:db", "reads and writes orders"},
	)}
}

func TestLabelAppliesTheModelsOperations(t *testing.T) {
	complete, _ := scripted(t, good())

	out, rep, err := Label(context.Background(), complete, Model, fixture())
	if err != nil {
		t.Fatalf("label: %v", err)
	}

	api, _ := out.Node("svc:api")
	if api.Description != "Serves the public API" || api.Technology != "Go" {
		t.Errorf("operations not applied: %+v", api)
	}
	if rep.Ops != 3 || rep.Attempts != 1 {
		t.Errorf("report: %+v", rep)
	}
}

// PRV-05 depends on this: every value the model supplied must say the model supplied it, so the
// document can mark it as interpretation rather than fact.
func TestModelValuesAreMarkedAsInterpretation(t *testing.T) {
	complete, _ := scripted(t, good())
	out, _, err := Label(context.Background(), complete, Model, fixture())
	if err != nil {
		t.Fatal(err)
	}

	api, _ := out.Node("svc:api")
	for what, p := range map[string]archdoc.Provenance{"description": api.DescProv, "technology": api.TechProv} {
		if !p.Origin.Interpretation() {
			t.Errorf("%s is not marked as interpretation: %+v", what, p)
		}
		if p.Note != Model {
			t.Errorf("%s does not name the model that wrote it: %q", what, p.Note)
		}
	}
}

// AC-2 in miniature — the thesis. With the model on, the words change; the structure does not.
// Every box and every arrow present without the model is present with it, and nothing more.
func TestLabellingNeverChangesStructure(t *testing.T) {
	before := fixture()
	complete, _ := scripted(t, good())

	after, _, err := Label(context.Background(), complete, Model, before)
	if err != nil {
		t.Fatal(err)
	}

	if len(after.Nodes) != len(before.Nodes) || len(after.Edges) != len(before.Edges) {
		t.Fatalf("structure changed: %d/%d nodes, %d/%d edges",
			len(after.Nodes), len(before.Nodes), len(after.Edges), len(before.Edges))
	}
	for i := range before.Nodes {
		if after.Nodes[i].ID != before.Nodes[i].ID || after.Nodes[i].Kind != before.Nodes[i].Kind {
			t.Errorf("node %d changed identity or kind", i)
		}
	}
	for i := range before.Edges {
		if after.Edges[i].From != before.Edges[i].From || after.Edges[i].To != before.Edges[i].To {
			t.Errorf("edge %d changed endpoints", i)
		}
	}
}

// VAL-07. An operation the validator rejects goes back to the model with the reason, and a
// corrected answer is accepted.
func TestRejectedOperationsAreSentBackForCorrection(t *testing.T) {
	bad := Reply{Text: reply(op{"set_description", "svc:ghost", "", "Does not exist"})}
	complete, sent := scripted(t, bad, good())

	out, rep, err := Label(context.Background(), complete, Model, fixture())
	if err != nil {
		t.Fatalf("a correctable answer failed: %v", err)
	}
	if rep.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", rep.Attempts)
	}

	second := (*sent)[1]
	feedback := second[len(second)-1].Text
	if !strings.Contains(feedback, "svc:ghost") {
		t.Errorf("the retry did not tell the model what was wrong:\n%s", feedback)
	}
	if api, _ := out.Node("svc:api"); api.Description == "" {
		t.Error("the corrected answer was not applied")
	}
}

// VAL-08. When the budget runs out, the run fails loudly and the model comes back untouched.
func TestExhaustedBudgetChangesNothing(t *testing.T) {
	bad := Reply{Text: reply(op{"set_description", "svc:ghost", "", "Does not exist"})}
	complete, _ := scripted(t, bad, bad, bad)

	before := fixture()
	out, rep, err := Label(context.Background(), complete, Model, before)
	if err == nil {
		t.Fatal("three rejected answers did not fail the run")
	}
	if rep.Attempts != Attempts {
		t.Errorf("attempts = %d, want %d", rep.Attempts, Attempts)
	}
	if api, _ := out.Node("svc:api"); api.Description != "" {
		t.Error("a failed run left a partial change behind")
	}
	if !strings.Contains(err.Error(), "svc:ghost") {
		t.Errorf("the failure does not name the offending operation: %v", err)
	}
}

// Hard rule 4 holds even if the schema were somehow bypassed: a structural operation from the
// model is rejected by the validator, not trusted.
func TestStructuralOperationsFromTheModelNeverApply(t *testing.T) {
	sneaky := Reply{Text: reply(op{"exclude", "svc:db", "", ""})}
	complete, _ := scripted(t, sneaky, sneaky, sneaky)

	out, _, err := Label(context.Background(), complete, Model, fixture())
	if err == nil {
		t.Error("the model was allowed to exclude an element")
	}
	if _, ok := out.Node("svc:db"); !ok {
		t.Error("an element disappeared")
	}
}

func TestRefusalLeavesTheModelUnchanged(t *testing.T) {
	complete, _ := scripted(t, Reply{Refused: true})

	out, _, err := Label(context.Background(), complete, Model, fixture())
	if err == nil {
		t.Fatal("a refusal was treated as an answer")
	}
	if api, _ := out.Node("svc:api"); api.Description != "" {
		t.Error("a refused run changed the model")
	}
}

// AC-8 — structure only. The prompt must carry names and relationships and nothing read from a
// file: no paths, no line numbers, no provenance.
func TestPromptCarriesStructureOnly(t *testing.T) {
	complete, sent := scripted(t, good())
	if _, _, err := Label(context.Background(), complete, Model, fixture()); err != nil {
		t.Fatal(err)
	}

	prompt := (*sent)[0][0].Text
	for _, leak := range []string{"docker-compose.yml", "provenance", "\"line\""} {
		if strings.Contains(prompt, leak) {
			t.Errorf("the prompt contains %q — only structure may leave the machine", leak)
		}
	}
	for _, want := range []string{"svc:api", "svc:db", "connects to"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt is missing %q", want)
		}
	}
}

// The schema is the first line of defence: structural verbs cannot even be expressed.
func TestSchemaOffersOnlyLabellingVerbs(t *testing.T) {
	b, _ := json.Marshal(Schema())
	s := string(b)

	for _, forbidden := range []string{"exclude", "add_edge", "remove_edge", "set_kind", "group"} {
		if strings.Contains(s, `"`+forbidden+`"`) {
			t.Errorf("the schema offers %q", forbidden)
		}
	}
	if !strings.Contains(s, `"additionalProperties":false`) {
		t.Error("the schema is not strict")
	}
}

// The same model must produce the same prompt, byte for byte — it is what makes the request
// cacheable and two runs comparable.
func TestPromptIsDeterministic(t *testing.T) {
	first, _ := describe(fixture())
	for range 10 {
		if got, _ := describe(fixture()); got != first {
			t.Fatal("the prompt differs between runs")
		}
	}
}
