// What the surface spec designs and the app does not do yet (surface-spec §4, §5, §11). One list:
// the placeholders on screen read it, and the Not built yet page shows all of it, so nothing that
// was designed is forgotten because nothing on screen points at it. Each entry names the spec
// section that describes it. When one is built, its entry goes and its placeholder with it.

export interface Plan {
  id: string
  area: string
  title: string
  /** What it will do, in a sentence. */
  what: string
  /** Where it is specified. */
  spec: string
  /** What it waits for, when that is more than building it. */
  waits?: string
  /** The Claude Design file that draws it, where one does. */
  design?: string
  /** Set when it has no placeholder on screen — a hover, a style, a badge — so this list is its only mark. */
  listOnly?: boolean
}

const S = 'Surface Screens'
const C = 'Surface Controls'
const F = 'Surface Foundations'
const T = 'Surface Themes'


export const planned: Plan[] = [
  // Control — §11
  { id: 'freshness', design: C, area: 'Control', title: 'Freshness pill', spec: '§11.2, §11.7',
    what: 'Says in the top bar whether the code changed since this version was generated: Up to date, Code changed, Running…, Failed.' },
  { id: 'run', design: C, area: 'Control', title: 'Run from the app', spec: '§11.2 (C-1), §11.7',
    what: 'scan and generate with one click; generate with the model only after a dialog showing egress mode, model, items, bytes and estimated cost. Progress drawer with stages and cancel; one run at a time.' },
  { id: 'publish', design: C, area: 'Control', title: 'Publish dialog', spec: '§11.2, §11.7',
    what: 'Builds the site into .archdoc/site and shows the CI workflow to copy. Never deploys.' },
  { id: 'git', design: C, area: 'Control', title: 'Uncommitted files card', spec: '§11.2 (C-4), §11.7',
    what: "Which of archdoc's own files are uncommitted, their diff against the last commit, and the command to commit them. Read-only." },
  { id: 'settings', design: C, area: 'Control', title: 'Settings panel', spec: '§11.2 (C-3), §11.7',
    what: 'Egress mode, model, provider and effort, read-only. The API key is never entered in the browser.' },
  { id: 'watch', listOnly: true, area: 'Control', title: 'Watch mode', spec: '§11.2, F-29',
    what: 'Regenerate on save, deterministic only: never calls a model on its own.' },

  // Navigation and shell — §4
  { id: 'deployment', design: S, area: 'Architecture', title: 'Deployment view', spec: '§5.9',
    what: 'Where and how it runs: image, published ports, networks and mounts per container, as a diagram and a table. Today it is only the arc42 §7 Markdown section.' },
  { id: 'flows', design: S, area: 'Architecture', title: 'Flows index', spec: '§5.6',
    what: 'Every traced flow on a page of its own, searchable and grouped by feature. Today a flow is shown only in its row on Features.' },
  { id: 'step-through', design: S, area: 'Architecture', title: 'Step through a flow', spec: '§5.6',
    what: 'Advance a sequence diagram one step at a time, the step highlighted with its line.' },
  { id: 'flow-on-canvas', design: S, area: 'Architecture', title: 'A flow over the diagram', spec: '§5.6',
    what: "The same flow drawn as a highlighted path over the explorer's diagram." },
  { id: 'entities', design: S, listOnly: true, area: 'Architecture', title: 'Entity table and clusters', spec: '§5.7',
    what: "A table of every entity with its fields, relations and owning component; a large data view clustered, opening a cluster to its tables." },
  { id: 'dep-groups', listOnly: true, area: 'Architecture', title: 'Dependencies by purpose', spec: '§5.8',
    what: "Packages grouped as framework, database driver, HTTP client, SDK, testing, build; an SDK linked to its external system's box." },
  { id: 'on-this-page', design: S, listOnly: true, area: 'Shell', title: 'On this page', spec: '§4.1',
    what: 'The right rail of a reading page: its headings, to jump between them.' },
  { id: 'keys', listOnly: true, area: 'Shell', title: 'Page shortcuts', spec: '§4.2',
    what: 'g o for the overview, g a for the architecture, [ and ] for the previous and next page.' },

  // Changes — §5.12
  { id: 'commits', design: S, area: 'Changes', title: 'Compare two commits in the app', spec: '§5.12, F-37',
    what: 'Pick two commits and see what changed between them. Built on the command line — archdoc diff <path> <commit> [<commit>] — not yet in the app.' },
  { id: 'session-summary', area: 'Changes', title: 'What the AI did, in a paragraph', spec: '§5.12, F-38',
    what: '"This session added X, touched Y, introduced Z": a summary of a commit or a session a person reads, every sentence cited. The next step back to the vision.' },

  // Corrections — §5.13
  { id: 'correct', design: C, area: 'Corrections', title: 'Correct this…', spec: '§5.13 (C-2)',
    what: "From the inspector, a small form that writes the correction into .archdoc/rules.yaml — appended, after a preview of the exact lines." },

  // Coverage — §5.11
  { id: 'sample-values', listOnly: true, area: 'Coverage', title: 'Values from sample files', spec: '§5.11, F-53',
    what: 'Which values were filled from example.env-style files, each cited.' },

  // Seen in the Claude Design files on 9 Oct, beyond what the spec listed
  { id: 'plain', design: S, area: 'Reading', title: 'Plain language', spec: '§5.1, §5.4; vision §2.7',
    what: 'A second reading of every interpreted summary — the Overview and each component page — for someone who does not read code: "This part of immich takes in every photo and video people add from their phone." Same citations.' },
  { id: 'neighbours', design: S, area: 'Reading', title: 'Neighbours, one hop', spec: '§5.4',
    what: "A small diagram on a component's page: what uses it on the left, what it uses on the right, itself in the middle." },
  { id: 'touches', design: S, area: 'Reading', title: 'Data it touches, and outbound', spec: '§5.4',
    what: "On a component's page, the tables its code reads and writes (from its flows), the queues it puts work on, and the calls it makes that leave — each cited, an unresolved one hatched." },
  { id: 'component-nav', design: S, area: 'Reading', title: 'Citations, and next and previous', spec: '§5.4',
    what: "A component page's numbered citations listed at its foot, and links to the previous and next component and to a flow that passes through it." },
  { id: 'cite-preview', design: F, listOnly: true, area: 'Reading', title: 'Citation hover preview', spec: '§6.2',
    what: 'Hovering a citation shows the cited line with one line of context either side; clicking opens it. Today a citation only opens.' },
  { id: 'lens', design: S, area: 'Explorer', title: 'Lens', spec: '§5.2',
    what: 'Recolour the diagram by one question — who owns it, what changed, what is unresolved — without changing what is drawn.' },
  { id: 'share-card', design: S, area: 'Explorer', title: 'Share card', spec: '§5.2',
    what: 'A 1200 × 630 picture of the view on screen, with the commit and date, downloadable as PNG.' },
  { id: 'inspector-tabs', design: S, area: 'Explorer', title: 'Inspector tabs and history', spec: '§5.3',
    what: 'The inspector in three tabs — facts, relationships, history — the last showing every version in which the element changed, since it first appeared.' },
  { id: 'status-egress', design: S, listOnly: true, area: 'Explorer', title: 'Unresolved and egress in the status bar', spec: '§5.2',
    what: "The explorer's status bar counting the unresolved calls in the view and saying how many bytes ever left the machine." },
  { id: 'moved-tag', design: F, listOnly: true, area: 'Explorer', title: 'Moved, on the diagram', spec: '§6.7',
    what: 'A renamed element drawn with the → tag when comparing versions — the engine reports renames since 9 Oct; the canvas does not draw them yet.' },
  { id: 'coverage-headline', design: S, area: 'Coverage', title: 'Coverage as a percentage, unresolved by reason', spec: '§5.11',
    what: 'The headline "87% of outbound references resolved: 96 of 110", and the unresolved ones grouped by why archdoc could not resolve them.' },
  { id: 'nav-badges', design: T, listOnly: true, area: 'Shell', title: 'Badges that say how far along', spec: '§4.1',
    what: 'Coverage as a percentage, Documents as written of planned (7/12), Changes as a dot with the number since the compared version.' },
  { id: 'run-download', design: S, area: 'Network runs', title: 'Copy and download a request', spec: '§5.14',
    what: 'Copy a request body exactly as sent, or download it as .json.' },
  { id: 'print', design: T, listOnly: true, area: 'Shell', title: 'Print with citations in the margin', spec: '§2 rule 6',
    what: 'A page printed without chrome, every citation written out in full in a margin column, hatching kept for unresolved.' },

  // Stretch — designed as slots
  { id: 'ask', design: S, area: 'Later', title: 'Ask the map', spec: '§5.16, F-26', waits: 'out of this phase',
    what: 'A question in your words, answered only from the validated model, every sentence cited; "not resolved — here is what archdoc saw" as a designed answer.' },
  { id: 'intent', design: S, area: 'Later', title: 'Intent versus actual', spec: '§5.15, F-40, F-41', waits: 'out of this phase',
    what: 'The plan files you name, requirement by requirement: found in the code (cited), missing, and built but not asked for.' },
]

export const plan = (id: string) => planned.find((p) => p.id === id)
