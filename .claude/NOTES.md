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

- [ ] `web/dist/index.html` is a tracked placeholder; confirm the real build overwrites it
      cleanly rather than colliding with the `.gitignore` exception
- [ ] Clone Mastodon into `subjects/` at the pinned revision and run `scan` against it — the
      third subject is chosen but has never actually been read by the tool
- [ ] Fixtures: the O-8 spike used the real subject repos on disk. Decide what goes in
      `testdata/` — trimmed copies of the hard compose files, kept small and pinned

---

## Recently resolved

Staging, not an archive. An item lands here when it is resolved, and is deleted at the
**next** checkpoint after that. Nothing stays under *Open* once it has a marker.

- [→ CLAUDE.md] Branch protection — convention rather than a GitHub ruleset; no enforcement
      exists, so the discipline lives in the context file — 2026-08-25
- [→ docs/delivery-schedule.md] The schedule's published mirrors — Sheet and artifact URLs
      recorded so a future edit does not silently leave them stale — 2026-08-25
- [x] Rewrite `README.md` — opens with §0's problem statement, leads with the --explain
      output rather than a description of it — 2026-08-27
- [→ docs/decisions.md] `compose-go` profile resolution — answered by the O-8 spike:
      `!reset []` empties `profiles` correctly, and `!override` replaces lists rather than
      appending — 2026-08-26
