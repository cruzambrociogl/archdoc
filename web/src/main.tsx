import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import './styles/fonts.css'
import './styles/tokens.css'
import './styles.css'
import './styles/shell.css'
import './styles/ui.css'
import './styles/inspector.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
