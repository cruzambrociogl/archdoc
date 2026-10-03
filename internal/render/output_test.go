package render

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// fixtureFacts is what extraction would have produced for fixture(): the deployment detail the
// container view drops on the floor, which §7 is supposed to show.
func fixtureFacts() archdoc.FactSet {
	return archdoc.FactSet{
		Root:   "/tmp/example",
		Name:   "example",
		Source: "docker-compose.yml",
		Considered: []archdoc.Candidate{
			{File: "docker-compose.yml", Reason: "deployable — 4 services with an image or build", Chosen: true},
			{File: "docker-compose.override.yml", Reason: "development override", Chosen: false},
		},
		Services: []archdoc.Service{
			{Name: "api", Evidence: archdoc.Declared, Prov: at(10),
				Ports:  []archdoc.Port{{Published: "8080", Target: 3000, Prov: at(11)}},
				Mounts: []archdoc.Mount{{Source: "./config/api.yml", Target: "/etc/api.yml", Prov: at(14)}},
				Endpoints: []archdoc.Endpoint{
					{Var: "S3_ENDPOINT", Scheme: "https", Host: "s3.amazonaws.com",
						Prov: archdoc.Provenance{File: ".env", Line: 3}},
				}},
			{Name: "db", Image: "postgres:16", Evidence: archdoc.Declared, Prov: at(20)},
			{Name: "gateway", Image: "nginx:1.27", Evidence: archdoc.Declared, Prov: at(3),
				Mounts: []archdoc.Mount{{Source: "./nginx.conf", Target: "/etc/nginx/nginx.conf", Prov: at(5)}}},
		},
		Networks: []archdoc.Network{{Name: "backend", Internal: true, Prov: at(40)}},
		Routes:   []archdoc.Route{{Gateway: "gateway", Target: "api", Path: "/api", Config: "nginx.conf", Prov: at(7)}},
	}
}

// The deployment view is where Compose belongs: a compose file is a deployment descriptor, and
// everything below was extracted and then suppressed because §5 must not show deployment concepts.
func TestDeploymentViewShowsWhatComposeActuallyStates(t *testing.T) {
	out := Arc42(fixture(), fixtureFacts(), Sections(), meta())["07-deployment-view.generated.md"]

	for _, want := range []string{
		"postgres:16",                // the image a container runs
		"built from this repository", // and the fact that one is built, not pulled
		"8080",                       // published on the host
		"3000",                       // seen inside the container
		"./nginx.conf",               // what the repository mounts in
		"/etc/nginx/nginx.conf",      // and where the container sees it
		"docker-compose.yml:11",      // every row cites its line
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the deployment view does not state %q", want)
		}
	}
}

func TestDeploymentViewSaysSoWhenNothingIsPublished(t *testing.T) {
	facts := fixtureFacts()
	for i := range facts.Services {
		facts.Services[i].Ports = nil
	}

	out := Arc42(fixture(), facts, Sections(), meta())["07-deployment-view.generated.md"]
	if !strings.Contains(out, "No port is published to the host") {
		t.Error("a system with no published ports should say so, not show an empty table")
	}
}

// A single-service repository has no deployment story and no relationships. Twelve sections for it
// would be a filing cabinet for a postcard, and four of the stubs would ask about structure it
// does not have.
func TestSmallProjectGetsASmallDocumentSet(t *testing.T) {
	m := archdoc.Model{
		Name:   "script",
		Source: "docker-compose.yml",
		Nodes: []archdoc.Node{
			{ID: "svc:worker", Name: "worker", Kind: archdoc.Application, Evidence: archdoc.Declared, Prov: at(2)},
		},
	}.Normalise()
	facts := archdoc.FactSet{Name: "script", Source: "docker-compose.yml",
		Services: []archdoc.Service{{Name: "worker", Evidence: archdoc.Declared, Prov: at(2)}}}

	plan := PlanFor(m, facts)
	for _, gone := range []int{2, 7, 8, 10, 11} {
		if Planned(plan, gone) {
			t.Errorf("§%d was emitted for a repository with nothing to put in it", gone)
		}
	}
	for _, kept := range []int{1, 3, 4, 5, 9, 12} {
		if !Planned(plan, kept) {
			t.Errorf("§%d is answerable by any project and must be kept", kept)
		}
	}

	// And the index must say the omission was a decision, not an oversight.
	idx := Index(m, plan, meta())
	if !strings.Contains(idx, "are not here, because this repository states") {
		t.Error("the index does not explain the sections it leaves out")
	}
}

func TestRealSystemKeepsAllTwelveSections(t *testing.T) {
	plan := PlanFor(fixture(), fixtureFacts())
	if len(plan) != 12 {
		t.Errorf("got %d sections for a system with networks, ports and relationships, want 12", len(plan))
	}
}

// The coverage report is the page no competitor publishes: what was read, what could not be
// resolved, and what the evidence cannot state at all.
func TestCoverageShowsTheEdgeOfTheMap(t *testing.T) {
	gaps := []Gap{
		{Rule: "VAL-03", Element: "svc:api → svc:db", Message: "no protocol"},
		{Rule: "VAL-06", Element: "svc:api", Message: "no description"},
	}
	out := Coverage(fixture(), fixtureFacts(), gaps, meta())

	for _, want := range []string{
		"docker-compose.override.yml", // a file that was looked at and skipped
		"development override",        // and why
		"nginx.conf",                  // the gateway config routes came from
		".env",                        // the env file endpoints were read from
		"VAL-03",                      // the gaps, grouped by rule
		"no protocol",
		"What configuration cannot state", // and the method's own limits
		"Application source",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the coverage report does not mention %q", want)
		}
	}
}

func TestCoverageReportsACleanModelHonestly(t *testing.T) {
	out := Coverage(fixture(), fixtureFacts(), nil, meta())
	if !strings.Contains(out, "Nothing. Every element and relationship") {
		t.Error("with no gaps the report should say so plainly")
	}
}

// AC-7: two runs on identical input produce identical bytes. Map iteration is the standing threat.
func TestCoverageIsDeterministic(t *testing.T) {
	gaps := []Gap{
		{Rule: "VAL-06", Element: "svc:db", Message: "no description"},
		{Rule: "VAL-03", Element: "svc:api → svc:db", Message: "no protocol"},
		{Rule: "VAL-06", Element: "svc:api", Message: "no description"},
	}
	first := Coverage(fixture(), fixtureFacts(), gaps, meta())
	for range 10 {
		if Coverage(fixture(), fixtureFacts(), gaps, meta()) != first {
			t.Fatal("the coverage report differs between identical runs")
		}
	}
}

// Layer 3 of the output contract: the site must list exactly the sections that exist, or MkDocs
// fails the build on a missing file.
func TestMkDocsNavMatchesTheDocumentSet(t *testing.T) {
	m := fixture()
	plan := PlanFor(m, fixtureFacts())
	out := MkDocs(m, plan)

	if !strings.Contains(out, "site_name: example — architecture") {
		t.Errorf("site name missing: %.80s", out)
	}
	if !strings.Contains(out, "docs_dir: .") || !strings.Contains(out, "name: material") {
		t.Error("the site would not build from the documentation directory")
	}
	if !strings.Contains(out, "format: !!python/name:pymdownx.superfences.fence_code_format") {
		t.Error("without the Mermaid fence the diagrams render as code blocks")
	}
	for _, s := range plan {
		if !strings.Contains(out, s.File()) {
			t.Errorf("nav does not list %s", s.File())
		}
	}
	if !strings.Contains(out, IndexFile) || !strings.Contains(out, CoverageFile) {
		t.Error("nav must carry the overview and the coverage report")
	}

	// A section not in the plan must not be in the nav: MkDocs fails on a missing file.
	short := []Section{Sections()[0]}
	if nav := MkDocs(m, short); strings.Contains(nav, "07-deployment-view") {
		t.Error("nav lists a section the plan does not include")
	}
}

// Project names come from other people's repositories, so they are not trusted to be plain YAML.
func TestMkDocsQuotesAwkwardNames(t *testing.T) {
	m := fixture()
	m.Name = "my: project #2"
	if out := MkDocs(m, Sections()); !strings.Contains(out, `site_name: "my: project #2" — architecture`) &&
		!strings.Contains(out, `"my: project #2"`) {
		t.Errorf("an awkward name was not quoted: %.80s", out)
	}
}
