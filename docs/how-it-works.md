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
   │  4  REFINE     your rules.yaml corrections       │ ✅
   │                                                  │
   │  5  LABEL      ← the LLM, and only here          │ ⬜
   │                names, descriptions, groupings    │
   │                                                  │
   │  6  VALIDATE   reject contradictions             │ ✅
   │                                                  │
   │  7  STORE      one version per scan  (SQLite)    │ ⬜
   │                                                  │
   │  8  RENDER     Mermaid ✅ · SVG + layout ⬜      │ 🟡
   └──────────────────────────────────────────────────┘
                          │
                          ▼
              WRITTEN BACK INTO YOUR REPO
   ┌──────────────────────────────────────────────────┐
   │  docs/architecture/                              │
   │     ├── *.svg      C4 diagrams                   │
   │     ├── *.mmd      same, as text for GitHub      │
   │     └── *.md       arc42, 12 sections            │
   │  .archdoc/                                       │
   │     ├── model.json     machine-readable          │
   │     └── history        every past version        │
   └──────────────────────────────────────────────────┘
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
| 6 Deployment | ⚠️ If orchestrator manifests return | Compose alone yields a shallow one — a host containing containers. Real value arrives with Kubernetes, which is R2 |
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
node.kind      application | datastore | queue | proxy | external | actor | system
node.evidence  declared | referenced
node.parent    which container it lives inside   (empty until level 3)
```

| View | Is a filter over the model | New evidence needed |
|---|---|---|
| **Container** | `kind ∈ {application, datastore, queue}`, plus the external systems they touch | Node kind, edge protocol, catalog for technology |
| **Context** | Collapse everything `declared` into one box; keep `referenced` and actors; keep edges crossing the line | Only **actors** |
| **Component** | Nodes whose `parent` is a given container | Everything — a new source entirely |

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
| 2 Extract | `internal/extract` | Two passes — O-8 established that positions do not survive the merge. Also reads what the compose file *points at*: dotenv files it names, and gateway configs it mounts |
| 3 Derive | `internal/model` | The seam where the two workstreams meet: above it reads files, below it draws |
| 4 Refine | `internal/rules` | `rules.yaml`; load-bearing, since O-4 made rules the primary mechanism for contract attachment. Compiles to the same operations the semantic layer emits |
| 5 Label | `internal/semantic` | The only package permitted outbound calls |
| 6 Validate | `internal/validate` | Every rule traceable to a failure seen in the draw.io experiment. Two severities: wrong is refused, thin is published and reported |
| 7 Store | `internal/store` | SQLite behind a driver interface |
| 8 Render | `internal/render` | Mermaid today. Layout and SVG come with sprint 2 |

## Two ways in, one engine

```
archdoc scan | generate | diff | export        the command line
archdoc serve                                  the same engine, in a browser
```

The web app holds no logic — it renders what the engine already computed. That is what stops
the two surfaces from drifting into two implementations.

---

**Try it:** `archdoc generate <path-to-a-repo>` writes both diagrams and their evidence into
`docs/architecture/` in that repository.

**Read next:** `product-definition.md` §11 for the pipeline in full, §8 for what lands on disk,
and `stack-decision.md` §2.3 for what each library does.

**On C4 specifically:** the quotations above are from *"The C4 Model — Beyond the Basics"*,
Simon Brown, YOW! 2025 — transcript in this directory. Worth reading if you are deciding what a
view may show; it is the source for the derivation rule and for excluding gateways.
