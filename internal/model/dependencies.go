package model

import (
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// dependencies lists what each container's manifest depends on, with where its code imports it
// (F-14): the manifest line declares, the imports show use. A Python distribution is matched to
// what is imported by its name with dashes as underscores — python-dotenv is not found as dotenv,
// and then simply shows no imports rather than a guess.
func dependencies(m *archdoc.Model, f *archdoc.FactSet, componentOf map[string]string) {
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	sources := map[string]archdoc.Source{}
	for _, s := range f.Sources {
		sources[s.App] = s
	}
	for _, a := range f.Apps {
		container, ok := containerOf[a.Dir]
		if !ok || len(a.Requires) == 0 {
			continue
		}
		imports := map[string]int{}
		users := map[string]map[string]bool{}
		for _, file := range sources[a.Dir].Files {
			for _, imp := range file.Imports {
				if imp.How != archdoc.ByPackage {
					continue
				}
				key := normal(imp.Package)
				imports[key]++
				if c := componentOf[file.Path]; c != "" {
					if users[key] == nil {
						users[key] = map[string]bool{}
					}
					users[key][c] = true
				}
			}
		}
		for _, r := range a.Requires {
			key := normal(r.Name)
			if r.Dev && imports[key] == 0 {
				continue // a tool of the build, never imported: not part of what runs
			}
			m.Dependencies = append(m.Dependencies, archdoc.Package{Container: container, Name: r.Name, Version: r.Version,
				Dev: r.Dev, Imports: imports[key], Components: sortedKeys(users[key]), Prov: r.Prov})
		}
	}
	sort.SliceStable(m.Dependencies, func(i, j int) bool {
		a, b := m.Dependencies[i], m.Dependencies[j]
		if a.Container != b.Container {
			return a.Container < b.Container
		}
		return a.Name < b.Name
	})
}

func normal(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "-", "_")
}
