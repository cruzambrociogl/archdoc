// Package arrange holds the arrangements and saved views a person makes in the app, and applies
// an arrangement to a laid-out view (surface-spec §9 S-6, S-7).
//
// An arrangement is where a person dragged the boxes of a diagram. Position is presentation, not
// fact (vision §3.4): nothing here can add, remove, rename or reconnect an element, so an
// arrangement cannot falsify anything. It is kept in .archdoc/layout.yaml, committed, and applied
// on every run like rules.yaml — to the app, the committed SVG and the published site alike.
//
// The engine's layout stays what it computed and is stored unchanged; the arrangement is applied
// on top whenever a view is drawn, so resetting a view is just deleting its entry.
package arrange

import (
	"math"
	"sort"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Arrangement is, per view ("context", "container"), where a person placed each element: the
// top-left corner of its box, in the layout's own units.
type Arrangement map[string]map[string]core.Point

// Report says how an arrangement met the view it was applied to.
type Report struct {
	// Placed are the elements a person positioned, in this view.
	Placed []string `json:"placed"`
	// New are elements the view holds that the arrangement does not mention — added since it was
	// saved — so the engine placed them. Empty when the view has no arrangement at all.
	New []string `json:"new"`
	// Stale are saved positions that name no element of the view, as an unmatched rule names
	// nothing. They are reported, never silently dropped.
	Stale []string `json:"stale"`
}

// arrowLength is how far before the tip a re-routed arrow's line ends, so its head has a
// direction to point in — Graphviz leaves the same gap.
const arrowLength = 8.0

// Apply moves the boxes of a laid-out view to where a person placed them, regrows the boundaries
// around what they contain, and re-routes the relationships whose ends moved. A relationship whose
// two ends moved together keeps the engine's curve, translated; one whose ends moved apart is
// drawn straight, clipped to the two boxes.
func Apply(view string, l core.Layout, a Arrangement) (core.Layout, Report) {
	saved := a[view]
	rep := Report{Placed: []string{}, New: []string{}, Stale: []string{}}

	inView := map[string]bool{}
	for _, b := range l.Boxes {
		inView[b.ID] = true
	}
	for _, id := range sortedKeys(saved) {
		if !inView[id] {
			rep.Stale = append(rep.Stale, id)
		}
	}
	if len(saved) == 0 {
		return l, rep
	}

	out := l
	out.Boxes = make([]core.Box, len(l.Boxes))
	before := map[string]core.Rect{}
	after := map[string]core.Rect{}
	for i, b := range l.Boxes {
		before[b.ID] = b.Rect
		r := b.Rect
		if p, ok := saved[b.ID]; ok {
			r.X, r.Y = math.Max(0, p.X), math.Max(0, p.Y)
			rep.Placed = append(rep.Placed, b.ID)
		} else {
			rep.New = append(rep.New, b.ID)
		}
		after[b.ID] = r
		out.Boxes[i] = core.Box{ID: b.ID, Rect: r}
	}
	sort.Strings(rep.Placed)
	sort.Strings(rep.New)

	// A boundary contains the boxes that lay inside it as the engine drew it, and keeps the same
	// margin around them wherever they now are.
	out.Groups = make([]core.Boundary, len(l.Groups))
	for i, g := range l.Groups {
		var members []string
		for _, b := range l.Boxes {
			if inside(b.Rect, g.Rect) {
				members = append(members, b.ID)
			}
		}
		out.Groups[i] = g
		if len(members) == 0 || !anyMoved(members, before, after) {
			continue
		}
		was, now := bounds(members, before), bounds(members, after)
		r := core.Rect{
			X: now.X - (was.X - g.Rect.X),
			Y: now.Y - (was.Y - g.Rect.Y),
			W: now.W + (g.Rect.W - was.W),
			H: now.H + (g.Rect.H - was.H),
		}
		out.Groups[i].Rect = r
		out.Groups[i].LabelAt = core.Point{X: g.LabelAt.X + r.X - g.Rect.X, Y: g.LabelAt.Y + r.Y - g.Rect.Y}
	}

	out.Paths = make([]core.Path, len(l.Paths))
	for i, p := range l.Paths {
		df, dt := delta(p.From, before, after), delta(p.To, before, after)
		switch {
		case df == (core.Point{}) && dt == (core.Point{}):
			out.Paths[i] = p
		case df == dt:
			out.Paths[i] = translate(p, df)
		default:
			out.Paths[i] = straight(p, after[p.From], after[p.To])
		}
	}

	// The drawing grows to hold whatever moved outwards.
	for _, b := range out.Boxes {
		out.Width = math.Max(out.Width, b.Rect.X+b.Rect.W)
		out.Height = math.Max(out.Height, b.Rect.Y+b.Rect.H)
	}
	for _, g := range out.Groups {
		out.Width = math.Max(out.Width, g.Rect.X+g.Rect.W)
		out.Height = math.Max(out.Height, g.Rect.Y+g.Rect.H)
	}
	return out, rep
}

func inside(r, outer core.Rect) bool {
	return r.X >= outer.X && r.Y >= outer.Y && r.X+r.W <= outer.X+outer.W && r.Y+r.H <= outer.Y+outer.H
}

func anyMoved(ids []string, before, after map[string]core.Rect) bool {
	for _, id := range ids {
		if before[id] != after[id] {
			return true
		}
	}
	return false
}

func bounds(ids []string, rects map[string]core.Rect) core.Rect {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, id := range ids {
		r := rects[id]
		minX, minY = math.Min(minX, r.X), math.Min(minY, r.Y)
		maxX, maxY = math.Max(maxX, r.X+r.W), math.Max(maxY, r.Y+r.H)
	}
	return core.Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func delta(id string, before, after map[string]core.Rect) core.Point {
	b, a := before[id], after[id]
	return core.Point{X: a.X - b.X, Y: a.Y - b.Y}
}

func translate(p core.Path, d core.Point) core.Path {
	q := p
	q.Curve = make([]core.Point, len(p.Curve))
	for i, c := range p.Curve {
		q.Curve[i] = core.Point{X: c.X + d.X, Y: c.Y + d.Y}
	}
	if p.Tip != nil {
		t := core.Point{X: p.Tip.X + d.X, Y: p.Tip.Y + d.Y}
		q.Tip = &t
	}
	if p.LabelAt != nil {
		at := core.Point{X: p.LabelAt.X + d.X, Y: p.LabelAt.Y + d.Y}
		q.LabelAt = &at
	}
	return q
}

// straight routes a relationship between the centres of its two boxes, from the edge of one to
// the edge of the other, as a single Bézier segment so every renderer reads it like the engine's.
func straight(p core.Path, from, to core.Rect) core.Path {
	c1 := core.Point{X: from.X + from.W/2, Y: from.Y + from.H/2}
	c2 := core.Point{X: to.X + to.W/2, Y: to.Y + to.H/2}
	start := clip(c1, c2, from)
	tip := clip(c2, c1, to)
	dx, dy := tip.X-start.X, tip.Y-start.Y
	length := math.Hypot(dx, dy)
	end := tip
	if length > arrowLength {
		end = core.Point{X: tip.X - dx/length*arrowLength, Y: tip.Y - dy/length*arrowLength}
	}
	third := func(f float64) core.Point {
		return core.Point{X: start.X + (end.X-start.X)*f, Y: start.Y + (end.Y-start.Y)*f}
	}
	q := core.Path{From: p.From, To: p.To, Curve: []core.Point{start, third(1.0 / 3), third(2.0 / 3), end}}
	if p.Tip != nil {
		t := tip
		q.Tip = &t
	}
	if p.LabelAt != nil {
		mid := core.Point{X: (start.X + tip.X) / 2, Y: (start.Y + tip.Y) / 2}
		q.LabelAt = &mid
	}
	return q
}

// clip is where the line from c towards toward leaves the rectangle r centred on c.
func clip(c, toward core.Point, r core.Rect) core.Point {
	dx, dy := toward.X-c.X, toward.Y-c.Y
	if dx == 0 && dy == 0 {
		return c
	}
	sx, sy := math.Inf(1), math.Inf(1)
	if dx != 0 {
		sx = (r.W / 2) / math.Abs(dx)
	}
	if dy != 0 {
		sy = (r.H / 2) / math.Abs(dy)
	}
	s := math.Min(sx, sy)
	return core.Point{X: c.X + dx*s, Y: c.Y + dy*s}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
