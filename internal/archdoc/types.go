package archdoc

import "fmt"

// Provenance is where a fact came from: the file that proves it, and the line within that
// file. Every fact carries exactly one — the file that won the merge.
//
// O-8 established that compose-go discards source positions, so extraction runs two passes
// and reconciles them by key path. See docs/decisions.md, 2026-08-26.
type Provenance struct {
	File   string `json:"file"`             // repository-relative
	Line   int    `json:"line"`             // 1-indexed; 0 means unknown
	Column int    `json:"column,omitempty"` // 1-indexed; 0 means unknown
}

// String renders provenance the way an editor expects: path/to/file.yml:13
func (p Provenance) String() string {
	if p.Line == 0 {
		return p.File
	}
	return fmt.Sprintf("%s:%d", p.File, p.Line)
}

// Known reports whether this provenance actually points somewhere. A fact whose provenance is
// unknown must not be emitted — P1 requires every element to be traceable to the file that
// proves it exists.
func (p Provenance) Known() bool {
	return p.File != "" && p.Line > 0
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

	// Source is the compose file discovery selected, repository-relative.
	Source string `json:"source"`

	// Considered lists every file discovery examined, whether or not it was chosen. Recorded
	// so a reader can see what was skipped rather than having to trust that nothing was.
	Considered []Candidate `json:"considered"`

	Services []Service `json:"services"`
}

// Candidate is a file discovery looked at, and what it decided about it.
type Candidate struct {
	File   string `json:"file"`
	Reason string `json:"reason"` // why it was chosen, or why it was not
	Chosen bool   `json:"chosen"`
}
