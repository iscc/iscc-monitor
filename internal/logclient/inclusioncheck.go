// This file cross-checks a hub's own IsccLogInclusionProof against the inclusion
// proof the monitor recomputes from its mirrored hash tiles — the second external
// oracle (the hub-computed proof) in M2's Verify bar. It decodes the hub's
// evidence member (ISCC-Log §10.1: {type, checkpoint, treeSize, leafIndex,
// inclusionProof}, the proof base64-Std encoded as iscc_hub/log_tree.py's
// inclusion_evidence emits) and asserts InclusionProofFromTiles byte-equals it for
// the same leaf. It is the first production-shaped caller of InclusionProofFromTiles.
//
// Scope discipline: this cross-checks the proof *hashes* only — it does NOT
// re-verify the embedded checkpoint signature or its tree size/root (that is
// AcceptCheckpoint's job), and it does NOT resolve a leaf index from iscc_index or
// wire into PollHub (later slices). treeSize/leafIndex are read from the JSON but
// the bundled checkpoint string is left untouched.
//
// Purity (Correctness rule: proof/verify is pure; keep WASM-shareable): this file
// imports only stdlib (bytes, context, encoding/base64, encoding/json, errors, fmt)
// plus the in-package InclusionProofFromTiles. It does not pull net / net/http /
// database/sql / os — a missing tile's os.ErrNotExist rides through the %w-wrap from
// InclusionProofFromTiles, so errors.Is survives without referencing the sentinel.
package logclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// inclusionEvidenceType is the VC evidence member type a hub emits for an
// inclusion proof (iscc_hub schema.py Evidence.type Literal).
const inclusionEvidenceType = "IsccLogInclusionProof"

// ErrInclusionMismatch marks a hub's IsccLogInclusionProof whose proof hashes do
// not byte-equal the proof the monitor recomputes from its mirrored tiles for the
// same leaf. It is a sentinel so a future PollHub caller can errors.Is on it,
// distinct from a transport/tile fault (which preserves os.ErrNotExist).
var ErrInclusionMismatch = errors.New("hub inclusion proof does not match proof computed from tiles")

// InclusionEvidence mirrors the hub's IsccLogInclusionProof VC evidence member
// (iscc_hub/log_tree.py inclusion_evidence): the verbatim signed checkpoint, the
// tree size it commits, the zero-based leaf index, and the RFC-6962 inclusion proof
// as base64-Std-encoded sibling hashes.
type InclusionEvidence struct {
	Type           string   `json:"type"`
	Checkpoint     string   `json:"checkpoint"`
	TreeSize       uint64   `json:"treeSize"`
	LeafIndex      uint64   `json:"leafIndex"`
	InclusionProof []string `json:"inclusionProof"`
}

// ParseInclusionEvidence decodes a hub's IsccLogInclusionProof JSON into an
// InclusionEvidence, rejecting a wrong type or a zero tree size (the schema's
// treeSize ge=1 floor). It is a thin json.Unmarshal in the ParseDIDDocument style;
// the proof hashes are decoded later, by VerifyInclusionEvidence.
func ParseInclusionEvidence(raw []byte) (InclusionEvidence, error) {
	var ev InclusionEvidence
	if err := json.Unmarshal(raw, &ev); err != nil {
		return InclusionEvidence{}, fmt.Errorf("ParseInclusionEvidence: invalid JSON: %w", err)
	}
	if ev.Type != inclusionEvidenceType {
		return InclusionEvidence{}, fmt.Errorf("ParseInclusionEvidence: type %q is not %q", ev.Type, inclusionEvidenceType)
	}
	if ev.TreeSize == 0 {
		return InclusionEvidence{}, fmt.Errorf("ParseInclusionEvidence: treeSize must be >= 1")
	}
	return ev, nil
}

// VerifyInclusionEvidence asserts the hub-supplied inclusion proof in ev byte-equals
// the proof the monitor recomputes for the same leaf from its mirrored tiles via
// fetch. It is the monitor-vs-hub inclusion cross-check at the heart of M2's Verify
// bar; on a full match it returns nil, on a hash mismatch it returns a wrapped
// ErrInclusionMismatch, and a tile-fetch fault is propagated so errors.Is(err,
// os.ErrNotExist) survives.
//
// It does not re-verify the embedded checkpoint signature, tree size, or root — that
// is AcceptCheckpoint's job; this compares the proof node lists only, which is
// exactly what the Verify bar asks ("computed inclusion proof matches the hub's
// evidence.IsccLogInclusionProof").
func VerifyInclusionEvidence(ctx context.Context, fetch TileFetcher, ev InclusionEvidence) error {
	if ev.LeafIndex >= ev.TreeSize {
		return fmt.Errorf("VerifyInclusionEvidence: leafIndex %d out of range for treeSize %d", ev.LeafIndex, ev.TreeSize)
	}

	got, err := InclusionProofFromTiles(ctx, fetch, ev.LeafIndex, ev.TreeSize)
	if err != nil {
		return fmt.Errorf("VerifyInclusionEvidence: compute proof from tiles: %w", err)
	}

	want := make([][]byte, len(ev.InclusionProof))
	for i, enc := range ev.InclusionProof {
		h, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			return fmt.Errorf("VerifyInclusionEvidence: decode inclusionProof[%d]: %w", i, err)
		}
		want[i] = h
	}

	if len(got) != len(want) {
		return fmt.Errorf("VerifyInclusionEvidence: leaf %d: %d proof hashes from tiles, hub supplied %d: %w", ev.LeafIndex, len(got), len(want), ErrInclusionMismatch)
	}
	for i := range got {
		if !bytes.Equal(got[i], want[i]) {
			return fmt.Errorf("VerifyInclusionEvidence: leaf %d: proof hash %d differs from hub: %w", ev.LeafIndex, i, ErrInclusionMismatch)
		}
	}
	return nil
}
