package semantic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Interpret explains a model the way an architect would: what the system is and how it is organised
// (the thesis), its features, concepts and areas, a few key scenarios told across flows and the
// background work they set off, and a review. From facts only — what the model already proves — in
// two requests:
//
//  1. The grouping request names and merges the seeds archdoc made by rule (interpret_seeds.go) into
//     concepts, areas and features. It never splits a seed, so whatever it answers, every group is a
//     union of groups a rule made.
//  2. The writing request writes the thesis, the scenarios and the review from a summary of those
//     groups.
//
// Nothing is sent back for repair: what the model gets wrong mechanically is fixed in code and
// counted (interpret_fix.go), and every claim it makes cites facts that resolve, or is dropped.
// Measured on Immich: about 1¢ for a first read with Haiku writing, 5¢ with Sonnet.

// The models the two requests go to. Both are Haiku by default: measured on three repositories, its
// writing matched Sonnet's at a twentieth of the cost, once the slips it makes more often — shortened
// IDs, steps given the same start, too many points — are repaired in code (interpret_fix.go). Sonnet
// stays a choice of the caller's (decisions, 18 Nov 2026).
const (
	Haiku      = "claude-haiku-5-5"
	Sonnet     = "claude-sonnet-5-5"
	GroupModel = Haiku
	WriteModel = Haiku
)

// The marks a claim carries (platform-surface §4.7): Proven when it rests on facts alone, a grouping a
// rule made; Written by AI when a model wrote it on top of them.
const (
	Proven      = "proven"
	WrittenByAI = "written-by-ai"
)

// Interpretation is what Interpret makes of a model: every claim with the facts it cites and its
// mark, and what it cost.
type Interpretation struct {
	Thesis Claim `json:"thesis"`
	// Features are the program by purpose: each a group of entry points.
	Features []Group `json:"features"`
	// Concepts are the domain's things, each a group of tables.
	Concepts []Group `json:"concepts"`
	// Areas divide the components of AreasOf, the container with the most, when it has enough to need
	// them.
	Areas     []Group    `json:"areas,omitempty"`
	AreasOf   string     `json:"areas_of,omitempty"`
	Scenarios []Scenario `json:"scenarios"`
	Review    []Point    `json:"review"`
	// Flags are features that look drawn by container or by kind rather than by purpose. Said, not
	// changed.
	Flags []string `json:"flags,omitempty"`
	// Fixes counts what was corrected in code, by kind of correction.
	Fixes map[string]int `json:"fixes,omitempty"`
	Calls []Call         `json:"calls"`
	// Cost is the run's cost in US dollars; CostKnown is false when a model's price is unknown.
	Cost      float64 `json:"cost"`
	CostKnown bool    `json:"cost_known"`
}

// Claim is a sentence and the facts it rests on.
type Claim struct {
	Text  string   `json:"text"`
	Cites []string `json:"cites"`
	Mark  string   `json:"mark"`
}

// Group is a feature, a concept or an area: its name, one sentence, and what it holds — entry IDs, a
// concept's table IDs, an area's component IDs.
type Group struct {
	Name    string   `json:"name"`
	Says    string   `json:"says"`
	Members []string `json:"members"`
	// Parts are the components a feature's flows pass through; empty for the others.
	Parts []string `json:"parts,omitempty"`
	// Everywhere is the area that holds the plumbing: the parts nearly every other one uses.
	Everywhere bool   `json:"everywhere,omitempty"`
	Mark       string `json:"mark"`
}

// Scenario is one key story, told along a flow's steps and through the background work it sets off.
type Scenario struct {
	Title string `json:"title"`
	// Flow is the entry point whose flow the story follows.
	Flow  string         `json:"flow"`
	Steps []ScenarioStep `json:"steps"`
	Mark  string         `json:"mark"`
}

// ScenarioStep is one sentence of a scenario, standing for steps From to To of the flow as the model
// was given it: the flow's own steps, then its background work numbered on.
type ScenarioStep struct {
	Text string `json:"text"`
	From int    `json:"from"`
	To   int    `json:"to"`
	// Background is a step that happens later, in a job: not while the request waits.
	Background bool `json:"background,omitempty"`
	// Crosses are the elements those steps touch: components, tables, jobs, routes.
	Crosses []string `json:"crosses,omitempty"`
}

// Point is one point of the review: an observation the facts show, or a question they cannot settle.
type Point struct {
	Text  string   `json:"text"`
	Kind  string   `json:"kind"` // "observation" or "question"
	Cites []string `json:"cites"`
	Mark  string   `json:"mark"`
}

// Call is one request a run sent.
type Call struct {
	Stage        string  `json:"stage"` // "group" or "write"
	Model        string  `json:"model"`
	Stop         string  `json:"stop,omitempty"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost"`
	CostKnown    bool    `json:"cost_known"`
}

const audience = "an architecture explanation of a software repository, for a developer who uses AI to write code " +
	"and can no longer keep up with what it built. They have some technical skill; what they lack is time."

const groupSystem = `You are grouping the parts of a software repository for ` + audience + `

archdoc has already made a first grouping by rule. Each seed below has an id. Name and merge seeds; never split ` +
	`one, and use the ids exactly as written.

- concepts: merge the table seeds into the domain's concepts, each a thing the system's users would recognise. ` +
	`Every table seed goes in exactly one concept. If there are no table seeds, give none.
- areas: merge the component seeds into 6 to 12 areas by responsibility. One seed already holds the shared ` +
	`plumbing and some join parts, by rule; the seed says why. Every component seed goes in exactly one area. If there ` +
	`are no component seeds, give none.
- systems: merge the entry-point seeds into the systems of the program. A system is a part that exists for one ` +
	`purpose, as its users would name it, served by routes, pages, jobs and commands together. Some seeds already join ` +
	`routes, jobs and pages, by rule; the seed says why. Name a system for what it is for, never for where it runs or ` +
	`how it starts: not "web app", "API server" or "background jobs". A seed may serve more than one system; every ` +
	`seed serves at least one.

Give each group a short name and one plain sentence.`

const writeSystem = `You are writing ` + audience + ` Explain it the way a software architect would explain it to a colleague.

Below are archdoc's facts about the repository, each with an id in square brackets and proven at a line, and ` +
	`archdoc's grouping of them into concepts, areas and systems. Write:

- thesis: two to four sentences: what the system is and how it is organised: its shape, its main path, what ` +
	`sets it apart. The point of view an architect opens with, not an inventory. Cite the facts that show it.
- scenarios: up to 3 key scenarios, from the flows given step by step. Tell each in 3 to 8 steps in plain ` +
	`words, through what happens later in the background where the flow sets that off. Each step names the flow step ` +
	`number it starts at: the first starts at 1, each later one after the one before.
- review: 3 to 8 points an architect would raise about the code: risks, couplings, inconsistencies, decisions ` +
	`worth knowing. Kind "observation" when the facts show it, "question" when they cannot settle it. Cite what each ` +
	`rests on. The concepts, areas and systems are archdoc's grouping, not the code's: use them to explain, never ` +
	`review them.

Cite facts by their exact id, without the brackets. Write plainly. Leave out what the facts do not support.`

var (
	str = map[string]any{"type": "string"}
	arr = map[string]any{"type": "array", "items": str}
)

func groupsOf(field string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"name", field, "seeds"},
		"properties": map[string]any{"name": str, field: str, "seeds": arr}}}
}

// GroupSchema and WriteSchema are the two answers' shapes, enforced by structured outputs.
var (
	GroupSchema = map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"concepts", "areas", "systems"},
		"properties": map[string]any{"concepts": groupsOf("meaning"), "areas": groupsOf("responsibility"), "systems": groupsOf("purpose")},
	}
	WriteSchema = map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"thesis", "scenarios", "review"},
		"properties": map[string]any{
			"thesis": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"text", "cites"},
				"properties": map[string]any{"text": str, "cites": arr}},
			"scenarios": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"title", "flow", "steps"},
				"properties": map[string]any{"title": str, "flow": str, "steps": map[string]any{"type": "array", "items": map[string]any{
					"type": "object", "additionalProperties": false, "required": []string{"text", "from"},
					"properties": map[string]any{"text": str, "from": map[string]any{"type": "integer"}}}}}}},
			"review": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"text", "kind", "cites"},
				"properties": map[string]any{"text": str, "kind": map[string]any{"type": "string", "enum": []string{"observation", "question"}},
					"cites": arr}}},
		},
	}
)

// Before sending, only a request's size is known. Measured on Immich: the ids and paths these
// requests are made of tokenise densely, about 2.4 characters a token. Stage 1 answers about fifty
// tokens per seed it names and merges (Immich: 187 seeds, about 10,000 tokens); stage 2 about 3,000.
const (
	charsPerToken     = 2.4
	writeAnswerTokens = 3000
	answerCap         = 16000
)

// InterpretPlans are the two requests Interpret would send for m, without sending them, and the
// tokens each answer should come to. The second request depends on the first's answer, so it is
// sized with every seed a group of its own: an upper bound.
func InterpretPlans(m core.Model) (group, write Planned, groupAnswer, writeAnswer int64) {
	s := seed(m)
	group = Planned{Element: core.SystemID, System: groupSystem, Prompt: s.groupInput()}
	bound, _ := s.summary(s.ungrouped())
	write = Planned{Element: core.SystemID, System: writeSystem, Prompt: bound}
	seeds := len(s.tableIDs) + len(s.areaIDs) + len(s.seedIDs)
	return group, write, min(answerCap, int64(300+50*seeds)), writeAnswerTokens
}

// Tokens estimates the tokens a request of Interpret sends.
func (p Planned) Tokens() int64 { return int64(float64(p.Bytes()) / charsPerToken) }

// ungrouped is every seed a group of its own: what stage 2 is sized by before stage 1 has answered.
func (s *seeding) ungrouped() grouping {
	var g grouping
	for _, id := range s.tableIDs {
		g.Concepts = append(g.Concepts, rawGroup{Name: s.tables[id].Key, Seeds: []string{id}})
	}
	for _, id := range s.areaIDs {
		g.Areas = append(g.Areas, rawGroup{Name: id, Seeds: []string{id}})
	}
	for _, id := range s.seedIDs {
		g.Systems = append(g.Systems, rawGroup{Name: s.seeds[id].Key, Seeds: []string{id}})
	}
	return g
}

// ErrNotUsable is an answer with nothing left to show once its unresolved citations are dropped.
var ErrNotUsable = errors.New("the explanation cited nothing that resolves")

// Interpret runs both requests: group answers stage 1 and write stage 2, each on the model named
// beside it, which is what its cost is counted at. Whatever happened, the calls sent so far come back
// with their cost, so a failure is still accounted for.
func Interpret(ctx context.Context, group Completer, groupModel string, write Completer, writeModel string, m core.Model) (*Interpretation, error) {
	s := seed(m)
	out := &Interpretation{CostKnown: true, Fixes: map[string]int{}}
	call := func(stage, model string, complete Completer, system, prompt string, into any) error {
		reply, err := complete(ctx, system, []Turn{{Role: "user", Text: prompt}})
		c := Call{Stage: stage, Model: model, Stop: reply.Stop, InputTokens: reply.InputTokens, OutputTokens: reply.OutputTokens}
		c.Cost, c.CostKnown = Cost(model, reply.InputTokens, reply.OutputTokens)
		if err == nil || reply.InputTokens > 0 {
			out.Calls = append(out.Calls, c)
			out.Cost += c.Cost
			out.CostKnown = out.CostKnown && c.CostKnown
		}
		switch {
		case err != nil:
			return err
		case reply.Refused:
			return fmt.Errorf("%s: the model declined to answer", stage)
		case reply.Stop == "max_tokens":
			return fmt.Errorf("%s: the answer was cut off at %d tokens", stage, reply.OutputTokens)
		}
		if err := json.Unmarshal([]byte(reply.Text), into); err != nil {
			return fmt.Errorf("%s: the answer is not the schema's: %w", stage, err)
		}
		return nil
	}

	var a1 stage1Answer
	if err := call("group", groupModel, group, groupSystem, s.groupInput(), &a1); err != nil {
		return out, err
	}
	g, fixes := s.fixGroups(a1)
	merge(out.Fixes, fixes)
	out.Flags = s.drawnByContainer(g)

	summary, given := s.summary(g)
	var a2 stage2Answer
	if err := call("write", writeModel, write, writeSystem, summary, &a2); err != nil {
		return out, err
	}
	thesis, scenarios, review, fixes2 := fixWriting(a2, citable(m), given)
	merge(out.Fixes, fixes2)
	out.Thesis, out.Scenarios, out.Review = thesis, scenarios, review
	out.Features, out.Concepts, out.Areas = s.features(g), s.concepts(g), s.areaGroups(g)
	if len(out.Areas) > 0 {
		out.AreasOf = s.areaOf
	}
	if len(out.Fixes) == 0 {
		out.Fixes = nil
	}
	if len(thesis.Cites) == 0 && len(review) == 0 {
		return out, ErrNotUsable
	}
	return out, nil
}

func merge(into, from map[string]int) {
	for k, v := range from {
		into[k] += v
	}
}

func mark(g rawGroup) string {
	if g.ByRule {
		return Proven
	}
	return WrittenByAI
}

func (s *seeding) features(g grouping) []Group {
	var out []Group
	for _, f := range g.Systems {
		members := s.expand("systems", f.Seeds)
		parts := map[string]bool{}
		for _, x := range members {
			if c := s.entries[x].Component; c != "" {
				parts[c] = true
			}
			if r := s.reach[x]; r != nil {
				for _, c := range r.components {
					parts[c] = true
				}
			}
		}
		out = append(out, Group{Name: f.Name, Says: f.Says, Members: members, Parts: sortedKeys(parts), Mark: mark(f)})
	}
	return out
}

func (s *seeding) concepts(g grouping) []Group {
	tableID := map[string]string{}
	for _, n := range s.m.Nodes {
		if n.Kind == core.Table {
			tableID[n.Name] = n.ID
		}
	}
	var out []Group
	for _, c := range g.Concepts {
		var members []string
		for _, t := range s.expand("concepts", c.Seeds) {
			members = append(members, tableID[t])
		}
		out = append(out, Group{Name: c.Name, Says: c.Says, Members: members, Mark: mark(c)})
	}
	return out
}

func (s *seeding) areaGroups(g grouping) []Group {
	var out []Group
	for _, a := range g.Areas {
		out = append(out, Group{Name: a.Name, Says: a.Says, Members: s.expand("areas", a.Seeds),
			Everywhere: contains(a.Seeds, "plumbing"), Mark: mark(a)})
	}
	return out
}
