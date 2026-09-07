package extract

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// maxConfigSize bounds what is worth reading as a routing table. Gateway configs are small;
// anything larger is a data file that happens to be mounted.
const maxConfigSize = 1 << 20

// Routes reads gateway routing tables and returns the rules they declare (EXT-03, MDL-03).
//
// The discovery path is the interesting part. archdoc does not guess where a gateway keeps its
// configuration — it follows the repository's own bind mounts. `docker-compose.yml` says
// `./volumes/api/envoy/cds.yaml:/etc/envoy/cds.yaml`, so that file is, by the repository's own
// statement, this gateway's configuration. The mount line is the citation for *why* the file
// was read at all.
//
// Which services are gateways is decided by what their mounted files contain, not by their
// image name. Same principle as discovery: sniff for recall, then reject precisely. A gateway
// running an image no catalog knows is still a gateway.
func Routes(root, rel string, services []archdoc.Service) []archdoc.Route {
	dir := filepath.Dir(rel)

	var out []archdoc.Route
	for _, s := range services {
		for _, m := range s.Mounts {
			path := filepath.Clean(filepath.Join(dir, m.Source))
			if !worthReading(path) {
				continue
			}

			content, err := os.ReadFile(filepath.Join(root, path))
			if err != nil || len(content) > maxConfigSize {
				continue
			}

			out = append(out, parseGatewayConfig(content, s.Name, path)...)
		}
	}

	// Stable order: the mount order is already deterministic, but two files can declare the
	// same route and AC-7 requires the result not to depend on which was read first.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Gateway != out[j].Gateway {
			return out[i].Gateway < out[j].Gateway
		}
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return less(out[i].Prov, out[j].Prov)
	})

	return out
}

func less(a, b archdoc.Provenance) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	return a.Line < b.Line
}

func worthReading(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

// parseGatewayConfig tries each dialect archdoc understands. A file that is not a routing table
// yields nothing, which is the normal case — most mounts are data directories and seed scripts.
func parseGatewayConfig(content []byte, gateway, path string) []archdoc.Route {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil
	}

	if routes := envoyClusters(&doc, gateway, path); len(routes) > 0 {
		return routes
	}
	return kongServices(&doc, gateway, path)
}

// envoyClusters reads an Envoy CDS document. Each cluster names an upstream, and the upstream's
// socket address is a hostname the compose file usually also declares as a service.
//
// Supabase's `cds.yaml` is exactly this: seven clusters, seven services.
func envoyClusters(doc *yaml.Node, gateway, path string) []archdoc.Route {
	resources := lookup(doc, "resources")
	if resources == nil || resources.Kind != yaml.SequenceNode {
		return nil
	}

	var out []archdoc.Route
	for _, res := range resources.Content {
		if !typeIs(res, "Cluster") {
			continue
		}

		// The address sits several levels down, under load_assignment → endpoints →
		// lb_endpoints → endpoint → address → socket_address. Searching for the innermost
		// key rather than walking the whole path keeps this working across the layouts
		// Envoy accepts for the same thing.
		for _, sock := range findAll(res, "socket_address") {
			addr := lookup(sock, "address")
			if addr == nil || addr.Kind != yaml.ScalarNode {
				continue
			}
			if host := strings.TrimSpace(addr.Value); isRoutableHost(host) {
				out = append(out, archdoc.Route{
					Gateway: gateway,
					Target:  host,
					Config:  path,
					Prov:    archdoc.Provenance{File: path, Line: addr.Line, Column: addr.Column},
				})
			}
		}
	}

	return out
}

// kongServices reads a Kong declarative configuration: a list of services, each with a url or a
// host naming its upstream.
func kongServices(doc *yaml.Node, gateway, path string) []archdoc.Route {
	services := lookup(doc, "services")
	if services == nil || services.Kind != yaml.SequenceNode {
		return nil
	}
	// A Compose file also has a top-level `services`, and a gateway must never be handed one
	// by mistake. Kong's own version marker is the discriminator.
	if lookup(doc, "_format_version") == nil {
		return nil
	}

	var out []archdoc.Route
	for _, svc := range services.Content {
		node := lookup(svc, "url")
		if node == nil {
			node = lookup(svc, "host")
		}
		if node == nil || node.Kind != yaml.ScalarNode {
			continue
		}

		host := node.Value
		if i := strings.Index(host, "://"); i >= 0 {
			host = host[i+3:]
		}
		host = strings.TrimSpace(strings.SplitN(strings.SplitN(host, "/", 2)[0], ":", 2)[0])

		if isRoutableHost(host) {
			out = append(out, archdoc.Route{
				Gateway: gateway,
				Target:  host,
				Config:  path,
				Prov:    archdoc.Provenance{File: path, Line: node.Line, Column: node.Column},
			})
		}
	}

	return out
}

// isRoutableHost rejects the addresses a gateway uses to talk about itself. A listener bound to
// 0.0.0.0 and an admin endpoint on 127.0.0.1 are not routes to anything.
func isRoutableHost(host string) bool {
	return host != "" && !notAHost[host] && !strings.ContainsAny(host, "/ \t$")
}

// typeIs reports whether a node's Protobuf `@type` names the given kind. Envoy tags every
// resource this way, which is what makes clusters separable from listeners in one document.
func typeIs(n *yaml.Node, kind string) bool {
	t := lookup(n, "@type")
	return t != nil && strings.Contains(t.Value, kind)
}

// findAll returns every value node stored under the given key anywhere in a subtree.
func findAll(n *yaml.Node, key string) []*yaml.Node {
	if n == nil {
		return nil
	}

	var out []*yaml.Node
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				out = append(out, n.Content[i+1])
			}
			out = append(out, findAll(n.Content[i+1], key)...)
		}
		return out
	}

	for _, child := range n.Content {
		out = append(out, findAll(child, key)...)
	}
	return out
}

// mounts reads a service's bind mounts. compose-go canonicalises every short form into a map
// carrying source and target.
func mounts(v any, pos servicePos) []archdoc.Mount {
	list, ok := v.([]any)
	if !ok {
		return nil
	}

	out := make([]archdoc.Mount, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		source := asString(m["source"])
		if source == "" || asString(m["type"]) != "bind" {
			continue // a named volume holds data, not configuration the repository wrote
		}

		out = append(out, archdoc.Mount{
			Source: source,
			Target: asString(m["target"]),
			Prov:   pos.mount(i),
		})
	}
	return out
}
