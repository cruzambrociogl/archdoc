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
	// 1200 × $5/M + 300 × $25/M = $0.0135
	if cost < 0.0134 || cost > 0.0136 {
		t.Errorf("cost %.4f, want 0.0135", cost)
	}
}

// A model whose price archdoc does not know reports no cost, rather than a guessed one.
func TestUnknownModelHasNoGuessedCost(t *testing.T) {
	if _, known := Cost("some-future-model", 1000, 1000); known {
		t.Error("a price was invented for an unknown model")
	}
}
