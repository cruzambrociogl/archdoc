package render

import (
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Line is one sentence of a story, and the line of code it rests on.
type Line struct {
	Text string             `json:"text"`
	Prov archdoc.Provenance `json:"provenance,omitzero"`
	// Interpreted is set on the one sentence a model wrote: what the project is for.
	Interpreted bool `json:"interpreted,omitempty"`
}

// storyEntries is how many ways in a story names before it counts the rest.
const storyEntries = 6

// Story tells a small project top down, in the order a person who did not write it asks: what is
// this, how do I use it, what does it talk to, what does it keep, what is it made of (vision §2.7:
// a vibe coder starts from a story, not from a reference). Every sentence is assembled from a
// fact and cites it; the only words a model may have written are the first line, marked. It is the
// same story on the page and in the app.
func Story(m archdoc.Model) []Line {
	var app archdoc.Node
	var parts, tables []archdoc.Node
	for _, n := range m.Nodes {
		switch {
		case n.Kind.Container():
			app = n
		case n.Kind == archdoc.Component:
			parts = append(parts, n)
		case n.Kind == archdoc.Table:
			tables = append(tables, n)
		}
	}
	var out []Line
	if m.Description != "" {
		out = append(out, Line{Text: m.Description, Prov: m.DescProv, Interpreted: m.DescProv.Origin.Interpretation()})
	}

	files, lines := 0, 0
	for _, n := range parts {
		files += len(n.Files)
		lines += n.Lines
	}
	kind := "an application"
	for _, e := range m.Edges {
		if e.To != app.ID {
			continue
		}
		if from, ok := m.Node(e.From); ok && from.Kind == archdoc.Actor {
			switch e.Label {
			case "runs":
				kind = "a tool you run"
			case "uses":
				kind = "an app you open"
				for _, entry := range m.Entries {
					if entry.Kind == "page" && (strings.HasSuffix(entry.Path, ".html") || strings.HasSuffix(entry.Path, ".htm")) {
						kind = "a page you open"
					}
				}
			}
		}
	}
	what := fmt.Sprintf("%s is %s: %d %s, %d lines", m.Name, kind, files, plural(files, "file", "files"), lines)
	if app.Technology != "" {
		what += " of " + app.Technology
	}
	out = append(out, Line{Text: what + ".", Prov: app.Prov})

	for i, e := range m.Entries {
		if i == storyEntries {
			out = append(out, Line{Text: fmt.Sprintf("There are %d more ways in.", len(m.Entries)-storyEntries)})
			break
		}
		var text string
		switch e.Kind {
		case "command":
			text = fmt.Sprintf("You run `%s`", e.Path)
		case "page":
			text = fmt.Sprintf("You open `%s`", e.Path)
		case "job":
			text = fmt.Sprintf("On its own it runs `%s`", e.Path)
		default:
			text = fmt.Sprintf("It answers `%s %s`", e.Method, e.Path)
		}
		if e.Summary != "" {
			text += " — " + strings.TrimSuffix(e.Summary, ".")
		}
		out = append(out, Line{Text: text + ".", Prov: e.Prov})
	}

	reaches := 0
	for _, e := range m.Edges {
		to, ok := m.Node(e.To)
		if !ok || to.Kind != archdoc.External || len(e.Prov) == 0 {
			continue
		}
		reaches++
		text := fmt.Sprintf("It %s %s", e.Label, to.Name)
		if e.Technology != "" {
			text += " over " + e.Technology
		}
		out = append(out, Line{Text: text + ".", Prov: e.Prov[0]})
	}
	if reaches == 0 && len(m.Unresolved) == 0 {
		out = append(out, Line{Text: "It reaches nothing outside itself that its code names."})
	}
	if n := len(m.Unresolved); n > 0 {
		out = append(out, Line{Text: fmt.Sprintf("It makes %d %s to an address it works out as it runs, which archdoc cannot name.",
			n, plural(n, "call", "calls")), Prov: m.Unresolved[0].Prov})
	}

	if len(tables) > 0 {
		names := make([]string, 0, len(tables))
		for _, t := range tables {
			names = append(names, t.Name)
		}
		out = append(out, Line{Text: fmt.Sprintf("It keeps %d %s: %s.", len(tables), plural(len(tables), "table", "tables"), strings.Join(names, ", ")),
			Prov: tables[0].Prov})
	}

	switch len(parts) {
	case 0:
	case 1:
		out = append(out, Line{Text: fmt.Sprintf("All of its code is in `%s`.", parts[0].Dir), Prov: parts[0].Prov})
	default:
		names := make([]string, 0, len(parts))
		for _, p := range parts {
			names = append(names, p.Name)
		}
		out = append(out, Line{Text: fmt.Sprintf("Its code is in %d parts: %s.", len(parts), strings.Join(names, ", ")), Prov: parts[0].Prov})
	}

	var deps []string
	var first archdoc.Provenance
	for _, d := range m.Dependencies {
		if !d.Dev && d.Container == app.ID {
			if len(deps) == 0 {
				first = d.Prov
			}
			deps = append(deps, d.Name)
		}
	}
	if len(deps) > 0 {
		out = append(out, Line{Text: "It is built on " + strings.Join(deps, ", ") + ".", Prov: first})
	}
	return out
}
