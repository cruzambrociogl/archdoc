# archdoc — the platform surface: flows and screens

> **What this document is:** the shape of archdoc as a platform — how people arrive, what they see
> first, what brings them back, and the screens around the repository viewer. Written to be handed
> to **Claude Design** as the brief, so the shape can be seen and its flaws found before anything is
> built; then brought back here and corrected.
>
> **What it is not:** a replacement for `surface-spec.md`. That document still specifies the content
> of the repository screens — the explorer, inspector, component pages, flows, data model and the
> rest — and every screen in it survives. This one adds what surrounds them and, since revision 3,
> **decides the information architecture of the whole app** (§4): where each screen lives, what it
> is called, and where it appears. §4 supersedes `surface-spec.md` §4.1. Where the two disagree
> otherwise, §1 says which wins.
>
> Drafted 2026-10-19. **Revision 2, 2026-10-19**, after the first design round: the Claude Design
> files `Platform Components`, `Platform Repository`, `Platform Flows`, `Platform Notifications` and
> its `Platform Findings`. What changed is listed in §11. **Revision 3, 2026-10-19**: one app for every
place, a density budget, plain names, and related repositories — the information architecture of §4;
what changed is listed in §12. **Revision 4, 2026-10-20**, after round 3 (`App.dc.html`, `App Findings`):
corrections, not a redesign — listed in §13. **Revision 5, 2026-10-20**: the Landing leaves the app
for a static website at the root domain — §14. **Nothing here is built**: the code is
> frozen until the platform's development environment exists (`platform-plan.md`); designing is not
> building, and is the right work now. Direction: `hosted-proposal.md` §8.

### Where to look

| If you want… | Read |
|---|---|
| The platform in one page, for someone who has never seen it | §0 |
| What changes from `surface-spec.md`, rule by rule | §1 |
| Who it is for, and the moments they come | §2 |
| The journeys, step by step | §3 |
| The information architecture — one app, where everything lives, related repositories | §4 |
| The new platform screens | §5 |
| The repository screens, rebuilt around questions | §6 |
| The new shared components | §7 |
| The data each new screen needs | §8 |
| What to hand Claude Design, and in what order | §9 |
| What is still open | §10 |
| What the first design round changed | §11 |
| What revision 3 changed | §12 |
| What revision 4 changed | §13 |
| What revision 5 changed — the website | §14 |

### Pieces

Screens are marked with the piece of `hosted-proposal.md` §3 that first needs them:

| Mark | Piece |
|---|---|
| **L** | The local tool — `archdoc` on the user's machine |
| **C** | Hosted, public repositories — no account |
| **D** | Hosted, private repositories — signed in with GitHub |

These marks are the **business pieces**, not the places the app runs. The places are **three**:
**Local**, the **Website** — either **signed in** or **signed out** — and the **Published site**
(§4.5). Whether a visitor may *change* a repository is a separate permission, owner or viewer.

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

**It is one app.** Local, hosted and published, it is the same application with the same pages and
names; each place adds or lacks a few features (§4). And it reaches beyond one repository: several
repositories that belong together form a **system**, with a map of how they fit (§4.6).

**The one design principle that is new: readable before complete.** archdoc today shows a great deal
at once, and it overwhelms and is hard to read. On every screen of the platform, a plain sentence
comes first, then one picture, then the evidence — never removed. On screens that explain, the
evidence starts one step behind; on screens that review, it stays in view (rule 12).

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
| 7. **Technical first, plain language one click away** | **Reversed: plain first, the evidence one step behind** (6.15) — on screens that explain; on screens that review, the evidence stays in view (rule 12). Citations still never removed | The first user, and "readable before complete" (`hosted-proposal.md` §8) |
| 8. The surface never writes to the repository | **Kept, extended:** on the platform, a correction becomes a **pull request** the user merges; archdoc still never writes directly | Hard rule 2 |

**New rules:**

9. **One question per screen.** Every screen states, in its first line, the question it answers.
   If a screen answers two, it is two screens or one is behind "more".
10. **No screen opens on a bare number.** A screen opens on a sentence, never on a figure, chart or
    tile. Numbers may appear inside sentences, where the sentence says what they count — "3 pushes
    changed the architecture" passes; a "122 components" tile as the first thing read does not.
11. **The wait is part of the product.** Anything that takes time says what it is finding while it
    works (6.17).
12. **Evidence closed where the screen explains, open where it reviews.** Overview, the Map and Reference
    explain: their evidence starts one step behind. What changed, Impact and Did it
    build what I asked? review: checking is the job, so their evidence stays in view, muted.
13. **AI-written text is marked by the block, not the line.** A block written by the model says so
    once — "written by AI from these facts" with the interpreted mark. Line-level marking is kept
    only inside blocks that mix proven and interpreted text. Otherwise, once screens open in plain
    words, nearly every line carries the mark and it stops meaning anything.
14. **A judgment is never presented as a measurement.** Where the model's reading decides a result —
    matching a plan to code — the sentence says it is a reading, and nothing shows it as a score:
    no percentage, no progress bar, no tick.
15. **The density budget.** The first view of a screen holds one question, one sentence, one picture
    and at most one primary action. Lists stop at five, with "see all". More is a deliberate step
    away — another page or a panel the user opens — never an expanded section of the same page. No
    menu holds more than seven entries at any level (§4.1). **Two exceptions:** the Overview's trust
    sentence is its one allowed second sentence, because it states the guarantee; and the limit of
    five is for lists, not sequences — a flow shows every step, grouped into at most five stages
    when it is longer.
16. **Plain names first.** Menus, levels and headings are named in plain words; the technical term
    follows, smaller, where it helps (§4.1).
17. **Absent, not disabled.** A feature that does not apply in a place — local, hosted, published —
    is not there, rather than greyed out (§4.5).
18. **Meaning first, the count second.** A sentence leads with what something is or does, and the
    number follows: "Everything is kept in one Postgres database; most of its 68 tables hang off
    two", not "68 tables in one Postgres database". Lists and tables say what each row is before
    how many; an inspector opens on a sentence, not on a table of figures.
19. **Nothing costs money or sends data without saying so** (`surface-spec.md` §11.1, line 3). The
    one-click action is always the free, offline one. Updating with AI explanations is a separate,
    deliberate step that shows what will be sent and roughly what it costs; Ask says once that it
    uses the model and roughly what a question costs.
20. **Dates and pushes, not version numbers.** Versions are archdoc's bookkeeping. Screens say "since
    12 Oct, when you last looked", "pinned to the 18 Oct push"; version numbers appear only under
    Advanced, where versions are compared.
21. **Copy says what the user can do and what something means — never why we designed it so.**
    No sentence on screen explains a rule of this document ("named in a fixed order… never
    chosen").
22. **One vocabulary** (§4.7), the same in the legend, the marks, the sentences and the notifications.

Rule 7's reversal applies to the **local app too**: it is the same app (§4.1), and the first user
meets it there as often as on the platform.

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
are redistributed under **Map**, **What it does** and **Reference** (§4.4), for when someone wants to
dig.

---

## 3. The flows

### Flow 1 — First look, no account · C

1. **The website's home** (§5.1) — paste `github.com/owner/repo`; the app's **Start** takes it.
2. **Generating** (§5.2) — the narrative of what is being found.
3. **Overview** (§6.1) of that repository.

The goal: understanding before trusting. No sign-up stands between the link and the result. A second
visitor to the same repository at the same commit gets the result instantly.

### Flow 2 — Connect and keep current · D

1. The website or Overview → **Sign in with GitHub**.
2. **Connect a repository** (§5.4) — install the GitHub App, pick repositories.
3. **Generating** for each.
4. **Home** (§5.3) — from the second repository on. With one repository, sign-in lands on that
   repository's Overview; while it is still generating, on Generating.

From then on, every push to the tracked branch regenerates on its own.

### Flow 3 — After a push or an agent session · D, L

The hook that brings people back.

1. A message: "Your architecture changed" — by **email**, as a **pull-request comment**, or **inside
   the coding agent** through the MCP server (6.21).
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

1. `archdoc` with no arguments opens **Home** (§5.3) — recent folders and systems.
2. Open a folder → **Generate** → the narrative of §5.2, locally.
3. The same app as the platform (§4.1): the same pages and names, with the local features of §4.5.

Later: "publish to archdoc" sends the result to the platform. Not designed in detail now; leave the
slot.

---

## 4. Information architecture — one app

> **Revision 3, 2026-10-19.** Replaces this section's earlier navigation, and supersedes the shell and
> left navigation of `surface-spec.md` §4.1 for the whole app — local, hosted and published. The
> screens of `surface-spec.md` §5 keep their content; this section decides where each one lives,
> what it is called, and where it appears.

### 4.1 Four principles

1. **One app.** The local app, the hosted platform and the published site are the same application:
   one shell, one navigation, one set of pages, the same names. A feature appears where what it
   needs is available, and nowhere else (§4.5). As with a product run locally and in the cloud,
   the pages are the same and each place has its own few features. The shared piece of
   `platform-plan.md` step 2 is therefore the whole app, shell included, with an adapter per place.
2. **Density budget** (rule 15). The first view of any screen holds one question, one sentence, one
   picture and at most one primary action. Lists stop at five, with "see all". Anything more is a
   deliberate step away — another page, or a panel the user opens — never an expanded section on the
   same page. No menu holds more than seven entries at any level.
3. **Plain names** (rule 16). Every menu entry, level and heading is named in plain words first; the
   technical term (C4, arc42, archdoc's own) follows it, smaller, where it helps someone who knows
   it — "Apps and services" over *containers*.
4. **Nothing lost, little removed.** Every feature already built or designed is kept unless it no
   longer makes sense. Nearly all of the change is moving things, not cutting them (§4.4).

### 4.2 Three levels

```
Home                      your systems and repositories          (L, D)
└─ System                 how several repositories fit together  (L, D) — §4.6
   └─ Repository          everything below                       (L, C, D, published)
```

The breadcrumb in the top bar carries the levels: `Home › archdoc › engine › What changed`. A
repository in no system sits directly under Home. A single public repository (C) and a published
site have only the repository level.

### 4.3 The shell

```
┌───────────────────────────────────────────────────────────────────────────────┐
│ archdoc   Home › archdoc › engine ▾   ● Up to date ▾  [⌘K Search or ask…] Share ◐ ⋯ ◯│
├────────────────────┬──────────────────────────────────────────┬───────────────┤
│ Overview           │                                          │ Inspector /   │
│ What changed    Δ  │              main content                │ Impact /      │
│ Did it build what  │                                          │ On this page  │
│   I asked?         │                                          │ (opened, not  │
│ ─────────────      │                                          │  always on)   │
│ Map                │                                          │               │
│ What it does       │                                          │               │
│ Reference ▸        │                                          │               │
│ ─────────────      │                                          │               │
│ Settings           │                                          │               │
│                    │                                          │               │
└────────────────────┴──────────────────────────────────────────┴───────────────┘
```

**Top bar:**
- **Breadcrumb** with a switcher at each level.
- **Status** (6.16, and the freshness pill of `surface-spec.md` §11.7): *Up to date* · *Out of date*
  · *Generating…* · *Failed*. "Out of date" means the documentation is behind the code; it is never
  called "changed", which is kept for the architecture (What changed) — two concepts, two words, two
  glyphs. Its menu holds the action that fits: locally **Update** — free, offline, one click,
  with the progress of 6.17 — and, as a separate and deliberate step, **Update with AI
  explanations**, which first shows what will be sent and roughly what it costs (rule 19); on the hosted platform regeneration follows each push; in the published site,
  "Generated from commit … on …". Locally the same menu shows **archdoc's uncommitted files** and the
  command to commit them (`surface-spec.md` C-4).
- **Search and Ask, one box** (`surface-spec.md` §5.17, §6.5 here): it finds parts, routes, tables
  and pages as you type; when what you type is a question, it offers **Ask about…**. Two boxes made
  the user guess which one to use.
- **Share** — the same entry everywhere, with what each place can do (§5.6).
- **Theme**, in every place — the account only remembers a default.
- **⋯** for what belongs to the page: the version picker, keyboard shortcuts (`?`, in every place),
  the truth-state legend.
- **The far-right slot is "you"**, and differs by place:

  | Place | Far-right slot |
  |---|---|
  | Local | A settings icon → **This machine** (§5.9) |
  | Website, signed in | Your avatar → **Account settings**, **Connect a repository**, **Sign out** |
  | Website, signed out | **Sign in with GitHub** |
  | Published site | Nothing |

- **+ Connect**, beside the avatar, on Home and the system pages when signed in. Home keeps its own
  Connect button only in its empty state, so no page has two of the same action (rule 15).

**Left menu — the repository level, six entries and Settings:**

| Entry | Plain question | Holds |
|---|---|---|
| **Overview** | What is this? | §6.1 |
| **What changed** | What just changed? | §6.2; the timeline and comparing commits under Advanced |
| **Did it build what I asked?** | Did the code do what I asked? | §6.3 |
| **Map** | How is it put together? | The explorer, with its levels as a control on the canvas (§4.4) |
| **What it does** | What can it do, and how does each thing work? | Features, each opening onto its flows |
| **Reference** | Where is the complete detail? | The documentation, every component, every table, everything it relies on |
| **Settings** | How does this repository behave? | Per place (§4.5) |

**Ask** lives in the search box (above), wherever Ask exists — locally and for a signed-in owner on
the website; absent signed out and on the published site (§4.5). Its answer opens in the right rail.

The **right rail** opens when something is selected — the inspector, Impact, an Ask answer, "On this
page" — and is closed otherwise.

**Where am I** is said once: the breadcrumb carries every level and sub-page (`Reference › Every
table`, `What changed › Every version`), with the menu's highlight. No separate back links inside a
section, and no place note at the foot of the menu — the far-right slot already says the place.

**The system level** has its own short menu: **System map**, **What changed** (across its
repositories), **Settings**. **Home** has none: it is a page.

### 4.4 Where every feature goes

Every screen and control of `surface-spec.md` and every screen of this document, with where it
now lives. *Moved* means reached from a new place; nothing below is cut unless marked **removed**.

**Repository screens (`surface-spec.md` §5)**

| Feature | Plain name (technical) | Lives in | Change |
|---|---|---|---|
| Overview §5.1 | Overview | Menu | Reworked (§6.1) |
| Architecture explorer §5.2 | **Map** | Menu | Its levels become a control on the canvas, not menu entries. **Two ways to move, both kept:** the tabs, and **double-clicking a part** to go inside it. The tabs are grouped as what they are — **zoom** (the outside world, apps and services, inside a part) and **lenses** on the same system (what it stores, how data moves, where it runs) — so "inside" always has a part |
| — level: context | The outside world *(context)* | Map level | Moved |
| — level: containers | Apps and services *(containers)* | Map level | Moved |
| — level: components | Inside *the part* — "Inside immich-server" *(components)* | Map level | Moved; named after the part it opens, not generically. "Components diagram" in the menu **removed** as a duplicate |
| — level: data | What it stores *(entities)* | Map level | Moved; "Data model" in the menu **removed** — it opened the same screen |
| — level: data flow | How data moves *(data flow)* | Map level | Moved |
| Deployment §5.9 | Where it runs *(deployment)* | Map level | Moved |
| Saved views, lens | Saved views | Inside Map | Moved; **only views a person saved** |
| The diagram in delta mode (§5.12) | Compare | A **mode** of the Map, beside Saved views — not a saved view | Moved from What changed; "See it on the map" opens it |
| Inspector §5.3 | — | Right rail, on selection | Unchanged |
| Impact (§6.4) | What does this touch? | Right rail, from any element | Unchanged |
| Component pages §5.4 | Every component | Reference | Moved; also reached from the inspector, search and Ask |
| Features §5.5 | **What it does** | Menu | Merged with Flows |
| Flows §5.6 | How each thing works *(flows)* | Inside What it does, under each feature | Merged |
| Data model tables §5.7 | Every table | Reference | Moved; the picture stays the Map's Data level |
| Dependencies §5.8 | What it relies on *(dependencies)* | Reference | Moved |
| The list of entry points — every route, page, command and job (the old Features, §5.5) | Every entry point | Reference | **Kept**: "What it does" replaced the screen with feature sentences, and the list had lost its home |
| Documents §5.10 | Documentation *(arc42)* | Reference | Moved; each chapter its own page with "On this page" in the rail — a twelve-chapter document is not cut to five rows |
| Coverage §5.11 | What archdoc couldn't read *(coverage)* | **About these docs** | Moved; reached from the Overview's trust sentence |
| Changes §5.12 | What changed | Menu | Reworked (§6.2) |
| Corrections §5.13 | Your corrections | Settings | Moved; "Correct this…" stays in the inspector. On the published site, which has no Settings: **Corrections in force**, in About these docs |
| Network runs §5.14 | What left this machine | About these docs (local) · AI usage in Account settings (website) | Moved |
| Intent vs actual §5.15 | Did it build what I asked? | Menu | Promoted (§6.3) |
| Ask the map §5.16 | Ask | In the search box, where it exists (§4.5) | Promoted and merged with search (§6.5) |
| Global search §5.17 | Search | Top bar, one box with Ask | Merged |
| "Not built yet" | — | — | **Removed from the product**; shown in development builds only |

**About these docs** is one page, reached from the trust sentence and from the ⋯ menu: what archdoc
could and could not read, which commit and version this is, how it was generated (with or without
AI), and — locally — what left the machine. It is the honesty of the product in one place, not a
menu entry.

**Controls (`surface-spec.md` §11)**

| Control | Lives in |
|---|---|
| Freshness pill, Run, Generate, progress | Status in the top bar (§4.3) |
| Watch mode | Settings (local) |
| Arrange, saved views | Map |
| Correct this… | Inspector → Settings › Your corrections |
| Compare two commits | What changed › Advanced |
| Uncommitted files | Status menu (local) |
| Publish (build the site, the CI workflow) | Share (local) |
| Settings panel (egress, model, read-only for now — C-3) | Settings |

**Platform screens (§5 of this document)**

| Screen | Lives in | Change |
|---|---|---|
| Landing §5.1 | **The website**, outside the app; the app keeps **Start** | **Moved out** (revision 5) |
| Generating §5.2 | The status's progress; full screen on a first run | Unchanged |
| Home §5.3 | Home level — **local too** | Start (§5.8) merged into it |
| Connect a repository §5.4 | Home (D) | Unchanged |
| Repository settings §5.5 | Settings | Only the repository's own choices; the person's defaults in Account settings or This machine (§5.9) |
| Share §5.6 | Top bar | Local publishing joins it |
| Connect your agent §5.7 | Settings, and Home's ⋯ | Unchanged |
| Start §5.8 | — | **Merged into Home** |
| Account §5.9 | **Account settings**, from the avatar (signed in); **This machine** locally | Reworked (§5.9) |
| System map | System level | **New** (§4.6) |

### 4.5 What appears where

One app; a feature shows where what it needs exists.

**Three places, and the website has two states.** archdoc runs on your machine, on the archdoc
website, or as static files you published. On the website, what changes the screen is the visitor,
not the repository: signed in or signed out. A signed-in visitor to someone else's public
repository has their avatar and account but none of that repository's settings — **viewer**, not
**owner**. The "signed in" column below is the owner's.

| Feature | Local | Website, signed out | Website, signed in | Published site |
|---|---|---|---|---|
| Home | Recent folders and systems | — (the website instead) | Your systems and repositories | — |
| System level | A folder of repositories, or a workspace file | — | Connected repositories | — |
| Every repository screen | ✓ | ✓ | ✓ | ✓, one version and its baseline |
| Generate | From the status menu | Automatic, on request | Automatic, on every push | — |
| What changed — "since you last looked" | **Since you last opened it on this machine** — remembered locally, so it means the same as on the platform | Since the previous commit | Since this user last opened it | Since the publisher's baseline |
| Did it build what I asked? | ✓ — with its plans in local Settings | — | ✓ | Read-only |
| Ask | Local model or the user's key | — | ✓ | — |
| Your corrections | Written to `rules.yaml`, append-only, after a preview (C-2) | — | Opens a pull request | Read-only list |
| Share | Publish: build the site, copy the CI workflow | Link | Link, embed, badge | Link |
| Uncommitted files | ✓ | — | — | — |
| What left this machine | ✓ | — | AI usage in Account settings | Summary |
| Watch mode | ✓ | — | — | — |
| Notifications | Through the agent | — | Agent, email, pull requests | — |
| Connect your agent | `archdoc mcp` | — | ✓ | — |
| Sign in, account | — | Sign in only | ✓ | — |

A feature that does not apply is **absent**, not disabled. The one exception: Start and Overview
on C may say "sign in to keep this repository current" — the one place the platform invites.

### 4.6 Related repositories — the system level

**Answers:** how do my repositories fit together?

Software is often more than one repository — archdoc itself is about to become an engine, a platform
and a shared app. A **system** is a named group of repositories, and its screen shows how they
relate. Not every repository at once: each is one box, and the arrows are the evidence that links
them.

**What counts as a link** — always cited **at both ends**:
- **Code:** one repository imports another — a Go module in `go.mod`, a package in `package.json`,
  pinned at a version.
- **Images:** a Compose file runs an image another repository builds.
- **Calls:** a configured address in one repository resolves to a server another declares.
- **Shared contracts:** an API schema or table definitions one publishes and another uses.
- **Distribution:** one installs or deploys the other's releases — a package formula, a deployment
  repository, a release workflow that publishes where another fetches.

What archdoc cannot prove stays what it is today — an outside system, unresolved where the evidence
is incomplete. **A link is never guessed.**

**Version drift** is shown wherever a link is pinned: "the platform uses the engine at v0.3.0; the
engine is at v0.5.0".

**Grouping:**
- **The user groups first.** A system is what the user says it is — on the platform from connected
  repositories; locally from a folder holding several repositories, or a workspace file.
- **archdoc suggests second**, and only suggests: when a repository links to one the user can reach
  and that is not in the system — "the platform imports the engine; add it to this system?" Grouped
  repositories always come first; suggestions are marked as suggestions and never grouped on their
  own. A `go.work` or similar workspace file is a suggestion too, not a grouping.

**The System map screen:**
1. **The sentence**: "archdoc is three repositories. The platform imports the engine at v0.3.0 and
   installs its app; the engine is two versions ahead."
2. **The picture**: repositories as boxes, each with **what it is** in one line (from the same source
   as the Overview's paragraph, §6.1) and its status line (6.16); arrows labelled by kind of link;
   drift marked on the arrow. Two links between the same pair draw as parallel arrows; from five
   up, they bundle into one arrow with a count.
3. **The evidence**, for any arrow: the lines at both ends.
4. Any box opens that repository's Overview.

**What changed across the system** — the change stories of its repositories, together. Later, when
the engine can tell: a change in one repository's public surface marked on the repositories that
depend on it.

**Engine work this needs** — after the freeze: composing several repositories' models and matching
one's outside systems against another's declared services (`product-definition.md`'s R2
"cross-repo visibility"; S6 in `hosted-proposal.md` §7).

### 4.7 Words

One word per idea, everywhere — the legend, the marks, the sentences, the notifications (rule 22):

| Idea | Say | Not |
|---|---|---|
| Extracted from the code, with its line | **Proven** | — |
| Written by the model on top of proven facts | **Written by AI** | interpreted, the AI's reading, AI-written |
| Evidence that something exists, which archdoc could not identify | **Couldn't be resolved** | unresolved, not provable, seen but not provable |
| A file or language archdoc does not parse | **Not read** | couldn't read, not read yet, named — and never mixed up with *couldn't be resolved* |
| The documentation is behind the code | **Out of date** → **Update** | code changed, Generate |
| A match the model made against a plan | **Looks built** / **looks missing** | found, missing |
| Archdoc proposes, the user decides | **Suggested** | reachable, not grouped |
| When something happened | a date, a push, a commit | v14 (only under Advanced, rule 20) |

The truth states keep their technical names in this document and in `surface-spec.md`; on screen,
the left column is never shown — only "Say".

The website's link box says what the visitor gets — **Show its architecture** — rather than "Read it".

---

## 5. Platform screens

### 5.1 Landing · C — moved to the website in revision 5

**Answers:** what is this, and can I try it right now?

The Landing is no longer a screen of the app. It is the home page of **the website** — a static site
at the root domain, outside the app and its shell (`platform-plan.md` §7, "The website"). Its content
is unchanged:

- One sentence of what archdoc is, and one input: **paste a GitHub link**.
- A **gallery** of well-known repositories, already generated — Immich, Supabase, Mastodon, others.
  Each card: name, one-line summary, the diagram's thumbnail.
- The three ways to use it: local (how to install), the GitHub Action, hosted.
- Sign in with GitHub, for private repositories.

Around it, the website's other pages: How it works, Gallery (a page per repository), Trust, Pricing,
Docs, Changelog, Terms and Privacy.

**What stays in the app — Start** (`/new?repo=…`): the link box submits to it, and it shows the
states: link not a GitHub repository; repository private (→ sign in); already generated at this
commit (→ straight to Overview); otherwise → Generating (§5.2). It has a link box of its own for
anyone who arrives without one. A signed-out visitor at the app's root is sent to the website.

**Tone:** the website is the one place allowed to sell. Still calm; the gallery does the selling.
**Designed in a round of its own**, after the app's — same design system, no app shell.

### 5.2 Generating · C, D, L

**Answers:** is it working, and what is it finding?

- A live narrative from the engine's progress (6.17): "Found 3 applications · reading TypeScript ·
  38 routes so far · 12 tables".
- What it is reading now, and roughly how far along.
- D: "You can leave — we'll tell you when it's done."
- The picture of what has been found so far carries a visible **in progress** mark on the canvas. A
  half-built diagram is a wrong diagram the moment it is screenshot without one.

**States, each designed as a screen, not an error:**
- **Too big** — the repository exceeds the limit; what the limit is. Starting values, provisional
  until measured on real repositories: **50,000 files, 2 GB, 10 minutes**.
- **Timed out** — the run stopped at the time limit and **keeps nothing**, as the engine never writes
  a partial model (VAL-08). The screen says what was being read when it stopped.
- **Stack not read yet** — what *was* read, what was not, and the honest partial result (6.18).
- **Failed** — what happened, in plain words, and what to try.
- **Private** — sign in to continue.

### 5.3 Home · L, D

**Answers:** did anything change since I last looked?

Locally, Home is what Start was (§5.8): recent folders and systems, **Open a folder**, and for a folder
never generated, one **Generate** button that says it runs offline and sends nothing.

- One **card per system** — its repositories' status lines together — and one **card per repository**
  not in a system (6.19): name, a single status line — *current*, *3 pushes changed the
  architecture since you last looked*, *2 things couldn't be resolved*, *generating…* — and when it last changed.
- A **feed** of recent changes across repositories, each a sentence linking to What changed.
- **Connect a repository.**

**States:** no repositories yet (→ connect, or try a public one).
**With one repository there is no Home**: a card and a feed would repeat that repository's What
changed. Sign-in goes to its Overview, and the breadcrumb's `Home` becomes a repository switcher.
Home appears from the second repository on.

### 5.4 Connect a repository · D

**Answers:** what will you read, and what do you keep?

- **First**, in plain words: what archdoc reads, what it stores (the documentation, which contains no
  code and no secrets), and what it does not keep — **subject to the promise in
  `hosted-proposal.md` question 6**, still open. The promise comes before the install, because the
  install is where trust is decided.
- **Then** the GitHub App installation and the repository picker.
- Which branch to track.

### 5.5 Repository settings · L, D

**Answers:** how should this repository behave?

Only this repository's own choices. Anything that belongs to the person — AI, notifications, agent
setup, theme — lives in **Account settings** on the website, or **This machine** locally (§5.9); a
repository setting that overrides one of those defaults says so and links to it.

- Branch tracked; visibility (private, link-only, public). *Website.*
- **Plans** for Flow 4: which files, issues or tasks are the intent. *Both.*
- **Corrections**: the rules in force; locally written to `rules.yaml` after a preview, on the website
  a pull request. *Both.*
- **Watch mode**. *Local.*
- **Overrides** of the account's defaults — notifications for this repository, AI off for this
  repository — each linking to the default it overrides. *Website.*

### 5.6 Share · L, C, D, published

**Answers:** how do I show someone?

Locally, Share is **publishing**: build the site, and the CI workflow to copy (`surface-spec.md` §11.2).
On the platform:

- **The link is the answer**: a link to this view (public repositories always; private ones when the
  owner allows). **Pinned to its version by default**, so what it shows stays true; **"always
  latest"** as an option.
- Under **More**: an embed for a diagram, and a **README badge** — "architecture: current" — with the
  snippet to paste. The badge is always latest.

### 5.7 Connect your agent · D, L

**Answers:** how does my coding agent use this?

- Copy-paste configuration for Claude Code, Cursor and other MCP clients, with a token. Locally, the
  same server is `archdoc mcp`.
- What the agent can ask, in examples: "who writes this table?", "what does this route set off?".
- What the agent will be told after a session (Flow 3).

The first user's agent may use archdoc more often than they do. This screen makes the agent a
first-class surface rather than an afterthought.

### 5.8 Start · L — merged into Home (§5.3) in revision 3

Kept for the record; Home holds all of it now.

**Answers:** what do you want to look at?

- **Open a folder**; recent projects, each with the status line of 6.16.
- For a folder never generated: one button, **Generate**, and what it will do — offline, nothing
  sent.

### 5.9 Account settings · signed in — and This machine · local

**Answers:** how is archdoc set up for me?

**Account settings**, on the website, reached from the avatar:
- Profile, and the connected GitHub account.
- **AI: usage and cap** — this month's use and the monthly cap. **When the cap is reached**,
  explanations fall back to templates (§6.1) and the screens say so — the product is complete
  without the model. No key field yet: whether the website uses archdoc's key or the
  user's own is `hosted-proposal.md` question 3, still open.
- **Agent tokens** — connect your agent (§5.7).
- **Notification defaults** — agent, email, pull-request comments (6.21); a repository may override.
- **Theme default.**
- **Billing** — a slot. Kept together with the AI cap as one section, since both move to organisations
  when teams arrive.

**This machine**, locally, reached from the settings icon:
- **AI** — the key read from the environment, shown and never changed here (`surface-spec.md` C-3).
- **Connect your agent** — `archdoc mcp`.
- **Theme default.**

---

## 6. Repository screens, rebuilt around questions

### 6.1 Overview — replaces `surface-spec.md` §5.1

**Answers:** what is this system?

In order, nothing else above the fold:

1. **The system in one paragraph**, plain words: "A photo app. A web front end and a mobile app talk
   to one server, which keeps its data in Postgres and hands heavy work to a machine-learning
   service." Each clause cited (6.15).
2. **One diagram** — the most useful level, clickable into the Map.
3. **What it does** — the main features as sentences, each linking to its flow.
4. **Trust in one sentence** — the one second sentence the density budget allows (rule 15): "Every part and connection here is proven from your code; the plain
   sentences are written by AI on top of them. 11 things couldn't be resolved." Links to Coverage.
   With no AI used: "Everything here is proven from your code; 11 things couldn't be resolved."
   The sentence opens **About these docs** (§4.4).

The stat strip of today's Overview goes; its counts live in Reference and About these docs. The
trust strip becomes the trust sentence.

**Where the paragraph and the features come from** (§10, question 1 — answered):
- **Structure, always from templates.** A deterministic sentence from the container graph: "immich is
  made of six parts. Two clients call immich-server, which stores data in database, queues work in
  redis and calls machine-learning." Proven; the only option on C.
- **Purpose, from the manifest where it exists.** A `description` in `package.json`, `pubspec.yaml`
  and the like, quoted and cited — proven, because the repository says it.
- **Purpose and features from the AI** on D and L when it is on, marked by the block (rule 13).
- **On C with no manifest description**, the Overview leads with the structure sentence and **drops
  "What it does"** rather than show route counts in its place.

**No "since you last looked" here** (revision 4): it is What changed's opening strip and the count
beside What changed in the menu.

**States:** nothing could be read (6.18); no AI used — the structure sentence alone, "What it does"
only if a manifest describes it.

### 6.2 What changed — reworks `surface-spec.md` §5.12

**Answers:** what just changed in the architecture?

- **Opens with a strip** answering "what changed since I last looked?" — "3 pushes changed the
  architecture since you last looked" — which is also the picker between them.
- **Then one push**, the latest unless another was picked, **told as a story**: "This push added
  comments on albums — a new route `POST /albums/:id/comments`, an `album_comment` table, a job that
  notifies album members, and a call to Firebase Cloud Messaging." 
- Grouped by **what it means**, not by kind of edit:
  - new capability (routes, pages, jobs);
  - new outside service;
  - data changes (tables, columns, keys);
  - **changed** — renamed, rewired, or a new technology for an existing part ("the job queue moved
    from Valkey 8 to Valkey 9"); the engine's `diff` already tells these apart;
  - removed;
  - wording only (the model relabelled something) — kept apart, as today.
- **"See it on the map"** opens the Map's Compare mode (§4.4); the diagram is not on this screen.
- **Evidence in view** (rule 12): each item's citation stays visible, muted, right-aligned.
- **Advanced**: the timeline, and comparing any two versions or commits — today's screen, intact.

**States:** nothing changed architecturally ("this push changed code but not the architecture" — a
useful answer, designed); first version.

### 6.3 Did it build what I asked? — promotes `surface-spec.md` §5.15

**Answers:** did the code do what I asked for?

- The plan, requirement by requirement, in three groups:
  - **found** — where in the code, cited;
  - **missing**;
  - **built but not asked for**.
- **A verdict that says whose judgment it is** (rule 14): "Reading your plan against the code: 4 of
  6 requirements look built, 2 look missing, and 1 thing was built that the plan didn't ask for."
  Matching a plan's sentence to code is the model's reading — "look built", never "built".
- **Every match is interpreted**, marked by the block (rule 13), with the code it matched shown beside
  it, so any line can be checked at a glance. **Evidence in view** (rule 12).
- **"Missing" carries the strongest evidence of all**, being the claim most likely to be wrong: each
  missing requirement shows what archdoc looked for and did not find — "no route, job or page
  mentions deleting a comment".
- **Never a score**: no percentage, no bar, no tick. The same wording carries into the email, the
  pull-request comment and the agent message (6.21).

**States:** no plan yet (→ settings, with an example of a plan that works); a requirement too vague
to check ("couldn't tell — here's why").
Top-level because it is the most direct answer to the problem archdoc exists for.

### 6.4 Impact — new

**Answers:** what does this touch?

A panel, opened from any element — in a diagram, a search result, an Ask answer:

- The sentence first, **deterministic**: it names every flow and job affected, in a fixed order, or
  past five, names the first five and counts the rest — "Changing `users` affects 6 flows and 4
  jobs: login, sign-up, delete account, …". It never picks which ones to mention.
- **Evidence in view** (rule 12).
- Who **reads** it and who **writes** it.
- Which **routes** and **flows** pass through it.
- Which **outside services** it reaches.
- Each item cited; each opens.

### 6.5 Ask — promotes `surface-spec.md` §5.16

In the search box (§4.3): a question offers **Ask about…**, and the answer opens in the right rail.
Answers drawn only from the validated model, each sentence cited; "not resolved — here is what archdoc
saw" is a designed answer. **Ask uses the model, so it says so** (rule 19): once, plainly, with roughly
what a question costs — locally on the user's key, on the website within the AI cap.

### 6.6 Map, What it does, Reference — formerly Explore

`surface-spec.md` §5.2–§5.11, §5.13, §5.17 — redistributed in revision 3 (§4.4): the explorer and
its levels into **Map**, features and flows into **What it does**, component pages, tables,
dependencies and documents into **Reference**, coverage into **About these docs**, corrections into
**Settings**. Unchanged in content, but each now under the density budget (rule 15).

**What it does without AI** (hosted public, or AI off): never a list of route counts. Deterministic
templates turn each area's routes into verbs — `POST`, `GET`, `DELETE /assets` → "Adds, lists and
deletes assets" — cited to the routes they come from. Where that is not possible, the entry is
absent rather than a list of numbers (rule 17). Each gains the three-layer opening of 6.15 where it does not already start with a
sentence.

---

## 7. New shared components

Numbered on from `surface-spec.md` §6, so the design system stays one list.

| # | Component | Notes |
|---|---|---|
| 6.15 | **Three-layer block** — sentence, picture, evidence | The shape of every screen's opening. A question line, then the plain sentence; the picture is one diagram or a short list; the evidence (citations, the full table). **Evidence closed on screens that explain, open on screens that review** (rule 12). AI-written sentences marked by the block (rule 13); a citation marker in the sentence opens the evidence at its row. The single most important new component |
| 6.16 | **Status line** | One sentence of state for a repository: current, changed, unresolved, generating, failed. Used on cards, Home, the shell, the notification subject. **What is counted: pushes that changed the architecture** — "3 pushes changed the architecture since you last looked". Locally the unit is commits. A push counts when its diff has anything in any group of 6.20 but wording only; wording only never counts, or the model relabelling something would raise it. Sessions as the unit are a later refinement. **Priority:** failed, then generating, then changed. **The shell's pill is separate** (§4.3): Up to date · Out of date · Generating… · Failed — freshness, with its own glyph, never Δ **Unresolved shows only when its count rose** since the user last looked; a standing count belongs in the Overview's trust sentence, or it would hide every change |
| 6.17 | **Progress narrative** | The live "what I'm finding" of §5.2. Lines appear as the engine reports them; a calm sense of progress without a fake percentage. Any picture of what is found so far carries an **in progress** mark |
| 6.18 | **Partial-result state** | When archdoc read only part of a repository: what was read, what was not, what is still shown. Extends `surface-spec.md` 6.8 |
| 6.19 | **Repository card** | Name, status line, last change, thumbnail. Home, the System map, the gallery. A **system card** holds its repositories' status lines together |
| 6.20 | **Change story** | A push or session as a sentence, its grouped changes beneath, in **six** groups always in this order: new capability, new outside service, data changes, changed (renamed, rewired, new technology), removed, wording only. The diagram is not part of the story; it is the Map's Compare mode. Wording only uses `~`. **Grouped by the engine** (§10, question 2), so every channel and the local app agree. The template has a **tense and subject slot**: "this push added" once pushed, "this pull request adds" before the merge |
| 6.21 | **Notifications** | The same change story in three places: an email, a pull-request comment, a message the agent receives. Designed together so they say the same thing. **The agent is the default channel** for the first user (§10, question 4): the session is where the work happens, and the plan check is something it can act on at once. Email for pushes that did not come from an agent; pull-request comments when work arrives as pull requests |

---

## 8. Data — what the new screens need

All of it is platform API, except where marked L (the local app's `serve`).

| Data | Needed for | New work |
|---|---|---|
| Run status and progress stream | Generating | The engine's progress callback (`platform-plan.md` step 1), relayed live |
| Repositories with status line | Home (L, D), System map | Per repository: last run, last change, unresolved count |
| Last seen, per user and repository | "Since you last looked" | D on the platform; L in the local cache; C falls back to the previous commit |
| Change story | What changed, Home feed, notifications | Grouping of diff results by meaning (§6.2), in the engine; whether a push counts (6.16) |
| Plain-language summary | Overview | The structure sentence from templates; manifest descriptions read and cited; the model's purpose and feature sentences where AI is on (§6.1) |
| Impact for an element | Impact | A query over the model: readers, writers, routes, flows, outside services |
| Intent results | Did it build what I asked? | F-40, F-41 |
| Gallery | The website's home and gallery pages | A curated list of repositories, generated by the engine at pinned commits when the website is built — no platform needed |
| Settings, sharing, tokens | Settings, Share, Connect your agent | Platform tables |
| Systems and their links | System map, Home | Which repositories a system holds; links between them with both citations; pinned versions for drift; suggestions kept apart from groupings — multi-repository composition in the engine (§4.6) |

---

## 9. Handing this to Claude Design

**Round 3 — one app.** Rounds 1 and 2 drew the platform beside the existing app: the new screens link
out to `Surface Screens` for everything under Explore. Round 3 draws **one application** with the
architecture of §4, and the earlier files become the "before".

**Give it:** this document, §0–§8 and §12 first; `surface-spec.md` §0, §2, §5 (for each screen's
content) and §6 (the design system).

**Ask for, in this order** — the first two set what everything else reuses:

1. **The shell** (§4.3) on Immich: breadcrumb, status, search, Share, the menu of six entries and
   Settings, search and Ask in one box, the right rail closed and open. Then the same shell in the three places of
   §4.5 — local, hosted private, published — so the differences are visible and nothing else differs.
2. **The Map**, with its levels as a control on the canvas under their plain names (§4.4), saved
   views, the inspector, and Impact.
3. **Overview** under the density budget (rule 15) — and the round-2 fixes of §11.
4. **What it does** — a feature opening onto its flows; **Reference** — documentation, every
   component, every table, what it relies on; **About these docs**; **Settings**, local and hosted.
5. **What changed** and **Did it build what I asked?**, with the round-2 fixes.
6. **The System map** (§4.6) — archdoc's own split: the engine and the platform, grouped by the
   user; the platform imports the engine at v0.3.0 while the engine is at v0.5.0; one suggestion
   kept apart — a `homebrew-archdoc` repository whose formula installs the engine's releases. And
   **Home** with a system card and a single repository, locally and on the platform.
7. Start, Generating, Connect a repository, Share, Connect your agent, Account — updated to the
   shell, and the notifications. The Landing is now the website (§5.1, revision 5), designed in a
   round of its own.

**Use realistic content.** Immich as in `surface-spec.md` §8; Supabase for the large case; Mastodon
for the small case. For What changed and Did it build what I asked?, the album-comments push
(`9c3e7d1`, 18 Oct, v14) and the plan "Comments on albums", every name consistent across screens.

**Tone:** a precise instrument that opens in plain words — a good explainer first, a good code editor
one step behind. Less on every screen than before: when in doubt, it goes one step away. The website
is the one place allowed to sell, and the gallery does it.

**What we want from the design:** to see the shape and find its flaws. A screen that does not work
once drawn is a finding, not a failure.

---

## 10. Open

Answered in revision 2: ~~where the Overview's plain language comes from~~ (§6.1); ~~who groups
changes~~ — the engine (6.20); ~~Home with one repository~~ — none (§5.3); ~~the default notification
channel~~ — the agent (6.21).

Still open:

1. **Corrections as pull requests** — kept: for a solo owner it is friction, but it is also the
   guarantee. Worth testing: a one-click "open and merge" for a repository's sole admin.
2. **"Publish to archdoc" from local** — the bridge between L and D; slot only for now.
3. **The never-keep-your-code promise** (`hosted-proposal.md` question 6) — Connect a repository
   (§5.4) shows it as a draft until it is decided; the `.env` line is the undecided part.
4. **The gallery** — which repositories. Who keeps it current is answered by revision 5: the
   website's build, running the engine at pinned commits. Kept from the design: a partly read
   repository (Mastodon) says so on its card — the product's honesty, before sign-up.
5. **The name and domain** — `archdoc.dev` in the designs is a placeholder (`platform-plan.md`
   question 4).
6. **Clustering in the Data view** — grouping tables by name prefix leaves one cluster of 34 of
   Immich's 68 tables. Needs a better rule, in the engine, before the Data picture works.
7. **The workspace file** — how a local system is declared when its repositories are not in one
   folder: a small file listing them, or reading a `go.work`, `pnpm-workspace.yaml` and the like as
   suggestions only (§4.6).
8. **Systems on C** — can a public organisation's repositories form a system without an account?
   Not in this revision; the slot stays.
9. **The plain names** of §4.4 — first wording, to be tested on the first user before they settle.

---

## 11. Revision 2 — what the first design round changed

The round (Claude Design, 19 Oct) drew every screen and component of this brief and wrote
`Platform Findings`. Drawn, the brief showed flaws it did not show as text. Taken into this revision:

| Finding | Now |
|---|---|
| "Everything here is proven" sat under a sentence the AI wrote | The trust sentence names what is proven and what is written by AI (§6.1) |
| Plain-first means AI-first: every line carried the interpreted mark | Marked by the block (rule 13) |
| Evidence closed by default was wrong where checking is the job | Closed where a screen explains, open where it reviews (rule 12) |
| What changed answered two questions | Always opens on one push; "since you last looked" is the Overview's (§6.2) |
| A standing unresolved count hid every change on the status line | Unresolved shows only when it rose (6.16) |
| "3 changes" had no definition | Pushes that changed the architecture; wording only never counts (6.16) |
| "4 of 6 found" presented the model's reading as a measurement | Said as a reading, missing items with what was looked for, never a score (rule 14, §6.3) |
| Rule 10 conflicted with the sentences it required | No screen opens on a *bare* number (rule 10) |
| Impact's sentence chose which flows to name | Deterministic: all, or the first five and a count (§6.4) |
| The half-built diagram while generating reads as a wrong one | An in-progress mark (6.17, §5.2) |
| The promise came after the GitHub install | Before it (§5.4) |
| Share held three answers | The link first; embed and badge under More (§5.6) |
| Explore held most of the navigation | Starts collapsed — superseded in revision 3 by Map, What it does and Reference (§4.3) |
| The pull-request comment needed a different tense | A tense and subject slot in the story (6.20) |
| Open questions 1–4 | Answered: templates + manifest + AI (§6.1), the engine groups (6.20), no Home with one repository (§5.3), the agent first (6.21) |

**The design's guesses, confirmed:** hosted limits of 50,000 files, 2 GB and 10 minutes, provisional
until measured; a timed-out run keeps nothing; share links pinned to a version, with "always latest"
as an option; templates when the AI cap is reached; `archdoc mcp` for the local MCP server.
`archdoc.dev` stays a placeholder.

**Corrections to the drawings, for round 2:** ochre is for unresolved only — the agent message's
plan-check line must not use it; Flows and Notifications need their dark theme.

**Example content from now on:** the album-comments push (`9c3e7d1`, 18 Oct, v14) and the plan
"Comments on albums" replace the refunds example of revision 1.

---

## 12. Revision 3 — one app, and beyond one repository

After the first round the team found four flaws the findings had not named:

| Flaw | Now |
|---|---|
| **Still too much on every screen.** The plain sentence was added on top of the density instead of replacing some of it | A density budget for every screen and menu (rule 15, §4.1) |
| **The rounds did not merge.** The platform screens were drawn beside the existing app — every Explore entry linked out to `Surface Screens` — so there were two shells and the old navigation with three entries added | One app (§4.1): one shell, one navigation, every existing screen placed in it (§4.4) |
| **The navigation had no clear purpose.** In the built app, up to twenty entries: "Data" and "Data model" opening the same screen, "Components" three times, a development list ("Not built yet") in the product, and names — Coverage, Corrections, Network runs, Documents — that mean little to the first user | Six entries and Settings, each answering a plain question (§4.3); levels inside the Map; duplicates removed; plain names (rule 16) |
| **Local and hosted were drawn as different places.** A separate Start screen and separate shells | The same pages everywhere; each feature appears where it can work, and is absent elsewhere (§4.5, rule 17) |

And one feature not discussed before: **related repositories.** archdoc is about to become several
repositories itself. A **system** groups repositories, and its map shows how they link — imports,
images, calls and shared contracts, each cited at both ends, with version drift — grouped by the
user first, suggested by archdoc second, never guessed (§4.6).

**Nothing built or designed was cut** except two duplicate menu entries and the development list in
the product. Everything else moved; §4.4 lists where.

**Follow-on changes:** Start merged into Home; Settings, Share and Connect your agent exist locally
too; the corrections composer locally writes `rules.yaml` after a preview, as `surface-spec.md` C-2
decided — revision 1 had said it copies YAML; `platform-plan.md` step 2 shares the whole app, not a
viewer.

---

## 13. Revision 4 — corrections after round 3

Round 3 drew one app (`App.dc.html`), and it holds: one shell, six entries and Settings, the places
differing only where §4.5 says. Revision 4 corrects what reviewing it showed.

**Found in review:**

| Problem | Now |
|---|---|
| "Changed" meant two things: the pill's "Code changed" (the docs are behind the code) and What changed (the architecture changed), side by side on the local Home card | The pill says **Out of date**, with its own glyph; "changed" is the architecture's only (§4.3, 6.16) |
| "Since you last looked" meant four things; locally, the last generate against the one before | Locally it is since you last opened it on this machine, remembered in the cache — the same as on the platform; only the published site and hosted public differ (§4.5) |
| The five change groups had no place for a rename, a rewiring or a new technology — round 3 counted "Valkey 8 → 9" with no group for it | A sixth group, **changed**, before removed (§6.2, 6.20) |
| The list of every route, page, command and job — the old Features screen — had lost its home when What it does replaced it | **Every entry point**, a fifth door of Reference (§4.4) |
| Without AI, What it does fell back to route counts | Verb templates from the routes, or absent (§6.6) |
| "Compared with v11" sat among the user's saved views | **Compare** is a mode of the Map; Saved views are only what a person saved (§4.4) |
| Numbers crept back to the front of sentences, rows and the inspector | Rule 18: meaning first, the count second |
| The design invented `archdoc publish` and `archdoc.yaml` | Wrong: publishing is `archdoc export --site`, and no `archdoc.yaml` exists — local settings are read-only (`surface-spec.md` C-3). Corrected in round 4 |
| The five-item limit fit no flow longer than five steps | The limit is for lists; a sequence shows every step, grouped into at most five stages (rule 15) |
| The round's notifications and its four frame files were described in its findings but never saved; the notifications still carry round 1's "4 of 6 found" and ochre | To be drawn in round 4 |

**Taken from the round's findings:**
- The Overview drops "since you last looked"; What changed opens with it as a strip and picker, and the
  menu carries the count (§6.1, §6.2).
- The trust sentence is the Overview's one allowed second sentence (rule 15).
- Plain names: **What it stores** (data), **Where it runs** (deployment), and the components level
  named after the part it opens — "Inside immich-server" (§4.4). "Reference" and "Did it build what I
  asked?" stay.
- A fifth kind of link between repositories, **distribution**; each System map box says what its
  repository is; parallel arrows bundle from five up (§4.6).
- On the published site, corrections in force live in About these docs; plans exist in local
  Settings; Ask is absent on hosted public pages and the published site, as §4.5 always said (§4.3).
- arc42 chapters are pages with "On this page", not a list cut to five (§4.4).

**Settings and places**, settled after round 3 in discussion with the design:
- The places are **three — Local, the Website, the Published site — and the website is signed in or
  signed out** (Pieces, §4.5). "Hosted public" and "hosted private" named the repository; what
  changes the screen is the visitor. Owner and viewer are a permission inside the website.
- **Account settings** hold what belongs to the person — AI usage and cap, agent tokens, notification
  defaults, theme default, billing; **Repository settings** hold only that repository's choices, each
  override linking to the default (§5.5, §5.9). Locally, **This machine** plays Account's part.
- **The far-right slot is "you"**: settings icon locally, avatar signed in, "Sign in with GitHub"
  signed out, nothing published (§4.3). Theme stays in the top bar everywhere; keyboard shortcuts stay
  on `?` and ⋯, not in the avatar menu; **+ Connect** sits beside the avatar on Home and the system
  pages, and Home's own button only in its empty state.

**After a review of round 3** (`App.dc.html`), also settled:
- **Update** is free, offline and one click; updating with AI explanations is a separate step that
  shows what is sent and what it costs. Ask says it uses the model and what a question costs (rule
  19, §4.3, §6.5). Round 3 had one Generate sending to the model, and a silent Ask.
- **Dates and pushes, not v-numbers** on screen (rule 20).
- **One vocabulary** (§4.7, rule 22): Proven · Written by AI · Couldn't be resolved · Not read · Out
  of date → Update · Looks built · Suggested. Round 3 used up to four words for each.
- **Copy never explains the design** (rule 21).
- **The Map**: tabs and double-clicking into a part, both kept; the tabs grouped as zoom and lenses.
- **Search and Ask are one box.**
- **Where am I is said once**: the breadcrumb carries sub-pages; no back links, no place note.

**Still open from the findings:** the Home feed page and component and table detail pages are not
drawn; a "place note" at the foot of the menu ("On this machine · ~/code/immich") is kept as drawn,
to be judged in use; the rule for clustering tables (§10, question 6) is still needed — round 3 shows
the 34-table cluster on the Map.

---

## 14. Revision 5 — the website

The Landing (§5.1) was the app's one page for strangers, and the one place allowed to sell. A
product with a business aim needs more than one such page — how it works, trust, pricing, docs — and
pages strangers find through search, which a single-page app serves badly. Revision 5 moves it out.

- **A static website at the root domain** holds the Landing's content as its home page, and How it
  works, Gallery, Trust, Pricing, Docs, Changelog, Terms and Privacy around it. Its stack and its
  three origins — website, app, generated sites — are in `platform-plan.md` §7, "The website".
- **The app keeps Start** (`/new?repo=…`), which the website's link box opens, with every state the
  Landing had (§5.1). A signed-out visitor at the app's root is sent to the website.
- **The gallery is built with the website**, by the engine at pinned commits — no platform needed
  (§8, §10 question 4).
- **The app now has no page for strangers.** Every app page is for someone using a repository; the
  selling is the website's.
- **The website is designed in a round of its own**, after the app's: same design system, no app
  shell. This round draws Start in place of the Landing (§9).
