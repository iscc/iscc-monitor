## 2026-06-22 — Review of: Wire the first SSR WASM `<script>` caller — certificate tier-2 "your browser verified"

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The certificate page (`GET /inclusion/<iscc_id>`) now ships the first real SSR caller of the
in-browser WASM verifier: under `{{if .HasBundle}}` it embeds the proof in a `<script type="application/json">`
data island, loads `/_ds/wasm_exec.js`+`/_ds/verify.wasm` via a progressive-enhancement loader, and renders an
honest tier-2 verdict — verified live end-to-end against the testnet (the WASM ran headlessly and produced the
correct `verified` verdict matching server §3). The `cmd/wasm/main.go` shim's `safeIndex` guard closes the
`js.Value.Int()` truncation in production. All gates green; the work is clean and well-scoped (6 source/test/doc
files), with two non-blocking `normal` follow-ups (an untested guard, a no-JS honesty-copy overstatement).

**Verification:**
- [x] `mise run check` — green (25 packages `ok`); `gofmt -l .` excl. `cauldron/` empty.
- [x] `mise run build:wasm` reproduces `internal/web/verify.wasm` byte-identical (`7d57ab1b…`) == `WasmVerifyHash`; rebuild left no git diff; `TestWasmVerifyHashPinned` PASS.
- [x] `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./cmd/wasm` — OK with the new integer guard.
- [x] `TestVerifyJSON ./cmd/wasm/verifyadapter` — PASS (WASM-vs-server golden parity, adapter untouched).
- [x] `TestCertificate ./internal/certificate` — PASS, incl. `TestCertificateRendersWasmVerifier` (positive + negative subtests); mutation-proven non-vacuous (blanking `RecordB64` → FAIL, restore → PASS).
- [x] No-CDN: `TestCertificateKnownID` (holds the no-CDN body assertions) PASS — same-origin `/_ds/` script+wasm refs do not trip the third-party-CDN ban.
- [x] No-JS baseline asserted in-test AND verified live: §1–§6 + Tier 1/Tier 2 + Download proof bundle all render before the first executable loader `<script src>`; the tier-2 verdict panel's default text is honest with JS disabled.
- [x] Live end-to-end (testnet `ISCC:MAIGKSETI7MJ4EAB`, fresh binary): the certificate served full §1–§3 + all tier-2 markers; agent-browser executed the WASM and rendered a green-bordered `verified` panel ("Your browser re-verified this inclusion proof against the accepted root").
- [x] Gate-integrity scan over all unpushed commits — no `nolint`/`t.Skip`/swallowed errors/build-tag exclusions/deleted assertions.
- [x] Oracle/conformance gate: N/A for the trust-root crypto — no signature/RFC-6962/Merkle/`verifyadapter.VerifyJSON`/`proof/verify` change; only the `cmd/wasm` arg-marshaling guard. WASM-vs-server parity (`TestVerifyJSON`) still green; the live render matched server §3.

**Issues found:**
- (normal, filed) **`safeIndex` integer guard is untested.** The `js.Value.Int()` truncation IS closed in production code (`cmd/wasm/main.go:52-83`, verified live), but `safeIndex` is a pure `float64→(uint64,string)` fn trapped in the `//go:build js && wasm` `main.go`, so no linux test exercises its NaN/fractional/negative/range branches — the build gate only proves it compiles, and the caller's test feeds only valid integers. The original truncation issue's "Verify fixed" (a `1.9` test) was NOT met. Fix: move `safeIndex`+`maxSafeInteger` into `verifyadapter` and table-test. (Reframed the prior `normal` truncation issue to this residual test gap.)
- (normal, filed) **Tier-2 honesty header overstates "This browser re-verifies" on the no-JS baseline** (`cert.html:465`). With JS disabled no verdict runs, yet the `HasBundle` header asserts present-tense re-verification while the verdict panel below correctly hedges "with JavaScript enabled". Make the static header describe only the bundle/offline path; let the script's panel be the sole asserter.

**Codex second opinion:** Finished, one P2 finding (two sub-claims), triaged:
- **Sub-claim 1 (no-JS / load-failure overstatement) — CONFIRMED (minor).** Filed as the tier-2-honesty-header `normal` issue above. Real imprecision on a Tier-1 honesty surface; non-blocking (the verdict panel itself is honest; every clause renders no-JS).
- **Sub-claim 2 (`!HasBundle` branch shows stale "lands in a later release") — REFUTED / false positive.** That copy renders ONLY in the `{{else}}`/no-bundle branch, which is accurate — an uncertifiable id has no bundle and no verifier (the negative test confirms zero tier-2 markers; `grep "land in a later release"` on a certifiable page → 0). Dismissed, no action.

**Visual check:** Done (SSR surface `internal/certificate/cert.html` changed). Launched the freshly-built binary against the testnet, screenshotted `/inclusion/ISCC:MAIGKSETI7MJ4EAB` and the `Certificate.dc.html` mockup with agent-browser. The live page faithfully matches the mockup's clause structure and ADDS the new green `verified` tier-2 panel below the actions (constraint > mockup — the WASM milestone mandates the tier-2 result; the mockup predates it). No NEW visual delta from this increment. Pre-existing tracked deltas unchanged: masthead logo (open `critical`), instance identity (`normal`), §5/§6 (testnet has no OTS; timestamp `normal`).

**Next:** The lone open `critical` is the human-filed **ISCC logo masthead** issue — prioritize it (self-hosted `internal/web` `go:embed`+serve at `/_ds/iscc-logo-black.png`, downscaled, referenced from all six masthead templates). After that, continue the WASM milestone: wire the same tier-2 caller into the **dossier**, then the standalone `monitor.iscc.codes` Independent Verification app (Surface C). The guided **split-view alert** keys on the now-distinct `data-state="failed"` verdict.

**Notes:**
- **`issues.md` reconciled (orchestrator note):** the uncommitted change was legitimate — a `define-next`/`update-state` phase added the human-filed `critical` ISCC-logo issue and extracted sub-item (1) "No logo" from the realm-index-deltas issue into it (cross-referenced). Kept it; committed as part of this review's issue bookkeeping. It is now the front-of-queue `critical`, so DONE is not reachable until it lands.
- **Stray `/wasm` artifact:** the bare WASM build gate (`go build ./cmd/wasm`, no `-o`) again dropped a 2.9 MB `wasm` binary at the repo root (untracked, not gitignored). Deleted it. Still worth adding `/wasm` to `.gitignore` or always using `mise run build:wasm` (it has `-o`).
- **`maxSafeInteger` bound:** the guard rejects `>= 2^53` (i.e. `> maxSafeInteger = 2^53-1`), slightly stricter than the issue's literal "`> 2^53`". Fine — well outside any realistic leaf-index domain; both reject the unsafe range.
- **Two independent re-verifications must agree:** the server §3 ✓ and the browser tier-2 ✓ both gate on the same `proof.VerifyInclusion`-against-the-accepted-root; the data island carries the `record` only when §3 passed (the `if ok` block), so the browser can never re-verify a proof the server declined. Do not weaken either.
