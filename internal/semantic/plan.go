package semantic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// What a paid run would send, worked out without sending it: so a person can be told what it will
// cost, and shown it, before it leaves the machine (archdoc label --dry-run, archdoc explain
// --dry-run). Built by the same functions the requests are, so what is shown is what would go.

// Planned is one request a run would make: the element it is about, and its first message. A
// request the validator sends back for correction repeats it, with the errors, and is not planned.
type Planned struct {
	Element string
	System  string
	Prompt  string
}

// Bytes is how much the request carries, system prompt included.
func (p Planned) Bytes() int { return len(p.System) + len(p.Prompt) }

// LabelPlan is the one request --label makes for this model.
func LabelPlan(m archdoc.Model) (Planned, error) {
	prompt, err := describe(m)
	if err != nil {
		return Planned{}, err
	}
	return Planned{Element: archdoc.SystemID, System: system, Prompt: prompt}, nil
}

// ExplainPlan is the requests ExplainSome would make with the same memory, filter and limit: one
// per component whose facts have no remembered answer and are worth asking about.
func ExplainPlan(m archdoc.Model, mem Memory, only string, limit int) []Planned {
	var out []Planned
	for _, id := range askable(m, mem, only) {
		if limit > 0 && len(out) == limit {
			break
		}
		out = append(out, Planned{Element: id, System: explainSystem, Prompt: factsPrompt(ComponentFacts(m, id))})
	}
	return out
}

// askable is the components a run would ask about, in the order it asks: no remembered answer for
// their facts, not too small to ask about, and matching only.
func askable(m archdoc.Model, mem Memory, only string) []string {
	var ids []string
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Component {
			ids = append(ids, n.ID)
		}
	}
	sort.Strings(ids)
	var out []string
	for _, id := range ids {
		facts := ComponentFacts(m, id)
		if _, ok := mem[Fingerprint(facts)]; ok || trivial(facts) || !strings.Contains(id, only) {
			continue
		}
		out = append(out, id)
	}
	return out
}

// factsPrompt is the first message of an explanation's request: the facts, one a line, by ID.
func factsPrompt(facts []Fact) string {
	var list strings.Builder
	for _, f := range facts {
		fmt.Fprintf(&list, "%s: %s\n", f.ID, f.Text)
	}
	return "Facts:\n\n" + list.String()
}
