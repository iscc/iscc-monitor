// This file holds the networked checkpoint-fetch primitive: given a hub base URL
// and an injected Fetcher it returns the raw bytes of the hub's latest signed
// checkpoint from the canonical https://<domain>/log/checkpoint resource
// (iscc-log §9, "GET /log/checkpoint"). It is the transport-only counterpart to
// ResolveVerifierKey — it produces exactly the raw []byte that AcceptCheckpoint
// takes, closing the gap between "verify bytes I'm handed" and "go get the bytes"
// without parsing or persistence. The signed-note framing belongs to
// VerifyCheckpoint/AcceptCheckpoint; this function returns the body verbatim,
// mirroring Tessera's ReadCheckpoint.
package logclient

import (
	"context"
	"fmt"
)

// FetchCheckpoint fetches the raw signed-checkpoint bytes for a hub base URL.
//
// It derives the canonical checkpoint URL by reusing the shared origin() helper
// (which yields the scheme-less "<domain>/log") and appending "/checkpoint", so a
// base URL such as "https://sb0.iscc.id" or the bare "sb0.iscc.id" both resolve to
// "https://sb0.iscc.id/log/checkpoint" (tlog/did:web are always HTTPS). It fetches
// through the injected Fetcher and returns the body verbatim — no trimming or
// parsing; the signed-note framing is the caller's concern. The Fetcher's error is
// wrapped with %w so a 404's errors.Is(err, os.ErrNotExist) still holds, letting
// the follower tell "no checkpoint served" from other transport faults; an
// origin() error is likewise wrapped and returned.
func FetchCheckpoint(ctx context.Context, fetcher Fetcher, baseURL string) ([]byte, error) {
	name, err := origin(baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch checkpoint: %w", err)
	}
	url := "https://" + name + "/checkpoint"
	raw, err := fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch checkpoint %q: %w", url, err)
	}
	return raw, nil
}
