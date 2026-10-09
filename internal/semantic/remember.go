package semantic

import (
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Labels are remembered (F-30) as explanations are: a description a model wrote is kept beside what
// it was written about, and applied again on every run for as long as that has not changed — so a
// run that asks nothing still shows it, the documents do not flicker between labelled and bare,
// and nothing is paid for twice.
//
// What a label was written about is its basis: for an element its kind, its name and its
// technology as extracted; for a relationship its two ends and its protocol. Change any of those
// and the label is no longer about this element as it is; it is dropped from what is applied, and
// the next --label writes a new one.

// RememberedLabel is one remembered operation and the basis it was written on.
type RememberedLabel struct {
	Op    archdoc.Op `json:"op"`
	Basis string     `json:"basis"`
}

// Labels are every remembered label, in a fixed order.
type Labels []RememberedLabel

func labelKey(op archdoc.Op) string { return string(op.Kind) + "\x00" + op.Target + "\x00" + op.To }

// basis is what an operation's target looks like in a model, as extracted.
func basis(m archdoc.Model, op archdoc.Op) (string, bool) {
	if op.To != "" {
		for _, e := range m.Edges {
			if e.From == op.Target && e.To == op.To {
				return "edge\x00" + e.From + "\x00" + e.To + "\x00" + e.Technology, true
			}
		}
		return "", false
	}
	if op.Target == archdoc.SystemID {
		// The system is what it contains: described again when a container comes or goes.
		var inside []string
		for _, n := range m.Nodes {
			if !n.Kind.Part() && n.Evidence == archdoc.Declared && n.Kind != archdoc.Actor {
				inside = append(inside, n.ID)
			}
		}
		sort.Strings(inside)
		return "system\x00" + m.Name + "\x00" + strings.Join(inside, "\x00"), true
	}
	n, ok := m.Node(op.Target)
	if !ok {
		return "", false
	}
	return "node\x00" + string(n.Kind) + "\x00" + n.Name + "\x00" + n.Technology, true
}

// Remember records the operations a labelling run was given credit for against the model they
// were written about — the model before they were applied — over what was remembered before. A
// newer label for the same value replaces the older; nothing else is dropped.
func Remember(before Labels, extracted archdoc.Model, ops []archdoc.Op) Labels {
	byKey := map[string]RememberedLabel{}
	for _, l := range before {
		byKey[labelKey(l.Op)] = l
	}
	for _, op := range ops {
		if b, ok := basis(extracted, op); ok {
			byKey[labelKey(op)] = RememberedLabel{Op: op, Basis: b}
		}
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(Labels, 0, len(keys))
	for _, k := range keys {
		out = append(out, byKey[k])
	}
	return out
}

// Recall returns the remembered operations that still apply to a model as extracted: the ones
// whose target is there and is what it was when the label was written.
func Recall(labels Labels, extracted archdoc.Model) []archdoc.Op {
	var out []archdoc.Op
	for _, l := range labels {
		if b, ok := basis(extracted, l.Op); ok && b == l.Basis {
			out = append(out, l.Op)
		}
	}
	return out
}
