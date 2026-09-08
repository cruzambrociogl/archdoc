// Package validate is the gate every change to the model passes through.
//
// Its rules are not invented. Each one traces to a way the draw.io experiment in §0 failed: an
// edge pointing at a component that was never placed, an element nobody could account for, a
// picture that looked finished and was not. A drawing tool has no model, so it cannot notice any
// of that. This package is what noticing looks like.
package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Severity separates two questions that look alike and are not: *is this model wrong*, and *is
// this model thin*.
//
// A wrong model must never reach disk — an edge to a node that does not exist is a lie, and the
// reader has no way to detect it. A thin model is published, with its gaps reported. Most of
// R1.a's output is thin by construction: configuration does not state what a service is *for*,
// which is the whole reason the semantic layer exists.
//
// Conflating the two would mean refusing to write a correct diagram because it lacked prose,
// which is the opposite of the honest-partial-output principle in §8.
type Severity string

const (
	// Error means the model contradicts itself or claims something it cannot support.
	Error Severity = "error"
	// Warning means the model is incomplete but true.
	Warning Severity = "warning"
)

// Finding is one rule, one element, one problem.
type Finding struct {
	Rule     string // the capability ID, so a reader can look up what it means
	Severity Severity
	Element  string
	Message  string
	Prov     archdoc.Provenance
}

func (f Finding) String() string {
	where := ""
	if f.Prov.Known() {
		where = " (" + f.Prov.String() + ")"
	}
	return fmt.Sprintf("%s %s: %s — %s%s", f.Severity, f.Rule, f.Element, f.Message, where)
}

// Result is everything validation found, in a stable order.
type Result struct {
	Findings []Finding
}

// OK reports whether the model may be written. Warnings do not block.
func (r Result) OK() bool { return len(r.Errors()) == 0 }

func (r Result) Errors() []Finding   { return r.bySeverity(Error) }
func (r Result) Warnings() []Finding { return r.bySeverity(Warning) }

func (r Result) bySeverity(s Severity) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Severity == s {
			out = append(out, f)
		}
	}
	return out
}

// Error renders the failures the way VAL-08 requires: loudly, naming every offending element,
// rather than a count.
func (r Result) Error() string {
	errs := r.Errors()
	if len(errs) == 0 {
		return ""
	}

	lines := make([]string, 0, len(errs)+1)
	lines = append(lines, fmt.Sprintf("%d validation error(s):", len(errs)))
	for _, f := range errs {
		lines = append(lines, "  "+f.String())
	}
	return strings.Join(lines, "\n")
}

// Add records a finding. Exported because validation is not the only thing that finds problems
// — rule compilation reports unreachable and conflicting rules, and there is no reason for a
// reader to meet two different vocabularies for "here is what is wrong".
func (r *Result) Add(rule string, sev Severity, element, msg string, prov archdoc.Provenance) {
	r.Findings = append(r.Findings, Finding{rule, sev, element, msg, prov})
}

func (r *Result) add(rule string, sev Severity, element, msg string, prov archdoc.Provenance) {
	r.Add(rule, sev, element, msg, prov)
}

// Sort orders findings deterministically. Exported for callers that assemble a Result from more
// than one source.
func (r *Result) Sort() { r.sort() }

// sort gives findings a deterministic order. Validation output reaches the run log and the
// report, so AC-7 applies to it too.
func (r *Result) sort() {
	sort.SliceStable(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if a.Severity != b.Severity {
			return a.Severity == Error // errors first: they are what stops the run
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Element < b.Element
	})
}

var kinds = map[archdoc.Kind]bool{
	archdoc.Application: true, archdoc.Datastore: true, archdoc.Queue: true,
	archdoc.Proxy: true, archdoc.External: true, archdoc.System: true, archdoc.Actor: true,
}

// Model checks a whole model against every rule.
func Model(m archdoc.Model) Result {
	var r Result

	seen := make(map[string]bool, len(m.Nodes))

	for _, n := range m.Nodes {
		// VAL-01 — schema conformance.
		switch {
		case n.ID == "":
			r.add("VAL-01", Error, "(unnamed)", "node has no id", n.Prov)
		case n.Name == "":
			r.add("VAL-01", Error, n.ID, "node has no name", n.Prov)
		}
		if !kinds[n.Kind] {
			r.add("VAL-01", Error, n.ID, fmt.Sprintf("unknown kind %q", n.Kind), n.Prov)
		}

		// MDL-13 depends on identity being unique; a duplicate silently merges two things.
		if seen[n.ID] {
			r.add("VAL-01", Error, n.ID, "duplicate node id", n.Prov)
		}
		seen[n.ID] = true

		// VAL-04 — every node sits on exactly one side of the system boundary. In R1.a the
		// evidence kind *is* that boundary, so an unset one leaves the node nowhere.
		if n.Evidence != archdoc.Declared && n.Evidence != archdoc.Referenced {
			r.add("VAL-04", Error, n.ID, fmt.Sprintf("unknown evidence kind %q", n.Evidence), n.Prov)
		}

		// VAL-05 — P1 in executable form. An element nothing can vouch for must not be drawn.
		if !n.Prov.Known() {
			r.add("VAL-05", Error, n.ID, "no traceable provenance", n.Prov)
		}
		// A label without its own citation would be credited to the line that declares the
		// node, which never said it.
		if n.Technology != "" && !n.TechProv.Known() {
			r.add("VAL-05", Error, n.ID, "technology has no provenance", n.Prov)
		}
		if n.Description != "" && !n.DescProv.Known() {
			r.add("VAL-05", Error, n.ID, "description has no provenance", n.Prov)
		}

		// VAL-06 — completeness, not correctness. Configuration never states what a service
		// is *for*; the semantic layer is what fills these, and a model without it is thin
		// rather than wrong.
		if n.Description == "" && n.Kind != archdoc.Actor {
			r.add("VAL-06", Warning, n.ID, "no description", n.Prov)
		}
		if n.Technology == "" && n.Kind.Container() {
			r.add("VAL-06", Warning, n.ID, "no technology", n.Prov)
		}
	}

	for _, e := range m.Edges {
		id := e.From + " → " + e.To
		first := archdoc.Provenance{}
		if len(e.Prov) > 0 {
			first = e.Prov[0]
		}

		// VAL-02 — referential integrity. This is the draw.io failure exactly: an edge whose
		// endpoint was never placed. A drawing tool cannot notice; a model must.
		if !seen[e.From] {
			r.add("VAL-02", Error, id, fmt.Sprintf("edge from undefined node %q", e.From), first)
		}
		if !seen[e.To] {
			r.add("VAL-02", Error, id, fmt.Sprintf("edge to undefined node %q", e.To), first)
		}
		if e.From == e.To {
			r.add("VAL-02", Error, id, "edge points at itself", first)
		}

		// VAL-05 — an edge needs at least one entry that came from reading a file. A
		// relationship attested only by the model is the model inventing a fact.
		if !hasExtraction(e.Prov) {
			r.add("VAL-05", Error, id, "no extraction-tier provenance", first)
		}

		// VAL-03 — completeness. Most edges have no protocol because configuration did not
		// state one; that is a gap to report, not a reason to refuse the diagram.
		if e.Technology == "" {
			r.add("VAL-03", Warning, id, "no protocol", first)
		}
	}

	r.sort()
	return r
}

// hasExtraction reports whether any citation came from reading a file. Rules count: a person
// wrote them in a file with a line, and is accountable for them. The model does not.
func hasExtraction(ps []archdoc.Provenance) bool {
	for _, p := range ps {
		if !p.Known() {
			continue
		}
		if p.Origin == "" || p.Origin == archdoc.Extraction || p.Origin == archdoc.Rules {
			return true
		}
	}
	return false
}
