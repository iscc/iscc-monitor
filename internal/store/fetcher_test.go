// Tests for SQLiteFetcher — the read-only Fetcher view over a hub's mirrored
// tiles / entry bundles / latest checkpoint. They drive the public Fetcher methods
// against a t.TempDir() database and assert on observable bytes: the p→width
// mapping round-trip, the os.ErrNotExist contract, the partial→full fallback, and
// that ReadCheckpoint returns the highest-tree_size bytes. Conformance to the
// tessera Fetcher shape is pinned by the local fsckFetcher interface below.
package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// fsckFetcher is a byte-for-byte copy of tessera's fsck.Fetcher interface
// (github.com/transparency-dev/tessera/fsck, v1.0.2: ReadCheckpoint / ReadTile /
// ReadEntryBundle). It is duplicated here rather than imported because the real
// fsck package's compile closure drags otel / klog / errgroup / transparency-dev
// formats into the module graph, which would force new go.mod / go.sum indirect
// entries and break the store's leaf purity. The var assertion below pins
// SQLiteFetcher to this exact three-method shape, so any signature drift from the
// real interface fails the build — same guarantee as importing fsck, without its
// dependency closure.
type fsckFetcher interface {
	ReadCheckpoint(ctx context.Context) ([]byte, error)
	ReadTile(ctx context.Context, l, i uint64, p uint8) ([]byte, error)
	ReadEntryBundle(ctx context.Context, i uint64, p uint8) ([]byte, error)
}

// var assertion: SQLiteFetcher structurally satisfies the tessera Fetcher shape.
var _ fsckFetcher = SQLiteFetcher{}

// TestFetcherReadTileFull confirms a full tile written at width 256 reads back via
// the p == 0 ("full") path-API request.
func TestFetcherReadTileFull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	data := bytes.Repeat([]byte{0x7a}, 8192)
	if err := s.RecordTile(ctx, hub, 0, 0, 0, data, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile: %v", err)
	}

	got, err := f.ReadTile(ctx, 0, 0, 0)
	if err != nil {
		t.Fatalf("ReadTile(p=0): %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("ReadTile(p=0) bytes = %x…, want %x…", got[:4], data[:4])
	}
}

// TestFetcherReadTilePartial confirms a width 44 partial reads back via the p == 44
// request, mapping p directly onto the width column.
func TestFetcherReadTilePartial(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	data := bytes.Repeat([]byte{0x44}, 44*32)
	if err := s.RecordTile(ctx, hub, 0, 1, uint8(44), data, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile partial: %v", err)
	}

	got, err := f.ReadTile(ctx, 0, 1, 44)
	if err != nil {
		t.Fatalf("ReadTile(p=44): %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("ReadTile(p=44) bytes = %x…, want %x…", got[:2], data[:2])
	}
}

// TestFetcherReadTileNotExist confirms an un-written tile returns an error
// satisfying errors.Is(err, os.ErrNotExist) — the contract the fsck consumer and
// the fallback rely on.
func TestFetcherReadTileNotExist(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	_, err := f.ReadTile(ctx, 0, 5, 0)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadTile miss err = %v, want os.ErrNotExist", err)
	}
}

// TestFetcherReadTilePartialFallsBackToFull confirms the PartialOrFullResource
// semantics: with only a full (width 256) tile stored, a partial request (p > 0)
// that misses falls back to the full tile's bytes.
func TestFetcherReadTilePartialFallsBackToFull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	full := bytes.Repeat([]byte{0xfa}, 8192)
	if err := s.RecordTile(ctx, hub, 0, 0, 0, full, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile full: %v", err)
	}

	got, err := f.ReadTile(ctx, 0, 0, 200)
	if err != nil {
		t.Fatalf("ReadTile(p=200) fallback: %v", err)
	}
	if !bytes.Equal(got, full) {
		t.Errorf("partial->full fallback bytes = %x…, want full %x…", got[:4], full[:4])
	}
}

// TestFetcherReadTilePartialNoFallbackNoFull confirms that when neither the partial
// nor the full tile exists, a partial request still surfaces os.ErrNotExist.
func TestFetcherReadTilePartialNoFallbackNoFull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	_, err := f.ReadTile(ctx, 0, 0, 100)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadTile(p=100) both-missing err = %v, want os.ErrNotExist", err)
	}
}

// TestFetcherReadEntryBundle confirms the same p→width mapping and partial→full
// fallback for entry bundles.
func TestFetcherReadEntryBundle(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	full := bytes.Repeat([]byte{0xb0}, 4096)
	partial := bytes.Repeat([]byte{0xb1}, 512)
	if err := s.RecordEntryBundle(ctx, hub, 0, 0, full, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordEntryBundle full: %v", err)
	}
	if err := s.RecordEntryBundle(ctx, hub, 1, uint8(44), partial, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordEntryBundle partial: %v", err)
	}

	gotFull, err := f.ReadEntryBundle(ctx, 0, 0)
	if err != nil || !bytes.Equal(gotFull, full) {
		t.Errorf("ReadEntryBundle(0,p=0) err=%v equal=%v", err, bytes.Equal(gotFull, full))
	}
	gotPartial, err := f.ReadEntryBundle(ctx, 1, 44)
	if err != nil || !bytes.Equal(gotPartial, partial) {
		t.Errorf("ReadEntryBundle(1,p=44) err=%v equal=%v", err, bytes.Equal(gotPartial, partial))
	}
	// Partial request for index 0 (only the full bundle exists) falls back to full.
	gotFallback, err := f.ReadEntryBundle(ctx, 0, 200)
	if err != nil || !bytes.Equal(gotFallback, full) {
		t.Errorf("ReadEntryBundle(0,p=200) fallback err=%v equal=%v", err, bytes.Equal(gotFallback, full))
	}
}

// TestFetcherReadEntryBundleNotExist confirms an un-written bundle surfaces
// os.ErrNotExist.
func TestFetcherReadEntryBundleNotExist(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	_, err := f.ReadEntryBundle(ctx, 9, 0)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadEntryBundle miss err = %v, want os.ErrNotExist", err)
	}
}

// TestFetcherReadCheckpointHighestSize confirms ReadCheckpoint returns the
// highest-tree_size raw checkpoint bytes when two sizes are stored.
func TestFetcherReadCheckpointHighestSize(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	rawLow := []byte("cp-100")
	rawHigh := []byte("cp-200")
	mustRecordCP(t, s, hub, 100, []byte("root-lo"), rawLow)
	mustRecordCP(t, s, hub, 200, []byte("root-hi"), rawHigh)

	got, err := f.ReadCheckpoint(ctx)
	if err != nil {
		t.Fatalf("ReadCheckpoint: %v", err)
	}
	if !bytes.Equal(got, rawHigh) {
		t.Errorf("ReadCheckpoint = %q, want %q (highest tree_size)", got, rawHigh)
	}
}

// TestFetcherReadCheckpointNotExist confirms ReadCheckpoint on a hub with no
// checkpoint row surfaces os.ErrNotExist.
func TestFetcherReadCheckpointNotExist(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)
	f := SQLiteFetcher{Store: s, HubID: hub}

	_, err := f.ReadCheckpoint(ctx)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadCheckpoint miss err = %v, want os.ErrNotExist", err)
	}
}
