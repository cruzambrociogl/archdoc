package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/recent"
)

// Home lists what this machine remembers, serves each place under /at/{id}/, and counts what
// changed since a place was opened; opening one is an action, behind the gate.
func TestHomeServesEveryRememberedPlace(t *testing.T) {
	file, was := filepath.Join(t.TempDir(), "recent.json"), recent.File
	recent.File = func() (string, error) { return file, nil }
	t.Cleanup(func() { recent.File = was })

	repoRoot := repo(t) // two versions
	sys := folder(t)
	at := time.Now().UTC()
	if err := recent.Open(repoRoot, at, map[string]int64{repoRoot: 1}); err != nil { // opened at version 1
		t.Fatal(err)
	}
	if err := recent.Touch(sys, at.Add(time.Second)); err != nil { // generated, never opened
		t.Fatal(err)
	}
	h := NewHome()
	do := func(method, path, body string, header map[string]string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Host = "localhost:7474"
		for k, v := range header {
			req.Header.Set(k, v)
		}
		h.ServeHTTP(rec, req)
		return rec
	}
	type place struct {
		ID, Kind string
		Repos    []struct {
			Name  string
			Since int
		}
	}
	home := func() []place {
		var out struct{ Places []place }
		if err := json.Unmarshal(do("GET", "/api/home", "", nil).Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Places
	}

	places := home()
	if len(places) != 2 || places[0].Kind != "system" || places[1].Kind != "repository" {
		t.Fatalf("places: %+v", places)
	}
	if places[1].Repos[0].Since != 1 || places[0].Repos[0].Since != -1 {
		t.Errorf("since: the repository %d (want 1, one version after the one opened), the system %d (want -1, never opened)", places[1].Repos[0].Since, places[0].Repos[0].Since)
	}
	sysID, repoID := places[0].ID, places[1].ID
	for path, want := range map[string]int{
		"/at/" + repoID + "/api/summary":             http.StatusOK,
		"/at/" + sysID + "/api/system":               http.StatusOK,
		"/at/" + sysID + "/repos/engine/api/summary": http.StatusOK,
		"/at/nope/api/summary":                       http.StatusNotFound,
	} {
		if rec := do("GET", path, "", nil); rec.Code != want {
			t.Errorf("%s: %d, want %d", path, rec.Code, want)
		}
	}

	open := `{"id":"` + repoID + `"}`
	if rec := do("POST", "/api/home/open", open, nil); rec.Code != http.StatusForbidden {
		t.Errorf("opening without the gate: %d", rec.Code)
	}
	var session struct{ Token string }
	json.Unmarshal(do("GET", "/api/session", "", nil).Body.Bytes(), &session)
	if rec := do("POST", "/api/home/open", open, map[string]string{"Origin": "http://localhost:7474", TokenHeader: session.Token}); rec.Code != http.StatusOK {
		t.Fatalf("opening: %d %s", rec.Code, rec.Body)
	}
	for _, p := range home() {
		if p.ID == repoID && p.Repos[0].Since != 0 {
			t.Errorf("after opening, since %d", p.Repos[0].Since)
		}
	}
}
