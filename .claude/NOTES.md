# Notes

The capture inbox. Anything that does not have a home yet — an idea, a reminder, a question,
something noticed in passing, a thing to try later.

**Add without thinking about where it goes.** That is the whole point: a note that costs
effort to file does not get written down. Routing happens later.

`PROGRESS.md` tracks work that was *planned* — capability IDs and acceptance gates. This file
holds everything that was not.

---

## How it drains

`/checkpoint` sweeps this file and gives every open item one of four outcomes. Nothing is
meant to sit here indefinitely.

| Marker | Meaning |
|---|---|
| `[ ]` | Open — still an inbox item |
| `[→ path]` | **Promoted.** Its real home now holds it; the line stays one cycle, then goes |
| `[x]` | Done, and nothing further was needed |
| `[-]` | Dropped — keep the reason on the line |

Where things get promoted to:

| A note about… | Goes to |
|---|---|
| Why we chose X over Y | `docs/decisions.md` |
| A capability finished, a gate passed | `PROGRESS.md` |
| A rule that must hold in every session | `CLAUDE.md`, Hard rules |
| Something true about a test subject | `docs/survey-test-subjects.md` |
| Scope or timing | `docs/delivery-schedule.md` |
| What the product *is* | `docs/product-definition.md` — rare, flag it |

**Precedence still applies:** if a note contradicts `docs/product-definition.md`, it is
raised, not quietly promoted over it.

---

## Open

- [ ] **Immich and Supabase are not cloned on this machine** (deleted 23 Sep for disk space; only
      Mastodon remains). Supabase is needed for AC-9 (5–8 Oct), Immich for the hand-drawn reference
      (F-44) and the Code Wiki / DeepWiki comparison (F-43). Re-clone at the pinned revisions in
      `docs/survey-test-subjects.md` §Method

---

## Recently resolved

Staging, not an archive. An item lands here when it is resolved, and is deleted at the
**next** checkpoint after that. Nothing stays under *Open* once it has a marker.

- [x] The pre-arc42 `architecture.generated.md` left behind — generate now removes files only it
      writes that it no longer emits (F-52) — 2026-10-05
- [x] A schema change minted a version for every repository — the fingerprint is the architecture,
      recomputed from the stored model, so upgrades mint nothing (F-50) — 2026-10-05
- [x] A layout recomputed for an unchanged architecture was not stored — a same-architecture run now
      refreshes the version's evidence, layouts and commit (F-51) — 2026-10-05
- [x] Sample env files not reported — coverage now names the interpolation file and calls a sample a
      sample (F-53). Mastodon's `.env.production.sample` turned out never to be read — 2026-10-05
- [x] Web app clicked through in a browser — every screen, live and published, 4–5 Oct
- [x] `rules.yaml` location — now `.archdoc/rules.yaml`; the root one is read with a note — 2026-10-05
- [x] Stubs reading "may be stale" without `sections.json` — they read "unknown" now, with the reason.
      (Regenerating would not have fixed it: stubs are created once) — 2026-10-05
- [x] Published completeness from checkout mtimes — the export judges by size and claims no
      staleness — 2026-10-05
- [→ docs/decisions.md] The published site from `file://` — decided against; it is served — 2026-10-05
- [→ PROGRESS.md, surface §11] The settings flags the design assumes — they arrive with the §11
      controls, still to build — 2026-10-05
- [→ docs/feature-inventory.md F-56] Measuring labelling effort low/medium — 2026-10-05
- [→ docs/vision.md] The 13 Sep directions — folded into the vision (O-11) — 2026-10-05
