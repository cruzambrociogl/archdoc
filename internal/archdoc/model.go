package archdoc

import (
	"sort"
	"strings"
)

// Kind is what a node *is*, which decides whether it may appear in a given view. §MDL-07.
//
// The set is deliberately small. Compose describes deployment; a C4 container diagram must not
// show deployment concepts, so the kinds here are the ones that survive that translation.
type Kind string

const (
	// Application is something that executes code — an API, a worker, a frontend.
	Application Kind = "application"
	// Datastore holds state — a database, an object store, a search index.
	Datastore Kind = "datastore"
	// Queue moves messages between applications.
	Queue Kind = "queue"
	// Proxy routes traffic without owning behaviour — gateways, load balancers, poolers.
	// Infrastructure: it stays in the model and leaves the container view.
	Proxy Kind = "proxy"
	// External is a system this repository points at but does not define.
	External Kind = "external"
	// System is the whole thing under documentation, collapsed into one box. It exists only
	// in the context view, where everything declared becomes a single element.
	System Kind = "system"
	// Actor is a person or system outside the boundary that reaches in.
	Actor Kind = "actor"
	// Component is a part of an application's code — a directory under its source root, or a
	// Python module — inside the container named by Parent (F-03). It appears only in that
	// container's component view.
	Component Kind = "component"
	// Table is a table the code declares — a class it marks as one, with its columns — inside the
	// container whose code declares it (F-07). It appears only in that container's data view.
	Table Kind = "table"
	// Module is a folder of an application's code, inside the container named by Parent. It
	// appears only in that container's "by folder" view, where components are features and the
	// folders are the other way to read the same code.
	Module Kind = "module"
)

// Part reports whether this kind lives inside a container, read from its code, rather than being
// one of the elements configuration describes: a component, a table.
func (k Kind) Part() bool {
	return k == Component || k == Table || k == Module
}

// Container reports whether this kind is a C4 container.
//
// The derivation rule, from docs/how-it-works.md: a Compose service becomes a C4 container when
// it is an application or a data store. Everything else is infrastructure.
func (k Kind) Container() bool {
	return k == Application || k == Datastore || k == Queue
}

// rank orders kinds for output. Not alphabetical: a diagram reads better outside-in, and a
// provenance table reads better the same way — who uses the system, what it is made of, what it
// depends on. Alphabetical would put externals in the middle for no reason.
func (k Kind) rank() int {
	switch k {
	case Actor:
		return 0
	case System, Application, Component, Table, Module:
		return 1
	case Datastore:
		return 2
	case Queue:
		return 3
	case Proxy:
		return 4
	case External:
		return 5
	default:
		return 6
	}
}

// Node is one element of the model. Every node carries the line that proves it exists — a node
// without known provenance must not be emitted (P1).
type Node struct {
	ID   string `json:"id"`   // stable across runs and renames (MDL-13)
	Name string `json:"name"` // what a reader is shown

	// NameProv is set when the display name was changed after extraction — by a rule or by
	// the model. Empty means the name is the key the configuration declared.
	NameProv Provenance `json:"name_provenance,omitzero"`
	Kind     Kind       `json:"kind"`

	// Description is the one-line responsibility a C4 container should carry. Configuration
	// never states it, so it stays empty until the semantic layer or a rule supplies one.
	Description string `json:"description,omitempty"`

	// Technology is what runs *inside* the container — "PostgreSQL 14", "Node.js". Never
	// "Docker": that is the deployment mechanism, not an architectural choice. Empty until
	// the catalog fills it.
	Technology string `json:"technology,omitempty"`

	Evidence EvidenceKind `json:"evidence"`

	// Parent is the container a component lives inside. Empty at levels 1 and 2; the field
	// exists so level 3 is a filter over the same model rather than a second one.
	Parent string `json:"parent,omitempty"`

	// Networks this node is attached to, in name order (MDL-09).
	Networks []string `json:"networks,omitempty"`

	// Dir is the repository-relative directory of the application's own code, when one was
	// found (F-02); DirProv proves the tie — the manifest, or the Compose line that builds the
	// service from that directory. What lies inside is the component level's evidence.
	Dir     string     `json:"dir,omitempty"`
	DirProv Provenance `json:"dir_provenance,omitzero"`

	// Files are a component's files, repository-relative, in path order, and Lines their total —
	// what the component is made of, and how much of it there is.
	Files []string `json:"files,omitempty"`
	Lines int      `json:"lines,omitempty"`

	// Columns are a table's columns, in the order the code declares them.
	Columns []Column `json:"columns,omitempty"`

	// UsedBy and Among are set on a component most of the others use — utilities, shared types —
	// in its component view only: it is used by UsedBy of the Among others, and the picture says
	// that in words on its box rather than with an arrow from each. The arrows stay in the model.
	UsedBy int `json:"used_by,omitempty"`
	Among  int `json:"among,omitempty"`
	// UsesMany is the mirror: a part that uses that many of the others — a base class given every
	// repository, an index file that re-exports a folder — whose outgoing arrows are not drawn either.
	UsesMany int `json:"uses_many,omitempty"`

	// DescProv and TechProv are separate from Prov because they can come from somewhere
	// else. A node is proven by the line that declares it; its technology may come from the
	// catalog and its description from the model. PRV-05 must tell a reader which parts of a
	// box were read and which were interpreted, and one provenance per node cannot.
	DescProv Provenance `json:"description_provenance,omitzero"`
	TechProv Provenance `json:"technology_provenance,omitzero"`

	Prov Provenance `json:"provenance"`
}

// Column is one column of a table: its name and type as the code declares them, and the table
// it references when it is a foreign key.
type Column struct {
	Name       string     `json:"name"`
	Type       string     `json:"type,omitempty"`
	Primary    bool       `json:"primary,omitempty"`
	Nullable   bool       `json:"nullable,omitempty"`
	References string     `json:"references,omitempty"` // a table node's ID
	Prov       Provenance `json:"provenance"`
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`

	// Label is what the relationship does, in the reader's terms — "reads from", "publishes
	// to". Kept short: C4 relationship labels are verbs, not sentences.
	Label string `json:"label,omitempty"`

	// LabelProv is where the label came from when something other than extraction wrote it.
	// Kept apart from Prov on purpose: Prov is the evidence that the relationship exists, and
	// a model that only reworded the label must never appear as evidence for the arrow.
	LabelProv Provenance `json:"label_provenance,omitzero"`

	// Technology is how, when the configuration says so — a URL scheme, a known port.
	Technology string `json:"technology,omitempty"`

	// Traffic distinguishes two kinds of evidence that look alike and are not.
	//
	// An endpoint or a published port says something *flows*: a host was configured, so it is
	// reached. depends_on says only that one service starts before another. Both are worth
	// drawing, but only the first justifies reasoning about a path — bridging a route through
	// an excluded gateway on start-order evidence invents a relationship the file never
	// declared.
	Traffic bool `json:"traffic,omitempty"`

	// Prov is plural because one relationship can be attested more than once: declared in
	// depends_on *and* named by an environment URL. It also lets an edge derived by bridging
	// through an excluded proxy cite both hops it was built from, rather than appearing from
	// nowhere.
	Prov []Provenance `json:"provenance"`

	// Weight is how many times the relationship is attested when that is more than its citations
	// show: a component edge cites one import per importing file, at most ten, and counts them all.
	Weight int `json:"weight,omitempty"`
}

// Model is the whole system as archdoc understands it: one graph, from which every view is a
// projection. The C4 levels are not extraction levels — they are filters over this.
type Model struct {
	// Name is the system under documentation.
	Name string `json:"name"`

	// Source is the file the model was derived from, repository-relative.
	Source string `json:"source"`

	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`

	// Networks carries what the file declared about each network, so a view can name a
	// boundary and cite the line that created it.
	Networks []Network `json:"networks,omitempty"`

	// Boundary is what a view's enclosing box says, when it is not the system: a component view
	// is drawn inside its container — "immich-server [Container: NestJS · TypeScript]".
	Boundary string `json:"boundary,omitempty"`

	// Entries are the ways into the system its code declares — HTTP routes, today (F-04, F-13).
	// They are what the system does, read from where it says so; the features a reader looks for.
	Entries []Entry `json:"entries,omitempty"`

	// Unresolved is what the code does that archdoc saw and could not tie to an element: a call
	// to an address computed at run time. Shown, never dropped (D-6, F-32).
	Unresolved []Unresolved `json:"unresolved,omitempty"`

	// Flows are what happens when an entry is called, followed through the code (F-12).
	Flows []Flow `json:"flows,omitempty"`

	// Dependencies are the third-party packages each container's manifest declares, and which
	// of its components import them (F-14).
	Dependencies []Package `json:"dependencies,omitempty"`

	// Explanations are model-written prose about an element — what a component does — every
	// sentence citing the facts it rests on (F-19, F-36). Interpretation, marked as such.
	Explanations []Explanation `json:"explanations,omitempty"`
}

// Package is a package a container's manifest declares: where it is declared, and where the
// code uses it. One declared and never imported is still listed — it may be used by a tool, or
// not at all — with no components.
type Package struct {
	Container  string     `json:"container"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	Dev        bool       `json:"dev,omitempty"`
	Imports    int        `json:"imports"`              // how many import statements name it
	Components []string   `json:"components,omitempty"` // the components that import it
	Prov       Provenance `json:"provenance"`           // the manifest line
}

// Explanation is what the model wrote about one element, and what it was given to write it.
// Fingerprint identifies those facts: while they are unchanged, the explanation is reused rather
// than asked for again (F-30), so a run with no change costs nothing and changes nothing.
type Explanation struct {
	Element     string  `json:"element"`
	Claims      []Claim `json:"claims"`
	Fingerprint string  `json:"fingerprint"`
	// Stale is set when the element's facts have changed since this was written: the last answer
	// is shown rather than none, and says so, until the model is asked again.
	Stale bool       `json:"stale,omitempty"`
	Prov  Provenance `json:"provenance"` // Origin Semantic; the note names the model
}

// Claim is one sentence and the facts it cites. Every cite resolves to a fact archdoc gave the
// model, each with the line that proves it; a sentence that cited nothing, or something it was
// not given, was refused before it got here (F-36).
type Claim struct {
	Text  string       `json:"text"`
	Facts []string     `json:"facts"` // the facts as the model saw them
	Cites []Provenance `json:"cites"` // where each is proven
}

// Flow is the calls an entry sets off, in the order the code makes them: from its handler into
// the classes it is given, down to the tables their queries name and the calls that leave the
// container. Followed by name — a field's declared type — to a fixed depth; Cut says it was cut.
type Flow struct {
	Entry        string        `json:"entry"`
	Participants []Participant `json:"participants"`
	Steps        []Step        `json:"steps"`
	Cut          bool          `json:"cut,omitempty"`
}

// Participant is a lifeline in a flow: a class, a table, a job put on a queue, or a call whose
// target is unknown.
type Participant struct {
	ID        string `json:"id"` // "AlbumService", "table:album", "job:AssetDelete", "unresolved"
	Name      string `json:"name"`
	Kind      string `json:"kind"` // "class", "module", "table", "job", "unresolved"
	Component string `json:"component,omitempty"`
	Element   string `json:"element,omitempty"` // a table's node, or a queued job's entry
}

// Step is one call in a flow, at the line that makes it.
type Step struct {
	From  string     `json:"from"`
	To    string     `json:"to"`
	Call  string     `json:"call"` // "getAll", "reads"
	Depth int        `json:"depth"`
	Note  string     `json:"note,omitempty"`
	Prov  Provenance `json:"provenance"`
}

// Entry is one way into the system: an HTTP route, its handler, and what the handler is given.
type Entry struct {
	// ID is what the entry is — its container, method and path — so moving the handler to another
	// file is not a new feature (F-31).
	ID      string `json:"id"`
	Kind    string `json:"kind"` // "http"
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"` // "AlbumController.getAllAlbums"
	// Summary is the code's own description of the route — a summary its decorators state.
	Summary     string     `json:"summary,omitempty"`
	SummaryProv Provenance `json:"summary_provenance,omitzero"`
	Container   string     `json:"container"`
	Component   string     `json:"component,omitempty"`
	// Uses are what the handler's class is given — NestJS's injected services — each resolved by
	// name to where it is declared.
	Uses []Symbol `json:"uses,omitempty"`
	// PrefixProv cites the global prefix the path begins with, when one was applied.
	PrefixProv Provenance `json:"prefix_provenance,omitzero"`
	// PathNote says how a part of the path was resolved when it was not written out: a constant
	// resolved by name, or an expression archdoc could not resolve, kept in braces.
	PathNote string     `json:"path_note,omitempty"`
	Prov     Provenance `json:"provenance"`
}

// Group is how a feature list groups an entry: a route by the class or module that handles it, a
// page by the first segment of its path.
func (e Entry) Group() string {
	switch e.Kind {
	case "page":
		first, _, _ := strings.Cut(strings.TrimPrefix(e.Path, "/"), "/")
		return "Pages /" + first
	case "command":
		return "Commands"
	}
	cls, _, _ := strings.Cut(e.Handler, ".")
	return cls
}

// Symbol is a named thing in the code, resolved to where it is declared. How says how the name
// was tied: by name means a class of that name was found in the application's own code (D-4).
type Symbol struct {
	Name      string     `json:"name"`
	Component string     `json:"component,omitempty"`
	How       string     `json:"how"` // "name", or "unresolved" when no such class was found
	Prov      Provenance `json:"provenance"`
}

// Unresolved is one thing the code does that could not be tied to an element.
type Unresolved struct {
	Container string     `json:"container"`
	Component string     `json:"component,omitempty"`
	What      string     `json:"what"` // "fetch(new URL('predict', url))"
	Note      string     `json:"note"`
	Prov      Provenance `json:"provenance"`
}

// Normalise puts a freshly built model into the shape every view expects: one edge per pair of
// endpoints with its citations unioned, no self-edges, and a deterministic order throughout.
//
// A model is built by appending, so the same relationship can arrive twice — once from
// depends_on and once from an environment URL naming the same host. Those are one relationship
// with two citations.
func (m Model) Normalise() Model {
	m.Edges = dedupe(dropSelfEdges(m.Edges))
	sortNodes(m.Nodes)
	return m
}

// Node returns the node with the given ID.
func (m Model) Node(id string) (Node, bool) {
	for _, n := range m.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// Container projects the model into a C4 container view (VIE-02).
//
// Applications, data stores and queues become containers. Actors and external systems stay,
// because a container diagram shows what surrounds the system as well as what is in it.
// Infrastructure — proxies, gateways, poolers — is excluded, and edges are bridged through it:
// with everything routed via a gateway a diagram looks hub-and-spoke and hides the real
// couplings.
func (m Model) Container() Model {
	keep := map[string]bool{}
	reach := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Kind.Container() || n.Kind == Actor || n.Kind == External {
			keep[n.ID] = true
		}
		// Only an actor may originate a bridge. See bridge() for why.
		if n.Kind == Actor {
			reach[n.ID] = true
		}
	}
	return m.project(keep, reach, nil)
}

// Context projects the model into a C4 context view (VIE-01).
//
// Everything the repository *declares* collapses into one box; everything it merely
// *references* stays outside it, along with the actors. The evidence kind already draws that
// line — declared means the repository defines it, so it is inside the system — which is why
// this view costs almost nothing once container works.
func (m Model) Context() Model {
	system := Node{
		ID:       "system",
		Name:     m.Name,
		Kind:     System,
		Evidence: Declared,
	}

	keep := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Kind.Part() {
			continue // a part of a container is inside the system twice over; it has no say in the box
		}
		if n.Kind == Actor || n.Evidence == Referenced {
			keep[n.ID] = true
			continue
		}
		// Declared: collapses into the system box. Its provenance is the earliest one, so
		// the box still cites a real line.
		if !system.Prov.Known() || less(n.Prov, system.Prov) {
			system.Prov = n.Prov
		}
	}

	remap := func(id string) string {
		if keep[id] {
			return id
		}
		return system.ID
	}
	keep[system.ID] = true

	view := m.project(keep, nil, remap)

	view.Nodes = append([]Node{system}, view.Nodes...)
	view.Edges = dedupe(dropSelfEdges(view.Edges))
	sortNodes(view.Nodes)

	return view
}

// Component projects one container into its C4 component view (F-10): the components whose
// parent it is, and how they use each other. The view is named after the container.
func (m Model) Component(of string) Model { return m.inside(of, Component) }

// Data projects one container into its data view (F-11): the tables its code declares, and the
// foreign keys between them.
func (m Model) Data(of string) Model { return m.inside(of, Table) }

// Structure projects one container into its folders: the same code as its component view, by
// where the files are rather than by what they are for.
func (m Model) Structure(of string) Model { return m.inside(of, Module) }

func (m Model) inside(of string, kind Kind) Model {
	keep := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Kind == kind && n.Parent == of {
			keep[n.ID] = true
		}
	}
	view := m.project(keep, nil, func(id string) string { return id })
	view.Networks = nil
	// What the code says about the parts on view travels with them: the routes a component
	// handles, and what the model wrote about each.
	for _, e := range m.Entries {
		if keep[e.Component] {
			view.Entries = append(view.Entries, e)
		}
	}
	for _, x := range m.Explanations {
		if keep[x.Element] {
			view.Explanations = append(view.Explanations, x)
		}
	}
	if c, ok := m.Node(of); ok {
		view.Name = c.Name
		view.Boundary = c.Name + " [Container"
		if c.Technology != "" {
			view.Boundary += ": " + c.Technology
		}
		view.Boundary += "]"
	}
	return view
}

// Components lists the containers that have a component view, in model order.
func (m Model) Components() []string { return m.holding(Component) }

// Datas lists the containers that have a data view, in model order.
func (m Model) Datas() []string { return m.holding(Table) }

// Structures lists the containers that have a by-folder view beside their components.
func (m Model) Structures() []string { return m.holding(Module) }

func (m Model) holding(kind Kind) []string {
	has := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Kind == kind && n.Parent != "" {
			has[n.Parent] = true
		}
	}
	var out []string
	for _, n := range m.Nodes {
		if has[n.ID] {
			out = append(out, n.ID)
		}
	}
	return out
}

// project builds a view containing only the kept nodes. Edges touching a dropped node are
// rewritten by remap — either onto a replacement node, or bridged through the dropped one when
// remap is nil.
func (m Model) project(keep, reach map[string]bool, remap func(string) string) Model {
	view := Model{Name: m.Name, Source: m.Source, Networks: m.Networks}

	for _, n := range m.Nodes {
		if keep[n.ID] {
			view.Nodes = append(view.Nodes, n)
		}
	}

	edges := m.Edges
	if remap == nil {
		edges = bridge(edges, keep, reach)
	} else {
		out := make([]Edge, 0, len(edges))
		for _, e := range edges {
			e.From, e.To = remap(e.From), remap(e.To)
			out = append(out, e)
		}
		edges = out
	}

	for _, e := range edges {
		if keep[e.From] && keep[e.To] {
			view.Edges = append(view.Edges, e)
		}
	}

	view.Edges = dedupe(dropSelfEdges(view.Edges))
	sortNodes(view.Nodes)

	return view
}

// bridge replaces paths that pass through a dropped node with direct edges. a→proxy→b becomes
// a→b, citing both hops: the relationship is real, and the reader can still check it.
//
// Two restrictions, both learned from real output rather than from tests.
//
// **Both hops must carry traffic.** A path made of start-order evidence is not a path — see the
// second loop.
//
// **Only a node in reach may originate one.** A gateway routes by path, and a bridge cannot see
// paths: joining every inbound edge to every outbound one turns one call into a fan-out across
// everything the gateway serves. Supabase showed this exactly — `functions` sets
// SUPABASE_URL=http://api-gw:8000 and calls one endpoint, and an unrestricted bridge drew it
// reaching all seven services behind the gateway.
//
// reach holds the nodes whose inbound evidence is *reachability* rather than a specific call:
// an actor, which arrived from a published port. "Anyone outside can reach whatever this
// gateway routes to" is true and is what a public entry point means. "This service calls
// everything behind the gateway" is not. Resolving the rest needs route paths, which is more of
// MDL-03 than R1.a builds.
//
// One hop only. A chain of two excluded nodes is rare enough that inventing a path across it
// would be a bigger claim than the evidence supports.
func bridge(edges []Edge, keep, reach map[string]bool) []Edge {
	out := make([]Edge, 0, len(edges))
	for _, e := range edges {
		if keep[e.From] && keep[e.To] {
			out = append(out, e)
		}
	}

	// Then, for every dropped node, join what reaches it to what it reaches.
	//
	// Both hops must carry traffic. A path made of start-order evidence is not a path: a
	// gateway that depends_on an admin console does not thereby route users to it, and
	// bridging on that evidence would draw a relationship the file never declared. Routes
	// live in the gateway's own configuration, which is MDL-03 and a source archdoc does not
	// yet read — so where the evidence stops, so does the arrow.
	for _, a := range edges {
		if keep[a.To] || !a.Traffic {
			continue
		}
		if !reach[a.From] {
			continue
		}
		for _, b := range edges {
			if b.From != a.To || !keep[b.To] || !keep[a.From] || !b.Traffic {
				continue
			}
			label, labelProv := bridgeLabel(a, b)
			out = append(out, Edge{
				From:       a.From,
				To:         b.To,
				Label:      label,
				LabelProv:  labelProv,
				Technology: firstNonEmpty(b.Technology, a.Technology),
				Traffic:    true,
				Prov:       append(append([]Provenance{}, a.Prov...), b.Prov...),
			})
		}
	}

	return out
}

func dropSelfEdges(edges []Edge) []Edge {
	out := make([]Edge, 0, len(edges))
	for _, e := range edges {
		if e.From != e.To {
			out = append(out, e)
		}
	}
	return out
}

// dedupe merges edges with the same endpoints, unioning their provenance. Two attestations of
// one relationship are one edge with two citations, not two edges.
func dedupe(edges []Edge) []Edge {
	index := map[string]int{}
	out := make([]Edge, 0, len(edges))

	for _, e := range edges {
		key := e.From + "\x00" + e.To
		i, seen := index[key]
		if !seen {
			index[key] = len(out)
			out = append(out, e)
			continue
		}
		out[i].Prov = append(out[i].Prov, e.Prov...)
		out[i].Weight += e.Weight
		out[i].Technology = firstNonEmpty(out[i].Technology, e.Technology)

		// The stronger evidence names the relationship. Where depends_on and a configured
		// endpoint describe the same pair, "connects to postgres" is what the reader needs;
		// "depends on" would be true and would waste the better fact.
		// The label's citation travels with the label, whichever edge it came from.
		if e.Traffic && !out[i].Traffic {
			out[i].Label, out[i].LabelProv = e.Label, e.LabelProv
		} else if out[i].Label == "" {
			out[i].Label, out[i].LabelProv = e.Label, e.LabelProv
		}
		out[i].Traffic = out[i].Traffic || e.Traffic
	}

	for i := range out {
		out[i].Prov = dedupeProv(out[i].Prov)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})

	return out
}

func dedupeProv(ps []Provenance) []Provenance {
	seen := map[Provenance]bool{}
	out := make([]Provenance, 0, len(ps))
	for _, p := range ps {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

// bridgeLabel picks the label for an edge reconnected past excluded infrastructure.
//
// The second hop's label wins when a person or the model wrote it. That hop is the one that says
// what reaches the target — each gateway route has its own line in the gateway's config, and a
// label written for it ("forwards signup and token requests to") describes that route
// specifically. Found on a live Supabase run: keeping the first hop's label gave all seven user
// arrows the same generic text, while the specific route labels existed and were hidden with the
// gateway.
//
// Otherwise the first hop's extracted label stays, as before: with no model, "reaches" reads
// better on a user's arrow than the gateway's "routes to".
func bridgeLabel(a, b Edge) (string, Provenance) {
	if b.Label != "" && b.LabelProv.Known() {
		return b.Label, b.LabelProv
	}
	if a.Label != "" {
		return a.Label, a.LabelProv
	}
	return b.Label, b.LabelProv
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func less(a, b Provenance) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	return a.Line < b.Line
}

// sortNodes gives every view a deterministic order. Go randomises map iteration, and AC-7
// requires five consecutive runs to produce byte-identical output.
func sortNodes(ns []Node) {
	// By rank, then by ID — and by nothing else. Comparing kinds first, as this once did, is not an
	// order at all where two kinds share a rank: a component and a table were each "not before"
	// the other, so where either landed depended on what else was in the list, and adding one
	// element reshuffled the rest.
	sort.Slice(ns, func(i, j int) bool {
		if ri, rj := ns[i].Kind.rank(), ns[j].Kind.rank(); ri != rj {
			return ri < rj
		}
		return ns[i].ID < ns[j].ID
	})
}
