// This file holds the OpenTimestamps background upgrade loop's deterministic
// control core: OTSTick drives one pass over the store's due pending stamped roots,
// first submitting any not-yet-stamped root's digest to a calendar (the injected
// Stamper, filling its empty OTSBytes), then asking an injected Upgrader whether each
// stamped root is Bitcoin-confirmed yet and either marking it confirmed
// (MarkOTSUpgraded) or recording a backed-off retry (MarkOTSAttempted: Attempts++,
// NextRetry pushed out via backoff). It is the pure, injected-now analogue of
// loop.go's Tick — all wall-clock stays in the ticker wrapper, so tests never sleep
// and the pass is fully deterministic.
//
// OTS is best-effort and NEVER blocks the follower (ADR-0004): the upgrade loop is a
// separate driver off the poll path, so a Stamper/Upgrader transport fault or a
// calendar "not yet confirmed" answer never freezes a hub and never surfaces to
// PollHub. The follower writes a pending row with an empty OTSBytes sentinel on the
// poll path (a synchronous calendar round-trip there would violate "OTS never
// blocks"); OTSTick does the calendar submission off the poll path. A flaky single
// row logs-and-continues exactly like Tick; the pass attempts every due row and
// returns the first error, never aborting. Both Stamper and Upgrader are func seams
// (not interfaces, matching AlertFunc, YAGNI), so internal/follower stays import-free
// of any anchoring package and the real calendar-HTTP clients become closures of
// these types wired in cmd/iscc-monitor.
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

// Stamper submits one not-yet-stamped root's digest to an OpenTimestamps calendar,
// returning the serialized initial pending proof bytes (the calendar-signed pending
// sequence the Upgrader later upgrades) and the calendar URL(s) the submission used.
// Like Upgrader it is a func seam (not an interface, matching AlertFunc) so
// internal/follower imports no anchoring package: the real calendar-HTTP client is a
// closure of this type wired in cmd/iscc-monitor. A non-nil error is a transport
// fault (calendar unreachable, garbled response); the loop treats it as best-effort —
// it records a back-off and logs-and-continues, never freezing a hub or aborting the
// pass (OTS never blocks the follower, ADR-0004). The follower writes pending rows
// with an empty OTSBytes sentinel on the poll path (stamping never blocks polling);
// OTSTick runs the Stamper off the poll path to fill those rows in.
type Stamper func(ctx context.Context, root [32]byte) (otsBytes []byte, calendars string, err error)

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

// OTSTick makes one stamp-then-upgrade pass over the store's due pending roots at
// now, the pure injected-now analogue of loop.go's Tick. It reads
// st.PendingOTS(ctx, now) (already filtered to rows whose back-off has elapsed),
// oldest-first, and for each row:
//
//   - if the row is not yet stamped (len(r.OTSBytes) == 0 — the empty sentinel the
//     follower's poll path writes), it submits the root's digest to a calendar via
//     the injected Stamper, persists the returned proof / calendar URL(s) with
//     MarkOTSStamped, and continues; the next tick re-reads the now-stamped row and
//     upgrades it. (A nil Stamper skips stamping, leaving the row empty — a bare call
//     path stays well-defined, mirroring the nil-Logger discipline.) A Stamper
//     transport fault records a back-off (MarkOTSAttempted) and logs-and-continues,
//     exactly like an Upgrader fault — a stamp fault never aborts the pass and never
//     freezes a hub (OTS never blocks the follower, ADR-0004).
//
// For an already-stamped row it asks the injected Upgrader whether the root is
// Bitcoin-confirmed:
//
//   - Confirmed → MarkOTSUpgraded records the upgraded proof / height / instant and
//     the row drops out of future PendingOTS;
//   - not confirmed, no error → MarkOTSAttempted records a backed-off retry
//     (Attempts+1, NextRetry = now+backoff) so the row re-surfaces once it elapses;
//   - Upgrader error (transport fault) → ALSO a back-off, then log-and-continue.
//
// All wall-clock stays out of OTSTick (the caller injects now), exactly like Tick.
// Errors do not abort the pass: a Stamper/Upgrader transport fault or a store-write
// fault on one row is logged with the hub_id/tree_size, folded into the first error,
// and the pass continues to the next due row — OTS is best-effort and never blocks
// the follower (ADR-0004), so the first error is returned only for observability,
// never to freeze a hub or stop the loop. logger is the optional structured-logging
// sink; a nil logger falls back to slog.Default().
func OTSTick(ctx context.Context, st *store.Store, stamper Stamper, up Upgrader, now time.Time, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	pending, err := st.PendingOTS(ctx, now)
	if err != nil {
		return fmt.Errorf("follower.OTSTick: pending: %w", err)
	}
	var firstErr error
	for _, r := range pending {
		// A not-yet-stamped pending row (the empty-OTSBytes sentinel the poll path
		// writes) must first get its calendar-submitted proof before it can be
		// upgraded. Stamping happens here, off the poll path, so OTS never blocks the
		// follower. On success persist the proof and continue — the next tick upgrades
		// the now-stamped row (mirroring the back-off flow's "re-surface next tick").
		if stamper != nil && len(r.OTSBytes) == 0 {
			var root [32]byte
			copy(root[:], r.Root)
			otsBytes, calendars, stampErr := stamper(ctx, root)
			if stampErr != nil {
				attempts := r.Attempts + 1
				if err := st.MarkOTSAttempted(ctx, r.HubID, r.TreeSize, r.Root, attempts, now.Add(backoff(attempts))); err != nil {
					logger.ErrorContext(ctx, "ots mark attempted failed", "hub_id", r.HubID, "tree_size", r.TreeSize, "err", err)
					if firstErr == nil {
						firstErr = err
					}
				}
				logger.ErrorContext(ctx, "ots stamp failed", "hub_id", r.HubID, "tree_size", r.TreeSize, "err", stampErr)
				if firstErr == nil {
					firstErr = fmt.Errorf("follower.OTSTick: stamp hub %d size %d: %w", r.HubID, r.TreeSize, stampErr)
				}
				continue
			}
			if err := st.MarkOTSStamped(ctx, r.HubID, r.TreeSize, r.Root, otsBytes, calendars); err != nil {
				logger.ErrorContext(ctx, "ots mark stamped failed", "hub_id", r.HubID, "tree_size", r.TreeSize, "err", err)
				if firstErr == nil {
					firstErr = err
				}
			}
			continue
		}
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
