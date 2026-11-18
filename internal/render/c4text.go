package render

import (
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// The C4 diagrams as text other tools open (platform-surface §17.4, Download): Structurizr DSL, the
// format of C4's author, and C4-PlantUML. Each is one view — the elements and relationships that
// diagram draws, nothing more — with its title, and is written from the same view the picture is,
// so the two cannot disagree. Only the C4 views have them: What it stores and How data moves are
// not C4.

// isC4 reports whether a view is one of C4's: context, containers, components, deployment.
func isC4(v View) bool {
	return v.Name == "context" || v.Name == "context"+AllSuffix || v.Name == "container" || v.Name == "container"+AllSuffix ||
		v.Name == Deployment || strings.HasPrefix(v.Name, ComponentPrefix) || strings.HasPrefix(v.Name, MainPrefix) ||
		strings.HasPrefix(v.Name, StructurePrefix)
}

// HasText reports whether a view downloads as Structurizr DSL and C4-PlantUML.
func HasText(v View) bool { return isC4(v) }

// textIDs are identifiers both formats take: letters, digits and underscores, never a keyword.
func textIDs(m core.Model) map[string]string {
	ids := identifiers(m)
	for k, v := range ids {
		ids[k] = "e_" + v
	}
	return ids
}

// quoted is a string either format reads between double quotes: no quote, no line break.
func quoted(s string) string {
	return `"` + strings.NewReplacer(`"`, "'", "\n", " ", `\`, "/").Replace(s) + `"`
}

// relationships are the view's arrows, each pair and wording once: both formats refuse a second
// relationship that says the same thing.
func relationships(view core.Model, ids map[string]string) []core.Edge {
	seen := map[string]bool{}
	var out []core.Edge
	for _, e := range view.Edges {
		if ids[e.From] == "" || ids[e.To] == "" {
			continue
		}
		k := e.From + "\x00" + e.To + "\x00" + edgeText(e)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, e)
	}
	return out
}

// relWords is what an arrow says, without its technology: each format puts that in a field of its own.
func relWords(e core.Edge) string {
	if e.Label != "" {
		return e.Label
	}
	return "uses"
}

// innerOf is the container a component view is drawn inside, by its name.
func innerOf(v View) string {
	for _, p := range []string{ComponentPrefix, MainPrefix, StructurePrefix} {
		if strings.HasPrefix(v.Name, p) {
			return strings.TrimPrefix(v.Name, p)
		}
	}
	return ""
}

// Structurizr is a view as a Structurizr DSL workspace: its elements, its relationships and the one
// view, laid out by Structurizr itself. False for a view that is not C4.
func Structurizr(m core.Model, v View) (string, bool) {
	if !isC4(v) {
		return "", false
	}
	h := HeadingOf(m, v)
	view := v.Model
	ids := textIDs(view)
	inner := innerOf(v)
	const sys = "e_the_system"

	var b strings.Builder
	fmt.Fprintf(&b, "workspace %s %s {\n\n", quoted(m.Name), quoted(h.Title+". From archdoc: every element and relationship read from the code."))
	b.WriteString("    model {\n")
	var inside, outside []core.Node
	for _, n := range view.Nodes {
		switch {
		case n.Kind == core.Actor || n.Kind == core.External || n.Evidence == core.Referenced || n.Kind == core.System:
			outside = append(outside, n)
		default:
			inside = append(inside, n)
		}
	}
	for _, n := range outside {
		switch {
		case n.Kind == core.Actor:
			fmt.Fprintf(&b, "        %s = person %s %s\n", ids[n.ID], quoted(n.Name), quoted(n.Description))
		case n.Kind == core.System:
			fmt.Fprintf(&b, "        %s = softwareSystem %s %s\n", ids[n.ID], quoted(n.Name), quoted(n.Description))
		default:
			fmt.Fprintf(&b, "        %s = softwareSystem %s %s %s\n", ids[n.ID], quoted(n.Name), quoted(n.Description), quoted("External"))
		}
	}
	if len(inside) > 0 {
		fmt.Fprintf(&b, "        %s = softwareSystem %s {\n", sys, quoted(m.Name))
		container := func(n core.Node, indent string) {
			tags := ""
			switch n.Kind {
			case core.Datastore:
				tags = " " + quoted("Database")
			case core.Queue:
				tags = " " + quoted("Queue")
			}
			fmt.Fprintf(&b, "%s%s = container %s %s %s%s\n", indent, ids[n.ID], quoted(n.Name), quoted(n.Description), quoted(n.Technology), tags)
		}
		if inner != "" {
			// A component view: the container it is drawn inside holds the components; the others
			// it touches stand beside it.
			parent, _ := m.Node(inner)
			fmt.Fprintf(&b, "            %s = container %s %s %s {\n", "e_the_container", quoted(parent.Name), quoted(parent.Description), quoted(parent.Technology))
			for _, n := range inside {
				if n.Kind == core.Component || n.Kind == core.Module {
					fmt.Fprintf(&b, "                %s = component %s %s %s\n", ids[n.ID], quoted(n.Name), quoted(n.Description), quoted(n.Technology))
				}
			}
			b.WriteString("            }\n")
			for _, n := range inside {
				if n.Kind != core.Component && n.Kind != core.Module {
					container(n, "            ")
				}
			}
		} else {
			for _, n := range inside {
				container(n, "            ")
			}
		}
		b.WriteString("        }\n")
	}
	b.WriteString("\n")
	for _, e := range relationships(view, ids) {
		if e.Technology != "" {
			fmt.Fprintf(&b, "        %s -> %s %s %s\n", ids[e.From], ids[e.To], quoted(relWords(e)), quoted(e.Technology))
		} else {
			fmt.Fprintf(&b, "        %s -> %s %s\n", ids[e.From], ids[e.To], quoted(relWords(e)))
		}
	}
	if v.Name == Deployment {
		fmt.Fprintf(&b, "\n        deploymentEnvironment %s {\n", quoted("Docker Compose"))
		fmt.Fprintf(&b, "            deploymentNode %s %s %s {\n", quoted(m.Source), quoted("What the Compose file runs"), quoted("Docker Compose"))
		for _, n := range inside {
			fmt.Fprintf(&b, "                containerInstance %s\n", ids[n.ID])
		}
		b.WriteString("            }\n        }\n")
	}
	b.WriteString("    }\n\n    views {\n")
	key := strings.NewReplacer(":", "-", "/", "-", " ", "-").Replace(v.File)
	switch {
	case v.Name == "context" || v.Name == "context"+AllSuffix:
		system := sys
		for _, n := range outside {
			if n.Kind == core.System {
				system = ids[n.ID]
			}
		}
		fmt.Fprintf(&b, "        systemContext %s %s %s {\n", system, quoted(key), quoted(h.Title))
	case v.Name == Deployment:
		fmt.Fprintf(&b, "        deployment %s %s %s %s {\n", sys, quoted("Docker Compose"), quoted(key), quoted(h.Title))
	case inner != "":
		fmt.Fprintf(&b, "        component %s %s %s {\n", "e_the_container", quoted(key), quoted(h.Title))
	default:
		fmt.Fprintf(&b, "        container %s %s %s {\n", sys, quoted(key), quoted(h.Title))
	}
	b.WriteString("            include *\n            autoLayout\n        }\n")
	b.WriteString("        styles {\n")
	b.WriteString("            element \"External\" {\n                background #999999\n                color #ffffff\n            }\n")
	b.WriteString("            element \"Database\" {\n                shape cylinder\n            }\n")
	b.WriteString("            element \"Queue\" {\n                shape pipe\n            }\n")
	b.WriteString("        }\n        theme default\n    }\n}\n")
	return b.String(), true
}

// PlantUML is a view in C4-PlantUML, the standard library's C4 macros: its title, its elements in
// their boundary, its relationships and C4-PlantUML's own legend. False for a view that is not C4.
func PlantUML(m core.Model, v View) (string, bool) {
	if !isC4(v) {
		return "", false
	}
	h := HeadingOf(m, v)
	view := v.Model
	ids := textIDs(view)
	inner := innerOf(v)

	library := "C4_Container"
	switch {
	case v.Name == "context" || v.Name == "context"+AllSuffix:
		library = "C4_Context"
	case inner != "":
		library = "C4_Component"
	case v.Name == Deployment:
		library = "C4_Deployment"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "@startuml\n!include <C4/%s>\n\ntitle %s\n\n", library, strings.ReplaceAll(h.Title, "\n", " "))
	element := func(n core.Node, indent string) {
		macro := "Container"
		switch {
		case n.Kind == core.Actor:
			fmt.Fprintf(&b, "%sPerson(%s, %s, %s)\n", indent, ids[n.ID], quoted(n.Name), quoted(n.Description))
			return
		case n.Kind == core.System:
			fmt.Fprintf(&b, "%sSystem(%s, %s, %s)\n", indent, ids[n.ID], quoted(n.Name), quoted(n.Description))
			return
		case n.Kind == core.External:
			fmt.Fprintf(&b, "%sSystem_Ext(%s, %s, %s)\n", indent, ids[n.ID], quoted(n.Name), quoted(n.Description))
			return
		case n.Kind == core.Component || n.Kind == core.Module:
			macro = "Component"
		case n.Kind == core.Datastore:
			macro = "ContainerDb"
		case n.Kind == core.Queue:
			macro = "ContainerQueue"
		}
		if n.Evidence == core.Referenced {
			macro += "_Ext"
		}
		fmt.Fprintf(&b, "%s%s(%s, %s, %s, %s)\n", indent, macro, ids[n.ID], quoted(n.Name), quoted(n.Technology), quoted(n.Description))
	}

	var inside, outside []core.Node
	for _, n := range view.Nodes {
		if n.Kind == core.Actor || n.Kind == core.External || n.Kind == core.System || n.Evidence == core.Referenced {
			outside = append(outside, n)
		} else {
			inside = append(inside, n)
		}
	}
	for _, n := range outside {
		element(n, "")
	}
	if len(inside) > 0 {
		switch {
		case v.Name == Deployment:
			fmt.Fprintf(&b, "Deployment_Node(e_the_node, %s, %s) {\n", quoted(m.Source), quoted("Docker Compose"))
		case inner != "":
			parent, _ := m.Node(inner)
			fmt.Fprintf(&b, "Container_Boundary(e_the_container, %s) {\n", quoted(parent.Name))
		default:
			fmt.Fprintf(&b, "System_Boundary(e_the_system, %s) {\n", quoted(m.Name))
		}
		for _, n := range inside {
			element(n, "    ")
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")
	for _, e := range relationships(view, ids) {
		if e.Technology != "" {
			fmt.Fprintf(&b, "Rel(%s, %s, %s, %s)\n", ids[e.From], ids[e.To], quoted(relWords(e)), quoted(e.Technology))
		} else {
			fmt.Fprintf(&b, "Rel(%s, %s, %s)\n", ids[e.From], ids[e.To], quoted(relWords(e)))
		}
	}
	b.WriteString("\nSHOW_LEGEND()\n@enduml\n")
	return b.String(), true
}
