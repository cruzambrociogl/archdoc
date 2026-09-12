package archdoc

// A Layout is where things sit in one view: computed once by the layout engine, stored with the
// version, and drawn from — never recomputed by whatever displays it. That is what lets the
// committed SVG and a future web canvas show the same picture (VIE-10), and what keeps an
// unchanged architecture from reshuffling between runs (VIE-04, in its weaker form: see
// docs/decisions.md on why positions are not yet pinned across *changed* architectures).
//
// Coordinates are SVG coordinates in points: origin top-left, y growing downward.
type Layout struct {
	// Version is the layout engine's version when this was computed. A stored layout is reused
	// only when it matches the running engine, so changing how layouts are computed takes
	// effect even for an architecture that has not changed.
	Version int `json:"version"`

	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	Boxes  []Box      `json:"boxes"`
	Groups []Boundary `json:"groups,omitempty"`
	Paths  []Path     `json:"paths,omitempty"`
}

// Point is one coordinate.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Rect is an axis-aligned rectangle, top-left corner plus size.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Box is one element's rectangle, keyed by the element's model ID.
type Box struct {
	ID   string `json:"id"`
	Rect Rect   `json:"rect"`
}

// Boundary is a boundary drawn around elements: the system, or a network inside it.
type Boundary struct {
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`    // the text drawn; given to the engine so it reserves room
	Internal bool   `json:"internal,omitempty"` // Compose's `internal: true`
	System   bool   `json:"system,omitempty"`   // the outer system boundary, not a network
	Rect     Rect   `json:"rect"`
	LabelAt  Point  `json:"label_at"`
}

// Path is one relationship's route: the curve from source to target, where its arrowhead points,
// and where its label is centred.
type Path struct {
	From string `json:"from"`
	To   string `json:"to"`

	// Curve is a cubic Bézier chain: a start point, then groups of three (two control points
	// and an end point). The format Graphviz produces, kept as-is.
	Curve []Point `json:"curve"`

	// Tip is where the arrowhead touches the target. Empty when the engine drew no head.
	Tip *Point `json:"tip,omitempty"`

	LabelAt *Point `json:"label_at,omitempty"`
}
