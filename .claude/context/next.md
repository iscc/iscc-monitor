# Next Work Package

## Step: Enforce the CID 1.0 key validity window (pure `DIDKey.ValidAt`)

## Goal
Turn the parsed-but-unenforced CID 1.0 validity fields on `DIDKey`
(`ValidFrom`/`ValidUntil`/`Revoked`) into a pure, golden-tested predicate that decides whether a
resolved key is valid at a given observation time. This is the precondition the follower's
checkpoint-acceptance path needs before it can trust a `verified` verdict, and it closes the
documented "parses-but-does-not-enforce" gap — without yet pulling in SQLite or the follower.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/didweb/validity.go` (a single pure method on `DIDKey`)
- **Create**: `/workspace/iscc-monitor/internal/didweb/validity_test.go` (table-driven test)
- **Modify**: `/workspace/iscc-monitor/internal/didweb/resolve.go` — only the `DIDKey` doc comment
  that today says the follower "decides enforcement later, not this parser", to point at the new
  `ValidAt`. No logic change to `ParseDIDDocument` or `parseTime`. (Counts as the single non-test
  source touch besides the new file; total ≤3.)
- **Reference**:
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` (the `DIDKey` struct + `parseTime` it builds
    on; note `parseTime` maps absent/unparseable timestamps to the zero `time.Time`)
  - `/workspace/iscc-monitor/.claude/adr/0009-didweb-trust-root.md` (validity windows are the domain
    owner's rotation/revocation mechanism — `verificationMethod` + `revoked` validity windows; a
    signature outside the window must not be `verified`)
  - `/workspace/iscc-monitor/internal/didweb/resolve_test.go` (existing assertion that live fixtures
    have zero-valued validity fields = "currently valid" — the new predicate must agree)

## Not In Scope
- **No SQLite, no `hub_keys` cache, no follower acceptance path** — those are the next step. This
  step only adds the pure predicate that step will call.
- **No status mapping** (`unresolvable`/`unverified`/`verified`) and no wiring into `VerifyCheckpoint`
  or `ResolveVerifierKey`. `ValidAt` returns a bool; the caller decides what an out-of-window key
  means for hub status.
- **No fixture refresh** (`sb1.amlet.id_did.json` / `derive_vkey.py` to `069d0f14`) — handoff pins
  that to the `hub_keys`/validity-cache step, not here.
- **Do not change `parseTime`'s lenient behavior** (unparseable → zero time). Changing it would alter
  parsing semantics and the existing golden test; if fail-open on a *malformed* (non-empty) timestamp
  is a concern, file it as an issue for the parser/fixture step rather than fixing it here.

## Implementation Notes
- Add a method on the value type: `func (k DIDKey) ValidAt(now time.Time) bool` — pure, takes the
  observation time as an argument so it never reads the wall clock itself (deterministic, testable).
  Returning a bool keeps it minimal; the follower owns the status decision. (If a reason string is
  wanted later, add it then — YAGNI now.)
- Semantics (CID 1.0 / ADR-0009 wall-clock window; the zero `time.Time` means "no constraint",
  matching `parseTime` and the existing `resolve_test.go` assertion that live docs are "currently
  valid"):
  - `ValidFrom` non-zero and `now.Before(k.ValidFrom)` → not valid (key not yet active).
  - `ValidUntil` non-zero and `!now.Before(k.ValidUntil)` → not valid (half-open `[from, until)`;
    `now == ValidUntil` is expired).
  - `Revoked` non-zero and `!now.Before(k.Revoked)` → not valid (revoked at/after that instant;
    half-open the same way, so `now == Revoked` is already revoked).
  - All-zero validity fields (the live testnet case) → always valid.
- Guard each field with `!field.IsZero()` before comparing, so an absent field never constrains.
  Compare with `time.Time.Before` only (avoid `==`/`After` on `time.Time` — monotonic-clock and
  half-open-boundary pitfalls). Pin the boundary instants explicitly in tests.
- Keep the file pure: import only `time`. No `net`/`os`/`database/sql`. This is the same WASM-purity
  rule as the rest of `internal/didweb` (Correctness rule: `proof/verify`/`didweb` stay import-clean
  so the WASM build does not break). Verify with the `GOOS=js GOARCH=wasm` build below.
- Start the new file with a docstring explaining it enforces the CID 1.0 validity window parsed by
  `ParseDIDDocument`, consumed later by the follower's checkpoint-acceptance path. Short, pure
  function, evergreen docstring.
- Relevant Correctness rule (learnings.md): "**did:web is the only key source (ADR-0009).** … A
  signature matching no listed key → `unverified`." A key that verifies a signature but is *outside
  its validity window* is the rotation/revocation case this predicate gates — the follower will treat
  an out-of-window key as not-`verified`, exactly as a key rotation/revocation should.
- Do NOT use `t.Skip`, `//nolint`, build tags, or swallow errors to pass the gate (target quality
  bar; learnings "Never weaken a gate").

## Verification
- `mise run check` is green (`go build ./...` + `go vet ./...` + `go test ./...` all exit 0).
- `gofmt -l /workspace/iscc-monitor/internal/didweb` prints nothing.
- `go test -run TestValidAt ./internal/didweb` passes (all subtests).
- Assertion: a zero-value `DIDKey{}` (no validity fields) `.ValidAt(time.Now())` returns `true`
  (matches the live-fixture "currently valid" case in `resolve_test.go`).
- Assertion: with `ValidUntil = 2020-01-01T00:00:00Z`, `.ValidAt(2026-06-20T00:00:00Z)` returns
  `false` (expired); with `ValidFrom = 2030-01-01T00:00:00Z`, `.ValidAt(2026-06-20T00:00:00Z)`
  returns `false` (not yet active); with `Revoked = 2020-01-01T00:00:00Z`,
  `.ValidAt(2026-06-20T00:00:00Z)` returns `false`.
- Assertion (half-open boundary): for `ValidUntil = T`, `.ValidAt(T)` returns `false` and
  `.ValidAt(T.Add(-time.Nanosecond))` returns `true`.
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (validity.go stays WASM-pure).

## Done When
`internal/didweb` exposes a pure `DIDKey.ValidAt(now)` predicate enforcing the CID 1.0
`ValidFrom`/`ValidUntil`/`Revoked` window with half-open boundaries and zero = "no constraint", all
Verification criteria pass, and the WASM build stays green.
