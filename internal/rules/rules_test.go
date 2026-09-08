package rules

import (
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
		if c.prov.File != "rules.yaml" || c.prov.Line == 0 {
			t.Errorf("%s cites %s, want a line in rules.yaml", c.what, c.prov)
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
