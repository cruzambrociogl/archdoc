package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/model"
)

// The drift set (F-39): the fixtures AC-6 is scored against. Each case changes the repository in
// testdata/drift/base in one way a real change does, and names every change archdoc must then
// report, under its class — and nothing else, so a change that is not one is scored too. The whole
// pipeline runs on both sides, from files on disk: this is the comparison `archdoc diff` makes.

type edit func(t *testing.T, dir string)

// replace rewrites one file, failing when the text it expects is not there — a case that silently
// changes nothing would pass for the wrong reason.
func replace(file, old, new string) edit {
	return func(t *testing.T, dir string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(file))
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), old) {
			t.Fatalf("%s does not contain %q", file, old)
		}
		if err := os.WriteFile(p, []byte(strings.ReplaceAll(string(b), old, new)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func create(file, content string) edit {
	return func(t *testing.T, dir string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(file))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const compose = "docker-compose.yml"

var driftSet = []struct {
	name  string
	edits []edit
	want  []string // "class element", in the order Drift lists them
}{
	{"a service appears",
		[]edit{replace(compose, "  db:\n", "  metrics:\n    image: prom/prometheus:v2\n\n  db:\n")},
		[]string{"added svc:metrics"}},

	{"a service disappears, and what it reached with it",
		[]edit{replace(compose, "  worker:\n    image: example/worker:1.0\n    depends_on:\n      - cache\n    environment:\n      SEARCH_URL: http://search:9200\n\n", "")},
		[]string{"removed svc:worker", "removed svc:worker → svc:cache", "removed svc:worker → svc:search"}},

	{"a service is renamed: one change, not a removal, an addition and a rewiring",
		[]edit{
			replace(compose, "  cache:\n    image: redis:7", "  store:\n    image: redis:7"),
			replace(compose, "      - cache\n", "      - store\n"),
			replace(compose, "CACHE_HOST: cache", "CACHE_HOST: store"),
		},
		[]string{"renamed svc:store"}},

	{"a container with code is renamed: its components, tables and routes follow it",
		[]edit{replace(compose, "  api:\n    build: ./api", "  backend:\n    build: ./api")},
		[]string{"renamed svc:backend"}},

	{"a dependency appears",
		[]edit{replace(compose, "      - cache\n    environment:\n      SEARCH_URL", "      - cache\n      - db\n    environment:\n      SEARCH_URL")},
		[]string{"added svc:worker → svc:db"}},

	{"a dependency disappears",
		[]edit{replace(compose, "    depends_on:\n      - cache\n    environment:\n      SEARCH_URL", "    environment:\n      SEARCH_URL")},
		[]string{"removed svc:worker → svc:cache"}},

	{"a connection changes protocol",
		[]edit{replace(compose, "postgres://shop:secret@db:5432/shop", "mysql://shop:secret@db:3306/shop")},
		[]string{"protocol-changed svc:api → svc:db"}},

	{"a service leaves the repository and is still called: it crossed the boundary",
		[]edit{replace(compose, "  search:\n    image: elasticsearch:8.13.0\n\n", "")},
		[]string{"re-bounded ext:search"}},

	{"a store changes technology",
		[]edit{replace(compose, "image: postgres:16", "image: mysql:8")},
		[]string{"changed svc:db"}},

	{"a route appears",
		[]edit{replace("api/src/order.controller.ts", "  @Post()\n", "  @Delete(':id')\n  remove() {}\n\n  @Post()\n")},
		[]string{"added route:api DELETE /orders/:id"}},

	{"a route disappears",
		[]edit{replace("api/src/order.controller.ts", "  @Post()\n  create() {\n    return this.service.create();\n  }\n", "")},
		[]string{"removed route:api POST /orders"}},

	{"a column appears",
		[]edit{replace("api/src/order.table.ts", "  @Column({ type: 'numeric' })", "  @Column({ type: 'text' })\n  status!: string;\n\n  @Column({ type: 'numeric' })")},
		[]string{"changed tbl:api/orders"}},

	{"a column changes type",
		[]edit{replace("api/src/order.table.ts", "type: 'numeric'", "type: 'integer'")},
		[]string{"changed tbl:api/orders"}},

	{"a table is renamed",
		[]edit{replace("api/src/order.table.ts", "name: 'orders'", "name: 'purchases'")},
		[]string{"renamed tbl:api/purchases"}},

	{"a table appears, with the key that ties it in",
		[]edit{create("api/src/invoice.table.ts", "import { OrderTable } from './order.table';\n\n@Table({ name: 'invoices' })\nexport class InvoiceTable {\n  @PrimaryGeneratedColumn()\n  id!: string;\n\n  @ForeignKeyColumn(() => OrderTable)\n  orderId!: string;\n}\n")},
		[]string{"added cmp:api/invoice", "added cmp:api/invoice → cmp:api/order", "added tbl:api/invoices", "added tbl:api/invoices → tbl:api/orders"}},

	{"nothing changes: comments, blank lines and the order services are written in",
		[]edit{
			replace(compose, "name: shop\n", "name: shop\n# reviewed\n\n"),
			replace(compose, "  db:\n    image: postgres:16\n\n  cache:\n    image: redis:7\n", "  cache:\n    image: redis:7\n\n  db:\n    image: postgres:16\n"),
			replace("api/src/order.service.ts", "  list() {", "  // every order\n  list() {"),
		},
		nil},
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// AC-6 — every drift in the set detected and classified.
func TestDriftSetIsDetectedAndClassified(t *testing.T) {
	base := filepath.Join("..", "..", "testdata", "drift", "base")
	before, err := modelOf(base)
	if err != nil {
		t.Fatal(err)
	}
	passed := 0
	for _, c := range driftSet {
		// Both sides are read from a directory of the same name, as two commits of one repository are.
		dir := filepath.Join(t.TempDir(), "base")
		copyTree(t, base, dir)
		for _, e := range c.edits {
			e(t, dir)
		}
		after, err := modelOf(dir)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		var got []string
		for _, d := range model.Compare(before, after).Drift() {
			got = append(got, d.Class+" "+d.Element)
		}
		if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, got, c.want)
			continue
		}
		passed++
	}
	t.Logf("AC-6: %d of %d drifts detected and classified", passed, len(driftSet))
}

// F-37 — the same comparison between two real commits, through the command.
func TestDiffBetweenTwoCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := filepath.Join(t.TempDir(), "shop")
	copyTree(t, filepath.Join("..", "..", "testdata", "drift", "base"), dir)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "first")
	replace(compose, "  cache:\n    image: redis:7", "  store:\n    image: redis:7")(t, dir)
	replace(compose, "      - cache\n", "      - store\n")(t, dir)
	replace(compose, "CACHE_HOST: cache", "CACHE_HOST: store")(t, dir)
	git("commit", "-q", "-am", "second")
	replace(compose, "image: postgres:16", "image: mysql:8")(t, dir)

	var out bytes.Buffer
	if err := run([]string{"diff", dir, "HEAD~1", "HEAD"}, &out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "renamed") || !strings.Contains(got, "svc:store") || strings.Contains(got, "removed") || strings.Contains(got, "mysql") {
		t.Errorf("between the two commits:\n%s", got)
	}
	out.Reset()
	if err := run([]string{"diff", dir, "HEAD"}, &out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "the working tree") || !strings.Contains(got, "svc:db") || strings.Contains(got, "renamed") {
		t.Errorf("since the last commit:\n%s", got)
	}
	if err := run([]string{"diff", dir, "no-such-commit"}, &out); err == nil {
		t.Error("an unknown revision was accepted")
	}
}
