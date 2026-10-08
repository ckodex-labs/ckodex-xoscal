#!/usr/bin/env node
// Dagger-owned read-only browser acceptance on exact candidate output bytes.
import fs from 'node:fs'
import path from 'node:path'
import http from 'node:http'
import crypto from 'node:crypto'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { chromium, request } from 'playwright'
import { checkApiLandmarks, checkApiEntryKeyboard, checkApiEntryReentry, checkApiEntryRouteDuringLoading, checkApiEntryFallbackFocus, checkApiEntryCancellation, checkApiEntryForwardFocus, checkApiReferenceFallback, checkStaticApiDeepLink, checkApiTheme } from './check-api-landmarks.mjs'

const args = Object.fromEntries(process.argv.slice(2).reduce((pairs, value, index, all) => index % 2 ? pairs : [...pairs, [value.replace(/^--/, ''), all[index + 1]]], []))
const root = fs.realpathSync(args.root), output = path.resolve(args.output)
const toolRoot = path.dirname(fileURLToPath(import.meta.url)), require = createRequire(import.meta.url)
const config = JSON.parse(fs.readFileSync(path.join(toolRoot, 'browser-toolchain.json')))
const receiptPath = path.join(output, 'presentation-browser.json')
const sha = bytes => 'sha256:' + crypto.createHash('sha256').update(bytes).digest('hex')
const started = performance.now(), deadlineMs = 1500000
const receipt = { schema_version: 1, status: 'failed', checks: [], failures: [], modes: [], negative_controls: [], subjects: {}, subject_sizes: {}, tools: {}, coverage_inventory: {}, proxy_denials: [] }
let browser, api, proxy, base, deadline

function verify(id, passed, detail) {
  const record = { id, passed: passed === true, detail }
  receipt.checks.push(record)
  if (!record.passed) receipt.failures.push(record)
}
async function bounded(id, fn) {
  try { await fn() } catch (error) { verify(id + ': exception', false, String(error.stack || error)) }
}

function subjectPaths() {
  const selected = fs.readdirSync(root).filter(name => fs.lstatSync(path.join(root, name)).isFile() && (/\.(html|css|js)$/.test(name) || /\.(js|css)\.map$/.test(name)))
  const walk = relative => {
    for (const name of fs.readdirSync(path.join(root, relative)).sort()) {
      const child = path.join(relative, name), stat = fs.lstatSync(path.join(root, child))
      if (stat.isSymbolicLink()) throw Error('Presentation subject is a symlink: ' + child)
      if (stat.isDirectory()) walk(child)
      else selected.push(child)
    }
  }
  for (const directory of ['assets', 'fonts']) if (fs.existsSync(path.join(root, directory))) walk(directory)
  selected.push('openapi.json', 'downloads.json', 'provenance.json', 'sbom-assessment-results.json', 'THIRD-PARTY-NOTICES.txt', 'scripts/install.sh')
  return [...new Set(selected)].sort()
}

function bindSubjects() {
  for (const relative of subjectPaths()) {
    const bytes = fs.readFileSync(path.join(root, relative))
    receipt.subjects[relative] = sha(bytes)
    receipt.subject_sizes[relative] = bytes.length
  }
}

function stageTools() {
  fs.mkdirSync(output, { recursive: true })
  for (const name of ['check-presentation-browser.mjs', 'check-api-landmarks.mjs', 'package.json', 'package-lock.json', 'browser-toolchain.json']) fs.copyFileSync(path.join(toolRoot, name), path.join(output, name))
  const lock = JSON.parse(fs.readFileSync(path.join(toolRoot, 'package-lock.json')))
  receipt.tools = { image: args.image, playwright: require('playwright/package.json').version, chromium: null, node: process.version, npm: execFileSync('npm', ['--version'], { encoding: 'utf8' }).trim(), packages: config.packages, chromium_revision: config.chromium_revision }
  for (const [key, name] of [['package_lock_digest', 'package-lock.json'], ['runner_digest', 'check-presentation-browser.mjs'], ['api_helper_digest', 'check-api-landmarks.mjs'], ['toolchain_digest', 'browser-toolchain.json'], ['package_digest', 'package.json']]) receipt.tools[key] = sha(fs.readFileSync(path.join(toolRoot, name)))
  verify('toolchain: immutable image and package version', args.image === config.image && receipt.tools.playwright === config.playwright)
  for (const [name, integrity] of Object.entries(config.packages)) verify('toolchain: SRI ' + name, lock.packages['node_modules/' + name]?.integrity === integrity && lock.packages['node_modules/' + name]?.version === config.playwright)
}

function admitReceipt() {
  const saved = JSON.parse(fs.readFileSync(receiptPath))
  if (saved.schema_version !== 1 || saved.status !== 'passed' || !Array.isArray(saved.failures) || saved.failures.length || !Array.isArray(saved.checks) || saved.checks.length < 100 || saved.checks.some(check => typeof check.id !== 'string' || check.passed !== true)) throw Error('Failed or incomplete browser receipt')
  if (new Set(saved.checks.map(check => check.id)).size !== saved.checks.length) throw Error('Duplicate browser check IDs')
  const actualIds = new Set(saved.checks.map(check => check.id))
  for (const id of requiredIds()) if (!actualIds.has(id)) throw Error('Missing mandatory browser check: ' + id)
  if (JSON.stringify(saved.modes.map(mode => mode.id)) !== JSON.stringify(config.modes)) throw Error('Missing browser mode')
  admitBindings(saved); admitObservations(saved)
  console.log('presentation browser admission: passed (' + saved.checks.length + ' checks)')
}

function admitBindings(saved) {
  if (saved.tools.image !== config.image || saved.tools.playwright !== config.playwright || saved.tools.chromium !== config.chromium || JSON.stringify(saved.tools.packages) !== JSON.stringify(config.packages)) throw Error('Browser tooling differs from pinned configuration')
  for (const [key, name] of [['package_lock_digest', 'package-lock.json'], ['runner_digest', 'check-presentation-browser.mjs'], ['api_helper_digest', 'check-api-landmarks.mjs'], ['toolchain_digest', 'browser-toolchain.json'], ['package_digest', 'package.json']]) if (saved.tools[key] !== sha(fs.readFileSync(path.join(toolRoot, name))) || saved.tools[key] !== sha(fs.readFileSync(path.join(output, name)))) throw Error('Changed browser producer: ' + name)
  if (JSON.stringify(Object.keys(saved.subjects).sort()) !== JSON.stringify(subjectPaths())) throw Error('Changed browser subject selection')
  for (const [relative, digest] of Object.entries(saved.subjects)) if (sha(fs.readFileSync(path.join(root, relative))) !== digest) throw Error('Changed browser subject: ' + relative)
}

function admitObservations(saved) {
  for (const mode of saved.modes) if (JSON.stringify(Object.keys(mode.observations)) !== JSON.stringify(['/', ...config.routes])) throw Error('Missing actual page coverage: ' + mode.id)
  for (const mode of saved.modes) if (mode.javaScriptEnabled !== mode.id.endsWith('-js') || ['console', 'network', 'page_errors'].some(key => !Array.isArray(mode[key]) || mode[key].length)) throw Error('Invalid or failed browser mode: ' + mode.id)
  const controls = config.modes.filter(id => id.endsWith('-js')).flatMap(id => ['404', 'tamper', 'script', 'focus-return', 'cancel', 'forward-focus', 'host-failure'].map(failure => id + ':' + failure))
  if (JSON.stringify(saved.negative_controls.map(control => control.id).sort()) !== JSON.stringify(controls.sort())) throw Error('Missing required negative control')
}

function requiredIds() {
  const failedEntries = config.modes.filter(mode => mode.endsWith('-js')).flatMap(mode => ['404', 'tamper'].map(failure => mode + ':' + failure + ': failed explorer: ready transition retains visible unoccluded keyboard focus'))
  const coldEntries = config.modes.filter(mode => mode.endsWith('-js')).flatMap(mode => ['focus-return', 'cancel'].flatMap(failure => ['cold contract exact actual held response', 'cold contract: actual Tab enters explorer summary', 'cold contract: cancellation keydown occurs during loading', 'cold contract: cancelled explorer never resurrects', failure === 'cancel' ? 'cancel explorer: ready transition retains visible unoccluded keyboard focus' : 'cold contract: moved Plain focus remains visible and owned'].map(name => mode + ':' + failure + ': ' + name)))
  const forwardEntries = config.modes.filter(mode => mode.endsWith('-js')).flatMap(mode => ['forward-focus', 'host-failure'].flatMap(failure => ['cold contract exact actual held response', 'cold forward: actual Tab enters explorer selector', 'cold forward: actual Tab keydown occurs during loading', 'cold forward: actual Tab selects host contract link', failure === 'host-failure' ? 'failed host explorer: ready transition retains visible unoccluded keyboard focus' : 'cold forward: ready retains visible unoccluded owned host focus'].map(name => mode + ':' + failure + ': ' + name)))
  return ['toolchain: immutable image and package version', 'toolchain: actual pinned Chromium', 'inventory: exact eight actual pages', 'served root alias: exact index bytes', 'receipt: complete fourteen negative controls', 'proxy: no non-service origins requested', 'inventory: read-only same-origin resources', ...config.modes.flatMap(modeRequiredIds), ...failedEntries, ...coldEntries, ...forwardEntries, ...config.modes.filter(mode => mode.endsWith('-js')).map(mode => mode + ':host-failure: cold host failure: selects only Plain and disposes explorer')]
}

function modeRequiredIds(mode) {
  const ids = [...routeRequiredIds(mode), ...nativeRequiredIds(mode), mode + ': all 491 native operation and model disclosures', mode + ': exact 96 operations and 395 model definitions', mode + '/portal.html: Blueprint six native anchors', mode + '/portal.html: Blueprint all 18 prebuilt sections', mode + '/portal.html: sample downloads distinct from release downloads']
  if (!mode.endsWith('-js')) ids.push(mode + '/portal.html: all six no-JS views visible without overlap')
  else ids.push(...lifecycleRequiredIds(mode))
  return ids
}

function routeRequiredIds(mode) {
  const ids = []
  for (const entry of ['/', ...config.routes]) for (const phase of ['initial', 'after native content']) ids.push(mode + '/' + entry + ': ' + phase + ': sole canonical main, unique IDs and ARIA', mode + '/' + entry + ': ' + phase + ': viewport fits native content')
  return ids
}

function nativeRequiredIds(mode) {
  const ids = []
  for (let index = 0; index < 491; index++) for (const phase of ['keyboard expansion', 'keyboard collapse', 'actual visible point unoccluded']) ids.push(mode + ': native contract ' + index + ': ' + phase)
  return ids
}

function lifecycleRequiredIds(mode) {
  const ids = ['view link', 'plain view link', 'native summary', 'intro reentry', 'model reentry'].map(kind => mode + (kind.endsWith('reentry') ? ': API native reentry/' : ': API keyboard entry/') + kind + ': ready transition retains visible unoccluded keyboard focus')
  for (const name of ['view link: actual Tab reaches explorer selector', 'plain view link: actual Shift+Tab reaches Plain selector', 'native summary: actual Tab reaches explorer summary after far Plain contract']) ids.push(mode + ': API keyboard entry/' + name)
  ids.push(mode + ': cold contract exact actual held response', ...['intro reentry: actual prior Scalar fragment', 'intro reentry: native close retains prior fragment', 'model reentry: actual prior Scalar fragment', 'model reentry: native close retains prior fragment', 'initial model deep link preserves requested route', 'initial model deep link: requested model owns visible focus after readiness', ...['intro reentry', 'model reentry'].flatMap(kind => [kind + ': actual Tab reaches explorer summary after far Plain contract', kind + ': ready transition selects only interactive reference'])].map(name => mode + ': API native reentry/' + name), ...['new route: actual hash navigation occurs during loading', 'new route: requested model owns visible focus after readiness'].map(name => mode + ': API route during loading/' + name))
  for (const phase of ['online API lifecycle', 'loaded offline API lifecycle']) for (let cycle = 0; cycle < 3; cycle++) ids.push(mode + ': ' + phase + '/cycle ' + cycle + ': API client is a visible labelled dialog', mode + ': ' + phase + '/cycle ' + cycle + ' dismissed, including hidden DOM: no duplicate DOM IDs')
  return ids
}
async function startProxy() {
  // Loopback supplies the same secure-context semantics as hosted HTTPS without
  // changing browser security flags. Every byte comes from Dagger's HTTP service.
  proxy = http.createServer(async (req, res) => {
    try {
      if (!['GET', 'HEAD'].includes(req.method)) { res.writeHead(405); res.end(); return }
      const target = new URL(req.url, args.base)
      if (target.origin !== new URL(args.base).origin) { receipt.proxy_denials.push({ url: req.url, method: req.method }); res.writeHead(403); res.end(); return }
      const remote = await fetch(target, { method: req.method, redirect: 'error', signal: AbortSignal.timeout(15000) })
      const bytes = Buffer.from(await remote.arrayBuffer())
      res.writeHead(remote.status, { 'Content-Type': remote.headers.get('content-type') || 'application/octet-stream', 'Cache-Control': 'no-store' })
      res.end(req.method === 'HEAD' ? undefined : bytes)
    } catch { res.writeHead(502); res.end('Candidate service unavailable') }
  })
  await new Promise((resolve, reject) => { proxy.once('error', reject); proxy.listen(8081, '127.0.0.1', resolve) })
  base = 'http://127.0.0.1:8081/'
}
async function httpBinding() {
  api = await request.newContext()
  for (const relative of Object.keys(receipt.subjects)) {
    const response = await api.get(new URL(relative, base).href), bytes = await response.body()
    verify('served subject: ' + relative, response.status() === 200 && sha(bytes) === receipt.subjects[relative], { status: response.status(), digest: sha(bytes), size: bytes.length })
  }
  const alias = await api.get(base)
  verify('served root alias: exact index bytes', alias.status() === 200 && sha(await alias.body()) === receipt.subjects['index.html'])
}

function inventoryDOM() {
  const selector = element => {
    if (element.id) return '#' + CSS.escape(element.id)
    const parts = []
    while (element && element !== document.documentElement) {
      const siblings = [...element.parentElement.children].filter(node => node.tagName === element.tagName)
      parts.unshift(element.tagName.toLowerCase() + ':nth-of-type(' + (siblings.indexOf(element) + 1) + ')')
      element = element.parentElement
      if (element?.id) { parts.unshift('#' + CSS.escape(element.id)); break }
    }
    return parts.join(' > ')
  }
  const record = element => ({ selector: selector(element), view: element.closest('.view')?.id, text: element.textContent.replace(/\s+/g, ' ').trim(), fragments: [...element.querySelectorAll('h1,h2,h3,h4,h5,h6,p,dt,dd,summary')].filter(node => !node.closest('noscript')).map(node => node.textContent.replace(/\s+/g, ' ').trim()).filter(Boolean) })
  return { ids: [...document.querySelectorAll('[id]')].map(node => node.id), sections: [...document.querySelectorAll('section')].map(record), headings: [...document.querySelectorAll('h1,h2,h3,h4,h5,h6')].map(record), links: [...document.querySelectorAll('a[href]')].map(node => ({ href: node.getAttribute('href'), disabled: node.getAttribute('aria-disabled') })) }
}
async function inventory() {
  const context = await browser.newContext({ javaScriptEnabled: false, serviceWorkers: 'block' }), page = await context.newPage()
  const log = { console: [], network: [], requests: [], page_errors: [], network_failures: [], http_failures: [] }
  receipt.inventory_observations = log; observe(page, log); await safeRequests(context, log)
  for (const route of config.routes) { await page.goto(base + route, { waitUntil: 'networkidle' }); receipt.coverage_inventory[route] = await page.evaluate(inventoryDOM) }
  cleanNetwork(log, 'inventory'); await context.close()
  verify('inventory: exact eight actual pages', JSON.stringify(fs.readdirSync(root).filter(name => name.endsWith('.html')).sort()) === JSON.stringify(config.routes))
}
async function reveal(page, locator, js) {
  const view = await locator.evaluate(node => node.closest('.view')?.id)
  if (view && js) { await page.locator('#nav a[href="#' + view + '"]').focus(); await page.keyboard.press('Enter'); await page.locator('#' + view).waitFor({ state: 'visible' }) }
  for (const details of await locator.locator('xpath=ancestor::details').all()) if (await details.getAttribute('open') === null) { await details.locator(':scope > summary').focus(); await page.keyboard.press('Enter'); if (js && await details.getAttribute('id') === 'interactive-api-view') await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready') }
  await locator.waitFor({ state: 'visible' })
}

async function semantics(page, id) {
  const raw = await page.evaluate(() => {
    const ids = [...document.querySelectorAll('[id]')].map(node => node.id)
    const unresolved = [...document.querySelectorAll('[aria-labelledby],[aria-describedby]')].flatMap(node => ['aria-labelledby', 'aria-describedby'].flatMap(attr => (node.getAttribute(attr) || '').split(/\s+/).filter(Boolean).filter(id => !document.getElementById(id))))
    return { mains: document.querySelectorAll('main,[role="main"]').length, canonical: document.querySelectorAll('main#main').length, duplicates: ids.filter((id, index) => ids.indexOf(id) !== index), unresolved, width: innerWidth, scrollWidth: document.documentElement.scrollWidth }
  })
  verify(id + ': sole canonical main, unique IDs and ARIA', raw.mains === 1 && raw.canonical === 1 && !raw.duplicates.length && !raw.unresolved.length, raw)
  verify(id + ': viewport fits native content', raw.scrollWidth <= raw.width, raw)
  return raw
}

async function hitTest(page, locator, id, native = false) {
  const attempts = []
  for (let attempt = 0; attempt < 3; attempt++) {
    if (!native) await locator.scrollIntoViewIfNeeded()
    await locator.evaluate(node => node.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' }))
    if (!native) await page.waitForTimeout(100)
    const hit = await locator.evaluate(node => {
      const r = node.getBoundingClientRect(), left = Math.max(r.left, 0), right = Math.min(r.right, innerWidth), top = Math.max(r.top, 0), bottom = Math.min(r.bottom, innerHeight)
      const point = { x: (left + right) / 2, y: (top + bottom) / 2 }, topmost = document.elementFromPoint(point.x, point.y)
      const finite = Object.values(r.toJSON()).every(Number.isFinite)
      return { rect: r.toJSON(), viewport: { width: innerWidth, height: innerHeight }, point, hit: topmost?.tagName + '#' + topmost?.id, unobscured: finite && right > left && bottom > top && topmost !== null && (topmost === node || node.contains(topmost)), time: performance.now(), focus: document.activeElement?.tagName + '#' + document.activeElement?.id, hash: location.hash, scrollY, documentHeight: document.documentElement.scrollHeight }
    })
    attempts.push(hit)
    if (hit.unobscured) break
  }
  const hit = attempts.at(-1)
  verify(id + ': actual visible point unoccluded', hit.unobscured, { ...hit, attempts })
  return hit
}

async function nativeContent(page, route, profile, observation) {
  const source = receipt.coverage_inventory[route]
  for (const kind of ['sections', 'headings']) for (const item of source[kind]) {
    const locator = page.locator(item.selector)
    await reveal(page, locator, profile.js)
    const rect = await locator.boundingBox(), text = (await locator.textContent()).replace(/\s+/g, ' ').trim()
    const content = kind === 'headings' ? text === item.text : item.fragments.length ? item.fragments.every(fragment => text.includes(fragment)) : text === item.text
    verify(profile.id + '/' + route + '/' + kind + '/' + item.selector, Boolean(rect) && content && rect.width > 0 && rect.height > 0, { rect, text: text.slice(0, 160), prebuiltFragments: item.fragments.length })
    await hitTest(page, locator, profile.id + '/' + route + '/' + kind + '/' + item.selector)
  }
  observation.native = { sections: source.sections.length, headings: source.headings.length, links: source.links.length }
  if (route === 'portal.html') observation.blueprint = await blueprint(page, profile)
}

async function localLinks() {
  for (const route of config.routes) for (const [index, link] of receipt.coverage_inventory[route].links.entries()) {
    const id = 'native link: ' + route + '/' + index, target = new URL(link.href, new URL(route, base))
    if (link.href === '#') { verify(id, link.disabled === 'true'); continue }
    if (target.origin !== new URL(base).origin) { verify(id, ['https:', 'mailto:'].includes(target.protocol), { href: link.href, scope: 'documented external URI syntax' }); continue }
    const relative = decodeURIComponent(target.pathname.slice(1) || 'index.html')
    if (target.hash) verify(id, receipt.coverage_inventory[relative]?.ids.includes(decodeURIComponent(target.hash.slice(1))) === true, { href: link.href })
    else { const response = await api.head(target.href); verify(id, response.status() === 200, { href: link.href, status: response.status() }) }
  }
}

async function framing(page, route, profile) {
  const id = profile.id + '/' + route
  const nav = await page.locator('nav[aria-label="Primary"] a').evaluateAll(nodes => nodes.map(node => ({ href: node.getAttribute('href'), current: node.getAttribute('aria-current') })))
  verify(id + ': consistent eight-route global navigation', JSON.stringify(nav.map(node => node.href).sort()) === JSON.stringify(config.routes) && nav.filter(node => node.current === 'page').length === 1 && nav.find(node => node.current === 'page')?.href === route, nav)
  verify(id + ': persistent evidence and authority framing', await page.getByRole('complementary', { name: 'Evidence Margin', exact: true }).count() === 1 && await page.getByRole('contentinfo', { name: 'Authority Footer', exact: true }).count() === 1)
  const geometry = await page.evaluate(() => { const h = document.querySelector('main h1').getBoundingClientRect(); const s = [...document.querySelectorAll('main section')].filter(node => node.getClientRects().length); return { h1: h.toJSON(), firstSection: s.length ? Math.min(...s.map(node => node.getBoundingClientRect().top)) : null } })
  verify(id + ': title precedes visible section content', geometry.firstSection === null || geometry.h1.top < geometry.firstSection, geometry)
  const skip = page.locator('a[href="#main"]').first(); await skip.focus(); await page.keyboard.press('Enter')
  await page.waitForURL(url => url.hash === '#main')
  verify(id + ': native keyboard skip link', new URL(page.url()).hash === '#main', { url: page.url() })
  return geometry
}

async function blueprint(page, profile) {
  const links = page.locator('#nav a[data-view]'), ids = await links.evaluateAll(nodes => nodes.map(node => node.hash))
  verify(profile.id + ': Blueprint six native anchors', ids.length === 6 && await page.locator('#nav button[data-view]').count() === 0, ids)
  await sampleDownloadLabels(page, profile.id)
  for (const hash of ids) { await page.locator('#nav a[href="' + hash + '"]').focus(); await page.keyboard.press('Enter'); await page.locator(hash).waitFor({ state: 'visible' }); verify(profile.id + ': Blueprint keyboard ' + hash, new URL(page.url()).hash === hash) }
  const rects = await page.locator('.view').evaluateAll(nodes => nodes.map(node => ({ id: node.id, visible: Boolean(node.getClientRects().length), rect: node.getBoundingClientRect().toJSON() })))
  if (!profile.js) {
    const overlap = rects.some((a, i) => rects.slice(i + 1).some(b => Math.min(a.rect.right, b.rect.right) > Math.max(a.rect.left, b.rect.left) + 1 && Math.min(a.rect.bottom, b.rect.bottom) > Math.max(a.rect.top, b.rect.top) + 1))
    verify(profile.id + ': all six no-JS views visible without overlap', rects.every(node => node.visible) && !overlap, rects)
  }
  verify(profile.id + ': Blueprint all 18 prebuilt sections', receipt.coverage_inventory['portal.html'].sections.length === 18)
  for (const container of ['release-artifacts', 'sdks', 'frameworks', 'prov', 'canonical-cards', 'interop-cards', 'defensive-cards', 'ext-cards', 'interop-matrix', 'dux-grid', 'persona-grid', 'pipe-flow', 'pipe-grid', 'wf-list', 'trace-table']) await containerChecks(page, profile, container)
  return { anchors: ids, views: rects, containers: 15 }
}

async function sampleDownloadLabels(page, id) {
  const labels = await page.evaluate(() => {
    const text = selector => [...document.querySelectorAll(selector)].map(node => node.textContent.trim())
    return { release: text('nav[aria-label="Primary"] a[href="downloads.html"]'), local: text('#nav a[href="#view-downloads"]'), adjacent: text('.view-nav a[href="#view-downloads"] .vname'), heading: text('#view-downloads-title') }
  })
  verify(id + ': sample downloads distinct from release downloads', JSON.stringify(labels.release) === '["Downloads"]' && JSON.stringify(labels.local) === '["Sample downloads"]' && JSON.stringify(labels.adjacent) === '["Sample downloads","Sample downloads"]' && labels.heading.length === 1 && labels.heading[0].startsWith('Sample downloads —'), labels)
}

async function containerChecks(page, profile, container) {
  const node = page.locator('#' + container), children = node.locator(':scope > *')
  await reveal(page, node, profile.js)
  const count = await children.count()
  verify(profile.id + ': prebuilt container ' + container, (await node.textContent()).trim().length > 0 && count > 0)
  for (const index of [...new Set([0, Math.floor(count / 2), count - 1])]) await hitTest(page, children.nth(index), profile.id + ': container ' + container + ' child ' + index)
}

function contrast(color, ground) {
  const luminance = value => value.match(/[\d.]+/g).slice(0, 3).map(Number).map(v => v / 255).map(v => v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4).reduce((sum, v, i) => sum + v * [0.2126, 0.7152, 0.0722][i], 0)
  const a = luminance(color), b = luminance(ground)
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
}

async function themes(page, id, scalar = false) {
  const colors = []
  if (scalar) await openClient(page)
  for (const theme of ['ledger', 'vault', 'hc']) {
    await page.evaluate(theme => document.documentElement.setAttribute('data-theme', theme), theme)
    await page.waitForTimeout(80)
    const palette = await page.evaluate(scalar => { const node = scalar ? document.querySelector('.references-rendered h1') : document.querySelector('main h1'); const body = getComputedStyle(document.body); return { color: getComputedStyle(node).color, ground: body.backgroundColor, bodyColor: body.color, theme: document.documentElement.dataset.theme } }, scalar)
    colors.push(palette); verify(id + ': ' + theme + ' heading contrast', contrast(palette.color, palette.ground) >= 4.5, palette)
    if (scalar) await checkApiTheme(page, { check: (name, passed, detail) => verify(id + ': ' + theme + '/' + name, passed, detail) })
    const width = await page.evaluate(() => ({ width: innerWidth, scroll: document.documentElement.scrollWidth })); verify(id + ': ' + theme + ' width', width.scroll <= width.width, width)
  }
  verify(id + ': three distinct canonical body themes', new Set(colors.map(color => color.ground)).size === 3, colors)
  await page.evaluate(() => document.documentElement.setAttribute('data-theme', 'ledger'))
  await page.emulateMedia({ reducedMotion: 'reduce', forcedColors: 'active' })
  verify(id + ': system motion and colors media', await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches && matchMedia('(forced-colors: active)').matches))
  const animation = await page.locator('main *').evaluateAll(nodes => nodes.filter(node => { const s = getComputedStyle(node); return node.getClientRects().length && s.animationName !== 'none' && s.animationDuration.split(',').some(value => parseFloat(value) > 0.01) }).map(node => node.className))
  verify(id + ': reduced motion no long animation', animation.length === 0, animation)
  if (scalar) await checkApiTheme(page, { check: (name, passed, detail) => verify(id + ': forced colors/' + name, passed, detail) })
  if (scalar) { await page.keyboard.press('Escape'); await page.getByRole('dialog', { name: 'API Client', exact: true }).waitFor({ state: 'hidden' }) }
  const skip = page.locator('a[href="#main"]').first(); await skip.focus()
  verify(id + ': forced colors visible keyboard focus', await skip.evaluate(node => { const r = node.getBoundingClientRect(), s = getComputedStyle(node); return document.activeElement === node && r.width > 1 && r.height > 1 && s.clipPath === 'none' && s.color !== s.backgroundColor && ((s.outlineStyle !== 'none' && parseFloat(s.outlineWidth) > 0) || (s.borderTopStyle !== 'none' && parseFloat(s.borderTopWidth) > 0)) }))
  await page.emulateMedia({ reducedMotion: 'no-preference', forcedColors: 'none' })
  return colors
}

async function openClient(page) {
  const operation = page.getByRole('link', { name: /^ListConflicts/ }).first()
  if (!(await operation.isVisible())) { await page.getByRole('button', { name: 'Open Menu', exact: true }).focus(); await page.keyboard.press('Enter') }
  await operation.focus(); await page.keyboard.press('Enter')
  const button = page.getByRole('button', { name: 'Test Request (get /v1/conflicts)', exact: true }); await button.waitFor()
  await button.focus(); await page.keyboard.press('Enter'); await page.getByRole('dialog', { name: 'API Client', exact: true }).waitFor()
}

function observe(page, log, expected = []) {
  page.on('pageerror', error => log.page_errors.push(String(error)))
  page.on('console', message => { if (['error', 'warning'].includes(message.type())) log.console.push({ type: message.type(), text: message.text() }) })
  page.on('request', req => log.requests.push({ url: req.url(), method: req.method(), type: req.resourceType() }))
  page.on('requestfailed', req => { const failure = { url: req.url(), error: req.failure()?.errorText }; failure.intentional = failure.error === 'net::ERR_ABORTED' && expected.some(item => req.url() === new URL(item.path, base).href && (item.abort || log.http_failures.some(response => response.url === req.url() && response.intentional))); log.network_failures.push(failure); if (!failure.intentional) log.network.push(failure) })
  page.on('response', response => { if (response.status() >= 400) { const failure = { url: response.url(), status: response.status(), intentional: expected.some(item => response.url().endsWith(item.path) && response.status() === item.status) }; log.http_failures.push(failure); if (!failure.intentional) log.network.push(failure) } })
}

function cleanNetwork(log, id, expectedErrors = false) {
  verify(id + ': zero runtime errors', log.page_errors.length === 0, log.page_errors)
  verify(id + ': console only explicit negative controls', expectedErrors ? log.console.length === log.http_failures.filter(item => item.intentional).length && log.console.every(item => /404|503/.test(item.text)) : log.console.length === 0, log.console)
  verify(id + ': no unexpected HTTP or request failures', log.http_failures.every(item => item.intentional) && log.network_failures.every(item => item.intentional), { http: log.http_failures, requests: log.network_failures })
  verify(id + ': read-only same-origin resources', log.requests.every(item => ['GET', 'HEAD'].includes(item.method) && new URL(item.url).origin === new URL(base).origin), log.requests)
}

async function safeRequests(context, log) {
  await context.route('**/*', route => {
    const req = route.request(), permitted = ['GET', 'HEAD'].includes(req.method()) && new URL(req.url()).origin === new URL(base).origin
    if (!permitted) { log.network.push({ denied: true, url: req.url(), method: req.method() }); return route.abort() }
    return route.continue()
  })
}

async function nativeContractTargets(page) {
  return page.locator('#static-api-reference [data-api-operation] details:has(> pre),#api-model-schemas [data-api-schema] > details').evaluateAll(nodes => nodes.map(node => {
    const pre = node.querySelector(':scope > pre')
    return { selector: pre.id ? '#' + CSS.escape(pre.id) : '#' + CSS.escape(node.parentElement.id) + ' > details > pre', operation: Boolean(node.closest('[data-api-operation]')) }
  }))
}

async function nativeContracts(page, profile, observation) {
  const targets = await nativeContractTargets(page)
  const count = targets.length
  verify(profile.id + ': all 491 native operation and model disclosures', count === 491, { count })
  for (const [index, target] of targets.entries()) {
    const pre = page.locator(target.selector), details = pre.locator('xpath=..')
    const outer = details.locator('xpath=ancestor::details[@data-api-operation-details]')
    if (index % 100 === 0) console.log('presentation browser: ' + profile.id + ' native contracts ' + index + '/' + count)
    if (target.operation && await outer.getAttribute('open') !== null) { await outer.locator(':scope > summary').focus(); await page.keyboard.press('Enter') }
    await reveal(page, details, profile.js)
    if (target.operation) verify(profile.id + ': native contract ' + index + ': outer keyboard expansion', await outer.getAttribute('open') !== null)
    const summary = details.locator(':scope > summary'), code = pre.locator('code')
    if (await details.getAttribute('open') !== null) { await summary.focus(); await page.keyboard.press('Enter') }
    await summary.focus(); await page.keyboard.press('Enter')
    const width = await page.evaluate(() => ({ viewport: innerWidth, document: document.documentElement.scrollWidth }))
    verify(profile.id + ': native contract ' + index + ': keyboard expansion', await details.getAttribute('open') !== null && await code.isVisible() && width.document <= width.viewport, { summary: await summary.textContent(), width })
    await hitTest(page, pre, profile.id + ': native contract ' + index, true)
    await summary.focus(); await page.keyboard.press('Enter')
    verify(profile.id + ': native contract ' + index + ': keyboard collapse', await details.getAttribute('open') === null && !(await code.isVisible()))
    if (target.operation) {
      await outer.locator(':scope > summary').focus(); await page.keyboard.press('Enter')
      verify(profile.id + ': native contract ' + index + ': outer keyboard collapse', await outer.getAttribute('open') === null && !(await code.isVisible()))
    }
  }
  observation.api_native_contracts = count
}

async function apiChecks(page, context, profile, spec, observation, log) {
  const callback = prefix => ({ check: (name, passed, detail) => verify(profile.id + ': ' + prefix + '/' + name, passed, detail) })
  await checkApiReferenceFallback(page, spec, callback('initial fallback'))
  observation.api = { operations: await page.locator('[data-api-operation]').count(), models: await page.locator('[data-api-schema]').count() }
  verify(profile.id + ': exact 96 operations and 395 model definitions', observation.api.operations === 96 && observation.api.models === 395, observation.api)
  await nativeContracts(page, profile, observation)
  if (profile.js) {
    await checkApiEntryKeyboard(page, callback('API keyboard entry'))
    await checkApiEntryReentry(page, callback('API native reentry'))
    await checkApiEntryRouteDuringLoading(page, { ...callback('API route during loading'), hold: () => holdContract(page, log, 'route-during-load') })
    await checkApiLandmarks(page, callback('online API lifecycle'))
    observation.scalar_themes = await themes(page, profile.id + ': Scalar', true)
    const before = log.requests.length
    await context.setOffline(true)
    await checkApiLandmarks(page, callback('loaded offline API lifecycle'))
    observation.offline = { before, after: log.requests.length }
    verify(profile.id + ': loaded offline lifecycle requires zero requests', log.requests.length === before, observation.offline)
    await context.setOffline(false)
  }
  const fragments = await page.locator('#static-api-reference [id^="api-operation-"],#static-api-reference [id^="api-schema-"]').evaluateAll(nodes => [nodes[0].id, nodes.find(node => node.id.startsWith('api-schema-')).id, nodes.at(-1).id])
  for (const fragment of fragments) { await page.goto(base + 'docs.html#' + fragment, { waitUntil: 'networkidle' }); await checkStaticApiDeepLink(page, callback('deep link ' + fragment)); await semantics(page, profile.id + ': deep link ' + fragment) }
}

async function modeRun(profile, spec) {
  const context = await browser.newContext({ viewport: profile.viewport, javaScriptEnabled: profile.js, isMobile: profile.mobile, hasTouch: profile.mobile, serviceWorkers: 'block' }), page = await context.newPage()
  const log = { id: profile.id, viewport: profile.viewport, javaScriptEnabled: profile.js, observations: {}, console: [], network: [], requests: [], page_errors: [], network_failures: [], http_failures: [] }; receipt.modes.push(log); observe(page, log)
  await safeRequests(context, log)
  verify(profile.id + ': fresh context no cookies', (await context.cookies()).length === 0)
  for (const entry of ['/', ...config.routes]) await bounded(profile.id + '/' + entry, async () => {
    console.log('presentation browser: ' + profile.id + '/' + entry)
    const route = entry === '/' ? 'index.html' : entry, observation = { route: entry }; log.observations[entry] = observation
    await page.goto(new URL(entry === '/' ? '' : entry, base).href, { waitUntil: 'networkidle' })
    observation.initial = await semantics(page, profile.id + '/' + entry + ': initial')
    observation.geometry = await framing(page, route, { ...profile, id: profile.id + '/' + entry })
    if (route === 'docs.html') { await apiChecks(page, context, profile, spec, observation, log); await page.goto(base + route, { waitUntil: 'networkidle' }) }
    observation.themes = await themes(page, profile.id + '/' + entry)
    await nativeContent(page, route, { ...profile, id: profile.id + '/' + entry }, observation)
    await semantics(page, profile.id + '/' + entry + ': after native content')
  })
  cleanNetwork(log, profile.id); await context.close()
}

async function holdContract(page, log, failure) {
  let received, release
  const held = new Promise(resolve => { received = resolve }), unlocked = new Promise(resolve => { release = resolve })
  await page.route(new URL('openapi.json', base).href, async route => {
    const response = await route.fetch(), bytes = await response.body()
    log.held_contract = { status: response.status(), digest: sha(bytes), size: bytes.length }
    verify(log.id + ': cold contract exact actual held response', response.status() === 200 && bytes.equals(fs.readFileSync(path.join(root, 'openapi.json'))), log.held_contract)
    received(); await unlocked
    try { await route.fulfill(failure === 'host-failure' ? { status: 404, body: 'Deliberate held host-failure control' } : { response, body: bytes }) } catch (error) {
      log.cancelled_route = String(error)
      verify(log.id + ': cold response release without unexpected route error', false, String(error))
    }
  })
  return { held: () => held, release: () => release() }
}

async function negativeControl(profile, spec, failure) {
  const context = await browser.newContext({ viewport: profile.viewport, javaScriptEnabled: true, isMobile: profile.mobile, hasTouch: profile.mobile, serviceWorkers: 'block' }), page = await context.newPage()
  const cold = ['focus-return', 'cancel', 'forward-focus', 'host-failure'].includes(failure)
  const expected = failure === 'host-failure' ? [{ path: '/openapi.json', status: 404 }] : failure === 'forward-focus' ? [] : cold ? [{ path: '/openapi.json', abort: true }] : failure === 'script' ? [{ path: '/scalar.js', status: 503 }] : failure === '404' ? [{ path: '/openapi.json', status: 404 }] : []
  const log = { id: profile.id + ':' + failure, console: [], network: [], requests: [], page_errors: [], network_failures: [], http_failures: [] }; receipt.negative_controls.push(log); observe(page, log, expected)
  await safeRequests(context, log)
  const control = cold ? await holdContract(page, log, failure) : null
  if (failure === 'script') await context.route(new URL('scalar.js', base).href, route => route.fulfill({ status: 503, body: 'Deliberate failed-script control' }))
  else if (!cold) await context.route(new URL('openapi.json', base).href, route => route.fulfill({ status: failure === '404' ? 404 : 200, contentType: 'application/json', body: failure === '404' ? 'Missing contract control' : fs.readFileSync(path.join(root, 'openapi.json'), 'utf8') + '\n' }))
  await page.goto(base + 'docs.html', { waitUntil: 'networkidle' })
  const check = (name, passed, detail) => verify(log.id + ': ' + name, passed, detail)
  if (['forward-focus', 'host-failure'].includes(failure)) await checkApiEntryForwardFocus(page, { ...control, check, failed: failure === 'host-failure' })
  else if (cold) await checkApiEntryCancellation(page, { kind: failure, ...control, check })
  else {
    if (failure !== 'script') { await page.locator('#interactive-api-view > summary').focus(); await page.keyboard.press('Enter'); await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'failed') }
    if (failure !== 'script') await checkApiEntryFallbackFocus(page, { check })
    await checkApiReferenceFallback(page, spec, { failed: failure !== 'script', check })
  }
  await semantics(page, log.id); cleanNetwork(log, log.id, failure === 'host-failure' || !cold && failure !== 'tamper'); await context.close()
}

async function run() {
  stageTools(); bindSubjects(); await startProxy(); await httpBinding()
  browser = await chromium.launch({ headless: true }); receipt.tools.chromium = browser.version()
  verify('toolchain: actual pinned Chromium', browser.version() === config.chromium, { actual: browser.version(), expected: config.chromium })
  await inventory(); await localLinks()
  const spec = JSON.parse(fs.readFileSync(path.join(root, 'openapi.json')))
  for (const id of config.modes) {
    const profile = { id, js: id.endsWith('-js'), mobile: id.startsWith('mobile'), viewport: id.startsWith('mobile') ? { width: 390, height: 844 } : { width: 1280, height: 900 } }
    await modeRun(profile, spec)
    if (profile.js) for (const failure of ['404', 'tamper', 'script', 'focus-return', 'cancel', 'forward-focus', 'host-failure']) await bounded(profile.id + ': negative ' + failure, () => negativeControl(profile, spec, failure))
  }
  verify('receipt: check IDs uniquely identify observations', new Set(receipt.checks.map(check => check.id)).size === receipt.checks.length)
  verify('receipt: complete fourteen negative controls', receipt.negative_controls.length === 14 && new Set(receipt.negative_controls.map(control => control.id)).size === 14)
  verify('proxy: no non-service origins requested', receipt.proxy_denials.length === 0, receipt.proxy_denials)
  const observed = new Set(receipt.checks.map(check => check.id))
  verify('receipt: every mandatory check actually ran', requiredIds().every(id => observed.has(id)), requiredIds().filter(id => !observed.has(id)))
}

async function finish() {
  if (browser) await browser.close(); if (api) await api.dispose(); if (proxy) await new Promise(resolve => proxy.close(resolve))
  clearTimeout(deadline); writeReceipt()
  console.log(JSON.stringify({ status: receipt.status, checks: receipt.checks.length, failures: receipt.failures }, null, 2))
  if (args['report-only'] !== 'true' && receipt.status !== 'passed') process.exitCode = 1
}

function writeReceipt() {
  receipt.status = receipt.failures.length ? 'failed' : 'passed'
  receipt.execution = { deadline_ms: deadlineMs, elapsed_ms: performance.now() - started }
  fs.mkdirSync(output, { recursive: true }); fs.writeFileSync(receiptPath, JSON.stringify(receipt, null, 2) + '\n')
}

if (args['require-pass'] === 'true') admitReceipt()
else {
  deadline = setTimeout(() => {
    verify('producer: bounded twenty-five minute deadline', false, { limit_ms: deadlineMs })
    writeReceipt(); process.exit(args['report-only'] === 'true' ? 0 : 1)
  }, deadlineMs)
  await run().catch(error => verify('producer: fatal error', false, String(error.stack || error))).finally(finish)
}
