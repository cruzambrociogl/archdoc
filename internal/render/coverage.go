package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// The coverage report: what archdoc read, what it could not resolve, and what the evidence it had
// cannot state at all.
//
// Every tool in this space produces documentation that looks complete. None of them says what it
// missed. A reader who cannot see the edge of the map has no way to know they are standing at it —
// which is the failure the draw.io experiment produced and the whole provenance design exists to
// prevent. Saying "eight relationships carry no protocol, and here they are" costs nothing and is
// the difference between a document a reader can rely on and one they must verify by hand.
//
// Nothing here is interpretation: every line is a count of facts, a validator finding, or a stated
// limitation of reading configuration.

// CoverageFile sits beside the arc42 sections. It is generated, hence the suffix (hard rule 2).
const CoverageFile = "coverage.generated.md"

// Gap is one thing the configuration does not state, as the validator reported it. Mirrored here
// rather than imported so that rendering does not depend on validation.
type Gap struct {
	Rule    string `json:"rule"` // VAL-03, VAL-06 …
	Element string `json:"element"`
	Message string `json:"message"`
}

// CoverageReport is the report as data: computed once by generate, rendered as the committed
// Markdown, and written as .archdoc/coverage.json for the app and the published site — so the page
// and the document cannot disagree. Every slice is sorted (AC-7).
type CoverageReport struct {
	Source string     `json:"source"`
	Read   []ReadFile `json:"read"`
	Known  []Count    `json:"known"`
	// Complete counts the elements and relationships no validator finding names, out of Items.
	Complete int        `json:"complete"`
	Items    int        `json:"items"`
	Gaps     []GapGroup `json:"gaps"`
	// Unconnected are elements nothing reaches and that reach nothing.
	Unconnected []string `json:"unconnected"`
	Limits      []Limit  `json:"limits"`
}

// ReadFile is one file discovery considered, used or passed over, and why.
type ReadFile struct {
	File string `json:"file"`
	Used bool   `json:"used"`
	Why  string `json:"why"`
}

// Count is one line of "how much is known".
type Count struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// GapGroup is the validator's findings under one rule.
type GapGroup struct {
	Rule string `json:"rule"`
	Gaps []Gap  `json:"gaps"`
}

// Limit is something reading configuration cannot know, and what would be needed to know it.
type Limit struct {
	Limit  string `json:"limit"`
	Needed string `json:"needed"`
}

// CoverageJSONFile is where generate writes the report as data, beside model.json.
const CoverageJSONFile = "coverage.json"

// BuildCoverage computes the report.
func BuildCoverage(m archdoc.Model, facts archdoc.FactSet, gaps []Gap) CoverageReport {
	r := CoverageReport{Source: facts.Source, Read: sourcesRead(facts), Known: knownCounts(m, gaps)}

	byRule := map[string][]Gap{}
	named := map[string]bool{}
	for _, g := range gaps {
		byRule[g.Rule] = append(byRule[g.Rule], g)
		named[g.Element] = true
	}
	rules := make([]string, 0, len(byRule))
	for rule := range byRule {
		rules = append(rules, rule)
	}
	sort.Strings(rules) // AC-7: map iteration is randomised
	r.Gaps = []GapGroup{}
	for _, rule := range rules {
		list := byRule[rule]
		sort.Slice(list, func(i, j int) bool {
			if list[i].Element != list[j].Element {
				return list[i].Element < list[j].Element
			}
			return list[i].Message < list[j].Message
		})
		r.Gaps = append(r.Gaps, GapGroup{Rule: rule, Gaps: list})
	}

	r.Items = len(m.Nodes) + len(m.Edges)
	for _, n := range m.Nodes {
		if !named[n.ID] {
			r.Complete++
		}
	}
	for _, e := range m.Edges {
		if !named[e.From+" → "+e.To] {
			r.Complete++
		}
	}

	r.Unconnected = unconnected(m.Container())
	if r.Unconnected == nil {
		r.Unconnected = []string{}
	}
	r.Limits = limits
	return r
}

// Coverage renders the report.
func Coverage(m archdoc.Model, facts archdoc.FactSet, gaps []Gap, meta Meta) string {
	r := BuildCoverage(m, facts, gaps)
	var b strings.Builder

	b.WriteString("# Coverage\n\n")
	b.WriteString(meta.stamp())
	b.WriteString("What archdoc read, what it could not resolve, and what configuration cannot state.\n")
	b.WriteString("Published so the edge of the map is visible: everything in the other sections is\n")
	b.WriteString("proven, and this page says where the proof runs out.\n\n")

	b.WriteString("## What was read\n\n")
	b.WriteString("| File | Used | Why |\n|---|---|---|\n")
	for _, f := range r.Read {
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", f.File, yesNo(f.Used), f.Why)
	}

	b.WriteString("\n## How much is known\n\n")
	b.WriteString("| | Count |\n|---|---|\n")
	for _, c := range r.Known {
		fmt.Fprintf(&b, "| %s | %s |\n", c.Label, c.Value)
	}

	b.WriteString("\n## What could not be resolved\n\n")
	if len(r.Gaps) == 0 {
		b.WriteString("Nothing. Every element and relationship the configuration states is complete.\n")
	}
	for _, g := range r.Gaps {
		fmt.Fprintf(&b, "**%s** — %d\n\n", g.Rule, len(g.Gaps))
		for _, gap := range g.Gaps {
			fmt.Fprintf(&b, "- `%s` — %s\n", gap.Element, gap.Message)
		}
		b.WriteString("\n")
	}
	if len(r.Unconnected) > 0 {
		fmt.Fprintf(&b, "\n**Elements nothing reaches and that reach nothing** — %s.\n",
			strings.Join(r.Unconnected, ", "))
		b.WriteString("This is rarely true of a running system. It usually means the connection is made\n")
		b.WriteString("in application code or at deploy time, where configuration cannot see it. The\n")
		b.WriteString("element is drawn unconnected rather than joined up on a guess.\n")
	}

	b.WriteString("\n## What configuration cannot state\n\n")
	b.WriteString("These are not gaps in this repository. They are the limits of the evidence:\n")
	b.WriteString("configuration declares *what exists and what may reach what*, and nothing else.\n\n")
	b.WriteString("| Not knowable from configuration | What would be needed |\n|---|---|\n")
	for _, l := range r.Limits {
		fmt.Fprintf(&b, "| %s | %s |\n", l.Limit, l.Needed)
	}

	return b.String()
}

// sourcesRead lists every file discovery looked at — including the ones it skipped, and why. A
// reader can then tell "archdoc did not look" from "archdoc looked and found nothing".
func sourcesRead(facts archdoc.FactSet) []ReadFile {
	out := []ReadFile{}
	seen := map[string]bool{}
	for _, c := range sortedCandidates(facts.Considered) {
		seen[c.File] = true
		// Discovery's reason says what it found, not what it decided. A candidate it passed
		// over was often perfectly readable — another file simply won — and saying so is the
		// difference between "archdoc could not read this" and "archdoc chose the other one".
		reason := c.Reason
		if !c.Chosen {
			reason = "not selected — " + reason
		}
		out = append(out, ReadFile{File: c.File, Used: c.Chosen, Why: reason})
	}
	if !seen[facts.Source] && facts.Source != "" {
		out = append(out, ReadFile{File: facts.Source, Used: true, Why: "the configuration the model was built from"})
	}
	for _, f := range configFiles(facts) {
		out = append(out, ReadFile{File: f, Used: true, Why: "routing rules read from a mounted gateway configuration"})
	}
	for _, f := range envFiles(facts) {
		out = append(out, ReadFile{File: f, Used: true, Why: "network locations read from environment values"})
	}
	if in := facts.Interpolation; in != nil {
		why := "the values Compose's ${VARIABLES} were filled from"
		if in.Sample {
			why = "a sample — Compose's ${VARIABLES} were filled from its defaults, which a real deployment may override"
		}
		out = append(out, ReadFile{File: in.File, Used: true, Why: why})
	}
	return out
}

// knownCounts is the one place the documents say how much of themselves is backed by what.
func knownCounts(m archdoc.Model, gaps []Gap) []Count {
	declared, referenced := 0, 0
	for _, n := range m.Nodes {
		if n.Evidence == archdoc.Referenced {
			referenced++
			continue
		}
		declared++
	}

	withProtocol := 0
	for _, e := range m.Edges {
		if e.Technology != "" {
			withProtocol++
		}
	}

	described, interpreted := 0, 0
	for _, n := range m.Nodes {
		if n.Description != "" {
			described++
		}
		if n.DescProv.Origin.Interpretation() || n.NameProv.Origin.Interpretation() ||
			n.TechProv.Origin.Interpretation() {
			interpreted++
		}
	}

	return []Count{
		{"Elements declared by this repository", fmt.Sprint(declared)},
		{"Elements only referenced — they exist, nothing more is known", fmt.Sprint(referenced)},
		{"Relationships", fmt.Sprint(len(m.Edges))},
		{"… of which state a protocol", fmt.Sprint(withProtocol)},
		{"Elements carrying a description", fmt.Sprintf("%d of %d", described, len(m.Nodes))},
		{"Elements carrying any model-written value", fmt.Sprint(interpreted)},
		{"Gaps the validator reported", fmt.Sprint(len(gaps))},
	}
}

// limits is fixed, and deliberately so: these are properties of reading configuration, not of this
// repository. Naming them is how the document stays honest about its own method.
var limits = []Limit{
	{"The order of calls in a scenario", "Application source, or runtime traces"},
	{"What happens on failure — retries, fallbacks, queues", "Application source"},
	{"The structure *inside* a container — its modules and components", "Application source"},
	{"The shape of the data a container stores", "Schema or migration files"},
	{"Connections built at runtime from assembled values", "Application source"},
	{"Why any of it is this way", "A person — sections 1, 4 and 9"},
}

func sortedCandidates(in []archdoc.Candidate) []archdoc.Candidate {
	out := append([]archdoc.Candidate(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}

// configFiles are the gateway configurations routes were read from, deduplicated.
func configFiles(facts archdoc.FactSet) []string {
	return uniqueSorted(func(add func(string)) {
		for _, r := range facts.Routes {
			add(r.Config)
		}
	})
}

// envFiles are files other than the compose file that endpoints were cited in — the dotenv files
// compose names, which are read for interpolation.
func envFiles(facts archdoc.FactSet) []string {
	return uniqueSorted(func(add func(string)) {
		for _, s := range facts.Services {
			for _, e := range s.Endpoints {
				if e.Prov.File != "" && e.Prov.File != facts.Source {
					add(e.Prov.File)
				}
			}
		}
	})
}

func uniqueSorted(collect func(add func(string))) []string {
	seen := map[string]bool{}
	collect(func(s string) {
		if s != "" {
			seen[s] = true
		}
	})
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
