# Progress

Tracked against the capability catalog (`docs/product-definition.md` §4) and the acceptance
criteria (§12). These are the project's own work breakdown — no parallel TODO list.

**Status key:** `·` not started · `~` in progress · `✓` done · `⊘` superseded, see note

Last updated: 2026-08-26

---

## Acceptance criteria — the definition of done

Four of these need no human judgement and no external subjects. They are the ones to run
constantly, not just at a review.

| | Criterion | Threshold | Self-checking | Status |
|---|---|---|---|---|
| AC-1 | Nodes and edges carry extraction or catalog provenance | ≥ 95% | no | · |
| AC-2 | Container diagram produced with the LLM disabled | renders + validates, both subjects | **yes** | · |
| AC-3 | Structural accuracy vs hand-drawn reference | ≥ 0.85 on Immich | no — needs the reference | · |
| AC-4 | Validator rejects malformed models | 100% of fault-injection suite | **yes** | · |
| AC-5 | Rule persistence | 10 rules survive regeneration | **yes** | · |
| AC-6 | Drift detection | 100% of synthetic drift set | **yes** | · |
| AC-7 | Determinism | 5 runs, byte-identical FactSets | **yes** | · |
| AC-8 | Egress | `structure-only` transmits zero file contents | **yes** | · |
| AC-9 | Performance | NFR-1 and NFR-2 met on Supabase | no | · |

> **AC-3 is predicted to miss its threshold.** Immich declares none of its three service
> connections in configuration — all live in TypeScript source. Reporting that as a measured
> limitation of declaration-based extraction is the intended outcome, not a defect to fix.
> See `docs/delivery-schedule.md` §7.

---

## Capabilities

| Group | What it covers | Count | Done | Gate |
|---|---|---|---|---|
| `DSC` | Discovery — which files are the architecture | 4 | 0 | — |
| `EXT` | Extraction — parsers, provenance, the FactSet | 15 | 0 | AC-7 |
| `MDL` | Model construction — nodes, edges, boundaries, identity | 17 | 0 | — |
| `VAL` | Validation | 8 | 0 | AC-4 |
| `RUL` | Rules — `rules.yaml` | 6 | 0 | AC-5 |
| `SEM` | Semantic layer — the LLM | 10 | 0 | — |
| `PRV` | Provenance | 6 | 0 | AC-1 |
| `MEM` | Memory and diff | 8 | 0 | AC-6 |
| `VIE` | Views and rendering | 10 | 0 | AC-2 |
| `SUR` | Surfaces — CLI and web app | 15 | 0 | AC-8 |
| `OUT` | Output and deliverables | 10 | 0 | — |
| `ANS` | Answer surface | 7 | 0 | — *(R1.c, stretch)* |
| | **Total** | **116** | **0** | |

109 are R1.a; the 7 `ANS` capabilities are R1.c.

### Superseded — do not implement as written

Two catalog entries are contradicted by later decisions. Recorded here so they are not
mistaken for pending work.

| ID | Catalog text | Superseded by |
|---|---|---|
| `EXT-05` | Parse Kubernetes manifests — Deployments, Services, Ingresses… | Orchestrator manifests are **out of R1 scope**. `docs/product-definition.md` §13, Settled |
| `VIE-03` | Compute layout (`dagre` / `elkjs`) | Layout is `goccy/go-graphviz`, computed in the engine and persisted. `docs/stack-decision.md` §3 |

---

## Sprint gates

From `docs/delivery-schedule.md` §4. A sprint is done when its gate passes, not when its
items are ticked.

| | Ends | Gate | Status |
|---|---|---|---|
| Week 0 | 28 Aug | O-8 answered; `go test ./...` green in CI | ✓ **gate met** — Go 1.27, module, CLI, CI, tests green; O-8 closed on evidence |
| Sprint 1 | 11 Sep | AC-7, AC-4, AC-5 | · |
| Sprint 2 | 25 Sep | AC-2, AC-6, AC-8 — **the thesis, demonstrated** | · |
| Sprint 3 | 9 Oct | All nine evaluated and written up | · |

---

## Open questions

| | Question | Blocks | Status |
|---|---|---|---|
| O-7 | Test subject #3 | — | ✓ **closed 26 Aug — Mastodon**, pinned `47ac677`. Chosen to cover `MDL-09`/`MDL-11`/`MDL-16`/`EXT-09`, which the other two leave untested |
| O-8 | Does `file:line` provenance survive the Compose merge? | — | ✓ **closed 26 Aug — no.** Extraction is two passes; a `Fact` carries one position. See `docs/decisions.md` |
| O-6 | R1.b sequencing — analysis before clustering? | R1.b only | deferred by design, decide with a working spine |

---

## Manual work — does not compress

Verification-bound items that AI assistance does not accelerate.

| | Needed for | Status |
|---|---|---|
| Hand-drawn Immich reference architecture | AC-3 — it is the answer key | · scheduled 31 Aug – 11 Sep |
