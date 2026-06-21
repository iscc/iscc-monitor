// This file wires tessera's fsck root-rebuild integrity check over a local mirror
// fetcher: RunFsck fetches and verifies a hub-signed checkpoint, then re-hashes the
// mirrored entry bundles (via LeafHashes), re-derives the hash tiles, and compares
// the rebuilt RFC-6962 root against the signed checkpoint root. It is the M2
// root-rebuild conformance seam — proving fsck.New(...).Check(...) rebuilds each
// accepted root from the SQLiteFetcher, so the monitor's mirror is verifiable
// against the hub's own signature without re-trusting the hub.
//
// RunFsck is thin glue: it takes the fsck.Fetcher interface (NOT the concrete
// store.SQLiteFetcher) so the production logclient package gains no store import
// edge; the caller supplies the concrete fetcher. fsck is an in-process structural
// self-check — it shares the monitor's own LeafHashes / RFC-6962 code, so it
// catches mirror corruption and rebuild bugs but is NOT the fully-independent
// oracle (notecheck, run in CI, is that). This file is a server-side mirror check,
// never part of the WASM verify path, so it may pull fsck's heavier closure.
package logclient

import (
	"context"
	"fmt"

	"github.com/transparency-dev/tessera/fsck"
	"golang.org/x/mod/sumdb/note"
)

// RunFsck rebuilds a hub's signed checkpoint root from the mirrored tiles and entry
// bundles reachable through f, returning nil only when the rebuilt root matches the
// checkpoint's claimed root. vkey is exactly the verifier-key string
// ResolveVerifierKey produces and origin the signed-note name (<domain>/log). It
// fetches and verifies the checkpoint via f.ReadCheckpoint, re-hashes each entry
// bundle with LeafHashes, re-derives the lower hash tiles, and compares the rebuilt
// root to the checkpoint root; a corrupted mirrored BLOB or a stale root yields a
// non-nil error. A bad vkey wraps an error before any fetch, mirroring
// VerifyCheckpoint.
func RunFsck(ctx context.Context, vkey, origin string, f fsck.Fetcher) error {
	v, err := note.NewVerifier(vkey)
	if err != nil {
		return fmt.Errorf("logclient.RunFsck: bad vkey: %w", err)
	}
	if err := fsck.New(origin, v, f, LeafHashes, fsck.Opts{N: 1}).Check(ctx); err != nil {
		return fmt.Errorf("logclient.RunFsck: %w", err)
	}
	return nil
}
