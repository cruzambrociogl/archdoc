package model

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func tableFacts(lang string, files map[string][]core.Class) *core.FactSet {
	src := core.Source{App: "api", Root: "api/src"}
	for _, path := range []string{"api/src/a", "api/src/b"} {
		for p, cs := range files {
			if p[:len(path)] == path {
				src.Files = append(src.Files, core.SourceFile{Path: p, Language: lang, Lines: 10, Classes: cs})
			}
		}
	}
	return &core.FactSet{Name: "x",
		Apps: []core.App{{Name: "api", Dir: "api", Manifest: "api/package.json", Role: core.RoleService,
			Prov: cite("api/package.json", 1)}},
		Sources: []core.Source{src}}
}

func columns(t *testing.T, m core.Model, id string) map[string]core.Column {
	t.Helper()
	n, ok := m.Node(id)
	if !ok || n.Kind != core.Table {
		t.Fatalf("no table %s", id)
	}
	out := map[string]core.Column{}
	for _, c := range n.Columns {
		out[c.Name] = c
	}
	return out
}

// Immich's sql-tools, TypeORM's shape: @Table with a name, a column per decorated field, and a
// foreign key naming its class.
func TestTablesFromDecoratedClasses(t *testing.T) {
	f := "api/src/a/album.table.ts"
	g := "api/src/b/asset.table.ts"
	m := Derive(tableFacts("TypeScript", map[string][]core.Class{
		f: {{Name: "AlbumTable", Prov: cite(f, 3),
			Decorators: []core.Decorator{{Name: "Table", Options: map[string]string{"name": "album"}}},
			Fields: []core.Field{
				{Name: "id", Type: "Generated<string>", Prov: cite(f, 5), Decorators: []core.Decorator{{Name: "PrimaryGeneratedColumn"}}},
				{Name: "thumbnailId", Type: "string | null", Prov: cite(f, 7), Decorators: []core.Decorator{{Name: "ForeignKeyColumn", Target: "AssetTable"}}},
				{Name: "description", Type: "string | null", Prov: cite(f, 9), Decorators: []core.Decorator{{Name: "Column", Options: map[string]string{"type": "text", "nullable": "true"}}}},
				{Name: "helper", Type: "string"},
			}}},
		g: {{Name: "AssetTable", Prov: cite(g, 2), Decorators: []core.Decorator{{Name: "Entity", Arg: "asset", HasArg: true}},
			Fields: []core.Field{{Name: "id", Type: "string", Prov: cite(g, 4), Decorators: []core.Decorator{{Name: "PrimaryColumn"}}}}}},
	}))
	cols := columns(t, m, "tbl:api/album")
	if c := cols["id"]; !c.Primary || c.Type != "string" {
		t.Errorf("id %+v", c)
	}
	if c := cols["thumbnailId"]; c.References != "tbl:api/asset" || !c.Nullable || c.Prov.Line != 7 {
		t.Errorf("thumbnailId %+v", c)
	}
	if c := cols["description"]; c.Type != "text" || !c.Nullable {
		t.Errorf("description %+v", c)
	}
	if _, ok := cols["helper"]; ok {
		t.Error("an undecorated field became a column")
	}
	view := m.Data("app:api")
	if len(view.Nodes) != 2 || len(view.Edges) != 1 || view.Edges[0].From != "tbl:api/album" {
		t.Errorf("data view: %d tables, edges %+v", len(view.Nodes), view.Edges)
	}
	if len(m.Container().Nodes) != 1 || len(m.Component("app:api").Nodes) == 0 {
		t.Error("tables leaked into the container view, or components went missing")
	}
}

// SQLModel: table=True, columns inherited from the model it extends, a foreign key by table name.
func TestSQLModelTables(t *testing.T) {
	f := "api/src/a/models.py"
	field := func(name, typ string, line int, opts map[string]string, call string) core.Field {
		fl := core.Field{Name: name, Type: typ, Prov: cite(f, line)}
		if call != "" {
			fl.Decorators = []core.Decorator{{Name: call, Options: opts}}
		}
		return fl
	}
	m := Derive(tableFacts("Python", map[string][]core.Class{f: {
		{Name: "UserBase", Extends: []string{"SQLModel"}, Fields: []core.Field{field("email", "EmailStr", 2, map[string]string{"unique": "True"}, "Field")}},
		{Name: "User", Extends: []string{"UserBase"}, Options: map[string]string{"table": "True"}, Prov: cite(f, 5), Fields: []core.Field{
			field("id", "uuid.UUID", 6, map[string]string{"primary_key": "True"}, "Field"),
			field("items", `list["Item"]`, 7, nil, "Relationship"),
		}},
		{Name: "Item", Options: map[string]string{"table": "True"}, Prov: cite(f, 10), Fields: []core.Field{
			field("owner_id", "uuid.UUID | None", 11, map[string]string{"foreign_key": "user.id"}, "Field"),
		}},
	}}))
	user := columns(t, m, "tbl:api/user")
	if _, ok := user["email"]; !ok || !user["id"].Primary {
		t.Errorf("user %+v", user)
	}
	if _, ok := user["items"]; ok {
		t.Error("a Relationship became a column")
	}
	if c := columns(t, m, "tbl:api/item")["owner_id"]; c.References != "tbl:api/user" || !c.Nullable {
		t.Errorf("owner_id %+v", c)
	}
	if _, ok := m.Node("tbl:api/userbase"); ok {
		t.Error("a model that is not a table became one")
	}
}

// A Prisma schema's models, read beside the code, are tables like any other.
func TestPrismaTables(t *testing.T) {
	f := "api/prisma/schema.prisma"
	fs := tableFacts("TypeScript", map[string][]core.Class{"api/src/a/index.ts": nil})
	col := func(name string, line int, opts map[string]string, target string) core.Field {
		return core.Field{Name: name, Type: opts["type"], Prov: cite(f, line), Decorators: []core.Decorator{{Name: "Column", Options: opts, Target: target}}}
	}
	fs.Sources[0].Schemas = []core.SourceFile{{Path: f, Language: "Prisma", Classes: []core.Class{
		{Name: "User", Prov: cite(f, 1), Decorators: []core.Decorator{{Name: "Table"}}, Fields: []core.Field{col("id", 2, map[string]string{"type": "Int", "primary": "true"}, "")}},
		{Name: "Post", Prov: cite(f, 6), Decorators: []core.Decorator{{Name: "Table", Arg: "posts", HasArg: true}}, Fields: []core.Field{col("authorId", 9, map[string]string{"type": "Int"}, "User")}},
	}}}
	m := Derive(fs)
	if c := columns(t, m, "tbl:api/posts")["authorId"]; c.References != "tbl:api/User" || c.Type != "Int" {
		t.Errorf("authorId %+v", c)
	}
	if c := columns(t, m, "tbl:api/User")["id"]; !c.Primary {
		t.Errorf("id %+v", c)
	}
}

// One schema that several programs read is one set of tables: a Go module's migrations, which its
// api and its worker both reach. They go to the database the programs keep them in; each program's
// queries still find them by name.
func TestSharedSchemaIsDrawnOnce(t *testing.T) {
	file := "db/migrations/001.sql"
	sql := core.SourceFile{Path: file, Language: "SQL", Lines: 9, Classes: []core.Class{
		{Name: "users", Prov: cite(file, 1), Decorators: []core.Decorator{{Name: "Table", Arg: "users", HasArg: true}},
			Fields: []core.Field{{Name: "id", Prov: cite(file, 2),
				Decorators: []core.Decorator{{Name: "Column", Options: map[string]string{"type": "bigint", "primary": "true"}}}}}},
		{Name: "posts", Prov: cite(file, 4), Decorators: []core.Decorator{{Name: "Table", Arg: "posts", HasArg: true}},
			Fields: []core.Field{{Name: "author_id", Prov: cite(file, 5),
				Decorators: []core.Decorator{{Name: "Column", Target: "users", Options: map[string]string{"type": "bigint"}}}}}},
	}}
	code := func(dir string) []core.SourceFile {
		p := dir + "/main.go"
		return []core.SourceFile{{Path: p, Language: "Go", Lines: 5, Funcs: []core.Func{{Key: dir + "#main", Prov: cite(p, 1),
			Queries: []core.Query{{Table: "users", Op: "reads", Prov: cite(p, 3)}}}}}}
	}
	facts := func(store bool) *core.FactSet {
		fs := &core.FactSet{Name: "x",
			Apps: []core.App{
				{Name: "api", Dir: "cmd/api", Manifest: "go.mod", Role: core.RoleService, Language: "Go", Prov: cite("go.mod", 1)},
				{Name: "worker", Dir: "cmd/worker", Manifest: "go.mod", Role: core.RoleService, Language: "Go", Prov: cite("go.mod", 1)},
			},
			Sources: []core.Source{
				{App: "cmd/api", Root: ".", Module: "x", Files: code("cmd/api"), Schemas: []core.SourceFile{sql}},
				{App: "cmd/worker", Root: ".", Module: "x", Files: code("cmd/worker"), Schemas: []core.SourceFile{sql}},
			}}
		if store {
			fs.Services = []core.Service{
				{Name: "worker", Evidence: core.Declared, Prov: at(3), Endpoints: []core.Endpoint{
					{Var: "DATABASE_URL", Scheme: "postgres", Host: "db", Port: 5432, Prov: at(5)}}},
				{Name: "db", Image: "postgres:18-alpine", Evidence: core.Declared, Prov: at(8)},
			}
			fs.Apps[1].Deployed = &core.Deployment{Service: "worker", Prov: at(3)}
		}
		return fs
	}
	for _, c := range []struct {
		store bool
		owner string
	}{{true, "svc:db"}, {false, "app:cmd/api"}} {
		m := Derive(facts(c.store))
		_, local, _ := strings.Cut(c.owner, ":")
		var tables []string
		for _, n := range m.Nodes {
			if n.Kind == core.Table {
				tables = append(tables, n.ID)
				if n.Parent != c.owner {
					t.Errorf("%s in %s, want %s", n.ID, n.Parent, c.owner)
				}
			}
		}
		if len(tables) != 2 {
			t.Errorf("store %v: tables %v, want users and posts once", c.store, tables)
		}
		refs := 0
		for _, e := range m.Edges {
			if e.Label == "references" {
				refs++
				if e.From != "tbl:"+local+"/posts" || e.To != "tbl:"+local+"/users" {
					t.Errorf("foreign key %s → %s", e.From, e.To)
				}
			}
		}
		if refs != 1 {
			t.Errorf("store %v: %d foreign keys, want 1", c.store, refs)
		}
		if cols := columns(t, m, "tbl:"+local+"/posts"); cols["author_id"].References != "tbl:"+local+"/users" {
			t.Errorf("author_id references %q", cols["author_id"].References)
		}
		readers := map[string]bool{}
		for _, a := range m.Access {
			if a.Element != "tbl:"+local+"/users" {
				t.Errorf("%s's query of %s found %q", a.Container, a.Table, a.Element)
			}
			readers[a.Container] = true
		}
		if len(readers) != 2 {
			t.Errorf("store %v: queries from %v, want both programs", c.store, readers)
		}
		if len(m.Datas()) != 1 || m.Datas()[0] != c.owner {
			t.Errorf("store %v: data views %v, want one, of %s", c.store, m.Datas(), c.owner)
		}
	}
}
