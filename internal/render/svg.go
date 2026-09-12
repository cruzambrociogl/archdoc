package render

import (
	"fmt"
	"html"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// SVG draws a view from its stored layout (VIE-10). archdoc draws the picture itself rather than
// taking Graphviz's own SVG, for two reasons: the C4 conventions — a box's name, type, technology
// and responsibility, external systems dashed — are archdoc's to apply, and a future web canvas
// must be able to draw the identical picture from the same stored coordinates.
//
// PRV-05 on the diagram itself: text a model wrote is set in italics, the same rule the evidence
// tables follow. A reader can tell interpretation from reading without leaving the picture.
func SVG(view archdoc.Model, l archdoc.Layout) string {
	const margin = 16.0

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s" width="%s" height="%s" font-family="Helvetica, Arial, sans-serif">`+"\n",
		num(-margin), num(-margin), num(l.Width+2*margin), num(l.Height+2*margin),
		num(l.Width+2*margin), num(l.Height+2*margin))

	b.WriteString(`<defs><marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="#707070"/></marker></defs>` + "\n")
	fmt.Fprintf(&b, `<rect x="%s" y="%s" width="%s" height="%s" fill="#ffffff"/>`+"\n",
		num(-margin), num(-margin), num(l.Width+2*margin), num(l.Height+2*margin))

	// Boundaries first, outermost first, so everything else sits on top of them.
	for _, g := range l.Groups {
		dash, stroke := "6 4", "#8a8a8a"
		if g.System {
			dash, stroke = "8 4", "#4a4a4a"
		}
		label := g.Label
		if label == "" {
			label = g.Name // a layout stored before labels were recorded
		}
		fmt.Fprintf(&b, `<rect x="%s" y="%s" width="%s" height="%s" rx="6" fill="none" stroke="%s" stroke-dasharray="%s"/>`+"\n",
			num(g.Rect.X), num(g.Rect.Y), num(g.Rect.W), num(g.Rect.H), stroke, dash)

		// Where the engine reserved room for the label: centred along the top edge, clear of
		// every nested boundary's own label.
		at := g.LabelAt
		if at == (archdoc.Point{}) {
			at = archdoc.Point{X: g.Rect.X + g.Rect.W/2, Y: g.Rect.Y + 12}
		}
		fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="11" fill="%s" text-anchor="middle" dominant-baseline="middle">%s</text>`+"\n",
			num(at.X), num(at.Y), stroke, esc(label))
	}

	// Arrows next, so boxes sit over the ends of their curves.
	for _, p := range l.Paths {
		if len(p.Curve) < 4 {
			continue
		}
		var d strings.Builder
		fmt.Fprintf(&d, "M%s,%s", num(p.Curve[0].X), num(p.Curve[0].Y))
		for i := 1; i+2 < len(p.Curve); i += 3 {
			fmt.Fprintf(&d, " C%s,%s %s,%s %s,%s",
				num(p.Curve[i].X), num(p.Curve[i].Y), num(p.Curve[i+1].X), num(p.Curve[i+1].Y),
				num(p.Curve[i+2].X), num(p.Curve[i+2].Y))
		}
		if p.Tip != nil {
			fmt.Fprintf(&d, " L%s,%s", num(p.Tip.X), num(p.Tip.Y))
		}
		fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#707070" stroke-width="1.2" marker-end="url(#arrow)"/>`+"\n", d.String())

		if p.LabelAt == nil {
			continue
		}
		e, ok := edgeOf(view, p.From, p.To)
		if !ok {
			continue
		}
		lines := wrap(edgeText(e), labelChars, 3)
		style := ""
		if e.LabelProv.Origin.Interpretation() {
			style = ` font-style="italic"`
		}
		top := p.LabelAt.Y - float64(len(lines)-1)*(labelSize+2)/2
		for i, line := range lines {
			fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="%g" fill="#505050" text-anchor="middle" dominant-baseline="middle"%s>%s</text>`+"\n",
				num(p.LabelAt.X), num(top+float64(i)*(labelSize+2)), labelSize, style, esc(line))
		}
	}

	for _, bx := range l.Boxes {
		n, ok := view.Node(bx.ID)
		if !ok {
			continue
		}
		b.WriteString(box(n, bx.Rect))
	}

	b.WriteString("</svg>\n")
	return b.String()
}

// box draws one element in C4 style.
func box(n archdoc.Node, r archdoc.Rect) string {
	fill, stroke, text, dash := "#1168bd", "#0b4884", "#ffffff", ""
	switch {
	case n.Kind == archdoc.Actor:
		fill, stroke = "#08427b", "#052e56"
	case n.Evidence == archdoc.Referenced || n.Kind == archdoc.External:
		// Referenced, not declared: hollow and dashed, the same convention as the Mermaid view.
		fill, stroke, text, dash = "#ffffff", "#8b8b8b", "#3d3d3d", ` stroke-dasharray="5 3"`
	case n.Kind == archdoc.Datastore:
		fill, stroke = "#2574b8", "#0b4884"
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<g id="%s">`+"\n", esc(n.ID))
	fmt.Fprintf(&b, `<rect x="%s" y="%s" width="%s" height="%s" rx="8" fill="%s" stroke="%s" stroke-width="1.2"%s/>`+"\n",
		num(r.X), num(r.Y), num(r.W), num(r.H), fill, stroke, dash)

	name, kind, desc := boxLines(n)
	cx := r.X + r.W/2
	y := r.Y + boxPad + nameLine - 3

	fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="13" font-weight="bold" fill="%s" text-anchor="middle"%s>%s</text>`+"\n",
		num(cx), num(y), text, italic(n.NameProv), esc(name))
	y += textLine
	fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="10" fill="%s" text-anchor="middle" opacity="0.85"%s>%s</text>`+"\n",
		num(cx), num(y), text, italic(n.TechProv), esc(kind))

	if len(desc) > 0 {
		y += 4
		for _, line := range desc {
			y += textLine
			fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="10" fill="%s" text-anchor="middle"%s>%s</text>`+"\n",
				num(cx), num(y-2), text, italic(n.DescProv), esc(line))
		}
	}

	b.WriteString("</g>\n")
	return b.String()
}

// svgTypeLabel is typeLabel without the Mermaid escaping, for text the SVG escapes itself.
func svgTypeLabel(n archdoc.Node) string {
	kind := "Container"
	switch n.Kind {
	case archdoc.Actor:
		kind = "Person"
	case archdoc.External:
		kind = "External System"
	case archdoc.System:
		kind = "Software System"
	}
	if n.Technology == "" {
		return "[" + kind + "]"
	}
	return "[" + kind + ": " + n.Technology + "]"
}

func italic(p archdoc.Provenance) string {
	if p.Origin.Interpretation() {
		return ` font-style="italic"`
	}
	return ""
}

func edgeOf(m archdoc.Model, from, to string) (archdoc.Edge, bool) {
	for _, e := range m.Edges {
		if e.From == from && e.To == to {
			return e, true
		}
	}
	return archdoc.Edge{}, false
}

func esc(s string) string { return html.EscapeString(s) }

// num formats a coordinate with two decimals: enough precision for a picture, and a fixed format
// so identical layouts produce identical bytes.
func num(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}
