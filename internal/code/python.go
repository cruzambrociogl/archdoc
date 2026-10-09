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
	held := map[string]archdoc.Provenance{} // a name → the literal it is given: API = "https://…"
	var byName []namedCall
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
			case "function_definition":
				// A module's functions — decorated or not — are its methods: what a route's handler
				// calls is usually a plain function in another module.
				owner := n.Parent()
				var decorators []archdoc.Decorator
				if owner != nil && owner.Type(l) == "decorated_definition" {
					for i := 0; i < owner.NamedChildCount(); i++ {
						if d := owner.NamedChild(i); d.Type(l) == "decorator" {
							decorators = append(decorators, pyDecorator(d, l, src, at))
						}
					}
					owner = owner.Parent()
				}
				if owner == nil || owner.Type(l) != "module" {
					return // a method of a class, or a nested function
				}
				module.Methods = append(module.Methods, pyMethod(n, decorators, l, src, at))
			case "if_statement":
				// if __name__ == "__main__": the file is run directly.
				if p := n.Parent(); p != nil && p.Type(l) == "module" && n.NamedChildCount() > 0 {
					if cond := text(n.NamedChild(0)); strings.Contains(cond, "__name__") && strings.Contains(cond, "__main__") {
						where := at(n)
						out.main = &where
					}
				}
			case "class_definition":
				out.classes = append(out.classes, pyClass(n, l, src, at))
			case "call":
				if n.ChildCount() < 2 {
					return
				}
				fn, args := n.Child(0), n.Child(1)
				if strings.HasSuffix(text(fn), ".include_router") && args.Type(l) == "argument_list" && args.NamedChildCount() > 0 {
					m := archdoc.Include{Parent: strings.TrimSuffix(text(fn), ".include_router"), Child: text(args.NamedChild(0)), Prov: at(n)}
					for i := 1; i < args.NamedChildCount(); i++ {
						a := args.NamedChild(i)
						if a.Type(l) == "keyword_argument" && a.NamedChildCount() >= 2 && text(a.NamedChild(0)) == "prefix" {
							if v, ok := pyString(a.NamedChild(1), l, src); ok {
								m.Prefix = v
							} else {
								m.PrefixExpr = text(a.NamedChild(1))
							}
						}
					}
					for p := n.Parent(); p != nil; p = p.Parent() {
						if p.Type(l) == "if_statement" && p.NamedChildCount() > 0 {
							m.Condition = text(p.NamedChild(0))
							break
						}
					}
					out.incs = append(out.incs, m)
					return
				}
				if args.Type(l) != "argument_list" || !pyHTTPCallee.MatchString(text(fn)) || args.NamedChildCount() == 0 {
					return
				}
				first := args.NamedChild(0)
				if v, ok := pyString(first, l, src); ok && urlLiteral.MatchString(v) {
					called[at(first)] = true
					return
				}
				if first.Type(l) == "identifier" {
					byName = append(byName, namedCall{text(first), len(out.calls)})
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
				if n.Type(l) == "assignment" && key.Type(l) == "identifier" && value.Type(l) == "string" {
					held[text(key)] = at(value)
				}
				if n.Type(l) == "assignment" && value.Type(l) == "call" && value.ChildCount() > 1 {
					// router = APIRouter(prefix="/items"), app = FastAPI(…)
					if callee := text(value.Child(0)); callee == "APIRouter" || callee == "FastAPI" || strings.HasSuffix(callee, ".APIRouter") || strings.HasSuffix(callee, ".FastAPI") {
						r := archdoc.Router{Var: text(key), Kind: callee[strings.LastIndex(callee, ".")+1:], Prov: at(n)}
						args := value.Child(1)
						for i := 0; i < args.NamedChildCount(); i++ {
							a := args.NamedChild(i)
							if a.Type(l) == "keyword_argument" && a.NamedChildCount() >= 2 && text(a.NamedChild(0)) == "prefix" {
								r.Prefix, _ = pyString(a.NamedChild(1), l, src)
							}
						}
						out.routers = append(out.routers, r)
						return
					}
				}
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
	out.calls = resolveNamed(out.calls, byName, held, out.hosts, called)
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
				if def, decorators := pyDef(st, l, src, at); def != nil {
					meth := pyMethod(def, decorators, l, src, at)
					if meth.Name == "__init__" {
						c.Params = pyFields(def, l, src, at)
					}
					c.Methods = append(c.Methods, meth)
					continue
				}
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

// pyDef is the function a statement of a class body defines, with its decorators, or nil.
func pyDef(st *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) (*ts.Node, []archdoc.Decorator) {
	if st.Type(l) == "function_definition" {
		return st, nil
	}
	if st.Type(l) != "decorated_definition" {
		return nil, nil
	}
	var def *ts.Node
	var decorators []archdoc.Decorator
	for i := 0; i < st.NamedChildCount(); i++ {
		switch ch := st.NamedChild(i); ch.Type(l) {
		case "decorator":
			decorators = append(decorators, pyDecorator(ch, l, src, at))
		case "function_definition":
			def = ch
		}
	}
	return def, decorators
}

// pyMethod reads a function — a module's, or a class's method — as a method is read.
func pyMethod(n *ts.Node, decorators []archdoc.Decorator, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Method {
	name := ""
	if n.NamedChildCount() > 0 {
		name = n.NamedChild(0).Text(src)
	}
	meth := archdoc.Method{Name: name, Decorators: decorators, Prov: at(n),
		EndLine: at(n).Line + int(n.EndPoint().Row-n.StartPoint().Row)}
	meth.Doc, meth.DocProv = docstring(n, l, src, at)
	meth.Invokes, meth.Queries = pyBody(n, l, src, at)
	return meth
}

// pyTyped are a function's parameters that state a type: (self, repo: UserRepository).
func pyTyped(def *ts.Node, l *ts.Language, src []byte) map[string]string {
	out := map[string]string{}
	for i := 0; i < def.NamedChildCount(); i++ {
		ps := def.NamedChild(i)
		if ps.Type(l) != "parameters" {
			continue
		}
		for j := 0; j < ps.NamedChildCount(); j++ {
			p := ps.NamedChild(j)
			if t := p.Type(l); (t != "typed_parameter" && t != "typed_default_parameter") || p.NamedChildCount() < 2 {
				continue
			}
			for k := 1; k < p.NamedChildCount(); k++ {
				if ty := p.NamedChild(k); ty.Type(l) == "type" {
					out[p.NamedChild(0).Text(src)] = ty.Text(src)
				}
			}
		}
	}
	return out
}

// pyFields are the fields __init__ gives an object whose class the code states: self.repo = repo,
// where repo is a typed parameter, and self.cache = ModelCache(…).
func pyFields(init *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Param {
	typed := pyTyped(init, l, src)
	var out []archdoc.Param
	walk(init, func(n *ts.Node) {
		if n.Type(l) != "assignment" || n.NamedChildCount() < 2 {
			return
		}
		left, right := n.NamedChild(0), n.NamedChild(n.NamedChildCount()-1)
		if left.Type(l) != "attribute" || left.NamedChildCount() != 2 || left.NamedChild(0).Text(src) != "self" {
			return
		}
		typ := ""
		for k := 1; k < n.NamedChildCount()-1; k++ {
			if t := n.NamedChild(k); t.Type(l) == "type" {
				typ = t.Text(src) // self.repo: UserRepository = …
			}
		}
		switch {
		case typ != "":
		case right.Type(l) == "identifier":
			typ = typed[right.Text(src)]
		case right.Type(l) == "call" && right.ChildCount() > 0 && right.Child(0).Type(l) == "identifier":
			typ = right.Child(0).Text(src)
		}
		if typ != "" {
			out = append(out, archdoc.Param{Name: left.NamedChild(1).Text(src), Type: typ, Prov: at(n)})
		}
	})
	return out
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
		// SQLAlchemy: Column(Integer, ForeignKey("users.id")) — the key is a call among the arguments.
		if a.Type(l) == "call" && a.ChildCount() > 1 && strings.HasSuffix(a.Child(0).Text(src), "ForeignKey") && a.Child(1).NamedChildCount() > 0 {
			if v, ok := pyString(a.Child(1).NamedChild(0), l, src); ok {
				if out.Options == nil {
					out.Options = map[string]string{}
				}
				out.Options["foreign_key"] = v
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

// Names a function body calls that are Python's or a query builder's, never the application's.
var pyBuiltins = map[string]bool{"print": true, "len": true, "str": true, "int": true, "float": true, "bool": true,
	"list": true, "dict": true, "set": true, "tuple": true, "range": true, "isinstance": true, "getattr": true,
	"setattr": true, "hasattr": true, "sorted": true, "min": true, "max": true, "sum": true, "any": true, "all": true,
	"enumerate": true, "zip": true, "map": true, "filter": true, "open": true, "super": true, "type": true,
	"select": true, "col": true, "func": true, "Depends": true, "HTTPException": true}

// pyBody reads what a function calls — a function of its own module, one in a module it imported
// (crud.create_user), a method of its own object (self.save(), self.repo.get()), or a method of a
// local whose class the code states — and the tables its queries name: select(Item),
// session.get(User, id).
func pyBody(def *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) (invokes []archdoc.Invocation, queries []archdoc.Query) {
	// What the code says a local is: a parameter's annotation, or the class it was built from.
	locals := pyTyped(def, l, src)
	walk(def, func(n *ts.Node) {
		if n.Type(l) != "assignment" || n.NamedChildCount() < 2 {
			return
		}
		left, right := n.NamedChild(0), n.NamedChild(n.NamedChildCount()-1)
		if left.Type(l) == "identifier" && right.Type(l) == "call" && right.ChildCount() > 0 && right.Child(0).Type(l) == "identifier" {
			if _, stated := locals[left.Text(src)]; !stated {
				locals[left.Text(src)] = right.Child(0).Text(src)
			}
		}
	})
	walk(def, func(n *ts.Node) {
		if n.Type(l) != "call" || n.ChildCount() < 2 {
			return
		}
		fn, args := n.Child(0), n.Child(1)
		var first *ts.Node
		if args.Type(l) == "argument_list" && args.NamedChildCount() > 0 {
			first = args.NamedChild(0)
		}
		switch fn.Type(l) {
		case "identifier":
			name := fn.Text(src)
			if name == "select" && first != nil && first.Type(l) == "identifier" {
				queries = append(queries, archdoc.Query{Table: first.Text(src), Op: "reads", Prov: at(n)})
				return
			}
			if !pyBuiltins[name] {
				invokes = append(invokes, archdoc.Invocation{Method: name, Prov: at(n)})
			}
		case "attribute":
			if fn.NamedChildCount() != 2 {
				return
			}
			on, method := fn.NamedChild(0), fn.NamedChild(1).Text(src)
			if on.Type(l) == "attribute" && on.NamedChildCount() == 2 && on.NamedChild(0).Text(src) == "self" {
				invokes = append(invokes, archdoc.Invocation{Self: true, Object: on.NamedChild(1).Text(src), Method: method, Prov: at(n)})
				return
			}
			if on.Type(l) != "identifier" {
				return
			}
			obj := on.Text(src)
			if method == "get" && first != nil && first.Type(l) == "identifier" && strings.ToUpper(first.Text(src)[:1]) == first.Text(src)[:1] {
				queries = append(queries, archdoc.Query{Table: first.Text(src), Op: "reads", Prov: at(n)})
				return
			}
			if obj == "self" || obj == "cls" {
				invokes = append(invokes, archdoc.Invocation{Self: true, Method: method, Prov: at(n)})
				return
			}
			invokes = append(invokes, archdoc.Invocation{Object: obj, Method: method, Type: locals[obj], Prov: at(n)})
		}
	})
	return invokes, queries
}

// docstring is the first line of a function's docstring, and where it is.
func docstring(def *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) (string, archdoc.Provenance) {
	for i := 0; i < def.NamedChildCount(); i++ {
		b := def.NamedChild(i)
		if b.Type(l) != "block" || b.NamedChildCount() == 0 {
			continue
		}
		first := b.NamedChild(0)
		if first.Type(l) == "expression_statement" && first.NamedChildCount() > 0 {
			first = first.NamedChild(0)
		}
		if first.Type(l) != "string" {
			return "", archdoc.Provenance{}
		}
		v, _ := pyString(first, l, src)
		for _, line := range strings.Split(v, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				return strings.TrimSuffix(line, "."), at(first)
			}
		}
	}
	return "", archdoc.Provenance{}
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
