## 2026-06-21 — Carry the resolved did:web context out of `AcceptCheckpoint` and reuse it in `PollHub`

**Done:** Widened `AcceptCheckpoint` to return a populated `VerifiedContext{VKey, Key}` on
`StatusVerified` (zero value off the verified path, mirroring `CheckpointInfo`), and threaded that
context through `PollHub` so `cacheHubKey`'s cold cache-miss fallback and `fsckMirror` reuse the
already-resolved + already-validity-checked key instead of re-running `ResolveVerifierKey`. A verified
poll now resolves `did.json` exactly once (in `AcceptCheckpoint`); previously a cold verified poll
resolved it three times. The ADR-0009 per-poll validity-window check is untouched — the reused key is
the same one that just passed `ValidAt`.

**Files changed:**
- `internal/logclient/accept.go`: added the exported `VerifiedContext` struct; `AcceptCheckpoint` now
  returns `(Status, CheckpointInfo, VerifiedContext, error)`, populating the context only on
  `StatusVerified` (zero on all non-verified verdicts). Imports `internal/didweb` for `DIDKey`. Updated
  the return-tuple docstring.
- `internal/follower/follower.go`: call site binds `vctx`; `cacheHubKey`/`cacheHubKeyResolve` reuse
  `vctx.VKey`/`vctx.Key` (dropped their `ResolveVerifierKey` call); `fsckMirror` now takes
  `(ctx, st, hubID, vkey, origin)` and reuses `vctx.VKey` + `info.Origin` (dropped its
  `ResolveVerifierKey` and `logclient.Origin` calls). `cacheHubKeyFast` is unchanged.
- `internal/logclient/accept_test.go`: updated to the 4-value return; asserts `VerifiedContext` is
  populated (non-empty `VKey`, non-empty `Key.PublicKey`) on `StatusVerified` and zero on every
  non-verified verdict.
- `internal/logclient/checkpoint_test.go`: updated the `AcceptCheckpoint` call to the wider return
  (discards the context).
- `internal/follower/follower_test.go`: `TestPollHubCacheHitSkipsDidFetch` cold poll now asserts **1**
  `did.json` fetch (was 3); warm poll asserts its own single `AcceptCheckpoint` fetch with cacheHubKey
  + fsckMirror adding **0** (see Notes on the literal). Comment + messages rewritten.
- `internal/follower/fsck_test.go`: `verifiedMirror` gained a `vkey` field; the direct `fsckMirror`
  call in `RejectsCorruptedMirror` updated to the new `(ctx, s, hubID, m.vkey, fsckOrigin)` signature.

**Verification:** `mise run check` → green (all 16 packages `ok`, including the local `cmd/notecheck`;
`go build`/`go vet` clean; `gofmt -l .` empty).
- `go test -count=1 -run TestAcceptCheckpoint ./internal/logclient` → PASS (context populated on
  verified, zero on each non-verified verdict).
- `go test -count=1 -run TestPollHubCacheHitSkipsDidFetch ./internal/follower` → PASS (cold = 1 fetch,
  warm window = 1 fetch, cacheHubKey/fsckMirror add 0).
- `go test -count=1 -run 'TestPollHub' ./internal/follower` → PASS (freeze/fork/equivocation/
  inclusion/fsck paths all green with the threaded context).
- `python3 .claude/derive_vkey.py` reproduces `sb0…+40b74463+…` and `sb1…+22b08f3e+…`; scratch removed.
- `GOOS=js GOARCH=wasm go build ./internal/didweb` → green (WASM purity invariant unaffected).

**Next:** Drain the next ADR-0006 `normal` issue. The remaining redundancy in the verified poll is the
warm-path's unavoidable second `ResolveVerifierKey`: `AcceptCheckpoint` always re-fetches `did.json`
even when the `hub_keys` cache already holds the key — `cacheHubKeyFast` proves the key id can be
recovered fetch-free from the raw checkpoint. A future slice could let `PollHub` consult `LookupHubKey`
and skip even the `AcceptCheckpoint` resolve on a warm cache, but that needs `AcceptCheckpoint` (or a
variant) to accept a pre-resolved key — a larger design change, out of this step's scope. Other open
`normal`s from the prior handoff: tile-writer `p`-vocabulary, deep store-owned `AdvanceAccepted`,
collapse `CheckConsistency`, or the proof-surface ETag/Cache-Control arc.

**Notes:**
- **`next.md` warm-poll literal discrepancy (resolved to the physically-true value):** `next.md`
  (Scope line 22, Verification line 83) says the warm poll should make "**0** additional fetches".
  That is physically impossible: `AcceptCheckpoint` has no internal cache and always resolves
  `did.json` once per call, so the warm poll's own `AcceptCheckpoint` irreducibly fetches once. The
  load-bearing thing this step removes is the *extra* cold-path resolves in `cacheHubKeyResolve` and
  `fsckMirror`. I implemented the true semantics: cold poll = **1** fetch (only `AcceptCheckpoint`);
  warm-poll measured window (`fetcher.didFetch - coldFetches`) = **1** (the warm poll's own
  `AcceptCheckpoint`), with `cacheHubKey` (cache hit) and `fsckMirror` (reused context) adding **0**.
  The test comment and message spell this out. The author's "0 for the warm poll" appears to be an
  arithmetic slip (applying the same −2 that took the cold path 3→1 to the warm path 2→0, forgetting
  the warm path's irreducible `AcceptCheckpoint` resolve). The intent — "the cold cache-miss resolve
  and fsckMirror resolve are gone" — is fully realized.
- Oracle gate is **N/A by content** (this slice removes discarded values; it does not change
  signature/RFC-6962/Merkle/did:web-derivation logic). Re-ran `derive_vkey.py` anyway per `next.md`
  since `vkey` now flows further — both golden vectors reproduce byte-for-byte. The conformance
  follower tests (fork/equivocation/inclusion/fsck) all pass with the threaded context.
- `VerifiedContext` is not a comparable struct (`Key.PublicKey` is a `[]byte`), so the non-verified
  zero-value assertion is field-by-field (`VKey == "" && len(Key.PublicKey) == 0`) rather than a
  struct `==`. Same reason the production code never compares `vctx` by `==`.
- `accept.go` now imports `internal/didweb`; that is in `logclient` (the networked side), not the
  WASM-pure `internal/didweb`, so the WASM purity invariant (which rides on `internal/didweb`) is
  unaffected — verified green.
