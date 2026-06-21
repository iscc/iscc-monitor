// This file is the pure projection fold: it decodes one tlog-tiles entry bundle
// into per-leaf projection records {Seq, IsccID, NoteSchema, RecordSHA256} for the
// schema-agnostic iscc_index (ADR-0008). Each entry is the JCS canonicalization of
// the canonical log-entry envelope {$schema, iscc_id, note}; this fold extracts the
// committed iscc_id and the RAW inner note.$schema discriminator (declaration vs
// deletion vs any future/unknown note type) and the record-content SHA-256, without
// ever interpreting them. Interpretation — resolving the ISCC-ID, parsing the
// ISCC-CODE, folding deletion status into a projection — is deliberately deferred
// (ADR-0008: the index is a raw fold; only a JSON parse failure is an error).
//
// It is an unwired-until-M2 export seam: the store writer (RecordProjections into
// iscc_index) and the PollHub leaf-index lookup are separate later slices. go vet is
// clean; this is not dead code.
//
// Purity (Correctness rule: proof/verify is pure; keep WASM-shareable): imports are
// exactly crypto/sha256 + encoding/json + fmt + tessera/api — no
// net / net/http / database/sql / os / internal/store. The file-level import set
// stays WASM-shareable even though the logclient package as a whole pulls net/http
// via didresolve.go.
package logclient

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/transparency-dev/tessera/api"
)

// Projection is one entry-bundle leaf folded into the columns of the iscc_index
// table (ADR-0008). Seq is the leaf's absolute index in the tree; IsccID is the
// raw ISCC:-prefixed iscc_id string from the envelope; NoteSchema is the verbatim
// inner note.$schema discriminator (never interpreted, never validated against a
// known list); RecordSHA256 is the SHA-256 of the canonical record bytes — the
// record-content hash for fsck/debug cross-reference, NOT the RFC-6962 leaf hash
// (no 0x00 prefix).
type Projection struct {
	Seq          uint64
	IsccID       string
	NoteSchema   string
	RecordSHA256 [32]byte
}

// recordEnvelope is the minimal view of the canonical log-entry envelope this fold
// reads (iscc-log.md §5.1): the top-level committed iscc_id and the inner
// note.$schema discriminator. The envelope's own top-level $schema (the log-entry
// schema) is deliberately ignored — the projection keys on the INNER note schema
// (ADR-0008 Declaration vs Deletion). Unknown members are dropped by the standard
// unmarshal.
type recordEnvelope struct {
	IsccID string `json:"iscc_id"`
	Note   struct {
		Schema string `json:"$schema"`
	} `json:"note"`
}

// BundleProjections decodes a tlog-tiles entry bundle into one Projection per leaf,
// in index order, with each Seq made absolute via baseSeq (the bundle's first leaf
// index, bundleIndex*256, supplied by the caller). It is schema-agnostic (ADR-0008):
// an empty iscc_id or an unmodeled note.$schema is indexed verbatim, never rejected.
//
// An empty bundle (len == 0) decodes to zero leaves and returns (nil, nil). A
// malformed bundle frame surfaces the wrapped UnmarshalText error; a record whose
// bytes are not valid JSON is a genuine fault and surfaces a wrapped error naming
// the absolute seq — never a silently skipped leaf.
func BundleProjections(bundle []byte, baseSeq uint64) ([]Projection, error) {
	eb := &api.EntryBundle{}
	if err := eb.UnmarshalText(bundle); err != nil {
		return nil, fmt.Errorf("logclient.BundleProjections: unmarshal entry bundle: %w", err)
	}
	out := make([]Projection, 0, len(eb.Entries))
	for i, e := range eb.Entries {
		seq := baseSeq + uint64(i)
		var env recordEnvelope
		if err := json.Unmarshal(e, &env); err != nil {
			return nil, fmt.Errorf("logclient.BundleProjections: decode record at seq %d: %w", seq, err)
		}
		out = append(out, Projection{
			Seq:          seq,
			IsccID:       env.IsccID,
			NoteSchema:   env.Note.Schema,
			RecordSHA256: sha256.Sum256(e),
		})
	}
	return out, nil
}
