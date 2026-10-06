package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func cite(file string, line int) archdoc.Provenance {
	return archdoc.Provenance{File: file, Line: line}
}

// Immich's shape: the server is a Compose service built from server/, its code names the
// machine-learning service in a URL and redis as a host's default, serves routes under a global
// prefix, and calls one address it computes.
func immichLike() *archdoc.FactSet {
	ctl := "server/src/controllers/album.controller.ts"
	svc := "server/src/services/album.service.ts"
	return &archdoc.FactSet{
		Name: "x", Source: "docker-compose.yml",
		Services: []archdoc.Service{
			{Name: "server", Image: "example/server", Evidence: archdoc.Declared, Prov: cite("docker-compose.yml", 2)},
			{Name: "ml", Image: "example/ml", Evidence: archdoc.Declared, Prov: cite("docker-compose.yml", 5)},
			{Name: "redis", Image: "redis:7", Evidence: archdoc.Declared, Prov: cite("docker-compose.yml", 8)},
		},
		Apps: []archdoc.App{{Name: "server", Dir: "server", Manifest: "server/package.json", Role: archdoc.RoleService,
			Language: "TypeScript", Prov: cite("server/package.json", 1),
			Deployed: &archdoc.Deployment{Service: "server", Prov: cite("docker-compose.yml", 3)}}},
		Sources: []archdoc.Source{{App: "server", Root: "server/src", Files: []archdoc.SourceFile{
			{Path: "server/src/main.ts", Language: "TypeScript", Lines: 5,
				Prefix: &archdoc.Literal{Value: "api", Prov: cite("server/src/main.ts", 3)}},
			{Path: ctl, Language: "TypeScript", Lines: 20, Classes: []archdoc.Class{{
				Name: "AlbumController", Prov: cite(ctl, 4),
				Decorators: []archdoc.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
				Injects:    []archdoc.Literal{{Value: "AlbumService", Prov: cite(ctl, 5)}, {Value: "Missing", Prov: cite(ctl, 5)}},
				Methods: []archdoc.Method{{Name: "getAll", Prov: cite(ctl, 8), Decorators: []archdoc.Decorator{
					{Name: "Get", Prov: cite(ctl, 7)},
					{Name: "Endpoint", Summary: "List all albums", SummaryProv: cite(ctl, 8), Prov: cite(ctl, 8)},
				}}},
			}}},
			{Path: svc, Language: "TypeScript", Lines: 30,
				Classes: []archdoc.Class{{Name: "AlbumService", Prov: cite(svc, 2)}},
				Hosts: []archdoc.HostRef{
					{Host: "ml", Port: "3003", Scheme: "http", Value: "http://ml:3003", Prov: cite(svc, 10)},
					{Host: "redis", Value: "redis", Prov: cite(svc, 11)},
					{Host: "docs.example.com", Scheme: "https", Value: "https://docs.example.com", Prov: cite(svc, 12)},
					{Host: "api.example.com", Scheme: "https", Value: "https://api.example.com", Called: true, Prov: cite(svc, 13)},
				},
				Calls: []archdoc.Call{{Callee: "fetch", Target: "new URL('predict', url)", Prov: cite(svc, 20)}}},
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
	edges := map[string]archdoc.Edge{}
	for _, e := range m.Edges {
		edges[e.From+">"+e.To] = e
	}
	if e := edges["svc:server>svc:ml"]; e.Label != "calls" || e.Technology != "http" || !e.Traffic || e.Prov[0].Line != 10 {
		t.Errorf("server → ml: %+v", e)
	}
	if e := edges["svc:server>svc:redis"]; e.Label != "connects to" || !e.Traffic {
		t.Errorf("server → redis: %+v", e)
	}
	if _, ok := edges["svc:server>ext:docs.example.com"]; ok {
		t.Error("a URL the code only mentions became a relationship")
	}
	if e, ok := edges["svc:server>ext:api.example.com"]; !ok || e.Label != "calls" {
		t.Errorf("a URL the code calls: %+v", e)
	}
	if n, ok := m.Node("ext:api.example.com"); !ok || n.Evidence != archdoc.Referenced {
		t.Errorf("external %+v", n)
	}
	if len(m.Unresolved) != 1 || m.Unresolved[0].What != "fetch(new URL('predict', url))" || m.Unresolved[0].Component != "cmp:server/services" {
		t.Errorf("unresolved %+v", m.Unresolved)
	}
}
