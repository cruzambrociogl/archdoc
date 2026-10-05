package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// guarded is a server with one stand-in action behind the gate, and the token its page would get.
func guarded(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	s, err := New(repo(t))
	if err != nil {
		t.Fatal(err)
	}
	s.mux.HandleFunc("/api/probe", s.action(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)

	code, b := get(t, ts, "/api/session")
	if code != 200 {
		t.Fatalf("session: %d %s", code, b)
	}
	var sess struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &sess); err != nil || len(sess.Token) < 32 {
		t.Fatalf("session token: %q %v", sess.Token, err)
	}
	return ts, sess.Token
}

func act(t *testing.T, ts *httptest.Server, method, origin, token string) int {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+"/api/probe", nil)
	req.Host = "localhost"
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if token != "" {
		req.Header.Set(TokenHeader, token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// Only this server's own page can trigger an action. Each refused case is a way another site,
// or a stray link, could otherwise make archdoc run something or write a file.
func TestActionsOnlyFromTheAppItself(t *testing.T) {
	ts, token := guarded(t)
	cases := []struct {
		name           string
		method, origin string
		token          string
		want           int
	}{
		{"a GET, as a link or an image would send", "GET", "http://localhost", token, http.StatusMethodNotAllowed},
		{"no Origin", "POST", "", token, http.StatusForbidden},
		{"a form on another site", "POST", "https://evil.example", token, http.StatusForbidden},
		{"another port on this machine", "POST", "http://localhost:9999", token, http.StatusForbidden},
		{"no token", "POST", "http://localhost", "", http.StatusForbidden},
		{"a guessed token", "POST", "http://localhost", "0000", http.StatusForbidden},
		{"the app itself", "POST", "http://localhost", token, http.StatusNoContent},
	}
	for _, c := range cases {
		if got := act(t, ts, c.method, c.origin, c.token); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

// Every run of serve has its own token, so one leaked from a past session is worthless.
func TestEachSessionHasItsOwnToken(t *testing.T) {
	_, a := guarded(t)
	_, b := guarded(t)
	if a == b {
		t.Error("two sessions share a token")
	}
}
