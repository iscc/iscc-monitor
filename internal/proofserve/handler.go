// Package proofserve serves RFC-6962 inclusion proofs over net/http, computed
// from one hub's mirrored hash tiles via the local store.SQLiteFetcher and
// logclient.InclusionProofFromTiles — never re-hitting the hub. It is the
// proof-computing half of M2's Verify bar (the static mirror in tilesserve is
// the other half): a client fetches GET /inclusion?iscc_id=<id> and checks the
// returned proof against the hub's own IsccLogInclusionProof.
//
// The served proof is byte-compatible with the hub's evidence member: it reuses
// logclient.InclusionEvidence's field names and the base64-Std proof encoding
// (matching iscc_hub/log_tree.py inclusion_evidence), so the JSON feeds straight
// into logclient.VerifyInclusionEvidence. The proof is computed against the
// monitor's accepted tree size (store.FollowState.LastSize), the tree the monitor
// vouches for, not a size re-parsed from the raw checkpoint.
//
// It is a separate leaf package precisely so net/http stays out of the store and
// logclient closures: proofserve depends on both, never the reverse. The proof
// computation itself is RFC-6962 crypto in logclient; this package only resolves
// the leaf seq (schema-agnostically, ADR-0008) and maps faults to HTTP status.
package proofserve

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
)

// contentType is the media type for the JSON proof response body.
const contentType = "application/json"

// Handler returns an http.Handler that serves one hub's computed inclusion
// proofs from the local mirror. It handles GET /inclusion?iscc_id=<id>
// (optionally &index=<n>): it resolves the leaf seq for the iscc_id, builds the
// RFC-6962 inclusion proof from the hub's mirrored tiles against the monitor's
// accepted tree size, and returns it as JSON shaped like the hub's
// IsccLogInclusionProof.
//
// Status mapping: non-GET → 405; an unmatched path → 404; a missing iscc_id
// param → 400; an index param that is not one of the iscc_id's committed seqs →
// 400; no accepted checkpoint yet (LastSize == 0), an iscc_id not in the index,
// or a leaf the accepted tree does not yet cover → 404; a tile not yet mirrored
// (a wrapped os.ErrNotExist) → 404; any other read/build error → 500. CORS,
// caching, and conditional GET are intentionally out of scope for this slice.
func Handler(st *store.Store, hubID int64) http.Handler {
	f := store.SQLiteFetcher{Store: st, HubID: hubID}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// THIS handler is mounted at the hub-log root, so it sees the path suffix
		// of the hub's /log origin; the one proof route is /inclusion.
		if r.URL.Path != "/inclusion" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		serveInclusion(w, r, st, f, hubID)
	})
}

// serveInclusion resolves the iscc_id (and optional index) to a leaf seq, builds
// the inclusion proof from the mirror against the accepted tree size, and writes
// the JSON evidence. It owns the full request flow and status mapping for the
// /inclusion route.
func serveInclusion(w http.ResponseWriter, r *http.Request, st *store.Store, f store.SQLiteFetcher, hubID int64) {
	ctx := r.Context()

	isccID := r.URL.Query().Get("iscc_id")
	if isccID == "" {
		http.Error(w, "missing iscc_id", http.StatusBadRequest)
		return
	}

	// LastSize is the latest accepted tree size — the tree the monitor vouches
	// for. A hub with no accepted checkpoint yet has nothing to prove against.
	fs, err := st.FollowState(ctx, hubID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	size := fs.LastSize
	if size == 0 {
		http.Error(w, "no accepted checkpoint", http.StatusNotFound)
		return
	}

	// iscc_id → seq is one-to-many and schema-agnostic (ADR-0008): a declaration,
	// its deletion, and any other note type sharing the id all come back. We do
	// not interpret the ISCC-ID, only index by seq.
	seqs, err := st.SeqsForISCCID(ctx, hubID, isccID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if len(seqs) == 0 {
		http.Error(w, "iscc_id not found", http.StatusNotFound)
		return
	}

	leafIndex, ok := selectSeq(seqs, r.URL.Query().Get("index"))
	if !ok {
		http.Error(w, "index not committed under iscc_id", http.StatusBadRequest)
		return
	}

	// The seq was committed into some accepted tree, but a stale/racing size
	// could make it fall outside the current accepted tree — guard before
	// building so an out-of-range leaf is a 404, never a 500/panic.
	if leafIndex >= size {
		http.Error(w, "leaf not covered by accepted checkpoint", http.StatusNotFound)
		return
	}

	proof, err := logclient.InclusionProofFromTiles(ctx, f.ReadTile, leafIndex, size)
	if err != nil {
		// A tile not yet mirrored surfaces as a wrapped os.ErrNotExist (the
		// SQLiteFetcher contract) — a 404, not a 500.
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "tile not mirrored", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeEvidence(w, size, leafIndex, proof)
}

// selectSeq picks the leaf seq to prove. When the index query param is empty it
// defaults to seqs[0] — the first committed seq is the deterministic default
// because iscc_id → seq is one-to-many (ADR-0008) and the seqs are ascending.
// When index is present it must equal one of seqs (never silently prove a seq
// the id does not commit); a non-numeric or absent index returns ok=false.
func selectSeq(seqs []uint64, index string) (uint64, bool) {
	if index == "" {
		return seqs[0], true
	}
	want, err := parseUint(index)
	if err != nil {
		return 0, false
	}
	for _, s := range seqs {
		if s == want {
			return s, true
		}
	}
	return 0, false
}

// parseUint parses a base-10 unsigned index, rejecting any non-digit input so a
// malformed index becomes a 400 rather than a silent default.
func parseUint(s string) (uint64, error) {
	var n uint64
	if s == "" {
		return 0, errors.New("empty index")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("non-numeric index")
		}
		n = n*10 + uint64(c-'0')
	}
	return n, nil
}

// writeEvidence writes the JSON proof body shaped like the hub's
// IsccLogInclusionProof (logclient.InclusionEvidence field names), so it feeds
// directly into logclient.VerifyInclusionEvidence. Each proof hash is base64-Std
// encoded (matching iscc_hub inclusion_evidence). The checkpoint field is omitted
// for this slice — the client refetches /checkpoint from the mirror — so only the
// proof hashes, treeSize, and leafIndex are served.
func writeEvidence(w http.ResponseWriter, size, leafIndex uint64, proof [][]byte) {
	encoded := make([]string, len(proof))
	for i, h := range proof {
		encoded[i] = base64.StdEncoding.EncodeToString(h)
	}
	ev := logclient.InclusionEvidence{
		Type:           "IsccLogInclusionProof",
		TreeSize:       size,
		LeafIndex:      leafIndex,
		InclusionProof: encoded,
	}
	w.Header().Set("Content-Type", contentType)
	// The 200 is sent on the first write; a mid-write encode error cannot un-send
	// it, and the only failure mode after a successful header write is a broken
	// client connection, so it is dropped deliberately rather than writing a
	// misleading second status (matching tilesserve / metricshttp). Not a gate
	// dodge — the marshal of a fixed-shape struct of strings/uints cannot fail
	// for content reasons.
	_ = encodeJSON(w, ev)
}

// encodeJSON marshals v to w. It keeps writeEvidence's single
// drop-the-write-error site tidy.
func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
