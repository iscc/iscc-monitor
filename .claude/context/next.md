# Next Work Package

## Step: Carry the resolved did:web context out of `AcceptCheckpoint` and reuse it in `PollHub`

## Goal
Stop a verified poll from fetching the same `did.json` two or three times. Widen
`AcceptCheckpoint` to return the verifier-key context it already resolved (`vkey` + `didweb.DIDKey`)
so the follower's `fsckMirror` and cold-cache `cacheHubKey` paths reuse it instead of re-running
`ResolveVerifierKey`, while preserving ADR-0009's per-poll validity-window check.

## Scope
- **Modify**:
  - `internal/logclient/accept.go` — change `AcceptCheckpoint`'s return to also yield the resolved
    `vkey string` and `didweb.DIDKey` (the per-poll verified key context).
  - `internal/follower/follower.go` — thread that context from the `AcceptCheckpoint` call site into
    `fsckMirror` and `cacheHubKey`/`cacheHubKeyResolve`, dropping their own `ResolveVerifierKey` calls.
- **Modify (tests/docs — not counted against the ≤3 budget)**:
  - `internal/logclient/accept_test.go` — update the `AcceptCheckpoint` call to the wider return and
    assert the returned `vkey`/`DIDKey` is populated on `StatusVerified` and zero/empty otherwise.
  - `internal/follower/follower_test.go` — update `TestPollHubCacheHitSkipsDidFetch`: cold poll now
    fetches `did.json` exactly **1** time (was 3), warm poll **0** more (was 2). Adjust its comment
    and the `want` literals/messages accordingly.
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/accept.go` (current 3-value return, lines 84-105).
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` (`ResolveVerifierKey` returns
    `(string, didweb.DIDKey, error)`, lines 93-116 — this is the context to surface).
  - `/workspace/iscc-monitor/internal/follower/follower.go` (`fsckMirror` lines 375-388,
    `cacheHubKeyResolve` lines 340-357, the `AcceptCheckpoint` call site line 142).
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` (`DIDKey` fields: `Multibase`, `PublicKey`,
    `Revoked` at lines 54-60 — used by `cacheHubKeyResolve` to build `store.HubKey`).
  - `/workspace/iscc-monitor/internal/logclient/keyid.go` (`KeyIDFromVerifier(vkey string)` line 26 —
    still derives the key id from the reused `vkey`, no fetch).

## Not In Scope
- Do NOT change the warm-path fast cache (`cacheHubKeyFast` / `KeyIDFromCheckpoint`) — it already
  avoids a fetch; this step only removes the *cold* `cacheHubKeyResolve` resolve and the `fsckMirror`
  resolve.
- Do NOT touch the size-varying proof-surface ETag/Cache-Control arc, the dashboard, verify-for-me,
  or any other open `normal` issue (tile-writer `p`-vocabulary, deep `AdvanceAccepted`,
  `CheckConsistency` collapse). One issue per step.
- Do NOT change `ResolveVerifierKey`'s own signature or behavior — `AcceptCheckpoint` already calls
  it; just stop discarding its result.

## Implementation Notes
- `AcceptCheckpoint` (accept.go:85) already binds `vkey, key, err := ResolveVerifierKey(...)` and then
  throws `vkey`/`key` away. Surface them. Recommended shape: add an exported result struct, e.g.
  `type VerifiedContext struct { VKey string; Key didweb.DIDKey }` (carrying the per-poll resolved
  key + CID 1.0 validity window), and return `(Status, CheckpointInfo, VerifiedContext, error)`.
  Populate the context **only on `StatusVerified`**; return the zero `VerifiedContext{}` on every
  non-verified verdict (mirroring how `CheckpointInfo` is zero off the verified path — keep that
  invariant symmetric and testable). Update the package docstring lines describing the return tuple.
- The validity check stays exactly where it is: `if !key.ValidAt(observedAt) { return StatusRotated,
  ... }` runs before the verified return, so the per-poll ADR-0009 window check is preserved — the
  reused context is the *same* key that just passed validity. Do not move or weaken it.
- In `follower.go`, the `AcceptCheckpoint` call site (line 142) becomes
  `status, info, vctx, err := logclient.AcceptCheckpoint(...)`. Thread `vctx` down only on the
  verified, non-violation path:
  - `fsckMirror` currently re-resolves to get `vkey` (line 376). Change its signature to accept the
    `vkey` directly (origin is already `info.Origin`, `<domain>/log`; prefer passing `info.Origin`
    over a second `logclient.Origin` derive). Drop the `ResolveVerifierKey` call inside `fsckMirror`.
  - `cacheHubKey` → `cacheHubKeyResolve` (line 340) currently re-resolves to map `DIDKey` into
    `store.HubKey`. Pass `vctx.VKey` + `vctx.Key` in so it uses `KeyIDFromVerifier(vctx.VKey)` and
    `vctx.Key.{PublicKey,Multibase,Revoked}` without re-fetching. `cacheHubKeyFast` is unchanged.
- Correctness rule (learnings.md, ADR-0009): "did:web is the only key source … reuse only within one
  verified poll." The context you reuse is resolved *this poll* and already validity-checked, so
  reuse is in-bounds; do **not** cache it across polls (the `hub_keys` cache, refreshed each poll, is
  the cross-poll story and is untouched here).
- Correctness rule (learnings.md): `AcceptCheckpoint`'s "callers must check `err` before the status,
  and `CheckpointInfo` is the zero value on every non-verified verdict." Keep both invariants and
  extend the second to the new `VerifiedContext` (zero off the verified path).
- Oracle gate: this slice does not change signature/RFC-6962/Merkle/did:web-derivation logic — it
  only stops discarding an already-resolved value. The gate is N/A *by content*, but
  `derive_vkey.py` must still reproduce both golden vectors (`40b74463`/`22b08f3e`) since `vkey` now
  flows further; re-run it and `rm -rf .claude/.scratch` after.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` pass; `gofmt -l .`
  empty).
- `go test -count=1 -run TestAcceptCheckpoint ./internal/logclient` passes, asserting the returned
  `VerifiedContext` is populated (non-empty `VKey`, non-zero `Key.PublicKey`) on `StatusVerified` and
  is the zero value (`VerifiedContext{}`) on each non-verified verdict.
- `go test -count=1 -run TestPollHubCacheHitSkipsDidFetch ./internal/follower` passes with the cold
  poll asserting **1** `did.json` fetch and the warm poll **0** additional fetches.
- `go test -count=1 -run 'TestPollHub' ./internal/follower` passes (freeze/fork/equivocation/
  inclusion/fsck paths all still green with the threaded context).
- `python3 .claude/derive_vkey.py` reproduces `sb0…+40b74463+…` and `sb1…+22b08f3e+…`; scratch dir
  removed afterward.

## Done When
`AcceptCheckpoint` returns the resolved verifier context, `PollHub` reuses it so a cold verified poll
fetches `did.json` exactly once (warm poll zero more), and all Verification criteria pass.
