package model

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func cite(file string, line int) core.Provenance {
	return core.Provenance{File: file, Line: line}
}

// Immich's shape: the server is a Compose service built from server/, its code names the
// machine-learning service in a URL and redis as a host's default, serves routes under a global
// prefix, and calls one address it computes.
func immichLike() *core.FactSet {
	ctl := "server/src/controllers/album.controller.ts"
	svc := "server/src/services/album.service.ts"
	return &core.FactSet{
		Name: "x", Source: "docker-compose.yml",
		Services: []core.Service{
			{Name: "server", Image: "example/server", Evidence: core.Declared, Prov: cite("docker-compose.yml", 2)},
			{Name: "ml", Image: "example/ml", Evidence: core.Declared, Prov: cite("docker-compose.yml", 5)},
			{Name: "redis", Image: "redis:7", Evidence: core.Declared, Prov: cite("docker-compose.yml", 8)},
		},
		Apps: []core.App{{Name: "server", Dir: "server", Manifest: "server/package.json", Role: core.RoleService,
			Language: "TypeScript", Prov: cite("server/package.json", 1),
			Deployed: &core.Deployment{Service: "server", Prov: cite("docker-compose.yml", 3)}}},
		Sources: []core.Source{{App: "server", Root: "server/src", Files: []core.SourceFile{
			{Path: "server/src/main.ts", Language: "TypeScript", Lines: 5,
				Prefix: &core.Literal{Value: "api", Prov: cite("server/src/main.ts", 3)}},
			{Path: ctl, Language: "TypeScript", Lines: 20, Classes: []core.Class{{
				Name: "AlbumController", Prov: cite(ctl, 4),
				Decorators: []core.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
				Injects:    []core.Literal{{Value: "AlbumService", Prov: cite(ctl, 5)}, {Value: "Missing", Prov: cite(ctl, 5)}},
				Methods: []core.Method{{Name: "getAll", Prov: cite(ctl, 8), Decorators: []core.Decorator{
					{Name: "Get", Prov: cite(ctl, 7)},
					{Name: "Endpoint", Summary: "List all albums", SummaryProv: cite(ctl, 8), Prov: cite(ctl, 8)},
				}}},
			}}},
			{Path: svc, Language: "TypeScript", Lines: 30,
				Classes: []core.Class{{Name: "AlbumService", Prov: cite(svc, 2)}},
				Hosts: []core.HostRef{
					{Host: "ml", Port: "3003", Scheme: "http", Value: "http://ml:3003", Prov: cite(svc, 10)},
					{Host: "redis", Value: "redis", Prov: cite(svc, 11)},
					{Host: "docs.partner.io", Scheme: "https", Value: "https://docs.partner.io", Prov: cite(svc, 12)},
					{Host: "api.partner.io", Scheme: "https", Value: "https://api.partner.io", Called: true, Prov: cite(svc, 13)},
					{Host: "tiles.partner.io", Scheme: "https", Value: "https://tiles.partner.io/style.json", Key: "lightStyle", Prov: cite(svc, 14)},
					{Host: "links.partner.io", Scheme: "https", Value: "https://links.partner.io/", Key: "releaseUrl", Built: true, Prov: cite(svc, 15)},
				},
				Imports: []core.Import{{Spec: "nodemailer", Package: "nodemailer", How: core.ByPackage, Prov: cite(svc, 1)}},
				Calls:   []core.Call{{Callee: "fetch", Target: "new URL('predict', url)", Prov: cite(svc, 20)}}},
		}}},
	}
}

func TestRoutesFromControllers(t *testing.T) {
	m := Derive(immichLike())
	if len(m.Entries) != 1 {
		t.Fatalf("entries %+v", m.Entries)
	}
	e := m.Entries[0]
	if e.ID != "route:server GET /api/albums" || e.Handler != "AlbumController.getAll" || e.Summary != "List all albums" ||
		e.Container != "svc:server" || e.Component != "cmp:server/controllers" || e.PrefixProv.Line != 3 || e.Prov.Line != 7 {
		t.Errorf("entry %+v", e)
	}
	if len(e.Uses) != 2 || e.Uses[0].How != "name" || e.Uses[0].Component != "cmp:server/services" || e.Uses[0].Prov.Line != 2 || e.Uses[1].How != "unresolved" {
		t.Errorf("uses %+v", e.Uses)
	}
}

func TestEdgesFromTheCode(t *testing.T) {
	m := Derive(immichLike())
	edges := map[string]core.Edge{}
	for _, e := range m.Edges {
		edges[e.From+">"+e.To] = e
	}
	if e := edges["svc:server>svc:ml"]; e.Label != "calls" || e.Technology != "http" || !e.Traffic || e.Prov[0].Line != 10 {
		t.Errorf("server → ml: %+v", e)
	}
	if e := edges["svc:server>svc:redis"]; e.Label != "connects to" || !e.Traffic {
		t.Errorf("server → redis: %+v", e)
	}
	if _, ok := edges["svc:server>ext:docs.partner.io"]; ok {
		t.Error("a URL the code only mentions became a relationship")
	}
	if e, ok := edges["svc:server>ext:api.partner.io"]; !ok || e.Label != "calls" {
		t.Errorf("a URL the code calls: %+v", e)
	}
	if n, ok := m.Node("ext:api.partner.io"); !ok || n.Evidence != core.Referenced {
		t.Errorf("external %+v", n)
	}
	// A URL a configuration key holds is an endpoint; the same in a template file, or built, is a link.
	if e, ok := edges["svc:server>ext:tiles.partner.io"]; !ok || e.Label != "is configured to reach" {
		t.Errorf("a configured endpoint: %+v", e)
	}
	if _, ok := edges["svc:server>ext:links.partner.io"]; ok {
		t.Error("a link the code builds became a relationship")
	}
	// A client library names a system outside: read at the import, looked up in the catalog.
	if e, ok := edges["svc:server>ext:email-server-smtp"]; !ok || e.Label != "sends email through" || e.LabelProv.Origin != core.Catalog || e.Prov[0].Line != 1 {
		t.Errorf("an SDK's system: %+v", e)
	}
	if n, _ := m.Node("ext:email-server-smtp"); n.NameProv.Origin != core.Catalog || n.Evidence != core.Referenced {
		t.Errorf("the SMTP server: %+v", n)
	}
	if len(m.Unresolved) != 1 || m.Unresolved[0].What != "fetch(new URL('predict', url))" || m.Unresolved[0].Component != "cmp:server/services" {
		t.Errorf("unresolved %+v", m.Unresolved)
	}
}

// A FastAPI route's path is its router's prefix under every router that includes it, up to the
// application; a prefix written as a setting is resolved by name.
func TestFastAPIRoutes(t *testing.T) {
	items, main, cfg := "api/app/routes/items.py", "api/app/main.py", "api/app/core/config.py"
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "api", Dir: "api", Manifest: "api/pyproject.toml", Role: core.RoleService, Language: "Python",
			Prov: cite("api/pyproject.toml", 1)}},
		Sources: []core.Source{{App: "api", Root: "api/app", Files: []core.SourceFile{
			{Path: cfg, Language: "Python", Lines: 5, Classes: []core.Class{{Name: "Settings",
				Fields: []core.Field{{Name: "API_V1_STR", Type: "str", Value: "/api/v1", Prov: cite(cfg, 3)}}}}},
			{Path: items, Language: "Python", Lines: 9,
				Routers: []core.Router{{Var: "router", Kind: "APIRouter", Prefix: "/items", Prov: cite(items, 1)}},
				Classes: []core.Class{{Methods: []core.Method{{Name: "read_item", Doc: "Get item by ID", DocProv: cite(items, 5), Prov: cite(items, 4),
					Decorators: []core.Decorator{{Name: "router.get", Arg: "/{id}", HasArg: true, Prov: cite(items, 3)}}}}}}},
			{Path: main, Language: "Python", Lines: 7,
				Imports: []core.Import{{Spec: "app.routes.items", Target: items, How: core.ByModule}},
				Routers: []core.Router{{Var: "api_router", Kind: "APIRouter"}, {Var: "app", Kind: "FastAPI"}},
				Includes: []core.Include{
					{Parent: "api_router", Child: "items.router", Prov: cite(main, 3)},
					{Parent: "app", Child: "api_router", PrefixExpr: "settings.API_V1_STR", Prov: cite(main, 7)},
				}},
		}}},
	}
	m := Derive(fs)
	if len(m.Entries) != 1 {
		t.Fatalf("entries %+v", m.Entries)
	}
	e := m.Entries[0]
	if e.Method != "GET" || e.Path != "/api/v1/items/{id}" || e.Handler != "items.read_item" || e.Summary != "Get item by ID" || e.SummaryProv.Line != 5 {
		t.Errorf("entry %+v", e)
	}
	if !strings.Contains(e.PathNote, "resolved by name at api/app/core/config.py:3") {
		t.Errorf("path note %q", e.PathNote)
	}
}

// Pages where their framework declares them: SvelteKit by file, groups dropped; a +server.ts
// by its exported methods; TanStack Router by createFileRoute, layouts dropped.
func TestPages(t *testing.T) {
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "web", Dir: "web", Manifest: "web/package.json", Role: core.RoleWeb, Prov: cite("web/package.json", 1)}},
		Sources: []core.Source{{App: "web", Root: "web/src", Files: []core.SourceFile{
			{Path: "web/src/routes/+page.svelte", Language: "Svelte", Lines: 3},
			{Path: "web/src/routes/(user)/albums/[albumId=id]/+page.svelte", Language: "Svelte", Lines: 3},
			{Path: "web/src/routes/api/health/+server.ts", Language: "TypeScript", Lines: 3,
				Exports: []core.Literal{{Value: "GET", Prov: cite("web/src/routes/api/health/+server.ts", 2)}, {Value: "helper"}}},
			{Path: "web/src/lib/items.tsx", Language: "TSX", Lines: 3,
				Pages: []core.Literal{{Value: "/_layout/items", Prov: cite("web/src/lib/items.tsx", 4)}}},
		}}},
	}
	ids := map[string]core.Entry{}
	for _, e := range Derive(fs).Entries {
		ids[e.ID] = e
	}
	for _, want := range []string{"page:web /", "page:web /albums/[albumId=id]", "route:web GET /api/health", "page:web /items"} {
		if _, ok := ids[want]; !ok {
			t.Errorf("no %s in %v", want, ids)
		}
	}
	if len(ids) != 4 {
		t.Errorf("%d entries, want 4 (helper is not a method)", len(ids))
	}
	if g := ids["page:web /albums/[albumId=id]"].Group(); g != "Pages /albums" {
		t.Errorf("group %q", g)
	}
}

// Express declares routes as FastAPI does — a verb on a router, the router included under a prefix.
func TestExpressRoutes(t *testing.T) {
	api, main := "srv/src/api.js", "srv/src/index.js"
	route := func(file, dec, arg, handler string, line int) core.Method {
		return core.Method{Name: handler, Prov: cite(file, line), Decorators: []core.Decorator{{Name: dec, Arg: arg, HasArg: true, Prov: cite(file, line)}}}
	}
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "srv", Dir: "srv", Manifest: "srv/package.json", Role: core.RoleService, Prov: cite("srv/package.json", 1)}},
		Sources: []core.Source{{App: "srv", Root: "srv/src", Files: []core.SourceFile{
			{Path: api, Language: "JavaScript", Lines: 9, Routers: []core.Router{{Var: "router", Kind: "Router"}},
				Classes: []core.Class{{Methods: []core.Method{route(api, "router.get", "/users/:id", "getUser", 4), route(api, "cache.get", "/not-a-route", "x", 6)}}}},
			{Path: main, Language: "JavaScript", Lines: 9, Routers: []core.Router{{Var: "app", Kind: "express"}},
				Imports:  []core.Import{{Spec: "./api", Target: api, How: core.ByPath}},
				Includes: []core.Include{{Parent: "app", Child: "router", Prefix: "/api", Prov: cite(main, 5)}},
				Classes:  []core.Class{{Methods: []core.Method{route(main, "app.get", "/health", "handler", 7)}}}},
		}}},
	}
	ids := map[string]bool{}
	for _, e := range Derive(fs).Entries {
		ids[e.ID] = true
	}
	if !ids["route:srv GET /api/users/:id"] || !ids["route:srv GET /health"] || len(ids) != 2 {
		t.Errorf("entries %v", ids)
	}
}

// Next.js pages: app/ by page and route files, pages/ by file; React Router by the paths it is given.
func TestNextAndReactRouterPages(t *testing.T) {
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "web", Dir: "web", Manifest: "web/package.json", Role: core.RoleWeb, Prov: cite("web/package.json", 1)}},
		Sources: []core.Source{{App: "web", Root: "web/src", Framework: "Next.js", Files: []core.SourceFile{
			{Path: "web/src/app/(shop)/items/[id]/page.tsx", Language: "TSX", Lines: 3},
			{Path: "web/src/app/api/items/route.ts", Language: "TypeScript", Lines: 3, Exports: []core.Literal{{Value: "POST", Prov: cite("web/src/app/api/items/route.ts", 2)}}},
			{Path: "web/src/pages/about.tsx", Language: "TSX", Lines: 3},
			{Path: "web/src/pages/_app.tsx", Language: "TSX", Lines: 3},
			{Path: "web/src/pages/api/ping.ts", Language: "TypeScript", Lines: 3},
			{Path: "web/src/router.tsx", Language: "TSX", Lines: 3, Pages: []core.Literal{{Value: "/albums/:id", Prov: cite("web/src/router.tsx", 5)}}},
		}}},
	}
	ids := map[string]bool{}
	for _, e := range Derive(fs).Entries {
		ids[e.ID] = true
	}
	for _, want := range []string{"page:web /items/[id]", "route:web POST /api/items", "page:web /about", "route:web ANY /api/ping", "page:web /albums/:id"} {
		if !ids[want] {
			t.Errorf("no %s in %v", want, ids)
		}
	}
	if len(ids) != 5 {
		t.Errorf("%d entries, want 5 (_app is not a page): %v", len(ids), ids)
	}
}

// The other ways in: a command a class or a builder declares, a job on a queue named by an enum.
func TestCommandsAndJobs(t *testing.T) {
	cmd, svc, cli := "srv/src/commands/reset.ts", "srv/src/services/asset.service.ts", "srv/src/cli.ts"
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "srv", Dir: "srv", Manifest: "srv/package.json", Role: core.RoleService, Prov: cite("srv/package.json", 1)}},
		Sources: []core.Source{{App: "srv", Root: "srv/src", Files: []core.SourceFile{
			{Path: cmd, Language: "TypeScript", Lines: 9, Classes: []core.Class{{Name: "ResetCommand", Prov: cite(cmd, 5),
				Decorators: []core.Decorator{{Name: "Command", Options: map[string]string{"name": "reset-password", "description": "Reset the admin password"}, Prov: cite(cmd, 4)}}}}},
			{Path: svc, Language: "TypeScript", Lines: 9,
				Constants: []core.Constant{{Name: "JobName.AssetDelete", Value: "AssetDelete", Prov: cite(svc, 1)}},
				Classes: []core.Class{{Name: "AssetService", Methods: []core.Method{{Name: "handleDelete", Prov: cite(svc, 6), EndLine: 8,
					Decorators: []core.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName.AssetDelete"}, Prov: cite(svc, 5)}},
					Queries:    []core.Query{{Table: "asset", Op: "deletes", Prov: cite(svc, 7)}}}}}}},
			{Path: cli, Language: "TypeScript", Lines: 9, Commands: []core.Command{{Name: "upload", Summary: "Upload assets", Prov: cite(cli, 3)}}},
		}}},
	}
	m := Derive(fs)
	got := map[string]core.Entry{}
	for _, e := range m.Entries {
		got[e.ID] = e
	}
	if e := got["command:srv reset-password"]; e.Summary != "Reset the admin password" || e.Group() != "Commands" {
		t.Errorf("class command %+v", e)
	}
	if e := got["command:srv upload"]; e.Summary != "Upload assets" {
		t.Errorf("builder command %+v", e)
	}
	if e := got["job:srv AssetDelete"]; e.Handler != "AssetService.handleDelete" || !strings.Contains(e.PathNote, "resolved by name") {
		t.Errorf("job %+v", e)
	}
	if len(m.Flows) != 1 || m.Flows[0].Entry != "job:srv AssetDelete" || m.Flows[0].Steps[0].To != "table:asset" {
		t.Errorf("a job's flow: %+v", m.Flows)
	}
}

// A field of a record among many is not a setting, and the top-level domains reserved for examples
// name no real system.
func TestRowsAndReservedNamesAreNotEndpoints(t *testing.T) {
	f := core.SourceFile{Path: "src/config.ts", Language: "TypeScript"}
	setting := core.HostRef{Host: "api.io", Scheme: "https", Key: "url"}
	if !configured(setting, f) {
		t.Error("a URL a url key holds is configured")
	}
	setting.Row = true
	if configured(setting, f) {
		t.Error("a URL in a list of records was taken for a setting")
	}
	for host, want := range map[string]bool{"mcp.northwind.example": true, "avatars.example": true, "x.test": true,
		"api.example.net": true, "example.io": false, "supabase.com": false} {
		if placeholder(host) != want {
			t.Errorf("placeholder(%q) = %v, want %v", host, !want, want)
		}
	}
}
