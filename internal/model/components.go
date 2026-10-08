package model

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Components: the parts of an application's code, and which uses which (F-03, F-10).
//
// A C4 component is a responsibility behind an interface — "albums", "authentication" — not a
// folder. Where the code names its files by what they are and what they do —
// album.controller.ts, album.service.ts, album.repository.ts, album.table.ts — that responsibility
// is written in the file names, and a component is a name that spans those roles (D-2: the
// code's conventions first). The rule, measured on Immich's server:
//
//   - A role is a suffix many files carry: controller, service, repository, dto, table. It is found
//     by counting, not from a list.
//   - A feature is a name with at least two roles. A longer name joins the feature it extends —
//     album-user and album-asset-audit are part of album — and so does a lone file of that name.
//   - A lone file that does something, in a role the features themselves have — the
//     machine-learning repository — is a component of its own. A lone file that only shapes data,
//     or whose role is a rare one, and every file with no role, stays with its folder.
//
// The convention has to carry the application for this to apply: several features, covering a
// good part of the files. Otherwise — and always as a second view, "by folder" — a part is a
// directory under the source root, with two refinements:
//
//   - A directory holding more than half of the application is split into its subdirectories, once
//     or twice. SvelteKit keeps nearly everything in src/lib; one box called "lib" says nothing.
//   - In Python each top-level module of the package is its own part — the module is Python's
//     unit of code. So is each file of an application with no directories at all. Elsewhere, files
//     loose at the root form one part.
//
// A part uses another when one of its files imports one of the other's. The edge cites the first
// such import in each importing file, at most ten, and counts every one.

const (
	maxCited  = 10
	maxSplits = 2
	minSplit  = 50 // files: below this an application is small enough to read as it is laid out

	minRole     = 3 // files carrying a suffix before it counts as a role
	minFeatures = 3 // features before the naming convention is taken to organise the application
)

// Suffixes that are not roles: a language's own, a test's, a tool's.
var notRoles = map[string]bool{"svelte": true, "d": true, "min": true, "test": true, "spec": true, "config": true,
	"stories": true, "module": true, "generated": true, "g": true}

// Roles that shape data rather than do something. A lone file of one is not a component of its own.
var dataRoles = map[string]bool{"table": true, "dto": true, "entity": true, "model": true, "schema": true,
	"type": true, "types": true, "interface": true, "enum": true, "constant": true, "constants": true}

// components adds each source's components and their uses — and, where components are features, the
// folders as a second view — and returns the component every file belongs to.
func components(m *archdoc.Model, sources []archdoc.Source) map[string]string {
	componentOf := map[string]string{}
	containerOf := map[string]archdoc.Node{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n
		}
	}
	for _, src := range sources {
		parent, ok := containerOf[src.App]
		if !ok || len(src.Files) == 0 {
			continue
		}
		python := src.Files[0].Language == "Python"
		folders := grouping(src, python)
		// Identity is what the component is — a part of this container, by its name — not where
		// the source root happens to be (F-31): moving src/ is not a new system.
		_, local, _ := strings.Cut(parent.ID, ":")

		group, roles := folders, map[string][]string(nil)
		if features, r, ok := slices(src, folders); ok {
			group, roles = features, r
			emit(m, src, parent.ID, archdoc.Module, "dir:"+local+"/", folders, nil)
		}
		for file, id := range emit(m, src, parent.ID, archdoc.Component, "cmp:"+local+"/", group, roles) {
			componentOf[file] = id
		}
	}
	return componentOf
}

// slices groups a source's files into features by the roles their names carry. roles gives, for
// each feature, the roles it spans. ok is false when the convention does not carry the application.
func slices(src archdoc.Source, folders map[string]string) (group map[string]string, roles map[string][]string, ok bool) {
	type named struct{ stem, role string }
	names := map[string]named{}
	count := map[string]int{}
	for _, f := range src.Files {
		base := path.Base(f.Path)
		base = strings.TrimSuffix(base, path.Ext(base))
		if i := strings.LastIndex(base, "."); i > 0 && !notRoles[base[i+1:]] {
			names[f.Path] = named{base[:i], base[i+1:]}
			count[base[i+1:]]++
		}
	}
	spans := map[string]map[string]bool{}
	for _, n := range names {
		if count[n.role] >= minRole {
			if spans[n.stem] == nil {
				spans[n.stem] = map[string]bool{}
			}
			spans[n.stem][n.role] = true
		}
	}
	var features []string
	for stem, r := range spans {
		if len(r) >= 2 {
			features = append(features, stem)
		}
	}
	sort.Slice(features, func(i, j int) bool {
		if len(features[i]) != len(features[j]) {
			return len(features[i]) < len(features[j])
		}
		return features[i] < features[j]
	})
	// root is the shortest feature a name is or extends: asset-media is part of asset.
	root := func(stem string) string {
		for _, f := range features {
			if stem == f || strings.HasPrefix(stem, f+"-") || strings.HasPrefix(stem, f+".") {
				return f
			}
		}
		return ""
	}

	// A role is a main one when several features have it: controller, service, repository — not
	// the email template two features happen to have.
	main := map[string]int{}
	for _, f := range features {
		for r := range spans[f] {
			main[r]++
		}
	}

	group = map[string]string{}
	covered := 0
	kept := map[string]bool{}
	for _, f := range src.Files {
		n, conventional := names[f.Path]
		conventional = conventional && count[n.role] >= minRole
		switch r := root(n.stem); {
		case conventional && r != "":
			group[f.Path] = r
			kept[r] = true
			covered++
		case conventional && !dataRoles[n.role] && main[n.role] >= minFeatures:
			group[f.Path] = n.stem
		default:
			group[f.Path] = folders[f.Path]
		}
	}
	if len(kept) < minFeatures || 10*covered < 3*len(src.Files) {
		return nil, nil, false
	}
	roles = map[string][]string{}
	for file, key := range group {
		if n, ok := names[file]; ok && kept[key] && count[n.role] >= minRole {
			roles[key] = append(roles[key], n.role)
		}
	}
	for key, list := range roles {
		sort.Strings(list)
		uniq := list[:0]
		for i, r := range list {
			if i == 0 || r != list[i-1] {
				uniq = append(uniq, r)
			}
		}
		roles[key] = uniq
	}
	return group, roles, true
}

// emit adds one grouping of a source's files as nodes of a kind, with the uses between them, and
// returns the node every file went to. roles, when given, names the roles a feature spans.
func emit(m *archdoc.Model, src archdoc.Source, parent string, kind archdoc.Kind, prefix string, group map[string]string, roles map[string][]string) map[string]string {
	type part struct {
		files     []string
		lines     int
		languages map[string]bool
	}
	parts := map[string]*part{}
	for _, f := range src.Files {
		key := group[f.Path]
		p := parts[key]
		if p == nil {
			p = &part{languages: map[string]bool{}}
			parts[key] = p
		}
		p.files = append(p.files, f.Path)
		p.lines += f.Lines
		p.languages[f.Language] = true
	}

	for _, key := range sortedKeys(parts) {
		p := parts[key]
		if p.lines == 0 {
			delete(parts, key) // an empty __init__.py marks a package; it is not a part of it
			continue
		}
		// Where it is: the one directory its files share, or — a feature across layers — none.
		dir := path.Dir(p.files[0])
		for _, f := range p.files {
			for dir != "." && !strings.HasPrefix(f, dir+"/") {
				dir = path.Dir(dir)
			}
		}
		name := key
		note := fmt.Sprintf("%d %s in %s", len(p.files), plural(len(p.files), "file", "files"), dir)
		switch {
		case key == ".":
			name = path.Base(src.Root) + " (top level)"
		case len(p.files) == 1:
			dir = p.files[0]
			note = "the file " + dir
		}
		if r := roles[key]; len(r) > 0 {
			note = fmt.Sprintf("%d %s named %s.* — %s", len(p.files), plural(len(p.files), "file", "files"), key, strings.Join(r, ", "))
		}
		m.Nodes = append(m.Nodes, archdoc.Node{
			ID:         prefix + key,
			Name:       name,
			Kind:       kind,
			Technology: strings.Join(sortedKeys(p.languages), " · "),
			Evidence:   archdoc.Declared,
			Parent:     parent,
			Dir:        dir,
			Files:      p.files,
			Lines:      p.lines,
			TechProv:   archdoc.Provenance{File: p.files[0], Line: 1, Note: "the language of its files"},
			Prov:       archdoc.Provenance{File: p.files[0], Line: 1, Note: note},
		})
	}

	of := map[string]string{}
	for file, key := range group {
		if parts[key] != nil {
			of[file] = prefix + key
		}
	}

	type link struct {
		cited  map[string]archdoc.Provenance // importing file → its first import of the other
		weight int
	}
	links := map[[2]string]*link{}
	for _, f := range src.Files {
		from := group[f.Path]
		for _, imp := range f.Imports {
			to, ok := group[imp.Target]
			if imp.Target == "" || !ok || to == from || parts[to] == nil || parts[from] == nil {
				continue
			}
			k := [2]string{from, to}
			l := links[k]
			if l == nil {
				l = &link{cited: map[string]archdoc.Provenance{}}
				links[k] = l
			}
			l.weight++
			if _, seen := l.cited[f.Path]; !seen {
				l.cited[f.Path] = imp.Prov
			}
		}
	}
	keys := make([][2]string, 0, len(links))
	for k := range links {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	for _, k := range keys {
		l := links[k]
		var prov []archdoc.Provenance
		for _, file := range sortedKeys(l.cited) {
			if len(prov) == maxCited {
				break
			}
			prov = append(prov, l.cited[file])
		}
		m.Edges = append(m.Edges, archdoc.Edge{
			From: prefix + k[0], To: prefix + k[1], Label: "uses", Prov: prov, Weight: l.weight,
		})
	}
	return of
}

// grouping assigns each file of a source to its component, keyed by the component's path under
// the source root: "services", "lib/components", "main" for a Python module, "." for loose files.
func grouping(src archdoc.Source, python bool) map[string]string {
	rel := map[string]string{}
	for _, f := range src.Files {
		rel[f.Path] = strings.TrimPrefix(f.Path, src.Root+"/")
		if src.Root == "." {
			rel[f.Path] = f.Path
		}
	}
	// An application with no directories at all is its files: Mastodon's streaming server is nine
	// modules side by side, and one box for all of them would be a component view of one.
	flat := true
	for _, r := range rel {
		if strings.Contains(r, "/") {
			flat = false
			break
		}
	}
	keyAt := func(r string, depth int) string {
		segs := strings.Split(r, "/")
		if len(segs) == 1 {
			if (python || flat) && r != "__init__.py" {
				return stem(r)
			}
			return "."
		}
		if depth >= len(segs) {
			depth = len(segs) - 1
		}
		return strings.Join(segs[:depth], "/")
	}

	depth := map[string]int{} // file → how deep its key goes
	for f := range rel {
		depth[f] = 1
	}
	for split := 0; split < maxSplits && len(rel) >= minSplit; split++ {
		count := map[string]int{}
		deeper := map[string]bool{} // a key with files below its own level
		for f, r := range rel {
			k := keyAt(r, depth[f])
			count[k]++
			if strings.Count(r, "/") > depth[f] {
				deeper[k] = true
			}
		}
		changed := false
		for f, r := range rel {
			k := keyAt(r, depth[f])
			if k != "." && 2*count[k] > len(rel) && deeper[k] {
				depth[f]++
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	out := make(map[string]string, len(rel))
	for f, r := range rel {
		out[f] = keyAt(r, depth[f])
	}
	return out
}

// stem is a file's name without its extension: "main.py" is the module main.
func stem(name string) string {
	if i := strings.Index(name, "."); i > 0 {
		return name[:i]
	}
	return name
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
