## 2026-06-22 — Review of: Bind the WASM verifier's verdict to the requested id (id-binding half of the verifier-scope trust gap)

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The Go-level work is excellent — a pure, mutation-proven `verifyadapter.RecordCommitsID`
preserving the three-way verdict contract, wired into a 5-or-6-arg shim that protects the certificate's
live 5-arg caller (a well-reasoned, flagged deviation from the literal `5→6` bump). But the increment
ships a self-contradiction: `verifier.html:628` now passes a 6th `id` arg while the PINNED
`internal/web/verify.wasm` artifact the page actually loads was NOT rebuilt and still has the old
`expected 5 args` shim — so the deployed Surface-C verifier would render `error` for EVERY target. The
step's user-visible goal is not achieved by the committed tree; the fix is a one-command rebuild + re-pin.

**Verification:**
- [x] `mise run check` green — all 27 packages ok (build + vet + test).
- [x] `gofmt -l .` empty — no formatting failures.
- [x] `GOOS=js GOARCH=wasm go build ./cmd/wasm` exits 0.
- [x] `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` exits 0 (adapter stays WASM-pure; new
  `encoding/json`+`strings` pull no `net`/`os`/`syscall/js`).
- [x] `go test -count=1 ./cmd/wasm/verifyadapter` PASS — new `TestRecordCommitsID` (6 cases) +
  unchanged `TestVerifyJSON` (golden parity) + `TestSafeIndex`.
- [x] Mutation non-vacuity (I re-ran all three, file restored byte-identical):
  collapse-mismatch-to-true → 2 cases FAIL; drop `strings.TrimPrefix` → bare-request case FAIL;
  parse-error→bare-false → both broken-input cases FAIL.
- [x] Three-way contract + render-state distinctness verified: id mismatch → `verified=false,errMsg==""`
  → `failed` (negative verdict); decode/parse fault → `errMsg!=""` → `error`. The cert's 5-arg call
  (`cert.html:565`) is preserved by the optional-6th-arg design.
- [x] `idEnvelope` correctly mirrors `logclient.recordEnvelope.IsccID` (top-level `iscc_id`, tag matches).
- [x] Quality-gate integrity scan (unpushed commits) — no `nolint`/`t.Skip`/swallowed-error/build-tag/
  deleted-assertion in the `.go` diff.
- [ ] **Live id-binding actually takes effect** — FAILS. The committed `internal/web/verify.wasm` is
  stale (`strings … | grep -c "expected 5 or 6 args"` → 0; `… "expected 5 args"` → 1; `… "decode record
  envelope"` → 0). A clean `mise run build:wasm` yields a DIFFERENT hash (`96b2a40d…` vs committed
  `7d57ab1b…`) that DOES carry the new shim. The 6-arg `verifier.html` call against the old wasm returns
  `error` for every target.
- Oracle/conformance gate: N/A — no signature/RFC-6962/Merkle/did:web/fsck/proof code touched; the
  id-binding is a pure string-canonicalize + byte-compare. `TestVerifyJSON` re-confirms inclusion parity.

**Issues found:**
1. **(NEEDS_WORK gate, new `normal`)** Source carries the id-binding but the pinned `verify.wasm` does
   not — the deployed verifier would render `error` for every target until `mise run build:wasm` +
   re-pin `web.WasmVerifyHash`. The fix is verified-reproducible (rebuilt hash `96b2a40d…`). Filed.
2. The existing `normal` "WASM verifier proves only inclusion math" issue is NARROWED — its id-binding
   half is now closed in source; the signature half (browser did:web resolution + note-signature) stays
   open, design-first. Issue text updated to reflect the narrowing.

**Codex second opinion:** One **[P1]** finding (`verifier.html:628`): the 6-arg call runs against a
shipped `verify.wasm` that was not updated and still has the old `expected 5 args` guard, so every live
Surface-C verification returns `out.error`. **CONFIRMED** — reviewer-reproduced via `strings` on the
committed artifact (old shim) + a clean rebuild (new shim, different hash). This is exactly the
source-vs-artifact skew the handoff flagged; it is what drives the NEEDS_WORK verdict. Filed as a
`normal` issue (#1 above). No other Codex findings.

**Visual check:** n/a — `internal/verifier` is the standalone Surface-C Pages page (not mounted in the
instance binary), and a meaningful screenshot needs the working WASM + static deploy, both blocked
(stale wasm above + the human Pages-Settings gate). No instance SSR surface changed.

**Next:** Close the artifact skew — a small advance: `mise run build:wasm` to rebuild
`internal/web/verify.wasm` with the 6-arg/id-binding shim, then re-pin `web.WasmVerifyHash` to the
emitted SHA-256 (`TestWasmVerifyHashPinned` gates it). Confirm
`strings internal/web/verify.wasm | grep -c "expected 5 or 6 args"` → 1. That makes the live verifier
actually gate on id-binding and completes this step's goal. After that, the front-of-queue WASM work is
the deferred signature-verify half (browser did:web resolution + checkpoint-note signature) — needs a
design pass first.

**Notes:**
- The 5-or-6-arg design deviation is SOUND and well-defended: a hard `!= 6` would have regressed the
  certificate's live tier-2 verifier (5-arg `cert.html:565`) to an all-`error` state with no in-scope
  fix, and the cert test (markup-only) would not have caught it. Keep the optional-arg posture until a
  later increment pairs a strict guard with a `cert.html` 6th-id edit in the SAME step.
- The stale-artifact trap is now recorded in `learnings/cmd-wasm.md`: any runtime change to
  `cmd/wasm`/`verifyadapter` must be followed by `mise run build:wasm` + re-pin in the SAME increment;
  `mise run check` cannot catch the skew (it builds to `/tmp`; `TestWasmVerifyHashPinned` only checks the
  committed bytes match the committed hash — both stale → green).
- Not pushing: verdict is NEEDS_WORK (3 commits remain ahead of `origin/develop`; the next cycle's fix
  lands with them).
