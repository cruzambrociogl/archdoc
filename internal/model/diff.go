package model

import (
	"sort"
	"strings"

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
	// What the code declares: routes and pages that appeared or disappeared (F-13). A table's
	// columns are values of the table, and change under Changed.
	AddedEntries   []archdoc.Entry `json:"added_entries"`
	RemovedEntries []archdoc.Entry `json:"removed_entries"`
	// Moved are elements that are the same thing under another identity: renamed, or carried
	// across the system boundary (MEM-05, MEM-06). They are not also listed as removed and added.
	Moved []Move `json:"moved,omitempty"`
}

// Move is one element whose identity changed while what it is did not.
type Move struct {
	Class string `json:"class"` // Renamed or Rebounded
	From  string `json:"from"`  // its ID before
	To    string `json:"to"`    // its ID after
	Was   string `json:"was"`   // its name before
	Is    string `json:"is"`
	// Why is the evidence the two were matched on, in words.
	Why string `json:"why"`
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
	return len(d.AddedNodes)+len(d.RemovedNodes)+len(d.AddedEdges)+len(d.RemovedEdges)+
		len(d.AddedEntries)+len(d.RemovedEntries)+len(d.Moved) > 0
}

// Empty reports whether the two versions are the same in every respect this compares.
func (d Diff) Empty() bool { return !d.Structural() && len(d.Changed) == 0 }

// Compare returns what changed from a to b. Identity is the element ID (MDL-13). Where an ID
// itself changed — a service renamed in its Compose file, a table renamed, a service that left the
// repository and is now only referred to — the two are matched on what stayed the same (MEM-06),
// reported once as moved, and everything attached to the element is compared under its new ID, so
// a rename is one change and not a removal, an addition and a rewiring.
func Compare(a, b archdoc.Model) Diff {
	moves, as := moved(a, b)
	d := compare(as, b)
	d.Moved = moves
	return d
}

func compare(a, b archdoc.Model) Diff {
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
			{"parent", old.Parent, n.Parent},
			{"evidence", string(old.Evidence), string(n.Evidence)},
		} {
			if f.was != f.is {
				d.Changed = append(d.Changed, Change{Element: n.ID, Field: f.field, Before: f.was, After: f.is})
			}
		}
		// A table's columns: one that appeared, disappeared, or changed what it is.
		if n.Kind == archdoc.Table {
			was := map[string]string{}
			for _, c := range old.Columns {
				was[c.Name] = columnText(c)
			}
			is := map[string]string{}
			for _, c := range n.Columns {
				is[c.Name] = columnText(c)
				if w, ok := was[c.Name]; !ok || w != is[c.Name] {
					d.Changed = append(d.Changed, Change{Element: n.ID, Field: "column " + c.Name, Before: w, After: is[c.Name]})
				}
			}
			for _, c := range old.Columns {
				if _, ok := is[c.Name]; !ok {
					d.Changed = append(d.Changed, Change{Element: n.ID, Field: "column " + c.Name, Before: was[c.Name]})
				}
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

	beforeEntry := map[string]bool{}
	for _, e := range a.Entries {
		beforeEntry[e.ID] = true
	}
	afterEntry := map[string]bool{}
	for _, e := range b.Entries {
		afterEntry[e.ID] = true
		if !beforeEntry[e.ID] {
			d.AddedEntries = append(d.AddedEntries, e)
		}
	}
	for _, e := range a.Entries {
		if !afterEntry[e.ID] {
			d.RemovedEntries = append(d.RemovedEntries, e)
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

// columnText is a column as a reader would say it: its type, and what marks it.
func columnText(c archdoc.Column) string {
	out := c.Type
	if c.Primary {
		out += " · primary key"
	}
	if c.Nullable {
		out += " · nullable"
	}
	if c.References != "" {
		out += " · references " + c.References
	}
	if out == "" {
		return "column"
	}
	return out
}

// moved finds the elements of a that are in b under another ID, and returns a rewritten so that
// they, and everything that names them, carry the ID they have in b.
//
// Two elements are the same when exactly one candidate on each side shares what identifies it:
// for a container its directory, or else its kind, technology and every relationship it has; for
// a table its columns; for a component its files. A match that is not unique is not made — two
// changes reported honestly beat one rename guessed.
func moved(a, b archdoc.Model) ([]Move, archdoc.Model) {
	inA, inB := map[string]bool{}, map[string]bool{}
	for _, n := range a.Nodes {
		inA[n.ID] = true
	}
	for _, n := range b.Nodes {
		inB[n.ID] = true
	}
	var gone, come []archdoc.Node
	for _, n := range a.Nodes {
		if !inB[n.ID] {
			gone = append(gone, n)
		}
	}
	for _, n := range b.Nodes {
		if !inA[n.ID] {
			come = append(come, n)
		}
	}
	if len(gone) == 0 || len(come) == 0 {
		return nil, a
	}

	to := map[string]string{} // ID in a → ID in b
	taken := map[string]bool{}
	var moves []Move
	pair := func(class string, sign func(n archdoc.Node, m archdoc.Model) (string, string)) {
		type side struct {
			nodes []archdoc.Node
			why   string
		}
		was, is := map[string]*side{}, map[string]*side{}
		collect := func(nodes []archdoc.Node, m archdoc.Model, into map[string]*side, skip func(string) bool) {
			for _, n := range nodes {
				if skip(n.ID) {
					continue
				}
				if key, why := sign(n, m); key != "" {
					if into[key] == nil {
						into[key] = &side{why: why}
					}
					into[key].nodes = append(into[key].nodes, n)
				}
			}
		}
		collect(gone, a, was, func(id string) bool { return to[id] != "" })
		collect(come, b, is, func(id string) bool { return taken[id] })
		for _, n := range gone { // in a's order, so the result does not depend on map order
			key, _ := sign(n, a)
			w, i := was[key], is[key]
			if key == "" || to[n.ID] != "" || w == nil || i == nil || len(w.nodes) != 1 || len(i.nodes) != 1 {
				continue
			}
			other := i.nodes[0]
			to[n.ID], taken[other.ID] = other.ID, true
			moves = append(moves, Move{Class: class, From: n.ID, To: other.ID, Was: n.Name, Is: other.Name, Why: w.why})
		}
	}

	// Across the boundary: the same name, declared on one side and only referred to on the other.
	pair(Rebounded, func(n archdoc.Node, _ archdoc.Model) (string, string) {
		if n.Kind.Part() {
			return "", ""
		}
		return strings.ToLower(n.Name), "the same name, inside the system in one and outside it in the other"
	})
	// The pairing above matches any two with one name; keep only those that changed sides.
	kept := moves[:0]
	for _, mv := range moves {
		before, _ := a.Node(mv.From)
		after, _ := b.Node(mv.To)
		if (before.Kind == archdoc.External) != (after.Kind == archdoc.External) {
			kept = append(kept, mv)
		} else {
			delete(to, mv.From)
			delete(taken, mv.To)
		}
	}
	moves = kept

	pair(Renamed, func(n archdoc.Node, m archdoc.Model) (string, string) {
		switch {
		case n.Kind == archdoc.Table:
			if len(n.Columns) == 0 {
				return "", ""
			}
			cols := make([]string, 0, len(n.Columns))
			for _, c := range n.Columns {
				ref := c.References
				c.References = ""
				if ref != "" {
					ref = " → a table"
				}
				cols = append(cols, c.Name+" "+columnText(c)+ref)
			}
			return "table\x00" + n.Parent + "\x00" + strings.Join(cols, "\x00"), "the same columns"
		case n.Kind.Part():
			if len(n.Files) == 0 {
				return "", ""
			}
			return string(n.Kind) + "\x00" + n.Parent + "\x00" + strings.Join(n.Files, "\x00"), "the same files"
		case n.Dir != "":
			return "dir\x00" + n.Dir, "the same directory, " + n.Dir
		}
		var ties []string
		for _, e := range m.Edges {
			switch n.ID {
			case e.From:
				ties = append(ties, "→ "+e.To+" "+e.Label+" "+e.Technology)
			case e.To:
				ties = append(ties, "← "+e.From+" "+e.Label+" "+e.Technology)
			}
		}
		if len(ties) == 0 {
			return "", "" // nothing to recognise it by
		}
		sort.Strings(ties)
		return string(n.Kind) + "\x00" + n.Technology + "\x00" + strings.Join(ties, "\x00"), "the same kind, technology and relationships"
	})

	// A container's parts and entries carry its name in their IDs, and follow it:
	// cmp:api/orders is cmp:backend/orders, route:api GET /x is route:backend GET /x.
	local := func(id string) string { _, l, _ := strings.Cut(id, ":"); return l }
	containers := map[string]string{} // a renamed container's name in IDs → its new one
	for from, dest := range to {
		if n, ok := a.Node(from); ok && !n.Kind.Part() && local(from) != "" && local(dest) != "" {
			containers[local(from)] = local(dest)
		}
	}
	follow := func(id string) string {
		kind, rest, ok := strings.Cut(id, ":")
		if !ok {
			return id
		}
		for _, sep := range []string{"/", " "} {
			if name, tail, ok := strings.Cut(rest, sep); ok {
				if next, renamed := containers[name]; renamed {
					return kind + ":" + next + sep + tail
				}
			}
		}
		return id
	}
	for _, n := range gone {
		if n.Kind.Part() && to[n.ID] == "" {
			if next := follow(n.ID); next != n.ID && inB[next] {
				to[n.ID] = next
			}
		}
	}
	if len(to) == 0 {
		return nil, a
	}

	id := func(s string) string {
		if next, ok := to[s]; ok {
			return next
		}
		return s
	}
	out := a
	out.Nodes = make([]archdoc.Node, len(a.Nodes))
	for i, n := range a.Nodes {
		n.ID, n.Parent = id(n.ID), id(n.Parent)
		if len(n.Columns) > 0 {
			cols := make([]archdoc.Column, len(n.Columns))
			for j, c := range n.Columns {
				c.References = id(c.References)
				cols[j] = c
			}
			n.Columns = cols
		}
		out.Nodes[i] = n
	}
	out.Edges = make([]archdoc.Edge, len(a.Edges))
	for i, e := range a.Edges {
		e.From, e.To = id(e.From), id(e.To)
		out.Edges[i] = e
	}
	out.Entries = make([]archdoc.Entry, len(a.Entries))
	for i, e := range a.Entries {
		e.ID = follow(e.ID)
		e.Container, e.Component = id(e.Container), id(e.Component)
		out.Entries[i] = e
	}
	sort.SliceStable(moves, func(i, j int) bool { return moves[i].From < moves[j].From })
	return moves, out
}

// The classes of change MEM-05 names.
const (
	Added           = "added"
	Removed         = "removed"
	Renamed         = "renamed"
	Rebounded       = "re-bounded"
	ProtocolChanged = "protocol-changed"
	Changed         = "changed"
)

// Drift is one change, classified (MEM-05): what kind of change, to what, and the detail a reader
// needs to recognise it.
type Drift struct {
	Class   string `json:"class"`
	Element string `json:"element"`
	Detail  string `json:"detail,omitempty"`
}

// Drift lists every change in the diff under its class, in a fixed order: by class as MEM-05
// lists them, then by element.
func (d Diff) Drift() []Drift {
	var out []Drift
	edge := func(e archdoc.Edge) string { return e.From + " → " + e.To }
	for _, n := range d.AddedNodes {
		out = append(out, Drift{Added, n.ID, string(n.Kind) + " " + n.Name})
	}
	for _, e := range d.AddedEdges {
		out = append(out, Drift{Added, edge(e), e.Label})
	}
	for _, e := range d.AddedEntries {
		out = append(out, Drift{Added, e.ID, e.Handler})
	}
	for _, n := range d.RemovedNodes {
		out = append(out, Drift{Removed, n.ID, string(n.Kind) + " " + n.Name})
	}
	for _, e := range d.RemovedEdges {
		out = append(out, Drift{Removed, edge(e), e.Label})
	}
	for _, e := range d.RemovedEntries {
		out = append(out, Drift{Removed, e.ID, e.Handler})
	}
	moved, crossed := map[string]bool{}, map[string]bool{}
	for _, m := range d.Moved {
		moved[m.To] = true
		crossed[m.To] = m.Class == Rebounded
		out = append(out, Drift{m.Class, m.To, "was " + m.From + " — " + m.Why})
	}
	for _, c := range d.Changed {
		detail := c.Field + ": " + orNothing(c.Before) + " → " + orNothing(c.After)
		switch {
		case c.Field == "name" && moved[c.Element]:
			// said once already, as the move
		case crossed[c.Element] && (c.Field == "kind" || c.Field == "evidence" || c.Field == "technology"):
			// what crossing the boundary is: outside, an element is external, only referred to,
			// and nothing declares what it runs
		case c.Field == "name":
			out = append(out, Drift{Renamed, c.Element, detail})
		case c.Field == "parent" || c.Field == "evidence":
			out = append(out, Drift{Rebounded, c.Element, detail})
		case c.Field == "technology" && strings.Contains(c.Element, " → "):
			out = append(out, Drift{ProtocolChanged, c.Element, detail})
		default:
			out = append(out, Drift{Changed, c.Element, detail})
		}
	}
	order := map[string]int{Added: 0, Removed: 1, Renamed: 2, Rebounded: 3, ProtocolChanged: 4, Changed: 5}
	sort.SliceStable(out, func(i, j int) bool {
		if order[out[i].Class] != order[out[j].Class] {
			return order[out[i].Class] < order[out[j].Class]
		}
		return out[i].Element < out[j].Element
	})
	return out
}

func orNothing(s string) string {
	if s == "" {
		return "nothing"
	}
	return s
}
