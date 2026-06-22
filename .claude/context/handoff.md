## 2026-06-22 — Pin the mise Go toolchain to 1.26.4 and re-pin WasmVerifyHash so the committed verify.wasm is reproducible from `mise run build:wasm`

**Done:** Pinned `mise.toml` to the exact patch `go = "1.26.4"` (mise is the gate runner), rebuilt
`internal/web/verify.wasm` via the documented `mise run build:wasm`, and re-pinned `web.WasmVerifyHash`
to that toolchain's deterministic output. The published pin, the committed bytes, and the
documented-command output are now byte-identical (artifact == pin == command), closing the lone open
critical (the audited-artifact reproducibility contract).

**Files changed:**
- `mise.toml`: `[tools] go = "1.26"` → `go = "1.26.4"` (exact patch so the documented build is
  reproducible across machines, not floated to whatever patch mise has installed).
- `internal/web/web.go`: re-pinned `const WasmVerifyHash` from `96b2a40d…` (bare-go 1.26.1 output) to
  `2c91e61f20560fa98e0fbd6813746c40687861d4b4b3be604d4216557df0f48e` (mise go 1.26.4 output).
- `internal/web/verify.wasm`: regenerated via `mise run build:wasm`; embeds `go1.26.4`, SHA-256 ==
  the new const. Behaviorally identical to the prior artifact (same 6-arg/id-binding shim).

**Verification:** `mise run check` → green (all 27 packages build + vet + test). `gofmt -l .` → empty.
- `mise run build:wasm && sha256sum internal/web/verify.wasm` → `2c91e61f…`; re-running the documented
  command twice leaves `git status` showing no further drift (the artifact is reproducible from its own
  command) — **PASS**.
- That hash string == `web.WasmVerifyHash` — **PASS**.
- `go test -run TestWasmVerifyHashPinned ./internal/web` → `ok` (committed bytes == committed const) —
  **PASS**.
- `strings internal/web/verify.wasm | grep -oE 'go1\.26\.[0-9]+'` → `go1.26.4` (single value) — **PASS**.
- behavior content: `grep -c "expected 5 or 6 args"` → 1, `"expected 5 args"` → 0,
  `"decode record envelope"` → 1 — **PASS** (the rebuild preserved the 6-arg/id-binding shim content).
- `mise exec -- go version` → `go1.26.4` — **PASS**.

**Next:** With this critical closed, the front-of-queue WASM-verifier milestone resumes: the deferred
**signature half** of the verifier trust gap (the did:web-key + checkpoint-note signature check — only
inclusion + id-binding run today; a cross-origin `verified` still trusts the monitor for the signature).
Do NOT loosen `verifier.html`'s "hub-signed root" success copy until that lands. The Pages
repo-Settings enablement (custom domain + "GitHub Actions" source) and the dossier WASM caller are the
other milestone sub-steps (state.md "Next Milestone" 2 & 3).

**Notes:**
- Confirmed pre-edit state matched the review diagnosis exactly: `mise exec -- go version` → 1.26.4,
  bare `go version` → 1.26.1, committed wasm `96b2a40d…` embedding `go1.26.1`. This was option (a) from
  the issue/review (pin the gate runner forward), not (b) (downgrade mise) — preferred because mise is
  the single source of truth for artifact bytes.
- Per `next.md` Not-In-Scope: did NOT touch `wasm_exec.js` (byte-identical across 1.26.1/1.26.4, so the
  `web.go:149` "Go 1.26.1 toolchain" comment stays factually correct), the CI/Pages workflows
  (`go-version: "1.26"` is a separate mechanism that copies, never rebuilds, the pinned wasm), the
  `CLAUDE.md`/ADR/PRD "Go 1.26" stack prose (describes the locked minor series, still accurate), or any
  `cmd/wasm`/`verifyadapter` source (behavior unchanged).
- Working tree clean apart from the three scoped files. No tests added: this is a pure
  reproducibility-pin change; `TestWasmVerifyHashPinned` is the existing gate and it now passes against
  the documented-command output (the gap it could not catch — toolchain-reproducibility — is closed at
  the toolchain-pin level).
