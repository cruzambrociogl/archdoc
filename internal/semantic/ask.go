package semantic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Ask (F-26, platform-surface §6.5): a question in someone's own words, answered only from what
// archdoc read — the validated model — every sentence citing the facts it rests on.
//
// The facts are chosen here, without a model: the parts, routes, tables and calls whose names share
// words with the question, best first, and the calls around the best. That keeps a request small —
// what costs is what is sent — and keeps the answer to what was read. A question nothing matches is
// answered without asking: archdoc found nothing it can cite, which is a designed answer, and free.
//
// What leaves the machine is what --explain sends — names, paths, routes with their summaries,
// tables and their columns — and the question, which is the person's own words (StructureAndQuestion).
// An answer is remembered against the question and its facts, so asking again costs nothing.

// askVersion is part of every answer's fingerprint: changing the instructions asks again.
const askVersion = "ask-1"

// StructureAndQuestion is what Ask sends: what --explain sends, and the question as typed.
const StructureAndQuestion = "structure-and-your-question"

// AskMaxTokens bounds one answer, reasoning included: a few sentences fit many times over.
const AskMaxTokens = 2000

// AskEffort is how hard the model works on an answer: composing a few cited sentences from facts
// it is handed needs little reasoning, and reasoning is billed as output.
const AskEffort = "low"

const maxAskFacts = 40

// maxAskSteps is how many of those may be steps of a flow: a route's steps are many and alike, and
// would otherwise crowd out the routes and parts that say most.
const maxAskSteps = 15

// Answer is a question answered: its sentences, each with the facts it cites and where each is
// proven, and what the facts did not say.
type Answer struct {
	Question string       `json:"question"`
	Claims   []core.Claim `json:"claims"`
	// NotFound is what the facts did not settle, in the model's words — "couldn't be resolved".
	NotFound string `json:"not_found,omitempty"`
	Facts    int    `json:"facts"`
	Model    string `json:"model,omitempty"`
}

// stop are words a question asks with, not about.
var stopWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`a an and are as at be by can could do does done for from get gets got has have how
		i if in into is it its me my of on or our so than that the their them then there these they this those to
		up us was we were what when where which who whom whose why will with would you your about after all any
		archdoc code does each happen happens other some used uses using work works`) {
		stopWords[w] = true
	}
}

// words splits text into the lowercase words it is made of: camelCase and snake_case apart, each
// stemmed, the words a question asks with left out.
func words(text string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) == 0 {
			return
		}
		w := strings.ToLower(string(cur))
		cur = cur[:0]
		if !stopWords[w] && len(w) >= 3 {
			out = append(out, stem(w))
		}
	}
	prev := rune(0)
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if unicode.IsUpper(r) && unicode.IsLower(prev) {
				flush()
			}
			cur = append(cur, r)
		default:
			flush()
		}
		prev = r
	}
	flush()
	return out
}

// stem takes a word to what its forms share, lightly — detected and detection to detect, photos to
// photo — keeping at least four letters, so short names stay as they are.
func stem(w string) string {
	if strings.HasSuffix(w, "ss") {
		return w
	}
	for _, suffix := range []string{"ations", "ation", "ions", "ion", "ings", "ing", "ed", "s"} {
		if strings.HasSuffix(w, suffix) && len(w)-len(suffix) >= 4 {
			return strings.TrimSuffix(w, suffix)
		}
	}
	return w
}

// candidate is a fact a question may be about, with what it can be matched on.
type candidate struct {
	text  string
	prov  core.Provenance
	names []string // its own names: a match here says most
	more  []string // what else it says: description, technology, columns, a route's summary
	nodes []string // the elements it involves, to follow calls from the best
	step  bool     // a step of a followed flow: many are alike, so fewer are taken
	order int      // the model's own order, for ties
}

// QuestionFacts picks the facts a question is about, best first, numbered for the model to cite:
// those that share the most words with it — in their names above all — then the calls to and from
// the best of them. None: nothing in the model shares a word with the question.
func QuestionFacts(m core.Model, question string) []Fact {
	want := map[string]bool{}
	for _, w := range words(question) {
		want[w] = true
	}
	if len(want) == 0 {
		return nil
	}
	cands := candidates(m)
	type scored struct {
		c     candidate
		score int
	}
	var hits []scored
	for _, c := range cands {
		// Each word of the question counts once, where it matches best: a fact that meets more of
		// the question ranks above one that repeats a single word — a path of /photos/[[photos]].
		s := 0
		met := map[string]int{}
		for _, w := range c.names {
			if want[w] {
				met[w] = 3
			}
		}
		for _, w := range c.more {
			if want[w] && met[w] == 0 {
				met[w] = 1
			}
		}
		for _, v := range met {
			s += v
		}
		if s > 0 {
			hits = append(hits, scored{c, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].c.order < hits[j].c.order
	})
	// Not one word of the question names anything here: a word met only in a description or a
	// summary is too little to pay for an answer about.
	if len(hits) == 0 || hits[0].score < 3 {
		return nil
	}

	chosen := map[int]bool{}
	var picked []candidate
	steps := 0
	take := func(c candidate) {
		if len(picked) == maxAskFacts || chosen[c.order] || c.step && steps == maxAskSteps {
			return
		}
		if c.step {
			steps++
		}
		chosen[c.order] = true
		picked = append(picked, c)
	}
	for _, h := range hits {
		take(h.c)
	}
	// The calls around the best few: where something happens is often one call away from it.
	best := map[string]bool{}
	for _, h := range hits[:min(5, len(hits))] {
		for _, n := range h.c.nodes {
			best[n] = true
		}
	}
	for _, c := range cands {
		if len(c.nodes) == 2 && (best[c.nodes[0]] || best[c.nodes[1]]) {
			take(c)
		}
	}

	out := make([]Fact, len(picked))
	for i, c := range picked {
		out[i] = Fact{ID: fmt.Sprintf("F%d", i+1), Text: c.text, Prov: c.prov}
	}
	return out
}

// candidates is every fact a question could be about: each element, each call between two, each
// route and page, each step a followed route takes — in the model's order.
func candidates(m core.Model) []candidate {
	var out []candidate
	add := func(c candidate) {
		c.order = len(out)
		out = append(out, c)
	}
	name := func(id string) string {
		if n, ok := m.Node(id); ok {
			return n.Name
		}
		return id
	}
	for _, n := range m.Nodes {
		var text string
		more := append(words(n.Description), words(n.Technology)...)
		switch n.Kind {
		case core.Component:
			text = fmt.Sprintf("component %q inside %q, at %s", n.Name, name(n.Parent), n.Dir)
			more = append(more, words(n.Dir)...)
		case core.Table:
			cols := make([]string, 0, len(n.Columns))
			for _, c := range n.Columns {
				cols = append(cols, c.Name)
				more = append(more, words(c.Name)...)
			}
			text = fmt.Sprintf("table %q with columns %s", n.Name, strings.Join(cols, ", "))
		default:
			text = fmt.Sprintf("%s %q", strings.ReplaceAll(string(n.Kind), "_", " "), n.Name)
			if n.Technology != "" {
				text += " (" + n.Technology + ")"
			}
		}
		// A description helps find the element here; it is not sent, as --explain sends none.
		add(candidate{text: text, prov: n.Prov, names: words(n.Name), more: more, nodes: []string{n.ID}})
	}
	for _, e := range m.Edges {
		if len(e.Prov) == 0 {
			continue
		}
		verb := "uses"
		if e.Label != "" {
			verb = e.Label
		}
		add(candidate{text: fmt.Sprintf("%q %s %q", name(e.From), verb, name(e.To)), prov: e.Prov[0],
			more: append(words(e.Label), words(e.Technology)...), nodes: []string{e.From, e.To}})
	}
	for _, e := range m.Entries {
		var text string
		if e.Kind == "page" {
			text = fmt.Sprintf("the page at %s, in %q", e.Path, name(e.Container))
		} else {
			text = fmt.Sprintf("%s %s is handled by %s, in %q", e.Method, e.Path, e.Handler, name(e.Component))
		}
		if e.Summary != "" {
			text += fmt.Sprintf(", described by the code as %q", e.Summary)
		}
		names := append(words(e.Path), words(e.Handler)...)
		add(candidate{text: text, prov: e.Prov, names: names, more: words(e.Summary), nodes: []string{e.Component}})
	}
	for _, f := range m.Flows {
		parts := map[string]string{}
		for _, p := range f.Participants {
			parts[p.ID] = p.Name
		}
		for _, s := range f.Steps {
			add(candidate{text: fmt.Sprintf("in the flow of %s: %s %s %s", f.Entry, parts[s.From], s.Call, parts[s.To]),
				prov: s.Prov, names: append(words(parts[s.To]), words(s.Call)...), more: words(parts[s.From]), step: true})
		}
	}
	return out
}

// AnswerKey is what an answer is remembered by: the question, as asked, and the facts it was given.
func AnswerKey(question string, facts []Fact) string {
	h := sha256.New()
	h.Write([]byte(askVersion + "\x00" + strings.ToLower(strings.Join(strings.Fields(question), " "))))
	for _, f := range facts {
		h.Write([]byte("\x00" + f.ID + " " + f.Text))
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// AskPrompt is the one message a question sends: the facts, numbered, then the question.
func AskPrompt(question string, facts []Fact) string {
	var b strings.Builder
	b.WriteString("Facts archdoc read from the code:\n")
	for _, f := range facts {
		fmt.Fprintf(&b, "%s: %s\n", f.ID, f.Text)
	}
	fmt.Fprintf(&b, "\nQuestion: %s\n", strings.TrimSpace(question))
	return b.String()
}

// AskSystem is what the model is told an answer must be.
const AskSystem = `You answer a question about one software system, for its documentation.

You receive facts archdoc read from the system's code, numbered F1, F2, …, and a question. Answer in
one to four short sentences, each citing the facts it rests on by their numbers. Say only what the
facts support; when they do not settle something the question asks, say what is missing in
not_found, in one sentence, and leave it out of the answer. If the facts do not answer the question
at all, give no sentences and say so in not_found. Do not mention archdoc or the facts themselves.`

// AskSchema is the strict shape of an answer: sentences with the facts they cite, and what the
// facts did not settle.
func AskSchema() map[string]any {
	s := ExplainSchema()
	props := s["properties"].(map[string]any)
	props["not_found"] = map[string]any{"type": "string"}
	s["required"] = []string{"sentences", "not_found"}
	return s
}

type answered struct {
	Sentences []struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	} `json:"sentences"`
	NotFound string `json:"not_found"`
}

// AskQuestion puts a question and its facts to the model, and keeps the sentences whose citations
// resolve (F-36). A reply that cites a fact it was not given is sent back to be corrected, within
// the retry budget; what is still refused is left out, never shown unchecked.
func AskQuestion(ctx context.Context, complete Completer, model, question string, facts []Fact) (Answer, Report, error) {
	rep := Report{Model: model}
	out := Answer{Question: question, Facts: len(facts), Model: model}
	byID := map[string]Fact{}
	for _, f := range facts {
		byID[f.ID] = f
	}
	turns := []Turn{{Role: "user", Text: AskPrompt(question, facts)}}
	for attempt := 1; attempt <= Attempts; attempt++ {
		rep.Attempts++
		rep.BytesSent += len(AskSystem)
		for _, t := range turns {
			rep.BytesSent += len(t.Text)
		}
		reply, err := complete(ctx, AskSystem, turns)
		if err != nil {
			return out, rep, err
		}
		rep.InputTokens += reply.InputTokens
		rep.OutputTokens += reply.OutputTokens
		if reply.Refused {
			return out, rep, fmt.Errorf("the model declined to answer")
		}
		var a answered
		if err := json.Unmarshal([]byte(strings.TrimSpace(reply.Text)), &a); err != nil {
			return out, rep, fmt.Errorf("the model's answer was not in the agreed shape (stopped by %s)", reply.Stop)
		}
		kept, problems := checkClaims(sentences{Sentences: a.Sentences}, byID)
		out.Claims, out.NotFound = kept, strings.TrimSpace(a.NotFound)
		if len(problems) == 0 || attempt == Attempts {
			return out, rep, nil
		}
		turns = append(turns,
			Turn{Role: "assistant", Text: reply.Text},
			Turn{Role: "user", Text: "Some sentences were refused. Return a corrected, complete answer.\n\n" + strings.Join(problems, "\n")},
		)
	}
	return out, rep, nil
}
