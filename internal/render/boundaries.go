package render

import (
	"sort"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// A group is one network boundary and what sits directly inside it. Groups nest, because
// network membership often does: a service on both `internal_network` and `external_network`
// belongs to a set that is contained by the set on `internal_network` alone.
type group struct {
	Name     string
	Internal bool
	Prov     archdoc.Provenance
	Nodes    []archdoc.Node
	Children []*group
}

// boundaries arranges the nodes inside the system into nested network groups (MDL-09).
//
// Network membership is the one boundary Compose states outright rather than implies: two
// services sharing no network cannot reach each other. Drawing it is therefore a fact on the
// diagram, not a layout choice.
//
// Returns nil when the networks cannot be nested — see overlapping below. The caller then draws
// a flat boundary, which is the honest fallback: a wrong grouping claims more than no grouping.
func boundaries(m archdoc.Model, inside []archdoc.Node) []*group {
	members := map[string]map[string]bool{}
	for _, n := range inside {
		for _, net := range n.Networks {
			if members[net] == nil {
				members[net] = map[string]bool{}
			}
			members[net][n.ID] = true
		}
	}
	if len(members) == 0 {
		return nil
	}

	names := make([]string, 0, len(members))
	for k := range members {
		names = append(names, k)
	}
	sort.Strings(names) // Go randomises map iteration; AC-7

	// Two networks that share members without one containing the other cannot be drawn as
	// nested boxes, and flattening them into one would erase the distinction the file drew.
	// Better to draw no boundary than the wrong one.
	for i, a := range names {
		for _, b := range names[i+1:] {
			if overlapping(members[a], members[b]) {
				return nil
			}
		}
	}

	declared := map[string]archdoc.Network{}
	for _, n := range m.Networks {
		declared[n.Name] = n
	}

	groups := map[string]*group{}
	for _, name := range names {
		net := declared[name]
		groups[name] = &group{Name: name, Internal: net.Internal, Prov: net.Prov}
	}

	// Each node lands in the smallest network that contains it; each network nests inside the
	// smallest one that strictly contains it.
	for _, n := range inside {
		if net := smallest(n.Networks, members); net != "" {
			groups[net].Nodes = append(groups[net].Nodes, n)
		}
	}

	var roots []*group
	for _, name := range names {
		parent := ""
		for _, other := range names {
			if other == name || !subset(members[name], members[other]) {
				continue
			}
			if parent == "" || len(members[other]) < len(members[parent]) {
				parent = other
			}
		}
		if parent == "" {
			roots = append(roots, groups[name])
			continue
		}
		groups[parent].Children = append(groups[parent].Children, groups[name])
	}

	return roots
}

// ungrouped returns the nodes that belong to no network at all. They sit directly in the system
// boundary — a file may declare networks and still leave a service off all of them.
func ungrouped(inside []archdoc.Node) []archdoc.Node {
	var out []archdoc.Node
	for _, n := range inside {
		if len(n.Networks) == 0 {
			out = append(out, n)
		}
	}
	return out
}

// smallest returns the network with the fewest members, which is the most specific boundary a
// node sits in.
func smallest(nets []string, members map[string]map[string]bool) string {
	best := ""
	for _, n := range nets {
		if members[n] == nil {
			continue
		}
		if best == "" || len(members[n]) < len(members[best]) ||
			(len(members[n]) == len(members[best]) && n < best) {
			best = n
		}
	}
	return best
}

// subset reports whether a is strictly contained in b.
func subset(a, b map[string]bool) bool {
	if len(a) >= len(b) {
		return false
	}
	for id := range a {
		if !b[id] {
			return false
		}
	}
	return true
}

// overlapping reports whether two sets share a member while neither contains the other.
func overlapping(a, b map[string]bool) bool {
	if subset(a, b) || subset(b, a) || len(a) == len(b) && sameSet(a, b) {
		return false
	}
	for id := range a {
		if b[id] {
			return true
		}
	}
	return false
}

func sameSet(a, b map[string]bool) bool {
	for id := range a {
		if !b[id] {
			return false
		}
	}
	return true
}
