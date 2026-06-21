// This file is the M2 root-rebuild conformance test for RunFsck. It synthesizes a
// small tlog-tiles log in-process with merkle's testonly.Tree (over
// rfc6962.DefaultHasher) as the single source of truth for both leaf and node
// hashes, seeds its signed checkpoint + entry bundles + hash tiles into a real
// store.SQLiteFetcher, and asserts RunFsck rebuilds the signed root (and fails when
// a mirrored BLOB is corrupted). It lives in external package logclient_test so it
// may import both logclient and internal/store without adding any production import
// edge from logclient to store.
//
// fsck re-hashes each entry bundle with logclient.LeafHashes, re-derives the lower
// hash tiles, and compares the rebuilt root against the checkpoint's claimed root.
// The seeded entry bundles frame the SAME leaf preimages the tree is built from
// (their LeafHashes output equals the tree's leaf hashes), and the seeded level-0
// hash tile's bottom row equals the tree's node hashes — so the rebuild legitimately
// matches the signed root. The tree (prover) and fsck (verifier) are independent of
// the fetcher under test. fsck is an in-process structural self-check (it shares the
// monitor's own LeafHashes / RFC-6962 code); the fully-independent oracle
// (notecheck) is the deferred CI companion.
package logclient_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"
	"golang.org/x/mod/sumdb/note"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
)

// fsckOrigin is the synthetic signed-note name + checkpoint origin for the in-test
// log. It is a stable name, not a live hub — the keypair is generated in-test, so no
// real hub key is involved. The real testnet checkpoint fixtures are deliberately
// NOT reused: their root commits a ~10183-leaf tree whose tiles are not mirrored here.
const fsckOrigin = "sb0.iscc.id/log"

// fsckLeaves is the within-one-tile log size (< 256), so the log has exactly one
// partial entry bundle at index 0 and one partial level-0 hash tile at index 0, both
// of width fsckLeaves. This avoids the 256-leaf tile boundary for the first green.
const fsckLeaves = 5

// leafPreimages returns n distinct raw leaf preimages. fsck runs
// rfc6962.DefaultHasher.HashLeaf over each, so the tree must be built from these
// SAME bytes (tree.AppendData) for the rebuilt root to match tree.Hash().
func leafPreimages(n int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = []byte(fmt.Sprintf("leaf-%d", i))
	}
	return out
}

// encodeBundle frames raw records into a tlog-tiles entry bundle (each record
// prefixed with its big-endian uint16 length, then concatenated) — the C2SP
// encoding api.EntryBundle.UnmarshalText decodes inside LeafHashes. This is the
// test's independent encode path.
func encodeBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

// seedMirror builds the synthetic log, seeds the store, and returns the fetcher,
// the vkey RunFsck verifies with, and the tree size. The tree (built from the same
// preimages framed into the entry bundle) is the single source of truth: its leaf
// hashes fill the level-0 hash tile and its root is signed into the checkpoint.
func seedMirror(t *testing.T) (store.SQLiteFetcher, string, uint64) {
	t.Helper()
	ctx := context.Background()

	preimages := leafPreimages(fsckLeaves)
	tree := testonly.New(rfc6962.DefaultHasher)
	tree.AppendData(preimages...)
	size := tree.Size()

	// Open a real store and register the hub.
	s, err := store.Open(filepath.Join(t.TempDir(), "fsck.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", fsckOrigin, "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	now := time.Unix(1, 0)

	// Seed the partial entry bundle at index 0 (width == size), framing the SAME
	// preimages the tree was built from.
	bundle := encodeBundle(preimages)
	if err := s.RecordEntryBundle(ctx, hubID, 0, uint8(size), bundle, now); err != nil {
		t.Fatalf("RecordEntryBundle: %v", err)
	}

	// Seed the partial level-0 hash tile at index 0 (width == size): its bottom row
	// is the tree's leaf hashes, concatenated per the tlog-tiles spec.
	nodes := make([][]byte, size)
	for i := range nodes {
		nodes[i] = tree.LeafHash(uint64(i))
	}
	tileRaw, err := api.HashTile{Nodes: nodes}.MarshalText()
	if err != nil {
		t.Fatalf("HashTile.MarshalText: %v", err)
	}
	if err := s.RecordTile(ctx, hubID, 0, 0, uint8(size), tileRaw, now); err != nil {
		t.Fatalf("RecordTile: %v", err)
	}

	// Generate a synthetic keypair named for the origin and sign the checkpoint body
	// "<origin>\n<size>\n<base64(root)>\n" — the exact C2SP tlog-checkpoint framing.
	skey, vkey, err := note.GenerateKey(rand.Reader, fsckOrigin)
	if err != nil {
		t.Fatalf("note.GenerateKey: %v", err)
	}
	signer, err := note.NewSigner(skey)
	if err != nil {
		t.Fatalf("note.NewSigner: %v", err)
	}
	body := fmt.Sprintf("%s\n%d\n%s\n", fsckOrigin, size, base64.StdEncoding.EncodeToString(tree.Hash()))
	signed, err := note.Sign(&note.Note{Text: body}, signer)
	if err != nil {
		t.Fatalf("note.Sign: %v", err)
	}
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   size,
		Root:       tree.Hash(),
		Raw:        signed,
		ObservedAt: now,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}

	return store.SQLiteFetcher{Store: s, HubID: hubID}, vkey, size
}

// TestRunFsck proves fsck.New(...).Check(...) rebuilds the signed root from the
// local SQLiteFetcher and rejects a tampered mirror. The two subtests make the test
// non-vacuous: a green-but-wrong rebuild (one that trivially passes without
// comparing against the checkpoint root) would fail the Corrupted case.
func TestRunFsck(t *testing.T) {
	ctx := context.Background()

	t.Run("RebuildsSignedRoot", func(t *testing.T) {
		f, vkey, _ := seedMirror(t)
		if err := logclient.RunFsck(ctx, vkey, fsckOrigin, f); err != nil {
			t.Fatalf("RunFsck over correctly-seeded mirror = %v, want nil", err)
		}
	})

	t.Run("RejectsCorruptedTile", func(t *testing.T) {
		f, vkey, size := seedMirror(t)

		// Re-read the seeded level-0 hash tile, flip one byte, and re-record it at the
		// same (hub, level, index, width) key so the upsert overwrites in place. The
		// rebuilt root now diverges from the checkpoint root, so Check must fail.
		raw, err := f.ReadTile(ctx, 0, 0, uint8(size))
		if err != nil {
			t.Fatalf("ReadTile seed: %v", err)
		}
		corrupt := make([]byte, len(raw))
		copy(corrupt, raw)
		corrupt[0] ^= 0xff
		if err := f.Store.RecordTile(ctx, f.HubID, 0, 0, uint8(size), corrupt, time.Unix(2, 0)); err != nil {
			t.Fatalf("RecordTile corrupt: %v", err)
		}

		if err := logclient.RunFsck(ctx, vkey, fsckOrigin, f); err == nil {
			t.Fatal("RunFsck over corrupted mirror = nil, want non-nil error")
		}
	})

	t.Run("RejectsCorruptedBundle", func(t *testing.T) {
		f, vkey, size := seedMirror(t)

		// Flip one byte of the leaf preimage inside the entry bundle (past the 2-byte
		// length prefix), re-record it in place. fsck re-hashes the bundle, so the
		// rebuilt leaf — and the root — no longer match the checkpoint.
		raw, err := f.ReadEntryBundle(ctx, 0, uint8(size))
		if err != nil {
			t.Fatalf("ReadEntryBundle seed: %v", err)
		}
		corrupt := make([]byte, len(raw))
		copy(corrupt, raw)
		corrupt[2] ^= 0xff
		if err := f.Store.RecordEntryBundle(ctx, f.HubID, 0, uint8(size), corrupt, time.Unix(2, 0)); err != nil {
			t.Fatalf("RecordEntryBundle corrupt: %v", err)
		}

		if err := logclient.RunFsck(ctx, vkey, fsckOrigin, f); err == nil {
			t.Fatal("RunFsck over corrupted bundle = nil, want non-nil error")
		}
	})
}
