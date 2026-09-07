package archdoc

import "sort"

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
)

// container reports whether this kind is a C4 container.
//
// The derivation rule, from docs/how-it-works.md: a Compose service becomes a C4 container when
// it is an application or a data store. Everything else is infrastructure.
func (k Kind) container() bool {
	return k == Application || k == Datastore || k == Queue
}

// rank orders kinds for output. Not alphabetical: a diagram reads better outside-in, and a
// provenance table reads better the same way — who uses the system, what it is made of, what it
// depends on. Alphabetical would put externals in the middle for no reason.
func (k Kind) rank() int {
	switch k {
	case Actor:
		return 0
	case System, Application:
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
	Kind Kind   `json:"kind"`

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

	Prov Provenance `json:"provenance"`
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`

	// Label is what the relationship does, in the reader's terms — "reads from", "publishes
	// to". Kept short: C4 relationship labels are verbs, not sentences.
	Label string `json:"label,omitempty"`

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
	for _, n := range m.Nodes {
		if n.Kind.container() || n.Kind == Actor || n.Kind == External {
			keep[n.ID] = true
		}
	}
	return m.project(keep, nil)
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

	view := m.project(keep, remap)

	view.Nodes = append([]Node{system}, view.Nodes...)
	view.Edges = dedupe(dropSelfEdges(view.Edges))
	sortNodes(view.Nodes)

	return view
}

// project builds a view containing only the kept nodes. Edges touching a dropped node are
// rewritten by remap — either onto a replacement node, or bridged through the dropped one when
// remap is nil.
func (m Model) project(keep map[string]bool, remap func(string) string) Model {
	view := Model{Name: m.Name, Source: m.Source, Networks: m.Networks}

	for _, n := range m.Nodes {
		if keep[n.ID] {
			view.Nodes = append(view.Nodes, n)
		}
	}

	edges := m.Edges
	if remap == nil {
		edges = bridge(edges, keep)
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
// One hop only. A chain of two excluded nodes is rare enough that inventing a path across it
// would be a bigger claim than the evidence supports.
func bridge(edges []Edge, keep map[string]bool) []Edge {
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
		for _, b := range edges {
			if b.From != a.To || !keep[b.To] || !keep[a.From] || !b.Traffic {
				continue
			}
			// The label is the first hop's: this is a's relationship, extended past the
			// infrastructure in the way rather than b's relationship re-attributed.
			out = append(out, Edge{
				From:       a.From,
				To:         b.To,
				Label:      firstNonEmpty(a.Label, b.Label),
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
		out[i].Technology = firstNonEmpty(out[i].Technology, e.Technology)

		// The stronger evidence names the relationship. Where depends_on and a configured
		// endpoint describe the same pair, "connects to postgres" is what the reader needs;
		// "depends on" would be true and would waste the better fact.
		if e.Traffic && !out[i].Traffic {
			out[i].Label = e.Label
		} else if out[i].Label == "" {
			out[i].Label = e.Label
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
	sort.Slice(ns, func(i, j int) bool {
		if ns[i].Kind != ns[j].Kind {
			return ns[i].Kind.rank() < ns[j].Kind.rank()
		}
		return ns[i].ID < ns[j].ID
	})
}
