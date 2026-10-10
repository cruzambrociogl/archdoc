package code

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Java is read the way TypeScript is: its imports; its classes with their annotations — which is how
// Spring says what a class is, what a method answers and what a field holds — their fields, what
// the constructor is given, and the calls each method makes through a field; and the URLs its
// literals name. An annotation is recorded as a decorator: @GetMapping("/{id}") reads like NestJS's
// @Get(':id'), @Entity like TypeORM's.

// javaHTTP are the methods of Spring's HTTP clients — RestTemplate, WebClient, RestClient — that take
// the address they call.
var javaHTTP = map[string]bool{"getForObject": true, "getForEntity": true, "postForObject": true, "postForEntity": true,
	"postForLocation": true, "exchange": true, "patchForObject": true, "put": true, "delete": true, "uri": true, "baseUrl": true}

// javaClient is a receiver that is an HTTP client by its name: restTemplate, webClient, this.client.
var javaClient = regexp.MustCompile(`(?i)(template|webclient|restclient|httpclient|client)$`)

// JPA's relation annotations: the field holds another entity, or a collection of them.
var javaRelations = map[string]bool{"ManyToOne": true, "OneToOne": true, "OneToMany": true, "ManyToMany": true, "ElementCollection": true}

func javaFacts(src []byte, file string) (out facts, partial bool) {
	at := func(n *ts.Node) archdoc.Provenance {
		p := n.StartPoint()
		return archdoc.Provenance{File: file, Line: int(p.Row) + 1, Column: int(p.Column) + 1}
	}
	called := map[archdoc.Provenance]bool{}
	partial = parse("Java", src, func(root *ts.Node, l *ts.Language) {
		text := func(n *ts.Node) string { return n.Text(src) }
		walk(root, func(n *ts.Node) {
			switch n.Type(l) {
			case "import_declaration":
				var spec string
				static, wildcard := false, false
				for i := 0; i < n.ChildCount(); i++ {
					c := n.Child(i)
					switch c.Type(l) {
					case "scoped_identifier", "identifier":
						spec = text(c)
					case "asterisk":
						wildcard = true
					case "static":
						static = true
					}
				}
				if spec == "" {
					return
				}
				if static {
					spec = spec[:max(strings.LastIndex(spec, "."), 0)] // a member of a class: the class is what is imported
				}
				if wildcard {
					spec += ".*"
				}
				p := at(n)
				out.imports = append(out.imports, rawImport{spec: spec, line: p.Line, column: p.Column})
			case "class_declaration", "interface_declaration", "record_declaration", "enum_declaration":
				out.classes = append(out.classes, javaClass(n, l, src, at))
			case "method_invocation":
				name, args, obj := n.ChildByFieldName("name", l), n.ChildByFieldName("arguments", l), n.ChildByFieldName("object", l)
				if name == nil || args == nil || args.NamedChildCount() == 0 || !javaHTTP[text(name)] {
					return
				}
				first := args.NamedChild(0)
				if lit := leftmost(first, l); lit != nil {
					if v, ok := javaString(lit, l, src); ok && urlLiteral.MatchString(v) {
						called[at(lit)] = true
						return
					}
				}
				if obj != nil && javaClient.MatchString(text(obj)) {
					out.calls = append(out.calls, archdoc.Call{Callee: shorten(text(obj)) + "." + text(name), Target: shorten(text(first)), Prov: at(n)})
				}
			case "string_literal":
				v, ok := javaString(n, l, src)
				if !ok {
					return
				}
				key := ""
				if strings.HasPrefix(v, "${") {
					// @Value("${vets.url:http://vets-service:8080}"): the default is the address, the
					// property names what it is.
					name, def, found := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(v, "${"), "}"), ":")
					if !found {
						return
					}
					v, key = def, name[strings.LastIndex(name, ".")+1:]
				}
				h, ok := hostOf(v)
				if !ok {
					return
				}
				h.Prov = at(n)
				h.Key = key
				if h.Key == "" {
					h.Key = javaKey(n, l, src)
				}
				// "http://customers-service/owners/" + id: the address is made, not configured — but
				// what it is made from still names the host.
				if p := n.Parent(); p != nil && p.Type(l) == "binary_expression" {
					h.Built = true
				}
				out.hosts = append(out.hosts, h)
			}
		})
	})
	for i := range out.hosts {
		out.hosts[i].Called = called[out.hosts[i].Prov]
	}
	return out, partial
}

// leftmost is the string a concatenation starts with, or the node itself.
func leftmost(n *ts.Node, l *ts.Language) *ts.Node {
	for n != nil && n.Type(l) == "binary_expression" && n.NamedChildCount() > 0 {
		n = n.NamedChild(0)
	}
	if n != nil && n.Type(l) == "string_literal" {
		return n
	}
	return nil
}

// javaString is a string literal's value; a text block is not read.
func javaString(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	if n == nil || n.Type(l) != "string_literal" {
		return "", false
	}
	t := n.Text(src)
	if strings.HasPrefix(t, `"""`) || len(t) < 2 {
		return "", false
	}
	return t[1 : len(t)-1], true
}

// javaKey is what a literal is the value of: a field or variable, a property set by assignment, an
// annotation's element, or the builder method it is handed to — baseUrl("…").
func javaKey(n *ts.Node, l *ts.Language, src []byte) string {
	for p, hops := n.Parent(), 0; p != nil && hops < 4; p, hops = p.Parent(), hops+1 {
		switch p.Type(l) {
		case "binary_expression", "parenthesized_expression", "argument_list":
			continue
		case "variable_declarator":
			if name := p.ChildByFieldName("name", l); name != nil {
				return name.Text(src)
			}
		case "element_value_pair":
			if key := p.ChildByFieldName("key", l); key != nil {
				return key.Text(src)
			}
		case "assignment_expression":
			if left := p.ChildByFieldName("left", l); left != nil {
				t := left.Text(src)
				return t[strings.LastIndex(t, ".")+1:]
			}
		case "method_invocation":
			if name := p.ChildByFieldName("name", l); name != nil {
				return name.Text(src)
			}
		}
		return ""
	}
	return ""
}

func javaClass(n *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) archdoc.Class {
	text := func(n *ts.Node) string { return n.Text(src) }
	c := archdoc.Class{Prov: at(n), Interface: n.Type(l) == "interface_declaration"}
	if name := n.ChildByFieldName("name", l); name != nil {
		c.Name = text(name)
	}
	for i := 0; i < n.NamedChildCount(); i++ {
		ch := n.NamedChild(i)
		switch ch.Type(l) {
		case "modifiers":
			c.Decorators = javaAnnotations(ch, l, src, at)
		case "superclass", "super_interfaces", "extends_interfaces":
			// extends Person; implements A, B<C> — a type list holds the second kind.
			for j := 0; j < ch.NamedChildCount(); j++ {
				types := []*ts.Node{ch.NamedChild(j)}
				if types[0].Type(l) == "type_list" {
					types = types[:0]
					for k := 0; k < ch.NamedChild(j).NamedChildCount(); k++ {
						types = append(types, ch.NamedChild(j).NamedChild(k))
					}
				}
				for _, t := range types {
					name := javaTypeName(t, l, src)
					c.Extends = append(c.Extends, name)
					// JpaRepository<Owner, Integer>: a Spring Data repository of Owner's table.
					if strings.HasSuffix(name, "Repository") && t.Type(l) == "generic_type" {
						if el := javaFirstArg(t, l, src); el != "" {
							if c.Options == nil {
								c.Options = map[string]string{}
							}
							c.Options["entity"] = el
						}
					}
				}
			}
		case "formal_parameters": // a record's components are its fields
			for _, p := range javaParams(ch, l, src, at) {
				c.Params = append(c.Params, p)
				c.Fields = append(c.Fields, archdoc.Field{Name: p.Name, Type: p.Type, Prov: p.Prov})
			}
		}
	}
	body := n.ChildByFieldName("body", l)
	if body == nil {
		return c
	}
	members := []*ts.Node{}
	for i := 0; i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		if m.Type(l) == "enum_body_declarations" {
			for j := 0; j < m.NamedChildCount(); j++ {
				members = append(members, m.NamedChild(j))
			}
			continue
		}
		members = append(members, m)
	}
	for _, m := range members {
		switch m.Type(l) {
		case "field_declaration":
			var decs []archdoc.Decorator
			static := false
			for i := 0; i < m.NamedChildCount(); i++ {
				if mods := m.NamedChild(i); mods.Type(l) == "modifiers" {
					decs = javaAnnotations(mods, l, src, at)
					static = hasWord(text(mods), "static")
				}
			}
			typ := m.ChildByFieldName("type", l)
			if typ == nil || static {
				continue
			}
			for i := 0; i < m.NamedChildCount(); i++ {
				d := m.NamedChild(i)
				if d.Type(l) != "variable_declarator" {
					continue
				}
				name := d.ChildByFieldName("name", l)
				if name == nil {
					continue
				}
				f := archdoc.Field{Name: text(name), Type: text(typ), Prov: at(d)}
				for _, dec := range decs {
					if javaRelations[dec.Name] {
						dec.Target = javaElement(typ, l, src)
					}
					f.Decorators = append(f.Decorators, dec)
				}
				if v, ok := javaString(d.ChildByFieldName("value", l), l, src); ok {
					f.Value = v
				}
				c.Fields = append(c.Fields, f)
				c.Params = append(c.Params, archdoc.Param{Name: f.Name, Type: javaTypeName(typ, l, src), Prov: f.Prov})
			}
		case "constructor_declaration":
			if ps := m.ChildByFieldName("parameters", l); ps != nil {
				for _, p := range javaParams(ps, l, src, at) {
					c.Params = append(c.Params, p)
					c.Injects = append(c.Injects, archdoc.Literal{Value: p.Type, Prov: p.Prov})
				}
			}
		case "method_declaration":
			meth := archdoc.Method{Prov: at(m), EndLine: int(m.EndPoint().Row) + 1}
			if name := m.ChildByFieldName("name", l); name != nil {
				meth.Name = text(name)
			}
			for i := 0; i < m.NamedChildCount(); i++ {
				if mods := m.NamedChild(i); mods.Type(l) == "modifiers" {
					meth.Decorators = javaAnnotations(mods, l, src, at)
				}
			}
			if b := m.ChildByFieldName("body", l); b != nil {
				walk(b, func(x *ts.Node) {
					if x.Type(l) != "method_invocation" {
						return
					}
					name := x.ChildByFieldName("name", l)
					if name == nil {
						return
					}
					inv := archdoc.Invocation{Method: text(name), Prov: at(x)}
					switch obj := x.ChildByFieldName("object", l); {
					case obj == nil:
					case obj.Type(l) == "identifier":
						inv.Object = text(obj)
					case obj.Type(l) == "field_access" && obj.NamedChildCount() == 2 && obj.NamedChild(0).Type(l) == "this":
						inv.Object = text(obj.NamedChild(1))
					default:
						return // a call on what another call returned: its type is not written down
					}
					meth.Invokes = append(meth.Invokes, inv)
				})
			}
			c.Methods = append(c.Methods, meth)
		}
	}
	return c
}

func hasWord(s, w string) bool {
	for _, f := range strings.Fields(s) {
		if f == w {
			return true
		}
	}
	return false
}

func javaParams(ps *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Param {
	var out []archdoc.Param
	for i := 0; i < ps.NamedChildCount(); i++ {
		p := ps.NamedChild(i)
		if p.Type(l) != "formal_parameter" {
			continue
		}
		typ, name := p.ChildByFieldName("type", l), p.ChildByFieldName("name", l)
		if typ != nil && name != nil {
			out = append(out, archdoc.Param{Name: name.Text(src), Type: javaTypeName(typ, l, src), Prov: at(p)})
		}
	}
	return out
}

// javaTypeName is a type without its arguments or its enclosing class: List<Pet> is List,
// WebClient.Builder is Builder.
func javaTypeName(n *ts.Node, l *ts.Language, src []byte) string {
	switch n.Type(l) {
	case "generic_type":
		if n.NamedChildCount() > 0 {
			return javaTypeName(n.NamedChild(0), l, src)
		}
	case "scoped_type_identifier":
		if n.NamedChildCount() > 0 {
			return javaTypeName(n.NamedChild(n.NamedChildCount()-1), l, src)
		}
	}
	return n.Text(src)
}

// javaFirstArg is a generic type's first type argument: JpaRepository<Owner, Integer> is Owner.
func javaFirstArg(n *ts.Node, l *ts.Language, src []byte) string {
	for i := 0; i < n.NamedChildCount(); i++ {
		if args := n.NamedChild(i); args.Type(l) == "type_arguments" && args.NamedChildCount() > 0 {
			return javaTypeName(args.NamedChild(0), l, src)
		}
	}
	return ""
}

// javaElement is what a relation field holds: List<Pet> holds Pet, Owner holds Owner.
func javaElement(n *ts.Node, l *ts.Language, src []byte) string {
	if n.Type(l) == "generic_type" {
		for i := 0; i < n.NamedChildCount(); i++ {
			if args := n.NamedChild(i); args.Type(l) == "type_arguments" && args.NamedChildCount() > 0 {
				return javaTypeName(args.NamedChild(args.NamedChildCount()-1), l, src)
			}
		}
	}
	return javaTypeName(n, l, src)
}

// javaAnnotations are the annotations a modifiers node holds, as decorators.
func javaAnnotations(mods *ts.Node, l *ts.Language, src []byte, at func(*ts.Node) archdoc.Provenance) []archdoc.Decorator {
	var out []archdoc.Decorator
	for i := 0; i < mods.NamedChildCount(); i++ {
		a := mods.NamedChild(i)
		if a.Type(l) != "annotation" && a.Type(l) != "marker_annotation" {
			continue
		}
		d := archdoc.Decorator{Prov: at(a)}
		if name := a.ChildByFieldName("name", l); name != nil {
			t := name.Text(src)
			d.Name = t[strings.LastIndex(t, ".")+1:]
		}
		args := a.ChildByFieldName("arguments", l)
		if args == nil {
			out = append(out, d)
			continue
		}
		for j := 0; j < args.NamedChildCount(); j++ {
			x := args.NamedChild(j)
			if x.Type(l) == "element_value_pair" {
				key, val := x.ChildByFieldName("key", l), x.ChildByFieldName("value", l)
				if key == nil || val == nil {
					continue
				}
				k := key.Text(src)
				if v, ok := javaValue(val, l, src); ok {
					if d.Options == nil {
						d.Options = map[string]string{}
					}
					d.Options[k] = v
					if (k == "value" || k == "path") && !d.HasArg {
						d.Arg, d.HasArg = v, true
					}
				} else {
					if d.Exprs == nil {
						d.Exprs = map[string]string{}
					}
					d.Exprs[k] = val.Text(src)
				}
				continue
			}
			if v, ok := javaValue(x, l, src); ok && !d.HasArg {
				d.Arg, d.HasArg = v, true
			} else if !ok {
				d.ArgExpr = x.Text(src)
			}
		}
		out = append(out, d)
	}
	return out
}

// javaValue is an annotation element's literal value: a string, the first string of an array, or
// a number or boolean as written.
func javaValue(n *ts.Node, l *ts.Language, src []byte) (string, bool) {
	switch n.Type(l) {
	case "string_literal":
		return javaString(n, l, src)
	case "element_value_array_initializer":
		for i := 0; i < n.NamedChildCount(); i++ {
			if v, ok := javaString(n.NamedChild(i), l, src); ok {
				return v, true
			}
		}
		return "", n.NamedChildCount() == 0
	case "true", "false", "decimal_integer_literal":
		return n.Text(src), true
	}
	return "", false
}

// javaRoot is where a Java module's packages start: src/main/java by Maven's convention, else src/.
func javaRoot(repo, dir string) string {
	for _, r := range []string{path.Join(dir, "src", "main", "java"), path.Join(dir, "src")} {
		if fi, err := os.Stat(filepath.Join(repo, filepath.FromSlash(r))); err == nil && fi.IsDir() {
			return r
		}
	}
	return dir
}

// javaBase is the module's own base package as a directory: below the source root, the folders that
// hold only one folder and no code — com/example/petclinic. Its parts are the folders under it.
func javaBase(repo, root string) string {
	for depth := 0; depth < 12; depth++ {
		entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(root)))
		if err != nil {
			return root
		}
		var dirs []string
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				dirs = append(dirs, e.Name())
			} else if strings.HasSuffix(e.Name(), ".java") {
				return root
			}
		}
		if len(dirs) != 1 {
			return root
		}
		root = path.Join(root, dirs[0])
	}
	return root
}

// javaResolver ties an import to a file of the module by Java's own rule: a package is a folder
// under the source root, a class a file in it. Anything else is a package from outside — the
// standard library or a dependency — unless it starts with the module's own package, which is then
// missing.
type javaResolver struct {
	root  string // src/main/java
	own   string // the module's base package, dotted
	known map[string]bool
	dirs  map[string][]string
}

func newJavaResolver(root, base string, known map[string]bool) *javaResolver {
	r := &javaResolver{root: root, known: known, dirs: map[string][]string{}}
	if rel, ok := strings.CutPrefix(base, root+"/"); ok {
		r.own = strings.ReplaceAll(rel, "/", ".")
	}
	for f := range known {
		r.dirs[path.Dir(f)] = append(r.dirs[path.Dir(f)], f)
	}
	for d := range r.dirs {
		sort.Strings(r.dirs[d])
	}
	return r
}

func (r *javaResolver) resolve(file string, imp rawImport) []archdoc.Import {
	spec, wildcard := strings.CutSuffix(imp.spec, ".*")
	parts := strings.Split(spec, ".")
	if wildcard {
		if files := r.dirs[path.Join(r.root, strings.Join(parts, "/"))]; len(files) > 0 {
			out := make([]archdoc.Import, 0, len(files))
			for _, f := range files {
				out = append(out, archdoc.Import{Spec: imp.spec, Target: f, How: archdoc.ByModule})
			}
			return out
		}
	}
	// com.acme.Outer.Inner is in com/acme/Outer.java: drop names from the end until a file is found.
	for k := len(parts); k > 0; k-- {
		if f := path.Join(r.root, strings.Join(parts[:k], "/")) + ".java"; r.known[f] {
			return []archdoc.Import{{Spec: imp.spec, Target: f, How: archdoc.ByModule}}
		}
	}
	if r.own != "" && (spec == r.own || strings.HasPrefix(spec, r.own+".")) {
		return []archdoc.Import{{Spec: imp.spec, How: archdoc.NoMatch}}
	}
	// The package is the import's lowercase head: org.springframework.web.bind.annotation.GetMapping
	// is in org.springframework.web.bind.annotation.
	pkg := parts[:0:0]
	for _, p := range parts {
		if p == "" || p[0] >= 'A' && p[0] <= 'Z' {
			break
		}
		pkg = append(pkg, p)
	}
	if len(pkg) == 0 {
		pkg = parts
	}
	return []archdoc.Import{{Spec: imp.spec, How: archdoc.ByPackage, Package: strings.Join(pkg, ".")}}
}
