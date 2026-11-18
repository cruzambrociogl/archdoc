// The app in a real browser, as a reader uses it: opens One part from a box's panel, changes the
// subject by a box, a row and Find a part, switches the question, copies a link, narrows the menu;
// opens What archdoc knows from Home, both its tabs, its lists and the report one step away; Usage's
// faces, an entry chosen and filtered, its steps in Reference.
// Each check says what the page shows after the step. Immich's ids, at the revision
// docs/survey-test-subjects.md pins.
//
// By hand, not in CI — it needs a documented Immich and Chrome:
//   archdoc serve <your Immich clone> --port 7491
//   "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
//     --remote-debugging-port=9333 --user-data-dir="$(mktemp -d)" about:blank
//   node web/app/test/browser.mjs 9333 http://localhost:7491 [a repository with history, served too]
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
  // An element not there yet reads as nothing, and the check waits for it as for a wrong value.
  for (let i = 0; i < 40; i++) {
    try { got = await js(expr) } catch { got = undefined }
    if (i === 0) first = got
    if (got === want) break
    await sleep(150)
  }
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
await until(`!!document.querySelector('.view-diagram .react-flow__node[data-id="svc:database"]')`, 'the diagram')
await js(`document.querySelector('.view-diagram .react-flow__node[data-id="svc:database"]').click(), true`)
await check('a box becomes the subject, the question kept', hash, '#/architecture/part/svc%3Adatabase/depends')
await check('the band follows', title, 'What depends on database?')
await check('the sentence', `document.querySelector('.band-line').textContent`, 'immich-server uses database directly. @immich/cli, immich_mobile, immich-web and User reach it through immich-server.')
await check('five rows', `document.querySelectorAll('.part-row').length`, 5)

// 3. A person is not a subject: clicking it changes nothing.
await js(`document.querySelector('.view-diagram .react-flow__node[data-id="actor:user"]').click(), true`)
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
await until(`!!document.querySelector('.view-diagram .find input')`, 'the find box')
await key(null, '/')
await check('/ moves to Find a part', `document.activeElement?.getAttribute('aria-label')`, 'Find a part')
await typeInto('.view-diagram .find input', 'machine')
await check('the count', `document.querySelector('.view-diagram .find-count')?.textContent`, '1 found')
await check('the find is in the address', hash, '#/architecture/part/svc%3Aredis/depends?q=machine')
await key('.view-diagram .find input', 'Enter')
await check('Enter makes the match the subject, the find cleared', hash, '#/architecture/part/svc%3Aimmich-machine-learning/depends')
await typeInto('.view-diagram .find input', 'zzz')
await check('no match', `document.querySelector('.view-diagram .find-count')?.textContent`, 'none found')
await key('.view-diagram .find input', 'Escape')
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

// 11. What archdoc knows, from Home's trust sentence.
await go('/home')
await until(`!!document.querySelector('.trust-sentence')`, 'the trust sentence')
await js(`document.querySelector('.trust-sentence').click(), true`)
await check('Home opens What archdoc knows', hash, '#/about/what-was-read')
await check('its question', title, 'How much of this picture can you trust?')
await check('what was read, in numbers', `document.querySelector('.band-line').textContent.split('. ')[0]`, 'archdoc read the code of all 5 applications: 1,761 files, 189,980 lines')
await check('the calls it couldn’t follow, by part', `document.querySelector('.band-line').textContent.includes('11 calls go to an address set at run time, which archdoc couldn’t follow: 5 in immich-server, 4 in immich-web and 2 in @immich/cli.')`, true)
await check('no claim about tracing from a report that does not count it', `document.querySelector('.band-line').textContent.includes('traced')`, false)
await check('two parts hatched', `document.querySelectorAll('.view-diagram .el-hatch').length`, 2)
await check('tagged Partly read', `[...document.querySelectorAll('.view-diagram .el-tag')].map(t => t.textContent).join(',')`, 'Partly read,Partly read')
await check('nothing dimmed', `document.querySelectorAll('.view-diagram .is-dim').length`, 0)
await check('the key', `!!document.querySelector('.view-key')`, true)
await check('five of what couldn’t be resolved', `document.querySelectorAll('section[aria-label="What couldn’t be resolved"] .part-row').length`, 5)
await check('its count', `document.querySelector('section[aria-label="What couldn’t be resolved"] .part-list-head .muted').textContent`, '12')
await check('five of what wasn’t read', `document.querySelectorAll('section[aria-label="What wasn’t read"] .part-row').length`, 5)
await check('every unresolved row cites its line', `[...document.querySelectorAll('section[aria-label="What couldn’t be resolved"] .part-row')].every(r => r.querySelector('.cite'))`, true)
await js(`document.querySelector('.part-all').click(), true`)
await check('Show all: in the address', hash, '#/about/what-was-read?all=1')
await check('all twelve', `document.querySelectorAll('section[aria-label="What couldn’t be resolved"] .part-row').length`, 12)
await check('all twenty-three', `document.querySelectorAll('section[aria-label="What wasn’t read"] .part-row').length`, 23)

// 12. The other tab, and the whole report one step away.
await js(`[...document.querySelectorAll('.part-bar [role=tab]')].find(b => b.textContent === 'What the AI wrote').click(), true`)
await check('What the AI wrote', hash, '#/about/what-the-ai-wrote')
await check('its question', title, 'Which words did the AI write?')
await check('on a run without AI, it says so', `document.querySelector('.band-line').textContent.startsWith('None of it.')`, true)
await check('no box tagged', `document.querySelectorAll('.view-diagram .el-tag').length`, 0)
await check('the breadcrumb', `[...document.querySelectorAll('.crumbs .crumb')].map(c => c.textContent).join(' › ')`, 'immich › Home › What archdoc knows')
await check('the menu keeps Home lit', `document.querySelector('.nav-item.active .nav-label').textContent`, 'Home')
await js(`document.querySelector('.part-more').click(), true`)
await check('About these docs, one step away', hash, '#/about/report')
await check('its breadcrumb', `[...document.querySelectorAll('.crumbs .crumb')].map(c => c.textContent).join(' › ')`, 'immich › Home › About these docs')

// 13. Find a part on this diagram lights what matches; nothing here can be picked.
await go('/about/what-was-read')
await until(`!!document.querySelector('.view-diagram .find input')`, 'the find box')
await typeInto('.view-diagram .find input', 'redis')
await check('found', `document.querySelector('.view-diagram .find-count')?.textContent`, '1 found')
await key('.view-diagram .find input', 'Enter')
await sleep(300)
await check('Enter picks nothing on a view that has no subject', hash, '#/about/what-was-read?q=redis')

// 14. Usage: the faces from the menu, an entry chosen, the filter, its steps in Reference.
await go('/home')
await js(`[...document.querySelectorAll('.nav-item')].find(b => b.textContent === 'Usage').click(), true`)
await check('Usage from the menu', hash, '#/usage')
await check('the sentence naming the faces', `document.querySelector('.usage-lede').textContent`, 'immich is used 5 ways: on the screens of immich-web, through the APIs of immich-server and immich-machine-learning, and from the command lines of immich-server and @immich/cli.')
await check('five faces', `document.querySelectorAll('.usage-tab').length`, 5)
await check('the first is the screens', `document.querySelector('.usage-tab.on').textContent`, 'Screensimmich-web · 55 pages')
await check('every page listed', `document.querySelectorAll('.usage-row').length`, 55)
await check('the mark says what pages archdoc doesn’t read', `document.querySelector('.usage-mark').textContent.includes('doesn’t read what a page shows or calls yet')`, true)
await js(`[...document.querySelectorAll('.usage-tab')].find(b => b.textContent.startsWith('APIimmich-server')).click(), true`)
await check('the API of immich-server', hash, '#/usage/api/svc%3Aimmich-server')
await check('301 routes', `document.querySelectorAll('.usage-row').length`, 301)
await check('no background jobs among them', `[...document.querySelectorAll('.usage-row-address')].some(a => a.textContent.startsWith('JOB'))`, false)
await js(`[...document.querySelectorAll('.usage-row')].find(r => r.querySelector('.usage-row-name').textContent === 'Delete an album').click(), true`)
await check('an entry chosen, in the address', hash, '#/usage/api/svc%3Aimmich-server?entry=route%3Aimmich-server+DELETE+%2Fapi%2Falbums%2F%3Aid')
await check('the chosen entry at the right', `document.querySelector('.usage-entry-name').textContent`, 'Delete an album')
await check('its address', `document.querySelector('.usage-entry-address').textContent`, 'DELETE /api/albums/:id')
await check('what handles it, cited', `!!document.querySelector('.usage-facts dd .cite')`, true)
await check('what it sets off, as the engine followed it', `document.querySelectorAll('.usage-facts dd')[2].textContent`, 'Deletes album.Its 4 steps, each at its line →')
await check('the small map lights one part', `document.querySelectorAll('.view-diagram.small .react-flow__node-element:not(:has(.is-dim))').length`, 1)
await typeInto('.usage-filter', 'albums')
await check('the filter, in the address, the entry kept', hash, '#/usage/api/svc%3Aimmich-server?entry=route%3Aimmich-server+DELETE+%2Fapi%2Falbums%2F%3Aid&q=albums')
await check('fewer rows', `document.querySelectorAll('.usage-row').length < 301 && document.querySelectorAll('.usage-row').length > 0`, true)
await check('how many shown', `document.querySelector('.usage-list-foot').textContent.includes('shown of 301 routes')`, true)
await js(`[...document.querySelectorAll('.usage-facts .link-btn')].find(b => b.textContent.startsWith('Its ')).click(), true`)
await check('its steps, in the complete list, that entry open', hash, '#/reference/entry-points?entry=route%3Aimmich-server+DELETE+%2Fapi%2Falbums%2F%3Aid')
await check('Reference lit', `document.querySelector('.nav-item.active .nav-label').textContent`, 'Reference')
await go('/reference')
await js(`[...document.querySelectorAll('button, a')].find(b => b.textContent.includes('Every entry point')).click(), true`)
await check('Reference keeps every entry point, jobs too', hash, '#/reference/entry-points')

// 15. With a repository that has a history (a fourth argument): a change where it landed, and Home
// on a return visit. The Immich of 9 Oct, 30 versions, served from a copy.
const history = process.argv[4]
if (history) {
  const hgo = async (h) => { await send('Page.navigate', { url: `${history}/#${h}` }); await sleep(3000) }
  await hgo('/changes')
  await until(`document.querySelectorAll('.meaning-landed').length > 0`, 'the story')
  await check('each change can be seen where it landed', `document.querySelectorAll('.meaning-landed').length`, 8)
  await js(`document.querySelector('.meaning-landed').click(), true`)
  await check('one change, at its own address', hash, '#/changes/latest/cmp%3Amobile%2Fshared-link')
  await check('its band', title, 'shared-link, in immich_mobile')
  await check('where it landed', `document.querySelector('.band-line').textContent`, 'It landed in immich_mobile.')
  await check('what happened there', `[...document.querySelectorAll('.view-diagram .el-tag')].map(t => t.textContent).join(',')`, 'New part inside')
  await check('only there lit', `document.querySelectorAll('.view-diagram .react-flow__node-element:not(:has(.is-dim))').length`, 1)
  await check('cited', `!!document.querySelector('.band-meta .cite')`, true)
  await check('the breadcrumb', `[...document.querySelectorAll('.crumbs .crumb')].map(c => c.textContent).join(' › ')`, 'immich › Changes › One change')
  await js(`[...document.querySelectorAll('.part-more')].find(b => b.textContent.startsWith('Every change')).click(), true`)
  await check('every change in this version, one step away', hash, '#/changes')

  // Home, as if this reader last looked at version 25.
  await js(`(() => { const k = Object.keys(localStorage).find(k => k.startsWith('archdoc.seen.')); localStorage.setItem(k, '25'); return true })()`)
  await hgo('/home')
  await send('Page.reload'); await sleep(3500)
  await check('Home says since when', `document.querySelector('.return-since')?.textContent.startsWith('Since you last looked · ')`, true)
  await check('what the versions since did', `document.querySelector('.return-sentence').textContent`, '5 new versions since you last looked changed the architecture: added 62 parts; changed 33 connections.')
  await check('the first five changes', `document.querySelectorAll('.return-row').length`, 5)
  await check('a count on Changes', `[...document.querySelectorAll('.nav-item')].find(b => b.querySelector('.nav-label').textContent === 'Changes').querySelector('.nav-badge')?.textContent`, '5')
  await js(`document.querySelector('.return-row .link-btn').click(), true`)
  await check('a change since then, compared with what was seen', `location.hash.endsWith('?from=25')`, true)
  await check('the band says so', `document.querySelector('.band-meta').textContent.includes('compared with version 25')`, true)
  // The next visit has seen the latest: nothing new.
  await hgo('/home'); await send('Page.reload'); await sleep(3500)
  await check('the next visit: nothing new', `!!document.querySelector('.return-visit')`, false)
  await check('and no count', `!!document.querySelector('.nav-badge')`, false)
}

console.log(failed ? `\n${failed} failed` : '\nall passed')
ws.close()
process.exit(failed ? 1 : 0)
