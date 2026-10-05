package extract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func byDir(apps []archdoc.App) map[string]archdoc.App {
	out := map[string]archdoc.App{}
	for _, a := range apps {
		out[a.Dir] = a
	}
	return out
}

// Immich's shape, in miniature: a monorepo whose deployed Compose file names images, and whose dev
// Compose file builds the same service from the server's directory. That build line ties the
// service to the server; nothing ties the web app to anything, so it is a container of its own.
func TestApplicationsAndWhatRunsThem(t *testing.T) {
	root := tree(t, map[string]string{
		"pnpm-workspace.yaml":           "packages: ['*']\n",
		"package.json":                  `{"name": "mono", "private": true}`,
		"server/package.json":           "{\n  \"name\": \"api\",\n  \"dependencies\": {\n    \"@nestjs/core\": \"^11\",\n    \"react-email\": \"^5\"\n  },\n  \"devDependencies\": {\"typescript\": \"^5\"}\n}\n",
		"web/package.json":              `{"name": "site", "devDependencies": {"@sveltejs/kit": "^2", "typescript": "^5"}}`,
		"packages/sdk/package.json":     `{"name": "@x/sdk", "main": "dist/index.js"}`,
		"e2e/package.json":              `{"name": "e2e", "devDependencies": {"@playwright/test": "^1"}}`,
		"docker/docker-compose.yml":     "services:\n  api-server:\n    image: example/api:1\n    ports: ['80:3000']\n  db:\n    image: postgres:16\n",
		"docker/docker-compose.dev.yml": "services:\n  api-server:\n    build:\n      context: ../\n      dockerfile: server/Dockerfile\n",
	})
	fs, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	apps := byDir(fs.Apps)

	want := map[string]archdoc.AppRole{
		".": archdoc.RoleWorkspace, "server": archdoc.RoleService, "web": archdoc.RoleWeb,
		"packages/sdk": archdoc.RoleLibrary, "e2e": archdoc.RoleTest,
	}
	for dir, role := range want {
		if apps[dir].Role != role {
			t.Errorf("%s: role %q, want %q (%s)", dir, apps[dir].Role, role, apps[dir].Why)
		}
	}
	server := apps["server"]
	if server.Framework != "NestJS" || server.Language != "TypeScript" || server.FrameworkProv.Line != 4 {
		t.Errorf("server: %s %s, framework cited at line %d", server.Framework, server.Language, server.FrameworkProv.Line)
	}
	if d := server.Deployed; d == nil || d.Service != "api-server" || d.ByName || d.Prov.File != "docker/docker-compose.dev.yml" || d.Prov.Line != 3 {
		t.Errorf("server is not tied to api-server by the dev build line: %+v", d)
	}
	if apps["web"].Deployed != nil {
		t.Errorf("web was tied to %+v, though nothing builds it", apps["web"].Deployed)
	}
}

// With no build line anywhere, an exact name ties a service to an application, and says so.
// A resemblance never does.
func TestTiedByExactNameOnly(t *testing.T) {
	root := tree(t, map[string]string{
		"docker-compose.yml":     "services:\n  streaming:\n    image: example/streaming:1\n  webapp:\n    image: example/web:1\n",
		"streaming/package.json": `{"name": "@x/streaming", "dependencies": {"express": "^4"}}`,
		"web/package.json":       `{"name": "web", "dependencies": {"react": "^19"}}`,
	})
	fs, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	apps := byDir(fs.Apps)
	if d := apps["streaming"].Deployed; d == nil || !d.ByName || d.Service != "streaming" || d.Prov.Note == "" {
		t.Errorf("streaming not tied by name, with a note: %+v", d)
	}
	if d := apps["web"].Deployed; d != nil {
		t.Errorf("web tied to %q by resemblance", d.Service)
	}
}

// Two services built from one directory: ambiguity is not evidence.
func TestAmbiguousBuildsTieNothing(t *testing.T) {
	root := tree(t, map[string]string{
		"docker-compose.yml":  "services:\n  api:\n    build: ./server\n  worker:\n    build: ./server\n",
		"server/package.json": `{"name": "server", "dependencies": {"express": "^4"}}`,
	})
	fs, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if d := byDir(fs.Apps)["server"].Deployed; d != nil {
		t.Errorf("one directory, two services, tied to %q", d.Service)
	}
}

func TestPythonAndOtherManifests(t *testing.T) {
	root := tree(t, map[string]string{
		"pyproject.toml":                  "[project]\nname = \"root\"\n\n[tool.uv.workspace]\nmembers = [\"api\"]\n",
		"api/pyproject.toml":              "[project]\nname = \"app\"\ndependencies = [\n    \"fastapi[standard]>=0.1\",\n    \"sqlmodel\",\n]\n",
		"tool/pyproject.toml":             "[tool.poetry]\nname = \"tool\"\n\n[tool.poetry.dependencies]\npython = \"^3.11\"\ntyper = \"^0.9\"\n",
		"mobile/pubspec.yaml":             "name: phone\ndependencies:\n  flutter:\n    sdk: flutter\n",
		"mobile/packages/ui/pubspec.yaml": "name: ui\ndependencies:\n  flutter:\n    sdk: flutter\n",
		"emails/package.json":             `{"name": "emails", "dependencies": {"react": "^19", "react-email": "^5"}}`,
	})
	apps := byDir(Apps(root))
	cases := map[string]struct {
		role      archdoc.AppRole
		framework string
	}{
		".": {archdoc.RoleWorkspace, ""}, "api": {archdoc.RoleService, "FastAPI"}, "tool": {archdoc.RoleCLI, "Typer"},
		"mobile": {archdoc.RoleMobile, "Flutter"}, "mobile/packages/ui": {archdoc.RoleLibrary, ""},
		"emails": {archdoc.RoleTooling, ""},
	}
	for dir, c := range cases {
		a := apps[dir]
		if a.Role != c.role || a.Framework != c.framework {
			t.Errorf("%s: %q %q, want %q %q (%s)", dir, a.Role, a.Framework, c.role, c.framework, a.Why)
		}
	}
	if a := apps["api"]; a.Name != "app" || a.Prov.Line != 2 || a.FrameworkProv.Line != 4 {
		t.Errorf("api cited at name line %d, framework line %d", a.Prov.Line, a.FrameworkProv.Line)
	}
}
