# API keyboard focus correction

This bounded follow-up implements the remaining presentation defect found in
the live acceptance of immutable release `v0.3.3` (source
`76731bac90fcbfbd0caa84dd18588133ec680949`). The comprehensive production and
release plan remains in [RELEASE-IMPLEMENTATION-PLAN.md](RELEASE-IMPLEMENTATION-PLAN.md)
and [STATIC-FIRST-CORRECTION-PLAN.md](STATIC-FIRST-CORRECTION-PLAN.md).

## Confirmed problem

On desktop and mobile, selecting the interactive API reference with the keyboard
can leave focus and the explorer heading outside the viewport after the long
Plain HTML disclosure closes. The same problem occurs when tabbing from the
last expanded model to the explorer's native summary. Re-scrolling in the test
can conceal this failure, so measurements must observe the actual transition
before any test scroll or focus repair. Existing signed release evidence binds
the published bytes; it does not establish that every presentation behavior is
correct.

Tabbing forward during loading to the raw OpenAPI link exposes a second layout
transition: focus stays on that host link, but Scalar's growing content pushes
it outside the viewport. Keep that explicit, still-owned host control visible
without refocusing it or overriding Scalar's internal navigation.

## Implementation phases and dependencies

1. **Preserve and reproduce.** Keep `v0.3.3`, its proof inputs and failed browser
   runs unchanged. Use a separate `fix/api-keyboard-focus` checkout based on the
   release source. Reproduce actual Tab/Enter transitions on both view selectors
   and the far native summary. Separately repair any verifier timing defect;
   require independent negative controls before accepting the verifier.
2. **Own only explicit entry intent.** In `dagger/frontend/entry.js`, give the
   explorer's native summary focus when its view selector is activated. Retain
   that still-owned entry through the ready layout transition. Cancel intent
   when the user moves focus or interacts, and guard deferred work by attempt
   identity and current focus. Track a user's new host-control focus during
   loading only outside Scalar's rendered container, and compensate its ready
   layout without moving focus. Returning to Plain or failed enhancement must
   preserve visible meaningful focus and cancel optional loading. Preserve
   native modified links and existing operation/model deep links.
3. **Extend the enforced contract.** Update
   `scripts/check-api-landmarks.mjs` and `scripts/check-presentation-browser.mjs`
   with actual keyboard entry, return, cancellation, loading focus movement, and
   missing/tampered-contract checks. Add a fresh held-contract forward-Tab
   control that reaches real readiness without aborting its exact GET, plus
   a controlled missing-contract response after forward focus movement that
   returns visible focus to Plain. Record
   four delayed geometry samples before
   any verifier repair. Enforce required observations in
   `scripts/presentation_browser_verify.py`; independently validate finite
   rectangles against the actual viewport instead of trusting pass flags.
   Add refusal tests in `scripts/test_presentation_browser_verify.py`.
   Keep coverage reachability separate: at most three actual scroll-and-measure
   attempts on the same element, retaining every sample, with no focus repair
   or exception/network retry. Independently validate finite clipped geometry,
   observation context and the final actual unoccluded point. These coverage
   scrolls must never change the observe-only keyboard checks.
4. **Produce and validate exact bytes.** Update only the reviewed entry digest
   in `dagger/frontend_inventory.py`. Generate and analyze the frontend through
   the pinned Dagger engine. Validate focused behavior, then commit the reviewed
   source and generate a complete Dagger Site and unsigned PreviewCandidate.
   Run the complete independent browser suite on the exported bytes, including
   all routes/sections and desktop/mobile JavaScript, no-JavaScript and failed
   enhancement modes. Retain raw findings and producer/consumer evidence.
   Use `SitePresentationReports` to retain exact unsigned payloads and raw
   browser diagnostics when admission refuses; an export success is not
   admission or signing proof. The full browser producer has a fixed 25-minute
   ceiling, records its actual elapsed duration, and saves a failed receipt on
   expiry. The previous 15-minute run completed three modes and reached the
   fourth mode's API disclosures before expiry. Measured phase costs project
   about 17 minutes for the expanded workload; this projection is not a pass.
5. **Review and hand off.** Independently review the implementation, negative
   tests and actual generated-output evidence. Prepare a draft PR with the
   exact source revision and test limits. A separate authorized immutable
   release is required to update the public site; never mutate `v0.3.3` or reuse
   its signatures for changed bytes.

## Acceptance criteria

- Actual keyboard entry retains visible, unoccluded native-summary focus and
  the corresponding heading throughout four delayed samples on both view
  selectors and the far native summary, on desktop and mobile.
- Plain selection, load cancellation and failed enhancement return to a
  readable view with meaningful visible focus. Focus movement during loading
  cannot be undone by a later ready callback.
- Forward Tab to the raw OpenAPI host link during loading retains visible,
  unoccluded focus through real readiness and four subsequent observations.
  Subsequent user input cancels restoration; Scalar's own internal routes are
  outside this host-control intent.
- Missing-contract failure after forward focus movement returns the still-owned
  host control to visible Plain-summary focus, while focus moved outside that
  host remains owned by the user's new target.
- Missing checks, false ownership/visibility flags, offscreen or nonfinite
  rectangles, wrong viewport and non-loading key observations refuse admission.
- The complete mandatory Dagger browser producer and independent replay pass
  on exact generated bytes. The independent full preview has no unexplained
  failures; omitted production backend, signing and native platform exercises
  are explicitly reported.
- Existing OpenAPI identities, native deep links, themes, disclosure semantics,
  signed-download behavior, analysis thresholds and security findings remain
  covered by the existing production gates.

## Migration, rollback and risks

There is no persistent data or API migration. The optional enhancement changes
only after newly produced bytes pass the existing admission and promotion
boundary. Before publication, revert the isolated branch if validation fails.
After publication, recover through another immutable release of reviewed source;
do not overwrite a published tag, artifact, manifest or proof.

The admission contract now requires fourteen enhancement controls. The current
consumer consequently refuses earlier six-control and ten-control browser
receipts. Replaying historical evidence requires its reviewed, version-bound
consumer; that historical result cannot satisfy this stronger promotion gate.
No older receipt is upgraded or substituted. Produce fresh evidence for a new
immutable release, including any reviewed rollback release.

The coverage consumer additionally requires retained bounded scroll samples and
a finite duration within the fixed 25-minute ceiling. Historical receipts
without these fields remain historical evidence under their version-bound
consumer. A failed heading observation and deadline from the earlier full run
remain failed. A separate host probe preserved meaningful visible Scalar model
focus after restoring a retained model route; it did not reproduce the exact
heading miss or establish a Linux cause. This measurement change requires a
complete fresh run and does not justify changing the already-reviewed entry
implementation or weakening any security, coverage or error threshold.

Scalar can adjust scrolling after readiness, and cached rendering can make a
test miss the loading phase. Delayed observations and actual keydown phase
records must therefore remain enforced. Focus restoration must stop when user
intent changes. No-JavaScript verification must keep JavaScript disabled and
use bounded host-side waits when page animation callbacks do not run. Passing
these checks is scoped browser evidence, not a claim of formal WCAG certification
or absence of security findings. Hosted signing is outside unsigned local
validation and requires the existing authorized hosted identity boundary.

## Candidate evidence recovery

The first complete candidate refused artifact vulnerability analysis: the retained
Darwin archive report records a Trivy database lookup DNS failure. A separate
fresh observation of all four archive bytes passed the unchanged policy; it does
not change the earlier result. The next candidate refused the Go SDK SBOM with
twelve publisher lookup errors and one missing dependency-relationship warning.
The retained publisher records show DNS errors. The relationship warning also
reflects a separate inventory gap and cannot be attributed to DNS.

1. Retain exact delivered SDK `go.mod` bytes beside publisher metadata. Replay
   module and requirement identities against the unique manifest-cataloged
   requirements and its SHA-256. Add only declaration relationships, with
   explicit direct/indirect roles. Do not infer installed upstream bytes,
   transitive dependencies or supplier identities. Contradictory, missing or
   unsupported manifests and inventory refuse enrichment.
2. Add an explicit `with-analysis-attempt --attempt=<label>` selection at Trivy vulnerability and publisher-metadata
   observation boundaries, after pinned tool installation. Retain the label and
   purpose with raw findings and enforcing SBOM evidence. An empty default keeps
   existing cache behavior. No automatic retry, cache deletion, success fallback
   or policy exception is introduced. A new observation has new evidence; the
   failed historical observation keeps its result.
3. Select a run/attempt label for CI and release Trivy/publisher observations. Preserve
   existing hosted identity permissions and promotion ordering. PR CI checks out
   the actual head SHA so its artifacts can be compared with reviewed source.
4. Run positive and refusal controls, the nested Dagger module tests and the
   repository contract gate. Freeze a reviewed source commit, then produce a
   complete fresh Site and PreviewCandidate with an explicit attempt. Replay
   actual candidate reports and run the independent six-profile served preview.
   Require unchanged analysis policies and exact artifact/receipt bindings.

These changes require no data migration. Reverting before promotion restores the
previous producer. Historical Go enrichment receipts lacking retained manifest
bytes require their version-bound historical consumer and cannot satisfy this
stronger gate. Neither a successful diagnostic export nor a fresh raw scan grants
candidate admission. External DNS or registry failures continue to block it.

A fresh raw scan may retain additional resolved Go modules and dependency edges.
These are preserved as scanner observations, separately from exact manifest
declarations, only when their unique identities are valid and they are reachable
from the SDK project through the retained graph. Unconnected extras refuse
enrichment. No inferred transitive edge or installed upstream-byte claim is added.

## Native reentry follow-up

The reviewed `2a734a72f080` source passed all four exact-head hosted CI jobs,
but fresh local Site admission and a separately produced raw Site receipt failed
mobile interactive-heading coverage. Both failures remain retained. A passive
Linux trace reproduced a failed first sample without changing the three-sample
rule. Native Enter close/reentry, followed only by passive observation, then
confirmed the focused summary stayed offscreen at -264px. A separate read-only
CDP breakpoint diagnostic attributed the competing intro scrolls to Scalar's
lazy navigation functions; debugger timing effects are explicit.

Scalar's scroll observer retains `#description/...` or `#models/...` in the URL
when the native disclosure closes. On explicit native reentry, its new instance
mistook that previous fragment for a requested deep link. The host summary
restore competed with Scalar's repeated lazy-render scroll. This is distinct
from asynchronous restoration rescheduling through hashchange: the passive
trace captured no hashchange events during the conflicting scrolls.

1. In `dagger/frontend/entry.js`, clear only a Scalar fragment synchronously in
   `rememberEntry()`, which represents explicit native summary or view-link
   activation. Preserve loading host focus handling. Clear
   before fetching so a newer route requested during loading remains intact.
   Re-pin the exact producer source in `dagger/frontend_inventory.py`. Add the
   scoped `site/api-explorer.css` native scroll margin through `site/docs.html`
   so lazy-rendered deep-link controls remain fully inside the viewport. The
   new actual desktop control exposed a retained -0.08px edge clip; keep the
   finite/full-viewport geometry gate unchanged.
2. In `scripts/check-api-landmarks.mjs`, exercise real intro/model routes,
   native disclosure close/reentry, actual Tab from the distant Plain contract,
   and all four passive owned-focus observations. Preserve genuine initial
   model deep links and a newer same-document model navigation during a held
   exact contract response. Neither focus nor scrolling may repair the page
   after activation/readiness.
   Use controlled DOM `node.focus({preventScroll:true})` setup before native close to
   preserve the genuinely loaded model route; record its exact fragment and
   summary focus before Enter. Scrollful setup changed Scalar's scroll-spy
   fragment to introduction, so its failed receipts remain failed. This setup
   does not claim natural keyboard navigation to the close control never scrolls.
   Call DOM focus through locator evaluation: Playwright locator focus accepts
   signal/timeout options and ignores the DOM option. The earlier incorrect locator
   invocation and its failed complete receipts remain retained.
3. Require those checks in `scripts/check-presentation-browser.mjs` and
   `scripts/presentation_browser_verify.py`. Replay the held response digest
   and actual route-focus geometry; refuse missing, misrouted, nonfinite,
   hidden or false evidence in `scripts/test_presentation_browser_verify.py`.
   Keep the existing three-sample coverage rule, fourteen failure/cancellation
   controls, fixed producer deadline and analysis policies unchanged.
4. Freeze reviewed source, then run the complete Dagger Site and unsigned
   PreviewCandidate, actual raw-report replay, independent six-profile served
   preview, SDK consumer receipts and exact-head CI. Scoped browser diagnostics
   and prior successful CI do not replace these acceptance gates.

There is no data migration or new credential requirement. Before promotion,
rollback is a source revert followed by a fresh complete pipeline. Existing
immutable releases and historical failed receipts remain unchanged. Publishing
this correction still requires authorization for a new immutable release.

### Requested model ownership after lazy rendering

The complete local Site run on `d42c76c8e625` failed despite passing exact-head
hosted CI and Dagger All. On mobile the requested ComponentDefinition model
owned focus while the URL named the neighbouring CreateMapping model. The
failure and all unrun later controls remain retained; no candidate was produced.
Scalar's actual installed source observes model sections in a narrow strip at
the viewport centre and updates the URL through `history.replaceState`. A
focused heading near the top can therefore disagree with the observed section.
A scoped CSS-only centre-margin diagnostic clipped the heading and changed its
URL; that approach was discarded, not admitted.

1. Record initial and actual new model fragment intent in the host `route()`.
   After the exact requested model owns focus and the reference is ready,
   centre that focus anchor and preserve its requested URL while lazy siblings
   settle. Before initial model readiness, close Plain and require the exact
   focused target to be fully visible with unchanged measured container/target
   layout, target document position and exact requested hash across three
   consecutive animation-frame observations; require finite, unoccluded focus.
   Also require no real Scalar lazy placeholder inside its installed 1200px
   viewport overscan; three quiet frames alone allowed a deferred neighbour to
   change the URL after readiness. This tests actual pending DOM work, without
   asserting completion of every deferred callback. Keep the
   existing 30-second fail/fallback timeout. An instrumented scroll-call trace found
   readiness was previously declared during roughly 400ms of placeholder
   shrinkage; its failed first samples remain retained. After admission, a
   container ResizeObserver and bounded 250/500/1000ms observations compensate
   layout changes; enforce a monotonic expiry and disconnect after one second.
2. Require the same intent, attempt, open state and exact focus target before
   compensation; pre-admission observations run only during bounded loading,
   and post-admission settlement additionally requires readiness. Record model
   focus ownership before readiness, cancel it at the start of a later focus
   event, then establish any new loading host-control intent. Every key, pointer, wheel or touch gesture, actual
   new hash navigation, changed focus, cancellation or destruction invalidates
   it. Do not intercept Scalar's history implementation or patch its publisher
   modules. Host compensation operates only on its owned requested model.
3. Start the initial deep-link browser control from a blank document so it tests
   a genuine requested initial URL. Also exercise actual same-document model
   navigation in the ready reference and Tab away from model focus. Record four
   passive observations for each; strictly replay actual readiness, geometry,
   route and moved-focus booleans independently. Add a held actual-200 contract
   control that deliberately moves focus from the genuinely focused model to
   the host download link while loading. Observe four ready samples without
   subsequent test repairs. Bind `check-api-model-intent.mjs` and the other
   producer files by digest; reject missing controls, false/non-boolean ownership,
   wrong phases, changed response bytes and nonfinite or clipped rectangles.
   Synthetic fixtures in `presentation_browser_fixture.py` exercise receipt
   refusal only; they are never browser or candidate acceptance evidence. Preserve existing failed receipts and thresholds.
4. Re-pin the actual producer entry, freeze reviewed source, and repeat the
   complete Site, unsigned candidate, All, raw replay, six-profile preview,
   six SDK consumer and exact-head CI gates on that source. Scoped diagnostics
   are iteration evidence, not release acceptance.

The main risk is taking ownership from newer user intent. Actual Tab and newer
route controls must pass before admission; compensation remains bounded and
cancellable. Reverting this source change and rebuilding is the rollback.
