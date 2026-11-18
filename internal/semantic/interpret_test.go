package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// photos is a small system with everything the seeds join on: an audit table, a container with
// enough components to need areas and a part they all use, routes for users and for administrators,
// a page that calls routes across the API, jobs that queue jobs, and a command.
func photos() core.Model {
	p := func(f string, l int) core.Provenance { return core.Provenance{File: f, Line: l} }
	m := core.Model{Name: "photos", Description: "Keeps your photos."}
	m.Nodes = []core.Node{
		{ID: "svc:server", Name: "server", Kind: core.Application, Technology: "NestJS"},
		{ID: "app:web", Name: "web", Kind: core.Application, Technology: "SvelteKit"},
		{ID: "svc:db", Name: "db", Kind: core.Datastore, Technology: "PostgreSQL"},
		{ID: "table:asset", Name: "asset", Kind: core.Table, Parent: "svc:db", Columns: []core.Column{{Name: "id", Primary: true}}},
		{ID: "table:asset_audit", Name: "asset_audit", Kind: core.Table, Parent: "svc:db", Columns: []core.Column{{Name: "asset_id", References: "table:asset"}}},
		{ID: "table:album", Name: "album", Kind: core.Table, Parent: "svc:db", Columns: []core.Column{{Name: "cover", References: "table:asset"}}},
		{ID: "cmp:web/routes", Name: "routes", Kind: core.Component, Parent: "app:web", Files: []string{"web/src/routes/albums/+page.svelte"}},
	}
	parts := []string{"asset", "album", "user", "face", "utils"}
	for i := 5; i < 20; i++ {
		parts = append(parts, fmt.Sprintf("part%02d", i))
	}
	for _, n := range parts {
		m.Nodes = append(m.Nodes, core.Node{ID: "cmp:server/" + n, Name: n, Kind: core.Component, Parent: "svc:server",
			Files: []string{"server/src/" + n + "/" + n + ".ts"}})
	}
	m.Edges = []core.Edge{
		{From: "app:web", To: "svc:server", Label: callsAPI},
		{From: "svc:server", To: "svc:db", Label: "reads from and writes to"},
	}
	for _, n := range parts {
		if n != "utils" {
			m.Edges = append(m.Edges, core.Edge{From: "cmp:server/" + n, To: "cmp:server/utils", Label: "uses", Weight: 2})
		}
	}
	route := func(method, path, comp, handler string) core.Entry {
		return core.Entry{ID: "route:server " + method + " " + path, Kind: "http", Method: method, Path: path, Handler: handler,
			Container: "svc:server", Component: "cmp:server/" + comp, Prov: p("server/src/"+comp+"/"+comp+".controller.ts", 3)}
	}
	m.Entries = []core.Entry{
		route("GET", "/api/assets", "asset", "AssetController.getAll"),
		route("POST", "/api/assets", "asset", "AssetController.upload"),
		route("GET", "/api/albums", "album", "AlbumController.getAll"),
		route("GET", "/api/users", "user", "UserController.me"),
		route("GET", "/api/admin/users", "user", "UserAdminController.list"),
		{ID: "job:server AssetThumbnail", Kind: "job", Path: "AssetThumbnail", Handler: "AssetService.thumbnail",
			Container: "svc:server", Component: "cmp:server/asset", Prov: p("server/src/asset/asset.service.ts", 40)},
		{ID: "job:server FaceDetect", Kind: "job", Path: "FaceDetect", Handler: "FaceService.detect",
			Container: "svc:server", Component: "cmp:server/face", Prov: p("server/src/face/face.service.ts", 12)},
		{ID: "page:web /albums", Kind: "page", Path: "/albums", Container: "app:web", Component: "cmp:web/routes",
			Prov: p("web/src/routes/albums/+page.svelte", 1)},
		{ID: "command:server grant-admin", Kind: "command", Path: "grant-admin", Container: "svc:server", Component: "cmp:server/part05",
			Prov: p("server/src/commands/grant-admin.ts", 5)},
	}
	m.Flows = []core.Flow{
		{Entry: "route:server POST /api/assets", Participants: []core.Participant{
			{ID: "AssetController", Name: "AssetController", Kind: "class", Component: "cmp:server/asset"},
			{ID: "AssetService", Name: "AssetService", Kind: "class", Component: "cmp:server/asset"},
			{ID: "table:asset", Name: "asset", Kind: "table", Element: "table:asset"},
			{ID: "job:AssetThumbnail", Name: "AssetThumbnail", Kind: "job", Element: "job:server AssetThumbnail"},
		}, Steps: []core.Step{
			{From: "AssetController", To: "AssetService", Call: "upload"},
			{From: "AssetService", To: "table:asset", Call: "writes", Depth: 1},
			{From: "AssetService", To: "job:AssetThumbnail", Call: "queues", Depth: 1},
		}},
		{Entry: "job:server AssetThumbnail", Participants: []core.Participant{
			{ID: "AssetService", Name: "AssetService", Kind: "class", Component: "cmp:server/asset"},
			{ID: "job:FaceDetect", Name: "FaceDetect", Kind: "job", Element: "job:server FaceDetect"},
		}, Steps: []core.Step{{From: "AssetService", To: "job:FaceDetect", Call: "queues"}}},
		{Entry: "page:web /albums", Participants: []core.Participant{
			{ID: "page", Name: "albums page", Kind: "module", Component: "cmp:web/routes"},
			{ID: "route", Name: "GET /api/albums", Kind: "route", Element: "route:server GET /api/albums"},
		}, Steps: []core.Step{{From: "page", To: "route", Call: "getAll"}}},
		{Entry: "command:server grant-admin", Participants: []core.Participant{
			{ID: "GrantAdmin", Name: "GrantAdmin", Kind: "class", Component: "cmp:server/part05"},
			{ID: "UserService", Name: "UserService", Kind: "class", Component: "cmp:server/user"},
		}, Steps: []core.Step{{From: "GrantAdmin", To: "UserService", Call: "update"}}},
	}
	return m
}

func TestTheSeedsFollowTheRules(t *testing.T) {
	s := seed(photos())
	if got := s.tables["asset"].Members; strings.Join(got, ",") != "asset,asset_audit" {
		t.Errorf("asset's seed holds %v: a table and its audit table are one seed", got)
	}
	keyOf := map[string]string{}
	for _, id := range s.seedIDs {
		keyOf[id] = s.seeds[id].Key
	}
	seedOf := func(entry string) string { return s.seeds[s.seedOf[entry]].Key }
	if seedOf("job:server AssetThumbnail") != seedOf("route:server GET /api/assets") {
		t.Errorf("a job and the routes its component handles are one seed: %v", keyOf)
	}
	if seedOf("route:server GET /api/users") == seedOf("route:server GET /api/admin/users") {
		t.Error("a role keeps its routes apart: /admin/users joined /users")
	}
	if seedOf("page:web /albums") != seedOf("route:server GET /api/albums") {
		t.Errorf("a page joins the routes that name its resource across the API: %v", keyOf)
	}
	if seedOf("command:server grant-admin") != seedOf("route:server GET /api/admin/users") {
		t.Errorf("a command whose flow reaches one component joins its administrators' routes: %v", keyOf)
	}
	// The part every other uses is plumbing, and so are the parts nothing uses: where a process starts.
	if pl := s.areas["plumbing"].Members; s.areaOf != "svc:server" || !contains(pl, "cmp:server/utils") || contains(pl, "cmp:server/asset") {
		t.Errorf("areas of %q; plumbing %v", s.areaOf, pl)
	}
	if got := s.chain["AssetThumbnail"]; len(got) != 1 || got[0] != "FaceDetect" {
		t.Errorf("the job chain: %v", s.chain)
	}
}

func TestTheRulesReadKeysAsAReaderWould(t *testing.T) {
	for in, want := range map[string]string{"/api/assets/{id}": "assets", "/api/admin/users/:id": "admin/users", "/": "/", "/api": "/", "/admin": "admin"} {
		if got := pathSegment(in); got != want {
			t.Errorf("pathSegment(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"server http albums": "album", "web page memories": "memory", "server http access": "access", "server http admin/users": "user"} {
		if got := resourceOf(in); got != want {
			t.Errorf("resourceOf(%q) = %q, want %q", in, got, want)
		}
	}
	if roleOf("server http admin/users") != "admin" || roleOf("server http users") != "" {
		t.Error("roleOf")
	}
}

// answers is a Completer per stage, answering what the test wrote and recording every request.
type answers struct {
	group, write string
	sent         []string
}

func (a *answers) completer(stage string) Completer {
	return func(_ context.Context, system string, turns []Turn) (Reply, error) {
		a.sent = append(a.sent, system+"\n=====\n"+turns[0].Text)
		if stage == "group" {
			return Reply{Text: a.group, Stop: "end_turn", InputTokens: 3000, OutputTokens: 800}, nil
		}
		return Reply{Text: a.write, Stop: "end_turn", InputTokens: 5000, OutputTokens: 1200}, nil
	}
}

// canned answers both stages for photos: a grouping with an id normalised, one unknown, a table and
// an area left out; and a write-up with a bare job name, a citation that resolves nowhere, a review
// point that cites only that, and a scenario whose steps go backwards.
func canned(t *testing.T) *answers {
	s := seed(photos())
	var areas []string
	for _, id := range s.areaIDs[1:] { // the first area seed is left out
		areas = append(areas, `"cmp:server/`+id+`"`)
	}
	var systems []string
	for _, id := range s.seedIDs {
		systems = append(systems, `"`+id+`"`)
	}
	group := `{"concepts":[{"name":"Photos","meaning":"The pictures kept.","seeds":["[table:asset]"]}],
		"areas":[{"name":"Everything","responsibility":"All of it.","seeds":[` + strings.Join(areas, ",") + `]}],
		"systems":[{"name":"Library","purpose":"Keeping and finding photos.","seeds":[` + strings.Join(systems, ",") + `,"e99"]}]}`
	write := `{"thesis":{"text":"A NestJS server keeps photos in PostgreSQL and hands thumbnails to jobs.","cites":["svc:server","AssetThumbnail","svc:nowhere"]},
		"scenarios":[
			{"title":"Uploading a photo","flow":"route:server POST /api/assets","steps":[{"text":"The upload arrives.","from":1},{"text":"It is written.","from":1},{"text":"Thumbnails, then faces, in the background.","from":4}]},
			{"title":"Backwards","flow":"route:server POST /api/assets","steps":[{"text":"b","from":2},{"text":"a","from":1}]}],
		"review":[{"text":"Upload queues the thumbnail job.","kind":"observation","cites":["route:server POST /api/assets","job:server AssetThumbnail"]},
			{"text":"Unfounded.","kind":"question","cites":["nothing:here"]},
			{"text":"Shortened.","kind":"observation","cites":["route:POST /api/assets","job:FaceDetect","server/asset","web>server","area:library"]}]}`
	if !json.Valid([]byte(group)) || !json.Valid([]byte(write)) {
		t.Fatal("the canned answers are not JSON")
	}
	return &answers{group: group, write: write}
}

func TestInterpretChecksEveryClaimAndPlacesEverySeed(t *testing.T) {
	a := canned(t)
	got, err := Interpret(context.Background(), a.completer("group"), GroupModel, a.completer("write"), Sonnet, photos())
	if err != nil {
		t.Fatal(err)
	}
	if cites := strings.Join(got.Thesis.Cites, ","); cites != "svc:server,job:server AssetThumbnail" {
		t.Errorf("thesis cites %s: the bare job name made its id, the unknown dropped", cites)
	}
	if len(got.Review) != 2 {
		t.Fatalf("%d review points: the one citing nothing that resolves is dropped", len(got.Review))
	}
	if cites := strings.Join(got.Review[1].Cites, ","); cites != "route:server POST /api/assets,job:server FaceDetect,cmp:server/asset,app:web>svc:server" {
		t.Errorf("shortened IDs made whole, an area's name dropped: %s", cites)
	}
	if len(got.Scenarios) != 1 {
		t.Fatalf("%d scenarios: the one going backwards is dropped", len(got.Scenarios))
	}
	steps := got.Scenarios[0].Steps
	if steps[0].From != 1 || steps[0].To != 3 || steps[0].Background || !steps[1].Background || steps[1].To != 4 {
		t.Errorf("scenario steps %+v: three steps of its own, then one in the background", steps)
	}
	if steps[0].Text != "The upload arrives. It is written." {
		t.Errorf("two sentences starting at one step are merged: %q", steps[0].Text)
	}
	if !contains(steps[1].Crosses, "job:server FaceDetect") {
		t.Errorf("the background step crosses %v", steps[1].Crosses)
	}
	var other, album *Group
	for i := range got.Areas {
		if got.Areas[i].Name == "Other" {
			other = &got.Areas[i]
		}
	}
	for i := range got.Concepts {
		if got.Concepts[i].Members[0] == "table:album" {
			album = &got.Concepts[i]
		}
	}
	if other == nil || other.Mark != Proven || album == nil || album.Mark != Proven {
		t.Errorf("what the model left out is placed by the rule, and proven: areas %+v, concepts %+v", got.Areas, got.Concepts)
	}
	if got.Features[0].Mark != WrittenByAI || !contains(got.Features[0].Parts, "cmp:server/asset") {
		t.Errorf("feature %+v", got.Features[0])
	}
	everywhere := false
	for _, ar := range got.Areas {
		everywhere = everywhere || ar.Everywhere
	}
	if !everywhere {
		t.Error("no area is used everywhere: the plumbing's area is")
	}
	for _, want := range []string{"systems: unknown id dropped", "concepts: id normalised", "areas: left out, placed by the rule",
		"thesis: a job's bare name made its ID", "thesis: unresolved citation dropped", "review: a shortened ID made whole", "review: point dropped, nothing it cites resolves",
		"scenarios: dropped, its step numbers do not rise from 1 within the flow", "scenarios: steps starting at the same flow step merged"} {
		if got.Fixes[want] == 0 {
			t.Errorf("fixes %v lack %q", got.Fixes, want)
		}
	}
	if len(got.Calls) != 2 || got.Calls[0].Model != GroupModel || got.Calls[1].Model != Sonnet || !got.CostKnown {
		t.Errorf("calls %+v", got.Calls)
	}
	if want := (3000*0.10 + 800*0.50 + 5000*2.0 + 1200*10.0) / 1e6; got.Cost < want-1e-12 || got.Cost > want+1e-12 {
		t.Errorf("cost %f, want %f", got.Cost, want)
	}
}

// The same answers make the same interpretation, byte for byte; and what was planned is what was sent.
func TestInterpretIsDeterministicAndSendsWhatItPlanned(t *testing.T) {
	run := func() ([]byte, []string) {
		a := canned(t)
		got, err := Interpret(context.Background(), a.completer("group"), GroupModel, a.completer("write"), Sonnet, photos())
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(got)
		return b, a.sent
	}
	first, sent := run()
	for range 5 {
		if again, _ := run(); string(again) != string(first) {
			t.Fatal("two runs on the same answers differ")
		}
	}
	group, _, _, _ := InterpretPlans(photos())
	if sent[0] != group.System+"\n=====\n"+group.Prompt {
		t.Error("stage 1 sent something other than its plan")
	}
	for _, src := range []string{"server/src/asset/asset.controller.ts:3", "return "} {
		if strings.Contains(strings.Join(sent, "\n"), src) {
			t.Errorf("a request carries %q: no line numbers, no code", src)
		}
	}
}

func TestAFailedRequestIsStillAccountedFor(t *testing.T) {
	a := canned(t)
	refuse := func(context.Context, string, []Turn) (Reply, error) {
		return Reply{Refused: true, InputTokens: 5000, OutputTokens: 10}, nil
	}
	got, err := Interpret(context.Background(), a.completer("group"), GroupModel, refuse, WriteModel, photos())
	if err == nil || !strings.Contains(err.Error(), "declined") {
		t.Errorf("a refusal: %v", err)
	}
	if got == nil || len(got.Calls) != 2 || got.Cost == 0 {
		t.Errorf("a failed run still says what it spent: %+v", got)
	}
	cut := func(context.Context, string, []Turn) (Reply, error) {
		return Reply{Text: `{"concepts":[`, Stop: "max_tokens", InputTokens: 3000, OutputTokens: 16000}, nil
	}
	if _, err := Interpret(context.Background(), cut, GroupModel, a.completer("write"), WriteModel, photos()); err == nil || !strings.Contains(err.Error(), "cut off") {
		t.Errorf("an answer cut off: %v", err)
	}
}

func TestHaikuCostsMoreForALongPrompt(t *testing.T) {
	short, _ := Cost(GroupModel, 100_000, 1000)
	long, _ := Cost(GroupModel, 100_001, 1000)
	if short != (100_000*0.10+1000*0.50)/1e6 || long != (100_001*0.50+1000*2.50)/1e6 {
		t.Errorf("haiku: %f up to 100K tokens, %f beyond", short, long)
	}
}
