package archdoc

import (
	"context"
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// Comparison is what changed in the architecture from one model to another (F-37): every change
// classified (MEM-05), and the two models it was read from.
type Comparison struct {
	// From and To name the two sides — commits, or "the working tree".
	From, To string
	Before   Model
	After    Model
	Delta    Delta
	Drift    []Drift
}

// Compare compares two models. It needs no repository and no git: two stored models are enough.
func Compare(before, after Model) *Comparison {
	d := model.Compare(before, after)
	return &Comparison{Before: before, After: after, Delta: d, Drift: d.Drift()}
}

// Diff compares a repository at two commits — or at one, and as its files are now, when to is
// empty — whether or not archdoc was ever run at either. Each side is read the way Generate reads
// it, corrections included and no model asked, so the comparison is of what the repository states
// and nothing else. Needs git.
func Diff(ctx context.Context, root, from, to string) (*Comparison, error) {
	before, err := modelAt(root, from)
	if err != nil {
		return nil, err
	}
	label := "the working tree"
	var after core.Model
	if to != "" {
		label = to
		after, err = modelAt(root, to)
	} else {
		after, err = ReadModel(root)
	}
	if err != nil {
		return nil, err
	}
	c := Compare(before, after)
	c.From, c.To = from, label
	return c, nil
}

// modelAt is the model of a repository as it was at a revision.
func modelAt(root, rev string) (core.Model, error) {
	dir, remove, err := store.Tree(root, rev)
	if err != nil {
		return core.Model{}, err
	}
	defer remove()
	m, err := ReadModel(dir)
	if err != nil {
		return m, fmt.Errorf("at %s: %w", rev, err)
	}
	return m, nil
}

// wordFields are changes to what something is called or said to be, not to what it is.
var wordFields = map[string]bool{"description": true, "label": true, "name": true}

// Structural counts what changed in what the system is, leaving out what it is called or said to
// be: a model's descriptions are in a committed model and not in a fresh reading.
func (c *Comparison) Structural() int {
	d := c.Delta
	n := len(d.AddedNodes) + len(d.RemovedNodes) + len(d.AddedEdges) + len(d.RemovedEdges) +
		len(d.AddedEntries) + len(d.RemovedEntries) + len(d.Moved)
	for _, ch := range d.Changed {
		if !wordFields[ch.Field] {
			n++
		}
	}
	return n
}

// ChangeLine is one change, said the way a reader would: a mark (+ − → ± or a space), the
// sentence, and whether it is wording only.
type ChangeLine struct {
	Mark, Text string
	Words      bool
}

// Summary says what changed the way a reader would, and the way the app's Overview says it: the
// system's own elements one by one, by name; the parts inside a container counted under it; uses
// between parts and ways in counted; rewording kept apart and marked as such.
func (c *Comparison) Summary() []ChangeLine {
	d, before, after := c.Delta, c.Before, c.After
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
	var rows []ChangeLine
	moved := map[string]bool{}
	for _, m := range d.Moved {
		moved[m.To] = true
		if m.Class == model.Renamed {
			rows = append(rows, ChangeLine{"→", m.Was + " renamed to " + m.Is, false})
		} else {
			rows = append(rows, ChangeLine{"→", m.Is + " crossed the system boundary", false})
		}
	}
	counted := func(nodes []core.Node, verb, mark string) {
		byParent := map[string][]string{}
		var parents []string
		for _, n := range nodes {
			if !n.Kind.Part() {
				rows = append(rows, ChangeLine{mark, n.Name + " " + verb, false})
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
				rows = append(rows, ChangeLine{mark, fmt.Sprintf("%s %s in %s", names[0], verb, name(p)), false})
				continue
			}
			shown := strings.Join(names[:min(3, len(names))], ", ")
			if len(names) > 3 {
				shown += fmt.Sprintf(" and %d more", len(names)-3)
			}
			rows = append(rows, ChangeLine{mark, fmt.Sprintf("%d parts %s in %s — %s", len(names), verb, name(p), shown), false})
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
		rows = append(rows, ChangeLine{"+", name(e.From) + " → " + name(e.To), false})
	}
	for _, e := range d.RemovedEdges {
		if part(e.From) || part(e.To) {
			inner++
			continue
		}
		rows = append(rows, ChangeLine{"−", name(e.From) + " → " + name(e.To) + " removed", false})
	}
	if inner > 0 {
		rows = append(rows, ChangeLine{"±", fmt.Sprintf("%d %s between parts rewired", inner, plural(inner, "use", "uses")), false})
	}
	if a, r := len(d.AddedEntries), len(d.RemovedEntries); a+r > 0 {
		rows = append(rows, ChangeLine{"±", fmt.Sprintf("%d ways in appeared, %d disappeared", a, r), false})
	}
	columns := 0
	for _, ch := range d.Changed {
		if strings.HasPrefix(ch.Field, "column ") {
			columns++
			continue
		}
		if ch.Field == "name" && moved[ch.Element] {
			continue // said once already, as the rename
		}
		if wordFields[ch.Field] {
			rows = append(rows, ChangeLine{" ", fmt.Sprintf("%s: %s reworded (words only)", name(ch.Element), ch.Field), true})
			continue
		}
		rows = append(rows, ChangeLine{"±", fmt.Sprintf("%s: %s %s → %s", name(ch.Element), ch.Field, orNothing(ch.Before), orNothing(ch.After)), false})
	}
	if columns > 0 {
		rows = append(rows, ChangeLine{"±", fmt.Sprintf("%d %s changed", columns, plural(columns, "column", "columns")), false})
	}
	return rows
}

func orNothing(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
