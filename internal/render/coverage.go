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
	Rule    string // VAL-03, VAL-06 …
	Element string
	Message string
}

// Coverage renders the report.
func Coverage(m archdoc.Model, facts archdoc.FactSet, gaps []Gap, meta Meta) string {
	var b strings.Builder

	b.WriteString("# Coverage\n\n")
	b.WriteString(meta.stamp())
	b.WriteString("What archdoc read, what it could not resolve, and what configuration cannot state.\n")
	b.WriteString("Published so the edge of the map is visible: everything in the other sections is\n")
	b.WriteString("proven, and this page says where the proof runs out.\n\n")

	b.WriteString(sourcesRead(facts))
	b.WriteString(knownCounts(m, gaps))
	b.WriteString(unresolved(m, gaps))
	b.WriteString(cannotState())

	return b.String()
}

// sourcesRead lists every file discovery looked at — including the ones it skipped, and why. A
// reader can then tell "archdoc did not look" from "archdoc looked and found nothing".
func sourcesRead(facts archdoc.FactSet) string {
	var b strings.Builder

	b.WriteString("## What was read\n\n")
	b.WriteString("| File | Used | Why |\n|---|---|---|\n")

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
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", c.File, yesNo(c.Chosen), reason)
	}
	if !seen[facts.Source] && facts.Source != "" {
		fmt.Fprintf(&b, "| `%s` | yes | the configuration the model was built from |\n", facts.Source)
	}
	for _, f := range configFiles(facts) {
		fmt.Fprintf(&b, "| `%s` | yes | routing rules read from a mounted gateway configuration |\n", f)
	}
	for _, f := range envFiles(facts) {
		fmt.Fprintf(&b, "| `%s` | yes | network locations read from environment values |\n", f)
	}

	return b.String()
}

// knownCounts is the one place the documents say how much of themselves is backed by what.
func knownCounts(m archdoc.Model, gaps []Gap) string {
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

	var b strings.Builder
	b.WriteString("\n## How much is known\n\n")
	b.WriteString("| | Count |\n|---|---|\n")
	fmt.Fprintf(&b, "| Elements declared by this repository | %d |\n", declared)
	fmt.Fprintf(&b, "| Elements only referenced — they exist, nothing more is known | %d |\n", referenced)
	fmt.Fprintf(&b, "| Relationships | %d |\n", len(m.Edges))
	fmt.Fprintf(&b, "| … of which state a protocol | %d |\n", withProtocol)
	fmt.Fprintf(&b, "| Elements carrying a description | %d of %d |\n", described, len(m.Nodes))
	fmt.Fprintf(&b, "| Elements carrying any model-written value | %d |\n", interpreted)
	fmt.Fprintf(&b, "| Gaps the validator reported | %d |\n", len(gaps))

	return b.String()
}

// unresolved lists the validator's findings, grouped, plus the elements nothing connects to —
// which is usually evidence that the connection exists somewhere configuration cannot see.
func unresolved(m archdoc.Model, gaps []Gap) string {
	var b strings.Builder
	b.WriteString("\n## What could not be resolved\n\n")

	if len(gaps) == 0 {
		b.WriteString("Nothing. Every element and relationship the configuration states is complete.\n")
	} else {
		byRule := map[string][]Gap{}
		for _, g := range gaps {
			byRule[g.Rule] = append(byRule[g.Rule], g)
		}
		rules := make([]string, 0, len(byRule))
		for r := range byRule {
			rules = append(rules, r)
		}
		sort.Strings(rules) // AC-7: map iteration is randomised

		for _, r := range rules {
			list := byRule[r]
			sort.Slice(list, func(i, j int) bool {
				if list[i].Element != list[j].Element {
					return list[i].Element < list[j].Element
				}
				return list[i].Message < list[j].Message
			})
			fmt.Fprintf(&b, "**%s** — %d\n\n", r, len(list))
			for _, g := range list {
				fmt.Fprintf(&b, "- `%s` — %s\n", g.Element, g.Message)
			}
			b.WriteString("\n")
		}
	}

	if lonely := unconnected(m.Container()); len(lonely) > 0 {
		fmt.Fprintf(&b, "\n**Elements nothing reaches and that reach nothing** — %s.\n",
			strings.Join(lonely, ", "))
		b.WriteString("This is rarely true of a running system. It usually means the connection is made\n")
		b.WriteString("in application code or at deploy time, where configuration cannot see it. The\n")
		b.WriteString("element is drawn unconnected rather than joined up on a guess.\n")
	}

	return b.String()
}

// cannotState is fixed text, and deliberately so: these are properties of reading configuration,
// not of this repository. Naming them is how the document stays honest about its own method.
func cannotState() string {
	var b strings.Builder

	b.WriteString("\n## What configuration cannot state\n\n")
	b.WriteString("These are not gaps in this repository. They are the limits of the evidence:\n")
	b.WriteString("configuration declares *what exists and what may reach what*, and nothing else.\n\n")
	b.WriteString("| Not knowable from configuration | What would be needed |\n|---|---|\n")
	b.WriteString("| The order of calls in a scenario | Application source, or runtime traces |\n")
	b.WriteString("| What happens on failure — retries, fallbacks, queues | Application source |\n")
	b.WriteString("| The structure *inside* a container — its modules and components | Application source |\n")
	b.WriteString("| The shape of the data a container stores | Schema or migration files |\n")
	b.WriteString("| Connections built at runtime from assembled values | Application source |\n")
	b.WriteString("| Why any of it is this way | A person — sections 1, 4 and 9 |\n")

	return b.String()
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
