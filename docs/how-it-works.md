# How archdoc works — the one-page view

A quick orientation. The authoritative versions are §11 (pipeline) and §8 (output contract) of
`product-definition.md`; this is the shape of it on one screen.

**Status key:** ✅ built · 🟡 partly built · ⬜ not started

---

```
                    YOUR REPOSITORY
                 (someone else's code)
                          │
                          ▼
   ┌──────────────────────────────────────────────────┐
   │                  archdoc scan                    │
   │                                                  │
   │  1  DISCOVER   which file is the architecture?   │ ✅
   │                10 candidates → 1 chosen          │
   │                                                  │
   │  2  EXTRACT    read it, twice                    │ ✅
   │                compose-go → what is true         │
   │                yaml.v3    → what line said it    │
   │                          ↓                       │
   │                      FACTSET                     │
   │       services · depends_on · ports · endpoints  │
   │              every one with file:line            │
   │        ── everything below is derived ──         │
   │                                                  │
   │  3  DERIVE     facts → one graph                 │ ✅
   │                kinds · edges · actors · external │
   │                                                  │
   │  4  REFINE     your .archdoc/rules.yaml          │ ✅
   │                                                  │
   │  5  LABEL      ← the LLM, and only here          │ 🟡
   │                names, descriptions, groupings    │
   │                                                  │
   │  6  VALIDATE   reject contradictions             │ ✅
   │                                                  │
   │  7  STORE      a version when the architecture   │ ✅
   │                changed (SQLite), layout stored   │
   │                                                  │
   │  8  RENDER     C4 SVG as you arranged it,        │ ✅
   │                Mermaid, arc42, coverage report   │
   └──────────────────────────────────────────────────┘
                          │
                          ▼
              WRITTEN BACK INTO YOUR REPO
   ┌──────────────────────────────────────────────────┐
   │  docs/architecture/                              │
   │     ├── index.generated.md   both diagrams       │
   │     ├── 01..12  arc42 — 5 generated, 7 yours     │
   │     ├── coverage.generated.md  what it missed    │
   │     ├── *.mmd   diagrams as text                 │
   │     └── *.svg   the drawn C4 diagrams            │
   │  .archdoc/                                       │
   │     ├── model.json     committed — the record    │
   │     ├── coverage.json  committed — as data       │
   │     ├── rules.yaml · layout.yaml · views.yaml    │
   │     │                  committed — yours         │
   │     └── history.db     local cache, git-ignored  │
   └──────────────────────────────────────────────────┘
                          │
              ┌───────────┴────────────┐
              ▼                        ▼
       archdoc serve            archdoc export --site
       the live app             the same app, static,
       (this machine)           for a team (.archdoc/site)
```

---

## Three properties the shape encodes

**Only step 5 touches the network.** Everything else is local computation. Remove the model
entirely and you still get a correct, complete, traceable diagram — just with duller labels.
This is enforced rather than promised: CI fails if any package outside `internal/semantic`
imports an HTTP client.

**Facts flow down, never up.** The FactSet is the dividing line. Above it, every value was read
from a file and carries the line that declared it. Below it, everything is derived *from* those
facts. Nothing invents a box.

**It stays true.** Step 7 keeps every past version, so running it again after new commits
answers *"this service appeared, this connection is new"* rather than producing a fresh
picture with no memory.

## What comes out: the C4 views

### C4 is four zoom levels — plus three other diagram types

The name comes from four levels of **static structure**, each a zoom step into the previous:

```
1 CONTEXT     your whole system as one box, and the world around it
2 CONTAINER   the separately deployable things inside it, and what talks to what
3 COMPONENT   the major parts inside one of those things
4 CODE        classes and functions
```

C4 also defines three diagrams that are **not** levels — they answer different questions
rather than zooming in:

| | Answers |
|---|---|
| **Dynamic** | How does one feature work at runtime? |
| **Deployment** | Where does this actually run? Maps containers onto infrastructure |
| **System Landscape** | How do many systems relate to each other? |

Seven types in total. A deployment diagram is *not* "level 5" — same containers, different
question.

### Which ones archdoc produces

| Type | Status | Why |
|---|---|---|
| **1 Context** | ✅ R1.a | Committed |
| **2 Container** | ✅ R1.a | Committed — the core |
| **3 Component** | 🟡 R1.b | Needs a new evidence source: build manifests or static analysis |
| 4 Code | ❌ Never | Out of scope by design. C4 itself suggests UML here |
| 5 Dynamic | ❌ Not planned | Needs call-flow analysis or runtime traces; we read static configuration |
| 6 Deployment | 🟡 As a table, arc42 §7 (F-15) | What Compose states — the image each container runs, published ports, networks, what the repository mounts in — every row cited. A drawn deployment diagram earns its place with orchestrator manifests, which are R2 |
| 7 System Landscape | ❌ **Structurally impossible** | §3's vantage point: archdoc sees inside *one* repository and treats everything else as opaque. A landscape needs to see inside many. archdoc would be a **participant** in that pattern — the thing that produces one team's model for a shared catalog |

**Two is the right target, not a shortfall.** C4's author, on the same point: *"Most
organizations only use the top two. Quick to create, don't change rapidly. That doesn't mean
there's not value in the bottom two levels, but you need to look at things like **automatic
generation** if you want to use them."* Level 3 is precisely where a tool like this earns its
existence.

### One model, many views

**The levels are not extraction levels. They are projections of one model.** `VIE-01` and
`VIE-02` say *project the model into* a context or container view — the same graph, filtered
differently:

```
node.kind      application | datastore | queue | proxy | external | actor | system | component | table | module
node.evidence  declared | referenced
node.parent    which container it lives inside   (set on components)
```

| View | Is a filter over the model | New evidence needed |
|---|---|---|
| **Container** | `kind ∈ {application, datastore, queue}`, plus the external systems they touch | Node kind, edge protocol, catalog for technology |
| **Context** | Collapse everything `declared` into one box; keep `referenced` and actors; keep edges crossing the line | Only **actors** |
| **Component** | Components whose `parent` is a given container, and the imports between them. A component is a feature — a name spanning the roles its files carry (`album.controller`, `album.service`, …) — where the code is named that way, a folder otherwise | The code itself: parsed with tree-sitter, every import resolved and cited (since 5 Oct) |
| **By folder** | Modules whose `parent` is a given container: the same code by where its files are, kept beside the features | The same |
| **Data** | Tables whose `parent` is a given container, and the foreign keys between them | The classes the code marks as tables — TypeORM-style decorators, SQLModel, SQLAlchemy |

Beside the graph, the model carries what the code says the system *does*: **entries** (HTTP routes
and pages, where a framework declares them), **flows** (what a route sets off, followed call by
call to the tables it touches), **unresolved** calls (an address computed at run time — listed,
never drawn), and **explanations** (model-written sentences about a component, each citing facts).

**The evidence rule is already the system boundary.** *Declared* means the repository defines
it, so it is inside our system. *Referenced* means the repository only points at it, so it is
outside. That distinction exists for evidence honesty (`MDL-15`/`MDL-16`) and turns out to be
exactly what C4 needs to draw the boundary. Context therefore costs almost nothing once
container works.

### The derivation rule

Compose describes **deployment**. A container diagram must not show deployment concepts. So
the mapping has to be stated rather than assumed:

> **A Compose service becomes a C4 container when it is an application or a data store.
> Infrastructure — gateways, proxies, sidecars — stays in the model but is excluded from the
> container view.**

Two consequences worth knowing:

- **`Docker` never appears as a technology on a container diagram.** The technology is what runs
  *inside* the container — `PostgreSQL 14`, `Node.js`, `Go`. Turning an image reference into a
  technology choice is what the catalog is for
- **Excluding the gateway reveals the real edges.** With everything routed through a proxy, a
  diagram looks hub-and-spoke and hides the actual couplings
- **Gateway routes are read, so the arrow does not have to stop at the gateway.** archdoc finds
  the routing table by following the compose file's own bind mounts, which makes the mount line
  the citation for why that file was read. Supabase: seven routes, seven cited lines
- **But only where the evidence says traffic flows.** Relationships through an excluded proxy
  are reconnected to their real endpoints and cite both hops — unless both hops came from
  `depends_on`, which records start-up order and not routing. Supabase makes the difference
  visible: `api-gw` waits for `studio` to be healthy, and reading that as "users reach studio"
  would draw an arrow the file never declared. The real routes are in the gateway's own
  configuration, which is `MDL-03` and a source archdoc does not read yet

### What that means for the three test subjects

| | Services | Containers | Notes |
|---|---|---|---|
| **Immich** | 4 | **4** | Nothing excluded. Boxes gain technology labels instead of image digests |
| **Mastodon** | 5 | **5** | Nothing excluded |
| **Supabase** | 11 | **10** | `api-gw` excluded as a proxy. `supavisor`, a connection pooler, is a genuine judgement call — exactly the case `rules.yaml` exists to settle, and it is currently drawn as a container |

**Measured, 6 Sep.** The prediction above was right about containers and wrong about which
subject has a context diagram worth looking at:

- **Supabase** is the only one with an external system — `supabase-mail`, from
  `GOTRUE_SMTP_HOST`. It is also the only subject that puts its environment *in* the Compose
  file
- **Mastodon** was expected to be the interesting one, for S3, mail and Elasticsearch. Those are
  real, and they are declared in `.env.production.sample` — a file the compose file references
  through `env_file` and which does not exist in the repository. Not a modelling failure: the
  evidence is in a source archdoc does not yet read
- **Immich** is the same case, and additionally its `immich-machine-learning` service has **no
  edges at all**. Every service reaches it over a URL built at runtime. That is §3's finding
  drawn as a picture: configuration declares what exists, not what talks to what

The pattern is one line: **the container view is as good as the compose file; the context view
is as good as the environment**, and the environment is usually somewhere else.

---

## Where each stage lives

| Stage | Package | Notes |
|---|---|---|
| 1 Discover | `internal/extract` | Content-sniff for recall, reject fragments for precision |
| 2 Extract | `internal/extract`, `internal/code` | Two passes — O-8 established that positions do not survive the merge. Also reads what the compose file *points at*: dotenv files it names, and gateway configs it mounts. Applications are found by their manifests, and each running one's code is parsed: files, and every import resolved by path, alias or module, or kept unresolved |
| 3 Derive | `internal/model` | The seam where the two workstreams meet: above it reads files, below it draws. Code becomes components — a directory under the source root, split where one holds most of a large application — and imports become "uses"; table classes become tables; route decorators and page files become entries; a host named in a literal becomes an edge between containers; a configured URL or a known client library becomes an external system; each route is followed into a flow |
| 4 Refine | `internal/rules` | `.archdoc/rules.yaml`; load-bearing, since O-4 made rules the primary mechanism for contract attachment. Compiles to the same operations the semantic layer emits |
| 5 Label, explain | `internal/semantic` | The only package permitted outbound calls. `--label` names and describes containers; run live on Supabase, where structure was identical with it on and off (AC-2). `--explain` asks what each component does: sent names and the code's own route summaries, answered in sentences that must each cite a fact or be refused, remembered in `.archdoc/interpretations.json` by a fingerprint of the facts so nothing is asked twice |
| 6 Validate | `internal/validate` | Every rule traceable to a failure seen in the draw.io experiment. Two severities: wrong is refused, thin is published and reported |
| 7 Store | `internal/store` | SQLite, cgo-free. A version is a change of *architecture*: a run that changes nothing records nothing, and a moved citation refreshes the latest version instead of minting one |
| 8 Render | `internal/render` | Graphviz places, archdoc draws the C4 SVG from stored coordinates, as arranged; Mermaid kept as text — an `erDiagram` for tables, a `sequenceDiagram` for flows; `render.Views` is the one list of diagrams; the coverage report as a page and as data, with what code was read |
| — Arrange | `internal/arrange` | `layout.yaml` and `views.yaml`: where a person put the boxes, and the views they named. Applied over the stored layout, never instead of it |
| — Serve, export | `internal/serve` | The API, the action gate, and the published site built by asking the API's own handlers |

## Three ways in, one engine

```
archdoc scan | generate | history | runs       the command line
archdoc serve                                  the same engine, in a browser — this machine only
archdoc export --site                          the same app as static files, for a team
```

The web app holds no logic — it renders what the engine already computed. It draws each diagram
from the engine's *scene*: the view plus the layout stored for it, arranged as a person saved it,
which is also what the committed SVG is drawn from. That is what stops the surfaces drifting into
separate implementations, and what makes the published site the same as the live one: its data is
the live API's responses, written to files. The app's few writes — an arrangement, a named view —
pass a gate that admits only its own page. The surface in full: `surface-spec.md`.

---

**Try it:** `archdoc generate <path-to-a-repo>` writes both diagrams and their evidence into
`docs/architecture/` in that repository.

**Read next:** `product-definition.md` §11 for the pipeline in full, §8 for what lands on disk,
and `stack-decision.md` §2.3 for what each library does.

**On C4 specifically:** the quotations above are from *"The C4 Model — Beyond the Basics"*,
Simon Brown, YOW! 2025 — transcript in this directory. Worth reading if you are deciding what a
view may show; it is the source for the derivation rule and for excluding gateways.
