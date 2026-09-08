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
	m := archdoc.Model{Name: f.Name, Source: f.Source, Networks: f.Networks}

	// A host names a declared service if it matches the service key or any name that service
	// also answers to. Without alias resolution a real container is drawn twice — once as
	// itself, once as a stranger the repository appears to depend on.
	declared := make(map[string]string, len(f.Services))
	for _, s := range f.Services {
		declared[s.Name] = s.Name
	}
	for _, s := range f.Services {
		for _, a := range s.Aliases {
			if _, taken := declared[a]; !taken {
				declared[a] = s.Name
			}
		}
	}

	for _, s := range f.Services {
		kind, tech, techProv := classify(s.Image)

		nets := make([]string, 0, len(s.Networks))
		for _, n := range s.Networks {
			nets = append(nets, n.Name)
		}

		m.Nodes = append(m.Nodes, archdoc.Node{
			ID:         serviceID(s.Name),
			Name:       s.Name,
			Kind:       kind,
			Technology: tech,
			TechProv:   techProv,
			Evidence:   archdoc.Declared,
			Networks:   nets,
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
			if _, ok := declared[d.Service]; !ok {
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
			to := target(e.Host, declared, external, e.Prov)

			m.Edges = append(m.Edges, archdoc.Edge{
				From:       from,
				To:         to,
				Label:      "connects to",
				Technology: e.Scheme,
				Traffic:    true,
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
				From:    actorID,
				To:      from,
				Label:   "reaches",
				Traffic: true,
				Prov:    []archdoc.Provenance{p.Prov},
			})
		}
	}

	// Routes are the gateway's own configuration saying where traffic goes. depends_on says a
	// gateway starts after a service; only a route says it reaches one.
	for _, r := range f.Routes {
		from, ok := declared[r.Gateway]
		if !ok {
			continue
		}
		m.Edges = append(m.Edges, archdoc.Edge{
			From:    serviceID(from),
			To:      target(r.Target, declared, external, r.Prov),
			Label:   "routes to",
			Traffic: true,
			Prov:    []archdoc.Provenance{r.Prov},
		})
	}

	if actor != nil {
		m.Nodes = append(m.Nodes, *actor)
	}
	for _, id := range sortedKeys(external) {
		m.Nodes = append(m.Nodes, external[id])
	}

	return m.Normalise()
}

// target resolves a hostname to the node it names, creating an external node when the
// repository never declared it. That is the evidence rule doing double duty: referenced is also
// where C4 draws the system boundary.
func target(host string, declared map[string]string, external map[string]archdoc.Node, prov archdoc.Provenance) string {
	if name, ok := declared[host]; ok {
		return serviceID(name)
	}

	id := externalID(host)
	if _, seen := external[id]; !seen {
		external[id] = archdoc.Node{
			ID:       id,
			Name:     host,
			Kind:     archdoc.External,
			Evidence: archdoc.Referenced,
			Prov:     prov,
		}
	}
	return id
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
