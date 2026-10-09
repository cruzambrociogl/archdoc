package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// The data-flow view (F-16): the system's containers read left to right, from the people who put
// data in to where it is kept and where it leaves — Archify's data-flow diagram, built from facts.
// Archify's agent decides which column a box sits in; here a rule does, from what the box is and
// what reaches it, so the same model always draws the same picture:
//
//   - People: the actors.
//   - Clients: applications no other application calls, that call one — a web app, a phone app, a CLI.
//   - Services: every other application.
//   - Stores: data stores and queues.
//   - Outside: the external systems the code calls.
//
// The arrows are the container view's. An arrow into the store a container's tables live in says
// which tables its code writes and reads, counted over every query (Model.Access) and citing them.

// DataFlow names the system's data-flow view.
const DataFlow = "dataflow"

const (
	stagePeople   = "People"
	stageClients  = "Clients"
	stageServices = "Services"
	stageStores   = "Stores"
	stageOutside  = "Outside"
)

var stageOrder = []string{stagePeople, stageClients, stageServices, stageStores, stageOutside}

// dataNames is how many tables an arrow names before it counts the rest.
const dataNames = 3

// HasDataFlow reports whether the model has a data-flow view: a project with more than one
// application's worth to say — something stored or reached outside, and something that does it.
func HasDataFlow(m archdoc.Model) bool {
	if Tiny(m) {
		return false
	}
	return len(dataFlowView(m).Model.Stages) >= 3
}

func dataFlowView(m archdoc.Model) View {
	view := m.Container()
	view.Networks = nil
	kind := map[string]archdoc.Kind{}
	for _, n := range view.Nodes {
		kind[n.ID] = n.Kind
	}
	calledByApp, callsApp := map[string]bool{}, map[string]bool{}
	for _, e := range view.Edges {
		if kind[e.From] == archdoc.Application && kind[e.To] == archdoc.Application {
			calledByApp[e.To], callsApp[e.From] = true, true
		}
	}
	used := map[string]bool{}
	for i := range view.Nodes {
		n := &view.Nodes[i]
		switch {
		case n.Kind == archdoc.Actor:
			n.Stage = stagePeople
		case n.Kind == archdoc.Application && callsApp[n.ID] && !calledByApp[n.ID]:
			n.Stage = stageClients
		case n.Kind == archdoc.Application:
			n.Stage = stageServices
		case n.Kind == archdoc.Datastore || n.Kind == archdoc.Queue:
			n.Stage = stageStores
		default:
			n.Stage = stageOutside
		}
		used[n.Stage] = true
	}
	for _, s := range stageOrder {
		if used[s] {
			view.Stages = append(view.Stages, s)
		}
	}
	edges := make([]archdoc.Edge, len(view.Edges))
	copy(edges, view.Edges)
	for i, e := range edges {
		if store, ok := storeOf(m, e.From); ok && store.ID == e.To {
			if label, cites := carried(m, e.From); label != "" {
				edges[i].Label, edges[i].LabelProv = label, cites[0]
				edges[i].Prov = append(append([]archdoc.Provenance(nil), e.Prov...), cites...)
			}
		}
	}
	view.Edges = edges
	return View{Name: DataFlow, File: "dataflow", Title: "Data flow", Model: view}
}

// carried is what a container's code puts into and takes out of its store, in words — "writes
// asset, album, user and 38 more tables · reads 11 more" — and the first query of each table it
// names, as the citation. Only tables the code declares count: a query naming anything else is
// not known to be in that store.
func carried(m archdoc.Model, container string) (string, []archdoc.Provenance) {
	type use struct {
		table         string
		writes, reads int
		first         archdoc.Provenance
	}
	byTable := map[string]*use{}
	for _, a := range m.Access {
		if a.Container != container || a.Element == "" || len(a.Prov) == 0 {
			continue
		}
		u, ok := byTable[a.Table]
		if !ok {
			u = &use{table: a.Table, first: a.Prov[0]}
			byTable[a.Table] = u
		}
		if a.Writes() {
			u.writes += a.Count
		} else {
			u.reads += a.Count
		}
		if less(a.Prov[0], u.first) {
			u.first = a.Prov[0]
		}
	}
	var written, readOnly []*use
	for _, u := range byTable {
		if u.writes > 0 {
			written = append(written, u)
		} else {
			readOnly = append(readOnly, u)
		}
	}
	busiest := func(us []*use) {
		sort.Slice(us, func(i, j int) bool {
			a, b := us[i].writes+us[i].reads, us[j].writes+us[j].reads
			if a != b {
				return a > b
			}
			return us[i].table < us[j].table
		})
	}
	busiest(written)
	busiest(readOnly)
	named := func(verb string, us []*use) string {
		var names []string
		for i := 0; i < len(us) && i < dataNames; i++ {
			names = append(names, us[i].table)
		}
		s := verb + " " + strings.Join(names, ", ")
		switch more := len(us) - len(names); {
		case more > 0:
			s += fmt.Sprintf(" and %d more tables", more)
		case len(us) == 1:
			s += " table"
		default:
			s += " tables"
		}
		return s
	}
	var parts []string
	var cites []archdoc.Provenance
	switch {
	case len(written) > 0:
		parts = append(parts, named("writes", written))
		if len(readOnly) > 0 {
			parts = append(parts, fmt.Sprintf("reads %d more", len(readOnly)))
		}
	case len(readOnly) > 0:
		parts = append(parts, named("reads", readOnly))
	}
	for _, us := range [][]*use{written, readOnly} {
		for _, u := range us {
			if len(cites) < accessCites {
				cites = append(cites, u.first)
			}
		}
	}
	return strings.Join(parts, " · "), cites
}

// accessCites is how many queries an arrow into a store cites: the first of each table it names,
// busiest first.
const accessCites = 10

func less(a, b archdoc.Provenance) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	return a.Line < b.Line
}
