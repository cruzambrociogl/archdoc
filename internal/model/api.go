package model

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Who calls whose API (F-08), where a repository says so in a file: an OpenAPI document.
//
// The document is tied to the container it describes by counting — its operations against the
// routes each container's code declares. A client of the document is then a caller of that
// container: an application that holds the client, or one whose manifest depends on the package
// that does. Every link in that chain is a line: the dependency, the client, the document.
//
// Nothing is drawn from a document no container's routes match, nor to a client nobody uses.

var pathParam = regexp.MustCompile(`\{[^}]*\}|:[A-Za-z_][A-Za-z0-9_]*`)

// samePath normalises a path so /albums/{id} and /albums/:albumId are the same.
func samePath(p string) string {
	return strings.TrimSuffix(pathParam.ReplaceAllString(p, "{}"), "/")
}

// routeMatches reports whether a route ("METHOD path", normalised) is an operation of a document. A
// route may sit under a prefix the document leaves out (/api), so it matches when it ends with the
// operation's path.
func routeMatches(r string, op core.Operation) bool {
	want, suffix := op.Method+" ", samePath(op.Path)
	return strings.HasPrefix(r, want) && strings.HasSuffix(r, suffix) &&
		(len(r) == len(want)+len(suffix) || r[len(r)-len(suffix)-1] != '{' && strings.HasPrefix(suffix, "/"))
}

// provider is the container whose routes a document's operations are, and how many of them match.
func provider(m *core.Model, api core.API) (string, int) {
	routes := map[string][]string{} // container → "METHOD path"
	for _, e := range m.Entries {
		if e.Kind == "http" {
			routes[e.Container] = append(routes[e.Container], e.Method+" "+samePath(e.Path))
		}
	}
	best, matched := "", 0
	for _, c := range sortedKeys(routes) {
		n := 0
		for _, op := range api.Operations {
			for _, r := range routes[c] {
				if routeMatches(r, op) {
					n++
					break
				}
			}
		}
		if n > matched {
			best, matched = c, n
		}
	}
	return best, matched
}

func apiCalls(m *core.Model, f *core.FactSet) {
	if len(f.APIs) == 0 {
		return
	}
	containerOf := map[string]string{} // application directory → its container
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == core.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	appAt := map[string]core.App{}
	for _, a := range f.Apps {
		appAt[a.Dir] = a
	}
	name := func(id string) string {
		if n, ok := m.Node(id); ok {
			return n.Name
		}
		return id
	}
	drawn := map[string]bool{}
	for _, e := range m.Edges {
		if e.Traffic {
			drawn[e.From+"\x00"+e.To] = true
		}
	}

	imports := map[string]map[string]bool{} // application directory → the packages its code imports
	for _, src := range f.Sources {
		imports[src.App] = map[string]bool{}
		for _, file := range src.Files {
			for _, i := range file.Imports {
				if i.How == core.ByPackage {
					imports[src.App][i.Spec] = true
					if scope, rest, ok := strings.Cut(i.Spec, "/"); ok && strings.HasPrefix(scope, "@") {
						pkg, _, _ := strings.Cut(rest, "/")
						imports[src.App][scope+"/"+pkg] = true
					}
				}
			}
		}
	}

	for _, api := range f.APIs {
		provider, matched := provider(m, api)
		if provider == "" || matched*2 < len(api.Operations) {
			continue // it describes something whose routes archdoc did not read
		}
		describes := fmt.Sprintf("%s describes %s: %d of its %d operations are routes the code declares",
			api.File, name(provider), matched, len(api.Operations))

		for _, c := range api.Clients {
			holder, ok := appAt[c.App]
			if !ok {
				continue
			}
			type caller struct {
				id   string
				prov []core.Provenance
				via  string
			}
			var callers []caller
			if id, isContainer := containerOf[holder.Dir]; isContainer {
				callers = append(callers, caller{id, []core.Provenance{c.Prov}, c.How})
			} else {
				// A package of its own: whoever depends on it, to run, calls through it.
				for _, a := range f.Apps {
					id, isContainer := containerOf[a.Dir]
					if !isContainer {
						continue
					}
					for _, r := range a.Requires {
						// A development dependency counts only where the code imports it: a tool
						// that is bundled lists what it ships with under devDependencies.
						if r.Name == holder.Name && (!r.Dev || imports[a.Dir][holder.Name]) {
							callers = append(callers, caller{id, []core.Provenance{r.Prov, c.Prov},
								"through " + holder.Name + ", which it depends on — " + c.How})
						}
					}
				}
			}
			sort.Slice(callers, func(i, j int) bool { return callers[i].id < callers[j].id })
			for _, from := range callers {
				if from.id == provider || drawn[from.id+"\x00"+provider] {
					continue
				}
				drawn[from.id+"\x00"+provider] = true
				prov := append(from.prov, api.Prov)
				prov[0].Note = from.via + "; " + describes
				m.Edges = append(m.Edges, core.Edge{From: from.id, To: provider, Label: "calls the API of",
					Technology: "http", Traffic: true, Prov: prov})
			}
		}
	}
}

// pageCalls (F-12): what a page calls, where its code imports a generated client's functions. A client
// generated from an OpenAPI document names each function after an operation's operationId, and the
// operation is a route a container declares: a page whose code imports getAllAlbums from @immich/sdk
// calls GET /api/albums. Each link is a line: the import (or the call, where the code makes it in a
// function), the operationId, the route. A page's own files are read, the page and the +page load
// SvelteKit keeps beside it, not the components it uses, which pages share. The flow is the page's: a
// step to each route it calls, whose handler has a flow of its own.
func pageCalls(m *core.Model, f *core.FactSet, componentOf map[string]string) {
	if len(f.APIs) == 0 {
		return
	}
	files := map[string]core.SourceFile{}
	inDir := map[string][]string{}
	for _, src := range f.Sources {
		for _, file := range src.Files {
			files[file.Path] = file
			inDir[path.Dir(file.Path)] = append(inDir[path.Dir(file.Path)], file.Path)
		}
	}
	appAt := map[string]core.App{}
	for _, a := range f.Apps {
		appAt[a.Dir] = a
	}
	calls := map[string]bool{} // the containers apiCalls found calling another's API
	for _, e := range m.Edges {
		if e.Label == "calls the API of" {
			calls[e.From+"\x00"+e.To] = true
		}
	}
	// A page's own files: the page, and beside a SvelteKit +page.svelte the +page load files.
	own := func(file string) []string {
		out := []string{file}
		if strings.HasPrefix(path.Base(file), "+page.") {
			for _, p := range inDir[path.Dir(file)] {
				if p != file && strings.HasPrefix(path.Base(p), "+page.") {
					out = append(out, p)
				}
			}
		}
		sort.Strings(out[1:])
		return out
	}
	// where a file calls a function by name, in one of its functions.
	callAt := func(file, name string) (core.Provenance, bool) {
		for _, c := range files[file].Classes {
			for _, meth := range c.Methods {
				for _, inv := range meth.Invokes {
					if inv.Free && inv.Method == name {
						return inv.Prov, true
					}
				}
			}
		}
		return core.Provenance{}, false
	}
	// A client's function is the operationId, as the generator spells it: get_all_albums or getAllAlbums.
	key := func(s string) string {
		return strings.Map(func(r rune) rune {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				return r
			case r >= 'A' && r <= 'Z':
				return r + 'a' - 'A'
			}
			return -1
		}, s)
	}

	type call struct {
		route core.Entry
		step  core.Step
	}
	byPage := map[string][]call{}
	seen := map[string]bool{}
	for _, api := range f.APIs {
		prov, matched := provider(m, api)
		if prov == "" || matched*2 < len(api.Operations) {
			continue
		}
		served := prov
		if n, ok := m.Node(prov); ok {
			served = n.Name
		}
		// The route each operation is: of the routes that end with its path, the shortest.
		route := map[string]core.Entry{}
		op := map[string]core.Operation{}
		for _, o := range api.Operations {
			if o.ID == "" {
				continue
			}
			var best core.Entry
			for _, e := range m.Entries {
				if e.Kind == "http" && e.Container == prov && routeMatches(e.Method+" "+samePath(e.Path), o) &&
					(best.ID == "" || len(e.Path) < len(best.Path)) {
					best = e
				}
			}
			if best.ID != "" {
				route[key(o.ID)], op[key(o.ID)] = best, o
			}
		}
		for _, c := range api.Clients {
			holder := appAt[c.App]
			via := holder.Name
			if via == "" {
				via = c.Dir
			}
			for _, e := range m.Entries {
				if e.Kind != "page" || !calls[e.Container+"\x00"+prov] {
					continue
				}
				for _, file := range own(e.Prov.File) {
					for _, imp := range files[file].Imports {
						fromClient := imp.How == core.ByPackage && holder.Name != "" && imp.Package == holder.Name ||
							imp.Target != "" && strings.HasPrefix(imp.Target, c.Dir+"/")
						if !fromClient {
							continue
						}
						for _, name := range imp.Names {
							r, ok := route[key(name)]
							if !ok || seen[e.ID+"\x00"+r.ID+"\x00"+name] {
								continue
							}
							seen[e.ID+"\x00"+r.ID+"\x00"+name] = true
							o := op[key(name)]
							at, called := callAt(file, name)
							how := "called through " + via
							if !called {
								at, how = imp.Prov, "imported here from "+via
							}
							byPage[e.ID] = append(byPage[e.ID], call{r, core.Step{From: e.Prov.File, To: r.ID, Call: name,
								Note: fmt.Sprintf("%s: operation %s at %s:%d, a route of %s with a flow of its own",
									how, o.ID, o.Prov.File, o.Prov.Line, served), Prov: at}})
						}
					}
				}
			}
		}
	}
	for _, e := range m.Entries {
		cs := byPage[e.ID]
		if len(cs) == 0 {
			continue
		}
		sort.SliceStable(cs, func(i, j int) bool {
			a, b := cs[i].step.Prov, cs[j].step.Prov
			if a.File != b.File {
				return a.File < b.File
			}
			if a.Line != b.Line {
				return a.Line < b.Line
			}
			return cs[i].step.Call < cs[j].step.Call
		})
		flow := core.Flow{Entry: e.ID, Participants: []core.Participant{{ID: e.Prov.File, Name: e.Path, Kind: "module", Component: componentOf[e.Prov.File]}}}
		met := map[string]bool{}
		for _, c := range cs {
			if len(flow.Steps) == flowSteps {
				flow.Cut = true
				break
			}
			if !met[c.route.ID] {
				met[c.route.ID] = true
				flow.Participants = append(flow.Participants, core.Participant{ID: c.route.ID, Name: c.route.Method + " " + c.route.Path, Kind: "route", Element: c.route.ID})
			}
			flow.Steps = append(flow.Steps, c.step)
		}
		m.Flows = append(m.Flows, flow)
	}
	sort.SliceStable(m.Flows, func(i, j int) bool { return m.Flows[i].Entry < m.Flows[j].Entry })
}

// firstPage is where a container's code first declares a page, in entry order.
func firstPage(m *core.Model, container string) (core.Provenance, bool) {
	for _, e := range m.Entries {
		if e.Container == container && e.Kind == "page" {
			return e.Prov, true
		}
	}
	return core.Provenance{}, false
}

// people ties the person to the applications a person runs (F-08): a web front end, a mobile app,
// a command-line tool. What makes an application one of those is a line in its manifest — the
// framework it depends on, the command it declares — and that line is what the arrow cites.
func people(m *core.Model, f *core.FactSet) {
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == core.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	reached := map[string]bool{}
	hasActor := false
	for _, n := range m.Nodes {
		hasActor = hasActor || n.ID == actorID
	}
	for _, e := range m.Edges {
		if e.From == actorID {
			reached[e.To] = true
		}
	}
	for _, a := range f.Apps {
		id, ok := containerOf[a.Dir]
		if !ok || reached[id] {
			continue
		}
		var label, why string
		switch a.Role {
		case core.RoleWeb:
			label, why = "uses", "a web front end is what a person opens"
		case core.RoleMobile:
			label, why = "uses", "a mobile app is what a person opens"
		case core.RoleCLI:
			label, why = "runs", "a command-line tool is what a person runs"
		default:
			// A service that serves pages — Razor Pages, an MVC view — is a site: what a person opens.
			if p, ok := firstPage(m, id); ok {
				p.Note = "it serves pages, and a page is what a person opens — " + a.Why
				if !hasActor {
					hasActor = true
					m.Nodes = append(m.Nodes, core.Node{ID: actorID, Name: "User", Kind: core.Actor, Evidence: core.Declared, Prov: p})
				}
				reached[id] = true
				m.Edges = append(m.Edges, core.Edge{From: actorID, To: id, Label: "uses", Traffic: true, Prov: []core.Provenance{p}})
			}
			continue
		}
		prov := a.FrameworkProv
		if !prov.Known() {
			prov = a.Prov
		}
		prov.Note = why + " — " + a.Why
		if !hasActor {
			hasActor = true
			m.Nodes = append(m.Nodes, core.Node{ID: actorID, Name: "User", Kind: core.Actor, Evidence: core.Declared, Prov: prov})
		}
		reached[id] = true
		m.Edges = append(m.Edges, core.Edge{From: actorID, To: id, Label: label, Traffic: true, Prov: []core.Provenance{prov}})
	}
}
