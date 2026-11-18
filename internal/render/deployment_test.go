package render

import (
	"context"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func ids(m core.Model) string {
	var out []string
	for _, n := range m.Nodes {
		out = append(out, n.ID)
	}
	return strings.Join(out, ",")
}

// Where it runs: the Compose file is the one deployment node, holding what it runs; the outside
// systems those services reach are drawn outside it, and the people who use the system are not —
// a deployment diagram is about where things run.
func TestDeploymentViewIsWhatTheComposeFileRuns(t *testing.T) {
	m := fixture()
	if !HasDeployment(m) {
		t.Fatal("a Compose file that runs services has a deployment view")
	}
	v := deploymentView(m)
	if got := ids(v.Model); got != "svc:api,svc:db,ext:s3.amazonaws.com" {
		t.Errorf("drawn: %s", got)
	}
	if v.Model.Boundary != "docker-compose.yml [Deployment Node: Docker Compose]" {
		t.Errorf("boundary %q", v.Model.Boundary)
	}
	if !v.Group || v.Name != Deployment || v.File != "deployment" {
		t.Errorf("view %+v", v)
	}
	var edges []string
	for _, e := range v.Model.Edges {
		edges = append(edges, e.From+">"+e.To)
	}
	if got := strings.Join(edges, ","); got != "svc:api>ext:s3.amazonaws.com,svc:api>svc:db" {
		t.Errorf("arrows: %s", got)
	}
}

// An application its own manifest declares is not run by the Compose file, whatever it is called;
// and a project read from a manifest alone has no deployment view at all.
func TestDeploymentLeavesOutWhatComposeDoesNotRun(t *testing.T) {
	m := fixture()
	m.Nodes = append(m.Nodes, core.Node{ID: "app:mobile", Name: "mobile", Kind: core.Application, Evidence: core.Declared, Prov: core.Provenance{File: "mobile/pubspec.yaml", Line: 1}})
	m.Edges = append(m.Edges, core.Edge{From: "app:mobile", To: "svc:api", Label: "calls", Traffic: true, Prov: []core.Provenance{{File: "mobile/lib/api.dart", Line: 3}}})
	m = m.Normalise()
	if got := ids(deploymentView(m).Model); strings.Contains(got, "app:mobile") {
		t.Errorf("the mobile app is drawn as run by Compose: %s", got)
	}

	m.Source = "package.json"
	if HasDeployment(m) {
		t.Error("a project read from package.json has no Compose file to draw")
	}
	if _, ok := ViewOf(m, Deployment); ok {
		t.Error("ViewOf gives a deployment view the model does not have")
	}
}

func TestComposeFileNames(t *testing.T) {
	for path, want := range map[string]bool{
		"docker-compose.yml": true, "docker/docker-compose.yml": true, "compose.yaml": true, "docker-compose.prod.yml": true,
		"package.json": false, "compose.json": false, "pubspec.yaml": false,
	} {
		if composeFile(path) != want {
			t.Errorf("%s: %v", path, !want)
		}
	}
}

// The view is listed last, opens by name, and lays out the same way every time (AC-7).
func TestDeploymentIsListedAndDrawnTheSameEachTime(t *testing.T) {
	m := fixture()
	vs := Views(m)
	if last := vs[len(vs)-1]; last.Name != Deployment {
		t.Errorf("last view is %s", last.Name)
	}
	v, ok := ViewOf(m, Deployment)
	if !ok {
		t.Fatal("no deployment view by name")
	}
	l1, err := Layout(context.Background(), v.Model, v.Group)
	if err != nil {
		t.Fatal(err)
	}
	l2, _ := Layout(context.Background(), v.Model, v.Group)
	a, b := SVG(v.Model, l1), SVG(v.Model, l2)
	if a != b {
		t.Error("the same model drew two different deployment diagrams")
	}
	if !strings.Contains(a, "Deployment Node: Docker Compose") {
		t.Error("the deployment node is not named on the diagram")
	}
}
