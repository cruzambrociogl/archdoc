package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func planItems() []core.PlanItem {
	at := func(l int) core.Provenance { return core.Provenance{File: "plan.md", Line: l} }
	return []core.PlanItem{
		{ID: "R1", Text: "Members can list albums.", Prov: at(3)},
		{ID: "R2", Text: "Members can delete their own comments.", Prov: at(4)},
		{ID: "R3", Text: "Comments should load quickly.", Prov: at(5)},
	}
}

// A check answers every requirement once, cites what "looks built" matched, and cites the
// addition each extra is about. What is wrong goes back once; the corrected check is kept.
func TestAPlanIsReadRequirementByRequirement(t *testing.T) {
	m := explainModel()
	items := planItems()
	facts := PlanFacts(m, items)
	adds := Additions([]core.Node{{ID: "ext:fcm", Name: "Firebase Cloud Messaging", Kind: core.External, Prov: core.Provenance{File: "fcm.ts", Line: 19}}}, nil)
	if len(facts) == 0 || !strings.Contains(facts[0].Text, "/api/albums") || len(adds) != 1 || adds[0].ID != "A1" {
		t.Fatalf("facts %+v, additions %+v", facts, adds)
	}
	complete, sent := scripted(t,
		// R1 built without a citation; R3 forgotten; an extra citing a fact, not an addition.
		Reply{Text: `{"requirements":[{"id":"R1","status":"built","sentence":"GET /api/albums lists them.","cites":[]},{"id":"R2","status":"missing","sentence":"No route deletes a comment.","cites":[]}],"extras":[{"sentence":"A push service.","cites":["F1"]}]}`},
		Reply{Text: `{"requirements":[{"id":"R1","status":"built","sentence":"GET /api/albums lists them.","cites":["F1"]},{"id":"R2","status":"missing","sentence":"No route deletes a comment.","cites":[]},{"id":"R3","status":"unclear","sentence":"Speed is not in how code is built.","cites":[]}],"extras":[{"sentence":"Notifications go through Firebase Cloud Messaging, which no requirement asks for.","cites":["A1"]}]}`},
	)
	check, rep, err := CheckPlan(context.Background(), complete, ExplainModel, items, facts, adds)
	if err != nil {
		t.Fatal(err)
	}
	if len(*sent) != 2 || rep.Attempts != 2 {
		t.Fatalf("%d requests", len(*sent))
	}
	got := []string{}
	for _, v := range check.Verdicts {
		got = append(got, v.Item.ID+" "+v.Status)
	}
	if strings.Join(got, ", ") != "R1 built, R2 missing, R3 unclear" {
		t.Errorf("verdicts: %v", got)
	}
	if v := check.Verdicts[0]; len(v.Claim.Cites) != 1 || v.Claim.Cites[0].Line != 7 || v.Item.Prov.Line != 3 {
		t.Errorf("R1: %+v", v)
	}
	if len(check.Extras) != 1 || check.Extras[0].Cites[0].File != "fcm.ts" {
		t.Errorf("extras: %+v", check.Extras)
	}
}

// A requirement with no usable verdict after every attempt is "couldn't check", and says so —
// never dropped, never guessed.
func TestARequirementLeftUnansweredSaysSo(t *testing.T) {
	bad := Reply{Text: `{"requirements":[{"id":"R1","status":"built","sentence":"Yes.","cites":["F404"]}],"extras":[]}`}
	complete, _ := scripted(t, bad, bad, bad)
	check, _, err := CheckPlan(context.Background(), complete, ExplainModel, planItems()[:1], PlanFacts(explainModel(), planItems()[:1]), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(check.Verdicts) != 1 || check.Verdicts[0].Status != "unclear" || !strings.Contains(check.Verdicts[0].Claim.Text, "No usable answer") {
		t.Errorf("verdicts: %+v", check.Verdicts)
	}
}

// On the wire: the requirements as the plan wrote them, and the facts as --explain words them.
func TestAPlanCheckSendsThePlanAndStructureAndNothingElse(t *testing.T) {
	srv, received := fakeAnthropic(t, `{"requirements":[{"id":"R1","status":"built","sentence":"GET /api/albums.","cites":["F1"]}],"extras":[]}`)
	rec := &Recorder{}
	complete := ClaudeWith(ExplainModel, PlanSchema(), PlanMaxTokens, PlanEffort, rec, option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"))
	m := explainModel()
	for i := range m.Nodes {
		m.Nodes[i].Prov.Line = 7777
	}
	items := planItems()[:1]
	if _, _, err := CheckPlan(context.Background(), complete, ExplainModel, items, PlanFacts(m, items), nil); err != nil {
		t.Fatal(err)
	}
	sent := ""
	for _, b := range *received {
		sent += string(b)
	}
	for _, want := range []string{"Members can list albums.", "GET /api/albums", `"effort":"medium"`} {
		if !strings.Contains(sent, want) {
			t.Errorf("%q was not sent", want)
		}
	}
	for _, leak := range []string{"7777", "plan.md", "provenance", "test-key"} {
		if strings.Contains(sent, leak) {
			t.Errorf("%q left the machine", leak)
		}
	}
}
