// Package code reads an application's source with a parser (F-03): every file, and what each
// one imports, at the line that imports it.
//
// The parser is tree-sitter, run in pure Go (gotreesitter; docs/decisions.md, 5 Oct). Only the
// grammars archdoc uses are imported — one package each — so the binary carries five grammars,
// not two hundred.
//
// Reading code is extraction like reading a Compose file: every fact cites a line. What an import
// names is resolved by path, by a declared alias, or as a module of the application's own package,
// and the import records which (vision D-4). An import that looks like the application's own code
// and matches no file stays, marked unresolved (D-6).
package code

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// skipDirs are never read: installed dependencies, build output, caches, and tests — a test
// imports what it tests, and those edges would describe the test suite, not the system.
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true, "dist": true, "build": true, ".venv": true,
	"venv": true, "__pycache__": true, "target": true, ".next": true, ".svelte-kit": true,
	"coverage": true, ".archdoc": true, ".turbo": true, ".cache": true, "out": true,
	"test": true, "tests": true, "__tests__": true, "__mocks__": true, "test-data": true,
	"e2e": true, "fixtures": true,
}

// servedAsIs are the folders a framework copies to the browser untouched — Next.js's and Vite's
// public/, SvelteKit's static/. What is in them was not built from the application's code: it is
// someone else's, copied in (Supabase's studio keeps all of Monaco there). A page with no
// manifest has no such convention, and its public/ may be all the code it has.
var servedAsIs = map[string]bool{"public": true, "static": true}

// dotnetSkip are a .NET project's folders that hold no code of its own: the build's output, the
// static files it serves, and the migrations Entity Framework generates.
var dotnetSkip = map[string]bool{"bin": true, "obj": true, "wwwroot": true, "Migrations": true}

// Reads reports whether archdoc reads code in this language.
func Reads(language string) bool {
	switch language {
	case "TypeScript", "JavaScript", "Python", "Dart", "Java", "C#":
		return true
	}
	return false
}

// Read parses the code of the application in dir, a repository-relative directory of repo.
// nested lists other applications' directories: a package inside this one is its own code, not
// part of this application's. ok is false when nothing was read: no file in a language archdoc
// reads, and no schema file beside it.
func Read(repo string, app archdoc.App, nested []string) (src archdoc.Source, ok bool) {
	if !Reads(app.Language) {
		// Its code is not read, but what it says its tables are still is.
		src = archdoc.Source{App: app.Dir, Root: app.Dir, Schemas: schemas(repo, app.Dir)}
		return src, len(src.Schemas) > 0
	}
	python := app.Language == "Python"
	dart := app.Language == "Dart"
	java := app.Language == "Java"
	csharp := app.Language == "C#"
	root := sourceRoot(repo, app, python)
	if dart {
		root = app.Dir
		if fi, err := os.Stat(filepath.Join(repo, filepath.FromSlash(app.Dir), "lib")); err == nil && fi.IsDir() {
			root = path.Join(app.Dir, "lib") // a Dart package's own code is its lib/
		}
	}
	// Java's packages start at src/main/java; its parts are the folders under the module's own base
	// package, com/example/petclinic. C# keeps a project's code at the project's own folder.
	var jroot string
	if java {
		jroot = javaRoot(repo, app.Dir)
		root = javaBase(repo, jroot)
	}
	if csharp {
		root = app.Dir
	}
	if app.Loose {
		root = app.Dir // no manifest, so no src/ convention: every file is the application's
	}
	src = archdoc.Source{App: app.Dir, Root: root, Loose: app.Loose, Framework: app.Framework}

	skipNested := map[string]bool{}
	for _, n := range nested {
		if n != app.Dir && (app.Dir == "." || strings.HasPrefix(n+"/", app.Dir+"/")) {
			skipNested[n] = true
		}
	}

	var paths []string
	filepath.WalkDir(filepath.Join(repo, filepath.FromSlash(root)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(repo, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != root && (skipDirs[d.Name()] || skipNested[rel] || strings.HasPrefix(d.Name(), ".") || !app.Loose && servedAsIs[d.Name()] ||
				csharp && dotnetSkip[d.Name()]) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".min.js") {
			return nil // minified: a copy of someone else's code, as a tool left it
		}
		switch {
		case dart:
			if isDart(d.Name()) {
				paths = append(paths, rel)
			}
		case java:
			if strings.HasSuffix(d.Name(), ".java") && !strings.HasSuffix(d.Name(), "Test.java") && !strings.HasSuffix(d.Name(), "Tests.java") {
				paths = append(paths, rel)
			}
		case csharp:
			if isCSharp(d.Name()) {
				paths = append(paths, rel)
			}
		default:
			if lang := languageOf(rel, python); lang != "" && !isTest(d.Name()) && !isConfig(d.Name()) {
				paths = append(paths, rel)
			} else if app.Loose && !python && isPage(rel) {
				paths = append(paths, rel) // a page with no manifest: its own <script> blocks are its code
			}
		}
		return nil
	})
	src.Schemas = schemas(repo, app.Dir)
	if java || csharp {
		src.Settings = settings(repo, app.Dir, java)
	}
	if app.Loose {
		src.Documents = documents(repo, app.Dir)
	}
	if len(paths) == 0 {
		return src, len(src.Schemas) > 0
	}

	// Every file is read before any import is resolved: a C# file's uses are the types it names, and
	// which file declares a type is only known once all of them have been read.
	got := make(map[string]facts, len(paths))
	files := make([]archdoc.SourceFile, 0, len(paths))
	for _, p := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p)))
		if err != nil {
			continue
		}
		f := archdoc.SourceFile{Path: p, Language: languageOf(p, python), Lines: lines(content)}
		switch {
		case dart:
			f.Language = "Dart"
		case java:
			f.Language = "Java"
		case csharp:
			f.Language = "C#"
		}
		if f.Language == "" && isPage(p) {
			f.Language = "HTML"
		}
		var g facts
		switch f.Language {
		case "Dart":
			g = dartFacts(content)
		case "Python":
			g, f.Partial = pythonFacts(content, p)
		case "Java":
			g, f.Partial = javaFacts(content, p)
		case "C#":
			g, f.Partial = csharpFacts(content, p)
		case "Svelte", "HTML":
			// A page's <script> blocks are read as a Svelte component's are; the markup is not.
			g, f.Partial = svelteFacts(content, p)
		default:
			g, f.Partial = scriptFacts(content, p, f.Language, 0)
		}
		got[p] = g
		files = append(files, f)
	}

	known := make(map[string]bool, len(files))
	for _, f := range files {
		known[f.Path] = true
	}
	var r resolver
	switch {
	case dart:
		r = &dartResolver{name: app.Name, lib: root, known: known}
	case python:
		r = newPyResolver(root, app.Dir, known)
	case java:
		r = newJavaResolver(jroot, root, known)
	case csharp:
		names := make([]string, 0, len(files))
		for _, f := range files {
			names = append(names, f.Path)
		}
		r = newCSResolver(names, got)
	default:
		r = newTSResolver(repo, app, known)
	}

	for _, f := range files {
		g := got[f.Path]
		imports := g.imports
		for _, ref := range g.refs {
			ref.spec = csMention + ref.spec
			imports = append(imports, ref)
		}
		for _, imp := range imports {
			for _, res := range r.resolve(f.Path, imp) {
				res.Prov = archdoc.Provenance{File: f.Path, Line: imp.line, Column: imp.column}
				f.Imports = append(f.Imports, res)
			}
		}
		f.Classes, f.Hosts, f.Calls, f.Prefix, f.Constants = g.classes, g.hosts, g.calls, g.prefix, g.consts
		f.Routers, f.Includes, f.Exports, f.Pages, f.Commands = g.routers, g.incs, g.exports, g.pages, g.cmds
		f.Main = g.main
		src.Files = append(src.Files, f)
	}
	return src, true
}

// documents are the HTML pages of an application with no manifest, each at its <title> or its
// first line: what a person opens.
func documents(repo, dir string) []archdoc.Literal {
	var out []archdoc.Literal
	filepath.WalkDir(filepath.Join(repo, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() != "." && p != filepath.Join(repo, filepath.FromSlash(dir)) && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(p)); ext != ".html" && ext != ".htm" {
			return nil
		}
		rel, _ := filepath.Rel(repo, p)
		rel = filepath.ToSlash(rel)
		content, err := os.ReadFile(p)
		if err != nil || len(content) > 2<<20 {
			return nil
		}
		title, line := "", 1
		lower := strings.ToLower(string(content))
		if i := strings.Index(lower, "<title>"); i >= 0 {
			if j := strings.Index(lower[i:], "</title>"); j > 0 {
				title = strings.TrimSpace(string(content[i+len("<title>") : i+j]))
				line = 1 + strings.Count(lower[:i], "\n")
			}
		}
		out = append(out, archdoc.Literal{Value: title, Prov: archdoc.Provenance{File: rel, Line: line}})
		return nil
	})
	return out
}

// schemas are the schema files beside an application's code: Prisma's, and the SQL its
// migrations are written in.
func schemas(repo, dir string) []archdoc.SourceFile {
	return append(prismaSchemas(repo, dir), sqlSchemas(repo, dir)...)
}

// rawImport is an import as the parser found it, before resolution.
type rawImport struct {
	spec         string
	names        []string // Python's `from x import a, b`: a and b may be modules themselves
	line, column int
}

type resolver interface {
	resolve(file string, imp rawImport) []archdoc.Import
}

// sourceRoot is where an application's code starts. JavaScript and TypeScript keep it in src/ by
// near-universal convention; a Python application is its package — the directory holding an
// __init__.py, at the top or under src/: the only one, or the one named like the project
// (immich-ml's immich_ml). Anything else: the application's directory itself.
func sourceRoot(repo string, app archdoc.App, python bool) string {
	dir := app.Dir
	join := func(parts ...string) string { return path.Clean(path.Join(parts...)) }
	isDir := func(rel string) bool {
		fi, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel)))
		return err == nil && fi.IsDir()
	}
	if !python {
		if isDir(join(dir, "src")) {
			return join(dir, "src")
		}
		return dir
	}
	for _, base := range []string{join(dir, "src"), dir} {
		entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(base)))
		if err != nil {
			continue
		}
		var pkgs []string
		for _, e := range entries {
			if !e.IsDir() || skipDirs[e.Name()] || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(base), e.Name(), "__init__.py")); err == nil {
				pkgs = append(pkgs, e.Name())
			}
		}
		if len(pkgs) == 1 {
			return join(base, pkgs[0])
		}
		named := strings.ToLower(strings.NewReplacer("-", "_", ".", "_").Replace(app.Name))
		for _, p := range pkgs {
			if p == named {
				return join(base, p)
			}
		}
	}
	return dir
}

func languageOf(rel string, python bool) string {
	if python {
		if strings.HasSuffix(rel, ".py") {
			return "Python"
		}
		return ""
	}
	switch {
	case strings.HasSuffix(rel, ".d.ts"):
		return ""
	case strings.HasSuffix(rel, ".ts"), strings.HasSuffix(rel, ".mts"), strings.HasSuffix(rel, ".cts"):
		return "TypeScript"
	case strings.HasSuffix(rel, ".tsx"):
		return "TSX"
	case strings.HasSuffix(rel, ".js"), strings.HasSuffix(rel, ".mjs"), strings.HasSuffix(rel, ".cjs"), strings.HasSuffix(rel, ".jsx"):
		return "JavaScript"
	case strings.HasSuffix(rel, ".svelte"):
		return "Svelte"
	}
	return ""
}

func isPage(rel string) bool {
	return strings.HasSuffix(rel, ".html") || strings.HasSuffix(rel, ".htm")
}

// isTest reports a test file by its name: foo.spec.ts, foo.test.tsx, test_foo.py, foo_test.py,
// conftest.py.
func isTest(name string) bool {
	if strings.Contains(name, ".spec.") || strings.Contains(name, ".test.") || name == "conftest.py" {
		return true
	}
	return strings.HasSuffix(name, ".py") && (strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test.py"))
}

// isConfig reports a tool's configuration file — eslint.config.mjs, vite.config.ts,
// lint-staged.config.js: code that configures the build, not code of the application.
func isConfig(name string) bool {
	return strings.Contains(name, ".config.") || strings.HasPrefix(name, ".")
}

func lines(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	n := bytes.Count(b, []byte{'\n'})
	if b[len(b)-1] != '\n' {
		n++
	}
	return n
}
