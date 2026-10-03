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
	view := m.Container()
	deployment := len(facts.Networks) > 0 || len(facts.Routes) > 0 || published(facts)
	related := len(view.Edges) > 0
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
