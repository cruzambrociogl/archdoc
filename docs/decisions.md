# Decision log

Dated, terse. Newest last. A reversed decision is **edited to point at its reversal**, never
deleted — the reasoning is the record.

Full rationale for the larger decisions lives in `docs/stack-decision.md`; this is the index
and the short form.

---

**2026-08-23 — Identity model: the registry (O-1)**
Canonical identities with alias history, resolved from stable IDs at extraction time, rules
able to pin an alias by hand. Renames are alias additions, not delete-plus-add.
The one open question that could not wait for the stack.

**2026-08-24 — Empirical survey before the stack conversation**
Read Immich and Supabase rather than choosing a stack from assumptions. Justified itself
immediately: Immich declares **none** of its three service edges in configuration, and a
naive discovery glob missed two real compose files. Both would have been found late.
→ `docs/survey-test-subjects.md`

**2026-08-24 — Orchestrator manifests out of R1 scope**
Kubernetes, Helm and Kustomize deferred, R2 the likely home. Neither test subject contains a
single k8s or application-level Terraform file, so the capability would be built with nothing
to validate it against — contradicting the evidence-first posture. `EXT-05` is superseded.
Consequence: extraction sources are Compose, `.env`, gateway/proxy config, interface
contracts.

**2026-08-24 — O-4 contract-file matching: the answer is rules**
Four attachment strategies tested against six specs across both subjects. Directory proximity
fails on both. Self-identification works once in six. The build-context chain breaks because
Immich's canonical compose file has no `build:` key. `rules.yaml` is therefore the **primary**
mechanism — which makes the rules layer load-bearing rather than merely refining, and
strengthens AC-5.

**2026-08-24 — Stack: Go for the engine, TypeScript for the web app (O-2)**
Decided on an asymmetry in failure modes: a mishandled Compose `!override` produces a
*silently wrong* model — the draw.io failure this project exists to prevent — while weak
layout produces an ugly diagram, which is visible and which §1 already subordinates to the
model. `compose-go` is the Compose specification's reference implementation, so the
highest-risk input becomes a solved problem rather than one re-derived. Rejected: TypeScript
(re-implementation on the fact path), Rust (same, without the one-language benefit), Python
(weakest distribution).
→ `docs/stack-decision.md` §5

**2026-08-24 — Storage: SQLite via `modernc.org/sqlite` (O-3)**
Pure Go, cgo-free, chosen over `mattn/go-sqlite3` to preserve cross-compilation and the
single-static-binary property that justified Go. Slower; irrelevant at dozens of nodes.
Stays behind §7's driver interface.

**2026-08-24 — Layout: `goccy/go-graphviz`, and the frontend never lays out**
Graphviz as WebAssembly — no cgo, no system install. Supersedes `VIE-03`'s `dagre`/`elkjs`
hint. Layout is computed once in the engine and persisted (`VIE-04`), so the web canvas
renders stored coordinates. That makes `VIE-10`'s fidelity guarantee structural rather than a
matter of keeping two layout engines in agreement.

**2026-08-24 — One repository, packages by seam not by catalog chapter**
Mapping §4's capability groups onto directories was rejected: it breaks on `PRV`, which is a
property every fact carries rather than a stage, and on `FactSet`, which four groups would
share. Six packages, each justified by an architectural seam. Making the network boundary
structural means AC-8 is provable by an import test rather than a hand audit.
→ `docs/stack-decision.md` §3

**2026-08-24 — Types are settled before directories**
Directory names refactor in minutes; `Fact`, `Provenance`, `FactSet`, `Node`, `Edge`, `Model`
and `Version` propagate into every signature and into `model.json`, which is a *committed
deliverable*. O-8's answer gates this — it decides whether a `Fact` carries one source
position or two.

**2026-08-25 — Repository name: `archdoc`, lowercase**
Renamed from `ArchDoc`. Everywhere else the name is already lowercase — the binary, the CLI
verbs, `internal/archdoc`, and all running text in `docs/`. Go package names must be lowercase
regardless, so capitals in the repository name made it the outlier rather than the standard.
Concretely, an uppercase module path is escaped as `!arch!doc` in the module cache for the
life of the project. The display name stays **ArchDoc** in the README title and in prose.
Module path: `github.com/cruzambrociogl/archdoc`.

**2026-08-24 — Delivery: R1.a complete, committed for 9 Oct**
Scope tiered by what each piece is *limited* by, not by volume. Implementation-bound work
(parsers, model, renderers, web app) compresses heavily under AI assistance;
verification-bound work (acceptance measurement, the hand-drawn reference) does not;
research-bound work (R1.b semantic clustering) does not compress at all and is **not
committed**. Web app is in scope. R1.c and R1.b static analysis are stretch.
→ `docs/delivery-schedule.md` §2
