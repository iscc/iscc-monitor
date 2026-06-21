# Next Work Package

## Step: Validate the ISCC-IDv1 Length nibble in `internal/index.Decode`

## Advances
Closes the open `normal` issue **"ISCC-IDv1 decoder accepts a nonzero Length nibble (fail-closed gap
on the trust root)"** — Codex-found P2, reviewer-confirmed against the hub schema + golden vectors,
and the explicit `**Next:**` in `handoff.md`. This preempts new milestone feature work because it is
the trust root of the M-UI **certificate of inclusion** Verify criterion: the realm-wide certificate
is "keyed by the self-describing ISCC-IDv1 — decode realm + 12-bit `hub_id`, resolve the issuing hub
via the registry" (`target.md` M-UI Verify). A wrong/lenient decode routes a malformed id to a real
hub and proves the wrong leaf, so the decoder must fail closed before the next sub-step (the
`hub_id` → hub resolver) consumes it. The fix also restores the decoder's own fail-closed contract
("a header that is not an ISCC-IDv1 ... must return a descriptive error").

## Goal
Make `Decode` reject any ISCC-IDv1 whose header **Length nibble** (`raw[1] & 0xF`) is nonzero, so a
non-canonical header like `MAIQAAAAAAAAAAAA` (byte1 = 0x11) is rejected instead of mis-read as a
64-bit body. Both real golden vectors (Length nibble 0) must keep decoding unchanged.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/index/iscc.go` (add the Length-nibble guard before reading the body; the file's own
    docstring already lists Length as a header nibble it validates, so this also makes the doc true)
  - `internal/index/iscc_test.go` (test — does not count against the ≤3 non-test/doc budget)
- **Reference**:
  - `.claude/context/learnings/index.md` (layout facts; the open Length-gap note; the
    "golden test tied to ground truth, not the symbol" rule)
  - `.claude/context/learnings.md` (the always-loaded index — the `proof/verify` purity rule applies:
    keep `internal/index` import-clean / WASM-shareable)

## Not In Scope
- The 12-bit-`hub_id` → issuing-hub resolver (registry / ADR-0010 Hub-List) — the next sub-step *after*
  this lands and pushes; do not start it here.
- The `/inclusion/{iscc_id}` HTML certificate page or the downloadable proof-bundle assembler (the
  oracle-gate slice) — later iterations.
- Exporting `encode` or widening the public surface beyond `Decode` + `ISCCID` (YAGNI, per
  `learnings/index.md`).
- Refactoring the hardcoded header-nibble constants is optional polish; adding a single `lengthV1`
  const next to `versionV1` for the literal `0` is fine, but keep the change minimal — do not rework
  the existing constants.
- The `low` "single-record label test is vacuous" issue and the other 5 `low` items — loop-skipped.

## Implementation Notes
- **The guard.** After the existing Version check in `internal/index/iscc.go` (currently lines 91-94)
  and before `realm := raw[0] & 0xF` / the body read `binary.BigEndian.Uint64(raw[2:10])`
  (lines 95-96), add a check on the low nibble of `raw[1]`:
  - a canonical ISCC-IDv1 has **Length nibble 0** (a 64-bit body); reject `raw[1] & 0xF != 0` with a
    descriptive `fmt.Errorf` in the same style as the MainType/Version errors (include the offending
    nibble value and the `isccID`), e.g.
    `"index: ISCC-IDv1 Length nibble %d is unsupported (want 0): %q"`.
  - This is a ≤1-line guard plus the error return; do not touch the body-decode arithmetic.
- **Test (mirror `TestDecodeWrongVersion`'s shape — it is the right ground-truth pattern).** Add a
  focused test (e.g. `TestDecodeRejectsNonzeroLength`) that:
  1. constructs the malformed header bytes directly:
     `raw := []byte{0x60, 0x11, 0, 0, 0, 0, 0, 0, 0, 0}` (MainType 6, Version 1, **Length 1**),
     `s := iscBase32.EncodeToString(raw)`, and asserts `Decode(s)` returns a non-nil error. Building
     from raw bytes ties the test to the header contract, not to a string literal. You may also add the
     `"MAIQAAAAAAAAAAAA"` literal as a second sub-case since the issue/handoff name it explicitly.
  2. as a sanity counterpart (same pattern as the Version test), flip `raw[1]` to `0x10` (Length 0),
     re-encode, and assert `Decode` now succeeds — proving the rejection is specifically the Length
     check, not a codec/length error.
  - Do **not** add this case into `TestDecodeGoldenVectors` (those assert decoded *fields*); the
    rejection belongs with the other reject/version tests.
- **Keep both existing golden vectors passing** — they have byte1 = 0x10 (Length nibble 0), so the
  guard must not affect them. `TestDecodeGoldenVectors` and `TestDecodeRoundTrip` stay green
  (`encode` always writes `raw[1] = versionV1 << 4 = 0x10`, Length nibble 0).
- **Mutation check (do this, don't just claim it).** Temporarily flip the guard's condition
  (`!= 0` → `== 0`, or comment the guard out) and confirm a test FAILS; then restore. A guard with no
  failing test is vacuous — `learnings/index.md` explicitly warns the golden test alone does not
  exercise every header field.
- **Purity / WASM rule (always-loaded learnings).** `internal/index` is a pure leaf shared with the
  future WASM verifier — add no imports beyond what is already present (`encoding/base32`,
  `encoding/binary`, `fmt`, `strings`). Confirm `GOOS=js GOARCH=wasm go build ./internal/index` still
  succeeds.

## Verification
- `mise run check` is green (build + vet + all packages + `gofmt -l .` empty).
- `go test -count=1 ./internal/index` passes (all existing sub-tests + the new Length-nibble case).
- Assertion: `Decode("MAIQAAAAAAAAAAAA")` returns a **non-nil** error (byte1 = 0x11, Length nibble 1).
- Assertion: `Decode("MAIGHFECJMOPMIAB")` still returns `{Realm:0, HubID:1, Timestamp:1751831876325218}`
  with nil error, and `Decode("MEIGHFECJMOPMIAC")` still returns `{Realm:1, HubID:2, ...}` (both
  Length-0 golden vectors unchanged).
- `GOOS=js GOARCH=wasm go build ./internal/index` succeeds (still WASM-shareable; no new imports).
- Mutation proof: flipping/removing the Length guard makes a test in `./internal/index` FAIL (the
  guard is non-vacuous); restored clean before commit.

## Done When
`internal/index.Decode` rejects any nonzero Length nibble with a descriptive error, both golden
vectors still decode, the new test fails if the guard is flipped, and `mise run check` is green.
