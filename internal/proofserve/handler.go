// Package proofserve serves RFC-6962 inclusion and consistency proofs over
// net/http, computed from one hub's mirrored hash tiles via the local
// store.SQLiteFetcher and logclient.{Inclusion,Consistency}ProofFromTiles — never
// re-hitting the hub. It is the proof-computing half of M2's Verify bar (the
// static mirror in tilesserve is the other half): a client fetches GET
// /inclusion?iscc_id=<id> and checks the returned proof against the hub's own
// IsccLogInclusionProof, or GET /consistency?from=<n> and checks its own prior
// (size, root) against the monitor's mirrored tree.
//
// The served inclusion proof is byte-compatible with the hub's evidence member: it
// reuses logclient.InclusionEvidence's field names and the base64-Std proof
// encoding (matching iscc_hub/log_tree.py inclusion_evidence), so the JSON feeds
// straight into logclient.VerifyInclusionEvidence. The consistency proof has no
// hub-served counterpart (iscc-log §10.2: it is verifier-computed, not served by
// the hub), so this package defines its own response shape. Both proofs are
// computed against the monitor's accepted tree size (store.FollowState.LastSize),
// the tree the monitor vouches for, not a size re-parsed from the raw checkpoint.
//
// It is a separate leaf package precisely so net/http stays out of the store and
// logclient closures: proofserve depends on both, never the reverse. The proof
// computation itself is RFC-6962 crypto in logclient; this package only resolves
// the leaf seq (schema-agnostically, ADR-0008) or the prior size, and maps faults
// to HTTP status.
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

// Handler returns an http.Handler that serves one hub's computed inclusion and
// consistency proofs from the local mirror. It handles GET /inclusion?iscc_id=<id>
// (optionally &index=<n>) — resolving the leaf seq for the iscc_id, building the
// RFC-6962 inclusion proof from the hub's mirrored tiles against the monitor's
// accepted tree size, and returning it as JSON shaped like the hub's
// IsccLogInclusionProof — and GET /consistency?from=<n> — building the RFC-6962
// consistency proof relating the prior root at size from to the accepted root at
// LastSize, returning it as JSON.
//
// Status mapping: non-GET → 405; an unmatched path → 404. The per-route flow
// owns the rest (serveInclusion / serveConsistency); see each for its 400/404/500
// mapping. CORS, caching, and conditional GET are intentionally out of scope for
// this slice.
func Handler(st *store.Store, hubID int64) http.Handler {
	f := store.SQLiteFetcher{Store: st, HubID: hubID}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// THIS handler is mounted at the hub-log root, so it sees the path suffix
		// of the hub's /log origin; the proof routes are /inclusion and /consistency.
		switch r.URL.Path {
		case "/inclusion":
			serveInclusion(w, r, st, f, hubID)
		case "/consistency":
			serveConsistency(w, r, st, f, hubID)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
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

// ConsistencyEvidence is the JSON response shape for a served consistency proof.
// Unlike inclusion there is no hub-defined IsccLogConsistencyProof member to
// mirror (iscc-log §10.2: the consistency proof is verifier-computed, not served
// by the hub), so this is a small local shape: Type tags it, FirstSize is the
// client's prior trusted tree size (the from query param), SecondSize the
// monitor's accepted tree size (LastSize), and ConsistencyProof the RFC-6962 proof
// hashes, each base64-Std encoded (matching the inclusion encoding and the iscc_hub
// convention). A client verifies it via proof.VerifyConsistency(hasher, FirstSize,
// SecondSize, decoded, priorRoot, acceptedRoot).
type ConsistencyEvidence struct {
	Type             string   `json:"type"`
	FirstSize        uint64   `json:"firstSize"`
	SecondSize       uint64   `json:"secondSize"`
	ConsistencyProof []string `json:"consistencyProof"`
}

// serveConsistency builds the RFC-6962 consistency proof relating the prior root
// at size from to the monitor's accepted root at LastSize, sourced entirely from
// the hub's mirrored tiles, and writes the JSON evidence. It owns the full request
// flow and status mapping for the /consistency route.
//
// Status mapping: a missing or non-numeric from → 400; no accepted checkpoint yet
// (LastSize == 0) → 404; from > LastSize (a future/over-large prior — RFC-6962
// requires M ≤ N) → 400; from has no recorded checkpoint row → 404; a tile not yet
// mirrored (a wrapped os.ErrNotExist) → 404; any other read/build error → 500. The
// degenerate from == 0 and from == LastSize cases yield an empty (nil) proof
// without touching the fetcher — a valid degenerate consistency proof — and are
// served as a 200 with an empty consistencyProof array, never a 400.
func serveConsistency(w http.ResponseWriter, r *http.Request, st *store.Store, f store.SQLiteFetcher, hubID int64) {
	ctx := r.Context()

	from, err := parseUint(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "missing or non-numeric from", http.StatusBadRequest)
		return
	}

	// LastSize is the latest accepted tree size — the tree the monitor vouches
	// for and the larger size of the proof. A hub with no accepted checkpoint yet
	// has nothing to relate the prior root to.
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

	// RFC-6962 consistency relates a smaller size to a larger one (M ≤ N); a from
	// past the accepted tree is a 400, never a 500 from proof.Consistency.
	if from > size {
		http.Error(w, "from exceeds accepted tree size", http.StatusBadRequest)
		return
	}

	// The prior root must be one the monitor actually recorded a checkpoint for,
	// so the client's prior (size, root) is checkable against a known accepted
	// root. An unrecorded from is a 404. CheckpointAt(size) also confirms the
	// accepted-size row exists for the degenerate from == size roundtrip. from == 0
	// is the empty-tree prior — there is no checkpoint at size 0 by construction —
	// so it skips the row requirement and serves the empty degenerate proof.
	if from > 0 {
		if _, _, found, err := st.CheckpointAt(ctx, hubID, from); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		} else if !found {
			http.Error(w, "no checkpoint at from", http.StatusNotFound)
			return
		}
	}

	proof, err := logclient.ConsistencyProofFromTiles(ctx, f.ReadTile, from, size)
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

	writeConsistency(w, from, size, proof)
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

// writeConsistency writes the JSON ConsistencyEvidence body for a served
// consistency proof. Each proof hash is base64-Std encoded (matching the inclusion
// encoding and the iscc_hub convention); a nil/empty proof (the degenerate from ==
// 0 or from == LastSize case) serializes as an empty consistencyProof array, a
// valid degenerate proof. It keeps writeEvidence's drop-the-write-error-after-200
// posture: a marshal of a fixed-shape struct of strings/uints cannot fail for
// content reasons, and a mid-write fault cannot un-send the 200.
func writeConsistency(w http.ResponseWriter, from, size uint64, proof [][]byte) {
	encoded := make([]string, len(proof))
	for i, h := range proof {
		encoded[i] = base64.StdEncoding.EncodeToString(h)
	}
	ev := ConsistencyEvidence{
		Type:             "IsccLogConsistencyProof",
		FirstSize:        from,
		SecondSize:       size,
		ConsistencyProof: encoded,
	}
	w.Header().Set("Content-Type", contentType)
	_ = encodeJSON(w, ev)
}

// encodeJSON marshals v to w. It keeps the single drop-the-write-error site tidy.
func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
