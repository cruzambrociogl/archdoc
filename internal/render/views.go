package render

import (
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// View is one diagram archdoc draws. Its Name keys its stored layout, its arrangement and its
// address in the app; File is the stem of its committed .svg and .mmd.
type View struct {
	Name  string // "context", "container", "component:svc:immich-server"
	File  string // "context", "container", "component-immich-server"
	Title string
	Model archdoc.Model
	// Group draws a boundary around what is inside: the system in a container view, the container
	// in a component view.
	Group bool
}

// ComponentPrefix starts the name of every component view; the container's ID follows.
const ComponentPrefix = "component:"

// Views lists every diagram of the model, in a fixed order: context, container, then one
// component view per container whose code was read, in model order.
func Views(m archdoc.Model) []View {
	out := []View{
		{Name: "context", File: "context", Title: "System context", Model: m.Context()},
		{Name: "container", File: "container", Title: "Containers", Model: m.Container(), Group: true},
	}
	for _, id := range m.Components() {
		out = append(out, componentView(m, id))
	}
	return out
}

// ViewOf returns the named view, if the model has it.
func ViewOf(m archdoc.Model, name string) (View, bool) {
	switch {
	case name == "context":
		return View{Name: name, File: name, Title: "System context", Model: m.Context()}, true
	case name == "container":
		return View{Name: name, File: name, Title: "Containers", Model: m.Container(), Group: true}, true
	case strings.HasPrefix(name, ComponentPrefix):
		id := strings.TrimPrefix(name, ComponentPrefix)
		for _, c := range m.Components() {
			if c == id {
				return componentView(m, id), true
			}
		}
	}
	return View{}, false
}

func componentView(m archdoc.Model, id string) View {
	view := m.Component(id)
	// Every component edge says "uses"; drawn on each of a hundred arrows it says nothing, and
	// the weight — how many imports — is drawn as the arrow's thickness instead.
	edges := make([]archdoc.Edge, len(view.Edges))
	for i, e := range view.Edges {
		e.Label, e.LabelProv = "", archdoc.Provenance{}
		edges[i] = e
	}
	view.Edges = edges
	_, local, _ := strings.Cut(id, ":")
	return View{
		Name:  ComponentPrefix + id,
		File:  "component-" + strings.ReplaceAll(local, "/", "-"),
		Title: "Components of " + view.Name,
		Model: view,
		Group: true,
	}
}

// ComponentFile reports whether name is a file archdoc writes for a component view.
func ComponentFile(name string) bool {
	return strings.HasPrefix(name, "component-") && (strings.HasSuffix(name, ".svg") || strings.HasSuffix(name, ".mmd"))
}
