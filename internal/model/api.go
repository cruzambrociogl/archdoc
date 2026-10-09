package model

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
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

func apiCalls(m *archdoc.Model, f *archdoc.FactSet) {
	if len(f.APIs) == 0 {
		return
	}
	containerOf := map[string]string{} // application directory → its container
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	appAt := map[string]archdoc.App{}
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
				if i.How == archdoc.ByPackage {
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
		// The container whose routes these operations are. A route may sit under a prefix the
		// document leaves out (/api), so a route matches when it ends with the operation's path.
		routes := map[string][]string{} // container → "METHOD path"
		for _, e := range m.Entries {
			if e.Kind == "http" {
				routes[e.Container] = append(routes[e.Container], e.Method+" "+samePath(e.Path))
			}
		}
		provider, matched := "", 0
		for _, c := range sortedKeys(routes) {
			n := 0
			for _, op := range api.Operations {
				want := op.Method + " "
				suffix := samePath(op.Path)
				for _, r := range routes[c] {
					if strings.HasPrefix(r, want) && strings.HasSuffix(r, suffix) &&
						(len(r) == len(want)+len(suffix) || r[len(r)-len(suffix)-1] != '{' && strings.HasPrefix(suffix, "/")) {
						n++
						break
					}
				}
			}
			if n > matched {
				provider, matched = c, n
			}
		}
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
				prov []archdoc.Provenance
				via  string
			}
			var callers []caller
			if id, isContainer := containerOf[holder.Dir]; isContainer {
				callers = append(callers, caller{id, []archdoc.Provenance{c.Prov}, c.How})
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
							callers = append(callers, caller{id, []archdoc.Provenance{r.Prov, c.Prov},
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
				m.Edges = append(m.Edges, archdoc.Edge{From: from.id, To: provider, Label: "calls the API of",
					Technology: "http", Traffic: true, Prov: prov})
			}
		}
	}
}

// people ties the person to the applications a person runs (F-08): a web front end, a mobile app,
// a command-line tool. What makes an application one of those is a line in its manifest — the
// framework it depends on, the command it declares — and that line is what the arrow cites.
func people(m *archdoc.Model, f *archdoc.FactSet) {
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
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
		case archdoc.RoleWeb:
			label, why = "uses", "a web front end is what a person opens"
		case archdoc.RoleMobile:
			label, why = "uses", "a mobile app is what a person opens"
		case archdoc.RoleCLI:
			label, why = "runs", "a command-line tool is what a person runs"
		default:
			continue
		}
		prov := a.FrameworkProv
		if !prov.Known() {
			prov = a.Prov
		}
		prov.Note = why + " — " + a.Why
		if !hasActor {
			hasActor = true
			m.Nodes = append(m.Nodes, archdoc.Node{ID: actorID, Name: "User", Kind: archdoc.Actor, Evidence: archdoc.Declared, Prov: prov})
		}
		reached[id] = true
		m.Edges = append(m.Edges, archdoc.Edge{From: actorID, To: id, Label: label, Traffic: true, Prov: []archdoc.Provenance{prov}})
	}
}
