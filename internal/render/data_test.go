package render

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// A data view's arrows are foreign keys, and its key says so. Mermaid's line names the columns that
// hold the key, and is drawn optional when every one of them may be empty.
func TestDataViewKeysAndLines(t *testing.T) {
	m := core.Model{Name: "x",
		Nodes: []core.Node{
			{ID: "app:api", Name: "api", Kind: core.Application, Evidence: core.Declared},
			{ID: "tbl:api/users", Name: "users", Kind: core.Table, Parent: "app:api", Evidence: core.Declared,
				Columns: []core.Column{{Name: "id", Type: "bigint", Primary: true}}},
			{ID: "tbl:api/posts", Name: "posts", Kind: core.Table, Parent: "app:api", Evidence: core.Declared,
				Columns: []core.Column{
					{Name: "author_id", Type: "bigint", References: "tbl:api/users"},
					{Name: "editor_id", Type: "bigint", References: "tbl:api/users", Nullable: true},
				}},
			{ID: "tbl:api/drafts", Name: "drafts", Kind: core.Table, Parent: "app:api", Evidence: core.Declared,
				Columns: []core.Column{{Name: "post_id", Type: "bigint", References: "tbl:api/posts", Nullable: true}}},
		},
		Edges: []core.Edge{
			{From: "tbl:api/posts", To: "tbl:api/users", Label: "references", Prov: []core.Provenance{{File: "a.sql", Line: 4}}},
			{From: "tbl:api/posts", To: "tbl:api/users", Label: "references", Prov: []core.Provenance{{File: "a.sql", Line: 5}}},
			{From: "tbl:api/drafts", To: "tbl:api/posts", Label: "references", Prov: []core.Provenance{{File: "a.sql", Line: 8}}},
		},
	}
	view := dataView(m, "app:api")
	var arrow string
	for _, k := range KeyOf(view.Model) {
		if k.Style == "arrow" {
			arrow = k.Label
		}
	}
	if !strings.Contains(arrow, "foreign key") {
		t.Errorf("a data view's arrow reads %q", arrow)
	}
	out := Mermaid(view.Model, false)
	for _, want := range []string{
		`posts }o--|| users : "author_id, editor_id"`, // one column must be filled: every post has a user
		`drafts }o--o| posts : "post_id"`,             // the only column may be empty
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}
