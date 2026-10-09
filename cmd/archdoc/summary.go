package main

import (
	"fmt"
	"io"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// tally is what a model holds, counted: what generate reports at its end, scan previews, and
// status repeats — said the same way by all three.
type tally struct {
	Elements      int `json:"elements"`
	Relationships int `json:"relationships"`
	Components    int `json:"components"`
	WithCode      int `json:"containers_with_code"`
	Uses          int `json:"uses"`
	Tables        int `json:"tables"`
	WithTables    int `json:"containers_with_tables"`
	ForeignKeys   int `json:"foreign_keys"`
	Routes        int `json:"routes"`
	Pages         int `json:"pages"`
	Commands      int `json:"commands"`
	Jobs          int `json:"jobs"`
	Flows         int `json:"flows"`
	Unresolved    int `json:"unresolved"`
}

func countModel(m archdoc.Model) tally {
	t := tally{WithCode: len(m.Components()), WithTables: len(m.Datas()), Flows: len(m.Flows), Unresolved: len(m.Unresolved)}
	kind := map[string]archdoc.Kind{}
	for _, n := range m.Nodes {
		kind[n.ID] = n.Kind
		switch {
		case n.Kind == archdoc.Component:
			t.Components++
		case n.Kind == archdoc.Table:
			t.Tables++
		case !n.Kind.Part():
			t.Elements++
		}
	}
	for _, e := range m.Edges {
		switch k := kind[e.From]; {
		case k == archdoc.Component:
			t.Uses++
		case k == archdoc.Table:
			t.ForeignKeys++
		case k.Part():
			// the same code by folder: counted as components, not twice
		default:
			t.Relationships++
		}
	}
	for _, e := range m.Entries {
		switch e.Kind {
		case "http":
			t.Routes++
		case "page":
			t.Pages++
		case "command":
			t.Commands++
		case "job":
			t.Jobs++
		}
	}
	return t
}

// print says it: configuration's elements first, then what the code adds inside them.
func (t tally) print(out io.Writer, source string) {
	fmt.Fprintf(out, "%d %s, %d %s, from %s\n", t.Elements, plural(t.Elements, "element", "elements"),
		t.Relationships, plural(t.Relationships, "relationship", "relationships"), source)
	if t.Components > 0 {
		fmt.Fprintf(out, "%d %s in %s, %d %s between them, from the code\n", t.Components, plural(t.Components, "component", "components"),
			containers(t.WithCode), t.Uses, plural(t.Uses, "use", "uses"))
	}
	if t.Tables > 0 {
		keys := "foreign keys"
		if t.ForeignKeys == 1 {
			keys = "foreign key"
		}
		fmt.Fprintf(out, "%d %s in %s, %d %s between them, from the code\n", t.Tables, plural(t.Tables, "table", "tables"), containers(t.WithTables), t.ForeignKeys, keys)
	}
	if t.Routes+t.Pages+t.Commands+t.Jobs+t.Unresolved > 0 {
		fmt.Fprintf(out, "%d %s, %d %s, %d %s, %d %s, and %d %s whose target is computed at run time\n",
			t.Routes, plural(t.Routes, "route", "routes"), t.Pages, plural(t.Pages, "page", "pages"),
			t.Commands, plural(t.Commands, "command", "commands"), t.Jobs, plural(t.Jobs, "job", "jobs"),
			t.Unresolved, plural(t.Unresolved, "call", "calls"))
	}
	if t.Flows > 0 {
		fmt.Fprintf(out, "%d %s followed through the code from them\n", t.Flows, plural(t.Flows, "flow", "flows"))
	}
}

func containers(n int) string {
	if n == 1 {
		return "1 container"
	}
	return fmt.Sprintf("%d containers", n)
}
