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

// Reads reports whether archdoc reads code in this language.
func Reads(language string) bool {
	switch language {
	case "TypeScript", "JavaScript", "Python":
		return true
	}
	return false
}

// Read parses the code of the application in dir, a repository-relative directory of repo.
// nested lists other applications' directories: a package inside this one is its own code, not
// part of this application's. ok is false when the language is not one archdoc reads, or when
// no file was found.
func Read(repo string, app archdoc.App, nested []string) (src archdoc.Source, ok bool) {
	if !Reads(app.Language) {
		return src, false
	}
	python := app.Language == "Python"
	root := sourceRoot(repo, app, python)
	src = archdoc.Source{App: app.Dir, Root: root}

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
			if rel != root && (skipDirs[d.Name()] || skipNested[rel] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if lang := languageOf(rel, python); lang != "" && !isTest(d.Name()) && !isConfig(d.Name()) {
			paths = append(paths, rel)
		}
		return nil
	})
	if len(paths) == 0 {
		return src, false
	}

	known := make(map[string]bool, len(paths))
	for _, p := range paths {
		known[p] = true
	}
	var r resolver
	if python {
		r = newPyResolver(root, app.Dir, known)
	} else {
		r = newTSResolver(repo, app, known)
	}

	for _, p := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p)))
		if err != nil {
			continue
		}
		f := archdoc.SourceFile{Path: p, Language: languageOf(p, python), Lines: lines(content)}
		var got facts
		switch f.Language {
		case "Python":
			got, f.Partial = pythonFacts(content, p)
		case "Svelte":
			got, f.Partial = svelteFacts(content, p)
		default:
			got, f.Partial = scriptFacts(content, p, f.Language, 0)
		}
		for _, imp := range got.imports {
			for _, res := range r.resolve(p, imp) {
				res.Prov = archdoc.Provenance{File: p, Line: imp.line, Column: imp.column}
				f.Imports = append(f.Imports, res)
			}
		}
		f.Classes, f.Hosts, f.Calls, f.Prefix, f.Constants = got.classes, got.hosts, got.calls, got.prefix, got.consts
		f.Routers, f.Includes, f.Exports, f.Pages = got.routers, got.incs, got.exports, got.pages
		src.Files = append(src.Files, f)
	}
	return src, true
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
