package semantic

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// What the two requests of Interpret carry: the seeds for the model to group (stage 1), and a summary
// built from its groups for the model to explain (stage 2). Facts only — names, paths, routes with
// the summaries their code states, tables and columns, what each flow reaches — never a line of code.

const (
	stepped  = 4  // flows given step by step: the widest of the largest features
	maxSteps = 60 // a stepped flow's own steps, at most
	later    = 20 // background steps a stepped flow is given at most
	trees    = 3  // job trees drawn whole: the deepest
	drawnBy  = 8  // a container, or a kind in one, with this many entry seeds is too large to be one purpose
	sweep    = "QueueAll"
)

// clip shortens text to n characters, its last an ellipsis.
func clip(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return string([]rune(text)[:n-1]) + "…"
}

func (s *seeding) entryLabel(e core.Entry) string {
	what := e.Path
	if e.Kind == "http" {
		what = strings.TrimSpace(e.Method + " " + e.Path)
	} else if what == "" {
		what = e.ID
	}
	if e.Summary != "" {
		what += " — " + clip(e.Summary, 60)
	}
	return what
}

// reachLine says what a set of entry points' flows reach: tables with the first letter of each call
// on them, the jobs they queue, the components they pass through most, and the seeds their pages call.
func (s *seeding) reachLine(members []string) string {
	tables := map[string]map[string]bool{}
	jobs := map[string]bool{}
	comps, called := newTally(), newTally()
	own := map[string]bool{}
	for _, x := range members {
		own[s.seedOf[x]] = true
	}
	for _, x := range members {
		r := s.reach[x]
		if r == nil {
			continue
		}
		for t, calls := range r.tables {
			for c := range calls {
				add(tables, t, c)
			}
		}
		for j := range r.jobs {
			jobs[j] = true
		}
		for _, c := range r.components {
			comps.add(short(c), 1)
		}
		for _, route := range sortedKeys(r.routes) {
			if sd, ok := s.seedOf[route]; ok && !own[sd] {
				called.add(sd, 1)
			}
		}
	}
	var out []string
	if len(tables) > 0 {
		var ts []string
		for i, t := range sortedKeys(tables) {
			if i == 14 {
				break
			}
			var letters []string
			for c := range tables[t] {
				letters = append(letters, c[:1])
			}
			sort.Strings(letters)
			ts = append(ts, t+"("+strings.Join(letters, "")+")")
		}
		out = append(out, "tables "+strings.Join(ts, ", "))
	}
	if len(jobs) > 0 {
		js := sortedKeys(jobs)
		out = append(out, "queues "+strings.Join(js[:min(10, len(js))], ", "))
	}
	if comps.size() > 0 {
		out = append(out, "through "+strings.Join(comps.top(6), ", "))
	}
	if called.size() > 0 {
		var cs []string
		for _, k := range called.top(6) {
			cs = append(cs, fmt.Sprintf("%s x%d", k, called.count(k)))
		}
		out = append(out, "its pages also call routes of "+strings.Join(cs, ", "))
	}
	return strings.Join(out, "; ")
}

// groupInput is stage 1's request: every seed, with what it holds and reaches.
func (s *seeding) groupInput() string {
	m := s.m
	out := []string{"# " + m.Name, "", "## Containers"}
	for _, n := range m.Nodes {
		if containerish(n.Kind) {
			tech := ""
			if n.Technology != "" {
				tech = ", " + n.Technology
			}
			out = append(out, fmt.Sprintf("- %s (%s%s)", n.Name, n.Kind, tech))
		}
	}
	out = append(out, "\n## Table seeds (columns; * a key; -> a reference)")
	tnodes := map[string]core.Node{}
	for _, n := range m.Nodes {
		if n.Kind == core.Table {
			tnodes[n.Name] = n
		}
	}
	for _, id := range s.tableIDs {
		t := s.tables[id]
		var cols []string
		for _, c := range tnodes[t.Key].Columns {
			col := c.Name
			if c.Primary {
				col += "*"
			}
			if ref, ok := s.nodes[c.References]; ok && c.References != "" {
				col += "->" + ref.Name
			}
			cols = append(cols, col)
		}
		var extra []string
		for _, x := range t.Members {
			if x != t.Key {
				extra = append(extra, x)
			}
		}
		with := ""
		if len(extra) > 0 {
			with = " (with " + strings.Join(extra, ", ") + ")"
		}
		out = append(out, fmt.Sprintf("%s%s: %s", id, with, strings.Join(cols[:min(16, len(cols))], ", ")))
	}
	if len(s.areas) > 0 {
		out = append(out, fmt.Sprintf("\n## Component seeds of %s (files; what each uses most)", short(s.areaOf)))
		uses := map[string]*tally{}
		for _, e := range m.Edges {
			if e.Label == "uses" && strings.HasPrefix(e.From, "cmp:") && strings.HasPrefix(e.To, "cmp:") {
				if uses[e.From] == nil {
					uses[e.From] = newTally()
				}
				uses[e.From].add(part(e.To), max(e.Weight, 1))
			}
		}
		for _, id := range s.areaIDs {
			a := s.areas[id]
			var files []string
			inner := map[string]bool{}
			for _, c := range a.Members {
				files = append(files, s.nodes[c].Files...)
				inner[part(c)] = true
			}
			var shown []string
			for _, f := range files[:min(3, len(files))] {
				shown = append(shown, f[strings.LastIndex(f, "/")+1:])
			}
			names := strings.Join(shown, ", ")
			if len(files) > 3 {
				names += fmt.Sprintf(", +%d", len(files)-3)
			}
			used := newTally()
			for _, c := range a.Members {
				if t := uses[c]; t != nil {
					for _, k := range t.order {
						if !inner[k] {
							used.add(k, t.n[k])
						}
					}
				}
			}
			parts := ""
			if len(a.Members) > 1 {
				parts = fmt.Sprintf(" [%d parts: %s; joined: %s]", len(a.Members), strings.Join(sortedKeys(inner), ", "), strings.Join(a.Why, ", "))
			}
			line := fmt.Sprintf("%s%s: %d files (%s)", id, parts, len(files), names)
			if used.size() > 0 {
				line += "; uses " + strings.Join(used.top(5), ", ")
			}
			out = append(out, line)
		}
	}
	out = append(out, "\n## Entry-point seeds (members, with the summary the code states; what their flows reach)")
	for _, id := range s.seedIDs {
		e := s.seeds[id]
		var order []string
		byKey := map[string][]string{}
		for _, x := range e.Members {
			k := entryKey(s.entries[x])
			if _, ok := byKey[k]; !ok {
				order = append(order, k)
			}
			byKey[k] = append(byKey[k], x)
		}
		var picked []string
		for row := 0; row < 8 && len(picked) < 8; row++ {
			for _, k := range order {
				if row < len(byKey[k]) && len(picked) < 8 {
					picked = append(picked, byKey[k][row])
				}
			}
		}
		var labels []string
		for _, x := range picked {
			labels = append(labels, s.entryLabel(s.entries[x]))
		}
		shown := strings.Join(labels, "; ")
		if len(e.Members) > 8 {
			shown += fmt.Sprintf("; +%d more", len(e.Members)-8)
		}
		why := ""
		if len(e.Joined) > 0 {
			why = "; joined: " + strings.Join(e.Joined, ", ")
		}
		line := fmt.Sprintf("%s [%s, %d%s]: %s", id, e.Key, len(e.Members), why, shown)
		if r := s.reachLine(e.Members); r != "" {
			line += " || " + r
		}
		out = append(out, line)
	}
	if len(s.chain) > 0 {
		out = append(out, "\n## Background work: what each job queues next")
		for _, j := range sortedKeys(s.chain) {
			out = append(out, j+" -> "+strings.Join(s.chain[j], ", "))
		}
	}
	return strings.Join(out, "\n")
}

func containerish(k core.Kind) bool {
	return k == core.Application || k == core.Datastore || k == core.External || k == core.Actor
}

// expand is a group's members: what its seeds hold.
func (s *seeding) expand(kind string, seeds []string) []string {
	var out []string
	for _, sd := range seeds {
		switch kind {
		case "concepts":
			out = append(out, s.tables[sd].Members...)
		case "areas":
			out = append(out, s.areas[sd].Members...)
		default:
			out = append(out, s.seeds[sd].Members...)
		}
	}
	return out
}

// width ranks flows: what they reach — tables, jobs, calls archdoc could not resolve — then length.
func width(f core.Flow) [2]int {
	n := 0
	for _, p := range f.Participants {
		if p.Kind == "table" || p.Kind == "job" || p.Kind == "unresolved" {
			n++
		}
	}
	return [2]int{n, len(f.Steps)}
}

func wider(a, b [2]int) bool { return a[0] > b[0] || (a[0] == b[0] && a[1] > b[1]) }

// givenStep is one numbered step of a flow as stage 2 was given it: what it touches, and whether it
// happens later, in the background.
type givenStep struct {
	elements   []string
	background bool
}

// summary is stage 2's request, from stage 1's groups; and, for each flow given step by step, its
// steps as numbered there.
func (s *seeding) summary(g grouping) (string, map[string][]givenStep) {
	m := s.m
	flows := map[string]core.Flow{}
	for _, f := range m.Flows {
		flows[f.Entry] = f
	}
	out := []string{"# " + m.Name, "", "## Containers and outside systems"}
	for _, n := range m.Nodes {
		if containerish(n.Kind) {
			line := fmt.Sprintf("[%s] %s: %s", n.ID, n.Kind, n.Name)
			if n.Technology != "" {
				line += " (" + n.Technology + ")"
			}
			out = append(out, line)
		}
	}
	out = append(out, "\n## How containers connect")
	for _, e := range m.Edges {
		if !structural(e.Label) {
			out = append(out, fmt.Sprintf("[%s>%s] %s %s %s", e.From, e.To, short(e.From), e.Label, short(e.To)))
		}
	}
	byC := map[string][]core.Package{}
	for _, p := range m.Dependencies {
		if p.Imports > 0 {
			byC[p.Container] = append(byC[p.Container], p)
		}
	}
	if len(byC) > 0 {
		out = append(out, "\n## Packages each container imports most")
		for _, c := range sortedKeys(byC) {
			ps := append([]core.Package(nil), byC[c]...)
			sort.SliceStable(ps, func(i, j int) bool { return ps[i].Imports > ps[j].Imports })
			var items []string
			for _, p := range ps[:min(8, len(ps))] {
				items = append(items, fmt.Sprintf("%s x%d", p.Name, p.Imports))
			}
			out = append(out, short(c)+": "+strings.Join(items, ", "))
		}
	}

	if len(g.Concepts) > 0 {
		out = append(out, "\n## Concepts (archdoc's grouping of the tables, named by a model)")
		conceptOf := map[string]string{}
		for _, c := range g.Concepts {
			for _, t := range s.expand("concepts", c.Seeds) {
				conceptOf[t] = c.Name
			}
		}
		links := map[string]map[string]bool{}
		for _, n := range m.Nodes {
			if n.Kind != core.Table {
				continue
			}
			for _, col := range n.Columns {
				ref, ok := s.nodes[col.References]
				if !ok || col.References == "" {
					continue
				}
				if to := conceptOf[ref.Name]; to != "" && to != conceptOf[n.Name] {
					add(links, conceptOf[n.Name], to)
				}
			}
		}
		for _, c := range g.Concepts {
			var ts []string
			for _, t := range s.expand("concepts", c.Seeds) {
				ts = append(ts, "[table:"+t+"]")
			}
			line := fmt.Sprintf("- %s: %s Tables: %s", c.Name, c.Says, strings.Join(ts, ", "))
			if to := sortedKeys(links[c.Name]); len(to) > 0 {
				line += ". References " + strings.Join(to, ", ") + "."
			}
			out = append(out, line)
		}
	}

	if len(g.Areas) > 0 {
		out = append(out, fmt.Sprintf("\n## Areas of %s (archdoc's grouping of its components, named by a model)", short(s.areaOf)))
		areaOf := map[string]string{}
		for _, a := range g.Areas {
			cs := s.expand("areas", a.Seeds)
			var ids []string
			for _, x := range cs {
				areaOf[x] = a.Name
				ids = append(ids, "["+x+"]")
			}
			out = append(out, fmt.Sprintf("- %s: %s Components: %s", a.Name, a.Says, strings.Join(ids, ", ")))
		}
		between := newTally()
		var fanOrder []string
		fanIn := map[string]map[string]bool{}
		for _, e := range m.Edges {
			if e.Label != "uses" || areaOf[e.From] == "" || areaOf[e.To] == "" {
				continue
			}
			if fanIn[e.To] == nil {
				fanOrder = append(fanOrder, e.To)
			}
			add(fanIn, e.To, e.From)
			if areaOf[e.From] != areaOf[e.To] {
				between.add(areaOf[e.From]+"\x00"+areaOf[e.To], max(e.Weight, 1))
			}
		}
		var bs []string
		for _, k := range between.top(15) {
			a, b, _ := strings.Cut(k, "\x00")
			bs = append(bs, fmt.Sprintf("%s -> %s x%d", a, b, between.count(k)))
		}
		out = append(out, "\nUses between areas (imports): "+strings.Join(bs, "; "))
		sort.SliceStable(fanOrder, func(i, j int) bool { return len(fanIn[fanOrder[i]]) > len(fanIn[fanOrder[j]]) })
		var most []string
		for _, x := range fanOrder[:min(8, len(fanOrder))] {
			most = append(most, fmt.Sprintf("[%s] %d", x, len(fanIn[x])))
		}
		out = append(out, "Most used components (by how many components use them): "+strings.Join(most, ", "))
	}

	out = append(out, "\n## Systems (archdoc's grouping of the entry points, named by a model; what their flows reach)")
	order := g.largestFirst(s)
	for _, f := range order {
		es := s.expand("systems", f.Seeds)
		kinds := newTally()
		for _, x := range es {
			kinds.add(s.entries[x].Kind, 1)
		}
		var ks []string
		for _, k := range kinds.top(0) {
			ks = append(ks, fmt.Sprintf("%d %s", kinds.count(k), k))
		}
		ex := s.widestFirst(es)
		var exs []string
		for _, x := range ex[:min(4, len(ex))] {
			item := "[" + x + "]"
			if sum := s.entries[x].Summary; sum != "" {
				item += " " + clip(sum, 50)
			}
			exs = append(exs, item)
		}
		reach := s.reachLine(es)
		if reach == "" {
			reach = "Flows reach no tables or jobs."
		}
		out = append(out, fmt.Sprintf("- %s: %s %d entry points (%s). For example: %s. %s",
			f.Name, f.Says, len(es), strings.Join(ks, ", "), strings.Join(exs, "; "), reach))
	}
	if len(s.chain) > 0 {
		out = append(out, "\n## Background work: what each job queues next (from the jobs' own flows)")
		for _, j := range sortedKeys(s.chain) {
			out = append(out, j+" -> "+strings.Join(s.chain[j], ", "))
		}
		out = append(out, s.chains()...)
	}

	// the flows given step by step: the widest flow of each of the largest features
	var picked []string
	taken := map[string]bool{}
	for _, f := range order {
		var cands []string
		for _, x := range s.expand("systems", f.Seeds) {
			if _, ok := flows[x]; ok && !taken[x] {
				cands = append(cands, x)
			}
		}
		cands = s.widestFirst(cands)
		if len(cands) > 0 && len(flows[cands[0]].Steps) >= 3 {
			picked = append(picked, cands[0])
			taken[cands[0]] = true
		}
		if len(picked) == stepped {
			break
		}
	}
	given := map[string][]givenStep{}
	if len(picked) > 0 {
		out = append(out, "\n## Flows given step by step (step numbers are what a scenario's steps start at)")
	}
	for _, eid := range picked {
		f := flows[eid]
		parts := map[string]core.Participant{}
		for _, p := range f.Participants {
			parts[p.ID] = p
		}
		name := func(id string) string {
			if p, ok := parts[id]; ok {
				return p.Name
			}
			return id
		}
		steps := f.Steps[:min(maxSteps, len(f.Steps))]
		head := "### steps of " + eid
		if len(f.Steps) > maxSteps {
			head += fmt.Sprintf(" (first %d of %d)", maxSteps, len(f.Steps))
		}
		out = append(out, head)
		for i, st := range steps {
			out = append(out, fmt.Sprintf("%d. %s%s -> %s: %s", i+1, strings.Repeat("  ", st.Depth), name(st.From), name(st.To), st.Call))
			given[eid] = append(given[eid], givenStep{elements: touched(parts[st.From], parts[st.To])})
		}
		bg, bgSteps := s.background(eid)
		if len(bg) > 0 {
			out = append(out, "then later, in the background (what its queued jobs queue, from their own flows):")
			for i, line := range bg {
				out = append(out, fmt.Sprintf("%d. %s", len(steps)+i+1, line))
			}
			given[eid] = append(given[eid], bgSteps...)
		}
	}
	return strings.Join(out, "\n"), given
}

// touched is what a step crosses, by the elements a reader can open: the components of its two ends,
// and the table, job or route it reaches.
func touched(from, to core.Participant) []string {
	var out []string
	for _, x := range []string{from.Component, to.Component, to.Element} {
		if x != "" && !contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// structural are the edge labels that hold the model together rather than say how containers talk.
func structural(label string) bool {
	return label == "uses" || label == "references" || label == "relates to"
}

// widestFirst orders entry points by their flows' width, widest first, keeping the order of ties.
func (s *seeding) widestFirst(ids []string) []string {
	flows := map[string]core.Flow{}
	for _, f := range s.m.Flows {
		flows[f.Entry] = f
	}
	w := func(id string) [2]int {
		if f, ok := flows[id]; ok {
			return width(f)
		}
		return [2]int{}
	}
	out := append([]string(nil), ids...)
	sort.SliceStable(out, func(i, j int) bool { return wider(w(out[i]), w(out[j])) })
	return out
}

func (s *seeding) jobIDs() map[string]string {
	ids := map[string]string{}
	for _, e := range s.m.Entries {
		if e.Kind == "job" {
			ids[e.Path] = e.ID
		}
	}
	return ids
}

// background is what a flow sets off later: each job it queues, then what that job queues, from the
// jobs' own flows, in the order a reader would follow them; sweeps left out, each job once.
func (s *seeding) background(eid string) ([]string, []givenStep) {
	jid := s.jobIDs()
	id := func(j string) string {
		if x, ok := jid[j]; ok {
			return x
		}
		return j
	}
	var out []string
	var steps []givenStep
	seen := map[string]bool{}
	var walk func(j string, depth int)
	walk = func(j string, depth int) {
		for _, k := range s.chain[j] {
			if seen[k] || strings.HasSuffix(k, sweep) || len(out) == later {
				continue
			}
			seen[k] = true
			out = append(out, fmt.Sprintf("%s[%s] -> [%s]: queues", strings.Repeat("  ", depth), id(j), id(k)))
			steps = append(steps, givenStep{elements: []string{id(j), id(k)}, background: true})
			walk(k, depth+1)
		}
	}
	var jobs []string
	if r := s.reach[eid]; r != nil {
		jobs = sortedKeys(r.jobs)
	}
	for _, j := range jobs {
		seen[j] = true
	}
	for _, j := range jobs {
		walk(j, 0)
	}
	return out, steps
}

// chains draws the deepest chains of jobs, each once, as trees with IDs, each job with the entry
// points that set it off: what a reader would otherwise assemble from the list of what each job
// queues. Chosen by depth alone; sweeps left out, and a job already drawn in a deeper tree not again.
func (s *seeding) chains() []string {
	jid := s.jobIDs()
	id := func(j string) string {
		if x, ok := jid[j]; ok {
			return x
		}
		return j
	}
	started := map[string][]string{}
	for _, e := range s.m.Entries {
		if e.Kind != "http" && e.Kind != "command" && e.Kind != "page" {
			continue
		}
		if r := s.reach[e.ID]; r != nil {
			for j := range r.jobs {
				started[j] = append(started[j], e.ID)
			}
		}
	}
	next := func(j string, seen map[string]bool) []string {
		var out []string
		for _, k := range s.chain[j] {
			if !seen[k] && !strings.HasSuffix(k, sweep) {
				out = append(out, k)
			}
		}
		return out
	}
	with := func(seen map[string]bool, k string) map[string]bool {
		c := make(map[string]bool, len(seen)+1)
		for x := range seen {
			c[x] = true
		}
		c[k] = true
		return c
	}
	var depth func(j string, seen map[string]bool) int
	depth = func(j string, seen map[string]bool) int {
		d := 0
		for _, k := range next(j, seen) {
			d = max(d, depth(k, with(seen, k)))
		}
		return 1 + d
	}
	drawn := map[string]bool{}
	var tree func(j string, seen map[string]bool, indent int) []string
	tree = func(j string, seen map[string]bool, indent int) []string {
		pad := strings.Repeat("  ", indent)
		if drawn[j] && len(next(j, seen)) > 0 {
			return []string{pad + "-> [" + id(j) + "]  (what it queues is drawn above)"}
		}
		drawn[j] = true
		es := append([]string(nil), started[j]...)
		sort.Strings(es)
		by := ""
		if len(es) > 0 {
			var shown []string
			for _, x := range es[:min(2, len(es))] {
				shown = append(shown, "["+x+"]")
			}
			by = "  (set off by " + strings.Join(shown, ", ")
			if len(es) > 2 {
				by += fmt.Sprintf(" and %d more", len(es)-2)
			}
			by += ")"
		}
		lines := []string{pad + "-> [" + id(j) + "]" + by}
		for _, k := range next(j, seen) {
			lines = append(lines, tree(k, with(seen, k), indent+1)...)
		}
		return lines
	}
	var roots []string
	depths := map[string]int{}
	for _, j := range sortedKeys(s.chain) {
		if !strings.HasSuffix(j, sweep) {
			roots = append(roots, j)
			depths[j] = depth(j, map[string]bool{j: true})
		}
	}
	sort.SliceStable(roots, func(a, b int) bool { return depths[roots[a]] > depths[roots[b]] })
	var drawnTrees [][]string
	for _, j := range roots {
		if len(drawnTrees) == trees || depths[j] < 3 {
			break
		}
		if !drawn[j] {
			drawnTrees = append(drawnTrees, tree(j, map[string]bool{j: true}, 0))
		}
	}
	if len(drawnTrees) == 0 {
		return nil
	}
	out := []string{"\n## The deepest chains of jobs (by depth; each job queues the ones under it)"}
	for _, t := range drawnTrees {
		out = append(out, t...)
	}
	return out
}
