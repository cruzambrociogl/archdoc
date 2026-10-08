package model

import (
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// The data model (F-07): the tables the code declares, their columns, and the foreign keys
// between them — read from the classes the code marks as tables, which is where an ORM's schema
// lives. Three conventions, all of them the code describing itself:
//
//   - TypeScript, TypeORM and its kin (Immich's sql-tools): a class decorated @Table or @Entity;
//     a column is a field whose decorator ends in Column; @ForeignKeyColumn(() => AssetTable) and
//     @ManyToOne(() => User) name the other end.
//   - Python, SQLModel: a class with table=True; its fields, and its bases' fields; a column's
//     Field(foreign_key="user.id") names the other end.
//   - Python, SQLAlchemy's declarative style: a class with a __tablename__.
//
// Migrations are the other source of a schema; they say how it got here, not what it is, and are
// not read for it.

// TypeORM's relation decorators: the field holds a relation, not necessarily a column.
var relationDecorators = map[string]bool{"ManyToOne": true, "OneToOne": true, "OneToMany": true, "ManyToMany": true}

type tableClass struct {
	class archdoc.Class
	file  string
	name  string
}

func tables(m *archdoc.Model, sources []archdoc.Source) {
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	for _, src := range sources {
		container, ok := containerOf[src.App]
		if !ok {
			continue
		}
		_, local, _ := strings.Cut(container, ":")
		prefix := "tbl:" + local + "/"

		classes := map[string]archdoc.Class{} // every class, by name — for a Python table's bases
		var found []tableClass
		all := append(append([]archdoc.SourceFile(nil), src.Files...), src.Schemas...)
		for _, f := range all {
			for _, c := range f.Classes {
				if c.Name != "" {
					if _, seen := classes[c.Name]; !seen {
						classes[c.Name] = c
					}
				}
			}
		}
		for _, f := range all {
			for _, c := range f.Classes {
				if name, ok := tableName(c, f.Language); ok {
					found = append(found, tableClass{class: c, file: f.Path, name: name})
				}
			}
		}
		if len(found) == 0 {
			continue
		}
		byClass := map[string]string{} // class name → table node ID
		byName := map[string]string{}  // table name → table node ID
		for _, t := range found {
			byClass[t.class.Name] = prefix + t.name
			byName[t.name] = prefix + t.name
		}

		seen := map[string]bool{}
		for _, t := range found {
			id := prefix + t.name
			if seen[id] {
				continue // two classes for one table: the first declaration is the table
			}
			seen[id] = true
			var cols []archdoc.Column
			python := strings.HasSuffix(t.file, ".py")
			if python {
				cols = pyColumns(t.class, classes, byName)
			} else {
				cols = tsColumns(t.class, byClass)
			}
			m.Nodes = append(m.Nodes, archdoc.Node{
				ID: id, Name: t.name, Kind: archdoc.Table, Evidence: archdoc.Declared,
				Parent: container, Dir: t.file, Files: []string{t.file}, Columns: cols, Prov: t.class.Prov,
			})
			for _, c := range cols {
				if c.References != "" && c.References != id {
					m.Edges = append(m.Edges, archdoc.Edge{From: id, To: c.References, Label: "references",
						Prov: []archdoc.Provenance{c.Prov}})
				}
			}
			if !python {
				// A relation the field declares without a column of its own — @OneToMany(() => Photo)
				// — still ties the two tables.
				for _, f := range t.class.Fields {
					for _, d := range f.Decorators {
						if to, ok := byClass[d.Target]; ok && relationDecorators[d.Name] && to != id {
							m.Edges = append(m.Edges, archdoc.Edge{From: id, To: to, Label: "relates to",
								Prov: []archdoc.Provenance{d.Prov}})
						}
					}
				}
			}
		}
	}
}

// tableName says whether a class is a table, and its name.
func tableName(c archdoc.Class, language string) (string, bool) {
	if language == "Python" {
		for _, f := range c.Fields {
			if f.Name == "__tablename__" && f.Value != "" {
				return f.Value, true
			}
		}
		if c.Options["table"] == "True" {
			return strings.ToLower(c.Name), true // SQLModel names a table after its class
		}
		return "", false
	}
	for _, d := range c.Decorators {
		if d.Name != "Table" && d.Name != "Entity" {
			continue
		}
		switch {
		case d.HasArg && d.Arg != "":
			return d.Arg, true
		case d.Options["name"] != "":
			return d.Options["name"], true
		default:
			return c.Name, true
		}
	}
	return "", false
}

// tsColumns are the fields a column decorator marks.
func tsColumns(c archdoc.Class, byClass map[string]string) []archdoc.Column {
	var out []archdoc.Column
	for _, f := range c.Fields {
		for _, d := range f.Decorators {
			if !strings.HasSuffix(d.Name, "Column") {
				continue
			}
			col := archdoc.Column{Name: f.Name, Type: d.Options["type"], Prov: f.Prov}
			if n := d.Options["name"]; n != "" {
				col.Name = n
			}
			if col.Type == "" {
				col.Type = tsType(f.Type)
			}
			col.Primary = strings.HasPrefix(d.Name, "Primary") || d.Options["primary"] == "true"
			col.Nullable = d.Options["nullable"] == "true" || strings.Contains(f.Type, "null")
			if to, ok := byClass[d.Target]; ok {
				col.References = to
			}
			out = append(out, col)
			break
		}
	}
	return out
}

// tsType is a field's type without the wrappers an ORM adds and without "| null", which nullable
// already says: Generated<Timestamp> | null is Timestamp.
func tsType(t string) string {
	t = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(t, "| null", ""), "null |", ""))
	for _, w := range []string{"Generated<", "ColumnType<"} {
		if strings.HasPrefix(t, w) && strings.HasSuffix(t, ">") {
			t = strings.TrimSuffix(strings.TrimPrefix(t, w), ">")
		}
	}
	return strings.TrimSpace(t)
}

// pyColumns are a SQLModel or SQLAlchemy table's fields, its bases' first — a SQLModel table
// inherits most of its columns from the model it extends.
func pyColumns(c archdoc.Class, classes map[string]archdoc.Class, byName map[string]string) []archdoc.Column {
	var fields []archdoc.Field
	seen := map[string]bool{}
	var collect func(c archdoc.Class, depth int)
	collect = func(c archdoc.Class, depth int) {
		if depth > 8 {
			return
		}
		for _, b := range c.Extends {
			if base, ok := classes[b]; ok && base.Name != c.Name {
				collect(base, depth+1)
			}
		}
		for _, f := range c.Fields {
			if !seen[f.Name] {
				seen[f.Name] = true
				fields = append(fields, f)
			} else {
				for i := range fields {
					if fields[i].Name == f.Name {
						fields[i] = f // a subclass redeclares a field: its declaration wins
					}
				}
			}
		}
	}
	collect(c, 0)

	var out []archdoc.Column
	for _, f := range fields {
		if strings.HasPrefix(f.Name, "__") || f.Type == "" && len(f.Decorators) == 0 {
			continue
		}
		var call archdoc.Decorator
		if len(f.Decorators) > 0 {
			call = f.Decorators[0]
		}
		if strings.EqualFold(call.Name, "Relationship") || call.Name == "relationship" {
			continue // the other side of a foreign key, not a column
		}
		col := archdoc.Column{Name: f.Name, Type: f.Type, Prov: f.Prov}
		if col.Type == "" {
			col.Type = call.Target // SQLAlchemy's classic style: id = Column(Integer, …)
		}
		col.Primary = call.Options["primary_key"] == "True"
		col.Nullable = call.Options["nullable"] == "True" || strings.Contains(f.Type, "None") || strings.HasPrefix(f.Type, "Optional")
		if fk := call.Options["foreign_key"]; fk != "" {
			table, _, _ := strings.Cut(fk, ".")
			col.References = byName[table]
		}
		out = append(out, col)
	}
	return out
}
