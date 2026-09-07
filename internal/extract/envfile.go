package extract

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// envEntry is one environment value and the line that set it. A service's environment is
// assembled from more than one file, so provenance travels with each value rather than being
// derived from where the service was declared.
type envEntry struct {
	Value any
	Prov  archdoc.Provenance
}

// serviceEnv assembles a service's environment from the Compose file and from the env_file
// entries it names (EXT-07).
//
// Only files that actually exist are read, and only files env_file names. A `.env.example` or
// `.env.production.sample` is deliberately *not* consulted for facts: Mastodon's sample sets
// REDIS_HOST=localhost and DB_HOST=/var/run/postgresql, which are correct defaults for a
// non-container deployment and wrong for the stack the compose file describes. Reading it would
// produce endpoints that contradict the diagram they appear on.
//
// Those files are still used for interpolation defaults — see environment() — because filling
// ${VAR} with the value the repository suggests is a different claim from asserting the value
// is a fact about the running system.
func serviceEnv(svc map[string]any, pos servicePos, root, rel string) map[string]envEntry {
	out := map[string]envEntry{}

	// env_file first: Compose's `environment` overrides it, so the inline values are written
	// second and win.
	for _, ref := range envFiles(svc["env_file"]) {
		path := filepath.Join(filepath.Dir(rel), ref)

		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			continue // absent is normal; a repository is usually not configured to run
		}

		for k, e := range parseEnvFile(content, path) {
			out[k] = e
		}
	}

	for k, v := range envValues(svc["environment"]) {
		out[k] = envEntry{Value: v, Prov: pos.env(k)}
	}

	return out
}

// envFiles normalises env_file to a list of paths. compose-go canonicalises the string and list
// forms into a list of {path, required} maps.
func envFiles(v any) []string {
	switch f := v.(type) {
	case string:
		return []string{f}
	case []any:
		out := make([]string, 0, len(f))
		for _, item := range f {
			switch e := item.(type) {
			case string:
				out = append(out, e)
			case map[string]any:
				if p, ok := e["path"].(string); ok {
					out = append(out, p)
				}
			}
		}
		return out
	default:
		return nil
	}
}

// parseEnvFile reads KEY=value lines and remembers which line each came from.
//
// Deliberately simple: no export prefixes, no multi-line values, no interpolation. A dotenv file
// that needs more than this is one archdoc should report rather than guess at.
func parseEnvFile(content []byte, rel string) map[string]envEntry {
	out := map[string]envEntry{}

	for i, line := range splitLines(content) {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}

		out[k] = envEntry{
			Value: strings.Trim(strings.TrimSpace(v), `"'`),
			Prov:  archdoc.Provenance{File: rel, Line: i + 1},
		}
	}

	return out
}
