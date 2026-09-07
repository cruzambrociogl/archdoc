package extract

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

const netFixture = "../../testdata/networks"

func scanNetworks(t *testing.T) *archdoc.FactSet {
	t.Helper()

	fs, err := Scan(netFixture)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return fs
}

// Network membership is the one reachability claim configuration states rather than implies:
// two services sharing no network cannot reach each other.
func TestServiceNetworkMembership(t *testing.T) {
	fs := scanNetworks(t)

	want := map[string][]string{
		"web":    {"backend", "frontend"},
		"worker": {"backend", "frontend"},
		"db":     {"backend"},
		"orphan": nil,
	}

	for _, s := range fs.Services {
		got := make([]string, 0, len(s.Networks))
		for _, n := range s.Networks {
			got = append(got, n.Name)
			if !n.Prov.Known() {
				t.Errorf("%s/%s membership has no provenance", s.Name, n.Name)
			}
		}
		if strings.Join(got, ",") != strings.Join(want[s.Name], ",") {
			t.Errorf("%s is on %v, want %v", s.Name, got, want[s.Name])
		}
	}
}

// Compose's own `internal: true` is the one trust boundary a file states outright — a network
// with no route out. MDL-11 records declared boundaries and infers none, and this is what
// declared looks like.
func TestInternalNetworkIsRecordedAndCited(t *testing.T) {
	fs := scanNetworks(t)

	if len(fs.Networks) != 2 {
		t.Fatalf("got %d networks, want 2", len(fs.Networks))
	}

	for _, n := range fs.Networks {
		if !n.Prov.Known() {
			t.Errorf("%s has no provenance", n.Name)
		}
		switch n.Name {
		case "backend":
			if !n.Internal {
				t.Error("backend declares internal: true and was not recorded as internal")
			}
		case "frontend":
			if n.Internal {
				t.Error("frontend declares nothing and was marked internal")
			}
		}
	}
}

// EXT-07. A value set in a dotenv file must cite that file, not the compose file that named it.
func TestEnvFileValuesCiteTheDotenvFile(t *testing.T) {
	worker := scanFixture(t)["worker"]

	for _, e := range worker.Endpoints {
		if e.Var != "QUEUE_URL" {
			continue
		}
		if e.Host != "broker.example.com" || e.Scheme != "amqp" {
			t.Errorf("got %+v, want amqp://broker.example.com", e)
		}
		if e.Prov.File != "worker.env" || e.Prov.Line != 3 {
			t.Errorf("cited %s, want worker.env:3", e.Prov)
		}
		return
	}
	t.Error("QUEUE_URL was not extracted from the env_file")
}

// Compose's precedence: an inline environment value overrides the env_file that also sets it,
// and the citation must follow the value that actually won.
func TestInlineEnvironmentOverridesEnvFile(t *testing.T) {
	worker := scanFixture(t)["worker"]

	for _, e := range worker.Endpoints {
		if e.Var != "SEARCH_HOST" {
			continue
		}
		if e.Host != "overridden-by-compose" {
			t.Errorf("host is %q — env_file won a value the compose file overrides", e.Host)
		}
		if e.Prov.File == "worker.env" {
			t.Error("cited the losing file")
		}
		// The sibling port still comes from the dotenv file: one environment, two sources.
		if e.Port != 9200 {
			t.Errorf("port is %d, want 9200 from SEARCH_PORT in worker.env", e.Port)
		}
		return
	}
	t.Error("SEARCH_HOST was not extracted")
}

// A dotenv file is exactly where credentials live, and it is now read. Nothing from it may reach
// the FactSet except parsed locations.
func TestEnvFileSecretsAreDiscarded(t *testing.T) {
	fs, err := Scan(factsFixture)
	if err != nil {
		t.Fatal(err)
	}

	b, err := json.Marshal(fs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "hunter2") {
		t.Error("a secret from an env_file reached the FactSet")
	}
}
