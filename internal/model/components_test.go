package model

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// codeFacts builds an application at dir with the given files; imports maps a file to the files it
// imports, each on its own line.
func codeFacts(dir, root string, files []string, imports map[string][]string) *archdoc.FactSet {
	src := archdoc.Source{App: dir, Root: root}
	for _, f := range files {
		sf := archdoc.SourceFile{Path: f, Language: "TypeScript", Lines: 10}
		for i, t := range imports[f] {
			sf.Imports = append(sf.Imports, archdoc.Import{Spec: t, Target: t, How: archdoc.ByAlias,
				Prov: archdoc.Provenance{File: f, Line: i + 1}})
		}
		src.Files = append(src.Files, sf)
	}
	return &archdoc.FactSet{
		Name: "x",
		Apps: []archdoc.App{{Name: dir, Dir: dir, Manifest: dir + "/package.json", Role: archdoc.RoleService,
			Language: "TypeScript", Prov: archdoc.Provenance{File: dir + "/package.json", Line: 1}}},
		Sources: []archdoc.Source{src},
	}
}

func byID(m archdoc.Model) map[string]archdoc.Node {
	out := map[string]archdoc.Node{}
	for _, n := range m.Nodes {
		out[n.ID] = n
	}
	return out
}

// Immich's server, in miniature: a component per directory, files loose at the root as one, and a
// "uses" edge per pair that cites one import per importing file and counts them all.
func TestComponentsFromDirectories(t *testing.T) {
	var files []string
	imports := map[string][]string{}
	for i := 0; i < 12; i++ {
		c := fmt.Sprintf("server/src/controllers/c%02d.controller.ts", i)
		s := fmt.Sprintf("server/src/services/s%02d.service.ts", i)
		files = append(files, c, s)
		imports[c] = []string{s, "server/src/enum.ts"}
	}
	files = append(files, "server/src/enum.ts", "server/src/main.ts")
	imports["server/src/main.ts"] = []string{"server/src/controllers/c00.controller.ts", "server/src/controllers/c01.controller.ts"}

	m := Derive(codeFacts("server", "server/src", files, imports))
	nodes := byID(m)
	ctl, ok := nodes["cmp:server/controllers"]
	if !ok || ctl.Kind != archdoc.Component || ctl.Parent != "app:server" || len(ctl.Files) != 12 || ctl.Lines != 120 {
		t.Fatalf("controllers: %+v", ctl)
	}
	if top := nodes["cmp:server/."]; top.Name != "src (top level)" || len(top.Files) != 2 {
		t.Errorf("loose files: %+v", top)
	}
	var uses *archdoc.Edge
	for i, e := range m.Edges {
		if e.From == "cmp:server/controllers" && e.To == "cmp:server/services" {
			uses = &m.Edges[i]
		}
	}
	if uses == nil || uses.Label != "uses" || uses.Weight != 12 || len(uses.Prov) != 10 {
		t.Fatalf("controllers → services: %+v", uses)
	}
	if p := uses.Prov[0]; p.File != "server/src/controllers/c00.controller.ts" || p.Line != 1 {
		t.Errorf("first citation %s, want the first importing file at its line", p)
	}
	view := m.Component("app:server")
	if len(view.Nodes) != 3 || view.Name != "server" {
		t.Errorf("component view: %d nodes, named %q", len(view.Nodes), view.Name)
	}
	for _, n := range m.Container().Nodes {
		if n.Kind == archdoc.Component {
			t.Errorf("a component in the container view: %s", n.ID)
		}
	}
	for _, n := range m.Context().Nodes {
		if n.Kind == archdoc.Component {
			t.Errorf("a component in the context view: %s", n.ID)
		}
	}
}

// SvelteKit keeps nearly everything in src/lib: a directory holding more than half of a large
// application is split into its subdirectories, and the files loose in it stay together.
func TestADominantDirectoryIsSplit(t *testing.T) {
	var files []string
	for i := 0; i < 40; i++ {
		files = append(files, fmt.Sprintf("web/src/lib/components/c%02d.svelte", i))
	}
	for i := 0; i < 10; i++ {
		files = append(files, fmt.Sprintf("web/src/lib/stores/s%02d.ts", i), fmt.Sprintf("web/src/routes/r%02d/+page.svelte", i))
	}
	files = append(files, "web/src/lib/index.ts")
	nodes := byID(Derive(codeFacts("web", "web/src", files, nil)))
	for _, id := range []string{"cmp:web/lib/components", "cmp:web/lib/stores", "cmp:web/lib", "cmp:web/routes"} {
		if _, ok := nodes[id]; !ok {
			t.Errorf("no %s", id)
		}
	}
	if n := nodes["cmp:web/lib"]; len(n.Files) != 1 {
		t.Errorf("lib holds %v, want only its loose file", n.Files)
	}
}

// An application with no directories is its files.
func TestAFlatApplicationIsItsFiles(t *testing.T) {
	files := []string{"streaming/index.js", "streaming/redis.js", "streaming/utils.js"}
	m := Derive(codeFacts("streaming", "streaming", files, map[string][]string{"streaming/index.js": {"streaming/redis.js"}}))
	nodes := byID(m)
	if n, ok := nodes["cmp:streaming/redis"]; !ok || n.Dir != "streaming/redis.js" {
		t.Errorf("redis: %+v", n)
	}
	if len(m.Component("app:streaming").Edges) != 1 {
		t.Errorf("edges %+v", m.Component("app:streaming").Edges)
	}
}

func TestComponentsAreDeterministic(t *testing.T) {
	var files []string
	imports := map[string][]string{}
	for i := 0; i < 30; i++ {
		f := fmt.Sprintf("api/src/d%d/f%d.ts", i%7, i)
		files = append(files, f)
		imports[f] = []string{fmt.Sprintf("api/src/d%d/f%d.ts", (i+3)%7, (i+3)%30)}
	}
	first, _ := json.Marshal(Derive(codeFacts("api", "api/src", files, imports)))
	for i := 0; i < 5; i++ {
		again, _ := json.Marshal(Derive(codeFacts("api", "api/src", files, imports)))
		if string(again) != string(first) {
			t.Fatal("two derivations of the same code differ")
		}
	}
}
