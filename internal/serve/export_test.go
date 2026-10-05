package serve

import (
	"bytes"
	"encoding/json"
	"testing"
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
