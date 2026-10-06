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
		m.Entries = append(m.Entries, fastapiRoutes(src, container, componentOf)...)

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

// FastAPI's route decorators: @router.get("/{id}") on a function, the verb after the router.
var fastapiVerbs = map[string]string{"get": "GET", "post": "POST", "put": "PUT", "patch": "PATCH",
	"delete": "DELETE", "head": "HEAD", "options": "OPTIONS", "api_route": "ANY", "websocket": "WS"}

type routerKey struct{ file, name string }

// fastapiRoutes reads a FastAPI application's routes: each decorated function under the router
// it is declared on, that router under every router it is included into, up to the application —
// router = APIRouter(prefix="/items"), api_router.include_router(items.router),
// app.include_router(api_router, prefix=settings.API_V1_STR). A prefix written as a setting is
// resolved by name to the class field that declares it; an include inside an if says so.
func fastapiRoutes(src archdoc.Source, container string, componentOf map[string]string) []archdoc.Entry {
	routers := map[routerKey]archdoc.Router{}
	values := map[string]archdoc.Field{} // a class field's literal, by name: API_V1_STR = "/api/v1"
	files := map[string]archdoc.SourceFile{}
	for _, f := range src.Files {
		files[f.Path] = f
		for _, r := range f.Routers {
			routers[routerKey{f.Path, r.Var}] = r
		}
		for _, c := range f.Classes {
			for _, fl := range c.Fields {
				if fl.Value != "" {
					if _, seen := values[fl.Name]; !seen {
						values[fl.Name] = fl
					}
				}
			}
		}
	}
	if len(routers) == 0 {
		return nil
	}

	type link struct {
		parent routerKey
		prefix string
		notes  []string
		cond   string
		prov   archdoc.Provenance
	}
	included := map[routerKey]link{}
	for _, f := range src.Files {
		for _, inc := range f.Includes {
			parent := routerKey{f.Path, inc.Parent}
			if _, ok := routers[parent]; !ok {
				continue
			}
			child, ok := childRouter(f, inc.Child, routers)
			if !ok {
				continue
			}
			l := link{parent: parent, prefix: inc.Prefix, cond: inc.Condition, prov: inc.Prov}
			if inc.PrefixExpr != "" {
				name := inc.PrefixExpr[strings.LastIndex(inc.PrefixExpr, ".")+1:]
				if v, ok := values[name]; ok {
					l.prefix = v.Value
					l.notes = append(l.notes, inc.PrefixExpr+" is \""+v.Value+"\", resolved by name at "+v.Prov.String())
				} else {
					l.prefix = "{" + inc.PrefixExpr + "}"
					l.notes = append(l.notes, inc.PrefixExpr+" could not be resolved")
				}
			}
			included[child] = l
		}
	}

	// prefixOf walks a router up to the application that includes it.
	prefixOf := func(k routerKey) (string, []string, bool) {
		var parts, notes []string
		seen := map[routerKey]bool{}
		for depth := 0; depth < 10 && !seen[k]; depth++ {
			seen[k] = true
			r := routers[k]
			if r.Prefix != "" {
				parts = append([]string{r.Prefix}, parts...)
			}
			if r.Kind == "FastAPI" {
				return path.Join(append([]string{"/"}, parts...)...), notes, true
			}
			l, ok := included[k]
			if !ok {
				break
			}
			if l.prefix != "" {
				parts = append([]string{l.prefix}, parts...)
			}
			notes = append(notes, l.notes...)
			if l.cond != "" {
				notes = append(notes, "included only when "+l.cond+", at "+l.prov.String())
			}
			k = l.parent
		}
		notes = append(notes, "its router is not included into an application archdoc found")
		return path.Join(append([]string{"/"}, parts...)...), notes, false
	}

	_, local, _ := strings.Cut(container, ":")
	var out []archdoc.Entry
	seen := map[string]bool{}
	for _, f := range src.Files {
		stem := strings.TrimSuffix(path.Base(f.Path), ".py")
		for _, c := range f.Classes {
			for _, meth := range c.Methods {
				for _, d := range meth.Decorators {
					dot := strings.LastIndex(d.Name, ".")
					if dot < 0 {
						continue
					}
					verb, ok := fastapiVerbs[d.Name[dot+1:]]
					key := routerKey{f.Path, d.Name[:dot]}
					if _, isRouter := routers[key]; !ok || !isRouter {
						continue
					}
					prefix, notes, _ := prefixOf(key)
					e := archdoc.Entry{Kind: "http", Method: verb, Path: path.Join(prefix, d.Arg), Handler: stem + "." + meth.Name,
						Container: container, Component: componentOf[f.Path], PathNote: strings.Join(notes, "; "), Prov: d.Prov}
					if strings.HasSuffix(d.Arg, "/") && e.Path != "/" {
						e.Path += "/" // FastAPI tells /items from /items/; so does the route
					}
					switch {
					case d.Summary != "":
						e.Summary, e.SummaryProv = d.Summary, d.SummaryProv
					case meth.Doc != "":
						e.Summary, e.SummaryProv = meth.Doc, meth.DocProv
					}
					e.ID = "route:" + local + " " + e.Method + " " + e.Path
					if seen[e.ID] {
						e.ID += " " + e.Handler
					}
					seen[e.ID] = true
					out = append(out, e)
				}
			}
		}
	}
	return out
}

// childRouter resolves what an include names: items.router — the router in the module the file
// imports as items — or a router the file itself declares.
func childRouter(f archdoc.SourceFile, expr string, routers map[routerKey]archdoc.Router) (routerKey, bool) {
	if mod, name, ok := strings.Cut(expr, "."); ok {
		for _, imp := range f.Imports {
			if imp.Target == "" {
				continue
			}
			spec := imp.Spec[strings.LastIndex(imp.Spec, ".")+1:]
			if spec == mod {
				k := routerKey{imp.Target, name}
				_, ok := routers[k]
				return k, ok
			}
		}
		return routerKey{}, false
	}
	k := routerKey{f.Path, expr}
	if _, ok := routers[k]; ok {
		return k, true
	}
	// from app.api.main import api_router: a router another module declares, imported by name.
	for _, imp := range f.Imports {
		if k := (routerKey{imp.Target, expr}); imp.Target != "" {
			if _, ok := routers[k]; ok {
				return k, true
			}
		}
	}
	return routerKey{}, false
}
