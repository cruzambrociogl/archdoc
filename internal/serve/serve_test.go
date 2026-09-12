package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/extract"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

// repo copies a fixture repository somewhere writable and gives it the history generate would:
// two versions, one run, a generated document and the human stubs.
func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "rules")
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	facts, err := extract.Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	m := model.Derive(facts)

	h, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if _, _, err := h.Save(m, nil, "abc123", "test"); err != nil {
		t.Fatal(err)
	}
	grown := m
	grown.Nodes = append(append([]archdoc.Node(nil), m.Nodes...), archdoc.Node{
		ID: "svc:cache", Name: "cache", Kind: archdoc.Datastore, Evidence: archdoc.Declared,
		Prov: archdoc.Provenance{File: "docker-compose.yml", Line: 99},
	})
	if _, _, err := h.Save(grown.Normalise(), nil, "def456", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.SaveRun(store.Run{Status: "ok", EgressMode: "structure-only", Model: "claude-opus-5",
		Requests: 1, BytesSent: 12, Exchanges: []store.Exchange{{Method: "POST", URL: "https://api", Status: 200, Body: `{"x":1}`}}}); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, docsDir)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, render.IndexFile), []byte("# generated\n"), 0o644)
	for name, content := range render.Stubs(m, render.Meta{}) {
		os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
		render.RecordStub(root, docsDir, name)
	}
	return root
}

func server(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := repo(t)
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	return ts, root
}

func get(t *testing.T, ts *httptest.Server, path string) (int, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", ts.URL+path, nil)
	req.Host = "localhost"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func TestSummaryAndVersions(t *testing.T) {
	ts, _ := server(t)

	code, b := get(t, ts, "/api/summary")
	if code != 200 {
		t.Fatalf("summary: %d %s", code, b)
	}
	var sum map[string]any
	json.Unmarshal(b, &sum)
	if sum["versions"].(float64) != 2 || sum["runs"].(float64) != 1 {
		t.Errorf("summary: %s", b)
	}

	if code, b := get(t, ts, "/api/versions"); code != 200 || !strings.Contains(string(b), "def456") {
		t.Errorf("versions: %d %s", code, b)
	}
}

// The canvas is the committed SVG, drawn by the same code. Every element is a group with its id,
// which is what the page attaches clicks to.
func TestCanvasIsTheEngineSVG(t *testing.T) {
	ts, _ := server(t)
	code, b := get(t, ts, "/api/svg?view=container")
	if code != 200 || !strings.HasPrefix(string(b), "<svg") {
		t.Fatalf("svg: %d %.80s", code, b)
	}
	if !strings.Contains(string(b), `<g id="svc:api">`) {
		t.Error("an element is not addressable by its id")
	}
	if code, _ := get(t, ts, "/api/svg?view=nonsense"); code != 400 {
		t.Errorf("an unknown view was accepted: %d", code)
	}
}

func TestDiffBetweenVersions(t *testing.T) {
	ts, _ := server(t)
	code, b := get(t, ts, "/api/diff?from=1&to=2")
	if code != 200 {
		t.Fatalf("diff: %d %s", code, b)
	}
	if !strings.Contains(string(b), `"structural": true`) || !strings.Contains(string(b), "svc:cache") {
		t.Errorf("the added element is not in the diff: %s", b)
	}
}

func TestRunsShowExactlyWhatWasSent(t *testing.T) {
	ts, _ := server(t)
	code, b := get(t, ts, "/api/runs/1")
	if code != 200 || !strings.Contains(string(b), `{\"x\":1}`) {
		t.Errorf("run: %d %s", code, b)
	}
	if code, _ := get(t, ts, "/api/runs/99"); code != 404 {
		t.Errorf("a missing run: %d", code)
	}
}

func TestRulesShowWhatTheyDid(t *testing.T) {
	ts, _ := server(t)
	code, b := get(t, ts, "/api/rules")
	if code != 200 {
		t.Fatalf("rules: %d %s", code, b)
	}
	if !strings.Contains(string(b), `"set_name"`) || !strings.Contains(string(b), "rules.yaml") {
		t.Errorf("the compiled operations are missing: %.300s", b)
	}
}

// OUT-03 in the app: generated documents are served, human sections are refused — not read.
func TestHumanSectionsAreNeverRead(t *testing.T) {
	ts, root := server(t)

	if code, b := get(t, ts, "/api/docs/"+render.IndexFile); code != 200 || !strings.Contains(string(b), "generated") {
		t.Errorf("a generated document was not served: %d", code)
	}

	secret := "This sentence is mine."
	os.WriteFile(filepath.Join(root, docsDir, "01-introduction-and-goals.md"), []byte(secret), 0o644)
	code, b := get(t, ts, "/api/docs/01-introduction-and-goals.md")
	if code != 403 {
		t.Errorf("a human section was served: %d", code)
	}
	if strings.Contains(string(b), secret) {
		t.Error("the content of a human section left the server")
	}

	if code, _ := get(t, ts, "/api/docs/..%2Fdocker-compose.yml"); code == 200 {
		t.Error("a path outside the documentation directory was served")
	}
}

func TestCompletenessReportsSections(t *testing.T) {
	ts, root := server(t)
	os.WriteFile(filepath.Join(root, docsDir, "04-solution-strategy.md"), []byte("written\n"), 0o644)

	code, b := get(t, ts, "/api/completeness")
	if code != 200 {
		t.Fatalf("completeness: %d %s", code, b)
	}
	s := string(b)
	if !strings.Contains(s, `"not started"`) || !(strings.Contains(s, `"written"`) || strings.Contains(s, `"may be stale"`)) {
		t.Errorf("states: %s", s)
	}
}

// A web page elsewhere can make a browser call localhost. Checking the Host header is what stops
// it reading the model through a rebound DNS name.
func TestOnlyThisMachineIsServed(t *testing.T) {
	ts, _ := server(t)
	req, _ := http.NewRequest("GET", ts.URL+"/api/summary", nil)
	req.Host = "evil.example"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("a request for another host was served: %d", resp.StatusCode)
	}
}

func TestNoHistoryIsAClearError(t *testing.T) {
	if _, err := New(t.TempDir()); err == nil || !strings.Contains(err.Error(), "archdoc generate") {
		t.Errorf("got %v", err)
	}
}
