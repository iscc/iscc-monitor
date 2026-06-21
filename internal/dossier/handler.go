// Package dossier serves the monitor's per-hub Evidence-Ledger page at the bare
// GET /<domain> (no /log). It renders ONE hub's store-provable status badge and
// honest coverage window in the Evidence-Ledger card pattern (ADR-0001), threading
// the same DS-token/font shell the realm index (internal/dashboard) and the log
// browser (internal/proofserve) use.
//
// Like dashboard.Handler it overlays two status sources so all five glossary
// statuses can render honestly: the store proves the durable subset (inactive /
// frozen / verified), and the richer live verdicts (unresolvable / unverified)
// come from the in-memory metrics registry via a StatusSource. Precedence is
// load-bearing — the overlay applies only when the store status is verified, so the
// durable inactive / frozen truths are never overridden by a fresher poll verdict.
//
// The dossier depends on internal/store + internal/badge + the StatusSource
// interface (NOT internal/metrics), so it stays golden-testable with a fake and
// keeps a minimal closure. The overlayStatus / hubStatus / coverageTime shape is
// deliberately a third local copy of the dashboard/proofserve pattern; consolidating
// it into internal/badge is its own tracked step, not a prerequisite here.
//
// The oracle/conformance gate is N/A: this is a pure HTML render of one persisted
// store row plus an in-memory status overlay, touching no signature, RFC-6962,
// Merkle, did:web, fsck, or proof path.
package dossier

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/iscc/iscc-monitor/internal/badge"
	"github.com/iscc/iscc-monitor/internal/store"
)

// pageTemplate is the embedded dossier template, parsed once at package init so a
// malformed template fails the build, not a request.
//
//go:embed dossier.html
var pageTemplate string

// tmpl is the parsed dossier template with the HubStatusBadge partial associated
// into the same set, so the page invokes {{template "hubStatusBadge" .}} over the
// view (which exposes .Status and .Label). template.Must panics at init if either
// source fails to parse, surfacing a template bug at startup.
var tmpl = func() *template.Template {
	t := template.Must(template.New("dossier").Parse(pageTemplate))
	return template.Must(t.Parse(badge.Source))
}()

// dossierData is the dossier template view-model for one hub: its domain and log
// origin, the precomputed glossary status string + its fixed-table badge label, and
// the honest coverage window (HasCoverage gates the "size N at <RFC3339>" vs "no
// coverage yet" split — the observed LastSize is never passed off as a coverage
// guarantee, ADR-0001). The hubStatusBadge partial reads .Label directly, so the
// view carries a precomputed Label from the badge package's single source of truth.
type dossierData struct {
	Domain      string
	Origin      string
	Status      string
	Label       string
	LastSize    uint64
	HasCoverage bool
	SinceSize   uint64
	SinceTime   string
}

// StatusSource reports a hub's current in-memory glossary status by hub_id. It is
// the read seam the dossier uses to overlay the live poll verdict (the richer
// unresolvable / unverified states the store cannot prove) onto the store-provable
// subset. ok is false when no live status is recorded for the hub. *metrics.Registry
// satisfies it structurally via its Status method; the dossier takes the interface,
// not the concrete package, so it never imports internal/metrics.
type StatusSource interface {
	Status(hubID int64) (string, bool)
}

// Handler returns an http.Handler that renders the per-hub dossier for hubID. It is
// mounted at the exact bare-domain path "/" + domain (e.g. /sb0.iscc.id), more
// specific than and disjoint from the hub's "/" + origin + "/" mirror subtree, so
// http.ServeMux routes only that exact path here — the handler needs no in-handler
// path guard (unlike dashboard, which owns the catch-all "/"). Only GET is served
// (any other method is 405).
//
// On GET it reads every hub summary and selects the one whose HubID == hubID. The
// binary always registers the hub before mounting this handler, so a not-found is a
// real store inconsistency → 500, never a 404. On a store read error or a template
// render error it writes a 500 before any 200 is committed: the page is rendered
// into a buffer first and only copied to the client on success, so a client never
// sees a half-rendered 200.
//
// st must be non-nil (the binary always passes the real store). statuses is the
// in-memory status overlay (the metrics registry); a nil statuses is tolerated and
// simply leaves a store-verified hub showing "verified".
func Handler(st *store.Store, hubID int64, statuses StatusSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		summaries, err := st.ListHubs(r.Context())
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		summary, ok := findHub(summaries, hubID)
		if !ok {
			// The binary registers the hub before mounting this handler, so a
			// missing summary is a real store inconsistency, not a client 404.
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, buildData(summary, statuses)); err != nil {
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

// findHub returns the summary whose HubID matches and ok=true, or the zero value and
// ok=false when no summary matches (a real store inconsistency the handler maps to a
// 500, never a client 404).
func findHub(summaries []store.HubSummary, hubID int64) (store.HubSummary, bool) {
	for _, s := range summaries {
		if s.HubID == hubID {
			return s, true
		}
	}
	return store.HubSummary{}, false
}

// buildData maps one store summary into the dossier view-model, precomputing the
// hub's overlaid glossary status and its fixed-table badge label so the template
// carries no status logic. The status is always a valid badge.Label key (the store
// subset and the registry both emit glossary keys), so ok is always true here; the
// view still falls back to the status string if badge.Label ever returns ok==false,
// so the page never renders an unlabeled badge.
func buildData(s store.HubSummary, statuses StatusSource) dossierData {
	status := overlayStatus(s, statuses)
	label, ok := badge.Label(status)
	if !ok {
		label = status
	}
	return dossierData{
		Domain:      s.Domain,
		Origin:      s.Origin,
		Status:      status,
		Label:       label,
		LastSize:    s.LastSize,
		HasCoverage: s.Coverage.Set,
		SinceSize:   s.Coverage.Size,
		SinceTime:   coverageTime(s.Coverage),
	}
}

// overlayStatus resolves a hub's displayed status from the store-provable subset
// plus the in-memory live verdict. The store status wins for the durable, harder
// truths: inactive (registry removed/paused) and frozen (a self-consistency
// violation, ADR-0006 evidence that survives restart) must never be overridden by a
// fresher in-memory verdict. Only when the store says "verified" does it consult
// statuses, adopting only the two richer live states (unresolvable / unverified);
// any other live verdict (including verified itself, or no record) keeps "verified".
// It mirrors dashboard.overlayStatus and proofserve.overlayStatus precedence verbatim.
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
// mirrors dashboard.hubStatus and deliberately invents no status the store cannot
// prove (unverified / unresolvable / rotated live in the metrics registry).
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
