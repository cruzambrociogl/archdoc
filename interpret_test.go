package archdoc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cruzambrociogl/archdoc"
)

// An interpretation sends nothing until its Consent agrees, says both requests and their cost when it
// asks, and is never sent above its cap.
func TestInterpretAsksFirstAndKeepsToItsCap(t *testing.T) {
	m, err := archdoc.ReadModel(shop(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{}); err == nil {
		t.Error("an interpretation with no Consent went ahead")
	}
	var asked archdoc.Request
	_, err = archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{DryRun: true,
		Consent: func(r archdoc.Request) error { asked = r; return nil }})
	if !errors.Is(err, archdoc.ErrStopped) {
		t.Fatalf("a dry run: %v", err)
	}
	if asked.What != "interpret" || len(asked.Plans) != 2 || !asked.EstimateKnown || asked.Estimate <= 0 ||
		asked.Model != archdoc.Haiku {
		t.Errorf("the Consent was asked %+v", asked)
	}
	var withSonnet archdoc.Request
	archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{DryRun: true, WriteModel: archdoc.Sonnet,
		Consent: func(r archdoc.Request) error { withSonnet = r; return nil }})
	if withSonnet.Model != archdoc.Haiku+", then "+archdoc.Sonnet || withSonnet.Estimate <= asked.Estimate {
		t.Errorf("writing on Sonnet: asked %q, about $%.4f against Haiku's $%.4f", withSonnet.Model, withSonnet.Estimate, asked.Estimate)
	}
	if strings.Contains(asked.Plans[0].Prompt+asked.Plans[1].Prompt, "return this.") {
		t.Error("a request carries code")
	}
	called := false
	_, err = archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{Cap: asked.Estimate / 2,
		Consent: func(archdoc.Request) error { called = true; return nil }})
	if !errors.Is(err, archdoc.ErrOverCap) || called {
		t.Errorf("over its cap: %v, asked %v", err, called)
	}
}

// fakeClaude answers the two requests of an interpretation for the shop, and counts what arrived. A
// writing request is refused with a 400 while fail is set.
func fakeClaude(t *testing.T, fail *bool) *[]map[string]any {
	t.Helper()
	var mu sync.Mutex
	var got []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		got = append(got, body)
		mu.Unlock()
		b, _ := json.Marshal(body["system"])
		answer := `{"concepts":[],"areas":[],"systems":[]}`
		if strings.Contains(string(b), "You are writing") {
			if *fail {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"no"}}`))
				return
			}
			answer = `{"thesis":{"text":"A shop: an API in front of a database and a cache.","cites":["svc:api","svc:db"]},
				"scenarios":[],"review":[{"text":"The API is the only way in.","kind":"observation","cites":["svc:api"]}],
				"notes":[{"diagram":"container","text":"Everything goes through the API.","cites":["svc:api"]}]}`
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "msg_test", "type": "message", "role": "assistant", "model": body["model"],
			"stop_reason": "end_turn", "content": []map[string]any{{"type": "text", "text": answer}},
			"usage": map[string]any{"input_tokens": 2000, "output_tokens": 400}})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_WORKSPACE_ID", "")
	return &got
}

// A run with Interpret writes the documentation, then explains it, keeps the explanation beside the
// model, logs what it sent, and does not pay again for an architecture it has explained.
func TestGenerateExplainsOnceAndKeepsIt(t *testing.T) {
	fail := false
	sent := fakeClaude(t, &fail)
	root := shop(t)
	yes := func(archdoc.Request) error { return nil }
	run := func(again bool) (*archdoc.Generated, error) {
		return archdoc.Generate(context.Background(), root, archdoc.GenerateOptions{History: true,
			Interpret: &archdoc.InterpretOptions{Consent: yes, Again: again}})
	}
	res, err := run(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(*sent) != 2 || res.Interpretation == nil || res.Interpretation.Fingerprint == "" || len(res.Interpretation.Notes) != 1 {
		t.Fatalf("sent %d, interpretation %+v", len(*sent), res.Interpretation)
	}
	for _, body := range *sent {
		if body["model"] != archdoc.Haiku || body["fallbacks"] != nil {
			t.Errorf("a request to %v with fallbacks %v: Haiku, which has none", body["model"], body["fallbacks"])
		}
	}
	kept, err := archdoc.ReadInterpretation(root)
	if err != nil || kept.Thesis.Text != res.Interpretation.Thesis.Text {
		t.Fatalf("kept %+v, %v", kept, err)
	}
	h, _ := archdoc.OpenHistory(root)
	runs, _ := h.Runs(10)
	h.Close()
	if len(runs) != 1 || runs[0].Requests != 2 || !runs[0].CostKnown {
		t.Errorf("the run log holds %+v", runs)
	}

	if _, err := run(false); err != nil || len(*sent) != 2 {
		t.Errorf("the same architecture again: %v, %d requests in all — it was explained already", err, len(*sent))
	}
	if _, err := run(true); err != nil || len(*sent) != 4 {
		t.Errorf("--again: %v, %d requests in all", err, len(*sent))
	}

	// A failed explanation costs the documentation nothing, and keeps the last explanation.
	fail = true
	res, err = run(true)
	if err == nil || !strings.Contains(err.Error(), "documentation is written") || res == nil {
		t.Errorf("a failed explanation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "architecture", "index.generated.md")); err != nil {
		t.Error("the documentation was not written")
	}
	if again, _ := archdoc.ReadInterpretation(root); again == nil || again.Thesis.Text != kept.Thesis.Text {
		t.Error("a failed explanation replaced the last one")
	}
}
