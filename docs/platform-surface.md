# archdoc — the platform surface: flows and screens

> **What this document is:** the shape of archdoc as a platform — how people arrive, what they see
> first, what brings them back, and the screens around the repository viewer. Written to be handed
> to **Claude Design** as the brief, so the shape can be seen and its flaws found before anything is
> built; then brought back here and corrected.
>
> **What it is not:** a replacement for `surface-spec.md`. That document still specifies the
> repository viewer — the explorer, inspector, component pages, flows, data model and the rest —
> and every screen in it survives. This one adds what surrounds the viewer and reorders the viewer's
> navigation around the user's questions. Where the two disagree, §1 says which wins.
>
> Drafted 2026-10-19. **Nothing here is built**: the code is frozen until the platform's development
> environment exists (`platform-plan.md`); designing is not building, and is the right work now.
> Direction: `hosted-proposal.md` §8.

### Where to look

| If you want… | Read |
|---|---|
| The platform in one page, for someone who has never seen it | §0 |
| What changes from `surface-spec.md`, rule by rule | §1 |
| Who it is for, and the moments they come | §2 |
| The journeys, step by step | §3 |
| Navigation, and the three places it runs | §4 |
| The new platform screens | §5 |
| The repository screens, rebuilt around questions | §6 |
| The new shared components | §7 |
| The data each new screen needs | §8 |
| What to hand Claude Design, and in what order | §9 |
| What is still open | §10 |

### Pieces

Screens are marked with the piece of `hosted-proposal.md` §3 that first needs them:

| Mark | Piece |
|---|---|
| **L** | The local tool — `archdoc` on the user's machine |
| **C** | Hosted, public repositories — no account |
| **D** | Hosted, private repositories — signed in with GitHub |

---

## 0. The platform in one page — for the designer

archdoc reads a code repository and produces the architecture documentation it should have had —
every box, arrow and sentence extracted from the code and citing the file and line that proves it
(`surface-spec.md` §0 has the full introduction, including the three truth states: **proven**,
**interpreted**, **unresolved**).

Until now archdoc was a local tool: a binary, eleven commands, a web app on `localhost`. It is
becoming a **platform**, reachable three ways:

- **Locally** — `archdoc` alone opens the app; no commands to learn.
- **Hosted, public** — paste a GitHub link on a web page; no install, no account.
- **Hosted, private** — sign in with GitHub, connect repositories, and they stay documented on every
  push.

The positioning is **"verifiable, wherever it runs."** The guarantee is the product; where it runs is
the user's choice.

**The first user** is a solo developer one level above vibe coding: real technical skill, but they
did not write this code — they directed an AI that did. They can read a diagram, a route, a table
name. What they never formed is the mental model, because they never typed the code.

**The one design principle that is new: readable before complete.** archdoc today shows a great deal
at once, and it overwhelms and is hard to read. On every screen of the platform, a plain sentence
comes first, then one picture, then the evidence — never removed, never first.

---

## 1. What changes from `surface-spec.md`

The rules of `surface-spec.md` §2, one by one:

| Rule there | Here | Why |
|---|---|---|
| 1. Every visible fact has a way back to its evidence | **Kept** | The guarantee |
| 2. Truth state always encoded, never only by colour | **Kept** | |
| 3. Views are earned by evidence | **Kept** | |
| 4. Same answer every time | **Kept** | |
| 5. Readable at Immich scale | **Kept** | |
| 6. Works offline and published; no account | **Kept for the local app and the published site.** The hosted platform has accounts and needs the network | The platform is a service |
| 7. **Technical first, plain language one click away** | **Reversed: plain first, the evidence one step behind** (§7.1). Citations still never removed | The first user, and "readable before complete" (`hosted-proposal.md` §8) |
| 8. The surface never writes to the repository | **Kept, extended:** on the platform, a correction becomes a **pull request** the user merges; archdoc still never writes directly | Hard rule 2 |

**New rules:**

9. **One question per screen.** Every screen states, in its first line, the question it answers.
   If a screen answers two, it is two screens or one is behind "more".
10. **No screen opens on a number.** Counts support a sentence; they are never the first thing read.
11. **The wait is part of the product.** Anything that takes time says what it is finding while it
    works (§7.3).

Rule 7's reversal applies to the **local app too**: it shares the viewer (`platform-plan.md` step 2),
and the first user meets it there as often as on the platform.

---

## 2. The first user and their moments

They do not come wanting "the data model". They come at a moment, with one question:

| Moment | Their question | Where it is answered |
|---|---|---|
| First look | "What did I actually build?" | Overview (§6.1) |
| After a push or an agent session | "What just changed?" | What changed (§6.2) |
| Checking the work | "Did it build what I asked?" | Did it build what I asked? (§6.3) |
| Before changing something | "What does this touch?" | Impact (§6.4) |
| A question of their own | "Where does X happen?" | Ask (§6.5) |
| Showing someone | "Can I send them a link?" | Share (§5.6) |

**The navigation is these questions.** The screens of `surface-spec.md`, organised by kind of data,
sit one level down under **Explore**, for when someone wants to dig.

---

## 3. The flows

### Flow 1 — First look, no account · C

1. **Landing** (§5.1) — paste `github.com/owner/repo`.
2. **Generating** (§5.2) — the narrative of what is being found.
3. **Overview** (§6.1) of that repository.

The goal: understanding before trusting. No sign-up stands between the link and the result. A second
visitor to the same repository at the same commit gets the result instantly.

### Flow 2 — Connect and keep current · D

1. Landing or Overview → **Sign in with GitHub**.
2. **Connect a repository** (§5.4) — install the GitHub App, pick repositories.
3. **Generating** for each.
4. **Home** (§5.3).

From then on, every push to the tracked branch regenerates on its own.

### Flow 3 — After a push or an agent session · D, L

The hook that brings people back.

1. A message: "Your architecture changed" — by **email**, as a **pull-request comment**, or **inside
   the coding agent** through the MCP server (§7.6).
2. **What changed** (§6.2), opened on that push or session.
3. Any item → its evidence.

### Flow 4 — Check the work · D, L

1. **Repository settings** (§5.5) → add a plan: a Markdown file in the repository, an issue, or the
   agent's task.
2. **Did it build what I asked?** (§6.3).

### Flow 5 — Before changing something · C, D, L

1. **Search** or **Ask**: "what does `users` touch?"
2. **Impact** (§6.4) for that element.

### Flow 6 — Share · C, D

1. **Share** (§5.6) from any repository screen.
2. A public link to the view, an embed, or a README badge.

### Flow 7 — Local · L

1. `archdoc` with no arguments opens **Start** (§5.8).
2. Open a folder → **Generate** → the narrative of §5.2, locally.
3. The same repository screens as the platform.

Later: "publish to archdoc" sends the result to the platform. Not designed in detail now; leave the
slot.

---

## 4. Navigation

### 4.1 Two levels

```
Platform level  (D; C has only Landing and Generating)
  Home — your repositories, what changed
  Connect a repository
  Connect your agent
  Account

Repository level  (C, D, L — the viewer)
  Overview
  What changed
  Did it build what I asked?
  Ask                       docked on every screen
  Explore ▸  Diagram · Features · Flows · Data · Dependencies ·
             Deployment · Documents · Coverage · Corrections
  Settings · Share          (D; Share also C)
```

`Explore` holds `surface-spec.md` §5.2–§5.11, §5.13 and §5.17, unchanged in content. Network runs
(§5.14) stays in the local app only; on the platform its equivalent is AI usage in Settings.

### 4.2 Three places, one viewer

| | Local app (L) | Hosted (C, D) | Published site |
|---|---|---|---|
| Start | Start (§5.8) | Landing / Home | The repository's Overview |
| Account | None | D only | None |
| What changed — "since you last looked" | Since the last generate | D: since this user last opened it. C: since the previous commit | Since the publisher's baseline |
| Corrections | Composer copies YAML | Composer opens a pull request (D) | Read-only |
| Network runs | Yes | AI usage in Settings | Summarised |
| Ask | Local model or user's key | D only | No |

---

## 5. Platform screens

### 5.1 Landing · C

**Answers:** what is this, and can I try it right now?

- One sentence of what archdoc is, and one input: **paste a GitHub link**.
- A **gallery** of well-known repositories, already generated — Immich, Supabase, Mastodon, others.
  Each card: name, one-line summary, the diagram's thumbnail.
- One line for those who prefer local: how to install.
- Sign in with GitHub, for private repositories.

**States:** link not a GitHub repository; repository private (→ sign in); already generated at this
commit (→ straight to Overview).
**Tone:** the one screen allowed to sell. Still calm; the gallery does the selling.

### 5.2 Generating · C, D, L

**Answers:** is it working, and what is it finding?

- A live narrative from the engine's progress (§7.3): "Found 3 applications · reading TypeScript ·
  38 routes so far · 12 tables".
- What it is reading now, and roughly how far along.
- D: "You can leave — we'll tell you when it's done."

**States, each designed as a screen, not an error:**
- **Too big** — the repository exceeds the limit; what the limit is.
- **Stack not read yet** — what *was* read, what was not, and the honest partial result (§7.4).
- **Failed** — what happened, in plain words, and what to try.
- **Private** — sign in to continue.

### 5.3 Home · D

**Answers:** did anything change since I last looked?

- One **card per repository** (§7.5): name, a single status line — *current*, *3 changes since you
  last looked*, *2 things couldn't be resolved*, *generating…* — and when it last changed.
- A **feed** of recent changes across repositories, each a sentence linking to What changed.
- **Connect a repository.**

**States:** no repositories yet (→ connect, or try a public one); one repository (no feed, open it
directly?) — §10.

### 5.4 Connect a repository · D

**Answers:** what will you read, and what do you keep?

- The GitHub App installation and the repository picker.
- In plain words: what archdoc reads, what it stores (the documentation, which contains no code and
  no secrets), and what it does not keep — **subject to the promise in `hosted-proposal.md`
  question 6**, still open.
- Which branch to track.

### 5.5 Repository settings · D

**Answers:** how should this repository behave?

- Branch tracked; visibility (private, link-only, public).
- **Plans** for Flow 4: which files, issues or tasks are the intent.
- **Notifications**: email, pull-request comments, the agent.
- **AI**: on or off; a monthly spending cap; usage so far.
- **Corrections**: the rules in force; new ones open a pull request.

### 5.6 Share · C, D

**Answers:** how do I show someone?

- A link to this view (public repositories always; private ones when the owner allows).
- An embed for a diagram.
- A **README badge** — "architecture: current" — with the snippet to paste.

### 5.7 Connect your agent · D, L

**Answers:** how does my coding agent use this?

- Copy-paste configuration for Claude Code, Cursor and other MCP clients, with a token.
- What the agent can ask, in examples: "who writes this table?", "what does this route set off?".
- What the agent will be told after a session (Flow 3).

The first user's agent may use archdoc more often than they do. This screen makes the agent a
first-class surface rather than an afterthought.

### 5.8 Start · L

**Answers:** what do you want to look at?

- **Open a folder**; recent projects, each with the status line of §7.5.
- For a folder never generated: one button, **Generate**, and what it will do — offline, nothing
  sent.

### 5.9 Account · D

Profile, connected GitHub, agent tokens. Later: usage and billing. No design effort beyond the slot.

---

## 6. Repository screens, rebuilt around questions

### 6.1 Overview — replaces `surface-spec.md` §5.1

**Answers:** what is this system?

In order, nothing else above the fold:

1. **The system in one paragraph**, plain words: "A photo app. A web front end and a mobile app talk
   to one server, which keeps its data in Postgres and hands heavy work to a machine-learning
   service." Each clause cited (§7.1).
2. **One diagram** — the most useful level, clickable into Explore.
3. **What it does** — the main features as sentences, each linking to its flow.
4. **Since you last looked** — at most three lines, linking to What changed.
5. **Trust in one sentence**: "Everything here is proven from your code; 2 things couldn't be
   resolved." Links to Coverage.

The stat strip and trust strip of today's Overview move to Explore.

**States:** first run (no "since you last looked"); nothing could be read (§7.4); no AI used — the
paragraph comes from templates (§10, question 1).

### 6.2 What changed — reworks `surface-spec.md` §5.12

**Answers:** what just changed in the architecture?

- Opens on the **latest push or session, told as a story**: "This push added refunds — a new route
  `POST /refunds`, a `refunds` table, and a call to Stripe."
- Grouped by **what it means**, not by kind of edit:
  - new capability (routes, pages, jobs);
  - new outside service;
  - data changes (tables, columns, keys);
  - removed;
  - wording only (the model relabelled something) — kept apart, as today.
- The diagram in delta mode, below the story.
- **Advanced**: the timeline, and comparing any two versions or commits — today's screen, intact.

**States:** nothing changed architecturally ("this push changed code but not the architecture" — a
useful answer, designed); first version.

### 6.3 Did it build what I asked? — promotes `surface-spec.md` §5.15

**Answers:** did the code do what I asked for?

- The plan, requirement by requirement, in three groups:
  - **found** — where in the code, cited;
  - **missing**;
  - **built but not asked for**.
- A one-sentence verdict first: "7 of 9 found; 2 missing; 3 things built that the plan didn't ask
  for."

**States:** no plan yet (→ settings, with an example of a plan that works); a requirement too vague
to check ("couldn't tell — here's why").
Top-level because it is the most direct answer to the problem archdoc exists for.

### 6.4 Impact — new

**Answers:** what does this touch?

A panel, opened from any element — in a diagram, a search result, an Ask answer:

- The sentence first: "Changing `users` affects login, sign-up and 4 background jobs."
- Who **reads** it and who **writes** it.
- Which **routes** and **flows** pass through it.
- Which **outside services** it reaches.
- Each item cited; each opens.

### 6.5 Ask — promotes `surface-spec.md` §5.16

Docked on every repository screen. Answers drawn only from the validated model, each sentence
cited; "not resolved — here is what archdoc saw" is a designed answer. Same content as there; now
always present.

### 6.6 Explore

`surface-spec.md` §5.2–§5.11, §5.13, §5.17 — the explorer, inspector, component pages, features,
flows, data model, dependencies, deployment, documents, coverage, corrections and search. Unchanged
in content. Each gains the three-layer opening of §7.1 where it does not already start with a
sentence.

---

## 7. New shared components

Numbered on from `surface-spec.md` §6, so the design system stays one list.

| # | Component | Notes |
|---|---|---|
| 6.15 | **Three-layer block** — sentence, picture, evidence | The shape of every screen's opening. The sentence is plain; the picture is one diagram or a short list; the evidence (citations, the full table) is one step behind. The single most important new component |
| 6.16 | **Status line** | One sentence of state for a repository: current, changed, unresolved, generating, failed. Used on cards, Start, the shell |
| 6.17 | **Progress narrative** | The live "what I'm finding" of §5.2. Lines appear as the engine reports them; a calm sense of progress without a fake percentage |
| 6.18 | **Partial-result state** | When archdoc read only part of a repository: what was read, what was not, what is still shown. Extends `surface-spec.md` 6.8 |
| 6.19 | **Repository card** | Name, status line, last change, thumbnail. Home, Start, the gallery |
| 6.20 | **Change story** | A push or session as a sentence, its grouped changes beneath. What changed, the Home feed, the notifications |
| 6.21 | **Notifications** | The same change story in three places: an email, a pull-request comment, a message the agent receives. Designed together so they say the same thing |

---

## 8. Data — what the new screens need

All of it is platform API, except where marked L (the local app's `serve`).

| Data | Needed for | New work |
|---|---|---|
| Run status and progress stream | Generating | The engine's progress callback (`platform-plan.md` step 1), relayed live |
| Repositories with status line | Home, Start (L) | Per repository: last run, last change, unresolved count |
| Last seen, per user and repository | "Since you last looked" | D only; C falls back to the previous commit |
| Change story | What changed, Home feed, notifications | Grouping of diff results by meaning (§6.2) — engine or platform, §10 |
| Plain-language summary | Overview | Templates over the model, `explain`, or both — §10, question 1 |
| Impact for an element | Impact | A query over the model: readers, writers, routes, flows, outside services |
| Intent results | Did it build what I asked? | F-40, F-41 |
| Gallery | Landing | A curated list of repositories, generated ahead |
| Settings, sharing, tokens | Settings, Share, Connect your agent | Platform tables |

---

## 9. Handing this to Claude Design

**Give it:** this document §0–§7, and `surface-spec.md` §0, §2 and §6 for the design system already
established. The existing designs in the Claude Design project are the "before" of the repository
screens.

**Ask for, in this order** — the first two set what everything else reuses:

1. **The three-layer block** (6.15) and the **status line** (6.16) — on the reworked Overview.
2. **Overview** (§6.1) on Immich — the screen that decides whether "readable before complete" works.
3. **Flow 1 end to end**: Landing (§5.1) → Generating (§5.2) → Overview. Include the "stack not read
   yet" state.
4. **What changed** (§6.2) with a change story — Immich, one push that adds a feature.
5. **Did it build what I asked?** (§6.3) — a small plan, some found, some missing, some unasked.
6. **Impact** (§6.4), opened from the data model.
7. **Home** (§5.3) with three repositories in different states, and the **notifications** (6.21).
8. **Connect a repository**, **Settings**, **Share**, **Connect your agent**, **Start**.

**Use realistic content.** Immich as in `surface-spec.md` §8; Supabase for the large case; Mastodon
for the small case. For What changed, invent one plausible push — a refunds feature, say — and keep
every name in it consistent across screens.

**Tone:** the repository screens stay a precise instrument, but they now *open* in plain words — a
good explainer first, a good code editor one step behind. The Landing is the one place allowed to
sell, and the gallery does it.

**What we want from the design:** to see the shape and find its flaws. A screen that does not work
once drawn is a finding, not a failure.

---

## 10. Open

1. **Where the Overview's plain language comes from** — deterministic templates, `explain`, or
   both. C has no AI, so templates must carry it there. Decides what the first screen can say.
2. **Change grouping** — is "what it means" (§6.2) computed by the engine, so local and hosted
   agree, or by the platform?
3. **Home with one repository** — show Home, or go straight to the repository?
4. **Notifications** — which channel is the default for the first user: email, the agent, or
   pull-request comments?
5. **Corrections as pull requests** — welcome to a solo developer, or friction? Is a direct "apply"
   ever acceptable on the platform?
6. **"Publish to archdoc" from local** — the bridge between L and D; slot only for now.
7. **The never-keep-your-code promise** (`hosted-proposal.md` question 6) — Connect a repository
   (§5.4) words it only once it is decided.
8. **The gallery** — which repositories, and who keeps it current.
