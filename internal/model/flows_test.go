package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// A route followed through the code: the handler's service, the repository the service's base
// class was given, the table its query names, and an HTTP call to a computed address.
func TestFlowsFollowTheCode(t *testing.T) {
	ctl, svc, base, repo := "server/src/controllers/album.controller.ts", "server/src/services/album.service.ts",
		"server/src/services/base.service.ts", "server/src/repositories/album.repository.ts"
	inv := func(obj, m, file string, line int) archdoc.Invocation {
		return archdoc.Invocation{Object: obj, Method: m, Prov: cite(file, line)}
	}
	fs := immichLike()
	fs.Sources[0].Files = []archdoc.SourceFile{
		{Path: ctl, Language: "TypeScript", Lines: 20, Classes: []archdoc.Class{{
			Name: "AlbumController", Prov: cite(ctl, 4),
			Decorators: []archdoc.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
			Params:     []archdoc.Param{{Name: "service", Type: "AlbumService"}},
			Methods: []archdoc.Method{{Name: "getAll", Prov: cite(ctl, 8), EndLine: 10,
				Decorators: []archdoc.Decorator{{Name: "Get", Prov: cite(ctl, 7)}},
				Invokes:    []archdoc.Invocation{inv("service", "getAll", ctl, 9)}}},
		}}},
		{Path: base, Language: "TypeScript", Lines: 10, Classes: []archdoc.Class{{
			Name: "BaseService", Params: []archdoc.Param{{Name: "albumRepository", Type: "AlbumRepository"}},
			Methods: []archdoc.Method{{Name: "requireAccess", Prov: cite(base, 5), EndLine: 6}},
		}}},
		{Path: svc, Language: "TypeScript", Lines: 30, Classes: []archdoc.Class{{
			Name: "AlbumService", Extends: []string{"BaseService"},
			Methods: []archdoc.Method{{Name: "getAll", Prov: cite(svc, 10), EndLine: 20, Invokes: []archdoc.Invocation{
				inv("", "requireAccess", svc, 11),
				inv("logger", "log", svc, 12),
				inv("albumRepository", "getAll", svc, 13),
				inv("albumRepository", "getAll", svc, 14),
				inv("cache", "get", svc, 15),
			}}},
		}}},
		{Path: repo, Language: "TypeScript", Lines: 30, Calls: []archdoc.Call{{Callee: "fetch", Target: "url", Prov: cite(repo, 7)}},
			Classes: []archdoc.Class{{Name: "AlbumRepository", Methods: []archdoc.Method{{Name: "getAll", Prov: cite(repo, 5), EndLine: 9,
				Queries: []archdoc.Query{{Table: "album", Op: "reads", Prov: cite(repo, 6)}}}}}}},
	}
	m := Derive(fs)
	if len(m.Flows) != 1 {
		t.Fatalf("flows %+v", m.Flows)
	}
	var got []string
	for _, s := range m.Flows[0].Steps {
		got = append(got, s.From+">"+s.To+":"+s.Call)
	}
	want := []string{
		"AlbumController>AlbumService:getAll",
		"AlbumService>AlbumService:requireAccess",
		"AlbumService>AlbumRepository:getAll",
		"AlbumRepository>table:album:reads",
		"AlbumRepository>unresolved:fetch",
	}
	if len(got) != len(want) {
		t.Fatalf("steps %v, want %v (logging, a repeat and an unknown field are left out)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d: %s, want %s", i, got[i], want[i])
		}
	}
	if s := m.Flows[0].Steps[2]; s.Depth != 1 || s.Prov.Line != 13 {
		t.Errorf("service → repository at depth %d, line %d", s.Depth, s.Prov.Line)
	}
}
