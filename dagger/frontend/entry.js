import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'
window.Scalar = { createApiReference }
// The complete reference is prebuilt HTML. Scalar is an optional presentation,
// admitted only after its already-generated contract matches those HTML inputs.
const marker = document.getElementById('api-reference')
if (marker) {
  const plain = document.getElementById('plain-api-view')
  const interactive = document.getElementById('interactive-api-view')
  const reference = document.getElementById('static-api-reference')
  const container = document.getElementById('interactive-api-reference')
  const status = document.getElementById('scalar-status')
  const links = [...document.querySelectorAll('#api-reference-views a')]
  const summary = interactive.querySelector(':scope > summary')
  let instance = null, configuration = null, loading = false, ready = false
  let timer = null, controller = null, attempt = 0, cachedSource = null
  let entryIntent = null, modelIntent = null
  const rememberEntry = () => {
    // Explicit host activation owns the entry, not a prior Scalar scroll-spy
    // fragment. Clear it before loading; subsequent route intent stays intact.
    if (scalarHash()) history.replaceState(null, '', location.pathname + location.search)
    summary.focus({ preventScroll: true })
    summary.scrollIntoView({ block: 'center', behavior: 'instant' })
    entryIntent = { target: summary, attempt: attempt + (ready || loading ? 0 : 1) }
  }
  const settleEntry = () => {
    const intent = entryIntent
    if (!intent) return
    const restore = () => {
      if (entryIntent !== intent || intent?.attempt !== attempt || !interactive.open ||
          !ready || document.activeElement !== intent.target) return
      // Closing the tall Plain disclosure changes Scalar's document geometry.
      // Keep the user's entry control visible before its observers sample it.
      intent.target.scrollIntoView({ block: 'center', behavior: 'instant' })
    }
    restore()
    requestAnimationFrame(() => requestAnimationFrame(restore))
    // Scalar lazily expands content above host footer controls. Bound the
    // compensation; any later user gesture or focus movement invalidates it.
    if (intent.host) for (const delay of [250, 500, 1000]) window.setTimeout(restore, delay)
  }
  const settleModel = () => {
    const intent = modelIntent, target = document.activeElement
    if (!intent || intent.settling || !ready || !target.id.endsWith('/' + intent.id)) return
    intent.settling = true
    intent.ownedTarget = target
    intent.expires = performance.now() + 1000
    const restore = () => {
      if (modelIntent !== intent || performance.now() > intent.expires || intent.attempt !== attempt || !ready ||
          !interactive.open || document.activeElement !== target) return
      // Scalar observes the viewport centre. Keep the requested model there
      // while lazy siblings settle; a short neighbour must not own its URL.
      target.scrollIntoView({ block: 'center', behavior: 'instant' })
      if (location.hash !== intent.hash)
        history.replaceState(null, '', location.pathname + location.search + intent.hash)
    }
    restore()
    requestAnimationFrame(() => requestAnimationFrame(restore))
    intent.observer = new ResizeObserver(restore)
    intent.observer.observe(container)
    for (const delay of [250, 500, 1000]) window.setTimeout(restore, delay)
    window.setTimeout(() => {
      intent.observer.disconnect()
      if (modelIntent === intent) modelIntent = null
    }, 1000)
  }
  const cancelEntry = () => {
    entryIntent = null
    modelIntent?.observer?.disconnect()
    modelIntent = null
  }
  for (const event of ['keydown', 'pointerdown', 'wheel', 'touchstart'])
    window.addEventListener(event, cancelEntry, { capture: true, passive: true })
  document.addEventListener('focusin', event => {
    if (entryIntent && event.target !== entryIntent.target) cancelEntry()
    const target = event.target
    if (modelIntent?.ownedTarget && target !== modelIntent.ownedTarget) cancelEntry()
    if (modelIntent && target.id.endsWith('/' + modelIntent.id)) modelIntent.ownedTarget = target
    if (loading && interactive.contains(target) && !container.contains(target) &&
        target !== summary && target.matches('a[href],button,input,select,textarea,summary,[tabindex]'))
      entryIntent = { target, attempt, host: true }
    // A user returning to the readable contract cancels optional loading;
    // its eventual ready callback must not close the newly focused disclosure.
    if (loading && plain.contains(event.target)) showPlain()
    if (modelIntent && target.id.endsWith('/' + modelIntent.id)) settleModel()
  })
  const darkState = () => ['vault', 'hc'].includes(document.documentElement.dataset.theme) ? 'dark' : 'light'
  const scalarHash = () => /^#(?:tag|model|models|operation|description)(?:\/|$)/.test(location.hash)
  const current = id => {
    for (const link of links) {
      if (link.hash === '#' + id) link.setAttribute('aria-current', 'true')
      else link.removeAttribute('aria-current')
    }
  }
  const showPlain = () => {
    const focused = document.activeElement
    const returnFocus = focused === summary || (interactive.contains(focused) &&
      !container.contains(focused) && focused.matches('a[href],button,input,select,textarea,summary,[tabindex]'))
    cancelEntry()
    if (loading || instance) {
      attempt += 1
      controller?.abort()
      window.clearTimeout(timer)
      loading = false
      ready = false
      instance?.destroy()
      instance = null
      container.setAttribute('aria-busy', 'false')
      interactive.dataset.enhancement = 'idle'
    }
    plain.open = true
    interactive.open = false
    current(plain.id)
    if (returnFocus) {
      const target = plain.querySelector(':scope > summary')
      target.focus({ preventScroll: true })
      target.scrollIntoView({ block: 'center', behavior: 'instant' })
    }
  }
  const openTarget = target => {
    showPlain()
    for (let node = target; node && node !== plain.parentElement; node = node.parentElement)
      if (node.tagName === 'DETAILS') node.open = true
    // Operation/schema anchors belong to their semantic wrapper. Open its
    // native disclosure too, so a direct link exposes the contract immediately.
    const details = target.querySelector('details')
    if (details) details.open = true
  }
  const unavailable = () => {
    attempt += 1
    controller?.abort()
    window.clearTimeout(timer)
    loading = false
    ready = false
    instance?.destroy()
    instance = null
    cachedSource = null
    container.setAttribute('aria-busy', 'false')
    showPlain()
    interactive.dataset.enhancement = 'failed'
    status.textContent = 'The interactive explorer is unavailable. The complete Plain HTML reference remains available; select the explorer again to retry.'
  }
  const ownedLayoutReady = () => {
    const intent = modelIntent || (entryIntent?.host ? entryIntent : null), target = document.activeElement
    if (!intent || !interactive.open) return true
    if (intent.attempt !== attempt || (intent.target ? target !== intent.target : !target.id.endsWith('/' + intent.id))) return false
    if (intent === modelIntent) intent.ownedTarget = target
    // A navigation target can mount before lazy placeholders above it shrink.
    // Close Plain first, then admit only a settled, visible requested model.
    plain.open = false
    target.scrollIntoView({ block: 'center', behavior: 'instant' })
    const rect = target.getBoundingClientRect(), height = container.getBoundingClientRect().height
    const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2)
    const visible = [height, rect.top, rect.bottom, rect.left, rect.right, rect.width, rect.height].every(Number.isFinite) &&
      rect.width > 0 && rect.height > 0 && rect.top >= 0 && rect.bottom <= innerHeight && rect.left >= 0 && rect.right <= innerWidth &&
      Boolean(hit && (hit === target || target.contains(hit))) && (!intent.hash || location.hash === intent.hash)
    const layout = JSON.stringify([height, rect.top + scrollY, rect.left, rect.width, rect.height, location.hash])
    // Scalar 1.72.4 schedules each real placeholder inside a 1200px overscan.
    // A quiet frame is insufficient while that deferred work remains visible.
    const pending = [...container.querySelectorAll('[data-testid="lazy-container"][data-placeholder="true"]')].some(node => {
      const r = node.getBoundingClientRect()
      return r.height > 0 && r.bottom >= -1200 && r.top <= innerHeight + 1200
    })
    intent.stable = visible && !pending && intent.layout === layout ? (intent.stable || 0) + 1 : 0
    intent.layout = layout
    return intent.stable >= 3
  }
  const complete = () => {
    const region = container.querySelector('section.references-rendered[aria-label]')
    // Operation bodies are lazy and depend on scroll position. Admit the view
    // after every contract operation has its real rendered navigation target.
    const navigation = [...container.querySelectorAll('a[href]')].map(link => {
      const match = link.getAttribute('href').match(/^#tag\/[^/]+\/(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|TRACE)(\/.*)$/)
      return match ? [match[1], decodeURIComponent(match[2])] : null
    }).filter(Boolean)
    const prebuilt = [...reference.querySelectorAll('[data-api-operation]')].map(node => [node.dataset.apiMethod, node.dataset.apiPath])
    const targets = entries => JSON.stringify(entries.map(entry => JSON.stringify(entry)).sort())
    const operationsReady = navigation.length === prebuilt.length && targets(navigation) === targets(prebuilt)
    const soleMain = document.querySelectorAll('main,[role="main"]').length === 1 && !container.querySelector('main,[role="main"]')
    const ids = [...container.querySelectorAll('[id]')].map(node => node.id)
    if (!region || !region.getAttribute('aria-label').trim() || !operationsReady || !soleMain || new Set(ids).size !== ids.length) return false
    if (!ownedLayoutReady()) return false
    window.clearTimeout(timer)
    loading = false
    ready = true
    container.setAttribute('aria-busy', 'false')
    interactive.dataset.enhancement = 'ready'
    status.textContent = 'Interactive explorer ready. Choose either reference view; both use the same generated API contract.'
    if (interactive.open) {
      plain.open = false
      current(interactive.id)
      settleEntry()
      settleModel()
    }
    return true
  }
  const fetchSource = async () => {
    const response = await fetch(marker.dataset.url, { credentials: 'same-origin', signal: controller.signal })
    if (!response.ok) throw Error('API contract unavailable')
    const bytes = await response.arrayBuffer()
    const digest = [...new Uint8Array(await crypto.subtle.digest('SHA-256', bytes))].map(byte => byte.toString(16).padStart(2, '0')).join('')
    if (digest !== reference.dataset.openapiSha256) throw Error('API contract differs from prebuilt HTML')
    cachedSource = new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  }
  const validatedDocument = () => {
    const document = JSON.parse(cachedSource)
    const methods = new Set(['get', 'post', 'put', 'patch', 'delete', 'head', 'options', 'trace'])
    const operations = Object.entries(document.paths).flatMap(([path, item]) => Object.entries(item)
      .filter(([method]) => methods.has(method)).map(([method, operation]) => [operation.operationId, method.toUpperCase(), path]))
    const prebuilt = [...reference.querySelectorAll('[data-api-operation]')].map(node => [node.dataset.apiOperation, node.dataset.apiMethod, node.dataset.apiPath])
    const identities = entries => JSON.stringify(entries.sort(([a], [b]) => a.localeCompare(b)))
    if (operations.length !== Number(reference.dataset.operationCount) || identities(operations) !== identities(prebuilt)) {
      cachedSource = null
      throw Error('API operation identities differ')
    }
    return document
  }
  const mountReference = (document, thisAttempt) => {
    // Remove host-only fragments before Scalar infers its canonical bare hash
    // routing. Preserve actual Scalar deep links; basePath:"#" expects "#/"
    // on reads and silently discards a valid initial "#tag/..." route.
    if (location.hash && !scalarHash())
      history.replaceState(null, '', location.pathname + location.search)
    configuration = { content: document, withDefaultFonts: false, agent: { disabled: true }, telemetry: false,
      showDeveloperTools: 'never', proxyUrl: '', theme: 'none', hideDarkModeToggle: true,
      forceDarkModeState: darkState(), defaultOpenAllTags: true,
      onLoaded: () => {
        const awaitRender = () => {
          if (!loading || thisAttempt !== attempt || complete()) return
          requestAnimationFrame(awaitRender)
        }
        requestAnimationFrame(awaitRender)
      } }
    instance = window.Scalar.createApiReference(container, configuration)
  }
  const enhance = async () => {
    if (ready) {
      plain.open = false
      current(interactive.id)
      settleEntry()
      settleModel()
      return
    }
    if (loading) return
    const thisAttempt = ++attempt
    controller = new AbortController()
    loading = true
    plain.open = true
    interactive.dataset.enhancement = 'loading'
    container.setAttribute('aria-busy', 'true')
    status.textContent = 'Loading the optional interactive explorer. The complete Plain HTML reference remains available.'
    timer = window.setTimeout(() => { if (thisAttempt === attempt) unavailable() }, 30000)
    try {
      if (!cachedSource) await fetchSource()
      if (thisAttempt !== attempt) return
      mountReference(validatedDocument(), thisAttempt)
    } catch {
      if (thisAttempt === attempt) unavailable()
    }
  }
  plain.addEventListener('toggle', () => {
    if (plain.open && ready) {
      showPlain()
    }
  })
  interactive.addEventListener('toggle', () => {
    if (interactive.open) enhance()
    else if (ready || loading) showPlain()
  })
  summary.addEventListener('click', () => {
    if (!interactive.open && document.activeElement === summary) rememberEntry()
  })
  const route = () => {
    cancelEntry()
    let target
    try { target = document.getElementById(decodeURIComponent(location.hash.slice(1))) }
    catch { return }
    if (target && (target === plain || plain.contains(target))) openTarget(target)
    else if (target && (target === interactive || interactive.contains(target)) || scalarHash()) {
      if (/^#models\/[^/]+$/.test(location.hash)) modelIntent = {
        hash: location.hash, id: decodeURIComponent(location.hash.slice(1)),
        attempt: attempt + (ready || loading ? 0 : 1)
      }
      interactive.open = true
      enhance()
    }
  }
  window.addEventListener('hashchange', route)
  for (const link of links) link.addEventListener('click', event => {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    // The native fragment could scroll past a still-open or closing view.
    // No-JS links remain native; only these two explicit view entries are owned.
    event.preventDefault()
    history.pushState(null, '', link.hash)
    if (link.hash === '#' + plain.id) {
      showPlain()
      const target = plain.querySelector(':scope > summary')
      target.focus({ preventScroll: true })
      target.scrollIntoView({ block: 'center', behavior: 'instant' })
    } else {
      rememberEntry()
      interactive.open = true
      enhance()
    }
  })
  new MutationObserver(() => {
    if (instance && configuration) instance.updateConfiguration({ ...configuration, forceDarkModeState: darkState() })
  }).observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
  route()
}
