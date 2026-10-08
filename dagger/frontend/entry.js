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
  let instance = null, configuration = null, loading = false, ready = false
  let timer = null, controller = null, attempt = 0, cachedSource = null
  const darkState = () => ['vault', 'hc'].includes(document.documentElement.dataset.theme) ? 'dark' : 'light'
  const scalarHash = () => /^#(?:tag|model|models|operation|description)(?:\/|$)/.test(location.hash)
  const current = id => {
    for (const link of links) {
      if (link.hash === '#' + id) link.setAttribute('aria-current', 'true')
      else link.removeAttribute('aria-current')
    }
  }
  const showPlain = () => {
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
    window.clearTimeout(timer)
    loading = false
    ready = true
    container.setAttribute('aria-busy', 'false')
    interactive.dataset.enhancement = 'ready'
    status.textContent = 'Interactive explorer ready. Choose either reference view; both use the same generated API contract.'
    if (interactive.open) {
      plain.open = false
      current(interactive.id)
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
  const route = () => {
    let target
    try { target = document.getElementById(decodeURIComponent(location.hash.slice(1))) }
    catch { return }
    if (target && (target === plain || plain.contains(target))) openTarget(target)
    else if (target && (target === interactive || interactive.contains(target)) || scalarHash()) {
      interactive.open = true
      enhance()
    }
  }
  window.addEventListener('hashchange', route)
  for (const link of links) link.addEventListener('click', () => {
    if (link.hash === '#' + plain.id) showPlain()
    else {
      interactive.open = true
      enhance()
    }
  })
  new MutationObserver(() => {
    if (instance && configuration) instance.updateConfiguration({ ...configuration, forceDarkModeState: darkState() })
  }).observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
  route()
}
