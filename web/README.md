# The web app

The surface people see: `archdoc serve`'s live app, and the published site `archdoc export --site`
builds from the same code. What it shows and why is specified in
[`../docs/surface-spec.md`](../docs/surface-spec.md); this file is how it is built.

React 19, TypeScript, Vite, React Flow for the diagrams. Built into `web/dist` and embedded in the
binary, so users install nothing.

## Running it

```console
npm ci                       # once
npm run build                # typecheck, then build into dist/ — what the binary embeds
npm run typecheck            # typecheck only
```

While working on the app, run Vite and a dev build of archdoc side by side — the dev build proxies
the page to Vite, so a change shows on reload without rebuilding Go:

```console
npm run dev                                          # Vite on :5173
go build -tags dev -o archdoc ../cmd/archdoc         # from web/, or adjust the path
./archdoc serve ../path/to/a/documented/repository   # http://localhost:7474
```

`../subjects/mastodon` is the small case; for scale, generate a repository of a few hundred Compose
services (the 300-service check in `docs/decisions.md`, 5 Oct, was one).

## Four rules

1. **The app computes nothing the engine did not store.** Every fact, position and route on screen
   came from the API. Counting, filtering and dimming are presentation; deciding what exists, how
   it is laid out or what it means is not. A diagram is drawn from the engine's *scene* — the view
   plus its stored layout — so the app and the committed SVG show the same arrangement.
2. **Every visible fact keeps its way back to its evidence.** Use `<Cite>` for any value with a
   provenance; mark model-written values with `<TruthMark state="interpreted">` and the
   `interpreted` class. A provenance with no file is "the element's own line": fall back to the
   element's provenance (see `cited()` in `screens/Inspector.tsx`).
3. **Every view state lives in the URL** (`route.ts`). Screen, version, explorer level, selection,
   search, focus, compared version, open document, open run. Copying the address reproduces the
   view; saved views are simply named addresses.
4. **Anything that writes goes through `action()`** (`api.ts`), which sends the session token the
   server checks with the Origin (`internal/serve/guard.go`). Never `fetch` a write directly.

## Two modes

The same build runs live and published (`docs/surface-spec.md` §3). `export --site` writes a
`<meta name="archdoc-mode" content="published">` into `index.html`; `api.ts` reads it into
`published`.

| | Live | Published |
|---|---|---|
| Data | `/api/…` | `data/…` — `staticName()` mirrors `serve.StaticName` in Go; change one, change both |
| Citations | `vscode://` at the line | the repository host at the commit, or plain text when no host is known |
| Actions | arrange, save views (more to come — §11) | none |
| Version | any stored version | the latest and its baseline |

A screen that offers an action checks `published` (or takes an `editable` prop) and hides it. A
path the published site does not export simply fails; `serve.Export` lists what it exports.

## Where things are

```
src/
  main.tsx, App.tsx      entry; the shell and which screen to show
  api.ts                 types, useApi, action(), live/published, citation links
  route.ts               URL state
  theme.ts               light / dark / system, per viewer
  shell/                 TopBar, Nav, Palette (⌘K), Shortcuts (?)
  screens/               Overview, Changes, Documents, Coverage, Corrections, NetworkRuns, Inspector
  screens/explorer/      Explorer, scene.ts (scene → React Flow; the draft preview; the change overlay),
                         parts.tsx (element, boundary, routed edge, tags, ghosts), Legend, SaveView
  ui/                    Cite, TruthMark/TruthChip, KindTile, kinds (icons and hues)
  styles/                tokens, fonts, base, shell, ui, inspector, explorer, page
```

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
- `api.ts` `staticName()` mirrors `internal/serve/export.go` `StaticName()`.
- The response types in `api.ts` mirror the handlers in `internal/serve`.

## Checking a change

`npm run build` typechecks. Beyond that, look at it: every screen in light and dark, at desktop and
phone width, against Mastodon and against a large repository — and the published site, served from
a static server under a subpath. The 4–5 Oct work was checked with headless Chrome driven by
`puppeteer-core`; those scripts are not kept in the repository.
