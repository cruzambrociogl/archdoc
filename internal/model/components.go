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
// A component is a directory under the source root — server/src/services, web/src/routes — because
// that is how most code is already organised, and the framework conventions that matter here put
// one kind of thing per directory (D-2: conventions first, then directories). Two refinements, both
// measured on real code:
//
//   - A directory holding more than half of the application is split into its subdirectories, once
//     or twice. SvelteKit keeps nearly everything in src/lib; one box called "lib" says nothing.
//   - In Python each top-level module of the package is its own component — the module is Python's
//     unit of code, and the FastAPI template keeps its models and its data access in two of them.
//     So is each file of an application with no directories at all. Elsewhere, files loose at the
//     root form one component.
//
// A component uses another when one of its files imports one of the other's. The edge cites the
// first such import in each importing file, at most ten, and counts every one.

const (
	maxCited  = 10
	maxSplits = 2
	minSplit  = 50 // files: below this an application is small enough to read as it is laid out
)

// components adds each source's components and their uses, and returns the component every
// file belongs to.
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
		group := grouping(src, python)

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

		// Identity is what the component is — a part of this container, by its name under the
		// source root — not where the root happens to be (F-31): moving src/ is not a new system.
		_, local, _ := strings.Cut(parent.ID, ":")
		prefix := "cmp:" + local + "/"
		for _, key := range sortedKeys(parts) {
			p := parts[key]
			if p.lines == 0 {
				delete(parts, key) // an empty __init__.py marks a package; it is not a part of it
				continue
			}
			name, dir := key, path.Join(src.Root, key)
			switch {
			case key == ".":
				name, dir = path.Base(src.Root)+" (top level)", src.Root
			case len(p.files) == 1 && path.Dir(p.files[0]) == src.Root:
				dir = p.files[0]
			}
			m.Nodes = append(m.Nodes, archdoc.Node{
				ID:         prefix + key,
				Name:       name,
				Kind:       archdoc.Component,
				Technology: strings.Join(sortedKeys(p.languages), " · "),
				Evidence:   archdoc.Declared,
				Parent:     parent.ID,
				Dir:        dir,
				Files:      p.files,
				Lines:      p.lines,
				TechProv:   archdoc.Provenance{File: p.files[0], Line: 1, Note: "the language of its files"},
				Prov: archdoc.Provenance{File: p.files[0], Line: 1,
					Note: fmt.Sprintf("%d %s in %s", len(p.files), plural(len(p.files), "file", "files"), dir)},
			})
		}

		for file, key := range group {
			if parts[key] != nil {
				componentOf[file] = prefix + key
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
	}
	return componentOf
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
