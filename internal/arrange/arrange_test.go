package arrange

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// A small laid-out view: a boundary around a and b, c outside it, a→b and a→c.
func layout() archdoc.Layout {
	tip := func(x, y float64) *archdoc.Point { return &archdoc.Point{X: x, Y: y} }
	return archdoc.Layout{
		Width: 400, Height: 300,
		Boxes: []archdoc.Box{
			{ID: "a", Rect: archdoc.Rect{X: 20, Y: 20, W: 100, H: 50}},
			{ID: "b", Rect: archdoc.Rect{X: 20, Y: 120, W: 100, H: 50}},
			{ID: "c", Rect: archdoc.Rect{X: 250, Y: 120, W: 100, H: 50}},
		},
		Groups: []archdoc.Boundary{{Name: "sys", Rect: archdoc.Rect{X: 10, Y: 10, W: 120, H: 170}, LabelAt: archdoc.Point{X: 14, Y: 14}}},
		Paths: []archdoc.Path{
			{From: "a", To: "b", Curve: []archdoc.Point{{X: 70, Y: 70}, {X: 70, Y: 90}, {X: 70, Y: 100}, {X: 70, Y: 110}}, Tip: tip(70, 120), LabelAt: tip(80, 95)},
			{From: "a", To: "c", Curve: []archdoc.Point{{X: 120, Y: 45}, {X: 180, Y: 60}, {X: 250, Y: 100}, {X: 290, Y: 110}}, Tip: tip(300, 120)},
		},
	}
}

func TestNoArrangementLeavesTheLayoutAlone(t *testing.T) {
	l := layout()
	got, rep := Apply("container", l, Arrangement{})
	if !reflect.DeepEqual(got, l) {
		t.Error("an empty arrangement changed the layout")
	}
	if len(rep.Placed)+len(rep.New)+len(rep.Stale) != 0 {
		t.Errorf("report for no arrangement: %+v", rep)
	}
}

// Moving c alone re-routes only the relationship that touches it, straight and clipped to the
// boxes, and leaves the boundary it was never inside untouched.
func TestMovingOneBoxReroutesOnlyItsRelationships(t *testing.T) {
	got, rep := Apply("container", layout(), Arrangement{"container": {"c": {X: 250, Y: 220}}})

	if got.Boxes[2].Rect.Y != 220 {
		t.Errorf("c not moved: %+v", got.Boxes[2].Rect)
	}
	if !reflect.DeepEqual(got.Paths[0], layout().Paths[0]) {
		t.Error("a→b changed though neither end moved")
	}
	ac := got.Paths[1]
	if len(ac.Curve) != 4 || ac.Tip == nil {
		t.Fatalf("a→c not re-routed as one segment: %+v", ac)
	}
	// The tip touches c's edge; the line ends short of it so the head has a direction.
	c := got.Boxes[2].Rect
	if ac.Tip.Y < c.Y-0.01 || ac.Tip.Y > c.Y+c.H+0.01 || ac.Tip.X < c.X-0.01 || ac.Tip.X > c.X+c.W+0.01 {
		t.Errorf("tip %+v is not on c %+v", *ac.Tip, c)
	}
	if ac.Curve[3] == *ac.Tip {
		t.Error("the line ends exactly at the tip; the arrowhead has no direction")
	}
	if got.Groups[0] != layout().Groups[0] {
		t.Error("a boundary c was never inside moved")
	}
	if !reflect.DeepEqual(rep.Placed, []string{"c"}) || !reflect.DeepEqual(rep.New, []string{"a", "b"}) {
		t.Errorf("report: %+v", rep)
	}
}

// Moving a and b together keeps the engine's curve between them, translated, and the boundary
// around them follows with the same margin.
func TestMovingTogetherKeepsTheCurveAndTheBoundary(t *testing.T) {
	got, _ := Apply("container", layout(), Arrangement{"container": {"a": {X: 120, Y: 20}, "b": {X: 120, Y: 120}}})

	ab := got.Paths[0]
	if ab.Curve[0].X != 170 || ab.Tip.X != 170 || ab.LabelAt.X != 180 {
		t.Errorf("a→b not translated by 100: %+v", ab)
	}
	g := got.Groups[0]
	if g.Rect.X != 110 || g.Rect.W != 120 || g.LabelAt.X != 114 {
		t.Errorf("boundary did not follow its members with the same margin: %+v", g)
	}
}

// A saved position that names nothing in the view is reported, as an unmatched rule is.
func TestStalePositionsAreReported(t *testing.T) {
	_, rep := Apply("container", layout(), Arrangement{"container": {"a": {X: 20, Y: 20}, "gone": {X: 1, Y: 1}}})
	if !reflect.DeepEqual(rep.Stale, []string{"gone"}) {
		t.Errorf("stale: %v", rep.Stale)
	}
}

// Two saves of the same arrangement are byte-identical, whatever order it was built in (AC-7's
// spirit), and ids with colons survive the round trip.
func TestLayoutFileIsDeterministicAndRoundTrips(t *testing.T) {
	a := Arrangement{"container": {"svc:web": {X: 850.4, Y: 258}, "svc:db": {X: 10, Y: 20}}, "context": {"system": {X: 5, Y: 6}}}
	b := Arrangement{"context": {"system": {X: 5, Y: 6}}, "container": {"svc:db": {X: 10, Y: 20}, "svc:web": {X: 850.4, Y: 258}}}
	if string(FormatLayout(a)) != string(FormatLayout(b)) {
		t.Fatal("the same arrangement formats two ways")
	}

	root := t.TempDir()
	hash, err := SaveLayout(root, a, "")
	if err != nil {
		t.Fatal(err)
	}
	back, h2, err := LoadLayout(root)
	if err != nil || h2 != hash {
		t.Fatalf("load: %v %s %s", err, h2, hash)
	}
	if back["container"]["svc:web"] != (archdoc.Point{X: 850, Y: 258}) {
		t.Errorf("round trip: %+v", back)
	}
}

// ANS-04: a file edited after the app loaded it is never overwritten.
func TestAChangedFileIsNotOverwritten(t *testing.T) {
	root := t.TempDir()
	if _, err := SaveLayout(root, Arrangement{"container": {"a": {X: 1, Y: 1}}}, ""); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, Dir, LayoutFile)
	edited := []byte("container:\n  a: { x: 99, y: 99 }\n")
	os.WriteFile(p, edited, 0o644)

	_, err := SaveLayout(root, Arrangement{"container": {"a": {X: 2, Y: 2}}}, Hash([]byte("whatever the app loaded")))
	if !errors.Is(err, ErrChanged) {
		t.Fatalf("want ErrChanged, got %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != string(edited) {
		t.Error("the edited file was overwritten")
	}
}

func TestViewsRoundTrip(t *testing.T) {
	root := t.TempDir()
	vs := []View{{Name: "Upload path", Level: "container", Focus: "svc:web", Dim: true}}
	hash, err := SaveViews(root, vs, "")
	if err != nil {
		t.Fatal(err)
	}
	back, h2, err := LoadViews(root)
	if err != nil || h2 != hash || !reflect.DeepEqual(back, vs) {
		t.Errorf("views round trip: %+v %v", back, err)
	}
	if _, err := SaveViews(root, vs, ""); !errors.Is(err, ErrChanged) {
		t.Errorf("a stale save was accepted: %v", err)
	}
}
