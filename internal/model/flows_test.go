package model

import (
	"sort"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// A route followed through the code: the handler's service, the repository the service's base
// class was given, the table its query names, and an HTTP call to a computed address.
func TestFlowsFollowTheCode(t *testing.T) {
	ctl, svc, base, repo := "server/src/controllers/album.controller.ts", "server/src/services/album.service.ts",
		"server/src/services/base.service.ts", "server/src/repositories/album.repository.ts"
	inv := func(obj, m, file string, line int) core.Invocation {
		return core.Invocation{Object: obj, Method: m, Prov: cite(file, line)}
	}
	fs := immichLike()
	fs.Sources[0].Files = []core.SourceFile{
		{Path: ctl, Language: "TypeScript", Lines: 20, Classes: []core.Class{{
			Name: "AlbumController", Prov: cite(ctl, 4),
			Decorators: []core.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
			Params:     []core.Param{{Name: "service", Type: "AlbumService"}},
			Methods: []core.Method{{Name: "getAll", Prov: cite(ctl, 8), EndLine: 10,
				Decorators: []core.Decorator{{Name: "Get", Prov: cite(ctl, 7)}},
				Invokes:    []core.Invocation{inv("service", "getAll", ctl, 9)}}},
		}}},
		{Path: base, Language: "TypeScript", Lines: 10, Classes: []core.Class{{
			Name: "BaseService", Params: []core.Param{{Name: "albumRepository", Type: "AlbumRepository"}},
			Methods: []core.Method{{Name: "requireAccess", Prov: cite(base, 5), EndLine: 6}},
		}}},
		{Path: svc, Language: "TypeScript", Lines: 30, Classes: []core.Class{{
			Name: "AlbumService", Extends: []string{"BaseService"},
			Methods: []core.Method{{Name: "getAll", Prov: cite(svc, 10), EndLine: 20, Invokes: []core.Invocation{
				inv("", "requireAccess", svc, 11),
				inv("logger", "log", svc, 12),
				inv("albumRepository", "getAll", svc, 13),
				inv("albumRepository", "getAll", svc, 14),
				inv("cache", "get", svc, 15),
			}}},
		}}},
		{Path: repo, Language: "TypeScript", Lines: 30, Calls: []core.Call{{Callee: "fetch", Target: "url", Prov: cite(repo, 7)}},
			Classes: []core.Class{{Name: "AlbumRepository", Methods: []core.Method{{Name: "getAll", Prov: cite(repo, 5), EndLine: 9,
				Queries: []core.Query{{Table: "album", Op: "reads", Prov: cite(repo, 6)}}}}}}},
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
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "api", Dir: "api", Manifest: "api/pyproject.toml", Role: core.RoleService, Language: "Python",
			Prov: cite("api/pyproject.toml", 1)}},
		Sources: []core.Source{{App: "api", Root: "api/app", Files: []core.SourceFile{
			{Path: crud, Language: "Python", Lines: 9, Classes: []core.Class{{Methods: []core.Method{
				{Name: "get_user", Prov: cite(crud, 3), Queries: []core.Query{{Table: "User", Op: "reads", Prov: cite(crud, 4)}}}}}}},
			{Path: users, Language: "Python", Lines: 9,
				Imports: []core.Import{{Spec: "app.crud", Target: crud, How: core.ByModule}},
				Routers: []core.Router{{Var: "router", Kind: "FastAPI"}},
				Classes: []core.Class{{Methods: []core.Method{{Name: "read_user", Prov: cite(users, 4),
					Decorators: []core.Decorator{{Name: "router.get", Arg: "/users", HasArg: true, Prov: cite(users, 3)}},
					Invokes:    []core.Invocation{{Object: "crud", Method: "get_user", Prov: cite(users, 5)}, {Object: "session", Method: "add", Prov: cite(users, 6)}}}}}}},
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
	fs.Sources[0].Files = []core.SourceFile{
		{Path: ctl, Language: "TypeScript", Lines: 20,
			Imports: []core.Import{{Spec: "src/utils/database", Target: db}, {Spec: "src/utils/misc", Target: util}},
			Classes: []core.Class{{Name: "AlbumController", Prov: cite(ctl, 4),
				Decorators: []core.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
				Methods: []core.Method{{Name: "getAll", Prov: cite(ctl, 8), EndLine: 12,
					Decorators: []core.Decorator{{Name: "Get", Prov: cite(ctl, 7)}},
					Invokes: []core.Invocation{
						{Method: "format", Free: true, Prov: cite(ctl, 9)},
						{Method: "searchBuilder", Free: true, Prov: cite(ctl, 10)},
						{Method: "fetch", Free: true, Prov: cite(ctl, 11)},
					}}}}}},
		{Path: db, Language: "TypeScript", Lines: 9, Classes: []core.Class{{Methods: []core.Method{{Name: "searchBuilder", Prov: cite(db, 2), EndLine: 6,
			Queries: []core.Query{{Table: "album", Op: "reads", Prov: cite(db, 3)}}}}}}},
		{Path: util, Language: "TypeScript", Lines: 9, Classes: []core.Class{{Methods: []core.Method{{Name: "format", Prov: cite(util, 2), EndLine: 4}}}}},
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
	fs.Sources[0].Files = []core.SourceFile{
		{Path: enum, Language: "TypeScript", Lines: 5, Constants: []core.Constant{{Name: "JobName.NotifyAlbum", Value: "NotifyAlbum", Prov: cite(enum, 2)}}},
		{Path: ctl, Language: "TypeScript", Lines: 20, Classes: []core.Class{{
			Name: "AlbumController", Prov: cite(ctl, 4),
			Decorators: []core.Decorator{{Name: "Controller", Arg: "albums", HasArg: true, Prov: cite(ctl, 3)}},
			Params:     []core.Param{{Name: "service", Type: "AlbumService"}},
			Methods: []core.Method{{Name: "invite", Prov: cite(ctl, 8), EndLine: 10,
				Decorators: []core.Decorator{{Name: "Put", Prov: cite(ctl, 7)}},
				Invokes:    []core.Invocation{{Object: "service", Method: "invite", Prov: cite(ctl, 9)}}}},
		}}},
		{Path: svc, Language: "TypeScript", Lines: 30, Classes: []core.Class{{
			Name: "AlbumService", Params: []core.Param{{Name: "eventRepository", Type: "EventRepository"}, {Name: "socket", Type: "Socket"}},
			Methods: []core.Method{{Name: "invite", Prov: cite(svc, 10), EndLine: 20, Invokes: []core.Invocation{
				{Object: "eventRepository", Method: "emit", Args: []string{"AlbumInvite"}, Prov: cite(svc, 11)},
				{Object: "socket", Method: "serverSend", Args: []string{"AlbumInvite"}, Prov: cite(svc, 12)},
			}}},
		}}},
		{Path: note, Language: "TypeScript", Lines: 40, Classes: []core.Class{{
			Name: "NotificationService",
			Methods: []core.Method{
				{Name: "onAlbumInvite", Prov: cite(note, 10), EndLine: 14,
					Decorators: []core.Decorator{{Name: "OnEvent", Options: map[string]string{"name": "AlbumInvite"}, Prov: cite(note, 9)}},
					Named:      []core.Mention{{Expr: "JobName.NotifyAlbum", Prov: cite(note, 12)}, {Value: "NotAJob", Prov: cite(note, 13)}}},
				{Name: "handleNotify", Prov: cite(note, 20), EndLine: 24,
					Decorators: []core.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName.NotifyAlbum"}, Prov: cite(note, 19)}},
					Queries:    []core.Query{{Table: "album", Op: "reads", Prov: cite(note, 21)}}},
			},
		}}},
	}
	m := Derive(fs)
	var route core.Flow
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
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "api", Dir: "api", Manifest: "api/pyproject.toml", Role: core.RoleService, Language: "Python",
			Prov: cite("api/pyproject.toml", 1)}},
		Sources: []core.Source{{App: "api", Root: "api/app", Files: []core.SourceFile{
			{Path: repo, Language: "Python", Lines: 9, Classes: []core.Class{{Name: "UserRepository", Prov: cite(repo, 1), Methods: []core.Method{
				{Name: "add", Prov: cite(repo, 3), Queries: []core.Query{{Table: "User", Op: "reads", Prov: cite(repo, 4)}}}}}}},
			{Path: svc, Language: "Python", Lines: 20, Classes: []core.Class{{Name: "UserService", Prov: cite(svc, 1),
				Params: []core.Param{{Name: "repo", Type: "Optional[UserRepository]"}},
				Methods: []core.Method{
					{Name: "create", Prov: cite(svc, 5), Invokes: []core.Invocation{
						{Self: true, Method: "check", Prov: cite(svc, 6)},
						{Self: true, Object: "repo", Method: "add", Prov: cite(svc, 7)},
						{Self: true, Object: "clock", Method: "now", Prov: cite(svc, 8)}}},
					{Name: "check", Prov: cite(svc, 10)}}}}},
			{Path: route, Language: "Python", Lines: 9,
				Routers: []core.Router{{Var: "router", Kind: "FastAPI"}},
				Classes: []core.Class{{Methods: []core.Method{{Name: "create_user", Prov: cite(route, 4),
					Decorators: []core.Decorator{{Name: "router.post", Arg: "/users", HasArg: true, Prov: cite(route, 3)}},
					Invokes: []core.Invocation{
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
	sql := core.SourceFile{Path: "api/migrations/001.sql", Language: "SQL", Lines: 9, Classes: []core.Class{
		{Name: "users", Prov: cite("api/migrations/001.sql", 1), Decorators: []core.Decorator{{Name: "Table", Arg: "users", HasArg: true}},
			Fields: []core.Field{{Name: "id", Prov: cite("api/migrations/001.sql", 2),
				Decorators: []core.Decorator{{Name: "Column", Options: map[string]string{"type": "serial", "primary": "true"}}}}}},
		{Name: "posts", Prov: cite("api/migrations/001.sql", 4), Decorators: []core.Decorator{{Name: "Table", Arg: "posts", HasArg: true}},
			Fields: []core.Field{{Name: "author_id", Prov: cite("api/migrations/001.sql", 5),
				Decorators: []core.Decorator{{Name: "Column", Target: "users", Options: map[string]string{"type": "int"}}}}}},
	}}
	facts := func(code []core.SourceFile) *core.FactSet {
		return &core.FactSet{Name: "x",
			Apps:    []core.App{{Name: "api", Dir: "api", Manifest: "api/go.mod", Role: core.RoleService, Language: "Go", Prov: cite("api/go.mod", 1)}},
			Sources: []core.Source{{App: "api", Root: "api", Files: code, Schemas: []core.SourceFile{sql}}}}
	}
	tablesOf := func(m core.Model) (names []string, refs int) {
		for _, n := range m.Nodes {
			if n.Kind == core.Table {
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
		if n.Kind == core.Application && (n.Technology != "Go" || !n.TechProv.Known()) {
			t.Errorf("technology %q, cited at %v", n.Technology, n.TechProv)
		}
	}
	names, refs := tablesOf(Derive(facts(nil)))
	if strings.Join(names, " ") != "posts users" && strings.Join(names, " ") != "users posts" || refs != 1 {
		t.Errorf("from migrations alone: tables %v, %d foreign keys", names, refs)
	}
	declared := []core.SourceFile{{Path: "api/src/account.ts", Language: "TypeScript", Lines: 5, Classes: []core.Class{
		{Name: "Account", Prov: cite("api/src/account.ts", 2), Decorators: []core.Decorator{{Name: "Entity", Prov: cite("api/src/account.ts", 1)}}}}}}
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
		core.App{Name: "web", Dir: "web", Manifest: "web/package.json", Role: core.RoleWeb, Framework: "React", Language: "TypeScript",
			FrameworkProv: cite("web/package.json", 4), Prov: cite("web/package.json", 1),
			Requires: []core.Requirement{{Name: "@x/sdk", Prov: cite("web/package.json", 6)}}},
		core.App{Name: "@x/sdk", Dir: "packages/sdk", Manifest: "packages/sdk/package.json", Role: core.RoleLibrary, Prov: cite("packages/sdk/package.json", 1)},
		core.App{Name: "app", Dir: "mobile", Manifest: "mobile/pubspec.yaml", Role: core.RoleMobile, Framework: "Flutter", Language: "Dart",
			FrameworkProv: cite("mobile/pubspec.yaml", 3), Prov: cite("mobile/pubspec.yaml", 1)},
		core.App{Name: "admin", Dir: "admin", Manifest: "admin/package.json", Role: core.RoleWeb, Framework: "React", Language: "TypeScript",
			FrameworkProv: cite("admin/package.json", 4), Prov: cite("admin/package.json", 1),
			Requires: []core.Requirement{{Name: "@x/sdk", Dev: true, Prov: cite("admin/package.json", 9)}}},
	)
	fs.APIs = []core.API{{File: "open-api/spec.json", Prov: cite("open-api/spec.json", 3),
		Operations: []core.Operation{{Method: "GET", Path: "/albums"}, {Method: "POST", Path: "/albums"}},
		Clients: []core.APIClient{
			{App: "packages/sdk", Dir: "packages/sdk/src", How: "its code names 2 of the 2 paths", Prov: cite("packages/sdk/src/client.ts", 1)},
			{App: "mobile", Dir: "mobile/generated/openapi", How: "a client generated into mobile/generated/openapi", Prov: cite("open-api/generate.sh", 3)},
		}}}
	m := Derive(fs)
	got := map[string]core.Edge{}
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
	fs.APIs[0].Operations = []core.Operation{{Method: "GET", Path: "/payments"}, {Method: "GET", Path: "/refunds"}}
	for _, e := range Derive(fs).Edges {
		if e.Label == "calls the API of" {
			t.Errorf("an API nobody here serves drew %s → %s", e.From, e.To)
		}
	}
}

func keysOf(m map[string]core.Edge) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// A project with no manifest: the root application takes its name, a script is a command, an HTML
// file is a page, and a hosted backend its code is a client of is drawn. A React app's src/pages
// folder is not Next's: its router says which are pages.
func TestSmallProjects(t *testing.T) {
	script := &core.FactSet{Name: "tool",
		Apps: []core.App{{Name: "My Tool", Dir: ".", Manifest: "report.py", Role: core.RoleCLI, Language: "Python", Loose: true, Prov: cite("report.py", 1)}},
		Sources: []core.Source{{App: ".", Root: ".", Loose: true, Files: []core.SourceFile{
			{Path: "report.py", Language: "Python", Lines: 9, Main: &core.Provenance{File: "report.py", Line: 8}}}}}}
	m := Derive(script)
	if _, ok := m.Node("app:my-tool"); !ok {
		t.Errorf("the root application is not named after itself: %+v", m.Nodes)
	}
	if len(m.Entries) != 1 || m.Entries[0].ID != "command:my-tool python report.py" || m.Entries[0].Prov.Line != 8 {
		t.Errorf("entries %+v", m.Entries)
	}

	web := &core.FactSet{Name: "notes",
		Apps: []core.App{{Name: "notes", Dir: ".", Manifest: "package.json", Role: core.RoleWeb, Framework: "React", Language: "TypeScript",
			FrameworkProv: cite("package.json", 3), Prov: cite("package.json", 1)}},
		Sources: []core.Source{{App: ".", Root: "src", Framework: "React", Documents: []core.Literal{{Value: "Notes", Prov: cite("index.html", 4)}},
			Files: []core.SourceFile{
				{Path: "src/main.tsx", Language: "TSX", Lines: 5, Pages: []core.Literal{{Value: "/", Prov: cite("src/main.tsx", 4)}}},
				{Path: "src/pages/NotesPage.tsx", Language: "TSX", Lines: 3},
				{Path: "src/supabase.ts", Language: "TypeScript", Lines: 2, Imports: []core.Import{
					{Spec: "@supabase/supabase-js", Package: "@supabase/supabase-js", How: core.ByPackage, Prov: cite("src/supabase.ts", 1)}}},
			}}}}
	m = Derive(web)
	var pages []string
	for _, e := range m.Entries {
		pages = append(pages, e.ID)
	}
	if strings.Join(pages, " | ") != "page:notes / | page:notes /index.html" {
		t.Errorf("pages %v — the router's page and the document, not the file in src/pages", pages)
	}
	if _, ok := m.Node("ext:supabase"); !ok {
		t.Errorf("the hosted backend is not drawn: %+v", m.Nodes)
	}
}
