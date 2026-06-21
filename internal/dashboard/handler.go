// Package dashboard serves the monitor's human-facing root page: a server-rendered
// HTML list of every followed hub with its status and coverage window (ADR-0001).
// It is a thin leaf in the shape of internal/healthz and internal/metricshttp — a
// package-local handler whose only internal dependencies are internal/store (read
// once per request via ListHubs) and internal/badge (the status partial). The
// dashboard owns the HTML; cmd/iscc-monitor only wires the handler at "/".
//
// The status it shows overlays two sources so all five glossary statuses can
// render honestly. The store proves the durable subset (inactive / frozen /
// verified); only those survive a restart. The richer live verdicts
// (unresolvable / unverified) come from the in-memory metrics registry via a
// StatusSource — a fresher poll verdict than the store's accepted-checkpoint flag.
// Precedence is load-bearing: inactive and frozen are harder, durable truths that
// the live verdict must never override, so the overlay applies only when the store
// status is verified.
//
// The dashboard depends on the StatusSource interface, not the concrete metrics
// package, so it stays golden-testable with a fake and keeps a minimal closure
// (bytes / embed / html/template / net/http / internal/store / internal/badge).
//
// The oracle/conformance gate is N/A: this is pure HTML rendering of persisted
// store rows plus an in-memory status overlay, touching no signature, RFC-6962,
// Merkle, did:web, fsck, or proof path.
package dashboard

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/iscc/iscc-monitor/internal/badge"
	"github.com/iscc/iscc-monitor/internal/store"
)

// pageTemplate is the embedded dashboard template, parsed once at package init so
// a malformed template fails the build, not a request.
//
//go:embed dashboard.html
var pageTemplate string

// tmpl is the parsed dashboard template with the HubStatusBadge partial
// associated into the same set, so the page invokes {{template "hubStatusBadge"
// .}} over each row (which exposes .Status and .Label). template.Must panics at
// init if either source fails to parse, surfacing a template bug at startup.
var tmpl = func() *template.Template {
	t := template.Must(template.New("dashboard").Parse(pageTemplate))
	return template.Must(t.Parse(badge.Source))
}()

// row is one hub's view-model for the template: the precomputed glossary status
// string, its fixed-table badge label, plus the raw summary fields the page
// renders. It is built from a store.HubSummary so the template stays free of
// status logic. The hubStatusBadge partial reads .Label directly (it does not
// re-derive the label from .Status), so the row carries a precomputed Label from
// the badge package's single source of truth.
type row struct {
	Domain      string
	Origin      string
	Status      string
	Label       string
	LastSize    uint64
	HasCoverage bool
	SinceSize   uint64
	SinceTime   string
}

// pageData is the whole template context: the rendered hub rows.
type pageData struct {
	Hubs []row
}

// StatusSource reports a hub's current in-memory glossary status by hub_id. It is
// the read seam the dashboard uses to overlay the live poll verdict (the richer
// unresolvable / unverified states the store cannot prove) onto the store-provable
// subset. ok is false when no live status is recorded for the hub. *metrics.Registry
// satisfies it structurally via its Status method; the dashboard takes the
// interface, not the concrete package, so it never imports internal/metrics.
type StatusSource interface {
	Status(hubID int64) (string, bool)
}

// Handler returns an http.Handler that renders the hub-list dashboard at the exact
// path "/". Only GET is served (any other method is 405); any path other than "/"
// is 404 — http.ServeMux routes everything unmatched by a more-specific pattern
// here, so the handler itself guards the exact path. On a store read error or a
// template render error it writes a 500 before any 200 is committed: the page is
// rendered into a buffer first and only copied to the client on success, so a
// client never sees a half-rendered 200.
//
// st must be non-nil (the binary always passes the real store); there is no
// nil-guard branch. statuses is the in-memory status overlay (the metrics
// registry); a nil statuses is tolerated and simply leaves every store-verified
// hub showing "verified".
func Handler(st *store.Store, statuses StatusSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		summaries, err := st.ListHubs(r.Context())
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, pageData{Hubs: buildRows(summaries, statuses)}); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// Post-200 write-drop: the status is already committed, so a copy error can
		// only signal a broken client connection, which a second status cannot fix.
		_, _ = buf.WriteTo(w)
	})
}

// buildRows maps the store summaries into template rows, precomputing each hub's
// glossary status and its fixed-table badge label so the template carries no
// logic. It overlays the in-memory status (overlayStatus) so all five glossary
// statuses render honestly. The resulting status is always a valid badge.Label key
// (the store subset and the registry both emit glossary keys), so ok is always
// true here; the row still falls back to the status string if badge.Label ever
// returns ok==false, so the page never renders an unlabeled badge.
func buildRows(summaries []store.HubSummary, statuses StatusSource) []row {
	rows := make([]row, 0, len(summaries))
	for _, s := range summaries {
		status := overlayStatus(s, statuses)
		label, ok := badge.Label(status)
		if !ok {
			label = status
		}
		rows = append(rows, row{
			Domain:      s.Domain,
			Origin:      s.Origin,
			Status:      status,
			Label:       label,
			LastSize:    s.LastSize,
			HasCoverage: s.Coverage.Set,
			SinceSize:   s.Coverage.Size,
			SinceTime:   coverageTime(s.Coverage),
		})
	}
	return rows
}

// overlayStatus resolves a hub's displayed status from the store-provable subset
// plus the in-memory live verdict. The store status wins for the durable, harder
// truths: inactive (registry removed/paused) and frozen (a self-consistency
// violation, ADR-0006 evidence that survives restart) must never be overridden by
// a fresher in-memory verdict. Only when the store says "verified" does it consult
// statuses: a hub with an old accepted checkpoint but a currently-failing did:web
// resolve is honestly "unresolvable" now, and a current signature matching no
// listed key is "unverified" — both fresher than the store's verified flag. Any
// other live verdict (including verified itself, or no record) keeps "verified".
func overlayStatus(s store.HubSummary, statuses StatusSource) string {
	status := hubStatus(s)
	if status != "verified" || statuses == nil {
		return status
	}
	switch live, ok := statuses.Status(s.HubID); {
	case ok && (live == "unresolvable" || live == "unverified"):
		return live
	default:
		return status
	}
}

// hubStatus maps a hub summary to the store-provable glossary status subset
// (CLAUDE.md "Hub status"): "inactive" when the hub is paused/removed in the realm
// registry, else "frozen" on a self-consistency violation (ADR-0006), else
// "verified" (only signature-verified checkpoints advance accepted state). It
// mirrors internal/proofserve.hubStatus and extends it with the realm-registry
// inactive case; it deliberately invents no status the store cannot prove
// (unverified / unresolvable / rotated live in the metrics registry).
func hubStatus(s store.HubSummary) string {
	switch {
	case !s.Active:
		return "inactive"
	case s.Frozen:
		return "frozen"
	default:
		return "verified"
	}
}

// coverageTime renders the coverage start time as RFC 3339 when both the coverage
// and a non-zero start time are set, else the empty string (the template shows "no
// coverage yet" in that case, never implying a pre-coverage guarantee, ADR-0001).
func coverageTime(c store.CoverageInfo) string {
	if !c.Set || c.Since.IsZero() {
		return ""
	}
	return c.Since.UTC().Format("2006-01-02T15:04:05Z")
}
