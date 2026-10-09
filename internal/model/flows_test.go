package model

import (
	"sort"
	"strings"
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

// A plain function call is followed into its module when it leads to a table, and left out when
// it is only a helper.
func TestFlowsFollowFunctionsThatLeadSomewhere(t *testing.T) {
	ctl, db, util := "server/src/controllers/album.controller.ts", "server/src/utils/database.ts", "server/src/utils/misc.ts"
	fs := immichLike()
	fs.Sources[0].Files = []archdoc.SourceFile{
		{Path: ctl, Language: "TypeScript", Lines: 20,
			Imports: []archdoc.Import{{Spec: "src/utils/database", Target: db}, {Spec: "src/utils/misc", Target: util}},
			Classes: []archdoc.Class{{Name: "AlbumController", Prov: cite(ctl, 4),
				Decorators: []archdoc.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
				Methods: []archdoc.Method{{Name: "getAll", Prov: cite(ctl, 8), EndLine: 12,
					Decorators: []archdoc.Decorator{{Name: "Get", Prov: cite(ctl, 7)}},
					Invokes: []archdoc.Invocation{
						{Method: "format", Free: true, Prov: cite(ctl, 9)},
						{Method: "searchBuilder", Free: true, Prov: cite(ctl, 10)},
						{Method: "fetch", Free: true, Prov: cite(ctl, 11)},
					}}}}}},
		{Path: db, Language: "TypeScript", Lines: 9, Classes: []archdoc.Class{{Methods: []archdoc.Method{{Name: "searchBuilder", Prov: cite(db, 2), EndLine: 6,
			Queries: []archdoc.Query{{Table: "album", Op: "reads", Prov: cite(db, 3)}}}}}}},
		{Path: util, Language: "TypeScript", Lines: 9, Classes: []archdoc.Class{{Methods: []archdoc.Method{{Name: "format", Prov: cite(util, 2), EndLine: 4}}}}},
	}
	m := Derive(fs)
	if len(m.Flows) != 1 {
		t.Fatalf("flows %+v", m.Flows)
	}
	f := m.Flows[0]
	if len(f.Steps) != 2 || f.Steps[0].To != db || f.Steps[0].Call != "searchBuilder" || f.Steps[1].To != "table:album" {
		t.Errorf("steps %+v", f.Steps)
	}
	if len(f.Participants) != 3 || f.Participants[1].Kind != "module" || f.Participants[1].Name != "database" {
		t.Errorf("participants %+v", f.Participants)
	}
}

// An event is followed into the methods that listen for it — only from a call that emits — and a
// job put on a queue is a step to that job, whose handler has a flow of its own.
func TestFlowsFollowEventsAndQueuedJobs(t *testing.T) {
	ctl, svc, note, enum := "server/src/controllers/album.controller.ts", "server/src/services/album.service.ts",
		"server/src/services/notification.service.ts", "server/src/enum.ts"
	fs := immichLike()
	fs.Sources[0].Files = []archdoc.SourceFile{
		{Path: enum, Language: "TypeScript", Lines: 5, Constants: []archdoc.Constant{{Name: "JobName.NotifyAlbum", Value: "NotifyAlbum", Prov: cite(enum, 2)}}},
		{Path: ctl, Language: "TypeScript", Lines: 20, Classes: []archdoc.Class{{
			Name: "AlbumController", Prov: cite(ctl, 4),
			Decorators: []archdoc.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
			Params:     []archdoc.Param{{Name: "service", Type: "AlbumService"}},
			Methods: []archdoc.Method{{Name: "invite", Prov: cite(ctl, 8), EndLine: 10,
				Decorators: []archdoc.Decorator{{Name: "Put", Prov: cite(ctl, 7)}},
				Invokes:    []archdoc.Invocation{{Object: "service", Method: "invite", Prov: cite(ctl, 9)}}}},
		}}},
		{Path: svc, Language: "TypeScript", Lines: 30, Classes: []archdoc.Class{{
			Name: "AlbumService", Params: []archdoc.Param{{Name: "eventRepository", Type: "EventRepository"}, {Name: "socket", Type: "Socket"}},
			Methods: []archdoc.Method{{Name: "invite", Prov: cite(svc, 10), EndLine: 20, Invokes: []archdoc.Invocation{
				{Object: "eventRepository", Method: "emit", Args: []string{"AlbumInvite"}, Prov: cite(svc, 11)},
				{Object: "socket", Method: "serverSend", Args: []string{"AlbumInvite"}, Prov: cite(svc, 12)},
			}}},
		}}},
		{Path: note, Language: "TypeScript", Lines: 40, Classes: []archdoc.Class{{
			Name: "NotificationService",
			Methods: []archdoc.Method{
				{Name: "onAlbumInvite", Prov: cite(note, 10), EndLine: 14,
					Decorators: []archdoc.Decorator{{Name: "OnEvent", Options: map[string]string{"name": "AlbumInvite"}, Prov: cite(note, 9)}},
					Named:      []archdoc.Mention{{Expr: "JobName.NotifyAlbum", Prov: cite(note, 12)}, {Value: "NotAJob", Prov: cite(note, 13)}}},
				{Name: "handleNotify", Prov: cite(note, 20), EndLine: 24,
					Decorators: []archdoc.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName.NotifyAlbum"}, Prov: cite(note, 19)}},
					Queries:    []archdoc.Query{{Table: "album", Op: "reads", Prov: cite(note, 21)}}},
			},
		}}},
	}
	m := Derive(fs)
	var route archdoc.Flow
	for _, f := range m.Flows {
		if strings.HasPrefix(f.Entry, "route:") {
			route = f
		}
	}
	var got []string
	for _, s := range route.Steps {
		got = append(got, s.From+">"+s.To+":"+s.Call)
	}
	want := []string{
		"AlbumController>AlbumService:invite",
		"AlbumService>NotificationService:onAlbumInvite",
		"NotificationService>job:NotifyAlbum:queues",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("steps %v, want %v", got, want)
	}
	if !strings.Contains(route.Steps[1].Note, "AlbumInvite") || !strings.Contains(route.Steps[2].Note, "NotificationService.handleNotify") {
		t.Errorf("notes %q, %q", route.Steps[1].Note, route.Steps[2].Note)
	}
	last := route.Participants[len(route.Participants)-1]
	if last.Kind != "job" || last.Element != "job:server NotifyAlbum" {
		t.Errorf("the job's lifeline %+v", last)
	}
	if len(m.Flows) != 2 {
		t.Errorf("%d flows, want the route's and the job's own", len(m.Flows))
	}
}

// A Python route followed through a class: a typed parameter's method, the method it calls on
// itself, a field __init__ gave it, down to a table.
func TestPythonFlowsFollowClassMethods(t *testing.T) {
	route, svc, repo := "api/app/routes/users.py", "api/app/service.py", "api/app/repo.py"
	fs := &archdoc.FactSet{Name: "x",
		Apps: []archdoc.App{{Name: "api", Dir: "api", Manifest: "api/pyproject.toml", Role: archdoc.RoleService, Language: "Python",
			Prov: cite("api/pyproject.toml", 1)}},
		Sources: []archdoc.Source{{App: "api", Root: "api/app", Files: []archdoc.SourceFile{
			{Path: repo, Language: "Python", Lines: 9, Classes: []archdoc.Class{{Name: "UserRepository", Prov: cite(repo, 1), Methods: []archdoc.Method{
				{Name: "add", Prov: cite(repo, 3), Queries: []archdoc.Query{{Table: "User", Op: "reads", Prov: cite(repo, 4)}}}}}}},
			{Path: svc, Language: "Python", Lines: 20, Classes: []archdoc.Class{{Name: "UserService", Prov: cite(svc, 1),
				Params: []archdoc.Param{{Name: "repo", Type: "Optional[UserRepository]"}},
				Methods: []archdoc.Method{
					{Name: "create", Prov: cite(svc, 5), Invokes: []archdoc.Invocation{
						{Self: true, Method: "check", Prov: cite(svc, 6)},
						{Self: true, Object: "repo", Method: "add", Prov: cite(svc, 7)},
						{Self: true, Object: "clock", Method: "now", Prov: cite(svc, 8)}}},
					{Name: "check", Prov: cite(svc, 10)}}}}},
			{Path: route, Language: "Python", Lines: 9,
				Routers: []archdoc.Router{{Var: "router", Kind: "FastAPI"}},
				Classes: []archdoc.Class{{Methods: []archdoc.Method{{Name: "create_user", Prov: cite(route, 4),
					Decorators: []archdoc.Decorator{{Name: "router.post", Arg: "/users", HasArg: true, Prov: cite(route, 3)}},
					Invokes: []archdoc.Invocation{
						{Object: "service", Method: "create", Type: "Annotated[UserService, Depends()]", Prov: cite(route, 5)},
						{Object: "session", Method: "commit", Prov: cite(route, 6)}}}}}}},
		}}},
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
		route + ">UserService:create",
		"UserService>UserService:check",
		"UserService>UserRepository:add",
		"UserRepository>table:user:reads",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("steps %v, want %v", got, want)
	}
}

// Migrations are the schema only where the code declares no table.
func TestMigrationsStandInForDeclaredTables(t *testing.T) {
	sql := archdoc.SourceFile{Path: "api/migrations/001.sql", Language: "SQL", Lines: 9, Classes: []archdoc.Class{
		{Name: "users", Prov: cite("api/migrations/001.sql", 1), Decorators: []archdoc.Decorator{{Name: "Table", Arg: "users", HasArg: true}},
			Fields: []archdoc.Field{{Name: "id", Prov: cite("api/migrations/001.sql", 2),
				Decorators: []archdoc.Decorator{{Name: "Column", Options: map[string]string{"type": "serial", "primary": "true"}}}}}},
		{Name: "posts", Prov: cite("api/migrations/001.sql", 4), Decorators: []archdoc.Decorator{{Name: "Table", Arg: "posts", HasArg: true}},
			Fields: []archdoc.Field{{Name: "author_id", Prov: cite("api/migrations/001.sql", 5),
				Decorators: []archdoc.Decorator{{Name: "Column", Target: "users", Options: map[string]string{"type": "int"}}}}}},
	}}
	facts := func(code []archdoc.SourceFile) *archdoc.FactSet {
		return &archdoc.FactSet{Name: "x",
			Apps:    []archdoc.App{{Name: "api", Dir: "api", Manifest: "api/go.mod", Role: archdoc.RoleService, Language: "Go", Prov: cite("api/go.mod", 1)}},
			Sources: []archdoc.Source{{App: "api", Root: "api", Files: code, Schemas: []archdoc.SourceFile{sql}}}}
	}
	tablesOf := func(m archdoc.Model) (names []string, refs int) {
		for _, n := range m.Nodes {
			if n.Kind == archdoc.Table {
				names = append(names, n.Name)
			}
		}
		for _, e := range m.Edges {
			if e.Label == "references" {
				refs++
			}
		}
		return names, refs
	}
	// A language with no framework recognised still cites what says so: the manifest.
	for _, n := range Derive(facts(nil)).Nodes {
		if n.Kind == archdoc.Application && (n.Technology != "Go" || !n.TechProv.Known()) {
			t.Errorf("technology %q, cited at %v", n.Technology, n.TechProv)
		}
	}
	names, refs := tablesOf(Derive(facts(nil)))
	if strings.Join(names, " ") != "posts users" && strings.Join(names, " ") != "users posts" || refs != 1 {
		t.Errorf("from migrations alone: tables %v, %d foreign keys", names, refs)
	}
	declared := []archdoc.SourceFile{{Path: "api/src/account.ts", Language: "TypeScript", Lines: 5, Classes: []archdoc.Class{
		{Name: "Account", Prov: cite("api/src/account.ts", 2), Decorators: []archdoc.Decorator{{Name: "Entity", Prov: cite("api/src/account.ts", 1)}}}}}}
	names, _ = tablesOf(Derive(facts(declared)))
	if strings.Join(names, " ") != "Account" {
		t.Errorf("with a declared table: %v, want only what the code declares", names)
	}
}

// An API description ties its clients to the container whose routes it describes: the application
// that holds a client, and the one that depends on the package that does — and nobody else. The
// person reaches the applications a person runs.
func TestAPIClientsCallTheContainerTheDocumentDescribes(t *testing.T) {
	fs := immichLike()
	fs.Apps = append(fs.Apps,
		archdoc.App{Name: "web", Dir: "web", Manifest: "web/package.json", Role: archdoc.RoleWeb, Framework: "React", Language: "TypeScript",
			FrameworkProv: cite("web/package.json", 4), Prov: cite("web/package.json", 1),
			Requires: []archdoc.Requirement{{Name: "@x/sdk", Prov: cite("web/package.json", 6)}}},
		archdoc.App{Name: "@x/sdk", Dir: "packages/sdk", Manifest: "packages/sdk/package.json", Role: archdoc.RoleLibrary, Prov: cite("packages/sdk/package.json", 1)},
		archdoc.App{Name: "app", Dir: "mobile", Manifest: "mobile/pubspec.yaml", Role: archdoc.RoleMobile, Framework: "Flutter", Language: "Dart",
			FrameworkProv: cite("mobile/pubspec.yaml", 3), Prov: cite("mobile/pubspec.yaml", 1)},
		archdoc.App{Name: "admin", Dir: "admin", Manifest: "admin/package.json", Role: archdoc.RoleWeb, Framework: "React", Language: "TypeScript",
			FrameworkProv: cite("admin/package.json", 4), Prov: cite("admin/package.json", 1),
			Requires: []archdoc.Requirement{{Name: "@x/sdk", Dev: true, Prov: cite("admin/package.json", 9)}}},
	)
	fs.APIs = []archdoc.API{{File: "open-api/spec.json", Prov: cite("open-api/spec.json", 3),
		Operations: []archdoc.Operation{{Method: "GET", Path: "/albums"}, {Method: "POST", Path: "/albums"}},
		Clients: []archdoc.APIClient{
			{App: "packages/sdk", Dir: "packages/sdk/src", How: "its code names 2 of the 2 paths", Prov: cite("packages/sdk/src/client.ts", 1)},
			{App: "mobile", Dir: "mobile/generated/openapi", How: "a client generated into mobile/generated/openapi", Prov: cite("open-api/generate.sh", 3)},
		}}}
	m := Derive(fs)
	got := map[string]archdoc.Edge{}
	for _, e := range m.Edges {
		if e.Label == "calls the API of" || e.From == "actor:user" {
			got[e.From+" → "+e.To] = e
		}
	}
	for _, want := range []string{"app:web → svc:server", "app:mobile → svc:server", "actor:user → app:web", "actor:user → app:mobile", "actor:user → app:admin"} {
		if _, ok := got[want]; !ok {
			t.Errorf("no %s among %v", want, keysOf(got))
		}
	}
	if _, ok := got["app:admin → svc:server"]; ok {
		t.Error("a development dependency the code never imports drew a call")
	}
	web := got["app:web → svc:server"]
	// Three lines hold it up: the dependency, the client, the document — the first saying how.
	said := false
	for _, p := range web.Prov {
		said = said || p.File == "web/package.json" && p.Line == 6 && strings.Contains(p.Note, "1 of its 2 operations")
	}
	if len(web.Prov) != 3 || !said {
		t.Errorf("the web app's call cites %+v", web.Prov)
	}

	// A document no container's routes match draws nothing.
	fs.APIs[0].Operations = []archdoc.Operation{{Method: "GET", Path: "/payments"}, {Method: "GET", Path: "/refunds"}}
	for _, e := range Derive(fs).Edges {
		if e.Label == "calls the API of" {
			t.Errorf("an API nobody here serves drew %s → %s", e.From, e.To)
		}
	}
}

func keysOf(m map[string]archdoc.Edge) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
