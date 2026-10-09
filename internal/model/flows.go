package model

import (
	"path"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Flows (F-12): what happens when an entry is called — arc42's runtime view, from the code.
//
// A flow starts at a route's handler and follows what each method does to its own object:
// this.albumRepository.getAll(…) is a call to AlbumRepository.getAll, because the constructor
// declared albumRepository an AlbumRepository (resolved by name, D-4). Inherited fields and
// methods are found through the classes a class extends. A query builder's table — .selectFrom
// ('album') — is a step to that table; an HTTP call to a computed address is a step to an
// unresolved participant. What happens out of line is followed by name too: emit('AlbumInvite')
// continues in the methods @OnEvent({ name: 'AlbumInvite' }) marks, and an object named after a
// job — { name: JobName.AssetDelete } — is a step to that job, which has a flow of its own.
// It stops at a fixed depth and length, and says when it did.

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
		for _, f := range src.Files {
			for _, c := range f.Classes {
				if _, seen := classes[c.Name]; !seen && c.Name != "" {
					classes[c.Name] = classAt{c, f.Path}
				}
			}
		}
		files := map[string]archdoc.SourceFile{}
		for _, f := range src.Files {
			files[f.Path] = f
		}
		table := func(t string) string { return tableOf[container+"\x00"+t] }
		// What runs out of line, by the name that starts it: the methods an event reaches, and the
		// entry a queued job is.
		out := outOfLine{events: map[string][]handler{}, jobs: map[string]archdoc.Entry{}, constants: map[string]string{}}
		for _, f := range src.Files {
			for _, k := range f.Constants {
				out.constants[k.Name] = k.Value
			}
			for _, c := range f.Classes {
				for _, meth := range c.Methods {
					for _, d := range meth.Decorators {
						if name := d.Options["name"]; d.Name == "OnEvent" && name != "" && c.Name != "" {
							out.events[name] = append(out.events[name], handler{name, c.Name, meth.Name})
						}
					}
				}
			}
		}
		for _, e := range m.Entries {
			if e.Container == container && e.Kind == "job" && e.Method == "JOB" {
				out.jobs[e.Path] = e
			}
		}
		for _, e := range m.Entries {
			// A route, and a job a class handles, are followed; a page or a command has no handler
			// method to start from.
			if e.Container != container || (e.Kind != "http" && e.Kind != "job") {
				continue
			}
			var f archdoc.Flow
			if strings.HasSuffix(e.Prov.File, ".py") {
				f = followPython(e, classes, files, componentOf, table)
			} else {
				f = follow(e, classes, files, componentOf, table, out)
			}
			if len(f.Steps) > 0 {
				m.Flows = append(m.Flows, f)
			}
		}
	}
	sort.SliceStable(m.Flows, func(i, j int) bool { return m.Flows[i].Entry < m.Flows[j].Entry })
}

// outOfLine is what a flow reaches without calling it: an event's handlers, a queued job's entry.
type outOfLine struct {
	events    map[string][]handler
	jobs      map[string]archdoc.Entry
	constants map[string]string
}

type handler struct{ event, class, method string }

// listeners are the methods a call reaches by the event it names. Only a call that says it emits —
// emit, emitAsync, publish, dispatch — is read so: the same word handed to another method
// (serverSend('ConfigUpdate')) is a message to somewhere else, not an event here.
func (o outOfLine) listeners(inv archdoc.Invocation) []handler {
	if !strings.HasPrefix(inv.Method, "emit") && !strings.HasPrefix(inv.Method, "publish") && !strings.HasPrefix(inv.Method, "dispatch") {
		return nil
	}
	var out []handler
	for _, name := range inv.Args {
		out = append(out, o.events[name]...)
	}
	return out
}

func follow(e archdoc.Entry, classes map[string]classAt, files map[string]archdoc.SourceFile, componentOf map[string]string, table func(string) string, out outOfLine) archdoc.Flow {
	flow := archdoc.Flow{Entry: e.ID}
	known := map[string]archdoc.Participant{} // every participant met, whether or not it stays
	visited := map[string]bool{}
	repeated := map[archdoc.Step]bool{}
	meet := func(p archdoc.Participant) { known[p.ID] = p }
	class := func(name string) archdoc.Participant {
		return archdoc.Participant{ID: name, Name: name, Kind: "class", Component: componentOf[classes[name].file]}
	}
	module := func(file string) archdoc.Participant {
		name := path.Base(file)
		if i := strings.Index(name, "."); i > 0 {
			name = name[:i]
		}
		return archdoc.Participant{ID: file, Name: name, Kind: "module", Component: componentOf[file]}
	}
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
	// function finds a module's own function: in the file, or in a file it imports.
	own := func(file, name string) (archdoc.Method, bool) {
		for _, c := range files[file].Classes {
			if c.Name != "" {
				continue
			}
			for _, m := range c.Methods {
				if m.Name == name {
					return m, true
				}
			}
		}
		return archdoc.Method{}, false
	}
	function := func(file, name string) (string, archdoc.Method, bool) {
		if m, ok := own(file, name); ok {
			return file, m, true
		}
		for _, imp := range files[file].Imports {
			if imp.Target != "" {
				if m, ok := own(imp.Target, name); ok {
					return imp.Target, m, true
				}
			}
		}
		return "", archdoc.Method{}, false
	}

	// visit follows one method or function. self is the participant doing the calling — a class,
	// or a module's file — cls the class whose fields and methods this.… means, file where the
	// code is.
	var visit func(self, cls, file string, m archdoc.Method, depth int) bool
	visit = func(self, cls, file string, m archdoc.Method, depth int) bool {
		key := file + "\x00" + cls + "." + m.Name
		if visited[key] {
			return true
		}
		visited[key] = true

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
				if inv.Free {
					// A plain function: followed, and kept only if it leads somewhere — to a table,
					// to a call that leaves, to a class. A helper that only shapes data is not a step.
					target, next, ok := function(file, inv.Method)
					if !ok || depth+1 >= flowDepth {
						return true
					}
					mark, was := len(flow.Steps), flow.Cut
					meet(module(target))
					if !step(archdoc.Step{From: self, To: target, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
						return false
					}
					if len(flow.Steps) == mark {
						return true // the same call, already drawn from here
					}
					more := visit(target, "", target, next, depth+1)
					if !leads(flow.Steps[mark+1:], known) {
						for _, s := range flow.Steps[mark:] {
							delete(repeated, archdoc.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth})
						}
						flow.Steps, flow.Cut = flow.Steps[:mark], was
						return true
					}
					return more
				}
				if cls == "" {
					return true // a function has no object of its own to call through
				}
				if hs := out.listeners(inv); len(hs) > 0 {
					// An event: what happens next is in the methods that listen for it.
					for _, h := range hs {
						found, next, ok := method(h.class, h.method, 0)
						if !ok {
							continue
						}
						meet(class(h.class))
						if !step(archdoc.Step{From: self, To: h.class, Call: h.method, Depth: depth,
							Note: "on the event " + h.event + ", matched by name", Prov: inv.Prov}) {
							return false
						}
						if depth+1 >= flowDepth {
							flow.Cut = true
						} else if !visit(h.class, h.class, found.file, next, depth+1) {
							return false
						}
					}
					return true
				}
				target := cls
				to := self
				if inv.Object != "" {
					target = typeOf(cls, inv.Object, 0)
					to = target
				}
				found, next, ok := method(target, inv.Method, 0)
				if !ok {
					return true // a library's method, or a field whose type is not in this code
				}
				meet(class(target))
				if !step(archdoc.Step{From: self, To: to, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
					return false
				}
				if depth+1 < flowDepth {
					return visit(to, target, found.file, next, depth+1)
				}
				flow.Cut = true
				return true
			}})
		}
		for _, q := range m.Queries {
			q := q
			acts = append(acts, act{q.Prov.Line, func() bool {
				p := archdoc.Participant{ID: "table:" + q.Table, Name: q.Table, Kind: "table", Element: table(q.Table)}
				meet(p)
				return step(archdoc.Step{From: self, To: p.ID, Call: q.Op, Depth: depth, Prov: q.Prov})
			}})
		}
		for _, n := range m.Named {
			n := n
			name := n.Value
			if name == "" {
				name = out.constants[n.Expr]
			}
			job, ok := out.jobs[name]
			if !ok {
				continue
			}
			acts = append(acts, act{n.Prov.Line, func() bool {
				p := archdoc.Participant{ID: "job:" + name, Name: name, Kind: "job", Element: job.ID}
				meet(p)
				return step(archdoc.Step{From: self, To: p.ID, Call: "queues", Depth: depth,
					Note: "handled later by " + job.Handler + ", a flow of its own", Prov: n.Prov})
			}})
		}
		for _, c := range files[file].Calls {
			c := c
			if c.Prov.Line < m.Prov.Line || c.Prov.Line > m.EndLine {
				continue
			}
			acts = append(acts, act{c.Prov.Line, func() bool {
				meet(archdoc.Participant{ID: "unresolved", Name: "an address computed at run time", Kind: "unresolved"})
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
	meet(class(handlerClass))
	visit(handlerClass, handlerClass, at.file, m, 0)

	// The lifelines are the participants the kept steps name, in the order they first appear.
	seen := map[string]bool{}
	add := func(id string) {
		if p, ok := known[id]; ok && !seen[id] {
			seen[id] = true
			flow.Participants = append(flow.Participants, p)
		}
	}
	add(handlerClass)
	for _, s := range flow.Steps {
		add(s.From)
		add(s.To)
	}
	return flow
}

// leads reports whether steps reach beyond helper functions: a table, a call that leaves, a class,
// a job.
func leads(steps []archdoc.Step, known map[string]archdoc.Participant) bool {
	for _, s := range steps {
		if k := known[s.To].Kind; k == "table" || k == "unresolved" || k == "class" || k == "job" {
			return true
		}
	}
	return false
}

func splitHandler(h string) (string, string) {
	for i := len(h) - 1; i >= 0; i-- {
		if h[i] == '.' {
			return h[:i], h[i+1:]
		}
	}
	return h, ""
}

// followPython follows a Python route: its function, the functions it calls — of its own module,
// or of a module it imports (crud.create_user) — the methods it calls on an object whose class the
// code states (a typed parameter, a local built from a class, self and what __init__ gave it), and
// the tables their queries name, select(Item) reading item. A module is a participant, as a class is.
func followPython(e archdoc.Entry, classes map[string]classAt, files map[string]archdoc.SourceFile, componentOf map[string]string, table func(string) string) archdoc.Flow {
	flow := archdoc.Flow{Entry: e.ID}
	seen := map[string]bool{}
	visited := map[string]bool{}
	repeated := map[archdoc.Step]bool{}
	module := func(file string) archdoc.Participant {
		return archdoc.Participant{ID: file, Name: strings.TrimSuffix(path.Base(file), ".py"), Kind: "class", Component: componentOf[file]}
	}
	add := func(p archdoc.Participant) {
		if !seen[p.ID] {
			seen[p.ID] = true
			flow.Participants = append(flow.Participants, p)
		}
	}
	step := func(s archdoc.Step) bool {
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
	function := func(file, name string) (archdoc.Method, bool) {
		for _, c := range files[file].Classes {
			if c.Name != "" {
				continue
			}
			for _, m := range c.Methods {
				if m.Name == name {
					return m, true
				}
			}
		}
		return archdoc.Method{}, false
	}
	// where finds the function a call names: in the file, or in a module the file imports.
	where := func(file string, inv archdoc.Invocation) (string, archdoc.Method, bool) {
		if inv.Object == "" {
			if m, ok := function(file, inv.Method); ok {
				return file, m, true
			}
		}
		for _, imp := range files[file].Imports {
			if imp.Target == "" {
				continue
			}
			last := imp.Spec[strings.LastIndex(imp.Spec, ".")+1:]
			if inv.Object != "" && last != inv.Object {
				continue
			}
			if m, ok := function(imp.Target, inv.Method); ok {
				return imp.Target, m, true
			}
		}
		return "", archdoc.Method{}, false
	}
	// classIn is the class of this code a type as written names: Optional[UserRepository],
	// "UserRepository", repos.UserRepository.
	classIn := func(typ string) string {
		for _, word := range strings.FieldsFunc(typ, func(r rune) bool {
			return r != '_' && (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z')
		}) {
			if at, ok := classes[word]; ok && strings.HasSuffix(at.file, ".py") {
				return word
			}
		}
		return ""
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
			if found, m, ok := method(classIn(b), name, depth+1); ok {
				return found, m, ok
			}
		}
		return classAt{}, archdoc.Method{}, false
	}
	// fieldOf is the class of a field: what __init__ gave it, or the class body declares.
	var fieldOf func(cls, field string, depth int) string
	fieldOf = func(cls, field string, depth int) string {
		at, ok := classes[cls]
		if !ok || depth > 8 {
			return ""
		}
		for _, p := range at.class.Params {
			if p.Name == field {
				return classIn(p.Type)
			}
		}
		for _, f := range at.class.Fields {
			if f.Name == field {
				return classIn(f.Type)
			}
		}
		for _, b := range at.class.Extends {
			if t := fieldOf(classIn(b), field, depth+1); t != "" {
				return t
			}
		}
		return ""
	}

	// visit follows one function or method. self is the participant doing the calling — a module's
	// file, or a class — and cls the class self.… means, empty in a module's function.
	var visit func(self, cls, file string, m archdoc.Method, depth int) bool
	visit = func(self, cls, file string, m archdoc.Method, depth int) bool {
		key := file + "\x00" + cls + "." + m.Name
		if visited[key] {
			return true
		}
		visited[key] = true
		type act struct {
			line int
			do   func() bool
		}
		var acts []act
		for _, inv := range m.Invokes {
			inv := inv
			acts = append(acts, act{inv.Prov.Line, func() bool {
				// A method, where the code states the object's class.
				target := ""
				switch {
				case inv.Self && cls == "":
					return true
				case inv.Self && inv.Object == "":
					target = cls
				case inv.Self:
					target = fieldOf(cls, inv.Object, 0)
				case inv.Type != "":
					target = classIn(inv.Type)
				case inv.Object != "":
					target = classIn(inv.Object) // UserRepository.find(…), called on the class
				}
				if target != "" {
					found, next, ok := method(target, inv.Method, 0)
					if !ok {
						return true
					}
					add(archdoc.Participant{ID: target, Name: target, Kind: "class", Component: componentOf[classes[target].file]})
					if !step(archdoc.Step{From: self, To: target, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
						return false
					}
					if depth+1 < flowDepth {
						return visit(target, target, found.file, next, depth+1)
					}
					flow.Cut = true
					return true
				}
				if inv.Self {
					return true // a field whose class is not stated, or not of this code
				}
				to, next, ok := where(file, inv)
				if !ok {
					return true // a library's function, or a method of a value
				}
				add(module(to))
				if !step(archdoc.Step{From: self, To: to, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
					return false
				}
				if depth+1 < flowDepth {
					return visit(to, "", to, next, depth+1)
				}
				flow.Cut = true
				return true
			}})
		}
		for _, q := range m.Queries {
			q := q
			acts = append(acts, act{q.Prov.Line, func() bool {
				name := strings.ToLower(q.Table) // SQLModel names a table after its class
				p := archdoc.Participant{ID: "table:" + name, Name: name, Kind: "table", Element: table(name)}
				add(p)
				return step(archdoc.Step{From: self, To: p.ID, Call: q.Op, Depth: depth, Prov: q.Prov})
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

	file := e.Prov.File
	_, name := splitHandler(e.Handler)
	m, ok := function(file, name)
	if !ok {
		return flow
	}
	add(module(file))
	visit(file, "", file, m, 0)
	return flow
}
