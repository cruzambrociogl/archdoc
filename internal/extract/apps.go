package extract

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Applications, found by their manifests (F-02, vision D-5).
//
// Discovery used to mean "find the right Compose file". A repository is usually several
// applications — Immich is a server, a web app, a phone app, a machine-learning service and a CLI —
// and most of them are invisible to Compose. Each declares itself in a manifest: package.json,
// pyproject.toml, pubspec.yaml, go.mod. What a package *is* is judged from what it depends on and
// declares, and the line that decided it is the citation. A package that is not a running part of
// the system — a library, a test suite, a documentation site, tooling — is still recorded, with
// why, so a reader can see it was looked at and passed over.

// appSkipDirs are never searched for manifests: installed dependencies, build output, caches,
// version control — wider than discovery's list, because a manifest in any of them is a copy.
var appSkipDirs = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true, "dist": true, "build": true, ".venv": true,
	"venv": true, "__pycache__": true, "target": true, ".next": true, ".svelte-kit": true,
	"coverage": true, ".dart_tool": true, ".archdoc": true, ".turbo": true, ".cache": true,
}

// frameworks decides a JavaScript or TypeScript package's role by its dependencies, in order:
// the first match wins, so a server that also bundles a little React is still a server.
var frameworks = []struct {
	dep, name string
	role      archdoc.AppRole
}{
	{"@nestjs/core", "NestJS", archdoc.RoleService},
	{"express", "Express", archdoc.RoleService},
	{"fastify", "Fastify", archdoc.RoleService},
	{"koa", "Koa", archdoc.RoleService},
	{"hono", "Hono", archdoc.RoleService},
	{"@hapi/hapi", "hapi", archdoc.RoleService},
	{"@docusaurus/core", "Docusaurus", archdoc.RoleDocs},
	{"vitepress", "VitePress", archdoc.RoleDocs},
	{"@sveltejs/kit", "SvelteKit", archdoc.RoleWeb},
	{"next", "Next.js", archdoc.RoleWeb},
	{"nuxt", "Nuxt", archdoc.RoleWeb},
	{"@remix-run/react", "Remix", archdoc.RoleWeb},
	{"@angular/core", "Angular", archdoc.RoleWeb},
	{"react", "React", archdoc.RoleWeb},
	{"vue", "Vue", archdoc.RoleWeb},
	{"svelte", "Svelte", archdoc.RoleWeb},
}

// pyFrameworks does the same for Python.
var pyFrameworks = []struct {
	dep, name string
	role      archdoc.AppRole
}{
	{"fastapi", "FastAPI", archdoc.RoleService},
	{"django", "Django", archdoc.RoleService},
	{"flask", "Flask", archdoc.RoleService},
	{"starlette", "Starlette", archdoc.RoleService},
	{"litestar", "Litestar", archdoc.RoleService},
	{"celery", "Celery", archdoc.RoleService},
	{"typer", "Typer", archdoc.RoleCLI},
	{"click", "Click", archdoc.RoleCLI},
}

// testDeps mark a package as a test suite when nothing above matched.
var testDeps = []string{"@playwright/test", "cypress", "playwright"}

// Apps finds every manifest under root and judges what each package is. Sorted by directory.
func Apps(root string) []archdoc.App {
	var out []archdoc.App
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (appSkipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") && d.Name() != ".github") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		var app *archdoc.App
		switch d.Name() {
		case "package.json":
			app = packageJSON(root, rel)
		case "pyproject.toml":
			app = pyproject(root, rel)
		case "pubspec.yaml":
			app = pubspec(root, rel)
		case "go.mod":
			app = goMod(root, rel)
		}
		if app != nil {
			out = append(out, *app)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest < out[j].Manifest })
	return out
}

func dirOf(rel string) string {
	if d := path.Dir(rel); d != "." {
		return d
	}
	return "."
}

// lineOf is the 1-indexed line of the first occurrence of needle, or 1.
func lineOf(content []byte, needle string) int {
	i := strings.Index(string(content), needle)
	if i < 0 {
		return 1
	}
	return strings.Count(string(content[:i]), "\n") + 1
}

// inTestDir reports a package that lives in a test directory — e2e/, tests/ — whatever it imports.
func inTestDir(dir string) bool {
	for _, seg := range strings.Split(dir, "/") {
		if seg == "e2e" || seg == "tests" || seg == "test" || strings.HasPrefix(seg, "e2e-") || strings.HasSuffix(seg, "-e2e") {
			return true
		}
	}
	return false
}

func packageJSON(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	var pkg struct {
		Name       string            `json:"name"`
		Deps       map[string]string `json:"dependencies"`
		DevDeps    map[string]string `json:"devDependencies"`
		Bin        json.RawMessage   `json:"bin"`
		Main       string            `json:"main"`
		Module     string            `json:"module"`
		Exports    json.RawMessage   `json:"exports"`
		Types      string            `json:"types"`
		Workspaces json.RawMessage   `json:"workspaces"`
	}
	if json.Unmarshal(content, &pkg) != nil {
		return nil
	}
	dir := dirOf(rel)
	app := &archdoc.App{Name: pkg.Name, Dir: dir, Manifest: rel, Language: "JavaScript",
		Prov: archdoc.Provenance{File: rel, Line: lineOf(content, `"name"`)}}
	if app.Name == "" {
		app.Name = path.Base(dir)
	}
	_, hasTS := pkg.Deps["typescript"]
	if _, dev := pkg.DevDeps["typescript"]; dev || hasTS {
		app.Language = "TypeScript"
	} else if _, err := os.Stat(filepath.Join(root, dir, "tsconfig.json")); err == nil {
		app.Language = "TypeScript"
	}
	has := func(dep string) bool {
		_, a := pkg.Deps[dep]
		_, b := pkg.DevDeps[dep]
		return a || b
	}
	cite := func(dep string) archdoc.Provenance {
		return archdoc.Provenance{File: rel, Line: lineOf(content, `"`+dep+`"`)}
	}

	switch {
	case inTestDir(dir):
		app.Role, app.Why = archdoc.RoleTest, "lives in a test directory"
		return app
	case len(pkg.Workspaces) > 0 && dir == ".":
		app.Role, app.Why = archdoc.RoleWorkspace, "a workspace root: it gathers the other packages"
		app.Prov = cite("workspaces")
		return app
	case dir == "." && workspaceFile(root) != "":
		app.Role, app.Why = archdoc.RoleWorkspace, "a workspace root: "+workspaceFile(root)+" gathers the other packages"
		return app
	}
	for _, f := range frameworks {
		if has(f.dep) {
			// React that only renders email templates is a build step, not a front end. A server
			// that uses it for its own emails (Immich's) has matched a server framework already.
			if f.role == archdoc.RoleWeb && (has("react-email") || has("@react-email/components")) {
				app.Role, app.Why, app.FrameworkProv = archdoc.RoleTooling, "email templates, rendered at build time", cite("react-email")
				return app
			}
			app.Framework, app.FrameworkProv, app.Role = f.name, cite(f.dep), f.role
			app.Why = whyRole(f.role, f.name)
			// A package that ships a command and uses a server framework is still a CLI when
			// the framework is only a dependency of its tooling; a bin is the stronger signal
			// for anything that is not a service.
			if f.role != archdoc.RoleService && len(pkg.Bin) > 0 {
				app.Role, app.Why = archdoc.RoleCLI, "declares a command (bin)"
			}
			return app
		}
	}
	for _, t := range testDeps {
		if has(t) {
			app.Role, app.Why, app.FrameworkProv = archdoc.RoleTest, "a test suite ("+t+")", cite(t)
			return app
		}
	}
	switch {
	case len(pkg.Bin) > 0:
		app.Role, app.Why, app.FrameworkProv = archdoc.RoleCLI, "declares a command (bin)", cite("bin")
	case pkg.Main != "" || pkg.Module != "" || len(pkg.Exports) > 0 || pkg.Types != "":
		app.Role, app.Why = archdoc.RoleLibrary, "a library: it exports code for other packages and runs inside them"
	default:
		app.Role, app.Why = archdoc.RoleTooling, "no application framework, command or exports — scripts or configuration"
	}
	return app
}

// workspaceFile names the file that makes the root a monorepo of other packages, or "".
func workspaceFile(root string) string {
	for _, f := range []string{"pnpm-workspace.yaml", "lerna.json", "turbo.json", "nx.json"} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return f
		}
	}
	return ""
}

func whyRole(role archdoc.AppRole, framework string) string {
	switch role {
	case archdoc.RoleService:
		return "a service: depends on " + framework
	case archdoc.RoleWeb:
		return "a web front end: depends on " + framework
	case archdoc.RoleDocs:
		return "a documentation site: depends on " + framework
	case archdoc.RoleCLI:
		return "a command-line tool: depends on " + framework
	}
	return string(role)
}

// pyproject reads the few fields archdoc needs line by line — the name and the dependency lists, in
// both the PEP 621 ([project]) and Poetry ([tool.poetry]) layouts — which keeps each fact's line.
func pyproject(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	dir := dirOf(rel)
	app := &archdoc.App{Name: path.Base(dir), Dir: dir, Manifest: rel, Language: "Python",
		Prov: archdoc.Provenance{File: rel, Line: 1}}
	if dir == "." {
		app.Name = path.Base(root)
	}

	deps := map[string]int{} // dependency → line
	section, inList := "", false
	workspace := 0
	for i, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "["):
			section, inList = strings.Trim(line, "[] "), false
			if section == "tool.uv.workspace" && workspace == 0 {
				workspace = i + 1
			}
			continue
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		}
		key, value, isKV := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if isKV && key == "name" && (section == "project" || section == "tool.poetry") {
			app.Name = strings.Trim(strings.TrimSpace(value), `"'`)
			app.Prov.Line = i + 1
			continue
		}
		if section == "tool.poetry.dependencies" && isKV {
			deps[strings.ToLower(key)] = i + 1
			continue
		}
		if isKV && key == "dependencies" && section == "project" {
			inList = true
			value = strings.TrimSpace(value)
			line = strings.TrimPrefix(value, "[")
		}
		if inList {
			for _, part := range strings.Split(line, ",") {
				name := strings.Trim(strings.TrimSpace(part), `"'[]`)
				if cut := strings.IndexAny(name, "<>=!~[ ;"); cut >= 0 {
					name = name[:cut]
				}
				if name != "" {
					deps[strings.ToLower(name)] = i + 1
				}
			}
			if strings.Contains(line, "]") {
				inList = false
			}
		}
	}

	if inTestDir(dir) {
		app.Role, app.Why = archdoc.RoleTest, "lives in a test directory"
		return app
	}
	if workspace > 0 {
		app.Role, app.Why = archdoc.RoleWorkspace, "a workspace root: [tool.uv.workspace] gathers the other packages"
		app.FrameworkProv = archdoc.Provenance{File: rel, Line: workspace}
		return app
	}
	for _, f := range pyFrameworks {
		if l, ok := deps[f.dep]; ok {
			app.Framework, app.Role, app.Why = f.name, f.role, whyRole(f.role, f.name)
			app.FrameworkProv = archdoc.Provenance{File: rel, Line: l}
			return app
		}
	}
	app.Role, app.Why = archdoc.RoleLibrary, "no application framework among its dependencies"
	return app
}

// pubspec reads a Dart package. A Flutter package that lives under a packages/ directory, or
// declares itself a plugin, is a library; any other Flutter package is an app on a phone.
func pubspec(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	var spec struct {
		Name         string         `yaml:"name"`
		Dependencies map[string]any `yaml:"dependencies"`
		Flutter      map[string]any `yaml:"flutter"`
	}
	if yaml.Unmarshal(content, &spec) != nil {
		return nil
	}
	dir := dirOf(rel)
	app := &archdoc.App{Name: spec.Name, Dir: dir, Manifest: rel, Language: "Dart",
		Prov: archdoc.Provenance{File: rel, Line: lineOf(content, "name:")}}
	_, flutter := spec.Dependencies["flutter"]
	_, plugin := spec.Flutter["plugin"]
	underPackages := strings.Contains("/"+dir+"/", "/packages/")
	switch {
	case inTestDir(dir):
		app.Role, app.Why = archdoc.RoleTest, "lives in a test directory"
	case flutter && !plugin && !underPackages:
		app.Framework, app.Role, app.Why = "Flutter", archdoc.RoleMobile, "an app: depends on the Flutter SDK"
		app.FrameworkProv = archdoc.Provenance{File: rel, Line: lineOf(content, "flutter:")}
	default:
		app.Role, app.Why = archdoc.RoleLibrary, "a Dart package used by others"
	}
	return app
}

// goMod reads a Go module. One with a main package at its root or under cmd/ ships a program.
func goMod(root, rel string) *archdoc.App {
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	dir := dirOf(rel)
	name := path.Base(dir)
	for _, line := range strings.Split(string(content), "\n") {
		if mod, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			name = path.Base(strings.TrimSpace(mod))
			break
		}
	}
	app := &archdoc.App{Name: name, Dir: dir, Manifest: rel, Language: "Go",
		Prov: archdoc.Provenance{File: rel, Line: lineOf(content, "module ")}}
	if inTestDir(dir) {
		app.Role, app.Why = archdoc.RoleTest, "lives in a test directory"
		return app
	}
	if _, err := os.Stat(filepath.Join(root, dir, "cmd")); err == nil {
		app.Role, app.Why = archdoc.RoleService, "a Go program: has a cmd/ directory"
		return app
	}
	if b, err := os.ReadFile(filepath.Join(root, dir, "main.go")); err == nil && strings.Contains(string(b), "package main") {
		app.Role, app.Why = archdoc.RoleService, "a Go program: has a main package"
		return app
	}
	app.Role, app.Why = archdoc.RoleLibrary, "a Go module with no main package"
	return app
}
