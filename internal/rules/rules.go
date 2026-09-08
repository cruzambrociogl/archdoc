// Package rules reads `rules.yaml` — the corrections a person makes to a generated model, and
// the only ones that survive regeneration.
//
// It exists because O-4 settled that automatic attachment does not work. Four strategies for
// matching interface contracts to services were tested against six specifications across two
// repositories, and directory proximity — the obvious one — failed on both. Rules are not a
// fallback for when extraction is weak; they are the primary mechanism for the judgements a
// repository does not contain.
//
// A rule compiles to operations, the same vocabulary the semantic layer produces, checked by the
// same validator. Rules carry no restriction on what they may change: a person is accountable,
// and the file records what they said, at a line.
package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/validate"
)

// Name is where archdoc looks. One file, at the root of the repository being documented.
const Name = "rules.yaml"

// A Rule is one correction: what it matches, and what it does.
type Rule struct {
	Line int

	// Match selects nodes. An empty matcher matches nothing — a rule that applies to
	// everything is almost always a mistake, and RUL-05 reports it as unreachable rather
	// than silently rewriting the whole model.
	Match Match

	// Edge selects a relationship instead of a node.
	Edge *EdgeRef

	Set     map[string]string
	Exclude bool
	Remove  bool
}

// Match is RUL-02: the expressions a rule may select on. Name and Image accept globs.
type Match struct {
	Name  string
	Image string
	Kind  string
}

func (m Match) empty() bool { return m.Name == "" && m.Image == "" && m.Kind == "" }

type EdgeRef struct {
	From string
	To   string
}

// File is a parsed rules.yaml.
type File struct {
	Path  string // repository-relative, for provenance
	Rules []Rule
}

// Load reads rules.yaml from a repository root. A missing file is not an error: most
// repositories have no corrections to make, and requiring one would be a barrier to the first
// run rather than a feature.
func Load(root string) (*File, error) {
	rel := Name
	content, err := os.ReadFile(filepath.Join(root, rel))
	if os.IsNotExist(err) {
		return &File{Path: rel}, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(content, rel)
}

// Parse reads the document, keeping the line each rule was written on. Provenance for a
// correction is the line a person wrote it on, exactly as provenance for a fact is the line a
// file declared it on.
func Parse(content []byte, rel string) (*File, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}

	f := &File{Path: rel}
	if len(doc.Content) == 0 {
		return f, nil
	}

	list := lookup(doc.Content[0], "rules")
	if list == nil || list.Kind != yaml.SequenceNode {
		return f, nil
	}

	for _, item := range list.Content {
		r := Rule{Line: item.Line, Set: map[string]string{}}

		if m := lookup(item, "match"); m != nil {
			r.Match = Match{
				Name:  scalar(m, "name"),
				Image: scalar(m, "image"),
				Kind:  scalar(m, "kind"),
			}
		}
		if e := lookup(item, "edge"); e != nil {
			r.Edge = &EdgeRef{From: scalar(e, "from"), To: scalar(e, "to")}
		}
		if s := lookup(item, "set"); s != nil && s.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(s.Content); i += 2 {
				r.Set[s.Content[i].Value] = s.Content[i+1].Value
			}
		}
		r.Exclude = scalar(item, "exclude") == "true"
		r.Remove = scalar(item, "remove") == "true"

		f.Rules = append(f.Rules, r)
	}

	return f, nil
}

// Compile turns rules into operations against a model, and reports the rules that could not be
// used (RUL-05).
//
// It needs the FactSet as well as the model because a rule may match on image, and the image is
// a fact about a container rather than a property of the C4 element derived from it. Passing
// both keeps `Node` free of a field that exists only for matching.
func (f *File) Compile(facts *archdoc.FactSet, m archdoc.Model) ([]archdoc.Op, validate.Result) {
	var (
		ops []archdoc.Op
		res validate.Result
	)

	images := map[string]string{}
	if facts != nil {
		for _, s := range facts.Services {
			images["svc:"+s.Name] = s.Image
		}
	}

	// Which rule last set which field on which element, so a later rule quietly overriding an
	// earlier one is reported rather than discovered.
	type claim struct{ rule, value string }
	claimed := map[string]claim{}

	for _, r := range f.Rules {
		id := fmt.Sprintf("%s:%d", f.Path, r.Line)
		prov := archdoc.Provenance{Origin: archdoc.Rules, File: f.Path, Line: r.Line}

		if r.Edge != nil {
			ops = append(ops, f.edgeOps(r, prov)...)
			continue
		}

		if r.Match.empty() {
			res.Add("RUL-05", validate.Warning, id,
				"rule selects nothing — a match on no criteria is almost never intended", prov)
			continue
		}

		targets := f.match(r.Match, m, images)
		if len(targets) == 0 {
			// RUL-05. A rule that matches nothing is usually a typo or a service that has
			// been renamed, and it fails silently unless someone says so.
			res.Add("RUL-05", validate.Warning, id, "rule matches no element", prov)
			continue
		}

		for _, target := range targets {
			if r.Exclude {
				ops = append(ops, archdoc.Op{
					Kind: archdoc.Exclude, Target: target,
					Origin: archdoc.Rules, Prov: prov,
				})
			}

			for _, field := range sortedKeys(r.Set) {
				kind, ok := opFor(field)
				if !ok {
					res.Add("RUL-01", validate.Error, id,
						fmt.Sprintf("unknown field %q", field), prov)
					continue
				}

				key := target + "/" + field
				if prev, seen := claimed[key]; seen && prev.value != r.Set[field] {
					res.Add("RUL-05", validate.Warning, id, fmt.Sprintf(
						"overrides %s, which set %s on %s to %q; the later rule wins",
						prev.rule, field, target, prev.value), prov)
				}
				claimed[key] = claim{id, r.Set[field]}

				ops = append(ops, archdoc.Op{
					Kind: kind, Target: target, Value: r.Set[field],
					Origin: archdoc.Rules, Prov: prov,
				})
			}
		}
	}

	res.Sort()
	return ops, res
}

func (f *File) edgeOps(r Rule, prov archdoc.Provenance) []archdoc.Op {
	if r.Remove {
		return []archdoc.Op{{
			Kind: archdoc.RemoveEdge, Target: "svc:" + r.Edge.From, To: "svc:" + r.Edge.To,
			Origin: archdoc.Rules, Prov: prov,
		}}
	}

	label := r.Set["label"]
	if label == "" {
		label = "connects to"
	}

	// An edge a person asserts is an addition, not a relabelling: rules are the answer to
	// "extraction cannot see this relationship", which O-4 and the Immich result both showed
	// is the common case.
	return []archdoc.Op{{
		Kind: archdoc.AddEdge, Target: "svc:" + r.Edge.From, To: "svc:" + r.Edge.To,
		Value: label, Origin: archdoc.Rules, Prov: prov,
	}}
}

// match returns the node IDs a matcher selects, in model order so the result is deterministic.
func (f *File) match(m Match, model archdoc.Model, images map[string]string) []string {
	var out []string

	for _, n := range model.Nodes {
		if m.Name != "" && !glob(m.Name, n.Name) {
			continue
		}
		if m.Kind != "" && string(n.Kind) != m.Kind {
			continue
		}
		if m.Image != "" && !glob(m.Image, images[n.ID]) {
			continue
		}
		out = append(out, n.ID)
	}

	return out
}

// glob matches a pattern that may contain `*`. An exact string with no wildcard is compared
// directly, which is the common case and the one worth being obviously correct.
func glob(pattern, value string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}
	ok, err := filepath.Match(pattern, value)
	return err == nil && ok
}

// opFor maps a rules.yaml field onto the operation vocabulary. Rules and the semantic layer
// speak the same language; only their permissions differ.
func opFor(field string) (archdoc.OpKind, bool) {
	switch field {
	case "name":
		return archdoc.SetName, true
	case "description":
		return archdoc.SetDescription, true
	case "technology":
		return archdoc.SetTechnology, true
	case "kind":
		return archdoc.SetKind, true
	case "group":
		return archdoc.Group, true
	default:
		return "", false
	}
}

func lookup(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func scalar(n *yaml.Node, key string) string {
	if v := lookup(n, key); v != nil && v.Kind == yaml.ScalarNode {
		return v.Value
	}
	return ""
}

// sortedKeys keeps operation order stable. Go randomises map iteration, and two runs that apply
// the same rules in a different order can produce different output — AC-7 and AC-5 both depend
// on this.
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Small and fixed; a plain insertion sort avoids importing sort for five keys.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
