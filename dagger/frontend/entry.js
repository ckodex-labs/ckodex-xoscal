import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'
window.Scalar = { createApiReference }
// Preserve the portal's existing data-url mounting contract.
const marker = document.getElementById('api-reference')
if (marker) {
  const container = document.createElement('div')
  marker.parentNode.insertBefore(container, marker)
  window.Scalar.createApiReference(container, { url: marker.dataset.url, agent: { disabled: true }, telemetry: false, showDeveloperTools: "never", proxyUrl: "" })
}
