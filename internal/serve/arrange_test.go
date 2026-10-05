package serve

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/arrange"
)

// live is a server, the root it serves, and its page's token.
func live(t *testing.T) (*httptest.Server, string, string) {
	t.Helper()
	ts, root := server(t)
	_, b := get(t, ts, "/api/session")
	var sess struct {
		Token string `json:"token"`
	}
	json.Unmarshal(b, &sess)
	return ts, root, sess.Token
}

func post(t *testing.T, ts *httptest.Server, path, token string, body any) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", ts.URL+path, bytes.NewReader(raw))
	req.Host = "localhost"
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set(TokenHeader, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

type sceneJSON struct {
	Layout      archdoc.Layout `json:"layout"`
	Arrangement struct {
		Hash   string   `json:"hash"`
		Placed []string `json:"placed"`
	} `json:"arrangement"`
}

func scene(t *testing.T, ts *httptest.Server) sceneJSON {
	t.Helper()
	code, b := get(t, ts, "/api/scene?view=container")
	if code != 200 {
		t.Fatalf("scene: %d %s", code, b)
	}
	var sc sceneJSON
	json.Unmarshal(b, &sc)
	return sc
}

func boxOf(l archdoc.Layout, id string) archdoc.Rect {
	for _, b := range l.Boxes {
		if b.ID == id {
			return b.Rect
		}
	}
	return archdoc.Rect{}
}

// Dragging a box and saving writes layout.yaml, and every later scene — the app, the SVG — draws
// it where it was placed. Resetting brings the engine's layout back.
func TestAnArrangementIsSavedAndApplied(t *testing.T) {
	ts, root, token := live(t)
	before := scene(t, ts)
	engine := boxOf(before.Layout, "svc:api")

	code, b := post(t, ts, "/api/layout", token, map[string]any{
		"view": "container", "positions": map[string]any{"svc:api": map[string]float64{"x": 900, "y": 40}}, "base": before.Arrangement.Hash,
	})
	if code != 200 {
		t.Fatalf("save: %d %s", code, b)
	}
	if _, err := os.Stat(filepath.Join(root, arrange.Dir, arrange.LayoutFile)); err != nil {
		t.Fatal("layout.yaml not written")
	}
	after := scene(t, ts)
	if r := boxOf(after.Layout, "svc:api"); r.X != 900 || r.Y != 40 {
		t.Errorf("not applied: %+v", r)
	}
	if len(after.Arrangement.Placed) != 1 || after.Arrangement.Placed[0] != "svc:api" {
		t.Errorf("placed: %v", after.Arrangement.Placed)
	}

	if code, _ := post(t, ts, "/api/layout/reset", token, map[string]any{"view": "container", "base": after.Arrangement.Hash}); code != 200 {
		t.Fatalf("reset: %d", code)
	}
	if r := boxOf(scene(t, ts).Layout, "svc:api"); r != engine {
		t.Errorf("reset did not restore the engine's position: %+v, want %+v", r, engine)
	}
}

// A save made against a file that changed since is refused (ANS-04), and an arrangement cannot
// place something that is not in the view.
func TestStaleOrInventedSavesAreRefused(t *testing.T) {
	ts, _, token := live(t)
	sc := scene(t, ts)
	if code, _ := post(t, ts, "/api/layout", token, map[string]any{
		"view": "container", "positions": map[string]any{"svc:api": map[string]float64{"x": 1, "y": 1}}, "base": "not-what-is-on-disk",
	}); code != http.StatusConflict {
		t.Errorf("a stale save: %d, want 409", code)
	}
	if code, _ := post(t, ts, "/api/layout", token, map[string]any{
		"view": "container", "positions": map[string]any{"svc:invented": map[string]float64{"x": 1, "y": 1}}, "base": sc.Arrangement.Hash,
	}); code != http.StatusBadRequest {
		t.Errorf("placing an element that does not exist: %d, want 400", code)
	}
}

func TestViewsAreSavedByName(t *testing.T) {
	ts, _, token := live(t)
	code, b := post(t, ts, "/api/views/save", token, map[string]any{
		"view": map[string]any{"name": "API", "level": "container", "focus": "svc:api"}, "base": "",
	})
	if code != 200 {
		t.Fatalf("save view: %d %s", code, b)
	}
	_, b = get(t, ts, "/api/views")
	var got struct {
		Views []arrange.View `json:"views"`
	}
	json.Unmarshal(b, &got)
	if len(got.Views) != 1 || got.Views[0].Name != "API" || got.Views[0].Focus != "svc:api" {
		t.Errorf("views: %+v", got.Views)
	}
}
