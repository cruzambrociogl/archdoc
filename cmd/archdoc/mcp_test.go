package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reply struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// session sends each line to a server for root, as an agent would, and returns its replies in order.
func session(t *testing.T, root string, lines ...string) []reply {
	t.Helper()
	var out bytes.Buffer
	if err := serveMCP(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, root); err != nil {
		t.Fatal(err)
	}
	var got []reply
	sc := bufio.NewScanner(&out)
	sc.Buffer(nil, 16<<20)
	for sc.Scan() {
		var r reply
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("a reply that is not one line of JSON: %v\n%s", err, sc.Text())
		}
		got = append(got, r)
	}
	return got
}

func call(id int, tool string, args map[string]string) string {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}})
	return string(b)
}

// answer is a tool's text, and whether the tool said it failed.
func answer(t *testing.T, r reply) (string, bool) {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("a protocol error where the tool should answer: %+v", r.Error)
	}
	var res struct {
		Content []struct{ Type, Text string }
		IsError bool
	}
	if err := json.Unmarshal(r.Result, &res); err != nil || len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("result %s", r.Result)
	}
	return res.Content[0].Text, res.IsError
}

// The protocol as an agent meets it: the version it asks for when this server speaks it, the newest
// when not; no reply to a notification; the tools, each promising to change nothing; and errors
// for what is not a request, an unknown method or tool, and a missing argument.
func TestMCPSpeaksTheProtocol(t *testing.T) {
	root := t.TempDir()
	rs := session(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":5,"method":"resources/list"}`,
		call(6, "no-such-tool", nil),
		call(7, "show", nil),
		`not json`,
		`{"id":9,"method":"tools/list"}`,
	)
	if len(rs) != 9 {
		t.Fatalf("%d replies to 9 requests and a notification", len(rs))
	}
	var init struct{ ProtocolVersion string }
	json.Unmarshal(rs[0].Result, &init)
	if init.ProtocolVersion != "2025-06-18" {
		t.Errorf("asked for 2025-06-18, given %q", init.ProtocolVersion)
	}
	json.Unmarshal(rs[1].Result, &init)
	if init.ProtocolVersion != mcpVersions[0] {
		t.Errorf("asked for an unknown version, given %q", init.ProtocolVersion)
	}
	var list struct {
		Tools []struct {
			Name        string
			Annotations map[string]bool
		}
	}
	json.Unmarshal(rs[2].Result, &list)
	var names []string
	for _, tl := range list.Tools {
		names = append(names, tl.Name)
		if !tl.Annotations["readOnlyHint"] || tl.Annotations["destructiveHint"] {
			t.Errorf("%s does not promise to change nothing: %v", tl.Name, tl.Annotations)
		}
	}
	if strings.Join(names, " ") != "overview show list impact changes status" {
		t.Errorf("tools %v", names)
	}
	if string(rs[3].Result) != "{}" {
		t.Errorf("ping: %s", rs[3].Result)
	}
	for i, code := range map[int]int{4: rpcNoMethod, 5: rpcInvalidParams, 6: rpcInvalidParams, 7: rpcParse, 8: rpcInvalidRequest} {
		if rs[i].Error == nil || rs[i].Error.Code != code {
			t.Errorf("reply %d: %+v, want code %d", i, rs[i].Error, code)
		}
	}
	if string(rs[7].ID) != "null" {
		t.Errorf("an unreadable message is answered with id %s, not null", rs[7].ID)
	}
}

// Each tool reads the committed documentation: before generate, every one says to run it; after,
// the overview, show, list and status answer, cited, and changes says one version has nothing to
// compare.
func TestMCPAnswersFromTheCommittedModel(t *testing.T) {
	root := copyFixture(t, "gateway")
	for i, tool := range []string{"overview", "list", "impact", "changes"} {
		text, failed := answer(t, session(t, root, call(i, tool, map[string]string{"kind": "services", "element": "db"}))[0])
		if !failed || !strings.Contains(text, "archdoc generate") {
			t.Errorf("%s before generate: %q", tool, text)
		}
	}
	gen(t, root)
	rs := session(t, root,
		call(1, "overview", nil),
		call(2, "show", map[string]string{"query": "db"}),
		call(3, "list", map[string]string{"kind": "services"}),
		call(4, "status", nil),
		call(5, "changes", nil),
		call(6, "list", map[string]string{"kind": "everything"}),
	)
	for i, want := range []string{"docker-compose.yml:", "id svc:db", "svc:db", "the architecture is as documented", "one version", "list lists one of"} {
		text, failed := answer(t, rs[i])
		if !strings.Contains(text, want) || failed != (i >= 4) {
			t.Errorf("reply %d (failed %v) does not say %q:\n%s", i, failed, want, text)
		}
	}
}

// A model with one route followed through two classes to a table, written as generate would.
const flowModel = `{
  "name": "albums", "source": "docker-compose.yml",
  "nodes": [
    {"id": "svc:api", "name": "api", "kind": "application", "provenance": {"file": "docker-compose.yml", "line": 2}},
    {"id": "cmp:api/album", "name": "album", "kind": "component", "parent": "svc:api", "provenance": {"file": "src/album/album.service.ts", "line": 1}},
    {"id": "tbl:api/album", "name": "album", "kind": "table", "parent": "svc:api", "provenance": {"file": "migrations/1.sql", "line": 1}}
  ],
  "edges": [],
  "entries": [
    {"id": "svc:api GET /albums", "kind": "http", "method": "GET", "path": "/albums", "handler": "AlbumController.getAll",
     "container": "svc:api", "component": "cmp:api/album", "provenance": {"file": "src/album/album.controller.ts", "line": 7},
     "uses": [{"name": "AlbumService", "component": "cmp:api/album", "how": "name", "provenance": {"file": "src/album/album.service.ts", "line": 3}}]}
  ],
  "flows": [
    {"entry": "svc:api GET /albums",
     "participants": [
       {"id": "AlbumController", "name": "AlbumController", "kind": "class", "component": "cmp:api/album"},
       {"id": "AlbumService", "name": "AlbumService", "kind": "class", "component": "cmp:api/album"},
       {"id": "table:album", "name": "album", "kind": "table", "element": "tbl:api/album"}
     ],
     "steps": [
       {"from": "AlbumController", "to": "AlbumService", "call": "getAll", "depth": 0, "provenance": {"file": "src/album/album.controller.ts", "line": 9}},
       {"from": "AlbumService", "to": "table:album", "call": "reads", "depth": 1, "provenance": {"file": "src/album/album.service.ts", "line": 12}}
     ]}
  ]
}`

// impact answers for an element and for a class — which an agent names by the code's own name —
// from the flows: the route through it, who calls it, what it calls. show says where a class is.
func TestMCPImpactFollowsTheFlows(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".archdoc"), 0o755)
	if err := os.WriteFile(filepath.Join(root, ".archdoc", "model.json"), []byte(flowModel), 0o644); err != nil {
		t.Fatal(err)
	}
	rs := session(t, root,
		call(1, "impact", map[string]string{"element": "AlbumService"}),
		call(2, "impact", map[string]string{"element": "tbl:api/album"}),
		call(3, "show", map[string]string{"query": "albumservice"}),
		call(4, "impact", map[string]string{"element": "album"}),
		call(5, "impact", map[string]string{"element": "GET /albums"}),
	)
	text, _ := answer(t, rs[0])
	for _, want := range []string{"Changing AlbumService affects 1 flow: GET /albums.", "GET /albums  src/album/album.controller.ts:7",
		"who uses it (1):\n  AlbumController  src/album/album.controller.ts:9", "what it uses (1):\n  album  tbl:api/album  src/album/album.service.ts:12"} {
		if !strings.Contains(text, want) {
			t.Errorf("impact of a class does not say %q:\n%s", want, text)
		}
	}
	if text, _ := answer(t, rs[1]); !strings.Contains(text, "Changing album affects 1 flow") || !strings.Contains(text, "who uses it (1):\n  album  cmp:api/album") {
		t.Errorf("impact of a table:\n%s", text)
	}
	if text, failed := answer(t, rs[2]); failed || !strings.Contains(text, "AlbumService  [class in component album, cmp:api/album]") || !strings.Contains(text, "in src/album/album.service.ts") {
		t.Errorf("show of a class:\n%s", text)
	}
	if text, failed := answer(t, rs[3]); !failed || !strings.Contains(text, "matches 2 things") {
		t.Errorf("an ambiguous name:\n%s", text)
	}
	if text, failed := answer(t, rs[4]); !failed || !strings.Contains(text, "a way in, not an element") {
		t.Errorf("a route:\n%s", text)
	}
}
