package extract

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// notAHost are values that parse as hostnames but name no system. A loopback or wildcard
// address is the machine the container runs on, or a bind address — never a thing to draw. The
// survey found all four in the test subjects.
var notAHost = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"0.0.0.0":   true,
	"::1":       true,
	"::":        true,
}

// endpoints reads a service's environment and returns the network locations it names (MDL-04).
//
// This is the only part of the environment that survives into the FactSet. Everything else —
// passwords, JWT secrets, API keys, all four of which sit in plain sight in the subjects — is
// read and discarded here, so nothing downstream has to remember to redact it.
func endpoints(env map[string]envEntry) []archdoc.Endpoint {
	names := make([]string, 0, len(env))
	for k := range env {
		names = append(names, k)
	}
	sort.Strings(names) // Go randomises map iteration; AC-7 requires a stable order

	out := make([]archdoc.Endpoint, 0, len(names))
	for _, name := range names {
		entry := env[name]

		value, ok := entry.Value.(string)
		if !ok {
			continue // a number or a bool is not a location
		}

		ep, ok := parseEndpoint(name, value, env)
		if !ok {
			continue
		}

		// The line that set this value, which is not always the compose file: an env_file
		// entry cites the dotenv file it came from.
		ep.Prov = entry.Prov
		out = append(out, ep)
	}

	return out
}

// parseEndpoint recognises the two shapes a configured location takes: a URL, or a bare host in
// a variable whose name says it is one.
//
// Nothing else is inspected. A value could be a hostname without either signal, but guessing
// from the value alone would produce edges the repository never declared — and MDL-04 is a
// deterministic capability precisely because it does not guess.
func parseEndpoint(name, value string, env map[string]envEntry) (archdoc.Endpoint, bool) {
	value = strings.TrimSpace(value)

	if strings.Contains(value, "://") {
		return parseURL(name, value)
	}
	if isHostVar(name) {
		return parseHost(name, value, env)
	}
	return archdoc.Endpoint{}, false
}

func parseURL(name, value string) (archdoc.Endpoint, bool) {
	u, err := url.Parse(value)
	if err != nil {
		return archdoc.Endpoint{}, false
	}

	host := u.Hostname()
	if host == "" || notAHost[host] {
		return archdoc.Endpoint{}, false
	}

	// Hostname() has already dropped any user:password — the credential never reaches the
	// FactSet even when the URL that named the host carried one.
	ep := archdoc.Endpoint{Var: name, Scheme: u.Scheme, Host: host}
	if p, err := strconv.Atoi(u.Port()); err == nil {
		ep.Port = p
	}
	return ep, true
}

func parseHost(name, value string, env map[string]envEntry) (archdoc.Endpoint, bool) {
	if value == "" || notAHost[value] {
		return archdoc.Endpoint{}, false
	}
	// A path is a unix socket, not a host — Supabase's own database sets
	// POSTGRES_HOST=/var/run/postgresql. An unresolved variable is not a host either.
	if strings.ContainsAny(value, "/ \t$") {
		return archdoc.Endpoint{}, false
	}
	// The suffix rule is a convention, and conventions collide: SEED_SELF_HOST=true is a
	// flag whose name happens to end in HOST. A boolean or a bare number names no system.
	if isFlagValue(value) {
		return archdoc.Endpoint{}, false
	}

	ep := archdoc.Endpoint{Var: name, Host: value}
	if p, ok := siblingPort(name, env); ok {
		ep.Port = p
	}
	return ep, true
}

// isFlagValue reports whether a value is a boolean or a bare number rather than a host.
func isFlagValue(v string) bool {
	switch strings.ToLower(v) {
	case "true", "false", "yes", "no", "on", "off", "enabled", "disabled":
		return true
	}
	_, err := strconv.Atoi(v)
	return err == nil
}

// isHostVar reports whether a variable name declares that its value is a host. The suffix is
// the convention every subject follows: DB_HOST, PG_META_DB_HOST, GOTRUE_SMTP_HOST.
func isHostVar(name string) bool {
	return hasSuffixWord(name, "HOST") || hasSuffixWord(name, "HOSTNAME")
}

// siblingPort finds the port that goes with a host variable: DB_HOST is answered by DB_PORT.
func siblingPort(name string, env map[string]envEntry) (int, bool) {
	base := strings.TrimSuffix(strings.TrimSuffix(name, "NAME"), "HOST")

	switch v := env[base+"PORT"].Value.(type) {
	case string:
		p, err := strconv.Atoi(strings.TrimSpace(v))
		return p, err == nil
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// hasSuffixWord matches a suffix that stands alone — HOST or _HOST, but not GHOST.
func hasSuffixWord(name, suffix string) bool {
	if name == suffix {
		return true
	}
	return strings.HasSuffix(name, "_"+suffix)
}

// envValues normalises a service's environment to a map. Compose accepts a mapping or a list of
// "KEY=value" strings, and both appear in the subjects — Supabase uses the first, Mastodon's
// database the second.
func envValues(v any) map[string]any {
	switch env := v.(type) {
	case map[string]any:
		return env
	case []any:
		out := make(map[string]any, len(env))
		for _, item := range env {
			s, ok := item.(string)
			if !ok {
				continue
			}
			k, val, found := strings.Cut(s, "=")
			if !found {
				continue // "KEY" alone means "inherit from the shell", which names nothing
			}
			out[strings.TrimSpace(k)] = val
		}
		return out
	default:
		return nil
	}
}

// dependencies reads depends_on. compose-go canonicalises both the list and mapping forms into
// a mapping before this sees it.
func dependencies(v any, pos servicePos) []archdoc.Dependency {
	deps, ok := v.(map[string]any)
	if !ok {
		return nil
	}

	names := make([]string, 0, len(deps))
	for k := range deps {
		names = append(names, k)
	}
	sort.Strings(names)

	out := make([]archdoc.Dependency, 0, len(names))
	for _, n := range names {
		out = append(out, archdoc.Dependency{Service: n, Prov: pos.dependency(n)})
	}
	return out
}

// networks reads a service's network memberships. compose-go canonicalises the list form into
// a mapping before this sees it.
//
// Membership is the one boundary Compose states rather than implies: two services sharing no
// network cannot reach each other, whatever the rest of the file says.
func networks(v any, pos servicePos) []archdoc.NetworkRef {
	nets, ok := v.(map[string]any)
	if !ok {
		return nil
	}

	names := make([]string, 0, len(nets))
	for k := range nets {
		names = append(names, k)
	}
	sort.Strings(names)

	out := make([]archdoc.NetworkRef, 0, len(names))
	for _, n := range names {
		out = append(out, archdoc.NetworkRef{Name: n, Prov: pos.network(n)})
	}
	return out
}

// aliases collects the other hostnames a service answers to: its container_name, and any
// aliases declared on the networks it joins.
func aliases(svc map[string]any) []string {
	seen := map[string]bool{}

	if name := asString(svc["container_name"]); name != "" {
		seen[name] = true
	}

	if nets, ok := svc["networks"].(map[string]any); ok {
		for _, cfg := range nets {
			c, ok := cfg.(map[string]any)
			if !ok {
				continue
			}
			list, ok := c["aliases"].([]any)
			if !ok {
				continue
			}
			for _, a := range list {
				if s := asString(a); s != "" {
					seen[s] = true
				}
			}
		}
	}

	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out) // Go randomises map iteration; AC-7
	return out
}

// ports reads the published ports only.
//
// An unpublished port is internal plumbing and says nothing about the architecture. A published
// one is declared evidence that something outside the system reaches in, which is where actors
// come from.
func ports(v any, pos servicePos) []archdoc.Port {
	list, ok := v.([]any)
	if !ok {
		return nil
	}

	out := make([]archdoc.Port, 0, len(list))
	for i, item := range list {
		p, ok := item.(map[string]any)
		if !ok {
			continue
		}

		published := asString(p["published"])
		if published == "" {
			continue // not published: nothing outside can reach it
		}

		out = append(out, archdoc.Port{
			Published: published,
			Target:    asInt(p["target"]),
			Protocol:  asString(p["protocol"]),
			Prov:      pos.port(i),
		})
	}
	return out
}

// asString and asInt read values out of compose-go's untyped model, which is not JSON: a port
// arrives as a uint32, a published port as a string, and a size as a float. Formatting through
// fmt covers every numeric width without a type switch that would silently return zero for the
// one case it missed — which is exactly how a target port first came out as 0 here.
func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func asInt(v any) int {
	n, err := strconv.Atoi(asString(v))
	if err != nil {
		return 0
	}
	return n
}
