package archdoc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/semantic"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// StatusReport answers "is the documentation up to date, and what is open?" — the question the
// app's top bar asks. Its JSON is what 'archdoc status --json' prints.
type StatusReport struct {
	Root       string `json:"root"`
	Documented bool   `json:"documented"`
	Name       string `json:"name,omitempty"`
	Version    int64  `json:"version,omitempty"`
	Recorded   string `json:"recorded,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Head       string `json:"head,omitempty"`
	// Changes is how many changes to the structure the code has now that the documentation does
	// not; -1 when the code could not be read.
	Changes      int     `json:"changes"`
	CodeError    string  `json:"code_error,omitempty"`
	Labelled     int     `json:"labelled"`
	Components   int     `json:"components"`
	Explained    int     `json:"explained"`
	Stale        int     `json:"stale"`
	WouldAsk     int     `json:"would_ask"`
	Gaps         int     `json:"gaps"`
	Runs         int     `json:"runs"`
	Spent        float64 `json:"spent_usd"`
	UnknownCosts int     `json:"runs_of_unknown_cost,omitempty"`

	// AskEstimate is what asking about the WouldAsk components should cost, in US dollars;
	// AskEstimateKnown is false when the model's price is unknown.
	AskEstimate      float64 `json:"-"`
	AskEstimateKnown bool    `json:"-"`
}

// Status compares the committed model with the code as it is now, read again offline, by
// structure: a description reworded by a model is not the code changing. Writes nothing.
func Status(ctx context.Context, root string) (*StatusReport, error) {
	r := &StatusReport{Root: absolute(root), Head: render.Commit(root)}

	stored, ok, err := CommittedModel(root)
	if err != nil {
		return nil, err
	}
	if !ok {
		return r, nil
	}
	r.Documented, r.Name = true, stored.Name

	if h, ok, err := openHistory(root); err != nil {
		return nil, err
	} else if ok {
		if v, err := h.Latest(); err == nil && v != nil {
			r.Version, r.Recorded, r.Commit = v.ID, v.CreatedAt.Local().Format("2 Jan 2006 15:04"), v.Commit
		}
		if all, err := h.Runs(1 << 30); err == nil {
			r.Runs = len(all)
			r.Spent, r.UnknownCosts = Spent(all)
		}
		h.Close()
	}

	r.Changes = -1
	if now, err := ReadModel(root); err != nil {
		r.CodeError = err.Error()
	} else {
		r.Changes = Compare(stored, now).Structural()
	}

	for _, n := range stored.Nodes {
		if n.DescProv.Origin.Interpretation() || n.NameProv.Origin.Interpretation() {
			r.Labelled++
		}
		if n.Kind == core.Component {
			r.Components++
		}
	}
	for _, x := range stored.Explanations {
		r.Explained++
		if x.Stale {
			r.Stale++
		}
	}
	memory, err := loadMemory(root)
	if err != nil {
		return nil, err
	}
	plans := semantic.ExplainPlan(stored, memory, "", 0)
	r.WouldAsk = len(plans)
	if r.WouldAsk > 0 {
		bytes := 0
		for _, p := range plans {
			bytes += p.Bytes()
		}
		r.AskEstimate, r.AskEstimateKnown = semantic.Cost(semantic.ExplainModel, int64(float64(bytes)/bytesPerToken), explainAnswer*int64(len(plans)))
	}
	r.Gaps = len(validate.Model(stored).Warnings())
	return r, nil
}

// CommittedModel is .archdoc/model.json: the durable record Generate writes and a repository
// commits, which a fresh clone has and its local history does not.
func CommittedModel(root string) (Model, bool, error) {
	b, err := os.ReadFile(filepath.Join(root, modelOut))
	if os.IsNotExist(err) {
		return core.Model{}, false, nil
	}
	if err != nil {
		return core.Model{}, false, err
	}
	var m core.Model
	if err := json.Unmarshal(b, &m); err != nil {
		return m, false, fmt.Errorf("%s: %w", modelOut, err)
	}
	return m, true, nil
}

// openHistory opens a repository's history only if it has one. store.Open creates the database,
// which a command that only reads must never do to a repository archdoc has not documented.
func openHistory(root string) (*store.Store, bool, error) {
	if _, err := os.Stat(filepath.Join(root, store.File)); os.IsNotExist(err) {
		return nil, false, nil
	}
	h, err := store.Open(root)
	return h, err == nil, err
}

// Spent is what runs cost in US dollars, and how many of them cost an unknown amount.
func Spent(rs []Run) (float64, int) {
	total, unknown := 0.0, 0
	for _, r := range rs {
		if r.CostKnown {
			total += r.CostUSD
		} else {
			unknown++
		}
	}
	return total, unknown
}

func absolute(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}
