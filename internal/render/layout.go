package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/goccy/go-graphviz"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Layout computes where every element, boundary and arrow sits in one view (VIE-03).
//
// Graphviz does the placing — it is the reference implementation of layered graph layout, and
// goccy/go-graphviz runs it as WebAssembly, so the single static binary survives (no cgo, no
// system install). archdoc does the sizing: each box is exactly as large as the text archdoc will
// draw in it, so the engine lays out the real diagram rather than a guess at one.
//
// Deterministic: the same view produces byte-identical input, and Graphviz produces identical
// output for identical input. AC-7 depends on both halves.
func Layout(ctx context.Context, view archdoc.Model, group bool) (archdoc.Layout, error) {
	ids := identifiers(view)
	back := make(map[string]string, len(ids))
	for model, dot := range ids {
		back[dot] = model
	}

	g, err := graphviz.New(ctx)
	if err != nil {
		return archdoc.Layout{}, fmt.Errorf("starting the layout engine: %w", err)
	}
	defer g.Close()

	var fold [][2]string // pairs kept a rank apart, to break up a rank too wide to read
	lay := func(dir string) (archdoc.Layout, error) {
		src, clusters := toDOT(view, group, ids, dir, fold)

		graph, err := graphviz.ParseBytes([]byte(src))
		if err != nil {
			return archdoc.Layout{}, fmt.Errorf("layout input rejected: %w", err)
		}
		defer graph.Close()

		var out bytes.Buffer
		if err := g.Render(ctx, graph, graphviz.Format("json0"), &out); err != nil {
			return archdoc.Layout{}, fmt.Errorf("laying out: %w", err)
		}
		return parseLayout(out.Bytes(), back, clusters)
	}

	// Top to bottom first: it reads as C4 diagrams usually do, people above the system.
	tall, err := lay("TB")
	if err != nil {
		return archdoc.Layout{}, err
	}
	if aspect(tall) <= maxAspect {
		return tall, nil
	}

	// Too wide to read. Found on Supabase: ten containers all reached by one user sat in a
	// single row, 2232 points wide, and a markdown preview scaled the text to a third of its
	// size. Left to right stacks them instead. Keep whichever is nearer a readable shape;
	// both are deterministic, so the choice is too.
	wide, err := lay("LR")
	if err != nil {
		return tall, nil // the first layout is still correct, only wide
	}
	best := tall
	if distance(aspect(wide)) < distance(aspect(tall)) {
		best = wide
	}
	if aspect(best) <= maxAspect && aspect(best) >= 1/maxAspect {
		return best, nil
	}

	// Still a strip, either way round. Found on Immich's server once its components carried
	// descriptions: sixteen boxes with few arrows between them make two ranks, ten boxes wide. A
	// rank with more boxes than fit a readable width is folded: each box past the limit is put a
	// rank below the one that many places before it, so one row of ten becomes two of five.
	fold = folds(tall, ids)
	if len(fold) == 0 {
		return best, nil
	}
	folded, err := lay("TB")
	if err != nil || distance(aspect(folded)) >= distance(aspect(best)) {
		return best, nil
	}
	return folded, nil
}

// rankWidth is how many boxes sit side by side before a rank is folded.
const rankWidth = 5

// folds reads the ranks off a top-to-bottom layout — boxes at one height — and, for each rank with
// more than rankWidth boxes, pairs every box with the one rankWidth places to its right.
func folds(l archdoc.Layout, ids map[string]string) [][2]string {
	ranks := map[int][]archdoc.Box{}
	for _, b := range l.Boxes {
		y := int(b.Rect.Y+b.Rect.H/2) / 20 // boxes of different heights in one rank share a centre line
		ranks[y] = append(ranks[y], b)
	}
	keys := make([]int, 0, len(ranks))
	for y := range ranks {
		keys = append(keys, y)
	}
	sort.Ints(keys)
	var out [][2]string
	for _, y := range keys {
		row := ranks[y]
		if len(row) <= rankWidth {
			continue
		}
		sort.Slice(row, func(i, j int) bool {
			if row[i].Rect.X != row[j].Rect.X {
				return row[i].Rect.X < row[j].Rect.X
			}
			return row[i].ID < row[j].ID
		})
		for i := 0; i+rankWidth < len(row); i++ {
			out = append(out, [2]string{ids[row[i].ID], ids[row[i+rankWidth].ID]})
		}
	}
	return out
}

// maxAspect is how much wider than tall a diagram may be before the other orientation is tried.
// A markdown preview scales a picture to the column; past this the text stops being legible.
const maxAspect = 2.0

// idealAspect is the shape a diagram is steered towards: a little wider than tall, like a page.
const idealAspect = 1.4

func aspect(l archdoc.Layout) float64 {
	if l.Height <= 0 {
		return 0
	}
	return l.Width / l.Height
}

func distance(a float64) float64 {
	if a > idealAspect {
		return a / idealAspect
	}
	return idealAspect / a
}

// LayoutVersion changes whenever what Layout produces changes. Found the first day: a fix to
// where boundary labels sit did not show on Immich or Mastodon, because their unchanged
// architectures reused layouts stored by the old code. Bump this with any such change.
const LayoutVersion = 10

// Box and text geometry, in points. Shared by the layout and the drawing, so a box is sized for
// exactly the text that will be drawn in it.
const (
	boxWidth    = 190.0
	boxPad      = 10.0
	nameLine    = 16.0 // the bold name
	textLine    = 13.0 // type line and description lines
	descChars   = 29   // wrap width for descriptions, in characters — what the app fits in a box, too
	descMax     = 5    // lines; longer descriptions are cut with an ellipsis
	labelChars  = 26   // wrap width for relationship labels
	labelSize   = 9.0  // relationship label font size
	pointsPerIn = 72.0
)

// boxLines is the text drawn in an element's box, in order: name, type, description lines.
func boxLines(n archdoc.Node) (name, kind string, desc []string) {
	desc = wrap(n.Description, descChars, descMax)
	if n.UsedBy > 0 || n.UsesMany > 0 {
		desc = append(wrap(sharedLine(n), descChars, 2), desc...)
	}
	return n.Name, svgTypeLabel(n), desc
}

// sharedLine is what a shared component's box says in place of the arrows into it.
func sharedLine(n archdoc.Node) string {
	switch {
	case n.UsedBy > 0 && n.UsesMany > 0:
		return fmt.Sprintf("used by %d, uses %d of %d · not drawn", n.UsedBy, n.UsesMany, n.Among)
	case n.UsesMany > 0:
		return fmt.Sprintf("uses %d of %d · not drawn", n.UsesMany, n.Among)
	}
	return fmt.Sprintf("used by %d of %d · not drawn", n.UsedBy, n.Among)
}

// A table is drawn with its columns, one line each, up to tableRows; the rest are counted.
const (
	tableWidth = 236.0
	tableRows  = 14
)

func boxWidthOf(n archdoc.Node) float64 {
	if n.Kind == archdoc.Table {
		return tableWidth
	}
	return boxWidth
}

// tableLines is how many column lines a table's box holds: every column, or tableRows of them
// and a line saying how many more.
func tableLines(n archdoc.Node) int {
	if len(n.Columns) > tableRows+1 {
		return tableRows + 1
	}
	return len(n.Columns)
}

func boxHeight(n archdoc.Node) float64 {
	if n.Kind == archdoc.Table {
		return boxPad*2 + nameLine + 6 + float64(tableLines(n))*textLine
	}
	_, _, desc := boxLines(n)
	h := boxPad*2 + nameLine + textLine
	if len(desc) > 0 {
		h += 4 + float64(len(desc))*textLine
	}
	return h
}

// toDOT writes the Graphviz input. Order is fixed — nodes and edges in view order, which is
// already sorted — because the output must be byte-identical across runs.
func toDOT(view archdoc.Model, group bool, ids map[string]string, rankdir string, fold [][2]string) (string, map[string]archdoc.Boundary) {
	var b strings.Builder
	clusters := map[string]archdoc.Boundary{}

	b.WriteString("digraph archdoc {\n")
	fmt.Fprintf(&b, "  graph [rankdir=%s, nodesep=0.6, ranksep=0.9, fontsize=11, fontname=\"Helvetica\"];\n", rankdir)
	b.WriteString("  node [shape=box, fixedsize=true, fontname=\"Helvetica\"];\n")
	fmt.Fprintf(&b, "  edge [fontsize=%g, fontname=\"Helvetica\"];\n", labelSize)

	node := func(indent string, n archdoc.Node) {
		fmt.Fprintf(&b, "%s%s [label=\"\", width=%.3f, height=%.3f];\n",
			indent, ids[n.ID], boxWidthOf(n)/pointsPerIn, boxHeight(n)/pointsPerIn)
	}

	shared, wiring := map[string]bool{}, map[string]bool{}
	var sinks []string
	for _, n := range view.Nodes {
		if n.UsedBy > 0 {
			shared[n.ID] = true
		}
		if n.UsesMany > 0 {
			wiring[n.ID] = true
		}
		if n.UsedBy > 0 || n.UsesMany > 0 {
			sinks = append(sinks, ids[n.ID])
		}
	}

	inside, outside := partition(view.Nodes, group)
	for _, n := range outside {
		node("  ", n)
	}

	if len(inside) > 0 {
		// The engine is given the exact text archdoc will draw, so the space it reserves at the
		// top of the boundary fits it — a first version drew labels at the bottom-left instead,
		// where nested boundaries share an edge and their labels collided.
		label := view.Name + " [Software System]"
		if view.Boundary != "" {
			label = view.Boundary
		}
		clusters["cluster_system"] = archdoc.Boundary{Name: view.Name, System: true, Label: label}
		fmt.Fprintf(&b, "  subgraph cluster_system {\n    label=%s;\n", quote(label))
		if groups := boundaries(view, inside); groups != nil {
			for _, n := range ungrouped(inside) {
				node("    ", n)
			}
			for _, g := range groups {
				writeDOTGroup(&b, g, ids, node, "    ", clusters)
			}
		} else {
			for _, n := range inside {
				node("    ", n)
			}
		}
		if len(sinks) > 1 {
			// Shared components sit together on one rank: a band, not scattered among what uses them.
			fmt.Fprintf(&b, "    { rank=same; %s; }\n", strings.Join(sinks, "; "))
		}
		b.WriteString("  }\n")
	}

	for _, e := range view.Edges {
		from, to := ids[e.From], ids[e.To]
		if from == "" || to == "" || shared[e.To] || wiring[e.From] {
			continue // into a shared part, or out of one that uses most things: said on its box, not drawn
		}
		// The label is given to the engine so it reserves room and reports where the label
		// sits; archdoc draws the text itself.
		fmt.Fprintf(&b, "  %s -> %s [label=%s];\n", from, to, quote(strings.Join(wrap(edgeText(e), labelChars, 3), "\n")))
	}

	for _, f := range fold {
		fmt.Fprintf(&b, "  %s -> %s [style=invis];\n", f[0], f[1])
	}

	b.WriteString("}\n")
	return b.String(), clusters
}

func writeDOTGroup(b *strings.Builder, g *group, ids map[string]string, node func(string, archdoc.Node), indent string, clusters map[string]archdoc.Boundary) {
	if len(g.Nodes) == 0 && len(g.Children) == 0 {
		return
	}
	label := g.Name
	if g.Internal {
		label += " — no external connectivity"
	}
	id := "cluster_net_" + sanitise(g.Name)
	clusters[id] = archdoc.Boundary{Name: g.Name, Internal: g.Internal, Label: label}
	fmt.Fprintf(b, "%ssubgraph %s {\n%s  label=%s;\n", indent, id, indent, quote(label))
	for _, n := range g.Nodes {
		node(indent+"  ", n)
	}
	for _, c := range g.Children {
		writeDOTGroup(b, c, ids, node, indent+"  ", clusters)
	}
	fmt.Fprintf(b, "%s}\n", indent)
}

func edgeText(e archdoc.Edge) string {
	if e.Technology != "" && e.Label != "" {
		return e.Label + " [" + e.Technology + "]"
	}
	if e.Label != "" {
		return e.Label
	}
	return e.Technology
}

func quote(s string) string { return strconv.Quote(s) }

// The subset of Graphviz's json0 output archdoc reads.
type gvGraph struct {
	BB      string `json:"bb"`
	Objects []struct {
		GVID   int    `json:"_gvid"`
		Name   string `json:"name"`
		BB     string `json:"bb"`
		LP     string `json:"lp"`
		Pos    string `json:"pos"`
		Width  string `json:"width"`
		Height string `json:"height"`
		Label  string `json:"label"`
	} `json:"objects"`
	Edges []struct {
		Tail int    `json:"tail"`
		Head int    `json:"head"`
		Pos  string `json:"pos"`
		LP   string `json:"lp"`
	} `json:"edges"`
}

// parseLayout turns Graphviz's answer into archdoc's own layout, flipping the y-axis: Graphviz
// measures up from the bottom, SVG down from the top.
func parseLayout(raw []byte, back map[string]string, clusters map[string]archdoc.Boundary) (archdoc.Layout, error) {
	var g gvGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return archdoc.Layout{}, fmt.Errorf("reading the layout: %w", err)
	}

	bb, err := numbers(g.BB, 4)
	if err != nil {
		return archdoc.Layout{}, fmt.Errorf("graph bounds: %w", err)
	}
	height := bb[3]
	flip := func(x, y float64) archdoc.Point { return archdoc.Point{X: x, Y: height - y} }

	out := archdoc.Layout{Version: LayoutVersion, Width: bb[2], Height: height}
	byGVID := map[int]string{}

	for _, o := range g.Objects {
		switch {
		case strings.HasPrefix(o.Name, "cluster_"):
			r, err := numbers(o.BB, 4)
			if err != nil {
				return archdoc.Layout{}, fmt.Errorf("boundary %s: %w", o.Name, err)
			}
			grp, known := clusters[o.Name]
			if !known {
				grp = archdoc.Boundary{Name: o.Label, Label: o.Label, System: o.Name == "cluster_system"}
			}
			grp.Rect = archdoc.Rect{X: r[0], Y: height - r[3], W: r[2] - r[0], H: r[3] - r[1]}
			if lp, err := numbers(o.LP, 2); err == nil {
				grp.LabelAt = flip(lp[0], lp[1])
			}
			out.Groups = append(out.Groups, grp)

		default:
			id, ok := back[o.Name]
			if !ok {
				continue
			}
			byGVID[o.GVID] = id

			pos, err := numbers(o.Pos, 2)
			if err != nil {
				return archdoc.Layout{}, fmt.Errorf("position of %s: %w", id, err)
			}
			w, _ := strconv.ParseFloat(o.Width, 64)
			h, _ := strconv.ParseFloat(o.Height, 64)
			w, h = w*pointsPerIn, h*pointsPerIn
			c := flip(pos[0], pos[1])
			out.Boxes = append(out.Boxes, archdoc.Box{
				ID: id, Rect: archdoc.Rect{X: c.X - w/2, Y: c.Y - h/2, W: w, H: h},
			})
		}
	}

	for _, e := range g.Edges {
		p := archdoc.Path{From: byGVID[e.Tail], To: byGVID[e.Head]}
		if p.From == "" || p.To == "" {
			continue
		}
		for _, tok := range strings.Fields(e.Pos) {
			switch {
			case strings.HasPrefix(tok, "e,"):
				if v, err := numbers(tok[2:], 2); err == nil {
					tip := flip(v[0], v[1])
					p.Tip = &tip
				}
			case strings.HasPrefix(tok, "s,"):
				// A start marker; archdoc draws no tails.
			default:
				if v, err := numbers(tok, 2); err == nil {
					p.Curve = append(p.Curve, flip(v[0], v[1]))
				}
			}
		}
		if v, err := numbers(e.LP, 2); err == nil {
			lp := flip(v[0], v[1])
			p.LabelAt = &lp
		}
		out.Paths = append(out.Paths, p)
	}

	// Stable order regardless of how the engine enumerated things. Groups outermost first, so
	// drawing them in order paints nested boundaries on top of their parents.
	sort.SliceStable(out.Groups, func(i, j int) bool {
		return out.Groups[i].Rect.W*out.Groups[i].Rect.H > out.Groups[j].Rect.W*out.Groups[j].Rect.H
	})
	sort.SliceStable(out.Boxes, func(i, j int) bool { return out.Boxes[i].ID < out.Boxes[j].ID })
	sort.SliceStable(out.Paths, func(i, j int) bool {
		if out.Paths[i].From != out.Paths[j].From {
			return out.Paths[i].From < out.Paths[j].From
		}
		return out.Paths[i].To < out.Paths[j].To
	})

	return out, nil
}

// numbers parses "a,b,c,d" into floats, requiring exactly n of them.
func numbers(s string, n int) ([]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != n {
		return nil, fmt.Errorf("expected %d numbers in %q", n, s)
	}
	out := make([]float64, n)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// wrap breaks text into lines of at most width characters on word boundaries, keeping at most max
// lines. A cut is marked with an ellipsis rather than hidden.
func wrap(text string, width, max int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	line := ""
	for _, w := range words {
		switch {
		case line == "":
			line = w
		case len(line)+1+len(w) <= width:
			line += " " + w
		default:
			lines = append(lines, line)
			line = w
		}
	}
	lines = append(lines, line)

	if len(lines) > max {
		lines = lines[:max]
		last := lines[max-1]
		if len(last) > width-1 {
			last = last[:width-1]
		}
		lines[max-1] = strings.TrimRight(last, " ,.;") + "…"
	}
	return lines
}
