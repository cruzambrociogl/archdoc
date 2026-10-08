package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

const fixture = "../../testdata/rules"

// regenerate runs the whole pipeline from disk, exactly as `archdoc generate` does. AC-5 is
// about surviving *regeneration*, so a test that applied rules to a cached model would prove
// nothing.
func regenerate(t *testing.T) archdoc.Model {
	t.Helper()

	facts, err := extract.Scan(fixture)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	f, err := Load(facts.Root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	ops, findings := f.Compile(facts, model.Derive(facts))
	if !findings.OK() {
		t.Fatalf("rules did not compile:\n%s", findings.Error())
	}

	out, applied := validate.Apply(model.Derive(facts), ops)
	if !applied.OK() {
		t.Fatalf("rules produced an invalid model:\n%s", applied.Error())
	}
	return out
}

// AC-5 — apply ten rules, regenerate from scratch, and all ten are still in effect.
//
// The criterion exists because corrections that do not survive regeneration are worse than no
// corrections: a person fixes the diagram, the next run silently undoes it, and they stop
// trusting the tool. Running the whole pipeline twice is the only honest way to test it.
func TestTenRulesSurviveRegeneration(t *testing.T) {
	for run := 1; run <= 2; run++ {
		m := regenerate(t)

		api, ok := m.Node("svc:api")
		if !ok {
			t.Fatalf("run %d: svc:api missing", run)
		}
		admin, ok := m.Node("svc:admin")
		if !ok {
			t.Fatalf("run %d: svc:admin missing", run)
		}
		db, ok := m.Node("svc:db")
		if !ok {
			t.Fatalf("run %d: svc:db missing", run)
		}

		checks := []struct {
			rule string
			got  string
			want string
		}{
			{"1 rename by name", api.Name, "API"},
			{"2 description", api.Description, "Serves the public HTTP API"},
			{"3 technology", api.Technology, "Go 1.27"},
			{"4 reclassify", string(admin.Kind), string(archdoc.Proxy)},
			{"5 match on image", db.Description, "Primary relational store"},
			{"6 match on kind", db.Parent, "persistence"},
			{"9 match on glob", admin.Technology, "Node.js"},
			{"10 rename the aliased service", admin.Name, "Admin Console"},
		}

		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("run %d: rule %s — got %q, want %q", run, c.rule, c.got, c.want)
			}
		}

		// 7 — a relationship extraction cannot see. Immich's case exactly.
		found := false
		for _, e := range m.Edges {
			if e.From == "svc:api" && e.To == "svc:db" && e.Label == "reads and writes" {
				found = true
			}
		}
		if !found {
			t.Errorf("run %d: rule 7 — the asserted edge is missing", run)
		}

		// 8 — an excluded element, and nothing left pointing at it.
		if _, ok := m.Node("svc:edge"); ok {
			t.Errorf("run %d: rule 8 — the excluded element is still present", run)
		}
		for _, e := range m.Edges {
			if e.From == "svc:edge" || e.To == "svc:edge" {
				t.Errorf("run %d: rule 8 — an edge to the excluded element survived", run)
			}
		}
	}
}

// A correction is a claim, and P1 does not exempt it. Every value a rule sets must cite the line
// the person wrote it on, exactly as a fact cites the line a file declared it on.
func TestCorrectionsCiteTheirLine(t *testing.T) {
	m := regenerate(t)

	api, _ := m.Node("svc:api")

	for _, c := range []struct {
		what string
		prov archdoc.Provenance
	}{
		{"description", api.DescProv},
		{"technology", api.TechProv},
	} {
		if !c.prov.Known() {
			t.Errorf("%s has no provenance", c.what)
		}
		if c.prov.Origin != archdoc.Rules {
			t.Errorf("%s origin is %q, want rules", c.what, c.prov.Origin)
		}
		if c.prov.File != Name || c.prov.Line == 0 {
			t.Errorf("%s cites %s, want a line in %s", c.what, c.prov, Name)
		}
	}
}

// RUL-05. A rule that matches nothing is usually a typo or a service that was renamed, and it
// fails silently unless something says so.
func TestUnreachableRuleIsReported(t *testing.T) {
	f, err := Parse([]byte(`
rules:
  - match: { name: nothing-called-this }
    set: { name: Ghost }
`), "rules.yaml")
	if err != nil {
		t.Fatal(err)
	}

	ops, res := f.Compile(nil, archdoc.Model{
		Nodes: []archdoc.Node{{ID: "svc:api", Name: "api"}},
	})

	if len(ops) != 0 {
		t.Errorf("a rule matching nothing produced %d operation(s)", len(ops))
	}
	if len(res.Warnings()) != 1 {
		t.Fatalf("expected one warning, got %d", len(res.Warnings()))
	}
	// A warning, not an error: an unreachable rule is a mistake in the file, not a reason to
	// refuse to document the repository.
	if !res.OK() {
		t.Error("an unreachable rule blocked the run")
	}
}

// A later rule silently overriding an earlier one is the kind of invisible wrongness this
// project exists to prevent. Later still wins — an override is often deliberate — but the reader
// is told which rule lost.
func TestConflictingRulesAreReported(t *testing.T) {
	f, err := Parse([]byte(`
rules:
  - match: { kind: application }
    set: { technology: Go }
  - match: { name: api }
    set: { technology: Rust }
`), "rules.yaml")
	if err != nil {
		t.Fatal(err)
	}

	m := archdoc.Model{Nodes: []archdoc.Node{
		{ID: "svc:api", Name: "api", Kind: archdoc.Application},
	}}

	ops, res := f.Compile(nil, m)

	if len(res.Warnings()) == 0 {
		t.Error("a rule overriding another was not reported")
	}
	if len(ops) != 2 {
		t.Fatalf("got %d operations, want both rules compiled", len(ops))
	}
	if ops[len(ops)-1].Value != "Rust" {
		t.Errorf("the later rule did not win: %+v", ops)
	}
}

// A repository with no corrections is the normal case. Requiring the file would be a barrier to
// the first run rather than a feature.
func TestMissingRulesFileIsNotAnError(t *testing.T) {
	f, err := Load("../../testdata/gateway")
	if err != nil {
		t.Fatalf("a missing rules.yaml was treated as an error: %v", err)
	}
	if len(f.Rules) != 0 {
		t.Errorf("got %d rules from a repository that has none", len(f.Rules))
	}
}

// AC-7 reaches here too: rules are compiled by iterating a map of fields.
func TestCompilationIsDeterministic(t *testing.T) {
	facts, err := extract.Scan(fixture)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Load(facts.Root)
	if err != nil {
		t.Fatal(err)
	}

	first, _ := f.Compile(facts, model.Derive(facts))

	for range 10 {
		got, _ := f.Compile(facts, model.Derive(facts))
		if len(got) != len(first) {
			t.Fatalf("got %d operations, want %d", len(got), len(first))
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("operation %d differs between runs:\n %+v\n %+v", i, got[i], first[i])
			}
		}
	}
}

// Rules live in .archdoc/ since 5 Oct 2026. A repository that still keeps them at the root is read,
// and told to move them; when both exist, .archdoc/ wins and the root file is reported as ignored.
func TestRulesMovedIntoArchdocDir(t *testing.T) {
	write := func(root, rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	body := "rules:\n  - match: { name: api }\n    set: { kind: proxy }\n"

	legacy := t.TempDir()
	write(legacy, LegacyName, body)
	f, err := Load(legacy)
	if err != nil || !f.Legacy || f.Path != LegacyName || len(f.Rules) != 1 {
		t.Errorf("legacy location: %+v %v", f, err)
	}

	both := t.TempDir()
	write(both, LegacyName, body)
	write(both, Name, body+"  - match: { name: db }\n    exclude: true\n")
	f, err = Load(both)
	if err != nil || f.Legacy || !f.Shadowed || f.Path != Name || len(f.Rules) != 2 {
		t.Errorf("both locations: %+v %v", f, err)
	}

	none := t.TempDir()
	if f, err := Load(none); err != nil || f.Path != Name || len(f.Rules) != 0 {
		t.Errorf("no rules: %+v %v", f, err)
	}
}

// Rules for what the code declares (F-33): a component by the container it is in, a route by its
// method and path — and a rule written for a service does not reach inside one.
func TestRulesForComponentsAndRoutes(t *testing.T) {
	p := archdoc.Provenance{File: "x", Line: 1}
	m := archdoc.Model{
		Nodes: []archdoc.Node{
			{ID: "svc:server", Name: "server", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: p},
			{ID: "svc:redis", Name: "redis", Kind: archdoc.Datastore, Evidence: archdoc.Declared, Prov: p},
			{ID: "cmp:server/services", Name: "services", Kind: archdoc.Component, Evidence: archdoc.Declared, Parent: "svc:server", Prov: p},
			{ID: "cmp:server/redis", Name: "redis", Kind: archdoc.Component, Evidence: archdoc.Declared, Parent: "svc:server", Prov: p},
		},
		Entries: []archdoc.Entry{
			{ID: "route:server GET /api/ping", Method: "GET", Path: "/api/ping", Container: "svc:server", Component: "cmp:server/services", Prov: p},
			{ID: "route:server GET /api/albums", Method: "GET", Path: "/api/albums", Container: "svc:server", Component: "cmp:server/services", Prov: p},
		},
		Flows: []archdoc.Flow{{Entry: "route:server GET /api/ping"}},
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".archdoc"), 0o755)
	os.WriteFile(filepath.Join(root, Name), []byte(`rules:
  - match: { name: services, in: server }
    set: { description: The business logic. }
  - match: { name: redis }
    exclude: true
  - route: "GET /api/ping"
    exclude: true
  - route: "/api/alb*"
    set: { summary: Lists the albums. }
`), 0o644)
	f, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	ops, findings := f.Compile(nil, m)
	if !findings.OK() {
		t.Fatalf("%s", findings.Error())
	}
	out, applied := validate.Apply(m, ops)
	if !applied.OK() {
		t.Fatalf("%s", applied.Error())
	}
	if n, _ := out.Node("cmp:server/services"); n.Description != "The business logic." || n.DescProv.Origin != archdoc.Rules {
		t.Errorf("services %+v", n)
	}
	if _, ok := out.Node("svc:redis"); ok {
		t.Error("the redis service was not excluded")
	}
	if _, ok := out.Node("cmp:server/redis"); !ok {
		t.Error("a rule naming the redis service reached the component of the same name")
	}
	if len(out.Entries) != 1 || out.Entries[0].Summary != "Lists the albums." || out.Entries[0].SummaryProv.Origin != archdoc.Rules || len(out.Flows) != 0 {
		t.Errorf("entries %+v, flows %d", out.Entries, len(out.Flows))
	}

	// Excluding a container takes with it everything read from its code.
	gone, res := validate.Apply(m, []archdoc.Op{{Kind: archdoc.Exclude, Target: "svc:server", Origin: archdoc.Rules, Prov: p}})
	if !res.OK() || len(gone.Nodes) != 1 || len(gone.Entries) != 0 || len(gone.Flows) != 0 {
		t.Errorf("after excluding the container: %d nodes, %d entries, %d flows, %s", len(gone.Nodes), len(gone.Entries), len(gone.Flows), res.Error())
	}
}
