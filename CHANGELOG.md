# Changelog

What changed in archdoc, per tagged version, for those who use it. Newest first. Decisions and their
reasons are in `docs/decisions.md`; what is done against the plan is in `PROGRESS.md`; this file says
what is different for a user.

## Unreleased

- **The app answers questions first.** One shell for the local app and published sites: Overview,
  What changed, Map, What it does, Reference and Settings, each named in plain words. The Overview
  says what the system is in one paragraph — the repository's own description where its manifest
  has one, then how its parts call each other — with one picture and one sentence on how far to trust
  it. What changed tells each version in a sentence, grouped by what the changes mean. What it does
  turns routes into sentences. Select any part to see what it touches. Old links still open.
- **Manifest descriptions are quoted.** A `description` in package.json, pyproject.toml or
  pubspec.yaml becomes the part's description, and the root manifest's the system's — cited at its
  line, as read, never interpreted.
- **archdoc is a Go library as well as a command.** `import "github.com/cruzambrociogl/archdoc"`:
  `Generate` documents a repository, `Compare` and `Diff` say what changed — between two stored
  models or two commits — `Status`, `BuildSite`, `Read`, `Find` and `OpenHistory` do what their
  commands do; `Fingerprint` says whether two models are one version, as history decides it, for a
  program that keeps models itself. The command is the API's first client and prints exactly what it
  printed before.
  `examples/library` shows a program of its own using it.

No version tagged yet. The first, `v0.1.0`, comes with the licence. Until then the state of the
engine is described in `PROGRESS.md`, "Where this stands".
