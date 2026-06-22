## 2026-06-22 — Surface-C live wiring: parse `?monitor=`/`?id=`, embed the WASM data-island + loader, gate the verdict/mismatch alert on a real re-verification

**Done:** Turned the static `internal/verifier` skeleton into a live in-browser re-verifier. The handler now
parses `?monitor=<instance-url>&id=<iscc_id>` (validated with `net/url`, fails closed to the no-target
baseline), and the template embeds a `type="application/json"` data-island + an end-of-body WASM loader that
fetches `<monitor>/inclusion/<id>.bundle`, extracts the root from the signed-note checkpoint, runs
`isccVerifyInclusion`, and gates the verdict region on the genuine re-VERIFICATION. The guided split-view
mismatch alert is now honest — illustratively framed in the baseline, revealed with present-tense guidance
only on a real `failed` verdict — closing the open `normal` honesty issue.

**Files changed:**
- `internal/verifier/handler.go`: added a `pageData{HasTarget,Monitor,ID}` view-model + `parseTarget(url.Values)`
  (require non-empty id + http/https monitor URL with a host and no fragment; fail closed otherwise);
  `Handler` now executes the template with the parsed target. Pure stdlib only (added `net/url`); no
  store/network import. Updated the package doc.
- `internal/verifier/verifier.html`: under `{{if .HasTarget}}` emit the `{monitor,id}` JSON data-island +
  `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader (three distinct `data-state`s error/failed/verified,
  mirroring `cert.html`); added a live `#verdict` region (honest no-verdict default) and made the `.mismatch`
  alert illustrative-by-default (`data-live="0"`, "Illustrative — what a real mismatch shows"), revealed with
  the present-tense guidance only when the loader proves a `failed` verdict. Added `.verdict`/`.mismatch-lede`
  CSS through DS tokens.
- `internal/verifier/handler_test.go`: added `serveTarget` helper; new tests — no-target baseline is honest
  (no present-tense "Your (size, root) does not match", "not yet run"/"no verdict is claimed"/"Illustrative"
  still render, no data-island/loader); configured target wires the data-island + loader + three data-states
  with the target reflected only inside the island; malformed/missing targets fall back to the baseline.
  Scoped the no-CDN ban comment to clarify it runs the no-target baseline (so a user `https://` target never
  trips it).

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`, `internal/verifier`
included); `gofmt -l .` (excl. `cauldron/`) empty. Per-criterion:
- No-target baseline asserts no un-run verdict — PASS (`TestVerifierNoTargetBaselineIsHonest`: present-tense
  "Your (size, root) does not match" absent; honest markers + "not yet run" present; no data-island/loader).
- Configured target renders data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader + the three
  `data-state` markers, target only in the island — PASS (`TestVerifierConfiguredTargetWiresLiveVerification`);
  confirmed the rendered island is `{"monitor":"https://monitor.iscc.id","id":"ISCC:MAIGKSETI7MJ4EAB"}`
  (JSON-context-escaped by `html/template`, not string-interpolated).
- Fail-closed validation — PASS (`TestVerifierRejectsMalformedTarget`: wrong scheme / empty host / fragment /
  missing id / missing monitor all render the baseline).
- No-CDN body ban still passes (baseline has no user URL); existing 5 tests still green.
- Mutation check — reverting the `.mismatch-body` to the unconditional present-tense "Your (size, root) does
  not match" makes `TestVerifierNoTargetBaselineIsHonest` FAIL; restored → PASS (reproduced).

**Next:** The sibling open WASM sub-step — the **dossier tier-2 caller** (no caller in `internal/dossier`
today). After that, the **GitHub Pages / `monitor.iscc.codes` deploy workflow** is the final Surface-C piece.
The open `normal` test-gap (move `safeIndex` into `verifyadapter` + table-test it) is still open but needs a
`cmd/wasm` touch, so pair it with a step that edits `cmd/wasm`.

**Notes:**
- **Bundle→`isccVerifyInclusion` arg mapping (load-bearing):** the proof bundle carries the verbatim
  signed-note `checkpoint` text, NOT a bare root. The loader extracts the base64-Std root as the THIRD line of
  the checkpoint body (`<origin>\n<size>\n<base64(root)>\n…`, per `logclient.parseCheckpointBody`) and calls
  `isccVerifyInclusion(bundle.record, root, inclusion.inclusionProof, inclusion.leafIndex, inclusion.treeSize)`.
  Field names match `proofBundle` + `logclient.InclusionEvidence` exactly (verified against
  `certificate/handler.go:430-437` + `logclient/inclusioncheck.go:45-51`).
- Three render states kept strictly distinct: `error` (fetch failed / broken input / bad checkpoint) stays in
  the `#verdict` region and NEVER reveals the mismatch alert; only `failed` (proof did not rebuild the root)
  sets `data-live="1"` on the alert. A network/parse exception is treated as `error`, not a mismatch.
- The browser JS run is NOT a `go test` gate — it is verified by the `review` step's headless `agent-browser`
  visual pass (ADR-0012). The unit gate asserts only the SSR HTML. **Recommend the visual pass exercise both a
  real `verified` bundle and a tampered one to confirm the `failed`→guided-alert path live** (the JS logic is
  only golden-tested as markup, not executed by `go test`).
- Stayed on scope: 3 files, `verifier.Handler()` deliberately NOT mounted in `buildMux` (chose the
  `learnings/verifier.md` cross-origin rule over the state.md "Next Milestone" mount hint, per `next.md`
  Not-In-Scope). `/_ds/` paths kept as template literals byte-identical to the existing template (synced to
  `web.*` consts by comment); no `internal/web` import. No `go.mod`/`go.sum` change. No other open issue touched.
