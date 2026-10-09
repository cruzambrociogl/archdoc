package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// show answers a question about one thing without opening the app: an element, a route and its
// flow, or a file. Read from the committed model, so it answers in a terminal, in a script, and to
// a coding agent asking what a part of the system is and where that is proven. Every line it prints
// is a fact with its citation, or is marked as a model's (◇).
func show(e env, args []string) error {
	fs := flags("show")
	asJSON := fs.Bool("json", false, "print what was found as JSON")
	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	if len(positional) < 2 {
		return usagef("archdoc show <path> <element | route | file> — e.g. archdoc show . \"POST /api/assets\"")
	}
	root, query := positional[0], strings.Join(positional[1:], " ")
	m, ok, err := committedModel(root)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s is not documented yet — 'archdoc generate %s' first", absolute(root), root)
	}

	found := resolve(m, root, query)
	switch {
	case len(found) == 0:
		return fmt.Errorf("nothing in %s is called %q — try part of a name, a route such as \"GET /api/users\", or a file path", m.Name, query)
	case len(found) > 1:
		fmt.Fprintf(e.out, "%q matches %d things — name one:\n\n", query, len(found))
		for i, f := range found {
			if i == 15 {
				fmt.Fprintf(e.out, "  … and %d more\n", len(found)-15)
				break
			}
			fmt.Fprintf(e.out, "  %-40s %s\n", f.key, f.what)
		}
		return fmt.Errorf("%q names %d things — ask again with one of the ids above", query, len(found))
	}
	f := found[0]
	if *asJSON {
		return printJSON(e.out, f.json(m))
	}
	f.print(e.out, m)
	return nil
}

// match is one thing a query found: an element, a route, or a file and the component it is in.
type match struct {
	key, what string
	node      *archdoc.Node
	entry     *archdoc.Entry
	file      string
}

// resolve finds what a query names, most exact first: an id; a route as METHOD /path or a path; a
// file; a name; and only then part of a name. The first kind of match that finds anything wins.
func resolve(m archdoc.Model, root, query string) []match {
	q := strings.TrimSpace(query)
	lower := strings.ToLower(q)
	node := func(n archdoc.Node) match {
		n2 := n
		return match{key: n.ID, what: kindWords(m, n), node: &n2}
	}
	entry := func(en archdoc.Entry) match {
		e2 := en
		return match{key: en.Method + " " + en.Path, what: en.Kind + " · " + en.Container, entry: &e2}
	}
	try := func(steps ...func() []match) []match {
		for _, s := range steps {
			if out := s(); len(out) > 0 {
				return out
			}
		}
		return nil
	}
	return try(
		func() []match {
			if n, ok := m.Node(q); ok {
				return []match{node(n)}
			}
			for _, en := range m.Entries {
				if en.ID == q {
					return []match{entry(en)}
				}
			}
			return nil
		},
		func() []match {
			var out []match
			for _, en := range m.Entries {
				if strings.EqualFold(en.Method+" "+en.Path, q) || en.Path == q {
					out = append(out, entry(en))
				}
			}
			return out
		},
		func() []match {
			rel := filepath.ToSlash(q)
			if abs, err := filepath.Abs(q); err == nil {
				if base, err := filepath.Abs(root); err == nil {
					if r, err := filepath.Rel(base, abs); err == nil && !strings.HasPrefix(r, "..") {
						rel = filepath.ToSlash(r)
					}
				}
			}
			var out []match
			seen := map[string]bool{}
			for _, n := range m.Nodes {
				if n.Kind != archdoc.Component {
					continue
				}
				for _, f := range n.Files {
					if (f == rel || strings.HasSuffix(f, "/"+rel)) && !seen[f] {
						seen[f] = true
						n2 := n
						out = append(out, match{key: f, what: "file in " + n.Name, node: &n2, file: f})
					}
				}
			}
			return out
		},
		func() []match {
			var out []match
			for _, n := range m.Nodes {
				if strings.EqualFold(n.Name, q) && n.Kind != archdoc.Module {
					out = append(out, node(n))
				}
			}
			return out
		},
		func() []match {
			var out []match
			for _, n := range m.Nodes {
				if n.Kind != archdoc.Module && (strings.Contains(strings.ToLower(n.Name), lower) || strings.Contains(strings.ToLower(n.ID), lower)) {
					out = append(out, node(n))
				}
			}
			for _, en := range m.Entries {
				if strings.Contains(strings.ToLower(en.Method+" "+en.Path), lower) || strings.Contains(strings.ToLower(en.Handler), lower) {
					out = append(out, entry(en))
				}
			}
			return out
		},
	)
}

func kindWords(m archdoc.Model, n archdoc.Node) string {
	what := string(n.Kind)
	if n.Parent != "" {
		if p, ok := m.Node(n.Parent); ok {
			what += " in " + p.Name
		}
	}
	return what
}

func (f match) print(out io.Writer, m archdoc.Model) {
	switch {
	case f.entry != nil:
		printEntry(out, m, *f.entry)
	case f.file != "":
		fmt.Fprintf(out, "%s\n  in component %s, %s\n", f.file, f.node.Name, kindWords(m, *f.node))
		var here []archdoc.Entry
		for _, en := range m.Entries {
			if en.Prov.File == f.file {
				here = append(here, en)
			}
		}
		if len(here) > 0 {
			fmt.Fprintf(out, "\ndeclares %d way(s) in:\n", len(here))
			for i, en := range here {
				if i == 20 {
					fmt.Fprintf(out, "  … and %d more\n", len(here)-20)
					break
				}
				fmt.Fprintf(out, "  %-7s %-50s %s\n", en.Method, en.Path, cite(en.Prov))
			}
		}
		fmt.Fprintln(out)
		printNode(out, m, *f.node)
	default:
		printNode(out, m, *f.node)
	}
}

func printNode(out io.Writer, m archdoc.Model, n archdoc.Node) {
	fmt.Fprintf(out, "%s  [%s", n.Name, kindWords(m, n))
	if n.Technology != "" {
		fmt.Fprintf(out, " · %s", n.Technology)
	}
	fmt.Fprintf(out, "]\n  id %s · declared at %s", n.ID, cite(n.Prov))
	if n.Evidence == archdoc.Referenced {
		fmt.Fprint(out, " — referred to there, not defined by this repository")
	}
	fmt.Fprintln(out)
	if n.Description != "" {
		fmt.Fprintf(out, "  %s%s\n", interpreted(n.DescProv), n.Description)
	}
	if n.Dir != "" {
		fmt.Fprintf(out, "  code in %s\n", n.Dir)
	}
	if len(n.Files) > 0 {
		fmt.Fprintf(out, "  %d file(s), %d lines\n", len(n.Files), n.Lines)
	}

	name := func(id string) string {
		if x, ok := m.Node(id); ok {
			return x.Name
		}
		return id
	}
	var uses, usedBy []archdoc.Edge
	for _, e := range m.Edges {
		if e.From == n.ID {
			uses = append(uses, e)
		}
		if e.To == n.ID {
			usedBy = append(usedBy, e)
		}
	}
	// A relationship is shown by the other end's name, and its label where it says more than "uses":
	// every arrow between components is a use, so the header says it once.
	rel := func(head string, es []archdoc.Edge, other func(archdoc.Edge) string) {
		if len(es) == 0 {
			return
		}
		sort.SliceStable(es, func(i, j int) bool { return es[i].Weight > es[j].Weight })
		fmt.Fprintf(out, "\n%s (%d):\n", head, len(es))
		for i, e := range es {
			if i == 12 {
				fmt.Fprintf(out, "  … and %d more\n", len(es)-12)
				break
			}
			text := name(other(e))
			if e.Label != "" && e.Label != "uses" {
				text += " — " + interpreted(e.LabelProv) + e.Label
			}
			if e.Weight > 1 {
				text += fmt.Sprintf(" (%d imports)", e.Weight)
			}
			at := ""
			if len(e.Prov) > 0 {
				at = cite(e.Prov[0])
			}
			fmt.Fprintf(out, "  %-56s %s\n", text, at)
		}
	}
	rel("uses", uses, func(e archdoc.Edge) string { return e.To })
	rel("used by", usedBy, func(e archdoc.Edge) string { return e.From })

	if n.Kind == archdoc.Table {
		fmt.Fprintf(out, "\ncolumns (%d):\n", len(n.Columns))
		for _, c := range n.Columns {
			mark := ""
			if c.Primary {
				mark = " PK"
			}
			if c.References != "" {
				mark += " → " + name(c.References)
			}
			fmt.Fprintf(out, "  %-28s %-16s %-14s %s\n", c.Name, c.Type, strings.TrimSpace(mark), cite(c.Prov))
		}
	}

	var handles []archdoc.Entry
	for _, e := range m.Entries {
		if e.Component == n.ID {
			handles = append(handles, e)
		}
	}
	if len(handles) > 0 {
		fmt.Fprintf(out, "\nhandles %d way(s) in — 'archdoc show <path> \"METHOD /path\"' follows one:\n", len(handles))
		for i, e := range handles {
			if i == 15 {
				fmt.Fprintf(out, "  … and %d more\n", len(handles)-15)
				break
			}
			fmt.Fprintf(out, "  %-7s %-50s %s\n", e.Method, e.Path, cite(e.Prov))
		}
	}

	var data []archdoc.Access
	for _, a := range m.Access {
		if a.Component == n.ID || a.Element == n.ID {
			data = append(data, a)
		}
	}
	if len(data) > 0 {
		head := "tables its code queries"
		if n.Kind == archdoc.Table {
			head = "queried by"
		}
		fmt.Fprintf(out, "\n%s:\n", head)
		for _, a := range data {
			who := a.Table
			if n.Kind == archdoc.Table {
				who = name(a.Component)
			}
			fmt.Fprintf(out, "  %-8s %-40s %3d× · first at %s\n", a.Op, who, a.Count, cite(a.Prov[0]))
		}
	}

	for _, x := range m.Explanations {
		if x.Element != n.ID {
			continue
		}
		fmt.Fprintf(out, "\nwhat it does — ◇ written by %s from the facts it cites", x.Prov.Note)
		if x.Stale {
			fmt.Fprint(out, "; those facts have since changed")
		}
		fmt.Fprintln(out, ":")
		for _, c := range x.Claims {
			var at []string
			for _, p := range c.Cites {
				at = append(at, cite(p))
			}
			fmt.Fprintf(out, "  ◇ %s\n      %s\n", c.Text, strings.Join(at, " · "))
		}
	}
}

func printEntry(out io.Writer, m archdoc.Model, en archdoc.Entry) {
	fmt.Fprintf(out, "%s %s  [%s in %s]\n", en.Method, en.Path, en.Kind, nameOf(m, en.Container))
	if en.Summary != "" {
		fmt.Fprintf(out, "  %q — the code's own description, %s\n", en.Summary, cite(en.SummaryProv))
	}
	fmt.Fprintf(out, "  handled by %s, declared at %s\n", en.Handler, cite(en.Prov))
	if en.Component != "" {
		fmt.Fprintf(out, "  in component %s\n", nameOf(m, en.Component))
	}
	if en.PathNote != "" {
		fmt.Fprintf(out, "  path: %s\n", en.PathNote)
	}
	for _, f := range m.Flows {
		if f.Entry != en.ID {
			continue
		}
		label := map[string]string{}
		for _, p := range f.Participants {
			switch p.Kind {
			case "table":
				label[p.ID] = "table " + p.Name
			case "job":
				label[p.ID] = "job " + p.Name
			case "unresolved":
				label[p.ID] = "? an address computed at run time"
			default:
				label[p.ID] = p.Name
			}
		}
		fmt.Fprintf(out, "\nflow — %d step(s) through %d participant(s), followed by name through the code:\n", len(f.Steps), len(f.Participants))
		for i, s := range f.Steps {
			text := fmt.Sprintf("%s%s → %s", strings.Repeat("  ", s.Depth), label[s.From], label[s.To])
			if s.Call != "" && !strings.HasPrefix(s.To, "table:") && s.Call != "queues" {
				text += "." + s.Call
			} else if s.Call != "" {
				text = fmt.Sprintf("%s%s %s %s", strings.Repeat("  ", s.Depth), label[s.From], s.Call, strings.TrimPrefix(strings.TrimPrefix(label[s.To], "table "), "job "))
			}
			fmt.Fprintf(out, "%3d  %-64s %s\n", i+1, text, cite(s.Prov))
			if s.Note != "" {
				fmt.Fprintf(out, "     %s%s\n", strings.Repeat("  ", s.Depth), s.Note)
			}
		}
		if f.Cut {
			fmt.Fprintln(out, "     … cut here: the flow goes deeper or longer than archdoc follows")
		}
		return
	}
	if en.Kind == "http" || en.Kind == "job" {
		fmt.Fprintln(out, "\nno flow: nothing the handler calls could be followed to a class, a table or a call that leaves")
	}
}

func (f match) json(m archdoc.Model) any {
	switch {
	case f.entry != nil:
		var flow *archdoc.Flow
		for i := range m.Flows {
			if m.Flows[i].Entry == f.entry.ID {
				flow = &m.Flows[i]
			}
		}
		return map[string]any{"route": f.entry, "flow": flow}
	case f.file != "":
		return map[string]any{"file": f.file, "component": f.node}
	}
	var access []archdoc.Access
	for _, a := range m.Access {
		if a.Component == f.node.ID || a.Element == f.node.ID {
			access = append(access, a)
		}
	}
	var explanation *archdoc.Explanation
	for i := range m.Explanations {
		if m.Explanations[i].Element == f.node.ID {
			explanation = &m.Explanations[i]
		}
	}
	return map[string]any{"element": f.node, "access": access, "explanation": explanation}
}

func nameOf(m archdoc.Model, id string) string {
	if n, ok := m.Node(id); ok {
		return n.Name
	}
	return id
}

// cite is a provenance as an editor opens it.
func cite(p archdoc.Provenance) string { return p.String() }

// interpreted marks what a model wrote rather than what was read.
func interpreted(p archdoc.Provenance) string {
	if p.Origin.Interpretation() {
		return "◇ "
	}
	return ""
}
