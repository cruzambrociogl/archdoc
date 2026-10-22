package serve

import (
	"context"
	"net/http"

	"github.com/cruzambrociogl/archdoc/internal/store"
)

// Updating, as the app sees it (platform-surface §4.3, §4.8): whether a run is updating the
// repository now — from the terminal, or one day from the app — who, since when, and what it is
// doing, read from the lock the run holds (store.Take).
//
// Updating from the app is built, and switched off (decisions, 22 Oct: a page that makes archdoc
// write into a repository is a security question before it is a feature). The action is here and
// behind the gate; nothing gives the server an update to run, so it answers that it is off. No
// place's capabilities say `update` either, so the app shows no button.

// switchedOff is what the action says while updating from the app is off.
const switchedOff = "updating from the app is switched off until it has had a security review — run archdoc generate in a terminal"

func (s *Server) registerUpdate() {
	s.mux.HandleFunc("GET /api/updating", s.updating)
	s.mux.HandleFunc("/api/update", s.action(s.startUpdate))
}

// updating is the run updating the repository now, if any.
func (s *Server) updating(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	busy, ok := store.Running(s.root)
	if !ok {
		send(w, map[string]any{"running": false})
		return
	}
	send(w, map[string]any{"running": true, "by": busy.By, "since": busy.Since, "doing": busy.Doing, "found": busy.Found})
}

// startUpdate starts an update from the app — when one has been given to the server, which today
// none is. It returns at once; the run's progress is read through updating.
func (s *Server) startUpdate(w http.ResponseWriter, r *http.Request) {
	if s.update == nil {
		refuse(w, http.StatusForbidden, switchedOff)
		return
	}
	if busy, ok := store.Running(s.root); ok {
		refuse(w, http.StatusConflict, busy.By+" is updating this repository already")
		return
	}
	go s.update(context.Background())
	w.WriteHeader(http.StatusAccepted)
	send(w, map[string]any{"started": true})
}
