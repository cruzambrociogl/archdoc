package system

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/store"
)

func cite(file string, line int) core.Provenance { return core.Provenance{File: file, Line: line} }

// archdoc's own shape: an engine that publishes a Go module and a web app, and a platform whose
// go.mod pins the engine at a commit and whose web app takes the engine's from the checkout beside it.
func workspace(t *testing.T) []Repo {
	dir := t.TempDir()
	engine := Repo{Name: "engine", Root: filepath.Join(dir, "engine"), Commit: "9a8b7c6d5e4f", Model: core.Model{Manifests: []core.Manifest{
		{Package: "github.com/acme/engine", Dir: ".", Language: "Go", Prov: cite("go.mod", 1)},
		{Package: "@acme/app", Dir: "web/app", Language: "JavaScript", Prov: cite("web/app/package.json", 2),
			Requires: []core.Requirement{{Name: "react", Version: "^19", Prov: cite("web/app/package.json", 9)}}},
		// A package of its own repository requiring another of its own is not a link between two.
		{Package: "@acme/host", Dir: "web/host", Language: "JavaScript", Prov: cite("web/host/package.json", 2),
			Requires: []core.Requirement{{Name: "@acme/app", Version: "*", Prov: cite("web/host/package.json", 13)}}},
	}}}
	platform := Repo{Name: "platform", Root: filepath.Join(dir, "platform"), Commit: "0f0f0f0f0f0f", Model: core.Model{Manifests: []core.Manifest{
		{Package: "github.com/acme/platform", Dir: ".", Language: "Go", Prov: cite("go.mod", 1), Requires: []core.Requirement{
			{Name: "github.com/acme/engine", Version: "v0.0.0-20261022155652-3203f69123a2", Prov: cite("go.mod", 6)},
			{Name: "github.com/jackc/pgx/v5", Version: "v5.7.6", Prov: cite("go.mod", 7)},
		}},
		{Package: "@acme/platform-web", Dir: "web", Language: "JavaScript", Prov: cite("web/package.json", 2), Requires: []core.Requirement{
			{Name: "@acme/app", Version: "file:../../engine/web/app", Prov: cite("web/package.json", 13)},
		}},
	}}}
	return []Repo{engine, platform}
}

func TestLinksAreTheRequirementsAnotherRepositoryPublishes(t *testing.T) {
	asked := ""
	ahead := func(root, from, to string) (int, error) {
		asked = filepath.Base(root) + " " + from + ".." + to
		return 2, nil
	}
	sys := Compose("acme", workspace(t), ahead)
	if len(sys.Links) != 2 {
		t.Fatalf("links: %+v", sys.Links)
	}
	app, mod := sys.Links[0], sys.Links[1]

	if mod.From != "platform" || mod.To != "engine" || mod.Package != "github.com/acme/engine" || mod.Uses != cite("go.mod", 6) || mod.Declares != cite("go.mod", 1) {
		t.Errorf("the module link: %+v", mod)
	}
	if mod.Pin == nil || *mod.Pin != (Pin{Kind: "commit", Ref: "3203f69123a2", Ahead: 2}) || asked != "engine 3203f69123a2..9a8b7c6d5e4f" {
		t.Errorf("the module's pin: %+v, git asked for %q", mod.Pin, asked)
	}
	if app.Package != "@acme/app" || app.Uses != cite("web/package.json", 13) || app.Declares != cite("web/app/package.json", 2) {
		t.Errorf("the web app link: %+v", app)
	}
	if app.Pin == nil || app.Pin.Kind != "checkout" {
		t.Errorf("a file: requirement that lands on the package is the checkout beside it: %+v", app.Pin)
	}
}

// What cannot be counted is not guessed: without git, the pin keeps its commit and no count.
func TestDriftUncountedIsUnknownNotZero(t *testing.T) {
	sys := Compose("acme", workspace(t), func(string, string, string) (int, error) { return 0, errors.New("no git") })
	if p := sys.Links[1].Pin; p == nil || p.Ahead != -1 {
		t.Errorf("pin without git: %+v", p)
	}
}

// A file: path that does not land on the package, or a range, names no code of the other's.
func TestPinsOnlyWhereTheVersionSaysWhich(t *testing.T) {
	to := Repo{Root: "/w/engine", Commit: "c"}
	for version, want := range map[string]string{
		"file:../../elsewhere/app": "",
		"^1.2.0":                   "",
		"v1.4.0":                   "tag",
		"2.0.0-rc.1":               "tag",
	} {
		p := pin(version, "/w/platform/web", "/w/engine/web/app", to, nil)
		if got := ""; p != nil {
			got = p.Kind
			if got != want {
				t.Errorf("%s: pinned as %q, want %q", version, got, want)
			}
		} else if want != "" {
			t.Errorf("%s: no pin, want %q", version, want)
		}
	}
}

// A system is the documented repositories directly inside a folder; a documented folder is one
// repository, and a folder with fewer than two is not a system.
func TestReadFindsTheDocumentedRepositoriesInAFolder(t *testing.T) {
	dir := t.TempDir()
	repos := workspace(t)
	for _, r := range repos {
		s, err := store.Open(filepath.Join(dir, r.Name))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.Save(r.Model, nil, r.Commit, "test"); err != nil {
			t.Fatal(err)
		}
		s.Close()
	}
	sys, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sys.Name != filepath.Base(dir) || len(sys.Repos) != 2 || sys.Repos[0].Name != "engine" || len(sys.Links) != 2 {
		t.Errorf("read: %s, %d repositories, %d links", sys.Name, len(sys.Repos), len(sys.Links))
	}
	if _, err := Read(filepath.Join(dir, "engine")); err == nil {
		t.Error("a documented repository was read as a system")
	}
	if _, err := Read(t.TempDir()); err == nil {
		t.Error("an empty folder was read as a system")
	}
}
