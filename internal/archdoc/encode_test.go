package archdoc

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// The laid-out model is the same model: it reads back equal to the plain encoding, a citation
// takes one line, a value with no source of its own is not written, and no line runs long unless
// a single string does.
func TestModelJSONIsCompactAndTheSameModel(t *testing.T) {
	long := make([]string, 40)
	for i := range long {
		long[i] = "server/src/services/some-fairly-long-file-name.service.ts"
	}
	m := Model{
		Name: "shop",
		Nodes: []Node{
			{ID: "svc:api", Name: "api", Kind: Application, Prov: Provenance{File: "compose.yml", Line: 3}},
			{ID: "cmp:api/orders", Name: "orders", Kind: Component, Parent: "svc:api", Files: long, Prov: Provenance{File: "src/orders.ts", Line: 1}},
		},
		Edges: []Edge{{From: "svc:api", To: "svc:db", Label: `reads "orders" <fast>`, Prov: []Provenance{{File: "compose.yml", Line: 9}}}},
	}
	got, err := m.JSON()
	if err != nil {
		t.Fatal(err)
	}
	again, _ := m.JSON()
	if !bytes.Equal(got, again) {
		t.Fatal("two encodings of one model differ")
	}

	var back, plain Model
	raw, _ := json.Marshal(m)
	if err := json.Unmarshal(got, &back); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, got)
	}
	json.Unmarshal(raw, &plain)
	if !reflect.DeepEqual(back, plain) {
		t.Errorf("the laid-out model reads back different:\n%+v\n%+v", back, plain)
	}

	text := string(got)
	if !strings.Contains(text, `"provenance": {"file": "compose.yml", "line": 3}`) {
		t.Errorf("a citation is not on one line:\n%s", text)
	}
	if strings.Contains(text, "name_provenance") {
		t.Errorf("an empty provenance was written:\n%s", text)
	}
	for _, line := range strings.Split(text, "\n") {
		if len(line) > inlineWidth {
			t.Errorf("a line of %d characters: %.80s…", len(line), line)
		}
	}
}

// The order of nodes depends on the nodes alone, not on how they arrived or what else is there:
// kinds that share a rank are ordered by ID among themselves, so adding an element moves nothing.
func TestNodeOrderIsTotal(t *testing.T) {
	nodes := func(ids ...string) []Node {
		kind := map[byte]Kind{'s': Application, 'c': Component, 't': Table, 'd': Module}
		var out []Node
		for _, id := range ids {
			out = append(out, Node{ID: id, Kind: kind[id[0]]})
		}
		return out
	}
	order := func(ns []Node) string {
		m := Model{Nodes: ns}.Normalise()
		var ids []string
		for _, n := range m.Nodes {
			ids = append(ids, n.ID)
		}
		return strings.Join(ids, " ")
	}
	want := "cmp:a cmp:b dir:x svc:api tbl:tag tbl:tag_asset tbl:tag_closure"
	for _, arrival := range [][]string{
		{"tbl:tag_closure", "cmp:b", "svc:api", "tbl:tag", "dir:x", "cmp:a", "tbl:tag_asset"},
		{"svc:api", "tbl:tag_asset", "tbl:tag_closure", "cmp:a", "tbl:tag", "cmp:b", "dir:x"},
		{"dir:x", "tbl:tag", "cmp:a", "tbl:tag_closure", "svc:api", "cmp:b", "tbl:tag_asset"},
	} {
		if got := order(nodes(arrival...)); got != want {
			t.Errorf("arriving as %v:\n got  %s\n want %s", arrival, got, want)
		}
	}
}
