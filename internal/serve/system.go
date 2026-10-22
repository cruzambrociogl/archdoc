package serve

import (
	"net/http"
	"path/filepath"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/system"
)

// NewSystem serves a folder of documented repositories (platform-surface §4.6): the system at
// /api/system, and each repository's own API under /repos/{name}/ — the same endpoints, answered by
// a server of its own, with its own session token for its own actions.
func NewSystem(dir string) (http.Handler, error) {
	mux, err := systemMux(dir)
	if err != nil {
		return nil, err
	}
	mux.Handle("/", frontend())
	return thisMachine(mux), nil
}

// systemMux is a system's API alone — what Home mounts under a place's address too.
func systemMux(dir string) (*http.ServeMux, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	sys, err := system.Read(abs)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	repos := map[string]*Server{}
	for _, r := range sys.Repos {
		repos[r.Name] = newServer(r.Root, r.Root)
	}
	mux.HandleFunc("GET /api/system", func(w http.ResponseWriter, r *http.Request) {
		// Read again on every request: a repository documented since answers with its new version.
		sys, err := system.Read(abs)
		if err != nil {
			fail(w, err)
			return
		}
		send(w, systemJSON(sys))
	})
	mux.HandleFunc("/repos/{name}/api/", func(w http.ResponseWriter, r *http.Request) {
		s, ok := repos[r.PathValue("name")]
		if !ok {
			notFound(w, "no repository "+r.PathValue("name")+" in this system")
			return
		}
		http.StripPrefix("/repos/"+r.PathValue("name"), s.mux).ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { notFound(w, "no such endpoint: "+r.URL.Path) })
	return mux, nil
}

// systemJSON is the system as the app reads it: each repository with what its latest version
// says it is, and the links.
func systemJSON(sys system.System) map[string]any {
	repos := make([]map[string]any, 0, len(sys.Repos))
	for _, r := range sys.Repos {
		apps := 0
		for _, n := range r.Model.Nodes {
			if n.Kind == core.Application {
				apps++
			}
		}
		repo := map[string]any{"name": r.Name, "root": r.Root, "version": r.Version, "commit": r.Commit, "created_at": r.At, "applications": apps}
		if r.Model.Description != "" {
			repo["description"], repo["description_provenance"] = r.Model.Description, r.Model.DescProv
		}
		repos = append(repos, repo)
	}
	return map[string]any{"name": sys.Name, "repos": repos, "links": sys.Links}
}
