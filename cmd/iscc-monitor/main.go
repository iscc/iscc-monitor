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
	"strings"
	"time"

	"github.com/iscc/iscc-monitor/internal/certificate"
	"github.com/iscc/iscc-monitor/internal/config"
	"github.com/iscc/iscc-monitor/internal/corsmw"
	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/dossier"
	"github.com/iscc/iscc-monitor/internal/follower"
	"github.com/iscc/iscc-monitor/internal/healthz"
	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/metricshttp"
	"github.com/iscc/iscc-monitor/internal/otsclient"
	"github.com/iscc/iscc-monitor/internal/proofserve"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tilesserve"
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
// route (metrics, healthz) or the web.Prefix subtree segment (_ds). The _ds entry
// is derived from web.Prefix (not hardcoded) so it tracks the const if it changes.
// registerHubs rejects any realm Domain in this set before mounting so a
// misconfigured realm fails loudly at startup instead of panicking http.ServeMux.
var reservedMountNames = map[string]struct{}{
	"metrics":                     {},
	"healthz":                     {},
	strings.Trim(web.Prefix, "/"): {},
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
// collide with a built-in route — metrics / healthz / the web.Prefix segment —
// and panic http.ServeMux). registerHubs rejects such a domain loudly at startup;
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

	// hubList maps a decoded ISCC-IDv1 hub_id slot to the issuing hub's domain for
	// the realm-wide certificate (internal/certificate). Production has no Hub-List
	// document path yet (realm.txt is line-based domains, a different format), so for
	// now the slot mapping is the interim realm-entry-order mapping (slot i = entry
	// i), which matches the testnet fixture (sb0 = slot 0, sb1 = slot 1). Swapping in
	// a real Hub-List source later replaces only this construction.
	hubList := hubListFromEntries(entries)

	m := metrics.New()
	go serveMetrics(ctx, cfg.Addr, st, routes, hubList, m, identity(cfg), logger)
	go runOTSLoop(ctx, st, stampFunc(), otsclient.NewUpgrader(), logger)

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
func serveMetrics(ctx context.Context, addr string, st *store.Store, routes []hubRoute, hubList *registry.HubList, m *metrics.Registry, id dashboard.Identity, logger *slog.Logger) {
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
// store readiness), the GET /inclusion/ subtree (the realm-wide Certificate of
// Inclusion, keyed on the self-describing ISCC-IDv1), the GET /_ds/ subtree (the
// shared ISCC Design System v2 token stylesheet, the self-hosted @font-face
// stylesheet, and the woff2 font binaries every SSR page links), plus every hub's
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
// live unresolvable / unverified verdicts the store cannot prove. Both /metrics and
// /healthz mount as exact paths next to the per-hub mirror subtrees on the same
// mux, so the single-listener invariant holds (no second socket). The assembled
// mux is wrapped once in corsmw.Handler — the lone convergence point all public
// routes pass through — so every served surface answers cross-origin browser GETs
// uniformly (Access-Control-Allow-Origin: * on every response; OPTIONS preflights
// succeed with 204) without per-handler CORS code. It is factored out of
// serveMetrics so the full routing is unit-testable against an
// httptest.ResponseRecorder without binding a socket. The operator-supplied
// instance identity (id) is rendered on the dashboard masthead; an empty field
// falls back to the static placeholder copy inside dashboard.Handler.
func buildMux(st *store.Store, routes []hubRoute, hubList *registry.HubList, m *metrics.Registry, id dashboard.Identity) http.Handler {
	mux := mirrorHandler(st, routes, m, id)
	mux.Handle("/", dashboard.Handler(st, m, id))
	mux.Handle("/metrics", metricshttp.Handler(m))
	mux.Handle("/healthz", healthz.Handler(st))
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

// hubListFromEntries builds the interim realm-wide Hub-List for the certificate
// route from the realm document's entries, assigning each entry's slot from its
// document order (slot i = entry i). Production has no Hub-List document path yet
// (realm.txt is line-based domains, a different format from the YAML Hub-List), so
// this order mapping stands in until a real Hub-List source lands; it matches the
// testnet fixture (sb0 = slot 0, sb1 = slot 1). The certificate decodes an
// ISCC-IDv1's 12-bit hub_id and resolves it through this list to the issuing hub's
// domain. The hubs carry only the slot and a https://<domain> base url (no key —
// keys come from did:web, ADR-0009); Active is true since realm.txt lists only
// followed domains. TODO: replace with a real Hub-List source (its own decision,
// not this skeleton's) once production carries one.
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
		mux.Handle(prefix, http.StripPrefix(strip, hubHandler(st, r.HubID, m)))
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
func hubHandler(st *store.Store, hubID int64, m *metrics.Registry) http.Handler {
	mux := http.NewServeMux()
	proofs := proofserve.Handler(st, hubID, m)
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
// name (metrics, healthz, the web.Prefix segment _ds — whose exact dossier mount
// would collide with the built-in route and panic http.ServeMux). A
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
