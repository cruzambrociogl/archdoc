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

func Index(m archdoc.Model, meta Meta) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s — architecture\n\n", m.Name)
	b.WriteString(meta.stamp())

	fmt.Fprintf(&b, "Derived from `%s`. Every element below cites the line that declares it.\n\n", meta.Source)

	b.WriteString("## System context\n\n")
	b.WriteString(fence(Mermaid(m.Context(), false)))

	b.WriteString("\n## Containers\n\n")
	b.WriteString(fence(Mermaid(m.Container(), true)))

	b.WriteString("\n## Sections\n\n")
	b.WriteString("Twelve arc42 sections. Five are generated from facts and overwritten on every\n")
	b.WriteString("run; seven are yours, created once and never read or written again.\n\n")
	b.WriteString("| | Section | Written by | archdoc |\n|---|---|---|---|\n")

	for _, s := range Sections() {
		owner, state := "archdoc", "overwritten every run"
		if s.Owner == Human {
			owner, state = "you", "never read or written"
		}
		fmt.Fprintf(&b, "| %d | [%s](%s) | %s | %s |\n", s.Number, s.Title, s.File(), owner, state)
	}

	b.WriteString("\nA section you have not written yet holds questions derived from this model rather\n")
	b.WriteString("than a blank template. archdoc cannot answer them; it can say which ones matter.\n")

	return b.String()
}
