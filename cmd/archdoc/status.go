package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/model"
	"github.com/cruzambrociogl/archdoc/internal/render"
	"github.com/cruzambrociogl/archdoc/internal/semantic"
	"github.com/cruzambrociogl/archdoc/internal/store"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// status answers "is the documentation up to date, and what is open?" — the question the app's
// top bar asks. The committed model is compared with the code as it is now, read again offline, by
// structure: a description reworded by a model is not the code changing. Writes nothing.
func status(e env, args []string) error {
	fs := flags("status")
	asJSON := fs.Bool("json", false, "print the status as JSON")
	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	root, err := readPath(fs, positional)
	if err != nil {
		return err
	}

	type report struct {
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
	}
	r := report{Root: absolute(root), Head: render.Commit(root)}

	stored, ok, err := committedModel(root)
	if err != nil {
		return err
	}
	if !ok {
		if *asJSON {
			return printJSON(e.out, r)
		}
		fmt.Fprintf(e.out, "%s is not documented yet — 'archdoc scan %s' previews it, 'archdoc generate %s' writes it.\n", r.Root, root, root)
		return nil
	}
	r.Documented, r.Name = true, stored.Name

	if h, ok, err := openHistory(root); err != nil {
		return err
	} else if ok {
		if v, err := h.Latest(); err == nil && v != nil {
			r.Version, r.Recorded, r.Commit = v.ID, v.CreatedAt.Local().Format("2 Jan 2006 15:04"), v.Commit
		}
		if all, err := h.Runs(1 << 30); err == nil {
			r.Runs = len(all)
			r.Spent, r.UnknownCosts = spent(all)
		}
		h.Close()
	}

	r.Changes = -1
	if now, err := modelOf(root); err != nil {
		r.CodeError = err.Error()
	} else {
		r.Changes = structuralChanges(model.Compare(stored, now))
	}

	for _, n := range stored.Nodes {
		if n.DescProv.Origin.Interpretation() || n.NameProv.Origin.Interpretation() {
			r.Labelled++
		}
		if n.Kind == archdoc.Component {
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
		return err
	}
	plans := semantic.ExplainPlan(stored, memory, "", 0)
	r.WouldAsk = len(plans)
	r.Gaps = len(validate.Model(stored).Warnings())

	if *asJSON {
		return printJSON(e.out, r)
	}
	out := e.out
	line := func(head, format string, a ...any) { fmt.Fprintf(out, "%-14s %s\n", head, fmt.Sprintf(format, a...)) }
	fmt.Fprintf(out, "%s — %s\n\n", r.Name, r.Root)
	switch {
	case r.Version > 0:
		line("documented", "version %d, recorded %s%s", r.Version, r.Recorded, atCommit(r.Commit))
	default:
		line("documented", "%s is committed; this clone has no local history — 'archdoc generate %s' records it", modelOut, root)
	}
	switch {
	case r.Changes < 0:
		line("the code now", "could not be read: %s", firstLine(r.CodeError))
	case r.Changes == 0:
		line("the code now", "%sthe architecture is as documented", headAt(r.Head))
	default:
		line("the code now", "%s%d change(s) to the structure since — 'archdoc generate %s' records them", headAt(r.Head), r.Changes, root)
		if r.Commit != "" {
			line("", "'archdoc diff %s %s' says what they are", root, r.Commit)
		}
	}
	if r.Labelled > 0 {
		line("descriptions", "%d written by a model, marked as such — 'archdoc label %s' asks again", r.Labelled, root)
	} else {
		line("descriptions", "none written by a model — 'archdoc label %s' asks for them", root)
	}
	if r.Components > 0 {
		x := fmt.Sprintf("%d of %d components", r.Explained, r.Components)
		if r.Stale > 0 {
			x += fmt.Sprintf(", %d of them about facts that have since changed", r.Stale)
		}
		if r.WouldAsk > 0 {
			bytes := 0
			for _, p := range plans {
				bytes += p.Bytes()
			}
			x += fmt.Sprintf(" — 'archdoc explain %s' would ask about %d", root, r.WouldAsk)
			if usd, ok := semantic.Cost(semantic.ExplainModel, int64(float64(bytes)/bytesPerToken), explainAnswer*int64(len(plans))); ok {
				x += ", " + dollars(usd)
			}
		}
		line("explanations", "%s", x)
	}
	if r.Gaps > 0 {
		line("gaps", "%d — 'archdoc generate %s --gaps' lists them", r.Gaps, root)
	} else {
		line("gaps", "none")
	}
	switch {
	case r.Runs == 0:
		line("network", "never used")
	default:
		x := fmt.Sprintf("%d run(s), $%.2f in all", r.Runs, r.Spent)
		if r.UnknownCosts > 0 {
			x += fmt.Sprintf(" and %d of unknown cost", r.UnknownCosts)
		}
		line("network", "%s — 'archdoc runs %s' lists them", x, root)
	}
	return nil
}

// structuralChanges counts what changed in what the system is, leaving out what it is called or
// said to be: a model's descriptions are in the committed model and not in a fresh reading.
func structuralChanges(d model.Diff) int {
	n := len(d.AddedNodes) + len(d.RemovedNodes) + len(d.AddedEdges) + len(d.RemovedEdges) +
		len(d.AddedEntries) + len(d.RemovedEntries) + len(d.Moved)
	for _, c := range d.Changed {
		if !wordFields[c.Field] {
			n++
		}
	}
	return n
}

// committedModel is .archdoc/model.json: the durable record generate writes and a repository
// commits, which a fresh clone has and its local history does not.
func committedModel(root string) (archdoc.Model, bool, error) {
	b, err := os.ReadFile(filepath.Join(root, modelOut))
	if os.IsNotExist(err) {
		return archdoc.Model{}, false, nil
	}
	if err != nil {
		return archdoc.Model{}, false, err
	}
	var m archdoc.Model
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

func atCommit(c string) string {
	if c == "" {
		return ""
	}
	if len(c) > 7 {
		c = c[:7]
	}
	return " at commit " + c
}

func headAt(c string) string {
	if c == "" {
		return ""
	}
	return strings.TrimSpace(atCommit(c)) + " — "
}

func firstLine(s string) string {
	first, _, _ := strings.Cut(s, "\n")
	return first
}
