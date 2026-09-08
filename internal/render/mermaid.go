// Package render turns a view into something a person can look at.
//
// Sprint 1 emits Mermaid only. §8 commits to both Mermaid and a pre-rendered SVG — the SVG for
// fidelity with the app's own layout, Mermaid for portability and diffing — but the SVG needs a
// layout engine, and Mermaid needs nothing: GitHub renders it inside markdown with no build
// step and no infrastructure at all.
package render

import (
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Mermaid renders a view as a flowchart.
//
// Not C4Context/C4Container, which Mermaid does provide. Those are still marked experimental,
// GitHub's renderer lags the upstream release, and §8 already flags the risk. A flowchart is
// plain, renders everywhere, and carries the same information — the C4 vocabulary is in the
// labels, where a reader can see it, rather than in a syntax that might not draw.
//
// group is the system boundary: true for a container view, where the declared containers sit
// inside a box, and false for a context view, where the system is already one node.
func Mermaid(m archdoc.Model, group bool) string {
	var b strings.Builder

	ids := identifiers(m)

	b.WriteString("flowchart TB\n")

	inside, outside := partition(m.Nodes, group)

	for _, n := range outside {
		fmt.Fprintf(&b, "    %s\n", node(ids[n.ID], n))
	}

	if len(inside) > 0 {
		fmt.Fprintf(&b, "\n    subgraph boundary[%q]\n", m.Name)

		if groups := boundaries(m, inside); groups != nil {
			// Networks nest, so the boundary does too: the reader sees which containers a
			// declared network can and cannot reach.
			for _, n := range ungrouped(inside) {
				fmt.Fprintf(&b, "        %s\n", node(ids[n.ID], n))
			}
			for _, g := range groups {
				writeGroup(&b, g, ids, 2)
			}
		} else {
			for _, n := range inside {
				fmt.Fprintf(&b, "        %s\n", node(ids[n.ID], n))
			}
		}

		b.WriteString("    end\n")
	}

	if len(m.Edges) > 0 {
		b.WriteString("\n")
	}
	for _, e := range m.Edges {
		from, to := ids[e.From], ids[e.To]
		if from == "" || to == "" {
			continue // an edge to a node this view dropped
		}
		if label := edgeLabel(e); label != "" {
			fmt.Fprintf(&b, "    %s -->|%q| %s\n", from, label, to)
			continue
		}
		fmt.Fprintf(&b, "    %s --> %s\n", from, to)
	}

	b.WriteString(styles(m, ids))

	return b.String()
}

// writeGroup renders one network boundary and everything nested inside it.
//
// An empty group is skipped rather than drawn: two networks with identical membership put every
// node in one of them, and an empty box on a diagram reads as a missing element.
func writeGroup(b *strings.Builder, g *group, ids map[string]string, depth int) {
	if len(g.Nodes) == 0 && len(g.Children) == 0 {
		return
	}

	pad := strings.Repeat("    ", depth)

	label := escape(g.Name)
	if g.Internal {
		// Compose's own `internal: true`. Worth saying on the diagram, because it is the one
		// reachability claim the configuration makes rather than implies.
		label += "<br/>[no external connectivity]"
	}

	fmt.Fprintf(b, "%ssubgraph net_%s[%q]\n", pad, sanitise(g.Name), label)
	for _, n := range g.Nodes {
		fmt.Fprintf(b, "%s    %s\n", pad, node(ids[n.ID], n))
	}
	for _, child := range g.Children {
		writeGroup(b, child, ids, depth+1)
	}
	fmt.Fprintf(b, "%send\n", pad)
}

// partition splits nodes into those inside the system boundary and those outside it. Actors and
// external systems are always outside — that is what the evidence kind already decided.
func partition(nodes []archdoc.Node, group bool) (inside, outside []archdoc.Node) {
	for _, n := range nodes {
		if group && n.Kind != archdoc.Actor && n.Kind != archdoc.External {
			inside = append(inside, n)
			continue
		}
		outside = append(outside, n)
	}
	return inside, outside
}

// node renders one element with its C4 label: name, then what it is and what it runs.
func node(id string, n archdoc.Node) string {
	label := fmt.Sprintf("<b>%s</b><br/>%s", escape(n.Name), typeLabel(n))
	if n.Description != "" {
		// The line a C4 container is supposed to carry and configuration never states.
		label += "<br/><br/>" + escape(n.Description)
	}

	switch n.Kind {
	case archdoc.Datastore:
		return fmt.Sprintf("%s[(%q)]", id, label)
	case archdoc.Actor:
		return fmt.Sprintf("%s([%q])", id, label)
	case archdoc.Queue:
		return fmt.Sprintf("%s[/%q/]", id, label)
	default:
		return fmt.Sprintf("%s[%q]", id, label)
	}
}

// typeLabel is the bracketed line under an element's name, in C4's own vocabulary. The
// technology appears only when the catalog knew it — an empty one says "not known from
// configuration", and writing "[Container: ]" would say nothing at greater length.
func typeLabel(n archdoc.Node) string {
	kind := "Container"
	switch n.Kind {
	case archdoc.Actor:
		kind = "Person"
	case archdoc.External:
		kind = "External System"
	case archdoc.System:
		kind = "Software System"
	}

	if n.Technology == "" {
		return "[" + kind + "]"
	}
	return "[" + kind + ": " + escape(n.Technology) + "]"
}

func edgeLabel(e archdoc.Edge) string {
	switch {
	case e.Label != "" && e.Technology != "":
		return e.Label + "<br/>[" + escape(e.Technology) + "]"
	case e.Label != "":
		return e.Label
	case e.Technology != "":
		return "[" + escape(e.Technology) + "]"
	default:
		return ""
	}
}

// styles colour the two things a reader must not confuse: what the repository declares, and
// what it only references. Referenced elements are drawn hollow and dashed — the diagram itself
// says "we know this is talked to, and nothing more".
func styles(m archdoc.Model, ids map[string]string) string {
	var declared, referenced, actors []string

	for _, n := range m.Nodes {
		id := ids[n.ID]
		switch {
		case n.Kind == archdoc.Actor:
			actors = append(actors, id)
		case n.Evidence == archdoc.Referenced:
			referenced = append(referenced, id)
		default:
			declared = append(declared, id)
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("    classDef declared fill:#1168bd,stroke:#0b4884,color:#ffffff\n")
	b.WriteString("    classDef referenced fill:#ffffff,stroke:#8b8b8b,color:#3d3d3d,stroke-dasharray:4 3\n")
	b.WriteString("    classDef actor fill:#08427b,stroke:#052e56,color:#ffffff\n")

	// A slice, not a map. Ranging a map here would reorder these three lines between runs and
	// break AC-7 without changing a single pixel of the rendered diagram — which is precisely
	// what makes it the defect worth guarding against.
	for _, c := range []struct {
		class   string
		members []string
	}{
		{"actor", actors},
		{"declared", declared},
		{"referenced", referenced},
	} {
		if len(c.members) > 0 {
			fmt.Fprintf(&b, "    class %s %s\n", strings.Join(c.members, ","), c.class)
		}
	}

	return b.String()
}

// identifiers maps model IDs to Mermaid-safe ones. Model IDs carry punctuation Mermaid cannot
// take — "ext:s3.amazonaws.com" — and two different IDs must never flatten onto the same
// identifier, so a suffix is added when they would.
func identifiers(m archdoc.Model) map[string]string {
	out := make(map[string]string, len(m.Nodes))
	taken := map[string]bool{}

	for _, n := range m.Nodes {
		id := sanitise(n.ID)
		for i := 2; taken[id]; i++ {
			id = fmt.Sprintf("%s_%d", sanitise(n.ID), i)
		}
		taken[id] = true
		out[n.ID] = id
	}

	return out
}

func sanitise(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// escape removes the two characters that would break out of a quoted Mermaid label.
func escape(s string) string {
	return strings.NewReplacer(`"`, "'", "\n", " ").Replace(s)
}
