// Command iscc-monitor is the monitor process entrypoint: it wires the core leaves
// — config.Load, the realm Hub-List loader (loadRealm over registry.ParseHubList),
// store.Open/UpsertHub, and follower.Loop — into a running follower that polls
// every hub in the realm document on a cadence and persists its observations into
// the network's SQLite database. The realm source is re-fetched hourly
// (runRealmRefresh) so the certificate's hub_id resolver tracks the authoritative
// Hub-List without a redeploy. It also constructs the in-memory metrics registry,
// threads it into the loop, and serves it at /metrics on the configured address so
// production both collects and exposes the alert-worthy series.
//
// main stays thin and owns process exit (config/file/store failures print to
// stderr and exit non-zero); all the registry -> store -> target wiring lives in
// the testable registerHubs helper. The poll loop runs until SIGINT or SIGTERM,
// which a signal-bound context cancels so Loop.Run returns cleanly; the /metrics
// server runs in a background goroutine and is shut down on the same context so
// serving never blocks polling.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/iscc/iscc-monitor/internal/certificate"
	"github.com/iscc/iscc-monitor/internal/config"
	"github.com/iscc/iscc-monitor/internal/corsmw"
	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/docs"
	"github.com/iscc/iscc-monitor/internal/dossier"
	"github.com/iscc/iscc-monitor/internal/follower"
	"github.com/iscc/iscc-monitor/internal/healthz"
	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/metricshttp"
	"github.com/iscc/iscc-monitor/internal/openapi"
	"github.com/iscc/iscc-monitor/internal/otsclient"
	"github.com/iscc/iscc-monitor/internal/proofserve"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tilesserve"
	"github.com/iscc/iscc-monitor/internal/version"
	"github.com/iscc/iscc-monitor/internal/web"
)

// hubRoute is one hub's mirror mount point: its store hub_id, its bare domain
// (e.g. sb0.iscc.id), and its log origin (<domain>/log, e.g. sb0.iscc.id/log). The
// router mounts tilesserve.Handler at "/" + Origin + "/" so a request to
// /<origin>/checkpoint reaches that hub's SQLiteFetcher, and dossier.Handler at the
// exact bare-domain path "/" + Domain so /<domain> serves the per-hub dossier page.
// It is package-local to main — the follower needs neither domain nor origin, so
// HubTarget does not carry them, and both are re-derived here from the same realm
// entry / logclient.Origin registerHubs already uses.
type hubRoute struct {
	HubID  int64
	Domain string
	Origin string
}

// reservedMountNames is the set of single-label tokens that, used as a realm
// domain, would mount the dossier at a path that collides with a built-in exact
// route (metrics, healthz, version, docs, the two openapi document routes) or the
// web.Prefix subtree segment (_ds). The _ds and openapi.* entries are derived from
// the package consts (not hardcoded) so they track the consts if they change.
// registerHubs rejects any realm Domain in this set before mounting so a
// misconfigured realm fails loudly at startup instead of panicking http.ServeMux.
var reservedMountNames = map[string]struct{}{
	"metrics":                     {},
	"healthz":                     {},
	"version":                     {},
	"docs":                        {},
	strings.Trim(web.Prefix, "/"): {},
	strings.TrimPrefix(openapi.JSONPath, "/"): {},
	strings.TrimPrefix(openapi.YAMLPath, "/"): {},
}

// otsUpgradeInterval is the cadence of the background OpenTimestamps upgrade loop:
// how often runOTSLoop asks the calendar whether each pending stamped root has been
// Bitcoin-confirmed yet. Daily matches the milestone ("upgrade loop … pending ->
// Bitcoin-confirmed") and the reality of Bitcoin confirmation latency — a root takes
// hours to confirm, so polling a calendar more often only spams it (OTSTick's own
// per-row back-off further spaces retries). It is deliberately decoupled from the
// follower's poll intervals: OTS is best-effort and NEVER blocks the follower
// (ADR-0004), so it runs on its own goroutine off the poll path at its own cadence.
const otsUpgradeInterval = 24 * time.Hour

// reservedDomain reports whether a realm domain cannot be safely mounted: it is
// empty/whitespace (the dossier would mount the exact "/", colliding with the
// dashboard) or a reserved mount name (its exact "/"+domain dossier mount would
// collide with a built-in route — metrics / healthz / version / the web.Prefix
// segment — and panic http.ServeMux). registerHubs rejects such a domain loudly at startup;
// mirrorHandler uses the same predicate as defense-in-depth so building the mux
// from a route slice can never panic even if a reserved route is constructed
// directly.
func reservedDomain(domain string) bool {
	if strings.TrimSpace(domain) == "" {
		return true
	}
	_, reserved := reservedMountNames[domain]
	return reserved
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "iscc-monitor:", err)
		os.Exit(1)
	}
}

// notifyShutdown returns the process's signal-bound shutdown context: it cancels
// on SIGINT or SIGTERM, so an interactive Ctrl-C and a container/orchestrator stop
// (docker / Compose / Kubernetes / systemd all send SIGTERM, not SIGINT) both
// drain the in-flight poll, run the deferred store.Close, and exit cleanly. It is
// extracted as a named seam so the SIGTERM registration is unit-testable without
// driving the whole of run. SIGTERM is defined on every Go platform (the runtime
// maps it on Windows too), so the registration is unconditional and cross-platform.
func notifyShutdown() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// run loads configuration, registers every realm hub into the store, and drives
// the follower loop until the signal-bound context is cancelled. It returns an
// error for any startup or shutdown fault so main owns the single os.Exit; on a
// clean SIGINT or SIGTERM shutdown Loop.Run returns ctx.Err(), which run reports
// as nil.
func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}

	ctx, stop := notifyShutdown()
	defer stop()

	// loadRealm reads the realm Hub-List from cfg.RealmPath — an http(s):// URL (the
	// authoritative iscc-hub Hub-List, fetched fresh) or a filesystem path — and
	// returns BOTH the certificate's hub_id resolver (carrying each hub's REAL
	// embedded 12-bit hub_id) and the follower's realm entries (the domains to poll).
	hubList, entries, err := loadRealm(ctx, cfg.RealmPath, httpGetBytes, logger)
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	targets, routes, err := registerHubs(ctx, st, entries)
	if err != nil {
		return err
	}

	// resolver maps a decoded ISCC-IDv1 hub_id slot to the issuing hub's domain for
	// the realm-wide certificate (internal/certificate). It is the hot-swappable
	// holder runRealmRefresh updates hourly, so a Hub-List membership/slot change at
	// the authoritative source takes effect without a redeploy. The store is
	// domain-keyed (UpsertHub) independent of this 12-bit slot, so a refresh changes
	// only resolution — never the mirrored log data.
	resolver := registry.NewAtomicHubList(hubList)

	m := metrics.New()
	go serveMetrics(ctx, cfg.Addr, st, routes, resolver, m, identity(cfg), logger)
	go runOTSLoop(ctx, st, stampFunc(), otsclient.NewUpgrader(), logger)
	go runRealmRefresh(ctx, cfg.RealmPath, httpGetBytes, resolver, logger)

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
func serveMetrics(ctx context.Context, addr string, st *store.Store, routes []hubRoute, hubList registry.HubResolver, m *metrics.Registry, id dashboard.Identity, logger *slog.Logger) {
	srv := &http.Server{Addr: addr, Handler: buildMux(st, routes, hubList, m, id)}

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

// runOTSLoop drives the background OpenTimestamps stamp-then-upgrade loop until ctx
// is cancelled: on every otsUpgradeInterval tick it calls follower.OTSTick(ctx, st,
// stamper, upgrader, t, logger), which first submits any not-yet-stamped pending
// root's digest to a calendar (through the real otsclient Stamper) and then asks the
// calendar (through the real otsclient Upgrader) whether each stamped root has been
// Bitcoin-confirmed, either records the confirmation or backs the row off. It is the
// first production caller of follower.OTSTick, the sibling of serveMetrics: started
// in its own background goroutine off the follower's poll path so OTS is best-effort
// and NEVER blocks the follower (ADR-0004) — the calendar HTTP round-trips happen
// here, never on the poll path. A non-nil OTSTick result is logged and the loop
// continues — the documented never-block / log-and-continue discipline (identical to
// loop.go's Run), so a flaky calendar pass never aborts the loop or freezes a hub.
// The ticker is defer-stopped and the loop exits cleanly on ctx cancellation
// (SIGINT). The ticker's t is the injected now, so OTSTick stays wall-clock-free and
// testable.
func runOTSLoop(ctx context.Context, st *store.Store, stamper follower.Stamper, upgrader follower.Upgrader, logger *slog.Logger) {
	ticker := time.NewTicker(otsUpgradeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if err := follower.OTSTick(ctx, st, stamper, upgrader, t, logger); err != nil {
				logger.ErrorContext(ctx, "ots upgrade tick failed", "err", err)
			}
		}
	}
}

// stampFunc builds the production follower.Stamper: a closure over otsclient.Stamp
// that submits a root's digest to otsclient.DefaultCalendarURL and returns the
// serialized initial pending proof bytes plus the calendar URL it used. It is the
// boundary that keeps internal/follower import-free of any anchoring package — the
// follower holds only the Stamper func type, while this binary owns the otsclient
// dependency, exactly as it does for the Upgrader (otsclient.NewUpgrader). A calendar
// transport fault propagates out as the closure's error so OTSTick records a back-off
// and retries off the poll path (OTS never blocks the follower, ADR-0004).
func stampFunc() follower.Stamper {
	return func(ctx context.Context, root [32]byte) ([]byte, string, error) {
		b, err := otsclient.Stamp(ctx, otsclient.DefaultCalendarURL, root)
		return b, otsclient.DefaultCalendarURL, err
	}
}

// buildMux assembles the monitor's single request multiplexer: GET / (the
// server-rendered hub-list dashboard), GET /metrics, GET /healthz (liveness +
// store readiness), GET /version (the build-provenance string — git SHA or the
// "dev" default), GET /openapi.json + GET /openapi.yaml (the hand-authored OpenAPI
// 3.1 contract for the machine surface, served byte-verbatim), GET /docs (the
// server-rendered, self-hosted Stoplight Elements API reference mounting
// <elements-api apiDescriptionUrl="/openapi.json">), the GET /inclusion/ subtree (the realm-wide Certificate of
// Inclusion, keyed on the self-describing ISCC-IDv1), the GET /_ds/ subtree (the
// shared ISCC Design System v2 token stylesheet, the self-hosted @font-face
// stylesheet, the woff2 font binaries, and the byte-pinned Stoplight Elements
// JS/CSS every SSR page links), plus every hub's
// mirror subtree AND bare-domain dossier from mirrorHandler. The dashboard mounts
// at the exact path "/" — http.ServeMux's most-specific match means it never
// shadows /metrics, /healthz, the /inclusion/ subtree, the /_ds/ subtree, any
// /<domain>/log/ subtree, or any /<domain> dossier (the dashboard.Handler itself
// 404s any path other than "/"). The certificate mounts at the /inclusion/
// subtree, disjoint from every /<domain>/log/ mirror subtree and every /<domain>
// dossier exact mount (the per-hub JSON proof route stays /<domain>/log/inclusion,
// a different mount), so it never shadows them. The static assets mount at the
// web.Prefix subtree ("/_ds/"), so the token stylesheet, the fonts stylesheet, and
// every /_ds/fonts/<file>.woff2 route to the one web.Handler; it is isolated and
// never shadows "/" or the per-hub subtrees. The same metrics registry m the
// /metrics handler exposes is also passed to the dashboard and the certificate as
// their in-memory status overlay (the StatusSource), so the pages can render the
// live unresolvable / unverified verdicts the store cannot prove. /metrics,
// /healthz, /version, /openapi.json, /openapi.yaml, and /docs all mount as exact
// paths next to the per-hub mirror subtrees on the same mux, so the single-listener
// invariant holds (no second socket). The assembled
// mux is wrapped once in corsmw.Handler — the lone convergence point all public
// routes pass through — so every served surface answers cross-origin browser GETs
// uniformly (Access-Control-Allow-Origin: * on every response; OPTIONS preflights
// succeed with 204) without per-handler CORS code. It is factored out of
// serveMetrics so the full routing is unit-testable against an
// httptest.ResponseRecorder without binding a socket. The operator-supplied
// instance identity (id) is rendered on the dashboard masthead; an empty field
// falls back to the static placeholder copy inside dashboard.Handler.
func buildMux(st *store.Store, routes []hubRoute, hubList registry.HubResolver, m *metrics.Registry, id dashboard.Identity) http.Handler {
	mux := mirrorHandler(st, routes, m, id)
	mux.Handle("/", dashboard.Handler(st, m, id))
	mux.Handle("/metrics", metricshttp.Handler(m))
	mux.Handle("/healthz", healthz.Handler(st))
	mux.Handle("/version", version.Handler())
	mux.Handle(openapi.JSONPath, openapi.Handler())
	mux.Handle(openapi.YAMLPath, openapi.Handler())
	mux.Handle("/docs", docs.Handler())
	mux.Handle(certificate.PathPrefix, certificate.Handler(hubList, st, m, id))
	mux.Handle(web.Prefix, web.Handler())
	return corsmw.Handler(mux)
}

// identity builds the dashboard's instance identity from the validated config's
// optional masthead-identity strings. Unset keys leave the Config fields empty,
// which each masthead handler renders as its static fallback, so an unconfigured
// binary is honest rather than asserting a false instance. RealmName feeds the
// dashboard's Realm subtitle (the human realm name, distinct from the realm-
// document path).
func identity(cfg config.Config) dashboard.Identity {
	return dashboard.Identity{
		Instance: cfg.Instance,
		Operator: cfg.Operator,
		Realm:    cfg.RealmName,
	}
}

// realmRefreshInterval is how often runRealmRefresh re-fetches the realm Hub-List
// to pick up membership/slot changes without a redeploy (the hourly poll). It
// mirrors otsUpgradeInterval's const-not-config shape: an operational cadence, not
// a tuned knob.
const realmRefreshInterval = time.Hour

// realmFetchTimeout bounds a single Hub-List fetch so a hung remote never stalls
// startup or a refresh tick.
const realmFetchTimeout = 30 * time.Second

// realmDocMaxBytes caps a fetched realm document; a Hub-List is a few hundred
// bytes, so this guards against a runaway/HTML response without truncating any real
// document.
const realmDocMaxBytes = 1 << 20

// realmGetter fetches the raw bytes of a realm document at a URL. It is injected so
// loadRealm stays testable without a network; the binary backs it with
// httpGetBytes.
type realmGetter func(ctx context.Context, url string) ([]byte, error)

// httpGetBytes is the production realmGetter: a timeout-bounded, size-capped HTTP
// GET of the realm document URL. A non-200 status is an error so a GitHub-raw
// 404/5xx never parses as an empty realm. It pulls in no dependency beyond the
// already-imported net/http.
func httpGetBytes(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, realmFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %q: status %s", url, resp.Status)
	}
	// Read one byte past the cap so an oversize body fails closed with an error
	// rather than being silently truncated to a still-parseable prefix.
	data, err := io.ReadAll(io.LimitReader(resp.Body, realmDocMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > realmDocMaxBytes {
		return nil, fmt.Errorf("GET %q: realm document exceeds %d bytes", url, realmDocMaxBytes)
	}
	return data, nil
}

// isURLSource reports whether the realm source is an http(s):// URL (the
// authoritative Hub-List) rather than a filesystem path. The legacy domains-only
// fallback applies only to file sources, so loadRealm uses this to decide whether a
// ParseHubList failure is a malformed Hub-List (URL) or a possible legacy document
// (file).
func isURLSource(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

// readRealmSource reads the realm document bytes from source: an http(s):// URL
// fetched via get, or otherwise a filesystem path read from disk. Splitting the
// source resolution out keeps loadRealm format-focused and lets tests drive a file
// path while production drives the authoritative URL.
func readRealmSource(ctx context.Context, source string, get realmGetter) ([]byte, error) {
	if isURLSource(source) {
		data, err := get(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("fetch realm document %q: %w", source, err)
		}
		return data, nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read realm document %q: %w", source, err)
	}
	return data, nil
}

// loadRealm reads the realm document from source and returns BOTH the certificate's
// hub_id resolver (a *registry.HubList carrying each hub's REAL embedded 12-bit
// hub_id) and the follower's realm entries (the domains to poll). The authoritative
// source is the iscc-hub Hub-List YAML (registry.ParseHubList), whose explicit
// hub_id is what lets the certificate resolve a decoded ISCC-IDv1 to the right
// issuing hub — a mainnet hub_id is 1 or 2, not a document-order index.
//
// For backward compatibility with the legacy line-based domains-only document
// (deploy/realm-testnet.txt), a FILE source whose body does not parse as a Hub-List
// falls back to registry.Parse with the interim document-order slot mapping (slot i
// = entry i) and a logged warning. That mapping is correct ONLY when each hub's real
// hub_id equals its line position — true for the old testnet fixture, FALSE in
// general (e.g. mainnet hub_ids 1,2) — so production must point source at the
// Hub-List URL.
//
// The legacy fallback is gated to file sources: the authoritative URL always serves
// the Hub-List YAML, so a URL body that fails ParseHubList is a malformed Hub-List,
// not a legacy document. Surfacing the real parse error (rather than misparsing YAML
// keys like "version:"/"hubs:" as bogus document-order domains) is what lets
// refreshRealmOnce keep the last-good resolver on a bad remote response instead of
// swapping in garbage entries.
//
// A realm that lists no hubs is rejected: ParseHubList/Parse treat zero hubs as a
// valid empty document (the pure-leaf contract), but a running monitor with nothing
// to follow is a misconfiguration, not a state to boot into. Rejecting here fails
// the binary closed at startup and — because refreshRealmOnce keeps the last-good
// snapshot on any loadRealm error — stops an hourly fetch of an empty/placeholder
// document from wiping a working resolver mid-run.
func loadRealm(ctx context.Context, source string, get realmGetter, logger *slog.Logger) (*registry.HubList, []registry.Entry, error) {
	data, err := readRealmSource(ctx, source, get)
	if err != nil {
		return nil, nil, err
	}

	var (
		hl      *registry.HubList
		entries []registry.Entry
	)
	if parsed, hlErr := registry.ParseHubList(data); hlErr == nil {
		hl = parsed
		if entries, err = parsed.Entries(); err != nil {
			return nil, nil, err
		}
	} else if isURLSource(source) {
		return nil, nil, fmt.Errorf("parse realm Hub-List %q: %w", source, hlErr)
	} else {
		if entries, err = registry.Parse(data); err != nil {
			return nil, nil, fmt.Errorf("realm document %q is neither an iscc-hub Hub-List nor a domains-only document: %w", source, err)
		}
		logger.Warn("realm document is the legacy domains-only format; hub_id slots are derived from document order and are correct only when each hub's real hub_id equals its line position — point ISCC_MONITOR_REALM at the iscc-hub Hub-List YAML for authoritative hub_ids",
			"source", source)
		hl = hubListFromEntries(entries)
	}

	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("realm document %q lists no hubs; refusing to run an empty realm", source)
	}
	return hl, entries, nil
}

// runRealmRefresh re-fetches the realm Hub-List every realmRefreshInterval and
// atomically swaps the certificate's resolver, so a membership/slot change at the
// authoritative source takes effect within the hour without a redeploy. It is
// best-effort and fault-tolerant exactly like runOTSLoop: a failed fetch/parse is
// logged and the LAST GOOD snapshot is kept (resolver is never cleared on error),
// so a transient remote outage never breaks resolution. It refreshes only the
// hub_id RESOLVER — a brand-new hub still needs a restart to be FOLLOWED and gain
// its mirror routes (the follower target set and the per-hub mux are built once at
// startup). The loop exits cleanly on ctx cancellation.
func runRealmRefresh(ctx context.Context, source string, get realmGetter, resolver *registry.AtomicHubList, logger *slog.Logger) {
	ticker := time.NewTicker(realmRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshRealmOnce(ctx, source, get, resolver, logger)
		}
	}
}

// refreshRealmOnce performs one realm-refresh pass: it re-loads the Hub-List and,
// on success, atomically swaps it into resolver; on any fetch/parse failure it logs
// and returns WITHOUT touching resolver, so the last good snapshot is preserved
// (the keep-last-good contract). It is the unit-testable tick body of
// runRealmRefresh, factored out so the swap/keep-last-good decision is asserted
// without waiting on the hour-long ticker.
func refreshRealmOnce(ctx context.Context, source string, get realmGetter, resolver *registry.AtomicHubList, logger *slog.Logger) {
	hl, _, err := loadRealm(ctx, source, get, logger)
	if err != nil {
		logger.ErrorContext(ctx, "realm refresh failed; keeping last good Hub-List", "source", source, "err", err)
		return
	}
	resolver.Store(hl)
	logger.InfoContext(ctx, "realm Hub-List refreshed", "source", source, "hubs", len(hl.Hubs))
}

// hubListFromEntries builds the interim document-order Hub-List from a domains-only
// realm document's entries, assigning each entry's slot from its line position
// (slot i = entry i). It is the legacy fallback loadRealm uses when source is the
// line-based domains-only format rather than the authoritative iscc-hub Hub-List
// YAML; the certificate decodes an ISCC-IDv1's 12-bit hub_id and resolves it
// through this list. The mapping is correct ONLY when each hub's real hub_id equals
// its line position (true for the old testnet fixture sb0=0/sb1=1, false for
// mainnet hub_ids 1,2), which is why loadRealm logs a warning and production points
// at the Hub-List URL. The hubs carry only the slot and a https://<domain> base url
// (no key — keys come from did:web, ADR-0009); Active is true since the domains-only
// document lists only followed domains.
func hubListFromEntries(entries []registry.Entry) *registry.HubList {
	hubs := make([]registry.Hub, 0, len(entries))
	for i, e := range entries {
		slot := uint16(i)
		hubs = append(hubs, registry.Hub{HubID: &slot, URL: e.BaseURL, Active: true})
	}
	return &registry.HubList{Version: 1, Hubs: hubs}
}

// mirrorHandler builds the per-hub mirror router: for each route it mounts a
// per-hub handler at the subtree prefix "/" + Origin + "/" (e.g. /sb0.iscc.id/log/),
// stripping the prefix WITHOUT its trailing slash so the inner handler sees a
// leading-slash path (/checkpoint, /inclusion). A request to /<origin>/checkpoint
// therefore reaches the handler as /checkpoint. The mount prefix keeps its trailing
// slash so http.ServeMux does subtree matching — all of /tile/... and /inclusion
// under the prefix route to the same hub handler; an unknown top-level prefix falls
// through to the mux's default 404. Stripping only down to the leading slash (not
// past it) is load-bearing: the inner hubHandler is itself an http.ServeMux, which
// 301-redirects a path missing its leading slash. The origin must be the full
// <domain>/log — a request missing the /log segment does not match the prefix and
// 404s. All routes share the store's single open connection (reads serialize on it,
// ADR-0005/0007); no second DB handle is opened. The metrics registry m is forwarded
// to each hubHandler so the log browser gets the same in-memory status overlay the
// dashboard does.
//
// Each route ALSO mounts dossier.Handler at the exact bare-domain path "/" + Domain
// (e.g. /sb0.iscc.id): an exact pattern, more-specific than and disjoint from the
// "/" + Origin + "/" mirror subtree, so http.ServeMux keeps both and routes only that
// exact path to the dossier. The same metrics registry m is the dossier's in-memory
// status overlay, so its five-status badge matches the dashboard and log browser. The
// same operator-supplied instance identity id is rendered on the dossier masthead so
// its chrome stays byte-identical to the "/" dashboard masthead; an empty field falls
// back to the static placeholder copy inside dossier.Handler.
//
// The dossier mount is skipped for a reservedDomain (empty or a reserved mount
// name) so building the mux can never panic on a duplicate pattern with a built-in
// exact route. registerHubs already rejects such a domain loudly at startup, so in
// production this branch is never taken; it is defense-in-depth keeping buildMux
// total over any route slice. The mirror subtree mount stays — "/"+Origin+"/" is a
// subtree (e.g. /metrics/log/), which never collides with the built-in exact
// /metrics, so only the exact dossier mount is conditional.
func mirrorHandler(st *store.Store, routes []hubRoute, m *metrics.Registry, id dashboard.Identity) *http.ServeMux {
	mux := http.NewServeMux()
	for _, r := range routes {
		prefix := "/" + r.Origin + "/"
		strip := "/" + r.Origin // leave the leading slash on the suffix
		mux.Handle(prefix, http.StripPrefix(strip, hubHandler(st, r.HubID, r.Domain, m, id)))
		if !reservedDomain(r.Domain) {
			mux.Handle("/"+r.Domain, dossier.Handler(st, r.HubID, m, id))
		}
	}
	return mux
}

// hubHandler combines one hub's computed-proof, HTML-browser, and static-mirror
// surfaces behind a single per-hub mux: GET /inclusion, GET /consistency, GET
// /entries, GET /verify, AND the bare GET / (the HTML log browser) reach
// proofserve.Handler (the inclusion/consistency proofs, the single-leaf record
// bytes, the verify-for-me JSON verdict, and the log-browser page — all computed
// from the local mirror), and every other path falls through to tilesserve.Handler
// (the static BLOB mirror — /checkpoint, /tile/..., /tile/entries/...). The four
// proof routes are mounted at their exact paths so http.ServeMux's most-specific
// match wins over the "/" subtree; proofserve.Handler internally switches on the
// path, so the same handler serves all of them.
//
// The "/" slot needs a tiny dispatch func rather than a plain mount because an
// http.ServeMux cannot hold both an exact "/" and a subtree "/" (the subtree
// pattern "/" IS the bare-"/" match): the func sends the bare path "/" to proofserve
// (which renders the browser) and delegates every deeper path to tilesserve. It
// receives leading-slash paths (mirrorHandler strips only down to the leading
// slash), which both inner handlers and this inner ServeMux require. Both read that
// hub's BLOBs through the store's single open connection, so the proof endpoints
// never re-hit the hub.
//
// /entries (the computed single-leaf record_bytes) is distinct from
// /tile/entries/<bundleindex> (the raw whole-bundle BLOB tilesserve serves under
// "/"): the former returns exactly one leaf's bytes, the latter 256 framed leaves a
// client must still parse — different routes, different shapes, not duplicates.
//
// The metrics registry m is the in-memory status overlay (the StatusSource) the log
// browser and record list use to render the live unresolvable / unverified verdicts
// the store cannot prove — the same overlay source the dashboard receives, so the
// five-status badge taxonomy is consistent across both surfaces.
//
// domain (the hub's bare domain) and id (the operator-supplied dashboard.Identity) feed
// the record list's shared chrome (instance-identity masthead + verify link), its
// "← <domain> dossier" breadcrumb, and its head — the SAME Identity value the dashboard
// and dossier mastheads receive, so all three stay byte-identical.
//
// /records (the no-JS paginated HTML record list) and /record (the no-JS single-
// record page each list row links to) are exact mounts like the other proof routes so
// they beat the "/" subtree dispatch; an unmounted /record would fall through to
// tilesserve and 404.
//
// /checkpoint.ots (the mirrored OpenTimestamps proof for the accepted root) is also an
// exact mount so it beats the "/" subtree dispatch — without it the path falls through
// to tilesserve, which 404s the unknown path. It is a DIFFERENT artifact from the raw
// /checkpoint signed-note BLOB tilesserve serves under "/": the .ots is the timestamp
// proof, read by proofserve from the ots table, not a tilesserve mirror BLOB.
func hubHandler(st *store.Store, hubID int64, domain string, m *metrics.Registry, id dashboard.Identity) http.Handler {
	mux := http.NewServeMux()
	proofs := proofserve.Handler(st, hubID, domain, m, id)
	tiles := tilesserve.Handler(store.SQLiteFetcher{Store: st, HubID: hubID})
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			proofs.ServeHTTP(w, r)
			return
		}
		tiles.ServeHTTP(w, r)
	}))
	mux.Handle("/records", proofs)
	mux.Handle("/record", proofs)
	mux.Handle("/inclusion", proofs)
	mux.Handle("/consistency", proofs)
	mux.Handle("/entries", proofs)
	mux.Handle("/checkpoint.ots", proofs)
	mux.Handle("/verify", proofs)
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
//
// It fails loudly before mounting on an empty/whitespace domain (which would
// mount the dossier at "/" and collide with the dashboard) or a reserved mount
// name (metrics, healthz, version, the web.Prefix segment _ds — whose exact
// dossier mount would collide with the built-in route and panic http.ServeMux). A
// misconfigured realm therefore surfaces at startup with the bad domain named,
// rather than crashing later in buildMux.
func registerHubs(ctx context.Context, st *store.Store, entries []registry.Entry) ([]follower.HubTarget, []hubRoute, error) {
	targets := make([]follower.HubTarget, 0, len(entries))
	routes := make([]hubRoute, 0, len(entries))
	for _, e := range entries {
		if reservedDomain(e.Domain) {
			return nil, nil, fmt.Errorf("register hub %q: domain is empty or a reserved mount name", e.Domain)
		}
		org, err := logclient.Origin(e.BaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("derive origin for %q: %w", e.Domain, err)
		}
		id, err := st.UpsertHub(ctx, e.Domain, org, e.BaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("register hub %q: %w", e.Domain, err)
		}
		targets = append(targets, follower.HubTarget{HubID: id, BaseURL: e.BaseURL})
		routes = append(routes, hubRoute{HubID: id, Domain: e.Domain, Origin: org})
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
