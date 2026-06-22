## 2026-06-22 — Review of: Pin the mise Go toolchain to 1.26.4 and re-pin WasmVerifyHash so the committed verify.wasm is reproducible from `mise run build:wasm`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance pinned `mise.toml` to the exact patch `go = "1.26.4"`, rebuilt
`internal/web/verify.wasm` via the documented `mise run build:wasm`, and re-pinned
`web.WasmVerifyHash` to that toolchain's deterministic output (`2c91e61f…`). The
audited-artifact reproducibility contract is now restored — artifact == pin ==
documented-command output, byte-for-byte across two consecutive rebuilds — closing the lone
open `critical`. Scope is exactly the three files `next.md` specified (2 non-binary source +
1 regenerated artifact); nothing from `## Not In Scope` was touched.

**Verification:**
- [x] `mise run build:wasm && sha256sum internal/web/verify.wasm` → `2c91e61f20560fa98e0fbd6813746c40687861d4b4b3be604d4216557df0f48e`, byte-equal to the committed artifact — **PASS** (reviewer ran it TWICE; both runs left `git status` showing `verify.wasm` clean → reproducible-from-command, the exact contract the critical broke).
- [x] That hash string == `web.WasmVerifyHash` (`internal/web/web.go:93`) — **PASS**.
- [x] `go test -count=1 -run TestWasmVerifyHashPinned ./internal/web` → `ok` (forced fresh, not cached) — **PASS**.
- [x] `strings internal/web/verify.wasm | grep -oE 'go1\.26\.[0-9]+'` → `go1.26.4` (single value; was `go1.26.1`) — **PASS**.
- [x] Content markers survived the rebuild: `expected 5 or 6 args` → 1, `expected 5 args` → 0, `decode record envelope` → 1 — **PASS** (6-arg/id-binding shim intact).
- [x] `mise exec -- go version` → `go1.26.4`; bare `go version` → `go1.26.1` (confirms mise.toml now pins the exact patch and is the canonical toolchain) — **PASS**.
- [x] `mise run check` → green, all 27 packages build + vet + test — **PASS**.
- [x] `gofmt -l .` → empty — **PASS**.
- [x] Conformance/oracle gate — N/A: the diff touches no signature/Merkle/proof code; only the build-toolchain pin + the byte-identical-behavior wasm artifact. Confirmed `internal/proof/verify` stays import-pure (no `net`/`net/http`/`database/sql`) and `GOOS=js GOARCH=wasm go build ./internal/proof/verify ./cmd/wasm/verifyadapter ./cmd/wasm` succeeds. CI `notecheck` oracle config unchanged (advance touched 0 files under `.github/`; 8 `notecheck` refs still present in `ci.yml`).
- [x] Gate-circumvention scan over all 11 unpushed commits — no `//nolint`, `t.Skip`, build-tag exclusions, or deleted assertions in the diff.

**Issues found:** (none) — the increment met its goal cleanly; the resolved reproducibility `critical` was deleted from `issues.md` after reviewer-verifying the fix.

**Codex second opinion:** Clean — "The committed WASM hash matches the regenerated artifact, `mise run build:wasm` leaves the tree clean, and `mise run check` passes. I found no introduced correctness issues in the patch." Independently corroborates the reviewer's reproducibility + gate verification; no findings to triage.

**Visual check:** n/a — no SSR surface changed. The diff is a build-toolchain pin + a behaviorally-identical regenerated wasm artifact; no template, render path, or chrome touched.

**Next:** The lone `critical` is closed and pushable. Resume the front-of-queue **WASM-verifier upgrade** milestone (state.md "Next Milestone" 2 & 3): the deferred **signature half** of the verifier trust gap (browser did:web-key resolution + checkpoint-note signature verify — only inclusion + id-binding run today; a cross-origin `verified` still trusts the monitor for the signature; filed `normal`). This one needs a DESIGN PASS before building (browser-side did:web resolution is non-trivial) — a good STOP-candidate if the design is unclear. Do NOT loosen `verifier.html`'s "hub-signed root" success copy until the signature check lands. The other open milestone sub-steps: the Pages repo-Settings enablement (custom domain + "GitHub Actions" source — a human step, filed `normal`) and the dossier WASM caller.

**Notes:**
- All 11 unpushed CID commits push together on this PASS (prior cycles were NEEDS_WORK and held them back). Upstream is `origin/develop`; never push `main`.
- Learnings: collapsed the cmd-wasm.md two-toolchain TRAP into a `settled:` entry recording the durable rule (pin the EXACT patch, never the floating minor; rebuild+re-pin via `mise run build:wasm` in one increment on a Go bump) — the prediction it made is now landed, so it no longer reads as pending work.
- This was option (a) from the issue/next.md (pin the gate runner forward), the preferred path since mise is the single source of truth for artifact bytes. Option (b) (downgrade mise to 1.26.1) was correctly NOT taken.
- The `web.go:149` comment still says "Go 1.26.1 toolchain's lib/wasm/wasm_exec.js" — confirmed correct and in scope to leave alone: `next.md` Not-In-Scope verified those bytes are byte-identical across 1.26.1/1.26.4, so the loader is unaffected by the build-toolchain switch.
