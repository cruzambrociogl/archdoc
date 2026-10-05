package code

import (
	"strings"
	"sync"

	ts "github.com/odvcencio/gotreesitter"
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

// scriptImports finds what a TypeScript or JavaScript file imports: import and export-from
// statements, dynamic import(), and require().
func scriptImports(src []byte, lang string) ([]rawImport, bool) {
	return scriptImportsAt(src, lang, 0)
}

// scriptImportsAt is scriptImports for a script embedded at a line offset — a Svelte component's
// <script> block — so every line it cites is the line in the file.
func scriptImportsAt(src []byte, lang string, offset int) (out []rawImport, partial bool) {
	partial = parse(lang, src, func(root *ts.Node, l *ts.Language) {
		walk(root, func(n *ts.Node) {
			switch n.Type(l) {
			case "import_statement", "export_statement":
				for i := 0; i < n.ChildCount(); i++ {
					if c := n.Child(i); c.Type(l) == "string" {
						out = append(out, stringImport(c, src, offset))
					}
				}
			case "call_expression":
				if n.ChildCount() < 2 {
					return
				}
				fn := n.Child(0)
				if t := fn.Type(l); t != "import" && !(t == "identifier" && fn.Text(src) == "require") {
					return
				}
				args := n.Child(1)
				if args.Type(l) != "arguments" {
					return
				}
				for i := 0; i < args.ChildCount(); i++ {
					if a := args.Child(i); a.Type(l) == "string" {
						out = append(out, stringImport(a, src, offset))
						return
					}
				}
			}
		})
	})
	return out, partial
}

func stringImport(n *ts.Node, src []byte, offset int) rawImport {
	at := n.StartPoint()
	return rawImport{
		spec: strings.Trim(n.Text(src), "'\"`"),
		line: int(at.Row) + 1 + offset, column: int(at.Column) + 1,
	}
}

// svelteImports reads each <script> block of a Svelte component as TypeScript or JavaScript,
// by its lang attribute.
func svelteImports(src []byte) (out []rawImport, partial bool) {
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
		imps, p := scriptImportsAt(src[b.start:b.end], b.lang, b.offset)
		out = append(out, imps...)
		partial = partial || p
	}
	return out, partial
}

// pythonImports finds `import a.b` and `from .a import b, c`. For a from-import the names are
// kept: `from app.api.routes import items, users` imports two modules, not one package.
func pythonImports(src []byte) (out []rawImport, partial bool) {
	partial = parse("Python", src, func(root *ts.Node, l *ts.Language) {
		walk(root, func(n *ts.Node) {
			switch n.Type(l) {
			case "import_statement":
				for i := 0; i < n.NamedChildCount(); i++ {
					c := n.NamedChild(i)
					if c.Type(l) == "aliased_import" && c.NamedChildCount() > 0 {
						c = c.NamedChild(0)
					}
					if c.Type(l) == "dotted_name" {
						at := c.StartPoint()
						out = append(out, rawImport{spec: c.Text(src), line: int(at.Row) + 1, column: int(at.Column) + 1})
					}
				}
			case "import_from_statement":
				if n.NamedChildCount() == 0 {
					return
				}
				mod := n.NamedChild(0)
				at := mod.StartPoint()
				imp := rawImport{spec: mod.Text(src), line: int(at.Row) + 1, column: int(at.Column) + 1}
				for i := 1; i < n.NamedChildCount(); i++ {
					c := n.NamedChild(i)
					if c.Type(l) == "aliased_import" && c.NamedChildCount() > 0 {
						c = c.NamedChild(0)
					}
					if c.Type(l) == "dotted_name" {
						imp.names = append(imp.names, c.Text(src))
					}
				}
				out = append(out, imp)
			}
		})
	})
	return out, partial
}
