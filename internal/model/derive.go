// Package model turns facts into a graph.
//
// It is the seam where the two halves of the pipeline meet: everything above it reads files,
// everything below it draws. Nothing here touches the disk or the network, and nothing here
// invents a node — every node and every edge is built from a fact that already carried the line
// proving it.
package model

import (
	"sort"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// The actor is a single node rather than one per entry point. A published port proves that
// something outside reaches in; it does not say who, or that two ports mean two audiences.
const actorID = "actor:user"

// Derive builds the model from a FactSet. One model — the C4 levels are projections over it,
// not separate derivations.
func Derive(f *archdoc.FactSet) archdoc.Model {
	m := archdoc.Model{Name: f.Name, Source: f.Source}

	declared := make(map[string]bool, len(f.Services))
	for _, s := range f.Services {
		declared[s.Name] = true
	}

	for _, s := range f.Services {
		kind, tech := classify(s.Image)
		m.Nodes = append(m.Nodes, archdoc.Node{
			ID:         serviceID(s.Name),
			Name:       s.Name,
			Kind:       kind,
			Technology: tech,
			Evidence:   archdoc.Declared,
			Prov:       s.Prov,
		})
	}

	external := map[string]archdoc.Node{}
	var actor *archdoc.Node

	for _, s := range f.Services {
		from := serviceID(s.Name)

		// depends_on states a relationship without saying what flows along it. The label
		// stays dull on purpose: this is a deterministic fact, and making it read well is
		// the semantic layer's job, not extraction's.
		for _, d := range s.DependsOn {
			if !declared[d.Service] {
				continue // a dependency on something not in this file is not ours to draw
			}
			m.Edges = append(m.Edges, archdoc.Edge{
				From:  from,
				To:    serviceID(d.Service),
				Label: "depends on",
				Prov:  []archdoc.Provenance{d.Prov},
			})
		}

		for _, e := range s.Endpoints {
			to := serviceID(e.Host)

			// A host the repository does not declare is a system outside it. That is the
			// evidence rule doing double duty: referenced is also the C4 system boundary.
			if !declared[e.Host] {
				to = externalID(e.Host)
				if _, seen := external[to]; !seen {
					external[to] = archdoc.Node{
						ID:       to,
						Name:     e.Host,
						Kind:     archdoc.External,
						Evidence: archdoc.Referenced,
						Prov:     e.Prov,
					}
				}
			}

			m.Edges = append(m.Edges, archdoc.Edge{
				From:       from,
				To:         to,
				Label:      "connects to",
				Technology: e.Scheme,
				Prov:       []archdoc.Provenance{e.Prov},
			})
		}

		// A published port is declared evidence that something outside the system reaches
		// in. It is the only actor evidence configuration offers.
		for _, p := range s.Ports {
			if actor == nil {
				actor = &archdoc.Node{
					ID:       actorID,
					Name:     "User",
					Kind:     archdoc.Actor,
					Evidence: archdoc.Declared,
					Prov:     p.Prov,
				}
			}
			m.Edges = append(m.Edges, archdoc.Edge{
				From:  actorID,
				To:    from,
				Label: "reaches",
				Prov:  []archdoc.Provenance{p.Prov},
			})
		}
	}

	if actor != nil {
		m.Nodes = append(m.Nodes, *actor)
	}
	for _, id := range sortedKeys(external) {
		m.Nodes = append(m.Nodes, external[id])
	}

	return m.Normalise()
}

func serviceID(name string) string  { return "svc:" + name }
func externalID(host string) string { return "ext:" + host }

// sortedKeys exists for the same reason as every other sort in this codebase: Go randomises map
// iteration, and AC-7 requires five runs to produce byte-identical output.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
