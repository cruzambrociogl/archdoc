package render

import (
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// What breaks if one part of Apps and services is down (platform-surface §17.3, One part; settled
// point 5: a rule over proven facts, so Proven, with what it was worked out from).
//
// It is worked out per way in — each route, page, command and background job of the parts that use
// the part — from what the engine followed of each: the tables it reads and writes, the jobs it
// queues. Two rules tie those to a part, and each says so:
//
//   - a part's tables are in the one relational store it connects to (storeOf, the rule the data-flow
//     view already draws by);
//   - a part's job queue is in the one queue store it connects to: Redis, Valkey, RabbitMQ, Kafka…
//
// What the facts cannot tie — a call to an address set at run time, a part that only reaches it
// through another — is said, never guessed. Whether a call always runs is not known either: a way in
// that only sometimes reaches a table stops here all the same.

// Breakage is what one part being down does.
type Breakage struct {
	Part string `json:"part"`
	// Dependents are the parts with an arrow into it, in model order.
	Dependents []Dependent `json:"dependents"`
	// Through are the parts that reach it only through a dependent, by name order.
	Through []Reached `json:"through"`
	// Basis says what it was worked out from, in a sentence.
	Basis string `json:"basis"`
}

// Dependent is one part that uses the part that is down, and how.
type Dependent struct {
	ID string `json:"id"`
	// How says how it depends, in the design's words: "reads and writes its tables", "queues its
	// jobs on it", "waits for the answer", "a person uses it".
	How string `json:"how"`
	// Known is whether which of its ways in need the part is recorded. False for a call archdoc
	// cannot tie to the ways in that make it.
	Known bool `json:"known"`
	// Stops are its ways in that cannot work without the part, each with why and the line.
	Stops []Effect `json:"stops"`
	// Unaffected counts its followed ways in that reach nothing of the part.
	Unaffected int `json:"unaffected"`
	// Unfollowed counts its ways in the engine did not follow, so cannot be told.
	Unfollowed int `json:"unfollowed"`
	// Prov cites the arrow.
	Prov []core.Provenance `json:"provenance"`
}

// Effect is one way in that stops, and why: "reads asset", "queues AssetDelete", "is taken from the queue".
type Effect struct {
	Entry string          `json:"entry"`
	Why   string          `json:"why"`
	Cite  core.Provenance `json:"cite"`
}

// Reached is a part that reaches the one that is down through another part.
type Reached struct {
	ID  string `json:"id"`
	Via string `json:"via"`
}

// queueTech is a technology a job queue lives in.
func queueTech(tech string) bool {
	t := strings.ToLower(tech)
	for _, s := range []string{"redis", "valkey", "keydb", "dragonfly", "rabbitmq", "kafka", "nats", "activemq", "sqs"} {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}

// queueOf is the one queue store a part connects to, where it connects to exactly one.
func queueOf(m core.Model, container string) (core.Node, bool) {
	var found []core.Node
	seen := map[string]bool{}
	for _, e := range m.Edges {
		if e.From != container || !e.Traffic || seen[e.To] {
			continue
		}
		n, ok := m.Node(e.To)
		if ok && (n.Kind == core.Queue || (n.Kind == core.Datastore && queueTech(n.Technology))) {
			seen[e.To] = true
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		return core.Node{}, false
	}
	return found[0], true
}

// HasBreaks reports whether a part can be asked what breaks without it: a part of Apps and services
// that is not a person.
func HasBreaks(m core.Model, part string) bool {
	n, ok := m.Node(part)
	return ok && n.Kind != core.Actor && n.Kind != core.Table && n.Kind != core.Component && n.Kind != core.Module
}

// Breaks works out what stops if part is down.
func Breaks(m core.Model, part string) (Breakage, bool) {
	if !HasBreaks(m, part) {
		return Breakage{}, false
	}
	view := m.Container()
	out := Breakage{Part: part, Dependents: []Dependent{}, Through: []Reached{},
		Basis: "Worked out from which tables each way in reads and writes, and which jobs it queues, as the engine followed it; " +
			"a part's tables are in the one database it connects to, its jobs in the one queue store. Whether a call always runs is not known."}

	flows := map[string]core.Flow{}
	for _, f := range m.Flows {
		flows[f.Entry] = f
	}
	entriesOf := map[string][]core.Entry{}
	for _, e := range m.Entries {
		entriesOf[e.Container] = append(entriesOf[e.Container], e)
	}

	direct := map[string]bool{}
	for _, e := range view.Edges {
		if e.To != part || direct[e.From] {
			continue
		}
		from, ok := view.Node(e.From)
		if !ok {
			continue
		}
		direct[e.From] = true
		d := Dependent{ID: e.From, Stops: []Effect{}, Prov: e.Prov}
		store, isStore := storeOf(m, e.From)
		queue, isQueue := queueOf(m, e.From)
		switch {
		case from.Kind == core.Actor:
			d.How, d.Known = "a person uses it", true
		case isStore && store.ID == part:
			d.How, d.Known = "reads and writes its tables", true
			d.Stops, d.Unaffected, d.Unfollowed = effects(m, entriesOf[e.From], flows, func(f core.Flow, p map[string]core.Participant) (Effect, bool) {
				for _, s := range f.Steps {
					q := p[s.To]
					if q.Kind != "table" || q.Element == "" {
						continue
					}
					if t, ok := m.Node(q.Element); ok && t.Parent == e.From {
						return Effect{Entry: f.Entry, Why: s.Call + " " + q.Name, Cite: s.Prov}, true
					}
				}
				return Effect{}, false
			})
		case isQueue && queue.ID == part:
			d.How, d.Known = "queues its jobs on it", true
			// A way in that queues a job waits for the queue to accept it; every background job is
			// taken from the queue, so none runs.
			var asks, jobs []core.Entry
			for _, en := range entriesOf[e.From] {
				if en.Kind == "job" {
					jobs = append(jobs, en)
				} else {
					asks = append(asks, en)
				}
			}
			d.Stops, d.Unaffected, d.Unfollowed = effects(m, asks, flows, func(f core.Flow, p map[string]core.Participant) (Effect, bool) {
				for _, s := range f.Steps {
					if p[s.To].Kind == "job" {
						return Effect{Entry: f.Entry, Why: "queues " + p[s.To].Name, Cite: s.Prov}, true
					}
				}
				return Effect{}, false
			})
			for _, en := range jobs {
				d.Stops = append(d.Stops, Effect{Entry: en.ID, Why: "is taken from the queue", Cite: en.Prov})
			}
			sortEffects(d.Stops)
		case e.Technology == "http" || e.Technology == "https" || e.Technology == "grpc" || strings.HasPrefix(e.Label, "calls"):
			d.How, d.Known = "waits for the answer", false
		default:
			d.How, d.Known = strings.TrimSpace(e.Label+" it"), false
		}
		out.Dependents = append(out.Dependents, d)
	}

	// Who reaches it only through a dependent: upstream, the first part on the way.
	for _, r := range reachUp(view, part) {
		if !direct[r.ID] {
			out.Through = append(out.Through, r)
		}
	}
	sort.Slice(out.Through, func(i, j int) bool { return out.Through[i].ID < out.Through[j].ID })
	return out, true
}

// effects splits a part's ways in by a test on each one's flow: the ones it stops, how many it leaves
// alone, and how many were not followed.
func effects(m core.Model, entries []core.Entry, flows map[string]core.Flow, stops func(core.Flow, map[string]core.Participant) (Effect, bool)) ([]Effect, int, int) {
	out := []Effect{}
	unaffected, unfollowed := 0, 0
	for _, en := range entries {
		f, ok := flows[en.ID]
		if !ok {
			unfollowed++
			continue
		}
		p := map[string]core.Participant{}
		for _, x := range f.Participants {
			p[x.ID] = x
		}
		if e, ok := stops(f, p); ok {
			out = append(out, e)
		} else {
			unaffected++
		}
	}
	sortEffects(out)
	return out, unaffected, unfollowed
}

func sortEffects(es []Effect) {
	sort.SliceStable(es, func(i, j int) bool { return es[i].Entry < es[j].Entry })
}

// reachUp is every part that reaches part along the arrows, each with the first part on its way
// (the dependent it goes through), breadth first in the arrows' order.
func reachUp(view core.Model, part string) []Reached {
	via := map[string]string{}
	var order []string
	queue := []string{part}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, e := range view.Edges {
			if e.To != c || e.From == part {
				continue
			}
			if _, seen := via[e.From]; seen {
				continue
			}
			if c == part {
				via[e.From] = ""
			} else if v := via[c]; v != "" {
				via[e.From] = v
			} else {
				via[e.From] = c
			}
			order = append(order, e.From)
			queue = append(queue, e.From)
		}
	}
	out := []Reached{}
	for _, id := range order {
		out = append(out, Reached{ID: id, Via: via[id]})
	}
	return out
}
