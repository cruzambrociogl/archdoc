package model

import (
	"path"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
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
// What is queued when a job is done is followed by name as well: a switch on job names whose case for
// a job queues another — JobService.onDone's case JobName.SidecarCheck queues AssetExtractMetadata — ends
// that job's flow with the step, past any cut; so are the jobs a cut flow's handler queues, and the events
// it emits with what their listeners queue. A call a class makes on itself is kept, as a function's
// is, only if it leads somewhere: to another class, a table, a job, a call that leaves.
// It stops at a fixed depth and length, and says when it did. A command a class runs is followed the
// same way, from its run method. A page has no handler method: its flow is what it calls through a
// generated client (pageCalls, api.go).

const (
	flowDepth = 4
	flowSteps = 40
)

// Calls that say nothing about what a flow does: logging.
var quiet = map[string]bool{"log": true, "debug": true, "verbose": true, "warn": true, "error": true, "info": true, "setContext": true, "fatal": true, "trace": true}

type classAt struct {
	class core.Class
	file  string
}

func flows(m *core.Model, sources []core.Source, componentOf map[string]string) {
	tableOf := map[string]string{} // table name → node ID, per container
	for _, n := range m.Nodes {
		if n.Kind == core.Table {
			tableOf[n.Parent+"\x00"+n.Name] = n.ID
		}
	}
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == core.Application {
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
		files := map[string]core.SourceFile{}
		for _, f := range src.Files {
			files[f.Path] = f
		}
		table := func(t string) string { return tableOf[container+"\x00"+t] }
		impl := implementations(classes)
		// What runs out of line, by the name that starts it: the methods an event reaches, and the
		// entry a queued job is.
		out := outOfLine{events: map[string][]handler{}, jobs: map[string]core.Entry{}, constants: map[string]string{}}
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
		after := queuedAfter(src, out)
		for _, e := range m.Entries {
			// A route, a job a class handles and a command a class runs are followed; a page has no
			// handler method to start from.
			if e.Container != container || (e.Kind != "http" && e.Kind != "job" && e.Kind != "command") {
				continue
			}
			var f core.Flow
			if strings.HasSuffix(e.Prov.File, ".py") {
				f = followPython(e, classes, files, componentOf, table)
			} else {
				f = follow(e, classes, files, componentOf, table, out, impl)
			}
			if e.Kind == "job" {
				f = whenDone(f, after[e.Path], classes, componentOf)
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
	jobs      map[string]core.Entry
	constants map[string]string
}

type handler struct{ event, class, method string }

// queued is a job a switch case queues: in the case for another job, in a method of a class.
type queued struct {
	class, method, label, name string
	job                        core.Entry
	prov                       core.Provenance
}

// queuedAfter is what a switch on job names queues in each job's case, by the job the case is for.
func queuedAfter(src core.Source, out outOfLine) map[string][]queued {
	after := map[string][]queued{}
	for _, f := range src.Files {
		for _, c := range f.Classes {
			for _, meth := range c.Methods {
				for _, n := range meth.Named {
					name := n.Value
					if name == "" {
						name = out.constants[n.Expr]
					}
					job, ok := out.jobs[name]
					if !ok || c.Name == "" {
						continue
					}
					for _, label := range n.Case {
						done := label
						if v, ok := out.constants[label]; ok {
							done = v
						}
						if _, ok := out.jobs[done]; ok {
							after[done] = append(after[done], queued{c.Name, meth.Name, label, name, job, n.Prov})
						}
					}
				}
			}
		}
	}
	return after
}

// whenDone ends a job's flow with what a switch case queues for it, each job once, past any cut: what a
// job leads to matters more than the steps it takes to get there.
func whenDone(f core.Flow, qs []queued, classes map[string]classAt, componentOf map[string]string) core.Flow {
	met := map[string]bool{}
	for _, p := range f.Participants {
		met[p.ID] = true
	}
	drawn := map[string]bool{}
	for _, q := range qs {
		if drawn[q.name] {
			continue
		}
		drawn[q.name] = true
		if !met[q.class] {
			met[q.class] = true
			f.Participants = append(f.Participants, core.Participant{ID: q.class, Name: q.class, Kind: "class", Component: componentOf[classes[q.class].file]})
		}
		if !met["job:"+q.name] {
			met["job:"+q.name] = true
			f.Participants = append(f.Participants, core.Participant{ID: "job:" + q.name, Name: q.name, Kind: "job", Element: q.job.ID})
		}
		f.Steps = append(f.Steps, core.Step{From: q.class, To: "job:" + q.name, Call: "queues",
			Note: "in the case for " + q.label + " of " + q.class + "." + q.method + ", matched by name; handled later by " +
				q.job.Handler + ", a flow of its own", Prov: q.prov})
	}
	return f
}

// listeners are the methods a call reaches by the event it names. Only a call that says it emits —
// emit, emitAsync, publish, dispatch — is read so: the same word handed to another method
// (serverSend('ConfigUpdate')) is a message to somewhere else, not an event here.
func (o outOfLine) listeners(inv core.Invocation) []handler {
	if !strings.HasPrefix(inv.Method, "emit") && !strings.HasPrefix(inv.Method, "publish") && !strings.HasPrefix(inv.Method, "dispatch") {
		return nil
	}
	var out []handler
	for _, name := range inv.Args {
		out = append(out, o.events[name]...)
	}
	return out
}

func follow(e core.Entry, classes map[string]classAt, files map[string]core.SourceFile, componentOf map[string]string, table func(string) string, out outOfLine, impl map[string]string) core.Flow {
	flow := core.Flow{Entry: e.ID}
	known := map[string]core.Participant{} // every participant met, whether or not it stays
	visited := map[string]bool{}
	repeated := map[core.Step]bool{}
	meet := func(p core.Participant) { known[p.ID] = p }
	class := func(name string) core.Participant {
		return core.Participant{ID: name, Name: name, Kind: "class", Component: componentOf[classes[name].file]}
	}
	module := func(file string) core.Participant {
		name := path.Base(file)
		if i := strings.Index(name, "."); i > 0 {
			name = name[:i]
		}
		return core.Participant{ID: file, Name: name, Kind: "module", Component: componentOf[file]}
	}
	step := func(s core.Step) bool {
		// The same call again from the same place in the flow says nothing new; drawn once.
		key := core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth}
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
	var method func(cls, name string, depth int) (classAt, core.Method, bool)
	method = func(cls, name string, depth int) (classAt, core.Method, bool) {
		at, ok := classes[cls]
		if !ok || depth > 8 {
			return classAt{}, core.Method{}, false
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
		return classAt{}, core.Method{}, false
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
	own := func(file, name string) (core.Method, bool) {
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
		return core.Method{}, false
	}
	function := func(file, name string) (string, core.Method, bool) {
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
		return "", core.Method{}, false
	}

	// leaf reports a method that can lead nowhere: it names no table and no job, makes no call that
	// leaves, emits nothing, and calls nothing of this code. cls is the class it is called on.
	leaf := func(cls string, at classAt, m core.Method) bool {
		if len(m.Queries) > 0 {
			return false
		}
		for _, n := range m.Named {
			name := n.Value
			if name == "" {
				name = out.constants[n.Expr]
			}
			if _, ok := out.jobs[name]; ok {
				return false
			}
		}
		for _, c := range files[at.file].Calls {
			if c.Prov.Line >= m.Prov.Line && c.Prov.Line <= m.EndLine {
				return false
			}
		}
		for _, inv := range m.Invokes {
			if quiet[inv.Method] {
				continue
			}
			if inv.Free {
				if _, _, ok := function(at.file, inv.Method); ok {
					return false
				}
				continue
			}
			if len(out.listeners(inv)) > 0 {
				return false
			}
			target := cls
			if inv.Object != "" {
				target = typeOf(cls, inv.Object, 0)
				if i, ok := impl[target]; ok {
					target = i
				}
			}
			if _, _, ok := method(target, inv.Method, 0); ok {
				return false
			}
		}
		return true
	}

	// visit follows one method or function. self is the participant doing the calling — a class,
	// or a module's file — cls the class whose fields and methods this.… means, file where the
	// code is.
	var visit func(self, cls, file string, m core.Method, depth int) bool
	visit = func(self, cls, file string, m core.Method, depth int) bool {
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
					if !step(core.Step{From: self, To: target, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
						return false
					}
					if len(flow.Steps) == mark {
						return true // the same call, already drawn from here
					}
					more := visit(target, "", target, next, depth+1)
					if !leads(flow.Steps[mark+1:], known) {
						for _, s := range flow.Steps[mark:] {
							delete(repeated, core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth})
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
						if !step(core.Step{From: self, To: h.class, Call: h.method, Depth: depth,
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
					if i, ok := impl[target]; ok {
						target = i // an interface it is given, and the one class that implements it
					}
					to = target
				}
				found, next, ok := method(target, inv.Method, 0)
				if !ok {
					return true // a library's method, or a field whose type is not in this code
				}
				meet(class(target))
				mark, was := len(flow.Steps), flow.Cut
				if !step(core.Step{From: self, To: to, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
					return false
				}
				if depth+1 >= flowDepth {
					if inv.Object == "" && leaf(target, found, next) {
						// Too deep to follow, and plainly leading nowhere: not a step, and no cut.
						delete(repeated, core.Step{From: self, To: to, Call: inv.Method, Depth: depth})
						flow.Steps, flow.Cut = flow.Steps[:mark], was
						return true
					}
					flow.Cut = true
					return true
				}
				more := visit(to, target, found.file, next, depth+1)
				if more && inv.Object == "" && len(flow.Steps) > mark && !leadsOut(flow.Steps[mark+1:], to) {
					// A call on itself that only shapes data — this.parseFloat(…) — is not a step.
					for _, s := range flow.Steps[mark:] {
						delete(repeated, core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth})
					}
					flow.Steps, flow.Cut = flow.Steps[:mark], was
				}
				return more
			}})
		}
		for _, q := range m.Queries {
			q := q
			acts = append(acts, act{q.Prov.Line, func() bool {
				p := core.Participant{ID: "table:" + q.Table, Name: q.Table, Kind: "table", Element: table(q.Table)}
				meet(p)
				return step(core.Step{From: self, To: p.ID, Call: q.Op, Depth: depth, Prov: q.Prov})
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
				p := core.Participant{ID: "job:" + name, Name: name, Kind: "job", Element: job.ID}
				meet(p)
				return step(core.Step{From: self, To: p.ID, Call: "queues", Depth: depth,
					Note: "handled later by " + job.Handler + ", a flow of its own", Prov: n.Prov})
			}})
		}
		for _, c := range files[file].Calls {
			c := c
			if c.Prov.Line < m.Prov.Line || c.Prov.Line > m.EndLine {
				continue
			}
			acts = append(acts, act{c.Prov.Line, func() bool {
				meet(core.Participant{ID: "unresolved", Name: "an address computed at run time", Kind: "unresolved"})
				return step(core.Step{From: self, To: "unresolved", Call: c.Callee, Depth: depth,
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

	// handOffs keeps what a cut flow's handler hands on, past the cut: the jobs it queues, and the events it
	// emits with the jobs their listeners queue. Where a flow leads matters more than the steps on the way.
	handOffs := func(self string, m core.Method) {
		queues := func(from string, n core.Mention, depth int) []core.Step {
			name := n.Value
			if name == "" {
				name = out.constants[n.Expr]
			}
			job, ok := out.jobs[name]
			if !ok {
				return nil
			}
			meet(core.Participant{ID: "job:" + name, Name: name, Kind: "job", Element: job.ID})
			return []core.Step{{From: from, To: "job:" + name, Call: "queues", Depth: depth,
				Note: "kept past the cut; handled later by " + job.Handler + ", a flow of its own", Prov: n.Prov}}
		}
		// In the handler's order: each event with what its listeners queue after it.
		type handOff struct {
			line  int
			steps []core.Step
		}
		var hs []handOff
		for _, n := range m.Named {
			hs = append(hs, handOff{n.Prov.Line, queues(self, n, 0)})
		}
		for _, inv := range m.Invokes {
			for _, h := range out.listeners(inv) {
				_, next, ok := method(h.class, h.method, 0)
				if !ok {
					continue
				}
				meet(class(h.class))
				steps := []core.Step{{From: self, To: h.class, Call: h.method,
					Note: "kept past the cut; on the event " + h.event + ", matched by name", Prov: inv.Prov}}
				for _, n := range next.Named {
					steps = append(steps, queues(h.class, n, 1)...)
				}
				hs = append(hs, handOff{inv.Prov.Line, steps})
			}
		}
		sort.SliceStable(hs, func(i, j int) bool { return hs[i].line < hs[j].line })
		for _, h := range hs {
			for _, s := range h.steps {
				if key := (core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth}); !repeated[key] {
					repeated[key] = true
					flow.Steps = append(flow.Steps, s)
				}
			}
		}
	}

	handlerClass, handlerMethod := splitHandler(e.Handler)
	at, m, ok := method(handlerClass, handlerMethod, 0)
	if !ok {
		return flow
	}
	meet(class(handlerClass))
	visit(handlerClass, handlerClass, at.file, m, 0)
	if flow.Cut {
		handOffs(handlerClass, m)
	}

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
func leads(steps []core.Step, known map[string]core.Participant) bool {
	for _, s := range steps {
		if k := known[s.To].Kind; k == "table" || k == "unresolved" || k == "class" || k == "job" {
			return true
		}
	}
	return false
}

// leadsOut reports whether steps reach beyond the participant making them: another class or module, a
// table, a job, a call that leaves.
func leadsOut(steps []core.Step, self string) bool {
	for _, s := range steps {
		if s.To != self {
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
func followPython(e core.Entry, classes map[string]classAt, files map[string]core.SourceFile, componentOf map[string]string, table func(string) string) core.Flow {
	flow := core.Flow{Entry: e.ID}
	seen := map[string]bool{}
	visited := map[string]bool{}
	repeated := map[core.Step]bool{}
	module := func(file string) core.Participant {
		return core.Participant{ID: file, Name: strings.TrimSuffix(path.Base(file), ".py"), Kind: "class", Component: componentOf[file]}
	}
	add := func(p core.Participant) {
		if !seen[p.ID] {
			seen[p.ID] = true
			flow.Participants = append(flow.Participants, p)
		}
	}
	step := func(s core.Step) bool {
		key := core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth}
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
	function := func(file, name string) (core.Method, bool) {
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
		return core.Method{}, false
	}
	// where finds the function a call names: in the file, or in a module the file imports.
	where := func(file string, inv core.Invocation) (string, core.Method, bool) {
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
		return "", core.Method{}, false
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
	var method func(cls, name string, depth int) (classAt, core.Method, bool)
	method = func(cls, name string, depth int) (classAt, core.Method, bool) {
		at, ok := classes[cls]
		if !ok || depth > 8 {
			return classAt{}, core.Method{}, false
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
		return classAt{}, core.Method{}, false
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
	var visit func(self, cls, file string, m core.Method, depth int) bool
	visit = func(self, cls, file string, m core.Method, depth int) bool {
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
					add(core.Participant{ID: target, Name: target, Kind: "class", Component: componentOf[classes[target].file]})
					if !step(core.Step{From: self, To: target, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
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
				if !step(core.Step{From: self, To: to, Call: inv.Method, Depth: depth, Prov: inv.Prov}) {
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
				p := core.Participant{ID: "table:" + name, Name: name, Kind: "table", Element: table(name)}
				add(p)
				return step(core.Step{From: self, To: p.ID, Call: q.Op, Depth: depth, Prov: q.Prov})
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
