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

- [ ] Seven of Immich's explanations are stale and never refreshed: they are for one-file components
      `--explain` no longer asks about. Hide them, or stop marking them — never decided
- [ ] Immich's history carries versions nothing caused: 27 (a label run before memory was applied
      first) and one each from the element-order fix and the layout changes. Harmless; say so if asked
- [ ] `people` and `person` are two features in Immich's web app, because the code spells both.
      A rule in `rules.yaml` to merge two components would be the fix
- [ ] The main view picks its sixteen components by count. A person naming them in `rules.yaml`
      is the better answer (also in PROGRESS, Not done)
- [ ] Coverage lists only the migration files that create a table, not ones that only alter
- [ ] A formatter run with default settings rewrote two web files on 9 Oct; both were restored.
      The web code has no formatter configuration — it is formatted by hand, 160 columns, no semicolons
- [ ] `/tmp/ad` is a copy of the binary used for testing this session; on macOS overwrite it with
      `rm` then `cp`, never `cp` over a running one — the process is killed (exit 137)
- [ ] The small subjects have not been through `--label` or `--explain`: a few cents for all three
- [x] The Claude Design files compared with the app, 9 Oct: 34 pieces in `web/src/planned.ts`.
      `Surface Screens.dc.html` is larger than DesignSync's 256 KB read limit and came back cut
      short: every screen's markup was read in full; the end of its example data (from the Changes
      version list on — likely Corrections, Network runs, Dependencies) was not
- [ ] Where the app still differs from the design (Surface Screens), after the 9 Oct pass on the
      navigation and the Overview — to work on later:
      - Coverage on the Overview and in the navigation is the share of elements and relationships
        with no gap (Immich 68%). The design's is "outbound references resolved" (96 of 110), which
        archdoc does not measure yet: count calls that leave a component, and how many reach a
        known element
      - Component pages in the navigation: "Components" is a placeholder (`wiki-tree` in
        `web/src/planned.ts`). The design lists each component's page under it; Immich has 179, so
        it needs deciding which ones show — main ones, or the open container's
      - Navigation badges: Coverage as %, Documents as written of planned (7/12), Changes as "•N"
        since the compared version (`nav-badges` in planned.ts)
      - Run, Publish, Uncommitted, Settings and the status pill are dashed placeholders (§11
        controls; `run`, `publish`, `git`, `settings`, `freshness` in planned.ts)
      - The Overview summary has no inline citations: the design's carries [1], [2] on each claim
        and "Interpreted by … from N proven facts". Ours is one `--label` sentence with one cite
- [x] The Overview's "What changed" named a component twice when two containers had one of that
      name; it now says the container and counts parts under it — 9 Oct

---

## Recently resolved

Staging, not an archive. An item lands here when it is resolved, and is deleted at the
**next** checkpoint after that. Nothing stays under *Open* once it has a marker.

- [x] Dense component views — a component most others use is drawn as shared: its box says "used by
  12 of 13" and no arrow is drawn into it; the arrows stay in the model and the inspector — 2026-10-08

- [x] Immich re-cloned at its pin, and the FastAPI template added as a subject; Supabase still
      not cloned — needed only for AC-9 — 2026-10-05
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
