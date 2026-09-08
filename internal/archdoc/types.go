package archdoc

import "fmt"

// Provenance is where a fact came from: the file that proves it, and the line within that
// file. Every fact carries exactly one — the file that won the merge.
//
// O-8 established that compose-go discards source positions, so extraction runs two passes
// and reconciles them by key path. See docs/decisions.md, 2026-08-26.
type Provenance struct {
	// Origin says what kind of evidence this is. The zero value is Extraction, because a
	// fact read from a file is the ordinary case and the one every parser produces.
	Origin Origin `json:"origin,omitempty"`

	File   string `json:"file"`             // repository-relative
	Line   int    `json:"line"`             // 1-indexed; 0 means unknown
	Column int    `json:"column,omitempty"` // 1-indexed; 0 means unknown

	// Note carries the evidence for an origin that has no line: which catalog entry matched,
	// or which model run proposed it. A catalog fact is traceable — just not to a file.
	Note string `json:"note,omitempty"`
}

// String renders provenance the way an editor expects: path/to/file.yml:13. A fact with no
// line renders as its origin and note instead — "catalog: postgres".
func (p Provenance) String() string {
	if p.File == "" && p.Note != "" {
		return string(p.origin()) + ": " + p.Note
	}
	if p.Line == 0 {
		return p.File
	}
	return fmt.Sprintf("%s:%d", p.File, p.Line)
}

func (p Provenance) origin() Origin {
	if p.Origin == "" {
		return Extraction
	}
	return p.Origin
}

// Known reports whether this provenance actually points somewhere. A fact whose provenance is
// unknown must not be emitted — P1 requires every element to be traceable.
//
// What counts as traceable depends on the origin. A file said it, so it needs a line. A catalog
// or the model supplied it, so it needs to name what supplied it — AC-1 admits catalog
// provenance explicitly, and demanding a line for something no file stated would make the
// criterion unsatisfiable rather than strict.
func (p Provenance) Known() bool {
	switch p.origin() {
	case Catalog, Semantic:
		return p.Note != ""
	default:
		return p.File != "" && p.Line > 0
	}
}

// EvidenceKind records how strongly a node is attested. §3 of the product definition:
// declared and referenced are both real evidence, distinguished so the reader knows which
// boxes are proven infrastructure and which are declared intentions. Inferred must not exist
// in R1.
type EvidenceKind string

const (
	// Declared means the repository defines it — a service in a compose file.
	Declared EvidenceKind = "declared"
	// Referenced means the repository points at it without defining it — a host in an
	// environment variable. Proves the system talks to it, and nothing more.
	Referenced EvidenceKind = "referenced"
)

// Service is a container-level component found in configuration.
type Service struct {
	Name     string       `json:"name"`  // the compose service key, and the canonical identity
	Image    string       `json:"image"` // empty when the service is built rather than pulled
	Evidence EvidenceKind `json:"evidence"`
	Prov     Provenance   `json:"provenance"`

	// DependsOn is every service this one names as a dependency (EXT/MDL-02).
	DependsOn []Dependency `json:"depends_on,omitempty"`

	// Ports are the published ones only. An unpublished port is internal plumbing; a
	// published one is declared evidence that something outside reaches in, which is where
	// actors come from.
	Ports []Port `json:"ports,omitempty"`

	// Endpoints are network locations named in this service's environment (MDL-04).
	Endpoints []Endpoint `json:"endpoints,omitempty"`

	// Networks this service is attached to (MDL-09). Membership is a declared boundary: two
	// services on no common network cannot reach each other, whatever else the file says.
	Networks []NetworkRef `json:"networks,omitempty"`

	// Aliases are the other names this service answers to — container_name, and any network
	// aliases. A gateway's routing table names upstreams by hostname, and the hostname is not
	// always the service key: Supabase routes to `realtime-dev.supabase-realtime`, which the
	// compose file declares as that service's container_name. Resolving through aliases is
	// what stops a real service being drawn twice, once as itself and once as a stranger.
	Aliases []string `json:"aliases,omitempty"`

	// Mounts are the files and directories the repository hands to this container. They
	// matter because a gateway's routing table arrives this way, and the mount line is the
	// repository saying where its own configuration lives.
	Mounts []Mount `json:"mounts,omitempty"`
}

// Mount is one bind mount: a path in the repository, and where the container sees it.
type Mount struct {
	Source string     `json:"source"` // repository-relative
	Target string     `json:"target"` // the path inside the container
	Prov   Provenance `json:"provenance"`
}

// Route is a routing rule read from a gateway's own configuration (MDL-03/EXT-03).
//
// depends_on says a gateway starts after a service; a route says traffic actually reaches it.
// Only the second justifies an arrow, which is why routes are extracted separately rather than
// inferred from the compose file.
type Route struct {
	Gateway string `json:"gateway"` // the service whose configuration declared it
	Target  string `json:"target"`  // the host it routes to
	Path    string `json:"path,omitempty"`

	// Config is the file the rule was read from, repository-relative. It is not the same file
	// as the compose file, and a reader following the citation needs to land in the right one.
	Config string     `json:"config"`
	Prov   Provenance `json:"provenance"`
}

// NetworkRef is one service's membership of one network.
type NetworkRef struct {
	Name string     `json:"name"`
	Prov Provenance `json:"provenance"`
}

// Network is a network the file declares, and what it declares about it.
type Network struct {
	Name string `json:"name"`

	// Internal is Compose's own `internal: true` — the network has no outbound external
	// connectivity. It is the one trust boundary configuration states outright, so MDL-11
	// can record it without inferring anything.
	Internal bool `json:"internal,omitempty"`

	Prov Provenance `json:"provenance"`
}

// Dependency is one entry of a service's depends_on.
type Dependency struct {
	Service string     `json:"service"`
	Prov    Provenance `json:"provenance"`
}

// Port is a published port mapping: the host side is what makes it reachable from outside.
type Port struct {
	Published string     `json:"published"` // string, because Compose allows ranges and "8080"
	Target    int        `json:"target"`
	Protocol  string     `json:"protocol,omitempty"` // tcp when unstated
	Prov      Provenance `json:"provenance"`
}

// Endpoint is a network location named by an environment value — REDIS_URL, S3_ENDPOINT,
// DATABASE_HOST. It is how a repository points at something it does not itself declare.
//
// Only the location is kept. The raw environment map is deliberately not carried in the
// FactSet: Compose environments hold credentials, and the FactSet is written to disk. Carrying
// only parsed locations means there is nothing to redact later.
type Endpoint struct {
	Var    string     `json:"var"`              // the variable that named it
	Scheme string     `json:"scheme,omitempty"` // postgres, redis, https, s3 — "" when bare host
	Host   string     `json:"host"`
	Port   int        `json:"port,omitempty"`
	Prov   Provenance `json:"provenance"`
}

// FactSet is everything extraction found, and the contract between the deterministic half of
// the pipeline and everything downstream. It is produced by reading files only — no guessing
// and no model involved. See §11 of the product definition.
type FactSet struct {
	Root string `json:"root"` // absolute path to the repository that was scanned

	// Name is the system being documented — Compose's own project name where the file
	// declares one, and the repository's directory otherwise. It is what the context view
	// puts on the box that everything declared collapses into.
	Name string `json:"name"`

	// Source is the compose file discovery selected, repository-relative.
	Source string `json:"source"`

	// Considered lists every file discovery examined, whether or not it was chosen. Recorded
	// so a reader can see what was skipped rather than having to trust that nothing was.
	Considered []Candidate `json:"considered"`

	Services []Service `json:"services"`

	// Networks the file declares at the top level, in name order.
	Networks []Network `json:"networks,omitempty"`

	// Routes read from gateway configuration the compose file mounts (MDL-03).
	Routes []Route `json:"routes,omitempty"`
}

// Candidate is a file discovery looked at, and what it decided about it.
type Candidate struct {
	File   string `json:"file"`
	Reason string `json:"reason"` // why it was chosen, or why it was not
	Chosen bool   `json:"chosen"`
}
