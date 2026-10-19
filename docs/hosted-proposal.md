# archdoc — Proposal: reachable by anyone

> **What this document is:** the direction archdoc takes after the current work is wrapped up —
> both easier to use locally and available as a hosted service anyone can reach from a browser,
> with a view to it becoming a business. It sets out the pieces, what each requires, what each
> breaks, and the questions still open.
>
> **Status — settled as a direction, 2026-10-19; not started.** What was agreed is in §8 and
> recorded in `decisions.md`. Work begins only after the current state is wrapped up, with a plan
> of its own. Until then `product-definition.md` still governs what is built, including §7, which
> rules a hosted service out; it is amended when the platform work starts.
>
> Raised 2026-10-17, from a suggestion by the project's professor that the team adopted.

---

## 1. The problem this answers

archdoc works, but only for someone who can:

- download a binary and put it on their `PATH`;
- clone the repository they want documented;
- remember which of eleven commands does what, and in which order (`generate`, then `serve`);
- know that `label` and `explain` need an API key, and where it goes.

That is a developer's tool for developers. The diagrams, routes and citations it produces are
useful to far more people than can get through that door: a new hire, a team lead, a student, an
evaluator, someone deciding whether to adopt an open-source project.

**The direction:** do both. Make the local tool something anyone can use without learning it, and
offer a hosted service that needs nothing at all. One engine, several ways to reach it.

---

## 2. The real tension: our positioning

The hard part is not technical. `vision.md` §2.2 positions archdoc as:

> **the local, verifiable alternative to hosted AI code wikis** … the code never has to leave the
> machine, and the documentation lives in the repository rather than on someone else's website.

A hosted archdoc steps onto Google Code Wiki's ground — free, hosted, every language. If "local" is
the identity, hosting contradicts it.

**Proposed reframe:** locality stops being the identity and becomes **a choice we offer and they
cannot**.

> **archdoc is verifiable architecture documentation — wherever you want it to run.** Every
> element is extracted from the code and cites the line that proves it, the model can never add
> something that does not exist, and you choose where it runs: on your machine, in your CI, on
> your own servers, or on ours.

The verification is the product; where it runs is a deployment option. That is a stronger
business position than "local only": a company that will never upload its code to Google can still
use archdoc, and a student with a public repository can use it in one click. Code Wiki cannot offer
the first. Today, archdoc cannot offer the second.

What has to hold for the reframe to be honest:

| Principle | Under the reframe |
|---|---|
| The diagram is never the source of truth — a validated model is | Untouched, every deployment |
| The LLM never produces facts | Untouched, every deployment |
| Every fact cites its file and line | Untouched — hosted citations can link straight to GitHub at the commit |
| Deterministic output (AC-7) | Untouched — and hosted, it is an asset: results cache by `repo@commit` |
| Only `internal/semantic` makes outbound calls (hard rule 3) | Untouched, **if** cloning, queues and accounts live in wrappers outside the engine |
| Never write to human-owned files (hard rule 2) | Untouched — hosted runs write into a throwaway clone; writing back to a repository is only ever a pull request |
| Egress: `structure-only` sends zero file contents (AC-8) | Holds for local, CI and self-hosted. **Does not hold** for our cloud, and the product must say so plainly |
| One process when serving, no background service | Holds for the local tool. The hosted service is a separate product built around it |

The two governing principles are safe everywhere. What changes is that "nothing leaves your
machine" becomes a property of three of the four deployments, not of the product.

---

## 3. The pieces

Not alternatives — layers. Each stands alone and each makes the next one stronger.

### A. The local tool, for anyone

- `archdoc` with no arguments opens the web app: pick a folder, press **Generate**.
- The app runs everything the CLI runs — generate, diff two versions, label, explain — with what
  each will cost shown before it runs.
- First-run guidance: what was found, what was not, what to do next.
- An installer per platform (Homebrew, winget, a download page) rather than a binary on `PATH`.
- Commands stay for those who want them.

**Requires:** the web app's action gate to cover more actions (surface-spec §11); a folder picker; packaging.
**Breaks:** nothing.

### B. In CI — the GitHub Action

- Push to `main` → `generate` + `export --site` → GitHub Pages.
- Pull request → `archdoc diff` posted as a comment ("cache renamed to store").
- `label` / `explain` with the repository owner's key, as a repository secret.

**Requires:** `action.yml`, release builds of the binary, a Pages step, a comment step.
**Breaks:** nothing — runs in the user's own CI. This is R2's "CI execution" and "docs delivery
beyond the machine".

### C. Hosted, public repositories — paste a link

One page, one input. Paste `github.com/owner/repo`; moments later, the published site for it.

- Shallow clone at a commit → `generate` → `export --site` → store → serve. The engine is unchanged.
- Cached by `owner/repo@commit`; every later visitor gets it instantly.
- No accounts. Deterministic product only, so nothing costs per visitor.
- Limits on repository size, run time and concurrent runs. archdoc parses and never executes the
  code it reads, which makes analysing strangers' repositories far safer than for most tools.
- A public gallery of well-known projects, already generated: the best marketing there is.

**Requires:** a wrapper service (e.g. `cmd/archdoc-web`), a job queue, storage, a landing page.
**Breaks:** local-only, for this path.

### D. Hosted, private repositories — the platform

- Sign in with GitHub; connect repositories through a GitHub App.
- Regenerate on push; architecture diff on every pull request; history over time.
- `label` / `explain` included, on our key with a quota, or the customer's own.
- Teams, permissions, a dashboard per organisation.
- Corrections (`rules.yaml`) and arrangements (`layout.yaml`) edited in the browser and proposed
  back as pull requests — never written directly.

**Requires:** a GitHub App, webhooks, accounts and tenants, isolation between them, billing, and
the obligations of holding private code — including `.env` files, which archdoc reads (see §5).
**Breaks:** "nothing leaves your machine", for this path, by design.

### E. Self-hosted — the platform on their servers

D, packaged to run inside a company's own infrastructure (a container image, a Helm chart). Their
code never leaves them, and they still get the team features.

**Requires:** D, built so that nothing in it assumes our infrastructure.
**Breaks:** nothing — this is where the local promise and the platform meet.

---

## 4. As a business

### Who would pay, and for what

| Who | Uses | Pays for |
|---|---|---|
| Individuals, students, open-source maintainers | A, B, C | Nothing — this is reach and reputation |
| Small teams | D | Private repositories, PR diffs, history, AI explanations |
| Companies that cannot upload code | A, B, E | Self-hosting, support, team features |
| Consultancies, auditors, acquirers doing due diligence | C, D | Understanding an unfamiliar codebase fast, with evidence |

The pattern is **open core**, as GitLab and Sentry run it: the engine free and local, convenience
hosted and paid, privacy self-hosted and paid.

### What we sell that others cannot

- **Verified, not generated.** Every claim cites its line; nothing is invented. Competitors' wikis
  are written by a model and checked by nobody.
- **The architecture diff.** "What changed in the architecture in this pull request" — a recurring
  reason to pay, not a one-off look.
- **Runs anywhere, including where code may not leave.** Self-hosted and offline are features a
  hosted-only competitor cannot match.
- **The customer owns the output.** Markdown, SVG and a static site committed in their repository,
  readable with archdoc uninstalled.

### Risks

- **Free competition.** Code Wiki and DeepWiki are free on public repositories. Winning on public
  repositories alone is unlikely; C is reach, not revenue.
- **AI cost.** `explain` costs money per component. Pricing has to absorb it or pass it through.
- **Trust.** Companies will not connect private repositories to an unknown service without
  security commitments (and eventually certifications). E is the answer for the ones who never will.
- **Coverage.** A hosted service meets every stack in the wild. What archdoc does not read, it must
  keep saying so (D-6), or it becomes the tool that silently missed half the system.

---

## 5. Questions to settle

**Identity**

1. ~~Do we adopt the reframe in §2?~~ **Agreed 2026-10-17:** "verifiable, wherever it runs" is
   how the team describes the product. Not yet recorded in `decisions.md`; when it is, it amends
   `product-definition.md` §7 and rewrites the positioning in `vision.md` §2.2 and `brief.md`.
2. ~~C or D first?~~ **Agreed 2026-10-17:** C first, built as the start of D — the queue, storage
   and workers D needs, with no accounts yet. Then D adds sign-in and private repositories on top
   of the same machinery. C is D without login, not a throwaway demo.

**Hosted product**

3. AI on the hosted path: none on C, quota on D, customer's key on E — or something else?
4. What does the hosted app do beyond the published site? Diff between commits, history, the
   explanations — each is a new hosted screen or a reuse of an existing one.
5. A repository too large or too slow to finish: refuse clearly, or document part of it and say
   which part? VAL-08's spirit favours refusing over a partial model.
6. `.env` files on the hosted path: read them as today, read only their keys, or skip them? Holding
   other people's secrets is the single largest liability in D.

**Business**

7. ~~Is the engine open source?~~ **Agreed 2026-10-17:** the engine is free, the platform is ours.
   The engine — CLI, local web app, Action (A, B) — is public under the Functional Source License
   (FSL, as Sentry uses): anyone may read, use, modify and self-host it, but not offer a competing
   service; each version becomes Apache 2.0 after two years. The platform — hosted service,
   accounts, billing (C, D, E) — is closed; E is sold as a licence to run it on the customer's
   servers. MIT/Apache would let anyone host the engine; AGPL is banned inside many companies,
   which would hurt E; closed would lose "verify the verifier".
8. Pricing model: per repository, per seat, per run, or usage-based AI?
9. ~~Ownership.~~ **Settled 2026-10-17:** the project belongs to the team, free to take wherever
   it wants.

**Architecture**

10. Where do the hosted pieces live? **Follows from 7:** a public engine and a closed platform
    means two repositories, the platform importing the engine as a library. It also enforces hard
    rule 3 structurally — the platform's network code cannot end up inside the engine.
11. Does the engine need a library API? **Yes — found 2026-10-17.** Go forbids importing `internal/`
    from another module, and the pipeline lives in `package main` (`document()` in
    `cmd/archdoc/generate.go`, ~370 lines). A small public package — `Generate`, `Diff`,
    `ExportSite` — with the CLI as one client and the platform another. Worth doing even if nothing
    is ever hosted: A and B call the same entry points.

---

## 6. A starting position, to argue against

Not a decision — a position to have the discussion against.

- **The reframe** (§2) — agreed. It keeps every principle that makes archdoc worth anything, and
  turns locality from a limit into a selling point.
- **A and B** first and alongside everything: they widen reach without changing what archdoc is.
- **C** next: it removes the barrier entirely, and determinism plus the existing published site do
  most of the work. A gallery of famous repositories doubles as the demo.
- **D** built so that **E** falls out of it: nothing in the platform assumes our infrastructure.
- **The licence** (question 7) — agreed: FSL engine, closed platform, two repositories.

---

## 7. Features — what we offer that they cannot

> **Suggestions, not a backlog.** Raised 2026-10-17 to discuss; nothing here is decided or
> scheduled. Numbered S1–S10 so they never collide with the inventory's F-01–F-57. Where one is
> already planned, the table below says so; anything adopted becomes an F- entry in
> `feature-inventory.md`. Builds on `comparison-matrix.md` (4 Oct) and the lenses of `vision.md` §2.3.

### Against what is already planned

| | Suggestion | In `feature-inventory.md` | What would be new |
|---|---|---|---|
| S1 | PR architecture review | F-37 diff between commits, F-38 change summary | Delivering it as a PR comment (piece B) |
| S2 | Rules that fail the build | Deferred to R2a (`archdoc check`) | Bringing it forward |
| S3 | Did it build what I asked? | **F-40 + F-41** — planned | Nothing |
| S4 | Time machine | F-37 is the engine of it | Replaying a whole history; changelog per release |
| S5 | Blast radius | — | All of it, as a lens |
| S6 | Organisation map | R2 "cross-repo visibility" | Bringing it forward, on the platform |
| S7 | Drift alerts, README badge | F-39 drift set | The badge and alerts |
| S8 | Egress and dependency inventory | F-14 dependency map; endpoints already extracted | Packaging it for security and compliance |
| S9 | Verified context for agents | **F-27 MCP, F-26 ask the map** — planned | Nothing |
| S10 | What did the agent change? | F-38 change summary ("per AI session", matrix G5) | Hooking it to an agent session |


### The test every feature has to pass

Code Wiki, DeepWiki, Archify and GitDiagram all **write** their documentation with a model. That
gives them breadth and polish, and costs them three things structurally:

1. **They cannot diff.** Regenerate an LLM wiki and the wording changes everywhere, whether or not
   the code did. Change cannot be told from noise.
2. **They cannot enforce.** You cannot fail a build on a description that might be invented.
3. **They cannot say what they missed.** A model does not know what it did not see.

archdoc has the opposite properties: deterministic, validated, cited, honest about coverage. "We
are verifiable" is a property; nobody buys a property. **The features worth building turn those
properties into jobs people already need done** — and are ones an LLM wiki cannot copy without
rebuilding itself on a validated model.

### Only possible on a validated model

| | Feature | The job it does | Why they cannot |
|---|---|---|---|
| S1 | **Architecture review on every pull request** — "adds an outbound call from `api` to Stripe; adds table `refunds`; removes `GET /v1/albums`; the upload flow now passes through `queue`" | A reviewer sees what a PR does to the *system*, not just to files | Needs a stable diff (1) |
| S2 | **Architecture rules that fail the build** — "nothing but `api` reaches the database", "the web app never imports server code", "every route is behind auth" | Keeps a design intact as AI writes more of the code | Needs enforcement (2); R2a's `archdoc check` |
| S3 | **Did it build what I asked?** — a plan, spec or agent's task compared against what now exists | The original problem: the person who directed the AI checks the result without reading it | Needs facts to compare against (D-9) |
| S4 | **Time machine** — replay a system's architecture across its history; an architecture changelog per release | Understand how a system got here; onboard by watching it grow | Needs a diff that means something (1) |
| S5 | **Blast radius** — "this table is read by 4 components, written by 2 jobs, reached from 11 routes" | Know what breaks before changing something | Needs a graph of facts, not prose |

### Only possible as a platform

| | Feature | The job it does |
|---|---|---|
| S6 | **The organisation map** — every repository of an organisation merged into one picture; services matched across repositories | Companies with many services have no current map anywhere. Per-repository wikis cannot draw it (R2 "cross-repo visibility") |
| S7 | **Drift alerts and a README badge** — "architecture docs: current", or what changed since last read | A growth loop (badges spread like Codecov's) and a reason to come back |
| S8 | **Egress and dependency inventory** — every external service, database and third-party API the system reaches, each cited | Security reviews, compliance vendor lists, due diligence — work people pay for today, done by hand |

### For the AI that writes the code

| | Feature | The job it does |
|---|---|---|
| S9 | **Verified context for coding agents** — an MCP server over the model: "who writes this table?", "what does this route set off?" — answers with citations, or "not resolved" | Agents waste tokens rediscovering structure and guess when they cannot find it. DeepWiki offers an MCP; its answers are not checked (D-17) |
| S10 | **What did the agent change?** — after an agent session, the architectural summary of what it did | The review step that never happens when AI writes the code |

### Where we must reach parity, or lose anyway

Being better on verification does not matter to someone who leaves in the first minute:

- **Coverage** — on the hosted gallery people will paste repositories in every language. Until a
  stack is read, the coverage report must say so plainly, and a generic baseline (files, imports,
  entry points) must still give something useful.
- **Ask the map** — chat is now expected. Ours answers from the model and can say "not resolved".
- **Visual polish** — Archify and GitDiagram look better today. The first screen sells the product.
- **Export where teams already work** — Backstage `catalog-info.yaml`, Structurizr DSL, Confluence,
  Notion. Generated and kept current, instead of maintained by hand.

### A starting position

~~Lead with S1 + S2 as the paid team product.~~ Superseded by the first user chosen in §8, a solo
developer: lead with **S3 + S10 + S9** — did it build what I asked, what did the agent change, and
verified context for the agent — the original problem answered at the moment the code is written.
**S4**, **S5** and the hosted gallery are what make that person share it. **S1 + S2**, the team
features, follow when teams do.

---

## 8. Settled — 2026-10-19

The discussion of 17–19 October ended with these agreed. Everything else in this document stays
open until the platform plan takes it up.

1. **Order.** The code is frozen as it stands — no new features — and that is the wrap-up. The
   freeze lasts until the development environment for the platform is set up; feature work resumes
   inside it. The path is planned separately.
2. **Direction.** Both: the local tool made usable by anyone (A, B), and a hosted service (C, then
   D, with E falling out of D). A business is an explicit aim.
3. **Positioning.** "Verifiable, wherever it runs" (§2). Locality is an option offered, not the
   identity.
4. **Licence.** The engine is free under the FSL; the platform is closed and ours. Two repositories,
   the engine exposing a public library API (§5, questions 7, 10, 11).
5. **Hosted order.** C first, built as the start of D (question 2).
6. **First user: a solo developer one level above vibe coding.** Someone with some or good
   technical skills who did not write the code — they directed an AI that did. They can read a
   diagram, a route or a table name; they never formed the mental model, because they never typed
   the code. Not a team, and not someone with no technical background.
7. **Readable before complete.** What archdoc shows must be understandable and easy to read. Today
   it gives the user so much that it overwhelms and is hard to read. For the first user, every
   screen and document answers one question in plain words first, and the full evidence sits one
   step behind it — never removed (citations stay), but never the first thing they meet. This holds
   for every piece, local and hosted, and for every S- feature adopted.

Still open: pricing (question 8); `.env` and the "we never keep your code" promise (question 6);
AI on the hosted path (question 3); what the hosted app shows beyond the published site (question
4); oversized repositories (question 5); and which S- features are adopted.
