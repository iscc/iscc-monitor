# Handoff

## 2026-06-20 — Enforce the CID 1.0 key validity window (pure `DIDKey.ValidAt`)

**Done:** Added a pure `func (k DIDKey) ValidAt(now time.Time) bool` to `internal/didweb` that
enforces the parsed CID 1.0 validity window — active in the half-open interval
`[ValidFrom, ValidUntil)`, not revoked at/after `now`, with each zero-valued field meaning "no
constraint" so a zero-value `DIDKey` is always valid (matching `parseTime` and the live-fixture
"currently valid" case). It takes `now` as an argument (deterministic, never reads the wall clock),
returns a bool, and imports only `time` so the WASM build stays green. No logic change to
`ParseDIDDocument` / `parseTime`; only the `DIDKey` doc comment was repointed at `ValidAt`.

**Files changed:**
- `internal/didweb/validity.go` (new): the `DIDKey.ValidAt` predicate, `time`-only.
- `internal/didweb/validity_test.go` (new): table-driven test, 12 cases pinning the half-open
  boundaries at exact instants (at-/just-before-`ValidFrom`, at-/just-before-`ValidUntil`,
  at-/just-before-`Revoked`, all-zero, expired, not-yet-active, revoked, Revoked-overrides-window) +
  a `ValidAt(time.Now())==true` zero-key check.
- `internal/didweb/resolve.go`: `DIDKey` doc comment now says zero = "no constraint" and points at
  `ValidAt` for window evaluation (was "the follower decides enforcement later, not this parser").
  No code change.

**Verification:** `mise run check` → green (build + vet + test, exit 0). Per-criterion:
- [x] `mise run check` green.
- [x] `gofmt -l internal/didweb` prints nothing.
- [x] `go test -run TestValidAt ./internal/didweb` PASS (all 12 subtests + `TestValidAtZeroKeyNow`).
- [x] Zero-value `DIDKey{}.ValidAt(time.Now())` → `true`.
- [x] `ValidUntil=2020-01-01` at 2026-06-20 → `false`; `ValidFrom=2030-01-01` at 2026-06-20 →
  `false`; `Revoked=2020-01-01` at 2026-06-20 → `false`.
- [x] Half-open boundary: `ValidUntil=T` → `ValidAt(T)==false`, `ValidAt(T-1ns)==true`.
  (Also covered symmetrically for `ValidFrom` and `Revoked`.)
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (validity.go stays WASM-pure).

**Next:** Wire the SQLite `hub_keys` cache + the follower's checkpoint-acceptance path
(`ResolveVerifierKey` → `VerifyCheckpoint`), and call `DIDKey.ValidAt(observedAt)` there: a key that
verifies the signature but is out-of-window maps to not-`verified` (rotation/revocation case),
distinct from `ErrUnresolvable → unresolvable` / `ErrUnverified → unverified`. That step should also
refresh the sb1 did.json fixture + `derive_vkey.py` `HUBS` entry to the current key (`069d0f14`) —
still pinned to the `hub_keys`/validity-cache step, not done here. The three-trigger RFC-6962
consistency check is the step after.

**Notes:**
- **Trust-root/oracle gate scope for this step:** N/A by construction. This is a pure arithmetic
  predicate over `time.Time` fields — no signature/vkey derivation, no Merkle code, no SQLiteFetcher,
  no `internal/proof`. The relevant gate (vkey golden parity via `derive_vkey.py` + live fixtures
  under `note.Open`) was satisfied by the prior step and is unaffected; `mise run check` re-runs all
  existing didweb + logclient tests and they stay green.
- **Did not touch `parseTime`'s lenient (unparseable → zero time) behavior**, per next.md "Not In
  Scope". Consequence to note for the parser/fixture step: a malformed (non-empty but unparseable)
  `revoked`/`validUntil` timestamp parses to zero time, which `ValidAt` reads as "no constraint" —
  i.e. fail-open on a malformed timestamp. This is by design here (changing it would alter parsing
  semantics + the existing golden test). next.md says to file it as an issue for the parser/fixture
  step rather than fix it here; flagging for `review` to record in `issues.md` if warranted.
- The `verificationMethod` struct doc in resolve.go still reads "the follower may later enforce"; I
  left it untouched to stay within the single non-test source touch budget (only the `DIDKey` doc was
  in scope). Minor wording drift only; the now-authoritative comment is on the consumer type `DIDKey`.
- Gate integrity: no `//nolint`, `t.Skip`, build-tag exclusion, swallowed error, loosened gate, or
  deleted assertion. Branch: `develop`.
