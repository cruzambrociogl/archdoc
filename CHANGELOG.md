# Changelog

What changed in archdoc, per tagged version, for those who use it. Newest first. Decisions and their
reasons are in `docs/decisions.md`; what is done against the plan is in `PROGRESS.md`; this file says
what is different for a user.

## Unreleased

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
- **Home: `archdoc home`.** Every repository, and folder of them, this machine has documented or
  served, with what changed in each since you last opened it here, what couldn't be resolved, and
  when it last changed. The list is a file in your user configuration folder, never in a repository;
  `generate` and `serve` add to it. With one place there is no Home: it opens that place.
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
- **Arrange the Map anywhere, with undo.** Drag a box, or a boundary by its name, on a published site
  too; only `archdoc serve` saves to `layout.yaml`, and elsewhere a move stays on your screen. Undo
  and redo each move (⌘Z, ⇧⌘Z, or the buttons by the zoom), and the padlock keeps every box where it
  is. Dragging the empty canvas inside a boundary pans again, instead of moving the boundary.
- **Links to the repository open in a new tab.** A citation on a published site opens the file at
  its line on GitHub without leaving the docs — and so does every link out of the app, in the
  documents too.
- **archdoc is a Go library as well as a command.** `import "github.com/cruzambrociogl/archdoc"`:
  `Generate` documents a repository, `Compare` and `Diff` say what changed — between two stored
  models or two commits — `Status`, `BuildSite`, `Read`, `Find` and `OpenHistory` do what their
  commands do; `Fingerprint` says whether two models are one version, as history decides it, for a
  program that keeps models itself. The command is the API's first client and prints exactly what it
  printed before.
  `examples/library` shows a program of its own using it.

No version tagged yet. The first, `v0.1.0`, comes with the licence. Until then the state of the
engine is described in `PROGRESS.md`, "Where this stands".
