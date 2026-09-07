package extract

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Positions are the second of the two passes O-8 settled on. compose-go resolves what is true
// and discards where it was written; this reads the raw document for the where, and the two are
// reconciled by name.
//
// Only the keys the model actually uses are recorded. A generic path index would be more
// general and would spend memory on the ninety-odd Compose fields nothing downstream reads.

// servicePos is where one service and its interesting sub-keys were written.
type servicePos struct {
	Decl      archdoc.Provenance
	DependsOn map[string]archdoc.Provenance
	Env       map[string]archdoc.Provenance
	Nets      map[string]archdoc.Provenance
	Ports     []archdoc.Provenance
	Vols      []archdoc.Provenance
}

// at returns the position of a named sub-key, falling back to the service declaration.
//
// The fallback is not a guess. A fact that came from inside a service block but could not be
// located more precisely is still genuinely attested by that block, and the reader is sent to
// the right service. It matters because P1 forbids emitting anything whose provenance is
// unknown — without a fallback, an unlocatable fact would have to be dropped, which loses more
// than it protects.
func (s servicePos) at(m map[string]archdoc.Provenance, key string) archdoc.Provenance {
	if p, ok := m[key]; ok && p.Known() {
		return p
	}
	return s.Decl
}

func (s servicePos) dependency(name string) archdoc.Provenance { return s.at(s.DependsOn, name) }
func (s servicePos) env(name string) archdoc.Provenance        { return s.at(s.Env, name) }
func (s servicePos) network(name string) archdoc.Provenance    { return s.at(s.Nets, name) }

func (s servicePos) port(i int) archdoc.Provenance  { return s.indexed(s.Ports, i) }
func (s servicePos) mount(i int) archdoc.Provenance { return s.indexed(s.Vols, i) }

// indexed locates an entry in a list that has no name of its own — ports and volumes are both
// positional. Falls back to the service declaration, for the reason given on at().
func (s servicePos) indexed(ps []archdoc.Provenance, i int) archdoc.Provenance {
	if i < len(ps) && ps[i].Known() {
		return ps[i]
	}
	return s.Decl
}

// readPositions walks the raw document and records where each service and its sub-keys appear.
func readPositions(content []byte, rel string) map[string]servicePos {
	out := map[string]servicePos{}

	var doc yaml.Node
	if err := yaml.Unmarshal(stripComposeTags(content), &doc); err != nil {
		return out
	}

	services := lookup(&doc, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return out
	}

	for i := 0; i+1 < len(services.Content); i += 2 {
		key, body := services.Content[i], services.Content[i+1]

		pos := servicePos{
			Decl:      provOf(rel, key),
			DependsOn: map[string]archdoc.Provenance{},
			Env:       map[string]archdoc.Provenance{},
			Nets:      map[string]archdoc.Provenance{},
		}

		if body.Kind == yaml.MappingNode {
			// depends_on and environment each have two shapes in the wild — a list of
			// scalars, or a mapping. Compose accepts both, so both must be located.
			readNames(rel, lookup(body, "depends_on"), pos.DependsOn)
			readNames(rel, lookup(body, "environment"), pos.Env)
			readNames(rel, lookup(body, "networks"), pos.Nets)
			pos.Ports = readSequence(rel, lookup(body, "ports"))
			pos.Vols = readSequence(rel, lookup(body, "volumes"))
		}

		out[key.Value] = pos
	}

	return out
}

// readNames records the position of every name in a node that is either a mapping (position of
// the key) or a sequence of scalars (position of the item). A sequence entry may be "K=V", in
// which case the name is the part before the first "=".
func readNames(rel string, n *yaml.Node, into map[string]archdoc.Provenance) {
	if n == nil {
		return
	}

	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			into[n.Content[i].Value] = provOf(rel, n.Content[i])
		}
	case yaml.SequenceNode:
		for _, item := range n.Content {
			if item.Kind != yaml.ScalarNode {
				continue
			}
			name := item.Value
			if k, _, ok := strings.Cut(name, "="); ok {
				name = k
			}
			into[strings.TrimSpace(name)] = provOf(rel, item)
		}
	}
}

// readSequence records positions by index, for lists whose entries have no name of their own.
func readSequence(rel string, n *yaml.Node) []archdoc.Provenance {
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}

	out := make([]archdoc.Provenance, 0, len(n.Content))
	for _, item := range n.Content {
		out = append(out, provOf(rel, item))
	}
	return out
}

func provOf(rel string, n *yaml.Node) archdoc.Provenance {
	return archdoc.Provenance{File: rel, Line: n.Line, Column: n.Column}
}
