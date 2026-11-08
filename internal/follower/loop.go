// This file holds the per-network poll loop that drives PollHub on a cadence.
// It turns the follower from "one observation per explicit call" into a driver
// that polls every registered hub from a single goroutine — the single writer
// per DB (ADR-0005/0007) — so all store writes serialize without a goroutine per
// hub. A frozen hub is re-polled only on a longer, backed-off cadence so it keeps
// recording evidence (re-detection is itself evidence) without spamming the hub
// or starving the unfrozen hubs (ADR-0006).
//
// The loop owns no fetch/verify/freeze logic of its own: PollHub stays the single
// source of poll behavior, and a frozen hub re-polled through PollHub already
// re-records the violation and never re-alerts (wasFrozen gates the alert). The
// loop's only added responsibility is *when* a hub is due, never *what* a poll
// does. The cadence is decided by the pure due() predicate against an injected
// now, so Tick is fully deterministic and tests never sleep; wall-clock time
// appears only in Run's ticker plumbing.
package follower

import (
	"context"
	"log/slog"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/store"
)

// HubTarget is one hub the loop polls, pre-resolved by the caller. HubID is the
// store's hub_id and BaseURL the hub's scheme+host (e.g. "https://sb0.iscc.id");
// PollHub derives the origin and verifier key from BaseURL. The frozen flag is
// deliberately NOT carried here — it is read fresh from store.FollowState each
// tick so a freeze that happens mid-run takes effect on the next pass.
type HubTarget struct {
	HubID   int64
	BaseURL string
}

// Loop drives PollHub over a fixed set of targets on a cadence from one
// goroutine. Normal is the poll interval for a clean hub; Frozen is the longer
// backed-off interval for a frozen hub (Frozen >= Normal encodes the back-off).
// Alert is fired once per not-frozen -> frozen transition (PollHub gates it).
// Logger is the optional structured-logging sink; a nil Logger falls back to
// slog.Default() via the logger() accessor, so a bare &Loop{…} works unchanged.
// Metrics is the optional in-memory metrics registry threaded through to PollHub;
// a nil Metrics disables metrics (every mutator call is skipped), so a bare
// &Loop{…} works unchanged. lastPoll holds each hub's most-recent successful-poll
// time in memory (keyed by HubID); v1 does not persist it (no schema change), so a
// restart re-polls every hub immediately, which is harmless — PollHub is
// idempotent on an unchanged checkpoint. Because Tick runs in the single owning
// goroutine, lastPoll needs no lock.
type Loop struct {
	Store    *store.Store
	Fetcher  logclient.Fetcher
	Targets  []HubTarget
	Normal   time.Duration
	Frozen   time.Duration
	Alert    AlertFunc
	Logger   *slog.Logger
	Metrics  *metrics.Registry
	lastPoll map[int64]time.Time
}

// logger returns the Loop's injected Logger or slog.Default() when none is set,
// so every log call site is nil-safe and a bare &Loop{…} (as in the tests and
// the binary) needs no logger to run.
func (l *Loop) logger() *slog.Logger {
	if l.Logger != nil {
		return l.Logger
	}
	return slog.Default()
}

// due reports whether a hub is due for a poll at now given when it was last
// polled and whether it is frozen. The interval is Frozen when the hub is frozen
// (the backed-off evidence-only cadence, ADR-0006) else Normal. A zero lastPoll
// (never polled) is always due, so a fresh hub is polled on the first tick. It is
// a pure predicate — the one easily golden-testable unit — and is the only place
// the back-off decision lives.
func due(frozen bool, lastPoll, now time.Time, normal, frozenInterval time.Duration) bool {
	if lastPoll.IsZero() {
		return true
	}
	interval := normal
	if frozen {
		interval = frozenInterval
	}
	return now.Sub(lastPoll) >= interval
}

// Tick makes one pass over the targets in the owning goroutine, polling each hub
// that is due at now through PollHub so all store writes serialize (single writer
// per DB, ADR-0005/0007). For each target it reads the freeze flag fresh from
// store.FollowState (frozen is not carried on the target) and consults due(); a
// frozen hub is re-polled only at the longer Frozen interval, which is exactly the
// backed-off evidence-only re-poll. A hub polled successfully records its poll time
// in lastPoll so a second Tick at the same now does not re-poll it.
//
// Errors do not kill the pass: a flaky single hub (transport fault, garbled body,
// store error) must never stall the whole network's loop, so Tick logs each fault
// as a structured record (with the hub_id), attempts every due hub, and returns
// the first error encountered (nil if all succeeded). A hub whose poll errored is
// NOT marked polled, so it is retried on the next due tick. Reading FollowState is
// itself a store read that can fault; that error is treated the same way (logged,
// recorded as the first error, the hub skipped, the pass continues).
//
// A PollHub error also increments the poll-failure counter for that hub (the
// transport/garbled-body fault path): PollHub returns early before the verdict is
// known on such a fault, so the counter fires here, on the same error branch as
// the log, rather than inside PollHub. The increment is nil-safe (l.Metrics may be
// nil) and never alters the log-and-continue control flow.
func (l *Loop) Tick(ctx context.Context, now time.Time) error {
	if l.lastPoll == nil {
		l.lastPoll = make(map[int64]time.Time)
	}
	var firstErr error
	for _, target := range l.Targets {
		fs, err := l.Store.FollowState(ctx, target.HubID)
		if err != nil {
			l.logger().ErrorContext(ctx, "follow state read failed", "hub_id", target.HubID, "err", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !due(fs.Frozen, l.lastPoll[target.HubID], now, l.Normal, l.Frozen) {
			continue
		}
		if _, err := PollHub(ctx, l.Store, l.Fetcher, target.HubID, target.BaseURL, now, l.Alert, l.Metrics); err != nil {
			l.logger().ErrorContext(ctx, "poll hub failed", "hub_id", target.HubID, "err", err)
			if l.Metrics != nil {
				l.Metrics.IncPollFailure(target.HubID)
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		l.lastPoll[target.HubID] = now
	}
	return firstErr
}

// Run drives Tick on a time.Ticker at the Normal interval until ctx is cancelled,
// returning ctx.Err() on cancellation. It owns the single ticker and calls Tick in
// this one goroutine, so the single-writer discipline holds. The ticker fires at
// the Normal cadence (the most-frequent interval any hub needs); due() throttles a
// frozen hub down to the Frozen cadence within those ticks. A Tick error does not
// stop the loop — a flaky hub never aborts the network's polling — so Run emits
// the per-tick error as a structured log record and continues to the next tick;
// only context cancellation ends the loop. Run never panics.
func (l *Loop) Run(ctx context.Context) error {
	ticker := time.NewTicker(l.Normal)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			// A per-tick error is logged but intentionally not propagated: one
			// flaky hub must not abort the whole network's loop. Tick already
			// isolates per-hub faults; logging makes the previously-swallowed
			// error observable without stopping the loop.
			if err := l.Tick(ctx, t); err != nil {
				l.logger().ErrorContext(ctx, "poll tick failed", "err", err)
			}
		}
	}
}
