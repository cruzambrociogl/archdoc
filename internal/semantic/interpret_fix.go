package semantic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// What the model answers is checked in code, never sent back for repair: an ID normalised, one that
// does not exist dropped, a seed left out placed by the rule, a citation that does not resolve dropped,
// a scenario whose steps cannot be read dropped. Each fix is counted, so a run says what it corrected.

// rawGroup is one group as stage 1 answered it, after the fixes.
type rawGroup struct {
	Name   string
	Says   string
	Seeds  []string
	ByRule bool // left out by the model and placed by the rule
}

// grouping is stage 1's answer, fixed.
type grouping struct {
	Concepts, Areas, Systems []rawGroup
}

// largestFirst is the features by how many entry points they hold, largest first, ties as answered.
func (g grouping) largestFirst(s *seeding) []rawGroup {
	out := append([]rawGroup(nil), g.Systems...)
	sort.SliceStable(out, func(i, j int) bool {
		return len(s.expand("systems", out[i].Seeds)) > len(s.expand("systems", out[j].Seeds))
	})
	return out
}

// maxPoints is the most review points shown: what the request asks for, and what a page holds.
const maxPoints = 8

// stage1Answer is stage 1's answer as the schema has it.
type stage1Answer struct {
	Concepts []struct {
		Name    string   `json:"name"`
		Meaning string   `json:"meaning"`
		Seeds   []string `json:"seeds"`
	} `json:"concepts"`
	Areas []struct {
		Name           string   `json:"name"`
		Responsibility string   `json:"responsibility"`
		Seeds          []string `json:"seeds"`
	} `json:"areas"`
	Systems []struct {
		Name    string   `json:"name"`
		Purpose string   `json:"purpose"`
		Seeds   []string `json:"seeds"`
	} `json:"systems"`
}

// fixGroups checks stage 1's answer: every ID one of the seeds, a table or area seed placed once, and
// every seed placed — by the rule, where the model left one out.
func (s *seeding) fixGroups(a stage1Answer) (grouping, map[string]int) {
	fixes := map[string]int{}
	tableSeedOf := map[string]string{}
	for _, id := range s.tableIDs {
		for _, x := range s.tables[id].Members {
			tableSeedOf[x] = id
		}
	}
	norm := func(kind, x string) (string, bool) {
		x = strings.Trim(strings.TrimSpace(x), "[]")
		switch kind {
		case "concepts":
			x = strings.TrimPrefix(x, "table:")
			if before, _, ok := strings.Cut(x, " (with "); ok { // the seed's line, copied: "asset (with asset_audit)"
				x = strings.Trim(before, "[]")
			}
			if _, ok := s.tables[x]; ok {
				return x, true
			}
			sd, ok := tableSeedOf[x]
			return sd, ok
		case "areas":
			if strings.HasPrefix(x, "cmp:") {
				x = x[strings.LastIndex(x, "/")+1:]
			}
			_, ok := s.areas[x]
			return x, ok
		}
		_, ok := s.seeds[x]
		return x, ok
	}
	place := func(kind string, groups []rawGroup, pool []string, exclusive bool) []rawGroup {
		placed := map[string]bool{}
		var out []rawGroup
		for _, g := range groups {
			var keep []string
			for _, x := range g.Seeds {
				y, ok := norm(kind, x)
				if !ok {
					fixes[kind+": unknown id dropped"]++
					continue
				}
				if y != x {
					fixes[kind+": id normalised"]++
				}
				if contains(keep, y) || (exclusive && placed[y]) {
					fixes[kind+": placed twice, first kept"]++
					continue
				}
				keep = append(keep, y)
				placed[y] = true
			}
			if len(keep) > 0 {
				g.Seeds = keep
				out = append(out, g)
			}
		}
		var missing []string
		for _, x := range pool {
			if !placed[x] {
				missing = append(missing, x)
			}
		}
		if len(missing) > 0 {
			fixes[kind+": left out, placed by the rule"] += len(missing)
			if kind == "areas" {
				out = append(out, rawGroup{Name: "Other", Says: "Left out by the model; placed by the rule.", Seeds: missing, ByRule: true})
			} else {
				for _, x := range missing {
					name := s.seeds[x].Key
					if kind == "concepts" {
						name = s.tables[x].Key
					}
					out = append(out, rawGroup{Name: name, Says: "Left out by the model; a group of its own by the rule.", Seeds: []string{x}, ByRule: true})
				}
			}
		}
		return out
	}
	var g grouping
	var concepts, areas, systems []rawGroup
	for _, c := range a.Concepts {
		concepts = append(concepts, rawGroup{Name: c.Name, Says: c.Meaning, Seeds: c.Seeds})
	}
	for _, c := range a.Areas {
		areas = append(areas, rawGroup{Name: c.Name, Says: c.Responsibility, Seeds: c.Seeds})
	}
	for _, c := range a.Systems {
		systems = append(systems, rawGroup{Name: c.Name, Says: c.Purpose, Seeds: c.Seeds})
	}
	g.Concepts = place("concepts", concepts, s.tableIDs, true)
	g.Areas = place("areas", areas, s.areaIDs, true)
	g.Systems = place("systems", systems, s.seedIDs, false)
	return g, fixes
}

// drawnByContainer flags features drawn by container or by kind instead of purpose: one holding half
// or more of a container's entry seeds, or one made of a single kind holding half or more of that
// kind's seeds in a container with other kinds, when that is drawnBy seeds or more. Flagged, not
// changed.
func (s *seeding) drawnByContainer(g grouping) []string {
	byC, byCK := map[string]map[string]bool{}, map[string]map[string]bool{}
	for _, id := range s.seedIDs {
		for _, x := range s.seeds[id].Members {
			e := s.entries[x]
			add(byC, e.Container, id)
			add(byCK, e.Container+"\x00"+e.Kind, id)
		}
	}
	kinds := map[string]int{}
	for ck := range byCK {
		c, _, _ := strings.Cut(ck, "\x00")
		kinds[c]++
	}
	var flags []string
	for _, f := range g.Systems {
		mine := map[string]bool{}
		only := map[string]bool{}
		for _, sd := range f.Seeds {
			mine[sd] = true
			for _, x := range s.seeds[sd].Members {
				only[s.entries[x].Kind] = true
			}
		}
		check := func(what string, pool map[string]bool) {
			n := 0
			for sd := range pool {
				if mine[sd] {
					n++
				}
			}
			if len(pool) >= drawnBy && 2*n >= len(pool) {
				flags = append(flags, fmt.Sprintf("%s: %d of %d %s", f.Name, n, len(pool), what))
			}
		}
		for _, c := range sortedKeys(byC) {
			check("entry seeds of "+short(c), byC[c])
		}
		for _, ck := range sortedKeys(byCK) {
			c, k, _ := strings.Cut(ck, "\x00")
			if kinds[c] > 1 && len(only) == 1 && only[k] {
				check(k+" seeds of "+short(c), byCK[ck])
			}
		}
	}
	return flags
}

// citable is every ID an answer may cite: the facts stage 2 was shown.
func citable(m core.Model) map[string]bool {
	ids := map[string]bool{}
	if m.Description != "" {
		ids["S"] = true
	}
	for _, n := range m.Nodes {
		switch {
		case containerish(n.Kind), n.Kind == core.Component:
			ids[n.ID] = true
		case n.Kind == core.Table:
			ids["table:"+n.Name] = true
		}
	}
	for _, e := range m.Edges {
		if !structural(e.Label) {
			ids[e.From+">"+e.To] = true
		}
	}
	for _, e := range m.Entries {
		ids[e.ID] = true
	}
	return ids
}

// shortened finds the ID a model meant when it left part of one out: the container in a job's or a
// route's ID (job:AssetDelete, route:POST /api/assets), a component's kind (web/workflow), the kinds
// on either side of a connection (user>app:shop). Only an ID it matches alone is taken; an ID the
// facts do not hold in any form — an area's name, say — is not one.
func shortened(ids map[string]bool) func(string) (string, bool) {
	alias := map[string]string{}
	ambiguous := map[string]bool{}
	put := func(short, id string) {
		if short == id || ambiguous[short] {
			return
		}
		if prev, ok := alias[short]; ok && prev != id {
			delete(alias, short)
			ambiguous[short] = true
			return
		}
		alias[short] = id
	}
	for id := range ids {
		kind, rest, ok := strings.Cut(id, ":")
		if !ok || strings.Contains(id, ">") {
			continue
		}
		switch kind {
		case "job", "route", "page", "command":
			if _, after, ok := strings.Cut(rest, " "); ok { // "<container> <what>"
				put(kind+":"+after, id)
				put(after, id)
			}
		case "cmp":
			put(rest, id)
		}
		put(rest, id) // a node by its name alone: "app:shop" as "shop"
	}
	for id := range ids {
		if !strings.Contains(id, ">") {
			for _, view := range []string{"component:", "data:", "main:", "structure:"} {
				put(view+id, id) // a diagram's name, for the part it draws
			}
		}
	}
	node := func(x string) (string, bool) {
		if ids[x] {
			return x, true
		}
		id, ok := alias[x]
		return id, ok
	}
	return func(c string) (string, bool) {
		if from, to, ok := strings.Cut(c, ">"); ok {
			f, okF := node(from)
			t, okT := node(to)
			if okF && okT && ids[f+">"+t] {
				return f + ">" + t, true
			}
			return "", false
		}
		id, ok := alias[c]
		return id, ok
	}
}

// The groups are citable by the IDs the summary gives them — concept:N, area:N, system:N, the last a
// feature in the result — and, since a model reaches for a name as often as for an ID, by name
// where no other group shares it: "Shared platform", "area:shared-platform", "system:Albums".
func (s *seeding) groupNames(g grouping, ids map[string]bool) map[string]string {
	names := map[string]string{}
	taken := map[string]int{}
	add := func(kind string, i int, name string) {
		id := fmt.Sprintf("%s:%d", kind, i+1)
		ids[id] = true
		for _, k := range []string{groupKey(name), groupKey(kind + ":" + name)} {
			taken[k]++
			names[k] = id
		}
	}
	for i, c := range g.Concepts {
		add("concept", i, c.Name)
	}
	for i, a := range g.Areas {
		add("area", i, a.Name)
	}
	for i, f := range g.largestFirst(s) {
		add("system", i, f.Name)
		k := groupKey("feature:" + f.Name)
		taken[k]++
		names[k] = fmt.Sprintf("system:%d", i+1)
	}
	for k, n := range taken {
		if n > 1 || ids[k] {
			delete(names, k)
		}
	}
	return names
}

// groupKey is a group's name as it may be cited: case, spaces and hyphens set aside.
func groupKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r == ' ' || r == '-' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// featureID is a system's ID in the summary as the result calls it: a feature.
func featureID(c string) string {
	if n, ok := strings.CutPrefix(c, "system:"); ok && n != "" && strings.Trim(n, "0123456789") == "" {
		return "feature:" + n
	}
	return c
}

// flowAtStart is the given flow an answer named and then went on about — "route:server POST
// /api/assets, then background jobs" — the longest whose ID the answer starts with, then a comma or a
// space.
func flowAtStart(said string, given map[string][]givenStep) string {
	best := ""
	for f := range given {
		if rest, ok := strings.CutPrefix(said, f); ok && (rest == "" || rest[0] == ',' || rest[0] == ' ') && len(f) > len(best) {
			best = f
		}
	}
	return best
}

// stage2Answer is stage 2's answer as the schema has it.
type stage2Answer struct {
	Thesis struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	} `json:"thesis"`
	Scenarios []struct {
		Title string `json:"title"`
		Flow  string `json:"flow"`
		Steps []struct {
			Text string `json:"text"`
			From int    `json:"from"`
		} `json:"steps"`
	} `json:"scenarios"`
	Review []struct {
		Text  string   `json:"text"`
		Kind  string   `json:"kind"`
		Cites []string `json:"cites"`
	} `json:"review"`
	Notes []struct {
		Diagram string   `json:"diagram"`
		Text    string   `json:"text"`
		Cites   []string `json:"cites"`
	} `json:"notes"`
}

// fixWriting checks stage 2's answer: a citation that does not resolve is dropped — a job's bare name
// made its ID first — and a review point left citing nothing is dropped, since nothing is shown
// unchecked; a scenario's steps must start at 1 and rise within the steps its flow was given.
func fixWriting(a stage2Answer, ids map[string]bool, given map[string][]givenStep, diagrams, groups map[string]string) (Claim, []Scenario, []Point, []Note, map[string]int) {
	fixes := map[string]int{}
	jobID := map[string]string{}
	for id := range ids {
		if strings.HasPrefix(id, "job:") {
			if _, bare, ok := strings.Cut(id, " "); ok {
				jobID[bare] = id
			}
		}
	}
	whole := shortened(ids)
	keep := func(cites []string, where string) []string {
		var good []string
		for _, c := range cites {
			c = strings.Trim(strings.TrimSpace(c), "[]")
			if !ids[c] {
				if id, ok := jobID[c]; ok {
					c = id
					fixes[where+": a job's bare name made its ID"]++
				} else if id, ok := whole(c); ok {
					c = id
					fixes[where+": a shortened ID made whole"]++
				} else if id, ok := groups[groupKey(c)]; ok {
					c = id
					fixes[where+": a group's name made its ID"]++
				}
			}
			if !ids[c] {
				fixes[where+": unresolved citation dropped"]++
				continue
			}
			if c = featureID(c); !contains(good, c) {
				good = append(good, c)
			}
		}
		return good
	}
	thesis := Claim{Text: a.Thesis.Text, Cites: keep(a.Thesis.Cites, "thesis"), Mark: WrittenByAI}
	var review []Point
	for _, r := range a.Review {
		cs := keep(r.Cites, "review")
		if len(cs) == 0 {
			fixes["review: point dropped, nothing it cites resolves"]++
			continue
		}
		if len(review) == maxPoints {
			fixes["review: a point beyond eight left out"]++
			continue
		}
		review = append(review, Point{Text: r.Text, Kind: r.Kind, Cites: cs, Mark: WrittenByAI})
	}
	flowIDs := map[string]bool{}
	for f := range given {
		flowIDs[f] = true
	}
	wholeFlow := shortened(flowIDs)
	var scenarios []Scenario
	for _, sc := range a.Scenarios {
		sc.Flow = strings.Trim(strings.TrimSpace(sc.Flow), "[]")
		steps, ok := given[sc.Flow]
		if !ok {
			if f, found := wholeFlow(sc.Flow); found {
				sc.Flow, steps, ok = f, given[f], true
				fixes["scenarios: a shortened flow made whole"]++
			} else if f := flowAtStart(sc.Flow, given); f != "" {
				sc.Flow, steps, ok = f, given[f], true
				fixes["scenarios: words after the flow's ID set aside"]++
			}
		}
		if !ok {
			fixes["scenarios: dropped, its flow was not given"]++
			continue
		}
		n := len(steps)
		// Two sentences that start at the same step are one step told twice: merged, in order.
		merged := sc.Steps[:0:0]
		for _, st := range sc.Steps {
			if last := len(merged) - 1; last >= 0 && merged[last].From == st.From {
				merged[last].Text += " " + st.Text
				fixes["scenarios: steps starting at the same flow step merged"]++
				continue
			}
			merged = append(merged, st)
		}
		sc.Steps = merged
		rising := len(sc.Steps) > 0 && sc.Steps[0].From == 1 && sc.Steps[len(sc.Steps)-1].From <= n
		for i := 1; i < len(sc.Steps) && rising; i++ {
			rising = sc.Steps[i].From > sc.Steps[i-1].From
		}
		if !rising {
			fixes["scenarios: dropped, its step numbers do not rise from 1 within the flow"]++
			continue
		}
		out := Scenario{Title: sc.Title, Flow: sc.Flow, Mark: WrittenByAI}
		for i, st := range sc.Steps {
			to := n
			if i+1 < len(sc.Steps) {
				to = sc.Steps[i+1].From - 1
			}
			// A job's story happens in the background, all of it: no one waits for a job.
			later := steps[st.From-1].background || strings.HasPrefix(sc.Flow, "job:")
			step := ScenarioStep{Text: st.Text, From: st.From, To: to, Background: later}
			for _, g := range steps[st.From-1 : to] {
				for _, x := range g.elements {
					if !contains(step.Crosses, x) {
						step.Crosses = append(step.Crosses, x)
					}
				}
			}
			out.Steps = append(out.Steps, step)
		}
		scenarios = append(scenarios, out)
	}
	// A note is for a diagram the app draws, one each, and rests on facts like any other claim.
	var notes []Note
	noted := map[string]bool{}
	for _, n := range a.Notes {
		said := strings.Trim(strings.TrimSpace(n.Diagram), "[]")
		d, known := diagrams[said]
		if !known {
			d, known = diagrams[strings.ToLower(said)]
			if known {
				fixes["notes: a diagram's title made its name"]++
			}
		}
		switch {
		case !known:
			fixes["notes: dropped, no such diagram"]++
			continue
		case noted[d]:
			fixes["notes: a second for one diagram dropped"]++
			continue
		}
		cs := keep(n.Cites, "notes")
		if len(cs) == 0 {
			fixes["notes: dropped, nothing it cites resolves"]++
			continue
		}
		noted[d] = true
		notes = append(notes, Note{Diagram: d, Text: n.Text, Cites: cs, Mark: WrittenByAI})
	}
	return thesis, scenarios, review, notes, fixes
}
