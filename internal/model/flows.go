package model

import (
	"sort"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Flows (F-12): what happens when an entry is called — arc42's runtime view, from the code.
//
// A flow starts at a route's handler and follows what each method does to its own object:
// this.albumRepository.getAll(…) is a call to AlbumRepository.getAll, because the constructor
// declared albumRepository an AlbumRepository (resolved by name, D-4). Inherited fields and
// methods are found through the classes a class extends. A query builder's table — .selectFrom
// ('album') — is a step to that table; an HTTP call to a computed address is a step to an
// unresolved participant. It stops at a fixed depth and length, and says when it did.

const (
	flowDepth = 4
	flowSteps = 40
)

// Calls that say nothing about what a flow does: logging.
var quiet = map[string]bool{"log": true, "debug": true, "verbose": true, "warn": true, "error": true, "info": true, "setContext": true, "fatal": true, "trace": true}

type classAt struct {
	class archdoc.Class
	file  string
}

func flows(m *archdoc.Model, sources []archdoc.Source, componentOf map[string]string) {
	tableOf := map[string]string{} // table name → node ID, per container
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Table {
			tableOf[n.Parent+"\x00"+n.Name] = n.ID
		}
	}
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	for _, src := range sources {
		container := containerOf[src.App]
		classes := map[string]classAt{}
		calls := map[string][]archdoc.Call{} // file → its HTTP calls, for attributing them to a method
		for _, f := range src.Files {
			calls[f.Path] = f.Calls
			for _, c := range f.Classes {
				if _, seen := classes[c.Name]; !seen && c.Name != "" {
					classes[c.Name] = classAt{c, f.Path}
				}
			}
		}
		for _, e := range m.Entries {
			if e.Container != container {
				continue
			}
			f := follow(e, classes, calls, componentOf, func(t string) string { return tableOf[container+"\x00"+t] })
			if len(f.Steps) > 0 {
				m.Flows = append(m.Flows, f)
			}
		}
	}
	sort.SliceStable(m.Flows, func(i, j int) bool { return m.Flows[i].Entry < m.Flows[j].Entry })
}

func follow(e archdoc.Entry, classes map[string]classAt, calls map[string][]archdoc.Call, componentOf map[string]string, table func(string) string) archdoc.Flow {
	flow := archdoc.Flow{Entry: e.ID}
	seen := map[string]bool{}
	visited := map[string]bool{}
	add := func(p archdoc.Participant) {
		if !seen[p.ID] {
			seen[p.ID] = true
			flow.Participants = append(flow.Participants, p)
		}
	}
	class := func(name string) archdoc.Participant {
		at := classes[name]
		return archdoc.Participant{ID: name, Name: name, Kind: "class", Component: componentOf[at.file]}
	}
	repeated := map[archdoc.Step]bool{}
	step := func(s archdoc.Step) bool {
		// The same call again from the same place in the flow says nothing new; drawn once.
		key := archdoc.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth}
		if repeated[key] {
			return true
		}
		repeated[key] = true
		if len(flow.Steps) == flowSteps {
			flow.Cut = true
			return false
		}
		flow.Steps = append(flow.Steps, s)
		return true
	}

	// method finds a method in a class or the classes it extends, and the class that declares it.
	var method func(cls, name string, depth int) (classAt, archdoc.Method, bool)
	method = func(cls, name string, depth int) (classAt, archdoc.Method, bool) {
		at, ok := classes[cls]
		if !ok || depth > 8 {
			return classAt{}, archdoc.Method{}, false
		}
		for _, m := range at.class.Methods {
			if m.Name == name {
				return at, m, true
			}
		}
		for _, b := range at.class.Extends {
			if found, m, ok := method(b, name, depth+1); ok {
				return found, m, ok
			}
		}
		return classAt{}, archdoc.Method{}, false
	}
	// typeOf is a field's declared type: a constructor parameter of the class or of a class it extends.
	var typeOf func(cls, field string, depth int) string
	typeOf = func(cls, field string, depth int) string {
		at, ok := classes[cls]
		if !ok || depth > 8 {
			return ""
		}
		for _, p := range at.class.Params {
			if p.Name == field {
				return p.Type
			}
		}
		for _, b := range at.class.Extends {
			if t := typeOf(b, field, depth+1); t != "" {
				return t
			}
		}
		return ""
	}

	var visit func(self string, at classAt, m archdoc.Method, depth int) bool
	visit = func(self string, at classAt, m archdoc.Method, depth int) bool {
		key := at.class.Name + "." + m.Name
		if visited[key] {
			return true
		}
		visited[key] = true

		// What the method does, in the order its lines do it.
		type act struct {
			line int
			do   func() bool
		}
		var acts []act
		for _, inv := range m.Invokes {
			inv := inv
			if quiet[inv.Method] {
				continue
			}
			acts = append(acts, act{inv.Prov.Line, func() bool {
				target := self
				if inv.Object != "" {
					target = typeOf(self, inv.Object, 0)
				}
				found, next, ok := method(target, inv.Method, 0)
				if !ok {
					return true // a library's method, or a field whose type is not in this code
				}
				add(class(target))
				if !step(archdoc.Step{From: self, To: target, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
					return false
				}
				if depth+1 < flowDepth {
					return visit(target, found, next, depth+1)
				}
				flow.Cut = true
				return true
			}})
		}
		for _, q := range m.Queries {
			q := q
			acts = append(acts, act{q.Prov.Line, func() bool {
				p := archdoc.Participant{ID: "table:" + q.Table, Name: q.Table, Kind: "table", Element: table(q.Table)}
				add(p)
				return step(archdoc.Step{From: self, To: p.ID, Call: q.Op, Depth: depth, Prov: q.Prov})
			}})
		}
		for _, c := range calls[at.file] {
			c := c
			if c.Prov.Line < m.Prov.Line || c.Prov.Line > m.EndLine {
				continue
			}
			acts = append(acts, act{c.Prov.Line, func() bool {
				add(archdoc.Participant{ID: "unresolved", Name: "an address computed at run time", Kind: "unresolved"})
				return step(archdoc.Step{From: self, To: "unresolved", Call: c.Callee, Depth: depth,
					Note: c.Callee + "(" + c.Target + ")", Prov: c.Prov})
			}})
		}
		sort.SliceStable(acts, func(i, j int) bool { return acts[i].line < acts[j].line })
		for _, a := range acts {
			if !a.do() {
				return false
			}
		}
		return true
	}

	handlerClass, handlerMethod := splitHandler(e.Handler)
	at, m, ok := method(handlerClass, handlerMethod, 0)
	if !ok {
		return flow
	}
	add(class(handlerClass))
	visit(handlerClass, at, m, 0)
	return flow
}

func splitHandler(h string) (string, string) {
	for i := len(h) - 1; i >= 0; i-- {
		if h[i] == '.' {
			return h[:i], h[i+1:]
		}
	}
	return h, ""
}
