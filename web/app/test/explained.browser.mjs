// The views that explain, in a real browser, on any repository with an explanation: the tour, the
// features, a scenario played and stepped, the review, what to notice beneath a diagram, the dynamic
// diagram — each checked against the explanation the server holds, not against numbers written here.
// And, given a second repository with none, the state without AI.
//
// By hand, not in CI — it needs a documented, explained repository and Chrome:
//   archdoc interpret <a repository> --yes
//   archdoc serve <that repository> --port 7491
//   [archdoc serve <a repository never explained> --port 7492]
//   "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
//     --remote-debugging-port=9333 --user-data-dir="$(mktemp -d)" about:blank
//   node web/app/test/explained.browser.mjs 9333 http://localhost:7491 [http://localhost:7492]
const [port, base, plain] = process.argv.slice(2)
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
let failed = 0
const check = async (what, expr, want) => {
  let got
  for (let i = 0; i < 40; i++) {
    try { got = await js(expr) } catch { got = undefined }
    if (got === want) break
    await sleep(150)
  }
  const ok = got === want
  if (!ok) failed++
  console.log(`${ok ? 'ok  ' : 'FAIL'} ${what}${ok ? '' : `\n     want: ${JSON.stringify(want)}\n     got:  ${JSON.stringify(got)}`}`)
}
// Each place is loaded afresh, so the app it runs is the one the server sends now.
const open = async (at) => { await send('Page.navigate', { url: `${at}/` }); await sleep(1500); await send('Page.reload', { ignoreCache: true }); await sleep(2500) }
const hgo = async (hash) => { await js(`location.hash = ${JSON.stringify(hash)}, true`); await sleep(1500) }
const slug = (s) => s.toLowerCase().normalize('NFKD').replace(/[̀-ͯ]/g, '').replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
await send('Network.enable')
await send('Network.setCacheDisabled', { cacheDisabled: true })

const it = await (await fetch(`${base}/api/interpretation`, { headers: { Host: 'localhost' } })).json()
console.log(`${it.features.length} features, ${(it.scenarios ?? []).length} scenarios, ${(it.review ?? []).length} review points, ${(it.notes ?? []).length} notes\n`)
await open(base)

// Home, a first visit: the tour.
await hgo('/home')
await check('the tour, its strip', `!!document.querySelector('.tour-strip')`, true)
await check('what it is: the thesis, as written', `document.querySelector('.tour-thesis')?.textContent`, it.thesis.text)
await check('marked as written by AI', `!!document.querySelector('.tour-mark .written-by-ai')`, true)
await check('its citations open what they name', `document.querySelectorAll('.tour-part .cite-chip').length > 0`, true)

// Features by purpose, and one.
await hgo('/features')
await check('every feature, a tile', `document.querySelectorAll('.feature-tile').length`, it.features.length)
await js(`document.querySelector('.feature-tile').click(), true`)
await check('a tile opens its feature', `location.hash`, `#/features/${slug(it.features[0].name)}`)
await check('the feature, by name', `document.querySelector('.band-title')?.textContent`, it.features[0].name)
await check('its parts lit on Apps and services', `document.querySelectorAll('.view-diagram .react-flow__node').length > 0`, true)
await hgo('/features/no-such-feature')
await check('an address no feature has says so', `document.querySelector('.lede')?.textContent.startsWith('There’s no feature')`, true)

// Scenarios: absent from the menu without one; read three ways with one.
const scenarios = it.scenarios ?? []
await check('Scenarios in the menu only when there is one', `[...document.querySelectorAll('.nav-item .nav-label')].some(l => l.textContent === 'Scenarios')`, scenarios.length > 0)
if (scenarios.length) {
  const s = scenarios[0]
  await hgo(`/scenarios/${slug(s.title)}/played`)
  await check('a dot for each step', `document.querySelectorAll('.player .step-dot').length`, s.steps.length)
  await check('on the first step', `document.querySelector('.player-now')?.textContent.includes(${JSON.stringify(s.steps[0].text)})`, true)
  if (s.steps.length > 1) {
    await js(`window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' })), true`)
    await check('→ moves a step, and the address keeps it', `location.hash.endsWith('?step=2')`, true)
    await check('the second step', `document.querySelector('.player-now')?.textContent.includes(${JSON.stringify(s.steps[1].text)})`, true)
  }
  await js(`[...document.querySelectorAll('.lens-btn')].find(b => b.textContent === 'As steps').click(), true`)
  await check('as steps: every sentence', `document.querySelectorAll('.as-step').length`, s.steps.length)
  await check('changing the way keeps the step', `location.hash.includes('/steps') && location.hash.endsWith(${JSON.stringify(s.steps.length > 1 ? '?step=2' : '/steps')})`, true)
  await js(`[...document.querySelectorAll('.lens-btn')].find(b => b.textContent === 'As a sequence').click(), true`)
  await check('as a sequence: a message for each step', `document.querySelectorAll('.scenario-sequence .seq-msg').length`, s.steps.length)
  await hgo(`/architecture/dynamic/${slug(s.title)}`)
  await check('the dynamic diagram, titled', `document.querySelector('.c4-title')?.textContent.endsWith(${JSON.stringify(s.title.toLowerCase())})`, true)
  await check('its steps beneath it', `document.querySelectorAll('.dynamic-steps li').length`, s.steps.length)
}

// The review, whole.
await hgo('/home/review')
await check('every review point', `document.querySelectorAll('.points .point').length`, (it.review ?? []).length)

// What to notice, beneath the container diagram, where the explanation wrote one.
await hgo('/architecture/containers')
const note = (it.notes ?? []).find((n) => n.diagram === 'container')
await check('what to notice beneath Apps and services', `!!document.querySelector('.notice-block')`, !!note)

// A repository never explained: the facts' version, and why.
if (plain) {
  await open(plain)
  await hgo('/features')
  await check('without AI: says so', `document.querySelector('.not-explained')?.textContent.includes('This version is not explained.')`, true)
  await check('without AI: the features from the facts stand in', `!!document.querySelector('.feature-tile')`, false)
  await check('without AI: no Scenarios in the menu', `[...document.querySelectorAll('.nav-item .nav-label')].some(l => l.textContent === 'Scenarios')`, false)
  await hgo('/home/review')
  await check('without AI: the review says why there is none', `!!document.querySelector('.not-explained')`, true)
}

console.log(failed ? `\n${failed} failed` : '\nall passed')
ws.close()
process.exit(failed ? 1 : 0)
