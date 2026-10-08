package semantic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Explanations (F-19, F-30, F-36): a few sentences on what a component does, each one citing the
// facts it rests on.
//
// The model is given a component's facts as a numbered list — F1, F2, … — and answers with
// sentences and the numbers they cite. Three rules, none of them a prompt instruction:
//
//  1. What leaves the machine is names, and the code's own descriptions of its routes (D-8, F-35):
//     the component's name, its files' paths, the components it uses and is used by, the routes it
//     handles by method, path and handler — with the summary a decorator or a docstring's first
//     line states for each — and the tables it declares with their columns. Never a line of code,
//     never any other string the code holds, and never a provenance.
//  2. A sentence that cites nothing, or cites a fact it was not given, is refused (F-36). The
//     model may correct itself within the retry budget; what is still refused is reported and
//     left out. A component is never described by an unchecked sentence.
//  3. An answer is remembered against a fingerprint of the facts it was given (F-30). While the
//     facts are unchanged the same sentences are reused — without asking, and without --explain —
//     so a run with nothing new costs nothing and changes nothing.

// explainVersion is part of every fingerprint: changing the instructions or the fact wording asks
// again, rather than reusing an answer to a different question.
const explainVersion = "explain-2"

// Fact is one thing known about an element, numbered for the model to cite. Prov is never sent.
type Fact struct {
	ID   string
	Text string
	Prov archdoc.Provenance
}

// Memory is the remembered answers, by the fingerprint of the facts each was given.
type Memory map[string]Remembered

// Remembered is one answer and the element it is about. Knowing the element is what lets an
// answer outlive its facts: when they change, the last answer is still shown — marked as written
// for an earlier version — until --explain asks again.
type Remembered struct {
	Element string          `json:"element,omitempty"`
	Claims  []archdoc.Claim `json:"claims"`
}

// UnmarshalJSON also reads the first format of the memory file: a bare list of claims.
func (r *Remembered) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '[' {
		return json.Unmarshal(b, &r.Claims)
	}
	type plain Remembered
	return json.Unmarshal(b, (*plain)(r))
}

// ExplainReport says what an explanation run did, for the run log.
type ExplainReport struct {
	Report
	Asked      int // components the model was asked about
	Remembered int // components whose remembered answer was reused
	Stale      int // components shown an answer written for an earlier version of their facts
	Refused    int // sentences refused: no citation, or one that does not resolve
	// Unanswered are components the model gave no usable answer for — an empty reply, or one not
	// in the agreed shape. Each is left without an explanation; the others keep theirs.
	Unanswered []string
	// Stopped is why asking ended early — the API could not be reached, a rate limit — when it
	// did. What was answered before it is kept and remembered.
	Stopped string
}

const (
	maxFiles     = 40
	maxEntries   = 40
	maxSentences = 4
	maxSentence  = 320
)

// ComponentFacts lists what is known about a component, in a fixed order. Every fact is a name, a
// path or a count, except a route's summary: the one line the code states about what the route does.
func ComponentFacts(m archdoc.Model, id string) []Fact {
	n, ok := m.Node(id)
	if !ok {
		return nil
	}
	var out []Fact
	add := func(text string, p archdoc.Provenance) {
		out = append(out, Fact{ID: fmt.Sprintf("F%d", len(out)+1), Text: text, Prov: p})
	}
	name := func(id string) string {
		if x, ok := m.Node(id); ok {
			return x.Name
		}
		return id
	}
	add(fmt.Sprintf("component %q inside container %q: %d files, %d lines, at %s", n.Name, name(n.Parent), len(n.Files), n.Lines, n.Dir), n.Prov)
	for i, f := range n.Files {
		if i == maxFiles {
			add(fmt.Sprintf("… and %d more files", len(n.Files)-maxFiles), n.Prov)
			break
		}
		add("file "+f, archdoc.Provenance{File: f, Line: 1})
	}
	for _, e := range m.Edges {
		if e.From == id && len(e.Prov) > 0 {
			add(fmt.Sprintf("uses component %q (%d imports)", name(e.To), e.Weight), e.Prov[0])
		}
	}
	for _, e := range m.Edges {
		if e.To == id && len(e.Prov) > 0 {
			if from, ok := m.Node(e.From); ok && from.Kind == archdoc.Component {
				add(fmt.Sprintf("is used by component %q (%d imports)", from.Name, e.Weight), e.Prov[0])
			}
		}
	}
	count := 0
	for _, e := range m.Entries {
		if e.Component != id {
			continue
		}
		if count == maxEntries {
			add("… and more routes", e.Prov)
			break
		}
		count++
		if e.Kind == "page" {
			add("is the page at "+e.Path, e.Prov)
		} else {
			text := fmt.Sprintf("handles %s %s in %s", e.Method, e.Path, e.Handler)
			if e.Summary != "" {
				text += fmt.Sprintf(", described by the code as %q", e.Summary)
			}
			add(text, e.Prov)
		}
	}
	files := map[string]bool{}
	for _, f := range n.Files {
		files[f] = true
	}
	for _, t := range m.Nodes {
		if t.Kind != archdoc.Table || !files[t.Dir] {
			continue
		}
		cols := make([]string, 0, len(t.Columns))
		for _, c := range t.Columns {
			cols = append(cols, c.Name)
		}
		add(fmt.Sprintf("declares table %q with columns %s", t.Name, strings.Join(cols, ", ")), t.Prov)
	}
	return out
}

// legacyFingerprint is a component's fingerprint under the first fact wording — explain-1, routes
// without their summaries — so answers remembered then can still be found.
func legacyFingerprint(m archdoc.Model, id string) string {
	plain := m
	plain.Entries = append([]archdoc.Entry(nil), m.Entries...)
	for i := range plain.Entries {
		plain.Entries[i].Summary = ""
	}
	h := sha256.New()
	h.Write([]byte("explain-1"))
	for _, f := range ComponentFacts(plain, id) {
		h.Write([]byte("\x00" + f.ID + " " + f.Text))
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// Fingerprint identifies a set of facts, as the model would see them.
func Fingerprint(facts []Fact) string {
	h := sha256.New()
	h.Write([]byte(explainVersion))
	for _, f := range facts {
		h.Write([]byte("\x00" + f.ID + " " + f.Text))
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

const explainSystem = `You explain one part of a software system for its documentation.

You receive the facts archdoc read from the code about one component — a directory of the code
inside a container — as a numbered list. Write two to four sentences on what this component does
and how it relates to the rest: its responsibility, what it handles, what it depends on.

Every sentence must cite the facts it rests on, by their numbers, and may say only what those
facts support. Prefer the facts that say most — the routes it handles, the tables it declares, the
components it uses — over listing files. Do not repeat the facts as a list; explain them. Do not
guess at behaviour the facts do not show, and do not mention archdoc or the facts themselves.`

// ExplainSchema is the strict schema of an answer: sentences, each with the facts it cites.
func ExplainSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"sentences"},
		"properties": map[string]any{
			"sentences": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"text", "cites"},
					"properties": map[string]any{
						"text":  map[string]any{"type": "string"},
						"cites": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
				},
			},
		},
	}
}

type sentences struct {
	Sentences []struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	} `json:"sentences"`
}

// Explain gives each component its explanation: remembered when its facts are unchanged, asked for
// when complete is set, left without one otherwise. It returns the model with Explanations set and
// what it remembered, for the caller to keep.
func Explain(ctx context.Context, complete Completer, model string, m archdoc.Model, mem Memory) (archdoc.Model, Memory, ExplainReport, error) {
	rep := ExplainReport{Report: Report{Model: model}}
	next := Memory{}
	prov := archdoc.Provenance{Origin: archdoc.Semantic, Note: model}

	var ids []string
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Component {
			ids = append(ids, n.ID)
		}
	}
	sort.Strings(ids)
	m.Explanations = nil
	// The last answer about each element, whatever facts it was given.
	earlier := map[string]string{}
	for fp, r := range mem {
		if r.Element != "" {
			if old, ok := earlier[r.Element]; !ok || fp < old {
				earlier[r.Element] = fp
			}
		}
	}
	for _, id := range ids {
		facts := ComponentFacts(m, id)
		fp := Fingerprint(facts)
		stale := false
		var claims []archdoc.Claim
		if r, ok := mem[fp]; ok {
			claims = r.Claims
			rep.Remembered++
		} else if complete != nil {
			// A request that was paid for is never thrown away: one component's bad answer leaves
			// that component unexplained, and a failed request stops the asking, not the run.
			var answered bool
			var err error
			var why string
			claims, answered, why, err = ask(ctx, complete, model, facts, &rep)
			if err != nil {
				rep.Stopped = fmt.Sprintf("at %s: %v", id, err)
				complete = nil
			} else {
				rep.Asked++
				if !answered {
					rep.Unanswered = append(rep.Unanswered, id+" ("+why+")")
				}
			}
		}
		if len(claims) == 0 {
			// Nothing for these facts: the last answer about this element stands in, marked. Found
			// by the element it names, or — for a memory written before elements were recorded —
			// by the fingerprint its facts had under the first wording.
			old, ok := earlier[id]
			if !ok {
				old = legacyFingerprint(m, id)
			}
			if r, found := mem[old]; found && len(r.Claims) > 0 {
				claims, stale, fp = r.Claims, true, old
				rep.Stale++
			}
		}
		if len(claims) == 0 {
			continue
		}
		next[fp] = Remembered{Element: id, Claims: claims}
		m.Explanations = append(m.Explanations, archdoc.Explanation{Element: id, Claims: claims, Fingerprint: fp, Stale: stale, Prov: prov})
	}
	return m, next, rep, nil
}

// ask puts one component's facts to the model, and keeps the sentences whose citations resolve.
// answered is false when the reply was empty or not in the agreed shape.
func ask(ctx context.Context, complete Completer, model string, facts []Fact, rep *ExplainReport) (claims []archdoc.Claim, answered bool, why string, err error) {
	byID := map[string]Fact{}
	var list strings.Builder
	for _, f := range facts {
		byID[f.ID] = f
		fmt.Fprintf(&list, "%s: %s\n", f.ID, f.Text)
	}
	turns := []Turn{{Role: "user", Text: "Facts:\n\n" + list.String()}}

	var kept []archdoc.Claim
	for attempt := 1; attempt <= Attempts; attempt++ {
		rep.Attempts++
		for _, t := range turns {
			rep.BytesSent += len(t.Text)
		}
		rep.BytesSent += len(explainSystem)
		reply, err := complete(ctx, explainSystem, turns)
		if err != nil {
			return nil, false, "", err
		}
		rep.InputTokens += reply.InputTokens
		rep.OutputTokens += reply.OutputTokens
		if reply.Refused {
			return nil, false, "the model declined", nil
		}
		var a sentences
		if err := json.Unmarshal([]byte(strings.TrimSpace(reply.Text)), &a); err != nil {
			why := "an answer not in the agreed shape"
			if strings.TrimSpace(reply.Text) == "" {
				why = "an empty answer"
			}
			if reply.Stop != "" {
				why += ", stopped by " + reply.Stop
			}
			return nil, false, why, nil
		}
		kept, problems := checkClaims(a, byID)
		if len(problems) == 0 || attempt == Attempts {
			rep.Refused += len(problems)
			return kept, true, "", nil
		}
		turns = append(turns,
			Turn{Role: "assistant", Text: reply.Text},
			Turn{Role: "user", Text: "Some sentences were refused. Return a corrected, complete answer.\n\n" + strings.Join(problems, "\n")},
		)
	}
	return kept, true, "", nil
}

// checkClaims is F-36: a sentence must cite at least one fact, every fact it cites must be one it
// was given, and it must be a sentence, not an essay.
func checkClaims(a sentences, byID map[string]Fact) ([]archdoc.Claim, []string) {
	var kept []archdoc.Claim
	var problems []string
	for i, s := range a.Sentences {
		text := strings.TrimSpace(s.Text)
		switch {
		case i >= maxSentences:
			problems = append(problems, fmt.Sprintf("sentence %d: at most %d sentences", i+1, maxSentences))
			continue
		case text == "":
			continue
		case len(text) > maxSentence:
			problems = append(problems, fmt.Sprintf("sentence %d: longer than %d characters", i+1, maxSentence))
			continue
		case len(s.Cites) == 0:
			problems = append(problems, fmt.Sprintf("sentence %d cites no fact: %q", i+1, text))
			continue
		}
		c := archdoc.Claim{Text: text}
		bad := ""
		seen := map[string]bool{}
		for _, id := range s.Cites {
			f, ok := byID[strings.TrimSpace(id)]
			if !ok {
				bad = id
				break
			}
			if seen[f.ID] {
				continue
			}
			seen[f.ID] = true
			c.Facts = append(c.Facts, f.Text)
			c.Cites = append(c.Cites, f.Prov)
		}
		if bad != "" {
			problems = append(problems, fmt.Sprintf("sentence %d cites %q, which is not one of the facts given", i+1, bad))
			continue
		}
		kept = append(kept, c)
	}
	return kept, problems
}
