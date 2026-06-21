## 2026-06-21 — Review of: Carry the resolved did:web context out of `AcceptCheckpoint` and reuse it in `PollHub`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `AcceptCheckpoint` now returns `(Status, CheckpointInfo, VerifiedContext, error)`,
populating the `VerifiedContext{VKey, Key}` only on `StatusVerified` (zero on every non-verified
verdict, mirroring `CheckpointInfo`), and `PollHub` threads it into `cacheHubKeyResolve` and
`fsckMirror` so those drop their own `ResolveVerifierKey` calls. A cold verified poll now resolves
`did.json` once instead of three times; the ADR-0009 per-poll validity window is untouched (the reused
key is the same one that just passed `ValidAt`). Scope-clean (2 prod files, exactly `next.md`'s
Modify set), gates green, trust-root oracle reproduces both golden vectors.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 15 packages `ok` (incl. local `cmd/notecheck`).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run TestAcceptCheckpoint ./internal/logclient` — PASS (context populated on verified;
  zero `VKey`/empty `Key.PublicKey` asserted on each of the 5 non-verified verdicts).
- [x] `go test -run TestPollHubCacheHitSkipsDidFetch ./internal/follower` — PASS (cold = 1 fetch,
  warm window = +1). See note on the `next.md` "0" literal.
- [x] `go test -run 'TestPollHub' ./internal/follower` — PASS (freeze/fork/equivocation/inclusion/fsck
  paths green with the threaded context).
- [x] `python3 .claude/derive_vkey.py` — both golden vectors reproduce byte-for-byte
  (`sb0…+40b74463+…`, `sb1…+22b08f3e+…`); `.claude/.scratch` removed after.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — green (WASM purity invariant unaffected;
  `accept.go`'s new `internal/didweb` import is the WASM-pure parser, not the networked resolver).
- [x] Gate-integrity scan over unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed-error/
  deleted-gate. The 3 "removed" `t.Errorf` lines are `want`-literal/message updates (3→1 cold, 2→1
  warm) and the assertions are net *strengthened* (cold count tightened, 2 new `VerifiedContext`
  assertions added).

**Issues found:** (none blocking). One documented deviation — see Notes.

**Next:** Drain the next ADR-0006 `normal`. Candidates from `issues.md`: collapse the self-consistency
policy into a deep `logclient.CheckConsistency(prior, next, fetcher) → (violated, kind, err)` (closes
the follower/logclient split, table-testable in one place); OR the store-owned `AdvanceAccepted`
single-transaction write; OR the tile-writer `p`-vocabulary unification (deletes the follower's
`widthForP` copy). The warm-path's irreducible second `did.json` resolve (`AcceptCheckpoint` always
re-fetches even on a warm `hub_keys` cache) is a larger design change (needs a pre-resolved-key
`AcceptCheckpoint` variant) — out of one-issue scope.

**Notes:**
- **`next.md`'s "warm poll 0 additional fetches" (lines 21, 83) is physically impossible, and the
  advance author correctly shipped the true value (warm window = 1).** `AcceptCheckpoint` has no
  internal cache and unconditionally calls `ResolveVerifierKey` (accept.go:108); the `cacheHubKeyFast`
  cache-hit is consulted only *after* `AcceptCheckpoint` returns. So every verified poll irreducibly
  fetches `did.json` exactly once. The load-bearing win — removing the *extra* cold resolves in
  `cacheHubKeyResolve` + `fsckMirror` (3→1) — is fully realized and pinned by the test. This is a
  legitimate physically-driven deviation, not a gate dodge; verdict is PASS_WITH_NOTES (not PASS)
  purely to flag that the shipped literal differs from `next.md`'s by physical necessity.
- **Oracle gate is N/A by content but re-run anyway.** This slice stops discarding an already-resolved
  value; it changes no signature/RFC-6962/Merkle/did:web-derivation logic. `derive_vkey.py` reproduces
  both vectors and the conformance follower tests (fork/equivocation/inclusion/fsck) pass with the
  threaded context, so the trust root is unaffected. (`notecheck` runs as part of `mise run check`'s
  `cmd/notecheck` package — green.)
- **`StatusRotated` correctly does NOT leak the resolved context** (accept.go:124-125 returns the zero
  `VerifiedContext` even though it resolved a key), so a rotated/revoked key never reaches the
  follower's cache/fsck reuse — only `StatusVerified` does. `TestAcceptCheckpoint` pins this.
- Production has **no** `ResolveVerifierKey` caller left in `follower.go` (verified by grep); the only
  references are doc comments + the definition/composition sites in `logclient`.
