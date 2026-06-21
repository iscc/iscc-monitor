## 2026-06-21 — Wire RunFsck into PollHub over the live SQLiteFetcher mirror

**Done:** Added `fsckMirror` (`internal/follower/follower.go`) — the first production caller of
`logclient.RunFsck` — and wired it into `PollHub` on the verified, non-violation path AFTER
`ingestTiles` and BEFORE the final `recordVerdict`. After the mirror is populated, it resolves the
vkey (`ResolveVerifierKey`, DIDKey discarded) + origin (`Origin`), builds a `store.SQLiteFetcher`, and
runs `RunFsck`, which rebuilds the RFC-6962 root from the local tiles/bundles and cross-checks it
against the signed checkpoint root. A rebuild mismatch / mirror fault is surfaced as a genuine fault
(NOT a violation, no freeze — ADR-0006).

**Files changed:**
- `internal/follower/follower.go`: added `fsckMirror` helper; called it from `PollHub` (verified,
  non-violation path, after `ingestTiles`); extended the package + call-site comments with the
  error-vs-violation discipline and the "in-process structural self-check, not the notecheck oracle"
  honesty. **(1 production file.)**
- `internal/follower/fsck_test.go` (NEW): `TestPollHubFsck` drives `PollHub` end-to-end over a
  byte-accurate in-process `testonly.Tree` mirror — `RebuildsSignedRoot` returns `(StatusVerified,
  nil)`; `RejectsCorruptedMirror` flips a mirrored tile BLOB and calls `fsckMirror` directly (PollHub's
  `ingestTiles` would overwrite the corruption), asserting non-nil. Also holds the shared
  `buildVerifiedMirror(t, leaves)` fixture (signed checkpoint + did.json advertising the generated
  key's `z6Mk` multibase + byte-accurate tiles/bundles for every coord) + `b58encode` /
  `multibaseFromVKey` helpers.
- `internal/follower/follower_test.go`, `loop_test.go`, `ingest_test.go`: **converted the 6
  verified-completion tests** off the real-sb0 checkpoint fixture onto `buildVerifiedMirror` (see
  HUMAN REVIEW below).

**Verification:** `mise run check` → green (11/11 `ok`, `go build`/`go vet`/`go test` all pass), `gofmt
-l .` empty.
- [x] `go test -run TestPollHubFsck` — good-mirror nil + corrupted-mirror non-nil, both pass.
- [x] `go test -run 'TestPollHub|TestIngest|TestWidthForP|TestEquivocation' ./internal/follower` — pass
  (after the fixture conversion below).
- [x] `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change; `tessera/fsck` already in
  the closure via `internal/logclient/fsck.go`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0.
- [x] Production follower imports unchanged: `{context, fmt, logclient, metrics, store, tiles,
  log/slog}` — no new prod import; store stays a leaf (no `net/http`, no reverse dep).
- [x] **Mutation (reverted):** forcing `fsckMirror` to `return nil` makes `TestPollHubFsck/
  RejectsCorruptedMirror` FAIL — the rebuild genuinely compares against the signed root, non-vacuous.
- [x] Oracle parity intact: `derive_vkey.py` reproduces both golden vectors (`40b74463` / `22b08f3e`);
  `internal/logclient` + `internal/didweb` + `cmd/notecheck` re-run uncached → `ok` (no crypto touched,
  this wiring only composes `ResolveVerifierKey`/`RunFsck`). `.claude/.scratch/` cleaned.
- [x] Determinism: full follower suite green 3×; tile-path suffixes for size 300 are non-colliding.

**Next:** The **inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof` — the SECOND
half of M2's Verify bar (the sibling slice). `InclusionProofFromTiles` already exists as an unwired
seam; it needs captured `IsccLogInclusionProof` fixtures (real or in-process). Also consider the
efficiency + fragility items in Notes.

**Notes:**
- **HUMAN REVIEW REQUESTED — design deviation from `next.md`'s "existing tests unaffected"
  assumption.** Wiring `fsckMirror` unconditionally into the verified `PollHub` path means **every**
  verified, non-violation `PollHub` that runs to completion now requires a *byte-accurate* mirror that
  rebuilds the signed root. The 6 pre-existing tests that drove a verified `PollHub` over the **real sb0
  checkpoint fixture** (size 10183, signed by sb0's real key) plus **synthetic / checkpoint-as-tile**
  bytes therefore broke at the new fsck step — and the real sb0 log's leaf preimages are **not
  captured**, so a byte-accurate mirror for it cannot be built (live-fixture capture was explicitly Not
  In Scope, and there is no network here). `next.md`'s Verification line "existing … paths unaffected"
  was based on an assumption that is impossible once fsck runs on the live path. I resolved it by
  **converting those 6 tests to an in-process byte-accurate `testonly.Tree` mirror**
  (`buildVerifiedMirror`, the same fixture style `next.md` endorsed for `fsck_test.go`):
  `TestPollHub{Fork,Shrink,VerifiedAdvances,CacheHitSkipsDidFetch,MirrorsTiles}` + `TestTick{,FrozenUnaffected}`.
  This keeps the build green and the gate un-weakened, and **does not lose trust-root coverage**:
  real-sb0 signature/key parity (`40b74463`, the real signed checkpoint, the `derive_vkey.py` vectors)
  stays thoroughly covered at the `internal/logclient` level (`accept_test.go`, `verify_test.go`,
  `keyid_test.go`, `checkpointkey_test.go`) — the follower tests' job is the composition/wiring, not
  re-asserting the raw signature. The size/keyid assertions moved from the hardcoded `10183` /
  `0x40b74463` to the fixture's `m.size` / `m.keyID` (the key is per-run random via `note.GenerateKey`,
  so it can't be a literal). **Please ratify** dropping the real-sb0 *checkpoint* fixture from the
  follower verified tests; the alternative is capturing byte-accurate live sb0/sb1 tile fixtures into
  `testdata/live/` (the deferred path) and keeping the real checkpoint. The diff is still 1 production
  file; the 3 extra changed files are all tests.
- **`fsckMirror` re-resolves the did:web key every verified poll (a redundant did.json fetch).** The
  vkey was already resolved inside `AcceptCheckpoint` (and `cacheHubKey` caches the key bytes), but
  `RunFsck` needs the vkey *string* and `fsckMirror` re-fetches did.json to get it. This is observable:
  `TestPollHubCacheHitSkipsDidFetch` now asserts cold=3 / warm=2 did.json fetches (was 2 / 1), with the
  +1 each being fsck's resolve. An efficiency slice could thread the already-resolved vkey from
  `AcceptCheckpoint`/`cacheHubKey` through to `fsckMirror` (e.g. via `KeyIDFromVerifier` + the cached
  `hub_keys` row, or by `AcceptCheckpoint` returning the vkey) to drop the extra fetch — out of scope
  here. Flagged for define-next.
- **Surfaced a PRE-EXISTING re-detection fragility (not introduced by this slice).** After a fork
  freeze, two checkpoint rows share the same `tree_size` (the seed + the contradictory evidence), and
  `store.CheckpointAt` uses an **unordered `LIMIT 1`** — so which root a re-poll compares against is
  data-dependent/undefined. The old `TestPollHubFork` happened to get the seed row (re-detection
  fired); the in-process mirror fixture deterministically gets the *evidence* (real) row, so a re-poll
  finds no fork and reaches fsck (which then passes over the byte-accurate mirror). To keep the test
  deterministic I drive fork **re-detection through `freeze` directly** now (the same pattern
  `TestPollHubEquivocation` already uses), while the **first** fork detection still goes through
  `PollHub` (one row → deterministic). Re-poll-of-a-frozen-hub via `PollHub` is still covered by
  `TestTickFrozenUnaffected`. The underlying production concern — re-detection comparing against an
  undefined row when two checkpoints share a size — was already noted in learnings ("implicit
  dependency on insert order"); the equivocation/serving slice that changes prior-root selection should
  add an `ORDER BY rowid` (or pick the *prior accepted* root explicitly). Not fixed here (touches
  `store`, out of scope).
- **`mirrorBundleFetcher.Fetch` matches coords by `strings.HasSuffix` over a map** (iteration order
  non-deterministic). Verified the size-300 tlog-tiles paths have **no** suffix collisions (none is a
  suffix of another), so it is unambiguous for this fixture; a future fixture with colliding paths would
  want exact/longest-suffix matching. Acceptable for test code.
- `buildVerifiedMirror` reuses `equivocation_test.go`'s `equivNodeHash` for the upper hash-tile levels
  (level ≥ 1 when leaves > 256), so the served tiles are byte-accurate across the 256-leaf boundary;
  fsck rebuilds from level-0 tiles + bundles, the upper tiles are mirrored by `ingestTiles` but
  byte-accurate anyway.
