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
}

export const planned: Plan[] = [
  // Control — §11
  { id: 'freshness', area: 'Control', title: 'Freshness pill', spec: '§11.2, §11.7',
    what: 'Says in the top bar whether the code changed since this version was generated: Up to date, Code changed, Running…, Failed.' },
  { id: 'run', area: 'Control', title: 'Run from the app', spec: '§11.2 (C-1), §11.7',
    what: 'scan and generate with one click; generate with the model only after a dialog showing egress mode, model, items, bytes and estimated cost. Progress drawer with stages and cancel; one run at a time.' },
  { id: 'publish', area: 'Control', title: 'Publish dialog', spec: '§11.2, §11.7',
    what: 'Builds the site into .archdoc/site and shows the CI workflow to copy. Never deploys.' },
  { id: 'git', area: 'Control', title: 'Uncommitted files card', spec: '§11.2 (C-4), §11.7',
    what: "Which of archdoc's own files are uncommitted, their diff against the last commit, and the command to commit them. Read-only." },
  { id: 'settings', area: 'Control', title: 'Settings panel', spec: '§11.2 (C-3), §11.7',
    what: 'Egress mode, model, provider and effort, read-only. The API key is never entered in the browser.' },
  { id: 'watch', area: 'Control', title: 'Watch mode', spec: '§11.2, F-29',
    what: 'Regenerate on save, deterministic only: never calls a model on its own.' },

  // Navigation and shell — §4
  { id: 'deployment', area: 'Architecture', title: 'Deployment view', spec: '§5.9',
    what: 'Where and how it runs: image, published ports, networks and mounts per container, as a diagram and a table. Today it is only the arc42 §7 Markdown section.' },
  { id: 'flows', area: 'Architecture', title: 'Flows index', spec: '§5.6',
    what: 'Every traced flow on a page of its own, searchable and grouped by feature. Today a flow is shown only in its row on Features.' },
  { id: 'step-through', area: 'Architecture', title: 'Step through a flow', spec: '§5.6',
    what: 'Advance a sequence diagram one step at a time, the step highlighted with its line.' },
  { id: 'flow-on-canvas', area: 'Architecture', title: 'A flow over the diagram', spec: '§5.6',
    what: "The same flow drawn as a highlighted path over the explorer's diagram." },
  { id: 'entities', area: 'Architecture', title: 'Entity table and clusters', spec: '§5.7',
    what: "A table of every entity with its fields, relations and owning component; a large data view clustered, opening a cluster to its tables." },
  { id: 'dep-groups', area: 'Architecture', title: 'Dependencies by purpose', spec: '§5.8',
    what: "Packages grouped as framework, database driver, HTTP client, SDK, testing, build; an SDK linked to its external system's box." },
  { id: 'on-this-page', area: 'Shell', title: 'On this page', spec: '§4.1',
    what: 'The right rail of a reading page: its headings, to jump between them.' },
  { id: 'keys', area: 'Shell', title: 'Page shortcuts', spec: '§4.2',
    what: 'g o for the overview, g a for the architecture, [ and ] for the previous and next page.' },

  // Changes — §5.12
  { id: 'commits', area: 'Changes', title: 'Compare two commits in the app', spec: '§5.12, F-37',
    what: 'Pick two commits and see what changed between them. Built on the command line — archdoc diff <path> <commit> [<commit>] — not yet in the app.' },
  { id: 'session-summary', area: 'Changes', title: 'What the AI did, in a paragraph', spec: '§5.12, F-38',
    what: '"This session added X, touched Y, introduced Z": a summary of a commit or a session a person reads, every sentence cited. The next step back to the vision.' },

  // Corrections — §5.13
  { id: 'correct', area: 'Corrections', title: 'Correct this…', spec: '§5.13 (C-2)',
    what: "From the inspector, a small form that writes the correction into .archdoc/rules.yaml — appended, after a preview of the exact lines." },

  // Coverage — §5.11
  { id: 'sample-values', area: 'Coverage', title: 'Values from sample files', spec: '§5.11, F-53',
    what: 'Which values were filled from example.env-style files, each cited.' },

  // Stretch — designed as slots
  { id: 'ask', area: 'Later', title: 'Ask the map', spec: '§5.16, F-26', waits: 'out of this phase',
    what: 'A question in your words, answered only from the validated model, every sentence cited; "not resolved — here is what archdoc saw" as a designed answer.' },
  { id: 'intent', area: 'Later', title: 'Intent versus actual', spec: '§5.15, F-40, F-41', waits: 'out of this phase',
    what: 'The plan files you name, requirement by requirement: found in the code (cited), missing, and built but not asked for.' },
]

export const plan = (id: string) => planned.find((p) => p.id === id)
