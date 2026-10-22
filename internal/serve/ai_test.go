package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/semantic"
)

// aiServer is a repository's server whose model is a script: each request takes the next reply.
func aiServer(t *testing.T, replies ...string) (*Server, *int, func(method, path, body string) *httptest.ResponseRecorder) {
	t.Helper()
	root := repo(t)
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.completer = func(model string, _ map[string]any, _ int64, _ string, rec *semantic.Recorder) semantic.Completer {
		return func(_ context.Context, _ string, turns []semantic.Turn) (semantic.Reply, error) {
			calls++
			// What the recorder would have seen leave: one exchange per request.
			rec.Middleware()(httptest.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(turns[0].Text)),
				func(*http.Request) (*http.Response, error) { return nil, nil })
			return semantic.Reply{Text: replies[min(calls-1, len(replies)-1)], InputTokens: 900, OutputTokens: 120}, nil
		}
	}
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Host = "localhost:7474"
		if method == "POST" {
			req.Header.Set("Origin", "http://localhost:7474")
			req.Header.Set(TokenHeader, s.token)
		}
		s.ServeHTTP(rec, req)
		return rec
	}
	return s, &calls, do
}

// A question says what it would send and cost before anything is sent; asking sends once, and
// the same question again is answered from memory. One nothing matches is answered without asking.
func TestAQuestionIsPricedThenAskedOnce(t *testing.T) {
	_, calls, do := aiServer(t, `{"sentences":[{"text":"The gateway routes to the admin console.","cites":["F1"]}],"not_found":""}`)
	q := "Which part routes requests to the admin console?"

	var preview struct {
		Facts    int
		Estimate struct {
			USD   float64
			Known bool
		}
		Remembered json.RawMessage
	}
	json.Unmarshal(do("GET", "/api/ask?q="+urlq(q), "").Body.Bytes(), &preview)
	if preview.Facts == 0 || !preview.Estimate.Known || preview.Estimate.USD <= 0 || preview.Remembered != nil || *calls != 0 {
		t.Fatalf("preview: %+v, %d calls", preview, *calls)
	}

	var sent struct {
		Answer semantic.Answer
		Sent   bool
		Run    int64
	}
	rec := do("POST", "/api/ask/send", `{"q":"`+q+`"}`)
	json.Unmarshal(rec.Body.Bytes(), &sent)
	if rec.Code != 200 || !sent.Sent || sent.Run == 0 || len(sent.Answer.Claims) != 1 || *calls != 1 {
		t.Fatalf("asking: %d %s, %d calls", rec.Code, rec.Body, *calls)
	}
	json.Unmarshal(do("POST", "/api/ask/send", `{"q":"`+q+`"}`).Body.Bytes(), &sent)
	if sent.Sent || len(sent.Answer.Claims) != 1 || *calls != 1 {
		t.Errorf("asking again went out: %+v, %d calls", sent, *calls)
	}
	json.Unmarshal(do("GET", "/api/ask?q="+urlq(q), "").Body.Bytes(), &preview)
	if preview.Remembered == nil {
		t.Error("the preview does not show the remembered answer")
	}

	json.Unmarshal(do("POST", "/api/ask/send", `{"q":"How is the weather in Lisbon?"}`).Body.Bytes(), &sent)
	if sent.Sent || len(sent.Answer.Claims) != 0 || *calls != 1 {
		t.Errorf("a question about nothing here went out: %+v", sent)
	}

	var runs []map[string]any
	json.Unmarshal(do("GET", "/api/runs", "").Body.Bytes(), &runs)
	found := false
	for _, r := range runs {
		found = found || r["egress_mode"] == semantic.StructureAndQuestion
	}
	if !found {
		t.Errorf("the question is not in the run log: %v", runs)
	}
}

// Asking costs money: it is an action, and a request without the gate's token sends nothing.
func TestAskingIsAnAction(t *testing.T) {
	s, calls, _ := aiServer(t, `{}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/ask/send", strings.NewReader(`{"q":"admin?"}`))
	req.Host = "localhost:7474"
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || *calls != 0 {
		t.Errorf("an ungated ask: %d, %d calls", rec.Code, *calls)
	}
}

// A plan is read from the files a person names, inside the repository only; the check goes out
// once, and comes back from memory while neither the plan nor the code changes.
func TestAPlanIsNamedThenChecked(t *testing.T) {
	s, calls, do := aiServer(t, `{"requirements":[{"id":"R1","status":"built","sentence":"The admin console is served.","cites":["F1"]},{"id":"R2","status":"missing","sentence":"No route mentions billing.","cites":[]}],"extras":[]}`)
	os.MkdirAll(filepath.Join(s.root, "docs"), 0o755)
	os.WriteFile(filepath.Join(s.root, "docs/plan.md"), []byte("# Plan\n\n- An admin console.\n- Billing for every customer.\n"), 0o644)

	if rec := do("POST", "/api/plan/files", `{"files":["../outside.md"]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a plan outside the repository: %d", rec.Code)
	}
	var plan struct {
		Files []string
		Items []struct{ ID, Text string }
		Check json.RawMessage
	}
	json.Unmarshal(do("POST", "/api/plan/files", `{"files":["docs/plan.md"]}`).Body.Bytes(), &plan)
	if len(plan.Files) != 1 || len(plan.Items) != 2 || plan.Items[1].Text != "Billing for every customer." {
		t.Fatalf("plan: %+v", plan)
	}
	if _, err := os.Stat(filepath.Join(s.root, plansFile)); err != nil {
		t.Errorf("the plan files were not kept: %v", err)
	}

	var checked struct {
		Check semantic.PlanCheck
		Sent  bool
	}
	json.Unmarshal(do("POST", "/api/plan/check", "").Body.Bytes(), &checked)
	if !checked.Sent || len(checked.Check.Verdicts) != 2 || checked.Check.Verdicts[0].Status != "built" || *calls != 1 {
		t.Fatalf("check: %+v, %d calls", checked, *calls)
	}
	json.Unmarshal(do("POST", "/api/plan/check", "").Body.Bytes(), &checked)
	if checked.Sent || *calls != 1 {
		t.Errorf("checking again went out")
	}
	json.Unmarshal(do("GET", "/api/plan", "").Body.Bytes(), &plan)
	if plan.Check == nil {
		t.Error("the plan does not show its check")
	}
}

func urlq(q string) string { return strings.NewReplacer(" ", "+", "?", "%3F").Replace(q) }
