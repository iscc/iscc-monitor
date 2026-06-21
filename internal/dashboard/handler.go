// Package dashboard serves the monitor's human-facing root page: a server-rendered
// HTML list of every followed hub with its store-provable status (frozen /
// verified / inactive) and coverage window (ADR-0001). It is a thin leaf in the
// shape of internal/healthz and internal/metricshttp — a package-local handler
// whose only internal dependency is internal/store, which it calls once per
// request via ListHubs. The dashboard owns the HTML; cmd/iscc-monitor only wires
// the handler at "/".
//
// The status it shows is deliberately the store-provable subset only — the same
// mapping internal/proofserve.hubStatus uses, extended with the realm-registry
// "inactive" — so the page is golden-testable on a fixture store. The richer
// in-memory statuses (unverified / unresolvable / rotated) live in the metrics
// registry and are not threaded in here.
//
// The oracle/conformance gate is N/A: this is pure HTML rendering of persisted
// store rows, touching no signature, RFC-6962, Merkle, did:web, fsck, or proof
// path.
package dashboard

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/iscc/iscc-monitor/internal/store"
)

// pageTemplate is the embedded dashboard template, parsed once at package init so
// a malformed template fails the build, not a request.
//
//go:embed dashboard.html
var pageTemplate string

// tmpl is the parsed dashboard template. template.Must panics at init if the
// embedded source fails to parse, surfacing a template bug at startup.
var tmpl = template.Must(template.New("dashboard").Parse(pageTemplate))

// row is one hub's view-model for the template: the precomputed glossary status
// string plus the raw summary fields the page renders. It is built from a
// store.HubSummary so the template stays free of status logic.
type row struct {
	Domain      string
	Origin      string
	Status      string
	LastSize    uint64
	HasCoverage bool
	SinceSize   uint64
	SinceTime   string
}

// pageData is the whole template context: the rendered hub rows.
type pageData struct {
	Hubs []row
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
// nil-guard branch.
func Handler(st *store.Store) http.Handler {
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
		if err := tmpl.Execute(&buf, pageData{Hubs: buildRows(summaries)}); err != nil {
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
// glossary status so the template carries no logic.
func buildRows(summaries []store.HubSummary) []row {
	rows := make([]row, 0, len(summaries))
	for _, s := range summaries {
		rows = append(rows, row{
			Domain:      s.Domain,
			Origin:      s.Origin,
			Status:      hubStatus(s),
			LastSize:    s.LastSize,
			HasCoverage: s.Coverage.Set,
			SinceSize:   s.Coverage.Size,
			SinceTime:   coverageTime(s.Coverage),
		})
	}
	return rows
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
