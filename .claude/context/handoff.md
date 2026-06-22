## 2026-06-22 — Move the WASM `safeIndex` integer guard into the untagged `verifyadapter` and table-test its reject branches

**Done:** Relocated the JS→Go integer guard (`safeIndex` + `maxSafeInteger`) from the build-tagged
`cmd/wasm/main.go` into the untagged, non-main `cmd/wasm/verifyadapter` package as exported
`SafeIndex`, re-pointed `main.go`'s two call sites, and added a mutation-proven `TestSafeIndex` table.
The guard's branch behavior is now an ordinary linux-tested, golden-gated unit on every `mise run
check` instead of merely compile-checked behind the WASM build tag — closing the `normal` issue "the
WASM `safeIndex` integer guard is trapped in the tagged `main.go` and has NO executable test."

**Files changed:**
- `cmd/wasm/verifyadapter/verify_adapter.go`: added exported `SafeIndex(v float64, name string)
  (uint64, string)` + `maxSafeInteger` const + the `math` import; logic ported byte-for-byte from
  `main.go` (only the package home, exported casing, and the docstring's framing changed — it is now a
  reusable adapter helper, not a `main.go`-local shim). Package stays WASM-pure (`math` is pure stdlib).
- `cmd/wasm/main.go`: deleted the `safeIndex` func, the `maxSafeInteger` const, and the now-unused
  `"math"` import; the two call sites (formerly lines 52/56) now call
  `verifyadapter.SafeIndex(args[3].Float(), "index")` / `(args[4].Float(), "size")`. No behavior change;
  the error-result folding stays in `main.go`.
- `cmd/wasm/verifyadapter/verify_adapter_test.go` (test, not counted): added `TestSafeIndex`
  (table-driven, `package verifyadapter` white-box) covering reject cases `1.9`/`NaN`/`+Inf`/`-1`/`2^53`
  and accept cases `0`/`5`/`2^53-1`, plus a `wantMsg` substring assertion that pins the specific branch
  for the NaN/Inf cases (see Notes — required for non-vacuity). Added `math`/`strings` imports.

**Verification:** `mise run check` → GREEN (all 27 packages ok; `go build ./...`, `go vet ./...`,
`go test ./...`). `gofmt -l .` empty. Per-criterion:
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` → exit 0 (entrypoint still compiles after the move). PASS
- `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` → exit 0 (`verifyadapter` stays WASM-pure,
  the real proof per the learnings rule — not a dep-list grep). PASS
- `go test -count=1 -run TestSafeIndex ./cmd/wasm/verifyadapter` → PASS (8 subtests).
- `go test -count=1 ./cmd/wasm/verifyadapter` → PASS (existing `TestVerifyJSON` golden parity still
  green — relocation introduced no behavior change). PASS
- Assertions: `SafeIndex(1.9,"index")`→`(0, non-empty)`; `SafeIndex(5,"index")`→`(5,"")`;
  `SafeIndex(float64(1<<53-1),"size")`→`(1<<53-1,"")`; `SafeIndex(float64(1<<53),"size")`→`(0,
  non-empty)`. PASS
- Mutation: reverting any one reject branch (NaN/Inf, fractional, lower bound, upper bound — each
  individually) makes `TestSafeIndex` FAIL. PASS (all four; see Notes for the NaN/Inf nuance).
- `safeIndex`/`maxSafeInteger`/`"math"` no longer present in `cmd/wasm/main.go` (grep-confirmed NONE).

**Next:** The two remaining WASM fronts from the prior handoff are still open and untouched here:
(1) the **cross-origin verifier-scope `normal`** — `isccVerifyInclusion` proves only inclusion math, not
checkpoint-signature / did:web-key / id-binding, so a malicious monitor can render a green `verified`
on the cross-origin Surface-C verifier (the most trust-root-meaningful WASM gap; needs its own design
pass — browser did:web resolution + signature verify + id-decode, too large for one ≤3-file step);
(2) the one-time human repo-Settings step for the Pages custom domain (`monitor.iscc.codes`) + its doc
note (`normal`). I'd queue the verifier-scope design pass next — it is the WASM milestone's actual
trust bar — but it likely needs a define-next split into design + implement sub-steps.

**Notes:**
- **Non-vacuity nuance worth review's attention.** My first `TestSafeIndex` (boolean `wantErr` only)
  was VACUOUS for the NaN/Inf branch: `math.NaN() != math.Trunc(math.NaN())` is true (the fractional
  check also catches NaN) and `+Inf > maxSafeInteger` is true (the range check also catches Inf), so
  dropping the `IsNaN||IsInf` branch left the test green — both values just fall through to the next
  reject branch and still produce a non-empty error. I caught this in the mutation sweep and fixed it by
  pinning the **specific error message** (`wantMsg` substring) for the NaN/Inf cases (`"not a finite
  number"`), which the fall-through branches do not produce. After that, all four reject branches fail
  the test under individual mutation. This is a stronger test than the issue's literal spec (it asserts
  branch identity, not just rejection) and is the genuine regression-gate the issue asked for. No bound
  was relaxed; the guard logic is byte-identical to the original.
- The guard is slightly stricter than the issue's "`> 2^53`" framing: it rejects `> maxSafeInteger`
  (i.e. `>= 2^53`), so `2^53` itself is REJECTED and `2^53 - 1` is the last ACCEPTED value. This matches
  the original `main.go` logic verbatim and is well outside any real leaf-index domain (per the existing
  learnings note); I preserved it exactly and pinned the boundary in the table.
- Scope honored: did not touch `internal/proof/verify`, `VerifyJSON`'s signature, the golden vector, or
  the `isccVerifyInclusion` arg-count/arg-order contract; did not widen verifier trust scope; did not
  wire a dossier WASM island; did not touch the Surface-C `readTarget` or the Pages doc note. Two
  production files modified (≤3).
- No signature/RFC-6962/Merkle/did:web/fsck/proof code was touched — this is a pure relocation of a
  marshaling-boundary integer guard plus its test — so the conformance/oracle gate is N/A here; the
  shared `verify.VerifyInclusion` core and its golden vector are unchanged (`TestVerifyJSON` re-confirms
  parity).
