# archdoc

Reads an existing codebase and produces the architecture documentation it should have had —
accurate, traceable to the code, and regenerable as the system changes. Go engine, TypeScript
web app, one static binary.

**Phase:** R1.a, pre-construction. No implementation code yet.
**Target:** feature-complete 2 Oct 2026, delivered 9 Oct 2026.

---

## Read this before anything else

**Document precedence.** Where `docs/stack-decision.md` and `docs/product-definition.md`
disagree, **the definition wins**. The definition is what the product *is*; the stack
decision is what it is built with. `docs/survey-test-subjects.md` is evidence, never
instruction.

**Token discipline.** `docs/` totals ~3,900 lines. Never load a document whole. Load the named
section for the task at hand and nothing else.

| Load this | When |
|---|---|
| `docs/product-definition.md` §4 | Implementing a capability — find its ID first |
| `docs/product-definition.md` §10, §11 | Touching core types or the pipeline |
| `docs/product-definition.md` §8 | Anything that writes files to disk |
| `docs/product-definition.md` §12 | Working on an acceptance criterion |
| `docs/stack-decision.md` §2.3 | Using one of the five libraries |
| `docs/stack-decision.md` §3 | Questions about layout, packages, or the build |
| `docs/survey-test-subjects.md` §1–2 | Writing or fixing a parser — real dialects, real failures |
| `docs/delivery-schedule.md` | Planning or scoping. **Not** while building |
| `PROGRESS.md` | Starting work — what is done, what gate is next |
| `.claude/NOTES.md` | Starting or ending a session — the inbox of unplanned items |
| `docs/origin/` | Asking *why* the project is shaped this way, or "did we consider X?" — the ideation record, including discarded alternatives |
| `docs/decisions.md` | Before revisiting anything that looks already-settled |

---

## Hard rules

Invariants that are **not** derivable from the code. Violating any of these breaks a stated
acceptance criterion.

1. **Sort map keys before iterating** whenever the result reaches output. Go randomises map
   iteration; unsorted output silently breaks AC-7 (five runs, byte-identical). This is the
   single most likely defect in generated Go here.
2. **Never write to human-owned files. Never read them either.** Only `*.generated.md` is
   written, and it is overwritten wholesale. Human sections are *linked*, not parsed —
   `OUT-02` / `OUT-03`. A documentation generator that eats someone's writing gets
   uninstalled once.
3. **Only `internal/semantic` may make outbound network calls.** This is what makes AC-8
   provable by an import test rather than a manual audit. No other package imports an HTTP
   client for egress.
4. **The LLM never produces facts.** It supplies labels, descriptions and groupings on top of
   facts extraction already established, and returns a structured operation diff — never a
   whole model (`SEM-10`).
5. **The validator never writes a partial model.** On retry-budget exhaustion, fail loudly
   with the offending operations (`VAL-08`).
6. **`testdata/` fixtures stand in for *other people's* repositories.** archdoc reads repos
   it does not own and writes `.archdoc/` and `docs/architecture/` into them. Nothing in this
   repo is the tool's own output.
7. **Compose parsing is in-process, never shelled out.** `docker compose config` is a
   development-time test oracle only — archdoc must run with no container runtime present.

---

## Package map

Packages follow architectural seams, not the capability catalog's chapters. See
`docs/stack-decision.md` §3.

| Package | Seam |
|---|---|
| `internal/archdoc` | Core types — `Fact`, `Provenance`, `FactSet`, `Node`, `Edge`, `Model`, `Version` |
| `internal/extract` | Discovery + the parser registry |
| `internal/store` | Storage interface + SQLite driver |
| `internal/semantic` | The network boundary |
| `internal/render` | Layout, SVG, Mermaid |
| `internal/serve` | Local HTTP + embedded web assets |
| `cmd/archdoc` | CLI — thin, no logic |

`validate`, `rules` and `output` live as files inside the packages that use them until
coupling earns them a package of their own.

**Build:** contributors need Node; users do not. `web/dist` is built before the binary and
embedded. A `dev` build tag proxies to the frontend dev server instead.

**Git.** Work happens on `develop`; `main` advances only by merge. Remote is
`git@github-edu:cruzambrociogl/archdoc.git` — the alias matters, it selects the right SSH key.
Commit identity is **repo-local** (`cruz.ambrocio@galileo.edu`), deliberately different from
the global config. Module path: `github.com/cruzambrociogl/archdoc`.

---

## Vocabulary

Plain phrases mapped to exact destinations. **Always announce the file before writing to it,
in the form `-> path — [what]`. Never save silently.**

| Say | Means |
|---|---|
| "add a note" / "take a note" / "remember this" / **"add a TODO"** / "we'll do this later" / "don't forget" | `.claude/NOTES.md` — the inbox. One line, no filing effort |
| "log this decision" | `docs/decisions.md`, dated entry, 3–6 lines |
| "mark this done" / "X is done" | `PROGRESS.md` — capability ID or AC gate |
| "update the context" | this file — and remove something stale in the same edit |
| "update the docs" | the relevant `docs/` file |
| "that's a gotcha" | Hard rules above, if it is an invariant; otherwise `docs/` |
| "checkpoint" | `/checkpoint` — sweep the session, route everything, prune |

**There is no `TODO.md` in this repository, and none should be created.** "TODO" means
`.claude/NOTES.md`. Planned work already has a tracker — `PROGRESS.md`, keyed on capability IDs and
acceptance gates — and a third list would immediately drift from both.

The only distinction worth holding: **`.claude/NOTES.md` is for things that were not planned.** If it
already has a capability ID or is an acceptance gate, it belongs in `PROGRESS.md`.

---

## Maintenance

`/checkpoint` is the only mechanism that keeps this file honest. It both **routes** new
material and **prunes** what has gone stale — this file grows only when something is also
removed. An auto-loaded file that only accretes is the failure mode the token discipline
exists to prevent.

When a decision is reversed, edit the original entry in `docs/decisions.md` to point at
the reversal. Do not silently delete it — the reasoning is the record.
