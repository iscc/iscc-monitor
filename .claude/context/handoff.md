## 2026-06-22 — Review of: Rebuild + re-pin verify.wasm to the 6-arg/id-binding shim (close the NEEDS_WORK skew)

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The advance correctly closed the stale-artifact skew at the CONTENT level — the committed
`internal/web/verify.wasm` now carries the 6-arg/id-binding shim (`expected 5 or 6 args`=1, old
`expected 5 args`=0, `decode record envelope`=1) and its SHA-256 matches the re-pinned
`WasmVerifyHash` (`96b2a40d…`), so the deployed verifier would no longer render all-`error`. BUT the
pinned hash is the output of a BARE `go build` (PATH `go1.26.1`), while the DOCUMENTED canonical
regeneration command `mise run build:wasm` runs under `mise`'s `go1.26.4` and deterministically emits a
DIFFERENT artifact (`2c91e61f…`) — so the published audited-artifact pin is NOT reproducible from the
command its own doc names, and next.md's Verification #1 ("`mise run build:wasm` … emits `96b2a40d…`")
is UNMET. This is the STOP-on-divergence condition next.md's Implementation Note called out; the prior
handoff mis-diagnosed the `2c91e61f…` as a "transient cache artifact" when it is the deterministic
mise-toolchain output.

**Verification:**
- [ ] `mise run build:wasm` exits 0 AND emits SHA-256 `96b2a40d…` — **FAIL.** It exits 0 but emits
  `2c91e61f20560fa98e0fbd6813746c40687861d4b4b3be604d4216557df0f48e` (embeds `go1.26.4`), NOT the
  pinned `96b2a40d…` (embeds `go1.26.1`). Only a BARE `go build` (PATH `go1.26.1`) reproduces the pin.
  Reviewer-confirmed: `mise run build:wasm` twice → `2c91e61f…` both times (stable, not transient);
  `mise exec -- go version` → `go1.26.4`; bare `go version` → `go1.26.1`. `mise.toml` pins `go = "1.26"`
  (a floating minor), which mise resolves to its installed `1.26.4`.
- [x] `strings internal/web/verify.wasm | grep -c "expected 5 or 6 args"` → 1; `"expected 5 args"` → 0;
  `"decode record envelope"` → 1 — **PASS** (the committed artifact content is correct).
- [x] `go test -run TestWasmVerifyHashPinned ./internal/web` — **PASS** (committed bytes == committed
  const; this gate cannot catch the toolchain-reproducibility gap, by design).
- [x] `mise run check` — **PASS** (all 27 packages build + vet + test green).
- [x] `gofmt -l .` — empty (clean).
- [x] Scope discipline — only the 2 scoped files changed (`web.go` one-line const re-pin, `verify.wasm`
  regenerated) + the handoff. No `cmd/wasm`/`verifyadapter`/`verifier.html`/`cert.html` source touched.
- [x] Gate-integrity scan over unpushed range — no `nolint`/`t.Skip`/loosened flags; `web_test.go` and
  `mise.toml` untouched. No gate weakened.
- [x] Determinism — both toolchains are individually deterministic (warm + cold cache reproduce their
  own hash byte-for-byte); the divergence is a toolchain-VERSION difference (embedded version stamp),
  not nondeterminism, and the two artifacts are behaviorally identical.

**Issues found:**
- **NEW critical:** "Pinned verify.wasm is NOT reproducible from the documented `mise run build:wasm`
  (bare-go 1.26.1 pin vs mise's 1.26.4) — fails TestWasmVerifyHashPinned on regeneration" (filed in
  issues.md). This is the NEEDS_WORK gate. CI is NOT broken today (CI runs `mise run check`, which only
  checks committed bytes vs const; `pages.yml` copies, never rebuilds), but the reproducibility contract
  — the entire point of an audited-artifact pin — is broken against the documented command.
- **RESOLVED (deleted):** "Source carries the WASM id-binding but the pinned artifact does NOT" — its
  verify-fixed criterion (`grep -c "expected 5 or 6 args"` → 1 AND `TestWasmVerifyHashPinned` passes)
  is now met by the committed tree; the content skew is genuinely closed.

**Codex second opinion:** Codex raised ONE finding, [P2] at `web.go:93` — "Re-pin the WASM artifact
using the mise toolchain": `mise run build:wasm` rewrites the artifact to `2c91e61f…` (embeds
`go1.26.4`) while the pin + committed bytes are `96b2a40d…` (embed `go1.26.1`), so the committed
verifier is not reproducible from the documented command. **CONFIRMED — reviewer-reproduced both
toolchains and read the embedded version stamp from each artifact.** This is exactly the divergence
next.md said to STOP on; filed as the new critical issue and is the basis for the NEEDS_WORK verdict.
(Codex under-stated severity as P2; I treat it as critical because it defeats the audited-artifact
reproducibility guarantee and fails a documented Verification criterion.)

**Visual check:** n/a — no SSR markup/template changed (only the byte artifact + a Go const + handoff).
The WASM is loaded by SSR pages but the rendered surface is unchanged.

**Next:** Fix the toolchain-reproducibility mismatch — a deliberate one-decision pick (per the new
issue): EITHER (a) pin `mise.toml` to the EXACT patch `go = "1.26.4"`, rebuild via `mise run
build:wasm`, and re-pin `WasmVerifyHash` to `2c91e61f…` (makes the documented gate-runner command the
single source of truth, reproducible across machines) — preferred, since `mise` is the project's gate
runner; OR (b) keep `96b2a40d…` but pin `mise.toml` to `go = "1.26.1"` so the documented command
reproduces the committed bytes. Either way: artifact == pin == output of the documented `mise run
build:wasm`. Verify: `mise run build:wasm && sha256sum internal/web/verify.wasm == web.WasmVerifyHash`,
then `go test -run TestWasmVerifyHashPinned ./internal/web` green. This is a tiny, mechanical fix once
the toolchain decision is made.

**Notes:**
- **The `2c91e61f…` hash is NOT a transient/cache artifact** (correcting the prior handoff): it is the
  stable, deterministic output of `mise`'s `go1.26.4`. The committed `96b2a40d…` is the stable output of
  the PATH `go1.26.1`. Both verified across warm + cold (`go clean -cache`) builds.
- The two artifacts are BEHAVIORALLY IDENTICAL — same 6-arg shim + `RecordCommitsID`, same content
  `strings`. The verifier would WORK with either; this is purely a reproducibility-of-the-pin gate, not
  a functional regression. So the live verifier's all-`error` regression (the original NEEDS_WORK) IS
  closed by content; only the reproducibility contract remains.
- Learnings: added a `cmd-wasm.md` TRAP — "`mise run build:wasm` and bare `go build` use DIFFERENT
  toolchains → different bytes; always rebuild + pin via `mise run build:wasm`, and pin `mise.toml` to
  the EXACT patch." Index unchanged (this is package-local, not cross-cutting).
- Working tree left clean (restored `verify.wasm` to the committed `96b2a40d…` after all experiments).
- The deferred signature-half of the verifier trust gap remains the front-of-queue `normal` once this
  reproducibility fix lands; do not loosen `verifier.html`'s "hub-signed root" copy until it does.
