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

- **RELOCATED (a90d884): the shrink→fork→equivocation `switch` + proof build + missing-tile swallow
  now live in `logclient.CheckConsistency`** — see the "Self-consistency verdict" section below. The
  bullets here describe the original in-follower wiring; the branch order, guards, and error-discipline
  are byte-identical after the move, so they still document the *logic*, just at its new home.
- **The third trigger lands in `checkConsistency`'s `switch` default (the growing-pair case).** Order
  is shrink (`next<prev`) → fork (`next==prev`) → equivocation (`next>prev`); the default-branch guard
  `if !prevFound || info.TreeSize <= prevSize` is load-bearing, NOT redundant: fork is `prevFound`-guarded,
  so a `next==prev && !prevFound` observation reaches default and must not be misread as a growing pair.
  Proof is built from the LOCAL mirror only — `store.SQLiteFetcher{Store,HubID}.ReadTile` straight into
  `ConsistencyProofFromTiles(ctx, …, prevSize, info.TreeSize)` — never re-hitting the hub. Production
  follower imports stay `{context,fmt,logclient,store,time}`; merkle/testonly/tessera are test-only.
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

- **`cacheHubKey` is wired ONLY on the verified, non-violation `PollHub` path** (after
  `AdvanceFollowState`, mirroring coverage placement), never inside `freeze` and never on a
  non-verified verdict — the fork/shrink/unverified tests assert `countRows(…, "hub_keys")==0`, the
  verified test asserts exactly 1 row with `key_id==0x40b74463` + 32-byte `pubkey_raw` + refresh-in-place.
  A `ResolveVerifierKey` failure here is wrapped (`cache hub key: %w`) and surfaced, never swallowed.
- **`KeyIDFromVerifier` recovers the key id from the vkey STRING, it does not re-derive crypto.**
  `SplitN(vkey, "+", 3)` (n=3 load-bearing: sb0's base64 tail `AaV+ivnly67…` itself has a `+`, so a
  plain `Split` over-splits), require 3 fields, `ParseUint(parts[1], 16, 32)`. The middle `+<hex>+`
  field IS the signed-note keyhash — reviewer independently decoded the sb0 checkpoint sig line
  (`base64→ raw[:4]`) to `40b74463` with a 64-byte sig, matching the golden vector, so the trust-root
  value is confirmed from the fixture, not the author. Oracle gate correctly N/A (string parse, not a
  derivation; `go.mod`/`go.sum`/`schema.sql` byte-identical), but the golden still pins it to
  `VerifierKey`'s `"%s+%08x+%s"` output so it cannot silently diverge.
- **The cache-hit fast path now skips the SECOND did.json fetch (`cacheHubKeyFast`).** On a warm cache
  `cacheHubKey` recovers `(name, keyID)` from raw (`KeyIDFromCheckpoint`), asserts `name ==
  Origin(baseURL)` (`sb0.iscc.id/log`, verified against fixture line 1), hits `LookupHubKey`, and
  `RecordHubKey`-refreshes in place — no `ResolveVerifierKey`. The first verified poll still resolves
  twice (cold cache → miss → `cacheHubKeyResolve` fallback). The `+1` fetch-count assertion is
  non-vacuous: a broken name-guard/lookup would fall through to +2 and fail the test. Fall-through
  cases (key-id miss, name mismatch, cache miss) return `(false, nil)`; genuine faults
  (origin/query/RecordHubKey) wrap a non-nil error and are never swallowed. The remaining FIRST resolve
  (inside `AcceptCheckpoint`, drives the `ValidAt` window check) is the next, larger efficiency slice.
- **The fast path reuses the *cached* `Revoked`/`PubkeyRaw` on a hit, NOT a re-resolve — and that is
  safe.** A same-`key_id` pubkey edit is cryptographically near-impossible (`key_id =
  SHA-256(name||0x0A||0x01||pub)[:4]` → different pubkey ⇒ different key_id ⇒ cache miss ⇒ full
  resolve), and a `revoked_at`/window edit is still caught by `AcceptCheckpoint`'s first resolve every
  poll (which gates `StatusVerified` before `cacheHubKey` ever runs). The `hub_keys` row is an
  identity/availability cache, never the verification authority. A fully cache-only window-honoring
  path would first need a `valid_from`/`valid_until` schema column (explicitly Not In Scope here).
- **`LookupHubKey(ctx, hubID, keyID)` is the read side of the cache and is now landed** — the exact
  column-by-column inverse of `RecordHubKey` (`pubkey_raw`→`[]byte`, `pubkey_z`/`revoked_at`/
  `resolved_at` via `sql.NullString`/`sql.NullInt64`→`""`/zero-time), `HubID`/`KeyID` reconstructed
  from the in-args (never re-scanned), absent row → `(HubKey{}, false, nil)` per `FollowState`/
  `Coverage`. **The `uint32` key id never has to be recovered from the signed `int64` column on read**
  (it comes from the lookup arg), so high-bit ids like `0xdeadbeef` round-trip losslessly —
  independently verified with a throwaway high-bit test (PASS, then removed). `LIMIT 1` (no `ORDER BY`)
  is sound because `RecordHubKey`'s UPDATE-then-INSERT keeps ≤1 row per `(hub_id, key_id)`. Still
  unwired into `PollHub`/verification (deliberate next slice). Oracle gate correctly N/A — pure CRUD,
  `go.mod`/`go.sum`/`schema.sql` byte-identical (`git diff --quiet HEAD~1..HEAD` exit 0).

## Live tile/bundle ingestion writer (`internal/follower/ingest.go`)

- **`ingestTiles` is the first production caller binding the four M2 seams** (`tiles.TileCoords`/
  `BundleCoords` → `logclient.FetchTile`/`FetchEntryBundle` → `store.RecordTile`/`RecordEntryBundle`),
  wired into `PollHub` AFTER `cacheHubKey`, BEFORE the final `recordVerdict` — on the verified,
  non-violation path only (not in `freeze`, not on non-verified verdicts). A fetch/store fault is a
  genuine transport error wrapped `follower.PollHub: hub %d: ingest tiles: %w` and surfaced (NOT a
  violation, NOT a freeze) — the checkpoint is already recorded/advanced above, so the next poll
  re-completes the mirror via the idempotent upsert (ADR-0005/0006). Follower prod imports stay
  `{context, fmt, logclient, metrics, store, tiles, log/slog}`; store stays a leaf (no `net/http`, no
  reverse dep). go.mod/go.sum byte-identical (`tessera/api/layout` already in closure via `tiles`).
- **The `widthForP` p↔width translation is the load-bearing bug surface and is triple-pinned.** The
  follower re-derives the store's unexported one-liner (`p==0 → tiles.TileWidth (256)`, else `int(p)`) —
  store's copy stays private, store package byte-untouched. Reviewer mutation-proved it: breaking the
  `p==0 → 256` mapping (return `int(p)` always) FAILS `TestWidthForP` + `TestIngestTilesWidthMapping`
  (full tile invisible at width 256, *readable* at width 0) + `TestPollHubMirrorsTiles`
  (`SQLiteFetcher.ReadTile(0,0,p0)` round-trip fails). A green-but-wrong width map cannot ship.
- **Both ingestion tests are non-vacuous (mutation-verified).** Neutering `ingestTiles` to a no-op FAILS
  `TestIngestTilesWidthMapping` and `TestPollHubMirrorsTiles` (no mirrored rows, full-tile round-trip
  fails) — so the green is real, not existence-vacuous. Tests assert only on observable store outputs
  (`ReadTileBlob`/`ReadEntryBundleBlob` `found==true` + exact synthetic bytes), never follower internals.
  Tree 300 enumerates exactly 5 coords (tiles `{0,0,full}`,`{0,1,p44}`,`{1,0,p1}` + bundles
  `{0,full}`,`{1,p44}`), independently re-derived against `tiles.TileCoords/BundleCoords` — the table is
  ground truth, and the `len(urls)==5` assert pins the enumeration count.
- **Oracle gate correctly N/A for this slice** — transport + CRUD only, no signature/RFC-6962/Merkle/
  did:web/fsck path; the equivocation branch it un-dormants is already golden-tested and unchanged.
  `derive_vkey.py` still reproduces `40b74463`/`22b08f3e` (the did:web cache path through `PollHub` is
  composed, not modified). Trust root re-arms at the `fsck`-over-`SQLiteFetcher` slice (next), which is
  where the mirrored tiles first face the RFC-6962 root-rebuild oracle.
- **`projectEntryBundle` wires the `iscc_index` fold into `ingestEntryBundles`, right after each
  `RecordEntryBundle`, reusing the already-fetched `raw` (no re-fetch).** `baseSeq = bundleIndex *
  tiles.TileWidth` is the load-bearing math — reviewer independently mutation-proved it two ways
  (reverted): (1) `BundleProjections(raw, 0)` → bundle-1 leaves (seq 256-299) collide onto bundle-0
  seqs (0-43) under `ON CONFLICT(seq) DO UPDATE`, `TestPollHubRecordsProjections` FAILS (read-back
  `[]`/wrong seq); (2) dropping the `RecordProjections` call → read-back empty. `tiles.TileWidth` is an
  untyped const so `uint64 * TileWidth` types cleanly as `uint64`. Store stays a leaf — `logclient.
  Projection → store.ProjectionRecord` is copied field-by-field at the call site (verified `go list`
  shows no `internal/logclient`/`net/http` in store's closure). A malformed/non-JSON record or store
  fault is wrapped `project entry bundle index %d: %w` and aborts the poll before accepted state
  advances (decode/store fault, NOT a self-consistency violation — never freezes, ADR-0008+0006).
- **The verified-path fixture `leafPreimages` had to become valid JSON envelopes (test-only, load-bearing),
  not just an additive test.** Once `ingestEntryBundles` folds every bundle, the old `leaf-%d` plaintext
  is a genuine `BundleProjections` JSON-parse fault on every verified poll → the poll aborts. The fix
  emits `{"$schema":"log-entry","iscc_id":<distinct>,"note":{"$schema":<declSchema>}}` per leaf; because
  `buildVerifiedMirror` rebuilds the tree AND frames the SAME preimages into the bundles, the signed root
  stays self-consistent and fsck still rebuilds it (confirmed: `TestPollHubFsck` green, "Successfully
  fsck'd log with size 300"). Distinct per-leaf `iscc_id` (`ISCC:LEAF%08d`) makes the read-back a clean
  one-seq-per-id lookup; `SeqsForISCCID(leafISCCID(i)) == [i]` because leaf `i` sits in bundle `i/256` at
  local index `i%256`, so `baseSeq + local == i`. `equivocation_test.go` is unaffected (own inline tree,
  never calls `BundleProjections`). `TestIngestTilesWidthMapping`'s `recordingFetcher` likewise had to
  frame a valid one-record bundle for `/tile/entries/` URLs only (hash-tile URLs are `tile/<digit>/`,
  no collision) — the width-mapping assertions themselves are unchanged.

## fsck root-rebuild wired into PollHub (`fsckMirror`) + the real-sb0-fixture retirement

- **Wiring `fsckMirror` into the verified `PollHub` path makes a byte-accurate mirror MANDATORY for
  every completing verified poll — this is why the 6 real-sb0-checkpoint follower tests had to convert
  to the in-process `testonly.Tree` mirror (`buildVerifiedMirror`), and the conversion is a sound
  equivalent, NOT a coverage loss.** The real sb0 log's leaf preimages were never captured (live
  capture is Not In Scope), so its mirror can't rebuild the signed root → fsck would always fail. The
  `testonly.Tree` fixture is byte-accurate to its OWN signed root (the same standard `fsck_test.go` and
  the equivocation tests use). Reviewer independently confirmed real-sb0 signature/key parity is fully
  retained at the correct (verification) layer: `internal/logclient/accept_test.go::TestAcceptCheckpoint/
  "verified"` verifies the real sb0 checkpoint (size 10183, real sig) against the real captured
  `sb0.iscc.id_did.json` key → `StatusVerified`/`Origin=sb0.iscc.id/log`/`TreeSize=10183`; the
  `notecheck` external oracle accepts it (`OK sb0.iscc.id/log`, exit 0); `derive_vkey.py` reproduces
  `40b74463`/`22b08f3e`. The follower tests' job is *composition/wiring*, not re-asserting the raw
  signature — moving the literal `10183`/`0x40b74463` asserts to fixture-relative `m.size`/`m.keyID`
  is architecturally right (net assertions went UP 32:16, not down). The HUMAN REVIEW REQUESTED was
  honest but over-cautious: no plan/ADR deviation, no public-API change, no gate weakening.
- **The required mutation is non-vacuous and reviewer-reproduced:** forcing `fsckMirror` to early-
  `return nil` makes `TestPollHubFsck/RejectsCorruptedMirror` FAIL (corrupted tile no longer caught),
  reverting restores green — so the rebuild genuinely compares the re-derived RFC-6962 root against the
  signed root. The corrupt-mirror subtest must call `fsckMirror` DIRECTLY (not via `PollHub`, whose
  `ingestTiles` re-fetches and overwrites the flipped BLOB first). `RunFsck` is an in-process
  STRUCTURAL self-check (shares the monitor's own `LeafHashes`/RFC-6962 code) — the docstring correctly
  does NOT over-claim it is the independent oracle (`notecheck` is); a non-nil return is a mirror/rebuild
  fault, NOT a self-consistency violation → surfaced to the caller, never `freeze` (ADR-0006).
- **`fsckMirror` placement is correct: AFTER `ingestTiles`, BEFORE the final `recordVerdict`, on the
  verified non-violation path only.** Prod follower imports unchanged (`{context, fmt, logclient,
  metrics, store, tiles, log/slog}`), store stays a leaf, go.mod/go.sum byte-identical (`tessera/fsck`
  already in the closure via `internal/logclient/fsck.go`), didweb WASM build green. Scope was clean:
  exactly 1 production file (`follower.go`) + 4 test files.
- **`buildVerifiedMirror(t, leaves)` is the reusable follower verified-path fixture now** — generates a
  per-run `note.GenerateKey` keypair (so key-id is NEVER a literal; assert `m.keyID`), signs the C2SP
  checkpoint body `"<origin>\n<size>\n<base64(root)>\n"`, advertises the key's `z6Mk` multibase via
  `multibaseFromVKey`+`b58encode` (the exact inverse of `didweb.b58decode`), and serves byte-accurate
  level-0 tiles + framed entry bundles for every `TileCoords`/`BundleCoords` coord (reusing
  `equivNodeHash` for level ≥ 1). `mirrorLeaves = 300` crosses the 256-leaf boundary. `mirrorBundleFetcher`
  routes by `strings.HasSuffix` over a `byPath` map — verified the size-300 tlog-tiles paths have no
  suffix collisions, so it is unambiguous for this fixture (a future colliding-path fixture would want
  exact/longest-suffix matching).
- **Wiring fsck onto the live path SURFACED a pre-existing `store.CheckpointAt` fragility (filed
  `normal`): an unordered `LIMIT 1` means fork re-detection compares against an undefined row when two
  same-size checkpoints exist post-freeze.** This forced `TestPollHubFork` to drive fork *re-detection*
  through `freeze` directly (first detection still via `PollHub`); shrink re-detection via `Tick` stays
  covered by `TestTickFrozenUnaffected` (size-only, immune). The fix (`ORDER BY rowid` / pick the prior
  accepted root explicitly) belongs to the store-touching equivocation/serving slice, not this one.

## Inclusion cross-check over the real follower mirror (`internal/follower/inclusion_test.go`)

- **M2's second Verify bar is closed test-only, and that is the honest scope — verified, not asserted.**
  `VerifyInclusionEvidence` is the *consumer* of a hub-supplied proof; there is no inbound hub-evidence
  transport on the follow path yet (no `FetchInclusionEvidence`; proof-serving is a later M2/M3 slice). A
  `PollHub` step recomputing the monitor's OWN proof and checking it against itself would be circular and
  is forbidden by `next.md`/target.md. The follower already mirrors tiles (`ingestTiles`) + indexes leaves
  (`projectEntryBundle`), so `TestPollHubInclusion` drives a verified `PollHub` over `buildVerifiedMirror(300)`,
  resolves `iscc_id→leafIndex` via the production `SeqsForISCCID` (sampled leaf 5 in bundle 0 + leaf 260
  past the 256-leaf boundary → `[5]`/`[260]`), builds the hub's `IsccLogInclusionProof` from `m.tree`, and
  asserts `VerifyInclusionEvidence(ctx, SQLiteFetcher.ReadTile, ev) == nil`. Zero production lines added,
  scope-clean (1 test file + handoff), no `schema.sql`/`go.mod`/`go.sum` touch — exactly as `next.md` scoped.
- **Oracle gate APPLIES (RFC-6962 inclusion crypto) and is reviewer-mutation-proven NON-VACUOUS over the
  real verified-poll mirror.** Reviewer short-circuited `VerifyInclusionEvidence` to `return nil` before the
  proof compare → BOTH negatives FAIL (`inclusion_test.go:125` wrong-leaf, `:145` corrupted-proof); reverted
  → green, tree clean. The wrong-leaf negative is the sharp one: a *valid* leaf-5 proof re-labelled leaf 6
  still fails because the monitor recomputes leaf 6's distinct proof from the mirror written by `ingestTiles`
  — three independent paths (`m.tree.InclusionProof` prover, `InclusionProofFromTiles` recompute, base64
  round-trip), not a tautology. `notecheck` accepts real sb0 / rejects corrupted (exit 0/1); `derive_vkey.py`
  reproduces `40b74463`/`22b08f3e`; CI `notecheck` parity job present + unchanged. `SQLiteFetcher.ReadTile`'s
  `(ctx, l, i uint64, p uint8)` is assignment-compatible with `logclient.TileFetcher` and passes straight in,
  as the inclusioncheck learnings predicted.
