package code

import (
	"path"
	"regexp"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// Dart is read for its files and imports — what components and their uses are made of. A Dart
// import is one directive on a line of its own, at the top of the file, naming a string:
//
//	import 'package:immich_mobile/utils/hash.dart';
//	export 'src/widget.dart';
//
// so the directives are read line by line, without a grammar; nothing else is read of a Dart
// file yet — no routes, no classes. Generated files (*.g.dart, *.freezed.dart, *.gr.dart) are a
// build's output, not the application's code.

var dartDirective = regexp.MustCompile(`^\s*(?:import|export)\s+['"]([^'"]+)['"]`)

func isDart(name string) bool {
	if !strings.HasSuffix(name, ".dart") || strings.HasSuffix(name, "_test.dart") {
		return false
	}
	for _, generated := range []string{".g.dart", ".freezed.dart", ".gr.dart", ".mocks.dart", ".drift.dart"} {
		if strings.HasSuffix(name, generated) {
			return false
		}
	}
	return true
}

func dartFacts(src []byte) (out facts) {
	for i, line := range strings.Split(string(src), "\n") {
		if m := dartDirective.FindStringSubmatchIndex(line); m != nil {
			out.imports = append(out.imports, rawImport{spec: line[m[2]:m[3]], line: i + 1, column: m[2] + 1})
		}
	}
	return out
}

// dartResolver ties an import to a file of the package: package:<its own name>/… is its lib/, a
// bare path is relative to the importing file. dart: and other packages are packages.
type dartResolver struct {
	name  string // the package's name, from pubspec.yaml
	lib   string
	known map[string]bool
}

func (r *dartResolver) resolve(file string, imp rawImport) []archdoc.Import {
	out := archdoc.Import{Spec: imp.spec}
	switch {
	case strings.HasPrefix(imp.spec, "dart:"):
		out.How, out.Package = archdoc.ByPackage, imp.spec
	case strings.HasPrefix(imp.spec, "package:"):
		pkg, rest, _ := strings.Cut(strings.TrimPrefix(imp.spec, "package:"), "/")
		if pkg != r.name {
			out.How, out.Package = archdoc.ByPackage, pkg
			break
		}
		out.How = archdoc.NoMatch
		if t := path.Join(r.lib, rest); r.known[t] {
			out.Target, out.How = t, archdoc.ByAlias
		} else if generated(rest) {
			return nil // a generated file: the build's, not read
		}
	default:
		out.How = archdoc.NoMatch
		if t := path.Clean(path.Join(path.Dir(file), imp.spec)); r.known[t] {
			out.Target, out.How = t, archdoc.ByPath
		} else if generated(imp.spec) {
			return nil
		}
	}
	return []archdoc.Import{out}
}

func generated(spec string) bool {
	return strings.HasSuffix(spec, ".dart") && !isDart(path.Base(spec))
}
