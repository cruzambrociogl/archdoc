package model

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// A route followed through the code: the handler's service, the repository the service's base
// class was given, the table its query names, and an HTTP call to a computed address. A call the
// service makes on itself is kept when it leads to another class, and left out when it only shapes data.
func TestFlowsFollowTheCode(t *testing.T) {
	ctl, svc, base, repo := "server/src/controllers/album.controller.ts", "server/src/services/album.service.ts",
		"server/src/services/base.service.ts", "server/src/repositories/album.repository.ts"
	access := "server/src/repositories/access.repository.ts"
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
			Name: "BaseService", Params: []core.Param{{Name: "albumRepository", Type: "AlbumRepository"}, {Name: "accessRepository", Type: "AccessRepository"}},
			Methods: []core.Method{{Name: "requireAccess", Prov: cite(base, 5), EndLine: 7,
				Invokes: []core.Invocation{inv("accessRepository", "check", base, 6)}}},
		}}},
		{Path: access, Language: "TypeScript", Lines: 10, Classes: []core.Class{{
			Name: "AccessRepository", Methods: []core.Method{{Name: "check", Prov: cite(access, 3), EndLine: 5}},
		}}},
		{Path: svc, Language: "TypeScript", Lines: 30, Classes: []core.Class{{
			Name: "AlbumService", Extends: []string{"BaseService"},
			Methods: []core.Method{{Name: "getAll", Prov: cite(svc, 10), EndLine: 20, Invokes: []core.Invocation{
				inv("", "requireAccess", svc, 11),
				inv("logger", "log", svc, 12),
				inv("albumRepository", "getAll", svc, 13),
				inv("albumRepository", "getAll", svc, 14),
				inv("cache", "get", svc, 15),
				inv("", "format", svc, 16),
			}}, {Name: "format", Prov: cite(svc, 22), EndLine: 24}},
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
		"AlbumService>AccessRepository:check",
		"AlbumService>AlbumRepository:getAll",
		"AlbumRepository>table:album:reads",
		"AlbumRepository>unresolved:fetch",
	}
	if len(got) != len(want) {
		t.Fatalf("steps %v, want %v (logging, a repeat, an unknown field and a helper that leads nowhere are left out)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d: %s, want %s", i, got[i], want[i])
		}
	}
	if s := m.Flows[0].Steps[3]; s.Depth != 1 || s.Prov.Line != 13 {
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

// What a job leads to when it is done: a switch on job names whose case for a job queues another ends
// that job's flow, even when the job's handler queues nothing itself. A case for something that is not a
// job, and a job that is not one of this code's, add nothing.
func TestJobsQueuedWhenAJobIsDone(t *testing.T) {
	enum, media, jobs := "srv/src/enum.ts", "srv/src/services/media.service.ts", "srv/src/services/job.service.ts"
	onJob := func(name string, line int) []core.Decorator {
		return []core.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName." + name}, Prov: cite(media, line)}}
	}
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "srv", Dir: "srv", Manifest: "srv/package.json", Role: core.RoleService, Prov: cite("srv/package.json", 1)}},
		Sources: []core.Source{{App: "srv", Root: "srv/src", Files: []core.SourceFile{
			{Path: enum, Language: "TypeScript", Lines: 5, Constants: []core.Constant{
				{Name: "JobName.Thumbnails", Value: "Thumbnails", Prov: cite(enum, 2)},
				{Name: "JobName.FaceDetect", Value: "FaceDetect", Prov: cite(enum, 3)},
				{Name: "JobName.SmartSearch", Value: "SmartSearch", Prov: cite(enum, 4)}}},
			{Path: media, Language: "TypeScript", Lines: 30, Classes: []core.Class{{Name: "MediaService", Methods: []core.Method{
				{Name: "thumbnails", Prov: cite(media, 6), EndLine: 8, Decorators: onJob("Thumbnails", 5)},
				{Name: "faces", Prov: cite(media, 11), EndLine: 13, Decorators: onJob("FaceDetect", 10),
					Queries: []core.Query{{Table: "face", Op: "writes", Prov: cite(media, 12)}}},
				{Name: "search", Prov: cite(media, 16), EndLine: 18, Decorators: onJob("SmartSearch", 15)}}}}},
			{Path: jobs, Language: "TypeScript", Lines: 30, Classes: []core.Class{{Name: "JobService", Methods: []core.Method{
				{Name: "onDone", Prov: cite(jobs, 10), EndLine: 25, Named: []core.Mention{
					{Expr: "JobName.FaceDetect", Case: []string{"JobName.Thumbnails"}, Prov: cite(jobs, 13)},
					{Expr: "JobName.SmartSearch", Case: []string{"JobName.Thumbnails"}, Prov: cite(jobs, 14)},
					{Expr: "JobName.FaceDetect", Case: []string{"JobName.Thumbnails"}, Prov: cite(jobs, 15)},
					{Expr: "JobName.Thumbnails", Case: []string{"Upload"}, Prov: cite(jobs, 18)},
					{Value: "Elsewhere", Case: []string{"JobName.SmartSearch"}, Prov: cite(jobs, 21)}}}}}}},
		}}},
	}
	m := Derive(fs)
	flows := map[string]core.Flow{}
	for _, f := range m.Flows {
		flows[f.Entry] = f
	}
	thumbs, ok := flows["job:srv Thumbnails"]
	if !ok {
		t.Fatalf("a job whose handler queues nothing has no flow, though its case queues two: %v", keysOfFlows(flows))
	}
	var got []string
	for _, s := range thumbs.Steps {
		got = append(got, fmt.Sprintf("%s>%s:%s@%d", s.From, s.To, s.Call, s.Prov.Line))
	}
	if want := "JobService>job:FaceDetect:queues@13 JobService>job:SmartSearch:queues@14"; strings.Join(got, " ") != want {
		t.Errorf("steps %v, want %s (each job once)", got, want)
	}
	if n := thumbs.Steps[0].Note; !strings.Contains(n, "case for JobName.Thumbnails of JobService.onDone") || !strings.Contains(n, "MediaService.faces") {
		t.Errorf("note %q", n)
	}
	if _, ok := flows["job:srv SmartSearch"]; ok {
		t.Error("a job that is not this code's was queued after SmartSearch")
	}
}

func keysOfFlows(m map[string]core.Flow) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// A flow cut for length still shows where it leads: the event its handler emits last, and the job the
// listener queues, are kept past the cut. A call too deep to follow that plainly leads nowhere — a class
// parsing a number for itself — is not drawn, and does not cut the flow.
func TestCutFlowsKeepWhereTheyLead(t *testing.T) {
	enum, svc, repo, store := "srv/src/enum.ts", "srv/src/services/metadata.service.ts", "srv/src/repositories/asset.repository.ts", "srv/src/services/storage.service.ts"
	var calls []core.Invocation
	var methods []core.Method
	for i := 0; i < 45; i++ {
		name := fmt.Sprintf("step%d", i)
		calls = append(calls, core.Invocation{Object: "assetRepository", Method: name, Prov: cite(svc, 10+i)})
		methods = append(methods, core.Method{Name: name, Prov: cite(repo, 10+i), EndLine: 10 + i})
	}
	calls = append(calls, core.Invocation{Object: "eventRepository", Method: "emit", Args: []string{"MetadataExtracted"}, Prov: cite(svc, 60)})
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "srv", Dir: "srv", Manifest: "srv/package.json", Role: core.RoleService, Prov: cite("srv/package.json", 1)}},
		Sources: []core.Source{{App: "srv", Root: "srv/src", Files: []core.SourceFile{
			{Path: enum, Language: "TypeScript", Lines: 5, Constants: []core.Constant{
				{Name: "JobName.Extract", Value: "Extract", Prov: cite(enum, 2)}, {Name: "JobName.Move", Value: "Move", Prov: cite(enum, 3)}}},
			{Path: svc, Language: "TypeScript", Lines: 70, Classes: []core.Class{{Name: "MetadataService",
				Params: []core.Param{{Name: "assetRepository", Type: "AssetRepository"}, {Name: "eventRepository", Type: "EventRepository"}},
				Methods: []core.Method{{Name: "extract", Prov: cite(svc, 9), EndLine: 61, Invokes: calls,
					Decorators: []core.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName.Extract"}, Prov: cite(svc, 8)}}}}}}},
			{Path: repo, Language: "TypeScript", Lines: 70, Classes: []core.Class{{Name: "AssetRepository", Methods: methods}}},
			{Path: store, Language: "TypeScript", Lines: 30, Classes: []core.Class{{Name: "StorageService", Methods: []core.Method{
				{Name: "onExtracted", Prov: cite(store, 6), EndLine: 8, Named: []core.Mention{{Expr: "JobName.Move", Prov: cite(store, 7)}},
					Decorators: []core.Decorator{{Name: "OnEvent", Options: map[string]string{"name": "MetadataExtracted"}, Prov: cite(store, 5)}}},
				{Name: "move", Prov: cite(store, 11), EndLine: 13,
					Decorators: []core.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName.Move"}, Prov: cite(store, 10)}}}}}}},
		}}},
	}
	var extract core.Flow
	for _, f := range Derive(fs).Flows {
		if f.Entry == "job:srv Extract" {
			extract = f
		}
	}
	n := len(extract.Steps)
	if !extract.Cut || n < 2 {
		t.Fatalf("a flow of 45 calls and an event: cut %v, %d steps", extract.Cut, n)
	}
	tail := extract.Steps[n-2].From + ">" + extract.Steps[n-2].To + ":" + extract.Steps[n-2].Call + " " +
		extract.Steps[n-1].From + ">" + extract.Steps[n-1].To + ":" + extract.Steps[n-1].Call
	if tail != "MetadataService>StorageService:onExtracted StorageService>job:Move:queues" || !strings.Contains(extract.Steps[n-1].Note, "past the cut") {
		t.Errorf("the cut flow ends %s (%q)", tail, extract.Steps[n-1].Note)
	}

	// Four calls deep, a class calls itself: a parser that calls nothing of this code is left out.
	chain := []core.SourceFile{}
	for i, c := range []string{"A", "B", "C", "D"} {
		file := fmt.Sprintf("srv/src/%s.ts", strings.ToLower(c))
		meth := core.Method{Name: "run", Prov: cite(file, 3), EndLine: 6}
		params := []core.Param{}
		if i < 3 {
			next := []string{"B", "C", "D"}[i]
			params = append(params, core.Param{Name: "next", Type: next})
			meth.Invokes = []core.Invocation{{Object: "next", Method: "run", Prov: cite(file, 4)}}
		} else {
			meth.Invokes = []core.Invocation{{Method: "parseRate", Prov: cite(file, 4)}, {Method: "save", Prov: cite(file, 5)}}
		}
		cls := core.Class{Name: c, Params: params, Methods: []core.Method{meth}}
		if c == "D" {
			cls.Methods = append(cls.Methods,
				core.Method{Name: "parseRate", Prov: cite(file, 8), EndLine: 9, Invokes: []core.Invocation{{Method: "parseInt", Free: true, Prov: cite(file, 9)}}},
				core.Method{Name: "save", Prov: cite(file, 11), EndLine: 12, Queries: []core.Query{{Table: "rate", Op: "writes", Prov: cite(file, 12)}}})
		}
		if c == "A" {
			cls.Methods[0].Decorators = []core.Decorator{{Name: "OnJob", Exprs: map[string]string{"name": "JobName.Move"}, Prov: cite(file, 2)}}
		}
		chain = append(chain, core.SourceFile{Path: file, Language: "TypeScript", Lines: 12, Classes: []core.Class{cls}})
	}
	fs.Sources[0].Files = append([]core.SourceFile{fs.Sources[0].Files[0]}, chain...)
	var got []string
	var cut bool
	for _, f := range Derive(fs).Flows {
		if f.Entry == "job:srv Move" {
			cut = f.Cut
			for _, s := range f.Steps {
				got = append(got, s.From+">"+s.To+":"+s.Call)
			}
		}
	}
	if want := "A>B:run B>C:run C>D:run D>D:save"; strings.Join(got, " ") != want || !cut {
		t.Errorf("steps %v (cut %v), want %s, cut by the call too deep to tell", got, cut, want)
	}
}

// A command a class runs is followed from its run method, as a route is from its handler; one a
// builder declares names no method, and has no flow.
func TestFlowsFollowCommands(t *testing.T) {
	cmd, cli, repo, main := "srv/src/commands/reset.ts", "srv/src/services/cli.service.ts", "srv/src/repositories/user.repository.ts", "srv/src/cli.ts"
	fs := &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "srv", Dir: "srv", Manifest: "srv/package.json", Role: core.RoleService, Prov: cite("srv/package.json", 1)}},
		Sources: []core.Source{{App: "srv", Root: "srv/src", Files: []core.SourceFile{
			{Path: cmd, Language: "TypeScript", Lines: 12, Classes: []core.Class{{Name: "ResetCommand", Prov: cite(cmd, 5),
				Decorators: []core.Decorator{{Name: "Command", Options: map[string]string{"name": "reset-password"}, Prov: cite(cmd, 4)}},
				Params:     []core.Param{{Name: "service", Type: "CliService"}},
				Methods: []core.Method{{Name: "run", Prov: cite(cmd, 8), EndLine: 10,
					Invokes: []core.Invocation{{Object: "service", Method: "resetPassword", Prov: cite(cmd, 9)}}}}}}},
			{Path: cli, Language: "TypeScript", Lines: 12, Classes: []core.Class{{Name: "CliService",
				Params: []core.Param{{Name: "userRepository", Type: "UserRepository"}},
				Methods: []core.Method{{Name: "resetPassword", Prov: cite(cli, 5), EndLine: 7,
					Invokes: []core.Invocation{{Object: "userRepository", Method: "update", Prov: cite(cli, 6)}}}}}}},
			{Path: repo, Language: "TypeScript", Lines: 12, Classes: []core.Class{{Name: "UserRepository",
				Methods: []core.Method{{Name: "update", Prov: cite(repo, 3), EndLine: 5,
					Queries: []core.Query{{Table: "user", Op: "updates", Prov: cite(repo, 4)}}}}}}},
			{Path: main, Language: "TypeScript", Lines: 9, Commands: []core.Command{{Name: "upload", Prov: cite(main, 3)}}},
		}}},
	}
	m := Derive(fs)
	if len(m.Flows) != 1 || m.Flows[0].Entry != "command:srv reset-password" {
		t.Fatalf("flows %+v, want the class command's alone", m.Flows)
	}
	var got []string
	for _, s := range m.Flows[0].Steps {
		got = append(got, s.From+">"+s.To+":"+s.Call)
	}
	if want := "ResetCommand>CliService:resetPassword CliService>UserRepository:update UserRepository>table:user:updates"; strings.Join(got, " ") != want {
		t.Errorf("steps %v, want %s", got, want)
	}
}

// A page calls the routes whose operations it imports from a generated client: the client names each
// function after an operationId, and the operation is a route the server declares. Called in a
// function, the step cites the call; imported only, the import. A name that is no operation, and a
// component the page uses, add nothing.
func TestPagesCallRoutesThroughAGeneratedClient(t *testing.T) {
	ctl := "server/src/controllers/album.controller.ts"
	page, load, comp := "web/src/routes/albums/+page.svelte", "web/src/routes/albums/+page.ts", "web/src/lib/AlbumCard.svelte"
	fs := immichLike()
	fs.Sources[0].Files[1].Classes[0].Methods = append(fs.Sources[0].Files[1].Classes[0].Methods,
		core.Method{Name: "create", Prov: cite(ctl, 12), Decorators: []core.Decorator{{Name: "Post", Prov: cite(ctl, 11)}}})
	fs.Apps = append(fs.Apps,
		core.App{Name: "web", Dir: "web", Manifest: "web/package.json", Role: core.RoleWeb, Framework: "SvelteKit", Language: "TypeScript",
			FrameworkProv: cite("web/package.json", 4), Prov: cite("web/package.json", 1),
			Requires: []core.Requirement{{Name: "@x/sdk", Prov: cite("web/package.json", 6)}}},
		core.App{Name: "@x/sdk", Dir: "packages/sdk", Manifest: "packages/sdk/package.json", Role: core.RoleLibrary, Prov: cite("packages/sdk/package.json", 1)})
	fs.APIs = []core.API{{File: "open-api/spec.json", Prov: cite("open-api/spec.json", 3),
		Operations: []core.Operation{
			{Method: "GET", Path: "/albums", ID: "getAllAlbums", Prov: cite("open-api/spec.json", 7)},
			{Method: "POST", Path: "/albums", ID: "createAlbum", Prov: cite("open-api/spec.json", 12)},
			{Method: "DELETE", Path: "/albums/{id}", ID: "deleteAlbum", Prov: cite("open-api/spec.json", 20)}},
		Clients: []core.APIClient{{App: "packages/sdk", Dir: "packages/sdk/src", How: "its code names 2 of the 2 paths", Prov: cite("packages/sdk/src/client.ts", 1)}}}}
	sdk := func(file string, line int, names ...string) core.Import {
		return core.Import{Spec: "@x/sdk", Package: "@x/sdk", How: core.ByPackage, Names: names, Prov: cite(file, line)}
	}
	fs.Sources = append(fs.Sources, core.Source{App: "web", Root: "web/src", Framework: "SvelteKit", Files: []core.SourceFile{
		{Path: comp, Language: "TypeScript", Lines: 9, Imports: []core.Import{sdk(comp, 2, "deleteAlbum")}},
		{Path: page, Language: "TypeScript", Lines: 30,
			Imports: []core.Import{sdk(page, 2, "createAlbum", "AlbumResponseDto"), {Spec: "$lib/AlbumCard.svelte", Target: comp, How: core.ByAlias, Prov: cite(page, 3)}},
			Classes: []core.Class{{Methods: []core.Method{{Name: "add", Prov: cite(page, 8), EndLine: 10,
				Invokes: []core.Invocation{{Method: "createAlbum", Free: true, Prov: cite(page, 9)}}}}}}},
		{Path: load, Language: "TypeScript", Lines: 9, Imports: []core.Import{sdk(load, 1, "getAllAlbums")}},
	}})
	m := Derive(fs)
	var flow core.Flow
	for _, f := range m.Flows {
		if f.Entry == "page:web /albums" {
			flow = f
		}
	}
	var got []string
	for _, s := range flow.Steps {
		got = append(got, fmt.Sprintf("%s>%s:%s@%s", s.From, s.To, s.Call, s.Prov))
	}
	want := []string{
		page + ">route:server POST /api/albums:createAlbum@" + page + ":9",
		page + ">route:server GET /api/albums:getAllAlbums@" + load + ":1",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("steps %v, want %v", got, want)
	}
	if n := flow.Steps[1].Note; !strings.HasPrefix(n, "imported here from @x/sdk") || !strings.Contains(n, "open-api/spec.json:7") {
		t.Errorf("an imported call's note %q", n)
	}
	if p := flow.Participants; len(p) != 3 || p[0].Name != "/albums" || p[1].Kind != "route" || p[1].Element != "route:server POST /api/albums" {
		t.Errorf("participants %+v", p)
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
