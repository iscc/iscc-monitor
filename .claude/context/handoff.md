## 2026-06-22 — Gate Surface-C's live verification CLIENT-side so the static GitHub-Pages artifact actually works

**Done:** Moved the `?monitor=&id=` target resolution out of the Go handler and into the
always-emitted browser loader (`new URLSearchParams(location.search)` + a JS port of `parseTarget`'s
validation), so `internal/verifier` is now a single static artifact rendered identically for every
request. A statically-generated `index.html` runs the WASM verdict client-side when loaded at
`/?monitor=…&id=…` and shows the honest no-target baseline otherwise — closing the filed `normal`
issue "Surface-C live wiring is gated on SERVER-side `.HasTarget`".

**Files changed:**
- `internal/verifier/handler.go`: dropped `parseTarget`, `pageData`, the `HasTarget`/`Monitor`/`ID`
  fields, and the `net/url` import; `Handler` now renders unconditionally with `tmpl.Execute(&buf, nil)`.
  Kept the GET-only 405 guard, buffer-then-200 render, and post-200 write-drop. Package + `Handler`
  docstrings rewritten to state the target is read CLIENT-side from `location.search`, never reflected.
- `internal/verifier/verifier.html`: removed every `{{if .HasTarget}}`/`{{.Monitor}}`/`{{.ID}}` branch
  and the `<script id="verify-target">` data-island; the end-of-body loader is now ALWAYS emitted and
  reads `new URLSearchParams(location.search)`, applying the same validation `parseTarget` did
  (non-empty id; `new URL(monitor)` with http/https `protocol`, non-empty `host`, no `hash`, wrapped in
  try/catch). On no/invalid target it returns early leaving the honest baseline untouched. The
  present-tense run-label / verdict copy ("verifying in your browser…", "Re-verifying…") moved into the
  JS, set only after a valid target is found; the static body keeps "not yet run" / "no verdict is
  claimed" / the `data-live="0"` illustrative mismatch. Bundle-fetch + root-derivation + three-state
  gating reused verbatim — only the source of `{monitor, id}` changed.
- `internal/verifier/handler_test.go` *(test)*: dropped `serveTarget` and the server-side-parse tests
  (`TestVerifierConfiguredTargetWiresLiveVerification`, `TestVerifierRejectsMalformedTarget`); added
  `TestVerifierStaticBodyAlwaysCarriesLoader` (loader + `URLSearchParams`/`location.search` always
  present), `TestVerifierNoServerSideTarget` (no `verify-target` island, no reflected query, query-bearing
  render byte-identical to baseline), and `TestVerifierLoaderHasThreeDistinctStates`; reworked
  `TestVerifierNoTargetBaselineIsHonest` to assert against the DISPLAYED (no-JS, script-stripped) body.

**Verification:** `mise run check` → GREEN (`go build`/`go vet`/`go test ./...` all ok, `gofmt -l .`
empty). `GOOS=js GOARCH=wasm go build ./cmd/wasm` → OK (WASM gate unchanged). `go test -run TestVerifier
./internal/verifier` → all 9 PASS. Per-criterion:
- Static body ALWAYS carries the loader (`/_ds/wasm_exec.js`, `/_ds/verify.wasm`, `isccVerifyInclusion`,
  `URLSearchParams`, `location.search`) on a plain no-query GET — PASS.
- Static body asserts no un-run verdict (no displayed "Your (size, root) does not match"; keeps "not yet
  run" / "no verdict is claimed" / "Illustrative — what a real mismatch shows") — PASS.
- No server-emitted data-island (`id="verify-target"` absent; no `?monitor=`/`?id=` reflected) — PASS.
- Named regions, independence statement, guided mismatch alert, no-CDN ban, 405 non-GET — all PASS.
- Mutations: (A) reverting the honesty copy into displayed markup FAILS
  `TestVerifierNoTargetBaselineIsHonest`; (B) removing the `URLSearchParams(location.search)` read FAILS
  `TestVerifierStaticBodyAlwaysCarriesLoader`; both restored → green.

**Next:** The Surface-C deploy sub-step is now unblocked — generate the static `index.html` (render
`verifier.Handler` once to a file) + the `/_ds/` assets and publish to GitHub Pages at
`monitor.iscc.codes` via a `.github/workflows/*` (still out of scope here, per next.md). Note the
remaining open verifier-scope issue: the WASM core proves inclusion math only (no checkpoint-signature /
did:web-key / id-binding check) — the step list still lists "Check the signature against the hub's
did:web key", which the core does not run; that honesty/scope fix is its own filed `normal` issue.

**Notes:**
- **Design reconciliation (flag for review, not a deviation):** next.md's Verification criterion
  ("static body does NOT contain the present-tense `Your (size, root) does not match`") and its
  Implementation Note ("move the present-tense copy INTO the JS") are in literal tension once the loader
  is ALWAYS emitted — the copy is necessarily a JS string literal in the always-present `<script>`, so a
  raw whole-body substring ban is unsatisfiable. I honored the stronger directive (move copy into JS) and
  scoped `TestVerifierNoTargetBaselineIsHonest` to the DISPLAYED body (loader `<script>` blocks stripped
  via a regexp helper `displayedBody`) — the genuine honesty boundary is what a no-JS reader SEES, not
  what sits inside a never-displayed script. The present-tense strings are set only by `setVerdict`/the
  in-progress branch at runtime after a valid target. Mutation A proves this is non-vacuous (moving the
  copy back into displayed markup fails the test).
- Scope: 2 non-test/doc files (`handler.go`, `verifier.html`) + 1 test file — within the ≤3 budget. No
  `cmd/wasm` / `internal/proof/verify` touched; the verifier core is unchanged. Not mounted in
  `cmd/iscc-monitor` (Surface C stays a different origin).
- The `/_ds/...` literals stay synced-by-comment to `web.*`; the no-CDN body ban still holds (the user
  `monitor` URL never touches the static body now — it lives only in `location.search` at runtime).
