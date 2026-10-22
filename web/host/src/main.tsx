// The build the archdoc binary embeds: the app, with the Source of the place it was opened in. The
// published site is this same build; `archdoc export --site` marks its index.html, and that marker
// is how the page knows it is static.

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import type { Source } from '@archdoc/app'
import { App, SourceProvider } from '@archdoc/app'
import { local, system } from './adapters/local'
import { published } from './adapters/published'

const isPublished = !!document.querySelector('meta[name="archdoc-mode"][content="published"]')

// archdoc serve on a folder of repositories answers /api/system; on one repository, it says there is none.
async function place(): Promise<Source> {
  if (isPublished) return published
  const r = await fetch('/api/system').catch(() => undefined)
  return r?.ok ? system : local
}

place().then((source) =>
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <SourceProvider value={source}>
        <App />
      </SourceProvider>
    </StrictMode>,
  ),
)
