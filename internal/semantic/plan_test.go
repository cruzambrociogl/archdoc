package semantic

import (
	"context"
	"testing"
)

// What a dry run shows is what a run sends: the plan's first message for each component is exactly
// the first message the run sends about it, in the same order, for the same components — none of
// the ones a remembered answer covers.
func TestExplainPlanIsWhatARunSends(t *testing.T) {
	m := explainModel()
	var firsts []string
	asked := func(_ context.Context, _ string, turns []Turn) (Reply, error) {
		if len(turns) == 1 {
			firsts = append(firsts, turns[0].Text)
		}
		return Reply{Text: `{"sentences":[{"text":"Does a thing.","cites":["F1"]}]}`}, nil
	}
	plans := ExplainPlan(m, nil, "", 0)
	if _, _, _, err := Explain(context.Background(), asked, "test-model", m, nil); err != nil {
		t.Fatal(err)
	}
	if len(plans) != len(firsts) || len(plans) != 2 {
		t.Fatalf("planned %d requests, sent %d", len(plans), len(firsts))
	}
	for i := range plans {
		if plans[i].Prompt != firsts[i] || plans[i].System != explainSystem {
			t.Errorf("request %d: planned\n%s\nsent\n%s", i+1, plans[i].Prompt, firsts[i])
		}
	}

	// Once answered, nothing is planned: the memory covers both.
	_, mem, _, _ := Explain(context.Background(), asked, "test-model", m, nil)
	if again := ExplainPlan(m, mem, "", 0); len(again) != 0 {
		t.Errorf("planned %d requests that the memory answers", len(again))
	}
	if one := ExplainPlan(m, nil, "services", 0); len(one) != 1 || one[0].Element != "cmp:server/services" {
		t.Errorf("--only services planned %+v", one)
	}
	if lim := ExplainPlan(m, nil, "", 1); len(lim) != 1 {
		t.Errorf("--limit 1 planned %d", len(lim))
	}
}

// The label plan is the label request: the same system prompt and the same description.
func TestLabelPlanIsWhatARunSends(t *testing.T) {
	m := explainModel()
	var sent []Turn
	var sys string
	capture := func(_ context.Context, s string, turns []Turn) (Reply, error) {
		sys, sent = s, turns
		return Reply{Text: `{"operations":[]}`}, nil
	}
	p, err := LabelPlan(m)
	if err != nil {
		t.Fatal(err)
	}
	Label(context.Background(), capture, "test-model", m)
	if len(sent) == 0 || p.Prompt != sent[0].Text || p.System != sys {
		t.Errorf("the label plan is not the request sent")
	}
}
