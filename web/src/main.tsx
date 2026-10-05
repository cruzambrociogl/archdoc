import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import './styles/fonts.css'
import './styles/tokens.css'
import './styles/base.css'
import './styles/shell.css'
import './styles/ui.css'
import './styles/inspector.css'
import './styles/explorer.css'
import './styles/page.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
