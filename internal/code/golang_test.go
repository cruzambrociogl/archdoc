package code

import (
	"slices"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// goShop is a module with two programs: an API, whose routes are spread over a ServeMux and chi,
// and a worker, which takes River's jobs. What each reads is its own main package and what it
// imports — and nothing else of the module.
func goShop(t *testing.T) string {
	return tree(t, map[string]string{
		"go.mod": "module github.com/acme/shop\n\ngo 1.27\n",
		"cmd/api/main.go": `package main

import (
	"net/http"

	"github.com/acme/shop/internal/orders"
	"github.com/acme/shop/internal/web"
)

func main() {
	mux := http.NewServeMux()
	s := &orders.Service{}
	mux.HandleFunc("GET /orders/{id}", s.Get)
	mux.HandleFunc("POST /orders", gated(s.Create))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {})
	mux.Handle("/", web.Frontend())
	http.ListenAndServe(":8080", mux)
}

func gated(h http.HandlerFunc) http.HandlerFunc { return h }
`,
		"internal/orders/orders.go": `package orders

import (
	"net/http"

	"github.com/acme/shop/internal/build"
)

// Service answers for orders.
type Service struct{}

// Get answers one order, by its id. It reads the store.
func (s *Service) Get(w http.ResponseWriter, r *http.Request) { _ = build.Version }

// Create places an order.
func (s *Service) Create(w http.ResponseWriter, r *http.Request) {}

const payments = "https://api.payments.io/v1"

func charge() { http.Get(payments + "/charge") }

func refund(url string) { http.Get(url) }
`,
		"internal/build/version.go": "package build\n\nvar Version = \"1\"\n",
		"internal/build/other.go":   "package build\n\nfunc Other() {}\n",
		"internal/web/web.go": `package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Frontend() http.Handler { return nil }

func Admin() http.Handler {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		r.Get("/users", listUsers)
		r.With(audit).Post("/users/{id}/ban", ban)
	})
	r.Get("/status", listUsers)
	return r
}

// listUsers lists every user.
func listUsers(w http.ResponseWriter, r *http.Request) {}
func ban(w http.ResponseWriter, r *http.Request)       {}
func audit(h http.Handler) http.Handler                { return h }
`,
		"internal/jobs/jobs.go": `package jobs

import "github.com/riverqueue/river"

type Ship struct{}

func (Ship) Kind() string { return "ship" }

type Bill struct{}

func (Bill) Kind() string { return "bill" }

var _ river.JobArgs = Ship{}
`,
		"internal/shipping/worker.go": `package shipping

import (
	"context"

	"github.com/acme/shop/internal/jobs"
	"github.com/riverqueue/river"
)

type Worker struct{}

func (w *Worker) Work(ctx context.Context, job *river.Job[jobs.Ship]) error { return nil }

// BillWorker exists but no program registers it.
type BillWorker struct{}

func (w *BillWorker) Work(ctx context.Context, job *river.Job[jobs.Bill]) error { return nil }
`,
		"cmd/worker/main.go": `package main

import (
	"github.com/acme/shop/internal/shipping"
	"github.com/riverqueue/river"
)

func main() {
	workers := river.NewWorkers()
	river.AddWorker(workers, &shipping.Worker{})
}
`,
		"internal/unused/unused.go":      "package unused\n",
		"internal/orders/orders_test.go": "package orders\n\nimport \"github.com/acme/shop/internal/unused\"\n",
		"testdata/fixture/fixture.go":    "package fixture\n",
	})
}

func goPaths(src core.Source) []string {
	var out []string
	for _, f := range src.Files {
		out = append(out, f.Path)
	}
	return out
}

func TestAGoProgramIsItsPackages(t *testing.T) {
	root := goShop(t)
	api, ok := Read(root, core.App{Name: "api", Dir: "cmd/api", Manifest: "go.mod", Language: "Go"}, nil)
	if !ok {
		t.Fatal("nothing read")
	}
	want := []string{"cmd/api/main.go", "internal/build/other.go", "internal/build/version.go", "internal/orders/orders.go", "internal/web/web.go"}
	if got := goPaths(api); !slices.Equal(got, want) {
		t.Errorf("the API's files: %v\n want %v — its packages and what they import, a package named build among them; not the worker's, not a test's", got, want)
	}
	if api.Root != "." || api.Module != "github.com/acme/shop" || api.App != "cmd/api" {
		t.Errorf("root %q, module %q, app %q", api.Root, api.Module, api.App)
	}
	worker, _ := Read(root, core.App{Name: "worker", Dir: "cmd/worker", Manifest: "go.mod", Language: "Go"}, nil)
	if got := goPaths(worker); !slices.Equal(got, []string{"cmd/worker/main.go", "internal/jobs/jobs.go", "internal/shipping/worker.go"}) {
		t.Errorf("the worker's files: %v", got)
	}

	// An import is tied to the file that declares what is used of it, or — nothing named — the
	// package's first file; another module's is a package.
	file := func(src core.Source, p string) core.SourceFile {
		for _, f := range src.Files {
			if f.Path == p {
				return f
			}
		}
		t.Fatalf("no %s", p)
		return core.SourceFile{}
	}
	var got []string
	for _, imp := range file(api, "internal/orders/orders.go").Imports {
		got = append(got, string(imp.How)+" "+imp.Spec+" "+imp.Target+" "+strings.Join(imp.Names, ","))
	}
	wantImports := []string{"package net/http  ", "module github.com/acme/shop/internal/build internal/build/version.go Version"}
	if !slices.Equal(got, wantImports) {
		t.Errorf("orders.go imports %q\n want %q", got, wantImports)
	}
}

func TestGoRoutes(t *testing.T) {
	root := goShop(t)
	api, _ := Read(root, core.App{Name: "api", Dir: "cmd/api", Manifest: "go.mod", Language: "Go"}, nil)
	var got []string
	summaries := map[string]string{}
	for _, f := range api.Files {
		for _, h := range f.Handles {
			got = append(got, h.Method+" "+h.Path+" "+h.Handler+" "+h.Prov.String())
			summaries[h.Method+" "+h.Path] = h.Summary
		}
	}
	want := []string{
		"GET /orders/{id} orders.Service.Get cmd/api/main.go:13",
		"POST /orders orders.Service.Create cmd/api/main.go:14",
		"GET /healthz main.main.func cmd/api/main.go:15",
		"ANY / web.Frontend cmd/api/main.go:16",
		"GET /admin/users web.listUsers internal/web/web.go:14",
		"POST /admin/users/{id}/ban web.ban internal/web/web.go:15",
		"GET /status web.listUsers internal/web/web.go:17",
	}
	if !slices.Equal(got, want) {
		t.Errorf("routes:\n %s\nwant\n %s", strings.Join(got, "\n "), strings.Join(want, "\n "))
	}
	if s := summaries["GET /orders/{id}"]; s != "Get answers one order, by its id." {
		t.Errorf("the handler's doc comment, its first sentence: %q", s)
	}
	if s := summaries["GET /admin/users"]; s != "listUsers lists every user." {
		t.Errorf("a function's doc comment: %q", s)
	}
}

func TestGoJobsAddressesAndCalls(t *testing.T) {
	root := goShop(t)
	worker, _ := Read(root, core.App{Name: "worker", Dir: "cmd/worker", Manifest: "go.mod", Language: "Go"}, nil)
	var jobs []string
	for _, f := range worker.Files {
		for _, j := range f.Jobs {
			jobs = append(jobs, j.Name+" "+j.Handler+" "+j.Prov.String()+" "+j.NameProv.String())
		}
	}
	// The bill job's worker is declared, but no program registers it: it is no job of this one.
	if want := []string{"ship shipping.Worker.Work internal/shipping/worker.go:12 internal/jobs/jobs.go:7"}; !slices.Equal(jobs, want) {
		t.Errorf("jobs %q, want %q", jobs, want)
	}

	api, _ := Read(root, core.App{Name: "api", Dir: "cmd/api", Manifest: "go.mod", Language: "Go"}, nil)
	for _, f := range api.Files {
		if f.Path != "internal/orders/orders.go" {
			continue
		}
		if len(f.Hosts) != 1 || f.Hosts[0].Host != "api.payments.io" || f.Hosts[0].Key != "payments" {
			t.Errorf("hosts %+v — the address a constant holds, by its name", f.Hosts)
		}
		// Both requests make their address at run time; the constant the first starts from is an
		// address of its own, above.
		if len(f.Calls) != 2 || f.Calls[0].Target != `payments + "/charge"` || f.Calls[1].Callee != "http.Get" || f.Calls[1].Target != "url" {
			t.Errorf("calls %+v — the requests whose address is computed", f.Calls)
		}
	}
}

func TestGoCommands(t *testing.T) {
	root := tree(t, map[string]string{
		"go.mod": "module github.com/acme/tool\n",
		"main.go": `package main

import (
	"os"

	"github.com/spf13/cobra"
)

type command struct {
	name, args, summary string
	run                 func()
}

var commands = []command{
	{"init", "<path>", "create the configuration", nil},
	{"serve", "", "serve the app on a port", nil},
}

func main() {
	root := &cobra.Command{Use: "tool"}
	root.AddCommand(&cobra.Command{Use: "deploy [env]", Short: "Deploy to an environment. Then wait."})
	switch os.Args[1] {
	case "version", "-h":
	}
}
`,
	})
	src, _ := Read(root, core.App{Name: "tool", Dir: ".", Manifest: "go.mod", Language: "Go"}, nil)
	var got []string
	for _, f := range src.Files {
		for _, c := range f.Commands {
			got = append(got, c.Name+": "+c.Summary)
		}
	}
	want := []string{"init: create the configuration", "serve: serve the app on a port", "tool: ", "deploy: Deploy to an environment.", "version: "}
	if !slices.Equal(got, want) {
		t.Errorf("commands %q\n want %q", got, want)
	}
}
