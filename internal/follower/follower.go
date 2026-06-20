// Package follower composes the pure logclient verification chain with the
// stateful store to follow a hub's transparency log. It is the dependency
// direction follower -> {logclient, store}, never the reverse, so net/http never
// enters the store's closure and the store stays a leaf.
//
// This file holds PollHub, the single-observation wiring: for one hub it fetches
// the latest signed checkpoint, runs the four-way AcceptCheckpoint verdict, runs
// the dep-free RFC-6962 self-consistency checks (shrink/fork) against the prior
// accepted checkpoint, and then either freezes the hub on a violation or advances
// the follow cursor — but only a StatusVerified observation may advance accepted
// state (ADR-0009). On that verified, non-violation path it also records the hub's
// coverage start once (ADR-0001, set-once) and caches the resolved did:web signing
// key (ADR-0009, hub_keys). The merkle-backed equivocation trigger and the poll
// loop are each their own later steps; PollHub does exactly one observation per
// call and returns.
//
// Record-only-on-verified: only a StatusVerified observation is persisted, since
// the non-verified verdicts carry a zero CheckpointInfo and therefore no
// trustworthy (size, root) to record. A later step may revisit recording
// non-verified observations (e.g. for evidence of an internally-broken hub); for
// now an unverified/unresolvable/rotated verdict is returned to the caller
// without touching the store.
//
// Freeze on violation (ADR-0006): when a verified observation contradicts the
// prior accepted checkpoint for the same hub (a strict tree-size decrease =
// shrink, or the same size with a different root = fork), PollHub records the
// violation as irreplaceable evidence, freezes the hub, and fires the injected
// alert exactly once on the not-frozen -> frozen transition. A frozen hub does
// not advance its cursor; the violation freezes, never crashes, so the verdict
// status is still returned with a nil error. Re-detecting on a later poll records
// the violation again (re-detection is itself evidence) but never re-alerts.
package follower

import (
	"context"
	"fmt"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
)

// AlertFunc is the minimal injected alert sink the follower fires once per
// not-frozen -> frozen transition. It is a func seam (not an interface) for
// YAGNI: a test passes a counter, production passes a real transport. Delivery
// (email/webhook/log sink) and the backed-off evidence-only re-poll cadence of a
// frozen hub are out of scope here; this only signals the transition.
type AlertFunc func(hubID int64, kind string)

// PollHub performs one observation of a hub's latest checkpoint.
//
// It fetches the raw checkpoint, runs the four-way AcceptCheckpoint verdict, and
// — only on StatusVerified — runs the RFC-6962 self-consistency checks against
// the prior accepted checkpoint, then either freezes the hub (recording the
// violation + alerting once) or records the checkpoint and advances the per-hub
// follow cursor. hubID and baseURL are pre-resolved by the caller and observedAt
// is injected (never time.Now() here) so the decision is deterministic and the
// seam stays small. alert is fired exactly once per not-frozen -> frozen
// transition; pass a no-op to ignore it.
//
// The returned Status is the verdict for any of the four outcomes: a non-verified
// verdict is a verdict, not a Go error, and a self-consistency violation freezes
// the hub but still returns StatusVerified with a nil error (the signature was
// valid; the violation is a separate axis — ADR-0006). The returned error is
// reserved for a genuine fault — a transport failure fetching the checkpoint, a
// verified-but-garbled body from AcceptCheckpoint (which returns a non-nil error
// alongside StatusUnverified's zero value, so the error is checked before the
// status), or a store failure. On any such fault accepted state does not advance.
func PollHub(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, observedAt time.Time, alert AlertFunc) (logclient.Status, error) {
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

	// Self-consistency check against the prior accepted checkpoint, before any
	// record/advance: a violation must freeze (not advance) the hub (ADR-0006).
	fs, err := st.FollowState(ctx, hubID)
	if err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: follow state: %w", hubID, err)
	}
	violated, kind, prevRaw, err := checkConsistency(ctx, st, hubID, fs.LastSize, info)
	if err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}
	if violated {
		return status, freeze(ctx, st, hubID, kind, prevRaw, raw, info, fs.Frozen, observedAt, alert)
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
	// Record coverage start once on the first verified, non-violation observation
	// (ADR-0001); SetCoverage is set-once, so a later poll never moves it.
	if err := st.SetCoverage(ctx, hubID, info.TreeSize, observedAt); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: set coverage: %w", hubID, err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, info.TreeSize); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: advance follow state: %w", hubID, err)
	}
	// Cache the resolved did:web signing key (ADR-0009). Only a verified,
	// non-violation observation writes a cache row, mirroring coverage: a
	// contradictory or unverified observation must never populate the key cache.
	if err := cacheHubKey(ctx, st, fetcher, hubID, baseURL, observedAt); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: cache hub key: %w", hubID, err)
	}
	return status, nil
}

// cacheHubKey resolves the hub's did:web signing key and upserts it into the
// hub_keys cache (ADR-0009: the DID document is the source of truth, this is a
// refreshed cache). It re-runs ResolveVerifierKey on the verified path — the key
// just resolved inside AcceptCheckpoint, so a failure here is an unexpected fault
// and is returned to the caller rather than swallowed. The key id is recovered
// from the vkey string's middle "+<hex>+" field (KeyIDFromVerifier) rather than
// re-derived, and the DIDKey maps field-for-field into store.HubKey
// (PublicKey->PubkeyRaw, Multibase->PubkeyZ, Revoked->Revoked); the injected
// observedAt is the resolution time. The follower owns this mapping so store stays
// a leaf (it never imports logclient/didweb).
func cacheHubKey(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, observedAt time.Time) error {
	vkey, didKey, err := logclient.ResolveVerifierKey(ctx, fetcher, baseURL)
	if err != nil {
		return fmt.Errorf("resolve verifier key: %w", err)
	}
	keyID, err := logclient.KeyIDFromVerifier(vkey)
	if err != nil {
		return fmt.Errorf("recover key id: %w", err)
	}
	return st.RecordHubKey(ctx, store.HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  didKey.PublicKey,
		PubkeyZ:    didKey.Multibase,
		Revoked:    didKey.Revoked,
		ResolvedAt: observedAt,
	})
}

// checkConsistency runs the dep-free shrink/fork triggers against the prior
// accepted checkpoint for a hub. prevSize is FollowState.LastSize; the CheckShrink
// and CheckFork prevSize>0 guards mean a fresh-store zero never trips a violation,
// so a never-advanced hub is always clean. The prior root and raw bytes come from
// a checkpoints lookup at prevSize, since follow_state does not persist the root;
// if no checkpoint is stored at that size (a hub that advanced before this code
// existed), the fork check is skipped while the size-only shrink check still runs.
// On a true verdict it returns the matching violation kind and the prior raw bytes
// (RawA evidence). Shrink and fork are mutually exclusive by size, so shrink is
// evaluated first and its kind used when true.
func checkConsistency(ctx context.Context, st *store.Store, hubID int64, prevSize uint64, info logclient.CheckpointInfo) (violated bool, kind logclient.ViolationKind, prevRaw []byte, err error) {
	if prevSize == 0 {
		return false, "", nil, nil
	}
	prevRootBytes, prevRaw, prevFound, err := st.CheckpointAt(ctx, hubID, prevSize)
	if err != nil {
		return false, "", nil, fmt.Errorf("checkpoint at prior size %d: %w", prevSize, err)
	}
	var prevRoot [32]byte
	copy(prevRoot[:], prevRootBytes)

	shrink := logclient.CheckShrink(prevSize, info.TreeSize)
	fork := prevFound && logclient.CheckFork(prevSize, prevRoot, info.TreeSize, info.Root)
	switch {
	case shrink:
		return true, logclient.ViolationShrink, prevRaw, nil
	case fork:
		return true, logclient.ViolationFork, prevRaw, nil
	default:
		return false, "", nil, nil
	}
}

// freeze records the violation as irreplaceable evidence, persists the
// contradictory checkpoint, freezes the hub, and fires the alert exactly once on
// the not-frozen -> frozen transition (ADR-0006). It deliberately does NOT
// advance the cursor — a frozen hub never advances accepted state. A violation
// freezes, never crashes, so this returns nil on success even though a violation
// was found; a non-nil error is only a store failure. wasFrozen gates the
// alert: a later poll of an already-frozen hub records the violation again
// (re-detection is evidence — RecordViolation has no ON CONFLICT) but never
// re-alerts.
func freeze(ctx context.Context, st *store.Store, hubID int64, kind logclient.ViolationKind, prevRaw, raw []byte, info logclient.CheckpointInfo, wasFrozen bool, observedAt time.Time, alert AlertFunc) error {
	if _, err := st.RecordViolation(ctx, store.Violation{
		HubID:      hubID,
		Kind:       string(kind),
		RawA:       prevRaw,
		RawB:       raw,
		DetectedAt: observedAt,
	}); err != nil {
		return fmt.Errorf("record violation: %w", err)
	}
	// Persist the contradictory checkpoint as evidence, but do not advance.
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     logclient.StatusVerified.String(),
		TreeSize:   info.TreeSize,
		Root:       info.Root[:],
		Raw:        raw,
		ObservedAt: observedAt,
	}); err != nil {
		return fmt.Errorf("record contradictory checkpoint: %w", err)
	}
	if err := st.Freeze(ctx, hubID); err != nil {
		return fmt.Errorf("freeze: %w", err)
	}
	if !wasFrozen {
		alert(hubID, string(kind))
	}
	return nil
}
