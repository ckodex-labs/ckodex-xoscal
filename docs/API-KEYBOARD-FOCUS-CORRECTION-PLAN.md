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
