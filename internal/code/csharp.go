package code

import (
	"regexp"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// C# is read as Java is — classes with their attributes as decorators, fields and properties, what
// the constructor is given, calls through fields, URLs in literals — with two differences that come
// from the language. A using directive imports a namespace, not a file, and a namespace is usually
// spread over many files and projects; so which file uses which is read from the types a file names,
// matched to the file that declares each one. And ASP.NET declares routes two ways: attributes on a
// controller ([HttpGet("{id}")]), read like Spring's, and calls — app.MapGet("/items", …) — read
// like Express's, with app.MapGroup("/api") as a router with a prefix.

// csHTTP are HttpClient's methods that take the address they call.
var csHTTP = map[string]bool{"GetAsync": true, "PostAsync": true, "PutAsync": true, "PatchAsync": true, "DeleteAsync": true,
	"SendAsync": true, "GetStringAsync": true, "GetStreamAsync": true, "GetByteArrayAsync": true, "GetFromJsonAsync": true,
	"PostAsJsonAsync": true, "PutAsJsonAsync": true, "PatchAsJsonAsync": true, "DeleteFromJsonAsync": true}

// csClient is a receiver that is an HTTP client by its name: _httpClient, client, Http.
var csClient = regexp.MustCompile(`(?i)(client|http)$`)

// Minimal APIs: the method that declares a route, by the verb it answers.
var csMap = map[string]string{"MapGet": "get", "MapPost": "post", "MapPut": "put", "MapPatch": "patch", "MapDelete": "delete"}

// The types an endpoint is mapped on: a parameter of one of them is the application, or a group of it.
var csRouteBuilders = map[string]bool{"WebApplication": true, "IEndpointRouteBuilder": true, "RouteGroupBuilder": true, "IApplicationBuilder": true}

// isCSharp is a C# file of the application's own: not one a tool generated.
func isCSharp(name string) bool {
	if !strings.HasSuffix(name, ".cs") {
		return false
	}
	for _, generated := range []string{".Designer.cs", ".g.cs", ".g.i.cs", "ModelSnapshot.cs", "AssemblyInfo.cs"} {
		if strings.HasSuffix(name, generated) {
			return false
		}
	}
	return true
}

func csharpFacts(src []byte, file string) (out facts, partial bool) {
	at := func(n *ts.Node) archdoc.Provenance {
		p := n.StartPoint()
		return archdoc.Provenance{File: file, Line: int(p.Row) + 1, Column: int(p.Column) + 1}
	}
	called := map[archdoc.Provenance]bool{}
	module := archdoc.Class{Prov: archdoc.Provenance{File: file, Line: 1}}
	mentioned := map[string]bool{}
	partial = parse("C#", src, func(root *ts.Node, l *ts.Language) {
		text := func(n *ts.Node) string { return n.Text(src) }
		walk(root, func(n *ts.Node) {
			switch n.Type(l) {
			case "using_directive":
				for i := 0; i < n.ChildCount(); i++ {
					if c := n.Child(i); c.Type(l) == "name_equals" || c.Type(l) == "=" {
						return // using Alias = Some.Type: names one type, read as a mention
					}
				}
				if n.NamedChildCount() == 0 {
					return
				}
				p := at(n)
				out.imports = append(out.imports, rawImport{spec: text(n.NamedChild(n.NamedChildCount() - 1)), line: p.Line, column: p.Column})
			case "namespace_declaration", "file_scoped_namespace_declaration":
				if name := n.ChildByFieldName("name", l); name != nil {
					out.namespaces = append(out.namespaces, text(name))
				}
			case "class_declaration", "interface_declaration", "record_declaration", "struct_declaration":
				c := csClass(n, l, src, at)
				out.classes = append(out.classes, c)
				out.declares = append(out.declares, c.Name)
				if strings.HasSuffix(file, ".cshtml.cs") && containsName(c.Extends, "PageModel") {
					if p, ok := razorPage(file); ok {
						out.pages = append(out.pages, archdoc.Literal{Value: p, Prov: c.Prov})
					}
				}
			case "enum_declaration", "delegate_declaration":
				if name := n.ChildByFieldName("name", l); name != nil {
					out.declares = append(out.declares, text(name))
				}
			case "identifier":
				// A type this file names — a mention, matched later to the file that declares it.
				if t := text(n); t != "" && t[0] >= 'A' && t[0] <= 'Z' && !mentioned[t] {
					mentioned[t] = true
					p := at(n)
					out.refs = append(out.refs, rawImport{spec: t, line: p.Line, column: p.Column})
				}
			case "parameter":
				// A parameter of type WebApplication or IEndpointRouteBuilder is where routes are mapped.
				typ, name := n.ChildByFieldName("type", l), n.ChildByFieldName("name", l)
				if typ != nil && name != nil && csRouteBuilders[csTypeName(typ, l, src)] {
					out.routers = append(out.routers, archdoc.Router{Var: text(name), Kind: "aspnet", Prov: at(n)})
				}
			case "variable_declarator":
				name := n.ChildByFieldName("name", l)
				if name == nil && n.NamedChildCount() > 0 {
					name = n.NamedChild(0)
				}
				if name == nil || n.NamedChildCount() < 2 {
					return
				}
				value := n.NamedChild(n.NamedChildCount() - 1)
				// var app = builder.Build(): the application. var api = app.MapGroup("/api"): a group
				// of it, under a prefix.
				if prefix, root, ok := csGroup(value, l, src); ok {
					out.routers = append(out.routers, archdoc.Router{Var: text(name), Kind: "group", Prefix: prefix, Prov: at(n)})
					out.incs = append(out.incs, archdoc.Include{Parent: root, Child: text(name), Prov: at(n)})
				} else if csCalls(value, l, src, "Build") {
					out.routers = append(out.routers, archdoc.Router{Var: text(name), Kind: "aspnet", Prov: at(n)})
				}
			case "invocation_expression":
				fn, args := n.ChildByFieldName("function", l), n.ChildByFieldName("arguments", l)
				if fn == nil || args == nil || fn.Type(l) != "member_access_expression" {
					return
				}
				obj, name := fn.ChildByFieldName("expression", l), fn.ChildByFieldName("name", l)
				if obj == nil || name == nil {
					return
				}
				first := csArg(args, 0)
				verb, isMap := csMap[text(name)]
				switch {
				case isMap && obj.Type(l) == "identifier" && first != nil:
					v, ok := csString(first, l, src)
					if !ok {
						return
					}
					handler := "handler"
					if h := csArg(args, 1); h != nil && (h.Type(l) == "identifier" || h.Type(l) == "member_access_expression") {
						handler = text(h)
					} else if m := enclosing(n, l, "method_declaration"); m != nil && m.ChildByFieldName("name", l) != nil {
						handler = text(m.ChildByFieldName("name", l))
					}
					module.Methods = append(module.Methods, archdoc.Method{Name: handler, Prov: at(n),
						Decorators: []archdoc.Decorator{{Name: text(obj) + "." + verb, Arg: v, HasArg: true, Prov: at(n)}}})
				case csHTTP[text(name)] && first != nil:
					target := first
					if target.Type(l) == "object_creation_expression" { // new Uri("http://…")
						target = csArg(target.ChildByFieldName("arguments", l), 0)
					}
					if target != nil {
						if v, ok := csString(target, l, src); ok && urlLiteral.MatchString(v) {
							called[at(target)] = true
							return
						}
					}
					if csClient.MatchString(text(obj)) {
						out.calls = append(out.calls, archdoc.Call{Callee: shorten(text(obj)) + "." + text(name), Target: shorten(text(first)), Prov: at(n)})
					}
				}
			case "string_literal", "verbatim_string_literal", "interpolated_string_expression":
				v, whole := csPrefix(n, l, src)
				h, ok := hostOf(v)
				if !ok {
					return
				}
				h.Prov, h.Built, h.Key = at(n), !whole, csKey(n, l, src)
				out.hosts = append(out.hosts, h)
			}
		})
	})
	for i := range out.hosts {
		out.hosts[i].Called = called[out.hosts[i].Prov]
	}
	if len(module.Methods) > 0 {
		out.classes = append(out.classes, module)
	}
	// A type the file declares itself is not a mention of another file.
	own := map[string]bool{}
	for _, d := range out.declares {
		own[d] = true
	}
	refs := out.refs[:0]
	for _, r := range out.refs {
		if !own[r.spec] {
			refs = append(refs, r)
		}
	}
	out.refs = refs
	return out, partial
}

func containsName(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// razorPage is the path a Razor page's code-behind answers: Pages/Basket/Checkout.cshtml.cs is
// /Basket/Checkout, an Index is its folder's, and a page in an area is under the area's name.
func razorPage(file string) (string, bool) {
	i := strings.LastIndex(file, "/Pages/")
	if i < 0 {
		return "", false
	}
	rel := strings.TrimSuffix(file[i+len("/Pages/"):], ".cshtml.cs")
	rel = strings.TrimSuffix(strings.TrimSuffix(rel, "Index"), "/")
	area := ""
	if j := strings.LastIndex(file[:i], "/Areas/"); j >= 0 {
		area = "/" + file[j+len("/Areas/"):i]
	}
	return area + "/" + rel, true
}

func enclosing(n *ts.Node, l *ts.Language, kind string) *ts.Node {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Type(l) == kind {
			return p
		}
	}
	return nil
}

// csArg is an argument list's i-th argument expression.
func csArg(args *ts.Node, i int) *ts.Node {
	if args == nil {
		return nil
	}
	k := 0
	for j := 0; j < args.NamedChildCount(); j++ {
		a := args.NamedChild(j)
		if k == i {
			if a.NamedChildCount() > 0 {
				return a.NamedChild(a.NamedChildCount() - 1)
			}
			return a
		}
		k++
	}
	return nil
}

// csGroup finds app.MapGroup("/api") in a chain of calls — app.MapGroup("/api").WithTags("x") — and
// the variable the chain starts from.
func csGroup(n *ts.Node, l *ts.Language, src []byte) (prefix, root string, ok bool) {
	walk(n, func(x *ts.Node) {
		if ok || x.Type(l) != "invocation_expression" {
			return
		}
		fn := x.ChildByFieldName("function", l)
		if fn == nil || fn.Type(l) != "member_access_expression" {
			return
		}
		if name := fn.ChildByFieldName("name", l); name == nil || name.Text(src) != "MapGroup" {
			return
		}
		v, isString := csString(csArg(x.ChildByFieldName("arguments", l), 0), l, src)
		if !isString {
			return
		}
		r := fn.ChildByFieldName("expression", l)
		for r != nil && r.Type(l) != "identifier" {
			switch r.Type(l) {
			case "invocation_expression":
				r = r.ChildByFieldName("function", l)
			case "member_access_expression":
				r = r.ChildByFieldName("expression", l)
			default:
				r = nil
			}
		}
		if r != nil {
			prefix, root, ok = v, r.Text(src), true
		}
	})
	return
}

// csCalls reports a call of the named method anywhere in n: builder.Build().
func csCalls(n *ts.Node, l *ts.Language, src []byte, method string) bool {
	if n.Type(l) != "invocation_expression" {
		return false
	}
	fn := n.ChildByFieldName("function", l)
	if fn == nil || fn.Type(l) != "member_access_expression" {
		return false
	}
	name := fn.ChildByFieldName("name", l)
	return name != nil && name.Text(src) == method
}

// csString is a plain string literal's value: "…" or @"…".
func csString(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Type(l) {
	case "string_literal":
		var b strings.Builder
		for i := 0; i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c.Type(l) == "string_literal_content" {
				b.WriteString(c.Text(src))
			}
		}
		return b.String(), true
	case "verbatim_string_literal":
		t := n.Text(src)
		if len(t) >= 3 {
			return t[2 : len(t)-1], true
		}
	}
	return "", false
}

// csPrefix is a string's literal start: the whole of a plain string, or an interpolated one up to its
// first hole — $"http://catalog/{id}" still names its host.
func csPrefix(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	if v, ok := csString(n, l, src); ok {
		return v, true
	}
	t := n.Text(src)
	t = strings.TrimLeft(t, "$@")
	t = strings.TrimPrefix(t, `"`)
	if i := strings.IndexByte(t, '{'); i >= 0 {
		return t[:i], false
	}
	return strings.TrimSuffix(t, `"`), true
}

// csKey is what a literal is the value of: a field or variable, a property, a property set by
// assignment — BaseAddress = new Uri("…") — or the method it is handed to.
func csKey(n *ts.Node, l *ts.Language, src []byte) string {
	for p, hops := n.Parent(), 0; p != nil && hops < 6; p, hops = p.Parent(), hops+1 {
		switch p.Type(l) {
		case "argument", "argument_list", "binary_expression", "parenthesized_expression", "equals_value_clause":
			continue
		case "object_creation_expression":
			if t := p.ChildByFieldName("type", l); t != nil && csTypeName(t, l, src) == "Uri" {
				continue // new Uri("…"): the key is where the Uri goes
			}
			return ""
		case "variable_declarator":
			if name := p.ChildByFieldName("name", l); name != nil {
				return name.Text(src)
			}
			if p.NamedChildCount() > 0 {
				return p.NamedChild(0).Text(src)
			}
		case "property_declaration":
			if name := p.ChildByFieldName("name", l); name != nil {
				return name.Text(src)
			}
		case "assignment_expression":
			if left := p.ChildByFieldName("left", l); left != nil {
				t := left.Text(src)
				return t[strings.LastIndex(t, ".")+1:]
			}
		case "invocation_expression":
			if fn := p.ChildByFieldName("function", l); fn != nil {
				t := fn.Text(src)
				return t[strings.LastIndex(t, ".")+1:]
			}
		}
		return ""
	}
	return ""
}

// csTypeName is a type without its arguments, namespace or nullability: List<Order> is List,
// Microsoft.Foo.Bar is Bar, Order? is Order.
func csTypeName(n *ts.Node, l *ts.Language, src []byte) string {
	switch n.Type(l) {
	case "generic_name":
		if n.NamedChildCount() > 0 {
			return n.NamedChild(0).Text(src)
		}
	case "qualified_name":
		if name := n.ChildByFieldName("name", l); name != nil {
			return csTypeName(name, l, src)
		}
	case "nullable_type":
		if t := n.ChildByFieldName("type", l); t != nil {
			return csTypeName(t, l, src)
		}
		if n.NamedChildCount() > 0 {
			return csTypeName(n.NamedChild(0), l, src)
		}
	}
	return n.Text(src)
}

// csElement is what a collection holds — List<Order> holds Order — or the type itself.
func csElement(n *ts.Node, l *ts.Language, src []byte) string {
	if n.Type(l) == "nullable_type" && n.NamedChildCount() > 0 {
		return csElement(n.NamedChild(0), l, src)
	}
	if n.Type(l) == "generic_name" {
		for i := 0; i < n.NamedChildCount(); i++ {
			if args := n.NamedChild(i); args.Type(l) == "type_argument_list" && args.NamedChildCount() > 0 {
				return csTypeName(args.NamedChild(args.NamedChildCount()-1), l, src)
			}
		}
	}
	return csTypeName(n, l, src)
}

func csClass(n *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Class {
	text := func(n *ts.Node) string { return n.Text(src) }
	c := archdoc.Class{Prov: at(n), Interface: n.Type(l) == "interface_declaration"}
	if name := n.ChildByFieldName("name", l); name != nil {
		c.Name = text(name)
	}
	for i := 0; i < n.NamedChildCount(); i++ {
		ch := n.NamedChild(i)
		switch ch.Type(l) {
		case "attribute_list":
			c.Decorators = append(c.Decorators, csAttributes(ch, l, src, at)...)
		case "base_list":
			for j := 0; j < ch.NamedChildCount(); j++ {
				if t := ch.NamedChild(j); t.Type(l) != "argument_list" {
					c.Extends = append(c.Extends, csTypeName(t, l, src))
				}
			}
		case "parameter_list": // a primary constructor: class OrderService(IOrderRepository orders)
			for _, p := range csParams(ch, l, src, at) {
				c.Params = append(c.Params, p)
				c.Injects = append(c.Injects, archdoc.Literal{Value: p.Type, Prov: p.Prov})
				if n.Type(l) == "record_declaration" {
					c.Fields = append(c.Fields, archdoc.Field{Name: p.Name, Type: p.Type, Prov: p.Prov})
				}
			}
		}
	}
	body := n.ChildByFieldName("body", l)
	if body == nil {
		return c
	}
	for i := 0; i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		switch m.Type(l) {
		case "field_declaration", "property_declaration":
			var decs []archdoc.Decorator
			static := false
			for j := 0; j < m.NamedChildCount(); j++ {
				switch x := m.NamedChild(j); x.Type(l) {
				case "attribute_list":
					decs = append(decs, csAttributes(x, l, src, at)...)
				case "modifier":
					static = static || text(x) == "static" || text(x) == "const"
				}
			}
			if static {
				continue
			}
			add := func(name string, typ *ts.Node, value *ts.Node, at0 archdoc.Provenance) {
				f := archdoc.Field{Name: name, Type: text(typ), Decorators: decs, Prov: at0}
				if v, ok := csString(value, l, src); ok {
					f.Value = v
				}
				// What a property of a collection holds, for a table's relations.
				if el := csElement(typ, l, src); el != csTypeName(typ, l, src) {
					f.Decorators = append(append([]archdoc.Decorator(nil), decs...), archdoc.Decorator{Name: "Collection", Target: el, Prov: at0})
				}
				c.Fields = append(c.Fields, f)
				c.Params = append(c.Params, archdoc.Param{Name: name, Type: csTypeName(typ, l, src), Prov: at0})
			}
			if m.Type(l) == "property_declaration" {
				typ, name := m.ChildByFieldName("type", l), m.ChildByFieldName("name", l)
				// int Total => _items.Sum(…): computed each time, not held — no accessors, nothing stored.
				// A collection read that way still says what the class holds many of.
				if typ != nil && name != nil && (m.ChildByFieldName("accessors", l) != nil || csElement(typ, l, src) != csTypeName(typ, l, src)) {
					add(text(name), typ, m.ChildByFieldName("value", l), at(m))
				}
				continue
			}
			for j := 0; j < m.NamedChildCount(); j++ {
				decl := m.NamedChild(j)
				if decl.Type(l) != "variable_declaration" {
					continue
				}
				typ := decl.ChildByFieldName("type", l)
				for k := 0; typ != nil && k < decl.NamedChildCount(); k++ {
					v := decl.NamedChild(k)
					if v.Type(l) != "variable_declarator" || v.NamedChildCount() == 0 {
						continue
					}
					var value *ts.Node
					if v.NamedChildCount() > 1 {
						value = v.NamedChild(v.NamedChildCount() - 1)
					}
					add(text(v.NamedChild(0)), typ, value, at(v))
				}
			}
		case "constructor_declaration":
			if ps := m.ChildByFieldName("parameters", l); ps != nil {
				for _, p := range csParams(ps, l, src, at) {
					c.Params = append(c.Params, p)
					c.Injects = append(c.Injects, archdoc.Literal{Value: p.Type, Prov: p.Prov})
				}
			}
		case "method_declaration":
			meth := archdoc.Method{Prov: at(m), EndLine: int(m.EndPoint().Row) + 1}
			if name := m.ChildByFieldName("name", l); name != nil {
				meth.Name = text(name)
			}
			for j := 0; j < m.NamedChildCount(); j++ {
				if x := m.NamedChild(j); x.Type(l) == "attribute_list" {
					meth.Decorators = append(meth.Decorators, csAttributes(x, l, src, at)...)
				}
			}
			body := m.ChildByFieldName("body", l)
			if body == nil {
				for j := 0; j < m.NamedChildCount(); j++ {
					if x := m.NamedChild(j); x.Type(l) == "arrow_expression_clause" {
						body = x
					}
				}
			}
			if body != nil {
				walk(body, func(x *ts.Node) {
					if x.Type(l) != "invocation_expression" {
						return
					}
					fn := x.ChildByFieldName("function", l)
					if fn == nil {
						return
					}
					switch fn.Type(l) {
					case "identifier":
						meth.Invokes = append(meth.Invokes, archdoc.Invocation{Method: text(fn), Prov: at(x)})
					case "member_access_expression":
						obj, name := fn.ChildByFieldName("expression", l), fn.ChildByFieldName("name", l)
						if obj == nil || name == nil {
							return
						}
						inv := archdoc.Invocation{Method: csTypeName(name, l, src), Prov: at(x)}
						switch {
						case obj.Type(l) == "identifier":
							inv.Object = text(obj)
						case obj.Type(l) == "member_access_expression" && obj.ChildByFieldName("expression", l) != nil &&
							obj.ChildByFieldName("expression", l).Type(l) == "this_expression":
							inv.Object = text(obj.ChildByFieldName("name", l))
						case obj.Type(l) == "this_expression":
						default:
							return
						}
						meth.Invokes = append(meth.Invokes, inv)
					}
				})
			}
			c.Methods = append(c.Methods, meth)
		}
	}
	return c
}

func csParams(ps *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Param {
	var out []archdoc.Param
	for i := 0; i < ps.NamedChildCount(); i++ {
		p := ps.NamedChild(i)
		if p.Type(l) != "parameter" {
			continue
		}
		typ, name := p.ChildByFieldName("type", l), p.ChildByFieldName("name", l)
		if typ != nil && name != nil {
			out = append(out, archdoc.Param{Name: name.Text(src), Type: csTypeName(typ, l, src), Prov: at(p)})
		}
	}
	return out
}

// csAttributes are the attributes an attribute list holds, as decorators: [HttpGet("{id}")] is HttpGet
// with argument "{id}"; [Table("Orders", Schema = "x")] has the option Schema.
func csAttributes(list *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Decorator {
	var out []archdoc.Decorator
	for i := 0; i < list.NamedChildCount(); i++ {
		a := list.NamedChild(i)
		if a.Type(l) != "attribute" {
			continue
		}
		d := archdoc.Decorator{Prov: at(a)}
		if name := a.ChildByFieldName("name", l); name != nil {
			d.Name = strings.TrimSuffix(csTypeName(name, l, src), "Attribute")
		}
		for j := 0; j < a.NamedChildCount(); j++ {
			args := a.NamedChild(j)
			if args.Type(l) != "attribute_argument_list" {
				continue
			}
			for k := 0; k < args.NamedChildCount(); k++ {
				arg := args.NamedChild(k)
				if arg.NamedChildCount() == 0 {
					continue
				}
				value := arg.NamedChild(arg.NamedChildCount() - 1)
				key := ""
				if arg.NamedChildCount() > 1 {
					key = strings.TrimRight(strings.TrimSpace(arg.NamedChild(0).Text(src)), "=: ")
				}
				v, ok := csString(value, l, src)
				if !ok && (value.Type(l) == "boolean_literal" || value.Type(l) == "integer_literal") {
					v, ok = value.Text(src), true
				}
				switch {
				case key != "" && ok:
					if d.Options == nil {
						d.Options = map[string]string{}
					}
					d.Options[key] = v
				case key != "":
					if d.Exprs == nil {
						d.Exprs = map[string]string{}
					}
					d.Exprs[key] = value.Text(src)
				case ok && !d.HasArg:
					d.Arg, d.HasArg = v, true
				case !ok && d.ArgExpr == "":
					d.ArgExpr = value.Text(src)
				}
			}
		}
		out = append(out, d)
	}
	return out
}

// csResolver ties what a C# file uses to the files of the application: a type it names, to the file
// that declares it; a using directive, to a package when the namespace is not the application's own
// (its own namespaces are read through the types instead).
type csResolver struct {
	types map[string][]string // a type's name → the files that declare it
	own   []string            // the namespaces the application declares
}

func newCSResolver(files []string, got map[string]facts) *csResolver {
	r := &csResolver{types: map[string][]string{}}
	seen := map[string]bool{}
	for _, f := range files {
		for _, d := range got[f].declares {
			r.types[d] = append(r.types[d], f)
		}
		for _, ns := range got[f].namespaces {
			if !seen[ns] {
				seen[ns] = true
				r.own = append(r.own, ns)
			}
		}
	}
	sort.Strings(r.own)
	return r
}

// csMention marks a raw import as a type the file names, not a using directive.
const csMention = "\x00type:"

func (r *csResolver) resolve(file string, imp rawImport) []archdoc.Import {
	if name, ok := strings.CutPrefix(imp.spec, csMention); ok {
		var out []archdoc.Import
		for _, f := range r.types[name] {
			if f != file {
				out = append(out, archdoc.Import{Spec: name, Target: f, How: archdoc.ByName})
			}
		}
		return out
	}
	for _, ns := range r.own {
		if ns == imp.spec || strings.HasPrefix(ns, imp.spec+".") {
			return nil // its own code, read through the types the file names
		}
	}
	return []archdoc.Import{{Spec: imp.spec, How: archdoc.ByPackage, Package: imp.spec}}
}
