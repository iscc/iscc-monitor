// Package follower composes the pure logclient verification chain with the
// stateful store to follow a hub's transparency log. It is the dependency
// direction follower -> {logclient, store}, never the reverse, so net/http never
// enters the store's closure and the store stays a leaf.
//
// This file holds PollHub, the single-observation wiring: for one hub it fetches
// the latest signed checkpoint, runs the four-way AcceptCheckpoint verdict, mirrors
// the candidate-size hash tiles and entry bundles into the local store (ADR-0005,
// ingestTiles), runs the three RFC-6962 self-consistency checks
// (shrink/fork/equivocation) against the prior accepted checkpoint, and then either
// freezes the hub on a violation or advances the follow cursor — but only a
// StatusVerified observation may advance accepted state (ADR-0009). The merkle-backed
// equivocation trigger sources its consistency proof from the local mirror (a
// store.SQLiteFetcher), never re-hitting the hub; mirroring the candidate tiles
// BEFORE the consistency check is what makes that proof buildable on a growing split
// view, so a growing equivocation is detected and frozen instead of silently
// advancing to the inconsistent root. On the verified, non-violation path PollHub
// also records the hub's coverage start once (ADR-0001, set-once), caches the
// resolved did:web signing key (ADR-0009, hub_keys), and rebuilds the accepted root
// from the mirror and cross-checks it against the signed checkpoint root (ADR-0005,
// fsckMirror -> logclient.RunFsck). The poll loop is its own later step; PollHub does
// exactly one observation per call and returns.
//
// Mirror root-rebuild (ADR-0005): after the tiles are mirrored, fsckMirror runs
// RunFsck over the SQLiteFetcher to re-derive the RFC-6962 root from the local
// tiles/bundles and compare it to the signed checkpoint root. This is an in-process
// structural self-check (it shares the monitor's own LeafHashes / RFC-6962 code), not
// the fully-independent notecheck oracle. A rebuild mismatch or mirror fault is a
// genuine fault returned to the caller — NOT a self-consistency violation, so it does
// NOT freeze the hub; the checkpoint is already recorded/advanced, so a transient
// fault is re-attempted next poll.
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
// shrink, the same size with a different root = fork, or a growing pair whose
// RFC-6962 consistency proof fails to relate the prior accepted root to the new
// root = equivocation), PollHub records the violation as irreplaceable evidence,
// freezes the hub, and fires the injected alert exactly once on the not-frozen ->
// frozen transition. A frozen hub does
// not advance its cursor; the violation freezes, never crashes, so the verdict
// status is still returned with a nil error. Re-detecting on a later poll records
// the violation again (re-detection is itself evidence) but never re-alerts.
package follower

import (
	"context"
	"fmt"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/store"
)

// AlertFunc is the minimal injected alert sink the follower fires once per
// not-frozen -> frozen transition. It is a func seam (not an interface) for
// YAGNI: a test passes a counter, production passes a real transport. Delivery
// (email/webhook/log sink) and the backed-off evidence-only re-poll cadence of a
// frozen hub are out of scope here; this only signals the transition.
type AlertFunc func(hubID int64, kind string)

// glossaryStatus maps a PollHub verdict to the glossary hub-status label set the
// metrics leaf expects (verified/unresolvable/unverified/frozen), NOT
// logclient.Status.String() (which returns "rotated", a non-glossary value).
//
// frozen takes precedence: a self-consistency violation freezes the hub even
// though the signature was StatusVerified, so a frozen observation always maps to
// "frozen". Otherwise StatusVerified -> "verified", StatusUnverified ->
// "unverified", StatusUnresolvable -> "unresolvable", and StatusRotated ->
// "unverified" (the glossary has no "rotated"; an out-of-window key is an
// internally-broken / non-accepted hub whose checkpoint does not advance accepted
// state, so it folds into "unverified"). "inactive" is the realm-registry
// removed/paused state and is never produced here — the follower only polls active
// hubs.
func glossaryStatus(st logclient.Status, frozen bool) string {
	if frozen {
		return "frozen"
	}
	switch st {
	case logclient.StatusVerified:
		return "verified"
	case logclient.StatusUnverified, logclient.StatusRotated:
		return "unverified"
	case logclient.StatusUnresolvable:
		return "unresolvable"
	default:
		return "unverified"
	}
}

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
// An already-frozen hub is re-polled evidence-only and never advances accepted
// state (ADR-0006, frozen = evidence-only until a manual unfreeze, of which v1 has
// none). After the self-consistency check, if the hub is already frozen and this
// poll did NOT re-detect a fresh violation, PollHub records the verdict metric (as
// the glossary "frozen" status) and returns WITHOUT recording the checkpoint,
// setting coverage, advancing the follow cursor, caching the key, or running the
// fsck root-rebuild. A frozen hub that re-serves a fresh contradiction still flows
// through the violation branch above (re-detection is itself evidence). The
// candidate tiles ingested earlier are left mirrored — they are rebuildable
// evidence, not accepted state.
//
// m is the optional metrics registry: on every non-error verdict PollHub records
// the hub's glossary status and the observation timestamp, and on a freeze it
// increments the violations counter. m may be nil (metrics disabled), in which
// case every mutator call is skipped — metrics writes never alter control flow
// (ADR-0006). The transport/garbled-body poll-failure counter is fired by the
// caller (Tick) on PollHub's error return, since a fetch/accept fault returns
// early before the verdict is known.
//
// The returned Status is the verdict for any of the four outcomes: a non-verified
// verdict is a verdict, not a Go error, and a self-consistency violation freezes
// the hub but still returns StatusVerified with a nil error (the signature was
// valid; the violation is a separate axis — ADR-0006). The returned error is
// reserved for a genuine fault — a transport failure fetching the checkpoint, a
// verified-but-garbled body from AcceptCheckpoint (which returns a non-nil error
// alongside StatusUnverified's zero value, so the error is checked before the
// status), or a store failure. On any such fault accepted state does not advance.
func PollHub(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, observedAt time.Time, alert AlertFunc, m *metrics.Registry) (logclient.Status, error) {
	raw, err := logclient.FetchCheckpoint(ctx, fetcher, baseURL)
	if err != nil {
		return logclient.StatusUnverified, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}

	status, info, vctx, err := logclient.AcceptCheckpoint(ctx, fetcher, baseURL, raw, observedAt)
	if err != nil {
		// A verified-but-garbled body is a genuine fault, returned alongside
		// StatusUnverified's zero value; the error is checked before the status.
		return status, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}

	// Only a verified observation advances accepted state; the other verdicts are
	// reported (and the hub is still mirrored elsewhere) but not persisted here.
	if status != logclient.StatusVerified {
		recordVerdict(m, hubID, status, false, observedAt)
		return status, nil
	}

	fs, err := st.FollowState(ctx, hubID)
	if err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: follow state: %w", hubID, err)
	}

	// Mirror the hub's hash tiles and entry bundles into the local store
	// (ADR-0005) BEFORE the self-consistency check, so the candidate-size tiles
	// are present when the equivocation trigger builds its consistency proof. On a
	// growing split view this is what makes ConsistencyProofFromTiles(prevSize,
	// info.TreeSize) buildable: without the candidate tiles the proof hits a
	// missing-tile error that checkConsistency swallows as a clean pass, and the
	// hub silently advances to the inconsistent root (the closed critical gap). A
	// fetch/store fault here is a genuine transport error (NOT a violation): it is
	// surfaced so accepted state does not advance, and the next poll re-fetches any
	// missing coords via the idempotent upsert. On a violation the candidate tiles
	// are already mirrored but the cursor is not advanced — tiles are
	// rebuildable/evidence, not accepted state (partial-tile discipline + ADR-0006
	// "preserve evidence").
	if err := ingestTiles(ctx, st, fetcher, hubID, baseURL, info.TreeSize, observedAt); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: ingest tiles: %w", hubID, err)
	}

	// Self-consistency check against the prior accepted checkpoint, before any
	// record/advance: a violation must freeze (not advance) the hub (ADR-0006).
	violated, kind, prevRaw, err := checkConsistency(ctx, st, hubID, fs.LastSize, info)
	if err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}
	if violated {
		// A violation froze the hub: record the glossary "frozen" status (not the
		// StatusVerified enum, which the freeze does not change) and increment the
		// cumulative violations counter. The counter re-fires on every re-detection
		// (correct: re-detection is evidence), even though alert stays once-per-
		// transition. Pure registry writes, so placement before freeze is fine.
		recordVerdict(m, hubID, status, true, observedAt)
		if m != nil {
			m.IncViolation(hubID, string(kind))
		}
		return status, freeze(ctx, st, hubID, kind, prevRaw, raw, info, fs.Frozen, observedAt, alert)
	}

	// Already-frozen hub, clean re-poll (no fresh violation above): short-circuit to
	// evidence-only (ADR-0006). The signature was valid (StatusVerified) but a frozen
	// hub never advances accepted state, so we record the verdict metric as the
	// glossary "frozen" status and return WITHOUT RecordCheckpoint / SetCoverage /
	// AdvanceFollowState / cacheHubKey / fsckMirror. The candidate tiles ingested above
	// stay mirrored (rebuildable evidence, not accepted state). Returns (status, nil):
	// freezing is a separate axis from the signature verdict.
	if fs.Frozen {
		recordVerdict(m, hubID, status, true, observedAt)
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
	// vctx is the verifier-key context AcceptCheckpoint already resolved this poll,
	// so the cache-miss fallback reuses it instead of re-fetching did.json.
	if err := cacheHubKey(ctx, st, hubID, baseURL, raw, vctx, observedAt); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: cache hub key: %w", hubID, err)
	}
	// Rebuild the accepted root from the mirrored tiles and cross-check it against
	// the signed checkpoint root (ADR-0005, RFC-6962 root-rebuild). The tiles were
	// already mirrored by the ingestTiles call earlier in this poll (moved ahead of
	// the consistency check), so the SQLiteFetcher has tiles to read. A mismatch is a
	// genuine mirror/rebuild fault (NOT a self-consistency violation): it is surfaced
	// without freezing the hub — the checkpoint is already recorded/advanced above, so
	// a transient fault is re-attempted next poll. The verifier key (vctx.VKey) and
	// origin (info.Origin, <domain>/log) come from this poll's AcceptCheckpoint, so
	// fsckMirror reuses them instead of re-resolving did.json.
	if err := fsckMirror(ctx, st, hubID, vctx.VKey, info.Origin); err != nil {
		return status, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err)
	}
	recordVerdict(m, hubID, status, false, observedAt)
	return status, nil
}

// recordVerdict records the hub's glossary status and observation timestamp in the
// metrics registry for one non-error verdict. It is nil-safe (a nil registry =
// metrics disabled) and the only mutation point for the hub_status and
// last_observed_at series, so the three verdict branches (non-verified, freeze,
// verified-advance) all funnel through it. frozen is true only on the freeze
// branch, where it overrides the StatusVerified enum to the glossary "frozen"
// label. observedAt.Unix() is the timestamp; the registry never reads the clock.
func recordVerdict(m *metrics.Registry, hubID int64, status logclient.Status, frozen bool, observedAt time.Time) {
	if m == nil {
		return
	}
	m.SetHubStatus(hubID, glossaryStatus(status, frozen))
	m.SetLastObservedAt(hubID, observedAt.Unix())
}

// cacheHubKey refreshes the hub's did:web signing key in the hub_keys cache
// (ADR-0009: the DID document is the source of truth, this is a refreshed cache).
//
// It first tries a fetch-free fast path: recover the signed-note key id directly
// from the raw checkpoint (KeyIDFromCheckpoint) and consult store.LookupHubKey. On
// a cache hit it refreshes the cached row in place (bumping ResolvedAt) WITHOUT
// re-resolving did.json — the second did.json fetch this path used to make. The
// recovered signer name is asserted equal to the hub's origin (<domain>/log, never
// the bare domain) before the lookup, so the cached key id is keyed on the hub's
// own identity; a name mismatch, a key-id-recovery miss, or a cache miss all fall
// through to the resolve path below.
//
// The fallback reuses the verifier-key context AcceptCheckpoint already resolved
// this poll (vctx): it recovers the key id from the vkey string's middle "+<hex>+"
// field (KeyIDFromVerifier) and maps the DIDKey field-for-field into store.HubKey
// (PublicKey->PubkeyRaw, Multibase->PubkeyZ, Revoked->Revoked); the injected
// observedAt is the resolution time. The first verified poll always takes this
// fallback (the cache is cold), populating the row so subsequent polls hit the fast
// path. The follower owns this mapping so store stays a leaf (it never imports
// logclient/didweb).
func cacheHubKey(ctx context.Context, st *store.Store, hubID int64, baseURL string, raw []byte, vctx logclient.VerifiedContext, observedAt time.Time) error {
	if hit, err := cacheHubKeyFast(ctx, st, hubID, baseURL, raw, observedAt); err != nil {
		return err
	} else if hit {
		return nil
	}
	return cacheHubKeyResolve(ctx, st, hubID, vctx, observedAt)
}

// cacheHubKeyFast attempts the fetch-free cache refresh. It recovers the key id
// from raw, guards the recovered signer name against the hub's origin, looks up the
// cached row, and on a hit refreshes it in place via RecordHubKey (bumping
// ResolvedAt). It returns hit=true only when the row was found and refreshed; a
// key-id-recovery miss or a name/origin mismatch returns hit=false (fall through to
// resolve) with a nil error, while a genuine origin-derivation, query, or
// RecordHubKey fault returns a non-nil error so it is never swallowed.
func cacheHubKeyFast(ctx context.Context, st *store.Store, hubID int64, baseURL string, raw []byte, observedAt time.Time) (bool, error) {
	name, keyID, err := logclient.KeyIDFromCheckpoint(raw)
	if err != nil {
		// A garbled note on a just-verified checkpoint is unexpected; the resolve
		// path is the safe superset, so fall through rather than fail the poll.
		return false, nil
	}
	expectedOrigin, err := logclient.Origin(baseURL)
	if err != nil {
		return false, fmt.Errorf("origin: %w", err)
	}
	if name != expectedOrigin {
		// The signed-note name must equal the hub's origin (<domain>/log) or the
		// cached key id would key on the wrong identity: treat it as a cache miss.
		return false, nil
	}
	cached, found, err := st.LookupHubKey(ctx, hubID, keyID)
	if err != nil {
		return false, fmt.Errorf("lookup hub key: %w", err)
	}
	if !found {
		return false, nil
	}
	// Cache hit: refresh the existing row in place, reusing the cached key bytes and
	// bumping the resolution time. RecordHubKey's guarded UPDATE keeps the count at 1.
	if err := st.RecordHubKey(ctx, store.HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  cached.PubkeyRaw,
		PubkeyZ:    cached.PubkeyZ,
		Revoked:    cached.Revoked,
		ResolvedAt: observedAt,
	}); err != nil {
		return false, fmt.Errorf("refresh cached hub key: %w", err)
	}
	return true, nil
}

// cacheHubKeyResolve is the cache-miss fallback: it upserts the verifier-key
// context AcceptCheckpoint already resolved this poll (vctx), so it makes NO
// did.json fetch. It recovers the key id from vctx.VKey (KeyIDFromVerifier) and
// maps vctx.Key into store.HubKey. A key-id-recovery failure here is unexpected on
// the just-verified path (vctx.VKey came straight from the verified resolution) and
// is returned to the caller rather than swallowed.
func cacheHubKeyResolve(ctx context.Context, st *store.Store, hubID int64, vctx logclient.VerifiedContext, observedAt time.Time) error {
	keyID, err := logclient.KeyIDFromVerifier(vctx.VKey)
	if err != nil {
		return fmt.Errorf("recover key id: %w", err)
	}
	return st.RecordHubKey(ctx, store.HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  vctx.Key.PublicKey,
		PubkeyZ:    vctx.Key.Multibase,
		Revoked:    vctx.Key.Revoked,
		ResolvedAt: observedAt,
	})
}

// fsckMirror rebuilds the hub's accepted root from the local mirror and verifies it
// against the signed checkpoint root (ADR-0005). The verifier key (vkey) and the
// signed-note origin (origin, <domain>/log = info.Origin) come from this poll's
// AcceptCheckpoint — the key's validity was already checked there — so fsckMirror
// makes NO did.json fetch of its own. It builds a read-only store.SQLiteFetcher over
// the just-ingested tiles/bundles and runs logclient.RunFsck, which re-hashes each
// entry bundle, re-derives the lower hash tiles, and compares the rebuilt RFC-6962
// root to the checkpoint's claimed root.
//
// RunFsck is an in-process STRUCTURAL self-check: it shares the monitor's own
// LeafHashes / RFC-6962 code, so it catches mirror corruption and rebuild bugs but is
// NOT the fully-independent oracle (notecheck, run in CI, is that). A non-nil return
// is a root-rebuild mismatch or a mirror fault — NOT a self-consistency violation
// (freezing is reserved for the three checkConsistency triggers). PollHub surfaces it
// to the caller without freezing the hub; since the checkpoint is already
// recorded/advanced, a transient mirror fault is simply re-attempted next poll. The
// RunFsck error is wrapped with %w.
func fsckMirror(ctx context.Context, st *store.Store, hubID int64, vkey, origin string) error {
	if err := logclient.RunFsck(ctx, vkey, origin, store.SQLiteFetcher{Store: st, HubID: hubID}); err != nil {
		return fmt.Errorf("fsck: %w", err)
	}
	return nil
}

// checkConsistency runs the three RFC-6962 self-consistency triggers against the
// prior accepted checkpoint for a hub. prevSize is FollowState.LastSize; the
// CheckShrink/CheckFork/CheckEquivocation prevSize>0 guards mean a fresh-store zero
// never trips a violation, so a never-advanced hub is always clean. The prior root
// and raw bytes come from a checkpoints lookup at prevSize, since follow_state does
// not persist the root; if no checkpoint is stored at that size (a hub that advanced
// before this code existed), the root-dependent fork/equivocation checks are skipped
// while the size-only shrink check still runs. On a true verdict it returns the
// matching violation kind and the prior raw bytes (RawA evidence). The three
// triggers are mutually exclusive by size — shrink is next<prev, fork is
// next==prev, equivocation is the growing-pair next>prev case — so they are
// evaluated in shrink → fork → equivocation order and the first true kind is used.
//
// Equivocation sources its RFC-6962 consistency proof from the LOCAL mirror only —
// ConsistencyProofFromTiles over a store.SQLiteFetcher — and never re-hits the hub.
// The roots compared are the prior ACCEPTED root at prevSize and the new
// observation's root (info.Root), never the contradicting-evidence row (ADR-0006).
//
// Error vs. violation discipline (ADR-0006 "freeze, never crash"): CheckEquivocation
// already turns a non-verifying proof into a (violated=true, err=nil) verdict. But
// ConsistencyProofFromTiles returns a genuine Go error on a tile-fetch/parse fault —
// most commonly a missing tile (a wrapped os.ErrNotExist), since production does not
// yet mirror tiles (that is M2 work). Such an error is NOT an equivocation verdict:
// freezing on a missing tile would be a false positive, and aborting the poll would
// break the loop. So a proof-build error is treated as "cannot evaluate equivocation
// this poll" — the equivocation branch is skipped (no violation, no crash) and the
// poll proceeds. The branch only becomes load-bearing once M2 mirrors tiles. This
// swallow is deliberately narrow: it suppresses only the proof-build error so a
// missing tile cannot freeze a hub; a genuine st failure surfaces via CheckpointAt
// above.
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
		// Equivocation: the only growing-pair trigger (info.TreeSize > prevSize),
		// reached only when the prior accepted checkpoint is on record (prevFound) so
		// prevRoot is the real prior root, not a zero placeholder.
		if !prevFound || info.TreeSize <= prevSize {
			return false, "", nil, nil
		}
		fetcher := store.SQLiteFetcher{Store: st, HubID: hubID}
		proofHashes, perr := logclient.ConsistencyProofFromTiles(ctx, fetcher.ReadTile, prevSize, info.TreeSize)
		if perr != nil {
			// Cannot build the proof from the mirror (most often: tiles not mirrored
			// yet). Skip the equivocation branch — never freeze on a missing tile.
			return false, "", nil, nil
		}
		eq, eerr := logclient.CheckEquivocation(prevSize, prevRoot, info.TreeSize, info.Root, proofHashes)
		if eerr != nil {
			return false, "", nil, fmt.Errorf("check equivocation at sizes %d->%d: %w", prevSize, info.TreeSize, eerr)
		}
		if eq {
			return true, logclient.ViolationEquivocation, prevRaw, nil
		}
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
