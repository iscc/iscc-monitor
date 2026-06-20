// Command iscc-monitor is the monitor process entrypoint: it wires the four M1
// leaves — config.Load, registry.Parse, store.Open/UpsertHub, and follower.Loop
// — into a running follower that polls every hub in the realm document on a
// cadence and persists its observations into the network's SQLite database.
//
// main stays thin and owns process exit (config/file/store failures print to
// stderr and exit non-zero); all the registry -> store -> target wiring lives in
// the testable registerHubs helper. The poll loop runs until SIGINT, which a
// signal-bound context cancels so Loop.Run returns cleanly.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/iscc/iscc-monitor/internal/config"
	"github.com/iscc/iscc-monitor/internal/follower"
	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
)

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

	targets, err := registerHubs(ctx, st, entries)
	if err != nil {
		return err
	}

	loop := &follower.Loop{
		Store:   st,
		Fetcher: logclient.NewHTTPFetcher(nil),
		Targets: targets,
		Normal:  cfg.Normal,
		Frozen:  cfg.Frozen,
		Alert:   alert,
	}
	if err := loop.Run(ctx); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

// registerHubs registers each realm entry in the store and returns the follower
// targets to poll. For each entry it derives the log origin (<domain>/log, never
// the bare domain) and upserts the hub, building a HubTarget from the returned
// hub_id and the entry's base URL. UpsertHub is idempotent on the domain, so a
// re-run returns identical hub_ids. The first error short-circuits, naming the
// offending domain.
func registerHubs(ctx context.Context, st *store.Store, entries []registry.Entry) ([]follower.HubTarget, error) {
	targets := make([]follower.HubTarget, 0, len(entries))
	for _, e := range entries {
		org, err := logclient.Origin(e.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("derive origin for %q: %w", e.Domain, err)
		}
		id, err := st.UpsertHub(ctx, e.Domain, org, e.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("register hub %q: %w", e.Domain, err)
		}
		targets = append(targets, follower.HubTarget{HubID: id, BaseURL: e.BaseURL})
	}
	return targets, nil
}

// alert is the placeholder freeze alert: it logs the not-frozen -> frozen
// transition to stderr. Real delivery (email/webhook) is a later step; this only
// signals the transition so a frozen hub is never silent.
func alert(hubID int64, kind string) {
	fmt.Fprintf(os.Stderr, "iscc-monitor: ALERT hub %d frozen: %s violation\n", hubID, kind)
}
