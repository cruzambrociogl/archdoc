package archdoc

import (
	"fmt"
	"io"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/rules"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// Reading is a repository read the way Generate reads it, with nothing written and nothing asked.
type Reading struct {
	// Facts is everything extraction read.
	Facts *FactSet
	// Documentable is false when there is nothing to document: no Compose file, no application
	// manifest, and no Python, JavaScript or HTML files.
	Documentable bool
}

// Read reads a repository: what Generate would find, before anything is derived from it.
func Read(root string) (*Reading, error) {
	facts, err := extract.Scan(root)
	if err != nil {
		return nil, err
	}
	return &Reading{Facts: facts, Documentable: documentable(facts)}, nil
}

// Model is the model Generate would write for this reading before any model is asked: derived,
// corrected by the repository's rules, validated.
func (r *Reading) Model() (Model, error) { return corrected(r.Facts) }

// ReadModel is the model Generate would write for a directory: extracted, corrected by its rules,
// validated. No model is asked, and nothing is written.
func ReadModel(root string) (Model, error) {
	facts, err := extract.Scan(root)
	if err != nil {
		return core.Model{}, err
	}
	if !documentable(facts) {
		return core.Model{}, fmt.Errorf("nothing to document: no Compose file, no application manifest, and no Python, JavaScript or HTML files")
	}
	return corrected(facts)
}

// documentable reports whether a repository has anything to document: a Compose file, or a
// manifest describing a running part of the system.
func documentable(facts *core.FactSet) bool {
	return facts.Source != "" || hasContainerApp(facts.Apps)
}

// corrected is the model of what extraction found, with the repository's rules applied and
// validated: what generate would write before any model is asked.
func corrected(facts *core.FactSet) (core.Model, error) {
	m := model.Derive(facts)
	rf, err := rules.Load(facts.Root)
	if err != nil {
		return m, err
	}
	ops, findings := rf.Compile(facts, m)
	if !findings.OK() {
		return m, fmt.Errorf("%s is not usable\n%s", rf.Path, findings.Error())
	}
	if len(ops) > 0 {
		var applied validate.Result
		if m, applied = validate.Apply(m, ops); !applied.OK() {
			return m, fmt.Errorf("rules.yaml produced an invalid model\n%s", applied.Error())
		}
	}
	if result := validate.Model(m); !result.OK() {
		return m, fmt.Errorf("model failed validation\n%s", result.Error())
	}
	return m, nil
}

// Tally is what a model holds, counted: what generate reports at its end, scan previews, and
// status repeats — said the same way by all three.
type Tally struct {
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

// Count counts what a model holds.
func Count(m Model) Tally {
	t := Tally{WithCode: len(m.Components()), WithTables: len(m.Datas()), Flows: len(m.Flows), Unresolved: len(m.Unresolved)}
	kind := map[string]core.Kind{}
	for _, n := range m.Nodes {
		kind[n.ID] = n.Kind
		switch {
		case n.Kind == core.Component:
			t.Components++
		case n.Kind == core.Table:
			t.Tables++
		case !n.Kind.Part():
			t.Elements++
		}
	}
	for _, e := range m.Edges {
		switch k := kind[e.From]; {
		case k == core.Component:
			t.Uses++
		case k == core.Table:
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

// Print says it: configuration's elements first, then what the code adds inside them.
func (t Tally) Print(out io.Writer, source string) {
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

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
