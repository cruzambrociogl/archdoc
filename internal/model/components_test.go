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

// Where files are named by what they are for and what they do, a component is a name that spans
// roles — album's controller, service, repository and tables — and the folders are a second view.
func TestComponentsAreFeaturesWhereTheCodeNamesThem(t *testing.T) {
	var files []string
	for _, f := range []string{"album", "asset", "user"} {
		files = append(files, "server/src/controllers/"+f+".controller.ts", "server/src/services/"+f+".service.ts",
			"server/src/repositories/"+f+".repository.ts", "server/src/schema/tables/"+f+".table.ts")
	}
	files = append(files,
		"server/src/repositories/album-user.repository.ts",       // extends a feature
		"server/src/schema/tables/album-asset-audit.table.ts",    // a lone table of a feature's name
		"server/src/repositories/machine-learning.repository.ts", // does something, alone
		"server/src/schema/tables/geodata.table.ts",              // shapes data, alone
		"server/src/utils/misc.ts", "server/src/main.ts")
	imports := map[string][]string{
		"server/src/controllers/album.controller.ts": {"server/src/services/album.service.ts", "server/src/services/asset.service.ts"},
		"server/src/services/album.service.ts":       {"server/src/repositories/album.repository.ts", "server/src/utils/misc.ts"},
	}
	m := Derive(codeFacts("server", "server/src", files, imports))
	nodes := byID(m)
	album, ok := nodes["cmp:server/album"]
	if !ok || album.Kind != archdoc.Component || len(album.Files) != 6 {
		t.Fatalf("album: %+v", album)
	}
	if album.Prov.Note != "6 files named album.* — controller, repository, service, table" {
		t.Errorf("album's note: %q", album.Prov.Note)
	}
	if n, ok := nodes["cmp:server/machine-learning"]; !ok || len(n.Files) != 1 {
		t.Errorf("a lone repository is a component of its own: %+v", n)
	}
	if _, ok := nodes["cmp:server/geodata"]; ok {
		t.Error("a lone table became a component")
	}
	if n := nodes["cmp:server/schema"]; len(n.Files) != 1 {
		t.Errorf("the lone table stays with its folder: %+v", n.Files)
	}
	// One import inside album, one to asset, one to utils: two uses leave it.
	var uses []string
	for _, e := range m.Component("app:server").Edges {
		if e.From == "cmp:server/album" {
			uses = append(uses, e.To)
		}
	}
	if len(uses) != 2 || uses[0] != "cmp:server/asset" || uses[1] != "cmp:server/utils" {
		t.Errorf("album uses %v", uses)
	}
	// The folders are still there, as the other view of the same code.
	if folder, ok := nodes["dir:server/controllers"]; !ok || folder.Kind != archdoc.Module || len(folder.Files) != 3 {
		t.Errorf("controllers folder: %+v", folder)
	}
	if got := len(m.Structure("app:server").Nodes); got != 6 {
		t.Errorf("%d folders, want controllers, services, repositories, schema, utils and the top level", got)
	}
}

// A front end laid out by layer: the feature's name runs across the layers — a service, a modal, a
// utility, a directory of components, a route — and that is the component. A name only one layer
// carries, a layer's own word, and a feature of two files stay with their folders. A router's
// grouping and parameter segments are not names.
func TestFeaturesAcrossLayers(t *testing.T) {
	files := []string{
		"web/src/lib/services/album.service.ts", "web/src/lib/services/shared-link.service.ts", "web/src/lib/services/asset.service.ts",
		"web/src/lib/modals/AlbumEditModal.svelte", "web/src/lib/modals/AlbumPickerModal.svelte", "web/src/lib/modals/SharedLinkCreateModal.svelte",
		"web/src/lib/modals/AssetTagModal.svelte", "web/src/lib/modals/HelpModal.svelte",
		"web/src/lib/utils/album-utils.ts", "web/src/lib/utils/asset-utils.ts", "web/src/lib/utils/date-time.ts",
		"web/src/lib/components/album-page/AlbumViewer.svelte", "web/src/lib/components/album-page/AlbumCover.svelte",
		"web/src/lib/components/shared-components/Button.svelte", "web/src/lib/components/shared-components/Icon.svelte",
		"web/src/lib/components/timeline/Timeline.svelte", "web/src/lib/components/timeline/Month.svelte",
		"web/src/lib/managers/timeline-manager.svelte.ts",
		"web/src/routes/(user)/albums/+page.svelte", "web/src/routes/(user)/albums/[albumId]/+page.svelte",
		"web/src/routes/(user)/shared-links/+page.svelte", "web/src/routes/+layout.svelte",
		"web/src/lib/stores/map.store.ts", "web/src/lib/modals/MapModal.svelte",
	}
	// An application large enough that src/lib is read as its layers, as a real one is.
	for i := 0; i < 30; i++ {
		files = append(files, fmt.Sprintf("web/src/lib/elements/Element%02d.svelte", i))
	}
	m := Derive(codeFacts("web", "web/src", files, nil))
	in := map[string]string{}
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Component {
			for _, f := range n.Files {
				in[f] = n.Name
			}
		}
	}
	for file, want := range map[string]string{
		"web/src/lib/services/album.service.ts":                "album",
		"web/src/lib/modals/AlbumEditModal.svelte":             "album",
		"web/src/lib/utils/album-utils.ts":                     "album",
		"web/src/lib/components/album-page/AlbumCover.svelte":  "album",
		"web/src/routes/(user)/albums/[albumId]/+page.svelte":  "album",
		"web/src/lib/modals/SharedLinkCreateModal.svelte":      "shared-link",
		"web/src/routes/(user)/shared-links/+page.svelte":      "shared-link",
		"web/src/lib/modals/AssetTagModal.svelte":              "asset",
		"web/src/lib/managers/timeline-manager.svelte.ts":      "timeline",
		"web/src/lib/components/timeline/Month.svelte":         "timeline",
		"web/src/lib/modals/HelpModal.svelte":                  "lib/modals",
		"web/src/lib/utils/date-time.ts":                       "lib/utils",
		"web/src/lib/components/shared-components/Icon.svelte": "lib/components",
		"web/src/lib/modals/MapModal.svelte":                   "lib/modals", // map: two files, too small
		"web/src/routes/+layout.svelte":                        "routes",
	} {
		if in[file] != want {
			t.Errorf("%s is in %q, want %q", file, in[file], want)
		}
	}
	modules := 0
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Module {
			modules++
		}
	}
	if modules == 0 {
		t.Error("the folders are not kept as a second view")
	}
}
