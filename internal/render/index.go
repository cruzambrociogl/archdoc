package render

import (
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Index is the landing page: both diagrams, and a link to every section with its state.
//
// OUT-05 calls this file `index.md`. It is `index.generated.md` here, because hard rule 2 makes
// the suffix the mechanism that tells a person — and the next run — what may be overwritten. A
// file that looks human-owned and is silently replaced is the failure the boundary exists to
// prevent, and an index of links has nothing in it for a human to write anyway.
const IndexFile = "index.generated.md"

// State is what the caller found on disk for each human-owned section before the run. It is
// reported on the terminal, not written into the index.
//
// The index cannot carry it. Whether a section "awaits you" changes the moment the same run
// creates it, so an index that reported it would differ between two otherwise identical runs —
// and a file that churns is a file nobody trusts in a diff. Run-specific news belongs on the
// terminal; the committed document says only what is durably true.
//
// Reporting *filled versus still-a-stub* is OUT-09, and it needs content hashes recorded in
// .archdoc/ — OUT-03 forbids reading the file to find out. That is not built yet.
type State map[string]bool // file name → existed before this run

// Index lists the sections in plan — the ones this repository has evidence for (see plan.go) —
// and the coverage report. Pass Sections() to list all twelve.
func Index(m archdoc.Model, plan []Section, meta Meta) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s — architecture\n\n", m.Name)
	b.WriteString(meta.stamp())

	fmt.Fprintf(&b, "Derived from `%s`. Every element below cites the line that declares it.\n\n", meta.Source)

	b.WriteString("## System context\n\n")
	b.WriteString(figure(meta, "System context", "context.svg", Mermaid(m.Context(), false)))

	b.WriteString("\n## Containers\n\n")
	b.WriteString(figure(meta, "Containers", "container.svg", Mermaid(m.Container(), true)))

	// One component view per container whose code was read: what it is made of, from its code.
	if views := Views(m)[2:]; len(views) > 0 {
		b.WriteString("\n## Components\n\n")
		b.WriteString("Inside each container whose code archdoc reads: its parts, and which uses which. Every\n")
		b.WriteString("arrow is an import, cited at its line in `.archdoc/model.json`.\n")
		for _, v := range views {
			fmt.Fprintf(&b, "\n### %s\n\n", v.Model.Name)
			b.WriteString(figure(meta, v.Title, v.File+".svg", Mermaid(v.Model, v.Group)))
		}
	}

	b.WriteString("\n## Sections\n\n")
	generated, human := 0, 0
	for _, s := range plan {
		if s.Owner == Human {
			human++
			continue
		}
		generated++
	}
	fmt.Fprintf(&b, "%d arc42 section(s): %d generated from facts and overwritten on every run,\n",
		len(plan), generated)
	fmt.Fprintf(&b, "%d yours — created once and never read or written again.\n\n", human)
	b.WriteString("| | Section | Written by | archdoc |\n|---|---|---|---|\n")

	for _, s := range plan {
		owner, state := "archdoc", "overwritten every run"
		if s.Owner == Human {
			owner, state = "you", "never read or written"
		}
		fmt.Fprintf(&b, "| %d | [%s](%s) | %s | %s |\n", s.Number, s.Title, s.File(), owner, state)
	}

	if omitted := omittedSections(plan); len(omitted) > 0 {
		fmt.Fprintf(&b, "\narc42 has twelve sections. %s are not here, because this repository states\n",
			strings.Join(omitted, ", "))
		b.WriteString("nothing that would fill them — a stub asking about structure a system does not have\n")
		b.WriteString("teaches a reader to ignore stubs. They appear if the architecture grows into them.\n")
	}

	b.WriteString("\nA section you have not written yet holds questions derived from this model rather\n")
	b.WriteString("than a blank template. archdoc cannot answer them; it can say which ones matter.\n")

	if len(m.Entries) > 0 {
		fmt.Fprintf(&b, "\n## What it does\n\n[Features](%s) lists the %d routes the code declares, each at its line.\n", FeaturesFile, len(m.Entries))
	}

	fmt.Fprintf(&b, "\n## What archdoc could not see\n\n[Coverage](%s) lists every file that was read, every\n", CoverageFile)
	b.WriteString("relationship the configuration leaves incomplete, and what configuration cannot\n")
	b.WriteString("state at all. The rest of these documents is proven; that page says where the proof\n")
	b.WriteString("runs out.\n")

	return b.String()
}

// omittedSections names the arc42 sections this document set leaves out, so the reader learns the
// omission was a decision rather than an oversight.
func omittedSections(plan []Section) []string {
	var out []string
	for _, s := range Sections() {
		if !Planned(plan, s.Number) {
			out = append(out, fmt.Sprintf("§%d (%s)", s.Number, s.Title))
		}
	}
	return out
}
