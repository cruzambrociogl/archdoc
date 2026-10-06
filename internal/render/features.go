package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// FeaturesFile lists what the system does, from its code. Generated, hence the suffix (hard rule 2).
const FeaturesFile = "features.generated.md"

// Features renders the routes the code declares, grouped by container and by the class that
// handles them — the app's Features screen, committed beside the other documents. Empty when the
// code declares none, and then not written.
func Features(m archdoc.Model, meta Meta) string {
	if len(m.Entries) == 0 {
		return ""
	}
	name := func(id string) string {
		if n, ok := m.Node(id); ok {
			return n.Name
		}
		return id
	}
	described := 0
	groups := map[string]map[string][]archdoc.Entry{} // container → handler class → routes
	for _, e := range m.Entries {
		if e.Summary != "" {
			described++
		}
		cls := e.Group()
		if groups[e.Container] == nil {
			groups[e.Container] = map[string][]archdoc.Entry{}
		}
		groups[e.Container][cls] = append(groups[e.Container][cls], e)
	}

	var b strings.Builder
	b.WriteString("# Features\n\n")
	b.WriteString(meta.stamp())
	fmt.Fprintf(&b, "What the system does, read from its code: %d routes and pages, %d described by the code itself — a\n", len(m.Entries), described)
	b.WriteString("summary its decorators state. A route with none shows only its handler; nothing here is\n")
	b.WriteString("written by a model.\n")
	for _, c := range sortedKeys(groups) {
		fmt.Fprintf(&b, "\n## %s\n", name(c))
		for _, cls := range sortedKeys(groups[c]) {
			fmt.Fprintf(&b, "\n### %s\n\n", strings.TrimSuffix(cls, "Controller"))
			b.WriteString("| Route | What it does | Handled by | Declared at |\n|---|---|---|---|\n")
			for _, e := range groups[c][cls] {
				what := "—"
				if e.Summary != "" {
					what = cited(e.Summary, e.SummaryProv)
				}
				route := e.Method + " " + e.Path
				if e.Kind == "page" {
					route = e.Path
				}
				fmt.Fprintf(&b, "| `%s` | %s | `%s` | `%s` |\n", route, what, e.Handler, e.Prov)
			}
		}
	}
	if len(m.Unresolved) > 0 {
		b.WriteString("\n## Calls whose target is computed at run time\n\n")
		b.WriteString("The code makes these calls; nothing in it names where they go. Listed, not drawn.\n\n")
		for _, u := range m.Unresolved {
			fmt.Fprintf(&b, "- `%s` — %s, `%s`\n", u.What, name(u.Container), u.Prov)
		}
	}
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
