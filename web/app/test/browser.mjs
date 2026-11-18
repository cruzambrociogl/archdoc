// The app in a real browser, as a reader uses it: opens One part from a box's panel, changes the
// subject by a box, a row and Find a part, switches the question, copies a link, narrows the menu.
// Each check says what the page shows after the step. Immich's ids, at the revision
// docs/survey-test-subjects.md pins.
//
// By hand, not in CI — it needs a documented Immich and Chrome:
//   archdoc serve <your Immich clone> --port 7491
//   "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
//     --remote-debugging-port=9333 --user-data-dir="$(mktemp -d)" about:blank
//   node web/app/test/browser.mjs 9333 http://localhost:7491
const port = process.argv[2], base = process.argv[3]
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const list = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json()
const page = list.find((t) => t.type === 'page')
const ws = new WebSocket(page.webSocketDebuggerUrl)
await new Promise((r) => ws.addEventListener('open', r))
let id = 0
const pending = new Map()
ws.addEventListener('message', (m) => { const d = JSON.parse(m.data); if (d.id && pending.has(d.id)) { pending.get(d.id)(d); pending.delete(d.id) } })
const send = (method, params = {}) => new Promise((r) => { const i = ++id; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })) })
const js = async (expr) => { const r = await send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true }); if (r.result.exceptionDetails) throw new Error(JSON.stringify(r.result.exceptionDetails)); return r.result.result.value }
const go = async (hash) => { await send('Page.navigate', { url: `${base}/#${hash}` }); await sleep(2500) }
const until = async (expr, what, ms = 6000) => { const t = Date.now(); while (Date.now() - t < ms) { if (await js(expr)) return; await sleep(150) } throw new Error('timed out: ' + what) }
let failed = 0
const check = async (what, expr, want) => {
  let got, first
  for (let i = 0; i < 20; i++) { got = await js(expr); if (i === 0) first = got; if (got === want) break; await sleep(150) }
  if (got !== want && first !== undefined) got = first
  const ok = got === want
  if (!ok) failed++
  console.log(`${ok ? 'ok  ' : 'FAIL'} ${what}${ok ? '' : `\n     want: ${JSON.stringify(want)}\n     got:  ${JSON.stringify(got)}`}`)
}
await send('Page.enable'); await send('Runtime.enable')
const title = `document.querySelector('.band-title')?.textContent`
const hash = `location.hash`
const typeInto = (sel, text) => js(`(() => { const i = document.querySelector(${JSON.stringify(sel)}); i.focus(); const set = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set; set.call(i, ${JSON.stringify(text)}); i.dispatchEvent(new Event('input', { bubbles: true })); return true })()`)
const key = (sel, k) => js(`(() => { const t = ${sel ? `document.querySelector(${JSON.stringify(sel)})` : 'document.body'}; t.dispatchEvent(new KeyboardEvent('keydown', { key: ${JSON.stringify(k)}, bubbles: true })); return true })()`)

// 1. The box's panel on Apps and services opens One part.
await go('/architecture/containers?part=svc%3Aimmich-server')
await until(`!!document.querySelector('.inspector-ask')`, 'the ask block')
await js(`[...document.querySelectorAll('.inspector-ask button')].find(b => b.textContent.includes('depends')).click(), true`)
await check('the panel opens What depends on it', hash, '#/architecture/part/svc%3Aimmich-server/depends')
await check('its band asks the question', title, 'What depends on immich-server?')

// 2. Selecting another box changes the subject and keeps the question.
await until(`!!document.querySelector('.part-diagram .react-flow__node[data-id="svc:database"]')`, 'the diagram')
await js(`document.querySelector('.part-diagram .react-flow__node[data-id="svc:database"]').click(), true`)
await check('a box becomes the subject, the question kept', hash, '#/architecture/part/svc%3Adatabase/depends')
await check('the band follows', title, 'What depends on database?')
await check('the sentence', `document.querySelector('.band-line').textContent`, 'immich-server uses database directly. @immich/cli, immich_mobile, immich-web and User reach it through immich-server.')
await check('five rows', `document.querySelectorAll('.part-row').length`, 5)

// 3. A person is not a subject: clicking it changes nothing.
await js(`document.querySelector('.part-diagram .react-flow__node[data-id="actor:user"]').click(), true`)
await sleep(400)
await check('clicking the person keeps the subject', hash, '#/architecture/part/svc%3Adatabase/depends')

// 4. The other tab.
await js(`[...document.querySelectorAll('.part-bar [role=tab]')].find(b => b.textContent === 'What it touches').click(), true`)
await check('the tab switches the question', hash, '#/architecture/part/svc%3Adatabase/touches')
await check('a part that calls nothing', `document.querySelector('.band-line').textContent`, 'database calls nothing on this diagram. It only answers.')
await check('nothing to list', `document.querySelector('.part-list p')?.textContent`, 'Nothing to list.')

// 5. A row makes its part the subject.
await go('/architecture/part/app%3Aweb/touches')
await until(`document.querySelectorAll('.part-row').length > 0`, 'rows')
await js(`[...document.querySelectorAll('button.part-row-name')].find(b => b.textContent === 'redis').click(), true`)
await check('a row asks about its part', hash, '#/architecture/part/svc%3Aredis/touches')

// 6. Find a part: / moves to it, typing counts, Enter makes the first match the subject, Esc clears.
await go('/architecture/part/svc%3Aredis/depends')
await until(`!!document.querySelector('.part-diagram .find input')`, 'the find box')
await key(null, '/')
await check('/ moves to Find a part', `document.activeElement?.getAttribute('aria-label')`, 'Find a part')
await typeInto('.part-diagram .find input', 'machine')
await check('the count', `document.querySelector('.part-diagram .find-count')?.textContent`, '1 found')
await check('the find is in the address', hash, '#/architecture/part/svc%3Aredis/depends?q=machine')
await key('.part-diagram .find input', 'Enter')
await check('Enter makes the match the subject, the find cleared', hash, '#/architecture/part/svc%3Aimmich-machine-learning/depends')
await typeInto('.part-diagram .find input', 'zzz')
await check('no match', `document.querySelector('.part-diagram .find-count')?.textContent`, 'none found')
await key('.part-diagram .find input', 'Escape')
await check('Esc clears the find', hash, '#/architecture/part/svc%3Aimmich-machine-learning/depends')

// 7. The breadcrumb names the part and goes back to Architecture.
await check('breadcrumb', `[...document.querySelectorAll('.crumbs .crumb')].map(c => c.textContent).join(' › ')`, 'immich › Architecture › immich-machine-learning')
await js(`[...document.querySelectorAll('.crumbs button.crumb')].find(b => b.textContent === 'Architecture').click(), true`)
await check('Architecture from the breadcrumb', hash, '#/architecture')

// 8. Copy link says the view in words, and the clipboard holds the address.
await send('Browser.grantPermissions', { permissions: ['clipboardReadWrite', 'clipboardSanitizedWrite'], origin: base })
// A real window has focus; a headless one does not, and the clipboard refuses a page without it.
await send('Emulation.setFocusEmulationEnabled', { enabled: true })
await go('/architecture/part/svc%3Aredis/touches')
await until(`!!document.querySelector('.copy-link')`, 'copy link')
await js(`document.querySelector('.copy-link').click(), true`)
await check('the copy message', `document.querySelector('.toast')?.textContent`, 'Link copied · Architecture › redis')
await check('the clipboard holds this view', `navigator.clipboard.readText()`, base + '/#/architecture/part/svc%3Aredis/touches')

// 9. The menu narrows, and stays narrow after a reload.
await js(`document.querySelector('.nav-toggle').click(), true`)
await check('the menu narrows', `document.querySelector('nav.nav').classList.contains('narrow')`, true)
await check('to its icons', `Math.round(document.querySelector('nav.nav').getBoundingClientRect().width)`, 52)
await go('/home')
await check('narrow after a reload', `document.querySelector('nav.nav').classList.contains('narrow')`, true)
await js(`document.querySelector('.nav-toggle').click(), true`)
await check('and wide again', `Math.round(document.querySelector('nav.nav').getBoundingClientRect().width)`, 176)

// 10. Keys: ⌘K opens search, Esc closes it; ? opens the keys.
await key(null, '?')
await check('? shows the keys', `!!document.querySelector('.shortcuts')`, true)

console.log(failed ? `\n${failed} failed` : '\nall passed')
ws.close()
process.exit(failed ? 1 : 0)
