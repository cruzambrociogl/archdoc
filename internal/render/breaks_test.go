package render

import (
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func breaksFixture() core.Model {
	code := func(file string, line int) core.Provenance { return core.Provenance{File: file, Line: line} }
	m := core.Model{
		Name: "photos", Source: "docker-compose.yml",
		Nodes: []core.Node{
			{ID: "actor:user", Name: "User", Kind: core.Actor, Evidence: core.Declared, Prov: at(1)},
			{ID: "app:web", Name: "web", Kind: core.Application, Evidence: core.Declared, Prov: code("web/package.json", 1)},
			{ID: "svc:api", Name: "api", Kind: core.Application, Evidence: core.Declared, Prov: at(10)},
			{ID: "svc:ml", Name: "ml", Kind: core.Application, Evidence: core.Declared, Prov: at(30)},
			{ID: "svc:db", Name: "db", Kind: core.Datastore, Technology: "PostgreSQL 16", Evidence: core.Declared, Prov: at(20)},
			{ID: "svc:cache", Name: "cache", Kind: core.Datastore, Technology: "Valkey 9", Evidence: core.Declared, Prov: at(25)},
			{ID: "tbl:api/album", Name: "album", Kind: core.Table, Parent: "svc:api", Evidence: core.Declared, Prov: code("api/album.table.ts", 1)},
		},
		Edges: []core.Edge{
			{From: "actor:user", To: "app:web", Label: "uses", Traffic: true, Prov: []core.Provenance{at(1)}},
			{From: "app:web", To: "svc:api", Label: "calls the API of", Technology: "http", Traffic: true, Prov: []core.Provenance{code("web/api.ts", 3)}},
			{From: "svc:api", To: "svc:db", Label: "connects to", Traffic: true, Prov: []core.Provenance{at(12)}},
			{From: "svc:api", To: "svc:cache", Label: "connects to", Traffic: true, Prov: []core.Provenance{at(13)}},
			{From: "svc:api", To: "svc:ml", Label: "calls", Technology: "http", Traffic: true, Prov: []core.Provenance{at(14)}},
		},
		Entries: []core.Entry{
			{ID: "route:api GET /albums", Kind: "http", Method: "GET", Path: "/albums", Container: "svc:api", Prov: code("api/albums.ts", 5)},
			{ID: "route:api POST /upload", Kind: "http", Method: "POST", Path: "/upload", Container: "svc:api", Prov: code("api/upload.ts", 5)},
			{ID: "route:api GET /health", Kind: "http", Method: "GET", Path: "/health", Container: "svc:api", Prov: code("api/health.ts", 5)},
			{ID: "command:api reset", Kind: "command", Method: "CMD", Path: "reset", Container: "svc:api", Prov: code("api/reset.ts", 5)},
			{ID: "job:api Thumbs", Kind: "job", Method: "JOB", Path: "Thumbs", Container: "svc:api", Prov: code("api/thumbs.ts", 5)},
		},
	}
	table := core.Participant{ID: "table:album", Name: "album", Kind: "table", Element: "tbl:api/album"}
	svc := core.Participant{ID: "Service", Name: "Service", Kind: "class"}
	job := core.Participant{ID: "job:Thumbs", Name: "Thumbs", Kind: "job", Element: "job:api Thumbs"}
	m.Flows = []core.Flow{
		{Entry: "route:api GET /albums", Participants: []core.Participant{svc, table}, Steps: []core.Step{{From: "Service", To: "table:album", Call: "reads", Prov: code("api/repo.ts", 10)}}},
		{Entry: "route:api POST /upload", Participants: []core.Participant{svc, table, job}, Steps: []core.Step{
			{From: "Service", To: "table:album", Call: "writes", Prov: code("api/repo.ts", 20)},
			{From: "Service", To: "job:Thumbs", Call: "queues", Prov: code("api/upload.ts", 9)}}},
		{Entry: "route:api GET /health", Participants: []core.Participant{svc}, Steps: []core.Step{}},
		{Entry: "job:api Thumbs", Participants: []core.Participant{svc, table}, Steps: []core.Step{{From: "Service", To: "table:album", Call: "updates", Prov: code("api/thumbs.ts", 12)}}},
	}
	return m.Normalise()
}

func stops(d Dependent) string {
	var out []string
	for _, e := range d.Stops {
		out = append(out, e.Entry+" "+e.Why+" @"+e.Cite.String())
	}
	return strings.Join(out, "; ")
}

// The database down: every way in that reads or writes its tables stops, at the query; one that
// reaches no table does not; one not followed is counted, not guessed. The web app and its user reach
// it only through the API.
func TestWhatBreaksWithoutTheDatabase(t *testing.T) {
	b, ok := Breaks(breaksFixture(), "svc:db")
	if !ok || len(b.Dependents) != 1 {
		t.Fatalf("%+v", b)
	}
	d := b.Dependents[0]
	if d.ID != "svc:api" || d.How != "reads and writes its tables" || !d.Known {
		t.Errorf("%+v", d)
	}
	if got := stops(d); got != "job:api Thumbs updates album @api/thumbs.ts:12; route:api GET /albums reads album @api/repo.ts:10; route:api POST /upload writes album @api/repo.ts:20" {
		t.Errorf("stops: %s", got)
	}
	if d.Unaffected != 1 || d.Unfollowed != 1 {
		t.Errorf("unaffected %d, unfollowed %d", d.Unaffected, d.Unfollowed)
	}
	var through []string
	for _, r := range b.Through {
		through = append(through, r.ID+" via "+r.Via)
	}
	if got := strings.Join(through, ", "); got != "actor:user via svc:api, app:web via svc:api" {
		t.Errorf("through: %s", got)
	}
}

// The queue store down: a way in that queues a job stops at the queue, and every background job,
// taken from it, stops; the rest is untouched.
func TestWhatBreaksWithoutTheQueue(t *testing.T) {
	b, _ := Breaks(breaksFixture(), "svc:cache")
	d := b.Dependents[0]
	if d.How != "queues its jobs on it" || !d.Known {
		t.Errorf("%+v", d)
	}
	if got := stops(d); got != "job:api Thumbs is taken from the queue @api/thumbs.ts:5; route:api POST /upload queues Thumbs @api/upload.ts:9" {
		t.Errorf("stops: %s", got)
	}
	if d.Unaffected != 2 || d.Unfollowed != 1 {
		t.Errorf("unaffected %d, unfollowed %d", d.Unaffected, d.Unfollowed)
	}
}

// A service called over HTTP at an address set at run time: the caller waits for the answer, but
// which of its ways in make the call is not recorded — said, not guessed.
func TestWhatBreaksWithoutACalledServiceIsNotGuessed(t *testing.T) {
	b, _ := Breaks(breaksFixture(), "svc:ml")
	d := b.Dependents[0]
	if d.How != "waits for the answer" || d.Known || len(d.Stops) != 0 {
		t.Errorf("%+v", d)
	}
}

// Two relational stores: which holds the tables is not known, so nothing is said to stop.
func TestTwoDatabasesTieNothing(t *testing.T) {
	m := breaksFixture()
	m.Nodes = append(m.Nodes, core.Node{ID: "svc:db2", Name: "db2", Kind: core.Datastore, Technology: "MySQL 8", Evidence: core.Declared, Prov: at(40)})
	m.Edges = append(m.Edges, core.Edge{From: "svc:api", To: "svc:db2", Label: "connects to", Traffic: true, Prov: []core.Provenance{at(41)}})
	b, _ := Breaks(m.Normalise(), "svc:db")
	if d := b.Dependents[0]; d.Known || len(d.Stops) != 0 {
		t.Errorf("%+v", d)
	}
}

func TestWhatBreaksAsksOnlyAboutParts(t *testing.T) {
	m := breaksFixture()
	for _, id := range []string{"actor:user", "tbl:api/album", "nothing"} {
		if _, ok := Breaks(m, id); ok {
			t.Errorf("%s is not a part to ask about", id)
		}
	}
	b, _ := Breaks(m, "actor:user")
	_ = b
	a, _ := Breaks(m, "svc:api")
	if len(a.Dependents) != 1 || a.Dependents[0].ID != "app:web" || a.Dependents[0].How != "waits for the answer" {
		t.Errorf("the api's dependents: %+v", a.Dependents)
	}
}
