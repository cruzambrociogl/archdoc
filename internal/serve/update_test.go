package serve

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// updatingServer is a repository's server updated through up, and a way to call it as its page does.
func updatingServer(t *testing.T, up Updater) (*Server, func(method, path, body string) *httptest.ResponseRecorder) {
	t.Helper()
	s, err := New(repo(t), up)
	if err != nil {
		t.Fatal(err)
	}
	return s, func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Host = "localhost:7474"
		req.Header.Set("Origin", "http://localhost:7474")
		req.Header.Set(TokenHeader, s.token)
		s.ServeHTTP(rec, req)
		return rec
	}
}

type updatingJSON struct {
	Running bool
	Can     bool
	Mine    bool
	By      string
	Doing   string
	Last    *finished
}

func updatingNow(t *testing.T, do func(method, path, body string) *httptest.ResponseRecorder) updatingJSON {
	t.Helper()
	var got updatingJSON
	if err := json.Unmarshal(do("GET", "/api/updating", "").Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

// The app sees a run updating the repository from the terminal, with what it is doing. Where the
// server was given no update to run — a published site's — updating is off, and nothing runs.
func TestUpdatingIsSeenAndOffWithoutAnUpdater(t *testing.T) {
	s, do := updatingServer(t, nil)
	if got := updatingNow(t, do); got.Running || got.Can {
		t.Fatalf("before anything runs, with no updater: %+v", got)
	}
	l, err := store.Take(context.Background(), s.root, "archdoc generate", nil)
	if err != nil {
		t.Fatal(err)
	}
	l.Report(core.Progress{Doing: "drawing the diagrams"})
	if got := updatingNow(t, do); !got.Running || got.By != "archdoc generate" || got.Doing != "drawing the diagrams" || got.Mine {
		t.Errorf("while the terminal updates: %+v", got)
	}
	l.Release()
	for _, path := range []string{"/api/update", "/api/update/preview"} {
		if rec := do("POST", path, `{"label":true}`); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "archdoc generate") {
			t.Errorf("%s with no updater: %d %s", path, rec.Code, rec.Body)
		}
	}
}

// Update from the app: it starts at once, one at a time — not while the terminal updates either —
// and the page is told how it ended.
func TestUpdateFromTheApp(t *testing.T) {
	release := make(chan struct{})
	var asked Run
	s, do := updatingServer(t, func(ctx context.Context, root string, r Run) (Outcome, error) {
		asked = r
		<-release
		r.Log.Write([]byte("recorded version 3\n"))
		return Outcome{Version: 3, Changed: true}, nil
	})
	if got := updatingNow(t, do); !got.Can {
		t.Fatal("the page isn't told it can update")
	}
	if rec := do("GET", "/api/update", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("a GET started an update: %d", rec.Code)
	}
	if rec := do("POST", "/api/update", ""); rec.Code != http.StatusAccepted {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if got := updatingNow(t, do); !got.Running || !got.Mine || got.By != "the app" {
		t.Errorf("while it runs: %+v", got)
	}
	if rec := do("POST", "/api/update", ""); rec.Code != http.StatusConflict {
		t.Errorf("a second update at once: %d %s", rec.Code, rec.Body)
	}
	close(release)
	var got updatingJSON
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if got = updatingNow(t, do); !got.Running {
			break
		}
	}
	if got.Running || got.Last == nil || got.Last.Version != 3 || !got.Last.Changed || got.Last.Error != "" || got.Last.AI || !strings.Contains(got.Last.Log, "recorded version 3") {
		t.Errorf("after it ended: %+v %+v", got, got.Last)
	}
	if asked.Label || asked.Explain || asked.DryRun || asked.Docs {
		t.Errorf("a plain update asked for %+v", asked)
	}

	l, err := store.Take(context.Background(), s.root, "archdoc generate", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	if rec := do("POST", "/api/update", ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "archdoc generate") {
		t.Errorf("while the terminal updates: %d %s", rec.Code, rec.Body)
	}
}

// The Documents screen's button writes the documents into the repository: an update that says so.
func TestUpdateWritesTheDocumentsWhenAsked(t *testing.T) {
	asked := make(chan Run, 1)
	_, do := updatingServer(t, func(ctx context.Context, root string, r Run) (Outcome, error) {
		asked <- r
		return Outcome{}, nil
	})
	if rec := do("POST", "/api/update", `{"docs":true}`); rec.Code != http.StatusAccepted {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	select {
	case r := <-asked:
		if !r.Docs || r.Label || r.Explain {
			t.Errorf("asked to write the documents, the run was %+v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no run started")
	}
}

// Update with AI: the preview is a dry run, every paid run's requests and cost; the run needs the
// key, and sends only within what was agreed to.
func TestUpdateWithAIIsShownThenAgreed(t *testing.T) {
	label := Ask{What: "label", Model: "m", Requests: 1, Bytes: 900, USD: 0.01, Known: true}
	explain := Ask{What: "explain", Model: "m", Requests: 4, Bytes: 4000, USD: 0.04, Known: true}
	var sent []string
	done := make(chan error, 1)
	_, do := updatingServer(t, func(ctx context.Context, root string, r Run) (Outcome, error) {
		for _, a := range []Ask{label, explain} {
			if (a.What == "label" && !r.Label) || (a.What == "explain" && !r.Explain) {
				continue
			}
			if err := r.Agree(a); err != nil {
				if !r.DryRun {
					done <- err
				}
				return Outcome{}, err
			}
			if !r.DryRun {
				sent = append(sent, a.What)
			}
		}
		if r.DryRun {
			return Outcome{}, ErrStopped
		}
		done <- nil
		return Outcome{Version: 2, Changed: true}, nil
	})

	t.Setenv(keyVar, "")
	rec := do("POST", "/api/update/preview", `{"label":true,"explain":true}`)
	var preview struct {
		Asks []Ask
		Key  bool
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &preview); err != nil || rec.Code != 200 {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body)
	}
	if len(preview.Asks) != 2 || preview.Asks[0] != label || preview.Asks[1] != explain || preview.Key {
		t.Errorf("preview: %+v", preview)
	}
	if len(sent) > 0 {
		t.Fatalf("the preview sent %v", sent)
	}
	if rec := do("POST", "/api/update/preview", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a preview of nothing: %d", rec.Code)
	}
	if rec := do("POST", "/api/update", `{"label":true}`); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), keyVar) {
		t.Errorf("with no key: %d %s", rec.Code, rec.Body)
	}

	t.Setenv(keyVar, "sk-test")
	wait := func() error {
		t.Helper()
		select {
		case err := <-done:
			for updatingNow(t, do).Running {
				time.Sleep(5 * time.Millisecond)
			}
			return err
		case <-time.After(5 * time.Second):
			t.Fatal("the run never asked")
			return nil
		}
	}
	// Agreed to labels only: explanations aren't sent.
	if rec := do("POST", "/api/update", `{"label":true,"explain":true,"agreed":{"label":{"what":"label","requests":1,"usd":0.01}}}`); rec.Code != http.StatusAccepted {
		t.Fatalf("update with AI: %d %s", rec.Code, rec.Body)
	}
	if err := wait(); err == nil || !strings.Contains(err.Error(), "explain wasn’t agreed to") || len(sent) != 1 {
		t.Errorf("explain unagreed: %v, sent %v", err, sent)
	}
	if last := updatingNow(t, do).Last; last == nil || !last.AI || last.Error == "" {
		t.Errorf("the page isn't told it stopped: %+v", last)
	}
	// Agreed to less than it would now cost: stopped before sending.
	sent = nil
	if rec := do("POST", "/api/update", `{"label":true,"agreed":{"label":{"what":"label","requests":1,"usd":0.005}}}`); rec.Code != http.StatusAccepted {
		t.Fatalf("update with AI: %d %s", rec.Code, rec.Body)
	}
	if err := wait(); err == nil || !strings.Contains(err.Error(), "more than") || len(sent) != 0 {
		t.Errorf("over the agreed cost: %v, sent %v", err, sent)
	}
	// Agreed to what it costs: sent.
	if rec := do("POST", "/api/update", `{"label":true,"explain":true,"agreed":{"label":{"what":"label","requests":1,"usd":0.01},"explain":{"what":"explain","requests":4,"usd":0.04}}}`); rec.Code != http.StatusAccepted {
		t.Fatalf("update with AI: %d %s", rec.Code, rec.Body)
	}
	if err := wait(); err != nil || len(sent) != 2 {
		t.Errorf("agreed: %v, sent %v", err, sent)
	}
}

// What was agreed to is a ceiling: no more requests, and the cost no more than a little over.
func TestWithinAgreed(t *testing.T) {
	agreed := Ask{What: "explain", Requests: 10, USD: 0.10, Known: true}
	cases := []struct {
		name string
		a    Ask
		ok   bool
	}{
		{"as shown", Ask{What: "explain", Requests: 10, USD: 0.10, Known: true}, true},
		{"fewer, a little dearer", Ask{What: "explain", Requests: 9, USD: 0.12, Known: true}, true},
		{"more requests", Ask{What: "explain", Requests: 11, USD: 0.10, Known: true}, false},
		{"much dearer", Ask{What: "explain", Requests: 10, USD: 0.20, Known: true}, false},
		{"another run", Ask{What: "label", Requests: 1, USD: 0.01, Known: true}, false},
	}
	for _, c := range cases {
		if err := withinAgreed(c.a, agreed); (err == nil) != c.ok {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if err := withinAgreed(Ask{What: "label"}, Ask{}); err == nil || errors.Is(err, ErrStopped) {
		t.Errorf("nothing agreed: %v", err)
	}
}
