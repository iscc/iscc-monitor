// This file is the golden + mutation test for the hub-vs-monitor inclusion
// cross-check: it builds a real RFC-6962 tree with merkle's testonly.Tree, plays
// the hub's role by base64-Std-encoding tree.InclusionProof(index, size) into an
// InclusionEvidence exactly as iscc_hub/log_tree.py inclusion_evidence does, serves
// the tree's tiles through the same in-test TileFetcher, and asserts
// VerifyInclusionEvidence returns nil. The prover (testonly.Tree.InclusionProof),
// the tile-sourced builder (InclusionProofFromTiles), and the base64+bytes compare
// are three independent paths, so a green check is ground truth, not a tautology.
// The mutation subtest flips one proof byte / passes a wrong leaf index and asserts
// errors.Is(err, ErrInclusionMismatch), proving the byte-comparison is load-bearing.
// It reuses buildTree / tileFetcherFor / treeLeaves from proofbuilder_test.go.
package logclient

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/transparency-dev/merkle/testonly"
)

// hubEvidenceFor plays the hub's role: it builds the RFC-6962 inclusion proof for
// leaf index in a tree of size leaves with tree.InclusionProof, then base64-Std
// encodes each sibling hash into an InclusionEvidence exactly as
// iscc_hub/log_tree.py inclusion_evidence emits it.
func hubEvidenceFor(t *testing.T, tree *testonly.Tree, index, leaves uint64) InclusionEvidence {
	t.Helper()
	hashes, err := tree.InclusionProof(index, leaves)
	if err != nil {
		t.Fatalf("tree.InclusionProof(%d, %d): %v", index, leaves, err)
	}
	enc := make([]string, len(hashes))
	for i, h := range hashes {
		enc[i] = base64.StdEncoding.EncodeToString(h)
	}
	return InclusionEvidence{
		Type:           "IsccLogInclusionProof",
		Checkpoint:     "",
		TreeSize:       leaves,
		LeafIndex:      index,
		InclusionProof: enc,
	}
}

// TestVerifyInclusionEvidence asserts the cross-check returns nil when the hub's
// base64-Std inclusionProof matches the monitor's tile-built proof, for leaf indices
// spanning the 256-leaf tile boundary on the 300-leaf tree.
func TestVerifyInclusionEvidence(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	fetch := tileFetcherFor(t, tree, treeLeaves)

	for _, index := range []uint64{0, 5, 255, 256, 299} {
		ev := hubEvidenceFor(t, tree, index, treeLeaves)
		if err := VerifyInclusionEvidence(ctx, fetch, ev); err != nil {
			t.Errorf("VerifyInclusionEvidence(leaf %d) = %v, want nil", index, err)
		}
	}
}

// TestVerifyInclusionEvidenceCorruptedProof flips one byte of one supplied proof
// hash and asserts the cross-check fails with ErrInclusionMismatch — proving the
// byte-comparison is load-bearing (a check that ignored the proof bytes would pass).
func TestVerifyInclusionEvidenceCorruptedProof(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	fetch := tileFetcherFor(t, tree, treeLeaves)

	ev := hubEvidenceFor(t, tree, 5, treeLeaves)
	if len(ev.InclusionProof) == 0 {
		t.Fatal("expected a non-empty inclusion proof to corrupt")
	}
	// Flip one byte of the first proof hash by decoding, mutating, re-encoding.
	h, err := base64.StdEncoding.DecodeString(ev.InclusionProof[0])
	if err != nil {
		t.Fatalf("decode proof hash: %v", err)
	}
	h[0] ^= 0xff
	ev.InclusionProof[0] = base64.StdEncoding.EncodeToString(h)

	err = VerifyInclusionEvidence(ctx, fetch, ev)
	if err == nil {
		t.Fatal("VerifyInclusionEvidence with a corrupted proof hash returned nil error")
	}
	if !errors.Is(err, ErrInclusionMismatch) {
		t.Errorf("error %v does not wrap ErrInclusionMismatch", err)
	}
}

// TestVerifyInclusionEvidenceWrongLeafIndex keeps a valid proof for leaf 5 but claims
// it proves leaf 6 — the monitor recomputes leaf 6's proof, which differs, so the
// cross-check fails with ErrInclusionMismatch.
func TestVerifyInclusionEvidenceWrongLeafIndex(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	fetch := tileFetcherFor(t, tree, treeLeaves)

	ev := hubEvidenceFor(t, tree, 5, treeLeaves)
	ev.LeafIndex = 6 // valid-but-wrong: proof bytes are still leaf 5's

	err := VerifyInclusionEvidence(ctx, fetch, ev)
	if err == nil {
		t.Fatal("VerifyInclusionEvidence with a wrong leaf index returned nil error")
	}
	if !errors.Is(err, ErrInclusionMismatch) {
		t.Errorf("error %v does not wrap ErrInclusionMismatch", err)
	}
}

// TestVerifyInclusionEvidenceOutOfRange asserts leafIndex >= treeSize returns a
// wrapped non-nil error without reaching the fetcher.
func TestVerifyInclusionEvidenceOutOfRange(t *testing.T) {
	ctx := context.Background()
	fetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		t.Fatal("out-of-range leafIndex must not fetch any tile")
		return nil, nil
	}
	ev := InclusionEvidence{Type: "IsccLogInclusionProof", TreeSize: treeLeaves, LeafIndex: treeLeaves}
	if err := VerifyInclusionEvidence(ctx, fetch, ev); err == nil {
		t.Fatal("VerifyInclusionEvidence(leafIndex == treeSize) returned nil error")
	}
}

// TestVerifyInclusionEvidenceMissingTile asserts a genuine tile-fetch fault surfaces
// as a wrapped Go error preserving os.ErrNotExist, distinct from ErrInclusionMismatch.
func TestVerifyInclusionEvidenceMissingTile(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	ev := hubEvidenceFor(t, tree, 5, treeLeaves)
	fetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		return nil, fmt.Errorf("read tile: %w", os.ErrNotExist)
	}
	err := VerifyInclusionEvidence(ctx, fetch, ev)
	if err == nil {
		t.Fatal("VerifyInclusionEvidence with a missing tile returned nil error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error %v does not wrap os.ErrNotExist", err)
	}
	if errors.Is(err, ErrInclusionMismatch) {
		t.Errorf("a tile fault must not be reported as ErrInclusionMismatch: %v", err)
	}
}

// TestParseInclusionEvidence round-trips a valid evidence JSON and rejects a wrong
// type and a zero tree size.
func TestParseInclusionEvidence(t *testing.T) {
	valid := []byte(`{
		"type": "IsccLogInclusionProof",
		"checkpoint": "sb0.iscc.id/log\n300\nrootb64\n",
		"treeSize": 300,
		"leafIndex": 5,
		"inclusionProof": ["AAA=", "/w8="]
	}`)
	ev, err := ParseInclusionEvidence(valid)
	if err != nil {
		t.Fatalf("ParseInclusionEvidence(valid) = %v, want nil", err)
	}
	if ev.Type != "IsccLogInclusionProof" || ev.TreeSize != 300 || ev.LeafIndex != 5 {
		t.Errorf("ParseInclusionEvidence(valid) = %+v, fields not round-tripped", ev)
	}
	if len(ev.InclusionProof) != 2 || ev.InclusionProof[0] != "AAA=" {
		t.Errorf("ParseInclusionEvidence(valid).InclusionProof = %v, not round-tripped", ev.InclusionProof)
	}

	wrongType := []byte(`{"type": "SomethingElse", "treeSize": 1, "leafIndex": 0, "inclusionProof": []}`)
	if _, err := ParseInclusionEvidence(wrongType); err == nil {
		t.Error("ParseInclusionEvidence(wrong type) returned nil error")
	}

	zeroSize := []byte(`{"type": "IsccLogInclusionProof", "treeSize": 0, "leafIndex": 0, "inclusionProof": []}`)
	if _, err := ParseInclusionEvidence(zeroSize); err == nil {
		t.Error("ParseInclusionEvidence(treeSize == 0) returned nil error")
	}

	badJSON := []byte(`{not json`)
	if _, err := ParseInclusionEvidence(badJSON); err == nil {
		t.Error("ParseInclusionEvidence(bad JSON) returned nil error")
	}
}
