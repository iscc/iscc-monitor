// Package follower composes the pure logclient verification chain with the
// stateful store to follow a hub's transparency log. It is the dependency
// direction follower -> {logclient, store}, never the reverse, so net/http never
// enters the store's closure and the store stays a leaf.
//
// This file holds PollHub, the single-observation wiring: for one hub it fetches
// the latest signed checkpoint, runs the four-way AcceptCheckpoint verdict,
// persists the observed checkpoint, and advances the follow cursor — but only
// when the verdict is StatusVerified, since that is the only outcome that may
// advance accepted state (ADR-0009). The poll loop, the RFC-6962 consistency
// check, freeze/alert, coverage, and the did:web key cache are each their own
// later steps; PollHub does exactly one observation per call and returns.
//
// Record-only-on-verified: only a StatusVerified observation is persisted, since
// the non-verified verdicts carry a zero CheckpointInfo and therefore no
// trustworthy (size, root) to record. A later step may revisit recording
// non-verified observations (e.g. for evidence of an internally-broken hub); for
// now an unverified/unresolvable/rotated verdict is returned to the caller
// without touching the store.
package follower

import (
	"context"
	"fmt"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
)

// PollHub performs one observation of a hub's latest checkpoint.
//
// It fetches the raw checkpoint, runs the four-way AcceptCheckpoint verdict, and
// — only on StatusVerified — records the observed checkpoint and advances the
// per-hub follow cursor. hubID and baseURL are pre-resolved by the caller and
// observedAt is injected (never time.Now() here) so the decision is deterministic
// and the seam stays small.
//
// The returned Status is the verdict for any of the four outcomes: a non-verified
// verdict is a verdict, not a Go error. The returned error is reserved for a
// genuine fault — a transport failure fetching the checkpoint, or a
// verified-but-garbled body from AcceptCheckpoint (which returns a non-nil error
// alongside StatusUnverified's zero value, so the error is checked before the
// status). On any such fault nothing is persisted.
func PollHub(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, observedAt time.Time) (logclient.Status, error) {
	raw, err := logclient.FetchCheckpoint(ctx, fetcher, baseURL)
	if err != nil {
		return logclient.StatusUnverified, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}

	status, info, err := logclient.AcceptCheckpoint(ctx, fetcher, baseURL, raw, observedAt)
	if err != nil {
		// A verified-but-garbled body is a genuine fault, returned alongside
		// StatusUnverified's zero value; the error is checked before the status.
		return status, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}

	// Only a verified observation advances accepted state; the other verdicts are
	// reported (and the hub is still mirrored elsewhere) but not persisted here.
	if status != logclient.StatusVerified {
		return status, nil
	}

	rec := store.CheckpointRecord{
		HubID:      hubID,
		Status:     status.String(),
		TreeSize:   info.TreeSize,
		Root:       info.Root[:],
		Raw:        raw,
		ObservedAt: observedAt,
	}
	if _, _, err := st.RecordCheckpoint(ctx, rec); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: record checkpoint: %w", hubID, err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, info.TreeSize); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: advance follow state: %w", hubID, err)
	}
	return status, nil
}
