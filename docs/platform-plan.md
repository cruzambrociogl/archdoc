# archdoc — Platform plan: the development environment

> **What this document is:** the plan for the step after the freeze — setting up everything needed
> to develop archdoc as a platform. Not the platform itself: the repositories, the engine's public
> API, the stack, and one request going end to end, so feature work has somewhere to land.
>
> **Why it exists:** `hosted-proposal.md` §8 settled the direction on 2026-10-19 — local and
> hosted, a business as an aim, "verifiable, wherever it runs". This is its first concrete step.
>
> **Status:** planned 2026-10-19, not started. The code is frozen for new features until this plan's
> exit criteria (§5) are met; the work below is allowed under the freeze because it restructures and
> sets up, and adds no capability.

---

## 1. The stack

| Layer | Choice | Why |
|---|---|---|
| Backend | **Go** | Imports the engine directly; one language from engine to API |
| Database | **Postgres** | The platform's own data — users, organisations, repositories, runs, jobs, later billing — is relational and needs joins and transactions |
| Flexible data | **`JSONB` columns** | The engine's output (`model.json`, coverage, history) is a document whose shape belongs to the engine and grows with it. Stored whole, queried into when needed, no migration when a field is added |
| Large files | **Object storage** — S3-compatible; SeaweedFS locally | Generated sites and SVGs are files, not rows |
| Driver, queries | **pgx**, **sqlc** | Plain SQL, type-safe Go generated from it; no ORM |
| Migrations | **goose** | Numbered SQL files. Not knowing the schema yet is answered by making schema changes cheap, not by having no schema |
| Job queue | **River** | Runs inside Postgres — no Redis, no extra service. Postgres + object storage is the whole stack |
| Frontend | **React 19 + TypeScript + Vite** — the same base as today's app | Keeps every screen already built; the mainstream choice |
| Routing | **TanStack Router** | Type-safe URLs, once there are organisations, repositories, runs and settings |
| Server data | **TanStack Query** | Caching, retries and live job status without hand-written fetch code |
| Components | **Radix primitives** (shadcn/ui on top where it helps), styled with archdoc's own design system | Accessible menus, dialogs and forms; the look stays the Claude Design project's |
| Forms | **React Hook Form + Zod** | Sign-up, settings, connecting repositories |
| Diagrams | **React Flow** (`@xyflow/react`) — as today | The standard for this |
| Rendering | **A single-page app served as static files**; Go serves the API | No Node server in production. What search engines must find lives on the website, so the app needs no pre-rendering — search does not decide the stack of the whole app |
| Website | **Astro**, with **Starlight** for the docs — static, on a CDN, at the root domain (§7) | Static HTML and no JavaScript by default; renders the app's React components at build time; takes the design system's tokens from the app package |
| Local dev stack | **A Dev Container** on **Docker Compose** — the workspace, Postgres, SeaweedFS, the worker, an egress proxy (step 4) | Docker and an editor are all anyone installs. The engine's "no services" rule is the engine's; the platform is a service |

**Considered and not chosen.** *MongoDB* — the schema would still exist, but in Go code and
unchecked; accounts, permissions and billing are its weakest fit; and one more service to run.
*Vue* — a rewrite of ~5,700 lines and of the diagram layer, for no gain. *Next.js* — a Node server in
production beside Go, to solve search for a handful of pages.

---

## 2. Repositories

| Repository | Visibility | Licence | Holds |
|---|---|---|---|
| `archdoc` — this one, `github.com/cruzambrociogl/archdoc` | **Public** — licensed and tagged in step 3 | **FSL** — free to read, use, modify and self-host; no competing service; each version becomes Apache 2.0 after two years | The engine, its public API, the CLI, the shared app (local, published and the platform's), the GitHub Action later |
| `archdoc-platform` — `github.com/cruzambrociogl/archdoc-platform` | **Private** | Closed | API server, worker, platform web app, migrations, deployment, **the development environment** (step 4) |
| `archdoc-website` — `github.com/cruzambrociogl/archdoc-website` | **Private** | Closed | The website at the root domain (§7): Astro, the gallery, its own deploy |

- **Private depends on public, never the reverse.** The platform and the website build on the
  public engine — `go get`, the app package, the CLI release and its docs need no credentials,
  locally, in the Dev Container or in CI. Nothing public depends on anything private.
- **Three repositories, side by side.** Cloned as siblings in one folder that holds nothing else —
  `archdoc/`, `archdoc-platform/`, `archdoc-website/` (here, `archdoc-workspace/`). The development environment (step 4) mounts
  that folder, so every repository sees the others at `../`.
- **Local development across them:** a Go workspace (`go.work`) in the platform repository points at
  the sibling engine, so the platform builds against local engine changes without a release per
  change. `go.work` is not committed; CI builds against a tagged engine version.
- **Engine versions:** semantic version tags (`v0.x.y`) on the engine; the platform and the website
  each pin one.
- **The app** (step 2) is published from the engine repository as a package the platform installs,
  with its hosted adapter. The website installs it too, for the design tokens and the diagram
  components the gallery renders.

**What flows between them.** Arrows point one way only, and every one carries a version:

```
            archdoc (public)                          archdoc-platform (private)
  ┌──────────────────────────────┐
  │ Go module @ vX.Y.Z           │──────────────────▶ api, worker   → app.archdoc.dev
  │ @archdoc/app (npm)           │─────────┬────────▶ web           → app.archdoc.dev
  │                              │         │          worker output → generated-sites domain
  │                              │         │
  │                              │         │          archdoc-website (private)
  │                              │         └────────▶ tokens, diagram components
  │ CLI binary (release)         │──────────────────▶ build: runs it on the gallery
  │ docs/user/ @ the same tag    │──────────────────▶ build: renders the docs → archdoc.dev
  └──────────────────────────────┘
```

The website and the platform do not depend on each other: the website's link box and gallery cards
point at the app's URLs, nothing more.

- **One engine version per repository.** The platform's `go.mod` and app package name one engine
  tag; the website's app package, CLI and docs name one tag. CI checks each (step 8).
- **The website never runs ahead of the platform.** A gallery card opens the app; the app must be
  able to show what the card shows. The platform takes a new engine version first, the website
  after.
- **The website, locally:** against its pinned release by default; pointed at the sibling
  `../archdoc` when working on the docs or the gallery's look — as `go.work` does for Go.
- The engine never knows the platform or the website exist: nothing in it names them, their APIs or
  their domains.

---

## 3. Steps

Ordered by dependency. Each says what is done when it is done.

### Step 1 — The engine's public API

Go forbids importing `internal/` from another module, and the pipeline lives in `package main`:
`document()` in `cmd/archdoc/generate.go` is ~370 lines, with more logic in `diff.go`, `status.go`,
`show.go` and `export.go`. Nothing in it can be called from outside.

- A public package — the module root, `import "github.com/cruzambrociogl/archdoc"` — with the
  entry points the CLI and the platform both need: `Generate`, `Diff`, `Status`, `ExportSite`.
- Options and results as plain structs; progress reported through a callback or `io.Writer`, never
  printed by the library.
- The public types the API returns: either type aliases of `internal/archdoc` types or the types
  moved out. Decided in the step; aliases are the smaller change.
- The CLI becomes a client of the API and keeps no pipeline logic.
- **`Diff` takes two models, not only two commits.** Today it checks out two revisions; the
  platform holds stored models and no clone. Comparing commits becomes loading two models and
  comparing those.
- **The history store is optional.** A caller can run `Generate` without `.archdoc/history.db`:
  the worker's clone is thrown away, so a SQLite history written there is wasted work. The
  platform keeps its history in `runs` (step 5).

**Done when:** the CLI calls only the public API; `generate` on Immich and Supabase at the pinned
SHAs is byte-identical to before (AC-7); the network-boundary check passes unchanged; an example
program outside the module generates a repository through the API, with no history store; a diff of
two models given directly equals the diff of the commits they came from.

### Step 2 — The shared app

**One app, not a shared viewer** (`platform-surface.md` §4.1, revision 3). The local app, the platform
and the published site are the same application — one shell, one navigation, one set of pages — and
each place adds or lacks a few features. Two copies would drift; so would a shared set of screens
inside two different shells.

- Move `web/` into an **app** package: the shell and every page, with the information architecture
  of `platform-surface.md` §4.
- The app reads its data and its **capabilities** through an interface: what this place can do —
  generate, write corrections, share, sign in — decides which features appear (§4.5 there). It never
  calls `archdoc serve`'s endpoints directly.
- Three adapters: **local** (`archdoc serve`), **published** (the static JSON of `export --site`) and,
  in the platform repository, **hosted** (the platform API).
- An npm workspace in the engine repository; the app published as a package.

**Done when:** the local app runs from the package through the local adapter and works as before;
`export --site` output is unchanged; the app renders a stored model through the published adapter
with no server.

### Step 3 — The engine, licensed and tagged

- `LICENSE` with the FSL; the README states what the licence allows.
- **No review before publishing.** The repository has been public since before this plan, so
  everything in it and its history has already been readable. What should not stay public from
  here on — anything naming people or courses — is removed going forward; its history stays.
- First tagged version, `v0.1.0`.

**Done when:** the repository is licensed and tagged, and `go get` of the tag works from a
clean machine.

### Step 4 — The platform repository and the development environment

```
archdoc-platform/
  .devcontainer/
    devcontainer.json   the workspace: joins compose.yaml as one more service
    Dockerfile          the workspace image — pinned Go, Node, sqlc, goose, task, psql, gh
  cmd/api/              HTTP API
  cmd/worker/           runs jobs: clone, generate, export, store
  internal/             platform packages
  db/migrations/        goose
  db/queries/           sqlc
  web/                  the hosted adapter — imports the app
  Dockerfile            api and worker: a development target (hot reload) and production targets
  compose.yaml          Postgres, SeaweedFS, the worker, the egress proxy
  Taskfile.yml
  .env.example
  CLAUDE.md
```

**A Dev Container, not one container.** The workspace is one container — the editor's server, the
tools, the terminal — and the services stay containers of their own beside it, on one Compose
network. Opening the repository in VS Code ("Reopen in Container") starts them all:

```
 VS Code (host) ──attaches──▶ workspace         the container you work in
                              /workspaces/archdoc
                              /workspaces/archdoc-platform
                              /workspaces/archdoc-website
                              you run here: api (with a debugger), web, the website, tests
                                 │
              ─────────────── compose network ───────────────
              │                  │                 │
          postgres           seaweedfs          worker ──▶ egress proxy ──▶ github.com
                                                non-root, read-only, CPU and memory
                                                limits, no direct internet
```

- **Docker and an editor that supports Dev Containers are all anyone installs.** Go, Node and every
  tool come pinned in the workspace image; no versions to match across machines. On Windows, Docker
  runs on WSL2 and the repositories are cloned inside WSL.
- **The workspace mounts the folder holding all three repositories** (§2), so `go.work`'s
  `../archdoc` works inside it as on the host, and the website runs from the same workspace.
- **The API runs in the workspace's terminal**, not a container of its own — breakpoints and the
  debugger work as on the host.
- **The worker runs in its own container, locked down as in production** (step 6): an unprivileged
  user, a read-only filesystem with one scratch volume for clones, CPU and memory limits, and no
  route to the internet except an **egress proxy that allows GitHub** (Smokescreen or similar). The
  lockdown is exercised on every run, not first met in production. It is built from the same
  `Dockerfile` as production, development target, with hot reload over the mounted source.
- **Postgres, SeaweedFS and the proxy are published images** — no Dockerfile of ours.
- **First start runs `task setup`:** migrations, the storage bucket, and the gallery's repositories
  generated as seed data, so the app has something to show at once.
- **Caches in named volumes** — Go modules, the Go build cache, `node_modules` — so rebuilds stay
  fast. File access through Docker is slower on macOS; cloning the repositories into a Docker volume
  instead of the host's disk fixes it, at the cost of not seeing the files in Finder.
- **Tests use the Compose Postgres**, each from a fresh database cloned from a template — isolated
  and quick, and the same way CI runs them.
- **The same `devcontainer.json` works in GitHub Codespaces**, with the engine and the website granted
  as extra repositories — a browser and nothing installed.
- **The task runner is the only interface** — `task setup`, `migrate`, `gen` (sqlc), `test`, `logs`,
  `reset` — so nobody needs Compose's flags.
- **Git inside the workspace** — open, to settle when this step is built. VS Code forwards the host's
  SSH agent into the container but not `~/.ssh/config`, so a remote written with a host alias
  (`git@github-edu:…`, as on the machine these were set up on) fails inside it. Either the
  workspace mounts the SSH config read-only, or remotes inside it use HTTPS through `gh`.
- **The engine has no Dev Container.** It is developed from this workspace, or natively with plain Go
  and Node as today; its CI keeps building on a bare runner, which is what protects "runs with no
  container runtime" (hard rule 7). The website keeps a small one of its own — Node and the archdoc
  CLI, no services — for work on it alone.
- **Configuration per environment, from the start.** Everything that differs between local, staging
  and production — database URL, storage, the sites origin, keys — comes from the environment;
  `.env.example` names every variable; no secret is ever in code or committed. Adding staging or
  production later is then a new set of values, not a refactor.
- **Observability hooks, from the start.** Structured logs (`log/slog`) with a run ID on every line
  of a job; an error-tracking SDK (Sentry) wired in, off when no DSN is set. Cheap now; retrofitting
  a run ID through every log line later is not.

**Done when:** on a machine with only Docker and VS Code, cloning the three repositories and opening
the platform in its Dev Container gives a running API, worker and seeded database with no other
step; the worker cannot reach any host but GitHub; nothing in the repository is a secret.

### Step 5 — The data foundation

Only the tables the first slice needs. Accounts, organisations and billing wait for piece D.

- `repositories` — owner, name, URL.
- `runs` — repository, commit, status, timings, the model and coverage as `JSONB`, the model's
  fingerprint, the site's object-storage key.
- River's own tables, through its migrations.

Five rules, set now because each is costly to retrofit:

- **`JSONB` does not keep bytes.** Postgres reorders keys and drops whitespace, so what comes back
  is not the file the engine wrote and AC-7 cannot be checked against it. The **fingerprint** is a
  hash of the engine's original bytes, stored beside the `JSONB` — it is what identifies a model and
  what tells two runs apart. The raw `model.json` also goes to object storage, for the exact file.
- **A run that changes nothing records nothing** — the engine's rule (`decisions.md`, 2026-09-10),
  carried over: the platform's versions of a repository are its runs with distinct fingerprints.
- **A run and its job are created in one transaction.** Insert the `runs` row and enqueue the River
  job together — never a run with no job, never a job with no run. This is why the queue lives in
  Postgres; it is the pattern for every job from the first one.
- **Public and private, from the first table.** A public repository's result at a commit is a
  cache shared by everyone who pastes it; a private one will belong to an organisation.
  `repositories` carries its visibility now and runs are reached through it, so organisations later
  add a column and a filter, not a re-keying of `runs`.
- **The `JSONB` is the truth; tables derived from it come only when a query needs them.** Impact
  ("who reads this table") and system links (one repository's outside services matched to another's
  declared ones) query inside the model. A GIN index may be enough; if not, projection tables —
  elements, edges, citations — rebuilt from the `JSONB`, never edited directly.

**Done when:** migrations apply up and down cleanly; sqlc generates without diff in CI; a stored
model's fingerprint matches the hash of the bytes the engine wrote.

### Step 6 — One request, end to end

The walking skeleton: proof that every layer connects, not a feature.

1. `POST /runs` with a public GitHub URL.
2. A River job: the worker shallow-clones at the default branch's commit into a throwaway
   directory, calls the engine's `Generate` and `ExportSite`, uploads the site to object storage,
   stores the model, and **deletes the clone, on success or failure**.
3. `GET /runs/{id}` reports status; when done, the app shows the result.
4. A second request for the same `repo@commit` is served from what is stored (AC-7 makes that safe).

**The worker is locked down from the first job.** It clones strangers' repositories, and git itself
is an attack surface:
- shallow clone, no submodules, no LFS (`GIT_LFS_SKIP_SMUDGE=1`), no git hooks or config from the
  repository honoured;
- a maximum repository size, checked before cloning where GitHub reports it and enforced during;
- a time limit per job, and CPU and memory limits on the worker's container;
- runs as an unprivileged user, and reaches only GitHub, through the egress proxy (step 4) — the engine itself needs
  nothing else unless `label`/`explain` are on, which they are not in this phase.

**Done when:** pasting Immich's URL in a local environment shows its documentation, and the clone is
gone afterwards; a repository over the size limit, and one that exceeds the time limit, each fail
with a clear message and leave nothing behind.

### Step 7 — The platform web app

- Vite + React + TypeScript, TanStack Router and Query, Radix, the design system's tokens.
- Three screens only: **Start** — paste a link, the page the website's link box will open
  (`/new?repo=…`); a run in progress; the result, in the shared app. The website itself is not part
  of this phase (§7).
- **Generated sites are served from a separate origin** — a different registrable domain in
  production (as GitHub serves raw files from `githubusercontent.com`), a different host and port
  locally. A site carries text from a stranger's repository; on the app's own origin a malicious
  repository could reach users' sessions. The app embeds or links the site; it never serves it.
- **Readable before complete** (`hosted-proposal.md` §8): every screen answers one question in plain
  words first, with the evidence one step behind. This is set from the first screen, not retrofitted.

**Done when:** the step 6 flow works through the browser, not only through the API.

### Step 8 — Continuous integration

- **Engine:** as today — gofmt, vet, build, race tests, the network boundary — plus the API example
  and the app build.
- **Platform:** gofmt, vet, tests against a Postgres service, migrations up and down, sqlc diff,
  web typecheck and build, the workspace image and the production images build; **one engine
  version** — `go.mod` and the app package name the same engine tag (§2).
- **Website:** build, links checked; **one engine version** — the app package, the CLI and the docs
  name the same engine tag (§2).

**Done when:** all three are green on `develop` and block a broken pull request.

### Step 9 — Project context

- The platform and website repositories each get their own `CLAUDE.md`, `PROGRESS.md` and
  `docs/decisions.md`, small from the start — it must not inherit the engine's 4,750 lines of documents.
- The engine's `CLAUDE.md` and `PROGRESS.md` describe the engine as one part of a larger product.
- `product-definition.md` §7 and `vision.md` §2.2 amended, as the 2026-10-19 decision requires.

**Done when:** a new session in any of the three repositories orients itself from its own context
file alone.

---

## 4. Out of scope for this phase

Accounts and sign-in, organisations, private repositories, the GitHub App, billing, AI on the hosted
path, deployment to a public host, and every S- feature of `hosted-proposal.md` §7. Each needs this
environment first; none of it is needed to have it.

---

## 5. Exit criteria — when the freeze lifts

1. The CLI runs on the engine's public API, with output byte-identical to before.
2. The local app, the published site and the platform are one shared app, each through its adapter.
3. The engine repository is public, FSL-licensed and tagged.
4. On a machine with only Docker and VS Code, the platform's Dev Container runs the whole stack with
   no other step.
5. Pasting a public repository's URL in the local platform shows its documentation, end to end —
   generated by a locked-down worker, served from a separate origin.
6. CI is green and enforcing in all three repositories.
7. Each repository has its own context, and the definition and vision are amended.

---

## 6. Open questions

1. **Public types** — aliases of `internal/archdoc`, or moved out (step 1)?
2. **The app package** — published to npm, to GitHub Packages, or consumed from git (step 2)?
3. ~~What stays out of the public repository~~ — moot: it is already public (step 3). What to remove
   going forward is decided in step 3.
4. **The product's name and domain** — does the platform stay "archdoc"? Three origins need it: the
   website, the app, and the generated sites on a domain of their own (§7).
5. **Hosting** — Fly.io, Render, Railway, a cloud provider. Not needed until deployment, which is
   outside this phase; worth knowing early because it shapes object storage and Postgres choices.
6. **Authentication** — recommended: sign-in through the GitHub App, WorkOS when companies ask for
   SSO (§7). Outside this phase; confirmed when piece D starts.

---

## 7. Beyond this phase — what the platform needs

> Added 2026-10-19. Not part of the development environment; each is taken up when the piece that
> needs it is built, and gets its own decision then. Listed now so nothing is discovered late.

### Authentication and permissions

**Recommended: sign-in through the GitHub App, sessions in Postgres.** Every user connects GitHub
anyway — private repositories (D) need a GitHub App — and the App can also sign users in. One flow
then answers both questions:

- **who is this** — the GitHub identity;
- **what may they see** — GitHub already knows which repositories each user can read, so archdoc's
  permissions **mirror GitHub's**: read the repository, read its documentation. No permission
  system of our own.

**WorkOS** when companies ask for single sign-on — it exists for exactly that. **Clerk** if sign-up
without GitHub is ever wanted.

*Considered: Firebase Authentication* — a second identity system beside the GitHub App that is
needed regardless; users stored in Google's database rather than ours; enterprise SSO only on the
paid Identity Platform upgrade.

### The website

Added 2026-10-20. A static site at the root domain: what archdoc is, how it works, the gallery, the
docs. It replaces the app's Landing (`platform-surface.md` §5.1) — the app no longer has a public
front page of its own.

**Three origins** (`archdoc.dev` is a placeholder — open question 4):

| Origin | Serves | Built with |
|---|---|---|
| `archdoc.dev` | **The website** — pitch, how it works, gallery, trust, docs, pricing, legal | Astro, static, on a CDN |
| `app.archdoc.dev` | **The app** and its API | Go and the shared app |
| A domain of its own, e.g. `archdoc-sites.dev` | **Generated sites** from strangers' repositories | Object storage behind a CDN (step 7) |

**What it settles:**
- **The app stays a single-page app with no exceptions.** Search engines find the website; nothing
  in the app is pre-rendered.
- **The link box is a plain form** to `app.archdoc.dev/new?repo=…` — the app's Start page (step 7),
  which handles every state: not a GitHub link, private (→ sign in), already generated (→ Overview).
  A signed-out visitor at the app's root is sent to the website.
- **The gallery needs no platform.** The engine is deterministic and has a CLI: the website's build
  runs `archdoc generate` on the gallery's repositories at pinned commits and renders the cards and
  pages from the output. The website can go live before the platform does, as the front door of
  the local tool.

**Pages:**
- **Home** — one sentence, the link box, the gallery, and the three ways to use it: local, the
  GitHub Action, hosted.
- **How it works** — the pipeline in plain words; what proven, interpreted and unresolved mean.
- **Gallery** — a page per repository: the plain summary, the diagram, "open in archdoc".
- **Trust** — what leaves your machine, what the platform keeps, the never-keep-your-code promise
  (`hosted-proposal.md` question 6). For this product a selling page, not a legal one.
- **Pricing** — once decided (`hosted-proposal.md` question 8).
- **Docs** — install, the CLI, `rules.yaml`, the Action, self-hosting.
- **Changelog**; **Terms** and **Privacy** (below, "Legal"); a link to the status page.

**Where it lives:** its own private repository, `archdoc-website` (§2) — it depends on the engine
and on nothing of the platform's, and deploys on its own. **The docs'
text lives in the engine repository** (`docs/user/`), versioned with the code it describes; the
website's build pulls it in from the engine version it pins. Written in the website instead, it
would drift from the engine with every release.

**Readable before complete** (`hosted-proposal.md` §8) holds here too, and this is the one place
allowed to sell — calmly; the gallery does the selling. Designed in Claude Design, in a round of its
own, with the same design system as the app.

### For C — hosted, public repositories

| Need | What | Notes |
|---|---|---|
| Hosting | dev, staging, production environments; domain; TLS | Shapes the Postgres and object-storage choices — open question 5 |
| A CDN for generated sites | In front of the sites origin | Sites are static and keyed by commit: ideal to cache forever |
| Rate limiting and abuse protection | Per IP on "paste a link"; a cap on queued jobs; a blocklist | From the first public day |
| Uptime alerts | On the API, the worker queue's depth, the sites origin | |
| Backups | Postgres, with a tested restore | An untested backup is not one |
| Storage lifecycle | Expire sites of old commits nobody has opened | Determinism means any can be regenerated |
| Legal | Terms of Service, Privacy Policy | GitHub requires a privacy policy before listing a GitHub App |

### For D — hosted, private repositories

| Need | What | Notes |
|---|---|---|
| The GitHub App | Webhooks on push and pull request, installation tokens, PR comments | Also the sign-in, above |
| Payments | **Stripe** — subscriptions, metering if AI usage is charged | Pricing is still open (`hosted-proposal.md` question 8) |
| Transactional email | **Resend** or **Postmark** — invitations, drift alerts, receipts | |
| AI cost controls | Quotas per organisation, prompt caching, a hard spending cap | `explain` costs per component |
| Secrets management | The GitHub App's private key, Stripe and Anthropic keys | The hosting provider's secret store, or Doppler |
| Data deletion | "Delete my organisation" deletes everything, objects included | Companies ask before connecting private code |
| Encryption at rest | Database and objects | Stored structure is sensitive: hostnames, routes, tables |
| A data processing agreement | For companies | Legal, alongside the terms |

### For E — self-hosted

| Need | What |
|---|---|
| Packaging | A container image and a Helm chart; nothing assuming our infrastructure |
| Licence keys | What a customer is entitled to run |
| An update channel | How a self-hosted install learns of and applies a new version |

### Worth having early, not blocking

| Need | What | Why |
|---|---|---|
| Product analytics | **PostHog** or **Plausible** — what people paste, where they give up | The only way to know whether "readable before complete" works |
| Admin view of jobs | River's own UI | Retry, inspect, cancel without a database console |
| Feature flags | To ship a feature to a few users first | |
| Status page | Public, for incidents | |
| Public docs site | How to use archdoc, what it reads | |
| Support channel | Email or a community space | |
