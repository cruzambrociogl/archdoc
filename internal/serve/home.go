package serve

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/recent"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/internal/system"
)

// NewHome serves Home (platform-surface §5.3): the places this machine remembers — repositories,
// and folders of them — at /api/home, and each one's own API under /at/{id}/, as `archdoc serve`
// would serve it alone. Opening a place from Home is an action: it remembers each repository's
// version then, which is what "changed since you last looked" counts from.
func NewHome() http.Handler {
	token := newToken()
	mux := http.NewServeMux()
	var mu sync.Mutex
	places := map[string]http.Handler{} // by id, built when first asked for

	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		send(w, map[string]any{"token": token, "mode": "home"})
	})
	mux.HandleFunc("GET /api/home", func(w http.ResponseWriter, r *http.Request) {
		list, err := recent.List()
		if err != nil {
			fail(w, err)
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, p := range list {
			out = append(out, homePlace(p))
		}
		send(w, map[string]any{"places": out})
	})
	mux.HandleFunc("/api/home/open", gated(token, func(w http.ResponseWriter, r *http.Request) {
		var body struct{ ID string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			refuse(w, http.StatusBadRequest, "the body names no place")
			return
		}
		p, ok := placeOf(body.ID)
		if !ok {
			notFound(w, "no such place")
			return
		}
		seen, err := system.Versions(p.Path)
		if err == nil {
			err = recent.Open(p.Path, time.Now().UTC(), seen)
		}
		if err != nil {
			fail(w, err)
			return
		}
		send(w, map[string]any{"opened": body.ID})
	}))
	mux.HandleFunc("/at/{id}/", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		mu.Lock()
		h, ok := places[id]
		if !ok {
			if p, known := placeOf(id); known {
				if h, ok = placeHandler(p.Path); ok {
					places[id] = h
				}
			}
		}
		mu.Unlock()
		switch {
		case ok:
			http.StripPrefix("/at/"+id, h).ServeHTTP(w, r)
		case strings.Contains(r.URL.Path, "/api/"):
			notFound(w, "no such place, or its folder is not documented any more")
		default:
			frontend().ServeHTTP(w, r) // the app's page, which says so
		}
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { notFound(w, "no such endpoint: "+r.URL.Path) })
	mux.Handle("/", frontend())
	return thisMachine(mux)
}

// placeOf is the remembered place with an id.
func placeOf(id string) (recent.Place, bool) {
	list, _ := recent.List()
	for _, p := range list {
		if recent.ID(p.Path) == id {
			return p, true
		}
	}
	return recent.Place{}, false
}

// placeHandler is a place's API, as `archdoc serve` would answer for it, with its page the app's.
func placeHandler(path string) (http.Handler, bool) {
	var api *http.ServeMux
	if system.IsRepo(path) {
		api = newServer(path, path).mux
	} else if mux, err := systemMux(path); err == nil {
		api = mux
	} else {
		return nil, false
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A repository's own server answers its page with the app; a system's mux has no page.
		if _, pattern := api.Handler(r); pattern == "" || pattern == "/" {
			frontend().ServeHTTP(w, r)
			return
		}
		api.ServeHTTP(w, r)
	}), true
}

// homePlace is one place as Home shows it: each of its repositories' latest version, how many
// versions came since it was last opened here, and what couldn't be resolved.
func homePlace(p recent.Place) map[string]any {
	out := map[string]any{"id": recent.ID(p.Path), "path": p.Path, "name": filepath.Base(p.Path), "touched": p.Touched}
	if !p.Opened.IsZero() {
		out["opened"] = p.Opened
	}
	var roots []string
	switch {
	case system.IsRepo(p.Path):
		out["kind"], roots = "repository", []string{p.Path}
	default:
		sys, err := system.Read(p.Path)
		if err != nil {
			if _, statErr := os.Stat(p.Path); statErr != nil {
				out["kind"], out["problem"] = "missing", "The folder isn’t there any more."
			} else {
				out["kind"], out["problem"] = "missing", "Nothing archdoc documented is there any more."
			}
			return out
		}
		out["kind"] = "system"
		for _, r := range sys.Repos {
			roots = append(roots, r.Root)
		}
	}
	repos := make([]map[string]any, 0, len(roots))
	for _, root := range roots {
		if r := homeRepo(root, p); r != nil {
			repos = append(repos, r)
		}
	}
	out["repos"] = repos
	return out
}

func homeRepo(root string, p recent.Place) map[string]any {
	s, err := store.Open(root)
	if err != nil {
		return nil
	}
	defer s.Close()
	latest, err := s.Latest()
	if err != nil || latest == nil {
		return nil
	}
	versions, _ := s.Versions(1000)
	// Since it was last opened here: how many versions are newer. Never opened: unknown.
	since := -1
	if !p.Opened.IsZero() {
		since = 0
		for _, v := range versions {
			if v.ID > p.Seen[root] {
				since++
			}
		}
	}
	var kinds []string
	for _, n := range latest.Model.Nodes {
		if n.Kind.Container() {
			kinds = append(kinds, string(n.Kind))
		}
	}
	return map[string]any{
		"name": filepath.Base(root), "root": root, "version": latest.ID, "commit": latest.Commit,
		"created_at": latest.CreatedAt, "since": since, "unresolved": len(latest.Model.Unresolved), "kinds": kinds,
	}
}
