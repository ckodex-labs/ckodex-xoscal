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
