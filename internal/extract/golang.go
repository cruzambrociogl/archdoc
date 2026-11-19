package extract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Go modules and the programs in them.
//
// A Go module is one go.mod, but what runs is a program: a main package. One module often builds
// several — a service's cmd/api and cmd/worker, a tool's cmd/<name> — and each is deployed and
// run on its own, so each is an application of its own here, a C4 container, named by its
// directory. Its code is the main package and every package of the module it imports (the code
// reader follows them). The module itself is kept as a library: it is what another repository
// requires, by its module path, and what declares the requirements.

// goModule reads a Go module and the programs it builds.
func goModule(root, rel string) []core.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	dir := dirOf(rel)
	name, module := path.Base(dir), ""
	if dir == "." {
		name = filepath.Base(root)
	}
	var requires []core.Requirement
	inRequire := false
	for i, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if mod, ok := strings.CutPrefix(line, "module "); ok && module == "" {
			module = strings.Trim(strings.TrimSpace(mod), `"`)
			name = path.Base(module)
			continue
		}
		// require x v1 — or a block of them. An indirect one is another package's need, not this one's.
		spec, single := strings.CutPrefix(line, "require ")
		switch {
		case line == "require (":
			inRequire = true
			continue
		case inRequire && line == ")":
			inRequire = false
			continue
		case !inRequire && !single:
			continue
		}
		if !inRequire {
			line = strings.TrimSpace(spec)
		}
		if strings.Contains(line, "// indirect") {
			continue
		}
		if f := strings.Fields(line); len(f) >= 2 && !strings.HasPrefix(f[0], "//") {
			requires = append(requires, core.Requirement{Name: strings.Trim(f[0], `"`), Version: f[1], Prov: core.Provenance{File: rel, Line: i + 1}})
		}
	}
	mod := core.App{Name: name, Package: module, Dir: dir, Manifest: rel, Language: "Go", Requires: requires,
		Prov: core.Provenance{File: rel, Line: lineOf(content, "module ")}}
	if inTestDir(dir) {
		mod.Role, mod.Why = core.RoleTest, "lives in a test directory"
		return []core.App{mod}
	}

	programs := goPrograms(root, dir)
	if len(programs) == 0 {
		mod.Role, mod.Why = core.RoleLibrary, "a Go module with no main package"
		return []core.App{mod}
	}
	var out []core.App
	var elsewhere []string
	for _, p := range programs {
		app := core.App{Name: path.Base(p.dir), Dir: p.dir, Manifest: rel, Language: "Go", Requires: requires, Prov: p.prov}
		if p.dir == dir {
			// The module's own root is the program: the module and the program are one application.
			app = mod
			app.Prov = p.prov
		}
		app.Role, app.Framework, app.FrameworkProv, app.Why = p.role, p.framework, p.frameworkProv, p.why
		if r := toolDir(p.dir, dir); r != "" && app.Role.Container() {
			app.Role, app.Why = core.RoleTooling, "a program kept in "+r+"/: a tool of the repository, not a part of the system"
		}
		out = append(out, app)
		if p.dir != dir {
			elsewhere = append(elsewhere, p.dir)
		}
	}
	if len(elsewhere) == len(programs) {
		mod.Role = core.RoleLibrary
		mod.Why = "a Go module: what it builds is read as " + choose(len(elsewhere), "its program", "its programs") + ", " + strings.Join(elsewhere, ", ")
		out = append([]core.App{mod}, out...)
	}
	return out
}

// toolDirs are folders a repository keeps programs in that are not part of what it runs: its
// scripts, its tools, its experiments and its documentation's.
var toolDirs = map[string]bool{"scripts": true, "script": true, "tools": true, "hack": true, "docs": true, "experiments": true}

// toolDir names the folder a program in one of toolDirs sits in, below the module.
func toolDir(dir, module string) string {
	rel := dir
	if module != "." {
		rel = strings.TrimPrefix(strings.TrimPrefix(dir, module), "/")
	}
	for _, seg := range strings.Split(rel, "/") {
		if toolDirs[seg] {
			return seg
		}
	}
	return ""
}

func choose(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// goProgram is one main package and what it was judged to be.
type goProgram struct {
	dir           string
	prov          core.Provenance // its package clause
	role          core.AppRole
	framework     string
	frameworkProv core.Provenance
	why           string
}

// goSkipDirs are folders the go tool itself never builds from — testdata, and any whose name starts
// with . or _ — and those that hold other people's code.
func goSkipDir(name string) bool {
	return name == "testdata" || name == "vendor" || name == "node_modules" ||
		strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// goPrograms finds the main packages of the module in dir: every folder of it, nested modules
// aside, whose Go files say package main.
func goPrograms(root, dir string) []goProgram {
	byDir := map[string][]string{} // a main package's folder → its files
	base := filepath.Join(root, filepath.FromSlash(dir))
	filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p == base {
				return nil
			}
			if goSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir // another module: its own go.mod reads it
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil || f.Name == nil || f.Name.Name != "main" || ignored(f) {
			return nil
		}
		byDir[dirOf(rel)] = append(byDir[dirOf(rel)], rel)
		return nil
	})
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	out := make([]goProgram, 0, len(dirs))
	for _, d := range dirs {
		files := byDir[d]
		sort.Strings(files)
		out = append(out, judgeProgram(root, d, files))
	}
	return out
}

// ignored reports a file built by no one: //go:build ignore, the way a generator script is kept.
func ignored(f *ast.File) bool {
	for _, g := range f.Comments {
		if g.Pos() > f.Package {
			break
		}
		for _, c := range g.List {
			if t, ok := strings.CutPrefix(c.Text, "//go:build "); ok && strings.TrimSpace(t) == "ignore" {
				return true
			}
		}
	}
	return false
}

// hasMain reports a file that declares func main.
func hasMain(f *ast.File) bool {
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.Name == "main" {
			return true
		}
	}
	return false
}

// Go's command-line frameworks and web frameworks, by module path: what a main package imports
// says what it is.
var (
	goCLIs = map[string]string{
		"github.com/spf13/cobra": "cobra", "github.com/urfave/cli": "urfave/cli", "github.com/urfave/cli/v2": "urfave/cli",
		"github.com/urfave/cli/v3": "urfave/cli", "github.com/alecthomas/kong": "kong", "github.com/peterbourgon/ff/v3/ffcli": "ff",
	}
	goWebs = map[string]string{
		"github.com/gin-gonic/gin": "Gin", "github.com/labstack/echo/v4": "Echo", "github.com/gofiber/fiber/v2": "Fiber",
		"github.com/gofiber/fiber/v3": "Fiber", "github.com/go-chi/chi/v5": "chi", "github.com/go-chi/chi": "chi",
		"github.com/gorilla/mux": "gorilla/mux", "google.golang.org/grpc": "gRPC",
	}
)

// judgeProgram decides what a main package is from its own files: a command-line tool, a server, or
// a worker. Only what the files show counts, each at its line.
func judgeProgram(root, dir string, files []string) goProgram {
	p := goProgram{dir: dir, role: core.RoleService, why: "a Go program: package main in " + dir}
	if dir == "." {
		p.why = "a Go program: package main at the root"
	}
	type seen struct {
		at  core.Provenance
		ok  bool
		via string
	}
	var cli, web, serves, worker, args seen
	mainFound := false
	for i, rel := range files {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, rel, content, parser.SkipObjectResolution)
		if f == nil {
			continue
		}
		at := func(n ast.Node) core.Provenance {
			pos := fset.Position(n.Pos())
			return core.Provenance{File: rel, Line: pos.Line, Column: pos.Column}
		}
		if i == 0 || !mainFound && hasMain(f) {
			// Cited at the file that holds func main, where there is one: main.go, as a rule.
			p.prov = at(f.Name)
			p.prov.Note = "package main"
			mainFound = hasMain(f)
		}
		aliases := map[string]string{} // import path → the name the file uses for it
		for _, imp := range f.Imports {
			ip, _ := strconv.Unquote(imp.Path.Value)
			name := path.Base(ip)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			aliases[ip] = name
			if fw, ok := goCLIs[ip]; ok && !cli.ok {
				cli = seen{at(imp), true, fw}
			}
			if fw, ok := goWebs[ip]; ok && !web.ok {
				web = seen{at(imp), true, fw}
			}
			if strings.HasPrefix(ip, "github.com/riverqueue/river") && !worker.ok {
				worker = seen{at(imp), true, "River"}
			}
			if (ip == "flag" || strings.HasPrefix(ip, "github.com/spf13/pflag")) && !args.ok {
				args = seen{at(imp), true, "flag"}
			}
		}
		httpName, usesHTTP := aliases["net/http"]
		isAlias := map[string]bool{}
		for _, n := range aliases {
			isAlias[n] = true
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				id, ok := x.X.(*ast.Ident)
				if !ok {
					return true
				}
				switch {
				case x.Sel.Name == "Args" && id.Name == "os" && !args.ok:
					args = seen{at(x), true, "os.Args"}
				case usesHTTP && !serves.ok && (x.Sel.Name == "ListenAndServe" || x.Sel.Name == "ListenAndServeTLS") &&
					(id.Name == httpName || !isAlias[id.Name]):
					// http.ListenAndServe, or srv.ListenAndServe on a server of its own — not another
					// package's function of the same name.
					serves = seen{at(x), true, "net/http"}
				}
			case *ast.CompositeLit:
				if sel, ok := x.Type.(*ast.SelectorExpr); ok && usesHTTP && !serves.ok && sel.Sel.Name == "Server" {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == httpName {
						serves = seen{at(x), true, "net/http"}
					}
				}
			}
			return true
		})
	}
	switch {
	case cli.ok:
		p.role, p.framework, p.frameworkProv = core.RoleCLI, cli.via, cli.at
		p.why = fmt.Sprintf("a command-line program: built on %s (%s)", cli.via, cli.at)
	case web.ok:
		p.role, p.framework, p.frameworkProv = core.RoleService, web.via, web.at
		p.why = fmt.Sprintf("a server: built on %s (%s)", web.via, web.at)
	case serves.ok:
		p.role, p.frameworkProv = core.RoleService, serves.at
		p.why = fmt.Sprintf("a server: it listens for HTTP (%s)", serves.at)
	case worker.ok:
		p.role, p.framework, p.frameworkProv = core.RoleService, worker.via, worker.at
		p.why = fmt.Sprintf("a worker: it runs River's jobs (%s)", worker.at)
	case args.ok:
		p.role, p.frameworkProv = core.RoleCLI, args.at
		p.why = fmt.Sprintf("a command-line program: it reads its arguments (%s, %s)", args.via, args.at)
	}
	return p
}
