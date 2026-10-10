package model

import (
	"path"
	"regexp"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Java and C#: what their frameworks declare, read from the annotations and attributes the readers
// recorded as decorators.
//
//   - Routes. Spring: a @RestController or @Controller class, its @RequestMapping as the prefix, a
//     method's @GetMapping("/{id}") as the route. ASP.NET: an [ApiController] or a Controller, its
//     [Route("api/[controller]")] as the prefix, a method's [HttpGet("{id}")] as the route. Minimal
//     APIs — app.MapGet("/items", …) — are routers, read with FastAPI's and Express's.
//   - Tables. JPA: an @Entity's fields, its superclasses' first. EF Core: every class a DbContext
//     holds a DbSet of, named after the property, with its properties as columns.
//   - Spring Data: a call to a repository interface — JpaRepository<Owner, Integer> — queries
//     Owner's table, and its method's name says how: findById reads, save writes.

// Spring's route annotations, by the method each answers.
var springVerbs = map[string]string{"GetMapping": "GET", "PostMapping": "POST", "PutMapping": "PUT",
	"DeleteMapping": "DELETE", "PatchMapping": "PATCH"}

var requestMethod = regexp.MustCompile(`RequestMethod\.([A-Z]+)`)

// ASP.NET's route attributes, by the method each answers.
var aspnetVerbs = map[string]string{"HttpGet": "GET", "HttpPost": "POST", "HttpPut": "PUT", "HttpDelete": "DELETE",
	"HttpPatch": "PATCH", "HttpHead": "HEAD", "HttpOptions": "OPTIONS"}

func has(c archdoc.Class, names ...string) (archdoc.Decorator, bool) {
	for _, d := range c.Decorators {
		for _, n := range names {
			if d.Name == n {
				return d, true
			}
		}
	}
	return archdoc.Decorator{}, false
}

// mappingPath is the path a Spring mapping annotation states: its value, or path.
func mappingPath(d archdoc.Decorator) string {
	if d.HasArg {
		return d.Arg
	}
	if v := d.Options["path"]; v != "" {
		return v
	}
	return d.Options["value"]
}

// controllerRoutes reads the routes of Spring and ASP.NET controllers.
func controllerRoutes(src archdoc.Source, container string, componentOf map[string]string) []archdoc.Entry {
	declaredAt := map[string]archdoc.Provenance{}
	twice := map[string]bool{}
	for _, f := range src.Files {
		for _, c := range f.Classes {
			if c.Name == "" {
				continue
			}
			if _, seen := declaredAt[c.Name]; seen {
				twice[c.Name] = true
			}
			declaredAt[c.Name] = c.Prov
		}
	}
	_, local, _ := strings.Cut(container, ":")
	var out []archdoc.Entry
	seen := map[string]bool{}
	for _, f := range src.Files {
		java, csharp := f.Language == "Java", f.Language == "C#"
		if !java && !csharp {
			continue
		}
		for _, c := range f.Classes {
			base, ok := "", false
			if java {
				if _, ok = has(c, "RestController", "Controller"); ok {
					if d, mapped := has(c, "RequestMapping"); mapped {
						base = mappingPath(d)
					}
				}
			} else {
				_, api := has(c, "ApiController")
				ok = api || containsAny(c.Extends, "Controller", "ControllerBase") || strings.HasSuffix(c.Name, "Controller")
				if d, routed := has(c, "Route"); routed {
					base = d.Arg
				}
				base = strings.ReplaceAll(base, "[controller]", strings.TrimSuffix(c.Name, "Controller"))
			}
			if !ok {
				continue
			}
			var uses []archdoc.Symbol
			for _, in := range c.Injects {
				s := archdoc.Symbol{Name: in.Value, How: "unresolved", Prov: in.Prov}
				if at, ok := declaredAt[in.Value]; ok && !twice[in.Value] {
					s.How, s.Prov, s.Component = "name", at, componentOf[at.File]
				}
				uses = append(uses, s)
			}
			// A controller that renders views — Spring's @Controller without @ResponseBody, ASP.NET's
			// Controller rather than ControllerBase — answers a GET with a page.
			views := false
			if java {
				_, rest := has(c, "RestController", "ResponseBody")
				_, mvc := has(c, "Controller")
				views = mvc && !rest
			} else {
				_, api := has(c, "ApiController")
				views = !api && containsAny(c.Extends, "Controller")
			}
			for _, meth := range c.Methods {
				_, body := decoratedMethod(meth, "ResponseBody")
				for _, r := range methodRoutes(meth, java) {
					p := path.Join("/", base, r.path)
					if r.absolute {
						p = path.Join("/", r.path)
					}
					if csharp {
						p = strings.ReplaceAll(p, "[action]", strings.TrimSuffix(meth.Name, "Async"))
					}
					e := archdoc.Entry{Kind: "http", Method: r.verb, Path: p, Handler: c.Name + "." + meth.Name, Container: container,
						Component: componentOf[f.Path], Uses: uses, Prov: r.prov}
					e.Summary, e.SummaryProv = summaryOf(meth)
					e.ID = "route:" + local + " " + e.Method + " " + e.Path
					if views && !body && r.verb == "GET" {
						e.Kind, e.Method = "page", "PAGE"
						e.ID = "page:" + local + " " + e.Path
					}
					if seen[e.ID] {
						e.ID += " " + e.Handler
					}
					seen[e.ID] = true
					out = append(out, e)
				}
			}
		}
	}
	return out
}

type methodRoute struct {
	verb, path string
	absolute   bool
	prov       archdoc.Provenance
}

// methodRoutes are the routes one controller method declares.
func methodRoutes(m archdoc.Method, java bool) []methodRoute {
	var out []methodRoute
	if java {
		for _, d := range m.Decorators {
			if verb, ok := springVerbs[d.Name]; ok {
				out = append(out, methodRoute{verb: verb, path: mappingPath(d), prov: d.Prov})
			}
			if d.Name == "RequestMapping" {
				// method = RequestMethod.GET, or method = {RequestMethod.GET, RequestMethod.POST}: a
				// route for each; none named answers every method.
				verbs := requestMethod.FindAllStringSubmatch(d.Exprs["method"], -1)
				if len(verbs) == 0 {
					out = append(out, methodRoute{verb: "ANY", path: mappingPath(d), prov: d.Prov})
				}
				for _, v := range verbs {
					out = append(out, methodRoute{verb: v[1], path: mappingPath(d), prov: d.Prov})
				}
			}
		}
		return out
	}
	route := ""
	for _, d := range m.Decorators {
		if d.Name == "Route" && d.HasArg {
			route = d.Arg
		}
	}
	for _, d := range m.Decorators {
		verb, ok := aspnetVerbs[d.Name]
		if !ok {
			continue
		}
		p := route
		if d.HasArg {
			p = d.Arg
		}
		abs := strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~/")
		out = append(out, methodRoute{verb: verb, path: strings.TrimPrefix(p, "~"), absolute: abs, prov: d.Prov})
	}
	return out
}

// summaryOf is the description a route's own annotation gives it: OpenAPI's @Operation(summary = …),
// Swashbuckle's [SwaggerOperation(Summary = …)], or [EndpointSummary("…")].
func summaryOf(m archdoc.Method) (string, archdoc.Provenance) {
	for _, d := range m.Decorators {
		switch {
		case d.Name == "Operation" && d.Options["summary"] != "":
			return d.Options["summary"], d.Prov
		case d.Name == "SwaggerOperation" && d.Options["Summary"] != "":
			return d.Options["Summary"], d.Prov
		case d.Name == "SwaggerOperation" && d.HasArg:
			return d.Arg, d.Prov
		case d.Name == "EndpointSummary" && d.HasArg:
			return d.Arg, d.Prov
		}
	}
	return "", archdoc.Provenance{}
}

func decoratedMethod(m archdoc.Method, names ...string) (archdoc.Decorator, bool) {
	for _, d := range m.Decorators {
		for _, n := range names {
			if d.Name == n {
				return d, true
			}
		}
	}
	return archdoc.Decorator{}, false
}

func containsAny(list []string, names ...string) bool {
	for _, x := range list {
		for _, n := range names {
			if x == n {
				return true
			}
		}
	}
	return false
}

// inherited is a class's fields with its superclasses' first; a redeclared field replaces the one it
// hides.
func inherited(c archdoc.Class, classes map[string]archdoc.Class) []archdoc.Field {
	var fields []archdoc.Field
	at := map[string]int{}
	var collect func(c archdoc.Class, depth int)
	collect = func(c archdoc.Class, depth int) {
		if depth > 8 {
			return
		}
		for _, b := range c.Extends {
			if base, ok := classes[b]; ok && base.Name != c.Name && !base.Interface {
				collect(base, depth+1)
			}
		}
		for _, f := range c.Fields {
			if i, ok := at[f.Name]; ok {
				fields[i] = f
				continue
			}
			at[f.Name] = len(fields)
			fields = append(fields, f)
		}
	}
	collect(c, 0)
	return fields
}

func decorated(f archdoc.Field, names ...string) (archdoc.Decorator, bool) {
	for _, d := range f.Decorators {
		for _, n := range names {
			if d.Name == n {
				return d, true
			}
		}
	}
	return archdoc.Decorator{}, false
}

// Java's primitive types: a column of one is never null.
var javaPrimitives = map[string]bool{"int": true, "long": true, "boolean": true, "double": true, "float": true,
	"short": true, "byte": true, "char": true}

// javaColumns are a JPA entity's columns: every field but a transient one and the inverse side of a
// relation. A relation it owns — @ManyToOne, @OneToOne without mappedBy — is a foreign key column,
// named by its @JoinColumn or after the field.
func javaColumns(c archdoc.Class, classes map[string]archdoc.Class, byClass map[string]string) []archdoc.Column {
	var out []archdoc.Column
	for _, f := range inherited(c, classes) {
		if _, skip := decorated(f, "Transient", "OneToMany", "ManyToMany", "ElementCollection"); skip {
			continue
		}
		col := archdoc.Column{Name: f.Name, Type: f.Type, Prov: f.Prov}
		if d, ok := decorated(f, "ManyToOne", "OneToOne"); ok {
			if d.Options["mappedBy"] != "" {
				continue // the other side owns the key
			}
			col.Name, col.References = f.Name+"_id", byClass[d.Target]
			if j, ok := decorated(f, "JoinColumn"); ok && j.Options["name"] != "" {
				col.Name = j.Options["name"]
			}
		}
		if d, ok := decorated(f, "Column"); ok && d.Options["name"] != "" {
			col.Name = d.Options["name"]
		}
		_, id := decorated(f, "Id", "EmbeddedId")
		col.Primary = id
		_, required := decorated(f, "NotNull", "NotBlank", "NotEmpty")
		d, _ := decorated(f, "Column", "JoinColumn")
		col.Nullable = !id && !required && !javaPrimitives[f.Type] && d.Options["nullable"] != "false"
		out = append(out, col)
	}
	return out
}

// csCollections are the types a navigation to many holds its entities in.
var csCollections = map[string]bool{"List": true, "IList": true, "ICollection": true, "IEnumerable": true,
	"IReadOnlyCollection": true, "IReadOnlyList": true, "HashSet": true, "ISet": true, "Collection": true}

// csColumns are an EF Core entity's columns: its properties and its base classes', by EF's
// conventions — Id or <Class>Id is the key, a property of another entity's type is a navigation
// whose key is the property named after it with Id, a collection is the other side of a key.
func csColumns(c archdoc.Class, classes map[string]archdoc.Class, byClass map[string]string) []archdoc.Column {
	fields := inherited(c, classes)
	navs := map[string]string{} // navigation name → the table it leads to
	for _, f := range fields {
		if to, ok := byClass[strings.TrimSuffix(f.Type, "?")]; ok {
			navs[f.Name] = to
		}
	}
	var out []archdoc.Column
	for _, f := range fields {
		typ := strings.TrimSuffix(f.Type, "?")
		generic, _, _ := strings.Cut(typ, "<")
		if strings.HasPrefix(f.Name, "_") || csCollections[generic] || navs[f.Name] != "" {
			continue
		}
		if _, skip := decorated(f, "NotMapped"); skip {
			continue
		}
		col := archdoc.Column{Name: f.Name, Type: typ, Prov: f.Prov, Nullable: strings.HasSuffix(f.Type, "?")}
		if d, ok := decorated(f, "Column"); ok && d.HasArg {
			col.Name = d.Arg
		}
		_, key := decorated(f, "Key")
		col.Primary = key || f.Name == "Id" || f.Name == c.Name+"Id"
		if stem, ok := strings.CutSuffix(f.Name, "Id"); ok && stem != "" && !col.Primary {
			if to := navs[stem]; to != "" {
				col.References = to
			} else if to, ok := byClass[stem]; ok {
				col.References = to
			}
		}
		out = append(out, col)
	}
	return out
}

// efTables are the classes a DbContext holds a DbSet of, each with the property that holds it: EF
// Core names the table after that property.
func efTables(files []archdoc.SourceFile) map[string]archdoc.Field {
	out := map[string]archdoc.Field{}
	for _, f := range files {
		if f.Language != "C#" {
			continue
		}
		for _, c := range f.Classes {
			context := false
			for _, e := range c.Extends {
				context = context || strings.HasSuffix(e, "DbContext")
			}
			if !context {
				continue
			}
			for _, fl := range c.Fields {
				if inner, ok := strings.CutPrefix(fl.Type, "DbSet<"); ok {
					entity := strings.TrimSuffix(inner, ">")
					entity = entity[strings.LastIndex(entity, ".")+1:]
					if _, seen := out[entity]; !seen {
						out[entity] = fl
					}
				}
			}
		}
	}
	return out
}

// springOps is what a Spring Data method does, by the word its name starts with.
var springOps = []struct{ prefix, op string }{
	{"find", "reads"}, {"get", "reads"}, {"read", "reads"}, {"query", "reads"}, {"search", "reads"},
	{"stream", "reads"}, {"count", "reads"}, {"exists", "reads"},
	{"save", "writes"}, {"insert", "writes"}, {"persist", "writes"},
	{"update", "updates"},
	{"delete", "deletes"}, {"remove", "deletes"},
}

// springData records, on every method that calls a Spring Data repository through a field, the table
// that call queries: owners.findById(id) reads owners. Done before flows and access are read, so both
// see it like any other query.
func springData(sources []archdoc.Source) {
	for _, src := range sources {
		classes := map[string]archdoc.Class{}
		for _, f := range src.Files {
			if f.Language != "Java" {
				continue
			}
			for _, c := range f.Classes {
				if c.Name != "" {
					classes[c.Name] = c
				}
			}
		}
		if len(classes) == 0 {
			continue
		}
		tableOf := func(entity string) string {
			if c, ok := classes[entity]; ok {
				if name, ok := tableName(c, "Java"); ok {
					return name
				}
			}
			return ""
		}
		for fi := range src.Files {
			f := &src.Files[fi]
			for ci := range f.Classes {
				c := &f.Classes[ci]
				types := map[string]string{}
				for _, p := range c.Params {
					types[p.Name] = p.Type
				}
				for mi := range c.Methods {
					m := &c.Methods[mi]
					for _, inv := range m.Invokes {
						repo, ok := classes[types[inv.Object]]
						if inv.Object == "" || !ok || repo.Options["entity"] == "" {
							continue
						}
						table := tableOf(repo.Options["entity"])
						if table == "" {
							continue
						}
						for _, o := range springOps {
							if strings.HasPrefix(inv.Method, o.prefix) {
								m.Queries = append(m.Queries, archdoc.Query{Table: table, Op: o.op, Prov: inv.Prov})
								break
							}
						}
					}
				}
			}
		}
	}
}

// implementations maps an interface to the one class that implements it, where there is exactly one:
// what a C# service is given is usually its interface, and the work is in the class.
func implementations(classes map[string]classAt) map[string]string {
	impl := map[string][]string{}
	for name, at := range classes {
		if at.class.Interface {
			continue
		}
		for _, e := range at.class.Extends {
			if i, ok := classes[e]; ok && i.class.Interface {
				impl[e] = append(impl[e], name)
			}
		}
	}
	out := map[string]string{}
	for i, cs := range impl {
		if len(cs) == 1 {
			out[i] = cs[0]
		}
	}
	return out
}
