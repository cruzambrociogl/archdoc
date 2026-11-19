package archdoc_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/cruzambrociogl/archdoc"
)

// A Go repository, documented: each program a container, each package it reaches a component of it,
// its routes and jobs read from the code, and its dependencies counted where its packages import
// them.
func TestAGoRepository(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	files := map[string]string{
		"go.mod": "module github.com/acme/shop\n\ngo 1.27\n\nrequire (\n\tgithub.com/go-chi/chi/v5 v5.2.0\n" +
			"\tgithub.com/riverqueue/river v0.20.0\n\tgithub.com/jackc/pgx/v5 v5.7.6\n)\n",
		"cmd/api/main.go": "package main\n\nimport (\n\t\"net/http\"\n\n\t\"github.com/acme/shop/internal/web\"\n)\n\n" +
			"func main() { http.ListenAndServe(\":8080\", web.Router()) }\n",
		"internal/web/web.go": "package web\n\nimport (\n\t\"net/http\"\n\n\t\"github.com/go-chi/chi/v5\"\n" +
			"\t\"github.com/go-chi/chi/v5/middleware\"\n\t\"github.com/acme/shop/internal/store\"\n)\n\n" +
			"func Router() http.Handler {\n\tr := chi.NewRouter()\n\tr.Use(middleware.Logger)\n" +
			"\tr.Get(\"/orders\", list)\n\treturn r\n}\n\n" +
			"// list answers every order.\nfunc list(w http.ResponseWriter, r *http.Request) { store.All() }\n",
		"internal/store/store.go": "package store\n\nimport \"github.com/jackc/pgx/v5\"\n\nvar _ = pgx.Connect\n\nfunc All() {}\n",
		"cmd/worker/main.go": "package main\n\nimport (\n\t\"github.com/acme/shop/internal/ship\"\n\t\"github.com/riverqueue/river\"\n)\n\n" +
			"func main() { w := river.NewWorkers(); river.AddWorker(w, &ship.Worker{}) }\n",
		"internal/ship/ship.go": "package ship\n\nimport (\n\t\"context\"\n\n\t\"github.com/riverqueue/river\"\n)\n\n" +
			"type Args struct{}\n\nfunc (Args) Kind() string { return \"ship\" }\n\ntype Worker struct{}\n\n" +
			"func (w *Worker) Work(ctx context.Context, j *river.Job[Args]) error { return nil }\n",
		"db/migrations/001_orders.sql":      "CREATE TABLE orders (id serial PRIMARY KEY);\n",
		"testdata/other/docker-compose.yml": "services:\n  fixture:\n    image: nginx\n",
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := archdoc.ReadModel(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "shop" {
		t.Errorf("named %q — a fixture's Compose file is not the system", m.Name)
	}
	kinds := map[string][]string{}
	for _, n := range m.Nodes {
		kinds[string(n.Kind)] = append(kinds[string(n.Kind)], n.ID)
	}
	if got := kinds["application"]; !slices.Equal(got, []string{"app:cmd/api", "app:cmd/worker"}) {
		t.Errorf("containers %v", got)
	}
	wantComponents := []string{"cmp:cmd/api/cmd/api", "cmp:cmd/api/internal/store", "cmp:cmd/api/internal/web",
		"cmp:cmd/worker/cmd/worker", "cmp:cmd/worker/internal/ship"}
	got := kinds["component"]
	slices.Sort(got)
	if !slices.Equal(got, wantComponents) {
		t.Errorf("components %v\n want %v", got, wantComponents)
	}
	// The migrations' tables are the API's, which talks to the database; the worker does not.
	if got := kinds["table"]; !slices.Equal(got, []string{"tbl:cmd/api/orders"}) {
		t.Errorf("tables %v", got)
	}
	var entries []string
	for _, e := range m.Entries {
		entries = append(entries, e.ID+" "+e.Handler+" "+e.Summary)
	}
	wantEntries := []string{"job:cmd/worker ship ship.Worker.Work ", "route:cmd/api GET /orders web.list list answers every order."}
	if !slices.Equal(entries, wantEntries) {
		t.Errorf("entries %q\n want %q", entries, wantEntries)
	}
	uses := false
	for _, e := range m.Edges {
		uses = uses || e.From == "cmp:cmd/api/internal/web" && e.To == "cmp:cmd/api/internal/store" && e.Label == "uses"
	}
	if !uses {
		t.Error("web uses store: no edge")
	}
	// chi's middleware is a package of the chi module: one dependency, imported twice.
	for _, d := range m.Dependencies {
		if d.Container == "app:cmd/api" && d.Name == "github.com/go-chi/chi/v5" && d.Imports != 2 {
			t.Errorf("chi imported %d times, want 2", d.Imports)
		}
	}
}
