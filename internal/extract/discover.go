// Package extract turns configuration files into facts, each carrying the file and line that
// proves it.
package extract

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// skipDirs are never walked. Not correctness — the survey measured ~17k tracked files on
// Supabase, and the cost is in the walk rather than the parse.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, ".next": true, "target": true, "__pycache__": true,
}

// Discover finds every Compose file in root and decides which one describes the architecture.
//
// Two survey findings shape this. A filename glob is not enough: Immich keeps two real Compose
// files at .devcontainer/server/container-compose-overrides.yml, which matches no conventional
// pattern, and a glob written for the survey missed both. But content-sniffing alone is not
// enough either: docker/hwaccel.ml.yml has a top-level services: key and parses cleanly, yet
// defines hardware-acceleration fragments named cpu, armnn and rknn that are not deployable
// services. Recall comes from sniffing; precision comes from rejecting fragments.
func Discover(root string) ([]archdoc.Candidate, error) {
	var found []archdoc.Candidate

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable directory is not a reason to abandon the scan
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if ext := filepath.Ext(path); ext != ".yml" && ext != ".yaml" {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}

		kind, reason := classify(path)
		if kind == notCompose {
			return nil
		}

		found = append(found, archdoc.Candidate{File: rel, Reason: reason})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(found, func(i, j int) bool { return found[i].File < found[j].File })

	if i := selectCanonical(root, found); i >= 0 {
		found[i].Chosen = true
		found[i].Reason = "selected — " + found[i].Reason
	}

	return found, nil
}

type kind int

const (
	notCompose kind = iota
	fragment
	deployable
)

// classify decides whether a YAML file is a deployable Compose file, a fragment, or neither.
func classify(path string) (kind, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return notCompose, ""
	}

	// Compose's !reset and !override are custom tags. A stock decoder rejects the document
	// rather than ignoring them, so strip the tags before sniffing — this pass only needs the
	// shape, and the semantic pass uses compose-go, which understands them properly.
	var doc yaml.Node
	if err := yaml.Unmarshal(stripComposeTags(b), &doc); err != nil {
		return notCompose, ""
	}

	services := lookup(&doc, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return notCompose, ""
	}

	// A fragment defines services that can never run: no image to pull and nothing to build.
	// hwaccel.ml.yml is the worked example — its "services" are device and volume snippets
	// meant to be pulled in by extends.
	runnable := 0
	for i := 0; i+1 < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		if lookup(svc, "image") != nil || lookup(svc, "build") != nil {
			runnable++
		}
	}

	if runnable == 0 {
		return fragment, "fragment — no service declares an image or a build"
	}

	return deployable, "deployable — " + plural(runnable, "service") + " with an image or build"
}

// selectCanonical picks the Compose file that describes the architecture, and returns its
// index in candidates, or -1.
//
// Supabase declares the answer outright: COMPOSE_FILE in .env is a native Compose variable
// holding a colon-separated list, base file first, and its own run.sh maintains it. Reading it
// is a standards-based resolution rather than a guess, so it is tried first. Immich offers no
// such signal, so convention decides: an unsuffixed name beats a suffixed one, and a shallower
// path beats a deeper one.
func selectCanonical(root string, candidates []archdoc.Candidate) int {
	deployables := []int{}
	for i, c := range candidates {
		if strings.HasPrefix(c.Reason, "deployable") {
			deployables = append(deployables, i)
		}
	}

	if len(deployables) == 0 {
		return -1
	}
	if len(deployables) == 1 {
		return deployables[0]
	}

	// Declared: COMPOSE_FILE in a .env beside any candidate.
	if named := composeFileVar(root, candidates, deployables); named >= 0 {
		candidates[named].Reason = "declared by COMPOSE_FILE in .env"
		return named
	}

	// Convention: score each candidate, lowest wins.
	best, bestScore := -1, 1<<30
	for _, i := range deployables {
		s := conventionScore(candidates[i].File)
		if s < bestScore {
			best, bestScore = i, s
		}
	}

	return best
}

// composeFileVar looks for COMPOSE_FILE in a .env sitting beside a candidate, and returns the
// index of the base file it names.
func composeFileVar(root string, candidates []archdoc.Candidate, deployables []int) int {
	seen := map[string]bool{}

	for _, i := range deployables {
		dir := filepath.Dir(candidates[i].File)
		if seen[dir] {
			continue
		}
		seen[dir] = true

		b, err := os.ReadFile(filepath.Join(root, dir, ".env"))
		if err != nil {
			// .env is often gitignored; .env.example carries the same declaration.
			b, err = os.ReadFile(filepath.Join(root, dir, ".env.example"))
			if err != nil {
				continue
			}
		}

		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "COMPOSE_FILE=") {
				continue
			}

			value := strings.Trim(strings.TrimPrefix(line, "COMPOSE_FILE="), `"' `)
			base, _, _ := strings.Cut(value, ":") // base file first
			want := filepath.Join(dir, strings.TrimSpace(base))

			for _, j := range deployables {
				if candidates[j].File == want {
					return j
				}
			}
		}
	}

	return -1
}

// conventionScore ranks a candidate path; lower is better.
func conventionScore(path string) int {
	base := filepath.Base(path)
	score := strings.Count(path, string(filepath.Separator)) // shallower wins

	switch base {
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		// Unsuffixed: the canonical name.
	default:
		score += 10 // docker-compose.dev.yml, .prod.yml, overrides, and so on
	}

	// Directories that signal a stack which is not the product being documented.
	for _, marker := range []string{".devcontainer", "e2e", "test", "example", "dev"} {
		if strings.Contains(path, marker) {
			score += 5
		}
	}

	return score
}

// lookup returns the value node for key in a mapping, or nil.
func lookup(n *yaml.Node, key string) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}

	return nil
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return itoa(n) + " " + word + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}

	return string(b)
}
