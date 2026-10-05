package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/arrange"
)

// The app's first writes (surface-spec S-6, S-7, C-*): an arrangement of a diagram, and a named
// view. Both land in archdoc's own directory, both refuse to overwrite a file that changed after
// the page loaded it (ANS-04), and both are actions behind the gate.

// views lists the saved views and the hash of views.yaml as read.
func (s *Server) views(w http.ResponseWriter, r *http.Request) {
	vs, hash, err := arrange.LoadViews(s.root)
	if err != nil {
		fail(w, err)
		return
	}
	send(w, map[string]any{"file": arrange.Dir + "/" + arrange.ViewsFile, "hash": hash, "views": vs})
}

type layoutRequest struct {
	View      string                   `json:"view"`
	Positions map[string]archdoc.Point `json:"positions"`
	Base      string                   `json:"base"`
}

// saveLayout records where a person placed boxes of one view, on top of what was saved before.
// Only elements the latest version of that view holds may be placed: an arrangement cannot name
// something that does not exist.
func (s *Server) saveLayout(w http.ResponseWriter, r *http.Request) {
	var req layoutRequest
	if err := decode(r, &req); err != nil {
		fail(w, err)
		return
	}
	known, err := s.viewIDs(req.View)
	if err != nil {
		fail(w, err)
		return
	}
	for id := range req.Positions {
		if !known[id] {
			fail(w, fmt.Errorf("%s is not an element of the %s view", id, req.View))
			return
		}
	}

	a, _, err := arrange.LoadLayout(s.root)
	if err != nil {
		fail(w, err)
		return
	}
	if a[req.View] == nil {
		a[req.View] = map[string]archdoc.Point{}
	}
	for id, p := range req.Positions {
		a[req.View][id] = p
	}
	s.writeLayout(w, a, req.Base)
}

// resetLayout forgets a view's arrangement, so the engine's layout shows again.
func (s *Server) resetLayout(w http.ResponseWriter, r *http.Request) {
	var req layoutRequest
	if err := decode(r, &req); err != nil {
		fail(w, err)
		return
	}
	if _, err := s.viewIDs(req.View); err != nil {
		fail(w, err)
		return
	}
	a, _, err := arrange.LoadLayout(s.root)
	if err != nil {
		fail(w, err)
		return
	}
	delete(a, req.View)
	s.writeLayout(w, a, req.Base)
}

func (s *Server) writeLayout(w http.ResponseWriter, a arrange.Arrangement, base string) {
	hash, err := arrange.SaveLayout(s.root, a, base)
	if errors.Is(err, arrange.ErrChanged) {
		refuse(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	send(w, map[string]any{"hash": hash})
}

type viewRequest struct {
	View arrange.View `json:"view"`
	Base string       `json:"base"`
}

// saveView adds a named view, or replaces the one with the same name.
func (s *Server) saveView(w http.ResponseWriter, r *http.Request) {
	var req viewRequest
	if err := decode(r, &req); err != nil {
		fail(w, err)
		return
	}
	v := req.View
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" {
		fail(w, errors.New("a view needs a name"))
		return
	}
	known, err := s.viewIDs(v.Level)
	if err != nil {
		fail(w, err)
		return
	}
	if v.Focus != "" && !known[v.Focus] && !strings.Contains(v.Focus, ">") {
		fail(w, fmt.Errorf("%s is not an element of the %s view", v.Focus, v.Level))
		return
	}

	vs, _, err := arrange.LoadViews(s.root)
	if err != nil {
		fail(w, err)
		return
	}
	replaced := false
	for i := range vs {
		if vs[i].Name == v.Name {
			vs[i], replaced = v, true
		}
	}
	if !replaced {
		vs = append(vs, v)
	}
	hash, err := arrange.SaveViews(s.root, vs, req.Base)
	if errors.Is(err, arrange.ErrChanged) {
		refuse(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	send(w, map[string]any{"hash": hash, "views": vs})
}

// viewIDs are the element ids of a view in the latest version.
func (s *Server) viewIDs(view string) (map[string]bool, error) {
	h, err := s.open()
	if err != nil {
		return nil, err
	}
	defer h.Close()
	v, err := h.Latest()
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, errors.New("no version recorded")
	}
	var m archdoc.Model
	switch view {
	case "context":
		m = v.Model.Context()
	case "container":
		m = v.Model.Container()
	default:
		return nil, fmt.Errorf("unknown view %q", view)
	}
	ids := map[string]bool{}
	for _, n := range m.Nodes {
		ids[n.ID] = true
	}
	return ids, nil
}

// decode reads a small JSON body, refusing anything large or malformed.
func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("request body: %w", err)
	}
	return nil
}
