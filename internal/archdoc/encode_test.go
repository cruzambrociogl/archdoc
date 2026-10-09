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
