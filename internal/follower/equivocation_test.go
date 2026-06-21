// Tests for the equivocation freeze trigger wired into checkConsistency — the
// third RFC-6962 self-consistency trigger (ADR-0006). The proof is sourced from the
// LOCAL mirror: a real testonly.Tree's level-0 hash tiles are recorded into the
// store via RecordTile (the M2 tile-ingestion writer is not yet wired, so the test
// seeds them directly), so checkConsistency can build a consistency proof over the
// store's SQLiteFetcher exactly as production will once tiles are mirrored.
//
// The tile synthesis ports buildTree/nodeHash from the proofbuilder golden test:
// a node at (treeLevel, treeIndex) is the compact-range fold of the leaf hashes it
// covers, and a level-0 tile holds the tree's bottom-row leaf hashes serialized via
// api.HashTile.MarshalText. Assertions are on observable store outputs only — the
// violations table, the follow cursor, and the alert count — through the
// countRows/assertViolation inspector helpers shared with follower_test.go, never on
// follower internals (per the PRD outbound-fetch seam rule).
package follower

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"
)

// equivTreeLeaves is the synthesized tree size: > 256 so the level-0 tile at index
// 0 is full (256 leaves) and the tile at index 1 is partial (44 leaves), so a
// (smaller, equivTreeLeaves) consistency proof needs nodes from both tiles —
// exercising the full and partial qualifiers in the mirror read.
const equivTreeLeaves = 300

// equivPrevSize is the prior accepted tree size the equivocation is measured
// against; the new observation grows the tree to equivTreeLeaves.
const equivPrevSize = 5

// buildEquivTree returns a testonly.Tree of n leaves over the RFC-6962 hasher,
// matching the proofbuilder golden test's tree construction.
func buildEquivTree(n int) *testonly.Tree {
	tree := testonly.New(rfc6962.DefaultHasher)
	for i := range n {
		tree.AppendData([]byte(fmt.Sprintf("leaf-%d", i)))
	}
	return tree
}

// equivNodeHash recomputes the tree-node hash at (treeLevel, treeIndex) by folding
// the leaf hashes it covers through a compact range, matching the builder's
// recompute so the synthesized tiles are consistent with the tree. The node covers
// leaves [treeIndex<<treeLevel, (treeIndex+1)<<treeLevel), clamped to size.
func equivNodeHash(t *testing.T, tree *testonly.Tree, treeLevel, treeIndex, size uint64) []byte {
	t.Helper()
	first := treeIndex << treeLevel
	last := (treeIndex + 1) << treeLevel
	if last > size {
		last = size
	}
	rf := compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}
	r := rf.NewEmptyRange(0)
	for i := first; i < last; i++ {
		if err := r.Append(tree.LeafHash(i), nil); err != nil {
			t.Fatalf("Append leaf %d: %v", i, err)
		}
	}
	h, err := r.GetRootHash(nil)
	if err != nil {
		t.Fatalf("GetRootHash for node (%d, %d): %v", treeLevel, treeIndex, err)
	}
	return h
}

// level0TileBytes serializes the level-0 hash tile at tileIndex within a tree of
// logSize leaves into tlog-tiles concatenated-hash form, returning the bytes and the
// leaf width (256 for the full tile, the remainder for the trailing partial). A
// level-0 tile holds the tree's bottom-row leaf hashes (treeLevel 0).
func level0TileBytes(t *testing.T, tree *testonly.Tree, tileIndex, logSize uint64) ([]byte, int) {
	t.Helper()
	firstNode := tileIndex * tiles.TileWidth
	var nodes [][]byte
	for n := uint64(0); n < tiles.TileWidth; n++ {
		nodeIndex := firstNode + n
		if nodeIndex >= logSize {
			break
		}
		nodes = append(nodes, equivNodeHash(t, tree, 0, nodeIndex, logSize))
	}
	raw, err := api.HashTile{Nodes: nodes}.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText tile (index %d): %v", tileIndex, err)
	}
	return raw, len(nodes)
}

// seedMirrorTiles records the level-0 hash tiles of tree into the store for hubID,
// so checkConsistency can build the consistency proof from the mirror. It writes the
// full tile at index 0 (width 256) and the trailing partial tile at index 1 (its
// remainder width), covering a tree of equivTreeLeaves leaves.
func seedMirrorTiles(t *testing.T, st *store.Store, hubID int64, tree *testonly.Tree) {
	t.Helper()
	ctx := context.Background()
	observedAt := time.Unix(1_700_000_000, 0)
	for tileIndex := uint64(0); ; tileIndex++ {
		if tileIndex*tiles.TileWidth >= equivTreeLeaves {
			break
		}
		raw, width := level0TileBytes(t, tree, tileIndex, equivTreeLeaves)
		if err := st.RecordTile(ctx, hubID, 0, tileIndex, width, raw, observedAt); err != nil {
			t.Fatalf("RecordTile (index %d, width %d): %v", tileIndex, width, err)
		}
	}
}

// rootArray converts a 32-byte root slice into the [32]byte array CheckpointInfo
// carries, failing the test on a wrong length.
func rootArray(t *testing.T, h []byte) [32]byte {
	t.Helper()
	if len(h) != 32 {
		t.Fatalf("root is %d bytes, want 32", len(h))
	}
	var a [32]byte
	copy(a[:], h)
	return a
}

// flipByte returns a copy of b with its first byte flipped, producing a root that
// cannot be the tree's real root at any size — so the consistency proof fails to
// verify and the observation is a genuine equivocation.
func flipByte(b [32]byte) [32]byte {
	b[0] ^= 0xff
	return b
}

// TestPollHubEquivocation drives the equivocation freeze trigger end-to-end through
// checkConsistency + freeze, sourcing the consistency proof from the local mirror.
// A real ~300-leaf tree's level-0 tiles are recorded into the store; the prior
// accepted checkpoint is seeded at size 5 with the tree's real root at 5. Two
// growing observations at size 300 are then presented over the SAME mirrored tiles:
//   - a WRONG-root observation (the real root with one byte flipped) makes the
//     consistency proof fail to verify, so it freezes with kind "equivocation",
//     frozen=1, a cursor that does not advance, exactly one alert, and evidence
//     surviving a store reopen.
//   - a CORRECT-root observation (the tree's real root at 300) over the same tiles
//     does NOT freeze and advances the cursor — proving the branch is non-vacuous
//     (a "never freezes" or "always freezes" wiring is caught by having both cases).
func TestPollHubEquivocation(t *testing.T) {
	ctx := context.Background()
	tree := buildEquivTree(equivTreeLeaves)

	// --- Freeze case: wrong root over consistent mirrored tiles. ---
	path := filepath.Join(t.TempDir(), "equiv.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	seedMirrorTiles(t, s, hubID, tree)

	// Seed the prior accepted checkpoint at the smaller size with the tree's real
	// root there, then advance the cursor to it.
	prevRoot := tree.HashAt(equivPrevSize)
	observedAt := time.Unix(1_700_000_100, 0)
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   equivPrevSize,
		Root:       prevRoot,
		Raw:        []byte("sb0.iscc.id/log\n5\nprior-root\n"),
		ObservedAt: observedAt,
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, equivPrevSize); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}

	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}

	// A growing observation at 300 with a WRONG root: the consistency proof built
	// from the mirror cannot relate prevRoot@5 to this root@300, so it equivocates.
	wrongInfo := logclient.CheckpointInfo{
		Origin:   "sb0.iscc.id/log",
		TreeSize: equivTreeLeaves,
		Root:     flipByte(rootArray(t, tree.HashAt(equivTreeLeaves))),
	}
	violated, kind, prevRaw, err := checkConsistency(ctx, s, hubID, fs.LastSize, wrongInfo)
	if err != nil {
		t.Fatalf("checkConsistency (wrong root): %v", err)
	}
	if !violated || kind != logclient.ViolationEquivocation {
		t.Fatalf("checkConsistency (wrong root) = (violated=%v, kind=%q), want (true, %q)", violated, kind, logclient.ViolationEquivocation)
	}

	// wrongRaw is the contradictory checkpoint's raw bytes recorded as RawB evidence
	// (the synthesized observation has no real signed note; the freeze path only
	// stores these bytes verbatim as irreplaceable evidence).
	wrongRaw := []byte("sb0.iscc.id/log\n300\nwrong-root\n")
	var alerts int
	alert := func(int64, string) { alerts++ }
	if err := freeze(ctx, s, hubID, kind, prevRaw, wrongRaw, wrongInfo, fs.Frozen, observedAt, alert); err != nil {
		t.Fatalf("freeze: %v", err)
	}

	assertViolation(t, path, hubID, "equivocation")
	fsAfter, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after freeze: %v", err)
	}
	if !fsAfter.Frozen {
		t.Errorf("Frozen = false after an equivocation, want true")
	}
	if fsAfter.LastSize != equivPrevSize {
		t.Errorf("LastSize = %d, want %d (a frozen hub must not advance)", fsAfter.LastSize, equivPrevSize)
	}
	if alerts != 1 {
		t.Errorf("alerts = %d after equivocation detection, want 1", alerts)
	}

	// Re-detection is itself evidence: a second freeze records another violation row
	// but must not re-alert (the hub is already frozen).
	if err := freeze(ctx, s, hubID, kind, prevRaw, wrongRaw, wrongInfo, fsAfter.Frozen, observedAt, alert); err != nil {
		t.Fatalf("second freeze: %v", err)
	}
	if n := countRows(t, path, "violations"); n != 2 {
		t.Errorf("violations after re-detection = %d, want 2 (re-detection is evidence)", n)
	}
	if alerts != 1 {
		t.Errorf("alerts after re-detection = %d, want 1 (exactly-one-alert)", alerts)
	}

	// Evidence + freeze survive a store reopen.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	s2, err := store.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })
	fs2, err := s2.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after reopen: %v", err)
	}
	if !fs2.Frozen {
		t.Errorf("frozen not preserved across restart, want true")
	}
	if n := countRows(t, path, "violations"); n != 2 {
		t.Errorf("violations after reopen = %d, want 2 (evidence survives restart)", n)
	}

	// --- Happy path: correct root over the SAME mirrored tiles does NOT freeze. ---
	cleanPath := filepath.Join(t.TempDir(), "equiv-clean.db")
	sc, err := store.Open(cleanPath)
	if err != nil {
		t.Fatalf("store.Open clean: %v", err)
	}
	t.Cleanup(func() { _ = sc.Close() })

	cleanHub, err := sc.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub clean: %v", err)
	}
	seedMirrorTiles(t, sc, cleanHub, tree)
	if _, _, err := sc.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      cleanHub,
		Status:     "verified",
		TreeSize:   equivPrevSize,
		Root:       tree.HashAt(equivPrevSize),
		Raw:        []byte("sb0.iscc.id/log\n5\nprior-root\n"),
		ObservedAt: observedAt,
	}); err != nil {
		t.Fatalf("seed clean RecordCheckpoint: %v", err)
	}
	if err := sc.AdvanceFollowState(ctx, cleanHub, equivPrevSize); err != nil {
		t.Fatalf("seed clean AdvanceFollowState: %v", err)
	}
	cleanFS, err := sc.FollowState(ctx, cleanHub)
	if err != nil {
		t.Fatalf("FollowState clean: %v", err)
	}

	goodInfo := logclient.CheckpointInfo{
		Origin:   "sb0.iscc.id/log",
		TreeSize: equivTreeLeaves,
		Root:     rootArray(t, tree.HashAt(equivTreeLeaves)),
	}
	violated, _, _, err = checkConsistency(ctx, sc, cleanHub, cleanFS.LastSize, goodInfo)
	if err != nil {
		t.Fatalf("checkConsistency (correct root): %v", err)
	}
	if violated {
		t.Errorf("checkConsistency (correct root) violated, want clean (a consistent growing observation must not freeze)")
	}
	if n := countRows(t, cleanPath, "violations"); n != 0 {
		t.Errorf("violations on the clean path = %d, want 0", n)
	}
}

// TestEquivocationMissingTilesDoesNotFreeze proves the error-vs-violation discipline
// (ADR-0006 "freeze, never crash"): when the mirror has no tiles, the proof cannot
// be built, so the equivocation branch is skipped — a missing tile must NOT be
// misread as an equivocation and freeze the hub. This is the common case in
// production today (tiles are not yet mirrored). A growing observation with an
// even-obviously-wrong root is presented but, with no tiles, yields no violation.
func TestEquivocationMissingTilesDoesNotFreeze(t *testing.T) {
	ctx := context.Background()
	tree := buildEquivTree(equivTreeLeaves)

	path := filepath.Join(t.TempDir(), "no-tiles.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	// Seed the prior accepted checkpoint but record NO tiles, so the proof build
	// fails with a missing-tile error.
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   equivPrevSize,
		Root:       tree.HashAt(equivPrevSize),
		Raw:        []byte("sb0.iscc.id/log\n5\nprior-root\n"),
		ObservedAt: time.Unix(1_700_000_100, 0),
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, equivPrevSize); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}

	wrongInfo := logclient.CheckpointInfo{
		Origin:   "sb0.iscc.id/log",
		TreeSize: equivTreeLeaves,
		Root:     flipByte(rootArray(t, tree.HashAt(equivTreeLeaves))),
	}
	violated, kind, _, err := checkConsistency(ctx, s, hubID, fs.LastSize, wrongInfo)
	if err != nil {
		t.Fatalf("checkConsistency with no mirrored tiles returned an error: %v (a missing tile must not abort the poll)", err)
	}
	if violated {
		t.Errorf("checkConsistency froze on a missing tile (kind=%q), want clean (a missing tile must not freeze)", kind)
	}
	if n := countRows(t, path, "violations"); n != 0 {
		t.Errorf("violations after a missing-tile poll = %d, want 0", n)
	}
}

// TestPollHubGrowingSplitViewFreezes is the end-to-end regression for the closed
// critical gap: a growing split view routed through the full PollHub chain must
// freeze, not silently advance to the inconsistent root. The candidate is a
// self-consistent in-process verified mirror at size 300 (its signature verifies and
// its tiles fsck-rebuild). The PRIOR accepted checkpoint is seeded at the smaller
// size 5 with a WRONG/fabricated root — the real root at 5 with one byte flipped —
// so the consistency proof from prior@5 -> candidate@300 cannot verify. That is a
// growing split view against THIS monitor: the candidate checkpoint is internally
// valid, but it is inconsistent with the prior accepted root.
//
// The reorder is what makes this fire: PollHub now mirrors the candidate-size tiles
// (via ingestTiles) BEFORE checkConsistency, so ConsistencyProofFromTiles(5, 300) is
// buildable and CheckEquivocation returns a true verdict on the inconsistent root.
// Before the reorder the proof hit a missing-candidate-tile error that
// checkConsistency swallowed as a clean pass, and the hub advanced to 300.
//
// Assertions are on observable store outputs ONLY (PRD outbound-fetch seam rule):
// the violations table, the follow cursor, and the alert count — never follower
// internals. Non-vacuity is provided by the consistent-growing advance path
// (TestPollHubVerifiedAdvances / TestPollHubMirrorsTiles) over the same mirror, which
// does NOT freeze — so the suite catches both a "never freezes" and an "always
// freezes" wiring.
func TestPollHubGrowingSplitViewFreezes(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "growing-split.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// The candidate: a self-consistent verified mirror at size 300 (== equivTreeLeaves).
	// Its fetcher serves the did.json, the signed candidate checkpoint, and the
	// byte-accurate candidate tiles ingestTiles will mirror before the check.
	m := buildVerifiedMirror(t, mirrorLeaves)

	// Seed the prior accepted checkpoint at the smaller size 5 with a WRONG root (the
	// real root at 5 with one byte flipped), then advance the cursor to it. The
	// consistency proof from this fabricated prior root to the candidate root cannot
	// verify, so the growing observation is a genuine equivocation.
	wrongPrevRoot := flipByte(rootArray(t, m.tree.HashAt(equivPrevSize)))
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   equivPrevSize,
		Root:       wrongPrevRoot[:],
		Raw:        []byte("sb0.iscc.id/log\n5\nwrong-prior-root\n"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, equivPrevSize); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}

	var alerts int
	alert := func(int64, string) { alerts++ }

	// A violation freezes, never crashes (ADR-0006): the signature is valid, so the
	// verdict is still StatusVerified with a nil error.
	status, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), alert, nil)
	if err != nil {
		t.Fatalf("PollHub over a growing split view = %v, want nil (a violation freezes, never crashes)", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified (a violation is a separate axis)", status)
	}

	assertViolation(t, path, hubID, "equivocation")
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("Frozen = false after a growing split view, want true")
	}
	if fs.LastSize != equivPrevSize {
		t.Errorf("LastSize = %d, want %d (a growing split view must NOT advance to the inconsistent root)", fs.LastSize, equivPrevSize)
	}
	if alerts != 1 {
		t.Errorf("alerts = %d after a growing equivocation, want 1 (exactly-one-alert)", alerts)
	}
	if n := countRows(t, path, "violations"); n != 1 {
		t.Errorf("violations after a growing split view = %d, want 1", n)
	}
}
