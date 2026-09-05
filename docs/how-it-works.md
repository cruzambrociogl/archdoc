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
   │  2  EXTRACT    read it, twice                    │ 🟡
   │                compose-go → what is true         │
   │                yaml.v3    → what line said it    │
   │                          ↓                       │
   │                      FACTSET                     │
   │            services · edges · file:line          │
   │        ── everything below is derived ──         │
   │                                                  │
   │  3  REFINE     your rules.yaml corrections       │ ⬜
   │                                                  │
   │  4  LABEL      ← the LLM, and only here          │ ⬜
   │                names, descriptions, groupings    │
   │                                                  │
   │  5  VALIDATE   reject contradictions             │ ⬜
   │                                                  │
   │  6  STORE      one version per scan  (SQLite)    │ ⬜
   │                                                  │
   │  7  RENDER     positions once, then draw         │ ⬜
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

**Only step 4 touches the network.** Everything else is local computation. Remove the model
entirely and you still get a correct, complete, traceable diagram — just with duller labels.
This is enforced rather than promised: CI fails if any package outside `internal/semantic`
imports an HTTP client.

**Facts flow down, never up.** The FactSet is the dividing line. Above it, every value was read
from a file and carries the line that declared it. Below it, everything is derived *from* those
facts. Nothing invents a box.

**It stays true.** Step 6 keeps every past version, so running it again after new commits
answers *"this service appeared, this connection is new"* rather than producing a fresh
picture with no memory.

## Where each stage lives

| Stage | Package | Notes |
|---|---|---|
| 1 Discover | `internal/extract` | Content-sniff for recall, reject fragments for precision |
| 2 Extract | `internal/extract` | Two passes — O-8 established that positions do not survive the merge |
| 3 Refine | *not yet* | `rules.yaml`; load-bearing, since O-4 made rules the primary mechanism for contract attachment |
| 4 Label | `internal/semantic` | The only package permitted outbound calls |
| 5 Validate | *not yet* | Every rule traceable to a failure seen in the draw.io experiment |
| 6 Store | `internal/store` | SQLite behind a driver interface |
| 7 Render | `internal/render` | Layout once in the engine, stored per version; the web canvas draws saved coordinates |

## Two ways in, one engine

```
archdoc scan | generate | diff | export        the command line
archdoc serve                                  the same engine, in a browser
```

The web app holds no logic — it renders what the engine already computed. That is what stops
the two surfaces from drifting into two implementations.

---

**Read next:** `product-definition.md` §11 for the pipeline in full, §8 for what lands on disk,
and `stack-decision.md` §2.3 for what each library does.
