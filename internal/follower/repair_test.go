// Tests for the mirror's repair path — what happens when a COMPLETED (full-width)
// coord is wrong in the local mirror. Such a coord is served from the mirror instead
// of being re-fetched (that is the follow-traffic cache), so nothing else revalidates
// it, and both the self-consistency proof and the fsck root-rebuild read it. These
// tests pin the two consequences PollHub must not have: convicting an HONEST hub of a
// self-consistency violation on corrupt local bytes, and wedging the poll (and with
// it, since Loop.Tick is sequential, the whole realm) on a rebuild that never returns.
//
// Every assertion is on observable outputs — the follow cursor, the violations table,
// the alert count, the projections in the store — never on follower internals.
package follower

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// stripProjections deletes a hub's iscc_index rows below seq over an independent
// connection to the store's file (the countRows pattern), recreating a database whose
// entry bundles are mirrored but whose projections were never written.
func stripProjections(t *testing.T, dbPath string, hubID int64, belowSeq int) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open inspector db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DELETE FROM iscc_index WHERE hub_id = ? AND seq < ?", hubID, belowSeq); err != nil {
		t.Fatalf("strip projections: %v", err)
	}
}

// corruptCompletedTile flips a byte in the mirrored FULL-width hash tile at (level,
// index), the coord kind the ingest walk serves from the mirror and never re-fetches.
func corruptCompletedTile(t *testing.T, s *store.Store, hubID int64, level, index uint64) {
	t.Helper()
	ctx := context.Background()
	raw, found, err := s.ReadTileBlob(ctx, hubID, level, index, tiles.TileWidth)
	if err != nil || !found {
		t.Fatalf("ReadTileBlob(L%d I%d full): err=%v found=%v", level, index, err, found)
	}
	corrupt := make([]byte, len(raw))
	copy(corrupt, raw)
	corrupt[0] ^= 0xff
	if err := s.RecordTile(ctx, hubID, level, index, 0, corrupt, time.Unix(2, 0)); err != nil {
		t.Fatalf("RecordTile corrupt: %v", err)
	}
}

// TestPollHubDoesNotFreezeHonestHubOnCorruptMirror is the load-bearing one. The
// consistency proof is built from the mirror, so a wrong completed tile makes an
// HONEST hub's growing observation fail RFC-6962 consistency against its own prior
// accepted root. Freezing on that would brand the hub an equivocator, fire the alert,
// and write a bogus violation row — permanently, since v1 has no unfreeze. PollHub
// must instead re-fetch authoritatively and re-derive the verdict before acting.
//
// Non-vacuity: TestPollHubGrowingSplitViewFreezes drives the same seam with a genuinely
// contradictory prior root and DOES freeze, so the suite catches a "never freezes"
// wiring as well as this "freezes an honest hub" one.
func TestPollHubDoesNotFreezeHonestHubOnCorruptMirror(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "repair-freeze.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	m := buildVerifiedMirror(t, mirrorLeaves)

	// Populate the mirror as a prior clean poll would, then corrupt one COMPLETED tile.
	if err := ingestTiles(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", m.size, time.Unix(1, 0), false); err != nil {
		t.Fatalf("seed ingestTiles: %v", err)
	}
	corruptCompletedTile(t, s, hubID, 0, 0)

	// The prior accepted checkpoint is the hub's REAL root at size 5, so this hub is
	// honest and its growth to 300 is genuinely consistent.
	correctPrevRoot := rootArray(t, m.tree.HashAt(equivPrevSize))
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   equivPrevSize,
		Root:       correctPrevRoot[:],
		Raw:        []byte("sb0.iscc.id/log\n5\ncorrect-prior-root\n"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, equivPrevSize); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}

	var alerts int
	if _, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(3, 0), func(int64, string) { alerts++ }, nil); err != nil {
		t.Fatalf("PollHub over an honest hub with a corrupt mirror = %v, want nil", err)
	}

	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if fs.Frozen {
		t.Error("Frozen = true for an HONEST hub, want false (the freeze was derived from corrupt local bytes, not from the hub)")
	}
	if fs.LastSize != m.size {
		t.Errorf("LastSize = %d, want %d (the honest advance must go through)", fs.LastSize, m.size)
	}
	if alerts != 0 {
		t.Errorf("alerts = %d, want 0 (no operator should be paged for local corruption)", alerts)
	}
	if n := countRows(t, path, "violations"); n != 0 {
		t.Errorf("violations = %d, want 0 (a bogus violation row is not evidence)", n)
	}
	// The repair actually happened: the mirror now rebuilds to the signed root.
	if err := fsckMirror(ctx, s, hubID, m.vkey, "sb0.iscc.id/log"); err != nil {
		t.Errorf("fsckMirror after the poll = %v, want nil (the corrupt coord must have been re-fetched)", err)
	}
}

// TestPollHubBoundsAndRepairsWedgedRootRebuild pins the other half. With a corrupt
// COMPLETED tile the upstream fsck does not fail — it blocks forever on an internal
// channel send and ignores context cancellation — so an inline call would hang the
// poll and, because Loop.Tick walks hubs sequentially, stop the whole realm from being
// monitored. PollHub must bound the rebuild, repair the mirror, and complete.
//
// It also proves the abandoned rebuild goroutine does not strand the store: the repair
// walk and the second rebuild both run afterwards on the same single-connection pool.
func TestPollHubBoundsAndRepairsWedgedRootRebuild(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", fsckOrigin, "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	m := buildVerifiedMirror(t, mirrorLeaves)

	// A clean poll first: mirror populated and an accepted checkpoint recorded.
	if _, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil); err != nil {
		t.Fatalf("clean PollHub: %v", err)
	}
	corruptCompletedTile(t, s, hubID, 0, 0)

	// Shorten the bound so the wedge is proven in seconds, not the production budget.
	restore := fsckTimeout
	fsckTimeout = 3 * time.Second
	t.Cleanup(func() { fsckTimeout = restore })

	// The hub has grown to the same size, so this is a clean re-poll whose only
	// obstacle is the corrupt mirror. It must RETURN — the whole point of the bound.
	done := make(chan error, 1)
	go func() {
		_, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(2, 0), noopAlert, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("PollHub over a wedged rebuild = %v, want nil (bound, then repair, then rebuild)", err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("PollHub did not return: the root rebuild is unbounded, so one hub can stall the entire realm")
	}

	// The store is still usable after the abandoned rebuild goroutine was left behind,
	// and the mirror was actually repaired.
	if err := fsckMirror(ctx, s, hubID, m.vkey, fsckOrigin); err != nil {
		t.Errorf("fsckMirror after the repair = %v, want nil", err)
	}
}

// emptyBundleFetcher serves a zero-byte body for one entry-bundle path and the
// ordinary recordingFetcher bytes for everything else. A zero-byte body is the nasty
// case: it decodes to zero leaves WITHOUT error, so nothing upstream objects to it.
type emptyBundleFetcher struct {
	recordingFetcher
	badPath string
}

// Fetch serves the empty body for the bad path, otherwise the embedded fetcher's.
func (f *emptyBundleFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	if strings.HasSuffix(url, f.badPath) {
		f.urls = append(f.urls, url)
		return []byte{}, nil
	}
	return f.recordingFetcher.Fetch(ctx, url)
}

// TestIngestRejectsShortFullEntryBundle pins the entry-bundle admission gate. A
// COMPLETED bundle carries exactly TileWidth records; an empty or truncated body
// decodes to fewer WITHOUT error, so without the gate a zero-byte 200 would be stored
// at full width, skipped by every later walk, and served from the mirror as the hub's
// authoritative entries — silently truncating both the mirror and the iscc_index.
func TestIngestRejectsShortFullEntryBundle(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := &emptyBundleFetcher{badPath: tiles.EntriesPath(0, 0)} // the FULL bundle
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", 300, time.Unix(1, 0), false); err == nil {
		t.Fatal("ingestTiles over an empty full bundle = nil, want a rejection")
	}
	// It must not be mirrored — being mirrored in full is what would make it permanent.
	if _, found, err := s.ReadEntryBundleBlob(ctx, hubID, 0, tiles.TileWidth); err != nil {
		t.Fatalf("ReadEntryBundleBlob: %v", err)
	} else if found {
		t.Error("empty full bundle was mirrored; the walk would skip it forever and serve zero entries as authoritative")
	}
}

// TestIngestRefoldsMirroredBundleWithMissingProjection pins the legacy-database half:
// a full entry bundle whose iscc_index projection is absent — what the earlier
// record-then-project write order could leave behind, and what a database predating
// projection support holds — must NOT be treated as fully mirrored. Presence of the
// BLOB alone does not establish target.md's "a full entry bundle present in the mirror
// implies its iscc_index projection was written", so the walk has to re-fetch and
// re-fold it instead of skipping it forever with a permanently incomplete index.
func TestIngestRefoldsMirroredBundleWithMissingProjection(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := &recordingFetcher{}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	const treeSize = 300

	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt, false); err != nil {
		t.Fatalf("first ingestTiles: %v", err)
	}
	// Recreate the legacy shape over an independent connection (as countRows does): the
	// full bundle 0 stays mirrored, its projections do not.
	stripProjections(t, path, hubID, tiles.TileWidth)
	if seqs, err := s.SeqsForISCCID(ctx, hubID, "ISCC:"+bundleURL(0, 0)); err != nil {
		t.Fatalf("SeqsForISCCID after strip: %v", err)
	} else if len(seqs) != 0 {
		t.Fatalf("projections still present after the strip (%v) — the fixture is wrong", seqs)
	}

	fetcher.urls = nil
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt, false); err != nil {
		t.Fatalf("second ingestTiles: %v", err)
	}

	var refetched bool
	for _, u := range fetcher.urls {
		if u == bundleURL(0, 0) {
			refetched = true
		}
	}
	if !refetched {
		t.Errorf("bundle 0 was NOT re-fetched (%v); a mirrored-but-unprojected bundle must stay re-fetchable", fetcher.urls)
	}
	seqs, err := s.SeqsForISCCID(ctx, hubID, "ISCC:"+bundleURL(0, 0))
	if err != nil {
		t.Fatalf("SeqsForISCCID: %v", err)
	}
	if len(seqs) != 1 || seqs[0] != 0 {
		t.Errorf("SeqsForISCCID after the re-walk = %v, want [0] (the missing projection must be re-folded)", seqs)
	}
}
