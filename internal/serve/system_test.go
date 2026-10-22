package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// A folder of two documented repositories, the second requiring the first's package.
func folder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, r := range []struct {
		name     string
		manifest core.Manifest
	}{
		{"engine", core.Manifest{Package: "@acme/app", Dir: ".", Language: "JavaScript", Prov: core.Provenance{File: "package.json", Line: 2}}},
		{"platform", core.Manifest{Package: "@acme/web", Dir: ".", Language: "JavaScript", Prov: core.Provenance{File: "package.json", Line: 2},
			Requires: []core.Requirement{{Name: "@acme/app", Version: "^1.0.0", Prov: core.Provenance{File: "package.json", Line: 5}}}}},
	} {
		s, err := store.Open(filepath.Join(dir, r.name))
		if err != nil {
			t.Fatal(err)
		}
		m := core.Model{Name: r.name, Manifests: []core.Manifest{r.manifest}}
		if _, _, err := s.Save(m, nil, "abc123", "test"); err != nil {
			t.Fatal(err)
		}
		s.Close()
	}
	return dir
}

// The system answers at /api/system; each repository answers its own API under /repos/{name}/;
// anything else under /api is an error, never the app's page.
func TestASystemServesEachRepositoryUnderItsName(t *testing.T) {
	h, err := NewSystem(folder(t))
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		req.Host = "localhost:7474"
		h.ServeHTTP(rec, req)
		return rec
	}

	var sys struct {
		Repos []struct{ Name string }
		Links []struct{ From, To, Package string }
	}
	rec := get("/api/system")
	if err := json.Unmarshal(rec.Body.Bytes(), &sys); err != nil || len(sys.Repos) != 2 || len(sys.Links) != 1 || sys.Links[0].From != "platform" {
		t.Fatalf("/api/system: %d %s (%v)", rec.Code, rec.Body, err)
	}
	if rec := get("/repos/engine/api/summary"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name": "engine"`) {
		t.Errorf("a repository's summary: %d %s", rec.Code, rec.Body)
	}
	for _, path := range []string{"/repos/nope/api/summary", "/api/nope"} {
		if rec := get(path); rec.Code != http.StatusNotFound || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s: %d %s", path, rec.Code, rec.Header().Get("Content-Type"))
		}
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/system", nil)
	req.Host = "evil.example:7474"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("another host's request: %d", rec.Code)
	}
}

// One repository's server has no system: the app asks, and is told so rather than given its page.
func TestARepositoryIsNotASystem(t *testing.T) {
	s, err := New(repo(t))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/system", nil)
	req.Host = "localhost"
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("/api/system on one repository: %d", rec.Code)
	}
}
