# The web app

The surface people see: `archdoc serve`'s live app, and the published site `archdoc export --site`
builds from the same code. What it shows and why is specified in
[`../docs/surface-spec.md`](../docs/surface-spec.md); this file is how it is built.

React 19, TypeScript, Vite, React Flow for the diagrams. Built into `web/host/dist` and embedded in
the binary, so users install nothing.

**One app, several places** (platform-plan step 2). An npm workspace of two packages:

- **`app/` — `@archdoc/app`**: the shell and every page, the styles, the data types. It reads every
  value through a **Source** it is given (`app/src/source.ts`) and never asks where it runs.
- **`host/`** — the build this binary embeds: `index.html`, the Vite config, and the two Sources the
  engine ships — **local** (`archdoc serve`) and **published** (`export --site`). The archdoc
  website gives the same app a third, **hosted**, from the platform repository.

## Running it

From `web/`:

```console
npm ci                       # once, for both packages
npm run build                # typecheck both, then build into host/dist/ — what the binary embeds
npm run typecheck            # typecheck only
```

While working on the app, run Vite and a dev build of archdoc side by side — the dev build proxies
the page to Vite, so a change shows on reload without rebuilding Go:

```console
npm run dev                                          # Vite on 127.0.0.1:5173
go build -tags dev -o archdoc ../cmd/archdoc         # from web/, or adjust the path
./archdoc serve ../path/to/a/documented/repository   # http://localhost:7474
```

`../subjects/mastodon` is the small case and `../subjects/immich` the one with code read — components,
tables, routes, flows; for scale, generate a repository of a few hundred Compose services (the
300-service check in `docs/decisions.md`, 5 Oct, was one).

## Four rules

1. **The app computes nothing the engine did not store.** Every fact, position and route on screen
   came from the API. Counting, filtering and dimming are presentation; deciding what exists, how
   it is laid out or what it means is not. A diagram is drawn from the engine's *scene* — the view
   plus its stored layout — so the app and the committed SVG show the same arrangement.
2. **Every visible fact keeps its way back to its evidence.** Use `<Cite>` for any value with a
   provenance; mark model-written values with `<TruthMark state="interpreted">` and the
   `interpreted` class. A provenance with no file is "the element's own line": fall back to the
   element's provenance (see `cited()` in `screens/Inspector.tsx`).
3. **Every view state lives in the URL** (`route.ts`), as a path in the hash — `#/map?level=container`
   — kept by TanStack Router, so one build works at any address. Each screen has a path of the
   information architecture (`docs/platform-surface.md` §4): `/overview`, `/changes`, `/map`, `/does`,
   `/reference/…`, `/about/…`, `/settings`; screens move with `go({ screen })` and never see the
   router. An old `#screen=…` link opens where its screen lives now. Version, Map level, selection,
   search, focus, compared version, open document and open run are the search. Copying the address
   reproduces the view; saved views are simply named addresses. A level is `context`, `container`,
   `component:<container id>` or `data:<container id>` (`componentLevel`, `dataLevel` in `api.ts`);
   with none, the explorer opens on the context.
4. **Every read and write goes through the Source.** Read with `useApi(path)`; write with
   `useSource().act(path, body)` — locally it sends the session token the server checks with the
   Origin (`internal/serve/guard.go`). Never `fetch` directly, and never ask which place this is:
   ask `useCapabilities()` whether the place can do the thing.

## Places and capabilities

The same app runs in three places (`docs/platform-surface.md` §4.5). Each host gives it a Source:
`get` and `act` by the paths of archdoc serve's API, `url` for a link to one, `link` for where a
citation opens, and `capabilities` — what the place can do. A feature appears only where its
capability is true; no screen checks which place it is in.

| | Local | Published | Hosted (the website) |
|---|---|---|---|
| Source | `host/src/adapters/local.ts` | `host/src/adapters/published.ts` | in `archdoc-platform` |
| Data | `/api/…` from archdoc serve | `data/…` — `staticName()` (`app/src/staticName.ts`) mirrors `serve.StaticName` in Go; change one, change both | the same `data/…` files, through the platform's API |
| `history` — versions to pick | yes | no: the latest and its baseline | yes |
| `edit` — save views, arrange | yes | no | (decided with the platform) |
| `runDetails` — requests as sent | yes | no: they stay on the machine that sent them | — |
| `plans` — the planned placeholders | in a development build only (§4.4) | no | — |
| `links` — where citations open | `editor`: `vscode://` at the line | `host`: the repository host at the commit, or plain text when none is known | `host` |

`export --site` writes `<meta name="archdoc-mode" content="published">` into `index.html`;
`host/src/main.tsx` reads it to choose the Source. A path the published site does not export simply
fails; `serve.Export` lists what it exports.

**A new place** is a new Source, not a branch in a screen. **A new capability** is a field in
`Capabilities`, set by every Source, and asked for by the screens that need it.

## Where things are

```
app/src/                 @archdoc/app
  index.ts               the package: App, SourceProvider, the Source and data types, the styles
                         (`@archdoc/app/styles/*.css` also alone, for a page that should look like the app
                         without being it)
  source.ts              Source and Capabilities — the contract every place implements
  App.tsx                the shell and which screen to show
  api.ts                 the data types; useApi; citation links through the Source
  route.ts               URL state: each screen's path, title and menu section; the Map levels' plain names
  theme.ts               light / dark / system, per viewer
  shell/                 TopBar (breadcrumb, status, search, Share, ⋯), Nav (the six entries and Settings),
                         Rail (the right rail; the first-visit band), Palette (⌘K), Shortcuts (?)
  screens/               Overview, Changes, Features (What it does), Reference, About (with Coverage), Documents,
                         Dependencies, Component, Corrections (Settings), NetworkRuns, Inspector
  screens/explorer/      Explorer, scene.ts (scene → React Flow; the draft preview; the change overlay),
                         parts.tsx (element, table, boundary, routed edge, tags, ghosts), Legend, SaveView
  ui/                    Cite, TruthMark/TruthChip, markInfo (the four marks, what each means), KindTile,
                         kinds (icons and hues), Sequence (a flow)
  styles/                tokens, fonts, base, shell, ui, inspector, explorer, page
host/                    the build the binary embeds
  index.html, vite.config.ts
  src/main.tsx           chooses the Source — published if index.html says so, else local
  src/adapters/          local.ts, published.ts, read.ts
```

`app/src/index.ts` exports the app before importing the styles: React Flow's base styles, which the
app's modules import, must precede the app's own. The order is the cascade.

## The design system

The design lives in a Claude Design project (`Surface Foundations`, `Surface Screens`,
`Surface Controls`, `Surface Themes`); `styles/tokens.css` is its tokens. Four channels, never
shared:

| Channel | Carried by | Never |
|---|---|---|
| Kind | hue — slate, teal, green, violet, grey | magenta, ochre |
| Truth state | form — solid square, slate diamond and italic, dotted ochre with ? | colour alone |
| Change | stroke weight and a corner tag (+, ±, −; hollow for words only) | a hue |
| Citation, link, selection | magenta | anything else |

Chrome tokens (`--page`, `--ink`, `--accent`, …) invert in dark mode. Diagram tokens (`--k-*`,
`--canvas-*`, `--canvas-accent`) do not: the canvas stays paper-white, because it is the drawing
that gets exported. Fonts come from Fontsource and are bundled — nothing is fetched at view time.

## Keep in step with Go

- `screens/explorer/scene.ts` `arrangeDraft()` is a port of `internal/arrange/arrange.go`
  `Apply()`: the preview while dragging must be what the engine draws after saving.
- `app/src/staticName.ts` `staticName()` mirrors `internal/serve/export.go` `StaticName()`; the published and hosted adapters both use it.
- The response types in `app/src/api.ts` mirror the handlers in `internal/serve`.
- `screens/Features.tsx` `groupOf()` mirrors `core.Entry.Group()`: the app and
  `features.generated.md` group routes and pages alike.
- `screens/explorer/parts.tsx` `TABLE_ROWS` and the table box's 28px header and 13px rows mirror
  `internal/render/layout.go` (`tableRows`, `boxHeight`): the engine sized the box for that text.

## Checking a change

`npm run build` typechecks. Beyond that, look at it: every screen in light and dark, at desktop and
phone width, against Mastodon and against a large repository — and the published site, served from
a static server under a subpath. The 4–5 Oct work was checked with headless Chrome driven by
`puppeteer-core`; those scripts are not kept in the repository.
