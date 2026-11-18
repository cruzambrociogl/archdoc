package semantic

import (
	"regexp"
	"sort"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// The first grouping, made by rule, that the model then names and merges (Interpret). A seed is the
// smallest group the model may move, so whatever it answers, the groups are unions of seeds. Three
// kinds, each a rule that holds on any repository:
//
//   - tables: a table with its audit table (x and x_audit);
//   - components of the container with the most, when it has enough to need areas: each a seed,
//     except the plumbing — the parts that handle no entry point and that a quarter of the others use,
//     or that use a quarter of the others, or that nothing uses — which is one seed; parts whose entry
//     points sit mostly in one entry-point seed, which join; and a part with no entry point that only
//     one seed uses, which joins it;
//   - entry points: routes and pages by their first path segment (after api; admin keeps the next),
//     jobs by the class that handles them, commands by their file; always within one container. Then
//     joined across kinds, so a feature cannot be drawn by container or by kind: routes and jobs the
//     same component handles; a command with the one component its flow reaches that handles routes or
//     jobs; pages with the routes that name the same resource, across a proven API call.
//
// It also derives, from the flows, what each entry point reaches and which jobs queue which: facts the
// model has, put where a reader of the summary can see them.

const (
	areasFrom = 20 // a container with fewer components is shown as it is, without areas
	layerFrom = 4  // a component holding this many seeds of one kind is a layer, not a purpose
	sharedMin = 4  // the engine's sharedMin: used by, or using, a quarter of the others and at least this many
	callsAPI  = "calls the API of"
)

// roles are path prefixes that name an audience rather than a resource: the segment after one does.
var roles = map[string]bool{"admin": true}

var auditSuffix = regexp.MustCompile(`_audit$`)

type tableSeed struct {
	Key     string
	Members []string // table names
}

type entrySeed struct {
	Key     string
	Members []string // entry IDs
	Joined  []string // why its keys joined
}

type areaSeed struct {
	Key     string   // the component the seed is rooted at
	Members []string // component IDs
	Why     []string
}

// reached is what one entry point's flow reaches.
type reached struct {
	tables     map[string]map[string]bool // table name → the calls on it
	jobs       map[string]bool            // job names
	components []string                   // in the order the flow meets them
	routes     map[string]bool            // the route entries a page calls
}

// seeding is the first grouping of one model, and what it was derived from.
type seeding struct {
	m       core.Model
	nodes   map[string]core.Node
	entries map[string]core.Entry

	tables   map[string]tableSeed
	tableIDs []string
	areas    map[string]areaSeed
	areaIDs  []string
	areaOf   string // the container the areas divide; empty when there are none
	seeds    map[string]entrySeed
	seedIDs  []string // e1, e2, … in order
	seedOf   map[string]string

	reach map[string]*reached
	chain map[string][]string // a job → the jobs its own flow queues
}

func seed(m core.Model) *seeding {
	s := &seeding{m: m, nodes: map[string]core.Node{}, entries: map[string]core.Entry{}, reach: reaches(m)}
	for _, n := range m.Nodes {
		s.nodes[n.ID] = n
	}
	for _, e := range m.Entries {
		s.entries[e.ID] = e
	}

	// tables: x with x_audit
	byBase := map[string][]string{}
	names := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Kind == core.Table {
			base := auditSuffix.ReplaceAllString(n.Name, "")
			byBase[base] = append(byBase[base], n.Name)
			names[n.Name] = true
		}
	}
	s.tables = map[string]tableSeed{}
	for _, base := range sortedKeys(byBase) {
		members := append([]string(nil), byBase[base]...)
		sort.Strings(members)
		if !names[base] { // an audit table whose table archdoc did not see: a seed of its own
			base = members[0]
		}
		s.tables[base] = tableSeed{Key: base, Members: members}
		s.tableIDs = append(s.tableIDs, base)
	}

	// entry points
	keys, byKey := []string{}, map[string][]string{}
	for _, e := range m.Entries {
		k := entryKey(e)
		if _, ok := byKey[k]; !ok {
			keys = append(keys, k)
		}
		byKey[k] = append(byKey[k], e.ID)
	}
	type joinedSeed struct {
		key     string
		members []string
		why     []string
	}
	var all []joinedSeed
	for _, g := range s.joined(keys, byKey) {
		var members []string
		for _, k := range g.keys {
			members = append(members, byKey[k]...)
		}
		sort.Strings(members)
		all = append(all, joinedSeed{strings.Join(g.keys, " + "), members, g.why})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].key < all[j].key })
	s.seeds, s.seedOf = map[string]entrySeed{}, map[string]string{}
	for i, g := range all {
		id := "e" + itoa(i+1)
		s.seeds[id] = entrySeed{Key: g.key, Members: g.members, Joined: g.why}
		s.seedIDs = append(s.seedIDs, id)
		for _, x := range g.members {
			s.seedOf[x] = id
		}
	}

	// components: the container with the most
	count := map[string]int{}
	for _, n := range m.Nodes {
		if n.Kind == core.Component {
			count[n.Parent]++
		}
	}
	main := ""
	for p, n := range count {
		if main == "" || n > count[main] || (n == count[main] && p > main) {
			main = p
		}
	}
	if main != "" && count[main] >= areasFrom {
		s.areaSeeds(main)
	}

	// which jobs queue which, from the job entry points' own flows
	jobs := map[string]string{}
	for _, e := range m.Entries {
		if e.Kind == "job" {
			jobs[e.Path] = e.ID
		}
	}
	s.chain = map[string][]string{}
	for j, id := range jobs {
		r, ok := s.reach[id]
		if !ok {
			continue
		}
		var next []string
		for k := range r.jobs {
			if k != j {
				next = append(next, k)
			}
		}
		if len(next) > 0 {
			sort.Strings(next)
			s.chain[j] = next
		}
	}
	return s
}

// unions is a union-find over string keys that remembers why each set joined.
type unions struct {
	parent map[string]string
	why    map[string]map[string]bool
}

func newUnions(keys []string) *unions {
	u := &unions{parent: map[string]string{}, why: map[string]map[string]bool{}}
	for _, k := range keys {
		u.parent[k] = k
	}
	return u
}

func (u *unions) find(k string) string {
	for u.parent[k] != k {
		u.parent[k] = u.parent[u.parent[k]]
		k = u.parent[k]
	}
	return k
}

// join puts b's set into a's: a's root stays the root.
func (u *unions) join(a, b, reason string) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[rb] = ra
		if u.why[ra] == nil {
			u.why[ra] = map[string]bool{}
		}
		for w := range u.why[rb] {
			u.why[ra][w] = true
		}
		delete(u.why, rb)
	}
	if u.why[ra] == nil {
		u.why[ra] = map[string]bool{}
	}
	u.why[ra][reason] = true
}

func (u *unions) reasons(root string) []string { return sortedKeys(u.why[root]) }

type keyGroup struct {
	keys []string
	why  []string
}

// joined joins the entry-point keys across kinds.
func (s *seeding) joined(keys []string, byKey map[string][]string) []keyGroup {
	first := map[string]core.Entry{}
	keyOf := map[string]string{}
	for _, k := range keys {
		first[k] = s.entries[byKey[k][0]]
		for _, x := range byKey[k] {
			keyOf[x] = k
		}
	}
	u := newUnions(keys)

	// routes and jobs handled by the same component. A role names an audience, so routes join only
	// routes of the same role; jobs, which have none, join the routes without a role, or a role's when
	// the component has no others.
	type handled struct {
		jobs   []string
		routes map[string][]string // role → keys
	}
	var comps []string
	by := map[string]*handled{}
	for _, k := range keys {
		kind := first[k].Kind
		if kind != "http" && kind != "job" {
			continue
		}
		t := newTally()
		for _, x := range byKey[k] {
			t.add(s.entries[x].Component, 1)
		}
		comp := t.top(1)[0]
		if comp == "" {
			continue
		}
		h, ok := by[comp]
		if !ok {
			h = &handled{routes: map[string][]string{}}
			by[comp] = h
			comps = append(comps, comp)
		}
		if kind == "job" {
			h.jobs = append(h.jobs, k)
		} else {
			r := roleOf(k)
			h.routes[r] = append(h.routes[r], k)
		}
	}
	layers := map[string]bool{}
	for _, comp := range comps {
		h := by[comp]
		most := len(h.jobs)
		for _, ks := range h.routes {
			most = max(most, len(ks))
		}
		if most >= layerFrom {
			layers[comp] = true
		}
	}
	for _, comp := range comps {
		if layers[comp] {
			continue // nothing joins through a layer
		}
		h := by[comp]
		reason := "same component " + short(comp)
		for _, r := range sortedKeys(h.routes) {
			ks := h.routes[r]
			for _, k := range ks[1:] {
				u.join(ks[0], k, reason)
			}
		}
		to := ""
		if rs := sortedKeys(h.routes); len(rs) > 0 {
			to = h.routes[rs[0]][0]
		} else {
			to = h.jobs[0]
		}
		for _, k := range h.jobs {
			if k != to {
				u.join(to, k, reason)
			}
		}
	}

	// a command, with the one component its flow reaches that handles routes or jobs. An operator
	// runs it, so it joins the administrators' routes where there are any.
	for _, k := range keys {
		if first[k].Kind != "command" {
			continue
		}
		reachedComps := map[string]bool{}
		for _, x := range byKey[k] {
			cs := []string{s.entries[x].Component}
			if r := s.reach[x]; r != nil {
				cs = append(cs, r.components...)
			}
			for _, c := range cs {
				if by[c] != nil && !layers[c] {
					reachedComps[c] = true
				}
			}
		}
		if len(reachedComps) != 1 {
			continue
		}
		comp := sortedKeys(reachedComps)[0]
		h := by[comp]
		to := ""
		for _, r := range sortedKeys(h.routes) {
			if r != "" {
				to = h.routes[r][0]
				break
			}
		}
		if to == "" {
			if rs := h.routes[""]; len(rs) > 0 {
				to = rs[0]
			} else {
				to = h.jobs[0]
			}
		}
		u.join(to, k, "its flow reaches "+short(comp))
	}

	// pages with the routes that name the same resource, across a proven API call; routes of the
	// page's own role when there are any. A page with none joins the routes its flows call, when all of
	// two or more calls go to one seed.
	api := map[string]bool{}
	for _, e := range s.m.Edges {
		if e.Label == callsAPI {
			api[e.From+">"+e.To] = true
		}
	}
	var routes []string
	for _, k := range keys {
		if first[k].Kind == "http" && resourceOf(k) != "" {
			routes = append(routes, k)
		}
	}
	for _, p := range keys {
		if first[p].Kind != "page" {
			continue
		}
		calls := newTally()
		for _, x := range byKey[p] {
			if r := s.reach[x]; r != nil {
				for _, route := range sortedKeys(r.routes) {
					if k, ok := keyOf[route]; ok {
						calls.add(k, 1)
					}
				}
			}
		}
		res := resourceOf(p)
		var same, named []string
		for _, r := range routes {
			if res != "" && resourceOf(r) == res && api[first[p].Container+">"+first[r].Container] {
				same = append(same, r)
			}
		}
		for _, r := range same {
			if roleOf(r) == roleOf(p) {
				named = append(named, r)
			}
		}
		if len(named) == 0 {
			named = same
		}
		for _, r := range named {
			reason := "by name: " + res
			if calls.count(r) > 0 {
				reason += ", confirmed by its calls"
			}
			u.join(p, r, reason)
		}
		if len(named) == 0 && calls.total() >= 2 && calls.size() == 1 {
			u.join(p, calls.top(1)[0], "by its calls")
		}
	}

	var roots []string
	groups := map[string][]string{}
	for _, k := range keys {
		r := u.find(k)
		if _, ok := groups[r]; !ok {
			roots = append(roots, r)
		}
		groups[r] = append(groups[r], k)
	}
	out := make([]keyGroup, 0, len(roots))
	for _, r := range roots {
		ks := append([]string(nil), groups[r]...)
		sort.Strings(ks)
		out = append(out, keyGroup{keys: ks, why: u.reasons(r)})
	}
	return out
}

// areaSeeds seeds the components of the container main.
func (s *seeding) areaSeeds(main string) {
	var comps []string
	isComp := map[string]bool{}
	for _, n := range s.m.Nodes {
		if n.Kind == core.Component && n.Parent == main {
			comps = append(comps, n.ID)
			isComp[n.ID] = true
		}
	}
	sort.Strings(comps)
	users, uses := map[string]map[string]bool{}, map[string]map[string]bool{}
	for _, e := range s.m.Edges {
		if e.Label == "uses" && isComp[e.From] && isComp[e.To] && e.From != e.To {
			add(users, e.To, e.From)
			add(uses, e.From, e.To)
		}
	}
	handles := map[string]*tally{}
	for _, e := range s.m.Entries {
		if isComp[e.Component] {
			if handles[e.Component] == nil {
				handles[e.Component] = newTally()
			}
			handles[e.Component].add(s.seedOf[e.ID], 1)
		}
	}
	need := max(sharedMin, (len(comps)-1+3)/4)
	u := newUnions(comps)

	var plumbing []string
	isPlumbing := map[string]bool{}
	for _, c := range comps {
		if handles[c] == nil && (len(users[c]) >= need || len(uses[c]) >= need || len(users[c]) == 0) {
			plumbing = append(plumbing, c)
			isPlumbing[c] = true
		}
	}
	for _, c := range plumbing[min(1, len(plumbing)):] {
		u.join(plumbing[0], c, "plumbing: shared or wiring, or where a process starts, and handles no entry point")
	}
	var bySeedOrder []string
	bySeed := map[string][]string{}
	for _, c := range comps {
		if handles[c] != nil {
			sd := handles[c].top(1)[0]
			if _, ok := bySeed[sd]; !ok {
				bySeedOrder = append(bySeedOrder, sd)
			}
			bySeed[sd] = append(bySeed[sd], c)
		}
	}
	for _, sd := range bySeedOrder {
		cs := bySeed[sd]
		for _, c := range cs[1:] {
			u.join(cs[0], c, "their entry points are in one seed")
		}
	}
	for range 3 { // a part only one seed uses belongs to it; three passes let a chain settle
		for _, c := range comps {
			if handles[c] != nil || len(users[c]) == 0 || isPlumbing[c] {
				continue
			}
			rootSet := map[string]bool{}
			for user := range users[c] {
				rootSet[u.find(user)] = true
			}
			if len(rootSet) == 1 && !rootSet[u.find(c)] {
				u.join(sortedKeys(rootSet)[0], c, "only that seed uses "+part(c))
			}
		}
	}
	groups := map[string][]string{}
	for _, c := range comps {
		r := u.find(c)
		groups[r] = append(groups[r], c)
	}
	s.areas, s.areaOf = map[string]areaSeed{}, main
	for _, root := range sortedKeys(groups) {
		id := part(root)
		if len(plumbing) > 0 && root == u.find(plumbing[0]) {
			id = "plumbing"
		}
		s.areas[id] = areaSeed{Key: root, Members: groups[root], Why: u.reasons(root)}
		s.areaIDs = append(s.areaIDs, id)
	}
}

// reaches is what each entry point's flow reaches.
func reaches(m core.Model) map[string]*reached {
	out := map[string]*reached{}
	for _, f := range m.Flows {
		parts := map[string]core.Participant{}
		for _, p := range f.Participants {
			parts[p.ID] = p
		}
		r := &reached{tables: map[string]map[string]bool{}, jobs: map[string]bool{}, routes: map[string]bool{}}
		seen := map[string]bool{}
		for _, p := range f.Participants {
			if p.Component != "" && !seen[p.Component] {
				seen[p.Component] = true
				r.components = append(r.components, p.Component)
			}
		}
		for _, st := range f.Steps {
			p := parts[st.To]
			switch p.Kind {
			case "table":
				add(r.tables, p.Name, st.Call)
			case "job":
				r.jobs[p.Name] = true
			case "route":
				r.routes[p.Element] = true
			}
		}
		out[f.Entry] = r
	}
	return out
}

// entryKey is the seed an entry point starts in.
func entryKey(e core.Entry) string {
	c := short(e.Container)
	switch e.Kind {
	case "http", "page":
		return c + " " + e.Kind + " " + pathSegment(e.Path)
	case "job":
		h := e.Handler
		if h == "" {
			h = e.ID
		}
		return c + " job " + strings.SplitN(h, ".", 2)[0]
	case "command":
		f := e.Prov.File
		return c + " command " + f[strings.LastIndex(f, "/")+1:]
	}
	return c + " " + e.Kind + " " + e.ID
}

// pathSegment is what a route or page is about: its first path segment after api, or a role's next.
func pathSegment(p string) string {
	var segs []string
	for _, s := range strings.Split(p, "/") {
		if s != "" && !strings.ContainsAny(s[:1], "{[(*") {
			segs = append(segs, s)
		}
	}
	if len(segs) > 0 && segs[0] == "api" {
		segs = segs[1:]
	}
	if len(segs) == 0 {
		return "/"
	}
	if roles[segs[0]] && len(segs) > 1 {
		return segs[0] + "/" + segs[1]
	}
	return segs[0]
}

// resourceOf is the resource a page or route key names: its last path segment, singular.
func resourceOf(key string) string {
	parts := strings.SplitN(key, " ", 3)
	seg := strings.ToLower(parts[len(parts)-1])
	seg = seg[strings.LastIndex(seg, "/")+1:]
	switch {
	case strings.HasSuffix(seg, "ies"):
		return strings.TrimSuffix(seg, "ies") + "y"
	case strings.HasSuffix(seg, "s") && !strings.HasSuffix(seg, "ss"):
		return strings.TrimSuffix(seg, "s")
	}
	return seg
}

// roleOf is the audience a page or route key is for — admin — or "".
func roleOf(key string) string {
	parts := strings.SplitN(key, " ", 3)
	seg, _, _ := strings.Cut(parts[len(parts)-1], "/")
	if roles[seg] {
		return seg
	}
	return ""
}

// short is an ID without its kind: "cmp:immich-server/asset" → "immich-server/asset".
func short(id string) string {
	if _, after, ok := strings.Cut(id, ":"); ok {
		return after
	}
	return id
}

// part is a component's name inside its container: "cmp:immich-server/asset" → "asset".
func part(id string) string {
	s := short(id)
	if _, after, ok := strings.Cut(s, "/"); ok {
		return after
	}
	return s
}

func add[V comparable](m map[string]map[V]bool, k string, v V) {
	if m[k] == nil {
		m[k] = map[V]bool{}
	}
	m[k][v] = true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// tally counts keys and remembers the order it first met them, so ties are broken the same way every
// run.
type tally struct {
	order []string
	n     map[string]int
}

func newTally() *tally { return &tally{n: map[string]int{}} }

func (t *tally) add(k string, by int) {
	if _, ok := t.n[k]; !ok {
		t.order = append(t.order, k)
	}
	t.n[k] += by
}

func (t *tally) count(k string) int { return t.n[k] }
func (t *tally) size() int          { return len(t.order) }

func (t *tally) total() int {
	sum := 0
	for _, v := range t.n {
		sum += v
	}
	return sum
}

// top is the n most counted keys, ties in the order first met; n ≤ 0 is all of them.
func (t *tally) top(n int) []string {
	keys := append([]string(nil), t.order...)
	sort.SliceStable(keys, func(i, j int) bool { return t.n[keys[i]] > t.n[keys[j]] })
	if n > 0 && len(keys) > n {
		keys = keys[:n]
	}
	return keys
}
