package code

import (
	"regexp"
	"strings"

	ts "github.com/odvcencio/gotreesitter"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// facts is everything one file says, before its imports are resolved.
type facts struct {
	imports []rawImport
	classes []archdoc.Class
	hosts   []archdoc.HostRef
	calls   []archdoc.Call
	prefix  *archdoc.Literal
	consts  []archdoc.Constant
	routers []archdoc.Router
	incs    []archdoc.Include
	exports []archdoc.Literal
	pages   []archdoc.Literal
}

func (f *facts) add(g facts) {
	f.imports = append(f.imports, g.imports...)
	f.classes = append(f.classes, g.classes...)
	f.hosts = append(f.hosts, g.hosts...)
	f.calls = append(f.calls, g.calls...)
	f.consts = append(f.consts, g.consts...)
	f.routers = append(f.routers, g.routers...)
	f.incs = append(f.incs, g.incs...)
	f.exports = append(f.exports, g.exports...)
	f.pages = append(f.pages, g.pages...)
	if f.prefix == nil {
		f.prefix = g.prefix
	}
}

// A URL with a host: what a configured endpoint looks like in code.
var urlLiteral = regexp.MustCompile(`^(https?|wss?|redis|rediss|postgres|postgresql|mysql|mongodb|amqp|amqps|nats|grpc)://(?:[^@/\s]*@)?([A-Za-z0-9_.-]+)(?::(\d+))?`)

// A key that names a host: host, hostname, redisHost, DB_HOSTNAME.
var hostKey = regexp.MustCompile(`(?i)(^|_|[a-z])host(name)?$`)

// A bare host name, as a host property's value holds one.
var hostName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// httpCallee names the functions that make an HTTP request, as they are called.
var httpCallee = regexp.MustCompile(`^(fetch|got|ky|axios|(axios|got|ky|http|https|this\.http|this\.httpService|httpService|request)\.(get|post|put|patch|delete|head|request))$`)

// scriptFacts reads a TypeScript or JavaScript file in one walk: imports, classes and their
// decorators, the hosts its literals name, and the HTTP calls it makes. offset places a script
// embedded in a Svelte component at its line in the file.
func scriptFacts(src []byte, file, lang string, offset int) (out facts, partial bool) {
	at := func(n *ts.Node) archdoc.Provenance {
		p := n.StartPoint()
		return archdoc.Provenance{File: file, Line: int(p.Row) + 1 + offset, Column: int(p.Column) + 1}
	}
	called := map[archdoc.Provenance]bool{}

	partial = parse(lang, src, func(root *ts.Node, l *ts.Language) {
		text := func(n *ts.Node) string { return n.Text(src) }
		walk(root, func(n *ts.Node) {
			switch n.Type(l) {
			case "import_statement", "export_statement":
				for i := 0; i < n.ChildCount(); i++ {
					c := n.Child(i)
					switch c.Type(l) {
					case "string":
						p := at(c)
						out.imports = append(out.imports, rawImport{spec: unquote(text(c)), line: p.Line, column: p.Column})
					case "function_declaration", "generator_function_declaration":
						if c.NamedChildCount() > 0 {
							out.exports = append(out.exports, archdoc.Literal{Value: text(c.NamedChild(0)), Prov: at(c)})
						}
					case "lexical_declaration", "variable_declaration":
						for j := 0; j < c.NamedChildCount(); j++ {
							if v := c.NamedChild(j); v.Type(l) == "variable_declarator" && v.NamedChildCount() > 0 {
								out.exports = append(out.exports, archdoc.Literal{Value: text(v.NamedChild(0)), Prov: at(v)})
							}
						}
					}
				}
			case "class_declaration", "abstract_class_declaration":
				out.classes = append(out.classes, classOf(n, l, src, at))
			case "enum_declaration":
				var name string
				for i := 0; i < n.ChildCount(); i++ {
					c := n.Child(i)
					switch c.Type(l) {
					case "identifier":
						name = text(c)
					case "enum_body":
						for j := 0; j < c.NamedChildCount(); j++ {
							a := c.NamedChild(j)
							if a.Type(l) != "enum_assignment" || a.NamedChildCount() < 2 {
								continue
							}
							if v, ok := literal(a.NamedChild(1), l, src); ok {
								out.consts = append(out.consts, archdoc.Constant{Name: name + "." + text(a.NamedChild(0)), Value: v, Prov: at(a)})
							}
						}
					}
				}
			case "call_expression":
				if n.ChildCount() < 2 {
					return
				}
				fn, args := n.Child(0), n.Child(1)
				if args.Type(l) != "arguments" {
					return
				}
				var first *ts.Node
				if args.NamedChildCount() > 0 {
					first = args.NamedChild(0)
				}
				callee := text(fn)
				switch {
				case fn.Type(l) == "import" || callee == "require":
					if first != nil && first.Type(l) == "string" {
						p := at(first)
						out.imports = append(out.imports, rawImport{spec: unquote(text(first)), line: p.Line, column: p.Column})
					}
				case callee == "createFileRoute":
					if first != nil && first.Type(l) == "string" {
						out.pages = append(out.pages, archdoc.Literal{Value: unquote(text(first)), Prov: at(first)})
					}
				case strings.HasSuffix(callee, "setGlobalPrefix"):
					if first != nil && first.Type(l) == "string" {
						out.prefix = &archdoc.Literal{Value: unquote(text(first)), Prov: at(first)}
					}
				case httpCallee.MatchString(callee) && first != nil:
					if v, ok := literal(first, l, src); ok && urlLiteral.MatchString(v) {
						called[at(first)] = true
						return
					}
					out.calls = append(out.calls, archdoc.Call{Callee: callee, Target: shorten(text(first)), Prov: at(n)})
				}
			case "string", "template_string":
				if v, ok := urlPrefix(n, l, src); ok {
					if h, ok := hostOf(v); ok {
						h.Prov = at(n)
						out.hosts = append(out.hosts, h)
					}
				}
			case "pair", "variable_declarator":
				if n.NamedChildCount() < 2 {
					return
				}
				key, value := n.NamedChild(0), n.NamedChild(n.NamedChildCount()-1)
				if !hostKey.MatchString(unquote(text(key))) {
					return
				}
				// host: 'redis', or host: env.REDIS_HOSTNAME || 'redis' — the default is the literal.
				if value.Type(l) == "binary_expression" && value.ChildCount() == 3 {
					if op := text(value.Child(1)); op == "||" || op == "??" {
						value = value.Child(2)
					}
				}
				if v, ok := literal(value, l, src); ok && hostName.MatchString(v) {
					out.hosts = append(out.hosts, archdoc.HostRef{Host: v, Value: v, Prov: at(value)})
				}
			}
		})
	})
	for i := range out.hosts {
		out.hosts[i].Called = called[out.hosts[i].Prov]
	}
	return out, partial
}

// classOf reads a class: its decorators — written before it, inside an export statement, or on
// the declaration itself — what its constructor takes, and its decorated methods.
func classOf(n *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Class {
	c := archdoc.Class{Prov: at(n)}
	var decorators []archdoc.Decorator
	if parent := n.Parent(); parent != nil && parent.Type(l) == "export_statement" {
		for i := 0; i < parent.ChildCount(); i++ {
			if d := parent.Child(i); d.Type(l) == "decorator" {
				decorators = append(decorators, decoratorOf(d, l, src, at))
			}
		}
	}
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		switch ch.Type(l) {
		case "decorator":
			decorators = append(decorators, decoratorOf(ch, l, src, at))
		case "type_identifier", "identifier":
			if c.Name == "" {
				c.Name = ch.Text(src)
			}
		case "class_heritage":
			walk(ch, func(h *ts.Node) {
				if h.Type(l) == "extends_clause" && h.NamedChildCount() > 0 {
					c.Extends = append(c.Extends, h.NamedChild(0).Text(src))
				}
			})
		case "class_body":
			var pending []archdoc.Decorator
			for j := 0; j < ch.ChildCount(); j++ {
				m := ch.Child(j)
				switch m.Type(l) {
				case "decorator":
					pending = append(pending, decoratorOf(m, l, src, at))
				case "method_definition":
					name := ""
					for k := 0; k < m.ChildCount(); k++ {
						if t := m.Child(k).Type(l); t == "property_identifier" || t == "private_property_identifier" {
							name = m.Child(k).Text(src)
							break
						}
					}
					if name == "constructor" {
						c.Injects = injected(m, l, src, at)
						c.Params = params(m, l, src, at)
					} else {
						meth := archdoc.Method{Name: name, Decorators: pending, Prov: at(m), EndLine: at(m).Line + int(m.EndPoint().Row-m.StartPoint().Row)}
						meth.Invokes, meth.Queries = body(m, l, src, at)
						c.Methods = append(c.Methods, meth)
					}
					pending = nil
				case "public_field_definition":
					f := archdoc.Field{Prov: at(m), Decorators: pending}
					for k := 0; k < m.ChildCount(); k++ {
						part := m.Child(k)
						switch part.Type(l) {
						case "decorator":
							f.Decorators = append(f.Decorators, decoratorOf(part, l, src, at))
						case "property_identifier":
							f.Name = part.Text(src)
						case "type_annotation":
							if part.NamedChildCount() > 0 {
								f.Type = part.NamedChild(0).Text(src)
							}
						}
					}
					if len(f.Decorators) > 0 {
						c.Fields = append(c.Fields, f)
					}
					pending = nil
				case "{", "}", ";", "comment":
				default:
					pending = nil
				}
			}
		}
	}
	c.Decorators = decorators
	return c
}

// injected lists the types a constructor takes: constructor(private service: AlbumService).
func injected(ctor *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Literal {
	var out []archdoc.Literal
	for i := 0; i < ctor.ChildCount(); i++ {
		params := ctor.Child(i)
		if params.Type(l) != "formal_parameters" {
			continue
		}
		for j := 0; j < params.NamedChildCount(); j++ {
			p := params.NamedChild(j)
			for k := 0; k < p.ChildCount(); k++ {
				ann := p.Child(k)
				if ann.Type(l) != "type_annotation" {
					continue
				}
				for t := 0; t < ann.NamedChildCount(); t++ {
					ty := ann.NamedChild(t)
					name := ty.Text(src)
					if ty.Type(l) == "generic_type" && ty.NamedChildCount() > 0 {
						name = ty.NamedChild(0).Text(src)
					}
					out = append(out, archdoc.Literal{Value: name, Prov: at(ty)})
					break
				}
			}
		}
	}
	return out
}

// params lists a constructor's parameters by name and type: (private service: AlbumService).
func params(ctor *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Param {
	var out []archdoc.Param
	for i := 0; i < ctor.ChildCount(); i++ {
		ps := ctor.Child(i)
		if ps.Type(l) != "formal_parameters" {
			continue
		}
		for j := 0; j < ps.NamedChildCount(); j++ {
			p := ps.NamedChild(j)
			var name, typ string
			for k := 0; k < p.ChildCount(); k++ {
				switch ch := p.Child(k); ch.Type(l) {
				case "identifier":
					name = ch.Text(src)
				case "type_annotation":
					if ch.NamedChildCount() > 0 {
						ty := ch.NamedChild(0)
						typ = ty.Text(src)
						if ty.Type(l) == "generic_type" && ty.NamedChildCount() > 0 {
							typ = ty.NamedChild(0).Text(src)
						}
					}
				}
			}
			if name != "" && typ != "" {
				out = append(out, archdoc.Param{Name: name, Type: typ, Prov: at(p)})
			}
		}
	}
	return out
}

// Query builders' table methods: Kysely's, and the verb each one is.
var queryOps = map[string]string{"selectFrom": "reads", "insertInto": "writes", "updateTable": "updates", "deleteFrom": "deletes", "mergeInto": "writes"}

// body reads what a method does to its own object, and which tables its queries name.
func body(m *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) (invokes []archdoc.Invocation, queries []archdoc.Query) {
	walk(m, func(n *ts.Node) {
		if fn := misreadGenericCall(n, l); fn != nil {
			// await this.predict<T>(…), read by the parser as (await this.predict) < T > (…).
			if fn.Type(l) == "member_expression" && fn.NamedChildCount() == 2 {
				prop, obj := fn.NamedChild(1).Text(src), fn.NamedChild(0)
				switch {
				case obj.Type(l) == "this":
					invokes = append(invokes, archdoc.Invocation{Method: prop, Prov: at(n)})
				case obj.Type(l) == "member_expression" && obj.NamedChildCount() == 2 && obj.NamedChild(0).Type(l) == "this":
					invokes = append(invokes, archdoc.Invocation{Object: obj.NamedChild(1).Text(src), Method: prop, Prov: at(n)})
				}
			}
			return
		}
		if n.Type(l) != "call_expression" || n.ChildCount() < 2 {
			return
		}
		fn := n.Child(0)
		if fn.Type(l) != "member_expression" || fn.NamedChildCount() < 2 {
			return
		}
		prop := fn.NamedChild(fn.NamedChildCount() - 1).Text(src)
		if op, ok := queryOps[prop]; ok {
			if args := n.Child(1); args.NamedChildCount() > 0 {
				if t, ok := literal(args.NamedChild(0), l, src); ok {
					t, _, _ = strings.Cut(t, " as ") // 'album_user as au': the alias is the query's, not the table's
					queries = append(queries, archdoc.Query{Table: strings.TrimSpace(t), Op: op, Prov: at(n)})
				}
			}
			return
		}
		obj := fn.NamedChild(0)
		switch {
		case obj.Type(l) == "this":
			invokes = append(invokes, archdoc.Invocation{Method: prop, Prov: at(n)})
		case obj.Type(l) == "member_expression" && obj.NamedChildCount() == 2 && obj.NamedChild(0).Type(l) == "this":
			invokes = append(invokes, archdoc.Invocation{Object: obj.NamedChild(1).Text(src), Method: prop, Prov: at(n)})
		}
	})
	return invokes, queries
}

// misreadGenericCall recognises a call with type arguments that the parser read as two
// comparisons — `await this.predict<T>(a, b)` as `(await this.predict) < T > (a, b)` — a known
// gap in the TypeScript grammar's runtime (docs/decisions.md, 5 Oct), and returns the function the
// call names. A real comparison is never followed by a parenthesised argument list.
func misreadGenericCall(n *ts.Node, l *ts.Language) *ts.Node {
	if n.Type(l) != "binary_expression" || n.ChildCount() != 3 || n.Child(1).Type(l) != ">" {
		return nil
	}
	if n.Child(2).Type(l) != "parenthesized_expression" {
		return nil
	}
	left := n.Child(0)
	if left.Type(l) != "binary_expression" || left.ChildCount() != 3 || left.Child(1).Type(l) != "<" {
		return nil
	}
	fn := left.Child(0)
	if fn.Type(l) == "await_expression" && fn.NamedChildCount() > 0 {
		fn = fn.NamedChild(0)
	}
	return fn
}

// decoratorOf reads @Name, @Name('arg'), @Name({ path: 'arg', summary: '…' }).
func decoratorOf(d *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Decorator {
	out := archdoc.Decorator{Prov: at(d)}
	if d.NamedChildCount() == 0 {
		return out
	}
	e := d.NamedChild(0)
	if e.Type(l) != "call_expression" {
		out.Name = e.Text(src)
		return out
	}
	out.Name = e.Child(0).Text(src)
	if e.ChildCount() < 2 {
		return out
	}
	args := e.Child(1)
	for i := 0; i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if i == 0 {
			if v, ok := literal(a, l, src); ok {
				out.Arg, out.HasArg = v, true
			} else if a.Type(l) == "arrow_function" {
				// () => AssetTable: the class at a relation's other end.
				if body := a.NamedChild(a.NamedChildCount() - 1); body != nil && body.Type(l) == "identifier" {
					out.Target = body.Text(src)
				}
			} else if a.Type(l) != "object" {
				out.ArgExpr = a.Text(src)
			}
		}
		if a.Type(l) != "object" {
			continue
		}
		for j := 0; j < a.NamedChildCount(); j++ {
			pair := a.NamedChild(j)
			if pair.Type(l) != "pair" || pair.NamedChildCount() < 2 {
				continue
			}
			key, value := unquote(pair.NamedChild(0).Text(src)), pair.NamedChild(1)
			v, ok := literal(value, l, src)
			if !ok {
				switch value.Type(l) {
				case "true", "false", "number", "null":
					v, ok = value.Text(src), true
				}
			}
			if !ok {
				continue
			}
			if out.Options == nil {
				out.Options = map[string]string{}
			}
			out.Options[key] = v
			switch key {
			case "summary":
				out.Summary, out.SummaryProv = v, at(value)
			case "path":
				if i == 0 {
					out.Arg, out.HasArg = v, true
				}
			}
		}
	}
	return out
}

// literal is a string node's value, or a template string with nothing substituted.
func literal(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	switch n.Type(l) {
	case "string":
		return unquote(n.Text(src)), true
	case "template_string":
		for i := 0; i < n.ChildCount(); i++ {
			if n.Child(i).Type(l) == "template_substitution" {
				return "", false
			}
		}
		return unquote(n.Text(src)), true
	}
	return "", false
}

// urlPrefix is the literal start of a string: the whole of a plain string, or a template string
// up to its first substitution — `http://ml:3003/${path}` still names its host.
func urlPrefix(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	if v, ok := literal(n, l, src); ok {
		return v, true
	}
	if n.Type(l) != "template_string" {
		return "", false
	}
	var b strings.Builder
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c.Type(l) == "template_substitution" {
			break
		}
		if c.Type(l) == "string_fragment" {
			b.WriteString(c.Text(src))
		}
	}
	return b.String(), b.Len() > 0
}

func hostOf(v string) (archdoc.HostRef, bool) {
	m := urlLiteral.FindStringSubmatch(v)
	if m == nil {
		return archdoc.HostRef{}, false
	}
	return archdoc.HostRef{Scheme: m[1], Host: m[2], Port: m[3], Value: v}, true
}

func unquote(s string) string {
	return strings.Trim(s, "'\"`")
}

func shorten(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		return s[:77] + "…"
	}
	return s
}
