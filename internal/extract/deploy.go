package extract

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Which Compose service runs which application.
//
// A deployed Compose file usually names images, not sources: Immich's docker-compose.yml runs
// ghcr.io/immich-app/immich-server, and nothing in it says that image is the server/ directory.
// Another Compose file in the same repository often does — Immich's docker-compose.dev.yml builds
// the same immich-server service from server/Dockerfile.dev. That build line is evidence a reader
// can open; a resemblance between names is not, and is never used.

// build is one service's build instruction, from any Compose file discovery considered.
type build struct {
	service string
	dirs    []string // the context, and the directory of the Dockerfile, repository-relative
	prov    archdoc.Provenance
}

// builds reads the build: of every service in every candidate Compose file.
func builds(root string, candidates []archdoc.Candidate) []build {
	var out []build
	for _, c := range candidates {
		content, err := os.ReadFile(filepath.Join(root, c.File))
		if err != nil {
			continue
		}
		var doc yaml.Node
		if yaml.Unmarshal(stripComposeTags(content), &doc) != nil || len(doc.Content) == 0 {
			continue
		}
		services := mapValue(doc.Content[0], "services")
		if services == nil || services.Kind != yaml.MappingNode {
			continue
		}
		base := path.Dir(c.File)
		for i := 0; i+1 < len(services.Content); i += 2 {
			name, svc := services.Content[i].Value, services.Content[i+1]
			key, b := mapEntry(svc, "build")
			if b == nil {
				continue
			}
			var context, dockerfile string
			switch b.Kind {
			case yaml.ScalarNode:
				context = b.Value
			case yaml.MappingNode:
				if v := mapValue(b, "context"); v != nil {
					context = v.Value
				}
				if v := mapValue(b, "dockerfile"); v != nil {
					dockerfile = v.Value
				}
			}
			if context == "" {
				context = "."
			}
			ctx := path.Clean(path.Join(base, context))
			dirs := []string{ctx}
			if dockerfile != "" {
				dirs = append(dirs, path.Dir(path.Clean(path.Join(ctx, dockerfile))))
			}
			out = append(out, build{service: name, dirs: dirs,
				prov: archdoc.Provenance{File: c.File, Line: key.Line, Column: key.Column}})
		}
	}
	return out
}

// deploy ties each container application to the service in the deployed Compose file that some
// Compose file builds from its directory. An application two services claim, or a service two
// applications claim, is left untied: ambiguity is not evidence.
func deploy(apps []archdoc.App, services []archdoc.Service, bs []build) {
	deployed := map[string]bool{}
	for _, s := range services {
		deployed[s.Name] = true
	}
	claims := map[string][]int{}        // service → indexes of apps it builds from
	byApp := map[int]map[string]build{} // app → services that build from it
	for i, a := range apps {
		if !a.Role.Container() {
			continue
		}
		for _, b := range bs {
			if !deployed[b.service] || !builtFrom(b, a.Dir) {
				continue
			}
			if byApp[i] == nil {
				byApp[i] = map[string]build{}
			}
			if _, seen := byApp[i][b.service]; !seen {
				byApp[i][b.service] = b
				claims[b.service] = append(claims[b.service], i)
			}
		}
	}
	for i, svcs := range byApp {
		if len(svcs) != 1 {
			continue
		}
		for name, b := range svcs {
			if len(claims[name]) == 1 {
				apps[i].Deployed = &archdoc.Deployment{Service: name, Prov: b.prov}
			}
		}
	}

	// Where no build line ties them, a service whose name is exactly the application's — its
	// directory or its unscoped package name — is tied by name, and says so: resolved by name is
	// weaker evidence than a build line, and the provenance records which it was (vision D-4).
	// Only an exact, unique match counts; a resemblance never does.
	tied := map[string]bool{}
	for _, a := range apps {
		if a.Deployed != nil {
			tied[a.Deployed.Service] = true
		}
	}
	byName := map[string][]int{}
	for i, a := range apps {
		if !a.Role.Container() || a.Deployed != nil || a.Dir == "." {
			continue
		}
		for _, n := range uniqueNames(path.Base(a.Dir), unscoped(a.Name)) {
			byName[n] = append(byName[n], i)
		}
	}
	for _, s := range services {
		idx := byName[s.Name]
		if tied[s.Name] || len(idx) != 1 || apps[idx[0]].Deployed != nil {
			continue
		}
		prov := s.Prov
		prov.Note = "tied by name: the service is named like the application, and no build line ties them"
		apps[idx[0]].Deployed = &archdoc.Deployment{Service: s.Name, Prov: prov, ByName: true}
	}
}

func unscoped(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func uniqueNames(a, b string) []string {
	if a == b {
		return []string{a}
	}
	return []string{a, b}
}

// builtFrom reports whether a build's context or Dockerfile lies in the application's directory.
// The repository root never counts: a root context says nothing about which application it is.
func builtFrom(b build, dir string) bool {
	if dir == "." {
		return false
	}
	for _, d := range b.dirs {
		if d == dir {
			return true
		}
	}
	return false
}

// mapEntry returns a mapping's key node and value node for key.
func mapEntry(m *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	_, v := mapEntry(m, key)
	return v
}

// sortedApps keeps applications in manifest order (AC-7).
func sortedApps(apps []archdoc.App) {
	sort.Slice(apps, func(i, j int) bool { return apps[i].Manifest < apps[j].Manifest })
}
