package model

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Every query counts, whether or not a route's flow reaches it: a job's repository method that no
// route calls still writes its table. One access per component, table and operation, citing its
// queries in file order.
func TestAccessReadsEveryQuery(t *testing.T) {
	repo, job := "server/src/repositories/album.repository.ts", "server/src/jobs/cleanup.ts"
	q := func(table, op, file string, line int) archdoc.Query {
		return archdoc.Query{Table: table, Op: op, Prov: cite(file, line)}
	}
	fs := immichLike()
	fs.Sources[0].Files = append(fs.Sources[0].Files,
		archdoc.SourceFile{Path: repo, Language: "TypeScript", Lines: 30, Classes: []archdoc.Class{{Name: "AlbumRepository",
			Methods: []archdoc.Method{
				{Name: "getAll", Prov: cite(repo, 5), Queries: []archdoc.Query{q("album", "reads", repo, 6)}},
				{Name: "update", Prov: cite(repo, 9), Queries: []archdoc.Query{q("album", "updates", repo, 10), q("album", "reads", repo, 11)}},
			}}}},
		archdoc.SourceFile{Path: job, Language: "TypeScript", Lines: 10, Classes: []archdoc.Class{{Name: "",
			Methods: []archdoc.Method{{Name: "purge", Prov: cite(job, 2), Queries: []archdoc.Query{q("audit", "deletes", job, 3)}}}}}},
	)
	m := Derive(fs)
	var got []string
	for _, a := range m.Access {
		got = append(got, a.Table+":"+a.Op)
		if a.Table == "album" && a.Op == "reads" && (a.Count != 2 || len(a.Prov) != 2 || a.Prov[0].Line != 6) {
			t.Errorf("album reads: %+v", a)
		}
		if a.Container != "svc:server" {
			t.Errorf("%s in %q", a.Table, a.Container)
		}
	}
	want := map[string]bool{"album:reads": true, "album:updates": true, "audit:deletes": true}
	if len(got) != len(want) {
		t.Fatalf("access %v, want %v", got, want)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected access %s", g)
		}
	}
}
