package render

import "github.com/cruzambrociogl/archdoc/internal/archdoc"

// A document set proportionate to the repository it describes.
//
// §8 holds that partial output is the honest output. The same reasoning applies to *size*: twelve
// arc42 sections for a single-service repository is a filing cabinet for a postcard, and seven
// stubs asking about trust boundaries and deployment targets that do not exist teaches the reader
// to ignore stubs. So a section is emitted when the repository has something to put in it.
//
// Two rules, both conservative:
//
//   - A **generated** section is emitted when its evidence exists. No networks and no published
//     ports means there is no deployment story to tell; no relationships means the runtime
//     section's "here is the skeleton a scenario would move along" has no skeleton.
//   - **Human** stubs are reduced to the three that any project can answer — what it is for, why
//     it is shaped this way, and what was decided — when the repository is small enough that the
//     other four would ask about structure it does not have.
//
// Nothing is ever deleted: a section emitted by an earlier run stays, because it may hold someone's
// writing, and the write set never removes human files. Growth is one-way, which is the safe
// direction.

// PlanFor returns the sections to emit for this model, in arc42 order.
func PlanFor(m archdoc.Model, facts archdoc.FactSet) []Section {
	if Tiny(m, facts) {
		return nil // one page says all of it: see onePage
	}
	view := m.Container()
	deployment := len(facts.Networks) > 0 || len(facts.Routes) > 0 || published(facts)
	// A person reaching an application is not a relationship between parts of the system: every
	// web front end has that arrow, and it says nothing about how much there is to document.
	related := false
	for _, e := range view.Edges {
		if from, ok := view.Node(e.From); ok && from.Kind != archdoc.Actor {
			related = true
		}
	}
	small := countContainers(view) <= 2 && !deployment && !related

	var out []Section
	for _, s := range Sections() {
		keep := true
		switch s.Number {
		case 6:
			keep = related || !small
		case 7:
			keep = deployment
		case 2, 8, 10, 11:
			keep = !small
		}
		if keep {
			out = append(out, s)
		}
	}
	return out
}

// tinyFiles is how many source files a project may have and still be said on one page.
const tinyFiles = 30

// Tiny reports a project small enough that twelve arc42 chapters would be absurd (vision D-13): a
// script, a page, a first version of an app. One application, nothing deployed beside it — no
// store, no second container, no Compose file — and a few files of code. It gets one page; the
// chapters appear when it grows into them.
func Tiny(m archdoc.Model, facts archdoc.FactSet) bool {
	if facts.Source != "" || len(facts.Services) > 0 {
		return false
	}
	containers := 0
	for _, n := range m.Nodes {
		if n.Kind.Container() {
			containers++
		}
	}
	files := 0
	for _, s := range facts.Sources {
		files += len(s.Files)
	}
	return containers == 1 && files > 0 && files <= tinyFiles
}

// Planned reports whether a plan contains a section number.
func Planned(plan []Section, number int) bool {
	for _, s := range plan {
		if s.Number == number {
			return true
		}
	}
	return false
}

func published(facts archdoc.FactSet) bool {
	for _, s := range facts.Services {
		if len(s.Ports) > 0 {
			return true
		}
	}
	return false
}
