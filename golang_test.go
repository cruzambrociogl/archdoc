package archdoc_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	writeTree(t, root, files)
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

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// A Go route's flow: through the store its handler is given, to the table its SQL names, and to the
// job it queues — which the other program works, and says so. A helper that leads nowhere is no step.
func TestAGoFlow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	writeTree(t, root, map[string]string{
		"go.mod": "module github.com/acme/shop\n\ngo 1.27\n",
		"cmd/api/main.go": `package main

import (
	"net/http"

	"github.com/acme/shop/internal/orders"
	"github.com/acme/shop/internal/store"
)

func main() {
	s := &orders.Service{Store: store.New()}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", s.Place)
	http.ListenAndServe(":8080", mux)
}
`,
		"internal/orders/orders.go": `package orders

import (
	"net/http"

	"github.com/acme/shop/internal/store"
)

type Service struct{ Store *store.Store }

// Place takes an order.
func (s *Service) Place(w http.ResponseWriter, r *http.Request) {
	id := parse(r)
	s.Store.Add(id)
}

func parse(r *http.Request) int { return 1 }
`,
		"internal/store/store.go": `package store

import (
	"github.com/acme/shop/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type Store struct {
	db    *pgx.Conn
	river *river.Client[pgx.Tx]
}

func New() *Store { return &Store{} }

const addOrder = "-- name: AddOrder :exec\nINSERT INTO orders (id) VALUES ($1)"

func (s *Store) Add(id int) {
	s.db.Exec(nil, addOrder, id)
	s.river.Insert(nil, jobs.Ship{Order: id}, nil)
}
`,
		"internal/jobs/jobs.go": "package jobs\n\nimport \"github.com/riverqueue/river\"\n\ntype Ship struct{ Order int }\n\n" +
			"func (Ship) Kind() string { return \"ship\" }\n\nvar _ river.JobArgs = Ship{}\n",
		"internal/ship/ship.go": "package ship\n\nimport (\n\t\"context\"\n\n\t\"github.com/acme/shop/internal/jobs\"\n\t\"github.com/riverqueue/river\"\n)\n\n" +
			"type Worker struct{}\n\nfunc (w *Worker) Work(ctx context.Context, j *river.Job[jobs.Ship]) error { return nil }\n",
		"cmd/worker/main.go": "package main\n\nimport (\n\t\"github.com/acme/shop/internal/ship\"\n\t\"github.com/riverqueue/river\"\n)\n\n" +
			"func main() { w := river.NewWorkers(); river.AddWorker(w, &ship.Worker{}) }\n",
		"db/migrations/001.sql": "CREATE TABLE orders (id int PRIMARY KEY);\n",
	})
	m, err := archdoc.ReadModel(root)
	if err != nil {
		t.Fatal(err)
	}
	var flow *archdoc.Flow
	for i := range m.Flows {
		if m.Flows[i].Entry == "route:cmd/api POST /orders" {
			flow = &m.Flows[i]
		}
	}
	if flow == nil {
		t.Fatalf("no flow for POST /orders: %+v", m.Flows)
	}
	var got []string
	for _, s := range flow.Steps {
		got = append(got, s.From+" -"+s.Call+"-> "+s.To+" "+s.Note)
	}
	want := []string{
		"internal/orders#Service -Add-> internal/store#Store ",
		"internal/store#Store -writes-> table:orders ",
		"internal/store#Store -queues-> job:ship handled later by cmd/worker's ship.Worker.Work, a flow of its own",
	}
	if !slices.Equal(got, want) {
		t.Errorf("steps:\n %s\nwant\n %s", strings.Join(got, "\n "), strings.Join(want, "\n "))
	}
	for _, p := range flow.Participants {
		if p.ID == "table:orders" && p.Element != "tbl:cmd/api/orders" {
			t.Errorf("the table's node: %q", p.Element)
		}
	}
	found := false
	for _, a := range m.Access {
		found = found || a.Container == "app:cmd/api" && a.Component == "cmp:cmd/api/internal/store" && a.Table == "orders" && a.Op == "writes"
	}
	if !found {
		t.Errorf("the store's access to orders: %+v", m.Access)
	}
}

// A package two programs share is a part of each: the api's flow goes through the api's queries,
// not the worker's, and what it writes is the api's access.
func TestASharedPackageIsEachProgramsPart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	writeTree(t, root, map[string]string{
		"go.mod": "module github.com/acme/shop\n\ngo 1.27\n",
		"cmd/api/main.go": "package main\n\nimport (\n\t\"net/http\"\n\n\t\"github.com/acme/shop/internal/queries\"\n)\n\n" +
			"func main() {\n\tq := &queries.Queries{}\n\tmux := http.NewServeMux()\n\tmux.HandleFunc(\"POST /orders\", func(w http.ResponseWriter, r *http.Request) { q.AddOrder() })\n" +
			"\thttp.ListenAndServe(\":8080\", mux)\n}\n",
		"cmd/worker/main.go": "package main\n\nimport (\n\t\"github.com/acme/shop/internal/queries\"\n\t\"github.com/riverqueue/river\"\n)\n\n" +
			"func main() { _ = river.NewWorkers; (&queries.Queries{}).AddOrder() }\n",
		"internal/queries/queries.go": "package queries\n\nimport \"github.com/jackc/pgx/v5\"\n\ntype Queries struct{ db *pgx.Conn }\n\n" +
			"const addOrder = \"INSERT INTO orders (id) VALUES ($1)\"\n\nfunc (q *Queries) AddOrder() { q.db.Exec(nil, addOrder) }\n",
		"db/migrations/001.sql": "CREATE TABLE orders (id int PRIMARY KEY);\n",
	})
	m, err := archdoc.ReadModel(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range m.Flows {
		if f.Entry != "route:cmd/api POST /orders" {
			continue
		}
		for _, p := range f.Participants {
			if p.Kind == "class" && p.Component != "cmp:cmd/api/internal/queries" {
				t.Errorf("the api's flow through %s, in %q", p.Name, p.Component)
			}
		}
	}
	for _, a := range m.Access {
		if a.Container == "app:cmd/api" && a.Component != "cmp:cmd/api/internal/queries" {
			t.Errorf("the api's access is %q's", a.Component)
		}
	}
}
