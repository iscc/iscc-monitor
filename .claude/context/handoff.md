# Handoff

## 2026-06-20 — Pure checkpoint-acceptance decision (`AcceptCheckpoint`) + close the `parseTime` fail-open gap

**Done:** Added `internal/logclient/accept.go` — a pure `Status` enum
(`StatusVerified`/`StatusUnverified`/`StatusUnresolvable`/`StatusRotated`, with `String()`) and
`AcceptCheckpoint`, which composes `ResolveVerifierKey → VerifyCheckpoint → DIDKey.ValidAt(observedAt)`
into the four-way ADR-0009 hub-status verdict (zero `CheckpointInfo` on every non-verified verdict).
Made `internal/didweb`'s `parseTime`/`ParseDIDDocument` reject a non-empty-but-unparseable
`validFrom`/`validUntil`/`revoked` (fail-closed), which now collapses to `StatusUnresolvable` via the
existing `ErrUnresolvable` mapping — closing the `parseTime` fail-open issue.

**Files changed:**
- `internal/logclient/accept.go` (new): `Status` enum + `String()`, `CheckpointInfo{Origin, TreeSize,
  Root}`, and the pure `AcceptCheckpoint(ctx, fetcher, baseURL, raw, observedAt)` decision. Imports only
  `context`/`errors`/`time` + the sibling primitives — no `database/sql`/`sqlite`, no clock read.
- `internal/logclient/accept_test.go` (new): table test covering all four statuses through a fake
  `Fetcher` (sb0 fixture + inline-built did.json for the rotated/malformed cases) against the live sb0
  checkpoint; asserts `StatusVerified` carries `TreeSize == 10183` and that non-verified verdicts return
  the zero `CheckpointInfo`.
- `internal/didweb/resolve.go`: `parseTime` now returns `(time.Time, error)` — empty → `(zero, nil)`
  (unchanged), non-empty + parse failure → wrapped error; `ParseDIDDocument` propagates it (wrapped, like
  its other failures). Doc comments updated to the fail-closed semantics.
- `internal/didweb/resolve_test.go`: two new `TestParseDIDDocumentErrors` subcases asserting a malformed
  `revoked` and a malformed `validUntil` each make `ParseDIDDocument` return a non-nil error.

**Verification:** `mise run check` → green on a cleared test cache (`go build`/`go vet`/`go test` all
ok); `gofmt -l .` empty. Per-criterion:
- [x] `go test -run TestAcceptCheckpoint ./internal/logclient` — PASS (verified, unverified, two
  unresolvable, two rotated subcases).
- [x] `go test -run TestParseDIDDocument ./internal/didweb` — PASS (existing goldens + the two new
  malformed-timestamp subcases).
- [x] Assertion: sb0 fixture + sb0 checkpoint + `observedAt=now` → `StatusVerified`,
  `CheckpointInfo.TreeSize == 10183` (sb0 checkpoint body line is `10183`).
- [x] Assertion: did.json with `"revoked":"not-a-date"` → `StatusUnresolvable` (NOT `StatusVerified`) —
  the closed fail-open issue.
- [x] Assertion: sb0 signer key with `validUntil` (and, separately, `revoked`) before `observedAt` →
  `StatusRotated`, distinct from unverified/unresolvable.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (parser fix imports only stdlib `time`/`fmt`).
- [x] Trust-root oracle unaffected: `python3 .claude/derive_vkey.py` still prints both golden vectors
  byte-exact (`sb0…+40b74463+…`, `sb1…+22b08f3e+…`); `.claude/.scratch/` removed afterward.

**Next:** Wire the SQLite `hub_keys` cache + the stateful follower's poll loop that calls
`AcceptCheckpoint` and persists the verdict — only `StatusVerified` advances accepted state; the other
three are recorded findings while mirroring continues. That step also refreshes the sb1 did.json fixture
+ `derive_vkey.py` `HUBS` to the current key (`069d0f14`). The three-trigger RFC-6962 consistency check
(fork/shrink/equivocation via `transparency-dev/merkle`) is the step after, once tiles land in
`testdata/live/`.

**Notes:**
- `AcceptCheckpoint`'s contract: a non-nil `error` is reserved for a genuine fault — a *verified-but-
  garbled body* (`VerifyCheckpoint` returns a non-`ErrUnverified` parse error). It is returned alongside
  `StatusUnverified`'s zero value + zero `CheckpointInfo`, so **callers must check `err` before the
  status**. This keeps "signature didn't match" (a status) separable from "body was garbage" (an error),
  mirroring `VerifyCheckpoint`. The follower step should decide how to surface that fault (it is neither a
  clean four-way verdict nor a resolution failure).
- Verified the `unverified` test path is real, not a false `unresolvable`: the mismatching key
  (`sb1Multibase`, sb1's prior key) resolves to a valid 32-byte vkey (keyhash `37ea3347`) and fails only
  at the signature check — confirmed with a throwaway test, since removed.
- Validity ordering is load-bearing and implemented per next.md: `ValidAt(observedAt)` is checked **only
  after** a good signature, so an out-of-window key whose signature also fails is `unverified`, not
  `rotated`.
- `issues.md` still records the `parseTime` fail-open `normal` issue; it is now closed by this step
  (review may delete it). The `parseTime` change altered only the validity-timestamp branch, not key
  derivation, so `DIDKey.ValidAt`/`validity_test.go` and the golden vectors are untouched.
- M1 remains only partially met: the four pure primitives (did:web chain, signed-note verify, validity
  predicate, and now the composed acceptance decision) are done and tested, but there is still no SQLite
  store, no follower loop, no binary entrypoint, no consistency check, no coverage/metrics. Loop stays
  CONTINUE.
