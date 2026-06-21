// Conformance test closing M2's second Verify criterion: "computed inclusion proof
// matches the hub's evidence.IsccLogInclusionProof for sampled iscc_id's". It drives a
// verified PollHub over the in-process byte-accurate 300-leaf mirror, resolves a
// sampled leaf's iscc_id -> leafIndex via the production iscc_index reader
// (SeqsForISCCID), builds that leaf's hub-side IsccLogInclusionProof from the fixture
// tree (the hub: testonly.Tree owns the real RFC-6962 inclusion proof), and asserts
// logclient.VerifyInclusionEvidence — recomputing the proof from the mirrored
// SQLiteFetcher tiles — byte-matches it. This re-arms the inclusion cross-check oracle
// gate on the real verified path.
//
// Why this is test-only (a deliberate, honest choice). VerifyInclusionEvidence is the
// CONSUMER of a hub-supplied proof; M2's Verify bar is a conformance assertion the
// plan says runs "as ordinary go test … once the package exists". The package exists
// and is built-but-unwired; the follower already mirrors the tiles (ingestTiles) and
// indexes the leaves (projectEntryBundle) — everything the cross-check reads. There is
// no inbound hub-evidence transport on the follow path yet (no FetchInclusionEvidence;
// proof-serving / verify-for-me is a later M2/M3 slice). A PollHub step that recomputed
// the monitor's OWN proof and checked it against itself would be circular and is
// forbidden (target.md: a green-but-wrong verify must not ship), so no production line
// is added. The missing piece this file supplies is the conformance test that drives
// the cross-check over a real verified-poll mirror.
//
// Non-circularity: three independent paths meet here — m.tree.InclusionProof (the
// prover, playing the hub), InclusionProofFromTiles inside VerifyInclusionEvidence (the
// monitor recompute over the mirror written by ingestTiles), and the base64 round-trip.
// The wrong-leaf and corrupted-proof negatives make it non-vacuous: a verify that
// ignored the proof bytes would still pass them.
package follower

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
)

// encodeProof base64-Std-encodes each RFC-6962 sibling hash into the []string shape a
// hub's IsccLogInclusionProof carries (matching iscc_hub/log_tree.py's
// inclusion_evidence, which base64-encodes the merkle proof hashes).
func encodeProof(proof [][]byte) []string {
	out := make([]string, len(proof))
	for i, h := range proof {
		out[i] = base64.StdEncoding.EncodeToString(h)
	}
	return out
}

// TestPollHubInclusion drives a verified PollHub over the 300-leaf mirror and proves
// the monitor's tile-recomputed inclusion proof matches the hub's tree-built
// IsccLogInclusionProof for sampled leaves, then rejects a wrong-leaf and a
// corrupted-proof evidence with ErrInclusionMismatch.
func TestPollHubInclusion(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// A verified poll mirrors the tiles (so SQLiteFetcher.ReadTile works) and folds the
	// iscc_index projection (so SeqsForISCCID works). The mirror is the same in-process
	// byte-accurate fixture the fsck/projection tests poll.
	m := buildVerifiedMirror(t, mirrorLeaves)
	status, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified", status)
	}

	f := store.SQLiteFetcher{Store: s, HubID: hubID}

	// One leaf in bundle 0 and one past the 256-leaf boundary, so the cross-check
	// exercises both the full level-0 tile and the 44-leaf partial at index 1.
	for _, leafIndex := range []int{5, 260} {
		// Resolve iscc_id -> leafIndex via the production index path. Each leaf carries
		// a distinct id, so this returns exactly [leafIndex].
		seqs, err := s.SeqsForISCCID(ctx, hubID, leafISCCID(leafIndex))
		if err != nil {
			t.Fatalf("SeqsForISCCID(leaf %d): %v", leafIndex, err)
		}
		if len(seqs) != 1 || seqs[0] != uint64(leafIndex) {
			t.Fatalf("SeqsForISCCID(leaf %d) = %v, want [%d]", leafIndex, seqs, leafIndex)
		}

		// The hub side: the tree owns the real RFC-6962 inclusion proof for this leaf.
		proof, err := m.tree.InclusionProof(uint64(leafIndex), m.size)
		if err != nil {
			t.Fatalf("tree.InclusionProof(%d, %d): %v", leafIndex, m.size, err)
		}
		ev := logclient.InclusionEvidence{
			Type:           "IsccLogInclusionProof",
			Checkpoint:     string(m.checkpoint),
			TreeSize:       m.size,
			LeafIndex:      uint64(leafIndex),
			InclusionProof: encodeProof(proof),
		}

		// The monitor recomputes the proof from the mirrored tiles and must match.
		if err := logclient.VerifyInclusionEvidence(ctx, f.ReadTile, ev); err != nil {
			t.Errorf("VerifyInclusionEvidence(leaf %d) = %v, want nil (monitor proof must match the hub's)", leafIndex, err)
		}
	}

	// Negative 1 — wrong leaf. A VALID proof for leaf 5 re-labelled leaf 6 still fails:
	// the monitor recomputes leaf 6's (different) proof from the mirror. This is the
	// sharp negative — it would pass a verify that ignored the proof bytes.
	proof5, err := m.tree.InclusionProof(5, m.size)
	if err != nil {
		t.Fatalf("tree.InclusionProof(5, %d): %v", m.size, err)
	}
	wrongLeaf := logclient.InclusionEvidence{
		Type:           "IsccLogInclusionProof",
		Checkpoint:     string(m.checkpoint),
		TreeSize:       m.size,
		LeafIndex:      6,
		InclusionProof: encodeProof(proof5),
	}
	if err := logclient.VerifyInclusionEvidence(ctx, f.ReadTile, wrongLeaf); !errors.Is(err, logclient.ErrInclusionMismatch) {
		t.Errorf("VerifyInclusionEvidence(leaf 5 proof labelled leaf 6) = %v, want ErrInclusionMismatch", err)
	}

	// Negative 2 — corrupted proof. Flip one byte of one decoded hash and re-encode;
	// the monitor's recomputed proof no longer byte-equals the hub's.
	corrupt := make([][]byte, len(proof5))
	for i, h := range proof5 {
		cp := make([]byte, len(h))
		copy(cp, h)
		corrupt[i] = cp
	}
	corrupt[0][0] ^= 0xff
	corruptedProof := logclient.InclusionEvidence{
		Type:           "IsccLogInclusionProof",
		Checkpoint:     string(m.checkpoint),
		TreeSize:       m.size,
		LeafIndex:      5,
		InclusionProof: encodeProof(corrupt),
	}
	if err := logclient.VerifyInclusionEvidence(ctx, f.ReadTile, corruptedProof); !errors.Is(err, logclient.ErrInclusionMismatch) {
		t.Errorf("VerifyInclusionEvidence(corrupted leaf 5 proof) = %v, want ErrInclusionMismatch", err)
	}
}
