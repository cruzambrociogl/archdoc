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

**2026-09-06 — C4: transparent intermediaries stay in the model, leave the container view**
Compose describes *deployment*; a C4 container diagram must not show deployment concepts, and
Brown is explicit that gateways are *"typically wrong"* on one — most exist only in production.
The live cases are Supabase's `api-gw` (Envoy/Kong) and `supavisor`, a connection pooler.

Both are separately deployable applications, so the literal definition would admit them. They
are excluded anyway because they are **transparent**: applications talk *through* them without
depending on them semantically. Drawing them turns `auth → db` into `auth → supavisor → db`,
which hides the real coupling behind an intermediary — Brown's own complaint about modelling a
message bus as one box: *"everything looks like a hub and spoke architecture... we're missing
the couplings between the individual services."*

**Rule: a Compose service becomes a C4 container when it is an application or a data store.
Proxies, gateways, poolers and sidecars remain in the model with `kind` recorded, and the
container view omits them — annotating the edge they mediate, `auth → db (via supavisor)`.**
That annotation is Brown's suggested treatment for a queue between two services. Nothing is
lost: the intermediary is still in `model.json`, and `rules.yaml` can force it back in.

Consequence: Supabase renders 9–10 containers rather than 11, and its real service-to-service
edges become visible for the first time. `Docker` never appears as a technology on a container
diagram — the technology is what runs *inside*, which is what the catalog is for.

**2026-09-06 — Actors are derived from published ports, not invented**
C4's context level needs people and external callers. Nothing in a Compose file says a person
exists, and §3 forbids inferred nodes in R1 — so an actor guessed by the model would be
inadmissible.

**A published port is declared evidence that something outside reaches in.** `immich-server`
exposes `2283:2283` at `docker-compose.yml:26`; Supabase's `api-gw` exposes `8000`. That is a
fact at a line, not a guess. What it does not say is *who* — a person, a mobile client, another
service.

So the actor is built in three steps, each staying inside the evidence rule: extraction emits a
generic **External client** node carrying the port's provenance; the semantic layer **names** it
more usefully (*"Photographer"*, *"Mobile app"*) — labelling a node that already exists rather
than inventing one; `rules.yaml` corrects it when the guess is wrong.

Mastodon is the case that proves the rule is doing real work: `web` binds `127.0.0.1:3000:3000`,
localhost only. Honestly read, that declares *"expects a reverse proxy in front"* rather than
*"a person talks to this directly"* — a distinction only available by reading what the file
actually says.

**2026-08-27 — Discovery: sniff for recall, reject fragments for precision**
The walking skeleton forced the discovery rule to be settled. A filename glob is not enough —
Immich keeps two real Compose files at `.devcontainer/server/container-compose-overrides.yml`,
matching no conventional pattern, and the glob written for the survey missed both. But
content-sniffing alone admits `docker/hwaccel.ml.yml`, whose services are named `cpu`, `armnn`
and `rknn` and are hardware snippets rather than anything deployable.

**Rule: sniff every YAML for a top-level `services:` key, then reject any file where no service
declares an image or a build.** Recall from the sniff, precision from the fragment test. On
Immich that yields 10 candidates, 7 deployable, 3 fragments.

Selection between deployables prefers the declared answer over convention: `COMPOSE_FILE` in a
neighbouring `.env` or `.env.example` is a native Compose variable holding a colon-separated
list, base file first — Supabase's own `run.sh` maintains it, so reading it is standards-based
rather than a guess. Only when absent does convention decide: unsuffixed name beats suffixed,
shallower path beats deeper, and `.devcontainer`/`e2e`/`test` paths are penalised.

`--explain` prints every candidate and the reason for its verdict. A tool whose premise is
traceability cannot answer "which file did you use?" with "trust me".

**2026-08-26 — O-7 closed: test subject #3 is Mastodon, not archdoc itself**
Chosen against a criterion rather than by preference. Measured what the existing two subjects
leave untested: **neither Immich nor Supabase declares a single top-level `networks:`**, and
neither references a genuine external managed dependency — Supabase's `SMTP_HOST=supabase-mail`
points at an internal service. So `MDL-09` (network boundaries), `MDL-11` (declared trust
boundaries), `MDL-15`/`MDL-16` (evidence kinds, external-system nodes) and `EXT-09` had
effectively **zero coverage**.

Mastodon fills exactly that gap. Its compose declares two networks with
`internal_network: {internal: true}`; `db` and `redis` sit only on the internal one while
`web`, `streaming` and `sidekiq` span both — a real trust boundary, declared. Its
`.env.production.sample` carries `S3_BUCKET=files.example.com`, `SMTP_SERVER` and `ES_HOST`,
which is **§3's worked example for referenced evidence, occurring naturally**. Five services
places it between Immich's four and Supabase's eleven.

Pinned at `47ac677a9b9392833d7cecab8fccbce34c738b83`. Candidates rejected on the same measure:
Outline, Paperless-ngx and Sentry self-hosted all declare zero networks.

*Replaces* the original intent of documenting archdoc itself, which §3's own vantage-point
argument predicted would be weak — a CLI tool with no declared services yields roughly one box.

**2026-08-26 — O-8 closed: provenance does NOT survive the Compose merge. Extraction is two passes.**
Tested against the two hardest files the survey found, at the pinned revisions:
`supabase/docker/docker-compose.kong.yml` (`!override` across a merge) and
`immich/.devcontainer/server/container-compose-overrides.yml` (`!reset`, plus two expansions
concatenated in one value).

*What works.* `compose-go` v2.14.0 gets the semantics right. `!override` replaced api-gw's
ports rather than appending them (8000, 8443 — exactly two), and swapped the image to Kong.
`!reset []` emptied `profiles`. The concatenated expansion
`${UPLOAD_LOCATION:-default}${UPLOAD_LOCATION:+/photos}` resolved correctly both ways —
`upload-devcontainer-volume` when unset, `/mnt/photos/photos` when set. The library earns its
place; this was the argument the stack decision rested on and it holds.

*What does not.* `types.ServiceConfig` has **100 fields and none carries a source position** —
no Line, Column, or File. `types.Project` exposes `ComposeFiles`, but that is which files
contributed to the project, not where any single fact came from. Nowhere near PRV-01.

*The fallback is proven, not assumed.* A raw `gopkg.in/yaml.v3` parse recovers positions and
they are addressable by key path — `services.api-gw.image` resolves to line 72 in the base
file and line 14 in the overlay. **The last file in which a key appears is the one that won
the merge**, which is exactly the provenance PRV-01 needs.

**Consequence:** extraction runs `compose-go` for semantics and `yaml.v3` for positions,
reconciled by key path. More work, no more risk, and the same would be required in any
language. It also settles the core type: a **`Fact` carries one source position** — the file
that won — not two.

**2026-08-25 — Context system: three layers, adapted rather than adopted**
A five-layer scheme from another workspace was considered. Its workspace spine, project table
and cross-project TODO tagging solve multi-repo problems this project does not have, and its
machine-local memory layer contradicts its own "repo over memory" principle. Kept: token
discipline, load-on-demand, the vocabulary table, announce-before-writing, and `/checkpoint`.
Added what it lacked — a **pruning rule**, since routing without deletion makes auto-loaded
context grow without bound, which is the failure the token discipline exists to prevent.

**2026-08-25 — No `TODO.md`. `PROGRESS.md` for planned work, `NOTES.md` for the rest**
The capability catalog is already the work breakdown, so a free-form todo list would drift
from it immediately. But planned work is not everything: unplanned ideas, reminders and
questions had no home, and a note that costs effort to file does not get written down.
`.claude/NOTES.md` is the low-friction inbox; `/checkpoint` drains it. "add a TODO" routes
there, so the habit does not need retraining.

**2026-08-25 — Meta files placed by audience, not by type**
`PROGRESS.md` at the root because it answers the question a supervisor asks and hidden folders
are not browsed. `.claude/NOTES.md` hidden because it is a private working inbox.
`docs/decisions.md` moved *out* of `.claude/` — an architecture decision record is a project
artifact, not tooling config. Rule: root is what a visitor should see, `.claude/` is what only
the tooling needs.

**2026-08-25 — Branch discipline by convention, not by GitHub ruleset**
`main` is left unprotected on GitHub. Rulesets are gated behind paid plans for private
repositories, and the alternative — making the repository public purely to unlock a setting —
is a poor reason to publish before there is anything to show. The discipline is written into
`CLAUDE.md` instead, where a session will actually read it: work on `develop`, `main` advances
only by pull request, never force-push a published branch. **Nothing enforces this**, which is
stated explicitly so no session assumes the platform is guarding it. Revisit if the repository
goes public. The PR trail is kept deliberately — it doubles as a dated record of what landed.

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
