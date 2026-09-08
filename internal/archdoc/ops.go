package archdoc

// Operations are how anything other than extraction changes the model.
//
// Three producers, one vocabulary: `rules.yaml` corrections a person wrote, the semantic layer's
// suggestions, and — in R1.c — the editable canvas. Each emits operations; the validator checks
// them; only then are they applied. Building one vocabulary rather than three means the
// write-safety story is designed once, and it is what lets the validator stand between the
// language model and the output rather than beside it.
//
// SEM-10 is the reason this exists at all: the model returns a diff, never a whole model. A tool
// that hands back a complete model has to be trusted; one that hands back a list of small,
// typed, checkable changes does not.

// Origin says where a fact or an operation came from. It is the answer to "how do you know?"
// for things a file did not directly state.
//
// PRV-02. Without it, a catalog lookup and a parsed line look identical in the output, and
// AC-1 — which admits extraction *or* catalog provenance — cannot be measured. It is also the
// prerequisite for PRV-05: a reader cannot be shown which parts of a diagram are interpretation
// unless something records that they are.
type Origin string

const (
	// Extraction means a file said it, at a line. The strongest evidence there is.
	Extraction Origin = "extraction"
	// Catalog means a lookup table supplied it — an image name resolved to a technology.
	// Deterministic and repeatable, but not something the repository stated.
	Catalog Origin = "catalog"
	// Rules means a person wrote it in rules.yaml, which has a file and a line of its own.
	Rules Origin = "rules"
	// Semantic means the semantic layer suggested it. Never a fact; only ever a label,
	// description or grouping placed on top of one.
	Semantic Origin = "model"
)

// Interpretation reports whether this origin is a judgement rather than a reading. What the
// model supplies is interpretation; what a file states is not. PRV-05 renders the difference.
func (o Origin) Interpretation() bool { return o == Semantic }

// OpKind is what an operation does.
type OpKind string

const (
	// SetName changes an element's display name — "immich-server" to "Immich Server".
	SetName OpKind = "set_name"
	// SetDescription gives an element the one-line responsibility a C4 container should
	// carry and configuration never states.
	SetDescription OpKind = "set_description"
	// SetTechnology fills what the catalog could not: a custom-built image's stack.
	SetTechnology OpKind = "set_technology"
	// SetKind reclassifies an element — Supabase's connection pooler is the standing
	// example, and exactly the judgement call rules.yaml exists to settle.
	SetKind OpKind = "set_kind"
	// SetEdgeLabel replaces a deterministic label with a better verb. "connects to" is true
	// and dull; "reads user profiles from" is what a reader wants.
	SetEdgeLabel OpKind = "set_edge_label"
	// Group places elements in a named logical grouping.
	Group OpKind = "group"

	// Exclude removes an element from the views. Structural, so rules only.
	Exclude OpKind = "exclude"
	// AddEdge asserts a relationship. Structural, so rules only.
	AddEdge OpKind = "add_edge"
	// RemoveEdge withdraws one. Structural, so rules only.
	RemoveEdge OpKind = "remove_edge"
)

// structural reports whether an operation changes what exists rather than how it is described.
func (k OpKind) structural() bool {
	switch k {
	case Exclude, AddEdge, RemoveEdge:
		return true
	default:
		return false
	}
}

// AllowedFrom reports whether an origin may produce this kind of operation.
//
// **This is where hard rule 4 stops being a promise.** The language model may name, describe,
// classify and group; it may never add, remove or exclude an element. A person writing
// rules.yaml may do all of it, because a person is accountable and a file records what they
// said.
//
// Enforcing it here rather than in the semantic layer means the check cannot be forgotten by
// whoever writes the next producer, and it means AC-2 — the same diagram with the model
// switched off — is a property of the type system rather than of anyone's discipline.
func (k OpKind) AllowedFrom(o Origin) bool {
	if o == Semantic {
		return !k.structural()
	}
	return true
}

// Op is one checked change to the model.
type Op struct {
	Kind OpKind `json:"kind"`

	// Target is the element the operation applies to. For edge operations it is the source
	// node, and To is the destination.
	Target string `json:"target"`
	To     string `json:"to,omitempty"`

	// Value is the new name, description, technology, kind, label or group.
	Value string `json:"value,omitempty"`

	Origin Origin `json:"origin"`

	// Prov is where the operation itself came from — a line in rules.yaml, or the model run
	// that proposed it. An applied operation stamps this onto whatever it changed, which is
	// how a description ends up traceable to the run that wrote it.
	Prov Provenance `json:"provenance"`
}
