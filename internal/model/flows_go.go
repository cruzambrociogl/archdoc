package model

import (
	"path"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Go flows: from a route's handler, a job's Work method or a command's function, through the
// functions and methods of the program it calls — tied by the types the code states
// (internal/code/golang_flow.go) — to the tables its SQL names, the jobs it queues and the
// requests it sends to an address made at run time.
//
// A type is a lifeline, as a class is; a package's functions share one, the package's. A call to
// another part is a step. A call within a part — a helper of the same package — is kept only if it
// leads somewhere beyond it: a table, a job, a request, another part. A table is kept where the container declares it, when it
// declares any: SQL names columns and functions in places a table could stand. A job another
// program works is still a step, and says which.

func goFlows(m *core.Model, src core.Source, container string, componentOf map[string]string, table func(string) string) []core.Flow {
	componentOf = componentsFor(src, container, componentOf) // a shared package is a part of each program
	funcs := map[string]core.Func{}
	calls := map[string][]core.Call{} // file → its requests to computed addresses
	for _, f := range src.Files {
		for _, fn := range f.Funcs {
			funcs[fn.Key] = fn
		}
		calls[f.Path] = f.Calls
	}
	hasTables := false
	for _, n := range m.Nodes {
		hasTables = hasTables || n.Kind == core.Table && n.Parent == container
	}
	jobs := map[string]core.Entry{} // a job's kind → its entry, in whichever container works it
	for _, e := range m.Entries {
		if e.Kind == "job" && e.Method == "JOB" {
			if _, seen := jobs[e.Path]; !seen || e.Container == container {
				jobs[e.Path] = e
			}
		}
	}
	g := goFollower{src: src, funcs: funcs, calls: calls, componentOf: componentOf, table: table, hasTables: hasTables, jobs: jobs, container: container}
	var out []core.Flow
	for _, e := range m.Entries {
		if e.Container != container || e.Func == "" || (e.Kind != "http" && e.Kind != "job" && e.Kind != "command") {
			continue
		}
		if f := g.follow(e); len(f.Steps) > 0 {
			out = append(out, f)
		}
	}
	return out
}

type goFollower struct {
	src         core.Source
	funcs       map[string]core.Func
	calls       map[string][]core.Call
	componentOf map[string]string
	table       func(string) string
	hasTables   bool
	jobs        map[string]core.Entry
	container   string
}

// beyond reports steps that reach out of a part: to a table, a job, a request that leaves, or
// another part's code.
func (g goFollower) beyond(steps []core.Step, component string, known map[string]core.Participant) bool {
	for _, s := range steps {
		p := known[s.To]
		if p.Kind == "table" || p.Kind == "job" || p.Kind == "unresolved" || p.Component != component {
			return true
		}
	}
	return false
}

// lifeline is who makes and takes a function's calls: its type, or its package.
func (g goFollower) lifeline(key string) core.Participant {
	dir, name, _ := strings.Cut(key, "#")
	if i := strings.Index(name, ".func@"); i >= 0 {
		name = name[:i]
	}
	pkg := path.Base(dir)
	if dir == g.src.Root && g.src.Module != "" {
		pkg = path.Base(g.src.Module)
	}
	component := g.componentOf[g.funcs[key].Prov.File]
	if recv, _, ok := strings.Cut(name, "."); ok {
		return core.Participant{ID: dir + "#" + recv, Name: pkg + "." + recv, Kind: "class", Component: component}
	}
	return core.Participant{ID: dir, Name: pkg, Kind: "module", Component: component}
}

// callName is what a step calls: the method's or the function's own name.
func callName(key string) string {
	_, name, _ := strings.Cut(key, "#")
	if i := strings.Index(name, ".func@"); i >= 0 {
		name = name[:i]
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func (g goFollower) follow(e core.Entry) core.Flow {
	flow := core.Flow{Entry: e.ID}
	known := map[string]core.Participant{}
	visited := map[string]bool{}
	// The same step again under the same call says nothing new; under another call it does — every
	// query of a sqlc Queries is one lifeline's, and which of them reads a table is the point.
	type stepKey struct {
		step   core.Step
		parent int
	}
	repeated := map[stepKey]bool{}
	parents := map[int]int{} // a step → the step it is made under, -1 at the top
	meet := func(p core.Participant) { known[p.ID] = p }
	step := func(s core.Step, parent int) bool {
		key := stepKey{core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth}, parent}
		if repeated[key] {
			return true
		}
		repeated[key] = true
		if len(flow.Steps) == flowSteps {
			flow.Cut = true
			return false
		}
		parents[len(flow.Steps)] = parent
		flow.Steps = append(flow.Steps, s)
		return true
	}
	drop := func(mark int, was bool) {
		for i, s := range flow.Steps[mark:] {
			delete(repeated, stepKey{core.Step{From: s.From, To: s.To, Call: s.Call, Depth: s.Depth}, parents[mark+i]})
			delete(parents, mark+i)
		}
		flow.Steps, flow.Cut = flow.Steps[:mark], was
	}

	var visit func(key string, depth, parent int) bool
	visit = func(key string, depth, parent int) bool {
		fn, ok := g.funcs[key]
		if !ok || visited[key] {
			return true
		}
		visited[key] = true
		self := g.lifeline(key)
		meet(self)

		type act struct {
			line int
			do   func() bool
		}
		var acts []act
		for _, c := range fn.Calls {
			c := c
			if _, ok := g.funcs[c.To]; !ok {
				continue
			}
			acts = append(acts, act{c.Prov.Line, func() bool {
				to := g.lifeline(c.To)
				meet(to)
				mark, was := len(flow.Steps), flow.Cut
				if !step(core.Step{From: self.ID, To: to.ID, Call: callName(c.To), Depth: depth, Prov: c.Prov}, parent) {
					return false
				}
				if len(flow.Steps) == mark {
					return true // the same call, already drawn from here
				}
				if depth+1 >= flowDepth {
					if to.Component == self.Component {
						drop(mark, was) // a helper of its own part, too deep to follow: not a step
					} else {
						flow.Cut = true
					}
					return true
				}
				more := visit(c.To, depth+1, mark)
				if more && to.Component == self.Component && !g.beyond(flow.Steps[mark+1:], self.Component, known) {
					drop(mark, was) // a helper of its own part that leads nowhere beyond it
				}
				return more
			}})
		}
		for _, q := range fn.Queries {
			q := q
			if g.hasTables && g.table(q.Table) == "" {
				continue
			}
			acts = append(acts, act{q.Prov.Line, func() bool {
				p := core.Participant{ID: "table:" + q.Table, Name: q.Table, Kind: "table", Element: g.table(q.Table)}
				meet(p)
				return step(core.Step{From: self.ID, To: p.ID, Call: q.Op, Depth: depth, Prov: q.Prov}, parent)
			}})
		}
		for _, q := range fn.Queues {
			q := q
			acts = append(acts, act{q.Prov.Line, func() bool {
				p := core.Participant{ID: "job:" + q.Value, Name: q.Value, Kind: "job"}
				note := "a job no program here takes"
				if j, ok := g.jobs[q.Value]; ok {
					p.Element = j.ID
					note = "handled later by " + j.Handler + ", a flow of its own"
					if j.Container != g.container {
						_, other, _ := strings.Cut(j.Container, ":")
						note = "handled later by " + other + "'s " + j.Handler + ", a flow of its own"
					}
				}
				meet(p)
				return step(core.Step{From: self.ID, To: p.ID, Call: "queues", Depth: depth, Note: note, Prov: q.Prov}, parent)
			}})
		}
		for _, c := range g.calls[fn.Prov.File] {
			c := c
			if c.Prov.Line < fn.Prov.Line || c.Prov.Line > fn.EndLine {
				continue
			}
			acts = append(acts, act{c.Prov.Line, func() bool {
				meet(core.Participant{ID: "unresolved", Name: "an address computed at run time", Kind: "unresolved"})
				return step(core.Step{From: self.ID, To: "unresolved", Call: c.Callee, Depth: depth,
					Note: c.Callee + "(" + c.Target + ")", Prov: c.Prov}, parent)
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

	start := g.lifeline(e.Func)
	visit(e.Func, 0, -1)

	seen := map[string]bool{}
	add := func(id string) {
		if p, ok := known[id]; ok && !seen[id] {
			seen[id] = true
			flow.Participants = append(flow.Participants, p)
		}
	}
	add(start.ID)
	for _, s := range flow.Steps {
		add(s.From)
		add(s.To)
	}
	return flow
}
