# archdoc — Project Definition

> The settled definition of the project: what it is, what it does, what it produces, and
> what remains open. Self-contained — start at §0.
>
> Working codename: **archdoc**. Placeholder.

---

## 0. Context for a new reader

**The problem.** AI now writes most of the code while a person directs it. That works, and
it is fast — but the understanding that normally forms while typing never forms. Reviewing
confirms the result works; it does not build a mental model. Weeks later the person who
directed every decision is as lost as a stranger would be, with nobody to ask, because
nobody ever knew. Documentation debt assumes someone knew and failed to write it down. This
is knowledge that never existed.

**The evidence this is built on.** The project began from a concrete test: a detailed spec
of a known system — roughly twelve services, seven groupings, twenty protocol-carrying
edges — hand-written and handed to draw.io's AI assistant. It failed in two distinct ways.
First, every fact had to come from human memory, and even a deliberate spec left one
component undefined and its incoming edge unwritten. Second, the tool also dropped and
misread things that *were* in the spec, because it is a drawing tool with no model of what
architecture is. Those two failures are what the validation rules (§4.5) and the whole
provenance design exist to prevent.

**Why extraction reads configuration, not code.** In a service-based system the
architecture is *declared* — in compose files, proxy config, manifests, and interface
contracts — not inferable from source. A pure code analyser would have produced a worse
diagram than the handwritten prompt did.

**Test subjects** are self-hosted Immich (simple), self-hosted Supabase (complex, close to
the system that motivated this), and this project itself. All public or ours, and none need
to be *run* — only cloned. A read-only survey of the first two — what config actually
exists, in what dialects, and what breaks a naive parser — is recorded in
**`survey-test-subjects.md`**, cited below as *the survey*. Several decisions in §13 rest on
it.

**Read §13 early.** It separates what is settled from what is still open, so settled
decisions don't get relitigated and open ones aren't assumed closed.

---

## 1. Identity

**archdoc reads an existing codebase and produces the architecture documentation it
should have had — accurate, traceable to the code, and regenerable as the system
changes.**

It is not a design tool. It is not an enforcement tool. **It describes what is there.**

The loop: point it at a repo → extract the verifiable structure from configuration and
infrastructure → validate it into a model that cannot be internally inconsistent →
render C4 diagrams and arc42-structured documentation where every element links back to
the file that proves it exists → run it again after new commits and it reports what
changed architecturally.

### Two governing principles

1. **The diagram is never the source of truth — a validated model is.**
2. **The LLM is never responsible for facts.**

### What "proper documentation" means — five testable properties

| # | Property | Test |
|---|---|---|
| P1 | **Accurate** | Every element traces to a real file and line. The tool cannot invent a component. |
| P2 | **Structured** | C4 levels, arc42 sections. Not freeform. (Both explained in §14.) |
| P3 | **Consistent** | Views never contradict each other, because they are projections of one model. |
| P4 | **Regenerable** | Re-running yields updated truth, not a different guess. |
| P5 | **Legible** | A person reads it and understands the system. |

P1 is non-negotiable — it is the entire separation from every AI diagram tool.

---

## 2. Releases

### R1 — Documentation (the current target)

Reads an already-started project and generates its architecture documentation. Describes
what is there. Delivered in three phases, in this order:

**R1.a — Container level, config-driven.** Services, boundaries, routes, queues, data
stores, external systems. Extraction from configuration, infrastructure, and interface
contracts. The full deterministic spine plus the semantic layer, both surfaces, and the
architectural diff. This is what §3 and §4 specify. Deployment view included as a stretch,
since manifests are declarative and nearly free once the spine exists.

**R1.b — Depth.** Static code analysis of application source, component-level detail, and
semantic clustering — turning many files inside a service into a few meaningful boxes.
This is where the genuine research problem lives.

**R1.c — Interaction.** Three write surfaces, designed together because they share one
safety story: chat refinement (writes rules), the editable canvas with bidirectional sync
(writes the model), and the **answer surface** (writes human-owned document sections —
§4.12). Building them as one phase means the write-safety design — content hashing,
conflict refusal, never-during-generation — is done once rather than three times.

**The ordering holds even with time open.** R1.a's deterministic spine is what everything
else is built on, and it is the phase that proves the central claim — that facts come from
extraction and the model only interprets. Building depth or interaction before that spine
is solid inverts the dependency and forfeits the argument.

### R2 — Authority (preserved roadmap, not being built now)

**Both R2 products were deferred for one reason: they require controlling a system you
don't own** — an enforcement point in someone's CI, or a hook inside an agent's loop.
R1 depends on nothing but the repository. Recording R2 here so the direction isn't lost.

**R2a — The Guardrail.** The documentation gains authority over the code.

- A constraint language in `rules.yaml`: "the API layer must not call the database
  directly", "nothing outside the gateway may be publicly routable", "no service may
  depend on more than N others".
- A second model: **intended** architecture alongside **actual**. R1 has only *actual*.
  The product becomes the delta between them — conformance, not just drift.
- `archdoc check` exits non-zero on violation; CI integration; PR comment with the
  offending edge and its provenance.
- Drift classification: drift-over-time (actual vs. actual at t−1) versus
  drift-from-intent (actual vs. intended). Different signals, different responses.

**R2b — The Agent's Map.** The architecture becomes context and constraint for the AI
writing the code.

- Dense, token-efficient serialization of the model for agent consumption — the opposite
  optimization target from human prose.
- The closed loop: the tool documents what the agent built, and that documentation
  constrains what the agent builds next.
- Delivery options to evaluate later: a generated context file, an MCP server exposing
  the model as queryable tools, or an agent-callable API.
- The interesting question deferred with it: is the agent *reading* the architecture
  (context) or *bound by* it (enforcement)? Those are different products again.

### R1.x — deliberately left open in R1, resolved in R2

Not oversights. Each is a place where R1 stops at what a declaration proves, and R2 picks
it up once it has a reason to go further.

| Left open in R1 | R1 behaviour | Why that's the right stopping point | Resolved in R2 by |
|---|---|---|---|
| **Trust boundary derivation** (MDL-11) | Declared boundaries only — networks, namespaces, and what proxy/ingress config explicitly states. No inference of "publicly routable" or "VPN-reachable". | Trust boundaries only acquire meaning when something enforces them. Inferring them in a purely descriptive tool produces claims nothing can check. | R2a's constraint language, where "nothing outside the gateway may be publicly routable" makes the boundary testable |
| **Intended architecture** | Only *actual* is modelled. Drift is actual-vs-actual over time. | R1 describes what is there. An intended model with nothing to compare it against is a design tool, which R1 explicitly is not. | R2a's second model — conformance becomes actual-vs-intended |
| **Docs delivery beyond the machine** | Served locally from `archdoc serve`, Swagger-UI style. Committed text in the repo. | Local is the whole point of a local tool, and committed markdown already travels through git. | R2's publishing surface — shareable and remotely reachable |
| **CI execution** | None. The tool is run by a person. | R1 has nothing to fail a build over; it only describes. | R2a's `archdoc check`, delivered as a container (§7) |
| **Agent-consumable output** | `model.json` — canonical, readable, tool-parseable, but optimized for humans and diffs. | It is already machine-readable. Optimizing it for token density before an agent consumes it is speculative. | R2b's dense serialization, MCP server, or agent API |
| **Cross-repo visibility** | A star centred on one repository — declared services plus opaque referenced externals (§3). | Honest and complete from one vantage point, which is how C4 and arc42 both work. | Multi-repo composition, merging referenced externals against declared services |

**Deferred beyond R1 and R2 entirely:**

| Deferred | Why |
|---|---|
| Narrative prose generation | Hardest LLM output to keep honest. An empty arc42 section beats a plausible fabricated one, and that property is worth more than the prose. |
| AI session transcript ingestion | Highest novelty, highest scope risk — transcripts are ephemeral, enormous, and formatted differently per tool. If it ever lands, it enriches nodes that already exist from extraction; it is never a source of structure. Belongs alongside R2b if anywhere. |
| Multi-repo composition | Run archdoc on N repos, then merge models by matching *referenced* externals in one against *declared* services in another — `ext:postgres@env` in the API repo unifies with `svc:postgres@compose` in the infra repo, turning the star into a graph. A clean extension rather than a rewrite, precisely because declared and referenced are separated in R1. |
| RAG over pattern catalogs | Marginal gain on naming; revisit only if semantic quality is the bottleneck. |
| Voice | No technical depth added. |

---

## 3. R1 scope

| | |
|---|---|
| **Input** | One existing local repository |
| **Extraction** | Configuration and infrastructure only. Language-agnostic. **Sources: Compose, `.env`, gateway/proxy config, and interface contracts.** Orchestrator manifests (Kubernetes, Helm, Kustomize) are out of R1 scope — see §13. |
| **Depth** | **Container level** — services, boundaries, routes, queues, data stores |
| **Model** | Validated, versioned, embedded file-based storage (§7). Cannot persist in an invalid state. |
| **Output** | C4 context + container diagrams as SVG and Mermaid; twelve-section arc42 skeleton with fact-backed sections filled; `model.json`; optional site config. Full contract in §8. |
| **Surfaces** | CLI + local web app |
| **Refinement** | `rules.yaml`, hand-edited |
| **Memory** | Container-level architectural diff between any two commits |

### Vantage point — what the tool can and cannot see

archdoc documents a system **from one repository's point of view**. That produces a star,
not a universe: everything this repo *declares*, plus opaque boxes for everything it
*references*.

It cannot see what those external boxes connect to, and it cannot see services in other
repositories that talk to the same shared infrastructure. This is the normal shape of
architecture documentation — C4 and arc42 both document *a system and its neighbours*,
not the whole world — but it is a real constraint, stated here rather than discovered in
it late.

The determining axis is **not monorepo versus polyrepo**. A monorepo with no compose file
tells the tool almost nothing; a single-service repo with a full `docker-compose.yml`
tells it a great deal. What matters is what the repository declares.

### How external systems are represented

Every node carries an **evidence kind**:

| Evidence | Source | What is known | Example |
|---|---|---|---|
| **Declared** | The repo defines it | Image, ports, networks, internals | a service in `docker-compose.yml` |
| **Referenced** | The repo points at it | That it exists and that this system talks to it — nothing more | `DATABASE_URL=postgres://prod-db.internal:5432` |
| **Inferred** | Nothing declares or references it | — | **must not exist in R1** |

**A reference is provenance.** `S3_BUCKET=training-data` at `.env.example:14` is a real
fact at a real line, proving this system talks to object storage. Referenced nodes are not
a degraded case that weakens P1 — they satisfy it through a different evidence kind, and
they carry file-and-line provenance exactly like declared ones.

This maps directly onto C4's **external system** concept: a box outside the system
boundary, conventionally rendered greyed out. The model needs no new machinery — a node
kind and a boundary of kind `external`. Protocol usually comes free from the URL scheme
(`postgres://`, `redis://`, `amqp://`, `s3://`, `https://`), which is catalog work rather
than model work.

The honest caveat: `.env.example` can be incomplete or aspirational. The evidence kind is
recorded on the node and surfaced in the UI, so the reader always knows which boxes are
proven infrastructure and which are declared intentions.

### Why container level is the right first depth

**Legibility (P5) is normally the hard research problem** — turning 400 files into 8
meaningful boxes, reproducibly. At container level with declaration-based extraction it is
nearly free: **services already are the boxes.** The clustering problem lives *inside* a
service, which is exactly what R1.b takes on.

So R1.a is not the easy version of the product — it is the version whose ground truth is
strongest, where every element is provable from a declaration and the model is never asked
to invent structure. R1.b then attacks clustering standing on a spine that already works,
rather than making the whole product depend on solving it first.

This is a deliberate sequencing choice, not an accidental one. Worth knowing which problem
you chose to defer and why.

### Why R1 still isn't a wrapper

Without code analysis, without chat, without enforcement — is this promptable?

No, for two reasons that survive every descope:

- **Determinism.** Extraction produces byte-identical output across runs. No model does.
- **Persistence.** The architectural diff compares two stored models. A prompt holds no
  state.

The LLM does one job in R1: naming and grouping, on top of facts it did not produce and
cannot override.

---

## 4. Capability catalog

Every capability **R1.a** needs, classified by what it depends on. Capabilities for the
later phases are specified when those phases are designed; §4.12 is the first of them and
is excluded from the R1.a count in §6.

**Classification codes:**

| Code | Meaning |
|---|---|
| **DET** | Deterministic. Pure code. Byte-identical across runs. No network. |
| **CAT** | Catalog lookup. Deterministic, against a curated knowledge base (§5). |
| **HYB** | Hybrid. Catalog first, LLM only for what the catalog misses. |
| **LLM** | Requires a model. |

---

### 4.1 Discovery — `DSC`

| ID | Capability | Class |
|---|---|---|
| DSC-01 | Detect which config files exist and which parsers apply | DET |
| DSC-02 | Determine project shape (compose / k8s / bare / hybrid) | DET |
| DSC-03 | Resolve include/exclude globs and multi-file compose overrides | DET |
| DSC-04 | Report coverage — what was found, what was skipped, what was unparseable | DET |

> DSC-04 matters more than it looks. Honest reporting of what the tool *couldn't* read is
> what keeps P1 credible.

### 4.2 Extraction — `EXT`

| ID | Capability | Class |
|---|---|---|
| EXT-01 | Parse `docker-compose.yml` — services, images, ports, volumes, networks, `depends_on`, env | DET |
| EXT-02 | Parse `Dockerfile` — base image, exposed ports, entrypoint, build args | DET |
| EXT-03 | Parse Traefik labels and dynamic config — hosts, routers, services, middlewares | DET |
| EXT-04 | Parse nginx / Caddy config — server blocks, upstreams, proxy passes | DET |
| EXT-05 | Parse Kubernetes manifests — Deployments, Services, Ingresses, ConfigMaps, Secrets (keys only) | DET |
| EXT-06 | Parse CI workflows — build, test, deploy jobs and their artifacts | DET |
| EXT-07 | Parse `.env` / `.env.example` — configuration surface, key names only | DET |
| EXT-08 | Extract service-to-service references from env values (`REDIS_URL=redis://cache:6379`) | DET |
| EXT-09 | Identify external managed dependencies from known env/config patterns (S3, RDS, SES, Stripe…) | CAT |
| EXT-10 | Identify unknown external dependencies from unrecognized URLs or credential-shaped keys | HYB |
| EXT-11 | Parse OpenAPI / Swagger specs — endpoints, operations, request and response shapes | DET |
| EXT-12 | Parse gRPC `.proto` files — services, RPC methods, message types | DET |
| EXT-13 | Parse GraphQL SDL — schema entry points, types, federation directives | DET |
| EXT-14 | Attach file path and line range to every extracted fact | DET |
| EXT-15 | Emit the **FactSet** — the contract artifact between extraction and everything downstream | DET |

> **Interface contract files belong in extraction, not code analysis.** `.proto`, GraphQL
> SDL, and OpenAPI specs are declarative, deterministically parseable, and describe
> *interfaces* — exactly the container-level concern. They recover part of what tier 2
> would have provided at none of tier 2's cost: a `.proto` file states what a service
> exposes without parsing a single line of Go.

### 4.3 Model construction — `MDL`

| ID | Capability | Class |
|---|---|---|
| MDL-01 | Build node set from discovered services | DET |
| MDL-02 | Build edges from `depends_on` and shared networks | DET |
| MDL-03 | Build edges from proxy routes (domain → service) | DET |
| MDL-04 | Build edges from env-value service references (EXT-08) | DET |
| MDL-05 | Assign protocol to edges by known port and image (5432→PostgreSQL, 6379→Redis, 443→HTTPS) | CAT |
| MDL-06 | Assign protocol where the port is non-standard or the image is unknown | HYB |
| MDL-07 | Classify node type (proxy / frontend / api / worker / datastore / queue / external) from image name | CAT |
| MDL-08 | Classify node type where the image is custom-built | HYB |
| MDL-09 | Derive **network boundaries** from compose networks / k8s namespaces | DET |
| MDL-10 | Derive **deployment boundaries** from compose files, profiles, k8s clusters | DET |
| MDL-11 | Record **trust boundaries** as declared by proxy/ingress config. R1 does not infer reachability — see R1.x | DET |
| MDL-12 | Reconcile the same service declared across multiple files | DET |
| MDL-13 | Generate **stable node IDs** that survive across runs and renames | DET |
| MDL-14 | Detect and mark external actors (browser, mobile client, third-party callers) | HYB |
| MDL-15 | Classify each node's **evidence kind** — declared or referenced | DET |
| MDL-16 | Create external-system nodes for referenced dependencies, placed in an `external` boundary | DET |
| MDL-17 | Derive edge protocol from URL scheme (`postgres://`, `amqp://`, `s3://`, `https://`) | CAT |

> **MDL-13 is load-bearing.** Without stable IDs the architectural diff reports every run
> as a total rewrite. Design it before anything downstream depends on it.

**Identifier scheme.** Two identifiers doing two different jobs:

- **Composite stable ID** — human-readable, derived from evidence:
  `svc:postgres@compose`, `ext:aws-s3@env`, `route:hub.example.com@traefik`.
  Kind, normalized name, source.
- **Opaque row ID** — for database referential plumbing only.

The composite form matters because `model.json` is committed to the repo: a readable
stable ID makes the *git diff of the model itself* readable (`+ svc:redis@compose` rather
than UUID churn).

**Identity registry.** Stable IDs are derived from evidence, so they change when the
evidence changes — a service renamed in compose, or moved from compose to a Kubernetes
manifest, yields a different composite ID for the same real thing. Left alone, the diff
reports that as a delete plus an add, and the architectural diff becomes noise.

The fix is one layer of indirection that survives versions: an **identity registry**
mapping observed stable IDs onto an internal canonical identity.

- Every real component has one canonical identity, created the first time it is seen and
  never reused.
- An identity accumulates **aliases** — each stable ID it has been observed under, with
  the version range in which that ID was current.
- Extraction emits stable IDs; the registry resolves them to canonical identities before
  the model is written. Nothing downstream sees raw stable IDs.
- A new stable ID matching an existing identity by heuristic — same kind, overlapping
  image/port/route evidence, adjacent version — is recorded as an alias, and reported as a
  **rename**. One matching nothing is a genuine **addition**.
- A rule can pin an alias by hand when the heuristic is wrong, and that pin persists like
  any other rule.

This also covers the harder case the composite scheme alone does not: a service moving
from compose to a k8s manifest keeps its identity even though its `@source` segment
changed.

> **Decided.** The identity registry is the model — MDL-13, MEM-05, and MEM-06 are built on
> it. It exists specifically so identity is never retrofitted after the diff.

### 4.4 Semantic layer — `SEM`

**This is the only area where the LLM does real work.**

| ID | Capability | Class |
|---|---|---|
| SEM-01 | Human-readable labels for known infrastructure (`pg` → "PostgreSQL Database") | CAT |
| SEM-02 | Human-readable labels for custom services | LLM |
| SEM-03 | One-line responsibility statement per component | LLM |
| SEM-04 | Responsibility for known infrastructure (Redis → "in-memory cache and job queue") | CAT |
| SEM-05 | Logical grouping beyond what config declares ("these three are the data layer") | LLM |
| SEM-06 | Naming of derived boundaries | LLM |
| SEM-07 | C4 **context** level — system boundary and external actors | LLM |
| SEM-08 | Edge labels describing what a connection is *for* | LLM |
| SEM-09 | Fill fact-backed arc42 sections in readable prose | LLM |
| SEM-10 | Emit all of the above as a **structured operation diff**, never as a whole model | DET (the contract) / LLM (the content) |

**Constraint on the entire SEM layer:** it may only *label, group, and describe*. It may
not introduce a node or edge lacking extraction provenance. Enforced by the validator
(VAL-05), not by prompt instructions.

### 4.5 Validation — `VAL`

All deterministic. This is the layer that makes P1 and P3 structural rather than hoped for.

| ID | Capability | Class |
|---|---|---|
| VAL-01 | Schema conformance | DET |
| VAL-02 | Referential integrity — no edge to an undefined node | DET |
| VAL-03 | Every edge declares protocol, direction, sync/async | DET |
| VAL-04 | Every node belongs to exactly one boundary | DET |
| VAL-05 | Every node and edge carries at least one extraction-tier provenance entry | DET |
| VAL-06 | Every node declares a responsibility and a technology | DET |
| VAL-07 | Feed validation errors back to the model and retry, to a fixed budget (default 3) | DET |
| VAL-08 | On budget exhaustion, fail loudly with the offending operations — never write a partial model | DET |

> Every one of VAL-02 through VAL-06 corresponds to a specific failure observed in the
> draw.io experiment. They are derived from evidence, not preference.

### 4.6 Views and rendering — `VIE`

| ID | Capability | Class |
|---|---|---|
| VIE-01 | Project the model into a **C4 context** view | DET |
| VIE-02 | Project the model into a **C4 container** view | DET |
| VIE-03 | Compute layout (`dagre` / `elkjs`) | DET |
| VIE-04 | Persist node positions per version — **stable layout**, no reshuffling on add | DET |
| VIE-05 | Render Mermaid source | DET |
| VIE-06 | Render draw.io XML | DET |
| VIE-07 | Render PlantUML | DET |
| VIE-08 | Render the interactive canvas in the web app | DET |
| VIE-09 | Render boundaries as visual containers | DET |
| VIE-10 | Render SVG from the persisted layout — the committed picture matches the app exactly | DET |

**All rendering is deterministic.** Views are projections of one model — that is what
makes P3 structural.

### 4.7 Provenance — `PRV`

| ID | Capability | Class |
|---|---|---|
| PRV-01 | Attach source file and line range to every node and edge | DET |
| PRV-02 | Label each provenance entry with its origin (extraction / catalog / model) | DET |
| PRV-03 | Show all provenance for a selected element in the inspector | DET |
| PRV-04 | Click a provenance entry → open that file at that line | DET |
| PRV-05 | **Visually distinguish verified fact from model interpretation** in the UI | DET |
| PRV-06 | Compute and report the traceability percentage per run | DET |

> PRV-05 is what makes the output safe to trust *selectively*. The user always knows which
> parts they can rely on and which are the model's reading.

### 4.8 Memory and diff — `MEM`

| ID | Capability | Class |
|---|---|---|
| MEM-01 | Create an immutable version per run, tagged with the git commit SHA | DET |
| MEM-02 | Store every change as an appended operation | DET |
| MEM-03 | Retrieve, render, and export any historical version | DET |
| MEM-04 | Structurally compare two versions | DET |
| MEM-05 | Classify changes — added / removed / renamed / re-bounded / protocol-changed | DET |
| MEM-06 | Distinguish a rename from a delete-plus-add (depends on MDL-13) | DET |
| MEM-07 | Present the diff as structured output | DET |
| MEM-08 | Present the diff as readable prose | LLM (optional) |

> **MEM-08 is the only LLM capability in this entire area, and it is optional.** The
> structured diff is already readable. That means the headline feature works with no
> model at all.

### 4.9 Rules — `RUL`

| ID | Capability | Class |
|---|---|---|
| RUL-01 | Parse and validate `rules.yaml` | DET |
| RUL-02 | Match expressions — path glob, service name, image, node type | DET |
| RUL-03 | Actions — classify, relabel, set boundary, merge, hide, pin | DET |
| RUL-04 | Apply rules after extraction and before validation | DET |
| RUL-05 | Detect conflicting or unreachable rules | DET |
| RUL-06 | Record which rules were applied in every generated document | DET |

Rules are applied *before* validation, so a rule can never produce an invalid model.

### 4.10 Surfaces — `SUR`

| ID | Capability | Class |
|---|---|---|
| SUR-01 | `archdoc init` — scaffold `.archdoc/` | DET |
| SUR-02 | `archdoc scan` — extraction only. Fast, free, offline, no LLM | DET |
| SUR-03 | `archdoc generate` — full pipeline → version + docs | mixed |
| SUR-04 | `archdoc diff <ref> [<ref>]` | DET (+ optional LLM prose) |
| SUR-05 | `archdoc export --format <fmt>` | DET |
| SUR-06 | `archdoc history` | DET |
| SUR-07 | `archdoc serve` — local web app | DET |
| SUR-08 | Web: canvas with C4 level switch | DET |
| SUR-09 | Web: inspector with provenance | DET |
| SUR-10 | Web: version timeline and diff viewer | DET |
| SUR-11 | Web: rules viewer | DET |
| SUR-12 | Web: generated docs preview | DET |
| SUR-15 | Web: **completeness view** — human-owned sections with empty/filled/stale status, each linking out to the file in the editor | DET |
| SUR-13 | Cost and token reporting per run | DET |
| SUR-14 | **Egress reporting** — print exactly what left the machine | DET |

### 4.11 Output and deliverables — `OUT`

The file-emission layer. Everything here is deterministic: given a model, the files that
land on disk are fixed. See §8 for the contract these implement.

| ID | Capability | Class |
|---|---|---|
| OUT-01 | Emit all twelve arc42 sections — fact-backed ones filled, the rest stubbed with an explicit note of what only a human can supply | DET |
| OUT-02 | **Enforce the regeneration boundary** — tool-owned `*.generated.md` are overwritten wholesale; human-owned files are never written | DET |
| OUT-03 | **Never read human-owned files** either — they are linked, not parsed | DET |
| OUT-04 | Stamp every generated file with source commit, model version, and rules applied | DET |
| OUT-05 | Emit `index.md` linking generated and human-owned sections into one document | DET |
| OUT-06 | Emit static-site config (`mkdocs.yml`) so the docs build unmodified | DET |
| OUT-07 | Guarantee the output renders with archdoc absent — GitHub-native Mermaid, embedded SVG, no build step required | DET |
| OUT-08 | Generate **contextual stubs** for human-owned sections — questions derived from the model, not generic TODOs | DET |
| OUT-09 | Report **completeness** — which human sections are empty, filled, or stale; arc42 conformance percentage | DET |
| OUT-10 | Detect **staleness** — a human section whose last commit predates architectural changes affecting it | DET |

> **OUT-02 and OUT-03 are the ones that decide whether anyone keeps using the tool.** A
> documentation generator that eats a person's writing gets uninstalled once. Strict
> file-level ownership means there is no merge step to get wrong.

### 4.12 Answer surface — `ANS` *(phase R1.c)*

Editing scoped to **filling gaps**, not managing files. The surface is "answer the question
this section is asking", never "open any file in the repository". Naming it `ANS` rather
than `EDT` is deliberate — it should not drift into being a general editor.

| ID | Capability | Class |
|---|---|---|
| ANS-01 | Present one human-owned section at a time, headed by its generated stub question (OUT-08) | DET |
| ANS-02 | Edit that section's body only — no file tree, no tabs, no arbitrary file access, no create or delete | DET |
| ANS-03 | Save writes exactly that one file, nothing else | DET |
| ANS-04 | Hash content on load; refuse to overwrite if the file changed on disk since it was opened | DET |
| ANS-05 | Refuse to write while a scan or generate is in flight | DET |
| ANS-06 | Preview the section as it will render in the published docs | DET |
| ANS-07 | Mark a section answered; completeness and staleness views (OUT-09, SUR-15) update from it | DET |

**Why this is worth building despite the editor already existing.** Context locality: you
are looking at the diagram, you see the gap, and the question is already on screen. Sending
the author to another application at that moment discards the context that prompted the
writing. What it must *not* become is a competitor to a real editor — no vim bindings, no
snippets, no git integration, and none of that is wanted. Anyone preferring their own editor
keeps using it; the files are plain markdown either way.

---

## 5. The catalog layer — the design element that emerged from this exercise

Classifying the capabilities above surfaced something worth stating on its own: a large
share of the "semantic" work isn't semantic at all. It's **known**.

`postgres:16` is a PostgreSQL database. Port 6379 is Redis. `traefik` is a reverse proxy.
`minio` is S3-compatible object storage. None of that needs a language model — it needs a
lookup table.

**Proposal: a curated knowledge base sitting between mechanical parsing and the LLM.**

| Contains | Example |
|---|---|
| Common images → type, label, responsibility, default protocol | `redis` → cache/queue, "in-memory data store", RESP |
| Well-known ports → protocol | 5432 → PostgreSQL wire protocol |
| Env-key patterns → external dependency | `AWS_S3_BUCKET` → AWS S3 |
| Framework markers → tech stack | `next.config.js` → Next.js |

Why this matters:

1. **It shrinks the LLM's job to genuinely custom services** — usually a minority of any
   compose file.
2. **It raises accuracy.** A table is right every time; a model is usually right.
3. **It cuts cost and latency**, and widens what works in fully offline mode.
4. **It is a real, inspectable, versionable engineering artifact** — something you built,
   not something you prompted.
5. It degrades honestly: an unknown image simply routes to the LLM path, flagged as such.

The catalog is the reason so many rows above read `CAT` rather than `LLM`. It's worth
building early — roughly 150–200 entries covers the overwhelming majority of real
compose files.

---

## 6. LLM dependency analysis

### The count

Counted from §4.1–§4.11 — the 109 **R1.a** capabilities (shares rounded). §4.12 is phase
R1.c and excluded:

| Class | Capabilities | Share |
|---|---|---|
| **DET** — pure deterministic | 89 | 82% |
| **CAT** — catalog lookup | 6 | 6% |
| **HYB** — catalog first, LLM for the tail | 4 | 4% |
| **LLM** — requires a model | 7 | 7% |
| **LLM optional** — works without, nicer with | 1 | 1% |
| **Split** — deterministic contract, model-generated content | 2 | 2% |

**95 of 109 capabilities (87%) require no model at all**, and every LLM-required
capability lives in a single area (SEM).

### What the LLM is actually for

Exactly three things:

1. **Naming** things the catalog doesn't recognize (SEM-02).
2. **Explaining** — responsibilities and edge purposes (SEM-03, SEM-08).
3. **Grouping and framing** — logical layers, boundary names, the C4 context view
   (SEM-05, SEM-06, SEM-07).

Plus prose for the arc42 sections (SEM-09) and, optionally, the diff narrative (MEM-08).

### What survives if the LLM is removed entirely

This is the test worth running, and the answer is the strongest structural argument the
project has:

- Full service inventory ✓
- All edges, with protocols for known ports and URL schemes ✓
- Declared vs. referenced evidence on every node, external systems included ✓
- Service interfaces from OpenAPI / gRPC / GraphQL contracts ✓
- Network and deployment boundaries ✓
- Validated, versioned model ✓
- Rendered C4 container diagram ✓
- Full provenance and click-to-code ✓
- **Architectural diff between commits ✓**
- Mermaid / draw.io / PlantUML export ✓

**What you lose:** friendly labels for custom services, responsibility sentences, derived
logical groupings, the context-level view, and generated prose.

In other words: **remove the model and you still have a working architecture
documentation tool.** It's less readable, not less true. That is what "the LLM is a
component, not the product" means concretely — and it is now demonstrable rather than
asserted.

### Consequence for the build order

Build the deterministic spine first. The LLM enters late and can be developed against a
system that already works. That also means:

- Development is cheap — most iteration costs nothing.
- Tests are real tests, not eyeball checks, because most of the system is deterministic.
- A failed or unavailable model degrades the product; it doesn't break it.

---

## 7. Distribution and shape

### What the product physically is

**A CLI-first local tool, installed as a package for whatever runtime the stack settles
on, which can also serve a local web app as a foreground process.**

Not a container. Not a daemon. Not a hosted service.

**Why not a container.** archdoc reads arbitrary local repositories, writes into them, and
its click-to-code feature has to open a file in *your* editor on *your* machine. From
inside a container that means bind-mounting every repo you touch, fighting file ownership
on every write, and click-to-code cannot work at all — the container has no idea what your
editor is. Containers suit services with a fixed workspace. This is a tool that visits
your filesystem.

**Why not a daemon.** Nothing here is always-on. Generation is on demand. `archdoc serve`
is a foreground process you start and stop, like any dev server. A daemon would add
start/stop/status, stale-process handling, and port conflicts for no benefit. If
regenerate-on-save ever arrives, it is a file watcher *inside* `serve` — still not a
daemon.

### One engine, two thin frontends

```
archdoc-core     extraction → catalog → rules → model → validate → render
   ├── CLI       argument parsing over the engine
   └── HTTP      the same engine behind a localhost API + pre-built static frontend
```

The CLI and the web app must never become two implementations. `archdoc serve` runs a
single process serving both the localhost API and the built frontend assets — there is no
separate frontend deployment and no second service. The engine stays testable with neither
frontend present, which matters a great deal given that most of it is deterministic.

This decision is independent of the stack and holds regardless of what gets chosen.

### The zero-service rule

> **archdoc runs as one process when serving, and zero processes otherwise. Any stack
> choice that requires a background service is disqualified.**

The moment the tool needs a database server, one of three bad things happens: the user
installs and runs a server in order to draw a diagram; or archdoc spawns containers, which
makes it depend on a container runtime being installed *and running*, forces it to manage
container lifecycle and port conflicts, and turns a sub-second `archdoc scan` into a
multi-second cold start; or it becomes a hosted service, contradicting the local-tool
identity entirely.

Every storage need in this project is satisfiable by an **embedded, file-based** engine —
a library that reads a file, with no process, no port, and no install:

| Need | Requirement on storage |
|---|---|
| Model and version history | Embedded persistence, single-writer |
| Graph traversal | Recursive queries, or an in-memory graph over loaded rows |
| Flexible/nested fields | JSON column support |
| Doc search (optional) | Embedded full-text index |
| Vector search (only if RAG returns) | Embedded vector extension |
| Web serving | In-process HTTP; no external web server |
| Frontend delivery | Pre-built static assets served by that same process |
| LLM | Remote API — or a local runtime the user already owns, never archdoc's dependency |

**Scale check:** a container-level model is dozens to low hundreds of nodes. Version
history across hundreds of commits is megabytes. Single user, single process, no
concurrency. No volume, concurrency, or query requirement anywhere in this project
justifies a database server.

**Escape hatch.** Put storage behind a narrow interface with the embedded engine as the
default driver. If a server-backed store is ever genuinely needed, it becomes a driver
rather than a rewrite — and costs nothing to keep open now.

*The specific engine is deliberately unnamed here; it is a stack decision, recorded in
`stack-decision.md`. What is decided here is the constraint it must satisfy.*

### What it leaves in a repository

```
.archdoc/
  config.yaml                    committed  — no secrets; API keys live in the environment
  rules.yaml                     committed  — your corrections
  model.<db>                     ignored    — rebuildable index
docs/architecture/
  index.md                       generated  — links both sets
  0N-*.generated.md              generated  — tool-owned, overwritten every run
  0N-*.md                        human      — never read, never written
  diagrams/*.svg                 generated  — exact app layout
  diagrams/*.mmd                 generated  — portable, hand-editable
  model.json                     generated  — the canonical artifact, one per version
  mkdocs.yml                     generated  — optional site build
```

The generated/human split is the **regeneration boundary** (§8) — the single most
important property of the output contract.

**The committed artifact is text; the database is a rebuildable index.** That keeps version
history surviving a clone without putting an unmergeable binary blob into git, and it is
exactly where the readable composite stable IDs (§4.3) pay off — the git diff of the model
itself stays readable.

### You never run the systems you document

archdoc reads declarations statically. Testing against a large system means `git clone`,
not starting it up. No running services, no databases, no gigabytes of containers on the
machine to validate the tool against real projects.

That falls directly out of choosing declarations as ground truth over runtime
introspection, and it means the entire test suite is a handful of cloned repositories and
some fixture files — which is also why the deterministic acceptance criteria (§12) run
trivially in CI.

### Where a container does belong

R2. When `archdoc check` runs inside someone else's CI, a container is the right delivery:
fixed workspace, no editor integration, no local filesystem to roam. Same engine, different
wrapper — a later addition, not something the architecture has to accommodate now.

---

## 8. Deliverables and the output contract

### What the standards actually require

**arc42 is a structure, not a format.** Twelve sections, shipped as templates in Markdown,
AsciiDoc, Word, and Confluence. It prescribes *what to say and in what order*, never how to
render it.

**C4 is a model, not a notation.** Worth noting: Simon Brown's own tooling (Structurizr)
is built on the same principle this project chose independently — define the model once,
render many views. The canonical-graph decision matches the reference implementation of
the standard rather than departing from it. Structurizr DSL is a plausible future export
target.

**Docs-as-code is the ecosystem norm:** text in the repository, diagrams as text, versioned
in git, optionally rendered by a static site generator.

No standard asks for a PDF.

### The four output layers

| Layer | Artifact | Purpose | Lives |
|---|---|---|---|
| 1 | `model.json` | Machine truth. Not for reading. | Committed |
| 2 | **Markdown + diagram sources** | **The primary human deliverable** | Committed in `docs/architecture/` |
| 3 | Static site | Browsable, searchable, shareable | Generated from layer 2 |
| 4 | `archdoc serve` | Explore, drill into code, compare versions | Local only |

> **Markdown is the deliverable. The app is the workbench.**

The app does what markdown cannot — click a box and land in the source, scrub a version
timeline, see fact distinguished from interpretation. But the app is where *you* work. What
you hand to someone else, and what survives archdoc being uninstalled, is the committed
text.

### The hard requirement

**The output must be fully readable with archdoc absent.**

This is what keeps the deliverable a deliverable rather than a view into a tool. It also
buys a free rendering surface: GitHub renders Mermaid inside markdown natively, so
committed docs are browsable in the repository with no site generator, no build step, and
no infrastructure at all.

Layer 3 therefore stays optional. MkDocs Material is the right choice when a real site is
wanted — native Mermaid, search, versioning — and archdoc emits an `mkdocs.yml` so it
builds unmodified. Nothing is blocked without it.

### Diagram format — both, deliberately

Layout is already computed deterministically and persisted per version (VIE-03/04). That
makes an option available that most tools do not have:

| Format | Strength | Weakness |
|---|---|---|
| **Mermaid** | Renders on GitHub and in every SSG; diffable; editable by hand | Does its *own* layout, so the committed picture will not match the app. C4 support still experimental |
| **Pre-rendered SVG** | Exactly the layout seen in the app; renders anywhere | Not hand-editable; a large blob in git |

**Decision: emit both.** Markdown embeds the SVG for fidelity; the `.mmd` sits alongside
for portability, hand-editing, and diffing. Both derive from the same model, so the second
one costs almost nothing. Sticky layout (VIE-04) means the SVG only churns when the
architecture actually changes.

### arc42 completeness — partial output is the honest output

archdoc can fill roughly five of twelve sections from facts: Context & Scope, Building
Block View, Runtime View, Deployment View, and part of the Glossary. Solution Strategy,
Quality Requirements, Architecture Decisions, and Risks & Technical Debt need a human who
knows *why*.

**Emit all twelve anyway.** Fill the fact-backed ones; stub the rest with an explicit note
of what only a person can supply. The skeleton then works as a structured prompt — archdoc
does the mechanical part and states precisely where human knowledge is required. That is
more useful than five orphaned files, and it is the honest expression of P1: the tool never
pretends to know something it cannot prove.

### The regeneration boundary

The requirement that falls out of the above, and the one most likely to destroy the product
if it is got wrong.

If archdoc overwrites human prose, people stop using it. If it never regenerates, it goes
stale. Both failure modes have killed documentation tools.

**The answer is strict file-level separation — no merge logic, no marker comments:**

```
docs/architecture/
  03-context-and-scope.generated.md      tool-owned, overwritten every run
  05-building-block-view.generated.md    tool-owned
  06-runtime-view.generated.md           tool-owned
  04-solution-strategy.md                human-owned, never touched
  09-architecture-decisions.md           human-owned, never touched
  index.md                               generated; links both sets
```

- Generated files are wholly owned by the tool and carry a header naming the source commit,
  the model version, and the rules applied.
- Human-owned files are **never read and never written**, only linked.
- There is no merge step to get wrong and no marker comment to mis-parse.

### Who writes what — there is no second application

The human-owned sections raise an obvious question: what interface does a person use to
write them? The answer is that **the interface already exists — it is their editor.**

`04-solution-strategy.md` is a markdown file in the repository. It is opened in whatever
editor the person already uses, tracked by git, reviewed in pull requests, and rendered by
the site. That is what docs-as-code means: the authoring tool is the one already in hand.
The same applies to `config.yaml` and `rules.yaml`.

Building a markdown editor inside archdoc would mean reimplementing a worse editor, adding
a save/dirty-state/conflict surface, for files that are better edited elsewhere. MkDocs
does not help here either — it is a renderer, and nothing was ever supposed to edit inside
it.

| Surface | Role | Writes |
|---|---|---|
| **The person's editor** | Author human sections and rules | The only writer of human content |
| **`archdoc serve`** | Explore, inspect provenance, verify, see gaps | Only the model and generated files |
| **Site / GitHub** | Read the finished document | Nothing |

One application, and in R1.a it is a **workbench, not an editor**. No automated code path
writes a human file, so there is no path by which regeneration can destroy someone's
writing.

**R1.c adds one narrow exception (§4.12): the answer surface.** It edits the body of a
single human-owned section at a time, headed by the question that section is asking, and
writes only on an explicit save after a hash check. It is not a file manager and not a
general editor — it exists because the moment you notice a gap is while looking at the
diagram, and sending the author elsewhere at that moment throws away the context that
prompted the writing. The invariant that matters is unchanged and is what NFR-11 now states
precisely: **nothing automated ever writes human content.**

### What archdoc owes the human author

The real gap is not editing. It is **knowing what to write, where, and whether it is still
true.** That is archdoc's responsibility, and it is three capabilities:

**Contextual stubs (OUT-08).** A stub reading "TODO: write solution strategy" is worthless.
A stub generated from the model is not:

> *This system has three data stores — PostgreSQL, Redis, and S3-compatible object storage
> — and one external identity provider. Explain why each was chosen and which alternatives
> were rejected.*

Derived entirely from facts already extracted, it turns an empty section into an answerable
question. This is what makes partial arc42 output a feature rather than an apology.

**Completeness reporting (OUT-09, SUR-15).** The app shows which human sections are empty,
filled, or stale, alongside the arc42 conformance percentage. Clicking a section opens the
file in the editor — the same mechanism as click-to-code (PRV-04), reused.

**Staleness detection (OUT-10).** Compare when a human file was last committed against when
the architecture last changed. If two services appeared after `04-solution-strategy.md` was
last touched, flag it. Deterministic, needs no model, and it addresses the failure that
actually occurs in practice: human sections quietly describing a system that has moved on.

### PDF

Not a first-class output. A PDF is a snapshot of a moving system, and shipping one means
somebody reads a stale architecture with confidence.

It is a legitimate *export* from layer 3 for a specific moment — a submission, a handoff, a
presentation — stamped with commit and date. Nothing in the architecture needs to
accommodate it.

---

## 9. Non-functional requirements

| # | Requirement | Target |
|---|---|---|
| NFR-1 | `archdoc scan` on a typical repo | < 30 s |
| NFR-2 | Full `archdoc generate` | < 3 min |
| NFR-3 | **Egress policy** — config declares what may leave: `structure-only` (names, paths, edges — no file contents), `excerpts`, or `local` (Ollama, nothing leaves). Default `structure-only`. Every run prints what was sent. | Enforced and auditable |
| NFR-4 | Fully offline operation via a local model | Supported |
| NFR-5 | Determinism — extraction byte-identical across runs on identical input | Absolute |
| NFR-6 | Cost estimate before a run, actual after | Reported |
| NFR-7 | The model is never persisted in an invalid state | Absolute |
| NFR-8 | Unsupported or unparseable config degrades with a warning, never a crash | Absolute |
| NFR-9 | **Zero runtime services** — no database server, broker, or sidecar. One process when serving, zero otherwise. | Absolute |
| NFR-10 | Install and first useful output on a clean machine | Single package install, then `scan` with no further setup |
| NFR-11 | **No automated path writes human-owned files.** Generation, regeneration, and rules never read or write them (§8). The R1.c answer surface writes only one named file, only on an explicit save, only after a hash check | Absolute |
| NFR-12 | Generated output renders with archdoc uninstalled | Absolute |

**NFR-3 is not optional.** You work on code you don't own. The tool must be able to prove
what it transmitted, and default to transmitting the least.

---

## 10. Data model

```
runs          id, commit_sha, started_at, finished_at, status,
              coverage_stats, tokens, cost, egress_mode

versions      id, run_id, parent_version_id, created_at

identities    id, kind, canonical_key, first_seen_version

aliases       id, identity_id, stable_id, from_version, to_version,
              origin              -- observed | rule

nodes         id, version_id, identity_id, stable_id, type, evidence, label,
              responsibility, tech, boundary_id, layout_x, layout_y
              -- evidence: declared | referenced

edges         id, version_id, from_node, to_node, label,
              protocol, sync, direction

boundaries    id, version_id, label, kind
              -- network | trust | deployment | external

provenance    id, version_id, element_id, element_kind,
              file, line_start, line_end, origin
              -- declaration | reference | contract | catalog | model

rules         id, match, action, value, note, added_at

operations    id, version_id, op_type, payload, origin  -- extraction | catalog | model | rule
```

`operations` is why undo, replay, and the architectural diff all fall out of one
mechanism instead of being three features. `stable_id` on `nodes` is what MEM-06 depends
on.

---

## 11. Pipeline

```
  repo
   │
   ├─ config / IaC parsers  (DET)  ──┐
   │                                 │
   └─ .md specs ────────────────────┼──▶  FactSet  ── the contract artifact
                                     │              { services, resources, edges,
                                     │                boundaries, provenance }
                          catalog (CAT) ──┘
                                     │
                    rules.yaml ──▶ [RULES] ──▶ FactSet′
                                     │
                                 [SEM / LLM] ──▶ OperationSet (structured diff)
                                     │
                               [VALIDATOR] ──▶ valid? ──no──▶ retry with error
                                     │ yes                    (budget 3, then fail loud)
                                     ▼
                               new Version ──▶ views ──▶ render / export / docs
```

**FactSet is the contract between the deterministic half and the probabilistic half.**
It is snapshot-testable. If the SEM stage disappeared, a FactSet still renders a real
diagram.

---

## 12. Acceptance criteria

Against the three test subjects: **Immich** (simple multi-service), **self-hosted
Supabase** (complex, close to the reference topology), and **this project itself**.

| # | Criterion | Threshold |
|---|---|---|
| AC-1 | Nodes and edges with extraction or catalog provenance | ≥ 95% (container level should be near-total) |
| AC-2 | Container diagram produced with the LLM disabled | Renders and validates on all three subjects |
| AC-3 | Structural accuracy vs. hand-drawn reference | ≥ 0.85 precision and recall on Immich; reported on Supabase |
| AC-4 | Validator rejects malformed models | 100% of a fault-injection suite |
| AC-5 | Rule persistence | Apply 10 rules, regenerate from scratch, all 10 still in effect |
| AC-6 | Drift detection | 100% of a synthetic drift set detected and classified |
| AC-7 | Determinism | 5 consecutive `scan` runs produce byte-identical FactSets |
| AC-8 | Egress | `structure-only` transmits zero file contents; run log accounts for every byte |
| AC-9 | Performance | NFR-1 and NFR-2 met on Supabase |

AC-2, AC-4, AC-6, and AC-7 need no human judgment and no external subjects. They are
self-contained suites — build them early, they're what make refactoring safe.

---

## 13. Open items

### Settled

- **Timeline: open.** No fixed submission date, so scope is not clock-driven. The extra
  room goes into **depth inside R1** (phases R1.b and R1.c) and makes R2 genuinely
  reachable — not into changing what the product is. The identity in §1, the release
  ordering in §2, and the deterministic-first build order in §6 were chosen for reasons
  independent of the schedule and stay fixed.
- **Shape: settled (§7).** CLI-first local tool, one engine behind two thin frontends,
  zero runtime services. Not a container, not a daemon.
- **Stack: settled, and recorded separately.** The idea was settled first, then the stack
  chosen, then the capability catalog revisited — the order §13 always intended. The
  decision, its rationale, the alternatives rejected and the resulting cost changes to §4
  live in **`stack-decision.md`**, deliberately kept out of this document so the definition
  stays about *what the product is* rather than what it is built with. Nothing in §§1–12
  changed as a result. The shape in §7 was chosen independently of the stack and still holds.
- **Catalog seeding: form settled, curation outstanding.** How entries are stored and
  processed is a stack matter and is answered in `stack-decision.md`. What the survey settled
  here is that the catalog **must go beyond common infrastructure images**: only ~14 of 36
  image references across both subjects are identifiable from well-known image names, and
  both projects ship their database under a vendor namespace. Which entries to seed remains
  curation work.
- **Orchestrator manifests: out of R1 scope, R2 the likely home.** Kubernetes, Helm and
  Kustomize extraction is deferred. Two reasons. First, evidence: neither test subject
  contains a single k8s, Helm, Kustomize or application-level Terraform file (survey §4.6),
  so the capability would be built with nothing to validate it against — which contradicts
  the evidence-first posture of §0. Second, cost: it is a second extraction surface with its
  own identity, templating and overlay semantics, and R1's value does not depend on it.
  Revisit if R1 finishes with room; otherwise it lands in R2. **Consequence:** §3's
  extraction sources are Compose, `.env`, gateway/proxy config, and interface contracts —
  "manifests" is no longer among them, and the third test subject (O-7) need not be a
  Kubernetes system.
- **Docs delivery in R1: local only** — served from the local app, Swagger-UI style.
  Publishing, sharing, and remote reachability are R2 concerns (§2, R1.x).
- **Trust boundaries in R1: declared only** — no inference of reachability (§2, R1.x).
- **Identity model: the registry (§4.3).** Canonical identities with alias history, resolved
  from stable IDs at extraction time, rules able to pin an alias by hand. Renames are alias
  additions, not delete-plus-add. This was the one open question that could not wait for the
  stack, and it is now closed.

### Now closed

O-1 (identity registry) closed earlier. O-2 through O-5 are now closed.

| # | Question | Outcome |
|---|---|---|
| O-2 | Stack | **Closed** — see `stack-decision.md`. |
| O-3 | Storage engine | **Closed** — a stack matter; the engine is named in `stack-decision.md`. §7's constraint and driver interface are unchanged. |
| O-4 | Contract-file matching | **Closed — the answer is rules.** The survey tested four strategies against six specs across both subjects. Directory proximity fails on both: Immich's spec sits in a top-level `open-api/`, Supabase's six sit inside the documentation app. Spec self-identification works once in six. The build-context chain breaks because Immich's canonical compose file has no `build:` key at all. **`rules.yaml` is therefore the primary mechanism**, with filename- and port-matching offered as suggestions a human confirms. This is a design consequence, not a stack one: it makes the rules layer (§4.9) **load-bearing rather than merely refining**, and it strengthens AC-5. |
| O-5 | Catalog seeding | **Closed in form.** Storage form is a stack matter (`stack-decision.md`). The design finding is that the catalog must reach beyond well-known image names — see Settled above. |

### Still open

| # | Question | Kind | How it gets resolved |
|---|---|---|---|
| O-7 | **Test subject #3** | Decision | Was expected to fall out of the stack choice, and did not. archdoc documenting itself is a weak subject by §3's own vantage-point argument — a CLI tool with zero declared services yields close to one box, and §3 is explicit that what matters is *what the repository declares*. With orchestrator manifests out of R1 scope, the third subject need not be a Kubernetes system either. The real question is what a third subject should demonstrate that Immich and Supabase do not. |

One further open item, **O-8**, is stack-specific — whether file-and-line provenance
survives Compose merge and interpolation, which bears directly on PRV-01 and P1. It is
recorded in `stack-decision.md` and is the first thing to establish before building.

### Resolve during building

- **O-6 — R1.b sequencing.** Does static code analysis come before semantic clustering?
  They are coupled: clustering needs the intra-service graph that analysis produces, which
  argues for analysis first — but clustering is the one genuinely unsolved problem, which
  argues for prototyping it early against hand-built fixtures. **Not a stack question.**
  Decide it with a working R1.a spine in front of you, not before.

### Left open by design

Everything in **R1.x** (§2). Those are not unresolved questions — they are places where R1
deliberately stops at what a declaration proves, with R2 named as the resolution.

---

## 14. Reference — the standards and formats named here

Short definitions of every term this document leans on, and what archdoc does with each.

### The two that shape the output

**C4 model** — an abstraction model for software architecture diagrams, by Simon Brown.
Four nested levels of zoom:

| Level | Shows | archdoc |
|---|---|---|
| 1. **Context** | The system as one box, plus its users and external systems | R1.a, LLM-assisted |
| 2. **Container** | The separately deployable/runnable pieces inside it | **R1.a — the core deliverable** |
| 3. **Component** | The major parts inside one container | R1.b |
| 4. **Code** | Classes and functions | Out of scope — IDEs do this |

C4 is a *model*, not a notation — it says nothing about how to draw. Its central rule is
that the model is separate from the diagram, and views are projections of it. That is the
same principle this project arrived at independently (§1, §8).

> ⚠️ **Naming collision to watch.** A C4 "container" is *not* a Docker container. It means
> anything separately deployable and runnable — a web app, an API service, a database, a
> file store. In this project both meanings appear constantly: archdoc parses Docker
> containers in order to produce C4 containers. Keep the words distinct in code and prose.

**arc42** — a template for architecture documentation, by Gernot Starke and Peter Hruschka.
Twelve sections, in fixed order. It is *structure*, not format — it ships as Markdown,
AsciiDoc, Word, and Confluence templates and prescribes only what to say and where.

| # | Section | archdoc |
|---|---|---|
| 1 | Introduction and Goals | human |
| 2 | Architecture Constraints | human |
| 3 | Context and Scope | **generated** |
| 4 | Solution Strategy | human |
| 5 | Building Block View | **generated** |
| 6 | Runtime View | **generated** |
| 7 | Deployment View | **generated** (R1.a stretch) |
| 8 | Cross-cutting Concepts | human |
| 9 | Architecture Decisions | human (see ADR below) |
| 10 | Quality Requirements | human |
| 11 | Risks and Technical Debt | human |
| 12 | Glossary | partly generated |

Five generated, seven human — which is why the regeneration boundary and the contextual
stubs (§8) exist at all.

### Practices

**ADR — Architecture Decision Record.** A format popularised by Michael Nygard: one file
per decision, with Status, Context, Decision, and Consequences. It records *why* a choice
was made and what was rejected — precisely what git does not capture. R1 does not generate
ADRs; capturing decisions is R2's direction (§2).

**Docs-as-code.** Not a standard but the ecosystem norm: documentation lives as text in the
repository, diagrams are text too, everything is versioned in git, reviewed in pull
requests, and built in CI. It is the reason the primary deliverable is committed markdown
rather than an application view (§8), and the reason no separate editor is needed.

### Diagram formats

| Format | What it is | archdoc |
|---|---|---|
| **Mermaid** | Text-to-diagram syntax rendered by a JS library. GitHub renders it natively inside markdown; most static site generators support it. Does its own layout. C4 support exists but is still experimental | **Emitted** — portable, diffable, hand-editable |
| **SVG** | Rendered vector image | **Emitted** — carries archdoc's own computed layout, so the committed picture matches the app |
| **PlantUML** | Older text-to-diagram language; the community **C4-PlantUML** macros give it strong C4 fidelity. Needs a renderer | Export target |
| **draw.io XML** | The file format of the diagrams.net editor | Export target — for hand-editing after generation |
| **Structurizr DSL** | Simon Brown's own C4 tooling. A DSL where you define the model once and render many views | Not used; noted as prior art for the model-first approach, and a plausible future export |

### Site generation

**MkDocs / MkDocs Material** — a static site generator that turns a directory of markdown
into a searchable site, with native Mermaid support and versioning. archdoc emits an
`mkdocs.yml` so the docs build unmodified, but the output is designed to be readable
without it (§8, NFR-12).

### Interface contract formats — extraction sources

These are declarative interface descriptions, which is why they belong in extraction rather
than code analysis (§4.2).

| Format | What it is | Yields |
|---|---|---|
| **OpenAPI** (the spec formerly called Swagger; *Swagger* now refers to the tooling around it) | A declarative description of a REST API — paths, operations, schemas | Endpoints a service exposes |
| **gRPC / Protocol Buffers** (`.proto`) | Declarative service and message definitions for RPC | Services and their RPC methods |
| **GraphQL SDL** | Schema Definition Language — types, queries, mutations | Schema entry points and types |
