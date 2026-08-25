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
- [ ] Third test subject (O-7) — look for a repo with real declared infrastructure that is
      neither a photo app nor a BaaS. Something in a different shape would test discovery harder
- [ ] Worth checking whether `compose-go` exposes profile resolution, since Immich's dev
      stack leans on `profiles:` and `!reset` — **relevant to the O-8 spike**
- [ ] Rewrite `README.md` once there is something to show. Open with §0's problem statement —
      AI writes the code, the understanding never forms — not with installation instructions
- [→ CLAUDE.md] Branch protection — kept as convention rather than a GitHub ruleset. No
      enforcement exists, so the discipline lives in the context file — 2026-08-25

---

## Recently resolved

*(cleared each checkpoint after one cycle — promoted items do not accumulate here)*

- [x] First commit split — two commits, docs then scaffolding. `go.mod` deferred to Wednesday
      so the module path lands after the naming decision — 2026-08-25
