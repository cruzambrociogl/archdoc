package model

import (
	"sort"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
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
// Migrations are the other source of a schema. They say how it got here, not what it is, so they
// are read only where the code declares no table: then the SQL they are written in, folded in file
// order, is the schema (internal/code/sql.go).

// TypeORM's relation decorators: the field holds a relation, not necessarily a column.
var relationDecorators = map[string]bool{"ManyToOne": true, "OneToOne": true, "OneToMany": true, "ManyToMany": true,
	"Collection": true} // the last: a C# property holding many of an entity, as the C# reader marks it

type tableClass struct {
	class core.Class
	file  string
	name  string
	sql   bool // from migrations, not from a declaration
}

func tables(m *core.Model, sources []core.Source) {
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == core.Application {
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

		classes := map[string]core.Class{} // every class, by name — for a Python table's bases
		var found []tableClass
		all := append(append([]core.SourceFile(nil), src.Files...), src.Schemas...)
		for _, f := range all {
			for _, c := range f.Classes {
				if c.Name != "" {
					if _, seen := classes[c.Name]; !seen {
						classes[c.Name] = c
					}
				}
			}
		}
		sets := efTables(all)
		// A DbSet's class is the one beside its context — both in the application's code, or both in a
		// library it is built with — not another class of the same name: eShopOnWeb's Web has a
		// Basket view component as well as the Basket entity.
		schema := map[string]bool{}
		for _, f := range src.Schemas {
			schema[f.Path] = true
		}
		for _, f := range all {
			for _, c := range f.Classes {
				name, ok := tableName(c, f.Language)
				if set, held := sets[c.Name]; !ok && held && f.Language == "C#" && !c.Interface && schema[f.Path] == schema[set.Prov.File] {
					name, ok = set.Name, true // EF Core names a table after the DbSet that holds it
				}
				if ok {
					found = append(found, tableClass{class: c, file: f.Path, name: name, sql: f.Language == "SQL"})
				}
			}
		}
		// What the code declares is the schema; migrations stand in only where it declares nothing.
		declared := false
		for _, t := range found {
			declared = declared || !t.sql
		}
		if declared {
			kept := found[:0]
			for _, t := range found {
				if !t.sql {
					kept = append(kept, t)
				}
			}
			found = kept
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
			var cols []core.Column
			python := strings.HasSuffix(t.file, ".py")
			switch {
			case python:
				cols = pyColumns(t.class, classes, byName)
			case strings.HasSuffix(t.file, ".java"):
				cols = javaColumns(t.class, classes, byClass)
			case strings.HasSuffix(t.file, ".cs"):
				cols = csColumns(t.class, classes, byClass)
			default:
				cols = tsColumns(t.class, byClass)
			}
			m.Nodes = append(m.Nodes, core.Node{
				ID: id, Name: t.name, Kind: core.Table, Evidence: core.Declared,
				Parent: container, Dir: t.file, Files: []string{t.file}, Columns: cols, Prov: t.class.Prov,
			})
			for _, c := range cols {
				if c.References != "" && c.References != id {
					m.Edges = append(m.Edges, core.Edge{From: id, To: c.References, Label: "references",
						Prov: []core.Provenance{c.Prov}})
				}
			}
			if !python {
				// A relation the field declares without a column of its own — @OneToMany(() => Photo)
				// — still ties the two tables.
				for _, f := range t.class.Fields {
					for _, d := range f.Decorators {
						if to, ok := byClass[d.Target]; ok && relationDecorators[d.Name] && to != id {
							m.Edges = append(m.Edges, core.Edge{From: id, To: to, Label: "relates to",
								Prov: []core.Provenance{d.Prov}})
						}
					}
				}
			}
		}
	}
}

// shareTables draws a table once when several programs read the schema that declares it: a Go
// module's migrations, which its api, its worker and its seed all reach. Until here each program has
// its own copy, which is how its flows and its queries find a table by name; read from one file, they
// are one table. It goes to the database the programs keep it in — when every one of them that
// connects to a relational store connects to the same one — or else to the first of them. Run after
// the connections are known. It returns where each program finds a table by name, for its flows and
// its queries: program, then name → the table's ID.
func shareTables(m *core.Model) map[string]string {
	index := map[string]string{}
	type copyOf struct{ file, name string }
	copies := map[copyOf][]int{}
	var order []copyOf // in the model's order, which is deterministic
	for i, n := range m.Nodes {
		if n.Kind == core.Table {
			index[n.Parent+"\x00"+n.Name] = n.ID
			k := copyOf{n.Dir, n.Name}
			if copies[k] == nil {
				order = append(order, k)
			}
			copies[k] = append(copies[k], i)
		}
	}
	stores := map[string]map[string]bool{} // a program → the relational stores it connects to
	kinds := map[string]core.Node{}
	for _, n := range m.Nodes {
		kinds[n.ID] = n
	}
	for _, e := range m.Edges {
		if to, ok := kinds[e.To]; ok && e.Traffic && to.Kind == core.Datastore && core.Relational(to.Technology) {
			if stores[e.From] == nil {
				stores[e.From] = map[string]bool{}
			}
			stores[e.From][e.To] = true
		}
	}

	rename := map[string]string{}
	drop := map[int]bool{}
	for _, k := range order {
		at := copies[k]
		if len(at) < 2 {
			continue
		}
		var readers []string
		for _, i := range at {
			readers = append(readers, m.Nodes[i].Parent)
		}
		sort.Strings(readers)
		owner := readers[0]
		held := map[string]bool{}
		for _, r := range readers {
			for s := range stores[r] {
				held[s] = true
			}
		}
		if len(held) == 1 {
			for s := range held {
				owner = s
			}
		}
		_, local, _ := strings.Cut(owner, ":")
		id := "tbl:" + local + "/" + k.name
		for _, i := range at {
			rename[m.Nodes[i].ID] = id
		}
		keep := at[0]
		m.Nodes[keep].ID, m.Nodes[keep].Parent = id, owner
		for _, i := range at[1:] {
			drop[i] = true
		}
	}
	if len(rename) == 0 {
		return index
	}
	to := func(id string) string {
		if r, ok := rename[id]; ok {
			return r
		}
		return id
	}
	nodes := m.Nodes[:0]
	for i, n := range m.Nodes {
		if drop[i] {
			continue
		}
		if n.Kind == core.Table {
			cols := append([]core.Column(nil), n.Columns...)
			for c := range cols {
				cols[c].References = to(cols[c].References)
			}
			n.Columns = cols
		}
		nodes = append(nodes, n)
	}
	m.Nodes = nodes
	// A foreign key each copy declared is one foreign key: the same two tables, cited at the same line.
	seen := map[string]bool{}
	edges := m.Edges[:0]
	for _, e := range m.Edges {
		_, merged := rename[e.From]
		e.From, e.To = to(e.From), to(e.To)
		if merged {
			key := e.From + "\x00" + e.To + "\x00" + e.Label
			if len(e.Prov) > 0 {
				key += "\x00" + e.Prov[0].File + ":" + strconv.Itoa(e.Prov[0].Line)
			}
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		edges = append(edges, e)
	}
	m.Edges = edges
	for k, id := range index {
		index[k] = to(id)
	}
	return index
}

// tableName says whether a class is a table, and its name.
func tableName(c core.Class, language string) (string, bool) {
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
	// A name the declaration gives wins over the class's: JPA's @Entity @Table(name = "owners").
	found := false
	for _, d := range c.Decorators {
		if d.Name != "Table" && d.Name != "Entity" {
			continue
		}
		found = true
		switch {
		case d.HasArg && d.Arg != "":
			return d.Arg, true
		case d.Options["name"] != "":
			return d.Options["name"], true
		}
	}
	return c.Name, found
}

// tsColumns are the fields a column decorator marks.
func tsColumns(c core.Class, byClass map[string]string) []core.Column {
	var out []core.Column
	for _, f := range c.Fields {
		for _, d := range f.Decorators {
			if !strings.HasSuffix(d.Name, "Column") {
				continue
			}
			col := core.Column{Name: f.Name, Type: d.Options["type"], Prov: f.Prov}
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
func pyColumns(c core.Class, classes map[string]core.Class, byName map[string]string) []core.Column {
	var fields []core.Field
	seen := map[string]bool{}
	var collect func(c core.Class, depth int)
	collect = func(c core.Class, depth int) {
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

	var out []core.Column
	for _, f := range fields {
		if strings.HasPrefix(f.Name, "__") || f.Type == "" && len(f.Decorators) == 0 {
			continue
		}
		var call core.Decorator
		if len(f.Decorators) > 0 {
			call = f.Decorators[0]
		}
		if strings.EqualFold(call.Name, "Relationship") || call.Name == "relationship" {
			continue // the other side of a foreign key, not a column
		}
		col := core.Column{Name: f.Name, Type: f.Type, Prov: f.Prov}
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
