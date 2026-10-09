package render

import (
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// ComponentsFile is a page per component (F-19). Generated, hence the suffix (hard rule 2).
const ComponentsFile = "components.generated.md"

// listedEntries is how many of a component's routes its page lists; features.generated.md has all.
const listedEntries = 8

// Components renders a section per component of every container whose code was read: what it is
// made of, what it uses and is used by, the routes it handles and the tables it declares — all read
// from the code — and, where a model was asked what it does, its sentences, set in italics and
// each followed by the lines it cites. Empty when no code was read, and then not written.
func Components(m archdoc.Model, meta Meta) string {
	containers := m.Components()
	if len(containers) == 0 {
		return ""
	}
	name := func(id string) string {
		if n, ok := m.Node(id); ok {
			return n.Name
		}
		return id
	}
	explained := map[string]archdoc.Explanation{}
	for _, x := range m.Explanations {
		explained[x.Element] = x
	}

	var b strings.Builder
	b.WriteString("# Components\n\n")
	b.WriteString(meta.stamp())
	b.WriteString("Each container's parts, read from its code: what each is made of, what it uses and is used\n")
	b.WriteString("by, the routes it handles and the tables it declares. Where a model was asked what a part\n")
	b.WriteString("does, its sentences are in italics, each followed by the lines it rests on; a sentence that\n")
	b.WriteString("cited nothing was refused before it got here.\n")

	for _, c := range containers {
		fmt.Fprintf(&b, "\n## %s\n", name(c))
		view := m.Component(c)
		for _, n := range view.Nodes {
			fmt.Fprintf(&b, "\n### %s\n\n", n.Name)
			if x, ok := explained[n.ID]; ok {
				for _, cl := range x.Claims {
					cites := make([]string, 0, len(cl.Cites))
					for _, p := range cl.Cites {
						cites = append(cites, "`"+p.String()+"`")
					}
					fmt.Fprintf(&b, "*%s* <sup>%s</sup>\n", cl.Text, strings.Join(cites, " "))
				}
				note := "from the facts below"
				if x.Stale {
					note = "for an earlier version of the facts below; `archdoc explain` asks again"
				}
				fmt.Fprintf(&b, "\n<sub>Interpreted by %s %s.</sub>\n\n", x.Prov.Note, note)
			}
			fmt.Fprintf(&b, "- **Code:** `%s` — %d %s, %d lines\n", n.Dir, len(n.Files), plural(len(n.Files), "file", "files"), n.Lines)
			var uses, usedBy []string
			for _, e := range m.Edges {
				if e.From == n.ID {
					uses = append(uses, fmt.Sprintf("%s (%d)", name(e.To), e.Weight))
				}
				if to, ok := m.Node(e.From); e.To == n.ID && ok && to.Kind == archdoc.Component {
					usedBy = append(usedBy, fmt.Sprintf("%s (%d)", to.Name, e.Weight))
				}
			}
			if len(uses) > 0 {
				fmt.Fprintf(&b, "- **Uses**, by imports: %s\n", strings.Join(uses, ", "))
			}
			if len(usedBy) > 0 {
				fmt.Fprintf(&b, "- **Used by**: %s\n", strings.Join(usedBy, ", "))
			}
			var routes, pages []archdoc.Entry
			for _, e := range m.Entries {
				if e.Component != n.ID {
					continue
				}
				if e.Kind == "page" {
					pages = append(pages, e)
				} else {
					routes = append(routes, e)
				}
			}
			for _, group := range []struct {
				label   string
				entries []archdoc.Entry
			}{{"Handles", routes}, {"Pages", pages}} {
				if len(group.entries) == 0 {
					continue
				}
				var shown []string
				for i, e := range group.entries {
					if i == listedEntries {
						shown = append(shown, fmt.Sprintf("and %d more in [Features](%s)", len(group.entries)-listedEntries, FeaturesFile))
						break
					}
					shown = append(shown, "`"+strings.TrimPrefix(e.Method+" ", "PAGE ")+e.Path+"`")
				}
				fmt.Fprintf(&b, "- **%s** %d: %s\n", group.label, len(group.entries), strings.Join(shown, ", "))
			}
			files := map[string]bool{}
			for _, f := range n.Files {
				files[f] = true
			}
			var tables []string
			for _, t := range m.Nodes {
				if t.Kind == archdoc.Table && files[t.Dir] {
					tables = append(tables, "`"+t.Name+"`")
				}
			}
			if len(tables) > 0 {
				fmt.Fprintf(&b, "- **Declares tables**: %s\n", strings.Join(tables, ", "))
			}
		}
	}
	return b.String()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
