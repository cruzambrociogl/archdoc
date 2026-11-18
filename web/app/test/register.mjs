// Lets Node run the app's TypeScript modules as they are written for the bundler: an import with no
// extension — './api' — resolves to the .ts file beside it. Node strips the types itself.
import { register } from 'node:module'

register('./resolve.mjs', import.meta.url)
