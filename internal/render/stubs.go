package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Stubs are the seven sections archdoc cannot fill, written once and never touched again.
//
// OUT-08: the questions are derived from *this* model rather than copied from a template. A
// generic "describe your quality requirements" gets deleted unread. "Two containers are
// reachable from outside — what is the availability target for each?" gets answered, because it
// is about the reader's own system and archdoc already did the looking-up.
//
// This is the one place the tool asks rather than tells, and the asking is only useful if it is
// specific.

// Stubs returns the human-owned sections, keyed on file name. The caller writes each one **only
// if it does not already exist** — that is the regeneration boundary, and it is the reason these
// are returned separately from Arc42's output rather than alongside it.
func Stubs(m archdoc.Model, meta Meta) map[string]string {
	view := m.Container()

	out := map[string]string{}
	for _, s := range Sections() {
		if s.Owner != Human {
			continue
		}

		var b strings.Builder
		fmt.Fprintf(&b, "# %d. %s\n\n", s.Number, s.Title)
		b.WriteString("<!-- This file is yours. archdoc created it once and will never read or\n")
		b.WriteString("     overwrite it. Delete these prompts as you answer them. -->\n\n")
		b.WriteString(questions(s.Number, m, view))

		out[s.File()] = b.String()
	}

	return out
}

func questions(section int, m archdoc.Model, view archdoc.Model) string {
	var b strings.Builder

	switch section {
	case 1: // Introduction and Goals
		b.WriteString("archdoc found what this system is *made of*. It cannot find what it is *for*.\n\n")
		fmt.Fprintf(&b, "- What does **%s** do, in two sentences, for someone who has never seen it?\n", m.Name)
		if actors := kindOf(view, archdoc.Actor); len(actors) > 0 {
			b.WriteString("- Configuration shows something outside reaching in, but not who. Who are the\n")
			b.WriteString("  users, and what are they trying to do?\n")
		}
		b.WriteString("- What are the top three quality goals? Section 10 is where they get measured.\n")

	case 2: // Architecture Constraints
		b.WriteString("A constraint is something you could not choose. archdoc sees the result of\n")
		b.WriteString("every choice and cannot tell which were free.\n\n")
		if tech := technologies(view); len(tech) > 0 {
			fmt.Fprintf(&b, "- These technologies are in use: %s. Which were imposed —\n", strings.Join(tech, ", "))
			b.WriteString("  by an existing system, a licence, a team skill, a regulation — and which were chosen?\n")
		}
		b.WriteString("- What may **not** change, and why?\n")

	case 4: // Solution Strategy
		fmt.Fprintf(&b, "The system is split into %d container(s). That split is a decision, and the\n", countContainers(view))
		b.WriteString("configuration records only its outcome.\n\n")
		b.WriteString("- Why this decomposition rather than fewer, larger pieces?\n")
		if stores := kindOf(view, archdoc.Datastore); len(stores) > 1 {
			fmt.Fprintf(&b, "- There are %d data stores (%s). Why more than one?\n",
				len(stores), strings.Join(stores, ", "))
		}
		if ext := kindOf(view, archdoc.External); len(ext) > 0 {
			fmt.Fprintf(&b, "- %s is depended on rather than built. What was the buy-versus-build reasoning?\n",
				strings.Join(ext, ", "))
		}

	case 8: // Cross-cutting Concepts
		b.WriteString("Concerns that touch every container, and that configuration never names.\n\n")
		b.WriteString("- **Authentication and authorisation** — where is identity established, and how\n")
		b.WriteString("  does it travel between containers?\n")
		b.WriteString("- **Logging, metrics, tracing** — is there a shared approach?\n")
		b.WriteString("- **Configuration and secrets** — how do they reach a running container?\n")
		if stores := kindOf(view, archdoc.Datastore); len(stores) > 0 {
			fmt.Fprintf(&b, "- **Persistence** — %s hold state. What is the backup and migration story?\n",
				strings.Join(stores, ", "))
		}

	case 9: // Architecture Decisions
		b.WriteString("The decisions behind the picture in section 5. Record why, not what — the what\n")
		b.WriteString("is already generated, and git already has the change.\n\n")
		if hidden := excludedNames(m, view); len(hidden) > 0 {
			fmt.Fprintf(&b, "- %s is infrastructure and is not shown as a container. Was that the\n",
				strings.Join(hidden, ", "))
			b.WriteString("  right call for a reader of this document?\n")
		}
		b.WriteString("- Which of the technologies in section 5 were contested, and what lost?\n")
		b.WriteString("- What would you do differently if you started again?\n")

	case 10: // Quality Requirements
		b.WriteString("Goals stated so they can be checked. \"Fast\" is not checkable; \"95th percentile\n")
		b.WriteString("under 200 ms at 500 requests per second\" is.\n\n")
		if entry := entryPoints(view); len(entry) > 0 {
			fmt.Fprintf(&b, "- %s is reachable from outside the system. What are its availability\n",
				strings.Join(entry, ", "))
			b.WriteString("  and latency targets?\n")
		}
		if stores := kindOf(view, archdoc.Datastore); len(stores) > 0 {
			b.WriteString("- What is the acceptable data loss window if a data store fails?\n")
		}

	case 11: // Risks and Technical Debt
		b.WriteString("What worries you, written down before it happens.\n\n")
		if busy := mostDependedOn(view); busy != "" {
			fmt.Fprintf(&b, "- **%s** has more inbound relationships than anything else in the model.\n", busy)
			b.WriteString("  What happens when it is unavailable?\n")
		}
		if ext := kindOf(view, archdoc.External); len(ext) > 0 {
			fmt.Fprintf(&b, "- %s is outside the system and outside your control. What is the\n",
				strings.Join(ext, ", "))
			b.WriteString("  fallback?\n")
		}
		if lonely := unconnected(view); len(lonely) > 0 {
			fmt.Fprintf(&b, "- %s appears in the model with no relationships at all. Either it is\n",
				strings.Join(lonely, ", "))
			b.WriteString("  genuinely isolated, or its connections are made somewhere configuration cannot\n")
			b.WriteString("  see — which is worth knowing either way.\n")
		}
	}

	return b.String()
}

func kindOf(m archdoc.Model, k archdoc.Kind) []string {
	var out []string
	for _, n := range m.Nodes {
		if n.Kind == k {
			out = append(out, n.Name)
		}
	}
	return out
}

func countContainers(m archdoc.Model) int {
	n := 0
	for _, node := range m.Nodes {
		if node.Kind.Container() {
			n++
		}
	}
	return n
}

func technologies(m archdoc.Model) []string {
	seen := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Technology != "" {
			seen[n.Technology] = true
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out) // Go randomises map iteration; AC-7
	return out
}

func excludedNames(full, view archdoc.Model) []string {
	shown := map[string]bool{}
	for _, n := range view.Nodes {
		shown[n.ID] = true
	}

	var out []string
	for _, n := range full.Nodes {
		if !shown[n.ID] {
			out = append(out, n.Name)
		}
	}
	return out
}

// entryPoints are the containers something outside the system reaches directly.
func entryPoints(m archdoc.Model) []string {
	var out []string
	for _, e := range m.Edges {
		from, ok := m.Node(e.From)
		if !ok || from.Kind != archdoc.Actor {
			continue
		}
		if to, ok := m.Node(e.To); ok {
			out = append(out, to.Name)
		}
	}
	return out
}

// mostDependedOn is the crudest possible single-point-of-failure hint, and it is still worth
// asking about: the thing everything talks to is the thing whose failure is worst.
func mostDependedOn(m archdoc.Model) string {
	inbound := map[string]int{}
	for _, e := range m.Edges {
		if from, ok := m.Node(e.From); ok && from.Kind == archdoc.Actor {
			continue // reaching in from outside is not a dependency between containers
		}
		inbound[e.To]++
	}

	best, count := "", 1 // one inbound edge is not a finding
	for _, n := range m.Nodes {
		if inbound[n.ID] > count {
			best, count = n.Name, inbound[n.ID]
		}
	}
	return best
}

// unconnected names elements in the model that nothing reaches and that reach nothing. Immich's
// machine-learning service is the standing example — every service calls it over a URL built at
// runtime, so configuration never declares the link.
func unconnected(m archdoc.Model) []string {
	touched := map[string]bool{}
	for _, e := range m.Edges {
		touched[e.From], touched[e.To] = true, true
	}

	var out []string
	for _, n := range m.Nodes {
		if n.Kind.Container() && !touched[n.ID] {
			out = append(out, n.Name)
		}
	}
	return out
}
