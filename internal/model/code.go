package model

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// What the code says about the system beyond its own structure (F-04, F-08, F-13, F-32): the
// routes it serves, the other containers it reaches, and the calls whose target only exists at
// run time.

// A property that holds an endpoint: url, baseUrl, endpoint, host, issuer, lightStyle.
var endpointKey = regexp.MustCompile(`(?i)(url|uri|endpoint|host|hostname|server|origin|issuer|style)s?$`)

// configured reports a URL a configuration key holds, whole, in code that is not a template: an
// endpoint the application is set up to reach. A link an email or a page shows is not one, and
// nor is a field of one record among many — the url of each company a page lists.
func configured(h archdoc.HostRef, f archdoc.SourceFile) bool {
	if h.Built || h.Row || h.Scheme == "" || !endpointKey.MatchString(h.Key) {
		return false
	}
	return f.Language != "TSX" && f.Language != "Svelte" && !strings.HasSuffix(f.Path, ".jsx")
}

// placeholder reports a host that stands for no real system: this machine, or documentation's —
// the names and the top-level domains RFC 2606 reserves for examples and tests.
func placeholder(host string) bool {
	if host == "localhost" || host == "0.0.0.0" || strings.HasPrefix(host, "127.") || strings.HasSuffix(host, ".local") ||
		host == "host.docker.internal" {
		return true
	}
	for _, d := range []string{"example.com", "example.net", "example.org", "example", "test", "invalid", "localhost"} {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

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
		sdks(m, src, container, has)
		m.Entries = append(m.Entries, routes(src, container, componentOf)...)
		m.Entries = append(m.Entries, controllerRoutes(src, container, componentOf)...)
		m.Entries = append(m.Entries, fastapiRoutes(src, container, componentOf)...)
		m.Entries = append(m.Entries, pages(src, container, componentOf)...)
		m.Entries = append(m.Entries, commandsAndJobs(src, container, componentOf)...)

		// A configuration file — application.yml, appsettings.json — holds settings and nothing else:
		// every address in one is an endpoint the application is set up to reach.
		setting := map[string]bool{}
		for _, f := range src.Settings {
			setting[f.Path] = true
		}
		for _, f := range append(append([]archdoc.SourceFile(nil), src.Files...), src.Settings...) {
			for _, h := range f.Hosts {
				to := ""
				if name, ok := declared[h.Host]; ok {
					to = serviceID(name)
				} else if (h.Called || configured(h, f) || setting[f.Path] && h.Scheme != "" && !h.Built) && !placeholder(h.Host) {
					// Not this repository's, and either called directly or held by a configuration
					// key — url, endpoint, host: an external system the code reaches. A URL in a
					// sentence, a link or a comment is none of these, and draws nothing.
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
				switch {
				case !h.Called && strings.HasPrefix(to, "ext:"):
					label = archdoc.Configured // a default the code holds, not a call it was seen to make
				case h.Scheme == "http" || h.Scheme == "https" || h.Scheme == "ws" || h.Scheme == "wss":
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
	"delete": "DELETE", "head": "HEAD", "options": "OPTIONS", "api_route": "ANY", "websocket": "WS", "all": "ALL"}

// The applications a router is finally included into: FastAPI's, Express's, ASP.NET's.
var rootRouters = map[string]bool{"FastAPI": true, "express": true, "aspnet": true}

type routerKey struct{ file, name string }

// fastapiRoutes reads the routes of a FastAPI or an Express application — the two declare them the
// same way, a verb on a router: @router.get("/{id}") over a function, router.get("/:id", handler).
// For FastAPI: each decorated function under the router
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
			if rootRouters[r.Kind] {
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
		stem := path.Base(f.Path)
		if i := strings.Index(stem, "."); i > 0 {
			stem = stem[:i]
		}
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

// SvelteKit's endpoint exports, by the HTTP method each one answers.
var kitMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "OPTIONS": true, "HEAD": true}

// pages reads a front end's pages and endpoints from where its framework declares them: SvelteKit
// by file — a +page.svelte under src/routes is a page at its directory's path, a +server.ts an
// endpoint for each method it exports — and TanStack Router by call, createFileRoute("/items").
// A SvelteKit route group, (user), is not part of the path; a TanStack layout, _layout, is not either.
func pages(src archdoc.Source, container string, componentOf map[string]string) []archdoc.Entry {
	_, local, _ := strings.Cut(container, ":")
	routesDir := src.Root + "/routes/"
	// Next.js keeps pages in app/ (a page.tsx per route, a route.ts per endpoint) or in pages/
	// (a file per page, pages/api for endpoints), under src/ or beside it.
	nextApp, nextPages := src.Root+"/app/", src.Root+"/pages/"
	// Only where the code is Next's: a React app keeps components in src/pages too, and its
	// router, not the folder, says which are pages. Found on a six-file app that reported four
	// pages for two.
	next := src.Framework == "Next.js"
	for _, f := range src.Files {
		for _, imp := range f.Imports {
			next = next || imp.Package == "next" || strings.HasPrefix(imp.Spec, "next/")
		}
	}
	if !next {
		nextApp, nextPages = "\x00", "\x00"
	}
	var out []archdoc.Entry
	seen := map[string]bool{}
	add := func(e archdoc.Entry) {
		prefix := "page:"
		if e.Kind == "http" {
			prefix = "route:"
		}
		e.ID = prefix + local + " " + strings.TrimPrefix(e.Method+" ", "PAGE ") + e.Path
		if seen[e.ID] {
			return
		}
		seen[e.ID] = true
		e.Container, e.Component = container, componentOf[e.Prov.File]
		out = append(out, e)
	}
	for _, f := range src.Files {
		if rel, ok := strings.CutPrefix(f.Path, routesDir); ok {
			dir, base := path.Split(rel)
			at := kitPath(dir)
			switch {
			case base == "+page.svelte":
				add(archdoc.Entry{Kind: "page", Method: "PAGE", Path: at, Handler: rel, Prov: archdoc.Provenance{File: f.Path, Line: 1}})
			case strings.HasPrefix(base, "+server."):
				for _, x := range f.Exports {
					if kitMethods[x.Value] {
						add(archdoc.Entry{Kind: "http", Method: x.Value, Path: at, Handler: rel + " " + x.Value, Prov: x.Prov})
					}
				}
			}
		}
		if rel, ok := strings.CutPrefix(f.Path, nextApp); ok {
			dir, base := path.Split(rel)
			name := base[:len(base)-len(path.Ext(base))]
			switch name {
			case "page":
				add(archdoc.Entry{Kind: "page", Method: "PAGE", Path: kitPath(dir), Handler: "app/" + rel, Prov: archdoc.Provenance{File: f.Path, Line: 1}})
			case "route":
				for _, x := range f.Exports {
					if kitMethods[x.Value] {
						add(archdoc.Entry{Kind: "http", Method: x.Value, Path: kitPath(dir), Handler: "app/" + rel + " " + x.Value, Prov: x.Prov})
					}
				}
			}
		}
		if rel, ok := strings.CutPrefix(f.Path, nextPages); ok && (f.Language == "TSX" || strings.HasSuffix(f.Path, ".jsx") || strings.HasPrefix(rel, "api/")) {
			at := "/" + strings.TrimSuffix(strings.TrimSuffix(rel[:len(rel)-len(path.Ext(rel))], "index"), "/")
			base := path.Base(rel)
			switch {
			case strings.HasPrefix(base, "_"):
				// _app, _document: the frame around every page, not a page.
			case strings.HasPrefix(rel, "api/"):
				add(archdoc.Entry{Kind: "http", Method: "ANY", Path: at, Handler: "pages/" + rel, Prov: archdoc.Provenance{File: f.Path, Line: 1}})
			default:
				add(archdoc.Entry{Kind: "page", Method: "PAGE", Path: at, Handler: "pages/" + rel, Prov: archdoc.Provenance{File: f.Path, Line: 1}})
			}
		}
		for _, p := range f.Pages {
			add(archdoc.Entry{Kind: "page", Method: "PAGE", Path: tanstackPath(p.Value), Handler: strings.TrimPrefix(f.Path, src.Root+"/"), Prov: p.Prov})
		}
	}
	// With no manifest and no router, a page is an HTML file: what a person opens.
	for _, d := range src.Documents {
		e := archdoc.Entry{Kind: "page", Method: "PAGE", Path: "/" + strings.TrimPrefix(strings.TrimPrefix(d.Prov.File, src.Root+"/"), "./"),
			Handler: d.Prov.File, Prov: d.Prov}
		if d.Value != "" {
			e.Summary, e.SummaryProv = d.Value, d.Prov
		}
		add(e)
	}
	return out
}

// kitPath is a SvelteKit route directory as a path: groups dropped, parameters as written.
func kitPath(dir string) string {
	var segs []string
	for _, s := range strings.Split(strings.Trim(dir, "/"), "/") {
		if s == "" || strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
			continue
		}
		segs = append(segs, s)
	}
	return "/" + strings.Join(segs, "/")
}

// tanstackPath is a createFileRoute path without its pathless layouts: /_layout/items is /items.
func tanstackPath(p string) string {
	var segs []string
	for _, s := range strings.Split(strings.Trim(p, "/"), "/") {
		if s == "" || strings.HasPrefix(s, "_") {
			continue
		}
		segs = append(segs, s)
	}
	return "/" + strings.Join(segs, "/")
}

// commandsAndJobs reads the other ways into an application (F-13): the commands it offers on the
// command line — nest-commander's @Command classes, commander's program.command(…), Typer's and
// Click's @app.command() — and the work it does without being asked: @OnJob handlers on a queue,
// @Cron and @Interval schedules. A job named by an enum member is resolved by name.
func commandsAndJobs(src archdoc.Source, container string, componentOf map[string]string) []archdoc.Entry {
	_, local, _ := strings.Cut(container, ":")
	constants := map[string]archdoc.Constant{}
	for _, f := range src.Files {
		for _, c := range f.Constants {
			constants[c.Name] = c
		}
	}
	var out []archdoc.Entry
	seen := map[string]bool{}
	add := func(e archdoc.Entry) {
		e.ID = e.Kind + ":" + local + " " + e.Path
		if seen[e.ID] {
			e.ID += " " + e.Handler
		}
		if e.Path == "" || seen[e.ID] {
			return
		}
		seen[e.ID] = true
		e.Container, e.Component = container, componentOf[e.Prov.File]
		out = append(out, e)
	}
	for _, f := range src.Files {
		rel := strings.TrimPrefix(f.Path, src.Root+"/")
		for _, c := range f.Commands {
			add(archdoc.Entry{Kind: "command", Method: "CMD", Path: c.Name, Handler: rel, Summary: c.Summary, SummaryProv: c.SummaryProv, Prov: c.Prov})
		}
		for _, c := range f.Classes {
			for _, d := range c.Decorators {
				if d.Name == "Command" && d.Options["name"] != "" {
					e := archdoc.Entry{Kind: "command", Method: "CMD", Path: d.Options["name"], Handler: c.Name + ".run", Prov: d.Prov}
					if s := d.Options["description"]; s != "" {
						e.Summary, e.SummaryProv = s, d.Prov
					}
					add(e)
				}
			}
			for _, meth := range c.Methods {
				handler := c.Name + "." + meth.Name
				if c.Name == "" {
					handler = rel + " " + meth.Name
				}
				for _, d := range meth.Decorators {
					switch {
					case d.Name == "OnJob":
						name, note := d.Options["name"], ""
						if expr := d.Exprs["name"]; name == "" && expr != "" {
							if k, ok := constants[expr]; ok {
								name, note = k.Value, expr+" is \""+k.Value+"\", resolved by name at "+k.Prov.String()
							} else {
								name, note = "{"+expr+"}", expr+" could not be resolved"
							}
						}
						add(archdoc.Entry{Kind: "job", Method: "JOB", Path: name, Handler: handler, PathNote: note, Prov: d.Prov})
					case d.Name == "Cron" || d.Name == "Interval":
						when := d.Arg
						if when == "" {
							when = d.ArgExpr
						}
						add(archdoc.Entry{Kind: "job", Method: strings.ToUpper(d.Name), Path: meth.Name, Handler: handler,
							PathNote: "scheduled: " + when, Prov: d.Prov})
					case strings.HasSuffix(d.Name, ".command") && strings.HasSuffix(f.Path, ".py"):
						e := archdoc.Entry{Kind: "command", Method: "CMD", Path: strings.ReplaceAll(meth.Name, "_", "-"), Handler: handler, Prov: d.Prov}
						if d.Arg != "" {
							e.Path = d.Arg
						}
						if meth.Doc != "" {
							e.Summary, e.SummaryProv = meth.Doc, meth.DocProv
						}
						add(e)
					}
				}
			}
		}
	}
	// With no manifest to declare a command, a script is what a person runs: each Python file that
	// says it is run directly, or — a JavaScript tool — the file it starts from.
	if src.Loose {
		for _, f := range src.Files {
			rel := strings.TrimPrefix(strings.TrimPrefix(f.Path, src.Root+"/"), "./")
			if f.Main != nil {
				add(archdoc.Entry{Kind: "command", Method: "CMD", Path: "python " + rel, Handler: rel, Prov: *f.Main})
			}
		}
	}
	return out
}
