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

// mainUses is how many of a component's uses of the others a main view draws: its strongest.
const mainUses = 2

// mainOver is how many components a container must have before it gets a main view. Above
// MainParts by half again: a view that leaves out two of eighteen hides more than it clears up —
// Immich's mobile app lost a three-line and a four-line component to it.
const mainOver = MainParts + MainParts/2

// MainExternals is how many external systems a context or container view draws when there are
// more than externalsOver: C4's overview holds a dozen or two elements, and Supabase's code names
// fifty systems outside it. Which ones is a count, not a judgement — the ones the most containers
// reach — and the rest stay in the element tables and one click away in the app.
const (
	MainExternals = 12
	externalsOver = MainExternals + MainExternals/2
)

// AllSuffix names the whole of a context or container view the overview cut: "container:all".
const AllSuffix = ":all"

// Views lists every diagram of the model, in a fixed order: context, container, then one
// component view per container whose code was read, in model order, and the data-flow view last.
func Views(m archdoc.Model) []View {
	out := []View{
		{Name: "context", File: "context", Title: "System context", Model: Overview(m, m.Context())},
		{Name: "container", File: "container", Title: "Containers", Model: Overview(m, m.Container()), Group: true},
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
	if HasDataFlow(m) {
		out = append(out, dataFlowView(m))
	}
	return out
}

// ViewOf returns the named view, if the model has it.
func ViewOf(m archdoc.Model, name string) (View, bool) {
	switch {
	case name == "context":
		return View{Name: name, File: name, Title: "System context", Model: Overview(m, m.Context())}, true
	case name == "container":
		return View{Name: name, File: name, Title: "Containers", Model: Overview(m, m.Container()), Group: true}, true
	case name == "context"+AllSuffix:
		return View{Name: name, File: "context-all", Title: "System context", Model: m.Context()}, true
	case name == "container"+AllSuffix:
		return View{Name: name, File: "container-all", Title: "Containers", Model: m.Container(), Group: true}, true
	case name == DataFlow:
		if HasDataFlow(m) {
			return dataFlowView(m), true
		}
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
	if len(view.Nodes) <= mainOver {
		describeParts(&view, m)
	}
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  ComponentPrefix + id,
		File:  "component-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Components of " + view.Name,
		Model: view,
		Group: true,
	}
}

// Overview is a context or container view of m with its external systems cut to the
// MainExternals that the most containers reach — then those seen used rather than only
// configured, then the most citations, then by ID — when it has more than externalsOver. What it
// leaves out is counted in Folded.
func Overview(m, view archdoc.Model) archdoc.Model {
	var externals []string
	for _, n := range view.Nodes {
		if n.Kind == archdoc.External {
			externals = append(externals, n.ID)
		}
	}
	if len(externals) <= externalsOver {
		return view
	}
	// Counted over the whole model, not the view: in the context view every container is one box.
	kind := make(map[string]archdoc.Kind, len(m.Nodes))
	for _, n := range m.Nodes {
		kind[n.ID] = n.Kind
	}
	reachedBy, cited, used := map[string]map[string]bool{}, map[string]int{}, map[string]bool{}
	for _, e := range m.Edges {
		if k, ok := kind[e.From]; !ok || k == archdoc.External {
			continue
		}
		if reachedBy[e.To] == nil {
			reachedBy[e.To] = map[string]bool{}
		}
		reachedBy[e.To][e.From] = true
		cited[e.To] += len(e.Prov)
		used[e.To] = used[e.To] || e.Label != archdoc.Configured
	}
	sort.SliceStable(externals, func(i, j int) bool {
		a, b := externals[i], externals[j]
		if len(reachedBy[a]) != len(reachedBy[b]) {
			return len(reachedBy[a]) > len(reachedBy[b])
		}
		if used[a] != used[b] {
			return used[a]
		}
		if cited[a] != cited[b] {
			return cited[a] > cited[b]
		}
		return a < b
	})
	drop := map[string]bool{}
	for _, id := range externals[MainExternals:] {
		drop[id] = true
	}
	out := view
	out.Nodes, out.Edges = nil, nil
	for _, n := range view.Nodes {
		if !drop[n.ID] {
			out.Nodes = append(out.Nodes, n)
		}
	}
	for _, e := range view.Edges {
		if !drop[e.From] && !drop[e.To] {
			out.Edges = append(out.Edges, e)
		}
	}
	out.Folded = len(drop)
	return out
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
	// Which arrows: each component's strongest uses of the others shown, by how many imports. The
	// device the full view uses — say "used by most" on a box and leave those arrows out — fails
	// here: among the main components of a tightly knit application nearly every one is used by
	// most, and every arrow went. Immich's mobile app drew sixteen boxes and no lines, with 155
	// uses between them.
	out := map[string][]archdoc.Edge{}
	among := 0
	for _, e := range view.Edges {
		if keep[e.From] && keep[e.To] {
			out[e.From] = append(out[e.From], e)
			among++
		}
	}
	var edges []archdoc.Edge
	for i := range nodes {
		nodes[i].UsedBy, nodes[i].UsesMany, nodes[i].Among = 0, 0, 0
		uses := out[nodes[i].ID]
		sort.SliceStable(uses, func(a, b int) bool {
			if uses[a].Weight != uses[b].Weight {
				return uses[a].Weight > uses[b].Weight
			}
			return uses[a].To < uses[b].To
		})
		if len(uses) > mainUses {
			uses = uses[:mainUses]
		}
		edges = append(edges, uses...)
	}
	total := len(view.Nodes)
	view.Nodes, view.Edges = nodes, edges
	describeParts(&view, m)
	view.Boundary = fmt.Sprintf("%s — the %d components that handle the most, of %d", view.Name, len(nodes), total)
	if len(handles) == 0 {
		// Nothing here declares a route, a page or a job — a mobile app's code — so the rule is size.
		view.Boundary = fmt.Sprintf("%s — its %d largest components, of %d", view.Name, len(nodes), total)
	}
	if among > len(edges) {
		view.Boundary += fmt.Sprintf(" · each one's %d strongest uses, of %d between them", mainUses, among)
	}
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  MainPrefix + id,
		File:  "main-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Main components of " + view.Name,
		Model: view,
		Group: true,
	}
}

// describeParts puts on each component's box the first sentence a model wrote about what it does
// (F-19), where one is current — marked as the model's, as a container's description is. A view
// of more components than a diagram shows is left to names: seventy-five paragraphs are not a
// picture. An answer about an earlier version of the code is not shown as if it were about this.
func describeParts(view *archdoc.Model, m archdoc.Model) {
	said := map[string]archdoc.Explanation{}
	for _, x := range m.Explanations {
		if !x.Stale && len(x.Claims) > 0 {
			said[x.Element] = x
		}
	}
	for i := range view.Nodes {
		n := &view.Nodes[i]
		if x, ok := said[n.ID]; ok && n.Description == "" {
			n.Description = x.Claims[0].Text
			n.DescProv = archdoc.Provenance{Origin: archdoc.Semantic, Note: x.Prov.Note}
		}
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
	return (strings.HasPrefix(name, "component-") || strings.HasPrefix(name, "data-") || strings.HasPrefix(name, "structure-") || strings.HasPrefix(name, "main-")) &&
		(strings.HasSuffix(name, ".svg") || strings.HasSuffix(name, ".mmd"))
}
