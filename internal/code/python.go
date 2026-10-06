package code

import (
	"regexp"
	"strings"

	ts "github.com/odvcencio/gotreesitter"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

var pyHTTPCallee = regexp.MustCompile(`^(requests|httpx|aiohttp)\.(get|post|put|patch|delete|head|request)$`)

// pythonFacts reads a Python file in one walk: `import a.b` and `from .a import b, c` — keeping the
// names, since `from app.api.routes import items, users` imports two modules — decorated functions
// and classes, the hosts its literals name, and the HTTP calls it makes.
func pythonFacts(src []byte, file string) (out facts, partial bool) {
	at := func(n *ts.Node) archdoc.Provenance {
		p := n.StartPoint()
		return archdoc.Provenance{File: file, Line: int(p.Row) + 1, Column: int(p.Column) + 1}
	}
	called := map[archdoc.Provenance]bool{}
	partial = parse("Python", src, func(root *ts.Node, l *ts.Language) {
		text := func(n *ts.Node) string { return n.Text(src) }
		// Module-level decorated functions — FastAPI's @router.get — gather under a class with no name.
		module := archdoc.Class{Prov: archdoc.Provenance{File: file, Line: 1}}
		walk(root, func(n *ts.Node) {
			switch n.Type(l) {
			case "import_statement":
				for i := 0; i < n.NamedChildCount(); i++ {
					c := n.NamedChild(i)
					if c.Type(l) == "aliased_import" && c.NamedChildCount() > 0 {
						c = c.NamedChild(0)
					}
					if c.Type(l) == "dotted_name" {
						p := at(c)
						out.imports = append(out.imports, rawImport{spec: text(c), line: p.Line, column: p.Column})
					}
				}
			case "import_from_statement":
				if n.NamedChildCount() == 0 {
					return
				}
				mod := n.NamedChild(0)
				p := at(mod)
				imp := rawImport{spec: text(mod), line: p.Line, column: p.Column}
				for i := 1; i < n.NamedChildCount(); i++ {
					c := n.NamedChild(i)
					if c.Type(l) == "aliased_import" && c.NamedChildCount() > 0 {
						c = c.NamedChild(0)
					}
					if c.Type(l) == "dotted_name" {
						imp.names = append(imp.names, text(c))
					}
				}
				out.imports = append(out.imports, imp)
			case "decorated_definition":
				var decorators []archdoc.Decorator
				var def *ts.Node
				for i := 0; i < n.NamedChildCount(); i++ {
					c := n.NamedChild(i)
					switch c.Type(l) {
					case "decorator":
						decorators = append(decorators, pyDecorator(c, l, src, at))
					case "function_definition", "class_definition":
						def = c
					}
				}
				if def == nil {
					return
				}
				name := ""
				if def.NamedChildCount() > 0 {
					name = text(def.NamedChild(0))
				}
				if def.Type(l) == "class_definition" {
					return // read as a class below, with its decorators
				}
				// A method of a class is recorded with the class only when the class itself is;
				// a decorated function anywhere else is the module's.
				module.Methods = append(module.Methods, archdoc.Method{Name: name, Decorators: decorators, Prov: at(def)})
			case "class_definition":
				out.classes = append(out.classes, pyClass(n, l, src, at))
			case "call":
				if n.ChildCount() < 2 {
					return
				}
				fn, args := n.Child(0), n.Child(1)
				if args.Type(l) != "argument_list" || !pyHTTPCallee.MatchString(text(fn)) || args.NamedChildCount() == 0 {
					return
				}
				first := args.NamedChild(0)
				if v, ok := pyString(first, l, src); ok && urlLiteral.MatchString(v) {
					called[at(first)] = true
					return
				}
				out.calls = append(out.calls, archdoc.Call{Callee: text(fn), Target: shorten(text(first)), Prov: at(n)})
			case "string":
				if v, ok := pyString(n, l, src); ok {
					if h, ok := hostOf(v); ok {
						h.Prov = at(n)
						out.hosts = append(out.hosts, h)
					}
				}
			case "keyword_argument", "assignment":
				if n.NamedChildCount() < 2 {
					return
				}
				key, value := n.NamedChild(0), n.NamedChild(n.NamedChildCount()-1)
				if !hostKey.MatchString(text(key)) {
					return
				}
				if v, ok := pyString(value, l, src); ok && hostName.MatchString(v) {
					out.hosts = append(out.hosts, archdoc.HostRef{Host: v, Value: v, Prov: at(value)})
				}
			}
		})
		if len(module.Methods) > 0 {
			out.classes = append(out.classes, module)
		}
	})
	for i := range out.hosts {
		out.hosts[i].Called = called[out.hosts[i].Prov]
	}
	return out, partial
}

// pyClass reads a class: its bases and keyword arguments — class User(UserBase, table=True) — its
// decorators, and the fields its body declares.
func pyClass(n *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Class {
	c := archdoc.Class{Prov: at(n)}
	if p := n.Parent(); p != nil && p.Type(l) == "decorated_definition" {
		for i := 0; i < p.NamedChildCount(); i++ {
			if d := p.NamedChild(i); d.Type(l) == "decorator" {
				c.Decorators = append(c.Decorators, pyDecorator(d, l, src, at))
			}
		}
	}
	for i := 0; i < n.NamedChildCount(); i++ {
		ch := n.NamedChild(i)
		switch ch.Type(l) {
		case "identifier":
			if c.Name == "" {
				c.Name = ch.Text(src)
			}
		case "argument_list":
			for j := 0; j < ch.NamedChildCount(); j++ {
				a := ch.NamedChild(j)
				switch a.Type(l) {
				case "identifier", "attribute":
					c.Extends = append(c.Extends, a.Text(src))
				case "keyword_argument":
					if a.NamedChildCount() >= 2 {
						if c.Options == nil {
							c.Options = map[string]string{}
						}
						c.Options[a.NamedChild(0).Text(src)] = pyValue(a.NamedChild(1), l, src)
					}
				}
			}
		case "block":
			for j := 0; j < ch.NamedChildCount(); j++ {
				st := ch.NamedChild(j)
				if st.Type(l) == "expression_statement" && st.NamedChildCount() > 0 {
					st = st.NamedChild(0)
				}
				if st.Type(l) != "assignment" || st.NamedChildCount() < 2 {
					continue
				}
				f := archdoc.Field{Name: st.NamedChild(0).Text(src), Prov: at(st)}
				value := st.NamedChild(st.NamedChildCount() - 1)
				for k := 1; k < st.NamedChildCount(); k++ {
					if t := st.NamedChild(k); t.Type(l) == "type" {
						f.Type = t.Text(src)
					}
				}
				switch value.Type(l) {
				case "call":
					f.Decorators = []archdoc.Decorator{pyCall(value, l, src, at)}
				case "string":
					f.Value, _ = pyString(value, l, src)
				}
				if value == st.NamedChild(0) || (f.Type == "" && len(f.Decorators) == 0 && f.Value == "") {
					continue
				}
				c.Fields = append(c.Fields, f)
			}
		}
	}
	return c
}

// pyCall reads a call the way a decorator is read: Field(foreign_key="user.id", nullable=False).
func pyCall(e *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Decorator {
	out := archdoc.Decorator{Name: e.Child(0).Text(src), Prov: at(e)}
	if e.ChildCount() < 2 {
		return out
	}
	args := e.Child(1)
	for i := 0; i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if a.Type(l) == "keyword_argument" && a.NamedChildCount() >= 2 {
			if out.Options == nil {
				out.Options = map[string]string{}
			}
			out.Options[a.NamedChild(0).Text(src)] = pyValue(a.NamedChild(1), l, src)
			continue
		}
		if i == 0 {
			if v, ok := pyString(a, l, src); ok {
				out.Arg, out.HasArg = v, true
			} else if a.Type(l) == "identifier" || a.Type(l) == "attribute" {
				out.Target = a.Text(src)
			}
		}
	}
	return out
}

// pyValue is a keyword argument's value: a string's content, or anything else as written.
func pyValue(n *ts.Node, l *ts.Language, src []byte) string {
	if v, ok := pyString(n, l, src); ok {
		return v
	}
	return n.Text(src)
}

// pyDecorator reads @name, @router.get("/items/{id}", summary="…").
func pyDecorator(d *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Decorator {
	out := archdoc.Decorator{Prov: at(d)}
	if d.NamedChildCount() == 0 {
		return out
	}
	e := d.NamedChild(0)
	if e.Type(l) != "call" || e.ChildCount() < 2 {
		out.Name = e.Text(src)
		return out
	}
	out.Name = e.Child(0).Text(src)
	args := e.Child(1)
	for i := 0; i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if i == 0 {
			if v, ok := pyString(a, l, src); ok {
				out.Arg, out.HasArg = v, true
			}
		}
		if a.Type(l) == "keyword_argument" && a.NamedChildCount() >= 2 {
			key, value := a.NamedChild(0).Text(src), a.NamedChild(1)
			if v, ok := pyString(value, l, src); ok {
				switch key {
				case "summary":
					out.Summary, out.SummaryProv = v, at(value)
				case "path":
					out.Arg, out.HasArg = v, true
				}
			}
		}
	}
	return out
}

// pyString is a string literal's value; an f-string with something interpolated has none.
func pyString(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	if n.Type(l) != "string" {
		return "", false
	}
	for i := 0; i < n.ChildCount(); i++ {
		if n.Child(i).Type(l) == "interpolation" {
			return "", false
		}
	}
	s := n.Text(src)
	s = strings.TrimLeft(s, "rRbBfFuU")
	for _, q := range []string{`"""`, `'''`, `"`, `'`} {
		if strings.HasPrefix(s, q) && strings.HasSuffix(s, q) && len(s) >= 2*len(q) {
			return s[len(q) : len(s)-len(q)], true
		}
	}
	return s, true
}
