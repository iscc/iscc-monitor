## 2026-06-23 — Stop the verifier app from claiming an un-run did:web signature check

**Done:** Closed the copy-honesty half of the open `normal` "WASM verifier never checks the checkpoint
signature against the hub's did:web key" issue. The Surface-C verifier page (`monitor.iscc.codes`)
no longer lists "Check the signature against the hub's did:web key" as a verification step it runs,
and its tier-2 verdict/in-progress copy no longer claims the browser re-verified a "hub-signed
checkpoint root" — it now states truthfully that the in-browser WASM re-runs only the RFC-6962
inclusion math + the id-binding against the **accepted/committed** root, and that the did:web
signature is verified server-side (certificate §4). No verifier core / WASM / proof code changed.

**Files changed:**
- `internal/verifier/verifier.html`: replaced the false 4th `vstep` ("Check the signature against the
  hub's did:web key") with "Confirm the record commits the requested ISCC-ID" (the id-binding the WASM
  actually checks); changed the 3rd `vstep` label, the in-progress copy, and both `setVerdict`
  ("verified"/"failed") strings from "hub-signed checkpoint root" → "committed/accepted root" (matching
  the certificate's already-honest sibling); updated the loader comment to state the WASM does NOT
  fetch did:web or verify the signature. Left line 500 (`#mismatch-body`, the user's own signed
  evidence) and line 579 (descriptive "hub-signed text" comment) unchanged per next.md — those refer
  to evidence the user holds, not a check the run performed.
- `internal/verifier/handler_test.go`: updated `TestVerifierRendersNamedRegions` to assert the two new
  step labels (dropped the old hub-signed/did:web step strings); added mutation-provable honesty test
  `TestVerifierDoesNotClaimSignatureCheck` (+ `verifiedVerdictLine` helper) asserting the served body
  drops the did:web step, the `verified`-verdict line carries no `hub-signed`, and the `verified` copy
  asserts only the inclusion / accepted-root + id-binding check.

**Verification:** `mise run check` → exit 0 (build + vet + test across all 30 packages green; `gofmt
-l .` empty). Per next.md criteria:
- [x] `mise run check` green.
- [x] `go test -count=1 -run TestVerifier ./internal/verifier` → ok (updated `TestVerifierRendersNamedRegions`).
- [x] NEW `TestVerifierDoesNotClaimSignatureCheck` passes; mutation-proven both ways: (A) re-adding the
  did:web step → FAILs (new test + named-regions test); (B) restoring "hub-signed checkpoint root" to
  the `verified` verdict → FAILs (new test). Restored → green.
- [x] `grep -c "Check the signature against the hub's did:web key" internal/verifier/verifier.html` → 0.
- [x] `setVerdict("verified", …)` line contains no `hub-signed` substring (confirmed by grep + test).
- [x] No new CDN literal (`http://`/`https://`/`cdn.`/`jsdelivr`) in added lines — `TestVerifierNoExternalCDN`
  stays green; the bare text "did:web" (no scheme) is allowed.

**Next:** This code-closes the copy-honesty half of the WASM-verifier `normal`. A future `update-state`/
`review` should narrow that issue to its remaining design-blocked half only (in-browser did:web
resolution + checkpoint-signature verification — out of scope here, needs a design pass) and PRUNE the
already-done/committed M-API contract-accuracy entries (the `verify`-`index` param + `checkpoint`
media-type fixes, committed in `53ee328`, gated by per-op tests; `state.md`/`issues.md` still steer to
them but they are stale, not open). The lone `critical` (dossier→log-browser navigation) stays
human-blocked — do not re-attempt in code.

**Notes:**
- Oracle/conformance gate is **N/A**: no signature, RFC-6962, Merkle, did:web, fsck, or proof code
  touched. `internal/proof/verify` and `cmd/wasm` are unchanged; this is a pure static-template +
  golden-test edit, and the template parses at init so a malformed edit would fail the build, not a
  request (`learnings/verifier.md`).
- The `verified` success copy intentionally adds a positive pointer to where the signature trust DOES
  live ("verified server-side, certificate §4") rather than silently dropping it — per next.md's note
  to "point it at the SERVER / certificate §4 (which IS server-verified), not at this in-browser run."
  It says "did:web" only inside a negation; no scheme, so the no-CDN ban holds.
- Scope: exactly 2 files (1 prod template + 1 test), within next.md's ≤3 file bound. Nothing in
  `## Not In Scope` was touched (cert.html, openapi, the WASM artifact, `cmd/wasm`, `proof/verify`,
  the human-blocked critical all untouched).
- The Verdict-UI honesty auto-memory pattern: this fixes the recurring "page asserts an un-run
  verification" gap at the source (a listed STEP must be one the code runs), not just the rendered ✓.
