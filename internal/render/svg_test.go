package render

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func laid(t *testing.T, view archdoc.Model, group bool) archdoc.Layout {
	t.Helper()
	l, err := Layout(context.Background(), view, group)
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	return l
}

// Every element in the view gets a box, inside the drawing, with a real size.
func TestLayoutPlacesEveryElement(t *testing.T) {
	view := fixture().Container()
	l := laid(t, view, true)

	if len(l.Boxes) != len(view.Nodes) {
		t.Fatalf("got %d boxes for %d elements", len(l.Boxes), len(view.Nodes))
	}
	for _, b := range l.Boxes {
		r := b.Rect
		if r.W <= 0 || r.H <= 0 {
			t.Errorf("%s has no size: %+v", b.ID, r)
		}
		if r.X < 0 || r.Y < 0 || r.X+r.W > l.Width+0.5 || r.Y+r.H > l.Height+0.5 {
			t.Errorf("%s lies outside the drawing: %+v in %gx%g", b.ID, r, l.Width, l.Height)
		}
	}
	if len(l.Paths) != len(view.Edges) {
		t.Errorf("got %d paths for %d relationships", len(l.Paths), len(view.Edges))
	}
}

// AC-7 reaches the picture: the same view must produce the same coordinates, or the committed SVG
// changes between runs that changed nothing.
func TestLayoutIsDeterministic(t *testing.T) {
	first := laid(t, fixture().Container(), true)
	for range 3 {
		if got := laid(t, fixture().Container(), true); !reflect.DeepEqual(got, first) {
			t.Fatal("layout differs between runs")
		}
	}
	if SVG(fixture().Container(), first) != SVG(fixture().Container(), first) {
		t.Fatal("drawing the same layout twice produced different bytes")
	}
}

// The container view draws the system boundary; the context view, which is already one box,
// does not.
func TestLayoutDrawsTheSystemBoundary(t *testing.T) {
	system := func(l archdoc.Layout) bool {
		for _, g := range l.Groups {
			if g.System {
				return true
			}
		}
		return false
	}
	if !system(laid(t, fixture().Container(), true)) {
		t.Error("the container view has no system boundary")
	}
	if system(laid(t, fixture().Context(), false)) {
		t.Error("the context view drew a boundary around a single box")
	}
}

// Networks nest in the picture as they do in the Mermaid view: frontend inside backend, both
// inside the system.
func TestNetworksNestInThePicture(t *testing.T) {
	l := laid(t, netFixture().Container(), true)

	rects := map[string]archdoc.Rect{}
	for _, g := range l.Groups {
		rects[g.Name] = g.Rect
	}
	inside := func(a, b archdoc.Rect) bool {
		return a.X >= b.X && a.Y >= b.Y && a.X+a.W <= b.X+b.W && a.Y+a.H <= b.Y+b.H
	}
	back, okB := rects["backend"]
	front, okF := rects["frontend"]
	if !okB || !okF {
		t.Fatalf("network boundaries missing: %v", l.Groups)
	}
	if !inside(front, back) {
		t.Error("frontend is not drawn inside backend")
	}
}

// Names come from other people's configuration. They must be escaped, or one ampersand breaks the
// picture for everyone.
func TestSVGEscapesText(t *testing.T) {
	m := fixture()
	for i := range m.Nodes {
		if m.Nodes[i].ID == "svc:api" {
			m.Nodes[i].Name = `api <"prod"> & co`
		}
	}
	view := m.Container()
	out := SVG(view, laid(t, view, true))

	if strings.Contains(out, `<"prod">`) {
		t.Error("an element name reached the SVG unescaped")
	}
	if !strings.Contains(out, "api &lt;&#34;prod&#34;&gt; &amp; co") {
		t.Error("the escaped name is missing")
	}
}

// PRV-05 on the diagram: text the model wrote is italic, text read from configuration is not.
func TestSVGMarksModelTextAsInterpretation(t *testing.T) {
	m := fixture()
	for i := range m.Nodes {
		if m.Nodes[i].ID == "svc:api" {
			m.Nodes[i].Description = "Serves the public API"
			m.Nodes[i].DescProv = archdoc.Provenance{Origin: archdoc.Semantic, Note: "claude-opus-5"}
		}
	}
	view := m.Container()
	out := SVG(view, laid(t, view, true))

	line := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Serves the public API") {
			line = l
		}
	}
	if !strings.Contains(line, `font-style="italic"`) {
		t.Errorf("a model-written description is not marked: %s", line)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, ">db<") && strings.Contains(l, "italic") {
			t.Error("a name read from configuration was set in italics")
		}
	}
}

// A long description is cut with an ellipsis rather than overflowing its box or vanishing.
func TestLongDescriptionsAreCutVisibly(t *testing.T) {
	lines := wrap(strings.Repeat("word ", 80), descChars, descMax)
	if len(lines) != descMax {
		t.Fatalf("got %d lines, want %d", len(lines), descMax)
	}
	if !strings.HasSuffix(lines[len(lines)-1], "…") {
		t.Errorf("the cut is not marked: %q", lines[len(lines)-1])
	}
	for _, l := range lines {
		if len([]rune(l)) > descChars {
			t.Errorf("line longer than the box allows: %q", l)
		}
	}
}

// With pictures, the documents show the SVG and keep the Mermaid source; without them, Mermaid
// alone, exactly as before.
func TestDocumentsEmbedThePictureWhenThereIsOne(t *testing.T) {
	with := Index(fixture(), Meta{Source: "docker-compose.yml", Pictures: true})
	if !strings.Contains(with, "![Containers](container.svg)") {
		t.Error("the index does not show the drawn diagram")
	}
	if !strings.Contains(with, "```mermaid") {
		t.Error("the Mermaid source was dropped when the picture was added")
	}

	without := Index(fixture(), Meta{Source: "docker-compose.yml"})
	if strings.Contains(without, ".svg") {
		t.Error("the index links a picture that is not being written")
	}
}

// Found on Supabase: ten containers all reached by one user laid out in a single row, 2232 points
// wide and a third as tall, unreadable once a markdown preview scaled it to the column. A wide
// fan-out must come back in a readable shape.
func TestWideFanOutStaysReadable(t *testing.T) {
	at := func(line int) archdoc.Provenance { return archdoc.Provenance{File: "compose.yml", Line: line} }
	m := archdoc.Model{Name: "wide"}
	m.Nodes = append(m.Nodes, archdoc.Node{ID: "actor:user", Name: "User", Kind: archdoc.Actor, Evidence: archdoc.Declared, Prov: at(1)})
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("svc:s%02d", i)
		m.Nodes = append(m.Nodes, archdoc.Node{ID: id, Name: id[4:], Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at(10 + i)})
		m.Edges = append(m.Edges, archdoc.Edge{From: "actor:user", To: id, Label: "reaches", Traffic: true, Prov: []archdoc.Provenance{at(2)}})
	}
	view := m.Normalise().Container()

	l := laid(t, view, true)
	if a := l.Width / l.Height; a > 2.5 {
		t.Errorf("a ten-way fan-out is %.1f times wider than tall (%gx%g)", a, l.Width, l.Height)
	}
}
