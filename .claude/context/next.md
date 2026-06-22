# Next Work Package

## Step: Move the WASM `safeIndex` integer guard into the untagged `verifyadapter` and table-test its reject branches

## Advances
WASM milestone Verify criterion (target.md): *"identical vectors yield identical verdicts (WASM vs
server)."* A truncated / non-integer / out-of-safe-range JS Number is **not** an identical vector —
the `safeIndex` guard is what rejects those before the `float64→uint64` narrowing silently truncates
(`js.Value.Int()` is `int(v.Float())`). Today that guard lives in the build-tagged `cmd/wasm/main.go`,
so **no linux `go test` exercises it** — the WASM build gate proves only that it compiles, and the one
SSR caller's test feeds valid integers only. This step makes the guard's branch behavior an ordinary
linux-tested, golden-gated unit, closing the open `normal` issue **"The WASM `safeIndex` integer guard
is trapped in the tagged `main.go` and has NO executable test"** (issues.md). The issue's own "Verify
fixed" criterion (a test feeding `index=1.9` asserting `verified=false, error!=""`) was never met.

Why this over the other two WASM fronts in the handoff Next: (1) the **dossier WASM island** would
contradict a settled review decision — the dossier's tier-2 affordance is *deliberately* the
cross-surface link to Surface C because it has no single ISCC-ID subject to re-verify
(`learnings/dashboard.md`); (2) the **verifier-scope signature/id-binding** gap is the most
trust-meaningful WASM `normal` but is too large for one ≤3-file step (browser did:web resolution +
signature verify + id-decode, needs its own design pass). The "published" half of the milestone is
blocked on a one-time human repo-Settings step and cannot close autonomously. This `safeIndex` step is
the code-closable WASM increment the state.md DRIFT-WATCH asks for.

## Goal
Relocate `safeIndex` + `maxSafeInteger` from the build-tagged `cmd/wasm/main.go` into the untagged,
non-main `cmd/wasm/verifyadapter` package and add a table-driven test for its reject branches, so the
JS→Go integer contract that protects the WASM-vs-server vector parity is regression-gated on every
`mise run check` instead of only compile-checked behind the WASM build tag.

## Scope
- **Create**: (none — extend the existing test file)
- **Modify** (2 non-test/doc production files):
  - `cmd/wasm/verifyadapter/verify_adapter.go` — add the exported `SafeIndex(v float64, name string)
    (uint64, string)` + `maxSafeInteger` const, ported verbatim from `main.go` (logic byte-for-byte
    unchanged; only the package home + exported name change).
  - `cmd/wasm/main.go` — delete the now-moved `safeIndex` func + `maxSafeInteger` const and the
    `"math"` import; call `verifyadapter.SafeIndex(...)` at the two call sites (lines 52, 56). No
    behavior change.
  - `cmd/wasm/verifyadapter/verify_adapter_test.go` (test file — does NOT count toward the ≤3 limit) —
    add `TestSafeIndex` table covering the reject + accept branches.
- **Reference**:
  - `.claude/context/learnings/cmd-wasm.md` — the layout rule (tagged glue vs untagged adapter; WHY an
    untagged file in `package main` breaks `go build ./...`) and the `safeIndex`-is-trapped forward rule.
  - `.claude/context/learnings/proof-verify.md` — the three-way verdict contract `VerifyJSON` preserves
    (don't disturb it; this step is upstream of it).
  - `cmd/wasm/main.go` (lines 20-24, 47-83) — the source `safeIndex` + `maxSafeInteger` to port verbatim.
  - `cmd/wasm/verifyadapter/verify_adapter.go` + `verify_adapter_test.go` — the destination package and
    its existing test conventions (table-driven, `package verifyadapter` white-box, golden-vector reuse).

## Not In Scope
- **Do NOT** widen the WASM verifier's trust scope (checkpoint-signature / did:web-key / id-binding).
  That is the separate, larger open `normal` (issues.md "The WASM verifier proves only inclusion math")
  and needs its own design pass — not this step.
- **Do NOT** wire a WASM proof island into the hub dossier. The dossier's tier-2 affordance is
  *deliberately* the cross-surface link to Surface C (`learnings/dashboard.md`: "no single ISCC-ID
  subject to re-verify") — adding an island there contradicts a settled review decision.
- **Do NOT** touch `internal/proof/verify`, `VerifyJSON`'s signature, the golden vector, or the
  `isccVerifyInclusion` arg-count / arg-order contract — this is a pure relocation + test, no behavior
  change to verification.
- **Do NOT** change the `safeIndex` logic (the `>= 2^53` strictness, the NaN/Inf/fractional/negative
  branches) — port it byte-for-byte; only its package and exported casing change.
- **Do NOT** address the Surface-C `readTarget` `u.href` normalization or the Pages custom-domain doc
  note here — separate filed issues, different files.

## Implementation Notes
- **Port verbatim, only relocate.** Lift `maxSafeInteger` (`= float64(1<<53 - 1)`) and the `safeIndex`
  body unchanged into `verify_adapter.go`. Export it as `SafeIndex` (capital S) because it now crosses
  the package boundary — same reason the adapter's `VerifyJSON` is exported (`learnings/cmd-wasm.md`:
  "its exported fn is `VerifyJSON` (capitalized), not the same-package lowercase a single-package layout
  would use"). Keep the evergreen docstring (update it to drop the `main.go`-local framing — it is now a
  reusable adapter helper, not a shim-local one).
- **`verifyadapter` stays WASM-pure.** `SafeIndex` needs `math` (`IsNaN`/`IsInf`/`Trunc`) — `math` is
  pure stdlib and WASM-safe, so the package's load-bearing purity (no `net`/`os`/`syscall/js`) is
  preserved. Confirm with `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` (the real proof; do
  NOT grep the dep list — `os` appears transitively via `fmt`, per `learnings/cmd-wasm.md`).
- **`main.go` must keep building under the WASM tag.** After moving the func + const out, `main.go` no
  longer uses `math` directly — remove the `"math"` import or `gofmt`/the WASM build will fail on the
  unused import. Re-point the two call sites (lines 52, 56) to
  `verifyadapter.SafeIndex(args[3].Float(), "index")` / `(args[4].Float(), "size")`. The error-result
  folding at the call sites stays in `main.go` (it returns the JS `map[string]any{"verified":false,
  "error":errMsg}`).
- **Test the branches the issue named.** Add `TestSafeIndex` (table-driven, in the existing
  `verify_adapter_test.go`, `package verifyadapter`): reject cases `1.9` (fractional), `NaN`
  (`math.NaN()`), `+Inf` (`math.Inf(1)`), `-1` (negative), and `2^53` (`float64(1<<53)`, just above the
  cap) — each asserts `errMsg != ""` and `got == 0`; accept cases `0` and `5` — each asserts
  `errMsg == ""` and `got == uint64(v)`. Assert the boundary precisely: `maxSafeInteger`
  (`2^53 - 1`, `float64(1<<53-1)`) is ACCEPTED, `2^53` is REJECTED (the guard is `> maxSafeInteger`).
- **Correctness rule (learnings.md, always-loaded):** *"a built proof is not a verified proof / fail
  closed."* A truncated index would verify against the wrong-but-truncated leaf — `SafeIndex` is the
  fail-closed gate that prevents that, so its reject branches MUST be executable-tested, not merely
  compiled. Keep every branch fail-closed (return `0, <msg>`); do not relax any bound.
- **Non-vacuity check (do this before declaring done):** reverting any single branch in `SafeIndex`
  (e.g. dropping the `v != math.Trunc(v)` fractional check, or the `> maxSafeInteger` upper bound)
  must make `TestSafeIndex` FAIL. If a revert leaves the test green, the table is vacuous — strengthen
  it. This is the regression-gate the issue requires.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` exits 0 (the WASM entrypoint still compiles after the move).
- `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` exits 0 (`verifyadapter` stays WASM-pure).
- `go test -count=1 -run TestSafeIndex ./cmd/wasm/verifyadapter` passes.
- `go test -count=1 ./cmd/wasm/verifyadapter` passes (the existing `TestVerifyJSON` golden parity still
  green — relocation introduced no behavior change).
- Assertion: `SafeIndex(1.9, "index")` returns `(0, errMsg)` with `errMsg != ""`; `SafeIndex(5, "index")`
  returns `(5, "")`; `SafeIndex(float64(1<<53-1), "size")` returns `(1<<53-1, "")`; `SafeIndex(float64(1<<53),
  "size")` returns `(0, errMsg)` with `errMsg != ""`.
- Mutation: reverting any one `SafeIndex` reject branch makes `go test -run TestSafeIndex
  ./cmd/wasm/verifyadapter` FAIL.

## Done When
`mise run check` and both WASM builds are green, `TestSafeIndex` passes and is mutation-proven
non-vacuous, and `safeIndex`/`maxSafeInteger` no longer live in the build-tagged `cmd/wasm/main.go` —
the guard is now an untagged, linux-tested `verifyadapter.SafeIndex`, closing the `normal` issue.
