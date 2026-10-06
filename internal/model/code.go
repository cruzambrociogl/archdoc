package model

import (
	"path"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// What the code says about the system beyond its own structure (F-04, F-08, F-13, F-32): the
// routes it serves, the other containers it reaches, and the calls whose target only exists at
// run time.

// NestJS's route decorators, and the HTTP method each declares.
var nestMethods = map[string]string{
	"Get": "GET", "Post": "POST", "Put": "PUT", "Patch": "PATCH", "Delete": "DELETE",
	"All": "ALL", "Head": "HEAD", "Options": "OPTIONS",
}

func fromCode(m *archdoc.Model, sources []archdoc.Source, componentOf map[string]string, declared map[string]string) {
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	has := map[string]bool{}
	for _, n := range m.Nodes {
		has[n.ID] = true
	}

	for _, src := range sources {
		container, ok := containerOf[src.App]
		if !ok {
			continue
		}
		m.Entries = append(m.Entries, routes(src, container, componentOf)...)

		for _, f := range src.Files {
			for _, h := range f.Hosts {
				to := ""
				if name, ok := declared[h.Host]; ok {
					to = serviceID(name)
				} else if h.Called && h.Host != "localhost" && !strings.HasPrefix(h.Host, "127.") {
					// Not this repository's, and called directly: an external system the code reaches.
					to = externalID(h.Host)
					if !has[to] {
						has[to] = true
						m.Nodes = append(m.Nodes, archdoc.Node{ID: to, Name: h.Host, Kind: archdoc.External,
							Evidence: archdoc.Referenced, Prov: h.Prov})
					}
				}
				if to == "" || to == container {
					continue
				}
				label := "connects to"
				if h.Scheme == "http" || h.Scheme == "https" || h.Scheme == "ws" || h.Scheme == "wss" {
					label = "calls"
				}
				m.Edges = append(m.Edges, archdoc.Edge{From: container, To: to, Label: label,
					Technology: h.Scheme, Traffic: true, Prov: []archdoc.Provenance{h.Prov}})
			}
			for _, c := range f.Calls {
				m.Unresolved = append(m.Unresolved, archdoc.Unresolved{
					Container: container, Component: componentOf[f.Path],
					What: c.Callee + "(" + c.Target + ")",
					Note: "an HTTP call whose address is computed at run time; nothing in the code names its target",
					Prov: c.Prov,
				})
			}
		}
	}
	sort.SliceStable(m.Entries, func(i, j int) bool { return m.Entries[i].ID < m.Entries[j].ID })
	sort.SliceStable(m.Unresolved, func(i, j int) bool {
		a, b := m.Unresolved[i].Prov, m.Unresolved[j].Prov
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
}

// routes reads a NestJS application's routes: each @Controller class's decorated methods, under
// the controller's path and the global prefix, with the services its constructor is given.
func routes(src archdoc.Source, container string, componentOf map[string]string) []archdoc.Entry {
	// Where each class is declared, for resolving an injected type by its name. A name declared
	// twice is ambiguous, and stays unresolved rather than picking one.
	declaredAt := map[string]archdoc.Provenance{}
	twice := map[string]bool{}
	constants := map[string]archdoc.Constant{}
	var prefix *archdoc.Literal
	for _, f := range src.Files {
		for _, c := range f.Constants {
			constants[c.Name] = c
		}
		if f.Prefix != nil && prefix == nil {
			prefix = f.Prefix
		}
		for _, c := range f.Classes {
			if c.Name == "" {
				continue
			}
			if _, seen := declaredAt[c.Name]; seen {
				twice[c.Name] = true
			}
			declaredAt[c.Name] = c.Prov
		}
	}

	_, local, _ := strings.Cut(container, ":")
	var out []archdoc.Entry
	seen := map[string]bool{}
	for _, f := range src.Files {
		for _, c := range f.Classes {
			base, note, controller := "", "", false
			for _, d := range c.Decorators {
				if d.Name != "Controller" {
					continue
				}
				base, controller = d.Arg, true
				if d.ArgExpr != "" {
					// @Controller(RouteKey.Asset): an enum member, resolved by its name where the
					// enum declares it; anything else stays in braces, visibly unresolved.
					if k, ok := constants[d.ArgExpr]; ok {
						base, note = k.Value, d.ArgExpr+" is \""+k.Value+"\", resolved by name at "+k.Prov.String()
					} else {
						base, note = "{"+d.ArgExpr+"}", d.ArgExpr+" could not be resolved"
					}
				}
			}
			if !controller {
				continue
			}
			var uses []archdoc.Symbol
			for _, in := range c.Injects {
				s := archdoc.Symbol{Name: in.Value, How: "unresolved", Prov: in.Prov}
				if at, ok := declaredAt[in.Value]; ok && !twice[in.Value] {
					s.How, s.Prov, s.Component = "name", at, componentOf[at.File]
				}
				uses = append(uses, s)
			}
			for _, meth := range c.Methods {
				e := archdoc.Entry{Kind: "http", Handler: c.Name + "." + meth.Name, Container: container,
					Component: componentOf[f.Path], Uses: uses, PathNote: note}
				for _, d := range meth.Decorators {
					if method, ok := nestMethods[d.Name]; ok && e.Method == "" {
						e.Method, e.Prov = method, d.Prov
						parts := []string{"/"}
						if prefix != nil {
							parts = append(parts, prefix.Value)
							e.PrefixProv = prefix.Prov
						}
						e.Path = path.Join(append(parts, base, d.Arg)...)
					}
					if d.Summary != "" && e.Summary == "" {
						e.Summary, e.SummaryProv = d.Summary, d.SummaryProv
					}
				}
				if e.Method == "" {
					continue
				}
				e.ID = "route:" + local + " " + e.Method + " " + e.Path
				if seen[e.ID] {
					e.ID += " " + e.Handler // two handlers for one route: both are kept, told apart
				}
				seen[e.ID] = true
				out = append(out, e)
			}
		}
	}
	return out
}
