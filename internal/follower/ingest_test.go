// Tests for the live tile/entry-bundle mirror writer (ingestTiles) and its
// PollHub-level integration. The unit test (TestIngestTilesWidthMapping) drives
// ingestTiles directly over a boundary tree size (300) through a recording fetcher
// and asserts the exact (level, index, width) BLOBs land in the store — pinning the
// load-bearing widthForP translation (a full coord, Partial == 0, stored at width
// 256; a 44-leaf partial at width 44). The integration test (TestPollHubMirrorsTiles)
// drives a verified PollHub over the in-process byte-accurate mirror
// (buildVerifiedMirror) and asserts the enumerated coords are mirrored — all on
// observable store outputs (ReadTileBlob / ReadEntryBundleBlob), never follower
// internals.
//
// The width-mapping unit test uses synthetic per-URL bytes (the writer is transport +
// CRUD, no crypto). The PollHub integration test must use byte-accurate bytes because
// PollHub now fscks the mirror against the signed root after ingest.
package follower

import (
	"context"
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
// otherwise) — the writer under test fetches only tile/bundle URLs. Entry-bundle
// URLs (the entries path) must return a VALID tlog-tiles frame because ingestTiles
// now folds each mirrored bundle into the iscc_index projection, so for those URLs it
// frames a single valid JSON record embedding the URL (still URL-unique); every other
// URL keeps the opaque "body:" + url synthetic byte form.
type recordingFetcher struct {
	urls []string
}

// recordingBundleBody is the byte-accurate body recordingFetcher serves for an entry
// bundle: one valid log-entry envelope (its iscc_id embeds the URL so the bytes stay
// URL-unique) framed as a one-record tlog-tiles entry bundle, so the projection fold
// decodes it cleanly. A store read-back compares against this same body.
func recordingBundleBody(url string) []byte {
	record := []byte(`{"iscc_id":"ISCC:` + url + `","note":{"$schema":"log-entry"}}`)
	return encodeBundle([][]byte{record})
}

// Fetch records the URL and returns deterministic URL-unique bytes: a valid framed
// entry bundle for the entries path (so the projection fold decodes it), otherwise
// the opaque "body:" + url synthetic form a store read-back can match to the coord.
func (f *recordingFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.urls = append(f.urls, url)
	if strings.Contains(url, "/tile/entries/") {
		return recordingBundleBody(url), nil
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

	if err := ingestTiles(ctx, s, fetcher, hubID, "https://sb0.iscc.id", treeSize, observedAt); err != nil {
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
			t.Errorf("tile L%d I%d not found at width %d (widthForP(%d) mismatch?)", tc.level, tc.index, tc.wantWidth, tc.partial)
			continue
		}
		if string(data) != "body:"+wantURL {
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
			t.Errorf("bundle I%d not found at width %d (widthForP(%d) mismatch?)", bc.index, bc.wantWidth, bc.partial)
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

// TestWidthForP pins the unexported width translation directly: a full qualifier
// (p == 0) maps to tiles.TileWidth (256), any partial to int(p). This is the
// re-derived store one-liner (the store's copy is unexported and must not be
// exported); a wrong p==0 mapping (e.g. to 0) would make full tiles unreadable by
// the SQLiteFetcher.
func TestWidthForP(t *testing.T) {
	cases := []struct {
		p    uint8
		want int
	}{
		{0, tiles.TileWidth}, // full -> 256, NOT 0
		{1, 1},
		{44, 44},
		{255, 255},
	}
	for _, tc := range cases {
		if got := widthForP(tc.p); got != tc.want {
			t.Errorf("widthForP(%d) = %d, want %d", tc.p, got, tc.want)
		}
	}
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

	// Every enumerated hash tile is mirrored at its widthForP width, with the exact
	// byte-accurate bytes the fetcher served for that coord's canonical path.
	for _, c := range tiles.TileCoords(m.size) {
		width := widthForP(c.Partial)
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

	// Every enumerated entry bundle is mirrored at its widthForP width.
	for _, c := range tiles.BundleCoords(m.size) {
		width := widthForP(c.Partial)
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
