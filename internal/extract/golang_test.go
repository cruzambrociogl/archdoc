package extract

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Each program a Go module builds is an application of its own, judged from its own files; the
// module stays, as the library others require. Fixtures, tools and other modules are not programs.
func TestEachGoProgramIsAnApplication(t *testing.T) {
	root := tree(t, map[string]string{
		"go.mod": "module github.com/acme/platform\n\ngo 1.27\n\nrequire github.com/riverqueue/river v0.20.0\n",
		"cmd/api/main.go": "package main\n\nimport \"net/http\"\n\nfunc main() {\n\tsrv := &http.Server{Addr: \":8080\"}\n" +
			"\tsrv.ListenAndServe()\n}\n",
		"cmd/api/routes.go":  "package main\n",
		"cmd/worker/main.go": "package main\n\nimport \"github.com/riverqueue/river\"\n\nfunc main() { _ = river.NewClient }\n",
		"cmd/tool/main.go":   "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() { fmt.Println(os.Args) }\n",
		// A CLI that starts a server through another package of its own is still a CLI.
		"cmd/cli/main.go": "package main\n\nimport (\n\t\"flag\"\n\t\"github.com/acme/platform/server\"\n)\n\n" +
			"func main() { flag.Parse(); server.ListenAndServe() }\n",
		"server/server.go":         "package server\n\nfunc ListenAndServe() {}\n",
		"scripts/gen/main.go":      "package main\n\nfunc main() {}\n",
		"internal/gen/gen.go":      "//go:build ignore\n\npackage main\n\nfunc main() {}\n",
		"internal/store/store.go":  "package store\n",
		"testdata/shop/go.mod":     "module shop\n",
		"testdata/shop/main.go":    "package main\n\nfunc main() {}\n",
		"_old/main.go":             "package main\n\nfunc main() {}\n",
		"plugins/x/go.mod":         "module github.com/acme/x\n",
		"plugins/x/cmd/x/main.go":  "package main\n\nfunc main() {}\n",
		"cmd/api/main_test.go":     "package main\n",
		"internal/store/x_test.go": "package store_test\n",
	})
	apps := byDir(Apps(root))
	want := map[string]struct {
		role core.AppRole
		why  string
	}{
		".":               {core.RoleLibrary, "a Go module: what it builds is read as its programs, cmd/api, cmd/cli, cmd/tool, cmd/worker, scripts/gen"},
		"cmd/api":         {core.RoleService, "a server: it listens for HTTP (cmd/api/main.go:6)"},
		"cmd/worker":      {core.RoleService, "a worker: it runs River's jobs (cmd/worker/main.go:3)"},
		"cmd/tool":        {core.RoleCLI, "a command-line program: it reads its arguments (os.Args, cmd/tool/main.go:8)"},
		"cmd/cli":         {core.RoleCLI, "a command-line program: it reads its arguments (flag, cmd/cli/main.go:4)"},
		"scripts/gen":     {core.RoleTooling, "a program kept in scripts/: a tool of the repository, not a part of the system"},
		"plugins/x":       {core.RoleLibrary, "a Go module: what it builds is read as its program, plugins/x/cmd/x"},
		"plugins/x/cmd/x": {core.RoleService, "a Go program: package main in plugins/x/cmd/x"},
	}
	for dir, w := range want {
		a, ok := apps[dir]
		if !ok {
			t.Errorf("%s: not found", dir)
			continue
		}
		if a.Role != w.role || a.Why != w.why {
			t.Errorf("%s: %s, %q\n want %s, %q", dir, a.Role, a.Why, w.role, w.why)
		}
	}
	for dir := range apps {
		if _, ok := want[dir]; !ok {
			t.Errorf("%s: found, and should not be — a fixture, an ignored file, or a folder Go never builds", dir)
		}
	}
	api := apps["cmd/api"]
	if api.Name != "api" || api.Manifest != "go.mod" || api.Package != "" || len(api.Requires) != 1 || api.Prov.File != "cmd/api/main.go" {
		t.Errorf("cmd/api: name %q, manifest %q, package %q, %d requirements, at %s", api.Name, api.Manifest, api.Package, len(api.Requires), api.Prov)
	}
	if apps["."].Package != "github.com/acme/platform" {
		t.Errorf("the module keeps its path, for what requires it: %q", apps["."].Package)
	}
}

// A module whose root package is the program is one application.
func TestAGoModuleThatIsItsProgram(t *testing.T) {
	root := tree(t, map[string]string{
		"go.mod":  "module github.com/acme/tool\n",
		"main.go": "package main\n\nimport \"github.com/spf13/cobra\"\n\nfunc main() { _ = cobra.Command{} }\n",
	})
	got := Apps(root)
	if len(got) != 1 || got[0].Dir != "." || got[0].Role != core.RoleCLI || got[0].Framework != "cobra" || got[0].Package != "github.com/acme/tool" {
		t.Fatalf("got %+v", got)
	}
}

// testdata holds fixtures: a Compose file or a manifest there is never this system's.
func TestTestdataIsNeverTheSystem(t *testing.T) {
	root := tree(t, map[string]string{
		"package.json":                     `{"name": "tool", "bin": {"tool": "index.js"}}`,
		"testdata/shop/docker-compose.yml": "services:\n  api:\n    image: shop\n",
		"testdata/shop/api/package.json":   `{"name": "api", "dependencies": {"@nestjs/core": "10"}}`,
	})
	cands, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cands {
		if strings.HasPrefix(c.File, "testdata/") {
			t.Errorf("considered %s", c.File)
		}
	}
	for _, a := range Apps(root) {
		if strings.HasPrefix(a.Dir, "testdata") {
			t.Errorf("found the application %s", a.Dir)
		}
	}
}
