package render

import (
	"fmt"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Look is how a diagram is coloured (platform-surface §16.1, the Classic C4 switch): archdoc's own —
// kind as a hue, a tinted fill and a cap — or C4's usual blues. The layout, the boxes and the rules
// are the same in both.
type Look string

const (
	Archdoc Look = "archdoc"
	Classic Look = "classic"
)

// LookOf reads a look from a request: "archdoc" is archdoc's; anything else, Classic C4 — the look
// the committed files have always had.
func LookOf(s string) Look {
	if s == string(Archdoc) {
		return Archdoc
	}
	return Classic
}

type boxStyle struct {
	fill, stroke, text string
	dashed             bool
}

type palette struct {
	look   Look
	radius float64
	cap    bool
	styles map[string]boxStyle // by style key: person, system, container, store, component, external
}

// styleKey is which style a box takes: what it is, and whether the repository only names it.
func styleKey(n core.Node) string {
	switch {
	case n.Kind == core.Actor:
		return "person"
	case n.Evidence == core.Referenced || n.Kind == core.External:
		return "external"
	case n.Kind == core.System:
		return "system"
	case n.Kind == core.Datastore || n.Kind == core.Queue:
		return "store"
	case n.Kind == core.Component:
		return "component"
	case n.Kind == core.Module:
		return "folder"
	case n.Kind == core.Table:
		return "table"
	}
	return "container"
}

func (p palette) of(n core.Node) boxStyle { return p.styles[styleKey(n)] }

// The two palettes. Classic is C4's own blues, as the committed SVGs have them; archdoc's is the
// app's kind hues (web/app/src/ui/kinds.ts, styles/tokens.css): slate, teal, green, violet.
var palettes = map[Look]palette{
	Classic: {look: Classic, radius: 8, styles: map[string]boxStyle{
		"person":    {fill: "#08427b", stroke: "#052e56", text: "#ffffff"},
		"system":    {fill: "#1168bd", stroke: "#0b4884", text: "#ffffff"},
		"container": {fill: "#1168bd", stroke: "#0b4884", text: "#ffffff"},
		"store":     {fill: "#2574b8", stroke: "#0b4884", text: "#ffffff"},
		"component": {fill: "#85bbf0", stroke: "#5d82a8", text: "#0b2a4a"},
		"folder":    {fill: "#85bbf0", stroke: "#5d82a8", text: "#0b2a4a"},
		"external":  {fill: "#ffffff", stroke: "#8b8b8b", text: "#3d3d3d", dashed: true},
		"table":     {fill: "#ffffff", stroke: "#5d82a8", text: "#0b2a4a"},
	}},
	Archdoc: {look: Archdoc, radius: 3, cap: true, styles: map[string]boxStyle{
		"person":    {fill: "#f1f4ee", stroke: "#4a6b3f", text: "#33502a"},
		"system":    {fill: "#edf2f6", stroke: "#3f5b72", text: "#2c4658"},
		"container": {fill: "#edf2f6", stroke: "#3f5b72", text: "#2c4658"},
		"store":     {fill: "#d7ece8", stroke: "#1e6b66", text: "#0e4643"},
		"component": {fill: "#edf2f6", stroke: "#3f5b72", text: "#2c4658"},
		"folder":    {fill: "#edf2f6", stroke: "#3f5b72", text: "#2c4658"},
		"external":  {fill: "#efeef7", stroke: "#5d5580", text: "#4a4468", dashed: true},
		"table":     {fill: "#d7ece8", stroke: "#1e6b66", text: "#0e4643"},
	}},
}

func paletteOf(l Look) palette {
	if p, ok := palettes[l]; ok {
		return p
	}
	return palettes[Classic]
}

// Heading is what a diagram says it is (rule 23): its C4 type and scope — "Container diagram for
// Immich" — and its plain name, "Apps and services". A view that is not C4 says what it shows.
type Heading struct {
	Title string `json:"title"`
	Plain string `json:"plain"`
	// C4 is true for the five C4 diagrams; What it stores and How data moves are not C4.
	C4 bool `json:"c4"`
}

// HeadingOf names a view of the model.
func HeadingOf(m core.Model, v View) Heading {
	name := func(id string) string {
		if n, ok := m.Node(id); ok {
			return n.Name
		}
		return id
	}
	switch {
	case v.Name == "context" || v.Name == "context"+AllSuffix:
		return Heading{"System context diagram for " + m.Name, "The outside world", true}
	case v.Name == "container" || v.Name == "container"+AllSuffix:
		return Heading{"Container diagram for " + m.Name, "Apps and services", true}
	case v.Name == Deployment:
		return Heading{"Deployment diagram for " + m.Name + " · " + m.Source, "Where it runs", true}
	case strings.HasPrefix(v.Name, ComponentPrefix):
		id := strings.TrimPrefix(v.Name, ComponentPrefix)
		return Heading{"Component diagram for " + name(id), "Inside " + name(id), true}
	case strings.HasPrefix(v.Name, MainPrefix):
		id := strings.TrimPrefix(v.Name, MainPrefix)
		return Heading{"Component diagram for " + name(id) + ": its main components", "Inside " + name(id), true}
	case strings.HasPrefix(v.Name, StructurePrefix):
		id := strings.TrimPrefix(v.Name, StructurePrefix)
		return Heading{"Component diagram for " + name(id) + ", by folder", "Inside " + name(id), true}
	case strings.HasPrefix(v.Name, DataPrefix):
		id := strings.TrimPrefix(v.Name, DataPrefix)
		return Heading{"The tables of " + name(id), "What it stores", false}
	case v.Name == DataFlow:
		return Heading{"How data moves in " + m.Name, "How data moves", false}
	}
	return Heading{v.Title, v.Title, false}
}

// KeyItem is one entry of a diagram's key: a box style it uses, or the arrow.
type KeyItem struct {
	// Style is the box style ("person", "container", "store", "component", "folder", "table",
	// "external", "system"), "arrow", or "italic" for words the AI wrote.
	Style string `json:"style"`
	Label string `json:"label"`
}

// keyOrder and keyLabel: every style a key can list, in a fixed order, in plain words.
var keyOrder = []string{"person", "system", "container", "store", "component", "folder", "table", "external"}
var keyLabel = map[string]string{
	"person": "Person", "system": "Software system", "container": "Container: an app or a service",
	"store": "Container: a data store or queue", "component": "Component", "folder": "Folder",
	"table": "Table", "external": "External system: named, not declared here",
}

// KeyOf is a view's key (rule 23, 6.23): exactly the box styles it uses, in a fixed order; the arrow,
// if it draws one; and italics, if a model wrote any of its words.
func KeyOf(view core.Model) []KeyItem {
	used := map[string]bool{}
	ai := false
	for _, n := range view.Nodes {
		used[styleKey(n)] = true
		if n.NameProv.Origin.Interpretation() || n.TechProv.Origin.Interpretation() || n.DescProv.Origin.Interpretation() {
			ai = true
		}
	}
	for _, e := range view.Edges {
		if e.LabelProv.Origin.Interpretation() {
			ai = true
		}
	}
	var out []KeyItem
	for _, k := range keyOrder {
		if used[k] {
			out = append(out, KeyItem{k, keyLabel[k]})
		}
	}
	if len(view.Edges) > 0 {
		arrow := "Arrow: what one calls or uses, and how"
		if len(used) == 1 && used["table"] {
			arrow = "Arrow: a foreign key, to the table it references"
		}
		out = append(out, KeyItem{"arrow", arrow})
	}
	if ai {
		out = append(out, KeyItem{"italic", "Italic words: written by AI"})
	}
	return out
}

// Figure is a view as a file a person keeps (rule 23): its title and plain name, its key, then the
// drawing — in a look. What generate commits and what the app downloads.
func Figure(m core.Model, v View, l core.Layout, look Look) string {
	const margin, head = 16.0, 64.0
	h := HeadingOf(m, v)
	key := KeyOf(v.Model)
	pal := paletteOf(look)

	// The key's width, item by item: a swatch, a gap, the words at about six points a letter.
	keyWidth := 0.0
	for _, k := range key {
		keyWidth += 28 + float64(len(k.Label))*5.6 + 18
	}
	width := l.Width + 2*margin
	if w := keyWidth + 2*margin; w > width {
		width = w
	}
	if w := float64(len(h.Title))*8.4 + 2*margin; w > width {
		width = w
	}
	height := head + l.Height + 2*margin

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %s %s" width="%s" height="%s" font-family="Helvetica, Arial, sans-serif">`+"\n",
		num(width), num(height), num(width), num(height))
	fmt.Fprintf(&b, `<rect x="0" y="0" width="%s" height="%s" fill="#ffffff"/>`+"\n", num(width), num(height))
	fmt.Fprintf(&b, `<text x="%s" y="24" font-size="15" font-weight="bold" fill="#15201e">%s</text>`+"\n", num(margin), esc(h.Title))
	fmt.Fprintf(&b, `<text x="%s" y="24" font-size="11" fill="#46514f" text-anchor="end">%s</text>`+"\n", num(width-margin), esc(h.Plain))

	x, y := margin, 46.0
	for _, k := range key {
		switch k.Style {
		case "arrow":
			fmt.Fprintf(&b, `<path d="M%s,%s h20" stroke="#707070" stroke-width="1.2" fill="none"/><path d="M%s,%s l-6,-3.5 v7 z" fill="#707070"/>`+"\n",
				num(x), num(y), num(x+22), num(y))
		case "italic":
			fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="11" font-style="italic" fill="#46514f" dominant-baseline="middle">Aa</text>`+"\n", num(x+3), num(y))
		default:
			st := pal.styles[k.Style]
			dash := ""
			if st.dashed {
				dash = ` stroke-dasharray="3 2"`
			}
			fmt.Fprintf(&b, `<rect x="%s" y="%s" width="22" height="13" rx="%s" fill="%s" stroke="%s" stroke-width="1"%s/>`+"\n",
				num(x), num(y-6.5), num(pal.radius/2), st.fill, st.stroke, dash)
		}
		fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="10.5" fill="#46514f" dominant-baseline="middle">%s</text>`+"\n", num(x+28), num(y), esc(k.Label))
		x += 28 + float64(len(k.Label))*5.6 + 18
	}

	// The drawing beneath, as drawn on its own.
	fmt.Fprintf(&b, `<svg x="0" y="%s" width="%s" height="%s" viewBox="%s %s %s %s">`+"\n",
		num(head), num(l.Width+2*margin), num(l.Height+2*margin), num(-margin), num(-margin), num(l.Width+2*margin), num(l.Height+2*margin))
	body := drawing(v.Model, l, look)
	// The inner drawing without its own outer element: its contents, inside this one.
	body = body[strings.Index(body, ">")+1 : strings.LastIndex(body, "</svg>")]
	b.WriteString(body)
	b.WriteString("</svg>\n</svg>\n")
	return b.String()
}
