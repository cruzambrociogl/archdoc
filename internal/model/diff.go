package model

import (
	"sort"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Diff is what changed between two versions of an architecture (MEM-04, MEM-05 in part).
//
// Structural changes — an element or relationship appearing or disappearing — are kept apart from
// wording changes, because they answer different questions. "A service appeared" is news about
// the system; "a description was reworded" is news about the documentation, and a reader
// scanning for the first should not have to wade through the second.
type Diff struct {
	AddedNodes   []archdoc.Node `json:"added_nodes"`
	RemovedNodes []archdoc.Node `json:"removed_nodes"`
	AddedEdges   []archdoc.Edge `json:"added_edges"`
	RemovedEdges []archdoc.Edge `json:"removed_edges"`
	Changed      []Change       `json:"changed"`
}

// Change is one field of one element or relationship that differs between the versions.
type Change struct {
	Element string `json:"element"` // node id, or "from → to" for a relationship
	Field   string `json:"field"`
	Before  string `json:"before"`
	After   string `json:"after"`
}

// Structural reports whether anything appeared or disappeared, as opposed to only being reworded.
func (d Diff) Structural() bool {
	return len(d.AddedNodes)+len(d.RemovedNodes)+len(d.AddedEdges)+len(d.RemovedEdges) > 0
}

// Empty reports whether the two versions are the same in every respect this compares.
func (d Diff) Empty() bool { return !d.Structural() && len(d.Changed) == 0 }

// Compare returns what changed from a to b. Identity is the element ID (MDL-13), so a renamed
// service is a changed name rather than a removal and an addition — which is only as good as ID
// stability; MEM-06's rename detection across *changed* IDs is not attempted here.
func Compare(a, b archdoc.Model) Diff {
	var d Diff

	before := map[string]archdoc.Node{}
	for _, n := range a.Nodes {
		before[n.ID] = n
	}
	after := map[string]archdoc.Node{}
	for _, n := range b.Nodes {
		after[n.ID] = n
	}

	for _, n := range b.Nodes {
		old, ok := before[n.ID]
		if !ok {
			d.AddedNodes = append(d.AddedNodes, n)
			continue
		}
		for _, f := range []struct{ field, was, is string }{
			{"name", old.Name, n.Name},
			{"kind", string(old.Kind), string(n.Kind)},
			{"technology", old.Technology, n.Technology},
			{"description", old.Description, n.Description},
		} {
			if f.was != f.is {
				d.Changed = append(d.Changed, Change{Element: n.ID, Field: f.field, Before: f.was, After: f.is})
			}
		}
	}
	for _, n := range a.Nodes {
		if _, ok := after[n.ID]; !ok {
			d.RemovedNodes = append(d.RemovedNodes, n)
		}
	}

	key := func(e archdoc.Edge) string { return e.From + " → " + e.To }
	beforeE := map[string]archdoc.Edge{}
	for _, e := range a.Edges {
		beforeE[key(e)] = e
	}
	afterE := map[string]archdoc.Edge{}
	for _, e := range b.Edges {
		afterE[key(e)] = e
	}
	for _, e := range b.Edges {
		old, ok := beforeE[key(e)]
		if !ok {
			d.AddedEdges = append(d.AddedEdges, e)
			continue
		}
		if old.Label != e.Label {
			d.Changed = append(d.Changed, Change{Element: key(e), Field: "label", Before: old.Label, After: e.Label})
		}
		if old.Technology != e.Technology {
			d.Changed = append(d.Changed, Change{Element: key(e), Field: "technology", Before: old.Technology, After: e.Technology})
		}
	}
	for _, e := range a.Edges {
		if _, ok := afterE[key(e)]; !ok {
			d.RemovedEdges = append(d.RemovedEdges, e)
		}
	}

	// Deterministic, like everything that reaches a reader.
	sort.SliceStable(d.Changed, func(i, j int) bool {
		if d.Changed[i].Element != d.Changed[j].Element {
			return d.Changed[i].Element < d.Changed[j].Element
		}
		return d.Changed[i].Field < d.Changed[j].Field
	})
	return d
}
