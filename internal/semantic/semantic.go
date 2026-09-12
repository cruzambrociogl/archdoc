// Package semantic is the only part of archdoc that talks to a language model, and the only
// package permitted to make outbound network calls. CI enforces the second; this file is how the
// first stays true.
//
// The model does one job: it names, describes, and labels what extraction already found. It
// never adds, removes, or excludes an element. Three things hold that line, and none of them is
// a prompt instruction a model could ignore:
//
//  1. It is sent structure only — names, kinds, relationships. No file contents (AC-8).
//  2. It answers in a strict JSON schema whose only verbs are labelling verbs.
//  3. Every operation it returns passes through the same validator as a rules.yaml correction,
//     and OpKind.AllowedFrom rejects anything structural before it can apply.
//
// Switch the model off and the diagram keeps every box and every arrow. That is AC-2, and it is
// the claim Review 2 is built around.
package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// Model is the default. A caller may override it; nothing else in archdoc depends on which model
// answered, because nothing else trusts the answer beyond what the validator accepts.
const Model = "claude-opus-5"

// Attempts is VAL-07's fixed retry budget: validation errors go back to the model this many times
// at most, then the run fails loudly with the offending operations (VAL-08).
const Attempts = 3

// Turn is one message in the exchange with the model.
type Turn struct {
	Role string // "user" or "assistant"
	Text string
}

// Reply is what came back.
type Reply struct {
	Text    string
	Refused bool // the model's safety classifiers declined; Text is not an answer

	InputTokens  int64
	OutputTokens int64
}

// Completer sends one request. The real one calls the Anthropic API (claude.go); tests pass a
// function that returns canned JSON, so the whole package is exercised without a network.
type Completer func(ctx context.Context, system string, turns []Turn) (Reply, error)

// Report says what a labelling run did, for the run log AC-8 asks for.
type Report struct {
	Model     string
	Attempts  int
	BytesSent int // every byte of every prompt, so egress is accounted for rather than assumed
	Ops       int

	// Tokens across every attempt, including the ones the validator sent back. A retry is
	// billed like any other request, so it is counted like one.
	InputTokens  int64
	OutputTokens int64
}

// Label asks the model for names, descriptions and edge labels, and returns the model with the
// accepted operations applied.
//
// On success every changed value carries Origin: Semantic, which is what PRV-05 renders as
// interpretation. On failure the original model comes back untouched — never a partial one.
func Label(ctx context.Context, complete Completer, model string, m archdoc.Model) (archdoc.Model, Report, error) {
	rep := Report{Model: model}

	prompt, err := describe(m)
	if err != nil {
		return m, rep, err
	}
	turns := []Turn{{Role: "user", Text: prompt}}

	var last validate.Result
	for attempt := 1; attempt <= Attempts; attempt++ {
		rep.Attempts = attempt
		for _, t := range turns {
			rep.BytesSent += len(t.Text)
		}
		rep.BytesSent += len(system)

		reply, err := complete(ctx, system, turns)
		if err != nil {
			return m, rep, err
		}
		rep.InputTokens += reply.InputTokens
		rep.OutputTokens += reply.OutputTokens
		if reply.Refused {
			return m, rep, fmt.Errorf("the model declined to label this architecture; the diagram is unchanged")
		}

		ops, err := decode(reply.Text, model)
		if err != nil {
			// Structured outputs should make this impossible. If it happens anyway, it is a
			// fault to report, not to guess around.
			return m, rep, fmt.Errorf("the model's answer was not the agreed shape: %w", err)
		}

		out, res := validate.Apply(m, ops)
		if res.OK() {
			rep.Ops = len(ops)
			return out, rep, nil
		}

		// VAL-07 — hand the errors back and let the model correct itself, within budget.
		last = res
		turns = append(turns,
			Turn{Role: "assistant", Text: reply.Text},
			Turn{Role: "user", Text: "Some operations were rejected. Return a corrected, complete list.\n\n" + res.Error()},
		)
	}

	// VAL-08 — out of budget. Fail loudly with what was wrong; write nothing.
	return m, rep, fmt.Errorf("the model's operations failed validation %d times, nothing applied\n%s",
		Attempts, last.Error())
}

// system is the standing instruction. Short on purpose: it states the job and the limits, and
// the schema and the validator enforce them regardless.
const system = `You label software architecture models for documentation.

You receive a model extracted from a repository's configuration: containers, external systems,
and relationships between them. Everything in it is a verified fact. Your job is to make it
readable — never to change what it says exists.

Return operations:
- set_description: one sentence on what an element is responsible for. Every container and
  external system should get one.
- set_technology: only for elements whose technology is empty, and only when the name makes the
  technology evident. Leave it out when unsure.
- set_edge_label: a short verb phrase for a relationship ("reads and writes user data"), replacing
  a generic label like "connects to" or "depends on". Describe what that particular target does
  for the source; relationships to different targets should read differently. Do not name a
  protocol, transport or security layer (HTTPS, gRPC, WebSocket, TLS) in a label: protocol is a
  fact the model records separately, and a label may only interpret, never assert.
- set_name: only when a clearer display name is obvious. Usually leave names alone.

Use the element ids exactly as given. For set_edge_label, "target" is the source id and "to" is the
destination id; for every other operation, leave "to" empty. You cannot add, remove, or reclassify
elements, and you cannot invent relationships.`

// payload is what the model sees: structure, never source. No provenance, no file paths, no
// values from any environment — AC-8's structure-only mode is the only mode there is.
type payload struct {
	System        string         `json:"system"`
	Elements      []element      `json:"elements"`
	Relationships []relationship `json:"relationships"`
}

type element struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Technology  string   `json:"technology,omitempty"`
	Description string   `json:"description,omitempty"`
	Networks    []string `json:"networks,omitempty"`
}

type relationship struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label,omitempty"`
	Technology string `json:"technology,omitempty"`
}

func describe(m archdoc.Model) (string, error) {
	p := payload{System: m.Name}

	for _, n := range m.Nodes {
		p.Elements = append(p.Elements, element{
			ID: n.ID, Name: n.Name, Kind: string(n.Kind),
			Technology: n.Technology, Description: n.Description, Networks: n.Networks,
		})
	}
	for _, e := range m.Edges {
		p.Relationships = append(p.Relationships, relationship{
			From: e.From, To: e.To, Label: e.Label, Technology: e.Technology,
		})
	}

	// Stable order keeps the prompt byte-identical for an unchanged model, which is what makes
	// it cacheable and what makes two runs comparable.
	sort.Slice(p.Elements, func(i, j int) bool { return p.Elements[i].ID < p.Elements[j].ID })
	sort.Slice(p.Relationships, func(i, j int) bool {
		if p.Relationships[i].From != p.Relationships[j].From {
			return p.Relationships[i].From < p.Relationships[j].From
		}
		return p.Relationships[i].To < p.Relationships[j].To
	})

	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return "Label this architecture.\n\n" + string(b), nil
}

// allowed is the whole vocabulary the model may use. The schema enumerates it, so a structural
// operation cannot even be expressed — and AllowedFrom would reject it if it somehow were.
var allowed = []archdoc.OpKind{
	archdoc.SetName, archdoc.SetDescription, archdoc.SetTechnology, archdoc.SetEdgeLabel,
}

// Schema is the strict JSON schema the answer must match.
func Schema() map[string]any {
	kinds := make([]string, 0, len(allowed))
	for _, k := range allowed {
		kinds = append(kinds, string(k))
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"operations"},
		"properties": map[string]any{
			"operations": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"kind", "target", "to", "value"},
					"properties": map[string]any{
						"kind":   map[string]any{"type": "string", "enum": kinds},
						"target": map[string]any{"type": "string"},
						"to":     map[string]any{"type": "string"},
						"value":  map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

type answer struct {
	Operations []struct {
		Kind   string `json:"kind"`
		Target string `json:"target"`
		To     string `json:"to"`
		Value  string `json:"value"`
	} `json:"operations"`
}

func decode(text, model string) ([]archdoc.Op, error) {
	var a answer
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &a); err != nil {
		return nil, err
	}

	// Every value the model supplies cites the model. The note names which one, so a reader
	// knows what produced it; it deliberately omits a per-request id, which would make every
	// labelled run a new version in history even when nothing about the architecture moved.
	prov := archdoc.Provenance{Origin: archdoc.Semantic, Note: model}

	ops := make([]archdoc.Op, 0, len(a.Operations))
	for _, o := range a.Operations {
		ops = append(ops, archdoc.Op{
			Kind: archdoc.OpKind(o.Kind), Target: o.Target, To: o.To, Value: o.Value,
			Origin: archdoc.Semantic, Prov: prov,
		})
	}
	return ops, nil
}
