package archdoc

import (
	"path/filepath"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Match is one thing a query found: an element, a route, or a file and the component it is in.
type Match struct {
	// Key is what names it — an id, "METHOD /path", or a file path — and What says what it is.
	Key, What string
	Node      *Node
	Entry     *Entry
	File      string
}

// Find finds what a query names in a model, most exact first: an id; a route as METHOD /path or a
// path; a file; a name; and only then part of a name. The first kind of match that finds anything
// wins. root resolves a file path given relative to the working directory.
func Find(m Model, root, query string) []Match {
	q := strings.TrimSpace(query)
	lower := strings.ToLower(q)
	node := func(n core.Node) Match {
		n2 := n
		return Match{Key: n.ID, What: KindWords(m, n), Node: &n2}
	}
	entry := func(en core.Entry) Match {
		e2 := en
		return Match{Key: en.Method + " " + en.Path, What: en.Kind + " · " + en.Container, Entry: &e2}
	}
	try := func(steps ...func() []Match) []Match {
		for _, s := range steps {
			if out := s(); len(out) > 0 {
				return out
			}
		}
		return nil
	}
	return try(
		func() []Match {
			if n, ok := m.Node(q); ok {
				return []Match{node(n)}
			}
			for _, en := range m.Entries {
				if en.ID == q {
					return []Match{entry(en)}
				}
			}
			return nil
		},
		func() []Match {
			var out []Match
			for _, en := range m.Entries {
				if strings.EqualFold(en.Method+" "+en.Path, q) || en.Path == q {
					out = append(out, entry(en))
				}
			}
			return out
		},
		func() []Match {
			rel := filepath.ToSlash(q)
			if abs, err := filepath.Abs(q); err == nil {
				if base, err := filepath.Abs(root); err == nil {
					if r, err := filepath.Rel(base, abs); err == nil && !strings.HasPrefix(r, "..") {
						rel = filepath.ToSlash(r)
					}
				}
			}
			var out []Match
			seen := map[string]bool{}
			for _, n := range m.Nodes {
				if n.Kind != core.Component {
					continue
				}
				for _, f := range n.Files {
					if (f == rel || strings.HasSuffix(f, "/"+rel)) && !seen[f] {
						seen[f] = true
						n2 := n
						out = append(out, Match{Key: f, What: "file in " + n.Name, Node: &n2, File: f})
					}
				}
			}
			return out
		},
		func() []Match {
			var out []Match
			for _, n := range m.Nodes {
				if strings.EqualFold(n.Name, q) && n.Kind != core.Module {
					out = append(out, node(n))
				}
			}
			return out
		},
		func() []Match {
			var out []Match
			for _, n := range m.Nodes {
				if n.Kind != core.Module && (strings.Contains(strings.ToLower(n.Name), lower) || strings.Contains(strings.ToLower(n.ID), lower)) {
					out = append(out, node(n))
				}
			}
			for _, en := range m.Entries {
				if strings.Contains(strings.ToLower(en.Method+" "+en.Path), lower) || strings.Contains(strings.ToLower(en.Handler), lower) {
					out = append(out, entry(en))
				}
			}
			return out
		},
	)
}

// KindWords says what kind of element a node is, and what it is in.
func KindWords(m Model, n Node) string {
	what := string(n.Kind)
	if n.Parent != "" {
		if p, ok := m.Node(n.Parent); ok {
			what += " in " + p.Name
		}
	}
	return what
}
