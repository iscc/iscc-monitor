// This file decodes a tlog-tiles entry bundle and returns the RAW record bytes of
// a single leaf within it — the record_bytes member every self-contained proof
// bundle needs. It is the sibling of leafhasher.go and projection.go: the same
// api.EntryBundle{}.UnmarshalText decode of the C2SP framing, but a different
// projection — leafhasher.go RFC-6962-hashes each entry, projection.go folds each
// into an iscc_index row, and this returns one entry's bytes verbatim (the
// JCS-canonical log-entry envelope), never re-hashed or interpreted.
//
// Purity (Correctness rule: proof/verify is pure; keep WASM-shareable): this file
// imports only stdlib (errors, fmt) + the dep-clean tessera/api closure — already
// proven WASM-clean by leafhasher.go / projection.go in this package. It must not
// pull net / net/http / database/sql / os, so the file-level import set stays
// WASM-shareable even though the logclient package as a whole pulls net/http via
// didresolve.go.
package logclient

import (
	"errors"
	"fmt"

	"github.com/transparency-dev/tessera/api"
)

// ErrLeafOutOfBundle is returned by RecordBytesFromBundle when the requested offset
// is past the last entry the bundle actually contains. The handler maps it via
// errors.Is — a mirrored-but-partial bundle that does not yet cover this leaf is a
// 404, not a 500.
var ErrLeafOutOfBundle = errors.New("logclient: leaf offset out of entry bundle")

// RecordBytesFromBundle decodes a tlog-tiles entry bundle and returns the raw
// record bytes of the leaf at offset within it (offset is the leaf's position in
// the bundle, seq % 256). The returned bytes are the JCS-canonical log-entry
// envelope verbatim — the record_bytes a proof bundle carries — NOT a leaf hash and
// never re-hashed or interpreted.
//
// A truncated bundle frame surfaces the wrapped UnmarshalText error. An offset at
// or past the number of decoded entries returns ErrLeafOutOfBundle (checkable via
// errors.Is) — a partial bundle may simply not cover this leaf yet.
func RecordBytesFromBundle(bundle []byte, offset uint64) ([]byte, error) {
	eb := &api.EntryBundle{}
	if err := eb.UnmarshalText(bundle); err != nil {
		return nil, fmt.Errorf("logclient.RecordBytesFromBundle: unmarshal entry bundle: %w", err)
	}
	if offset >= uint64(len(eb.Entries)) {
		return nil, fmt.Errorf("logclient.RecordBytesFromBundle: offset %d of %d entries: %w", offset, len(eb.Entries), ErrLeafOutOfBundle)
	}
	return eb.Entries[offset], nil
}
