package code

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// A Prisma schema is where a Prisma application declares its tables: a model per table, a line per
// field. It is its own small language, read line by line — which is all its grammar needs:
//
//	model Post {
//	  id       Int    @id @default(autoincrement())
//	  title    String?
//	  authorId Int
//	  author   User   @relation(fields: [authorId], references: [id])
//	  @@map("posts")
//	}
//
// Each model is reported as a table class, in the shape the decorators of TypeORM's kin take, so
// one derivation serves both: scalar fields are columns; a field whose type is another model is the
// relation, and the fields it names carry the foreign key.

var (
	prismaModel    = regexp.MustCompile(`^model\s+(\w+)\s*\{`)
	prismaField    = regexp.MustCompile(`^(\w+)\s+(\w+)(\[\])?(\?)?(.*)$`)
	prismaRelation = regexp.MustCompile(`@relation\([^)]*fields:\s*\[([^\]]*)\]`)
	prismaMap      = regexp.MustCompile(`@@map\("([^"]+)"\)`)
)

// prismaSchemas reads every *.prisma file under an application's directory.
func prismaSchemas(repo, dir string) []archdoc.SourceFile {
	var out []archdoc.SourceFile
	filepath.WalkDir(filepath.Join(repo, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".prisma") {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(repo, p)
		rel = filepath.ToSlash(rel)
		out = append(out, archdoc.SourceFile{Path: rel, Language: "Prisma", Lines: lines(content), Classes: prismaModels(string(content), rel)})
		return nil
	})
	return out
}

func prismaModels(content, file string) []archdoc.Class {
	var out []archdoc.Class
	var cur *archdoc.Class
	foreign := map[string]string{} // field → the model its relation points at
	models := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		if m := prismaModel.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			models[m[1]] = true
		}
	}
	flush := func() {
		if cur == nil {
			return
		}
		for i := range cur.Fields {
			if to := foreign[cur.Fields[i].Name]; to != "" {
				cur.Fields[i].Decorators[0].Target = to
			}
		}
		out = append(out, *cur)
		cur, foreign = nil, map[string]string{}
	}
	for i, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if c := strings.Index(line, "//"); c >= 0 {
			line = strings.TrimSpace(line[:c])
		}
		at := archdoc.Provenance{File: file, Line: i + 1}
		switch {
		case line == "":
		case cur == nil:
			if m := prismaModel.FindStringSubmatch(line); m != nil {
				cur = &archdoc.Class{Name: m[1], Prov: at, Decorators: []archdoc.Decorator{{Name: "Table", Prov: at}}}
			}
		case line == "}":
			flush()
		case strings.HasPrefix(line, "@@"):
			if m := prismaMap.FindStringSubmatch(line); m != nil {
				cur.Decorators[0].Arg, cur.Decorators[0].HasArg = m[1], true
			}
		default:
			m := prismaField.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name, typ, list, optional, rest := m[1], m[2], m[3] != "", m[4] != "", m[5]
			if models[typ] {
				// The relation itself: no column of its own; the fields it names hold the key.
				if r := prismaRelation.FindStringSubmatch(rest); r != nil {
					for _, f := range strings.Split(r[1], ",") {
						foreign[strings.TrimSpace(f)] = typ
					}
				}
				continue
			}
			if list {
				typ += "[]"
			}
			opts := map[string]string{"type": typ}
			if optional {
				opts["nullable"] = "true"
			}
			if strings.Contains(rest, "@id") {
				opts["primary"] = "true"
			}
			cur.Fields = append(cur.Fields, archdoc.Field{Name: name, Type: typ, Prov: at,
				Decorators: []archdoc.Decorator{{Name: "Column", Options: opts, Prov: at}}})
		}
	}
	flush()
	return out
}
