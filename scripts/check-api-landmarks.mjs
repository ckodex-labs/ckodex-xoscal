/**
 * Dependency-free Playwright assertions for the actual built documentation.
 * The caller supplies its isolated page and evidence recorder. This module
 * neither launches a browser nor sends an API request, edits DOM or repairs it.
 */
function recorder(check) {
  const results = []
  return { results, verify(name, passed, detail) {
    const record = { name, passed: Boolean(passed), detail }
    results.push(record)
    if (check) check(name, record.passed, detail)
    if (!record.passed) throw Error('API reference regression: ' + name)
  } }
}

function canonical(value) {
  if (Array.isArray(value)) return value.map(canonical)
  if (value && typeof value === 'object') return Object.fromEntries(Object.keys(value).sort().map(key => [key, canonical(value[key])]))
  return value
}

/** Complete contracts must already be HTML, including no-JS/failed-JS pages. */
export async function checkStaticApiReference(page, spec, { check } = {}) {
  const { results, verify } = recorder(check)
  const plain = page.locator('#plain-api-view')
  if (await plain.getAttribute('open') === null) {
    await page.locator('#plain-api-view > summary').focus()
    await page.keyboard.press('Enter')
  }
  const methods = new Set(['get', 'post', 'put', 'patch', 'delete', 'head', 'options', 'trace'])
  const expected = []
  for (const [path, item] of Object.entries(spec.paths)) {
    for (const [method, operation] of Object.entries(item)) {
      if (methods.has(method)) expected.push({ id: operation.operationId, method: method.toUpperCase(), path, operation })
    }
  }
  const actual = await page.locator('#static-api-reference article[data-api-operation]').evaluateAll(nodes =>
    nodes.map(node => ({ id: node.dataset.apiOperation, method: node.dataset.apiMethod, path: node.dataset.apiPath,
      operation: JSON.parse(node.querySelector('details pre code').textContent) })))
  const normalized = entries => JSON.stringify(canonical(entries.sort((a, b) => a.id.localeCompare(b.id))))
  verify('complete static operation identities and contracts', normalized(actual) === normalized(expected), { expected: expected.length, actual: actual.length })
  const schemas = await page.locator('#api-model-schemas [data-api-schema]').evaluateAll(nodes => nodes.map(node =>
    [node.dataset.apiSchema, JSON.parse(node.querySelector('pre code').textContent)]))
  const expectedSchemas = Object.entries(spec.components.schemas)
  verify('complete static model names and contracts', normalizedSchema(schemas) === normalizedSchema(expectedSchemas), { expected: expectedSchemas.length, actual: schemas.length })
  verify('static page has exactly one canonical main', await page.locator('main,[role="main"]').count() === 1 && await page.locator('main#main').count() === 1)
  const details = page.locator('#static-api-reference article details').first()
  if (await details.getAttribute('open') !== null) await details.locator(':scope > summary').click()
  await details.locator(':scope > summary').focus()
  await page.keyboard.press('Enter')
  verify('static contract expands using native keyboard disclosure', await details.getAttribute('open') !== null)
  verify('static contract expansion fits the viewport', await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  return results
}

function normalizedSchema(entries) {
  return JSON.stringify(canonical(entries.sort(([a], [b]) => a.localeCompare(b))))
}

async function entryGeometry(page, plain = false) {
  // Observe only: neither scrolling nor focusing can hide an entry regression.
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
  return page.evaluate(plain => {
    const active = document.activeElement, rect = active.getBoundingClientRect()
    const x = Math.max(0, Math.min(innerWidth - 1, rect.left + rect.width / 2))
    const y = Math.max(0, Math.min(innerHeight - 1, rect.top + rect.height / 2))
    const hit = document.elementFromPoint(x, y), heading = document.getElementById(plain ? 'static-reference-heading' : 'interactive-heading')
    const title = heading.getBoundingClientRect(), titleHit = document.elementFromPoint(title.left + title.width / 2, title.top + title.height / 2)
    return { target: active === document.querySelector((plain ? '#plain-api-view' : '#interactive-api-view') + ' > summary'),
      visible: rect.width > 0 && rect.height > 0 && rect.top >= 0 && rect.bottom <= innerHeight && rect.left >= 0 && rect.right <= innerWidth,
      unoccluded: Boolean(hit && (active === hit || active.contains(hit))), rect: rect.toJSON(), scrollY,
      headingVisible: title.width > 0 && title.height > 0 && title.top >= 0 && title.bottom <= innerHeight && title.left >= 0 && title.right <= innerWidth,
      headingUnoccluded: Boolean(titleHit && (heading === titleHit || heading.contains(titleHit))), heading: title.toJSON(), viewport: { width: innerWidth, height: innerHeight } }
  }, plain)
}

async function checkEntryGeometry(page, verify, kind, plain = false) {
  const samples = []
  for (const delay of [0, 250, 250, 500]) {
    await page.waitForTimeout(delay)
    samples.push(await entryGeometry(page, plain))
  }
  verify(kind + ': ready transition retains visible unoccluded keyboard focus', samples.every(geometry => geometry.target && geometry.visible && geometry.unoccluded && geometry.headingVisible && geometry.headingUnoccluded), samples)
}

/** The failed explorer must return its still-owned keyboard entry to Plain. */
export async function checkApiEntryFallbackFocus(page, { check } = {}) {
  const { results, verify } = recorder(check)
  await checkEntryGeometry(page, verify, 'failed explorer', true)
  return results
}

async function enterExplorer(page, verify, timeout, kind) {
  if (kind === 'view link') {
    await page.locator('#api-reference-views a[href="#plain-api-view"]').focus()
    await page.keyboard.press('Tab')
    verify(kind + ': actual Tab reaches explorer selector', await page.locator('#api-reference-views a[href="#interactive-api-view"]').evaluate(node => node === document.activeElement))
  } else {
    const models = page.locator('#api-model-definitions')
    if (await models.getAttribute('open') === null) {
      await models.locator(':scope > summary').focus()
      await page.keyboard.press('Enter')
    }
    const last = page.locator('#api-model-schemas [data-api-schema] > details').last()
    await last.locator(':scope > summary').focus()
    if (await last.getAttribute('open') === null) await page.keyboard.press('Enter')
    await page.keyboard.press('Tab')
    verify(kind + ': actual Tab reaches explorer summary after far Plain contract', await page.locator('#interactive-api-view > summary').evaluate(node => node === document.activeElement))
  }
  await page.keyboard.press('Enter')
  await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready', null, { timeout })
  await checkEntryGeometry(page, verify, kind)
  verify(kind + ': ready transition selects only interactive reference', await page.locator('#plain-api-view').getAttribute('open') === null)
}

/** Actual entry keys; geometry is measured before any verifier scroll repair. */
export async function checkApiEntryKeyboard(page, { check, timeout = 20000 } = {}) {
  const { results, verify } = recorder(check)
  await enterExplorer(page, verify, timeout, 'view link')
  await page.locator('#api-reference-views a[href="#interactive-api-view"]').focus()
  await page.keyboard.press('Shift+Tab')
  verify('plain view link: actual Shift+Tab reaches Plain selector', await page.locator('#api-reference-views a[href="#plain-api-view"]').evaluate(node => node === document.activeElement))
  await page.keyboard.press('Enter')
  await page.waitForFunction(() => !document.querySelector('#interactive-api-reference .references-rendered'))
  await checkEntryGeometry(page, verify, 'plain view link', true)
  await enterExplorer(page, verify, timeout, 'native summary')
  return results
}

/** Native close/reentry must not inherit Scalar's previous scroll-spy route. */
export async function checkApiEntryReentry(page, { check, timeout = 20000 } = {}) {
  const { results, verify } = recorder(check)
  for (const kind of ['intro reentry', 'model reentry']) {
    if (kind === 'model reentry') {
      await page.goto(page.url().split('#')[0] + '#models/oscalservicesv1CreateComponentDefinitionRequest', { waitUntil: 'networkidle' })
      await page.reload({ waitUntil: 'networkidle' })
      await page.waitForFunction(() => document.activeElement?.id === 'api-1/models/oscalservicesv1CreateComponentDefinitionRequest', null, { timeout })
      verify('initial model deep link preserves requested route', new URL(page.url()).hash === '#models/oscalservicesv1CreateComponentDefinitionRequest', { fragment: new URL(page.url()).hash })
      await checkModelRouteGeometry(page, verify, 'initial model deep link')
    }
    const fragment = new URL(page.url()).hash
    verify(kind + ': actual prior Scalar fragment', kind === 'intro reentry' ? fragment === '#description/introduction' : fragment === '#models/oscalservicesv1CreateComponentDefinitionRequest', { fragment })
    // Controlled pre-close setup preserves the genuine Scalar model route.
    await page.locator('#interactive-api-view > summary').focus({ preventScroll: true })
    const setup = await page.locator('#interactive-api-view > summary').evaluate(node => ({ fragment: location.hash, summaryFocused: node === document.activeElement }))
    verify(kind + ': pre-close setup preserves fragment and summary focus', setup.fragment === fragment && setup.summaryFocused, setup)
    await page.keyboard.press('Enter')
    await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'idle')
    verify(kind + ': native close retains prior fragment', new URL(page.url()).hash === fragment, { fragment: new URL(page.url()).hash })
    await enterExplorer(page, verify, timeout, kind)
  }
  return results
}

async function checkModelRouteGeometry(page, verify, kind) {
  const samples = []
  for (const delay of [0, 250, 250, 500]) {
    await page.waitForTimeout(delay)
    samples.push(await page.evaluate(() => {
      const node = document.activeElement, r = node.getBoundingClientRect(), hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2)
      return { target: node.id === 'api-1/models/oscalservicesv1CreateComponentDefinitionRequest', hash: location.hash,
        visible: r.width > 0 && r.height > 0 && r.top >= 0 && r.bottom <= innerHeight && r.left >= 0 && r.right <= innerWidth,
        unoccluded: Boolean(hit && (node === hit || node.contains(hit))), rect: r.toJSON(), viewport: { width: innerWidth, height: innerHeight } }
    }))
  }
  verify(kind + ': requested model owns visible focus after readiness', samples.every(s => s.target && s.hash === '#models/oscalservicesv1CreateComponentDefinitionRequest' && s.visible && s.unoccluded), samples)
}

/** A newer actual hash navigation during held loading remains authoritative. */
export async function checkApiEntryRouteDuringLoading(page, { check, hold, timeout = 20000 } = {}) {
  const { results, verify } = recorder(check), documentURL = page.url().split('#')[0]
  await page.goto(documentURL, { waitUntil: 'networkidle' })
  await page.reload({ waitUntil: 'networkidle' })
  const control = await hold()
  try {
    await page.locator('#interactive-api-view > summary').focus()
    await page.keyboard.press('Enter')
    await control.held()
    const phase = await page.locator('#interactive-api-view').getAttribute('data-enhancement')
    verify('new route: actual hash navigation occurs during loading', phase === 'loading', { phase, requested: '#models/oscalservicesv1CreateComponentDefinitionRequest' })
    await page.goto(documentURL + '#models/oscalservicesv1CreateComponentDefinitionRequest', { waitUntil: 'domcontentloaded' })
    control.release()
    await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready' && document.activeElement?.id === 'api-1/models/oscalservicesv1CreateComponentDefinitionRequest', null, { timeout })
    await checkModelRouteGeometry(page, verify, 'new route')
  } finally {
    control.release()
    await page.unroute(new URL('openapi.json', documentURL).href)
  }
  return results
}

/** Fresh cold-contract control: caller holds only its exact served response. */
export async function checkApiEntryCancellation(page, { check, kind, held, release } = {}) {
  const { results, verify } = recorder(check)
  const plain = page.locator('#plain-api-view > summary'), interactive = page.locator('#interactive-api-view > summary')
  const models = page.locator('#api-model-definitions')
  await models.locator(':scope > summary').focus()
  await page.keyboard.press('Enter')
  const last = page.locator('#api-model-schemas [data-api-schema] > details').last().locator(':scope > summary')
  await last.focus()
  await page.keyboard.press('Enter')
  await page.keyboard.press('Tab')
  verify('cold contract: actual Tab enters explorer summary', await interactive.evaluate(node => node === document.activeElement))
  await page.keyboard.press('Enter')
  await held()
  const key = kind === 'focus-return' ? 'Tab' : 'Enter'
  const phase = await observeEntryKey(page, key === 'Tab' ? 'Shift+Tab' : key, key)
  verify('cold contract: cancellation keydown occurs during loading', phase.phase === 'loading' && phase.key === key, phase)
  const focused = await page.evaluateHandle(() => document.activeElement)
  await release()
  await page.waitForTimeout(1000)
  verify('cold contract: cancelled explorer never resurrects', await page.evaluate(() => document.getElementById('plain-api-view').open && !document.getElementById('interactive-api-view').open && !document.querySelector('#interactive-api-reference .references-rendered')))
  if (kind === 'focus-return') {
    const geometry = await focused.evaluate(node => {
      const r = node.getBoundingClientRect(), hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2)
      return { target: node !== document.body && document.activeElement === node && document.getElementById('plain-api-view').contains(node),
        visible: r.width > 0 && r.height > 0 && r.top >= 0 && r.bottom <= innerHeight && r.left >= 0 && r.right <= innerWidth,
        unoccluded: Boolean(hit && (node === hit || node.contains(hit))), rect: r.toJSON(), viewport: { width: innerWidth, height: innerHeight } }
    })
    verify('cold contract: moved Plain focus remains visible and owned', geometry.target && geometry.visible && geometry.unoccluded, geometry)
  } else await checkEntryGeometry(page, verify, 'cancel explorer', true)
  return results
}

/** Forward Tab selects an actual host link while the exact contract is held. */
export async function checkApiEntryForwardFocus(page, { check, held, release, failed = false } = {}) {
  const { results, verify } = recorder(check)
  await page.locator('#api-reference-views a[href="#plain-api-view"]').focus()
  await page.keyboard.press('Tab')
  verify('cold forward: actual Tab enters explorer selector', await page.locator('#api-reference-views a[href="#interactive-api-view"]').evaluate(node => node === document.activeElement))
  await page.keyboard.press('Enter')
  await held()
  const phase = await observeEntryKey(page, 'Tab', 'Tab')
  verify('cold forward: actual Tab keydown occurs during loading', phase.phase === 'loading' && phase.key === 'Tab', phase)
  const focused = await page.evaluateHandle(() => document.activeElement)
  const identity = await focused.evaluate(node => ({ tag: node.tagName, href: node.getAttribute('href'),
    host: document.getElementById('interactive-api-view').contains(node) && !document.getElementById('interactive-api-reference').contains(node) }))
  verify('cold forward: actual Tab selects host contract link', identity.tag === 'A' && identity.href === 'openapi.json' && identity.host, identity)
  await release()
  if (failed) {
    await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'failed')
    const state = await page.evaluate(() => ({ plain: document.getElementById('plain-api-view').open, interactive: document.getElementById('interactive-api-view').open, mounted: Boolean(document.querySelector('#interactive-api-reference .references-rendered')) }))
    verify('cold host failure: selects only Plain and disposes explorer', state.plain && !state.interactive && !state.mounted, state)
    await checkEntryGeometry(page, verify, 'failed host explorer', true)
    return results
  }
  await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready', null, { timeout: 40000 })
  const samples = []
  for (const delay of [0, 250, 250, 500]) {
    await page.waitForTimeout(delay)
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
    samples.push(await focused.evaluate(node => {
      const r = node.getBoundingClientRect(), hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2)
      return { target: document.activeElement === node && node.tagName === 'A' && node.getAttribute('href') === 'openapi.json',
        visible: r.width > 0 && r.height > 0 && r.top >= 0 && r.bottom <= innerHeight && r.left >= 0 && r.right <= innerWidth,
        unoccluded: Boolean(hit && (node === hit || node.contains(hit))), rect: r.toJSON(), viewport: { width: innerWidth, height: innerHeight }, scrollY }
    }))
  }
  verify('cold forward: ready retains visible unoccluded owned host focus', samples.every(s => s.target && s.visible && s.unoccluded), samples)
  return results
}

async function observeEntryKey(page, press, key) {
  // Capture actual keydown state, not a Playwright read that can race rendering.
  // This instrumentation observes only; it never changes focus, DOM or routing.
  await page.evaluate(key => {
    const record = event => {
      if (event.key !== key) return
      window.__xoscalEntryKeyObservation = { key: event.key, phase: document.getElementById('interactive-api-view').dataset.enhancement }
      window.removeEventListener('keydown', record, true)
    }
    window.addEventListener('keydown', record, true)
  }, key)
  await page.keyboard.press(press)
  return page.evaluate(() => {
    const result = window.__xoscalEntryKeyObservation
    delete window.__xoscalEntryKeyObservation
    return result
  })
}

/** Run on the initial page or a deliberately failed enhancement fixture. */
export async function checkApiReferenceFallback(page, spec, { check, failed = false } = {}) {
  const { results, verify } = recorder(check)
  verify('complete Plain HTML is the initial or failed-enhancement view', await page.locator('#plain-api-view').getAttribute('open') !== null)
  verify('interactive view starts closed or returns closed after failure', await page.locator('#interactive-api-view').getAttribute('open') === null)
  verify('inactive explorer has no mounted or residual Scalar presentation', await page.locator('#interactive-api-reference .references-rendered').count() === 0)
  if (failed) {
    verify('failed explorer exposes an explicit failure state', await page.locator('#interactive-api-view').getAttribute('data-enhancement') === 'failed')
    verify('failure status offers the complete reference and retry', /unavailable.*complete Plain HTML.*retry/.test(await page.locator('#scalar-status').textContent()))
  }
  await checkStaticApiReference(page, spec, { check: (name, passed, detail) => verify(name, passed, detail) })
  return results
}

/** The caller navigates to an operation/model fragment with or without JS. */
export async function checkStaticApiDeepLink(page, { check } = {}) {
  const { results, verify } = recorder(check)
  const state = await page.evaluate(() => {
    const target = document.getElementById(decodeURIComponent(location.hash.slice(1)))
    const ancestors = []
    for (let node = target; node; node = node.parentElement) if (node.tagName === 'DETAILS') ancestors.push(node.open)
    return { exists: Boolean(target), disclosures: ancestors.length, open: ancestors.every(Boolean), plain: document.getElementById('plain-api-view').open }
  })
  verify('static deep link targets a prebuilt operation or model contract', state.exists && state.disclosures >= 2, state)
  verify('static deep link exposes all native disclosure ancestors', state.open && state.plain, state)
  return results
}

async function openInteractiveReference(page, verify, timeout) {
  if (await page.locator('#interactive-api-view').getAttribute('open') === null) {
    await page.locator('#interactive-api-view > summary').focus()
    await page.keyboard.press('Enter')
  }
  await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready', null, { timeout })
  verify('ready interactive view does not duplicate visible static content', await page.locator('#plain-api-view').getAttribute('open') === null)
  verify('interactive navigation exposes every generated operation once', await page.locator('#interactive-api-reference a[href]').evaluateAll(nodes => {
    const actual = nodes.map(node => node.getAttribute('href').match(/^#tag\/[^/]+\/(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|TRACE)(\/.*)$/))
      .filter(Boolean).map(match => JSON.stringify([match[1], decodeURIComponent(match[2])])).sort()
    const expected = [...document.querySelectorAll('#static-api-reference [data-api-operation]')]
      .map(node => JSON.stringify([node.dataset.apiMethod, node.dataset.apiPath])).sort()
    return actual.length === 96 && JSON.stringify(actual) === JSON.stringify(expected)
  }))
  const region = page.locator('section.references-rendered[aria-label]')
  await region.waitFor({ timeout })
  verify('Scalar is a named documentation region', await region.count() === 1 &&
    Boolean((await region.getAttribute('aria-label')).trim()))
  return region
}

async function checkMainLandmarks(page, verify, phase) {
  verify(phase + ': exactly one canonical main', await page.locator('main,[role="main"]').count() === 1 && await page.locator('main#main').count() === 1)
  verify(phase + ': no nested main landmark', await page.locator('main main,main [role="main"]').count() === 0)
  verify(phase + ': no duplicate DOM IDs', await page.locator('[id]').evaluateAll(nodes => {
    const ids = nodes.map(node => node.id)
    return new Set(ids).size === ids.length
  }))
}

async function openApiClient(page, operation, timeout) {
  const link = page.getByRole('link', { name: operation.name }).first()
  if (!(await link.isVisible())) {
    const menu = page.getByRole('button', { name: 'Open Menu', exact: true })
    await menu.focus()
    await page.keyboard.press('Enter')
  }
  await link.focus()
  await page.keyboard.press('Enter')
  const button = page.getByRole('button', { name: operation.button, exact: true })
  await button.waitFor({ timeout })
  await button.focus()
  await page.keyboard.press('Enter')
  const dialog = page.getByRole('dialog', { name: 'API Client', exact: true })
  await dialog.waitFor({ timeout })
  return { button, dialog }
}

async function checkFocusWrap(page, dialog, verify, index) {
  const close = dialog.getByRole('button', { name: 'Close Client', exact: true })
  await close.focus()
  await page.keyboard.press('Tab')
  verify('cycle ' + index + ': Tab wraps from the final close control into the dialog', await dialog.evaluate(node => node.contains(document.activeElement)) && !(await close.evaluate(node => document.activeElement === node)))
  await page.keyboard.press('Shift+Tab')
  verify('cycle ' + index + ': Shift+Tab wraps back to the final close control', await close.evaluate(node => document.activeElement === node))
}

async function checkRequestDisclosures(page, dialog, verify, index) {
  for (const name of [/^Cookies/, /^Headers/, /^Query Parameters/]) {
    const disclosure = dialog.getByRole('button', { name }).first()
    const panelId = await disclosure.getAttribute('aria-controls')
    verify('cycle ' + index + ': ' + name + ' controls exactly one panel', await page.locator('[id]').evaluateAll((nodes, id) => nodes.filter(node => node.id === id).length, panelId) === 1)
    await disclosure.focus()
    await page.keyboard.press('Enter')
    verify('cycle ' + index + ': ' + name + ' collapses by keyboard', await disclosure.getAttribute('aria-expanded') === 'false')
    await page.keyboard.press('Enter')
    verify('cycle ' + index + ': ' + name + ' reopens by keyboard', await disclosure.getAttribute('aria-expanded') === 'true')
  }
}

async function escapeApiClient(page, dialog, button, verify, index, timeout) {
  await page.keyboard.press('Tab')
  verify('cycle ' + index + ': keyboard focus remains in dialog', await dialog.evaluate(node => node.contains(document.activeElement)))
  // A focused request control can own a visible tooltip. Its first Escape
  // dismisses that inner presentation; the modal retains focus trapping.
  const tooltip = page.locator('#scalar-tooltip')
  if (await page.evaluate(() => document.activeElement.getAttribute('aria-describedby') === 'scalar-tooltip'))
    await tooltip.waitFor({ state: 'visible', timeout })
  if (await tooltip.isVisible()) {
    await page.keyboard.press('Escape')
    await tooltip.waitFor({ state: 'hidden', timeout })
    verify('cycle ' + index + ': Escape dismisses the nested tooltip first', await dialog.isVisible() && await dialog.evaluate(node => node.contains(document.activeElement)))
  }
  await page.keyboard.press('Escape')
  await dialog.waitFor({ state: 'hidden', timeout })
  verify('cycle ' + index + ': Escape dismisses the API client', !(await dialog.isVisible()))
  await page.waitForFunction(node => document.activeElement === node, await button.elementHandle(), { timeout })
  verify('cycle ' + index + ': dismissal returns focus to the operation trigger', await button.evaluate(node => document.activeElement === node))
}

async function checkApiClientCycle(page, operation, index, verify, timeout) {
  const { button, dialog } = await openApiClient(page, operation, timeout)
  verify('cycle ' + index + ': API client is a visible labelled dialog', await dialog.isVisible())
  await checkMainLandmarks(page, verify, 'cycle ' + index + ' open')
  verify('cycle ' + index + ': dialog contains no main', await dialog.locator('main,[role="main"]').count() === 0)
  await checkFocusWrap(page, dialog, verify, index)
  await checkRequestDisclosures(page, dialog, verify, index)
  await escapeApiClient(page, dialog, button, verify, index, timeout)
  await checkMainLandmarks(page, verify, 'cycle ' + index + ' dismissed, including hidden DOM')
}

async function switchReferenceViews(page, region, verify, timeout) {
  await page.locator('#plain-api-view > summary').focus()
  await page.keyboard.press('Enter')
  await page.waitForFunction(() => !document.querySelector('#interactive-api-reference .references-rendered'))
  verify('plain view switch destroys the inactive Scalar instance', await page.locator('#interactive-api-reference .references-rendered').count() === 0)
  verify('plain view switch retains all prebuilt operation content', await page.locator('#static-api-reference article[data-api-operation]').count() === 96)
  await checkMainLandmarks(page, verify, 'after switch to Plain HTML')
  await page.locator('#interactive-api-view > summary').focus()
  await page.keyboard.press('Enter')
  await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready', null, { timeout })
  verify('interactive view can be reopened after disposal', await page.locator('#plain-api-view').getAttribute('open') === null && await region.count() === 1)
  await checkMainLandmarks(page, verify, 'after reopening interactive explorer')
}

/** Run after the real Scalar bundle has loaded on either desktop or mobile. */
export async function checkApiLandmarks(page, { check, timeout = 20000 } = {}) {
  const { results, verify } = recorder(check)
  const region = await openInteractiveReference(page, verify, timeout)
  await checkMainLandmarks(page, verify, 'before API-client interaction')
  const operations = [
    { name: /^ListConflicts/, button: 'Test Request (get /v1/conflicts)' },
    { name: /^ListSnapshots/, button: 'Test Request (get /v1/snapshots)' },
    { name: /^ListConflicts/, button: 'Test Request (get /v1/conflicts)' },
  ]
  for (const [index, operation] of operations.entries())
    await checkApiClientCycle(page, operation, index, verify, timeout)
  await switchReferenceViews(page, region, verify, timeout)
  return results
}

/** Assert the current user-selected palette, including forced-colors fixtures. */
export async function checkApiTheme(page, { check } = {}) {
  const { results, verify } = recorder(check)
  for (const [name, locator] of [
    ['reference heading', page.locator('.references-rendered h1').first()],
    ['operation navigation', page.getByRole('link', { name: /^ListConflicts/ }).first()],
    ['API-client dialog', page.getByRole('dialog', { name: 'API Client', exact: true })],
  ]) {
    if (!(await locator.isVisible())) continue
    const colors = await locator.evaluate(node => {
      const parse = color => color.match(/[\d.]+/g).slice(0, 3).map(Number)
      const luminance = rgb => rgb.map(value => { const s = value / 255; return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4 })
        .reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0)
      const foreground = getComputedStyle(node).color
      let background = 'rgb(255, 255, 255)'
      for (let ancestor = node; ancestor; ancestor = ancestor.parentElement) {
        const color = getComputedStyle(ancestor).backgroundColor
        if (color !== 'rgba(0, 0, 0, 0)') { background = color; break }
      }
      const a = luminance(parse(foreground)), b = luminance(parse(background))
      return { foreground, background, contrast: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05) }
    })
    verify(name + ': actual foreground/background contrast meets 4.5:1', colors.contrast >= 4.5, colors)
  }
  return results
}
