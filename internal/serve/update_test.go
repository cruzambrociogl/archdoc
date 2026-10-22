package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// The app sees a run updating the repository, from the terminal, with what it is doing; asked to
// update, it is told updating from the app is off — and nothing runs.
func TestUpdatingIsSeenAndUpdatingFromTheAppIsOff(t *testing.T) {
	s, err := New(repo(t))
	if err != nil {
		t.Fatal(err)
	}
	do := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.Host = "localhost:7474"
		req.Header.Set("Origin", "http://localhost:7474")
		req.Header.Set(TokenHeader, s.token)
		s.ServeHTTP(rec, req)
		return rec
	}
	var got struct {
		Running bool
		By      string
		Doing   string
	}
	json.Unmarshal(do("GET", "/api/updating").Body.Bytes(), &got)
	if got.Running {
		t.Fatalf("running before anything runs: %+v", got)
	}

	l, err := store.Take(context.Background(), s.root, "archdoc generate", nil)
	if err != nil {
		t.Fatal(err)
	}
	l.Report(core.Progress{Doing: "drawing the diagrams"})
	json.Unmarshal(do("GET", "/api/updating").Body.Bytes(), &got)
	if !got.Running || got.By != "archdoc generate" || got.Doing != "drawing the diagrams" {
		t.Errorf("while the terminal updates: %+v", got)
	}
	l.Release()

	if rec := do("POST", "/api/update"); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "switched off") {
		t.Errorf("updating from the app: %d %s", rec.Code, rec.Body)
	}
	// Built, though off: given an update to run, the action starts it.
	s.update = func(context.Context) error { return nil }
	if rec := do("POST", "/api/update"); rec.Code != http.StatusAccepted {
		t.Errorf("with an update given: %d %s", rec.Code, rec.Body)
	}
}
