package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// declared collects the identifiers a text declares, and checks every relationship names two of them.
func checkRefs(t *testing.T, name, text string, decl, rel *regexp.Regexp) {
	t.Helper()
	ids := map[string]bool{}
	for _, m := range decl.FindAllStringSubmatch(text, -1) {
		ids[m[1]] = true
	}
	rels := rel.FindAllStringSubmatch(text, -1)
	if len(rels) == 0 {
		t.Errorf("%s: no relationships", name)
	}
	for _, m := range rels {
		if !ids[m[1]] || !ids[m[2]] {
			t.Errorf("%s: a relationship %s -> %s names something not declared", name, m[1], m[2])
		}
	}
}

func balanced(t *testing.T, name, text string) {
	t.Helper()
	if strings.Count(text, "{") != strings.Count(text, "}") {
		t.Errorf("%s: %d { and %d }", name, strings.Count(text, "{"), strings.Count(text, "}"))
	}
	if strings.Count(text, `"`)%2 != 0 {
		t.Errorf("%s: an unclosed quote", name)
	}
}

func textFixture() core.Model {
	m := fixture()
	m.Nodes = append(m.Nodes,
		core.Node{ID: "cmp:api/users", Name: "users", Kind: core.Component, Parent: "svc:api", Evidence: core.Declared, Prov: core.Provenance{File: "api/users.ts", Line: 1}},
		core.Node{ID: "cmp:api/orders", Name: `orders "v2"`, Kind: core.Component, Parent: "svc:api", Evidence: core.Declared, Prov: core.Provenance{File: "api/orders.ts", Line: 1}})
	m.Edges = append(m.Edges,
		core.Edge{From: "cmp:api/orders", To: "cmp:api/users", Label: "uses", Weight: 2, Prov: []core.Provenance{{File: "api/orders.ts", Line: 3}}},
		core.Edge{From: "svc:api", To: "svc:db", Label: "connects to", Technology: "postgres", Traffic: true, Prov: []core.Provenance{at(14)}}) // the same arrow said twice
	return m.Normalise()
}

// Every C4 view writes a Structurizr workspace whose relationships all name declared elements,
// once each, with the view of the right kind; a view that is not C4 writes none.
func TestStructurizrForEachC4View(t *testing.T) {
	m := textFixture()
	decl := regexp.MustCompile(`(?m)^\s*(e_\w+) = (?:person|softwareSystem|container|component)\b`)
	rel := regexp.MustCompile(`(?m)^\s*(e_\w+) -> (e_\w+) `)
	want := map[string]string{"context": "systemContext ", "container": "container e_the_system ", Deployment: "deployment e_the_system ", "component:svc:api": "component e_the_container "}
	for _, v := range Views(m) {
		text, ok := Structurizr(m, v)
		if !isC4(v) {
			if ok {
				t.Errorf("%s is not C4 and wrote Structurizr", v.Name)
			}
			continue
		}
		balanced(t, v.Name, text)
		checkRefs(t, v.Name, text, decl, rel)
		if k, ok := want[v.Name]; ok && !strings.Contains(text, k) {
			t.Errorf("%s: no %q view", v.Name, k)
		}
		if strings.Count(text, `e_svc_api -> e_svc_db "connects to" "postgres"`) > 1 {
			t.Errorf("%s: the same relationship twice", v.Name)
		}
	}
	dep, _ := ViewOf(m, Deployment)
	text, _ := Structurizr(m, dep)
	if !strings.Contains(text, "containerInstance e_svc_api") || !strings.Contains(text, `deploymentNode "docker-compose.yml"`) {
		t.Errorf("deployment: %s", text)
	}
	comp, _ := ViewOf(m, "component:svc:api")
	text, _ = Structurizr(m, comp)
	if !strings.Contains(text, `component "orders 'v2'"`) {
		t.Errorf("a quote in a name is not made safe: %s", text)
	}
}

// Every C4 view writes C4-PlantUML with the library it needs, its title, its elements inside their
// boundary and relationships that name declared elements.
func TestPlantUMLForEachC4View(t *testing.T) {
	m := textFixture()
	decl := regexp.MustCompile(`(?m)^\s*(?:Person|System|System_Ext|Container|ContainerDb|ContainerQueue|Component|Container_Ext|ContainerDb_Ext|Component_Ext)\((e_\w+),`)
	rel := regexp.MustCompile(`(?m)^Rel\((e_\w+), (e_\w+),`)
	libs := map[string]string{"context": "C4_Context", "container": "C4_Container", Deployment: "C4_Deployment", "component:svc:api": "C4_Component"}
	for _, v := range Views(m) {
		text, ok := PlantUML(m, v)
		if !isC4(v) {
			if ok {
				t.Errorf("%s is not C4 and wrote PlantUML", v.Name)
			}
			continue
		}
		balanced(t, v.Name, text)
		checkRefs(t, v.Name, text, decl, rel)
		if !strings.HasPrefix(text, "@startuml\n") || !strings.HasSuffix(text, "@enduml\n") {
			t.Errorf("%s: not one diagram", v.Name)
		}
		if lib, ok := libs[v.Name]; ok && !strings.Contains(text, "!include <C4/"+lib+">") {
			t.Errorf("%s: not %s", v.Name, lib)
		}
		if !strings.Contains(text, "title "+HeadingOf(m, v).Title) {
			t.Errorf("%s: no title", v.Name)
		}
	}
	dep, _ := ViewOf(m, Deployment)
	text, _ := PlantUML(m, dep)
	if !strings.Contains(text, `Deployment_Node(e_the_node, "docker-compose.yml", "Docker Compose") {`) || !strings.Contains(text, "ContainerDb(e_svc_db") {
		t.Errorf("deployment: %s", text)
	}
}

// The same model writes the same text (AC-7).
func TestC4TextIsTheSameEachTime(t *testing.T) {
	m := textFixture()
	for _, v := range Views(m) {
		a, _ := Structurizr(m, v)
		b, _ := Structurizr(m, v)
		c, _ := PlantUML(m, v)
		d, _ := PlantUML(m, v)
		if a != b || c != d {
			t.Errorf("%s differs between two writes", v.Name)
		}
	}
}

// A figure carries its title, its plain name and a key of exactly the styles it draws (rule 23).
func TestFigureHasItsTitleAndKey(t *testing.T) {
	m := fixture()
	for _, look := range []Look{Classic, Archdoc} {
		for _, v := range Views(m) {
			l := core.Layout{Width: 100, Height: 80}
			svg := Figure(m, v, l, look)
			h := HeadingOf(m, v)
			if !strings.Contains(svg, esc(h.Title)) || !strings.Contains(svg, esc(h.Plain)) {
				t.Errorf("%s (%s): no title", v.Name, look)
			}
			for _, k := range KeyOf(v.Model) {
				if !strings.Contains(svg, esc(k.Label)) {
					t.Errorf("%s (%s): the key leaves out %q", v.Name, look, k.Label)
				}
			}
		}
	}
	ctx, _ := ViewOf(m, "context")
	var styles []string
	for _, k := range KeyOf(ctx.Model) {
		styles = append(styles, k.Style)
	}
	if got := strings.Join(styles, ","); got != "person,system,external,arrow" {
		t.Errorf("the context key lists %s", got)
	}
	if h := HeadingOf(m, ctx); h.Title != "System context diagram for example" || !h.C4 {
		t.Errorf("context heading %+v", h)
	}
	dep, _ := ViewOf(m, Deployment)
	if h := HeadingOf(m, dep); h.Title != "Deployment diagram for example · docker-compose.yml" || h.Plain != "Where it runs" {
		t.Errorf("deployment heading %+v", h)
	}
}
