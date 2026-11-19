package code

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// What each Go function does, for its flow (internal/model/flows_go.go).
//
// Go states its types, so a call is tied the way the compiler ties it, as far as the code says:
// s.store.Finish(ctx, id) is Store.Finish because s is the method's receiver, a *Service, whose
// field store is a *store.Store. A local takes the type of what it is given — &orders.Service{},
// store.New(…)'s first result — and an interface the one type of the program that has its
// methods. What the code does not state — a value from a map, a function passed in — is left.
//
// Beside the calls: the tables its SQL names — a string of SQL in the function, or a constant
// holding one, as sqlc generates them — and the jobs it queues, a River job's arguments made in
// place. A function written inside another is its own Func, for a route it handles, and what it
// does counts for the function it is written in as well.

// goType is a named type of the program: its package's directory and its name.
type goType struct{ dir, name string }

type typeAt struct {
	decl *goTypeDecl
	file *goFile
}

type exprAt struct {
	expr ast.Expr
	file *goFile
}

// goIndex is what the program's packages declare, for tying calls.
type goIndex struct {
	byDir     map[string][]*goFile
	pkgOf     map[string]string
	dirOf     func(string) (string, bool)
	types     map[goType]typeAt
	funcs     map[string]bool // every Func key declared
	results   map[string]exprAt
	sqlConsts map[string]string // "dir#name" → SQL
	methods   map[goType]map[string]bool
	kinds     map[goName]core.Literal
	impls     map[goType]goType
}

func newGoIndex(byDir map[string][]*goFile, reached map[string]bool, pkgOf map[string]string, dirOf func(string) (string, bool),
	kinds map[goName]core.Literal) *goIndex {
	idx := &goIndex{byDir: byDir, pkgOf: pkgOf, dirOf: dirOf, types: map[goType]typeAt{}, funcs: map[string]bool{},
		results: map[string]exprAt{}, sqlConsts: map[string]string{}, methods: map[goType]map[string]bool{}, kinds: kinds,
		impls: map[goType]goType{}}
	for dir := range reached {
		for _, f := range byDir[dir] {
			for name, td := range f.types {
				idx.types[goType{dir, name}] = typeAt{td, f}
			}
			for _, fn := range f.funcs {
				key := fn.name
				if fn.recv != "" {
					key = fn.recv + "." + fn.name
					t := goType{dir, fn.recv}
					if idx.methods[t] == nil {
						idx.methods[t] = map[string]bool{}
					}
					idx.methods[t][fn.name] = true
				}
				idx.funcs[dir+"#"+key] = true
			}
			for key, e := range f.results {
				idx.results[dir+"#"+key] = exprAt{e, f}
			}
			for name, sql := range f.sqlConsts {
				idx.sqlConsts[dir+"#"+name] = sql
			}
		}
	}
	return idx
}

// importDir is the directory of the module's package a file uses by name.
func (idx *goIndex) importDir(f *goFile, name string) (string, bool) {
	for _, imp := range f.imports {
		n := imp.alias
		if n == "" {
			n = guessName(imp.path)
			if d, ok := idx.dirOf(imp.path); ok && idx.pkgOf[d] != "" {
				n = idx.pkgOf[d]
			}
		}
		if n == name {
			return idx.dirOf(imp.path)
		}
	}
	return "", false
}

// runnerKey resolves what runs a command, as goCommand wrote it, to a Func key.
func (idx *goIndex) runnerKey(g *goFile, raw string) string {
	switch {
	case strings.HasPrefix(raw, "ref:"):
		name := strings.TrimPrefix(raw, "ref:")
		dir := g.dir
		if pkg, fn, ok := strings.Cut(name, "."); ok {
			d, found := idx.importDir(g, pkg)
			if !found {
				return ""
			}
			dir, name = d, fn
		}
		if idx.funcs[dir+"#"+name] {
			return dir + "#" + name
		}
	case strings.HasPrefix(raw, "lit:"):
		encl, line, _ := strings.Cut(strings.TrimPrefix(raw, "lit:"), "@")
		return g.dir + "#" + encl + ".func@" + line
	}
	return ""
}

// predeclared are Go's own types: nothing of the program's.
var predeclared = map[string]bool{"string": true, "bool": true, "byte": true, "rune": true, "error": true, "any": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true,
	"uint32": true, "uint64": true, "uintptr": true, "float32": true, "float64": true, "complex64": true, "complex128": true}

// resolveType is the program's type a type expression in f names: *store.Store is store's Store.
func (idx *goIndex) resolveType(f *goFile, e ast.Expr) goType {
	switch x := e.(type) {
	case *ast.StarExpr:
		return idx.resolveType(f, x.X)
	case *ast.ParenExpr:
		return idx.resolveType(f, x.X)
	case *ast.IndexExpr:
		return idx.resolveType(f, x.X)
	case *ast.IndexListExpr:
		return idx.resolveType(f, x.X)
	case *ast.Ident:
		if predeclared[x.Name] {
			return goType{}
		}
		return goType{f.dir, x.Name}
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			if d, ok := idx.importDir(f, id.Name); ok {
				return goType{d, x.Sel.Name}
			}
		}
	}
	return goType{}
}

// fieldType is the type of a struct's field, through the types it embeds.
func (idx *goIndex) fieldType(t goType, field string, depth int) goType {
	at, ok := idx.types[t]
	if !ok || depth > 3 {
		return goType{}
	}
	if e, ok := at.decl.fields[field]; ok {
		return idx.resolveType(at.file, e)
	}
	for _, e := range at.decl.embedded {
		et := idx.resolveType(at.file, e)
		if et.name == field {
			return et
		}
		if ft := idx.fieldType(et, field, depth+1); ft.dir != "" {
			return ft
		}
	}
	return goType{}
}

// method is the Func key of a type's method: its own, an embedded type's, or — an interface — the
// one type of the program with its methods.
func (idx *goIndex) method(t goType, name string, depth int) (string, bool) {
	if t.dir == "" || depth > 3 {
		return "", false
	}
	if key := t.dir + "#" + t.name + "." + name; idx.funcs[key] {
		return key, true
	}
	at, ok := idx.types[t]
	if !ok {
		return "", false
	}
	if at.decl.isIface {
		if impl, ok := idx.implementation(t); ok {
			return idx.method(impl, name, depth+1)
		}
		return "", false
	}
	for _, e := range at.decl.embedded {
		if key, ok := idx.method(idx.resolveType(at.file, e), name, depth+1); ok {
			return key, true
		}
	}
	return "", false
}

// implementation is the one type of the program that has every method of an interface; none when
// there are several, or none.
func (idx *goIndex) implementation(iface goType) (goType, bool) {
	if impl, ok := idx.impls[iface]; ok {
		return impl, impl.dir != ""
	}
	want := idx.types[iface].decl.iface
	var found []goType
	if len(want) > 0 {
		for t, ms := range idx.methods {
			all := true
			for _, m := range want {
				all = all && ms[m]
			}
			if all {
				found = append(found, t)
			}
		}
	}
	var impl goType
	if len(found) == 1 {
		impl = found[0]
	}
	idx.impls[iface] = impl
	return impl, impl.dir != ""
}

// exprType is the program's type of a value, as far as the code states it.
func (idx *goIndex) exprType(f *goFile, env map[string]goType, e ast.Expr) goType {
	switch x := e.(type) {
	case *ast.Ident:
		return env[x.Name]
	case *ast.ParenExpr:
		return idx.exprType(f, env, x.X)
	case *ast.StarExpr:
		return idx.exprType(f, env, x.X)
	case *ast.UnaryExpr:
		return idx.exprType(f, env, x.X)
	case *ast.CompositeLit:
		return idx.resolveType(f, x.Type)
	case *ast.TypeAssertExpr:
		if x.Type != nil {
			return idx.resolveType(f, x.Type)
		}
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok && env[id.Name].dir == "" {
			return goType{} // a package's variable, or a value whose type is not stated
		}
		return idx.fieldType(idx.exprType(f, env, x.X), x.Sel.Name, 0)
	case *ast.CallExpr:
		if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "new" && len(x.Args) == 1 {
			return idx.resolveType(f, x.Args[0])
		}
		if to, ok := idx.callTarget(f, env, x); ok {
			if r, ok := idx.results[to]; ok {
				return idx.resolveType(r.file, r.expr)
			}
		}
	}
	return goType{}
}

// callTarget is the Func key a call reaches, when the code says which.
func (idx *goIndex) callTarget(f *goFile, env map[string]goType, call *ast.CallExpr) (string, bool) {
	fun := call.Fun
	if ix, ok := fun.(*ast.IndexExpr); ok {
		fun = ix.X // a generic function, instantiated: f[T](…)
	}
	switch x := fun.(type) {
	case *ast.Ident:
		if _, local := env[x.Name]; !local && idx.funcs[f.dir+"#"+x.Name] {
			return f.dir + "#" + x.Name, true
		}
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok && env[id.Name].dir == "" {
			if d, ok := idx.importDir(f, id.Name); ok {
				key := d + "#" + x.Sel.Name
				return key, idx.funcs[key]
			}
		}
		return idx.method(idx.exprType(f, env, x.X), x.Sel.Name, 0)
	}
	return "", false
}

// funcsOf is what each function and method of a file does.
func (idx *goIndex) funcsOf(g *goFile) []core.Func {
	if g.ast == nil {
		return nil
	}
	var out []core.Func
	for _, d := range g.ast.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		name := fd.Name.Name
		env := map[string]goType{}
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			recv := fd.Recv.List[0]
			rt := receiverType(recv.Type)
			name = rt + "." + name
			for _, n := range recv.Names {
				env[n.Name] = goType{g.dir, rt}
			}
		}
		idx.params(g, fd.Type, env)
		out = append(out, idx.walkFunc(g, fd.Body, env, name, fd.Pos(), fd.End())...)
	}
	return out
}

func (idx *goIndex) params(g *goFile, ft *ast.FuncType, env map[string]goType) {
	if ft == nil || ft.Params == nil {
		return
	}
	for _, p := range ft.Params.List {
		t := idx.resolveType(g, p.Type)
		for _, n := range p.Names {
			env[n.Name] = t
		}
	}
}

// walkFunc reads one function's body in order: what it calls, queries and queues, and the types
// its locals take. The first Func returned is this one; the rest are those written inside it.
func (idx *goIndex) walkFunc(g *goFile, body *ast.BlockStmt, env map[string]goType, name string, pos, end token.Pos) []core.Func {
	at := func(p token.Pos) core.Provenance {
		ps := g.fset.Position(p)
		return core.Provenance{File: g.path, Line: ps.Line, Column: ps.Column}
	}
	fn := core.Func{Key: g.dir + "#" + name, Prov: at(pos), EndLine: g.fset.Position(end).Line}
	var nested []core.Func
	sels := map[*ast.Ident]bool{} // a selector's name, which is not a name of this package
	query := func(sql string, p token.Pos) {
		for _, t := range sqlTables(sql) {
			fn.Queries = append(fn.Queries, core.Query{Table: t.table, Op: t.op, Prov: at(p)})
		}
	}
	assign := func(lhs ast.Expr, t goType) {
		if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" && t.dir != "" {
			env[id.Name] = t
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			inner := make(map[string]goType, len(env))
			for k, v := range env {
				inner[k] = v
			}
			idx.params(g, x.Type, inner)
			lits := idx.walkFunc(g, x.Body, inner, name+".func@"+strconv.Itoa(at(x.Pos()).Line), x.Pos(), x.End())
			fn.Calls = append(fn.Calls, lits[0].Calls...)
			fn.Queries = append(fn.Queries, lits[0].Queries...)
			fn.Queues = append(fn.Queues, lits[0].Queues...)
			nested = append(nested, lits...)
			return false
		case *ast.AssignStmt:
			switch {
			case len(x.Rhs) == 1:
				assign(x.Lhs[0], idx.exprType(g, env, x.Rhs[0]))
			case len(x.Lhs) == len(x.Rhs):
				for i := range x.Lhs {
					assign(x.Lhs[i], idx.exprType(g, env, x.Rhs[i]))
				}
			}
		case *ast.ValueSpec:
			for i, nm := range x.Names {
				switch {
				case x.Type != nil:
					assign(nm, idx.resolveType(g, x.Type))
				case i < len(x.Values):
					assign(nm, idx.exprType(g, env, x.Values[i]))
				}
			}
		case *ast.CallExpr:
			if to, ok := idx.callTarget(g, env, x); ok {
				fn.Calls = append(fn.Calls, core.FuncCall{To: to, Prov: at(x.Pos())})
			}
			for _, a := range x.Args {
				if u, ok := a.(*ast.UnaryExpr); ok {
					a = u.X
				}
				if cl, ok := a.(*ast.CompositeLit); ok {
					t := idx.resolveType(g, cl.Type)
					if k, ok := idx.kinds[goName{t.dir, t.name}]; ok {
						fn.Queues = append(fn.Queues, core.Literal{Value: k.Value, Prov: at(a.Pos())})
					}
				}
			}
		case *ast.SelectorExpr:
			sels[x.Sel] = true
			if id, ok := x.X.(*ast.Ident); ok && env[id.Name].dir == "" {
				if d, ok := idx.importDir(g, id.Name); ok {
					if sql, ok := idx.sqlConsts[d+"#"+x.Sel.Name]; ok {
						query(sql, x.Pos())
					}
				}
			}
		case *ast.Ident:
			if sels[x] {
				break
			}
			if _, local := env[x.Name]; !local {
				if sql, ok := idx.sqlConsts[g.dir+"#"+x.Name]; ok {
					query(sql, x.Pos())
				}
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				if v, err := strconv.Unquote(x.Value); err == nil && looksSQL(v) {
					query(v, x.Pos())
				}
			}
		}
		return true
	})
	return append([]core.Func{fn}, nested...)
}

var (
	sqlLineComment  = regexp.MustCompile(`--[^\n]*`)
	sqlBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	sqlFirstWord    = regexp.MustCompile(`^\s*([A-Za-z]+)\s`)
	sqlCTE          = regexp.MustCompile(`(?i)(?:\bWITH\s+(?:RECURSIVE\s+)?|,\s*)([A-Za-z_][A-Za-z0-9_]*)\s+AS\s*(?:NOT\s+)?(?:MATERIALIZED\s+)?\(`)
	sqlTableRef     = regexp.MustCompile(`(?i)\b(DISTINCT\s+FROM|DELETE\s+FROM|INSERT\s+INTO|DO\s+UPDATE|FROM|JOIN|UPDATE|INTO)\s+(?:ONLY\s+|LATERAL\s+)?("?[A-Za-z_][A-Za-z0-9_]*"?(?:\."?[A-Za-z_][A-Za-z0-9_]*"?)?)(\()?`)
)

// sqlKeywords never name a table where a table could be named.
var sqlKeywords = map[string]bool{"select": true, "set": true, "values": true, "where": true, "unnest": true, "lateral": true,
	"only": true, "generate_series": true, "jsonb_array_elements": true, "json_array_elements": true, "jsonb_each": true}

// looksSQL reports a string that is a SQL statement: a query or a change, whole.
func looksSQL(v string) bool {
	s := sqlLineComment.ReplaceAllString(v, "")
	m := sqlFirstWord.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	u := strings.ToUpper(s)
	switch strings.ToUpper(m[1]) {
	case "SELECT", "DELETE":
		return strings.Contains(u, "FROM")
	case "INSERT":
		return strings.Contains(u, "INTO")
	case "UPDATE":
		return strings.Contains(u, " SET ") || strings.Contains(u, "\nSET ") || strings.Contains(u, "\tSET ")
	case "WITH":
		return strings.Contains(u, "SELECT") || strings.Contains(u, "INSERT") || strings.Contains(u, "UPDATE") || strings.Contains(u, "DELETE")
	}
	return false
}

type sqlRef struct{ table, op string }

// sqlTables is what a statement does to which table: INSERT INTO writes, UPDATE updates, DELETE FROM
// deletes, FROM and JOIN read. A name a WITH gives a subquery, a function in FROM and a column in
// IS DISTINCT FROM are none.
func sqlTables(sql string) []sqlRef {
	s := sqlBlockComment.ReplaceAllString(sqlLineComment.ReplaceAllString(sql, ""), "")
	ctes := map[string]bool{}
	for _, m := range sqlCTE.FindAllStringSubmatch(s, -1) {
		ctes[strings.ToLower(m[1])] = true
	}
	seen := map[sqlRef]bool{}
	var out []sqlRef
	for _, m := range sqlTableRef.FindAllStringSubmatch(s, -1) {
		kw := strings.ToUpper(strings.Join(strings.Fields(m[1]), " "))
		name := strings.ReplaceAll(m[2], `"`, "")
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		low := strings.ToLower(name)
		// A name with ( right after it, in FROM or JOIN, is a function: unnest(…). After INTO it is
		// the table, and its columns.
		function := m[3] != "" && (kw == "FROM" || kw == "JOIN")
		if function || kw == "DISTINCT FROM" || kw == "DO UPDATE" || sqlKeywords[low] || ctes[low] {
			continue
		}
		op := "reads"
		switch kw {
		case "INSERT INTO", "INTO":
			op = "writes"
		case "UPDATE":
			op = "updates"
		case "DELETE FROM":
			op = "deletes"
		}
		r := sqlRef{low, op}
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}
