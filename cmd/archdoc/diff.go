package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/rules"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// diff is F-37: what changed in the architecture between two commits, or between a commit and
// the files as they are now — every change classified (MEM-05), whether or not archdoc was ever
// run at either. Each side is read the way generate reads it, corrections included and no model
// asked, so the comparison is of what the repository states and nothing else.
func diff(e env, args []string) error {
	fs := flags("diff")
	asJSON := fs.Bool("json", false, "print the comparison as JSON")
	detail := fs.Bool("detail", false, "list every change by its class and id, after the summary")

	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	if len(positional) < 2 || len(positional) > 3 {
		return usagef("archdoc diff <path> <commit> [<commit>] — one commit compares it with the files as they are")
	}
	out := e.out
	root, from := positional[0], positional[1]

	before, err := modelAt(root, from)
	if err != nil {
		return err
	}
	to := "the working tree"
	var after archdoc.Model
	if len(positional) == 3 {
		to = positional[2]
		after, err = modelAt(root, to)
	} else {
		after, err = modelOf(root)
	}
	if err != nil {
		return err
	}

	d := model.Compare(before, after)
	drift := d.Drift()
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			From  string        `json:"from"`
			To    string        `json:"to"`
			Drift []model.Drift `json:"drift"`
			Diff  model.Diff    `json:"diff"`
		}{from, to, append([]model.Drift{}, drift...), d})
	}

	fmt.Fprintf(out, "%s → %s\n\n", from, to)
	if len(drift) == 0 {
		fmt.Fprintln(out, "no change to the architecture")
		return nil
	}
	structural, words := 0, 0
	for _, r := range changeSummary(d, before, after) {
		if r.words {
			words++
		} else {
			structural++
		}
		fmt.Fprintf(out, "%s %s\n", r.mark, r.text)
	}
	fmt.Fprintf(out, "\n%d change(s) to the structure, %d to wording only", structural, words)
	if !*detail {
		fmt.Fprintln(out, " — --detail lists every one by class and id")
		return nil
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out)
	counts := map[string]int{}
	for _, c := range drift {
		counts[c.Class]++
		fmt.Fprintf(out, "%-17s %s", c.Class, c.Element)
		if c.Detail != "" {
			fmt.Fprintf(out, " — %s", c.Detail)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out)
	for _, class := range []string{model.Added, model.Removed, model.Renamed, model.Rebounded, model.ProtocolChanged, model.Changed} {
		if counts[class] > 0 {
			fmt.Fprintf(out, "%d %s  ", counts[class], class)
		}
	}
	fmt.Fprintln(out)
	return nil
}

// wordFields are changes to what something is called or said to be, not to what it is.
var wordFields = map[string]bool{"description": true, "label": true, "name": true}

type changeRow struct {
	mark, text string
	words      bool
}

// changeSummary says what changed the way a reader would, and the way the app's Overview says it:
// the system's own elements one by one, by name; the parts inside a container counted under it;
// uses between parts and ways in counted; rewording kept apart and marked as such.
func changeSummary(d model.Diff, before, after archdoc.Model) []changeRow {
	name := func(id string) string {
		if n, ok := after.Node(id); ok {
			return n.Name
		}
		if n, ok := before.Node(id); ok {
			return n.Name
		}
		return id
	}
	part := func(id string) bool {
		if n, ok := after.Node(id); ok {
			return n.Kind.Part()
		}
		n, ok := before.Node(id)
		return ok && n.Kind.Part()
	}
	var rows []changeRow
	moved := map[string]bool{}
	for _, m := range d.Moved {
		moved[m.To] = true
		if m.Class == model.Renamed {
			rows = append(rows, changeRow{"→", m.Was + " renamed to " + m.Is, false})
		} else {
			rows = append(rows, changeRow{"→", m.Is + " crossed the system boundary", false})
		}
	}
	counted := func(nodes []archdoc.Node, verb, mark string) {
		byParent := map[string][]string{}
		var parents []string
		for _, n := range nodes {
			if !n.Kind.Part() {
				rows = append(rows, changeRow{mark, n.Name + " " + verb, false})
				continue
			}
			if _, seen := byParent[n.Parent]; !seen {
				parents = append(parents, n.Parent)
			}
			byParent[n.Parent] = append(byParent[n.Parent], n.Name)
		}
		for _, p := range parents {
			names := byParent[p]
			if len(names) == 1 {
				rows = append(rows, changeRow{mark, fmt.Sprintf("%s %s in %s", names[0], verb, name(p)), false})
				continue
			}
			shown := strings.Join(names[:min(3, len(names))], ", ")
			if len(names) > 3 {
				shown += fmt.Sprintf(" and %d more", len(names)-3)
			}
			rows = append(rows, changeRow{mark, fmt.Sprintf("%d parts %s in %s — %s", len(names), verb, name(p), shown), false})
		}
	}
	counted(d.AddedNodes, "appeared", "+")
	counted(d.RemovedNodes, "removed", "−")
	inner := 0
	for _, e := range d.AddedEdges {
		if part(e.From) || part(e.To) {
			inner++
			continue
		}
		rows = append(rows, changeRow{"+", name(e.From) + " → " + name(e.To), false})
	}
	for _, e := range d.RemovedEdges {
		if part(e.From) || part(e.To) {
			inner++
			continue
		}
		rows = append(rows, changeRow{"−", name(e.From) + " → " + name(e.To) + " removed", false})
	}
	if inner > 0 {
		rows = append(rows, changeRow{"±", fmt.Sprintf("%d %s between parts rewired", inner, plural(inner, "use", "uses")), false})
	}
	if a, r := len(d.AddedEntries), len(d.RemovedEntries); a+r > 0 {
		rows = append(rows, changeRow{"±", fmt.Sprintf("%d ways in appeared, %d disappeared", a, r), false})
	}
	columns := 0
	for _, c := range d.Changed {
		if strings.HasPrefix(c.Field, "column ") {
			columns++
			continue
		}
		if c.Field == "name" && moved[c.Element] {
			continue // said once already, as the rename
		}
		if wordFields[c.Field] {
			rows = append(rows, changeRow{" ", fmt.Sprintf("%s: %s reworded (words only)", name(c.Element), c.Field), true})
			continue
		}
		rows = append(rows, changeRow{"±", fmt.Sprintf("%s: %s %s → %s", name(c.Element), c.Field, orNothing(c.Before), orNothing(c.After)), false})
	}
	if columns > 0 {
		rows = append(rows, changeRow{"±", fmt.Sprintf("%d %s changed", columns, plural(columns, "column", "columns")), false})
	}
	return rows
}

func orNothing(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// modelAt is the model of a repository as it was at a revision.
func modelAt(root, rev string) (archdoc.Model, error) {
	dir, remove, err := store.Tree(root, rev)
	if err != nil {
		return archdoc.Model{}, err
	}
	defer remove()
	m, err := modelOf(dir)
	if err != nil {
		return m, fmt.Errorf("at %s: %w", rev, err)
	}
	return m, nil
}

// modelOf is the model generate would write for a directory: extracted, corrected by its rules,
// validated. No model is asked, and nothing is written.
func modelOf(root string) (archdoc.Model, error) {
	facts, err := extract.Scan(root)
	if err != nil {
		return archdoc.Model{}, err
	}
	if facts.Source == "" && !hasContainerApp(facts.Apps) {
		return archdoc.Model{}, fmt.Errorf("nothing to document: no Compose file, no application manifest, and no Python, JavaScript or HTML files")
	}
	return corrected(facts)
}

// corrected is the model of what extraction found, with the repository's rules applied and
// validated: what generate would write before any model is asked.
func corrected(facts *archdoc.FactSet) (archdoc.Model, error) {
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
