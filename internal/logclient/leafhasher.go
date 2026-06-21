// This file decodes a tlog-tiles entry bundle and RFC-6962-leaf-hashes each
// entry it contains, in index order — the leaf hasher that fsck.New(...) takes as
// its hasher argument when rebuilding a tree's root from its mirrored tiles. It is
// a faithful port of the leafHasher in cauldron/iscc-hub/conformance/runfsck:
// api.EntryBundle{}.UnmarshalText parses the C2SP framing (2-byte big-endian
// length prefix + data per entry), then rfc6962.DefaultHasher.HashLeaf hashes each
// entry into a 32-byte leaf hash.
//
// Purity (Correctness rule: proof/verify is pure; keep WASM-shareable): this file
// imports only stdlib (fmt) + the dep-clean merkle/rfc6962 and tessera/api
// closures — both already proven WASM-clean by proofbuilder.go in this package. It
// must not pull net / net/http / database/sql / os, so the file-level import set
// stays WASM-shareable even though the logclient package as a whole pulls net/http
// via didresolve.go.
package logclient

import (
	"fmt"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api"
)

// LeafHashes decodes a tlog-tiles entry bundle and returns the RFC-6962 leaf hash
// of every entry it contains, in index order. Each returned hash is exactly 32
// bytes. The signature matches the fsck.New(...) hasher contract exactly, so a
// later slice can feed it (plus the SQLiteFetcher) into the real fsck root-rebuild
// integrity check.
//
// An empty bundle (len == 0) decodes to zero entries and returns (empty, nil). A
// truncated bundle — a 2-byte length prefix promising more data bytes than remain
// — surfaces the wrapped UnmarshalText error.
func LeafHashes(bundle []byte) ([][]byte, error) {
	eb := &api.EntryBundle{}
	if err := eb.UnmarshalText(bundle); err != nil {
		return nil, fmt.Errorf("logclient.LeafHashes: unmarshal entry bundle: %w", err)
	}
	out := make([][]byte, 0, len(eb.Entries))
	for _, e := range eb.Entries {
		h := rfc6962.DefaultHasher.HashLeaf(e)
		out = append(out, h[:])
	}
	return out, nil
}
