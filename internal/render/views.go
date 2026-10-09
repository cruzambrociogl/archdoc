package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// View is one diagram archdoc draws. Its Name keys its stored layout, its arrangement and its
// address in the app; File is the stem of its committed .svg and .mmd.
type View struct {
	Name  string // "context", "container", "component:svc:immich-server"
	File  string // "context", "container", "component-immich-server"
	Title string
	Model archdoc.Model
	// Group draws a boundary around what is inside: the system in a container view, the container
	// in a component view.
	Group bool
}

// ComponentPrefix starts the name of every component view, DataPrefix every data view; the
// container's ID follows.
const (
	ComponentPrefix = "component:"
	DataPrefix      = "data:"
	// StructurePrefix starts the name of a container's by-folder view, where its components are
	// features and the folders are the other way to read the same code.
	StructurePrefix = "structure:"
	// MainPrefix starts the name of a container's main-components view: the few that do the most,
	// for a container with more components than a diagram can show.
	MainPrefix = "main:"
)

// MainParts is how many components a diagram holds and can still be read; a container with more
// has a main view of this many. C4's own advice is to split a component diagram well before it
// has as many boxes as Immich's server has: 75.
const MainParts = 16

// mainOver is how many components a container must have before it gets a main view. Above
// MainParts by half again: a view that leaves out two of eighteen hides more than it clears up —
// Immich's mobile app lost a three-line and a four-line component to it.
const mainOver = MainParts + MainParts/2

// Views lists every diagram of the model, in a fixed order: context, container, then one
// component view per container whose code was read, in model order.
func Views(m archdoc.Model) []View {
	out := []View{
		{Name: "context", File: "context", Title: "System context", Model: m.Context()},
		{Name: "container", File: "container", Title: "Containers", Model: m.Container(), Group: true},
	}
	for _, id := range m.Components() {
		out = append(out, componentView(m, id))
	}
	for _, id := range Mains(m) {
		out = append(out, mainView(m, id))
	}
	for _, id := range m.Structures() {
		out = append(out, structureView(m, id))
	}
	for _, id := range m.Datas() {
		out = append(out, dataView(m, id))
	}
	return out
}

// ViewOf returns the named view, if the model has it.
func ViewOf(m archdoc.Model, name string) (View, bool) {
	switch {
	case name == "context":
		return View{Name: name, File: name, Title: "System context", Model: m.Context()}, true
	case name == "container":
		return View{Name: name, File: name, Title: "Containers", Model: m.Container(), Group: true}, true
	case strings.HasPrefix(name, ComponentPrefix):
		id := strings.TrimPrefix(name, ComponentPrefix)
		for _, c := range m.Components() {
			if c == id {
				return componentView(m, id), true
			}
		}
	case strings.HasPrefix(name, MainPrefix):
		id := strings.TrimPrefix(name, MainPrefix)
		for _, c := range Mains(m) {
			if c == id {
				return mainView(m, id), true
			}
		}
	case strings.HasPrefix(name, StructurePrefix):
		id := strings.TrimPrefix(name, StructurePrefix)
		for _, c := range m.Structures() {
			if c == id {
				return structureView(m, id), true
			}
		}
	case strings.HasPrefix(name, DataPrefix):
		id := strings.TrimPrefix(name, DataPrefix)
		for _, c := range m.Datas() {
			if c == id {
				return dataView(m, id), true
			}
		}
	}
	return View{}, false
}

func componentView(m archdoc.Model, id string) View {
	// Every component edge says "uses"; drawn on each of a hundred arrows it says nothing, and
	// the weight — how many imports — is drawn as the arrow's thickness instead.
	view := unlabelled(m.Component(id))
	markShared(&view)
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  ComponentPrefix + id,
		File:  "component-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Components of " + view.Name,
		Model: view,
		Group: true,
	}
}

// Mains lists the containers with more components than one diagram can show, in model order.
func Mains(m archdoc.Model) []string {
	count := map[string]int{}
	for _, n := range m.Nodes {
		if n.Kind == archdoc.Component {
			count[n.Parent]++
		}
	}
	var out []string
	for _, id := range m.Components() {
		if count[id] > mainOver {
			out = append(out, id)
		}
	}
	return out
}

// mainView is a container's components that do the most: the MainParts that handle the most
// routes, pages, commands and jobs — what the container is for, by what the code declares — or,
// in a container that declares none, the largest. Only the uses among them are drawn. What each
// is used by is still counted over all the container's components, so a shared one says so.
func mainView(m archdoc.Model, id string) View {
	view := unlabelled(m.Component(id))
	markShared(&view)
	handles := map[string]int{}
	for _, e := range m.Entries {
		if e.Container == id && e.Component != "" {
			handles[e.Component]++
		}
	}
	ranked := append([]archdoc.Node(nil), view.Nodes...)
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if handles[a.ID] != handles[b.ID] {
			return handles[a.ID] > handles[b.ID]
		}
		if a.Lines != b.Lines {
			return a.Lines > b.Lines
		}
		return a.ID < b.ID
	})
	keep := map[string]bool{}
	for i := 0; i < len(ranked) && i < MainParts; i++ {
		keep[ranked[i].ID] = true
	}
	nodes := view.Nodes[:0:0]
	for _, n := range view.Nodes {
		if keep[n.ID] {
			nodes = append(nodes, n)
		}
	}
	edges := view.Edges[:0:0]
	for _, e := range view.Edges {
		if keep[e.From] && keep[e.To] {
			edges = append(edges, e)
		}
	}
	total := len(view.Nodes)
	view.Nodes, view.Edges = nodes, edges
	view.Boundary = fmt.Sprintf("%s — the %d components that handle the most, of %d", view.Name, len(nodes), total)
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  MainPrefix + id,
		File:  "main-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Main components of " + view.Name,
		Model: view,
		Group: true,
	}
}

// structureView is one container's code by folder.
func structureView(m archdoc.Model, id string) View {
	view := unlabelled(m.Structure(id))
	markShared(&view)
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  StructurePrefix + id,
		File:  "structure-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Folders of " + view.Name,
		Model: view,
		Group: true,
	}
}

// dataView is one container's tables, drawn inside a boundary that names where they are stored.
func dataView(m archdoc.Model, id string) View {
	view := unlabelled(m.Data(id)) // an arrow between tables is a foreign key; the box says which column
	view.Boundary = view.Name + " — tables its code declares"
	if store, ok := storeOf(m, id); ok {
		view.Boundary += ", stored in " + store.Name
		if store.Technology != "" {
			view.Boundary += " [" + store.Technology + "]"
		}
	}
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  DataPrefix + id,
		File:  "data-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Data of " + view.Name,
		Model: view,
		Group: true,
	}
}

// A part is shared when many of the others use it, and wiring when it uses many of the others: at
// least sharedMin, and at least three in five of the others in a small view, a quarter in a large
// one. Immich's server, by feature, is 75 components and 758 uses; eleven shared ones — its types,
// utils, auth, logging, config — and seven that wire everything together carry 649 of them.
const (
	sharedMin = 4
	largeView = 20 // parts, beyond which a quarter of the others is already "most things"
)

// markShared marks the parts most others use, and the ones that use most others. The layout then
// draws no arrow into the first or out of the second, and their boxes say so in words: a picture
// of who depends on the shared kernel is a picture of everything, and hides the rest.
func markShared(view *archdoc.Model) {
	others := len(view.Nodes) - 1
	need := (3*others + 4) / 5
	if others > largeView {
		need = (others + 3) / 4
	}
	if need < sharedMin {
		need = sharedMin
	}
	usedBy, uses := map[string]int{}, map[string]int{}
	for _, e := range view.Edges {
		usedBy[e.To]++
		uses[e.From]++
	}
	nodes := make([]archdoc.Node, len(view.Nodes))
	copy(nodes, view.Nodes)
	for i, n := range nodes {
		if u := usedBy[n.ID]; u >= need {
			nodes[i].UsedBy, nodes[i].Among = u, others
		}
		if u := uses[n.ID]; u >= need {
			nodes[i].UsesMany, nodes[i].Among = u, others
		}
	}
	view.Nodes = nodes
}

// unlabelled drops the label every edge of a view would repeat.
func unlabelled(view archdoc.Model) archdoc.Model {
	edges := make([]archdoc.Edge, len(view.Edges))
	for i, e := range view.Edges {
		e.Label, e.LabelProv = "", archdoc.Provenance{}
		edges[i] = e
	}
	view.Edges = edges
	return view
}

// storeOf is the relational data store a container connects to, when there is exactly one: where
// the tables its code declares live. Two, or none, and the boundary says nothing rather than guess.
func storeOf(m archdoc.Model, container string) (archdoc.Node, bool) {
	var found []archdoc.Node
	for _, e := range m.Edges {
		if e.From != container || !e.Traffic {
			continue
		}
		n, ok := m.Node(e.To)
		if ok && n.Kind == archdoc.Datastore && relational(n.Technology) {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		return archdoc.Node{}, false
	}
	return found[0], true
}

func relational(tech string) bool {
	t := strings.ToLower(tech)
	for _, s := range []string{"postgres", "mysql", "mariadb", "sql", "cockroach", "oracle"} {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}

// ComponentFile reports whether name is a file archdoc writes for a component or a data view.
func ComponentFile(name string) bool {
	return (strings.HasPrefix(name, "component-") || strings.HasPrefix(name, "data-") || strings.HasPrefix(name, "structure-")) &&
		(strings.HasSuffix(name, ".svg") || strings.HasSuffix(name, ".mmd"))
}
