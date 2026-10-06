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
		"server/src/controllers/album.controller.ts:5 ./missing":                  {How: archdoc.NoMatch},
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
		"backend/app/api/routes/items.py:4 app.nothing": {How: archdoc.NoMatch},
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

// A NestJS controller, in miniature: the class's decorators, what its constructor is given, its
// decorated methods with their summaries, and an enum member a decorator names.
func TestNestFacts(t *testing.T) {
	root := tree(t, map[string]string{
		"server/src/enum.ts": "export enum RouteKey {\n  Asset = 'assets',\n}\n",
		"server/src/controllers/asset.controller.ts": "import { RouteKey } from '../enum';\n" +
			"@Controller(RouteKey.Asset)\n" +
			"export class AssetController {\n" +
			"  constructor(private service: AssetService, private repo: Repository<Asset>) {}\n" +
			"  @Get(':id')\n" +
			"  @Endpoint({ summary: 'Get an asset' })\n" +
			"  get() {}\n" +
			"  helper() {}\n" +
			"}\n",
		"server/src/main.ts": "app.setGlobalPrefix('api', { exclude: [] });\n",
	})
	src, _ := Read(root, archdoc.App{Dir: "server", Language: "TypeScript"}, nil)
	var ctl archdoc.Class
	var consts []archdoc.Constant
	var prefix *archdoc.Literal
	for _, f := range src.Files {
		for _, c := range f.Classes {
			if c.Name == "AssetController" {
				ctl = c
			}
		}
		consts = append(consts, f.Constants...)
		if f.Prefix != nil {
			prefix = f.Prefix
		}
	}
	if len(ctl.Decorators) != 1 || ctl.Decorators[0].Name != "Controller" || ctl.Decorators[0].ArgExpr != "RouteKey.Asset" {
		t.Errorf("class decorators %+v", ctl.Decorators)
	}
	if len(ctl.Injects) != 2 || ctl.Injects[0].Value != "AssetService" || ctl.Injects[1].Value != "Repository" || ctl.Injects[0].Prov.Line != 4 {
		t.Errorf("injects %+v", ctl.Injects)
	}
	if len(ctl.Methods) != 2 || ctl.Methods[0].Name != "get" || ctl.Methods[1].Name != "helper" || len(ctl.Methods[1].Decorators) != 0 {
		t.Fatalf("methods %+v", ctl.Methods)
	}
	get := ctl.Methods[0].Decorators
	if get[0].Name != "Get" || get[0].Arg != ":id" || get[1].Summary != "Get an asset" || get[1].SummaryProv.Line != 6 {
		t.Errorf("route decorators %+v", get)
	}
	if len(consts) != 1 || consts[0].Name != "RouteKey.Asset" || consts[0].Value != "assets" {
		t.Errorf("constants %+v", consts)
	}
	if prefix == nil || prefix.Value != "api" || prefix.Prov.File != "server/src/main.ts" {
		t.Errorf("prefix %+v", prefix)
	}
}

// What the code says about the network: a URL literal, a host property's default, and calls —
// a literal target is a host, a computed one stays a call, unresolved.
func TestHostsAndCalls(t *testing.T) {
	root := tree(t, map[string]string{
		"server/src/config.ts": "export const ml = { urls: [process.env.ML_URL || 'http://immich-machine-learning:3003'] };\n" +
			"export const redis = { host: env.REDIS_HOSTNAME || 'redis', port: 6379 };\n" +
			"// http://in-a-comment:1 is not a literal\n",
		"server/src/client.ts": "await fetch('https://api.example.com/v1');\nawait fetch(new URL('predict', url));\nconst u = `http://ml:3003/${p}`;\n",
		"api/app/__init__.py":  "",
		"api/app/main.py":      "import requests\nrequests.get(\"http://search:9200/_count\")\nrequests.post(url)\nclient = Redis(host=\"cache\")\n",
	})
	src, _ := Read(root, archdoc.App{Dir: "server", Language: "TypeScript"}, nil)
	hosts := map[string]archdoc.HostRef{}
	var calls []archdoc.Call
	for _, f := range src.Files {
		for _, h := range f.Hosts {
			hosts[h.Host] = h
		}
		calls = append(calls, f.Calls...)
	}
	if h := hosts["immich-machine-learning"]; h.Scheme != "http" || h.Port != "3003" || h.Prov.Line != 1 || h.Called {
		t.Errorf("ml %+v", h)
	}
	if h := hosts["redis"]; h.Scheme != "" || h.Prov.Line != 2 {
		t.Errorf("redis %+v", h)
	}
	if h := hosts["api.example.com"]; !h.Called {
		t.Errorf("a literal fetched is called: %+v", h)
	}
	if _, ok := hosts["ml"]; !ok {
		t.Error("a template string's literal start names its host")
	}
	if _, ok := hosts["in-a-comment"]; ok {
		t.Error("a URL in a comment was read as a host")
	}
	if len(calls) != 1 || calls[0].Callee != "fetch" || calls[0].Target != "new URL('predict', url)" || calls[0].Prov.Line != 2 {
		t.Errorf("calls %+v", calls)
	}

	py, _ := Read(root, archdoc.App{Dir: "api", Name: "app", Language: "Python"}, nil)
	hosts = map[string]archdoc.HostRef{}
	calls = nil
	for _, f := range py.Files {
		for _, h := range f.Hosts {
			hosts[h.Host] = h
		}
		calls = append(calls, f.Calls...)
	}
	if !hosts["search"].Called || hosts["cache"].Prov.Line != 4 {
		t.Errorf("python hosts %+v", hosts)
	}
	if len(calls) != 1 || calls[0].Callee != "requests.post" {
		t.Errorf("python calls %+v", calls)
	}
}

// FastAPI's route decorators are read as written; the routes themselves are the Python pack's.
func TestPythonDecorators(t *testing.T) {
	root := tree(t, map[string]string{
		"api/app/__init__.py": "",
		"api/app/items.py":    "@router.get(\"/items/{id}\", summary=\"Read an item\")\nasync def read(id: int):\n    pass\n",
	})
	src, _ := Read(root, archdoc.App{Dir: "api", Name: "app", Language: "Python"}, nil)
	var m archdoc.Method
	for _, f := range src.Files {
		for _, c := range f.Classes {
			if len(c.Methods) > 0 {
				m = c.Methods[0]
			}
		}
	}
	if m.Name != "read" || len(m.Decorators) != 1 || m.Decorators[0].Name != "router.get" || m.Decorators[0].Arg != "/items/{id}" || m.Decorators[0].Summary != "Read an item" {
		t.Errorf("%+v", m)
	}
}

// Table classes: a TypeScript field with its decorators, a relation's target, literal options;
// a Python class's bases, keyword arguments and fields with their defining calls.
func TestFieldFacts(t *testing.T) {
	root := tree(t, map[string]string{
		"server/src/album.table.ts": "@Table({ name: 'album' })\n" +
			"export class AlbumTable extends Base {\n" +
			"  @ForeignKeyColumn(() => AssetTable, { nullable: true, onDelete: 'SET NULL' })\n" +
			"  thumbnailId!: string | null;\n" +
			"  @Column({ type: 'text' }) description!: string;\n" +
			"}\n",
		"api/app/__init__.py": "",
		"api/app/models.py": "class Item(ItemBase, table=True):\n" +
			"    __tablename__ = \"items\"\n" +
			"    owner_id: uuid.UUID = Field(foreign_key=\"user.id\", nullable=False)\n" +
			"    owner: User | None = Relationship(back_populates=\"items\")\n",
	})
	ts, _ := Read(root, archdoc.App{Dir: "server", Language: "TypeScript"}, nil)
	c := ts.Files[0].Classes[0]
	if c.Name != "AlbumTable" || len(c.Extends) != 1 || c.Extends[0] != "Base" || c.Decorators[0].Options["name"] != "album" {
		t.Errorf("class %+v", c)
	}
	if len(c.Fields) != 2 {
		t.Fatalf("fields %+v", c.Fields)
	}
	fk := c.Fields[0]
	if fk.Name != "thumbnailId" || fk.Type != "string | null" || fk.Decorators[0].Target != "AssetTable" || fk.Decorators[0].Options["nullable"] != "true" || fk.Prov.Line != 3 {
		t.Errorf("foreign key field %+v", fk)
	}
	if d := c.Fields[1]; d.Name != "description" || d.Decorators[0].Options["type"] != "text" {
		t.Errorf("column field %+v", d)
	}

	py, _ := Read(root, archdoc.App{Dir: "api", Name: "app", Language: "Python"}, nil)
	var item archdoc.Class
	for _, f := range py.Files {
		for _, cl := range f.Classes {
			if cl.Name == "Item" {
				item = cl
			}
		}
	}
	if len(item.Extends) != 1 || item.Extends[0] != "ItemBase" || item.Options["table"] != "True" {
		t.Errorf("class %+v", item)
	}
	fields := map[string]archdoc.Field{}
	for _, f := range item.Fields {
		fields[f.Name] = f
	}
	if fields["__tablename__"].Value != "items" {
		t.Errorf("__tablename__ %+v", fields["__tablename__"])
	}
	if f := fields["owner_id"]; f.Type != "uuid.UUID" || f.Decorators[0].Name != "Field" || f.Decorators[0].Options["foreign_key"] != "user.id" || f.Decorators[0].Options["nullable"] != "False" {
		t.Errorf("owner_id %+v", f)
	}
	if f := fields["owner"]; f.Decorators[0].Name != "Relationship" {
		t.Errorf("owner %+v", f)
	}
}

// What a method does to its own object, and the tables its queries name — what a flow follows.
func TestMethodBodies(t *testing.T) {
	root := tree(t, map[string]string{
		"server/src/album.service.ts": "export class AlbumService extends BaseService {\n" +
			"  constructor(protected albumRepository: AlbumRepository, private logger: Logger) { super() }\n" +
			"  async getAll(id: string) {\n" +
			"    await this.requireAccess(id);\n" +
			"    const rows = await this.albumRepository.getAll(id);\n" +
			"    return this.db.selectFrom('album').where('id', '=', id).execute();\n" +
			"  }\n" +
			"}\n",
	})
	src, _ := Read(root, archdoc.App{Dir: "server", Language: "TypeScript"}, nil)
	c := src.Files[0].Classes[0]
	if len(c.Params) != 2 || c.Params[0].Name != "albumRepository" || c.Params[0].Type != "AlbumRepository" {
		t.Errorf("params %+v", c.Params)
	}
	m := c.Methods[0]
	if m.Name != "getAll" || m.Prov.Line != 3 || m.EndLine != 7 {
		t.Errorf("method %s at %d–%d", m.Name, m.Prov.Line, m.EndLine)
	}
	if len(m.Invokes) != 2 || m.Invokes[0].Object != "" || m.Invokes[0].Method != "requireAccess" ||
		m.Invokes[1].Object != "albumRepository" || m.Invokes[1].Method != "getAll" || m.Invokes[1].Prov.Line != 5 {
		t.Errorf("invokes %+v", m.Invokes)
	}
	if len(m.Queries) != 1 || m.Queries[0].Table != "album" || m.Queries[0].Op != "reads" || m.Queries[0].Prov.Line != 6 {
		t.Errorf("queries %+v", m.Queries)
	}
}

// FastAPI, in miniature: routers and what includes them, a prefix written as a setting, an include
// only made in development, and a docstring.
func TestFastAPIFacts(t *testing.T) {
	root := tree(t, map[string]string{
		"api/app/__init__.py":        "",
		"api/app/routes/__init__.py": "",
		"api/app/routes/items.py":    "router = APIRouter(prefix=\"/items\")\n\n@router.get(\"/{id}\")\ndef read_item(id: int):\n    \"\"\"\n    Get item by ID.\n    \"\"\"\n",
		"api/app/main.py":            "from app.routes import items\napi_router = APIRouter()\napi_router.include_router(items.router)\nif settings.ENV == \"dev\":\n    api_router.include_router(items.router)\napp = FastAPI()\napp.include_router(api_router, prefix=settings.API_V1_STR)\n",
	})
	src, _ := Read(root, archdoc.App{Dir: "api", Name: "app", Language: "Python"}, nil)
	files := map[string]archdoc.SourceFile{}
	for _, f := range src.Files {
		files[f.Path] = f
	}
	items := files["api/app/routes/items.py"]
	if len(items.Routers) != 1 || items.Routers[0].Var != "router" || items.Routers[0].Prefix != "/items" {
		t.Errorf("routers %+v", items.Routers)
	}
	m := items.Classes[0].Methods[0]
	if m.Doc != "Get item by ID" || m.DocProv.Line != 5 {
		t.Errorf("docstring %q at %d", m.Doc, m.DocProv.Line)
	}
	main := files["api/app/main.py"]
	if len(main.Routers) != 2 || main.Routers[1].Kind != "FastAPI" {
		t.Errorf("main routers %+v", main.Routers)
	}
	if len(main.Includes) != 3 || main.Includes[0].Child != "items.router" || main.Includes[1].Condition != `settings.ENV == "dev"` ||
		main.Includes[2].PrefixExpr != "settings.API_V1_STR" {
		t.Errorf("includes %+v", main.Includes)
	}
}
