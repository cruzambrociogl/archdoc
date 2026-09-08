package validate

import (
	"fmt"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Ops checks a set of operations against the model they would be applied to, without applying
// anything.
//
// Operations are checked *before* they run rather than after, because the alternative is
// applying half of them and discovering the problem with a corrupted model in hand. That is what
// VAL-08 forbids.
func Ops(m archdoc.Model, ops []archdoc.Op) Result {
	var r Result

	nodes := make(map[string]bool, len(m.Nodes))
	for _, n := range m.Nodes {
		nodes[n.ID] = true
	}

	edges := make(map[string]bool, len(m.Edges))
	for _, e := range m.Edges {
		edges[e.From+"\x00"+e.To] = true
	}

	for i, op := range ops {
		id := fmt.Sprintf("op[%d] %s", i, op.Kind)

		// Hard rule 4, checked here rather than trusted. The semantic layer may describe
		// what extraction found; it may never change what exists.
		if !op.Kind.AllowedFrom(op.Origin) {
			r.add("SEM-10", Error, id,
				fmt.Sprintf("%q may not produce a structural operation", op.Origin), op.Prov)
			continue
		}

		// An operation is a claim like any other, and P1 does not exempt it.
		if !op.Prov.Known() {
			r.add("VAL-05", Error, id, "operation has no provenance", op.Prov)
		}

		switch op.Kind {
		case archdoc.AddEdge:
			requireNode(&r, nodes, id, op.Target, op)
			requireNode(&r, nodes, id, op.To, op)
			if op.Target == op.To {
				r.add("VAL-02", Error, id, "edge would point at itself", op.Prov)
			}
			if edges[op.Target+"\x00"+op.To] {
				r.add("VAL-02", Warning, id, "edge already exists", op.Prov)
			}

		case archdoc.RemoveEdge:
			if !edges[op.Target+"\x00"+op.To] {
				r.add("VAL-02", Error, id,
					fmt.Sprintf("no edge %s → %s to remove", op.Target, op.To), op.Prov)
			}

		case archdoc.Exclude:
			// Names an element and nothing else — there is no value to supply.
			requireNode(&r, nodes, id, op.Target, op)

		case archdoc.SetKind:
			requireNode(&r, nodes, id, op.Target, op)
			if !kinds[archdoc.Kind(op.Value)] {
				r.add("VAL-01", Error, id, fmt.Sprintf("unknown kind %q", op.Value), op.Prov)
			}

		case archdoc.SetEdgeLabel:
			if !edges[op.Target+"\x00"+op.To] {
				r.add("VAL-02", Error, id,
					fmt.Sprintf("no edge %s → %s to label", op.Target, op.To), op.Prov)
			}
			requireValue(&r, id, op)

		default:
			requireNode(&r, nodes, id, op.Target, op)
			requireValue(&r, id, op)
		}
	}

	r.sort()
	return r
}

// requireNode is VAL-02 for operations: an operation on an element that does not exist is the
// same defect as an edge to one, arriving a step earlier.
func requireNode(r *Result, nodes map[string]bool, id, target string, op archdoc.Op) {
	if target == "" {
		r.add("VAL-01", Error, id, "operation names no target", op.Prov)
		return
	}
	if !nodes[target] {
		r.add("VAL-02", Error, id, fmt.Sprintf("no such element %q", target), op.Prov)
	}
}

func requireValue(r *Result, id string, op archdoc.Op) {
	if op.Value == "" {
		r.add("VAL-01", Error, id, "operation has no value", op.Prov)
	}
}

// Apply validates a set of operations and, if every one of them passes, applies them all.
//
// All or nothing. VAL-08 says never write a partial model, and the cheapest way to honour that
// is to make a partial model impossible to produce: on any error the original is returned
// untouched, along with the findings that explain why.
//
// The returned model is validated again afterwards. Operations are individually legal and can
// still combine into something wrong — removing the last edge into a node, say — and the point
// of a model is that it is checked as a whole.
func Apply(m archdoc.Model, ops []archdoc.Op) (archdoc.Model, Result) {
	if r := Ops(m, ops); !r.OK() {
		return m, r
	}

	out := clone(m)
	for _, op := range ops {
		apply(&out, op)
	}
	out = out.Normalise()

	if r := Model(out); !r.OK() {
		return m, r // the combination broke something the operations did not, individually
	}
	return out, Model(out)
}

func apply(m *archdoc.Model, op archdoc.Op) {
	switch op.Kind {
	case archdoc.AddEdge:
		m.Edges = append(m.Edges, archdoc.Edge{
			From: op.Target, To: op.To, Label: op.Value,
			Traffic: true, Prov: []archdoc.Provenance{op.Prov},
		})
		return

	case archdoc.RemoveEdge, archdoc.SetEdgeLabel:
		for i := range m.Edges {
			if m.Edges[i].From != op.Target || m.Edges[i].To != op.To {
				continue
			}
			if op.Kind == archdoc.SetEdgeLabel {
				m.Edges[i].Label = op.Value
				m.Edges[i].Prov = append(m.Edges[i].Prov, op.Prov)
				continue
			}
			m.Edges = append(m.Edges[:i], m.Edges[i+1:]...)
			return
		}
		return

	case archdoc.Exclude:
		for i := range m.Nodes {
			if m.Nodes[i].ID == op.Target {
				m.Nodes = append(m.Nodes[:i], m.Nodes[i+1:]...)
				break
			}
		}
		// An excluded node's edges would dangle, and VAL-02 would reject the result.
		kept := m.Edges[:0]
		for _, e := range m.Edges {
			if e.From != op.Target && e.To != op.Target {
				kept = append(kept, e)
			}
		}
		m.Edges = kept
		return
	}

	for i := range m.Nodes {
		if m.Nodes[i].ID != op.Target {
			continue
		}
		n := &m.Nodes[i]

		// Every applied operation stamps its own provenance onto what it changed. That is
		// how a description ends up traceable to the run that wrote it, and it is what
		// PRV-05 renders.
		switch op.Kind {
		case archdoc.SetName:
			n.Name = op.Value
		case archdoc.SetDescription:
			n.Description, n.DescProv = op.Value, op.Prov
		case archdoc.SetTechnology:
			n.Technology, n.TechProv = op.Value, op.Prov
		case archdoc.SetKind:
			n.Kind = archdoc.Kind(op.Value)
		case archdoc.Group:
			n.Parent = op.Value
		}
		return
	}
}

// clone copies deeply enough that a rejected Apply leaves the caller's model untouched. The
// slices inside a node and an edge are shared otherwise, and a partial application would be
// visible in the original — which is the exact failure VAL-08 exists to prevent.
func clone(m archdoc.Model) archdoc.Model {
	out := m
	out.Nodes = append([]archdoc.Node(nil), m.Nodes...)
	out.Edges = append([]archdoc.Edge(nil), m.Edges...)

	for i := range out.Nodes {
		out.Nodes[i].Networks = append([]string(nil), m.Nodes[i].Networks...)
	}
	for i := range out.Edges {
		out.Edges[i].Prov = append([]archdoc.Provenance(nil), m.Edges[i].Prov...)
	}
	return out
}
