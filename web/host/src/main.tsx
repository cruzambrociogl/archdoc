// The build the archdoc binary embeds: the app, with the Source of the place it was opened in. The
// published site is this same build; `archdoc export --site` marks its index.html, and that marker
// is how the page knows it is static.

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App, SourceProvider } from '@archdoc/app'
import { local } from './adapters/local'
import { published } from './adapters/published'

const isPublished = !!document.querySelector('meta[name="archdoc-mode"][content="published"]')

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <SourceProvider value={isPublished ? published : local}>
      <App />
    </SourceProvider>
  </StrictMode>,
)
