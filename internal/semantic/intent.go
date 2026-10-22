package semantic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Did it build what I asked? (F-41, platform-surface §6.3): the requirements of the plan files a
// person named, each read against what archdoc read from the code — looks built, looks missing, or
// couldn't be checked — and what the latest version added that no requirement asked for.
//
// Matching a plan's sentence to code is the model's reading, never a fact: every verdict is marked
// as written by AI, and "looks built" must cite the facts it matched, or it is refused (F-36).
// "Looks missing" says what was looked for. The facts are chosen here, without a model — for each
// requirement, the few that share its words — and the whole plan goes in one request, so a check
// costs one request; it is remembered against the plan and its facts, so opening it again costs
// nothing.

const intentVersion = "intent-1"

// StructureAndPlan is what a plan check sends: what --explain sends, and the plan's requirements
// as written.
const StructureAndPlan = "structure-and-your-plan"

// PlanMaxTokens bounds one check, reasoning included: a sentence per requirement, and the extras.
const PlanMaxTokens = 12000

// PlanEffort is how hard the model works on a check: reading a plan against code is judgment, which
// is what the check is for, so it is not the least.
const PlanEffort = "medium"

const (
	factsPerItem  = 12
	mostPlanFacts = 160
)

// Verdict is one requirement read against the code.
type Verdict struct {
	Item core.PlanItem `json:"item"`
	// Status is "built" (looks built), "missing" (looks missing) or "unclear" (couldn't check).
	Status string `json:"status"`
	// Claim is the sentence: where it was matched, citing what; what was looked for; or why it
	// could not be checked.
	Claim core.Claim `json:"claim"`
}

// PlanCheck is a plan read against the code.
type PlanCheck struct {
	Verdicts []Verdict `json:"verdicts"`
	// Extras are what the latest version added that no requirement asks for, each citing it.
	Extras    []core.Claim `json:"extras"`
	Facts     int          `json:"facts"`
	Additions int          `json:"additions"`
	Model     string       `json:"model,omitempty"`
}

// PlanFacts chooses the facts a plan is read against: for each requirement, the ones that share
// its words, numbered once across the plan.
func PlanFacts(m core.Model, items []core.PlanItem) []Fact {
	seen := map[string]bool{}
	var out []Fact
	for _, it := range items {
		for i, f := range QuestionFacts(m, it.Text) {
			if i == factsPerItem || len(out) == mostPlanFacts {
				break
			}
			if !seen[f.Text] {
				seen[f.Text] = true
				out = append(out, Fact{ID: fmt.Sprintf("F%d", len(out)+1), Text: f.Text, Prov: f.Prov})
			}
		}
	}
	return out
}

// CheckKey is what a check is remembered by: the plan as written, its facts and the additions.
func CheckKey(items []core.PlanItem, facts, additions []Fact) string {
	h := sha256.New()
	h.Write([]byte(intentVersion))
	for _, it := range items {
		h.Write([]byte("\x00" + it.ID + " " + it.Text))
	}
	for _, f := range append(append([]Fact(nil), facts...), additions...) {
		h.Write([]byte("\x00" + f.ID + " " + f.Text))
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// PlanPrompt is the one message a check sends: the requirements, the facts, and the additions.
func PlanPrompt(items []core.PlanItem, facts, additions []Fact) string {
	var b strings.Builder
	b.WriteString("Requirements, from the plan:\n")
	for _, it := range items {
		fmt.Fprintf(&b, "%s: %s\n", it.ID, it.Text)
	}
	b.WriteString("\nFacts archdoc read from the code:\n")
	for _, f := range facts {
		fmt.Fprintf(&b, "%s: %s\n", f.ID, f.Text)
	}
	b.WriteString("\nAdded in the latest version:\n")
	if len(additions) == 0 {
		b.WriteString("(nothing)\n")
	}
	for _, f := range additions {
		fmt.Fprintf(&b, "%s: %s\n", f.ID, f.Text)
	}
	return b.String()
}

// PlanSystem is what the model is told a check must be.
const PlanSystem = `You read a plan against what was built, for a software system's documentation.

You receive the plan's requirements, numbered R1, R2, …; facts archdoc read from the code, numbered
F1, F2, …; and what the latest version added, numbered A1, A2, …. For every requirement give one
verdict:
- "built" when the facts show it: one sentence saying where, citing the F facts it matched.
- "missing" when they do not: one sentence saying what you looked for and did not find, for example
  "No route, job or page mentions deleting a comment." Cite nearby facts if any help.
- "unclear" when the requirement cannot be checked from how code is built — speed, look, feel: one
  sentence saying why, and what a checkable line would say.
Then list each addition no requirement asks for, one sentence each, citing its A fact. Say only what
the facts support; never claim something is built without citing it. Do not mention archdoc or the
facts themselves.`

// PlanSchema is the strict shape of a check.
func PlanSchema() map[string]any {
	claim := func(extra map[string]any, required ...string) map[string]any {
		props := map[string]any{
			"sentence": map[string]any{"type": "string"},
			"cites":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}
		for k, v := range extra {
			props[k] = v
		}
		return map[string]any{"type": "object", "additionalProperties": false, "required": append([]string{"sentence", "cites"}, required...), "properties": props}
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"requirements", "extras"},
		"properties": map[string]any{
			"requirements": map[string]any{"type": "array", "items": claim(map[string]any{
				"id":     map[string]any{"type": "string"},
				"status": map[string]any{"type": "string", "enum": []string{"built", "missing", "unclear"}},
			}, "id", "status")},
			"extras": map[string]any{"type": "array", "items": claim(nil)},
		},
	}
}

type checked struct {
	Requirements []struct {
		ID       string   `json:"id"`
		Status   string   `json:"status"`
		Sentence string   `json:"sentence"`
		Cites    []string `json:"cites"`
	} `json:"requirements"`
	Extras []struct {
		Sentence string   `json:"sentence"`
		Cites    []string `json:"cites"`
	} `json:"extras"`
}

// CheckPlan reads a plan against the code in one request, and keeps what it can check: every
// requirement answered once, every citation one of the facts given, "looks built" cited. What is
// wrong goes back to be corrected, within the retry budget; a requirement still without a usable
// verdict is left "couldn't check", and says so.
func CheckPlan(ctx context.Context, complete Completer, model string, items []core.PlanItem, facts, additions []Fact) (PlanCheck, Report, error) {
	rep := Report{Model: model}
	out := PlanCheck{Facts: len(facts), Additions: len(additions), Model: model}
	byID := map[string]Fact{}
	for _, f := range append(append([]Fact(nil), facts...), additions...) {
		byID[f.ID] = f
	}
	turns := []Turn{{Role: "user", Text: PlanPrompt(items, facts, additions)}}
	for attempt := 1; attempt <= Attempts; attempt++ {
		rep.Attempts++
		rep.BytesSent += len(PlanSystem)
		for _, t := range turns {
			rep.BytesSent += len(t.Text)
		}
		reply, err := complete(ctx, PlanSystem, turns)
		if err != nil {
			return out, rep, err
		}
		rep.InputTokens += reply.InputTokens
		rep.OutputTokens += reply.OutputTokens
		if reply.Refused {
			return out, rep, fmt.Errorf("the model declined to read the plan")
		}
		var c checked
		if err := json.Unmarshal([]byte(strings.TrimSpace(reply.Text)), &c); err != nil {
			return out, rep, fmt.Errorf("the model's check was not in the agreed shape (stopped by %s)", reply.Stop)
		}
		verdicts, extras, problems := checkVerdicts(c, items, byID)
		out.Verdicts, out.Extras = verdicts, extras
		if len(problems) == 0 || attempt == Attempts {
			return out, rep, nil
		}
		turns = append(turns,
			Turn{Role: "assistant", Text: reply.Text},
			Turn{Role: "user", Text: "Some verdicts were refused. Return a corrected, complete check.\n\n" + strings.Join(problems, "\n")},
		)
	}
	return out, rep, nil
}

// checkVerdicts keeps a verdict per requirement, in plan order. A requirement with no usable
// verdict is "couldn't check", saying so — never dropped, never guessed.
func checkVerdicts(c checked, items []core.PlanItem, byID map[string]Fact) ([]Verdict, []core.Claim, []string) {
	var problems []string
	cite := func(ids []string, what string, allowed func(string) bool) ([]string, []core.Provenance, bool) {
		var facts []string
		var cites []core.Provenance
		for _, id := range ids {
			f, ok := byID[strings.TrimSpace(id)]
			if !ok || !allowed(f.ID) {
				problems = append(problems, fmt.Sprintf("%s cites %q, which is not one of the facts given for it", what, id))
				return nil, nil, false
			}
			facts, cites = append(facts, f.Text), append(cites, f.Prov)
		}
		return facts, cites, true
	}
	isFact := func(id string) bool { return strings.HasPrefix(id, "F") }
	isAddition := func(id string) bool { return strings.HasPrefix(id, "A") }

	given := map[string]Verdict{}
	for _, r := range c.Requirements {
		id := strings.TrimSpace(r.ID)
		text := strings.TrimSpace(r.Sentence)
		switch {
		case text == "" || len(text) > maxSentence*2:
			problems = append(problems, fmt.Sprintf("%s: one sentence, under %d characters", id, maxSentence*2))
			continue
		case r.Status == "built" && len(r.Cites) == 0:
			problems = append(problems, fmt.Sprintf("%s is called built but cites no fact: cite what matched, or call it missing", id))
			continue
		}
		facts, cites, ok := cite(r.Cites, id, isFact)
		if !ok {
			continue
		}
		given[id] = Verdict{Status: r.Status, Claim: core.Claim{Text: text, Facts: facts, Cites: cites}}
	}
	var verdicts []Verdict
	for _, it := range items {
		v, ok := given[it.ID]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s has no usable verdict: give one", it.ID))
			v = Verdict{Status: "unclear", Claim: core.Claim{Text: "No usable answer came back for this requirement."}}
		}
		v.Item = it
		verdicts = append(verdicts, v)
	}
	var extras []core.Claim
	for i, x := range c.Extras {
		text := strings.TrimSpace(x.Sentence)
		if text == "" || len(x.Cites) == 0 {
			problems = append(problems, fmt.Sprintf("extra %d must cite the addition it is about", i+1))
			continue
		}
		facts, cites, ok := cite(x.Cites, fmt.Sprintf("extra %d", i+1), isAddition)
		if ok {
			extras = append(extras, core.Claim{Text: text, Facts: facts, Cites: cites})
		}
	}
	return verdicts, extras, problems
}

// Additions are what the latest version added, as facts A1, A2, …: new parts, routes, pages, jobs
// and outside services — what a plan check asks whether anyone asked for.
func Additions(nodes []core.Node, entries []core.Entry) []Fact {
	var out []Fact
	add := func(text string, p core.Provenance) {
		out = append(out, Fact{ID: fmt.Sprintf("A%d", len(out)+1), Text: text, Prov: p})
	}
	for _, n := range nodes {
		switch n.Kind {
		case core.Application, core.Component, core.Datastore, core.Queue, core.External, core.Table:
			add(fmt.Sprintf("new %s %q", strings.ReplaceAll(string(n.Kind), "_", " "), n.Name), n.Prov)
		}
		if len(out) == 40 {
			return out
		}
	}
	for _, e := range entries {
		text := fmt.Sprintf("new route %s %s, handled by %s", e.Method, e.Path, e.Handler)
		if e.Kind == "page" {
			text = "new page " + e.Path
		}
		add(text, e.Prov)
		if len(out) == 60 {
			break
		}
	}
	return out
}
