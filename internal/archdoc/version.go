// Package archdoc holds the core types shared across the engine.
//
// The types that matter — Fact, Provenance, FactSet, Node, Edge, Model and Version — are
// deliberately not defined yet. Their shape depends on O-8: whether file-and-line provenance
// survives the Compose merge, which decides if a Fact carries one source position or two.
// They also propagate into model.json, a committed deliverable, so the shape is close to
// permanent. See docs/decisions.md, 2026-08-24.
package archdoc

import "runtime/debug"

// Version is the release identifier, set at build time with:
//
//	-ldflags "-X github.com/cruzambrociogl/archdoc/internal/archdoc.Version=v0.1.0"
//
// It is "dev" in an ordinary build.
var Version = "dev"

// BuildInfo describes the binary that is running. Every generated file is stamped with it
// (OUT-04), so a document can always be traced back to the build that produced it.
type BuildInfo struct {
	Version  string // release identifier, or "dev"
	Revision string // VCS revision, when the build embedded one
	Modified bool   // true if the working tree was dirty at build time
}

// Build reports the running binary's identity.
//
// Revision and Modified come from the VCS stamps the Go toolchain embeds automatically. They
// are empty when the build had no VCS information — a plain "go run", for instance — so
// callers must handle absence rather than assume a revision exists.
func Build() BuildInfo {
	b := BuildInfo{Version: Version}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return b
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			b.Revision = s.Value
		case "vcs.modified":
			b.Modified = s.Value == "true"
		}
	}

	return b
}

// String renders the build for display: "dev (a1b2c3d, modified)".
func (b BuildInfo) String() string {
	s := b.Version

	if b.Revision != "" {
		short := b.Revision
		if len(short) > 7 {
			short = short[:7]
		}

		s += " (" + short
		if b.Modified {
			s += ", modified"
		}
		s += ")"
	}

	return s
}
