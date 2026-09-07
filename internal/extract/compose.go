package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

func init() {
	// compose-go warns on every unset variable through logrus. That is right for a tool about
	// to start containers and noise for one reading a repository, where unset variables are
	// the normal case. Coverage is reported through the FactSet instead.
	logrus.SetOutput(io.Discard)
}

// composeTags are Compose's own additions to YAML. A stock decoder rejects a document
// containing them, so the position pass strips them; compose-go handles them properly in the
// semantic pass.
var composeTags = [][]byte{[]byte("!override"), []byte("!reset")}

// Scan reads a repository and returns everything extraction can prove about it.
//
// Two passes, which is what O-8 settled. compose-go resolves merge and interpolation semantics
// correctly but discards source positions; a raw yaml.v3 parse recovers them, addressable by
// key path. Neither pass alone is sufficient: the first knows what is true, the second knows
// where it was written.
func Scan(root string) (*archdoc.FactSet, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	candidates, err := Discover(abs)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}

	fs := &archdoc.FactSet{Root: abs, Name: filepath.Base(abs), Considered: candidates}

	chosen := ""
	for _, c := range candidates {
		if c.Chosen {
			chosen = c.File
			break
		}
	}

	if chosen == "" {
		return fs, nil // nothing to extract; the candidate list still explains why
	}

	fs.Source = chosen

	services, name, nets, err := extractServices(abs, chosen)
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", chosen, err)
	}
	fs.Services = services
	fs.Networks = nets

	// Routes come from files the compose file mounts, so they can only be read once the
	// services and their mounts are known.
	fs.Routes = Routes(abs, chosen, services)

	// Compose's own project name beats the directory: it is declared rather than incidental.
	if name != "" {
		fs.Name = name
	}

	return fs, nil
}

// extractServices runs both passes over one Compose file and reconciles them. It also returns
// the project name the file declares, if any, and the networks it declares.
func extractServices(root, rel string) ([]archdoc.Service, string, []archdoc.Network, error) {
	path := filepath.Join(root, rel)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", nil, err
	}

	// Pass 1 — semantics. compose-go applies interpolation and the Compose merge rules.
	model, err := loadModel(path, content, filepath.Dir(path))
	if err != nil {
		return nil, "", nil, err
	}

	// Pass 2 — positions. The raw document, addressable by key path.
	positions := readPositions(content, rel)

	raw, _ := model["services"].(map[string]any)
	services := make([]archdoc.Service, 0, len(raw))

	for name, v := range raw {
		svc, _ := v.(map[string]any)
		image, _ := svc["image"].(string)
		pos := positions[name]

		services = append(services, archdoc.Service{
			Name:      name,
			Image:     image,
			Evidence:  archdoc.Declared,
			Prov:      pos.Decl,
			DependsOn: dependencies(svc["depends_on"], pos),
			Ports:     ports(svc["ports"], pos),
			Endpoints: endpoints(serviceEnv(svc, pos, root, rel)),
			Networks:  networks(svc["networks"], pos),
			Mounts:    mounts(svc["volumes"], pos),
			Aliases:   aliases(svc),
		})
	}

	// Go randomises map iteration. Sorting here is not cosmetic: AC-7 requires five
	// consecutive scans to produce byte-identical output.
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })

	name, _ := model["name"].(string)

	return services, name, declaredNetworks(model["networks"], content, rel), nil
}

// declaredNetworks reads the top-level networks block.
//
// Compose's own `internal: true` is the one trust boundary a configuration file states outright
// rather than implies — a network with no outbound external connectivity. MDL-11 records
// declared boundaries only, and this is what "declared" looks like.
func declaredNetworks(v any, content []byte, rel string) []archdoc.Network {
	nets, ok := v.(map[string]any)
	if !ok {
		return nil
	}

	positions := map[string]archdoc.Provenance{}
	var doc yaml.Node
	if err := yaml.Unmarshal(stripComposeTags(content), &doc); err == nil {
		readNames(rel, lookup(&doc, "networks"), positions)
	}

	names := make([]string, 0, len(nets))
	for k := range nets {
		names = append(names, k)
	}
	sort.Strings(names)

	out := make([]archdoc.Network, 0, len(names))
	for _, n := range names {
		net := archdoc.Network{Name: n, Prov: positions[n]}
		if cfg, ok := nets[n].(map[string]any); ok {
			net.Internal, _ = cfg["internal"].(bool)
		}
		out = append(out, net)
	}
	return out
}

// loadModel runs compose-go's merge and interpolation and returns the merged document as a
// plain map.
//
// LoadModelWithContext is used rather than LoadWithContext deliberately. The typed loader
// insists that every value be a valid runtime spec, so an unset variable in a volume mount —
// "${DB_DATA_LOCATION}:/var/lib/postgresql/data" with no .env present — aborts the whole load.
// That is correct for a tool about to start containers and wrong for one documenting a
// repository it will never run. The untyped model keeps the merge semantics, which is the part
// worth having, and tolerates a stack that could not currently boot.
func loadModel(path string, content []byte, workdir string) (map[string]any, error) {
	env, err := environment(workdir)
	if err != nil {
		return nil, err
	}

	return loader.LoadModelWithContext(context.Background(), types.ConfigDetails{
		WorkingDir:  workdir,
		ConfigFiles: []types.ConfigFile{{Filename: path, Content: content}},
		Environment: env,
	}, func(o *loader.Options) {
		o.SetProjectName("archdoc", true)
		o.SkipValidation = true
		o.SkipConsistencyCheck = true
		o.SkipNormalization = true
		o.ResolvePaths = false
	})
}

// stripComposeTags removes Compose's custom YAML tags so a stock decoder will parse the
// document. Only the position pass needs this — meaning comes from compose-go.
func stripComposeTags(b []byte) []byte {
	for _, tag := range composeTags {
		b = bytes.ReplaceAll(b, tag, []byte(""))
	}
	return b
}

// environment reads a .env beside the Compose file. Absent is normal, and not an error: a
// repository being documented has usually never been configured to run.
func environment(dir string) (map[string]string, error) {
	env := map[string]string{}

	// Order matters: a real .env wins, then the conventional samples. Immich names its sample
	// example.env rather than .env.example, which is why the list is not two entries.
	for _, name := range []string{".env", ".env.example", "example.env", ".env.sample"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}

		for _, line := range splitLines(b) {
			if line == "" || line[0] == '#' {
				continue
			}
			if k, v, ok := cut(line, '='); ok {
				if _, exists := env[k]; !exists {
					env[k] = v
				}
			}
		}

		break // .env wins over .env.example
	}

	return env, nil
}

func splitLines(b []byte) []string {
	raw := bytes.Split(b, []byte("\n"))
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		out = append(out, string(bytes.TrimSpace(l)))
	}
	return out
}

func cut(s string, sep byte) (string, string, bool) {
	for i := range len(s) {
		if s[i] == sep {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
