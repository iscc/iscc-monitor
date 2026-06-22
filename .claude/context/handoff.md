## 2026-06-22 — Wire the first SSR WASM `<script>` caller — tier-2 "your browser verified" on the certificate

**Done:** The certificate page (`GET /inclusion/<iscc_id>`) now ships the first real SSR caller of the
in-browser WASM verifier: for a certifiable id (HasBundle), it embeds the proof data in a
`<script type="application/json">` data island, loads `/_ds/wasm_exec.js` + `/_ds/verify.wasm` via a
progressive-enhancement loader, and calls `globalThis.isccVerifyInclusion(record, root, proof, index, size)`
to render an honest tier-2 verdict (verified / failed / error) — without touching the no-JS baseline (§1–§6
stay server-rendered). The `cmd/wasm/main.go` shim now validates `index`/`size` are non-negative integers
within the JS safe-integer range before the `uint64` narrowing, closing the `js.Value.Int()` truncation
issue at its first real caller. The rebuilt `verify.wasm` is re-pinned.

**Files changed:**
- `cmd/wasm/main.go`: replaced `args[3].Int()`/`args[4].Int()` truncating reads with a `safeIndex` helper
  (NaN/Inf/fractional/negative/`>2^53` → `{verified:false, error:…}`, same shape as the arg-count guard);
  added `import "math"` + a `maxSafeInteger` const.
- `internal/certificate/handler.go`: added `RecordB64 string` to `certData`, populated base64-Std from the
  already-computed `arts.record` inside the §3 re-verification success block (same `if ok` gate as
  `arts.record`/`HasBundle`), so it is empty on every honest decline.
- `internal/certificate/cert.html`: added the tier-2 result `<div id="tier2-result">` (no-JS default text)
  + the JSON data island + the end-of-body `/_ds/wasm_exec.js` loader + inline instantiate/run/call script,
  all under `{{if .HasBundle}}`; added `.tier2*` CSS; corrected the now-stale HasBundle honesty copy
  ("in-browser re-verification lands in a later release" → "This browser re-verifies the proof below").
- `internal/web/web.go`: re-pinned `WasmVerifyHash` to the rebuilt artifact
  (`7d57ab1bbc0b11bd27bd0bc426121dc933158b1342a6656bc90bb5b012f22d2c`).
- `internal/web/verify.wasm`: regenerated via `mise run build:wasm` (the indivisible artifact+pin pair).
- `internal/certificate/handler_test.go`: added `TestCertificateRendersWasmVerifier` (positive +
  negative subtests).
- `CLAUDE.md`: extended the `GET /inclusion/<iscc_id>` bullet to note clauses §1–§6 + bundle render and the
  tier-2 in-browser re-verification result.

**Verification:** `mise run check` → green (all 25 packages `ok`; `gofmt -l .` excl. `cauldron/` empty).
Per-criterion:
- `mise run build:wasm` reproduces `internal/web/verify.wasm` == `WasmVerifyHash` (built twice + again at
  end, byte-identical hash); `TestWasmVerifyHashPinned` PASS.
- `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./cmd/wasm` succeeds with the new integer guard.
- `TestVerifyJSON ./cmd/wasm/verifyadapter` PASS (WASM-vs-server golden parity intact — adapter untouched).
- `TestCertificateRendersWasmVerifier` PASS, mutation-proven non-vacuous (3 mutations reproduced FAIL):
  (1) blanking `data.RecordB64` → empty `"record":""` fails; (2) removing the loader `<script src>` fails;
  (3) un-gating the loader from `{{if .HasBundle}}` fails the negative "uncertifiable wires no verifier"
  subtest. Restoring each → PASS.
- No-CDN: `TestCertificateKnownID` (the no-CDN body assertion at line ~200) PASS — same-origin `/_ds/`
  script+wasm refs do not trip `jsdelivr`/`cdn.`/`unpkg`/`googleapis`; `monitor.iscc.codes` still the one
  permitted external origin. (Note: there is no separate `TestCertificateNoCDN` function — the no-CDN bans
  live inside `TestCertificateKnownID`, despite `next.md` naming it `TestCertificateNoCDN`.)
- No-JS baseline asserted in-test: §1/§2/§3 + Tier 1/Tier 2 + Download proof bundle all render in `<main>`
  BEFORE the first executable loader `<script src=...>` (proving no clause is script-gated).

**Next:** Wire the same tier-2 WASM `<script>` caller into the **dossier** (the next SSR surface that
embeds a verifiable `(size, root)`), then the standalone `monitor.iscc.codes` Independent Verification app
(Surface C, monitor-agnostic via `?monitor=<url>`). The guided **split-view alert** on a `(size, root)`
mismatch is the natural follow-on now that the certificate distinguishes a negative VERDICT (`data-state
="failed"`) from an input error (`data-state="error"`) — the alert keys on the `failed` state.

**Notes:**
- **Out of my commit, not mine:** `git status` showed `.claude/context/issues.md` modified (a new
  critical "add ISCC logo to masthead" issue, added by an earlier `define-next`/`update-state` phase). I
  did NOT touch it and did NOT stage it (protocol: implementer writes only `handoff.md` + source/tests).
- **Stray build artifact gotcha:** running the bare WASM build gate `GOOS=js GOARCH=wasm go build ./cmd/wasm`
  (no `-o`) drops a 2.9 MB `wasm` binary at the repo root. I deleted it; it is NOT gitignored. Prefer
  `mise run build:wasm` (it has `-o internal/web/verify.wasm`), or add `/wasm` to `.gitignore` later. CI
  running the bare gate would leave the same stray file (harmless to the build, but untracked).
- **Honesty-copy fix was in-scope-adjacent but necessary:** the existing HasBundle honesty line claimed
  "in-browser re-verification lands in a later release" on the very page now shipping it — a direct M-UI
  two-tier-honesty contradiction. Fixed only the HasBundle branch; the `!HasBundle` branch (no bundle, no
  in-browser verify) is still accurate and untouched.
- **`data-state` distinction is load-bearing** for the deferred split-view alert: `error` (broken input)
  vs `failed` (negative verdict — the proof did not rebuild the accepted root) vs `verified`. The inline
  script keeps them distinct exactly as cmd-wasm.md / proof-verify.md require; do not collapse them.
- **Streaming + fallback:** the loader uses `WebAssembly.instantiateStreaming` (we serve `application/wasm`)
  with an `arrayBuffer()` fallback and an outer try/catch → graceful degradation to the no-JS default text.
  This is a plain end-of-body inline script (acceptable skeleton per next.md), not a module/bundler.
- No signature / RFC-6962 / Merkle / `verifyadapter.VerifyJSON` / `internal/proof/verify` crypto changed —
  only the `cmd/wasm/main.go` arg-marshaling guard. Oracle/conformance gate: the certificate's §3 path and
  the WASM-vs-server parity (`TestVerifyJSON`) are unchanged and still green.
