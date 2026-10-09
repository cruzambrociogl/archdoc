package model

import (
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Access (F-16): which tables each component's code queries, and how, read from every method and
// function rather than from the ones a flow reaches. A flow starts at a route and stops at a depth;
// a table only a scheduled job or a migration script writes would never show. This is the data
// view's other half: the tables say what is stored, this says who stores it.

// accessCites is how many queries an access cites; Count says how many there are.
const accessCites = 10

func access(m *archdoc.Model, sources []archdoc.Source, componentOf map[string]string) {
	tableOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Table {
			tableOf[n.Parent+"\x00"+n.Name] = n.ID
		}
	}
	containerOf := map[string]string{}
	for _, n := range m.Nodes {
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	index := map[string]int{}
	for _, src := range sources {
		container, ok := containerOf[src.App]
		if !ok {
			continue
		}
		for _, f := range src.Files {
			for _, c := range f.Classes {
				for _, meth := range c.Methods {
					for _, q := range meth.Queries {
						name := q.Table
						if strings.HasSuffix(f.Path, ".py") {
							name = strings.ToLower(name) // SQLModel names a table after its class
						}
						a := archdoc.Access{Container: container, Component: componentOf[f.Path], Table: name, Op: q.Op}
						key := a.Container + "\x00" + a.Component + "\x00" + a.Table + "\x00" + a.Op
						i, seen := index[key]
						if !seen {
							a.Element = tableOf[container+"\x00"+name]
							i = len(m.Access)
							index[key] = i
							m.Access = append(m.Access, a)
						}
						m.Access[i].Count++
						m.Access[i].Prov = append(m.Access[i].Prov, q.Prov)
					}
				}
			}
		}
	}
	for i := range m.Access {
		ps := m.Access[i].Prov
		sort.SliceStable(ps, func(a, b int) bool {
			if ps[a].File != ps[b].File {
				return ps[a].File < ps[b].File
			}
			return ps[a].Line < ps[b].Line
		})
		if len(ps) > accessCites {
			m.Access[i].Prov = ps[:accessCites]
		}
	}
	sort.SliceStable(m.Access, func(i, j int) bool {
		a, b := m.Access[i], m.Access[j]
		switch {
		case a.Container != b.Container:
			return a.Container < b.Container
		case a.Component != b.Component:
			return a.Component < b.Component
		case a.Table != b.Table:
			return a.Table < b.Table
		}
		return a.Op < b.Op
	})
}
