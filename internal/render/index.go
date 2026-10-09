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
	if len(plan) == 0 {
		return onePage(m, meta)
	}
	var b strings.Builder

	fmt.Fprintf(&b, "# %s — architecture\n\n", m.Name)
	b.WriteString(meta.stamp())

	fmt.Fprintf(&b, "Derived from `%s`. Every element below cites the line that declares it.\n\n", meta.Source)

	b.WriteString("## System context\n\n")
	b.WriteString(figure(meta, "System context", "context.svg", Mermaid(m.Context(), false)))

	b.WriteString("\n## Containers\n\n")
	b.WriteString(figure(meta, "Containers", "container.svg", Mermaid(m.Container(), true)))

	// One component view per container whose code was read, and one data view per container whose
	// code declares tables.
	var components, data []View
	for _, v := range Views(m)[2:] {
		if strings.HasPrefix(v.Name, DataPrefix) {
			data = append(data, v)
		} else {
			components = append(components, v)
		}
	}
	if len(components) > 0 {
		b.WriteString("\n## Components\n\n")
		b.WriteString("Inside each container whose code archdoc reads: its parts, and which uses which. Every\n")
		fmt.Fprintf(&b, "arrow is an import, cited at its line in `.archdoc/model.json`; [each part](%s) has its page.\n", ComponentsFile)
		for _, v := range components {
			fmt.Fprintf(&b, "\n### %s\n\n", v.Model.Name)
			b.WriteString(figure(meta, v.Title, v.File+".svg", Mermaid(v.Model, v.Group)))
		}
	}
	if len(data) > 0 {
		b.WriteString("\n## Data\n\n")
		b.WriteString("The tables each container's code declares, with their columns; every arrow is a foreign key.\n")
		for _, v := range data {
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

// onePage is the whole documentation of a small project (vision D-13): what it is, what it does,
// what it reaches and what it is made of, each line cited — and nothing to fill in. It is what a
// person who did not write the code reads first, so it is written top down and in plain words.
func onePage(m archdoc.Model, meta Meta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", m.Name)
	b.WriteString(meta.stamp())

	var app archdoc.Node
	var parts, outside []archdoc.Node
	for _, n := range m.Nodes {
		switch {
		case n.Kind.Container():
			app = n
		case n.Kind == archdoc.Component:
			parts = append(parts, n)
		case n.Kind == archdoc.External:
			outside = append(outside, n)
		}
	}
	files, lines := 0, 0
	for _, n := range parts {
		files += len(n.Files)
		lines += n.Lines
	}
	if m.Description != "" {
		fmt.Fprintf(&b, "*%s*\n\n", m.Description)
	}
	fmt.Fprintf(&b, "A small project: %d %s, %d lines", files, plural(files, "file", "files"), lines)
	if app.Technology != "" {
		fmt.Fprintf(&b, " of %s", app.Technology)
	}
	b.WriteString(". This page is all of its documentation;\nevery line below cites where in the code it comes from.\n\n")

	b.WriteString("## What it is\n\n")
	b.WriteString(figure(meta, "System context", "context.svg", Mermaid(m.Context(), false)))

	if len(m.Entries) > 0 {
		b.WriteString("\n## What it does\n\n")
		b.WriteString("The ways in that the code declares: what a person runs or opens, and what it answers.\n\n")
		for _, e := range m.Entries {
			what := map[string]string{"command": "run", "page": "page", "http": e.Method, "job": "job"}[e.Kind]
			fmt.Fprintf(&b, "- **%s** `%s`", what, e.Path)
			if e.Summary != "" {
				fmt.Fprintf(&b, " — %s", e.Summary)
			}
			fmt.Fprintf(&b, " <sup>`%s`</sup>\n", e.Prov)
		}
	}

	b.WriteString("\n## What it reaches\n\n")
	name := func(id string) string {
		if n, ok := m.Node(id); ok {
			return n.Name
		}
		return id
	}
	reached := 0
	for _, e := range m.Edges {
		to, ok := m.Node(e.To)
		if !ok || to.Kind != archdoc.External || len(e.Prov) == 0 {
			continue
		}
		reached++
		fmt.Fprintf(&b, "- %s **%s**", e.Label, name(e.To))
		if e.Technology != "" {
			fmt.Fprintf(&b, " over %s", e.Technology)
		}
		if to.Description != "" {
			fmt.Fprintf(&b, " — %s", to.Description)
		}
		fmt.Fprintf(&b, " <sup>`%s`</sup>\n", e.Prov[0])
	}
	if reached == 0 {
		b.WriteString("Nothing outside itself that the code names: no address it calls, no service it is a client of.\n")
	}
	if len(m.Unresolved) > 0 {
		fmt.Fprintf(&b, "\n%d %s to an address the code works out as it runs — archdoc saw the call and cannot say where it goes:\n\n",
			len(m.Unresolved), plural(len(m.Unresolved), "call goes", "calls go"))
		for _, u := range m.Unresolved {
			fmt.Fprintf(&b, "- `%s` <sup>`%s`</sup>\n", u.What, u.Prov)
		}
	}

	if len(parts) > 0 {
		b.WriteString("\n## What it is made of\n\n")
		said := map[string]archdoc.Explanation{}
		for _, x := range m.Explanations {
			if !x.Stale && len(x.Claims) > 0 {
				said[x.Element] = x
			}
		}
		for _, n := range parts {
			fmt.Fprintf(&b, "- **%s** — `%s`, %d lines", n.Name, n.Dir, n.Lines)
			if len(n.Files) > 1 {
				fmt.Fprintf(&b, " in %d files", len(n.Files))
			}
			var uses []string
			for _, e := range m.Edges {
				if e.From == n.ID {
					uses = append(uses, name(e.To))
				}
			}
			if len(uses) > 0 {
				fmt.Fprintf(&b, "; uses %s", strings.Join(uses, ", "))
			}
			b.WriteString("\n")
			if x, ok := said[n.ID]; ok {
				for _, c := range x.Claims {
					fmt.Fprintf(&b, "  *%s*\n", c.Text)
				}
				fmt.Fprintf(&b, "  <sub>Interpreted by %s from what archdoc read; each sentence cites it in [the app](%s).</sub>\n", x.Prov.Note, CoverageFile)
			}
		}
		if len(parts) > 1 {
			for _, v := range Views(m)[2:] {
				if strings.HasPrefix(v.Name, ComponentPrefix) {
					b.WriteString("\n")
					b.WriteString(figure(meta, v.Title, v.File+".svg", Mermaid(v.Model, v.Group)))
				}
			}
		}
	}
	for _, v := range Views(m)[2:] {
		if strings.HasPrefix(v.Name, DataPrefix) {
			b.WriteString("\n## What it stores\n\n")
			b.WriteString(figure(meta, v.Title, v.File+".svg", Mermaid(v.Model, v.Group)))
		}
	}
	if len(m.Dependencies) > 0 {
		var deps []string
		for _, d := range m.Dependencies {
			if !d.Dev {
				deps = append(deps, "`"+d.Name+"`")
			}
		}
		if len(deps) > 0 {
			fmt.Fprintf(&b, "\n## What it is built on\n\n%s\n", strings.Join(deps, ", "))
		}
	}

	fmt.Fprintf(&b, "\n## What archdoc could not see\n\n[Coverage](%s) lists every file that was read and where the evidence runs out.\n", CoverageFile)
	b.WriteString("This is one page because the project is small. When it grows — a second application, a\n")
	b.WriteString("database of its own, more code than a page can hold — the arc42 chapters are written for it.\n")
	return b.String()
}
