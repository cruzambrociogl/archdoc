package extract

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

const factsFixture = "../../testdata/endpoints"

func scanFixture(t *testing.T) map[string]archdoc.Service {
	t.Helper()

	fs, err := Scan(factsFixture)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	out := map[string]archdoc.Service{}
	for _, s := range fs.Services {
		out[s.Name] = s
	}
	return out
}

func TestDependenciesCarryTheirOwnLine(t *testing.T) {
	api := scanFixture(t)["api"]

	want := []struct {
		service string
		line    int
	}{
		{"cache", 16},
		{"db", 14},
	}

	if len(api.DependsOn) != len(want) {
		t.Fatalf("got %d dependencies, want %d", len(api.DependsOn), len(want))
	}

	for i, w := range want {
		got := api.DependsOn[i]
		if got.Service != w.service {
			t.Errorf("dependency %d is %q, want %q", i, got.Service, w.service)
		}
		// The line of the dependency itself, not of the service that declares it. A reader
		// following the citation should land on the entry they are being told about.
		if got.Prov.Line != w.line {
			t.Errorf("%s cited at line %d, want %d", w.service, got.Prov.Line, w.line)
		}
	}
}

// An unpublished port is internal plumbing. Only a published one is evidence that something
// outside the system can reach in.
func TestOnlyPublishedPortsAreFacts(t *testing.T) {
	api := scanFixture(t)["api"]

	if len(api.Ports) != 1 {
		t.Fatalf("got %d ports, want 1 — '9000' publishes nothing", len(api.Ports))
	}
	if got := api.Ports[0]; got.Published != "8080" || got.Target != 80 {
		t.Errorf("got %s -> %d, want 8080 -> 80", got.Published, got.Target)
	}
}

func TestEndpointsFromURLsAndHostVariables(t *testing.T) {
	api := scanFixture(t)["api"]

	want := []archdoc.Endpoint{
		{Var: "CACHE_HOST", Host: "cache", Port: 6379},
		{Var: "DATABASE_URL", Scheme: "postgres", Host: "db", Port: 5432},
		{Var: "S3_ENDPOINT", Scheme: "https", Host: "s3.example.com"},
		{Var: "SMTP_HOST", Host: "mail.example.com"},
	}

	if len(api.Endpoints) != len(want) {
		t.Fatalf("got %d endpoints, want %d: %+v", len(api.Endpoints), len(want), api.Endpoints)
	}

	for i, w := range want {
		got := api.Endpoints[i]
		if got.Var != w.Var || got.Scheme != w.Scheme || got.Host != w.Host || got.Port != w.Port {
			t.Errorf("endpoint %d: got %+v, want %+v", i, got, w)
		}
		if !got.Prov.Known() {
			t.Errorf("%s has no provenance", got.Var)
		}
	}
}

// SMTP_HOST is written ${SMTP_HOST} in the file and resolved from .env. This is the whole point
// of the two-pass design: the value comes from the first pass, the line from the second, and
// the citation still lands on text a reader can open.
func TestInterpolatedValueKeepsItsSourceLine(t *testing.T) {
	api := scanFixture(t)["api"]

	for _, e := range api.Endpoints {
		if e.Var != "SMTP_HOST" {
			continue
		}
		if e.Host != "mail.example.com" {
			t.Errorf("host is %q — interpolation did not run", e.Host)
		}
		if e.Prov.Line != 23 {
			t.Errorf("cited at line %d, want 23 — the line holding ${SMTP_HOST}", e.Prov.Line)
		}
		return
	}
	t.Error("SMTP_HOST not extracted")
}

// Four values in the fixture look like hosts and are not. Each was found in a real subject
// first: a socket path and a bind address in Supabase, a flag whose name ends in HOST, and a
// localhost URL that names the reader's own machine.
func TestValuesThatLookLikeHostsAndAreNot(t *testing.T) {
	api := scanFixture(t)["api"]

	for _, e := range api.Endpoints {
		switch e.Var {
		case "SELF_HOST", "SOCKET_HOST", "BIND_HOSTNAME", "PUBLIC_URL":
			t.Errorf("%s should not be an endpoint, got host %q", e.Var, e.Host)
		}
	}
}

// The list form of environment is as common as the mapping form — Mastodon's database uses it.
// It must not be mistaken for anything else.
func TestListFormEnvironmentYieldsNoFalseEndpoints(t *testing.T) {
	db := scanFixture(t)["db"]

	if len(db.Endpoints) != 0 {
		t.Errorf("POSTGRES_HOST_AUTH_METHOD is not a host: %+v", db.Endpoints)
	}
}

// The FactSet reaches disk. Compose environments hold credentials, so extraction reads them and
// keeps only parsed locations — there is nothing downstream to redact.
func TestSecretsNeverReachTheFactSet(t *testing.T) {
	fs, err := Scan(factsFixture)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	b, err := json.Marshal(fs)
	if err != nil {
		t.Fatal(err)
	}

	// hunter2 is the password in DATABASE_URL, in POSTGRES_PASSWORD, and in SECRET_KEY.
	if strings.Contains(string(b), "hunter2") {
		t.Error("a secret from the environment reached the FactSet")
	}
}
