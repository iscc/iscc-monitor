// This file holds the OpenTimestamps background upgrade loop's deterministic
// control core: OTSTick drives one pass over the store's due pending stamped roots,
// asking an injected Upgrader whether each is Bitcoin-confirmed yet and either
// marking it confirmed (MarkOTSUpgraded) or recording a backed-off retry
// (MarkOTSAttempted: Attempts++, NextRetry pushed out via backoff). It is the pure,
// injected-now analogue of loop.go's Tick — all wall-clock stays in the (later)
// ticker wrapper, so tests never sleep and the pass is fully deterministic.
//
// OTS is best-effort and NEVER blocks the follower (ADR-0004): the upgrade loop is a
// separate driver off the poll path, so an Upgrader transport fault or a calendar
// "not yet confirmed" answer never freezes a hub and never surfaces to PollHub. A
// flaky single row logs-and-continues exactly like Tick; the pass attempts every due
// row and returns the first error, never aborting. The Upgrader is a func seam (not
// an interface, matching AlertFunc, YAGNI), so internal/follower stays import-free of
// any anchoring package and the real calendar-HTTP client becomes a closure of this
// type in a later sub-step.
package follower

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/iscc/iscc-monitor/internal/store"
)

// UpgradeResult is the per-root outcome the Upgrader reports. Confirmed is true when
// the stamped root's OpenTimestamps proof has been upgraded to a Bitcoin-confirmed
// attestation; OTSBytes then carries the serialized upgraded proof and BTCHeight the
// confirming Bitcoin block height (both persisted by MarkOTSUpgraded). When Confirmed
// is false the root is still pending (the calendar has not yet seen a Bitcoin
// attestation) and the loop records a backed-off retry instead.
type UpgradeResult struct {
	Confirmed bool
	OTSBytes  []byte
	BTCHeight int64
}

// Upgrader asks whether one stamped root is Bitcoin-confirmed yet, returning the
// per-root outcome. It is a func seam (not an interface, matching AlertFunc) so
// internal/follower imports no anchoring package: the real OpenTimestamps
// calendar-HTTP client becomes a closure of this type in a later sub-step. A non-nil
// error is a transport fault (calendar unreachable, garbled response); the loop
// treats it as best-effort — it records a back-off and logs-and-continues, never
// freezing a hub or aborting the pass (OTS never blocks the follower, ADR-0004).
type Upgrader func(ctx context.Context, r store.OTSRecord) (UpgradeResult, error)

// otsBackoffBase is the first retry interval for a still-pending or erroring root,
// doubled on each subsequent attempt up to otsBackoffShiftCap and clamped to
// otsBackoffMax. The exact cadence is not safety-critical (OTS is best-effort) — it
// only needs to be deterministic and monotonic so the loop re-polls a pending root
// patiently without spamming a calendar.
const (
	otsBackoffBase     = time.Hour
	otsBackoffShiftCap = 5 // base << 5 = 32h, before the max clamp
	otsBackoffMax      = 24 * time.Hour
)

// backoff returns the retry interval for the given attempt number as a capped
// exponential: otsBackoffBase doubled (attempts-1) times, clamped to otsBackoffMax.
// It is a pure, deterministic, monotonic-up-to-the-cap function so it is directly
// golden-testable. attempts is the new (post-increment) count, so the first retry
// (attempts==1) waits one base interval. A non-positive attempts is treated as 1.
func backoff(attempts int64) time.Duration {
	shift := attempts - 1
	if shift < 0 {
		shift = 0
	}
	if shift > otsBackoffShiftCap {
		shift = otsBackoffShiftCap
	}
	d := otsBackoffBase << uint(shift)
	if d > otsBackoffMax {
		d = otsBackoffMax
	}
	return d
}

// OTSTick makes one upgrade pass over the store's due pending stamped roots at now,
// the pure injected-now analogue of loop.go's Tick. It reads st.PendingOTS(ctx, now)
// (already filtered to rows whose back-off has elapsed), oldest-first, and for each
// row asks the injected Upgrader whether the root is Bitcoin-confirmed:
//
//   - Confirmed → MarkOTSUpgraded records the upgraded proof / height / instant and
//     the row drops out of future PendingOTS;
//   - not confirmed, no error → MarkOTSAttempted records a backed-off retry
//     (Attempts+1, NextRetry = now+backoff) so the row re-surfaces once it elapses;
//   - Upgrader error (transport fault) → ALSO a back-off, then log-and-continue.
//
// All wall-clock stays out of OTSTick (the caller injects now), exactly like Tick.
// Errors do not abort the pass: an Upgrader transport fault or a store-write fault on
// one row is logged with the hub_id/tree_size, folded into the first error, and the
// pass continues to the next due row — OTS is best-effort and never blocks the
// follower (ADR-0004), so the first error is returned only for observability, never
// to freeze a hub or stop the loop. logger is the optional structured-logging sink; a
// nil logger falls back to slog.Default().
func OTSTick(ctx context.Context, st *store.Store, up Upgrader, now time.Time, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	pending, err := st.PendingOTS(ctx, now)
	if err != nil {
		return fmt.Errorf("follower.OTSTick: pending: %w", err)
	}
	var firstErr error
	for _, r := range pending {
		res, upErr := up(ctx, r)
		if upErr == nil && res.Confirmed {
			if err := st.MarkOTSUpgraded(ctx, r.HubID, r.TreeSize, r.Root, res.OTSBytes, res.BTCHeight, now); err != nil {
				logger.ErrorContext(ctx, "ots mark upgraded failed", "hub_id", r.HubID, "tree_size", r.TreeSize, "err", err)
				if firstErr == nil {
					firstErr = err
				}
			}
			continue
		}
		// Still pending (calendar not yet Bitcoin-confirmed) or a transport fault:
		// record a backed-off retry so the row re-surfaces once NextRetry elapses.
		attempts := r.Attempts + 1
		if err := st.MarkOTSAttempted(ctx, r.HubID, r.TreeSize, r.Root, attempts, now.Add(backoff(attempts))); err != nil {
			logger.ErrorContext(ctx, "ots mark attempted failed", "hub_id", r.HubID, "tree_size", r.TreeSize, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
		if upErr != nil {
			logger.ErrorContext(ctx, "ots upgrade failed", "hub_id", r.HubID, "tree_size", r.TreeSize, "err", upErr)
			if firstErr == nil {
				firstErr = fmt.Errorf("follower.OTSTick: hub %d size %d: %w", r.HubID, r.TreeSize, upErr)
			}
		}
	}
	return firstErr
}
