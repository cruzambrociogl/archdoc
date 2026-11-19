package code

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Go, read with Go's own parser (go/parser): the language archdoc is written in ships the exact
// parser for it, so no grammar is carried for it.
//
// A Go program is a main package and the packages of its module it imports, directly or through
// each other: those are its code, followed from the main package by its imports. A package is one
// directory — Go's own unit of code — so each is a part. An import names a package by its path; it
// is tied to the files that declare what the importing file uses of it — core.Model is the file of
// internal/core that declares Model — or, where nothing is named, to the package's first file.
//
// Read from the code, each at its line: the routes a program registers (net/http's ServeMux and
// the common routers), the commands it offers (cobra's and urfave/cli's, and a program's own table
// of commands), the jobs it works (River's), and the addresses it names.

// goFile is one Go file, parsed.
type goFile struct {
	path, dir, pkg string
	lines          int
	partial        bool
	imports        []goImport
	refs           map[string]map[string]bool // the names a file uses through an import's name: core.Model is refs["core"]["Model"]
	declares       map[string]bool            // what it declares at the top level, methods aside
	funcs          []goFunc
	hosts          []core.HostRef
	calls          []core.Call
	cmds           []core.Command
	handles        []goHandle
	kinds          map[string]core.Literal // a type → what its Kind() returns: a River job's kind
	works          []goWork
	workers        []goName // the workers it registers: river.AddWorker(workers, &build.Worker{…})
}

type goImport struct {
	path, alias  string // alias is "" when the import is not renamed
	line, column int
}

// goName is a name as written: in its own package (pkg "") or through an import's name.
type goName struct{ pkg, name string }

// goFunc is a function or a method, with the first sentence of its doc comment.
type goFunc struct {
	recv, name string // recv is the receiver's type, "" for a function
	doc        string
	docProv    core.Provenance
}

type goHandle struct {
	method, path string
	handler      goHandler
	prov         core.Provenance
}

// goHandler is a route's handler as written: a function by name, a method on some value, a function
// written in place, or a call that makes the handler.
type goHandler struct {
	ref    goName
	method string // s.emailAsk is the method emailAsk
	on     goName // the type the method is called on, when the function says: s := &orders.Service{}
	inline string // the function a function literal is written in
	made   string // frontend() makes the handler
}

type goWork struct {
	recv string
	args goName
	prov core.Provenance
}

// readGo reads the program in app.Dir: its main package and every package of its module it reaches.
func readGo(repo string, app core.App) (core.Source, bool) {
	modDir, gomod := app.Dir, path.Join(app.Dir, "go.mod")
	if app.Manifest != "" {
		modDir, gomod = path.Dir(app.Manifest), app.Manifest
	}
	module := goModulePath(filepath.Join(repo, filepath.FromSlash(gomod)))
	src := core.Source{App: app.Dir, Root: modDir, Framework: app.Framework, Module: module}
	moduleSchemas := schemas(repo, modDir)

	files := map[string]*goFile{}
	byDir := map[string][]*goFile{}
	base := filepath.Join(repo, filepath.FromSlash(modDir))
	filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(repo, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p == base {
				return nil
			}
			// The go tool's own rule, not the other languages': a Go package may well be called build
			// or test. Only testdata, vendor, and names starting with . or _ are never built.
			name := d.Name()
			if name == "testdata" || name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir // another module
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil || buildIgnored(content) {
			return nil
		}
		g := parseGo(content, rel)
		if strings.HasSuffix(g.pkg, "_test") {
			return nil
		}
		files[rel] = &g
		byDir[g.dir] = append(byDir[g.dir], &g)
		return nil
	})
	if len(byDir[app.Dir]) == 0 {
		// No code of its own here: what its schema says its tables are is still read.
		src.Schemas = moduleSchemas
		return src, len(src.Schemas) > 0
	}
	for _, fs := range byDir {
		sort.Slice(fs, func(i, j int) bool { return fs[i].path < fs[j].path })
	}
	pkgOf := map[string]string{} // a package's directory → its name
	for dir, fs := range byDir {
		pkgOf[dir] = fs[0].pkg
	}
	dirOf := func(importPath string) (string, bool) {
		if module == "" || importPath != module && !strings.HasPrefix(importPath, module+"/") {
			return "", false
		}
		return path.Clean(path.Join(modDir, strings.TrimPrefix(importPath, module))), true
	}

	// The program's packages: its own, and every one of the module it reaches by import. Another
	// program's main package is never imported.
	reached := map[string]bool{app.Dir: true}
	queue := []string{app.Dir}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		for _, f := range byDir[dir] {
			for _, imp := range f.imports {
				to, ok := dirOf(imp.path)
				if ok && !reached[to] && len(byDir[to]) > 0 && pkgOf[to] != "main" {
					reached[to] = true
					queue = append(queue, to)
				}
			}
		}
	}

	// The module's schema is a program's when it reaches the package the schema is kept in, or —
	// as where migrations are kept apart from the code — when it talks to a database at all. A
	// program that does neither has no tables.
	for _, sf := range moduleSchemas {
		dir := path.Dir(sf.Path)
		for d := range reached {
			if d != modDir && (dir == d || strings.HasPrefix(dir, d+"/")) {
				src.Schemas = append(src.Schemas, sf)
				break
			}
		}
	}
	if len(src.Schemas) == 0 && usesDatabase(reached, byDir) {
		src.Schemas = moduleSchemas
	}

	// What each package declares, and its functions by name, for tying imports and handlers.
	declaredIn := map[string]map[string]string{} // dir → name → the file declaring it
	funcsIn := map[string]map[string][]goFunc{}  // dir → name → the functions and methods of that name
	for dir := range reached {
		declaredIn[dir], funcsIn[dir] = map[string]string{}, map[string][]goFunc{}
		for _, f := range byDir[dir] {
			for name := range f.declares {
				if _, seen := declaredIn[dir][name]; !seen {
					declaredIn[dir][name] = f.path
				}
			}
			for _, fn := range f.funcs {
				funcsIn[dir][fn.name] = append(funcsIn[dir][fn.name], fn)
			}
		}
	}
	nameIn := func(f *goFile, imp goImport) string {
		if imp.alias != "" {
			return imp.alias
		}
		if to, ok := dirOf(imp.path); ok && pkgOf[to] != "" {
			return pkgOf[to]
		}
		return guessName(imp.path)
	}
	// importDir is the directory a name a file uses stands for, when it is an import of the module.
	importDir := func(f *goFile, name string) (string, bool) {
		for _, imp := range f.imports {
			if nameIn(f, imp) == name {
				return dirOf(imp.path)
			}
		}
		return "", false
	}

	// Jobs: the kinds the program's job arguments declare, and the workers it registers.
	kinds := map[goName]core.Literal{} // {dir, type} → its kind
	registered := map[goName]bool{}    // {dir, type} of each worker registered
	for dir := range reached {
		for _, f := range byDir[dir] {
			for t, k := range f.kinds {
				kinds[goName{dir, t}] = k
			}
			for _, w := range f.workers {
				at := dir
				if w.pkg != "" {
					if d, ok := importDir(f, w.pkg); ok {
						at = d
					} else {
						continue
					}
				}
				registered[goName{at, w.name}] = true
			}
		}
	}

	var dirs []string
	for dir := range reached {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		for _, g := range byDir[dir] {
			f := core.SourceFile{Path: g.path, Language: "Go", Lines: g.lines, Partial: g.partial,
				Hosts: g.hosts, Calls: g.calls, Commands: g.cmds}
			for _, imp := range g.imports {
				f.Imports = append(f.Imports, resolveGo(g, imp, nameIn(g, imp), dirOf, byDir, declaredIn)...)
			}
			for _, h := range g.handles {
				f.Handles = append(f.Handles, handleOf(g, h, pkgOf, funcsIn, importDir))
			}
			for _, w := range g.works {
				at := dir
				if w.args.pkg != "" {
					d, ok := importDir(g, w.args.pkg)
					if !ok {
						continue
					}
					at = d
				}
				kind, ok := kinds[goName{at, w.args.name}]
				if !ok || len(registered) > 0 && !registered[goName{dir, w.recv}] {
					continue // not a job this program takes: another program registers its worker
				}
				f.Jobs = append(f.Jobs, core.Job{Name: kind.Value, Handler: g.pkg + "." + w.recv + ".Work", NameProv: kind.Prov, Prov: w.prov})
			}
			src.Files = append(src.Files, f)
		}
	}
	return src, true
}

// resolveGo ties one import to the files it names: in the module, the files declaring what the
// importing file uses of the package; outside it, the package.
func resolveGo(g *goFile, imp goImport, name string, dirOf func(string) (string, bool), byDir map[string][]*goFile,
	declaredIn map[string]map[string]string) []core.Import {
	at := core.Provenance{File: g.path, Line: imp.line, Column: imp.column}
	to, internal := dirOf(imp.path)
	if !internal {
		return []core.Import{{Spec: imp.path, Package: imp.path, How: core.ByPackage, Prov: at}}
	}
	if len(byDir[to]) == 0 || declaredIn[to] == nil {
		return []core.Import{{Spec: imp.path, How: core.NoMatch, Prov: at}}
	}
	byFile := map[string][]string{}
	for used := range g.refs[name] {
		if file, ok := declaredIn[to][used]; ok {
			byFile[file] = append(byFile[file], used)
		}
	}
	if len(byFile) == 0 {
		// A blank or dot import, or one whose use was not seen: the package, at its first file.
		return []core.Import{{Spec: imp.path, Target: byDir[to][0].path, How: core.ByModule, Prov: at}}
	}
	targets := make([]string, 0, len(byFile))
	for file := range byFile {
		targets = append(targets, file)
	}
	sort.Strings(targets)
	out := make([]core.Import, 0, len(targets))
	for _, file := range targets {
		names := byFile[file]
		sort.Strings(names)
		out = append(out, core.Import{Spec: imp.path, Target: file, Names: names, How: core.ByModule, Prov: at})
	}
	return out
}

// handleOf names a route's handler by where it is, and gives it its doc comment's first sentence.
func handleOf(g *goFile, h goHandle, pkgOf map[string]string, funcsIn map[string]map[string][]goFunc,
	importDir func(*goFile, string) (string, bool)) core.Handle {
	out := core.Handle{Method: h.method, Path: h.path, Prov: h.prov}
	describe := func(fn goFunc) {
		if fn.doc != "" {
			out.Summary, out.SummaryProv = fn.doc, fn.docProv
		}
	}
	switch hd := h.handler; {
	case hd.inline != "":
		out.Handler = g.pkg + "." + hd.inline + ".func"
	case hd.method != "" && hd.on.name != "":
		// A method on a value whose type the function states: that type's method, in its package.
		dir, pkg := g.dir, g.pkg
		if hd.on.pkg != "" {
			d, ok := importDir(g, hd.on.pkg)
			if !ok {
				out.Handler = hd.on.pkg + "." + hd.on.name + "." + hd.method
				break
			}
			dir, pkg = d, pkgOf[d]
		}
		out.Handler = pkg + "." + hd.on.name + "." + hd.method
		for _, fn := range funcsIn[dir][hd.method] {
			if fn.recv == hd.on.name {
				describe(fn)
				break
			}
		}
	case hd.method != "":
		out.Handler = g.pkg + "." + hd.method
		// A method of this package by that name — one receiver only, or it is ambiguous.
		if fns := funcsIn[g.dir][hd.method]; len(fns) == 1 && fns[0].recv != "" {
			out.Handler = g.pkg + "." + fns[0].recv + "." + hd.method
			describe(fns[0])
		}
	case hd.ref.name != "":
		dir, pkg := g.dir, g.pkg
		if hd.ref.pkg != "" {
			pkg = hd.ref.pkg
			d, ok := importDir(g, hd.ref.pkg)
			if !ok {
				out.Handler = hd.ref.pkg + "." + hd.ref.name // another module's: http.NotFound
				break
			}
			dir, pkg = d, pkgOf[d]
		}
		out.Handler = pkg + "." + hd.ref.name
		for _, fn := range funcsIn[dir][hd.ref.name] {
			if fn.recv == "" {
				describe(fn)
				break
			}
		}
	case hd.made != "":
		out.Handler = hd.made
		if !strings.Contains(hd.made, ".") {
			out.Handler = g.pkg + "." + hd.made
		}
	}
	return out
}

// goDatabases are the packages a program talks to a database through: the standard library's and
// the common drivers and libraries.
var goDatabases = []string{"database/sql", "github.com/jackc/pgx", "github.com/lib/pq", "github.com/jmoiron/sqlx",
	"gorm.io/gorm", "entgo.io/ent", "github.com/uptrace/bun", "modernc.org/sqlite", "github.com/mattn/go-sqlite3",
	"github.com/go-sql-driver/mysql", "go.mongodb.org/mongo-driver"}

// usesDatabase reports a program any of whose packages imports one of goDatabases.
func usesDatabase(reached map[string]bool, byDir map[string][]*goFile) bool {
	for dir := range reached {
		for _, f := range byDir[dir] {
			for _, imp := range f.imports {
				for _, db := range goDatabases {
					if imp.path == db || strings.HasPrefix(imp.path, db+"/") {
						return true
					}
				}
			}
		}
	}
	return false
}

// goModulePath is the module path go.mod declares.
func goModulePath(gomod string) string {
	b, err := os.ReadFile(gomod)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if m, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.Trim(strings.TrimSpace(m), `"`)
		}
	}
	return ""
}

// buildIgnored reports a file built by no one: //go:build ignore before its package clause.
func buildIgnored(content []byte) bool {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if t, ok := strings.CutPrefix(line, "//go:build "); ok && strings.TrimSpace(t) == "ignore" {
			return true
		}
	}
	return false
}

var goMajor = regexp.MustCompile(`^v[0-9]+$`)

// guessName is the name a package outside the module is used by when it is not renamed: its path's
// last element, past a major version, without go- and .vN — github.com/go-chi/chi/v5 is chi,
// gopkg.in/yaml.v3 is yaml.
func guessName(importPath string) string {
	name := path.Base(importPath)
	if goMajor.MatchString(name) {
		name = path.Base(path.Dir(importPath))
	}
	name = strings.TrimPrefix(name, "go-")
	if i := strings.Index(name, ".v"); i > 0 {
		name = name[:i]
	}
	return strings.ReplaceAll(name, "-", "")
}

// The routers whose verbs register routes, by module path.
var goRouters = map[string]bool{
	"github.com/go-chi/chi": true, "github.com/go-chi/chi/v5": true, "github.com/gin-gonic/gin": true,
	"github.com/labstack/echo/v4": true, "github.com/gofiber/fiber/v2": true, "github.com/gofiber/fiber/v3": true,
	"github.com/gorilla/mux": true,
}

// goVerbs are the methods that register a route for one HTTP method: chi's and fiber's Get, gin's
// and echo's GET.
var goVerbs = map[string]string{
	"Get": "GET", "Post": "POST", "Put": "PUT", "Patch": "PATCH", "Delete": "DELETE", "Head": "HEAD", "Options": "OPTIONS",
	"GET": "GET", "POST": "POST", "PUT": "PUT", "PATCH": "PATCH", "DELETE": "DELETE", "HEAD": "HEAD", "OPTIONS": "OPTIONS",
	"Any": "ANY", "All": "ANY",
}

// The calls that send a request to an address: net/http's.
var goRequests = map[string]int{ // the function → which argument is the address
	"Get": 0, "Post": 0, "Head": 0, "PostForm": 0, "NewRequest": 1, "NewRequestWithContext": 2,
}

// The keys a command's literal names it and describes it by: cobra's Use and Short, urfave/cli's
// Name and Usage, a program's own.
var (
	cmdNameKeys    = map[string]bool{"Use": true, "Name": true, "name": true}
	cmdSummaryKeys = []string{"Short", "Usage", "Summary", "summary", "Description", "Help", "help", "Synopsis"}
	cmdName        = regexp.MustCompile(`^[a-z][a-z0-9:_-]*$`)
)

// parseGo reads one Go file: its package, imports, declarations, and what it registers and names.
func parseGo(content []byte, rel string) goFile {
	g := goFile{path: rel, dir: path.Dir(rel), lines: lines(content), refs: map[string]map[string]bool{},
		declares: map[string]bool{}, kinds: map[string]core.Literal{}}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, content, parser.ParseComments|parser.SkipObjectResolution)
	if f == nil {
		g.partial = true
		return g
	}
	g.partial = err != nil
	g.pkg = f.Name.Name
	at := func(p token.Pos) core.Provenance {
		pos := fset.Position(p)
		return core.Provenance{File: rel, Line: pos.Line, Column: pos.Column}
	}

	names := map[string]string{} // import path → the name the file uses for it
	isImport := map[string]bool{}
	routers := false
	for _, imp := range f.Imports {
		ip, _ := strconv.Unquote(imp.Path.Value)
		pos := fset.Position(imp.Pos())
		gi := goImport{path: ip, line: pos.Line, column: pos.Column}
		name := guessName(ip)
		if imp.Name != nil {
			gi.alias, name = imp.Name.Name, imp.Name.Name
		}
		g.imports = append(g.imports, gi)
		names[ip] = name
		isImport[name] = true
		routers = routers || goRouters[ip]
	}
	httpName, usesHTTP := names["net/http"]
	riverName, usesRiver := names["github.com/riverqueue/river"]
	muxes := usesHTTP || routers

	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			fn := goFunc{name: d.Name.Name}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				fn.recv = receiverType(d.Recv.List[0].Type)
			} else {
				g.declares[d.Name.Name] = true
			}
			if d.Doc != nil {
				fn.doc, fn.docProv = docSentence(d.Doc.Text()), at(d.Doc.Pos())
			}
			g.funcs = append(g.funcs, fn)
			if fn.recv == "" || !usesRiver {
				continue
			}
			// River: a job's arguments name its kind; a worker's Work method takes them.
			if fn.name == "Kind" && d.Body != nil && len(d.Body.List) == 1 {
				if ret, ok := d.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
					if v, ok := stringLit(ret.Results[0]); ok {
						g.kinds[fn.recv] = core.Literal{Value: v, Prov: at(ret.Results[0].Pos())}
					}
				}
			}
			if fn.name == "Work" && d.Type.Params != nil {
				var params []ast.Expr
				for _, p := range d.Type.Params.List {
					n := len(p.Names)
					if n == 0 {
						n = 1
					}
					for i := 0; i < n; i++ {
						params = append(params, p.Type)
					}
				}
				if len(params) == 2 {
					if args, ok := riverJobArgs(params[1], riverName); ok {
						g.works = append(g.works, goWork{recv: fn.recv, args: args, prov: at(d.Name.Pos())})
					}
				}
			}
		case *ast.GenDecl:
			for _, sp := range d.Specs {
				switch sp := sp.(type) {
				case *ast.TypeSpec:
					g.declares[sp.Name.Name] = true
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						g.declares[n.Name] = true
					}
				}
			}
		}
	}

	// What the file uses of each import, the addresses it names, its commands and its registered
	// workers — anywhere in it.
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && isImport[id.Name] {
				if g.refs[id.Name] == nil {
					g.refs[id.Name] = map[string]bool{}
				}
				g.refs[id.Name][x.Sel.Name] = true
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				if h, ok := goHost(x, parent, httpName, usesHTTP); ok {
					h.Prov = at(x.Pos())
					g.hosts = append(g.hosts, h)
				}
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}
			id, _ := sel.X.(*ast.Ident)
			if id != nil && usesHTTP && id.Name == httpName {
				if i, ok := goRequests[sel.Sel.Name]; ok && i < len(x.Args) {
					if _, lit := stringLit(x.Args[i]); !lit {
						g.calls = append(g.calls, core.Call{Callee: httpName + "." + sel.Sel.Name,
							Target: shorten(exprText(fset, x.Args[i])), Prov: at(x.Pos())})
					}
				}
			}
			if id != nil && usesRiver && id.Name == riverName && sel.Sel.Name == "AddWorker" && len(x.Args) == 2 {
				if w, ok := typeOfValue(x.Args[1]); ok {
					g.workers = append(g.workers, w)
				}
			}
		case *ast.CompositeLit:
			if c, ok := goCommand(x, parent, at); ok {
				g.cmds = append(g.cmds, c)
			}
		case *ast.SwitchStmt:
			if argSwitch(x.Tag) {
				for _, s := range x.Body.List {
					cc, ok := s.(*ast.CaseClause)
					if !ok {
						continue
					}
					for _, e := range cc.List {
						if v, ok := stringLit(e); ok && cmdName.MatchString(v) {
							g.cmds = append(g.cmds, core.Command{Name: v, Prov: at(e.Pos())})
						}
					}
				}
			}
		}
		stack = append(stack, n)
		return true
	})

	if muxes {
		r := &routeScan{g: &g, at: at, isImport: isImport, httpish: usesHTTP || routers, routers: routers, done: map[*ast.CallExpr]bool{}}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			enclosing := fd.Name.Name
			r.types = map[string]goName{}
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				recv := fd.Recv.List[0]
				enclosing = receiverType(recv.Type) + "." + enclosing
				for _, n := range recv.Names {
					r.types[n.Name] = goName{"", receiverType(recv.Type)}
				}
			}
			for _, p := range fd.Type.Params.List {
				if t, ok := nameOf(unstar(p.Type)); ok {
					for _, n := range p.Names {
						r.types[n.Name] = t
					}
				}
			}
			r.scan(fd.Body, map[string]string{}, enclosing)
		}
	}
	return g
}

// routeScan finds the routes registered in a function, carrying the prefix of each group by the
// variable that holds it.
type routeScan struct {
	g        *goFile
	at       func(token.Pos) core.Provenance
	isImport map[string]bool
	httpish  bool // the file can register routes on a ServeMux: HandleFunc, Handle
	routers  bool // it uses a router whose verbs register routes: Get, GET
	done     map[*ast.CallExpr]bool
	types    map[string]goName // what type each variable of the function holds, where it says
}

func (r *routeScan) scan(body ast.Node, prefix map[string]string, enclosing string) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			// v1 := r.Group("/v1"); api := r.PathPrefix("/api").Subrouter(); s := &orders.Service{…}
			if len(x.Lhs) == 1 && len(x.Rhs) == 1 {
				if id, ok := x.Lhs[0].(*ast.Ident); ok {
					if p, ok := r.groupPrefix(x.Rhs[0], prefix); ok {
						prefix[id.Name] = p
					}
					if t, ok := typeOfValue(x.Rhs[0]); ok {
						r.types[id.Name] = t
					}
				}
			}
		case *ast.ValueSpec:
			// var s orders.Service
			if t, ok := nameOf(unstar(x.Type)); ok {
				for _, n := range x.Names {
					r.types[n.Name] = t
				}
			}
		case *ast.CallExpr:
			if r.done[x] {
				return true
			}
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			// chi: r.Route("/runs", func(r chi.Router) {…}) and r.Group(func(r chi.Router) {…}) — the
			// function's router carries the prefix, and only inside it.
			if r.routers && (sel.Sel.Name == "Route" || sel.Sel.Name == "Group") && len(x.Args) >= 1 {
				if fl, ok := x.Args[len(x.Args)-1].(*ast.FuncLit); ok && fl.Type.Params != nil && len(fl.Type.Params.List) > 0 &&
					len(fl.Type.Params.List[0].Names) > 0 {
					inner := map[string]string{}
					for k, v := range prefix {
						inner[k] = v
					}
					p := r.receiverPrefix(sel.X, prefix)
					if s, ok := stringLit(x.Args[0]); ok && sel.Sel.Name == "Route" {
						p = joinRoute(p, s)
					}
					inner[fl.Type.Params.List[0].Names[0].Name] = p
					r.scan(fl.Body, inner, enclosing)
					return false
				}
			}
			// gorilla: r.HandleFunc("/x", h).Methods("GET", "POST")
			if sel.Sel.Name == "Methods" {
				if inner, ok := sel.X.(*ast.CallExpr); ok {
					if _, pattern, handler, recv, ok := r.registration(inner); ok {
						r.done[inner] = true
						for _, a := range x.Args {
							if m, ok := stringLit(a); ok {
								r.add(strings.ToUpper(m), joinRoute(r.receiverPrefix(recv, prefix), pattern), handler, enclosing, inner)
							}
						}
						return true
					}
				}
			}
			if method, pattern, handler, recv, ok := r.registration(x); ok {
				r.done[x] = true
				r.add(method, joinRoute(r.receiverPrefix(recv, prefix), pattern), handler, enclosing, x)
			}
		}
		return true
	})
}

func (r *routeScan) add(method, p string, handler ast.Expr, enclosing string, call *ast.CallExpr) {
	if method == "" {
		method = "ANY"
	}
	r.g.handles = append(r.g.handles, goHandle{method: method, path: p, handler: r.handler(handler, enclosing, 0), prov: r.at(call.Pos())})
}

// registration reads a call that registers a route: mux.HandleFunc("GET /runs/{id}", h), r.Get("/x", h),
// g.GET("/x", h), r.Method("GET", "/x", h).
func (r *routeScan) registration(call *ast.CallExpr) (method, pattern string, handler, recv ast.Expr, ok bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || len(call.Args) < 2 {
		return
	}
	name, last := sel.Sel.Name, call.Args[len(call.Args)-1]
	switch {
	case (name == "HandleFunc" || name == "Handle") && r.httpish:
		p, isLit := stringLit(call.Args[0])
		if !isLit {
			return
		}
		method, pattern = servePattern(p)
		return method, pattern, last, sel.X, pattern != ""
	case goVerbs[name] != "" && r.routers:
		p, isLit := stringLit(call.Args[0])
		if !isLit || !strings.HasPrefix(p, "/") && p != "" {
			return
		}
		return goVerbs[name], p, last, sel.X, true
	case (name == "Method" || name == "MethodFunc") && r.routers && len(call.Args) >= 3:
		m, ok1 := stringLit(call.Args[0])
		p, ok2 := stringLit(call.Args[1])
		if !ok1 || !ok2 {
			return
		}
		return strings.ToUpper(m), p, last, sel.X, true
	}
	return
}

// groupPrefix is the prefix a call makes a group with: r.Group("/v1"), r.PathPrefix("/api").Subrouter().
func (r *routeScan) groupPrefix(e ast.Expr, prefix map[string]string) (string, bool) {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	switch {
	case sel.Sel.Name == "Group" && r.routers && len(call.Args) >= 1:
		if s, ok := stringLit(call.Args[0]); ok {
			return joinRoute(r.receiverPrefix(sel.X, prefix), s), true
		}
	case sel.Sel.Name == "Subrouter":
		if pc, ok := sel.X.(*ast.CallExpr); ok {
			if ps, ok := pc.Fun.(*ast.SelectorExpr); ok && ps.Sel.Name == "PathPrefix" && len(pc.Args) == 1 {
				if s, ok := stringLit(pc.Args[0]); ok {
					return joinRoute(r.receiverPrefix(ps.X, prefix), s), true
				}
			}
		}
	}
	return "", false
}

// receiverPrefix is the prefix of the router a route is registered on: a group's variable, or a
// router with middleware added — chi's r.With(auth).Get(…).
func (r *routeScan) receiverPrefix(e ast.Expr, prefix map[string]string) string {
	switch x := e.(type) {
	case *ast.Ident:
		return prefix[x.Name]
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "With" || sel.Sel.Name == "Use") {
			return r.receiverPrefix(sel.X, prefix)
		}
	}
	return ""
}

// handler reads what handles a route. A wrapper — s.post(s.emailAsk), gated(token, func…) — takes
// the handler last; a call that does not is what makes it.
func (r *routeScan) handler(e ast.Expr, enclosing string, depth int) goHandler {
	switch x := e.(type) {
	case *ast.Ident:
		return goHandler{ref: goName{"", x.Name}}
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok && r.isImport[id.Name] {
			return goHandler{ref: goName{id.Name, x.Sel.Name}}
		}
		h := goHandler{method: x.Sel.Name}
		if id, ok := x.X.(*ast.Ident); ok {
			h.on = r.types[id.Name]
		}
		return h
	case *ast.FuncLit:
		return goHandler{inline: enclosing}
	case *ast.UnaryExpr:
		return r.handler(x.X, enclosing, depth+1)
	case *ast.CompositeLit:
		return goHandler{made: typeText(x.Type)}
	case *ast.CallExpr:
		if len(x.Args) > 0 && depth < 4 {
			switch last := x.Args[len(x.Args)-1].(type) {
			case *ast.Ident, *ast.SelectorExpr, *ast.FuncLit:
				return r.handler(last, enclosing, depth+1)
			case *ast.CallExpr:
				if h := r.handler(last, enclosing, depth+1); h.made == "" {
					return h
				}
			}
		}
		return goHandler{made: typeText(x.Fun)}
	}
	return goHandler{made: "?"}
}

// servePattern splits a ServeMux pattern: "GET /runs/{id}" is GET and /runs/{id}; "/static/" is any
// method; a host before the path is dropped.
func servePattern(p string) (method, route string) {
	p = strings.TrimSpace(p)
	if m, rest, ok := strings.Cut(p, " "); ok && m == strings.ToUpper(m) && !strings.Contains(m, "/") {
		method, p = m, strings.TrimSpace(rest)
	}
	if i := strings.Index(p, "/"); i > 0 {
		p = p[i:]
	}
	if !strings.HasPrefix(p, "/") {
		return "", ""
	}
	return method, p
}

// joinRoute puts a group's prefix before a route's path.
func joinRoute(prefix, p string) string {
	if prefix == "" {
		return p
	}
	if p == "" || p == "/" {
		return prefix
	}
	return strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(p, "/")
}

// receiverType is a method's receiver type: *Service[T] is Service.
func receiverType(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return receiverType(x.X)
	case *ast.IndexExpr:
		return receiverType(x.X)
	case *ast.IndexListExpr:
		return receiverType(x.X)
	case *ast.Ident:
		return x.Name
	}
	return ""
}

// riverJobArgs reads a Work method's job parameter: *river.Job[jobs.Build] is jobs.Build.
func riverJobArgs(e ast.Expr, river string) (goName, bool) {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return goName{}, false
	}
	idx, ok := star.X.(*ast.IndexExpr)
	if !ok {
		return goName{}, false
	}
	sel, ok := idx.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Job" {
		return goName{}, false
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != river {
		return goName{}, false
	}
	return nameOf(idx.Index)
}

// typeOfValue is the type of a value made in place: &build.Worker{…} is build.Worker.
func typeOfValue(e ast.Expr) (goName, bool) {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return goName{}, false
	}
	return nameOf(cl.Type)
}

func unstar(e ast.Expr) ast.Expr {
	if s, ok := e.(*ast.StarExpr); ok {
		return s.X
	}
	return e
}

func nameOf(e ast.Expr) (goName, bool) {
	switch x := e.(type) {
	case *ast.Ident:
		return goName{"", x.Name}, true
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			return goName{id.Name, x.Sel.Name}, true
		}
	}
	return goName{}, false
}

func typeText(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return typeText(x.X) + "." + x.Sel.Name
	case *ast.StarExpr:
		return typeText(x.X)
	case *ast.IndexExpr:
		return typeText(x.X)
	case *ast.CallExpr:
		return typeText(x.Fun)
	}
	return "?"
}

// goCommand reads a command declared as a literal: cobra's &cobra.Command{Use: "serve", Short: …},
// urfave/cli's &cli.Command{Name: …, Usage: …}, or a row of a program's own table of commands —
// []command{{"init", "<path>", "create .archdoc/…", …}}.
func goCommand(x *ast.CompositeLit, parent ast.Node, at func(token.Pos) core.Provenance) (core.Command, bool) {
	isCommand := func(t ast.Expr) bool {
		if s, ok := t.(*ast.StarExpr); ok {
			t = s.X
		}
		n := strings.ToLower(typeText(t))
		if i := strings.LastIndex(n, "."); i >= 0 {
			n = n[i+1:]
		}
		return n == "command" || n == "cmd"
	}
	typed := x.Type != nil && isCommand(x.Type)
	if !typed && x.Type == nil {
		if p, ok := parent.(*ast.CompositeLit); ok {
			if a, ok := p.Type.(*ast.ArrayType); ok && isCommand(a.Elt) {
				typed = true
			}
		}
	}
	if !typed || len(x.Elts) == 0 {
		return core.Command{}, false
	}
	c := core.Command{Prov: at(x.Pos())}
	if _, keyed := x.Elts[0].(*ast.KeyValueExpr); keyed {
		summaries := map[string]ast.Expr{}
		for _, e := range x.Elts {
			kv, ok := e.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			k, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if cmdNameKeys[k.Name] {
				if v, ok := stringLit(kv.Value); ok {
					c.Name = strings.Fields(v + " ")[0]
					c.Prov = at(kv.Value.Pos())
				}
			}
			summaries[k.Name] = kv.Value
		}
		for _, k := range cmdSummaryKeys {
			if v, ok := stringLit(summaries[k]); ok && v != "" {
				c.Summary, c.SummaryProv = docSentence(v), at(summaries[k].Pos())
				break
			}
		}
	} else {
		for i, e := range x.Elts {
			v, ok := stringLit(e)
			if !ok {
				continue
			}
			if c.Name == "" {
				if i > 0 {
					break // a command's name comes first
				}
				c.Name, c.Prov = v, at(e.Pos())
				continue
			}
			if strings.Contains(v, " ") && !strings.HasPrefix(v, "<") {
				c.Summary, c.SummaryProv = docSentence(v), at(e.Pos())
				break
			}
		}
	}
	if !cmdName.MatchString(c.Name) {
		return core.Command{}, false
	}
	return c, true
}

// argSwitch reports a switch on a program's first argument: switch os.Args[1], switch args[0],
// switch flag.Arg(0).
func argSwitch(tag ast.Expr) bool {
	switch x := tag.(type) {
	case *ast.IndexExpr:
		v, ok := x.Index.(*ast.BasicLit)
		if !ok || v.Value != "0" && v.Value != "1" {
			return false
		}
		name := typeText(x.X)
		return name == "os.Args" || name == "args" || strings.HasSuffix(name, ".Args") || name == "argv"
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Arg" && len(x.Args) == 1 {
			v, ok := x.Args[0].(*ast.BasicLit)
			return ok && v.Value == "0"
		}
	}
	return false
}

// goHost reads an address a string names, with the key that holds it — a field, a constant or a
// variable — and whether a request is sent to it.
func goHost(lit *ast.BasicLit, parent ast.Node, httpName string, usesHTTP bool) (core.HostRef, bool) {
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return core.HostRef{}, false
	}
	h, ok := hostOf(v)
	if !ok {
		return core.HostRef{}, false
	}
	switch p := parent.(type) {
	case *ast.KeyValueExpr:
		if k, ok := p.Key.(*ast.Ident); ok {
			h.Key = k.Name
		}
	case *ast.ValueSpec:
		if len(p.Names) > 0 {
			h.Key = p.Names[0].Name
		}
	case *ast.AssignStmt:
		if len(p.Lhs) > 0 {
			h.Key = typeText(p.Lhs[0])
		}
	case *ast.BinaryExpr:
		h.Built = p.Op == token.ADD
	case *ast.CallExpr:
		sel, ok := p.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}
		id, _ := sel.X.(*ast.Ident)
		switch {
		case id != nil && usesHTTP && id.Name == httpName:
			if i, ok := goRequests[sel.Sel.Name]; ok && i < len(p.Args) && p.Args[i] == ast.Expr(lit) {
				h.Called = true
			}
		case id != nil && id.Name == "fmt" && strings.HasPrefix(sel.Sel.Name, "Sprint") && strings.Contains(v, "%"):
			h.Built = true
		}
	}
	return h, true
}

// stringLit is a string literal's value.
func stringLit(e ast.Expr) (string, bool) {
	b, ok := e.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(b.Value)
	return v, err == nil
}

func exprText(fset *token.FileSet, e ast.Expr) string {
	var b strings.Builder
	if err := printer.Fprint(&b, fset, e); err != nil {
		return fmt.Sprintf("%T", e)
	}
	return b.String()
}

// docSentence is the first sentence of a doc comment, on one line.
func docSentence(text string) string {
	para, _, _ := strings.Cut(strings.TrimSpace(text), "\n\n")
	s := strings.Join(strings.Fields(para), " ")
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i+1]
	}
	if len(s) > 200 {
		s = s[:197] + "…"
	}
	return s
}
