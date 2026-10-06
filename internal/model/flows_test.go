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

// A Python route followed into the module it imports, to the table its query names.
func TestPythonFlows(t *testing.T) {
	users, crud := "api/app/routes/users.py", "api/app/crud.py"
	fs := &archdoc.FactSet{Name: "x",
		Apps: []archdoc.App{{Name: "api", Dir: "api", Manifest: "api/pyproject.toml", Role: archdoc.RoleService, Language: "Python",
			Prov: cite("api/pyproject.toml", 1)}},
		Sources: []archdoc.Source{{App: "api", Root: "api/app", Files: []archdoc.SourceFile{
			{Path: crud, Language: "Python", Lines: 9, Classes: []archdoc.Class{{Methods: []archdoc.Method{
				{Name: "get_user", Prov: cite(crud, 3), Queries: []archdoc.Query{{Table: "User", Op: "reads", Prov: cite(crud, 4)}}}}}}},
			{Path: users, Language: "Python", Lines: 9,
				Imports: []archdoc.Import{{Spec: "app.crud", Target: crud, How: archdoc.ByModule}},
				Routers: []archdoc.Router{{Var: "router", Kind: "FastAPI"}},
				Classes: []archdoc.Class{{Methods: []archdoc.Method{{Name: "read_user", Prov: cite(users, 4),
					Decorators: []archdoc.Decorator{{Name: "router.get", Arg: "/users", HasArg: true, Prov: cite(users, 3)}},
					Invokes:    []archdoc.Invocation{{Object: "crud", Method: "get_user", Prov: cite(users, 5)}, {Object: "session", Method: "add", Prov: cite(users, 6)}}}}}}},
		}}},
	}
	m := Derive(fs)
	if len(m.Flows) != 1 || len(m.Flows[0].Steps) != 2 {
		t.Fatalf("flows %+v", m.Flows)
	}
	s := m.Flows[0].Steps
	if s[0].From != users || s[0].To != crud || s[0].Call != "get_user" || s[1].To != "table:user" || s[1].Depth != 1 {
		t.Errorf("steps %+v", s)
	}
}
