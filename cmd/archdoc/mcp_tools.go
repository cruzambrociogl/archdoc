package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc"
)

// The tools an agent is offered. Each answers in the words the CLI prints — show and status are
// the commands themselves — so what an agent reads is what a person reads in a terminal.
type mcpTool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
	run         func(e env, root string, args map[string]string) error
}

func (t mcpTool) required() []string {
	r, _ := t.InputSchema["required"].([]string)
	return r
}

// readOnly is every tool's promise to the agent: nothing written, nothing sent, the same question
// the same answer while the documentation is unchanged.
var readOnly = map[string]any{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}

func schema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func text(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

// listKinds are what list lists, in the order a reader meets them.
var listKinds = []string{"services", "components", "tables", "routes", "jobs", "external", "unresolved"}

var mcpTools = []mcpTool{
	{
		Name:  "overview",
		Title: "What this system is",
		Description: "Start here. The system's services — applications, datastores, queues, proxies — and the outside services it " +
			"talks to, each with its technology, id and the line that declares it; what the code adds inside them, counted " +
			"(components, tables, routes, jobs, flows); and which version of the documentation this is.",
		InputSchema: schema(map[string]any{}),
		Annotations: readOnly,
		run:         overviewTool,
	},
	{
		Name:  "show",
		Title: "One element, route or file, with its proof",
		Description: "One thing, with every line that proves it. An element — a service, component or table, by id or name — with " +
			"what it uses and what uses it, the tables its code queries and the routes it handles. A route, as \"METHOD /path\", " +
			"with its flow through the code as numbered, cited steps. A file, with the component it belongs to. A word that " +
			"matches several things lists them: ask again with one of their ids.",
		InputSchema: schema(map[string]any{
			"query": text(`An id ("svc:db"), a name ("AlbumService"), a route ("POST /api/assets") or a file path.`),
		}, "query"),
		Annotations: readOnly,
		run:         showTool,
	},
	{
		Name:  "list",
		Title: "Everything of one kind",
		Description: "Every element of one kind, with its id and citation: services, components, tables, routes (HTTP routes, " +
			"pages and commands), jobs, external (outside services the system reaches), or unresolved (calls whose target is " +
			"computed at run time, which archdoc says it could not follow). filter narrows it to those whose name, id, path " +
			"or handler contains the text, or to what belongs to the service it names.",
		InputSchema: schema(map[string]any{
			"kind":   map[string]any{"type": "string", "enum": listKinds, "description": "What to list."},
			"filter": text("Optional: part of a name, id, path or handler; or a service's name or id, for what is in it."),
		}, "kind"),
		Annotations: readOnly,
		run:         listTool,
	},
	{
		Name:  "impact",
		Title: "What a change to something touches",
		Description: "What changing an element affects, from the flows archdoc followed through the code: the routes, pages and " +
			"jobs whose flow goes through it, who calls into it, what it calls, and the outside services it reaches — each " +
			"cited. Proven, and complete only as far as those flows go.",
		InputSchema: schema(map[string]any{
			"element": text(`A service, component or table, by id or name ("AlbumService", "table:album", "svc:server").`),
		}, "element"),
		Annotations: readOnly,
		run:         impactTool,
	},
	{
		Name:  "changes",
		Title: "What changed in the architecture",
		Description: "What changed in the architecture, the way a reader would say it: elements added, removed or renamed, " +
			"relationships and ways in. Without since, between the two latest versions archdoc recorded. With since, between " +
			"that commit and the code as it is now — read again, offline, and needing git.",
		InputSchema: schema(map[string]any{
			"since": text("Optional: a commit, branch or tag to compare the code as it is now with."),
		}),
		Annotations: readOnly,
		run:         changesTool,
	},
	{
		Name:  "status",
		Title: "Is the documentation up to date",
		Description: "Whether the documentation is current: the version and the commit it was read at, and whether the code has " +
			"changed the architecture since — the code is read again, offline, which takes a few seconds on a large " +
			"repository. Also what is open: gaps, descriptions a model wrote, stale explanations.",
		InputSchema: schema(map[string]any{}),
		Annotations: readOnly,
		run: func(e env, root string, _ map[string]string) error {
			return status(e, []string{root})
		},
	},
}

// committed is the documented model, or what to do when there is none.
func committed(root string) (archdoc.Model, error) {
	m, ok, err := archdoc.CommittedModel(root)
	if err != nil {
		return m, err
	}
	if !ok {
		return m, fmt.Errorf("%s is not documented yet — run 'archdoc generate %s' in a terminal first", absolute(root), absolute(root))
	}
	return m, nil
}

// at cites a provenance, or says nothing when it has no place to point at.
func at(p archdoc.Provenance) string {
	if p.File == "" && p.Note == "" {
		return ""
	}
	return cite(p)
}

func overviewTool(e env, root string, _ map[string]string) error {
	m, err := committed(root)
	if err != nil {
		return err
	}
	out := e.out
	fmt.Fprintf(out, "%s — %s\n", m.Name, absolute(root))
	if m.Description != "" {
		fmt.Fprintf(out, "%s%s  %s\n", interpreted(m.DescProv), m.Description, at(m.DescProv))
	}
	if h, err := archdoc.OpenHistory(root); err == nil {
		defer h.Close()
		if vs, err := h.Versions(1); err == nil && len(vs) == 1 {
			v := vs[0]
			fmt.Fprintf(out, "documented: version %d, recorded %s%s\n", v.ID, v.CreatedAt.Local().Format("2 Jan 2006 15:04"), atCommit(v.Commit))
		} else if !h.Recorded() {
			fmt.Fprintf(out, "documented: %s is committed; this clone has no local history\n", archdoc.ModelFile)
		}
	}
	fmt.Fprintln(out)
	archdoc.Count(m).Print(out, m.Source)

	var services, outside []archdoc.Node
	for _, n := range m.Nodes {
		switch {
		case n.Kind == "external":
			outside = append(outside, n)
		case n.Kind.Container() || n.Kind == "proxy":
			services = append(services, n)
		}
	}
	nodeList(out, m, "services", services)
	nodeList(out, m, "outside services it talks to", outside)
	fmt.Fprintln(out, "\nshow follows any of these by id; list names every component, table, route or job; impact says what a change touches.")
	return nil
}

func nodeList(out io.Writer, m archdoc.Model, head string, ns []archdoc.Node) {
	if len(ns) == 0 {
		return
	}
	fmt.Fprintf(out, "\n%s (%d):\n", head, len(ns))
	for _, n := range ns {
		what := string(n.Kind)
		if n.Technology != "" {
			what += " · " + n.Technology
		}
		parts := 0
		for _, x := range m.Nodes {
			if x.Parent == n.ID && x.Kind.Part() {
				parts++
			}
		}
		if parts > 0 {
			what += fmt.Sprintf(" · %d parts in its code", parts)
		}
		fmt.Fprintf(out, "  %s  [%s]  %s  %s\n", n.Name, what, n.ID, at(n.Prov))
	}
}

// listCap is how many lines list prints before it counts the rest: enough for most repositories'
// routes, few enough not to bury an agent.
const listCap = 150

func listTool(e env, root string, args map[string]string) error {
	m, err := committed(root)
	if err != nil {
		return err
	}
	kind, filter := args["kind"], strings.ToLower(args["filter"])

	// A filter naming a service lists what is in it; otherwise it is matched as text.
	in := ""
	for _, n := range m.Nodes {
		if filter != "" && !n.Kind.Part() && (strings.ToLower(n.ID) == filter || strings.ToLower(n.Name) == filter) {
			in = n.ID
		}
	}
	keep := func(container string, words ...string) bool {
		if filter == "" {
			return true
		}
		if in != "" {
			return container == in
		}
		for _, w := range words {
			if strings.Contains(strings.ToLower(w), filter) {
				return true
			}
		}
		return false
	}

	var lines []string
	add := func(format string, a ...any) { lines = append(lines, fmt.Sprintf(format, a...)) }
	switch kind {
	case "services", "components", "tables", "external":
		for _, n := range m.Nodes {
			var ok bool
			switch kind {
			case "services":
				ok = n.Kind.Container() || n.Kind == "proxy"
			case "components":
				ok = n.Kind == archdoc.Component
			case "tables":
				ok = n.Kind == archdoc.Table
			case "external":
				ok = n.Kind == "external"
			}
			if !ok || !keep(n.Parent, n.Name, n.ID, n.Dir) {
				continue
			}
			what := archdoc.KindWords(m, n)
			switch {
			case n.Kind == archdoc.Table:
				what += fmt.Sprintf(" · %d columns", len(n.Columns))
			case n.Technology != "":
				what += " · " + n.Technology
			}
			add("%s  [%s]  %s  %s", n.Name, what, n.ID, at(n.Prov))
		}
	case "routes", "jobs":
		for _, en := range m.Entries {
			if (kind == "jobs") != (en.Kind == "job") || !keep(en.Container, en.Method+" "+en.Path, en.Handler, en.Summary) {
				continue
			}
			way := en.Method + " " + en.Path
			switch en.Kind {
			case "page":
				way = "page " + en.Path
			case "job", "command":
				way = firstNonEmpty(en.Path, en.Handler)
			}
			add("%s  %s  in %s  %s", way, en.Handler, nameOf(m, en.Container), at(en.Prov))
		}
	case "unresolved":
		for _, u := range m.Unresolved {
			if !keep(u.Container, u.What, u.Note) {
				continue
			}
			add("%s  in %s  %s\n    %s", u.What, nameOf(m, u.Container), at(u.Prov), u.Note)
		}
	default:
		return fmt.Errorf("list lists one of %s, not %q", strings.Join(listKinds, ", "), kind)
	}

	head := fmt.Sprintf("%d %s", len(lines), kind)
	if filter != "" {
		head += fmt.Sprintf(" matching %q", args["filter"])
	}
	fmt.Fprintf(e.out, "%s in %s:\n", head, m.Name)
	for i, l := range lines {
		if i == listCap {
			fmt.Fprintf(e.out, "  … and %d more — narrow it with filter\n", len(lines)-listCap)
			break
		}
		fmt.Fprintf(e.out, "  %s\n", l)
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// impactTool is the app's Impact (platform-surface §6.4, web/app/src/screens/Impact.tsx) in words:
// built from the flows the engine followed — every flow the element, or a part of it, takes part
// in. The sentence names the flows, then the jobs, alphabetically; nothing is chosen to mention.
func impactTool(e env, root string, args map[string]string) error {
	m, err := committed(root)
	if err != nil {
		return err
	}
	found := archdoc.Find(m, root, args["element"])
	c, isClass := classNamed(m, args["element"])
	switch {
	case len(found) == 0 && isClass:
		return impactOf(e.out, m, c.id, c.id, "", func(id, element, component string) bool { return id == c.id }, func(id, element, component string) string { return firstNonEmpty(element, id) })
	case len(found) == 0:
		return fmt.Errorf("nothing in %s is called %q — list names what there is", m.Name, args["element"])
	case len(found) > 1:
		fmt.Fprintf(e.out, "%q matches %d things — name one:\n\n", args["element"], len(found))
		for i, f := range found {
			if i == 15 {
				fmt.Fprintf(e.out, "  … and %d more\n", len(found)-15)
				break
			}
			fmt.Fprintf(e.out, "  %-40s %s\n", f.Key, f.What)
		}
		return fmt.Errorf("%q names %d things — ask again with one of the ids above", args["element"], len(found))
	case found[0].Entry != nil:
		return fmt.Errorf("%s is a way in, not an element — show follows its flow; impact takes the service, component or table it goes through", found[0].Key)
	}
	node := *found[0].Node
	x := node.ID
	return impactOf(e.out, m, node.Name, x, node.Parent,
		func(id, element, component string) bool { return element == x || component == x },
		func(id, element, component string) string { return firstNonEmpty(element, component) })
}

// impactOf says what a change to x touches. is says whether a flow's participant is x — an
// element, a part of it, or the class itself — and listed the id a participant is listed under.
// parent is x's container, whose outside services a part reaches through it.
func impactOf(out io.Writer, m archdoc.Model, name, x, parent string, is func(id, element, component string) bool, listed func(id, element, component string) string) error {
	entries := map[string]archdoc.Entry{}
	for _, en := range m.Entries {
		entries[en.ID] = en
	}
	var touching []archdoc.Flow
	for _, f := range m.Flows {
		en, ok := entries[f.Entry]
		hit := ok && en.Container == x
		for _, p := range f.Participants {
			hit = hit || is(p.ID, p.Element, p.Component)
		}
		if hit {
			touching = append(touching, f)
		}
	}
	label := func(en archdoc.Entry) string {
		switch {
		case en.Kind == "job" || en.Kind == "command":
			return firstNonEmpty(en.Path, en.Handler)
		case en.Summary != "":
			return strings.ToLower(en.Summary[:1]) + en.Summary[1:]
		case en.Kind == "page":
			return "page " + en.Path
		}
		return en.Method + " " + en.Path
	}
	var flows, jobs []archdoc.Entry
	for _, f := range touching {
		if en, ok := entries[f.Entry]; ok {
			if en.Kind == "job" {
				jobs = append(jobs, en)
			} else {
				flows = append(flows, en)
			}
		}
	}
	byLabel := func(es []archdoc.Entry) {
		sort.SliceStable(es, func(i, j int) bool { return label(es[i]) < label(es[j]) })
	}
	byLabel(flows)
	byLabel(jobs)

	// Who calls into it, and what it calls, from the steps of those flows, each where it was first seen.
	type ref struct {
		id string
		at archdoc.Provenance
	}
	var users, uses []ref
	seen := map[string]bool{}
	for _, f := range touching {
		ps := map[string]int{}
		for i, p := range f.Participants {
			ps[p.ID] = i
		}
		for _, s := range f.Steps {
			i, ok1 := ps[s.From]
			j, ok2 := ps[s.To]
			if !ok1 || !ok2 {
				continue
			}
			from, to := f.Participants[i], f.Participants[j]
			fromID, toID := listed(from.ID, from.Element, from.Component), listed(to.ID, to.Element, to.Component)
			if is(to.ID, to.Element, to.Component) && fromID != "" && fromID != x && !seen["u"+fromID] {
				seen["u"+fromID] = true
				users = append(users, ref{fromID, s.Prov})
			}
			if is(from.ID, from.Element, from.Component) && toID != "" && toID != x && !seen["s"+toID] {
				seen["s"+toID] = true
				uses = append(uses, ref{toID, s.Prov})
			}
		}
	}
	// The outside services it — or, for a part inside an app, its app — reaches.
	var reach []ref
	for _, ed := range m.Edges {
		if to, ok := m.Node(ed.To); ok && to.Kind == "external" && (ed.From == x || (parent != "" && ed.From == parent)) {
			var p archdoc.Provenance
			if len(ed.Prov) > 0 {
				p = ed.Prov[0]
			}
			reach = append(reach, ref{ed.To, p})
		}
	}

	var named []string
	for _, en := range append(append([]archdoc.Entry{}, flows...), jobs...) {
		named = append(named, label(en))
	}
	var counts []string
	if len(flows) > 0 {
		counts = append(counts, plural(len(flows), "flow", "flows"))
	}
	if len(jobs) > 0 {
		counts = append(counts, plural(len(jobs), "job", "jobs"))
	}
	if len(counts) == 0 {
		fmt.Fprintf(out, "Changing %s affects no flow archdoc followed through the code. A flow it is part of may start somewhere archdoc doesn't read.\n", name)
	} else {
		said := joinAnd(named)
		if len(named) > 5 {
			said = fmt.Sprintf("%s and %d more", strings.Join(named[:5], ", "), len(named)-5)
		}
		fmt.Fprintf(out, "Changing %s affects %s: %s.\n", name, strings.Join(counts, " and "), said)
	}
	fmt.Fprintln(out, "Written from the flows archdoc followed — proven, and complete only as far as they go. show lists every relationship it has.")

	group := func(head string, n int, line func(i int) string) {
		if n == 0 {
			return
		}
		fmt.Fprintf(out, "\n%s (%d):\n", head, n)
		for i := 0; i < n; i++ {
			if i == 40 {
				fmt.Fprintf(out, "  … and %d more\n", n-40)
				break
			}
			fmt.Fprintf(out, "  %s\n", line(i))
		}
	}
	way := func(en archdoc.Entry) string {
		if en.Kind == "page" {
			return "page " + en.Path
		}
		return en.Method + " " + en.Path
	}
	group("routes and pages through it", len(flows), func(i int) string { return fmt.Sprintf("%s  %s", way(flows[i]), at(flows[i].Prov)) })
	group("background jobs", len(jobs), func(i int) string { return fmt.Sprintf("%s  %s", label(jobs[i]), at(jobs[i].Prov)) })
	refs := func(rs []ref) func(i int) string {
		return func(i int) string {
			if n := nameOf(m, rs[i].id); n != rs[i].id {
				return fmt.Sprintf("%s  %s  %s", n, rs[i].id, at(rs[i].at))
			}
			return fmt.Sprintf("%s  %s", rs[i].id, at(rs[i].at))
		}
	}
	group("who uses it", len(users), refs(users))
	group("what it uses", len(uses), refs(uses))
	group("outside services it reaches", len(reach), refs(reach))
	return nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func joinAnd(xs []string) string {
	if len(xs) <= 1 {
		return strings.Join(xs, "")
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

func changesTool(e env, root string, args map[string]string) error {
	if since := args["since"]; since != "" {
		return diff(e, []string{root, since})
	}
	if _, err := committed(root); err != nil {
		return err
	}
	h, err := archdoc.OpenHistory(root)
	if err != nil {
		return err
	}
	defer h.Close()
	if !h.Recorded() {
		return fmt.Errorf("this clone has no local history, only the committed documentation — give since a commit to compare the code with it")
	}
	vs, err := h.Versions(2)
	if err != nil {
		return err
	}
	if len(vs) < 2 {
		return fmt.Errorf("archdoc has recorded one version, so there is nothing to compare — give since a commit to compare the code with it")
	}
	after, before := vs[0], vs[1]
	out := e.out
	fmt.Fprintf(out, "version %d → version %d, recorded %s%s\n\n", before.ID, after.ID, after.CreatedAt.Local().Format("2 Jan 2006 15:04"), atCommit(after.Commit))
	c := archdoc.Compare(before.Model, after.Model)
	lines := c.Summary()
	if len(lines) == 0 {
		fmt.Fprintln(out, "no change to the architecture")
		return nil
	}
	for _, l := range lines {
		fmt.Fprintf(out, "%s %s\n", l.Mark, l.Text)
	}
	return nil
}

// class is a class the flows pass through. The engine follows calls by class, so a class is a
// flow's participant rather than an element of the model: what show and impact are asked about
// most by an agent, which thinks in the code's own names.
type class struct {
	id, component string
	decl          archdoc.Provenance // where it is declared, when a route's handler or injection says
}

func classNamed(m archdoc.Model, q string) (class, bool) {
	for _, f := range m.Flows {
		for _, p := range f.Participants {
			if (p.Kind == "class" || p.Kind == "module") && (p.ID == q || strings.EqualFold(p.Name, q)) {
				c := class{id: p.ID, component: p.Component}
				for _, en := range m.Entries {
					if cls, _, _ := strings.Cut(en.Handler, "."); cls == p.ID && c.decl.File == "" {
						c.decl = en.Prov
					}
					for _, u := range en.Uses {
						if u.Name == p.ID && u.How == "name" && c.decl.File == "" {
							c.decl = u.Prov
						}
					}
				}
				return c, true
			}
		}
	}
	return class{}, false
}

// showTool is show, and for a class — which show does not know — where it is and what to ask next.
func showTool(e env, root string, args map[string]string) error {
	q := args["query"]
	m, err := committed(root)
	if err != nil {
		return err
	}
	if len(archdoc.Find(m, root, q)) == 0 {
		if c, ok := classNamed(m, q); ok {
			fmt.Fprintf(e.out, "%s  [class", c.id)
			if c.component != "" {
				fmt.Fprintf(e.out, " in component %s, %s", nameOf(m, c.component), c.component)
			}
			fmt.Fprintln(e.out, "]")
			if c.decl.File != "" {
				fmt.Fprintf(e.out, "  in %s, as a route's handler or a class injected into one names it\n", c.decl.File)
			}
			fmt.Fprintf(e.out, "\nA class the flows pass through, not an element of the model: impact %s says which routes and jobs go through it and what it calls", c.id)
			if c.component != "" {
				fmt.Fprintf(e.out, "; show %s, its component", c.component)
			}
			fmt.Fprintln(e.out, ".")
			return nil
		}
	}
	return show(e, []string{root, q})
}
