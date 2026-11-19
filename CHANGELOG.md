# Changelog

What changed in archdoc, per tagged version, for those who use it. Newest first. Decisions and their
reasons are in `docs/decisions.md`; what is done against the plan is in `PROGRESS.md`; this file says
what is different for a user.

## Unreleased

- **Go is read.** Each program in a Go module — `cmd/api`, `cmd/worker` — is a container of its own,
  with its packages as its parts and what uses what between them. Its HTTP routes (net/http, chi,
  gin, echo, fiber, gorilla/mux), its commands (cobra, urfave/cli, or its own table of them) and its
  River jobs are read from the code, each at its line, a handler's doc comment as its summary.
  Each has its flow: the program's calls, followed by the types the code states, to the tables its
  SQL names (sqlc's queries included), the jobs it queues and the requests it sends.
- **Test fixtures are left alone.** Nothing under `testdata/` is taken for part of the system.

- **Security updates.** `golang.org/x/image` 0.43, for six advisories in the image and font decoders
  archdoc's diagrams use; archdoc now builds with Go 1.27.4, for an advisory in `net/http`. CI runs
  `govulncheck` and a secret scan on every push.

- **Publishing from your own CI.** The Action's `mode: generate` keeps the record of a checkout as an
  artifact of the run; `mode: publish`, in a job of its own, sends it to a project on archdoc's
  website with a token GitHub makes for it. GitLab CI/CD and Bitbucket Pipelines do the same
  (`ci/`). The record is exactly what is sent, and stays on the run for anyone to read.
- **A private project's documentation names files, never lines,** says Private in its bar, and has
  no Share. Where a place has accounts, the bar ends in Sign in or the person's menu.

- **The app answers questions first.** One shell for the local app and published sites: Overview,
  What changed, Map, What it does, Reference and Settings, each named in plain words. The Overview
  says what the system is in one paragraph — the repository's own description where its manifest
  has one, then how its parts call each other — with one picture and one sentence on how far to trust
  it. What changed tells each version in a sentence, grouped by what the changes mean. What it does
  turns routes into sentences. Select any part to see what it touches. Old links still open.
- **Manifest descriptions are quoted.** A `description` in package.json, pyproject.toml or
  pubspec.yaml becomes the part's description, and the root manifest's the system's — cited at its
  line, as read, never interpreted.
- **Other people's text can't run in the app.** Every document's Markdown is sanitised before it is
  shown, and the app runs under a Content-Security-Policy — sent by `archdoc serve`, and written into
  every published site's page — that allows only its own scripts.
- **Published sites show the documents' diagrams.** An exported site now carries the SVGs its
  generated documents name; before, the documents rendered with broken images.
- **The app says what went wrong, and what still works.** When `archdoc serve` stops, a notice says
  so and the page reconnects by itself; a version recorded from a terminal while the app is open is
  announced, never swapped in; a link to a page, version or part that is gone says where it went.
  The Overview's picture can be read as a list, and reduced motion is honoured.
- **A folder of repositories is one system.** `archdoc serve` on a folder holding several documented
  repositories opens the System map: how they fit together, in a sentence and a picture — each link
  one repository's requirement of another's package, cited at both ends — and how far a pinned
  commit or tag is behind. Each repository's pages open from it. Go modules' requirements are read.
- **Just `archdoc`.** In a project's folder — or any folder inside it — a bare `archdoc` documents the
  repository, free and offline, and opens it in the browser. It writes only `.archdoc/` until you ask
  for the documents (Documents › Write the documents, or `archdoc generate`); once they are there,
  every run keeps them current. In a folder of repositories it documents each one; in your home
  folder it documents nothing and opens the library.
- **One archdoc per machine.** A second `archdoc`, in another project, opens that project in the one
  already running instead of failing on the port. `archdoc restart` stops it and starts it again in
  the terminal you are in — on the same address, so open pages keep working — which is how a newly
  set `ANTHROPIC_API_KEY` is picked up.
- **The library: `archdoc library`.** Every repository, and folder of them, this machine has
  documented or served, with what changed in each since you last opened it here, what couldn't be
  resolved, and when it last changed — shown with one project too, and first in every project's
  breadcrumb. The list is a file in your user configuration folder, never in a repository.
  `archdoc home` is now `archdoc library`.
- **Enrich with AI.** The app's paid update is named for what it does; it still shows what it would
  send and cost first.
- **What changed across a system.** Beside its map, a system tells each repository's latest version
  in the sentence that repository's own What changed writes, the most recent first, and says which of
  the others require it.
- **Ask, and "Did it build what I asked?"** In the local app, on your own Anthropic key. A question in
  the search box is answered only from what archdoc read, every sentence cited; name your plan files
  and each requirement is read against the code — looks built, looks missing, couldn't check — with
  what was built that the plan didn't ask for. Both say what they send and roughly what it costs
  before sending, and remember answers so asking again costs nothing.
- **One update at a time.** `archdoc generate` holds the repository while it writes: a second run
  waits and says for whom, and the app says an update is running in the terminal, with what it is
  doing. A Map link to a part a later version removed says where it went.
- **Your coding agent can ask archdoc.** `archdoc mcp` answers Claude Code, Cursor or any MCP client
  from the documentation: what the system is, one element, route or file with its proof, every
  route or table, what a change to a part or a class touches, what changed, and whether the
  documentation is current. Every answer cited; nothing written, nothing sent.
  `claude mcp add archdoc -- archdoc mcp /path/to/repository` connects it.
  A skill, `skills/archdoc/SKILL.md`, tells the agent when to ask: what a part touches before
  changing it, and what the change did to the architecture after.
- **The archdoc Action.** `uses: cruzambrociogl/archdoc@develop` in a workflow: on a pull request,
  what it changes in the architecture, on the run's summary and as one comment kept up to date;
  `check` fails when the committed documentation no longer matches the code; `site` builds the
  site for GitHub Pages. Offline, like everything but `label` and `explain`.
- **The search box says when AI can answer.** Where Ask is available — the local app — the box reads
  "Search, or ask a question…" with a ◇ AI mark, and opening it suggests a question to try. A
  two-word question ending in "?" is offered to Ask too.
- **Arrange the Map anywhere, with undo.** Drag a box, or a boundary by its name, on a published site
  too; only `archdoc serve` saves to `layout.yaml`, and elsewhere a move stays on your screen. Undo
  and redo each move (⌘Z, ⇧⌘Z, or the buttons by the zoom), and the padlock keeps every box where it
  is. Dragging the empty canvas inside a boundary pans again, instead of moving the boundary.
- **Links to the repository open in a new tab.** A citation on a published site opens the file at
  its line on GitHub without leaving the docs — and so does every link out of the app, in the
  documents too.
- **The Overview shows how the parts fit.** Its picture puts the parts in columns by who calls whom
  — what people use, what they call, where data is kept — each opening on the Map, with an arrow only
  where a call between two columns is proven. Before, it was a grid of names.
- **The Map's boxes say their kind at a glance.** Each icon sits on a tile of its kind's colour. The
  count of calls archdoc couldn't resolve, in a box's corner, no longer changes with the theme: the
  diagram is the same drawing in light and dark.
- **The archdoc wordmark goes home**: to the library, a system's map, or the website's start — wherever is
  above the repository you are reading.
- **Update from the app, with or without AI.** On this machine the top bar says *Out of date* when
  another commit is checked out than the documentation was made from — the library says it on each card too
  — and its menu updates it: **Update**, free and offline, as `archdoc generate` does; and **Enrich
  with AI**, which first says what labels and explanations would send and roughly cost, each to tick
  or not, and then sends no more requests, and costs no more than a little over, what you agreed to.
  It needs `ANTHROPIC_API_KEY` in the shell archdoc started from (`archdoc restart` picks up a new one). One update runs at a time,
  from here or a terminal; a band above the page follows it and says how it ended.
- **`GenerateOptions.DryRun`**, in the library: every paid run's requests — label's, then explain's,
  from one reading — are put to its Consent, and the run stops: nothing sent, nothing written, and the
  repository not held.
- **archdoc is a Go library as well as a command.** `import "github.com/cruzambrociogl/archdoc"`:
  `Generate` documents a repository, `Compare` and `Diff` say what changed — between two stored
  models or two commits — `Status`, `BuildSite`, `Read`, `Find` and `OpenHistory` do what their
  commands do; `Fingerprint` says whether two models are one version, as history decides it, for a
  program that keeps models itself. The command is the API's first client and prints exactly what it
  printed before.
  `examples/library` shows a program of its own using it.

No version tagged yet. The first, `v0.1.0`, comes with the licence. Until then the state of the
engine is described in `PROGRESS.md`, "Where this stands".
