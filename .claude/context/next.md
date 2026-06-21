# Next Work Package

## Step: Close M2's inclusion cross-check Verify bar with a conformance test over the real mirror

## Goal
Satisfy M2's second Verify criterion — "computed inclusion proof matches the hub's
`evidence.IsccLogInclusionProof` for sampled `iscc_id`s" — as an ordinary `go test`. After a verified
`PollHub` populates the local mirror + `iscc_index`, resolve a sampled leaf's `iscc_id → leafIndex` via
`SeqsForISCCID`, build that leaf's hub-side `IsccLogInclusionProof` from the fixture tree, and assert
`logclient.VerifyInclusionEvidence` (recomputing the proof from the mirrored `SQLiteFetcher` tiles)
byte-matches it. This re-arms the inclusion-cross-check oracle gate on the real verified path and
completes M2's Verify bar.

## Scope
- **Create**: `internal/follower/inclusion_test.go` — the conformance test described below.
- **Modify**: (none — see "Why test-only" in Implementation Notes; this is a deliberate, honest choice.)
- **Reference** (exact paths):
  - `/workspace/iscc-monitor/internal/logclient/inclusioncheck.go` — `VerifyInclusionEvidence(ctx,
    fetch TileFetcher, ev InclusionEvidence) error`, `InclusionEvidence{Type, Checkpoint, TreeSize,
    LeafIndex, InclusionProof []string}`, sentinel `ErrInclusionMismatch`. `Type` must be the const
    value `"IsccLogInclusionProof"`; `InclusionProof` is base64-**Std**-encoded sibling hashes.
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` — `SeqsForISCCID(ctx, hubID, isccID)
    ([]uint64, error)` (absent → nil slice, nil err; sorted ascending).
  - `/workspace/iscc-monitor/internal/store/fetcher.go` — `SQLiteFetcher{Store, HubID}` + `ReadTile`
    (the `TileFetcher` the cross-check fetches over; learnings: its signature is byte-identical to
    `logclient.TileFetcher`).
  - `/workspace/iscc-monitor/internal/follower/fsck_test.go` — `buildVerifiedMirror(t, leaves)
    verifiedMirror`, the `verifiedMirror{fetcher, tree *testonly.Tree, checkpoint, size, keyID}` fields,
    `leafISCCID(i)`, `mirrorLeaves = 300`, `openTemp`, `noopAlert`. `m.tree` is the hub: it owns the
    real `InclusionProof`.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/merkle@v0.0.2/testonly/tree.go:100` —
    `(*Tree).InclusionProof(index, size uint64) ([][]byte, error)` (the hub-side prover).
  - `/workspace/iscc-monitor/internal/follower/ingest_test.go:237` — `TestPollHubRecordsProjections`,
    the exact "poll `buildVerifiedMirror`, read back via `SeqsForISCCID`" pattern to mirror.
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/log_tree.py:90` — `inclusion_proof(index, size)`
    = `merkle.inclusion_proof(index, leaf_hashes)`; confirms the hub's evidence is the RFC-6962
    inclusion proof over its leaf hashes, base64-encoded (matches `m.tree.InclusionProof`).

## Not In Scope
- **Do NOT add a tautological production caller in `PollHub`.** There is no inbound hub-evidence
  transport on the follow path yet (no `FetchInclusionEvidence`; proof-serving / `verify-for-me` is a
  later M2/M3 slice). A `PollHub` step that recomputed the monitor's OWN proof and checked it against
  itself would be circular and is forbidden (target.md: a green-but-wrong verify must not ship). The
  Verify-bar criterion is explicitly satisfiable "as ordinary `go test` / `mise run check`".
- **Do NOT capture a live `IsccLogInclusionProof` fixture from sb0/sb1.** Live capture is Not In Scope
  (same posture as the retired real-sb0 mirror); the `buildVerifiedMirror` `testonly.Tree` IS the hub
  for this fixture, exactly as the fsck/equivocation slices synthesize their hub evidence in-process.
- Do NOT serve `inclusion`/`consistency`/`entries` over HTTP — that is the next M2 slice.
- Do NOT touch the open `normal` follower issues (`CheckpointAt` ordering, `AcceptCheckpoint` context
  reuse, frozen-hub advance, tile `p`/`width` duplication, `AdvanceAccepted`, `CheckConsistency`
  collapse). None are required here; each is its own store/refactor-touching slice.
- Do NOT change `schema.sql`, `go.mod`, or `go.sum` (no new dep — `testonly.Tree`, `encoding/base64`,
  `errors`, and `VerifyInclusionEvidence` are all already in the build/test closure of this package).

## Implementation Notes
- **Why test-only is the honest scope here.** `VerifyInclusionEvidence` is the *consumer* of a
  hub-supplied proof; M2's Verify bar is a conformance assertion ("computed proof matches the hub's
  evidence"), and target.md's oracle-gate section says these run "as ordinary `go test` … once the
  package exists." The package exists and is built-but-unwired; the missing piece is the conformance
  test that drives it over a real verified-poll mirror. The follower production code already mirrors the
  tiles (`ingestTiles`) and indexes the leaves (`projectEntryBundle`) — everything the cross-check reads.
  No production line is needed to satisfy the bar without fabricating a circular caller. State this
  reasoning in the test's file docstring so `review` and `advance` see the deliberate choice.
- **Drive a real verified poll first.** `s, _ := openTemp(t)`; `hubID, _ := s.UpsertHub(ctx,
  "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")`; `m := buildVerifiedMirror(t,
  mirrorLeaves)`; `status, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1,
  0), noopAlert, nil)` — assert `err == nil` and `status == logclient.StatusVerified`. This populates
  the mirror BLOBs (so `SQLiteFetcher.ReadTile` works) and `iscc_index` (so `SeqsForISCCID` works).
- **Resolve leaf index from the index (the production resolution path).** For each sampled
  `leafIndex` in `{5, 260}` (one in bundle 0, one past the 256-leaf boundary): `seqs, _ :=
  s.SeqsForISCCID(ctx, hubID, leafISCCID(leafIndex))`; assert `seqs == []uint64{uint64(leafIndex)}`.
  This proves the `iscc_id → leafIndex` resolution the cross-check depends on.
- **Build the hub's evidence from the tree (the oracle's hub side).** `proof, err :=
  m.tree.InclusionProof(uint64(leafIndex), m.size)` (`[][]byte`); base64-Std-encode each hash into a
  `[]string`. Construct `logclient.InclusionEvidence{Type: "IsccLogInclusionProof", Checkpoint:
  string(m.checkpoint), TreeSize: m.size, LeafIndex: uint64(leafIndex), InclusionProof: encoded}`.
- **The cross-check itself (the wired unit).** `f := store.SQLiteFetcher{Store: s, HubID: hubID}`;
  `err := logclient.VerifyInclusionEvidence(ctx, f.ReadTile, ev)` → assert `err == nil`. This is
  exactly the seam the inclusioncheck learnings predicted: "first caller … passes `SQLiteFetcher.
  ReadTile` straight in." The recompute reads the mirror written by `ingestTiles` during the poll.
- **Non-vacuousness (Correctness rule + target.md oracle gate APPLIES — RFC-6962 inclusion crypto).**
  A green-but-wrong check that ignored the proof bytes must fail this test, so include two negatives:
  (1) **wrong leaf** — build the valid proof for leaf 5 but set `ev.LeafIndex = 6`; assert
  `errors.Is(err, logclient.ErrInclusionMismatch)` (the monitor recomputes leaf 6's different proof —
  the sharp negative per the learnings). (2) **corrupted proof** — flip one byte of one decoded hash
  (re-encode) and assert `errors.Is(err, logclient.ErrInclusionMismatch)`. Three independent paths keep
  it non-circular: `m.tree.InclusionProof` (prover/hub), `InclusionProofFromTiles` inside
  `VerifyInclusionEvidence` (monitor recompute over the mirror), and the base64 round-trip.
- **Imports**: the new test file needs `context`, `encoding/base64`, `errors`, `testing`, `time`, plus
  `internal/logclient` and `internal/store` (both already used by the package's other tests). No new
  module dependency; `testonly` is reachable via `m.tree` without importing it directly (but importing
  `merkle/testonly` is already done by `fsck_test.go`, so it is fine either way).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestPollHubInclusion -count=1 ./internal/follower` passes: the verified poll yields
  `StatusVerified`; for leaf 5 and leaf 260 `SeqsForISCCID` returns `[5]`/`[260]` and
  `VerifyInclusionEvidence` over `SQLiteFetcher.ReadTile` returns nil.
- The two negatives in the same test assert `errors.Is(err, logclient.ErrInclusionMismatch)` is true
  for both the wrong-`LeafIndex` and corrupted-proof cases (the happy case returns nil) — so a verify
  that ignored the proof bytes would fail. (`go test -run TestPollHubInclusion -count=1
  ./internal/follower` covers these.)
- `go test -run TestPollHub -count=1 ./internal/follower` still passes (no regression in the existing
  verified-path / mirror / fsck / projection tests).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change),
  and `git status --porcelain` shows only the new `internal/follower/inclusion_test.go` (+ context).
- `go list -deps ./internal/store | grep -E 'internal/logclient|net/http'` is empty (store stays a leaf).
- Conformance/oracle re-run (slice exercises the inclusion crypto path): `go test -count=1
  ./internal/logclient ./internal/follower ./internal/didweb` all pass; `python3 .claude/derive_vkey.py`
  reproduces `40b74463`/`22b08f3e` (then `rm -rf .claude/.scratch`); `cmd/notecheck` accepts the real
  sb0 checkpoint (`OK sb0.iscc.id/log`) and rejects a corrupted one.
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (WASM purity invariant intact).

## Done When
A conformance test drives a verified `PollHub` over `buildVerifiedMirror`, resolves the sampled leaves
via `SeqsForISCCID`, and proves `logclient.VerifyInclusionEvidence` over the real `SQLiteFetcher` mirror
matches the tree-built `IsccLogInclusionProof` for both sampled leaves and rejects the wrong-leaf and
corrupted-proof negatives with `ErrInclusionMismatch` — with `mise run check` green and no
`schema.sql`/`go.mod`/`go.sum` change.
