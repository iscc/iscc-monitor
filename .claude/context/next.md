# Next Work Package

## Step: Wire `CheckEquivocation` into `follower.checkConsistency` as the third self-consistency trigger

## Goal
Close the last open M1 self-consistency trigger end-to-end: when a hub presents a verified checkpoint
that *grows* the tree but whose RFC-6962 consistency proof fails to relate the prior accepted root to the
new root, the follower must freeze it (ADR-0006). All three ingredients are already pure and golden-tested
(`CheckEquivocation`, `SQLiteFetcher.ReadTile`, `ConsistencyProofFromTiles`); this step connects them.

## Scope
- **Create**: (none — tiles are synthesized in-test, see Implementation Notes; no on-disk fixtures)
- **Modify**:
  - `internal/follower/follower.go` (the only non-test production file: extend `checkConsistency` with an
    equivocation branch that sources the proof from the local mirror and calls `CheckEquivocation`)
  - `internal/follower/follower_test.go` (test only — add an equivocation freeze test)
- **Reference** (read, do not import or edit):
  - `internal/logclient/consistency.go` — `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot, proof)`
    and `ViolationEquivocation = "equivocation"`.
  - `internal/logclient/proofbuilder.go` — `ConsistencyProofFromTiles(ctx, fetch TileFetcher, smaller,
    larger uint64)`; the `TileFetcher` signature (byte-identical to `SQLiteFetcher.ReadTile`).
  - `internal/logclient/proofbuilder_test.go` — the **exact pattern** for synthesizing a real
    `testonly.Tree`'s hash tiles via `api.HashTile.MarshalText` (`buildTree`, `nodeHash`, `tileFetcherFor`);
    port this into the follower test to record tiles into the store.
  - `internal/store/fetcher.go` — `SQLiteFetcher{Store, HubID}` and `SQLiteFetcher.ReadTile` (the method
    value to pass as the `TileFetcher`).
  - `internal/store/tiles.go` — `RecordTile(ctx, hubID, level, index uint64, width int, data, observedAt)`
    to seed mirrored tiles in the test; `internal/tiles/layout.go` for `TileWidth`/`PartialTileSize`.
  - `internal/follower/follower.go` (`checkConsistency`, `follower.go:236`) and
    `internal/follower/follower_test.go` (`TestPollHubFork`, line 184) — the existing seeding +
    freeze-assertion pattern (`RecordCheckpoint` + `AdvanceFollowState` seed, `assertViolation`, `countRows`,
    restart) to mirror.

## Not In Scope
- **`fsck` root-rebuild conformance** (`fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion
  cross-check vs the hub's `IsccLogInclusionProof`). It needs the heavy `fsck`/otel/klog dep in a non-leaf
  package and is the *next* conformance slice — explicitly deferred.
- **Capturing real on-disk tile fixtures under `testdata/live/`.** This step synthesizes tiles in-test
  (the proven `proofbuilder_test` pattern); the real-fixture capture belongs to the `fsck`/M2 slice.
- **sb1 stale-key fixture refresh** (`22b08f3e`→`069d0f14`) — its own separate trust-root step.
- **Structured logs (`slog`), `/metrics`, real alert transport, CI/`notecheck` wiring, the `go mod tidy`
  go.sum divergence** — all separate later slices; do not touch them here.
- **Mirroring tiles inside `PollHub`'s production path.** This step only *reads* tiles for the proof; the
  tile-ingestion writer (fetching `.../tile/...` from the hub) is M2 work, not this step.

## Implementation Notes
- **Where**: `checkConsistency` already computes `prevRoot [32]byte` from `CheckpointAt`. Add the
  equivocation branch there, after the shrink/fork checks, honoring the **shrink → fork → equivocation**
  order (learnings: shrink and fork are dep-free and mutually exclusive by size; equivocation is the
  growing-pair case `info.TreeSize > prevSize`). Extend the existing `switch` to evaluate `shrink`, then
  `fork`, then equivocation.
- **Proof source**: build the proof from the local mirror only — never re-hit the hub. Construct
  `store.SQLiteFetcher{Store: st, HubID: hubID}` and pass its `ReadTile` method value straight into
  `ConsistencyProofFromTiles(ctx, fetcher.ReadTile, prevSize, info.TreeSize)` — the `TileFetcher` signature
  was made byte-identical to `SQLiteFetcher.ReadTile` for exactly this (learnings). Feed the resulting
  `[][]byte` to `CheckEquivocation(prevSize, prevRoot, info.TreeSize, info.Root, proofHashes)`.
- **Compare against the prior accepted root, not the contradicting evidence** (learnings + Correctness rule
  ADR-0006): `prevRoot`/`prevSize` come from `CheckpointAt(prevSize)` (the prior accepted checkpoint), and
  `info.Root`/`info.TreeSize` are the new observation. Do not source the proof or roots from the freeze
  evidence row. (Note the existing fork re-detection subtlety: `CheckpointAt`'s `LIMIT 1` returns the
  lower-rowid prior root; this step must preserve that "compare against the prior accepted root" semantics.)
- **Error vs. violation discipline (ADR-0006 "freeze, never crash")**: `CheckEquivocation` already turns a
  non-verifying proof into a `(violated=true, err=nil)` verdict. But `ConsistencyProofFromTiles` returns a
  genuine Go error on a *tile-fetch/parse fault* (e.g. tiles not mirrored yet, a wrapped `os.ErrNotExist`) —
  that is **not** a violation verdict. Decide and document the policy explicitly in the `checkConsistency`
  doc comment: a missing-tile / proof-build error must NOT be misread as an equivocation. The conservative
  choice that matches the current "shrink/fork only" behavior and the "no tile-ingestion yet" reality is to
  treat a proof-build error as "cannot evaluate equivocation this poll" — skip the equivocation branch (no
  violation, no crash) rather than freeze or abort the poll. (Rationale: until M2 mirrors tiles in production
  the branch will usually have no tiles; freezing on a missing-tile error would be a false positive, and
  aborting the poll would break the loop. The branch becomes load-bearing once tiles are mirrored.) Keep the
  swallow narrow and explained in a comment — distinguish it from a genuine `st` query failure if you can,
  but at minimum do not let a missing tile freeze a hub.
- **Guards**: only reach the equivocation branch when `prevFound && info.TreeSize > prevSize` (growth). The
  `prevSize == 0` early-return at the top of `checkConsistency` already covers the fresh-store case;
  `CheckEquivocation`'s own `nextSize <= prevSize` guard is a backstop.
- **Test (the load-bearing part)**: synthesize a real `testonly.New(rfc6962.DefaultHasher)` tree (port
  `buildTree`/`nodeHash`/`tileFetcherFor` from `proofbuilder_test.go`), record its level-0 hash tiles into
  the store via `RecordTile` (full tile at index 0 = width `tiles.TileWidth`; partial at index 1 = the
  `PartialTileSize` width), seed a prior accepted checkpoint at `prevSize` with `tree.HashAt(prevSize)`, then
  drive the equivocation by presenting a *new* checkpoint at `largerSize` whose root is **wrong** (e.g.
  `tree.HashAt(largerSize)` with one byte flipped, or a different tree's root) so `VerifyConsistency` fails →
  expect a freeze with `violations.kind == "equivocation"`, `frozen=1`, exactly-one-alert, and evidence
  surviving a restart — mirroring `TestPollHubFork`. Also assert the **happy path**: a growing observation at
  `largerSize` with the *correct* `tree.HashAt(largerSize)` and consistent mirrored tiles does NOT freeze
  (advances normally), proving the branch is non-vacuous (a green-but-wrong "always freezes" or "never
  freezes" wiring is caught by having both cases).
- **Driving the test**: `PollHub`'s production path needs a genuinely `StatusVerified` signed checkpoint, but
  the equivocation needs full control over `(prevSize, prevRoot, info.TreeSize, info.Root)`. Because
  `checkConsistency` is unexported, the test may call it directly (package-internal) for tight control over
  `info`, and/or exercise the full freeze via the `freeze` helper — choose whichever yields a clean,
  deterministic assertion on observable store outputs (per the PRD seam rule, assert on
  `violations`/`follow_state`/coverage, never follower internals). A direct `checkConsistency` +`freeze` test
  avoids needing a fixture checkpoint whose signature happens to encode the synthesized tree's root.
- **Purity / leaf discipline** (learnings): `internal/store` stays a leaf — do NOT import `logclient` into
  store. The follower already imports both `logclient` and `store`; `SQLiteFetcher` lives in `store`, so the
  follower constructs it directly. Production follower imports stay `{context, fmt, logclient, store, time}`;
  any `testonly` / `merkle` / `tessera/api` imports for tile synthesis go in the **test file only**.

## Verification
- `mise run check` is green (build + vet + test, all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestPollHubEquivocation ./internal/follower` passes (the new equivocation freeze test).
- `go test -count=1 -run 'TestPollHubFork|TestPollHubShrink|TestPollHubVerifiedAdvances' ./internal/follower`
  still passes (the other two triggers + the clean advance path are unregressed).
- The equivocation test asserts, on a wrong-root growing observation over consistent mirrored tiles:
  `violations.kind == "equivocation"`, `follow_state.frozen == 1`, the cursor did NOT advance, exactly one
  alert fired, and the violation rows survive a store reopen.
- The same test asserts the non-vacuous happy path: a correct-root growing observation over the same tiles
  does NOT freeze and advances the cursor.
- `GOOS=js GOARCH=wasm go build ./internal/logclient` still succeeds (the pure proof builder is untouched).

## Done When
`checkConsistency` freezes a hub on a failing growing-pair consistency proof sourced from the local mirror
(and only then), all Verification checks pass, and the other two triggers plus the clean-advance path remain
green.
