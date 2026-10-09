package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// fakeAnthropic answers like the Messages API and remembers every body it received, so a test can
// compare what the recorder says left the machine with what actually arrived.
func fakeAnthropic(t *testing.T, answer string) (*httptest.Server, *[][]byte) {
	t.Helper()
	var (
		mu       sync.Mutex
		received [][]byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		received = append(received, b)
		mu.Unlock()

		resp := map[string]any{
			"id": "msg_test", "type": "message", "role": "assistant", "model": Model,
			"stop_reason": "end_turn",
			"content":     []map[string]any{{"type": "text", "text": answer}},
			"usage":       map[string]any{"input_tokens": 1200, "output_tokens": 300},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv, &received
}

// AC-8, measured on the wire rather than asserted about a string: the real SDK talks to a local
// server, and every byte the server received is in the recorder, and nothing more.
func TestRecorderAccountsForEveryByte(t *testing.T) {
	srv, received := fakeAnthropic(t, good().Text)
	rec := &Recorder{}
	complete := Claude(Model, rec, option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"))

	if _, _, err := Label(context.Background(), complete, Model, fixture()); err != nil {
		t.Fatalf("label: %v", err)
	}

	got := rec.Exchanges()
	if len(got) != len(*received) {
		t.Fatalf("recorder has %d requests, server received %d", len(got), len(*received))
	}
	for i := range got {
		if !bytes.Equal(got[i].Body, (*received)[i]) {
			t.Errorf("request %d: the recorded body differs from what arrived", i)
		}
		if got[i].Status != http.StatusOK {
			t.Errorf("request %d: status %d not recorded", i, got[i].Status)
		}
		// What came back is kept too — and the SDK still read it, or Label would have failed.
		if !bytes.Contains(got[i].Response, []byte(`"usage"`)) {
			t.Errorf("request %d: the answer was not recorded: %q", i, got[i].Response)
		}
	}
	if rec.Bytes() == 0 {
		t.Error("no bytes accounted for")
	}
}

// AC-8's other half: structure only. Nothing read from a file — no path, no line number, no
// provenance — appears anywhere in what left the machine, including whatever the client added
// around the prompt.
func TestNothingFromAFileLeavesTheMachine(t *testing.T) {
	srv, _ := fakeAnthropic(t, good().Text)
	rec := &Recorder{}
	complete := Claude(Model, rec, option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"))

	if _, _, err := Label(context.Background(), complete, Model, fixture()); err != nil {
		t.Fatalf("label: %v", err)
	}

	for _, ex := range rec.Exchanges() {
		body := string(ex.Body)
		for _, leak := range []string{"docker-compose.yml", "provenance", `"line"`} {
			if strings.Contains(body, leak) {
				t.Errorf("%q left the machine", leak)
			}
		}
		// The key goes in a header, never in the body the log keeps.
		if strings.Contains(body, "test-key") {
			t.Error("the API key is in the recorded body")
		}
	}
}

// Tokens are read off the response and summed, which is what cost is made of.
func TestTokensAreReportedFromTheResponse(t *testing.T) {
	srv, _ := fakeAnthropic(t, good().Text)
	complete := Claude(Model, &Recorder{}, option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"))

	_, rep, err := Label(context.Background(), complete, Model, fixture())
	if err != nil {
		t.Fatalf("label: %v", err)
	}
	if rep.InputTokens != 1200 || rep.OutputTokens != 300 {
		t.Errorf("tokens: %d in, %d out; want 1200 and 300", rep.InputTokens, rep.OutputTokens)
	}

	cost, known := Cost(Model, rep.InputTokens, rep.OutputTokens)
	if !known {
		t.Fatal("the default model's price is unknown")
	}
	// 1200 × $2/M + 300 × $10/M = $0.0054
	if cost < 0.0053 || cost > 0.0055 {
		t.Errorf("cost %.4f, want 0.0054", cost)
	}
}

// A model whose price archdoc does not know reports no cost, rather than a guessed one.
func TestUnknownModelHasNoGuessedCost(t *testing.T) {
	if _, known := Cost("some-future-model", 1000, 1000); known {
		t.Error("a price was invented for an unknown model")
	}
}

// AC-8 for --explain, on the wire: what leaves is names, paths, counts and each route's own
// summary — and nothing else the model holds. No line number, no column type, no description, no
// note about how a path was resolved, no key.
func TestExplainSendsStructureAndSummariesAndNothingElse(t *testing.T) {
	srv, received := fakeAnthropic(t, `{"sentences":[{"text":"It serves albums.","cites":["F1"]}]}`)
	rec := &Recorder{}
	complete := ClaudeWith(ExplainModel, ExplainSchema(), ExplainMaxTokens, "low", rec, option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"))

	m := explainModel()
	for i := range m.Nodes {
		m.Nodes[i].Description = "a description nobody asked to send"
		m.Nodes[i].Prov.Line = 7777
	}
	m.Nodes = append(m.Nodes, archdoc.Node{ID: "tbl:server/album", Name: "album", Kind: archdoc.Table, Parent: "svc:server",
		Dir: "server/src/controllers/album.controller.ts", Prov: archdoc.Provenance{File: "server/src/controllers/album.controller.ts", Line: 7777},
		Columns: []archdoc.Column{{Name: "ownerId", Type: "uuid-type-not-sent"}}})
	for i := range m.Entries {
		m.Entries[i].Prov.Line = 7777
		m.Entries[i].PathNote = "a note about resolution, not sent"
	}
	if _, _, _, err := Explain(context.Background(), complete, ExplainModel, m, nil); err != nil {
		t.Fatalf("explain: %v", err)
	}
	if len(*received) == 0 {
		t.Fatal("nothing was sent")
	}
	sent := ""
	for _, b := range *received {
		sent += string(b)
	}
	for _, want := range []string{"List all albums", "server/src/controllers/album.controller.ts", "ownerId", "GET /api/albums"} {
		if !strings.Contains(sent, want) {
			t.Errorf("%q was not sent — the mode says it is", want)
		}
	}
	for _, leak := range []string{"7777", "uuid-type-not-sent", "a description nobody asked to send", "a note about resolution", "provenance", "test-key"} {
		if strings.Contains(sent, leak) {
			t.Errorf("%q left the machine", leak)
		}
	}
	if rec.Bytes() != len(sent) {
		t.Errorf("the recorder holds %d bytes, the server received %d", rec.Bytes(), len(sent))
	}
}
