// Tests for the live tile/entry-bundle mirror writer (ingestTiles) and its
// PollHub-level integration. The unit test (TestIngestTilesWidthMapping) drives
// ingestTiles directly over a boundary tree size (300) through a recording fetcher
// and asserts the exact (level, index, width) BLOBs land in the store — pinning the
// load-bearing p→width translation the store now owns (a full coord, Partial == 0,
// stored at width 256; a 44-leaf partial at width 44). The integration test (TestPollHubMirrorsTiles)
// drives a verified PollHub over the in-process byte-accurate mirror
// (buildVerifiedMirror) and asserts the enumerated coords are mirrored — all on
// observable store outputs (ReadTileBlob / ReadEntryBundleBlob), never follower
// internals.
//
// The rest of the file pins the follow-traffic contract (target.md) at the
// outbound-fetch seam — the URLs the injected Fetcher actually saw, never follower
// internals: an unchanged tree re-fetches only partials
// (TestIngestTilesSkipsMirroredFullCoords), a grown tree fetches only the coords the
// mirror lacks including a partial→full promotion
// (TestIngestTilesGrowthFetchesOnlyNewCoords), a `.p/<W>` path is never served from
// cache even when the coord is mirrored in full (TestIngestTilesAlwaysFetchesPartials
// — the guard against an over-eager skip), the same split holds at the PollHub seam
// with the checkpoint always re-fetched
// (TestPollHubRefetchesCheckpointNotCompletedTiles), and a bundle whose projection
// fold fails is left un-mirrored so the coord stays re-fetchable
// (TestIngestEntryBundleProjectionFaultLeavesCoordRefetchable).
//
// The width-mapping unit test uses synthetic per-URL bytes (the writer is transport +
// CRUD, no crypto). The PollHub integration test must use byte-accurate bytes because
// PollHub now fscks the mirror against the signed root after ingest.
package follower

import (
	"context"
	"crypto/sha256"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// recordingFetcher returns deterministic, URL-unique bytes for every fetch and
// records each requested URL, so a test can assert both the bytes that round-tripped
// into the store and which coords were fetched. It serves any URL (tile, bundle, or
// otherwise) — the writer under test fetches only tile/bundle URLs.
//
// Both coord kinds must be WELL-FORMED for their width, because the store and the
// walk now gate on shape: a hash tile is exactly width*32 bytes (store.RecordTile
// rejects anything else) and a full entry bundle carries exactly TileWidth records
// (ingestEntryBundles rejects a short one). So a tile URL gets widthFromTileURL(url)
// 32-byte hashes with the URL written into the leading bytes, and a bundle URL gets
// that many framed JSON records. Only a non-coord URL keeps the opaque "body:" + url
// form.
type recordingFetcher struct {
	urls []string
}

// widthFromTileURL derives the leaf count a coord's body must carry from its path: a
// `.p/<W>` partial names its own width, a plain path is a full TileWidth coord.
func widthFromTileURL(url string) int {
	if i := strings.LastIndex(url, ".p/"); i >= 0 {
		if w, err := strconv.Atoi(url[i+len(".p/"):]); err == nil {
			return w
		}
	}
	return tiles.TileWidth
}

// recordingTileBody is the well-formed hash-tile body recordingFetcher serves: exactly
// widthFromTileURL(url) 32-byte hashes, with the URL written into the leading bytes so
// the body stays URL-unique for a store read-back to match.
func recordingTileBody(url string) []byte {
	b := make([]byte, widthFromTileURL(url)*sha256.Size)
	copy(b, "body:"+url)
	return b
}

// recordingBundleBody is the body recordingFetcher serves for an entry bundle:
// widthFromTileURL(url) valid log-entry envelopes framed as a tlog-tiles entry
// bundle, so the projection fold decodes it and a full bundle carries its full
// TileWidth records. The FIRST record's iscc_id is "ISCC:" + url (so a test can look
// the bundle's base seq up by URL); later records suffix their offset to stay
// distinct. A store read-back compares against this same body.
func recordingBundleBody(url string) []byte {
	n := widthFromTileURL(url)
	records := make([][]byte, n)
	for i := range records {
		id := "ISCC:" + url
		if i > 0 {
			id += "#" + strconv.Itoa(i)
		}
		records[i] = []byte(`{"iscc_id":"` + id + `","note":{"$schema":"log-entry"}}`)
	}
	return encodeBundle(records)
}

// Fetch records the URL and returns deterministic URL-unique bytes: a framed entry
// bundle for the entries path, a well-formed hash tile for a tile path, otherwise the
// opaque "body:" + url synthetic form.
func (f *recordingFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.urls = append(f.urls, url)
	switch {
	case strings.Contains(url, "/tile/entries/"):
		return recordingBundleBody(url), nil
	case strings.Contains(url, "/tile/"):
		return recordingTileBody(url), nil
	}
	return []byte("body:" + url), nil
}

// TestIngestTilesWidthMapping drives ingestTiles directly over the boundary tree
// size 300 and asserts the exact (level, index, width) writes via store read-back.
// Tree 300 yields hash tiles {0,0,full}, {0,1,p44}, {1,0,p1} and entry bundles
// {0,full}, {1,p44}. The load-bearing assertion is that the full coords
// (Partial == 0) are stored at width tiles.TileWidth (256), not 0, and the
// 44-leaf partials at width 44 — read back at the matching width.
func TestIngestTilesWidthMapping(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := &recordingFetcher{}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	const treeSize = 300

	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt, false); err != nil {
		t.Fatalf("ingestTiles: %v", err)
	}

	// Hash tiles: width is the load-bearing translation. A full coord (Partial 0)
	// must be readable at width 256; a partial at its leaf count.
	tileCases := []struct {
		level, index uint64
		partial      uint8
		wantWidth    int
	}{
		{0, 0, 0, tiles.TileWidth}, // full level-0 tile -> width 256
		{0, 1, 44, 44},             // 44-leaf partial -> width 44
		{1, 0, 1, 1},               // 1-hash partial at level 1 -> width 1
	}
	for _, tc := range tileCases {
		wantURL := "https://sb0.iscc.id/log/" + tiles.TilePath(tc.level, tc.index, tc.partial)
		data, found, err := s.ReadTileBlob(ctx, hubID, tc.level, tc.index, tc.wantWidth)
		if err != nil {
			t.Fatalf("ReadTileBlob L%d I%d W%d: %v", tc.level, tc.index, tc.wantWidth, err)
		}
		if !found {
			t.Errorf("tile L%d I%d not found at width %d (p=%d -> width mismatch?)", tc.level, tc.index, tc.wantWidth, tc.partial)
			continue
		}
		if string(data) != string(recordingTileBody(wantURL)) {
			t.Errorf("tile L%d I%d W%d bytes = %q, want body for %q", tc.level, tc.index, tc.wantWidth, data, wantURL)
		}
	}

	// A full tile must NOT be readable at width 0 — that would make it invisible to
	// the SQLiteFetcher (which looks a full tile up at width 256).
	if _, found, err := s.ReadTileBlob(ctx, hubID, 0, 0, 0); err != nil {
		t.Fatalf("ReadTileBlob full-at-width-0: %v", err)
	} else if found {
		t.Errorf("full tile readable at width 0, want stored at width %d only", tiles.TileWidth)
	}

	// Entry bundles: same width translation.
	bundleCases := []struct {
		index     uint64
		partial   uint8
		wantWidth int
	}{
		{0, 0, tiles.TileWidth}, // full bundle -> width 256
		{1, 44, 44},             // 44-leaf partial -> width 44
	}
	for _, bc := range bundleCases {
		wantURL := "https://sb0.iscc.id/log/" + tiles.EntriesPath(bc.index, bc.partial)
		data, found, err := s.ReadEntryBundleBlob(ctx, hubID, bc.index, bc.wantWidth)
		if err != nil {
			t.Fatalf("ReadEntryBundleBlob I%d W%d: %v", bc.index, bc.wantWidth, err)
		}
		if !found {
			t.Errorf("bundle I%d not found at width %d (p=%d -> width mismatch?)", bc.index, bc.wantWidth, bc.partial)
			continue
		}
		if string(data) != string(recordingBundleBody(wantURL)) {
			t.Errorf("bundle I%d W%d bytes = %q, want body for %q", bc.index, bc.wantWidth, data, wantURL)
		}
	}

	// Every named coord was fetched exactly once: 3 tiles + 2 bundles = 5 URLs.
	if len(fetcher.urls) != 5 {
		t.Errorf("fetched %d URLs, want 5 (3 tiles + 2 bundles for tree 300)", len(fetcher.urls))
	}
}

// readWidth maps a tlog-tiles partial qualifier p (path-API "0 == full") to the
// store's width column so a test can read a mirrored coord back at the width the
// store wrote it (a full coord, p == 0, lands at tiles.TileWidth (256); a partial at
// int(p)). The translation itself now lives solely in internal/store (the production
// p→width authority RecordTile / SQLiteFetcher share); this test-local mirror exists
// only to address the read-side helpers, which still take width int. The
// load-bearing "full coord must be readable at width 256, not 0" invariant is pinned
// directly by TestIngestTilesWidthMapping over the public RecordTile / ReadTileBlob
// surface.
func readWidth(p uint8) int {
	if p == 0 {
		return tiles.TileWidth
	}
	return int(p)
}

// TestPollHubMirrorsTiles drives a verified PollHub over the in-process byte-accurate
// mirror (a 300-leaf tree, crossing the 256-leaf tile boundary), then asserts each
// enumerated coord is mirrored in the store with the exact byte-accurate bytes the
// fetcher served for that coord. The assertion is on observable store outputs
// (ReadTileBlob / ReadEntryBundleBlob returning found == true with the served bytes),
// never on follower internals. Byte-accurate bytes are required because PollHub now
// fscks the mirror against the signed root after ingest — synthetic per-URL bytes
// would fail that rebuild.
func TestPollHubMirrorsTiles(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	m := buildVerifiedMirror(t, mirrorLeaves)

	status, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified", status)
	}

	// Every enumerated hash tile is mirrored at the width its p qualifier maps to,
	// with the exact byte-accurate bytes the fetcher served for that coord's path.
	for _, c := range tiles.TileCoords(m.size) {
		width := readWidth(c.Partial)
		want := m.fetcher.byPath[tiles.TilePath(c.Level, c.Index, c.Partial)]
		data, found, err := s.ReadTileBlob(ctx, hubID, c.Level, c.Index, width)
		if err != nil {
			t.Fatalf("ReadTileBlob L%d I%d W%d: %v", c.Level, c.Index, width, err)
		}
		if !found {
			t.Errorf("tile L%d I%d not mirrored at width %d", c.Level, c.Index, width)
			continue
		}
		if string(data) != string(want) {
			t.Errorf("tile L%d I%d W%d bytes differ from the served bytes", c.Level, c.Index, width)
		}
	}

	// Every enumerated entry bundle is mirrored at the width its p qualifier maps to.
	for _, c := range tiles.BundleCoords(m.size) {
		width := readWidth(c.Partial)
		want := m.fetcher.byPath[tiles.EntriesPath(c.Index, c.Partial)]
		data, found, err := s.ReadEntryBundleBlob(ctx, hubID, c.Index, width)
		if err != nil {
			t.Fatalf("ReadEntryBundleBlob I%d W%d: %v", c.Index, width, err)
		}
		if !found {
			t.Errorf("bundle I%d not mirrored at width %d", c.Index, width)
			continue
		}
		if string(data) != string(want) {
			t.Errorf("bundle I%d W%d bytes differ from the served bytes", c.Index, width)
		}
	}

	// The first level-0 tile of the 300-leaf tree is full and must be readable at
	// width 256 — the SQLiteFetcher's full-tile read path.
	var fetcher2 = store.SQLiteFetcher{Store: s, HubID: hubID}
	if _, err := fetcher2.ReadTile(ctx, 0, 0, 0); err != nil {
		t.Errorf("SQLiteFetcher.ReadTile(0,0,p0) over mirrored store: %v (full tile must round-trip at width 256)", err)
	}
}

// TestPollHubRecordsProjections drives a verified PollHub over the in-process
// byte-accurate 300-leaf mirror and asserts, via the store read seam, that the
// iscc_index projection was persisted for every mirrored entry bundle. For a known
// fixture leaf's iscc_id, SeqsForISCCID returns exactly [seq] (a clean one-seq-per-id
// lookup because leafPreimages gives every leaf a distinct id). It checks a leaf in
// bundle 0 (seq < 256) and a leaf in bundle 1 (seq >= 256) so the bundle baseSeq
// (c.Index * tiles.TileWidth) is exercised across the 256-leaf boundary. The
// assertion is on observable store output (SeqsForISCCID), never follower internals.
func TestPollHubRecordsProjections(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	m := buildVerifiedMirror(t, mirrorLeaves)

	status, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified", status)
	}

	// Each leaf's absolute seq is its preimage index, so SeqsForISCCID(id) for the
	// distinct per-leaf id returns exactly that one seq. Leaf 0 lives in bundle 0
	// (baseSeq 0); leaf 260 in bundle 1 (baseSeq 256), proving the bundle baseSeq math.
	for _, seq := range []uint64{0, 5, 255, 256, 260, mirrorLeaves - 1} {
		id := leafISCCID(int(seq))
		seqs, err := s.SeqsForISCCID(ctx, hubID, id)
		if err != nil {
			t.Fatalf("SeqsForISCCID(%q): %v", id, err)
		}
		if len(seqs) != 1 || seqs[0] != seq {
			t.Errorf("SeqsForISCCID(%q) = %v, want [%d] (projection persisted at the absolute leaf seq)", id, seqs, seq)
		}
	}

	// A non-indexed id returns a nil slice, never a spurious match — proving the
	// read-back above is not vacuously matching everything.
	if seqs, err := s.SeqsForISCCID(ctx, hubID, leafISCCID(mirrorLeaves+1)); err != nil {
		t.Fatalf("SeqsForISCCID(absent): %v", err)
	} else if len(seqs) != 0 {
		t.Errorf("SeqsForISCCID(absent id) = %v, want empty", seqs)
	}
}

// tileURL returns the absolute URL ingestTiles requests for one hash-tile coord, so a
// test states its expectation in coordinates and compares against what the fetcher
// actually saw.
func tileURL(level, index uint64, p uint8) string {
	return "https://sb0.iscc.id/log/" + tiles.TilePath(level, index, p)
}

// bundleURL is tileURL's entry-bundle twin: the absolute URL for one bundle coord.
func bundleURL(index uint64, p uint8) string {
	return "https://sb0.iscc.id/log/" + tiles.EntriesPath(index, p)
}

// assertFetched compares the exact set of URLs a walk requested against the wanted
// set, reporting both the missing and the surplus URLs. Surplus is the load-bearing
// half here: a re-fetched immutable coord shows up as surplus.
func assertFetched(t *testing.T, got, want []string) {
	t.Helper()
	wantSet := make(map[string]bool, len(want))
	for _, u := range want {
		wantSet[u] = true
	}
	gotSet := make(map[string]bool, len(got))
	for _, u := range got {
		gotSet[u] = true
	}
	for u := range wantSet {
		if !gotSet[u] {
			t.Errorf("coord NOT fetched but expected: %s", u)
		}
	}
	for u := range gotSet {
		if !wantSet[u] {
			t.Errorf("coord fetched but must have been served from the mirror: %s", u)
		}
	}
	if len(got) != len(want) {
		t.Errorf("fetched %d URLs, want %d", len(got), len(want))
	}
}

// TestIngestTilesSkipsMirroredFullCoords proves a re-ingest at an unchanged tree size
// re-fetches ONLY the partial coords. Tree 300 names hash tiles {0,0,full},
// {0,1,p44}, {1,0,p1} and entry bundles {0,full}, {1,p44}; the second walk must
// request exactly the three partials, because a completed tile/bundle is immutable
// and already mirrored. The assertion is on the outbound-fetch seam (the URLs the
// injected Fetcher saw), never on follower internals.
func TestIngestTilesSkipsMirroredFullCoords(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
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
	if len(fetcher.urls) != 5 {
		t.Fatalf("cold walk fetched %d URLs, want 5 (3 tiles + 2 bundles)", len(fetcher.urls))
	}

	fetcher.urls = nil
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt, false); err != nil {
		t.Fatalf("second ingestTiles: %v", err)
	}
	assertFetched(t, fetcher.urls, []string{
		tileURL(0, 1, 44), // partial hash tile — gains hashes as the tree grows
		tileURL(1, 0, 1),  // partial hash tile at level 1
		bundleURL(1, 44),  // partial entry bundle
	})
}

// TestIngestTilesGrowthFetchesOnlyNewCoords drives the walk across a real tree growth
// (300 -> 600) and pins both halves of the rule: the coords already mirrored in full
// are not re-fetched, while a coord previously mirrored only as a PARTIAL is fetched
// again once it completes. At 600 the level-0 tile {0,1} and bundle 1 have been
// promoted from 44-leaf partials to full, so they MUST be fetched; tile {0,0} and
// bundle 0 were already full at 300 and must not be.
func TestIngestTilesGrowthFetchesOnlyNewCoords(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := &recordingFetcher{}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", 300, observedAt, false); err != nil {
		t.Fatalf("ingestTiles(300): %v", err)
	}

	fetcher.urls = nil
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", 600, observedAt, false); err != nil {
		t.Fatalf("ingestTiles(600): %v", err)
	}
	assertFetched(t, fetcher.urls, []string{
		tileURL(0, 1, 0),  // 44-leaf partial promoted to full — must be re-fetched
		tileURL(0, 2, 88), // new partial hash tile
		tileURL(1, 0, 2),  // level-1 partial, wider than at size 300
		bundleURL(1, 0),   // 44-leaf partial bundle promoted to full
		bundleURL(2, 88),  // new partial bundle
	})

	// The promoted coords are mirrored at full width, so the next walk skips them.
	fetcher.urls = nil
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", 600, observedAt, false); err != nil {
		t.Fatalf("ingestTiles(600) re-walk: %v", err)
	}
	assertFetched(t, fetcher.urls, []string{
		tileURL(0, 2, 88),
		tileURL(1, 0, 2),
		bundleURL(2, 88),
	})
}

// TestPollHubRefetchesCheckpointNotCompletedTiles asserts the split at the PollHub
// seam: the signed checkpoint is fetched fresh on EVERY poll (that fetch is the
// observation, and it is never served from the mirror), while the completed hash
// tiles and entry bundles of an unchanged tree are not fetched at all. Both halves
// matter — a cached checkpoint would blind the monitor, and a re-fetched full tile is
// the wasted traffic.
func TestPollHubRefetchesCheckpointNotCompletedTiles(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	m := buildVerifiedMirror(t, mirrorLeaves)
	fetcher := &countingFetcher{inner: m.fetcher}

	if _, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil); err != nil {
		t.Fatalf("first PollHub: %v", err)
	}

	fetcher.urls = nil
	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", time.Unix(2, 0), noopAlert, nil)
	if err != nil {
		t.Fatalf("second PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified", status)
	}

	const checkpointURL = "https://sb0.iscc.id/log/checkpoint"
	var sawCheckpoint bool
	for _, u := range fetcher.urls {
		if u == checkpointURL {
			sawCheckpoint = true
		}
	}
	if !sawCheckpoint {
		t.Errorf("second poll did not fetch %s — the checkpoint must never be cached", checkpointURL)
	}

	// Every completed coord of the unchanged tree must be absent from the second
	// poll's fetches; the partials may (and must) still be re-fetched.
	for _, c := range tiles.TileCoords(m.size) {
		if c.Partial != 0 {
			continue
		}
		for _, u := range fetcher.urls {
			if u == tileURL(c.Level, c.Index, 0) {
				t.Errorf("second poll re-fetched completed tile L%d I%d (%s)", c.Level, c.Index, u)
			}
		}
	}
	for _, c := range tiles.BundleCoords(m.size) {
		if c.Partial != 0 {
			continue
		}
		for _, u := range fetcher.urls {
			if u == bundleURL(c.Index, 0) {
				t.Errorf("second poll re-fetched completed entry bundle I%d (%s)", c.Index, u)
			}
		}
	}
}

// healingFetcher serves an undecodable body for one entry-bundle path until healed,
// and delegates everything else to the embedded recordingFetcher (so the bundle
// framing the projection assertions rely on has exactly one definition). It lets a
// test fail the iscc_index projection fold of one bundle and then observe whether the
// next walk re-fetches that bundle.
type healingFetcher struct {
	recordingFetcher
	badPath string
	healed  bool
}

// Fetch serves a two-byte frame promising 65535 record bytes that are not there for
// the bad path (so api.EntryBundle.UnmarshalText fails and the projection fold
// errors), otherwise the embedded recordingFetcher's bytes.
func (f *healingFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	if !f.healed && strings.HasSuffix(url, f.badPath) {
		f.urls = append(f.urls, url)
		return []byte{0xff, 0xff}, nil
	}
	return f.recordingFetcher.Fetch(ctx, url)
}

// TestIngestEntryBundleProjectionFaultLeavesCoordRefetchable pins the invariant that
// makes the mirror safe to use as the fetch cache: a full entry bundle present in the
// mirror implies its iscc_index projection was written. A bundle whose projection
// fold fails must NOT be left recorded, or the walk would skip it forever and the
// index gap would never heal. The assertion is on observable outputs — the URLs the
// second walk requested, and the projections finally in the store.
func TestIngestEntryBundleProjectionFaultLeavesCoordRefetchable(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	const treeSize = 300
	badBundle := tiles.EntriesPath(0, 0) // the FULL bundle — the skippable kind

	fetcher := &healingFetcher{badPath: badBundle}
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt, false); err == nil {
		t.Fatal("ingestTiles over an undecodable bundle = nil, want the wrapped projection fault")
	}

	// The faulting bundle must not be mirrored — being mirrored in full is exactly
	// what would make the next walk skip it.
	if _, found, err := s.ReadEntryBundleBlob(ctx, hubID, 0, tiles.TileWidth); err != nil {
		t.Fatalf("ReadEntryBundleBlob: %v", err)
	} else if found {
		t.Error("bundle 0 mirrored despite its projection fold failing — the next walk would skip it and the iscc_index gap would be permanent")
	}

	fetcher.healed = true
	fetcher.urls = nil
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt, false); err != nil {
		t.Fatalf("healed ingestTiles: %v", err)
	}

	var refetched bool
	for _, u := range fetcher.urls {
		if u == bundleURL(0, 0) {
			refetched = true
		}
	}
	if !refetched {
		t.Errorf("healed walk did not re-fetch bundle 0 (%s); fetched %v", bundleURL(0, 0), fetcher.urls)
	}
	// The projection the failed fold owed is now present.
	seqs, err := s.SeqsForISCCID(ctx, hubID, "ISCC:"+bundleURL(0, 0))
	if err != nil {
		t.Fatalf("SeqsForISCCID: %v", err)
	}
	if len(seqs) != 1 || seqs[0] != 0 {
		t.Errorf("SeqsForISCCID after the heal = %v, want [0] (the projection was re-folded)", seqs)
	}
}

// TestIngestTilesAlwaysFetchesPartials pins the other half of the rule: a `.p/<W>`
// path is ALWAYS fetched fresh, even when the same coord is already mirrored at full
// width. A coord can be enumerated as a partial after it is mirrored in full when the
// observed tree SHRINKS — the walk runs at the observed size before the
// self-consistency check, so the contradicting hub's own partial bytes are mirrored as
// evidence (ADR-0006) rather than silently served from the full row. Only its PARTIAL
// bytes: a completed coord the mirror already holds is skipped, so the contradicting
// hub's completed-coord bytes are never captured.
func TestIngestTilesAlwaysFetchesPartials(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := &recordingFetcher{}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	// At 600 the level-0 tile {0,1} and bundle 1 are complete and mirrored in full.
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", 600, observedAt, false); err != nil {
		t.Fatalf("ingestTiles(600): %v", err)
	}

	// A shrink back to 300 names those same coords as 44-leaf PARTIALS. Their bytes
	// differ from the full rows, so they must be fetched, not skipped.
	fetcher.urls = nil
	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", 300, observedAt, false); err != nil {
		t.Fatalf("ingestTiles(300) after 600: %v", err)
	}
	assertFetched(t, fetcher.urls, []string{
		tileURL(0, 1, 44), // mirrored in full at 600 — still re-fetched as a partial
		tileURL(1, 0, 1),
		bundleURL(1, 44), // ditto for the entry bundle
	})
}
