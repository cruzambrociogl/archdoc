package serve

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
)

// The gate in front of every action (surface-spec §11.5).
//
// The Host check in ServeHTTP stops another site from *reading* the app through a rebound DNS
// name. It does not stop another site from *submitting* to it: a form on any page can post to
// localhost, and the request arrives with a perfectly valid Host. Reading was all the app ever
// did, so that did not matter. Once the app can run the engine or write a file, it does.
//
// So an action passes three checks, each enough to stop a cross-site form on its own:
//
//   - it is a POST — nothing that changes anything rides on a GET a link or an image can trigger;
//   - its Origin is this server — browsers set Origin on every cross-origin POST and a page
//     cannot forge it;
//   - it carries this session's token in a header — a form cannot set headers, and a script on
//     another origin cannot read the token: /api/session answers only same-origin pages, since
//     the server sends no CORS headers and the Host check blocks the rebinding route.

// TokenHeader carries the session token on every action.
const TokenHeader = "X-Archdoc-Token"

// newToken is a fresh secret for one run of `archdoc serve`.
func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("archdoc: no randomness for the session token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// session gives the page the token it must send with every action. Same-origin pages only: there
// is deliberately no CORS header on any response.
func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	send(w, map[string]any{"token": s.token, "mode": "local"})
}

// action wraps a handler that runs or writes something, so that only this server's own page can
// trigger it. Every action endpoint is registered through it; none is registered without it.
func (s *Server) action(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			refuse(w, http.StatusMethodNotAllowed, "actions are POST requests")
			return
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			refuse(w, http.StatusForbidden, "an action must come from the archdoc page: no Origin")
			return
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			refuse(w, http.StatusForbidden, "an action must come from the archdoc page: Origin "+origin+" is another site")
			return
		}
		got := r.Header.Get(TokenHeader)
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			refuse(w, http.StatusForbidden, "an action must carry this session's token")
			return
		}
		h(w, r)
	}
}

func refuse(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	send(w, map[string]string{"error": msg})
}
