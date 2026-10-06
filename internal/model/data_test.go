package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func tableFacts(lang string, files map[string][]archdoc.Class) *archdoc.FactSet {
	src := archdoc.Source{App: "api", Root: "api/src"}
	for _, path := range []string{"api/src/a", "api/src/b"} {
		for p, cs := range files {
			if p[:len(path)] == path {
				src.Files = append(src.Files, archdoc.SourceFile{Path: p, Language: lang, Lines: 10, Classes: cs})
			}
		}
	}
	return &archdoc.FactSet{Name: "x",
		Apps: []archdoc.App{{Name: "api", Dir: "api", Manifest: "api/package.json", Role: archdoc.RoleService,
			Prov: cite("api/package.json", 1)}},
		Sources: []archdoc.Source{src}}
}

func columns(t *testing.T, m archdoc.Model, id string) map[string]archdoc.Column {
	t.Helper()
	n, ok := m.Node(id)
	if !ok || n.Kind != archdoc.Table {
		t.Fatalf("no table %s", id)
	}
	out := map[string]archdoc.Column{}
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
	m := Derive(tableFacts("TypeScript", map[string][]archdoc.Class{
		f: {{Name: "AlbumTable", Prov: cite(f, 3),
			Decorators: []archdoc.Decorator{{Name: "Table", Options: map[string]string{"name": "album"}}},
			Fields: []archdoc.Field{
				{Name: "id", Type: "Generated<string>", Prov: cite(f, 5), Decorators: []archdoc.Decorator{{Name: "PrimaryGeneratedColumn"}}},
				{Name: "thumbnailId", Type: "string | null", Prov: cite(f, 7), Decorators: []archdoc.Decorator{{Name: "ForeignKeyColumn", Target: "AssetTable"}}},
				{Name: "description", Type: "string | null", Prov: cite(f, 9), Decorators: []archdoc.Decorator{{Name: "Column", Options: map[string]string{"type": "text", "nullable": "true"}}}},
				{Name: "helper", Type: "string"},
			}}},
		g: {{Name: "AssetTable", Prov: cite(g, 2), Decorators: []archdoc.Decorator{{Name: "Entity", Arg: "asset", HasArg: true}},
			Fields: []archdoc.Field{{Name: "id", Type: "string", Prov: cite(g, 4), Decorators: []archdoc.Decorator{{Name: "PrimaryColumn"}}}}}},
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
	field := func(name, typ string, line int, opts map[string]string, call string) archdoc.Field {
		fl := archdoc.Field{Name: name, Type: typ, Prov: cite(f, line)}
		if call != "" {
			fl.Decorators = []archdoc.Decorator{{Name: call, Options: opts}}
		}
		return fl
	}
	m := Derive(tableFacts("Python", map[string][]archdoc.Class{f: {
		{Name: "UserBase", Extends: []string{"SQLModel"}, Fields: []archdoc.Field{field("email", "EmailStr", 2, map[string]string{"unique": "True"}, "Field")}},
		{Name: "User", Extends: []string{"UserBase"}, Options: map[string]string{"table": "True"}, Prov: cite(f, 5), Fields: []archdoc.Field{
			field("id", "uuid.UUID", 6, map[string]string{"primary_key": "True"}, "Field"),
			field("items", `list["Item"]`, 7, nil, "Relationship"),
		}},
		{Name: "Item", Options: map[string]string{"table": "True"}, Prov: cite(f, 10), Fields: []archdoc.Field{
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
