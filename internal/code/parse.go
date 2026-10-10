package code

import (
	"strings"
	"sync"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars/c_sharp"
	"github.com/odvcencio/gotreesitter/grammars/java"
	"github.com/odvcencio/gotreesitter/grammars/javascript"
	"github.com/odvcencio/gotreesitter/grammars/python"
	"github.com/odvcencio/gotreesitter/grammars/svelte"
	"github.com/odvcencio/gotreesitter/grammars/tsx"
	"github.com/odvcencio/gotreesitter/grammars/typescript"
)

// One parser per grammar, made on first use and kept: a parser is not safe for concurrent use,
// and archdoc reads files one at a time.
var (
	parseMu sync.Mutex
	parsers = map[string]*ts.Parser{}
)

func language(name string) *ts.Language {
	switch name {
	case "TypeScript":
		return typescript.Language()
	case "TSX":
		return tsx.Language()
	case "Python":
		return python.Language()
	case "Svelte":
		return svelte.Language()
	case "Java":
		return java.Language()
	case "C#":
		return c_sharp.Language()
	default:
		return javascript.Language()
	}
}

// parse runs fn over the tree of src. partial reports a tree the parser had to recover in; a file
// the parser rejects outright reads as partial with nothing in it.
func parse(lang string, src []byte, fn func(root *ts.Node, l *ts.Language)) (partial bool) {
	parseMu.Lock()
	defer parseMu.Unlock()
	l := language(lang)
	p := parsers[lang]
	if p == nil {
		p = ts.NewParser(l)
		parsers[lang] = p
	}
	tree, err := p.Parse(src)
	if err != nil || tree == nil {
		return true
	}
	defer tree.Release()
	root := tree.RootNode()
	fn(root, l)
	return root.HasErrorOrMissing()
}

// walk visits every node under n, depth first, in source order.
func walk(n *ts.Node, visit func(*ts.Node)) {
	visit(n)
	for i := 0; i < n.ChildCount(); i++ {
		walk(n.Child(i), visit)
	}
}

// svelteFacts reads each <script> block of a Svelte component as TypeScript or JavaScript, by its
// lang attribute, citing every fact at its line in the .svelte file.
func svelteFacts(src []byte, file string) (out facts, partial bool) {
	type block struct {
		lang   string
		start  uint32
		end    uint32
		offset int
	}
	var blocks []block
	partial = parse("Svelte", src, func(root *ts.Node, l *ts.Language) {
		walk(root, func(n *ts.Node) {
			if n.Type(l) != "script_element" {
				return
			}
			b := block{lang: "JavaScript"}
			for i := 0; i < n.ChildCount(); i++ {
				c := n.Child(i)
				switch c.Type(l) {
				case "start_tag":
					attrs := c.Text(src)
					if strings.Contains(attrs, `lang="ts"`) || strings.Contains(attrs, `lang='ts'`) || strings.Contains(attrs, `lang="typescript"`) {
						b.lang = "TypeScript"
					}
				case "raw_text":
					b.start, b.end, b.offset = c.StartByte(), c.EndByte(), int(c.StartPoint().Row)
				}
			}
			if b.end > b.start {
				blocks = append(blocks, b)
			}
		})
	})
	for _, b := range blocks {
		f, p := scriptFacts(src[b.start:b.end], file, b.lang, b.offset)
		out.add(f)
		partial = partial || p
	}
	return out, partial
}
