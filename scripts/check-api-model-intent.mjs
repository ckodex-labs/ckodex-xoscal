/** Deliberate pre-ready focus movement; later samples only observe the page. */
export async function checkApiModelFocusAway(page, { check, hold, timeout = 20000 }) {
  const verify = (name, passed, detail) => {
    check(name, passed === true, detail)
    if (!passed) throw Error('API model focus movement: ' + name)
  }
  const url = page.url().split('#')[0]
  await page.goto('about:blank')
  const controlled = await hold()
  try {
    await page.goto(url, { waitUntil: 'networkidle' })
    await installFocusMovement(page)
    await page.locator('#interactive-api-view > summary').focus()
    await page.keyboard.press('Enter')
    await controlled.held()
    await page.goto(url + '#models/oscalservicesv1CreateComponentDefinitionRequest')
    controlled.release()
    await page.waitForFunction(() => window.__modelFocusMovement?.observation, null, { timeout })
    const movement = await page.evaluate(() => window.__modelFocusMovement.observation)
    verify('pre-ready host focus is observed', movement.phase === 'loading' && movement.modelHadFocus && movement.hostFocus &&
      movement.requested === '#models/oscalservicesv1CreateComponentDefinitionRequest', movement)
    await page.waitForFunction(() => document.getElementById('interactive-api-view').dataset.enhancement === 'ready', null, { timeout })
    const samples = []
    for (const delay of [0, 250, 250, 500]) {
      await page.waitForTimeout(delay)
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
      samples.push(await observeMovedFocus(page))
    }
    verify('ready preserves moved host focus', samples.every(s => s.ready && s.target && s.visible && s.unoccluded), samples)
  } finally { controlled.release(); await page.unroute(new URL('openapi.json', url).href) }
}

async function installFocusMovement(page) {
  await page.evaluate(() => {
    const state = { observation: null, target: null }
    window.__modelFocusMovement = state
    const move = event => {
      if (event.target.id !== 'api-1/models/oscalservicesv1CreateComponentDefinitionRequest') return
      document.removeEventListener('focusin', move)
      const model = event.target
      queueMicrotask(() => {
        state.target = document.querySelector('#interactive-api-view > section > p a[href="openapi.json"]')
        const modelHadFocus = document.activeElement === model
        state.target.focus({ preventScroll: true })
        state.observation = { phase: document.getElementById('interactive-api-view').dataset.enhancement,
          modelHadFocus, hostFocus: document.activeElement === state.target, requested: location.hash,
          tag: state.target.tagName, href: state.target.getAttribute('href'),
          host: document.getElementById('interactive-api-view').contains(state.target) && !document.getElementById('interactive-api-reference').contains(state.target) }
      })
    }
    document.addEventListener('focusin', move)
  })
}

async function observeMovedFocus(page) {
  return page.evaluate(() => {
    const node = window.__modelFocusMovement.target, rect = node.getBoundingClientRect()
    const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2)
    return { ready: document.getElementById('interactive-api-view').dataset.enhancement === 'ready',
      target: document.activeElement === node, hash: location.hash,
      visible: rect.width > 0 && rect.height > 0 && rect.top >= 0 && rect.bottom <= innerHeight && rect.left >= 0 && rect.right <= innerWidth,
      unoccluded: Boolean(hit && (hit === node || node.contains(hit))), rect: rect.toJSON(), viewport: { width: innerWidth, height: innerHeight } }
  })
}
