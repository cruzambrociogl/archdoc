package serve

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/store"
)

func TestStaticNames(t *testing.T) {
	cases := map[string]string{
		"/api/summary":                        "data/summary.json",
		"/api/scene?view=container&version=3": "data/scene@version=3,view=container.json",
		"/api/scene?version=3&view=container": "data/scene@version=3,view=container.json",
		"/api/svg?view=context":               "data/svg@view=context.svg",
		"/api/docs/index.generated.md":        "data/docs/index.generated.md",
		"/api/diff?from=1&to=2":               "data/diff@from=1,to=2.json",
	}
	for in, want := range cases {
		if got := StaticName(in); got != want {
			t.Errorf("StaticName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The published site carries everything the app reads for the latest version and its baseline,
// says it is published, and nothing in it names a path on the publisher's machine.
func TestExportIsCompleteAndCarriesNoLocalPath(t *testing.T) {
	root := repo(t)
	files, err := Export(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"data/session.json", "data/summary.json", "data/versions.json", "data/model.json",
		"data/scene@view=container.json", "data/scene@view=context.json", "data/svg@view=container.svg",
		"data/diff@from=1,to=2.json", "data/scene@version=1,view=container.json",
		"data/model@version=1.json", "data/model@version=2.json", "data/docs.json",
		"data/docs/index.generated.md", "data/completeness.json", "data/questions@section=1.json",
		"data/rules.json", "data/views.json", "data/runs.json",
	} {
		if _, ok := files[want]; !ok {
			t.Errorf("the published site has no %s", want)
		}
	}
	var sess struct {
		Mode string `json:"mode"`
	}
	json.Unmarshal(files["data/session.json"], &sess)
	if sess.Mode != "published" {
		t.Errorf("session mode %q", sess.Mode)
	}
	for name, b := range files {
		if bytes.Contains(b, []byte(root)) {
			t.Errorf("%s names the publisher's machine (%s)", name, root)
		}
	}
	if _, ok := files["data/runs_1.json"]; ok {
		t.Error("a run's payload was published; only the summary should travel")
	}
}

// A fresh clone in CI has no history.db — it is never committed — but does have the committed
// model.json. The site is built from that, as one version, with nothing regenerated.
func TestExportFromTheCommittedRecordAlone(t *testing.T) {
	root := repo(t)
	h, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := h.Latest()
	h.Close()
	b, _ := json.Marshal(v.Model)
	os.WriteFile(filepath.Join(root, ".archdoc", "model.json"), b, 0o644)
	os.Remove(filepath.Join(root, store.File))

	files, err := Export(root, 0)
	if err != nil {
		t.Fatalf("export without history: %v", err)
	}
	var sum struct {
		Versions int `json:"versions"`
	}
	json.Unmarshal(files["data/summary.json"], &sum)
	if sum.Versions != 1 {
		t.Errorf("built from model.json: %d versions, want 1", sum.Versions)
	}
	if _, ok := files["data/scene@view=container.json"]; !ok {
		t.Error("no scene in a site built from model.json")
	}

	// With neither, it says what is missing.
	os.Remove(filepath.Join(root, ".archdoc", "model.json"))
	if _, err := Export(root, 0); err == nil || !strings.Contains(err.Error(), "model.json") {
		t.Errorf("export with nothing to build from: %v", err)
	}
}
