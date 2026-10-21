# Changelog

What changed in archdoc, per tagged version, for those who use it. Newest first. Decisions and their
reasons are in `docs/decisions.md`; what is done against the plan is in `PROGRESS.md`; this file says
what is different for a user.

## Unreleased

- **archdoc is a Go library as well as a command.** `import "github.com/cruzambrociogl/archdoc"`:
  `Generate` documents a repository, `Compare` and `Diff` say what changed — between two stored
  models or two commits — `Status`, `BuildSite`, `Read`, `Find` and `OpenHistory` do what their
  commands do. The command is the API's first client and prints exactly what it printed before.
  `examples/library` shows a program of its own using it.

No version tagged yet. The first, `v0.1.0`, comes with the licence. Until then the state of the
engine is described in `PROGRESS.md`, "Where this stands".
