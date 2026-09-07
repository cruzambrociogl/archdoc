package extract

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

const gwFixture = "../../testdata/gateway"

func scanGateway(t *testing.T) *archdoc.FactSet {
	t.Helper()

	fs, err := Scan(gwFixture)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return fs
}

// MDL-03. A gateway's routing table says where traffic actually goes, which depends_on never
// does — and the citation lands in the gateway's own file, not the compose file that mounted it.
func TestRoutesComeFromTheMountedConfig(t *testing.T) {
	fs := scanGateway(t)

	want := map[string]int{"api": 11, "admin-console.internal": 22}

	if len(fs.Routes) != len(want) {
		t.Fatalf("got %d routes, want %d: %+v", len(fs.Routes), len(want), fs.Routes)
	}

	for _, r := range fs.Routes {
		line, ok := want[r.Target]
		if !ok {
			t.Errorf("unexpected route to %s", r.Target)
			continue
		}
		if r.Gateway != "edge" {
			t.Errorf("route to %s attributed to %q, want edge", r.Target, r.Gateway)
		}
		if r.Prov.File != "conf/cds.yaml" || r.Prov.Line != line {
			t.Errorf("route to %s cited %s, want conf/cds.yaml:%d", r.Target, r.Prov, line)
		}
	}
}

// A listener's own bind address is not a route to anything. The same document holds both, so
// the two have to be told apart by resource type rather than by shape.
func TestListenerBindAddressIsNotARoute(t *testing.T) {
	for _, r := range scanGateway(t).Routes {
		if r.Target == "0.0.0.0" {
			t.Error("a listener's bind address was read as a route")
		}
	}
}

// Most mounts are data, not configuration. Reading them must be free of side effects — the
// fixture mounts a .sql file next to the routing table for exactly this reason.
func TestNonConfigMountsAreIgnored(t *testing.T) {
	for _, r := range scanGateway(t).Routes {
		if r.Config != "conf/cds.yaml" {
			t.Errorf("read routes from %s, which is not a routing table", r.Config)
		}
	}
}

// A gateway names its upstreams by hostname, and the hostname is often not the service key.
// Supabase routes to realtime-dev.supabase-realtime, which is that service's container_name.
func TestAliasesAreExtracted(t *testing.T) {
	fs := scanGateway(t)

	want := map[string]string{"edge": "edge-proxy", "admin": "admin-console.internal"}

	for _, s := range fs.Services {
		alias, expected := want[s.Name]
		if !expected {
			continue
		}
		found := false
		for _, a := range s.Aliases {
			found = found || a == alias
		}
		if !found {
			t.Errorf("%s does not answer to %q: %v", s.Name, alias, s.Aliases)
		}
	}
}
