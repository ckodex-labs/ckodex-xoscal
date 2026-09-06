(() => {
  "use strict";

  const form = document.getElementById("api-form");
  const importForm = document.getElementById("import-form");
  const apiInput = document.getElementById("api-base");
  const tokenInput = document.getElementById("api-token");
  const apiError = document.getElementById("api-error");
  const status = document.getElementById("review-status");
  const list = document.getElementById("claim-list");
  const detail = document.getElementById("claim-detail");
  const receipts = document.getElementById("receipts");
  const importRecords = document.getElementById("import-records");
  const addImportRecordButton = document.getElementById("add-import-record");
  const maxImportRecords = 10;
  const params = new URLSearchParams(window.location.search);
  const configuredBase = params.get("api") || document.querySelector('meta[name="xoscal-api-base"]')?.content || "";
  let apiBase = configuredBase;
  let apiToken = "";
  let selectedClaim = null;
  let selectedTrustState = "";
  let verificationDiagnostics = [];
  let verificationEvents = [];
  let verificationAuditError = "";
  let projectionEvents = [];
  let projectionAuditError = "";

  if (apiBase) apiInput.value = apiBase;

  const escapeHTML = value => String(value ?? "").replace(/[&<>"']/g, c => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
  }[c]));

  const field = (claim, camel, snake) => claim?.[camel] ?? claim?.[snake];
  const claimID = claim => field(claim, "id", "id") || "";
  const trustState = claim => field(claim, "trustState", "trust_state") || "candidate";
  const proofState = value => value || {};

  const stateChip = state => {
    const normalized = state === "verified" ? "attested" : state === "rejected" ? "contradicted" : "claimed";
    const glyph = normalized === "attested" ? "◆" : normalized === "contradicted" ? "⊭" : "○";
    return `<span class="ck-chip ck-chip--${normalized}"><span aria-hidden="true">${glyph}</span>${escapeHTML(state)}</span>`;
  };

  const digestButton = digest => {
    if (!digest) return '<span class="ck-chip ck-chip--claimed">○ digest not supplied</span>';
    if (!/^sha256:[0-9a-f]{16,}$/i.test(digest)) {
      return `<span class="ck-hash ck-hash--unprefixed">${escapeHTML(digest)}</span>`;
    }
    return `<button type="button" class="ck-hash" data-copy-digest="${escapeHTML(digest)}" title="Copy digest">${escapeHTML(digest)}</button>`;
  };

  const setStatus = message => { status.textContent = message; };

  const requestErrorMessage = (error, action = "complete the request") => {
    if (error?.status === 401) return `Authentication failed while trying to ${action} (HTTP 401). Check the bearer token and try again.`;
    if (error?.status === 403) return `Access was denied while trying to ${action} (HTTP 403). This token cannot access the configured beta workspace.`;
    if (error?.status === 404) return `The configured API does not expose the beta ${action} route (HTTP 404). Check the gateway base URL.`;
    if (error?.status >= 500) return `The API could not ${action} (HTTP ${error.status}). Try again or contact the deployment operator.`;
    if (error?.name === "AbortError") return `The API did not respond while trying to ${action}. Check the endpoint and try again.`;
    if (error?.message === "Failed to fetch" || error?.message === "Load failed") return `The configured API could not be reached while trying to ${action}. Check the URL, TLS certificate, and CORS configuration.`;
    return error?.message || `The API could not ${action}.`;
  };

  const setConnectionError = message => {
    apiInput.setAttribute("aria-invalid", "true");
    apiError.textContent = message;
    apiError.hidden = false;
  };

  const clearConnectionError = () => {
    apiInput.removeAttribute("aria-invalid");
    apiError.textContent = "";
    apiError.hidden = true;
  };

  const addReceipt = (event, description, kind = "") => {
    const item = document.createElement("div");
    item.className = `ck-receipt${kind ? ` ck-receipt--${kind}` : ""}`;
    item.innerHTML = `<span class="ck-receipt__rule"></span><div><div class="ck-receipt__event">${escapeHTML(event)}</div><div class="ck-receipt__detail">${escapeHTML(description)}</div></div>`;
    receipts.prepend(item);
  };

  const request = async (path, options = {}) => {
    if (!apiBase) throw new Error("API endpoint not configured");
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), 10000);
    try {
      const response = await fetch(`${apiBase.replace(/\/$/, "")}${path}`, {
        ...options,
        headers: { Accept: "application/json", ...(options.body ? { "Content-Type": "application/json" } : {}), ...(apiToken ? { Authorization: `Bearer ${apiToken}` } : {}), ...(options.headers || {}) },
        signal: controller.signal
      });
      const body = await response.json().catch(() => ({}));
      if (!response.ok) {
        const error = new Error(body.message || body.error || `API returned HTTP ${response.status}`);
        error.status = response.status;
        error.path = path;
        throw error;
      }
      return body;
    } finally {
      window.clearTimeout(timeout);
    }
  };

  const inputValue = (record, name) => record.querySelector(`[data-import-field="${name}"]`).value.trim();

  const importFieldLabels = {
    "claim-id": "Claim ID",
    "subject-kind": "Subject kind",
    "subject-value": "Subject ID or digest",
    "predicate-relation": "Relation",
    "object-kind": "Object kind",
    "object-value": "Object ID or digest",
    "evidence-id": "Evidence ID",
    "evidence-media-type": "Media type",
    "evidence-bom-kind": "BOM kind",
    "evidence-content": "Evidence content"
  };

  const importField = (record, name) => record.querySelector(`[data-import-field="${name}"]`);

  const clearFieldError = input => {
    if (!input) return;
    input.removeAttribute("aria-invalid");
    const error = document.getElementById(`${input.id}-error`);
    if (error) error.remove();
    const describedBy = (input.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean).filter(id => id !== `${input.id}-error`);
    if (describedBy.length) input.setAttribute("aria-describedby", describedBy.join(" "));
    else input.removeAttribute("aria-describedby");
  };

  const clearImportErrors = () => importRecords.querySelectorAll("[data-import-field]").forEach(clearFieldError);

  const markFieldError = (input, message) => {
    if (!input) return;
    input.setAttribute("aria-invalid", "true");
    const errorID = `${input.id}-error`;
    let error = document.getElementById(errorID);
    if (!error) {
      error = document.createElement("p");
      error.id = errorID;
      error.className = "ck-fieldnote ck-fieldnote--invalid ck-import-error";
      input.insertAdjacentElement("afterend", error);
    }
    error.textContent = message;
    const describedBy = (input.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean);
    if (!describedBy.includes(errorID)) describedBy.push(errorID);
    input.setAttribute("aria-describedby", describedBy.join(" "));
  };

  const updateAddRecordButton = () => {
    const count = importRecords.querySelectorAll("[data-import-record]").length;
    addImportRecordButton.disabled = count >= maxImportRecords;
    addImportRecordButton.setAttribute("aria-label", count >= maxImportRecords ? `Maximum of ${maxImportRecords} import records reached` : "Add another import record");
  };

  const addImportRecord = () => {
    const records = Array.from(importRecords.querySelectorAll("[data-import-record]"));
    if (records.length >= maxImportRecords) return;
    const index = records.length;
    const clone = records[0].cloneNode(true);
    clone.dataset.importRecord = String(index);
    clone.querySelectorAll("[id]").forEach(element => {
      element.id = element.id.replace(/-0$/, `-${index}`);
    });
    clone.querySelectorAll("label[for]").forEach(label => {
      label.htmlFor = label.htmlFor.replace(/-0$/, `-${index}`);
    });
    clone.querySelectorAll("[aria-describedby]").forEach(element => {
      element.setAttribute("aria-describedby", element.getAttribute("aria-describedby").replace(/-0$/, `-${index}`));
    });
    clone.querySelectorAll(".ck-import-error").forEach(error => error.remove());
    clone.querySelectorAll("[aria-invalid]").forEach(element => element.removeAttribute("aria-invalid"));
    clone.querySelector("h3").textContent = `Record ${index + 1}`;
    clone.querySelectorAll("input, textarea").forEach(input => {
      input.value = input.defaultValue;
    });
    importRecords.appendChild(clone);
    updateAddRecordButton();
    clone.querySelector('[data-import-field="claim-id"]').focus();
  };

  const resetImportRecords = () => {
    clearImportErrors();
    importRecords.querySelectorAll("[data-import-record]").forEach((record, index) => {
      if (index > 0) record.remove();
    });
    importForm.reset();
    updateAddRecordButton();
  };

  const digestBytes = async bytes => {
    if (!window.crypto?.subtle) throw new Error("This browser cannot calculate evidence digests.");
    const hash = await window.crypto.subtle.digest("SHA-256", bytes);
    return `sha256:${Array.from(new Uint8Array(hash), byte => byte.toString(16).padStart(2, "0")).join("")}`;
  };

  const base64Bytes = bytes => {
    let binary = "";
    for (let index = 0; index < bytes.length; index += 1) binary += String.fromCharCode(bytes[index]);
    return window.btoa(binary);
  };

  const referenceFromValue = (kind, value) => {
    const reference = { kind };
    if (/^sha256:[0-9a-f]{16,}$/i.test(value)) reference.digest = value;
    else reference.id = value;
    return reference;
  };

  const renderClaim = claim => {
    const id = claimID(claim);
    const subject = field(claim, "subject", "subject") || {};
    const predicate = field(claim, "predicate", "predicate") || {};
    const refs = field(claim, "sourceRefs", "source_refs") || [];
    return `<article class="ck-quiet ck-padded">
      <div class="ck-row-3"><h3>${escapeHTML(id)}</h3>${stateChip(trustState(claim))}</div>
      <dl class="ck-evidence-list">
        <div><dt>relation</dt><dd>${escapeHTML(predicate.relation || "not supplied")}</dd></div>
        <div><dt>subject</dt><dd>${escapeHTML(subject.id || subject.digest || "not supplied")}</dd></div>
        <div><dt>source evidence</dt><dd>${refs.length}</dd></div>
      </dl>
      <button type="button" class="ck-btn ck-btn--quiet" data-select-claim="${escapeHTML(id)}">Inspect claim</button>
    </article>`;
  };

  const renderList = claims => {
    if (!claims.length) {
      list.innerHTML = '<p class="ck-loading">No claims require review.</p>';
      return;
    }
    list.innerHTML = claims.map(renderClaim).join("");
  };

  const renderAudit = () => {
    if (verificationAuditError) {
      return `<p class="ck-site-note">${escapeHTML(verificationAuditError)}</p>`;
    }
    if (!verificationEvents.length) {
      return '<p class="ck-site-note">No persisted verification events for this claim yet.</p>';
    }
    const rows = verificationEvents.map(event => {
      let checks = field(event, "checksJson", "checks_json") || "[]";
      try { checks = JSON.parse(checks).join(", "); } catch (_) { /* preserve the recorded JSON below */ }
      const diagnostics = field(event, "diagnostics", "diagnostics") || [];
      const created = field(event, "createdAt", "created_at") || "time not recorded";
      const hash = field(event, "eventHash", "event_hash") || "hash not recorded";
      return `<tr>
        <td>${escapeHTML(field(event, "sequence", "sequence"))}</td>
        <td>${stateChip(field(event, "trustState", "trust_state") || "incomplete")}</td>
        <td><code>${escapeHTML(checks)}</code><br><span class="ck-caption">${escapeHTML(created)}</span></td>
        <td>${digestButton(hash)}${diagnostics.length ? `<details><summary>Diagnostics (${diagnostics.length})</summary><p class="ck-caption">${diagnostics.map(escapeHTML).join(" ")}</p></details>` : ""}</td>
      </tr>`;
    }).join("");
    return `<div class="ck-table-wrap"><table class="ck-table">
      <caption>Append-only verification history; audit chain valid</caption>
      <thead><tr><th scope="col">sequence</th><th scope="col">trust</th><th scope="col">checks</th><th scope="col">event hash</th></tr></thead>
      <tbody>${rows}</tbody>
    </table></div>`;
  };

  const renderProjectionAudit = () => {
    if (projectionAuditError) {
      return `<p class="ck-site-note">${escapeHTML(projectionAuditError)}</p>`;
    }
    if (!projectionEvents.length) {
      return '<p class="ck-site-note">No graph projection audit event is recorded for this claim.</p>';
    }
    const rows = projectionEvents.map(event => {
      const edge = field(event, "edgeId", "edge_id") || "edge not recorded";
      const from = field(event, "fromNode", "from_node") || "source not recorded";
      const to = field(event, "toNode", "to_node") || "target not recorded";
      const relation = field(event, "relation", "relation") || "relation not recorded";
      const hash = field(event, "eventHash", "event_hash") || "hash not recorded";
      const projected = field(event, "projectedAt", "projected_at") || "time not recorded";
      return `<tr>
        <td>${escapeHTML(field(event, "sequence", "sequence"))}</td>
        <td><code>${escapeHTML(edge)}</code><br><span class="ck-caption">${escapeHTML(projected)}</span></td>
        <td>${escapeHTML(from)} → ${escapeHTML(to)}<br><span class="ck-caption">${escapeHTML(relation)}</span></td>
        <td>${digestButton(hash)}</td>
      </tr>`;
    }).join("");
    return `<div class="ck-table-wrap"><table class="ck-table">
      <caption>Append-only graph projection history; audit chain valid</caption>
      <thead><tr><th scope="col">sequence</th><th scope="col">edge</th><th scope="col">projection</th><th scope="col">event hash</th></tr></thead>
      <tbody>${rows}</tbody>
    </table></div>`;
  };

  const renderDetail = (claim, explanation) => {
    const refs = field(claim, "sourceRefs", "source_refs") || [];
    const currentTrustState = field(explanation, "trustState", "trust_state") || selectedTrustState || trustState(claim);
    const ps = proofState(field(explanation, "proofStateJson", "proof_state_json") ? (() => {
      try { return JSON.parse(field(explanation, "proofStateJson", "proof_state_json")); } catch (_) { return {}; }
    })() : field(explanation, "proofState", "proof_state"));
    const evidence = explanation?.evidence || [];
    const diagnostics = [...verificationDiagnostics, ...(field(explanation, "diagnostics", "diagnostics") || [])];
    detail.innerHTML = `<div class="ck-stack ck-stack--tight">
      <div class="ck-row-3"><h3>${escapeHTML(claimID(claim))}</h3>${stateChip(currentTrustState)}</div>
      <dl class="ck-evidence-list">
        <div><dt>proof state</dt><dd><code>${escapeHTML(JSON.stringify(ps))}</code></dd></div>
        <div><dt>source references</dt><dd>${refs.length ? refs.map(ref => digestButton(ref.digest)).join(" ") : "none"}</dd></div>
        <div><dt>stored evidence</dt><dd>${evidence.length}</dd></div>
      </dl>
      ${diagnostics.length ? `<p class="ck-site-note">${diagnostics.map(escapeHTML).join(" ")}</p>` : ""}
      <section aria-labelledby="audit-heading-${escapeHTML(claimID(claim))}">
        <h4 id="audit-heading-${escapeHTML(claimID(claim))}">Verification history</h4>
        ${renderAudit()}
      </section>
      <section aria-labelledby="projection-audit-heading-${escapeHTML(claimID(claim))}">
        <h4 id="projection-audit-heading-${escapeHTML(claimID(claim))}">Graph projection history</h4>
        ${renderProjectionAudit()}
      </section>
      <fieldset class="ck-stack--tight">
        <legend>Verification profile</legend>
        <p class="ck-caption">Each selected check is recorded in the append-only event. Select all three for the complete beta profile.</p>
        <label><input type="checkbox" data-verification-check="${escapeHTML(claimID(claim))}" value="digest" checked> content digest</label>
        <label><input type="checkbox" data-verification-check="${escapeHTML(claimID(claim))}" value="signature" checked> Ed25519 signature</label>
        <label><input type="checkbox" data-verification-check="${escapeHTML(claimID(claim))}" value="policy" checked> xoscal-json policy</label>
      </fieldset>
      <p id="export-help" class="ck-caption">${currentTrustState === "verified" ? "The receipt includes the claim, proof state, evidence metadata, and the chained verification event." : "Receipt export is blocked until the complete beta proof envelope is verified."}</p>
      <div class="ck-row-3">
        <button type="button" class="ck-btn ck-btn--primary" data-verify-claim="${escapeHTML(claimID(claim))}">Run verification</button>
        <button type="button" class="ck-btn ck-btn--quiet" data-export-claim="${escapeHTML(claimID(claim))}"${currentTrustState === "verified" ? "" : " disabled aria-describedby=\"export-help\""}>Export verification receipt</button>
      </div>
    </div>`;
  };

  const loadClaims = async () => {
    clearConnectionError();
    setStatus("Loading claims from the configured API.");
    try {
      const response = await request("/v1/transparency/claims?page_size=50");
      const claims = response.claims || [];
      renderList(claims);
      setStatus(`${claims.length} claim${claims.length === 1 ? "" : "s"} loaded from the live API.`);
      addReceipt("evaluated", `claim queue loaded: ${claims.length}`);
    } catch (error) {
      const message = requestErrorMessage(error, "load the claim queue");
      list.innerHTML = `<p class="ck-site-note">${escapeHTML(message)}</p>`;
      setConnectionError(message);
      setStatus("Live claim queue unavailable. No sample data was substituted.");
      addReceipt("rejected", message);
    }
  };

  const inspectClaim = async id => {
    if (selectedClaim && claimID(selectedClaim) !== id) verificationDiagnostics = [];
    verificationEvents = [];
    verificationAuditError = "";
    projectionEvents = [];
    projectionAuditError = "";
    detail.setAttribute("aria-busy", "true");
    setStatus(`Loading claim ${id}.`);
    try {
      const response = await request("/v1/graph/explain", { method: "POST", body: JSON.stringify({ claimId: id }) });
      selectedClaim = response.claim || response;
      selectedTrustState = field(response, "trustState", "trust_state") || trustState(selectedClaim);
      try {
        const audit = await request(`/v1/transparency/claims/${encodeURIComponent(id)}/verification-events`);
        verificationEvents = audit.events || [];
      } catch (auditError) {
        verificationAuditError = `Verification history unavailable: ${auditError.message || "the audit endpoint rejected the request"}`;
      }
      try {
        const projectionAudit = await request(`/v1/graph/projection-events?claim_id=${encodeURIComponent(id)}`);
        projectionEvents = projectionAudit.events || [];
      } catch (auditError) {
        projectionAuditError = `Graph projection history unavailable: ${auditError.message || "the audit endpoint rejected the request"}`;
      }
      renderDetail(selectedClaim, response);
      setStatus(`Claim ${id} loaded.`);
      addReceipt("transformed", `claim explanation loaded: ${id}`);
    } catch (error) {
      const message = requestErrorMessage(error, "load the claim explanation");
      detail.innerHTML = `<p class="ck-site-note">${escapeHTML(message)}</p>`;
      setStatus(`Claim ${id} could not be loaded.`);
      addReceipt("rejected", message);
    } finally {
      detail.removeAttribute("aria-busy");
    }
  };

  const verifyClaim = async id => {
    const checks = Array.from(document.querySelectorAll("[data-verification-check]"))
      .filter(input => input.dataset.verificationCheck === id && input.checked)
      .map(input => input.value);
    if (!checks.length) {
      setStatus("Select at least one verification check before running verification.");
      return;
    }
    setStatus(`Verifying claim ${id}.`);
    try {
      const response = await request(`/v1/transparency/claims/${encodeURIComponent(id)}/verify`, {
        method: "POST",
        body: JSON.stringify({ checks })
      });
      const state = response.trustState || response.trust_state || "incomplete";
      verificationDiagnostics = field(response, "diagnostics", "diagnostics") || [];
      setStatus(`Verification completed with state: ${state}.`);
      addReceipt("evaluated", `verification completed: ${state}`);
      await inspectClaim(id);
      await loadClaims();
    } catch (error) {
      const message = requestErrorMessage(error, "verify the claim");
      setStatus("Verification failed. The claim state was not treated as verified.");
      addReceipt("rejected", message);
    }
  };

  const exportReceipt = async id => {
    setStatus(`Preparing a verification receipt for ${id}.`);
    try {
      const response = await request(`/v1/transparency/claims/${encodeURIComponent(id)}/receipt`);
      const receipt = response.receiptJson || response.receipt_json;
      if (!receipt) throw new Error("The API returned no receipt payload.");
      const blob = new Blob([receipt], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${id}.xoscal-receipt.json`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      const digest = response.receiptDigest || response.receipt_digest || "digest not returned";
      addReceipt("exported", `verification receipt: ${digest}`, "proof");
      setStatus(`Verification receipt exported for ${id}.`);
    } catch (error) {
      const message = requestErrorMessage(error, "export the verification receipt");
      setStatus(`Receipt export blocked: ${message}`);
      addReceipt("rejected", message);
    }
  };

  const importClaim = async event => {
    event.preventDefault();
    const submit = importForm.querySelector("button[type=submit]");
    submit.disabled = true;
    importForm.setAttribute("aria-busy", "true");
    try {
      if (!apiBase) throw new Error("Connect an API before importing evidence.");
      const formRecords = Array.from(importRecords.querySelectorAll("[data-import-record]"));
      if (!formRecords.length) throw new Error("At least one import record is required.");
      clearImportErrors();
      const requiredFields = Object.keys(importFieldLabels);
      const missingFields = [];
      formRecords.forEach((record, index) => {
        requiredFields.forEach(name => {
          const input = importField(record, name);
          if (!inputValue(record, name)) {
            markFieldError(input, `${importFieldLabels[name]} is required for record ${index + 1}.`);
            missingFields.push(input);
          }
        });
      });
      if (missingFields.length) {
        const firstMissing = missingFields[0];
        setStatus(`Import blocked: ${missingFields.length} required field${missingFields.length === 1 ? "" : "s"} need attention.`);
        addReceipt("rejected", `import fields missing: ${missingFields.length}`);
        firstMissing.focus();
        return;
      }
      setStatus(`Hashing ${formRecords.length} import record${formRecords.length === 1 ? "" : "s"} locally before preflight.`);
      const records = [];
      for (const [index, recordElement] of formRecords.entries()) {
        const claimId = inputValue(recordElement, "claim-id");
        const subjectKind = inputValue(recordElement, "subject-kind");
        const subjectValue = inputValue(recordElement, "subject-value");
        const predicateRelation = inputValue(recordElement, "predicate-relation");
        const objectKind = inputValue(recordElement, "object-kind");
        const objectValue = inputValue(recordElement, "object-value");
        const evidenceId = inputValue(recordElement, "evidence-id");
        const mediaType = inputValue(recordElement, "evidence-media-type");
        const bomKind = inputValue(recordElement, "evidence-bom-kind");
        const content = recordElement.querySelector('[data-import-field="evidence-content"]').value;
        if (!claimId || !subjectKind || !subjectValue || !predicateRelation || !objectKind || !objectValue || !evidenceId || !mediaType || !bomKind || !content) {
          throw new Error(`Record ${index + 1}: claim identity, source metadata, and evidence content are required.`);
        }
        const bytes = new TextEncoder().encode(content);
        const digest = await digestBytes(bytes);
        const now = new Date().toISOString();
        records.push({
          claim: {
            id: claimId,
            type: "beta.review",
            subject: referenceFromValue(subjectKind, subjectValue),
            predicate: { relation: predicateRelation, direction: "forward" },
            object: referenceFromValue(objectKind, objectValue),
            issuer: { kind: "human", id: "beta-operator" },
            bomKind,
            validTime: { fromTime: now },
            observedTime: now,
            sourceRefs: [{ ref: evidenceId, digest, mediaType, bomKind }]
          },
          evidence: [{
            evidence: { id: evidenceId, mediaType, bomKind, digest, sizeBytes: bytes.length },
            blob: base64Bytes(bytes)
          }]
        });
      }
      const preflight = await request("/v1/transparency/import/preflight", {
        method: "POST",
        body: JSON.stringify({ records })
      });
      if (!preflight.valid) {
        const diagnostics = (preflight.results || []).flatMap(result => result.diagnostics || []).join(" ");
        throw new Error(`Import preflight blocked: ${diagnostics || "one or more records are not ready"}`);
      }
      addReceipt("evaluated", `import preflight passed: ${records.length} record${records.length === 1 ? "" : "s"}`);
      const imported = await request("/v1/transparency/import", {
        method: "POST",
        body: JSON.stringify({ records, allOrNothing: true })
      });
      if (!imported.committed) {
        const diagnostics = (imported.results || []).flatMap(result => result.diagnostics || []).join(" ");
        throw new Error(`Import was rolled back: ${diagnostics || "the API did not commit the record"}`);
      }
      const claimIds = records.map(record => record.claim.id);
      addReceipt("transformed", `batch imported: ${claimIds.join(", ")}`);
      setStatus(`${claimIds.length} claim${claimIds.length === 1 ? "" : "s"} imported atomically.`);
      resetImportRecords();
      await loadClaims();
      await inspectClaim(claimIds[0]);
    } catch (error) {
      const message = requestErrorMessage(error, "import the batch");
      setStatus(`Import failed: ${message}`);
      addReceipt("rejected", message);
    } finally {
      submit.disabled = false;
      importForm.removeAttribute("aria-busy");
    }
  };

  form.addEventListener("submit", event => {
    event.preventDefault();
    apiBase = apiInput.value.trim().replace(/\/$/, "");
    apiToken = tokenInput.value.trim();
    clearConnectionError();
    loadClaims();
  });
  importForm.addEventListener("submit", importClaim);
  importForm.addEventListener("input", event => {
    if (event.target.matches("[data-import-field]")) clearFieldError(event.target);
  });
  importForm.addEventListener("change", event => {
    if (event.target.matches("[data-import-field]")) clearFieldError(event.target);
  });
  apiInput.addEventListener("input", clearConnectionError);
  addImportRecordButton.addEventListener("click", addImportRecord);
  updateAddRecordButton();

  document.addEventListener("click", event => {
    const select = event.target.closest("[data-select-claim]");
    if (select) inspectClaim(select.dataset.selectClaim);
    const verify = event.target.closest("[data-verify-claim]");
    if (verify) verifyClaim(verify.dataset.verifyClaim);
    const exportButton = event.target.closest("[data-export-claim]");
    if (exportButton && !exportButton.disabled) exportReceipt(exportButton.dataset.exportClaim);
    const digest = event.target.closest("[data-copy-digest]");
    if (digest && navigator.clipboard) {
      navigator.clipboard.writeText(digest.dataset.copyDigest).then(() => {
        digest.textContent = "copied";
        setStatus("Digest copied.");
      });
    }
  });

  if (apiBase) loadClaims();
})();
