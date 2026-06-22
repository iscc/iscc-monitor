# Next Work Package

## Step: Surface-C live wiring — parse `?monitor=`/`?id=`, embed the WASM data-island + loader, gate the verdict/mismatch alert on a real re-verification

## Advances
WASM verifier upgrade milestone — its Verify criterion:
> "identical vectors yield identical verdicts (WASM vs server); the verifier artifact hash matches the
> published value; **a `(size, root)` mismatch renders the guided split-view alert, not a dead error**."

This is the front-of-queue open Verify (state.md "WASM verifier: 1/1 open"; the `review` handoff
`**Next:**` is exactly this deferred Surface-C live-wiring sub-step). It simultaneously **closes the
newest open `normal` issue**: "Surface-C verifier renders the split-view mismatch alert unconditionally
with a present-tense (un-run) verdict" — the alert becomes conditional on a genuine WASM verdict (and,
with no `?monitor=` / no JS, the baseline asserts no un-run verdict). One coherent change advances both
the milestone Verify and the honesty issue.

## Goal
Turn the static `internal/verifier` skeleton into a real in-browser re-verifier: when given
`?monitor=<instance-url>&id=<iscc_id>`, embed the audited WASM loader + a JSON data-island naming the
target, fetch `<monitor>/inclusion/<id>.bundle` client-side, run `isccVerifyInclusion`, and render the
`verified` / `failed` / `error` states honestly — the mismatch alert appears only on a real negative
verdict, never as a static present-tense assertion.

## Scope
- **Modify**: `internal/verifier/handler.go` — parse `?monitor=` and `?id=` into a small page-data
  struct and pass it to `tmpl.Execute` (today it executes `nil`). Validate the target with `net/url`
  (require `Scheme in {http,https}` + non-empty `Host`, reject `Fragment`); fail closed to the
  no-target baseline (`HasTarget=false`) on any reject. Keep the leaf pure (stdlib only): the bundle
  fetch happens in the BROWSER, not the handler — do NOT add `internal/store`/network imports.
- **Modify**: `internal/verifier/verifier.html` — (a) under `{{if .HasTarget}}` emit a
  `type="application/json"` data-island carrying `{monitor, id}` plus an end-of-body `<script>` loader
  mirroring `cert.html` (`/_ds/wasm_exec.js` + `/_ds/verify.wasm`, three distinct `data-state`s
  `error`/`failed`/`verified`) that fetches `<monitor>/inclusion/<id>.bundle`, reads
  `record`/`inclusion.proof`/`checkpoint`/`index`/`size` out of it, and calls `isccVerifyInclusion`;
  (b) make the `.mismatch` alert honest — render it (or a tier-2 result region) so the no-target / no-JS
  baseline asserts NO un-run verdict (illustrative framing "On a mismatch you would see:" OR gated
  behind `HasTarget` + the script), and on a real `failed` verdict the script reveals the guided alert.
- **Modify**: `internal/verifier/handler_test.go` — add two server-rendered-state assertions: (1) no
  query → honest baseline (body lacks the present-tense "does not match" assertion; existing named
  regions + "not yet run" run-label still render); (2) `?monitor=https://monitor.iscc.id&id=ISCC:…` →
  the data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader + the three tier-2 markers render,
  target reflected only inside the data-island.
- **Reference**:
  - `internal/certificate/cert.html:492-583` — the canonical tier-2 caller to mirror: JSON
    `<script type="application/json">` data-island (JSON-context-escaped — do NOT string-interpolate
    base64 into executable JS), `instantiateStreaming(fetch("/_ds/verify.wasm"))` + arrayBuffer
    fallback, `go.run`, `globalThis.isccVerifyInclusion(record, root, proof, index, size)`, and the
    `error`/`verified`/`failed` `data-state` switch.
  - `internal/certificate/handler.go:417-437` — the proof-bundle JSON shape the browser fetches from
    `<monitor>/inclusion/<id>.bundle`: `{iscc_id, hub, checkpoint, inclusion, record, key}` (the
    `inclusion` member is `logclient.InclusionEvidence`). The JS reads `record`/`inclusion`/`checkpoint`
    out of THAT, not a pre-baked island.
  - `.claude/context/learnings/verifier.md` — the unmounted/different-origin posture, the `/_ds/`
    literal-sync convention, the correct `.codes` chrome divergence, the no-CDN body ban, and the exact
    honesty gap this step closes.
  - `.claude/context/learnings/cmd-wasm.md` — the `isccVerifyInclusion` JS signature/contract and the
    three-way verdict distinction (`error` broken-input vs `failed` negative-verdict vs `verified`).
  - `.claude/design/ISCC Monitor - Independent Verification.dc.html` — the mockup region for layout.

## Not In Scope
- **Do NOT mount `verifier.Handler()` in `cmd/iscc-monitor`'s `buildMux`.** The state.md "Next
  Milestone" line suggests mounting it, but this CONFLICTS with `learnings/verifier.md`: Surface C is a
  static site on a DIFFERENT origin (`monitor.iscc.codes`), deployed via GitHub Pages, and must stay out
  of the instance binary. The live wiring is fully testable through `verifier.Handler()` at the HTTP
  seam without mounting it. Chose the learnings rule; flagged here per protocol.
- The **dossier tier-2 caller** (no caller in `internal/dossier` today) — the sibling open WASM
  sub-step; a separate later step.
- The **GitHub Pages / `monitor.iscc.codes` deploy workflow** — the final Surface-C piece; later.
- Moving `safeIndex` into `verifyadapter` + table-testing it (the open `normal` test-gap) — that is a
  `cmd/wasm` touch; this step does not edit `cmd/wasm`, so defer it to a step that does.
- Browser-side end-to-end execution as a *gate* — the JS run is verified by the `review` step's headless
  `agent-browser` visual pass (ADR-0012), not by `go test`; the unit gate asserts only the SSR HTML.
- Do not touch any other open `normal` issue (certificate §5 digest binding, OTS `safeStamp`, `hubDomain`
  ForceQuery, §4/bundle `host:port` DID, §6 timestamp, the certificate tier-2 honesty copy, the `/`
  sub-region deltas) — none is on this step's path.

## Implementation Notes
- **The load-bearing correctness rule (always-loaded learnings):** "gate a rendered ✓/Merkle/mismatch on
  a re-VERIFICATION, not a status flag / static render." The skeleton violates this for the *negative*
  verdict. The fix is symmetric to the certificate's tier-2: the alert/verdict region carries an honest
  no-verdict default and is replaced ONLY by what `isccVerifyInclusion` returns. Keep the three states
  distinct — `error` (broken input / fetch failed) must NOT render the split-view alert; only `failed`
  (a real negative verdict — proof did not rebuild the root) does.
- **Pure leaf, stdlib only.** Validate `?monitor=` with `net/url`; on any reject fall back to
  `HasTarget=false`. For the JS data-island use the `type="application/json"` `<script>` so values are
  JSON-context-escaped (the cert.html pattern), NEVER string-interpolated into executable JS.
- **The data-island here DIFFERS from the certificate's:** the certificate bakes the proof bundle into
  the island server-side (same-origin, it has the data); Surface C is cross-origin and has only the
  *target* — so the island carries `{monitor, id}` and the `<script>` does
  `fetch(monitor + "/inclusion/" + encodeURIComponent(id) + ".bundle")`, then reads
  `record`/`inclusion.proof`/`checkpoint`/`index`/`size` out of the returned bundle JSON before calling
  `isccVerifyInclusion`. Match the `proofBundle`/`InclusionEvidence` field names exactly (read the
  reference). CORS is `*` on every public GET (M3), so the cross-origin fetch is allowed.
- **`/_ds/` paths are template literals synced to `web.*` consts by comment** (verifier does NOT import
  `internal/web`). Reuse the existing `verify.wasm`/`wasm_exec.js` literals already in the template; the
  golden test pins them — keep them byte-identical to `web.WasmVerifyPath`/`web.WasmExecPath`.
- **No-CDN ban still holds:** the existing `TestVerifierNoExternalCDN` bans `http://`/`https://`/`cdn.`/
  `jsdelivr` in the body. The `?monitor=` value (e.g. `https://monitor.iscc.id`) is *runtime* data; if
  it lands in the static golden body the test would trip on `https://`. Reflect the target ONLY inside
  the JSON data-island and scope the no-CDN test to ignore the data-island region (or serve the no-CDN
  test with no `?monitor=`), so the ban stays meaningful for stylesheet/script/img origins (its real
  purpose) without firing on a user-supplied target URL. Document the chosen approach in a comment.
- Preserve the GET-only 405 gate and the buffer-then-200 render discipline already in `Handler`.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test ./internal/verifier` passes, including the two new state assertions.
- No-target baseline: GET `/` (no query) does NOT contain the present-tense assertion "Your (size, root)
  does not match" — the un-run negative verdict is gone (a test asserts the baseline body lacks that
  present-tense claim while the named regions + "not yet run" run-label still render).
- Configured-target: GET `/?monitor=https://monitor.iscc.id&id=ISCC:MAIGKSETI7MJ4EAB` renders the
  `type="application/json"` data-island, the `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader, and the
  three tier-2 `data-state` markers (`error`/`failed`/`verified`), with the target URL reflected only in
  the data-island (the no-CDN body ban still passes).
- Mutation check (advance reproduces): reverting the alert-gating so the mismatch alert renders
  unconditionally in the present tense makes the new no-target baseline test FAIL.

## Done When
`mise run check` and `go test ./internal/verifier` are green, the no-target baseline asserts no un-run
verdict, and a `?monitor=`/`?id=` target renders the live WASM data-island + loader with the three
distinct tier-2 states — closing both the WASM Surface-C mismatch-alert Verify criterion and the open
`normal` honesty issue.
