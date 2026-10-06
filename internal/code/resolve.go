package code

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// tsResolver ties a JavaScript or TypeScript import to a file of the application: relative paths
// first, then the aliases its tsconfig.json declares, then its framework's own ($lib in
// SvelteKit). Anything else names a package.
type tsResolver struct {
	repo    string
	known   map[string]bool
	aliases []alias // longest prefix first
	virtual []string
	kit     bool
}

type alias struct {
	pattern string   // "src/*", "$lib", "@/*"
	targets []string // repository-relative, with the same *
}

// Extensions a TypeScript import leaves out, in the order a bundler tries them. An ESM TypeScript
// file imports its neighbour as "./x.js" and means x.ts, so .js is also replaced.
var scriptExts = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".mts", ".cjs", ".svelte"}

func newTSResolver(repo string, app archdoc.App, known map[string]bool) *tsResolver {
	r := &tsResolver{repo: repo, known: known}
	r.aliases = tsconfigAliases(repo, path.Join(app.Dir, "tsconfig.json"), 0)
	if app.Framework == "SvelteKit" || exists(repo, path.Join(app.Dir, "svelte.config.js")) {
		lib := path.Join(app.Dir, "src/lib")
		r.aliases = append(r.aliases, alias{"$lib", []string{lib}}, alias{"$lib/*", []string{lib + "/*"}})
		// SvelteKit's virtual modules: the framework provides them, no file does.
		r.virtual = []string{"$app/", "$env/", "$service-worker"}
		r.kit = true
	}
	sort.SliceStable(r.aliases, func(i, j int) bool {
		return len(strings.TrimSuffix(r.aliases[i].pattern, "*")) > len(strings.TrimSuffix(r.aliases[j].pattern, "*"))
	})
	return r
}

func (r *tsResolver) resolve(file string, imp rawImport) []archdoc.Import {
	spec := imp.spec
	out := archdoc.Import{Spec: spec}
	if i := strings.IndexByte(spec, '?'); i >= 0 {
		spec = spec[:i] // a bundler's query — "?url", "?raw" — is not part of the path
	}
	if spec == "" {
		return nil
	}
	if r.kit && (spec == "./$types" || strings.HasSuffix(spec, "/$types")) {
		// The types SvelteKit generates for each route: the framework's, and never on disk.
		out.How, out.Package = archdoc.ByPackage, "@sveltejs/kit"
		return []archdoc.Import{out}
	}
	if strings.HasPrefix(spec, ".") {
		base := path.Clean(path.Join(path.Dir(file), spec))
		if t, ok := r.file(base); ok {
			out.Target, out.How = t, archdoc.ByPath
		} else if r.asset(base) {
			return nil // a stylesheet, an image, JSON: not code, so not a dependency between parts
		} else {
			out.How = archdoc.NoMatch
		}
		return []archdoc.Import{out}
	}
	for _, v := range r.virtual {
		if strings.HasPrefix(spec, v) {
			out.How, out.Package = archdoc.ByPackage, "@sveltejs/kit"
			return []archdoc.Import{out}
		}
	}
	for _, a := range r.aliases {
		rest, ok := match(a.pattern, spec)
		if !ok {
			continue
		}
		for _, t := range a.targets {
			base := path.Clean(strings.Replace(t, "*", rest, 1))
			if f, ok := r.file(base); ok {
				out.Target, out.How = f, archdoc.ByAlias
				return []archdoc.Import{out}
			}
			if r.asset(base) {
				return nil
			}
		}
		// An alias matched and no file did. A bare-looking spec may still be a package that
		// merely shares the alias's first segment ("src/*" will not, "@/*" will not); a pattern
		// with no wildcard is an exact claim, so this is the application's own code, missing.
		if strings.Contains(a.pattern, "*") && strings.TrimSuffix(a.pattern, "*") == "" {
			continue
		}
		out.How = archdoc.NoMatch
		return []archdoc.Import{out}
	}
	out.How, out.Package = archdoc.ByPackage, packageName(spec)
	return []archdoc.Import{out}
}

// file finds the code file an import of base means.
func (r *tsResolver) file(base string) (string, bool) {
	if r.known[base] {
		return base, true
	}
	stem := base
	for _, e := range []string{".js", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(base, e) {
			stem = strings.TrimSuffix(base, e)
			break
		}
	}
	for _, e := range scriptExts {
		if r.known[stem+e] {
			return stem + e, true
		}
	}
	for _, e := range scriptExts {
		if r.known[base+"/index"+e] {
			return base + "/index" + e, true
		}
	}
	return "", false
}

// asset reports a non-code file an import names — a stylesheet, an image, JSON — or a file of
// the application archdoc does not read (a test helper, a declaration file).
func (r *tsResolver) asset(base string) bool {
	if fi, err := os.Stat(filepath.Join(r.repo, filepath.FromSlash(base))); err == nil && !fi.IsDir() {
		return true
	}
	return exists(r.repo, base+".d.ts")
}

func match(pattern, spec string) (string, bool) {
	i := strings.Index(pattern, "*")
	if i < 0 {
		return "", spec == pattern
	}
	prefix, suffix := pattern[:i], pattern[i+1:]
	if len(spec) < len(prefix)+len(suffix) || !strings.HasPrefix(spec, prefix) || !strings.HasSuffix(spec, suffix) {
		return "", false
	}
	return spec[len(prefix) : len(spec)-len(suffix)], true
}

// packageName is the package an import names: "@scope/name" or "name", without its subpath.
func packageName(spec string) string {
	parts := strings.Split(spec, "/")
	if strings.HasPrefix(spec, "@") && len(parts) > 1 {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

// tsconfigAliases reads compilerOptions.paths and baseUrl from a tsconfig.json, following a
// relative "extends" a few levels. A tsconfig that extends a generated file (SvelteKit's
// .svelte-kit/tsconfig.json) simply has nothing more to give when that file is absent.
func tsconfigAliases(repo, rel string, depth int) []alias {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil || depth > 3 {
		return nil
	}
	var cfg struct {
		Extends         json.RawMessage `json:"extends"`
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if json.Unmarshal(jsonc(content), &cfg) != nil {
		return nil
	}
	dir := path.Dir(rel)
	var out []alias
	var parent string
	if json.Unmarshal(cfg.Extends, &parent) == nil && strings.HasPrefix(parent, ".") {
		if !strings.HasSuffix(parent, ".json") {
			parent += ".json"
		}
		out = tsconfigAliases(repo, path.Clean(path.Join(dir, parent)), depth+1)
	}
	base := dir
	if cfg.CompilerOptions.BaseURL != "" {
		base = path.Clean(path.Join(dir, cfg.CompilerOptions.BaseURL))
	}
	patterns := make([]string, 0, len(cfg.CompilerOptions.Paths))
	for p := range cfg.CompilerOptions.Paths {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	for _, p := range patterns {
		a := alias{pattern: p}
		for _, t := range cfg.CompilerOptions.Paths[p] {
			a.targets = append(a.targets, path.Clean(path.Join(base, t)))
		}
		out = append(out, a)
	}
	if cfg.CompilerOptions.BaseURL != "" {
		// A baseUrl resolves any bare import that names a file under it.
		out = append(out, alias{pattern: "*", targets: []string{base + "/*"}})
	}
	return out
}

// jsonc strips the comments and trailing commas tsconfig.json allows and JSON does not.
func jsonc(b []byte) []byte {
	var out []byte
	inString := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < len(b) {
				i++
				out = append(out, b[i])
			} else if c == '"' {
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			out = append(out, c)
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			i += 2
			for i+1 < len(b) && !(b[i] == '*' && b[i+1] == '/') {
				i++
			}
			i++
		case c == ',':
			j := i + 1
			for j < len(b) && strings.ContainsRune(" \t\r\n", rune(b[j])) {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	return out
}

func exists(repo, rel string) bool {
	_, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel)))
	return err == nil
}

// pyResolver ties a Python import to a module of the application's own package. Relative imports
// climb from the importing file; absolute ones are looked up where the package lives — the
// directory that holds it, which is what Python puts on its path when the application runs.
type pyResolver struct {
	base  string // the directory absolute imports start from
	known map[string]bool
	tops  map[string]bool // the names importable from base: its modules and packages
}

func newPyResolver(root, dir string, known map[string]bool) *pyResolver {
	base := dir
	if root != dir {
		base = path.Dir(root)
	}
	r := &pyResolver{base: base, known: known, tops: map[string]bool{}}
	for f := range known {
		if rest, ok := strings.CutPrefix(f, base+"/"); ok || base == "." {
			if base == "." {
				rest = f
			}
			r.tops[strings.TrimSuffix(strings.SplitN(rest, "/", 2)[0], ".py")] = true
		}
	}
	return r
}

func (r *pyResolver) resolve(file string, imp rawImport) []archdoc.Import {
	spec := imp.spec
	var from string // the directory the module path starts in
	how := archdoc.ByModule
	mod := spec
	if strings.HasPrefix(spec, ".") {
		dots := len(spec) - len(strings.TrimLeft(spec, "."))
		mod = spec[dots:]
		from = path.Dir(file)
		for i := 1; i < dots; i++ {
			from = path.Dir(from)
		}
		how = archdoc.ByPath
	} else {
		top := strings.SplitN(spec, ".", 2)[0]
		if !r.tops[top] {
			return []archdoc.Import{{Spec: spec, How: archdoc.ByPackage, Package: top}}
		}
		from = r.base
	}
	modPath := path.Join(from, strings.ReplaceAll(mod, ".", "/"))

	// from x import a, b: each name that is itself a module is a dependency on that module.
	var out []archdoc.Import
	for _, n := range imp.names {
		if t, ok := r.module(path.Join(modPath, strings.ReplaceAll(n, ".", "/"))); ok {
			out = append(out, archdoc.Import{Spec: joinSpec(spec, n), Target: t, How: how})
		}
	}
	if len(out) > 0 {
		return out
	}
	if t, ok := r.module(modPath); ok {
		return []archdoc.Import{{Spec: spec, Target: t, How: how}}
	}
	return []archdoc.Import{{Spec: spec, How: archdoc.NoMatch}}
}

func joinSpec(mod, name string) string {
	if strings.HasSuffix(mod, ".") {
		return mod + name
	}
	return mod + "." + name
}

func (r *pyResolver) module(p string) (string, bool) {
	if r.known[p+".py"] {
		return p + ".py", true
	}
	if r.known[p+"/__init__.py"] {
		return p + "/__init__.py", true
	}
	return "", false
}
