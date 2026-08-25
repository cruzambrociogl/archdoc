Review this entire conversation — from the beginning, or since the last `/checkpoint` — and
extract everything worth persisting. Route each item using the table below. **Announce every
file before writing to it.**

## What to look for

Decisions made and their reasoning · plans changed · capabilities completed · acceptance
gates that now pass or fail · technical notes and gotchas · facts about the test subjects ·
anything now false in an auto-loaded file.

## Routing

| Item | Destination |
|---|---|
| Decision — why X over Y | `docs/decisions.md`, dated entry, 3–6 lines |
| Decision **reversed** | Edit the original entry to point at the reversal. Never delete it — the reasoning is the record |
| Capability finished | `PROGRESS.md` — the group's Done count, and its status mark |
| Acceptance gate passes or fails | `PROGRESS.md` — the AC table. A failure is a *finding*, record the number |
| Sprint gate reached | `PROGRESS.md` — sprint gates table |
| New invariant — a rule that must hold in every session | `CLAUDE.md`, Hard rules. **Only if not derivable from the code** |
| Orientation fact — where something lives, what loads when | `CLAUDE.md`. Keep it a pointer, never an explanation |
| Technical note, gotcha, dialect discovered in a real repo | `docs/survey-test-subjects.md` if it is evidence about a subject; otherwise a `docs/` file |
| Scope or schedule change | `docs/delivery-schedule.md` — **and flag its published mirrors**, listed at the top of that file. They do not update themselves, and a stale schedule shown to a supervisor is worse than none |
| Product decision that changes what archdoc *is* | `docs/product-definition.md` — rare, and say so explicitly |
| Open question raised or closed | `PROGRESS.md`, open questions table |

## Drain `.claude/NOTES.md` — do this every time

`.claude/NOTES.md` is the capture inbox for anything unplanned. Every open item gets one of four
outcomes; nothing sits there indefinitely.

1. **Promote** — route it by the table above, mark the line `[→ path]`, and **move it to
   *Recently resolved***
2. **Close** — mark `[x]`, **move it**
3. **Drop** — mark `[-]` with the reason, **move it**
4. **Leave open** — only if it is still genuinely undecided. It stays under *Open* unmarked

**Nothing marked stays under *Open*.** A marked line there means the last drain was
incomplete. Then delete anything already sitting in *Recently resolved* from a previous
cycle — it is staging, not an archive.

**Route the decision too.** Promoting a note to `CLAUDE.md` or a `docs/` file records *what*
was decided; if there was a real choice between alternatives, `docs/decisions.md` still needs
the *why*. A note can produce two writes.

If a note contradicts `docs/product-definition.md`, **raise it — do not promote over it.**

## Pruning — do this every time

Routing alone makes auto-loaded files grow without bound, which defeats the token discipline
they exist to serve. So on every checkpoint, also ask:

1. **What in `CLAUDE.md` is now false?** A superseded rule, a package that moved, a load-table
   row pointing at a section that no longer exists. Remove or correct it.
2. **What in `CLAUDE.md` is now redundant?** Anything derivable from the code once the code
   exists. Delete it — the code is the better source.
3. **What in `PROGRESS.md` is stale?** Capabilities marked in-progress that were finished or
   abandoned.

**`CLAUDE.md` should not grow across a checkpoint unless something was also removed.** If it
did, say so and justify it.

## Rules

- Announce before writing, in the form `-> path/to/file — [what]`
- List every file you will touch before starting
- Do not duplicate — check whether it is already captured
- Prefer the repo over any external memory. If it belongs in `docs/`, it goes in `docs/`
- Terse. Decisions 3–6 lines; progress entries one line
- Skip clarifying questions, small talk, and conversational back-and-forth
- Respect document precedence: the product definition outranks the stack decision. If a
  session's conclusion contradicts the definition, flag it rather than quietly editing it

## Output

A short summary of every file written and every line pruned. Then
`(nothing else worth capturing)` if that is the case.
