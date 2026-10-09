package extract

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

const openapiDoc = `{
  "openapi": "3.0.0",
  "paths": {
    "/albums": {"get": {}, "post": {}},
    "/albums/{id}": {"get": {}, "delete": {}, "parameters": []},
    "/assets": {"get": {}},
    "/assets/{id}/thumbnail": {"get": {}},
    "/users": {"get": {}},
    "/users/me": {"get": {}},
    "/search/places": {"get": {}}
  }
}
`

// An API description, and its two kinds of client: a package whose code names its paths, and a
// directory a generator command writes into. A package that calls two endpoints is not a client.
func TestAPIDocumentsAndTheirClients(t *testing.T) {
	root := tree(t, map[string]string{
		"open-api/specs-openapi.json":      openapiDoc,
		"open-api/bin/generate.sh":         "#!/bin/sh\ncd ..\nopenapi-generator-cli generate -g dart -i ./specs-openapi.json -o ../mobile/generated/openapi\n",
		"open-api/not-an-api.openapi.json": `{"name": "something else"}`,
		"packages/sdk/package.json":        `{"name": "@x/sdk", "main": "index.js"}`,
		"packages/sdk/src/client.ts": "export const a = () => get(`/albums/${id}`);\nexport const b = () => get('/albums');\n" +
			"export const c = () => get(\"/assets\");\nexport const d = () => get(`/assets/${id}/thumbnail`);\nexport const e = () => get('/users/me');\n",
		"mobile/pubspec.yaml": "name: app\ndependencies:\n  flutter:\n    sdk: flutter\n",
		"web/package.json":    `{"name": "web", "dependencies": {"react": "^19", "@x/sdk": "workspace:*"}}`,
		"web/src/health.ts":   "fetch('/albums'); fetch('/users');\n",
	})
	apis := APIs(root, Apps(root))
	if len(apis) != 1 || apis[0].File != "open-api/specs-openapi.json" || len(apis[0].Operations) != 9 || apis[0].Prov.Line != 3 {
		t.Fatalf("apis %+v", apis)
	}
	if op := apis[0].Operations[0]; op.Method != "GET" || op.Path != "/albums" {
		t.Errorf("first operation %+v — they are sorted by path, then method", op)
	}
	clients := map[string]archdoc.APIClient{}
	for _, c := range apis[0].Clients {
		clients[c.App] = c
	}
	if len(clients) != 2 {
		t.Fatalf("clients %+v, want the sdk and the mobile app — not the web app's two fetches", apis[0].Clients)
	}
	if c := clients["packages/sdk"]; c.Prov.File != "packages/sdk/src/client.ts" || !strings.Contains(c.How, "3 of the 5") {
		t.Errorf("sdk client %+v", c)
	}
	if c := clients["mobile"]; c.Dir != "mobile/generated/openapi" || c.Prov.File != "open-api/bin/generate.sh" || c.Prov.Line != 3 {
		t.Errorf("generated client %+v", c)
	}
}

// A package with a command and exports is a tool, unless a package that runs depends on it.
func TestAToolOthersDependOnIsALibrary(t *testing.T) {
	root := tree(t, map[string]string{
		"packages/cli/package.json": `{"name": "@x/cli", "bin": {"x": "./bin/x"}, "exports": "./dist/index.js"}`,
		"packages/kit/package.json": `{"name": "@x/kit", "bin": {"kit": "./kit.mjs"}, "exports": "./dist/index.js"}`,
		"server/package.json":       `{"name": "api", "dependencies": {"express": "^4", "@x/kit": "workspace:*"}}`,
		"e2e/package.json":          `{"name": "e2e", "devDependencies": {"@x/cli": "workspace:*", "vitest": "^3"}}`,
	})
	apps := byDir(Apps(root))
	if a := apps["packages/cli"]; a.Role != archdoc.RoleCLI {
		t.Errorf("the cli is %s — a test suite depending on it does not make it a library", a.Role)
	}
	if a := apps["packages/kit"]; a.Role != archdoc.RoleLibrary || !strings.Contains(a.Why, "server/package.json") {
		t.Errorf("the kit is %s: %s", a.Role, a.Why)
	}
}

// A repository with no manifest is still a program: scripts a person runs, or a page and what it
// loads. One with a manifest that describes something running is left to that manifest.
func TestAProjectWithNoManifest(t *testing.T) {
	script := Apps(tree(t, map[string]string{"report.py": "print(1)\n", "helpers/io.py": "x = 1\n", "tests/test_report.py": "import report\n"}))
	if len(script) != 1 || !script[0].Loose || script[0].Role != archdoc.RoleCLI || script[0].Language != "Python" ||
		script[0].Manifest != "report.py" || script[0].Prov.Line != 1 || !strings.Contains(script[0].Why, "2 Python file") {
		t.Errorf("a folder of scripts: %+v", script)
	}
	page := Apps(tree(t, map[string]string{"index.html": "<html></html>\n", "about.html": "<html></html>\n", "js/app.js": "fetch('/x')\n"}))
	if len(page) != 1 || !page[0].Loose || page[0].Role != archdoc.RoleWeb || page[0].Manifest != "index.html" {
		t.Errorf("a page and its script: %+v", page)
	}
	if got := Apps(tree(t, map[string]string{"package.json": `{"name": "x", "dependencies": {"react": "^19"}}`, "tool.py": "print(1)\n"})); len(got) != 1 || got[0].Loose {
		t.Errorf("a manifest describes it already: %+v", got)
	}
	if got := Apps(tree(t, map[string]string{"README.md": "nothing to run\n"})); len(got) != 0 {
		t.Errorf("nothing to run, yet: %+v", got)
	}
}
