<!-- area: internal/follower (follower.go, loop.go, ingest.go, checkConsistency, fsckMirror, hub_keys cache) -->
<!-- indexed-as: follower.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/follower` — PollHub composition & freeze wiring

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Follower composition (`internal/follower`)

- **`PollHub` is the first real caller composing the M1 chain + store CRUD** (`follower.go`):
  `FetchCheckpoint → AcceptCheckpoint → (only on StatusVerified) RecordCheckpoint → AdvanceFollowState`.
  Verified independently: `go list -deps ./internal/store` stays a single self-only line (store is a
  leaf), `./internal/follower` pulls in `logclient`+`store`(+transitive `didweb`) — direction
  follower → {logclient, store}, never the reverse, so `net/http` never enters the store closure.
- **The verified-path assertion `FollowState.LastSize == 10183` is non-vacuous** — `10183` is line 2
  of the `testdata/live/sb0.iscc.id_checkpoint` fixture (the signed tree size), so it proves the value
  flowed `info.TreeSize → AdvanceFollowState → persisted cursor`. The complementary non-advancing test
  asserts `== 0` after a mismatching-key `StatusUnverified`, so neither case is vacuously satisfied by
  the fresh-store zero.
- **Garbled-body fault returns `(status, wrapped-err)` where status is `AcceptCheckpoint`'s
  `StatusUnverified` zero** — meaningless when err != nil. `PollHub` honors the err-before-status
  contract (returns the wrapped err and persists nothing); callers of `PollHub` must do the same.
- **Freeze wiring composes the two pure verdicts in `PollHub` via `checkConsistency` + `freeze`
  helpers** (`follower.go`). Order is load-bearing: on `StatusVerified`, read `FollowState`, run
  shrink-then-fork BEFORE `RecordCheckpoint`/`AdvanceFollowState`; on a true verdict
  `RecordViolation` + `RecordCheckpoint`(evidence, no advance) + `Freeze`, then alert iff
  `!wasFrozen`. A violation returns `(StatusVerified, nil)` — freezes, never crashes (ADR-0006).
  The `AlertFunc func(int64,string)` is a func seam (YAGNI, not an interface); the `modernc.org/sqlite`
  blank import is added to the follower *test only* (production imports stay `{context,fmt,logclient,
  store,time}`, store stays a leaf, go.mod/go.sum byte-identical).
- **`ingestTiles` MUST run before `checkConsistency` in `PollHub` — this is the load-bearing order, and
  the closed `critical` gap.** The equivocation trigger builds its RFC-6962 consistency proof from the
  LOCAL mirror (`ConsistencyProofFromTiles` over `SQLiteFetcher`), so the candidate-size tiles must be
  mirrored first or the proof hits a missing-tile error that `checkConsistency` swallows as a clean pass
  → the hub silently advances to the inconsistent root. The fix was a pure reorder (move the single
  `ingestTiles` call up to right after `FollowState`, before `checkConsistency`); the missing-tile
  swallow at `checkConsistency`'s equivocation branch stays as a robustness guard for the genuine
  no-mirror case (a hub that advanced before tiles existed). Verify the order with `grep -n`: `FollowState`
  → `ingestTiles` → `checkConsistency` → `freeze`/`RecordCheckpoint`+`AdvanceFollowState`. An `ingestTiles`
  fault now aborts the poll BEFORE accepted state advances (missing proof tile → error, not clean pass).
- **The growing-split-view freeze test (`TestPollHubGrowingSplitViewFreezes`) is non-vacuous because the
  SAME `buildVerifiedMirror(mirrorLeaves=300)` fixture is polled clean by `TestPollHubVerifiedAdvances`
  (`prevSize==0`, advances, `Frozen==false`).** So an "always freezes" wiring breaks Advances and a "never
  freezes" wiring breaks GrowingSplitView. The freeze case seeds a prior accepted checkpoint at size 5
  whose root is `flipByte(m.tree.HashAt(5))` (the real root with one byte flipped) — the candidate
  checkpoint the fetcher signs is internally valid at 300, so the inconsistency is purely between the
  fabricated prior accepted root and the candidate root = a growing split view against THIS monitor.
- **Fork re-detection on an already-frozen hub is now driven through a real second `PollHub`, and the
  determinism it relies on is `CheckpointAt`'s EXPLICIT `ORDER BY rowid LIMIT 1` (no longer implicit).**
  The freeze path records the contradicting checkpoint as evidence (a higher rowid), so after the first
  detection two rows share the same `tree_size` (lowest-rowid seed/prior accepted root + the new
  contradicting root). On the second `PollHub`, `checkConsistency` runs BEFORE the `fs.Frozen`
  short-circuit and reads the prior root via `CheckpointAt(hubID, m.size)`, which deterministically
  returns the seed root → `CheckFork(seed != new)` re-fires → `freeze(wasFrozen=true)` records a 2nd
  `violations` row (re-detection = evidence; `RecordViolation` has no `ON CONFLICT`) WITHOUT re-alerting.
  `LastSize` stays `m.size` (a frozen hub never advances). Reviewer independently mutation-proved
  `TestPollHubFork` non-vacuous on the production path two ways (both reverted): asserting `alerts == 2`
  FAILS (alert fires once), and removing the second `PollHub` FAILS three assertions (violations count,
  the cumulative `kind="fork"} 2` metric, and reopen-survival). The earlier "no ORDER BY / implicit
  insert-order dependency" note is now obsolete: the ordering is explicit in `checkpoints.go:159` and
  pinned by `TestCheckpointAtDeterministicOnFork`. Any future change to prior-root selection must keep
  "compare against the prior accepted root, never the contradicting evidence row" (ADR-0006).
- **Fork-test non-vacuousness comes from the `kind == "fork"` (not "shrink") assertion at equal size,
  NOT the `sb0FixtureRootB64` guard.** That guard compares raw seed bytes to a base64 *string*, so it
  can never trip (and the seed is 33 bytes — `copy` into `[32]byte` truncates harmlessly). The real
  proof the fork branch fired is: verified observation at size 10183 with the real decoded root vs a
  seeded distinct root, asserting kind `"fork"` distinctly from the size-only shrink path.
- **The poll loop (`loop.go`) is pure cadence over `PollHub` — `due()` is the only testable unit, and
  the back-off works *because a freeze returns `(StatusVerified, nil)`*.** `Tick` marks `lastPoll[hub]
  = now` only on a nil-error `PollHub`, and a freeze is a nil error, so a just-frozen hub *does* get
  its `lastPoll` recorded → the next due decision correctly uses the longer `Frozen` interval. If a
  later change ever made freeze return a non-nil error, the frozen hub would be left unmarked and
  re-polled every `Normal` tick (no back-off) — keep freeze on the nil-error path. `due()` uses `>=`
  (exactly-at-interval is due); zero `lastPoll` is always due (fresh hub polled on tick 1, restart
  re-polls all — harmless, `PollHub` is idempotent on an unchanged checkpoint).
- **The ADR-0006 already-frozen evidence-only short-circuit is `if fs.Frozen { recordVerdict(m,
  hubID, status, true, observedAt); return status, nil }`, placed AFTER `checkConsistency` + the
  `violated` branch and BEFORE `RecordCheckpoint`.** Placement is the whole correctness story: a frozen
  hub that re-serves a *fresh* contradiction still flows through the `violated` branch above (records
  the re-detection as evidence, re-fires the violations counter); the short-circuit catches only the
  *clean* re-poll (no fresh violation) and suppresses `RecordCheckpoint`/`SetCoverage`/
  `AdvanceFollowState`/`cacheHubKey`/`fsckMirror`. `ingestTiles` (run earlier) is deliberately NOT
  skipped — tiles are rebuildable evidence, not accepted state. `frozen=true` maps to the glossary
  `"frozen"` label (not the `StatusVerified` enum), and the return stays `(status, nil)` because the
  signature was valid (freezing is a separate axis). Reviewer mutation-proved non-vacuous: deleting the
  block makes `TestPollHubFrozenCleanRepollIsEvidenceOnly` FAIL on `status="verified"` (the hub would
  re-advance the cursor, coverage, key cache). Oracle gate correctly N/A — pure freeze-decision wiring,
  no signature/RFC-6962/Merkle/did:web/fsck path touched; go.mod/go.sum/schema byte-unchanged.
- **The frozen-clean re-poll test seeds the freeze via `store.Freeze` directly, then re-polls the SAME
  300-leaf `buildVerifiedMirror` at the same size/root.** This is the deterministic way to reach the
  short-circuit: with `prevSize==info.TreeSize==300` and identical root, `CheckShrink`/`CheckFork` are
  both false and the equivocation branch short-circuits on `info.TreeSize <= prevSize`, so
  `violated==false` and `fs.Frozen==true` → the new branch fires. The direct `store.Freeze` seed is the
  clean isolation for THIS test (a clean re-poll, no fresh contradiction); the complementary fresh-
  contradiction-on-a-frozen-hub re-detection path is now exercised end-to-end by `TestPollHubFork`'s
  second `PollHub` (the prior "re-detection fragility" deferral is resolved). The metric assertion uses a
  *fresh* `metrics.New()` on the re-poll only (seed poll passes `m=nil`), so `status="frozen" 1` present
  + `status="verified"` absent is a clean single-verdict assert.
- **`Run` is deliberately untested and that is correct here** — it is a 12-line `select` over
  `ctx.Done()`/`ticker.C` with `defer ticker.Stop()` and one documented `_ = l.Tick(ctx, t)` (a flaky
  hub must not abort the network loop; `Tick` already surfaces the error to its caller, so this is not
  gate-dodging). All branching logic lives in the injected-`now` `Tick` + pure `due()`, both covered;
  the spec forbids wall-clock sleeps so testing `Run` would mean sleeping. The single swallowed error
  is justified inline. Verify `time.Now()` never appears in `loop.go` (the ticker delivers `t` via
  `ticker.C`) — the only wall-clock source is `time.NewTicker(l.Normal)`.

## OTS upgrade-loop control core (`internal/follower/otsloop.go`)

- **`OTSTick(ctx, st, up, now, logger)` is the pure injected-`now` analogue of `loop.go`'s `Tick`, a
  SEPARATE driver off the poll path — OTS NEVER blocks the follower (ADR-0004).** It reads
  `st.PendingOTS(ctx, now)` (back-off-filtered), and per row: `Confirmed` → `MarkOTSUpgraded`+`continue`;
  declined or errored → `MarkOTSAttempted(Attempts+1, now+backoff)`. Mirrors `Tick`'s error discipline
  exactly: a per-row Upgrader transport fault OR store-write fault is logged with `hub_id`/`tree_size`,
  folded into `firstErr`, and the pass CONTINUES (a flaky row never aborts the pass, never freezes a hub).
  `firstErr` is returned for observability only. No wall-clock in `OTSTick`; a `Run`-style ticker wrapper
  is deferred to the wiring sub-step (no `main.go` wiring yet — a no-op loop would be dead code).
- **`Upgrader` is a func seam, not an interface (YAGNI, matches `AlertFunc`)** —
  `func(ctx, store.OTSRecord) (UpgradeResult, error)` returning `{Confirmed, OTSBytes, BTCHeight}`. Keeps
  `internal/follower` import-free of any anchoring package; the real calendar-HTTP client becomes a
  closure of this type + the first `go.mod`/`go.sum` change (the NEXT sub-step). Oracle gate correctly
  N/A this slice (opaque `pending`→`confirmed`/back-off over an already-fsck-verified root; no
  signature/RFC-6962/Merkle/did:web/proof code; the `Upgrader` is injected so no `ots verify` crypto
  runs — that gate first applies at the real-`Upgrader` step).
- **`backoff(attempts)` is a pure capped-exponential helper** (base 1h, doubling, shift cap 5 = 32h
  pre-clamp, max 24h); `attempts` is the post-increment count so `attempts==1` waits one base, non-
  positive → 1. Exact cadence is not safety-critical (OTS best-effort); golden-tabled by `TestOTSBackoff`.
  Tests drive the public store seam (`RecordOTS` seed → `OTSTick` with fake confirming/declining/erroring
  `Upgrader`s → `OTSForRoot`/`PendingOTS` read-back), never loop internals; reviewer reproduced the
  next_retry-filter + no-op-`MarkOTSAttempted` mutations (both reverted) — non-vacuous.

## Structured logging (`log/slog`) at the loop + binary boundary

- **`log/slog` lives ONLY in `loop.go` (composition) and `main.go` (binary) — never a leaf.** A nil-safe
  `Loop.Logger *slog.Logger` + unexported `logger()` accessor (falls back to `slog.Default()`) keeps every
  bare `&Loop{…}` (the two `loop_test.go` literals + the binary) compiling unchanged. The previously
  `_ =`-discarded per-tick error in `Run` is now an `ErrorContext` emit that still does NOT propagate
  (log-and-continue invariant intact — verified the `_ = l.Tick(ctx, t)` line is genuinely gone, replaced
  by an `if err != nil { logger().ErrorContext }` branch that never `return`s). `go.mod`/`go.sum` +
  all six leaf packages byte-identical; oracle gate correctly N/A (stdlib, no signature/merkle/didweb path).
- **A faulting single-hub `Tick` emits EXACTLY one ERROR record — reviewer probed it.** Drove the fault
  through the real outbound-fetch seam (an `errFetcher` whose `Fetch` always errors → `FetchCheckpoint`
  fails → `PollHub` returns non-nil → `Tick` logs at `loop.go:119` `"poll hub failed"` with `hub_id` +
  populated `err`, then folds into `firstErr`). A throwaway record-count probe confirmed `total records: 1`
  with the `err` carrying the full wrapped chain (`follower.PollHub: hub 1: fetch checkpoint …`), so the
  test's `len(errorRecs) != 1` assertion is non-vacuous and the "swallowed error is now observable" claim
  is real, not asserted. The second log site (`"follow state read failed"` at `loop.go:109`) is reachable
  only via a store fault; `Loop.Store` is a concrete `*store.Store` (not an interface), so it can't be
  cleanly fault-injected without a wider seam — accepted limitation, documented in the handoff, NOT dead code.
- **Alert severity is WARN, not ERROR — a deliberate, documented distinction.** `alertFunc(logger)` in
  `main.go` returns a `follower.AlertFunc` closure emitting `logger.Warn("hub frozen", "hub_id", …, "kind",
  …)`; a freeze is an operator-actionable, evidence-preserved hub condition (distinct from a monitor-process
  fault, which is ERROR). The once-per-transition gating still lives in `PollHub`/`freeze` (untouched), the
  `AlertFunc func(int64,string)` signature is unchanged, and the old `func alert(…) { fmt.Fprintf(os.Stderr …) }`
  is fully removed. Logger is captured in the closure (injectable/testable), not read from a global —
  though `run()` also calls `slog.SetDefault(logger)` so any future leaf-free call site inherits it.

## Equivocation trigger wiring (`internal/follower/checkConsistency`)

- **settled (RELOCATED a90d884; full detail in git history):** the shrink→fork→equivocation `switch` +
  proof build + missing-tile swallow now live in `logclient.CheckConsistency`. Branch order is shrink
  (`next<prev`) → fork (`next==prev`) → equivocation (`next>prev`); the default-branch guard
  `if !prevFound || info.TreeSize <= prevSize` is load-bearing (a `next==prev && !prevFound` observation
  must not be misread as a growing pair). Proof is built from the LOCAL mirror only
  (`SQLiteFetcher.ReadTile` → `ConsistencyProofFromTiles`), never re-hitting the hub. The durable
  error-discipline + p↔width traps below still apply at its new home.
- **Error-vs-violation discipline is the load-bearing subtlety and is correct.** A proof-BUILD error
  from `ConsistencyProofFromTiles` (most often a missing tile = wrapped `os.ErrNotExist`, since prod
  doesn't mirror tiles until M2) is swallowed narrowly → branch skipped, no freeze (false-positive
  guard, ADR-0006). `CheckEquivocation`'s own (unreachable-by-type) `err` is propagated, not swallowed;
  a genuine `st` fault still surfaces via `CheckpointAt` above. The branch is **dormant in production**
  until M2 writes tiles — today the live path always hits the missing-tile skip.
- **Reviewer mutation-proved non-vacuousness three ways (throwaway copy, reverted):** (1) `if eq`→`if !eq`
  inverts the verdict → freeze case fails; (2) forcing the proof-build-error skip to always fire → freeze
  case fails, proving the happy/freeze cases genuinely build the proof over seeded tiles (NOT the
  missing-tile skip); (3) `if eq`→`if false` is a compile error (unused `eq`) — use an inverting mutation
  instead. So a green-but-wrong always/never-freezes wiring cannot ship. Oracle gate APPLIES (RFC-6962)
  and is satisfied by the in-test merkle ground truth + the unchanged `derive_vkey.py` vectors.
- **The p↔width seam is exercised end-to-end here:** test seeds the full tile (index 0, width 256) and
  the 44-leaf partial (index 1, width 44); the proof builder requests `p=0`→256 and `p=44`→44 over the
  SQLiteFetcher, so the equivocation proof crosses the 256-leaf tile boundary against real mirror reads,
  not a hand-rolled fetcher. Test seam: calls unexported `checkConsistency` + `freeze` directly (no
  `StatusVerified` fixture whose signature encodes the synthesized root exists), asserting only on
  observable store outputs (`violations`/`follow_state`/alert count), never follower internals.

## hub_keys cache wiring (`internal/follower` + `internal/logclient/keyid.go`)

- **settled (landed; full detail in git history pre-2026-06-22):** `cacheHubKey` is wired ONLY on the
  verified non-violation `PollHub` path (after `AdvanceFollowState`); fork/shrink/unverified tests assert
  `hub_keys`==0 rows, verified asserts 1 row `key_id==0x40b74463`. Cold cache resolves twice; the warm
  fast path (`cacheHubKeyFast` via `LookupHubKey`) skips the second did.json fetch (`+1` fetch assertion).
  `LookupHubKey` is the column-by-column inverse of `RecordHubKey`, absent → `(HubKey{},false,nil)`, key
  id round-trips losslessly from the in-arg (never re-scanned from the signed int64 column). Oracle N/A.
  Two durable traps below.
- **`KeyIDFromVerifier` recovers the key id from the vkey STRING — `SplitN(vkey, "+", 3)`, n=3 is
  load-bearing:** sb0's base64 tail `AaV+ivnly67…` itself contains a `+`, so a plain `Split` over-splits.
  Require 3 fields, `ParseUint(parts[1], 16, 32)`; the middle `+<hex>+` field IS the signed-note keyhash.
  It is a string parse, NOT a crypto re-derivation (oracle N/A), but the golden pins it to `VerifierKey`'s
  `"%s+%08x+%s"` output so it cannot silently diverge.
- **The `hub_keys` row is an identity/availability cache, NEVER the verification authority.** The fast
  path reuses the cached `Revoked`/`PubkeyRaw` safely: a same-`key_id` pubkey edit is cryptographically
  near-impossible (different pubkey ⇒ different `key_id` ⇒ cache miss ⇒ full resolve), and a
  `revoked_at`/window edit is still caught by `AcceptCheckpoint`'s first resolve every poll (gates
  `StatusVerified` before `cacheHubKey` runs). A cache-only window-honoring path would need a
  `valid_from`/`valid_until` schema column (Not In Scope).

## Live tile/bundle ingestion writer (`internal/follower/ingest.go`)

- **settled (landed; full detail in git history):** `ingestTiles` binds the four M2 seams
  (`tiles.TileCoords`/`BundleCoords` → `logclient.FetchTile`/`FetchEntryBundle` →
  `store.RecordTile`/`RecordEntryBundle`) on the verified non-violation path AFTER `cacheHubKey`; a
  fetch/store fault is wrapped `ingest tiles: %w` and surfaced (NOT a violation/freeze), so the next poll
  re-completes the mirror via idempotent upsert. `projectEntryBundle` folds `iscc_index` after each
  `RecordEntryBundle` (reuses `raw`); store stays a leaf (`logclient.Projection → store.ProjectionRecord`
  copied field-by-field). Two durable traps below.
- **The `widthForP` p↔width translation is the load-bearing bug surface (triple-pinned).** The follower
  re-derives store's unexported `p==0 → tiles.TileWidth (256)`, else `int(p)` — store's copy stays
  private. Mutation-proved: breaking `p==0 → 256` FAILS `TestWidthForP`/`TestIngestTilesWidthMapping`
  (full tile invisible at width 256, readable at width 0)/`TestPollHubMirrorsTiles`.
- **`baseSeq = bundleIndex * tiles.TileWidth` is load-bearing.** Mutation-proved (reverted)
  `BundleProjections(raw, 0)` collides bundle-1 leaves (seq 256-299) onto bundle-0 seqs (0-43) under
  `ON CONFLICT(seq) DO UPDATE` → `TestPollHubRecordsProjections` FAILS. A malformed-record/store fault is
  wrapped `project entry bundle index %d: %w` and aborts the poll BEFORE accepted state advances
  (decode/store fault, NOT a self-consistency violation — never freezes, ADR-0008+0006).

## fsck root-rebuild wired into PollHub (`fsckMirror`) + inclusion cross-check

- **settled (landed; full detail in git history):** `fsckMirror` runs on the verified non-violation path
  AFTER `ingestTiles`, BEFORE `recordVerdict`, re-deriving the RFC-6962 root from the mirror and comparing
  it to the signed root; a mismatch is a mirror/rebuild fault surfaced to the caller, NEVER a freeze
  (ADR-0006). The 6 real-sb0 follower tests converted to the in-process `buildVerifiedMirror` fixture
  (byte-accurate to its OWN signed root, per-run `note.GenerateKey` so key-id is never a literal — assert
  `m.keyID`); real-sb0 signature/key parity stays covered at the verification layer
  (`logclient/accept_test.go` + `notecheck` + `derive_vkey.py`). `RunFsck` is an in-process STRUCTURAL
  self-check (the independent oracle is `notecheck`). M2's inclusion cross-check (`TestPollHubInclusion`)
  is closed test-only: build the hub's `IsccLogInclusionProof` from `m.tree`, assert
  `VerifyInclusionEvidence(SQLiteFetcher.ReadTile, ev) == nil`. Both mutation-proven non-vacuous (early
  `return nil` FAILS the corrupt-mirror / wrong-leaf negatives, reverted).
- **Durable trap — corrupt-mirror tests must call `fsckMirror` DIRECTLY, not via `PollHub`**, whose
  `ingestTiles` re-fetches and overwrites the flipped BLOB first.
- **Durable trap — `store.CheckpointAt` fork re-detection** must compare against the prior ACCEPTED root,
  not the contradicting evidence row; the explicit `ORDER BY rowid LIMIT 1` (checkpoints.go:159) is what
  makes a post-freeze two-same-size-rows lookup deterministic. Keep it on any prior-root-selection change.
