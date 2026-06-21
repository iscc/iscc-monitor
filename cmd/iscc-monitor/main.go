// Command iscc-monitor is the monitor process entrypoint: it wires the four M1
// leaves — config.Load, registry.Parse, store.Open/UpsertHub, and follower.Loop
// — into a running follower that polls every hub in the realm document on a
// cadence and persists its observations into the network's SQLite database. It
// also constructs the in-memory metrics registry, threads it into the loop, and
// serves it at /metrics on the configured address so production both collects and
// exposes the alert-worthy series.
//
// main stays thin and owns process exit (config/file/store failures print to
// stderr and exit non-zero); all the registry -> store -> target wiring lives in
// the testable registerHubs helper. The poll loop runs until SIGINT, which a
// signal-bound context cancels so Loop.Run returns cleanly; the /metrics server
// runs in a background goroutine and is shut down on the same context so serving
// never blocks polling.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/iscc/iscc-monitor/internal/config"
	"github.com/iscc/iscc-monitor/internal/follower"
	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/metricshttp"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tilesserve"
)

// hubRoute is one hub's mirror mount point: its store hub_id and its log origin
// (<domain>/log, e.g. sb0.iscc.id/log). The router mounts tilesserve.Handler at
// "/" + Origin + "/" so a request to /<origin>/checkpoint reaches that hub's
// SQLiteFetcher. It is package-local to main — the follower needs no origin, so
// HubTarget does not carry one, and the origin is re-derived here from the same
// logclient.Origin registerHubs already uses.
type hubRoute struct {
	HubID  int64
	Origin string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "iscc-monitor:", err)
		os.Exit(1)
	}
}

// run loads configuration, registers every realm hub into the store, and drives
// the follower loop until the signal-bound context is cancelled. It returns an
// error for any startup or shutdown fault so main owns the single os.Exit; on a
// clean SIGINT shutdown Loop.Run returns ctx.Err(), which run reports as nil.
func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(cfg.RealmPath)
	if err != nil {
		return fmt.Errorf("read realm document %q: %w", cfg.RealmPath, err)
	}
	entries, err := registry.Parse(data)
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	targets, routes, err := registerHubs(ctx, st, entries)
	if err != nil {
		return err
	}

	m := metrics.New()
	go serveMetrics(ctx, cfg.Addr, st, routes, m, logger)

	loop := &follower.Loop{
		Store:   st,
		Fetcher: logclient.NewHTTPFetcher(nil),
		Targets: targets,
		Normal:  cfg.Normal,
		Frozen:  cfg.Frozen,
		Alert:   alertFunc(logger),
		Logger:  logger,
		Metrics: m,
	}
	if err := loop.Run(ctx); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

// serveMetrics runs the monitor's single HTTP server until ctx is cancelled. The
// one server serves both GET /metrics (via metricshttp.Handler) and each followed
// hub's mirrored tlog-tiles artifacts (via mirrorHandler over the same mux), so
// there is exactly one listener. It is started in a background goroutine so it
// never blocks the follower loop (the foreground blocker). On ctx cancellation it
// shuts the server down with a short-timeout context so a SIGINT exits promptly.
// http.ErrServerClosed from ListenAndServe is the normal-shutdown signal
// (mirroring how Run treats context.Canceled as clean) and is logged, not
// surfaced; any other listen error is logged so a misconfigured address is never
// silent.
func serveMetrics(ctx context.Context, addr string, st *store.Store, routes []hubRoute, m *metrics.Registry, logger *slog.Logger) {
	srv := &http.Server{Addr: addr, Handler: buildMux(st, routes, m)}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.ErrorContext(ctx, "metrics server shutdown failed", "err", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.ErrorContext(ctx, "metrics server failed", "addr", addr, "err", err)
	}
}

// buildMux assembles the monitor's single request multiplexer: GET /metrics plus
// every hub's mirror subtree from mirrorHandler. It is factored out of
// serveMetrics so the full routing (metrics + per-hub mirror) is unit-testable
// against an httptest.ResponseRecorder without binding a socket.
func buildMux(st *store.Store, routes []hubRoute, m *metrics.Registry) http.Handler {
	mux := mirrorHandler(st, routes)
	mux.Handle("/metrics", metricshttp.Handler(m))
	return mux
}

// mirrorHandler builds the per-hub mirror router: for each route it mounts a
// tilesserve.Handler (reading that hub's BLOBs through a read-only
// store.SQLiteFetcher) at the subtree prefix "/" + Origin + "/" (e.g.
// /sb0.iscc.id/log/), stripping that prefix down to the single leading slash the
// handler trims. A request to /<origin>/checkpoint therefore reaches the handler
// as /checkpoint. The trailing slash makes http.ServeMux do subtree matching, so
// all of /tile/... and /tile/entries/... under the prefix route to the same
// handler; an unknown top-level prefix falls through to the mux's default 404. The
// origin must be the full <domain>/log — a request missing the /log segment does
// not match the prefix and 404s. All routes share the store's single open
// connection (reads serialize on it, ADR-0005/0007); no second DB handle is
// opened.
func mirrorHandler(st *store.Store, routes []hubRoute) *http.ServeMux {
	mux := http.NewServeMux()
	for _, r := range routes {
		prefix := "/" + r.Origin + "/"
		h := tilesserve.Handler(store.SQLiteFetcher{Store: st, HubID: r.HubID})
		mux.Handle(prefix, http.StripPrefix(prefix, h))
	}
	return mux
}

// registerHubs registers each realm entry in the store and returns both the
// follower targets to poll and the mirror routes to serve. For each entry it
// derives the log origin (<domain>/log, never the bare domain) and upserts the
// hub, building a HubTarget from the returned hub_id and the entry's base URL and
// a hubRoute from the same hub_id and origin. UpsertHub is idempotent on the
// domain, so a re-run returns identical hub_ids. The first error short-circuits,
// naming the offending domain. The targets and routes are index-aligned and
// derive from the same upsert, so the poll set and the served mirror set never
// diverge.
func registerHubs(ctx context.Context, st *store.Store, entries []registry.Entry) ([]follower.HubTarget, []hubRoute, error) {
	targets := make([]follower.HubTarget, 0, len(entries))
	routes := make([]hubRoute, 0, len(entries))
	for _, e := range entries {
		org, err := logclient.Origin(e.BaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("derive origin for %q: %w", e.Domain, err)
		}
		id, err := st.UpsertHub(ctx, e.Domain, org, e.BaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("register hub %q: %w", e.Domain, err)
		}
		targets = append(targets, follower.HubTarget{HubID: id, BaseURL: e.BaseURL})
		routes = append(routes, hubRoute{HubID: id, Origin: org})
	}
	return targets, routes, nil
}

// alertFunc builds the freeze alert sink over the given logger: it emits a
// structured WARN record on each not-frozen -> frozen transition (PollHub gates
// it once per transition). A freeze is an operator-actionable condition rather
// than a process fault, so WARN — not ERROR — is the right severity; real
// delivery (email/webhook) is a later step, this only signals the transition so a
// frozen hub is never silent. The logger is captured here rather than read from a
// global so the alert sink is injectable and testable.
func alertFunc(logger *slog.Logger) follower.AlertFunc {
	return func(hubID int64, kind string) {
		logger.Warn("hub frozen", "hub_id", hubID, "kind", kind)
	}
}
