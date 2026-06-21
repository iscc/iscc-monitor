// Package proofserve serves RFC-6962 inclusion and consistency proofs plus single
// leaf record bytes over net/http, computed from one hub's mirrored hash tiles and
// entry bundles via the local store.SQLiteFetcher and
// logclient.{Inclusion,Consistency}ProofFromTiles / RecordBytesFromBundle — never
// re-hitting the hub. It is the proof-computing half of M2's Verify bar (the
// static mirror in tilesserve is the other half): a client fetches GET
// /inclusion?iscc_id=<id> and checks the returned proof against the hub's own
// IsccLogInclusionProof, GET /consistency?from=<n> and checks its own prior
// (size, root) against the monitor's mirrored tree, or GET /entries?index=<seq> to
// pull the one leaf's raw record bytes the proof bundle commits to.
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
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"math"
	"net/http"
	"os"

	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"

	"github.com/iscc/iscc-monitor/internal/badge"
	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// contentType is the media type for the JSON proof response body.
const contentType = "application/json"

// browserSource is the embedded HTML log-browser template, parsed once at package
// init so a malformed template fails the build, not a request.
//
//go:embed browser.html
var browserSource string

// browserTmpl is the parsed log-browser template with the HubStatusBadge partial
// associated into the same set, so the page invokes {{template "hubStatusBadge" .}}
// over the browserData view (which exposes .Status and .Label). template.Must panics
// at init if either source fails to parse, surfacing a template bug at startup. It is
// html/template (NOT text/template) so the base64 root and status strings
// auto-escape.
var browserTmpl = func() *template.Template {
	t := template.Must(template.New("browser").Parse(browserSource))
	return template.Must(t.Parse(badge.Source))
}()

// recordsSource is the embedded HTML record-list template, parsed once at package
// init so a malformed template fails the build, not a request.
//
//go:embed records.html
var recordsSource string

// recordsTmpl is the parsed record-list template with the HubStatusBadge partial
// associated into the same set, so the page invokes {{template "hubStatusBadge" .}}
// over the recordsData view (which exposes .Status and .Label), the same wiring
// browserTmpl uses. template.Must panics at init if either source fails to parse. It
// is html/template (NOT text/template) so the id / schema strings auto-escape.
var recordsTmpl = func() *template.Template {
	t := template.Must(template.New("records").Parse(recordsSource))
	return template.Must(t.Parse(badge.Source))
}()

// recordSource is the embedded HTML single-record template, parsed once at package
// init so a malformed template fails the build, not a request.
//
//go:embed record.html
var recordSource string

// recordTmpl is the parsed single-record template with the HubStatusBadge partial
// associated into the same set, so the page invokes {{template "hubStatusBadge" .}}
// over the recordData view (which exposes .Status and .Label), the same wiring
// recordsTmpl / browserTmpl use. template.Must panics at init if either source fails
// to parse. It is html/template (NOT text/template) so the id / schema / raw record
// bytes auto-escape.
var recordTmpl = func() *template.Template {
	t := template.Must(template.New("record").Parse(recordSource))
	return template.Must(t.Parse(badge.Source))
}()

// Record-kind labels mapping the verbatim note.$schema to a human-readable kind.
// This is the ONLY interpretation the single-record page performs (ADR-0008): the
// declaration / deletion schemas (CLAUDE.md glossary) map to friendly labels, and
// anything else — including an empty schema — is the catch-all kindUnknown. The
// mapping is fail-open: an unknown/empty schema still renders the page (never a 4xx/
// 5xx), and the verbatim schema string is always shown alongside the label.
const (
	schemaDeclaration = "iscc-note-0.8.0"
	schemaDeletion    = "iscc-note-delete-0.8.0"
	kindDeclaration   = "Declaration"
	kindDeletion      = "Deletion"
	kindUnknown       = "Unknown record type"
)

// defaultPageSize is the record-list page size used when n is absent or non-numeric;
// maxPageSize clamps a hostile n so it can never scan the whole index.
const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// octetStreamType is the media type for the raw record bytes /entries serves: the
// JCS-canonical log-entry envelope is an opaque BLOB, not the JSON proof shape.
const octetStreamType = "application/octet-stream"

// StatusSource reports a hub's current in-memory glossary status by hub_id. It is
// the read seam the log browser uses to overlay the live poll verdict (the richer
// unresolvable / unverified states the store cannot prove) onto the store-provable
// subset (frozen / verified). ok is false when no live status is recorded for the
// hub. *metrics.Registry satisfies it structurally via its Status method; proofserve
// takes the interface, not the concrete package, so it never imports internal/metrics
// (mirroring dashboard.StatusSource).
type StatusSource interface {
	Status(hubID int64) (string, bool)
}

// Handler returns an http.Handler that serves one hub's computed inclusion and
// consistency proofs from the local mirror. It handles GET /inclusion?iscc_id=<id>
// (optionally &index=<n>) — resolving the leaf seq for the iscc_id, building the
// RFC-6962 inclusion proof from the hub's mirrored tiles against the monitor's
// accepted tree size, and returning it as JSON shaped like the hub's
// IsccLogInclusionProof — and GET /consistency?from=<n> — building the RFC-6962
// consistency proof relating the prior root at size from to the accepted root at
// LastSize, returning it as JSON.
//
// It also handles GET /entries?index=<seq> — extracting the raw record bytes of a
// single accepted leaf from the hub's mirrored entry bundles and serving them
// verbatim as application/octet-stream.
//
// It handles GET /verify?iscc_id=<id> — the weaker verify-for-me path that
// returns a single self-contained JSON verdict (the caller trusts the verdict
// rather than verifying a proof bundle itself): the hub's persisted status, the
// accepted checkpoint (size, root), and a real RFC-6962 inclusion result recomputed
// from the mirror and Merkle-verified against the accepted root.
//
// It also handles GET / (the hub-log root) — a server-rendered HTML log browser
// exposing the monitor's accepted checkpoint (size, root) for this hub plus
// relative links into the entries and proof routes, so a human can browse the
// mirror and a client can discover the proof surface.
//
// It also handles GET /records (optionally ?from=<seq>&n=<size>) — a no-JS,
// newest-first, plain-link-paginated HTML list of the hub's indexed records, each
// row linking to that leaf's single-record page (record?index=<seq>), with an
// informative 200 empty state for a hub with no indexed records.
//
// Finally it handles GET /record?index=<seq> — a no-JS HTML page rendering one
// accepted leaf: its seq, a kind label (declaration / deletion / unknown) derived
// from the verbatim note.$schema, the verbatim ISCC-ID and schema, and the raw
// record bytes the proof bundle commits to.
//
// Status mapping: non-GET → 405; an unmatched path → 404. The per-route flow
// owns the rest (serveBrowser / serveRecords / serveRecord / serveInclusion /
// serveConsistency / serveEntries / serveVerify); see each for its 400/404/500
// mapping. CORS, caching, and conditional GET are intentionally out of scope for
// this slice.
//
// statuses is the in-memory status overlay (the metrics registry) the log browser
// uses to render the richer unresolvable / unverified verdicts the store cannot
// prove; only serveBrowser consults it. A nil statuses is tolerated and simply
// leaves every store-verified hub showing "verified" (the proof routes ignore it).
func Handler(st *store.Store, hubID int64, statuses StatusSource) http.Handler {
	f := store.SQLiteFetcher{Store: st, HubID: hubID}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// THIS handler is mounted at the hub-log root, so it sees the path suffix
		// of the hub's /log origin; the proof routes are /inclusion and /consistency.
		switch r.URL.Path {
		case "/":
			serveBrowser(w, r, st, hubID, statuses)
		case "/records":
			serveRecords(w, r, st, hubID, statuses)
		case "/record":
			serveRecord(w, r, st, f, hubID, statuses)
		case "/inclusion":
			serveInclusion(w, r, st, f, hubID)
		case "/consistency":
			serveConsistency(w, r, st, f, hubID)
		case "/entries":
			serveEntries(w, r, st, f, hubID)
		case "/verify":
			serveVerify(w, r, st, f, hubID)
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

// serveEntries extracts the raw record bytes of the single accepted leaf at the
// absolute seq index from the hub's mirrored entry bundles and writes them verbatim
// as application/octet-stream. It owns the full request flow and status mapping for
// the /entries route, mirroring serveInclusion's accepted-tree guards. The index is
// the absolute leaf seq (schema-agnostic, ADR-0008): this route interprets nothing,
// it returns the leaf's bytes by index, never an ISCC-ID or note.$schema.
//
// Status mapping: a missing or non-numeric index → 400; no accepted checkpoint yet
// (LastSize == 0) or seq >= LastSize (the leaf is not in the monitor's accepted
// tree) → 404; the bundle not yet mirrored (a wrapped os.ErrNotExist) → 404; the
// bundle mirrored but only partial and not yet covering this leaf
// (ErrLeafOutOfBundle) → 404; any other read/decode error → 500.
func serveEntries(w http.ResponseWriter, r *http.Request, st *store.Store, f store.SQLiteFetcher, hubID int64) {
	ctx := r.Context()

	seq, err := parseUint(r.URL.Query().Get("index"))
	if err != nil {
		http.Error(w, "missing or non-numeric index", http.StatusBadRequest)
		return
	}

	// LastSize is the latest accepted tree size — the tree the monitor vouches
	// for. A hub with no accepted checkpoint yet has no leaves to serve.
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
	if seq >= size {
		http.Error(w, "leaf not covered by accepted checkpoint", http.StatusNotFound)
		return
	}

	// The leaf lives in entry bundle seq/256 at position seq%256 (the bundle is up
	// to 256 framed records). The bundle qualifier p is the partial leaf count this
	// bundle is expected to hold within the accepted tree (0 == full): the final
	// bundle of a non-multiple-of-256 tree is a partial, so requesting p == 0
	// unconditionally would miss it. The SQLiteFetcher does the partial→full fallback
	// when p > 0, so a partial that was later promoted to full still resolves.
	bundleIndex := seq / tiles.TileWidth
	offset := seq % tiles.TileWidth
	p := tiles.PartialTileSize(0, bundleIndex, size)
	bundle, err := f.ReadEntryBundle(ctx, bundleIndex, p)
	if err != nil {
		// An entry bundle not yet mirrored surfaces as a wrapped os.ErrNotExist (the
		// SQLiteFetcher contract) — a 404, not a 500.
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "bundle not mirrored", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	record, err := logclient.RecordBytesFromBundle(bundle, offset)
	if err != nil {
		// A mirrored-but-partial bundle that does not yet contain this leaf is a 404,
		// not a 500 — the leaf is accepted but the bundle BLOB has not caught up.
		if errors.Is(err, logclient.ErrLeafOutOfBundle) {
			http.Error(w, "leaf not in mirrored bundle", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeRecord(w, record)
}

// VerifyVerdict is the JSON response shape for the verify-for-me route: a single
// self-contained verdict for an iscc_id that the caller trusts rather than
// verifying itself. HubStatus is the glossary status the store can substantiate
// ("frozen" if the hub is frozen, else "verified" once it has an accepted
// checkpoint — only signature-verified checkpoints advance the accepted size,
// ADR-0006). TreeSize and Root are the accepted checkpoint the monitor vouches for
// (Root base64-Std encoded, matching the package's other proof encodings).
// LeafIndex / Included describe the recomputed RFC-6962 inclusion result for the
// resolved leaf, Merkle-verified against Root. Verified is the overall verdict —
// true only when the leaf resolved AND proof.VerifyInclusion accepted; Reason is
// empty on success and carries the non-verified cause otherwise.
type VerifyVerdict struct {
	IsccID    string `json:"iscc_id"`
	HubStatus string `json:"hub_status"`
	TreeSize  uint64 `json:"tree_size"`
	Root      string `json:"root"`
	LeafIndex uint64 `json:"leaf_index"`
	Included  bool   `json:"included"`
	Verified  bool   `json:"verified"`
	Reason    string `json:"reason"`
}

// serveVerify returns the verify-for-me JSON verdict for an iscc_id. Unlike the
// other proof routes, an id-shaped input fault is always a 200 verdict
// ({verified:false, reason:...}), never a 5xx — the verify-for-me contract yields a
// verdict for every id input. A non-200 is reserved for a genuine infra fault: a
// FollowState / CheckpointAt / SeqsForISCCID / bundle-read DB error → 500, and a
// non-os.ErrNotExist proof build/read error → 500. A tile or bundle the mirror has
// not caught up to (os.ErrNotExist / ErrLeafOutOfBundle) is a 200 verdict
// {verified:false, reason:"tile not mirrored"} — the leaf is accepted but not yet
// mirrored, a verdict, not a fault.
func serveVerify(w http.ResponseWriter, r *http.Request, st *store.Store, f store.SQLiteFetcher, hubID int64) {
	ctx := r.Context()

	isccID := r.URL.Query().Get("iscc_id")
	if isccID == "" {
		writeVerdict(w, VerifyVerdict{Verified: false, Reason: "missing iscc_id"})
		return
	}

	// LastSize is the accepted tree size — the tree the monitor vouches for; Frozen
	// is the only hub-status fact the store persists. A DB fault here is a genuine
	// infra fault, so it is a 500, not a verdict.
	fs, err := st.FollowState(ctx, hubID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	status := hubStatus(fs)
	size := fs.LastSize
	if size == 0 {
		writeVerdict(w, VerifyVerdict{IsccID: isccID, HubStatus: status, Verified: false, Reason: "no accepted checkpoint"})
		return
	}

	// The accepted root is the RFC-6962 tree head the monitor vouches for at the
	// accepted size; the inclusion result is Merkle-verified against it.
	root, _, found, err := st.CheckpointAt(ctx, hubID, size)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	rootB64 := base64.StdEncoding.EncodeToString(root)

	// iscc_id → seq is one-to-many and schema-agnostic (ADR-0008): default to the
	// first committed seq (seqs is ascending) and interpret nothing about the id.
	// verify-for-me takes no index param, so seqs[0] is the deterministic subject —
	// the empty-seqs case is guarded just below before any index access.
	seqs, err := st.SeqsForISCCID(ctx, hubID, isccID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if len(seqs) == 0 {
		writeVerdict(w, VerifyVerdict{IsccID: isccID, HubStatus: status, TreeSize: size, Root: rootB64, Verified: false, Reason: "iscc_id not found"})
		return
	}
	leafIndex := seqs[0]

	// A stale/racing size could leave a committed seq outside the accepted tree; a
	// leaf the accepted checkpoint does not cover is a non-verified verdict.
	if leafIndex >= size {
		writeVerdict(w, VerifyVerdict{IsccID: isccID, HubStatus: status, TreeSize: size, Root: rootB64, LeafIndex: leafIndex, Verified: false, Reason: "leaf not covered by accepted checkpoint"})
		return
	}

	// Read the leaf's raw record bytes from the mirrored entry bundle, the same way
	// serveEntries does: the final bundle of a non-multiple-of-256 tree is a partial,
	// so request its expected p (the SQLiteFetcher does the partial→full fallback).
	bundleIndex := leafIndex / tiles.TileWidth
	offset := leafIndex % tiles.TileWidth
	p := tiles.PartialTileSize(0, bundleIndex, size)
	bundle, err := f.ReadEntryBundle(ctx, bundleIndex, p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeVerdict(w, VerifyVerdict{IsccID: isccID, HubStatus: status, TreeSize: size, Root: rootB64, LeafIndex: leafIndex, Verified: false, Reason: "tile not mirrored"})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	record, err := logclient.RecordBytesFromBundle(bundle, offset)
	if err != nil {
		if errors.Is(err, logclient.ErrLeafOutOfBundle) {
			writeVerdict(w, VerifyVerdict{IsccID: isccID, HubStatus: status, TreeSize: size, Root: rootB64, LeafIndex: leafIndex, Verified: false, Reason: "tile not mirrored"})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Build the inclusion proof from the mirror and Merkle-verify it against the
	// accepted root — a REAL RFC-6962 check, not a stub.
	builtProof, err := logclient.InclusionProofFromTiles(ctx, f.ReadTile, leafIndex, size)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeVerdict(w, VerifyVerdict{IsccID: isccID, HubStatus: status, TreeSize: size, Root: rootB64, LeafIndex: leafIndex, Verified: false, Reason: "tile not mirrored"})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Arg-order gotcha (learnings): VerifyInclusion(hasher, index, size, leafHash,
	// proof, root) — leafHash precedes proof, unlike VerifyConsistency.
	leafHash := rfc6962.DefaultHasher.HashLeaf(record)
	included := proof.VerifyInclusion(rfc6962.DefaultHasher, leafIndex, size, leafHash, builtProof, root) == nil

	verdict := VerifyVerdict{
		IsccID:    isccID,
		HubStatus: status,
		TreeSize:  size,
		Root:      rootB64,
		LeafIndex: leafIndex,
		Included:  included,
		Verified:  included,
	}
	if !included {
		verdict.Reason = "inclusion proof did not verify"
	}
	writeVerdict(w, verdict)
}

// browserData is the log-browser template view-model: the overlaid hub status, its
// fixed-table badge label, whether an accepted checkpoint exists, and the accepted
// (size, root) the monitor vouches for (root base64-Std encoded, matching the proof
// encodings). When HasCheckpoint is false the page renders a "no accepted checkpoint
// yet" state (Size 0, Root empty) — a followed-but-unpolled hub, never a fabricated
// guarantee (ADR-0001 coverage honesty). The hubStatusBadge partial reads .Label
// directly (it does not re-derive the label from .Status), so the view carries a
// precomputed Label from the badge package's single source of truth.
type browserData struct {
	Status        string
	Label         string
	HasCheckpoint bool
	Size          uint64
	Root          string
}

// serveBrowser renders the HTML log browser for the hub-log root (GET /): the
// monitor's accepted checkpoint (size, root) plus relative links into the proof
// surface. It reads only persisted store rows (FollowState + CheckpointAt) — no
// signature, RFC-6962, Merkle, or proof computation; the served (size, root) are
// read back verbatim, never recomputed. The displayed hub status overlays the
// store-provable subset with the in-memory live verdict (overlayStatus) so the
// richer unresolvable / unverified states render through the same hubStatusBadge
// partial the dashboard uses.
//
// Status mapping: a FollowState / CheckpointAt DB error → 500; a CheckpointAt
// found==false at the accepted size is the same real store inconsistency serveVerify
// treats as 500. A hub with no accepted checkpoint yet (LastSize == 0) renders a
// 200 "no accepted checkpoint yet" page (not a 404 — the browser page exists for a
// followed-but-unpolled hub, mirroring the dashboard's coverage honesty). The page
// is rendered into a buffer first so a template/store error is a 500 BEFORE any 200.
func serveBrowser(w http.ResponseWriter, r *http.Request, st *store.Store, hubID int64, statuses StatusSource) {
	ctx := r.Context()

	fs, err := st.FollowState(ctx, hubID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	status := overlayStatus(fs, hubID, statuses)
	// The hubStatusBadge partial reads .Label directly; precompute it from the
	// badge package's single source of truth. The !ok fallback is defensive-only —
	// overlayStatus only ever yields valid labels keys (frozen / verified /
	// unresolvable / unverified).
	label, ok := badge.Label(status)
	if !ok {
		label = status
	}
	data := browserData{Status: status, Label: label}

	size := fs.LastSize
	if size > 0 {
		root, _, found, err := st.CheckpointAt(ctx, hubID, size)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// A missing checkpoint row at the accepted size is a real store
		// inconsistency, the same fault serveVerify maps to 500.
		if !found {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		data.HasCheckpoint = true
		data.Size = size
		data.Root = base64.StdEncoding.EncodeToString(root)
	}

	var buf bytes.Buffer
	if err := browserTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Post-200 write-drop: the status is already committed, so a copy error can
	// only signal a broken client connection, which a second status cannot fix
	// (matching the dashboard and the package's other write helpers).
	_, _ = buf.WriteTo(w)
}

// recordsData is the record-list template view-model: the overlaid hub status and
// its fixed-table badge label (rendered through the hubStatusBadge partial, the same
// way browserData does), the newest-first page of indexed records, the hub's total
// indexed-record count for the honest "showing N of TOTAL" line, the page size echoed
// into the pagination links, and the precomputed older/newer cursors plus their
// liveness flags so the template renders plain no-JS pagination links without doing
// any arithmetic itself. Records carry the verbatim id / schema (ADR-0008: nothing is
// interpreted). When Records is empty the page renders the informative empty state.
type recordsData struct {
	Status    string
	Label     string
	Records   []store.RecordRow
	Total     int
	PageSize  int
	HasNewer  bool
	NewerFrom uint64
	HasOlder  bool
	OlderFrom uint64
}

// serveRecords renders the HTML record list for the hub-log /records route: a
// newest-first (seq DESC), plain-link-paginated window over the hub's iscc_index,
// each row linking to that leaf's single-record page (record?index=<seq>). It
// reads only persisted store rows (FollowState for the overlay status + ListRecords)
// — no signature, RFC-6962, Merkle, or proof computation. The displayed hub status
// overlays the store-provable subset with the in-memory live verdict (overlayStatus),
// the same five-status hubStatusBadge partial the browser and dashboard use.
//
// The list is capped at the accepted tree size (fs.LastSize): only leaves with
// seq < LastSize list, the same accepted-tree contract serveEntries / serveInclusion /
// serveVerify enforce — on a freeze/fault ingest can leave iscc_index rows above
// LastSize, and an uncapped list would imply those unaccepted leaves are vouched for
// (coverage honesty, ADR-0001).
//
// Pagination is a seq cursor, not OFFSET, so paging stays stable under concurrent
// ingest. from / n are parsed via parseUint, so an empty or non-numeric value falls
// back to the default (bare /records works, never a 400). from is NOT overloaded as
// the start-at-newest sentinel: an absent from means "no cursor (start at the newest
// leaf)", while from=0 is a real cursor (the page starting at seq 0), so the older
// chain reaches the oldest record. n is clamped to maxPageSize while still uint64
// (BEFORE the int() conversion, so a huge n cannot wrap negative and bypass the clamp)
// so a hostile n can never scan the whole index. The older link starts at one below
// the last (smallest) seq on the page and is live only while more records remain below
// it; the newer link starts at one above the first (largest) seq and is live only
// while this page does not already begin at the newest record.
//
// Status mapping: a FollowState / ListRecords DB error → 500; an empty index (or a
// hub with no accepted checkpoint) → 200 with the informative empty state (NEVER a
// 404 — coverage honesty, ADR-0001, mirroring serveBrowser's LastSize==0 → 200 empty
// page). The page is rendered into a buffer first so a template/store error is a 500
// BEFORE any 200.
func serveRecords(w http.ResponseWriter, r *http.Request, st *store.Store, hubID int64, statuses StatusSource) {
	ctx := r.Context()

	fs, err := st.FollowState(ctx, hubID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	status := overlayStatus(fs, hubID, statuses)
	// The hubStatusBadge partial reads .Label directly; precompute it from the badge
	// package's single source of truth. The !ok fallback is defensive-only —
	// overlayStatus only ever yields valid labels keys.
	label, ok := badge.Label(status)
	if !ok {
		label = status
	}

	// from is the inclusive upper-bound seq the newest-first page starts at. A present,
	// numeric from is a real cursor (from=0 starts at seq 0, so the older chain reaches
	// the oldest record); an absent or non-numeric from means "no cursor (start at the
	// newest leaf)", never a 400. hasFrom carries that present/absent distinction so the
	// seq cursor is never overloaded as the start-at-newest sentinel.
	from, fromErr := parseUint(r.URL.Query().Get("from"))
	hasFrom := fromErr == nil
	// n is the page size, defaulted and clamped. The clamp happens while n is still
	// uint64, BEFORE the int() conversion: a huge n (e.g. 9223372036854775808) would
	// wrap int(n) negative and slip past a post-conversion > maxPageSize check, and
	// modernc SQLite reads a negative LIMIT as unlimited (the whole index). parseUint
	// itself rejects an overflow now, but the uint64 clamp is the load-bearing guard.
	pageSize := defaultPageSize
	if n, err := parseUint(r.URL.Query().Get("n")); err == nil && n > 0 {
		if n > maxPageSize {
			n = maxPageSize
		}
		pageSize = int(n)
	}

	records, total, err := st.ListRecords(ctx, hubID, fs.LastSize, hasFrom, from, pageSize)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := recordsData{
		Status:   status,
		Label:    label,
		Records:  records,
		Total:    total,
		PageSize: pageSize,
	}
	if len(records) > 0 {
		oldest := records[len(records)-1].Seq // smallest seq on the page
		// A newer page exists only once we have paged down at all (hasFrom; the
		// no-cursor page always starts at the newest leaf). Its cursor is a full page
		// above the page's top seq; ListRecords clamps seq <= cursor with LIMIT
		// pageSize, so an over-large cursor simply lands on the newest page — the
		// newer link can never error or skip the newest run.
		if hasFrom {
			data.HasNewer = true
			data.NewerFrom = records[0].Seq + uint64(pageSize)
		}
		// An older page exists when a record sits below the smallest seq shown; its
		// cursor is one below that smallest seq. oldest == 0 means the page already
		// reached seq 0 (the oldest record), so there is nothing older. When oldest is
		// 1 the cursor is 0, a real cursor now (not the start-at-newest sentinel), so
		// the older chain reaches seq 0. This is a conservative live/dead signal under
		// contiguous leaf seqs (the projection writer indexes one row per accepted
		// leaf), and never produces a broken link.
		if oldest > 0 {
			data.HasOlder = true
			data.OlderFrom = oldest - 1
		}
	}

	var buf bytes.Buffer
	if err := recordsTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Post-200 write-drop: the status is already committed, so a copy error can only
	// signal a broken client connection (matching serveBrowser and the dashboard).
	_, _ = buf.WriteTo(w)
}

// recordData is the single-record template view-model: the overlaid hub status and
// its fixed-table badge label (rendered through the hubStatusBadge partial, the same
// way browserData / recordsData do), the leaf's absolute Seq, the human-readable Kind
// label and an IsDeletion flag (the only interpretation, mapped from the verbatim
// note.$schema, ADR-0008), HasProjection reporting whether an iscc_index row exists
// for this leaf (false → the id / schema cells render a "no projection indexed" state
// since the bytes, not the projection, are the source of truth), the verbatim IsccID
// and NoteSchema strings (never interpreted beyond Kind), and RecordBytes — the raw
// opaque JCS-canonical log-entry envelope cast to a string for html/template auto-
// escape (NOT template.HTML), shown verbatim for human inspection.
type recordData struct {
	Status        string
	Label         string
	Seq           uint64
	Kind          string
	IsDeletion    bool
	HasProjection bool
	IsccID        string
	NoteSchema    string
	RecordBytes   string
}

// recordKind maps the verbatim note.$schema to a human-readable kind label and a
// deletion flag — the ONLY interpretation the single-record page performs (ADR-0008).
// It is fail-open by construction: the declaration / deletion schemas map to their
// friendly labels, and anything else (including an empty schema) is kindUnknown, so
// an unknown or empty schema never gates or errors the page; the verbatim schema
// string is shown alongside the label regardless.
func recordKind(noteSchema string) (kind string, isDeletion bool) {
	switch noteSchema {
	case schemaDeclaration:
		return kindDeclaration, false
	case schemaDeletion:
		return kindDeletion, true
	default:
		return kindUnknown, false
	}
}

// serveRecord renders the HTML single-record page for the hub-log /record route: one
// accepted leaf's seq, kind label (declaration / deletion / unknown, the only
// interpretation, ADR-0008), verbatim ISCC-ID and note.$schema, and raw record bytes.
// The bytes are the source of truth: the page does its work from the mirrored bundle
// bytes (the same accepted-tree-capped bundle math serveEntries uses), while the
// iscc_index projection only supplies the id / schema labels — so a leaf whose bytes
// are mirrored but whose projection is absent still renders (HasProjection=false), the
// id / schema shown as a "no projection indexed" state rather than a 404.
//
// Status mapping (mirroring serveEntries' accepted-tree contract): a missing or non-
// numeric index → 400; no accepted checkpoint yet (LastSize == 0) or seq >= LastSize
// (the leaf is not in the monitor's accepted tree) → 404; the bundle not yet mirrored
// (a wrapped os.ErrNotExist) → 404; the bundle mirrored but only partial and not yet
// covering this leaf (ErrLeafOutOfBundle) → 404; a FollowState / RecordAt / bundle-read
// DB error → 500. A missing projection is NOT a 404 (the bytes are the truth). The page
// is rendered into a buffer first so a template/store error is a 500 BEFORE any 200.
func serveRecord(w http.ResponseWriter, r *http.Request, st *store.Store, f store.SQLiteFetcher, hubID int64, statuses StatusSource) {
	ctx := r.Context()

	seq, err := parseUint(r.URL.Query().Get("index"))
	if err != nil {
		http.Error(w, "missing or non-numeric index", http.StatusBadRequest)
		return
	}

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
	if seq >= size {
		http.Error(w, "leaf not covered by accepted checkpoint", http.StatusNotFound)
		return
	}

	// Read the leaf's raw record bytes from the mirrored entry bundle the same way
	// serveEntries does: the final bundle of a non-multiple-of-256 tree is a partial,
	// so request its expected p (the SQLiteFetcher does the partial→full fallback) —
	// passing p == 0 unconditionally would 404 a real leaf in that final partial.
	bundleIndex := seq / tiles.TileWidth
	offset := seq % tiles.TileWidth
	p := tiles.PartialTileSize(0, bundleIndex, size)
	bundle, err := f.ReadEntryBundle(ctx, bundleIndex, p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "bundle not mirrored", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	record, err := logclient.RecordBytesFromBundle(bundle, offset)
	if err != nil {
		if errors.Is(err, logclient.ErrLeafOutOfBundle) {
			http.Error(w, "leaf not in mirrored bundle", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// The projection is a derived view (ADR-0008) that only supplies the id / schema
	// labels; a missing row is a plain miss, not a fault. The bytes above are the
	// source of truth, so HasProjection=false still renders the page.
	row, found, err := st.RecordAt(ctx, hubID, seq)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	kind, isDeletion := recordKind(row.NoteSchema)

	status := overlayStatus(fs, hubID, statuses)
	// The hubStatusBadge partial reads .Label directly; precompute it from the badge
	// package's single source of truth. The !ok fallback is defensive-only —
	// overlayStatus only ever yields valid labels keys.
	label, ok := badge.Label(status)
	if !ok {
		label = status
	}

	data := recordData{
		Status:        status,
		Label:         label,
		Seq:           seq,
		Kind:          kind,
		IsDeletion:    isDeletion,
		HasProjection: found,
		IsccID:        row.IsccID,
		NoteSchema:    row.NoteSchema,
		// The record envelope is opaque bytes for human inspection — string-cast for
		// html/template auto-escape, never parsed and never template.HTML.
		RecordBytes: string(record),
	}

	var buf bytes.Buffer
	if err := recordTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Post-200 write-drop: the status is already committed, so a copy error can only
	// signal a broken client connection (matching serveBrowser / serveRecords).
	_, _ = buf.WriteTo(w)
}

// hubStatus maps the persisted follow state to the glossary hub-status subset the
// store can substantiate: "frozen" if the hub is frozen (a self-consistency
// violation, ADR-0006), else "verified" once it has an accepted checkpoint (only
// signature-verified checkpoints advance LastSize). It deliberately invents no
// status the store cannot prove (verified/unverified/unresolvable/rotated/inactive
// are tracked elsewhere; this route reads only what the store persists).
func hubStatus(fs store.FollowState) string {
	if fs.Frozen {
		return "frozen"
	}
	return "verified"
}

// overlayStatus resolves the log browser's displayed status from the store-provable
// subset (hubStatus) plus the in-memory live verdict. It mirrors
// dashboard.overlayStatus precedence verbatim: the store status wins for the durable,
// harder truth (frozen, an ADR-0006 self-consistency violation that survives restart,
// must never be overridden by a fresher in-memory verdict). Only when the store says
// "verified" does it consult statuses, and only to adopt "unresolvable" (a currently-
// failing did:web resolve) or "unverified" (a current signature matching no listed
// key) — both fresher than the store's verified flag (ADR-0009). It is nil-tolerant:
// a nil statuses keeps the store status. serveBrowser reads FollowState (not the
// realm-active flag), so there is no "inactive" input here; the overlay can only ever
// flip "verified" → "unresolvable"/"unverified".
func overlayStatus(fs store.FollowState, hubID int64, statuses StatusSource) string {
	status := hubStatus(fs)
	if status != "verified" || statuses == nil {
		return status
	}
	switch live, ok := statuses.Status(hubID); {
	case ok && (live == "unresolvable" || live == "unverified"):
		return live
	default:
		return status
	}
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
// malformed index becomes a 400 rather than a silent default. It also rejects a value
// that would overflow uint64 (the accumulator n*10 + digit wraps silently otherwise),
// so an over-large index never aliases a small one — the same error type as a
// non-numeric input, so the shared callers (selectSeq / serveInclusion / serveEntries /
// serveRecords) treat overflow exactly as they treat a malformed value.
func parseUint(s string) (uint64, error) {
	var n uint64
	if s == "" {
		return 0, errors.New("empty index")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("non-numeric index")
		}
		d := uint64(c - '0')
		// Reject before the multiply/add wraps: n*10 + d must stay <= math.MaxUint64.
		if n > (math.MaxUint64-d)/10 {
			return 0, errors.New("index overflow")
		}
		n = n*10 + d
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

// writeRecord writes the raw record bytes of one leaf verbatim as
// application/octet-stream — the opaque JCS-canonical log-entry envelope a proof
// bundle commits to, NOT wrapped in any JSON shape. It keeps the documented
// post-status write-drop convention (matching tilesserve.writeBlob): the 200 is
// sent on the first byte and a mid-write fault on an opaque BLOB cannot un-send it,
// so the only failure mode is a broken client connection, dropped deliberately.
func writeRecord(w http.ResponseWriter, record []byte) {
	w.Header().Set("Content-Type", octetStreamType)
	_, _ = w.Write(record)
}

// writeVerdict writes the verify-for-me JSON verdict with a 200 status. Every
// verdict — verified or not — is a 200: the route reserves non-200 for genuine
// infra faults (handled before this call). It keeps writeEvidence's drop-the-
// write-error-after-200 posture: a marshal of a fixed-shape struct of
// strings/uints/bools cannot fail for content reasons, and a mid-write fault
// cannot un-send the 200.
func writeVerdict(w http.ResponseWriter, v VerifyVerdict) {
	w.Header().Set("Content-Type", contentType)
	_ = encodeJSON(w, v)
}

// encodeJSON marshals v to w. It keeps the single drop-the-write-error site tidy.
func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
