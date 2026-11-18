package render

import (
	"path/filepath"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Deployment names the system's deployment view: where it runs, as the Compose file states it
// (platform-surface §16.1, revision 7: "Where it runs drawn").
const Deployment = "deployment"

// composeFile reports whether a path is a Compose file: docker-compose.yml, compose.yaml, and their
// suffixed kin — docker-compose.prod.yml.
func composeFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "compose") && (strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml"))
}

// deployed lists the parts the Compose file runs: the applications, stores and queues declared at a
// line of it, in model order. A part another file declares — an app's manifest — is not run by it,
// whatever it is called.
func deployed(m core.Model) []string {
	if !composeFile(m.Source) {
		return nil
	}
	var out []string
	for _, n := range m.Nodes {
		if n.Kind.Container() && n.Prov.File == m.Source {
			out = append(out, n.ID)
		}
	}
	return out
}

// HasDeployment reports whether the model has a deployment view: a Compose file that runs at least
// one part, in a project large enough to have a container view.
func HasDeployment(m core.Model) bool {
	return !Tiny(m) && len(deployed(m)) > 0
}

// deploymentView is C4's deployment diagram, as far as Compose states it: one deployment node, the
// Compose file, holding the services it runs — nested in its networks where it declares them — and,
// outside it, the systems those services reach. Ports, hosts and replicas are not drawn: the model
// does not keep them. What the file does not run — a mobile app, a command line — is not in it.
func deploymentView(m core.Model) View {
	run := map[string]bool{}
	for _, id := range deployed(m) {
		run[id] = true
	}
	all := m.Container()
	keep := map[string]bool{}
	for id := range run {
		keep[id] = true
	}
	kind := map[string]core.Kind{}
	for _, n := range all.Nodes {
		kind[n.ID] = n.Kind
	}
	for _, e := range all.Edges {
		if run[e.From] && kind[e.To] == core.External {
			keep[e.To] = true
		}
	}

	view := all
	view.Nodes, view.Edges = nil, nil
	for _, n := range all.Nodes {
		if keep[n.ID] {
			view.Nodes = append(view.Nodes, n)
		}
	}
	for _, e := range all.Edges {
		if keep[e.From] && keep[e.To] {
			view.Edges = append(view.Edges, e)
		}
	}
	view.Boundary = m.Source + " [Deployment Node: Docker Compose]"
	return View{Name: Deployment, File: "deployment", Title: "Deployment", Model: Overview(m, view), Group: true}
}
