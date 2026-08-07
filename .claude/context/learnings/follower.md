<!-- area: internal/follower (follower.go, loop.go, ingest.go, checkConsistency, fsckMirror, hub_keys cache) -->
<!-- indexed-as: follower.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/follower` — PollHub composition & freeze wiring

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Follower composition (`internal/follower`)

- **`PollHub` composes the M1 chain + store CRUD** (`follower.go`):
  `FetchCheckpoint → AcceptCheckpoint → [StatusVerified] FollowState → ingestTiles → checkConsistency →
  freeze | (RecordCheckpoint + AdvanceFollowState + cacheHubKey + fsckMirror)`. Direction is
  follower → {logclient, store}, never the reverse (store stays a leaf, `net/http` never in its closure).
  `PollHub` honors err-before-status (a garbled-body fault returns the wrapped err + persists nothing —
  the `StatusUnverified` zero is meaningless when err != nil); callers must do the same.
- **LOAD-BEARING ORDER — `ingestTiles` MUST run before `checkConsistency` (closed `critical` gap).** The
  equivocation trigger builds its RFC-6962 consistency proof from the LOCAL mirror
  (`ConsistencyProofFromTiles` over `SQLiteFetcher`), so candidate-size tiles must be mirrored first or
  the proof hits a missing-tile error `checkConsistency` swallows as a clean pass → silent advance to the
  inconsistent root. Verify with `grep -n`: `FollowState → ingestTiles → checkConsistency → freeze`. The
  missing-tile swallow stays as a robustness guard for the genuine no-mirror case.
- **Freeze on a true verdict returns `(StatusVerified, nil)` — freezes, never crashes (ADR-0006)** —
  `RecordViolation` + `RecordCheckpoint`(evidence, no advance) + `Freeze`, alert iff `!wasFrozen`. The
  nil-error return is what makes `loop.go`'s back-off work: `Tick` marks `lastPoll[hub]=now` only on a
  nil-error `PollHub`, so a just-frozen hub gets the longer `Frozen` interval. Keep freeze on the
  nil-error path (a non-nil return would leave it unmarked → re-polled every `Normal` tick, no back-off).
- **settled (landed; full detail in git history):** the freeze/fork/shrink-wiring tests
  (`TestPollHubVerifiedAdvances`/`GrowingSplitViewFreezes`/`Fork`/`FrozenCleanRepollIsEvidenceOnly`) are
  all mutation-proven non-vacuous off the SAME `buildVerifiedMirror` fixture (an always/never-freezes
  wiring breaks a complementary case). Durable rules surviving them: (1) the already-frozen evidence-only
  short-circuit `if fs.Frozen { recordVerdict(...,true); return status,nil }` sits AFTER `checkConsistency`
  + the `violated` branch, BEFORE `RecordCheckpoint` — a fresh contradiction on a frozen hub still records
  re-detection evidence; only a clean re-poll short-circuits; `ingestTiles` is NOT skipped (rebuildable
  evidence). (2) Fork re-detection compares against the prior ACCEPTED root via `CheckpointAt`'s explicit
  `ORDER BY rowid LIMIT 1`, never the contradicting evidence row — keep on any prior-root-selection change.
  (3) `due()` uses `>=`; zero `lastPoll` is always due (restart re-polls all, idempotent). (4) `Run` is
  deliberately untested (a 12-line `select` over `ctx.Done()`/`ticker.C`, all logic in injected-`now`
  `Tick`+pure `due()`); verify `time.Now()` never appears in `loop.go`.

## OTS stamp-then-upgrade loop control core (`internal/follower/otsloop.go`)

- **`OTSTick(ctx, st, stamper, up, now, logger)` is the pure injected-`now` analogue of `loop.go`'s
  `Tick`, a SEPARATE driver off the poll path — OTS NEVER blocks/crashes the follower (ADR-0004).** Per
  back-off-filtered `PendingOTS` row: if not-yet-stamped (`stamper != nil && len(r.OTSBytes) == 0`, the
  empty sentinel the poll path writes) → call `Stamper`, persist via `MarkOTSStamped`, `continue` (next
  tick upgrades it); else `Confirmed` → `MarkOTSUpgraded`; declined/errored → `MarkOTSAttempted(Attempts+1,
  now+backoff)`. Any per-row Stamper/Upgrader transport fault OR store-write fault is logged with
  `hub_id`/`tree_size`, folded into `firstErr`, and the pass CONTINUES; `firstErr` is observability-only.
  `Stamper`/`Upgrader` are func seams (not interfaces, YAGNI/matches `AlertFunc`) so `internal/follower`
  imports no anchoring package (verify deps == 0: `internal/otsclient`/`internal/ots`/`nbd-wtf/opentimestamps`).
  Stamping is placed HERE off the poll path, NOT in `stampRoot`/`PollHub` — a synchronous calendar
  round-trip on the poll path violates "OTS never blocks"; `follower.go` stays byte-unchanged.
- **DURABLE TRAP — the production `Stamp` path lacks the panic-recover AND per-request-timeout the upgrade
  path has (open `normal` issue).** When the real `otsclient.Stamp` closure is wired (it now is, via
  `stampFunc()` in main.go), it runs `opentimestamps.Stamp`'s panic-prone `parseCalendarServerResponse`
  parser and `http.DefaultClient` with NO recover and NO deadline — so a malformed calendar response
  CRASHES the monitor and a stalled one HANGS the OTS goroutine. The upgrade path solved exactly this with
  `safeUpgrade`/`recoverRead`; the stamp path needs the symmetric `safeStamp` guard. Any future stamp-path
  touch must route through such a guard (same FFI-boundary rule as `safeUpgrade` — do not strip it).
- **DURABLE TRAP — the nil-Stamper guard order is wrong for empty rows (open `low` issue).** The guard
  `if stamper != nil && len(r.OTSBytes) == 0` falls THROUGH to the Upgrader on a nil Stamper + empty row
  (bogus back-off), contradicting the docstring's "nil skips, leaves empty". The nil-tolerant contract
  only holds when there are no empty rows; a clean fix tests `len(r.OTSBytes) == 0` first and `continue`s
  when `stamper == nil`. Production wires a non-nil Stamper so this is test-only, but keep docstring==code.
- **`MarkOTSStamped` is a status-untouching UPDATE seam (NOT riding `RecordOTS`).** `RecordOTS` is
  `ON CONFLICT DO NOTHING` (cannot update an existing row's bytes); `MarkOTSUpgraded` flips status to
  confirmed; `MarkOTSAttempted` only touches attempts/next_retry. So persisting stamp bytes onto an
  existing pending row genuinely needed a 3rd store mutator that leaves status `pending` (the row must
  stay in `PendingOTS` for the next-tick upgrade). The stamp branch's `continue`-after-stamp (upgrade next
  tick) is deliberate — a fresh-stamped sequence is never Bitcoin-confirmed immediately, so falling
  through would just back it off. Oracle gate correctly N/A (opaque `pending`→`confirmed` over an
  already-fsck-verified root; no signature/RFC-6962/Merkle/did:web/proof code).
- **`backoff(attempts)` is a pure capped-exponential helper** (base 1h, doubling, shift cap 5 = 32h
  pre-clamp, max 24h); `attempts` is the post-increment count so `attempts==1` waits one base,
  non-positive → 1. OTS tests drive the public store seam (seed → `OTSTick` with fake `Stamper`/`Upgrader`
  → `OTSForRoot`/`PendingOTS` read-back), never loop internals.

## Structured logging (`log/slog`) at the loop + binary boundary

- **`log/slog` lives ONLY in `loop.go` (composition) and `main.go` (binary) — never a leaf.** A nil-safe
  `Loop.Logger *slog.Logger` + unexported `logger()` accessor (falls back to `slog.Default()`) keeps every
  bare `&Loop{…}` (the two `loop_test.go` literals + the binary) compiling unchanged. The previously
  `_ =`-discarded per-tick error in `Run` is now an `ErrorContext` emit that still does NOT propagate
  (log-and-continue invariant intact — verified the `_ = l.Tick(ctx, t)` line is genuinely gone, replaced
  by an `if err != nil { logger().ErrorContext }` branch that never `return`s). `go.mod`/`go.sum` +
  all six leaf packages byte-identical; oracle gate correctly N/A (stdlib, no signature/merkle/didweb path).
- **settled:** a faulting single-hub `Tick` emits exactly one ERROR record (probe-confirmed via an
  always-erroring `errFetcher`), so the `len(errorRecs) != 1` assertion is non-vacuous. Durable residue:
  the `"follow state read failed"` site is reachable only via a store fault, and `Loop.Store` is a
  concrete `*store.Store` (not an interface), so it cannot be fault-injected without a wider seam —
  an accepted limitation, NOT dead code.
- **Alert severity is WARN, not ERROR — a deliberate distinction.** A freeze is an operator-actionable,
  evidence-preserved HUB condition; ERROR is reserved for a monitor-process fault. `alertFunc(logger)` in
  `main.go` captures the logger in the closure (injectable, never a global read); the once-per-transition
  gating stays in `PollHub`/`freeze`.

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
- **settled (mutation-proven both ways):** an inverted verdict and an always-firing proof-build skip each
  break the freeze case, so a green-but-wrong always/never-freezes wiring cannot ship. Durable technique:
  **`if x`→`if false` is a compile error** (the condition's variable goes unused) — mutate by INVERTING
  the condition, or by `&& false`, never by replacing it.
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
- **The walk fetches only what can have changed — the mirror is the fetch cache.** Both walks read
  the store's already-mirrored-in-full sets (`store.MirroredFullTiles` / `MirroredFullEntryBundles`,
  filtered on `widthForP(0)`) once per poll and skip a coord when `c.Partial == 0 && alreadyFull`, so
  a poll costs one request per missing coord plus one per partial — never one per coord in the tree.
  **Both halves of that condition are load-bearing.** Dropping `Partial == 0` looks harmless (a
  partial coord is normally absent from a full-width set) but breaks the **shrink** case: after a
  coord completes, a *smaller* observed size enumerates it as a `.p/<W>` partial again, and the full
  row would suppress the fetch of the contradicting hub's own bytes — evidence `ingestTiles` exists to
  capture before `checkConsistency` runs (ADR-0006). Mutation-proven by
  `TestIngestTilesAlwaysFetchesPartials`; the two skips by `TestIngestTilesSkipsMirroredFullCoords` /
  `GrowthFetchesOnlyNewCoords` / `TestPollHubRefetchesCheckpointNotCompletedTiles`.
- **Equivocation detection survives the skip — re-derive this before touching the walk.** On a
  rewriting hub the mirror becomes a MIX (first-observed completed tiles + freshly-fetched newer
  ones), and `checkConsistency` builds its proof over that mix. It still holds because `ingestTiles`
  runs BEFORE `checkConsistency` (unchanged), so every coord the proof needs is present and the
  missing-tile swallow is not newly reachable; a mixed tree cannot reconstruct both the stored prior
  root and the new signed root, so the verdict is still `violated`. For an honest hub the mix is
  byte-identical to the real tree (completed tiles are immutable), so no false freeze. Net effect is a
  STRONGER evidence property: first-observed tile bytes are preserved instead of being overwritten by
  a re-serve. The cost is the loss of the incidental self-heal a full re-fetch gave a corrupt mirror
  row (open `normal` issue).
- **`projectEntryBundle` runs BEFORE `RecordEntryBundle` — the order is the invariant.** The bundle
  row is what makes a coord skippable, so writing it last is what upholds "a full bundle in the mirror
  implies its projection was written". Swapped, a projection fault after a successful bundle write
  leaves a row every later walk skips and a permanent `iscc_index` gap. Mutation-proven by
  `TestIngestEntryBundleProjectionFaultLeavesCoordRefetchable` (an undecodable frame must leave the
  coord UNMIRRORED and re-fetchable).
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
