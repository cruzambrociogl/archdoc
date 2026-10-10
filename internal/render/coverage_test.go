package render

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// AC-1, counted: an element a file states at a line, or one a named catalog entry supplies, is
// traced; one only a rule or a model vouches for is not, and neither is a relationship none of
// whose citations is a reading.
func TestTracedCountsWhatAFileOrTheCatalogStates(t *testing.T) {
	at := archdoc.Provenance{File: "compose.yml", Line: 3}
	m := archdoc.Model{
		Nodes: []archdoc.Node{
			{ID: "svc:api", Prov: at},
			{ID: "ext:stripe", Prov: archdoc.Provenance{Origin: archdoc.Catalog, Note: "the stripe package"}},
			{ID: "svc:added-by-rule", Prov: archdoc.Provenance{Origin: archdoc.Rules, File: ".archdoc/rules.yaml", Line: 2}},
			{ID: "svc:nowhere"},
		},
		Edges: []archdoc.Edge{
			{From: "svc:api", To: "ext:stripe", Prov: []archdoc.Provenance{{Origin: archdoc.Semantic, Note: "a model"}, at}},
			{From: "svc:api", To: "svc:nowhere", Prov: []archdoc.Provenance{{Origin: archdoc.Semantic, Note: "a model"}}},
		},
	}
	if traced, of := Traced(m); traced != 3 || of != 6 {
		t.Errorf("traced %d of %d, want 3 of 6", traced, of)
	}
	if got := percent(2499, 2500); got != "99.9%" {
		t.Errorf("2499 of 2500 is %s — it must not round up to 100%%", got)
	}
}

// A container with more components than a diagram shows has a main view: the ones that handle the
// most entries, then the largest, with only the uses among them — and a small container has none.
func TestMainViewKeepsTheComponentsThatHandleTheMost(t *testing.T) {
	at := archdoc.Provenance{File: "compose.yml", Line: 1}
	m := archdoc.Model{Name: "x", Nodes: []archdoc.Node{
		{ID: "svc:api", Name: "api", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
		{ID: "svc:small", Name: "small", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
		{ID: "cmp:small/only", Name: "only", Kind: archdoc.Component, Parent: "svc:small", Files: []string{"a.ts"}, Prov: at},
	}}
	const parts = 26 // a to z: more than half again what a main view holds
	for i := 0; i < parts; i++ {
		id := "cmp:api/c" + string(rune('a'+i))
		m.Nodes = append(m.Nodes, archdoc.Node{ID: id, Name: id[8:], Kind: archdoc.Component, Parent: "svc:api", Lines: 100 - i, Files: []string{id + ".ts"}, Prov: at})
	}
	last := "cmp:api/c" + string(rune('a'+parts-1)) // the smallest, and the only one that handles a route
	m.Entries = []archdoc.Entry{{ID: "route:api GET /x", Kind: "http", Method: "GET", Path: "/x", Container: "svc:api", Component: last, Prov: at}}
	m.Edges = []archdoc.Edge{
		{From: last, To: "cmp:api/ca", Label: "uses", Weight: 2, Prov: []archdoc.Provenance{at}},
		{From: "cmp:api/ca", To: "cmp:api/c" + string(rune('a'+parts-2)), Label: "uses", Weight: 1, Prov: []archdoc.Provenance{at}},
	}
	if got := Mains(m); len(got) != 1 || got[0] != "svc:api" {
		t.Fatalf("mains %v, want only the large container", got)
	}
	v, ok := ViewOf(m, MainPrefix+"svc:api")
	if !ok || len(v.Model.Nodes) != MainParts {
		t.Fatalf("main view: %v, %d components", ok, len(v.Model.Nodes))
	}
	if _, kept := v.Model.Node(last); !kept {
		t.Error("the component that handles a route was left out for being small")
	}
	if len(v.Model.Edges) != 1 || v.Model.Edges[0].From != last {
		t.Errorf("edges %+v, want only the use between two components shown", v.Model.Edges)
	}
	// A component that uses many of the others is drawn with its strongest uses, and no box says
	// its arrows are left out.
	busy := m
	busy.Edges = append([]archdoc.Edge(nil), m.Edges...)
	for i, to := range []string{"cmp:api/cb", "cmp:api/cc", "cmp:api/cd", "cmp:api/ce"} {
		busy.Edges = append(busy.Edges, archdoc.Edge{From: "cmp:api/ca", To: to, Label: "uses", Weight: 10 - i, Prov: []archdoc.Provenance{at}})
	}
	bv, _ := ViewOf(busy, MainPrefix+"svc:api")
	var from []string
	for _, e := range bv.Model.Edges {
		if e.From == "cmp:api/ca" {
			from = append(from, e.To)
		}
	}
	if len(from) != 2 || from[0] != "cmp:api/cb" || from[1] != "cmp:api/cc" {
		t.Errorf("drawn from the busy component: %v, want its two strongest uses", from)
	}
	for _, n := range bv.Model.Nodes {
		if n.UsedBy != 0 || n.UsesMany != 0 {
			t.Errorf("%s is marked as having arrows left out", n.ID)
		}
	}
	if _, ok := ViewOf(m, MainPrefix+"svc:small"); ok {
		t.Error("a container of one component has a main view")
	}
	// Two more than a main view holds is not worth hiding two.
	m.Nodes = m.Nodes[:3+MainParts+2]
	if got := Mains(m); len(got) != 0 {
		t.Errorf("a container of %d components has a main view: %v", MainParts+2, got)
	}
}

// A component's box says what it does in the model's first sentence about it — marked as the
// model's — unless that sentence is about an earlier version of the code.
func TestComponentBoxesCarryTheirCurrentExplanation(t *testing.T) {
	at := archdoc.Provenance{File: "compose.yml", Line: 1}
	by := archdoc.Provenance{Origin: archdoc.Semantic, Note: "a model"}
	m := archdoc.Model{Name: "x", Nodes: []archdoc.Node{
		{ID: "svc:api", Name: "api", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
		{ID: "cmp:api/orders", Name: "orders", Kind: archdoc.Component, Parent: "svc:api", Files: []string{"o.ts"}, Prov: at},
		{ID: "cmp:api/users", Name: "users", Kind: archdoc.Component, Parent: "svc:api", Files: []string{"u.ts"}, Prov: at},
	}, Explanations: []archdoc.Explanation{
		{Element: "cmp:api/orders", Prov: by, Claims: []archdoc.Claim{{Text: "It takes orders."}, {Text: "It stores them."}}},
		{Element: "cmp:api/users", Prov: by, Stale: true, Claims: []archdoc.Claim{{Text: "It was about something else."}}},
	}}
	v, _ := ViewOf(m, ComponentPrefix+"svc:api")
	orders, _ := v.Model.Node("cmp:api/orders")
	users, _ := v.Model.Node("cmp:api/users")
	if orders.Description != "It takes orders." || !orders.DescProv.Origin.Interpretation() {
		t.Errorf("orders says %q, by %q", orders.Description, orders.DescProv.Origin)
	}
	if users.Description != "" {
		t.Errorf("a stale answer is on the box: %q", users.Description)
	}
	if n, _ := m.Node("cmp:api/orders"); n.Description != "" {
		t.Error("the model itself was given the description; only the view should carry it")
	}
}

// A project small enough for one page gets one page: no chapters, nothing to fill in, and on it
// what the project does, reaches and is made of. One with a Compose file, or a second container,
// is not that.
func TestATinyProjectIsOnePage(t *testing.T) {
	at := archdoc.Provenance{File: "report.py", Line: 1}
	m := archdoc.Model{Name: "tool", Nodes: []archdoc.Node{
		{ID: "actor:user", Name: "User", Kind: archdoc.Actor, Evidence: archdoc.Declared, Prov: at},
		{ID: "app:tool", Name: "tool", Kind: archdoc.Application, Technology: "Python", Evidence: archdoc.Declared, Prov: at},
		{ID: "cmp:tool/report", Name: "report", Kind: archdoc.Component, Parent: "app:tool", Dir: "report.py", Files: []string{"report.py"}, Lines: 21, Prov: at},
		{ID: "ext:api.example.com", Name: "api.example.com", Kind: archdoc.External, Evidence: archdoc.Referenced, Prov: at},
	}, Edges: []archdoc.Edge{
		{From: "actor:user", To: "app:tool", Label: "runs", Prov: []archdoc.Provenance{at}},
		{From: "app:tool", To: "ext:api.example.com", Label: "calls", Technology: "https", Prov: []archdoc.Provenance{{File: "report.py", Line: 5}}},
	}, Entries: []archdoc.Entry{{ID: "command:tool python report.py", Kind: "command", Method: "CMD", Path: "python report.py", Prov: archdoc.Provenance{File: "report.py", Line: 20}}}}
	facts := archdoc.FactSet{Sources: []archdoc.Source{{App: ".", Root: ".", Files: []archdoc.SourceFile{{Path: "report.py"}}}}}
	if !Tiny(m) || len(PlanFor(m, facts)) != 0 {
		t.Fatalf("tiny %v, %d sections planned", Tiny(m), len(PlanFor(m, facts)))
	}
	page := Index(m, nil, Meta{})
	for _, want := range []string{"A small project: 1 file, 21 lines of Python", "**run** `python report.py`", "`report.py:20`",
		"calls **api.example.com** over https", "`report.py:5`", "**report** — `report.py`, 21 lines"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not say %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "arc42 section") {
		t.Error("the page lists sections it does not have")
	}
	if len(Stubs(m, PlanFor(m, facts), Meta{})) != 0 {
		t.Error("a tiny project was given sections to fill in")
	}
	deployed := m
	deployed.Nodes = append([]archdoc.Node(nil), m.Nodes...)
	deployed.Nodes[1].ID = "svc:tool"
	if Tiny(deployed) {
		t.Error("a service a Compose file runs is not a tiny project")
	}
	story := Story(m)
	if len(story) < 4 || story[0].Text != "tool is a tool you run: 1 file, 21 lines of Python." ||
		story[1].Text != "You run `python report.py`." || story[2].Text != "It calls api.example.com over https." {
		t.Errorf("story %+v", story)
	}
	if !strings.Contains(page, "## In short") || !strings.Contains(page, "You run `python report.py`.") {
		t.Errorf("the page does not open with the story:\n%s", page)
	}
}

// A context or container view with more external systems than a diagram holds draws the ones the
// most containers reach, then those seen used over those only configured; the rest are counted,
// and the whole view is still there by name.
func TestOverviewDrawsTheExternalsTheMostContainersReach(t *testing.T) {
	at := archdoc.Provenance{File: "compose.yml", Line: 1}
	m := archdoc.Model{Name: "x", Nodes: []archdoc.Node{
		{ID: "svc:a", Name: "a", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
		{ID: "svc:b", Name: "b", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at},
	}}
	const many = externalsOver + 2
	for i := 0; i < many; i++ {
		id := fmt.Sprintf("ext:e%02d", i)
		m.Nodes = append(m.Nodes, archdoc.Node{ID: id, Name: id[4:], Kind: archdoc.External, Evidence: archdoc.Referenced, Prov: at})
		m.Edges = append(m.Edges, archdoc.Edge{From: "svc:a", To: id, Label: archdoc.Configured, Traffic: true, Prov: []archdoc.Provenance{at, at}})
	}
	last := fmt.Sprintf("ext:e%02d", many-1)
	m.Edges = append(m.Edges, archdoc.Edge{From: "svc:b", To: last, Label: "calls", Traffic: true, Prov: []archdoc.Provenance{at}})
	called := fmt.Sprintf("ext:e%02d", many-2)
	m.Edges[many-2].Label = "calls"

	for _, name := range []string{"context", "container"} {
		v, _ := ViewOf(m, name)
		var drawn []string
		for _, n := range v.Model.Nodes {
			if n.Kind == archdoc.External {
				drawn = append(drawn, n.ID)
			}
		}
		if len(drawn) != MainExternals || v.Model.Folded != many-MainExternals {
			t.Fatalf("%s: %d drawn, %d folded", name, len(drawn), v.Model.Folded)
		}
		if !slices.Contains(drawn, last) || !slices.Contains(drawn, called) {
			t.Errorf("%s: %v, want the one two containers reach and the one called among them", name, drawn)
		}
		if slices.Contains(drawn, fmt.Sprintf("ext:e%02d", many-3)) {
			t.Errorf("%s: an external only configured was drawn ahead of the others by its name", name)
		}
		all, ok := ViewOf(m, name+AllSuffix)
		if !ok || all.Model.Folded != 0 || len(all.Model.Nodes) < many {
			t.Errorf("%s: the whole view has %d elements", name, len(all.Model.Nodes))
		}
	}
}

// A stored layout is drawn from only when it places every element of the view and nothing else: one
// from before the view changed would draw an arrow to a box that is not there.
func TestPlacesEveryElementOrIsNotReused(t *testing.T) {
	view := archdoc.Model{Nodes: []archdoc.Node{{ID: "svc:a"}, {ID: "ext:b"}}}
	l := archdoc.Layout{Boxes: []archdoc.Box{{ID: "svc:a"}, {ID: "ext:b"}}}
	if !Places(l, view) {
		t.Error("a layout of exactly the view's elements was refused")
	}
	l.Boxes[1].ID = "ext:gone"
	if Places(l, view) {
		t.Error("a layout with a box for an element no longer drawn, and none for one that is, was reused")
	}
}
