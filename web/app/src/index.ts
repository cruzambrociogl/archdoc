// @archdoc/app — the one app every place runs: the shell and every page (platform-plan step 2).
// A host gives it a Source — where its data comes from, and what the place can do — and renders it.

// The app first, then its stylesheets: what the app's modules import themselves — React Flow's
// base styles — must come before the app's own, which override them. Order is the cascade.
export { App } from './App'
export { Refusal, SourceProvider } from './source'
export { staticName } from './staticName'
export type { Capabilities, Homes, HostedRun, Place, Runs, Source, Systems } from './source'
export type * from './api'

import './styles/fonts.css'
import './styles/tokens.css'
import './styles/base.css'
import './styles/shell.css'
import './styles/ui.css'
import './styles/inspector.css'
import './styles/explorer.css'
import './styles/page.css'
