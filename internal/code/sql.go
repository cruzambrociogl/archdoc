package code

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// SQL migrations are where an application that declares no tables in its code — no ORM classes, no
// Prisma schema — still says what its tables are: one CREATE TABLE, then years of ALTER TABLE.
// They are read statement by statement, in file order, and folded into the schema they leave:
//
//	CREATE TABLE posts (id serial PRIMARY KEY, author_id int NOT NULL REFERENCES users (id));
//	ALTER TABLE posts ADD COLUMN title text, DROP COLUMN legacy;
//	ALTER TABLE posts RENAME TO articles;
//	DROP TABLE drafts;
//
// Each table is reported as a table class, in the shape the decorators of TypeORM's kin take, cited
// at its CREATE TABLE; each column at the statement that last defined it. Only what a migration
// spells out is read: a table built by a function, or by a tool's own DSL, is not here. The way
// back — a .down.sql file, the part of a file below "-- +goose Down" — is not applied.

var (
	sqlCreate = regexp.MustCompile(`(?is)^CREATE\s+(GLOBAL\s+|LOCAL\s+)?(TEMP\s+|TEMPORARY\s+|UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)\s*\(`)
	sqlAlter  = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?([^\s(]+)\s+(.*)$`)
	sqlDrop   = regexp.MustCompile(`(?is)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(.*?)(?:\s+(?:CASCADE|RESTRICT))?$`)

	sqlAdd        = regexp.MustCompile(`(?is)^ADD\s+(?:COLUMN\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(.*)$`)
	sqlDropColumn = regexp.MustCompile(`(?is)^DROP\s+(?:COLUMN\s+)?(?:IF\s+EXISTS\s+)?([^\s]+)`)
	sqlRenameCol  = regexp.MustCompile(`(?is)^RENAME\s+(?:COLUMN\s+)?([^\s]+)\s+TO\s+([^\s]+)`)
	sqlRenameTo   = regexp.MustCompile(`(?is)^RENAME\s+TO\s+([^\s]+)`)
	sqlAlterCol   = regexp.MustCompile(`(?is)^ALTER\s+(?:COLUMN\s+)?([^\s]+)\s+(.*)$`)
	sqlForeign    = regexp.MustCompile(`(?is)^FOREIGN\s+KEY\s*\(([^)]*)\)\s*REFERENCES\s+([^\s(]+)`)
	sqlPrimary    = regexp.MustCompile(`(?is)^PRIMARY\s+KEY\s*\(([^)]*)\)`)
	sqlConstraint = regexp.MustCompile(`(?is)^CONSTRAINT\s+[^\s]+\s+`)
	sqlReferences = regexp.MustCompile(`(?is)\bREFERENCES\s+([^\s(]+)`)
	sqlNotNull    = regexp.MustCompile(`(?is)\bNOT\s+NULL\b`)
	sqlPrimaryKey = regexp.MustCompile(`(?is)\bPRIMARY\s+KEY\b`)
	sqlTypeEnd    = regexp.MustCompile(`(?is)\s(NOT|NULL|DEFAULT|PRIMARY|REFERENCES|UNIQUE|CHECK|CONSTRAINT|GENERATED|COLLATE|AUTO_INCREMENT|AUTOINCREMENT|ON|COMMENT)\b`)
	sqlWayBack    = regexp.MustCompile(`(?im)^--\s*(\+goose\s+Down|migrate:down|\+migrate\s+Down)\b`)
	sqlDownFile   = regexp.MustCompile(`(?i)([._-]down\.sql$|(^|/)(down|rollback|rollbacks)/)`)
	// A table constraint, where a column would be: it defines no column of its own.
	sqlNoColumn = regexp.MustCompile(`(?is)^(UNIQUE|CHECK|EXCLUDE|LIKE|INDEX|KEY|FULLTEXT|SPATIAL|PERIOD)\b`)
)

type sqlColumn struct {
	name, typ         string
	nullable, primary bool
	references        string // the table it points at, as written
	at                archdoc.Provenance
}

type sqlTable struct {
	name    string
	columns []*sqlColumn
	at      archdoc.Provenance
	lines   int // of the file that creates it
}

func (t *sqlTable) column(name string) *sqlColumn {
	for _, c := range t.columns {
		if strings.EqualFold(c.name, name) {
			return c
		}
	}
	return nil
}

// sqlSchemas folds every *.sql file under an application's directory, in path order — which is
// the order migration tools apply them in — into the tables they leave.
func sqlSchemas(repo, dir string) []archdoc.SourceFile {
	var tables []*sqlTable // in the order created
	find := func(name string) (int, *sqlTable) {
		for i, t := range tables {
			if strings.EqualFold(t.name, name) {
				return i, t
			}
		}
		return -1, nil
	}
	filepath.WalkDir(filepath.Join(repo, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(repo, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != dir && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(p), ".sql") || sqlDownFile.MatchString(rel) {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		text := string(content)
		if loc := sqlWayBack.FindStringIndex(text); loc != nil {
			text = text[:loc[0]]
		}
		for _, st := range sqlStatements(text) {
			at := func(offset int) archdoc.Provenance {
				for offset < len(st.text) && strings.ContainsRune(" \t\r\n", rune(st.text[offset])) {
					offset++ // an item starts at its first word, not at the comma before it
				}
				return archdoc.Provenance{File: rel, Line: st.line + strings.Count(st.text[:offset], "\n")}
			}
			switch {
			case sqlCreate.MatchString(st.text):
				m := sqlCreate.FindStringSubmatchIndex(st.text)
				if m[2] >= 0 || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(st.text[max(m[4], 0):max(m[5], 0)])), "TEMP") {
					continue // a temporary table is not part of the schema
				}
				t := &sqlTable{name: sqlName(st.text[m[6]:m[7]]), at: at(0), lines: lines(content)}
				t.at.Note = "the schema its migrations leave, applied in file order"
				open := m[1] - 1
				for _, item := range sqlSplit(st.text, open+1, sqlClose(st.text, open)) {
					sqlDefine(t, st.text[item[0]:item[1]], at(item[0]))
				}
				if i, old := find(t.name); old != nil {
					if strings.Contains(strings.ToUpper(st.text[:m[1]]), "IF NOT EXISTS") {
						continue // already there: the first definition stands
					}
					tables[i] = t
				} else {
					tables = append(tables, t)
				}
			case sqlAlter.MatchString(st.text):
				m := sqlAlter.FindStringSubmatchIndex(st.text)
				_, t := find(sqlName(st.text[m[2]:m[3]]))
				if t == nil {
					continue
				}
				for _, item := range sqlSplit(st.text, m[4], len(st.text)) {
					sqlChange(t, st.text[item[0]:item[1]], at(item[0]))
				}
			case sqlDrop.MatchString(st.text):
				for _, name := range strings.Split(sqlDrop.FindStringSubmatch(st.text)[1], ",") {
					if i, t := find(sqlName(name)); t != nil {
						tables = append(tables[:i], tables[i+1:]...)
					}
				}
			}
		}
		return nil
	})

	// One file per file that creates a table still standing, in the order first met.
	var out []archdoc.SourceFile
	index := map[string]int{}
	for _, t := range tables {
		c := archdoc.Class{Name: t.name, Prov: t.at,
			Decorators: []archdoc.Decorator{{Name: "Table", Arg: t.name, HasArg: true, Prov: t.at}}}
		for _, col := range t.columns {
			opts := map[string]string{"type": col.typ}
			if col.nullable {
				opts["nullable"] = "true"
			}
			if col.primary {
				opts["primary"] = "true"
			}
			d := archdoc.Decorator{Name: "Column", Options: opts, Prov: col.at}
			if _, to := find(col.references); to != nil {
				d.Target = to.name
			}
			c.Fields = append(c.Fields, archdoc.Field{Name: col.name, Prov: col.at, Decorators: []archdoc.Decorator{d}})
		}
		i, seen := index[t.at.File]
		if !seen {
			i = len(out)
			index[t.at.File] = i
			out = append(out, archdoc.SourceFile{Path: t.at.File, Language: "SQL", Lines: t.lines})
		}
		out[i].Classes = append(out[i].Classes, c)
	}
	return out
}

// sqlDefine reads one item of a CREATE TABLE's list: a column, or a constraint on columns.
func sqlDefine(t *sqlTable, item string, at archdoc.Provenance) {
	item = strings.Join(strings.Fields(item), " ")
	item = sqlConstraint.ReplaceAllString(item, "")
	switch {
	case item == "":
	case sqlPrimary.MatchString(item):
		for _, name := range strings.Split(sqlPrimary.FindStringSubmatch(item)[1], ",") {
			if c := t.column(sqlName(name)); c != nil {
				c.primary, c.nullable = true, false
			}
		}
	case sqlForeign.MatchString(item):
		m := sqlForeign.FindStringSubmatch(item)
		for _, name := range strings.Split(m[1], ",") {
			if c := t.column(sqlName(name)); c != nil {
				c.references = sqlName(m[2])
			}
		}
	case sqlNoColumn.MatchString(item):
	default:
		name, rest, _ := strings.Cut(item, " ")
		if strings.HasPrefix(item, `"`) {
			if end := strings.Index(item[1:], `"`); end >= 0 {
				name, rest = item[:end+2], item[end+2:]
			}
		}
		rest = " " + strings.Join(strings.Fields(rest), " ")
		c := &sqlColumn{name: sqlName(name), typ: strings.TrimSpace(rest), at: at}
		if loc := sqlTypeEnd.FindStringIndex(rest); loc != nil {
			c.typ = strings.TrimSpace(rest[:loc[0]])
		}
		c.primary = sqlPrimaryKey.MatchString(rest)
		c.nullable = !c.primary && !sqlNotNull.MatchString(rest)
		if m := sqlReferences.FindStringSubmatch(rest); m != nil {
			c.references = sqlName(m[1])
		}
		if c.name == "" {
			return
		}
		if old := t.column(c.name); old != nil {
			*old = *c
		} else {
			t.columns = append(t.columns, c)
		}
	}
}

// sqlChange applies one action of an ALTER TABLE.
func sqlChange(t *sqlTable, action string, at archdoc.Provenance) {
	action = strings.TrimSpace(action)
	upper := strings.ToUpper(action)
	switch {
	case sqlAdd.MatchString(action):
		sqlDefine(t, sqlAdd.FindStringSubmatch(action)[1], at)
	case strings.HasPrefix(upper, "DROP CONSTRAINT"), strings.HasPrefix(upper, "RENAME CONSTRAINT"):
	case sqlDropColumn.MatchString(action):
		name := sqlName(sqlDropColumn.FindStringSubmatch(action)[1])
		for i, c := range t.columns {
			if strings.EqualFold(c.name, name) {
				t.columns = append(t.columns[:i], t.columns[i+1:]...)
				break
			}
		}
	case sqlRenameTo.MatchString(action):
		t.name = sqlName(sqlRenameTo.FindStringSubmatch(action)[1])
	case sqlRenameCol.MatchString(action):
		m := sqlRenameCol.FindStringSubmatch(action)
		if c := t.column(sqlName(m[1])); c != nil {
			c.name, c.at = sqlName(m[2]), at
		}
	case sqlAlterCol.MatchString(action):
		m := sqlAlterCol.FindStringSubmatch(action)
		c := t.column(sqlName(m[1]))
		if c == nil {
			return
		}
		what := strings.ToUpper(strings.Join(strings.Fields(m[2]), " "))
		switch {
		case what == "SET NOT NULL":
			c.nullable, c.at = false, at
		case what == "DROP NOT NULL":
			c.nullable, c.at = !c.primary, at
		case strings.HasPrefix(what, "TYPE "), strings.HasPrefix(what, "SET DATA TYPE "):
			typ := strings.Join(strings.Fields(m[2]), " ")
			typ = typ[strings.Index(strings.ToUpper(typ), "TYPE ")+5:]
			if using := strings.Index(strings.ToUpper(typ), " USING "); using >= 0 {
				typ = typ[:using]
			}
			c.typ, c.at = strings.TrimSpace(typ), at
		}
	}
}

// sqlName is a name without its quotes, and without the schema everything is in unless it says
// otherwise: "public"."users" is users, auth.users stays auth.users.
func sqlName(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.NewReplacer(`"`, "", "`", "", "[", "", "]", "").Replace(raw)
	raw = strings.TrimRight(raw, ";,")
	if schema, name, ok := strings.Cut(raw, "."); ok && (strings.EqualFold(schema, "public") || strings.EqualFold(schema, "dbo")) {
		return name
	}
	return raw
}

type sqlStatement struct {
	text string // comments blanked, lines kept
	line int    // of its first word
}

// sqlStatements splits a file at the semicolons that end statements — not those inside a string, a
// quoted name, a comment or a $$-quoted body — and blanks its comments.
func sqlStatements(content string) []sqlStatement {
	b := []byte(content)
	var out []sqlStatement
	start, line, startLine := 0, 1, 1
	flush := func(end int) {
		text := string(b[start:end])
		lead := len(text) - len(strings.TrimLeft(text, " \t\r\n"))
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			out = append(out, sqlStatement{text: strings.TrimRight(text[lead:], " \t\r\n"), line: startLine + strings.Count(text[:lead], "\n")})
		}
		start, startLine = end+1, line
	}
	for i := 0; i < len(b); i++ {
		switch c := b[i]; {
		case c == '\n':
			line++
		case c == '-' && i+1 < len(b) && b[i+1] == '-':
			for ; i < len(b) && b[i] != '\n'; i++ {
				b[i] = ' '
			}
			i--
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			for ; i < len(b) && !(b[i] == '*' && i+1 < len(b) && b[i+1] == '/'); i++ {
				if b[i] == '\n' {
					line++
				} else {
					b[i] = ' '
				}
			}
			if i+1 < len(b) {
				b[i], b[i+1] = ' ', ' '
				i++
			}
		case c == '\'' || c == '"' || c == '`':
			for i++; i < len(b) && b[i] != c; i++ {
				if b[i] == '\n' {
					line++
				}
			}
		case c == '$':
			end := i + 1
			for end < len(b) && (b[end] == '_' || b[end] >= 'a' && b[end] <= 'z' || b[end] >= 'A' && b[end] <= 'Z') {
				end++
			}
			if end >= len(b) || b[end] != '$' {
				continue
			}
			tag := string(b[i : end+1])
			if close := strings.Index(string(b[end+1:]), tag); close >= 0 {
				line += strings.Count(string(b[i:end+1+close]), "\n")
				i = end + close + len(tag)
			}
		case c == ';':
			flush(i)
		}
	}
	flush(len(b))
	return out
}

// sqlClose is where the parenthesis opened at open closes.
func sqlClose(text string, open int) int {
	depth := 0
	for i := open; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '\'':
			for i++; i < len(text) && text[i] != '\''; i++ {
			}
		}
	}
	return len(text)
}

// sqlSplit cuts text[from:to] at the commas that separate items, not those inside parentheses or
// strings, and gives each item's span.
func sqlSplit(text string, from, to int) [][2]int {
	var out [][2]int
	depth, start := 0, from
	for i := from; i < to; i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '\'':
			for i++; i < to && text[i] != '\''; i++ {
			}
		case ',':
			if depth == 0 {
				out = append(out, [2]int{start, i})
				start = i + 1
			}
		}
	}
	return append(out, [2]int{start, to})
}
