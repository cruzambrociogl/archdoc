package code

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

// imports indexes every import of a source by "file:line spec".
func imports(src archdoc.Source) map[string]archdoc.Import {
	out := map[string]archdoc.Import{}
	for _, f := range src.Files {
		for _, i := range f.Imports {
			out[i.Prov.String()+" "+i.Spec] = i
		}
	}
	return out
}

func paths(src archdoc.Source) []string {
	var out []string
	for _, f := range src.Files {
		out = append(out, f.Path)
	}
	return out
}

// Immich's server, in miniature: NestJS with ESM imports and a tsconfig paths alias.
func TestTypeScriptImports(t *testing.T) {
	root := tree(t, map[string]string{
		"server/tsconfig.json": `{
  // comments and trailing commas are allowed here
  "compilerOptions": { "paths": { "src/*": ["./src/*"], }, },
}`,
		"server/src/controllers/album.controller.ts": "import { Controller } from '@nestjs/common';\n" +
			"import { AlbumService } from 'src/services/album.service';\n" +
			"import { x } from './helpers.js';\n" +
			"import { readFile } from 'node:fs/promises';\n" +
			"import './missing';\n" +
			"import data from './data.json';\n",
		"server/src/controllers/helpers.ts":               "export const x = 1;\n",
		"server/src/controllers/data.json":                "{}",
		"server/src/controllers/album.controller.spec.ts": "import { AlbumController } from './album.controller';\n",
		"server/src/services/album.service.ts":            "export class AlbumService {}\nconst m = await import('../utils');\nconst r = require('lodash');\n",
		"server/src/utils/index.ts":                       "export {};\n",
		"server/src/types.d.ts":                           "declare module 'x';\n",
		"server/test/fixtures.ts":                         "import 'src/services/album.service';\n",
	})
	src, ok := Read(root, archdoc.App{Dir: "server", Language: "TypeScript"}, nil)
	if !ok || src.Root != "server/src" {
		t.Fatalf("read %v, root %q", ok, src.Root)
	}
	want := []string{"server/src/controllers/album.controller.ts", "server/src/controllers/helpers.ts",
		"server/src/services/album.service.ts", "server/src/utils/index.ts"}
	if got := paths(src); len(got) != len(want) {
		t.Fatalf("files %v, want %v (no tests, no declarations, nothing outside src)", got, want)
	}
	cases := map[string]archdoc.Import{
		"server/src/controllers/album.controller.ts:1 @nestjs/common":             {How: archdoc.ByPackage, Package: "@nestjs/common"},
		"server/src/controllers/album.controller.ts:2 src/services/album.service": {How: archdoc.ByAlias, Target: "server/src/services/album.service.ts"},
		"server/src/controllers/album.controller.ts:3 ./helpers.js":               {How: archdoc.ByPath, Target: "server/src/controllers/helpers.ts"},
		"server/src/controllers/album.controller.ts:4 node:fs/promises":           {How: archdoc.ByPackage, Package: "node:fs"},
		"server/src/controllers/album.controller.ts:5 ./missing":                  {How: archdoc.Unresolved},
		"server/src/services/album.service.ts:2 ../utils":                         {How: archdoc.ByPath, Target: "server/src/utils/index.ts"},
		"server/src/services/album.service.ts:3 lodash":                           {How: archdoc.ByPackage, Package: "lodash"},
	}
	got := imports(src)
	for key, w := range cases {
		g, ok := got[key]
		if !ok {
			t.Errorf("no import %s", key)
			continue
		}
		if g.How != w.How || g.Target != w.Target || g.Package != w.Package {
			t.Errorf("%s: %s %q %q, want %s %q %q", key, g.How, g.Target, g.Package, w.How, w.Target, w.Package)
		}
	}
	if _, ok := got["server/src/controllers/album.controller.ts:6 ./data.json"]; ok {
		t.Error("an import of JSON was kept as a dependency between parts of the code")
	}
}

// A SvelteKit component: imports are read from its <script> blocks, cited at their line in the
// .svelte file, and the framework's own modules are the framework's, not unresolved code.
func TestSvelteComponent(t *testing.T) {
	root := tree(t, map[string]string{
		"web/svelte.config.js":        "export default {};\n",
		"web/src/lib/api.ts":          "export const get = 1;\n",
		"web/src/lib/state.svelte.ts": "export const s = $state(0);\n",
		"web/src/lib/Thing.svelte":    "<p>x</p>\n",
		"web/src/routes/+page.svelte": "<script lang=\"ts\">\n" +
			"  import { get } from '$lib/api';\n" +
			"  import { goto } from '$app/navigation';\n" +
			"  import type { PageData } from './$types';\n" +
			"  import { s } from '$lib/state.svelte';\n" +
			"  import Thing from '$lib/Thing.svelte';\n" +
			"</script>\n\n<h1>{s}</h1>\n<Thing />\n",
	})
	src, ok := Read(root, archdoc.App{Dir: "web", Language: "TypeScript", Framework: "SvelteKit"}, nil)
	if !ok {
		t.Fatal("not read")
	}
	got := imports(src)
	for key, w := range map[string]archdoc.Import{
		"web/src/routes/+page.svelte:2 $lib/api":          {How: archdoc.ByAlias, Target: "web/src/lib/api.ts"},
		"web/src/routes/+page.svelte:3 $app/navigation":   {How: archdoc.ByPackage, Package: "@sveltejs/kit"},
		"web/src/routes/+page.svelte:4 ./$types":          {How: archdoc.ByPackage, Package: "@sveltejs/kit"},
		"web/src/routes/+page.svelte:5 $lib/state.svelte": {How: archdoc.ByAlias, Target: "web/src/lib/state.svelte.ts"},
		"web/src/routes/+page.svelte:6 $lib/Thing.svelte": {How: archdoc.ByAlias, Target: "web/src/lib/Thing.svelte"},
	} {
		if g := got[key]; g.How != w.How || g.Target != w.Target || g.Package != w.Package {
			t.Errorf("%s: %+v, want %+v", key, g, w)
		}
	}
}

// The FastAPI template's backend, in miniature: the package is the source root, absolute imports
// start from the directory holding it, and `from x import a, b` imports modules a and b.
func TestPythonImports(t *testing.T) {
	root := tree(t, map[string]string{
		"backend/pyproject.toml":             "[project]\nname = \"app\"\n",
		"backend/app/__init__.py":            "",
		"backend/app/main.py":                "from fastapi import FastAPI\nfrom app.api.routes import items, users\nfrom app.core.config import settings\n",
		"backend/app/api/__init__.py":        "",
		"backend/app/api/routes/__init__.py": "",
		"backend/app/api/routes/items.py":    "from ... import crud\nfrom ..deps import SessionDep\nimport app.models as m\nfrom app.nothing import x\n",
		"backend/app/api/routes/users.py":    "",
		"backend/app/api/deps.py":            "",
		"backend/app/core/__init__.py":       "",
		"backend/app/core/config.py":         "",
		"backend/app/crud.py":                "",
		"backend/app/models.py":              "",
		"backend/tests/test_items.py":        "from app import crud\n",
	})
	src, ok := Read(root, archdoc.App{Dir: "backend", Name: "app", Language: "Python"}, nil)
	if !ok || src.Root != "backend/app" {
		t.Fatalf("read %v, root %q", ok, src.Root)
	}
	got := imports(src)
	for key, w := range map[string]archdoc.Import{
		"backend/app/main.py:1 fastapi":                 {How: archdoc.ByPackage, Package: "fastapi"},
		"backend/app/main.py:2 app.api.routes.items":    {How: archdoc.ByModule, Target: "backend/app/api/routes/items.py"},
		"backend/app/main.py:2 app.api.routes.users":    {How: archdoc.ByModule, Target: "backend/app/api/routes/users.py"},
		"backend/app/main.py:3 app.core.config":         {How: archdoc.ByModule, Target: "backend/app/core/config.py"},
		"backend/app/api/routes/items.py:1 ...crud":     {How: archdoc.ByPath, Target: "backend/app/crud.py"},
		"backend/app/api/routes/items.py:2 ..deps":      {How: archdoc.ByPath, Target: "backend/app/api/deps.py"},
		"backend/app/api/routes/items.py:3 app.models":  {How: archdoc.ByModule, Target: "backend/app/models.py"},
		"backend/app/api/routes/items.py:4 app.nothing": {How: archdoc.Unresolved},
	} {
		g, ok := got[key]
		if !ok {
			t.Errorf("no import %s", key)
			continue
		}
		if g.How != w.How || g.Target != w.Target || g.Package != w.Package {
			t.Errorf("%s: %+v, want %+v", key, g, w)
		}
	}
	for _, p := range paths(src) {
		if p == "backend/tests/test_items.py" {
			t.Error("a test was read as part of the application")
		}
	}
}

// A package inside another application's directory is that package's code, not the host's.
func TestNestedApplicationsAreNotRead(t *testing.T) {
	root := tree(t, map[string]string{
		"mobile/lib/main.ts":          "import './screen';\n",
		"mobile/lib/screen.ts":        "",
		"mobile/packages/ui/index.ts": "",
	})
	src, _ := Read(root, archdoc.App{Dir: "mobile", Language: "TypeScript"}, []string{"mobile", "mobile/packages/ui"})
	for _, p := range paths(src) {
		if p == "mobile/packages/ui/index.ts" {
			t.Error("a nested application was read as part of its host")
		}
	}
}

func TestLanguagesNotReadYet(t *testing.T) {
	if _, ok := Read(t.TempDir(), archdoc.App{Dir: "mobile", Language: "Dart"}, nil); ok {
		t.Error("Dart was read")
	}
}
