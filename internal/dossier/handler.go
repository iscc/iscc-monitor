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
// keeps a minimal closure. It also reuses internal/dashboard's Identity type (the
// masthead identity value the "/" page already receives) rather than redefining it,
// applying its own resolveIdentity fail-safe so the dossier and dashboard mastheads
// render byte-identical chrome. The overlayStatus / hubStatus / coverageTime shape
// is deliberately a third local copy of the dashboard/proofserve pattern;
// consolidating it into internal/badge is its own tracked step, not a prerequisite
// here.
//
// The oracle/conformance gate is N/A: this is a pure HTML render of one persisted
// store row plus an in-memory status overlay, touching no signature, RFC-6962,
// Merkle, did:web, fsck, or proof path.
package dossier

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/iscc/iscc-monitor/internal/badge"
	"github.com/iscc/iscc-monitor/internal/dashboard"
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
//
// Instance and Operator are the resolved instance-identity strings the masthead
// chrome renders (Instance is this deployment's domain, Operator the operator/realm
// line beneath it), carried verbatim so the dossier and dashboard mastheads stay
// byte-identical. They are always non-empty — resolveIdentity applies the static
// fallback copy so an unconfigured binary renders today's masthead.
//
// Frozen gates the non-dismissable Exhibit panel (ADR-0006 irreplaceable evidence):
// when true the template renders the categorically-distinct "do not trust new
// state" panel listing each Violations row. A frozen hub may carry zero Violations
// (defended against), so the panel header renders even with an empty list, never a
// broken {{range}}.
//
// The numbered trust-document fields are honesty-gated (no fabricated value): §2's
// CoverageDays is "" unless coverage is set; §3's ObservedTime is "" when the latest
// checkpoint carries no recorded time; §4's AnchorLabel / AnchorDot come from the
// store OTS status (the label is the grayscale-safe load-bearing signal, the dot
// decorative, ADR-0010 inv.4) and AnchorHeight renders only when the anchor is
// confirmed AND the height is non-zero (HasAnchorHeight), never block 0. StatusNote
// is the soft-caution copy for a non-frozen unresolvable / unverified hub, with the
// store's "fork" violation kind mapped to the canonical "split view" vocabulary
// (CLAUDE.md Language) wherever a kind surfaces outside the Exhibit.
type dossierData struct {
	Domain          string
	Origin          string
	Status          string
	Label           string
	LastSize        uint64
	HasCoverage     bool
	SinceSize       uint64
	SinceTime       string
	CoverageDays    string
	ObservedTime    string
	AnchorLabel     string
	AnchorDot       string
	HasAnchorHeight bool
	AnchorHeight    uint64
	ShowCaution     bool
	StatusNote      string
	Instance        string
	Operator        string
	Frozen          bool
	Violations      []violationRow
}

// violationRow is one self-consistency violation rendered into the dossier Exhibit:
// the trigger Kind ("fork"/"shrink"/"equivocation") and the DetectedAt instant as
// RFC 3339, or the empty string when the detected time is unknown (the template
// shows "time unknown" rather than a fabricated epoch — coverage-honesty discipline
// applies to evidence timestamps too). The raw checkpoint bytes and proof are NOT
// rendered here; they belong with the future proof-bundle surface.
type violationRow struct {
	Kind       string
	DetectedAt string
}

// Default masthead identity copy used when an identity field is left empty, so an
// unconfigured deployment renders today's static placeholder rather than a false
// claim. These MUST stay byte-identical to internal/dashboard's instanceFallback /
// operatorFallback consts: both the dossier and the dashboard masthead are required
// to render the same chrome, and neither package can import the other's unexported
// consts, so the defaults are duplicated here as literals.
const (
	instanceFallback = "monitor instance"
	operatorFallback = "independent Trust & Transparency service · ISCC-Hub network"
)

// resolveIdentity applies the dossier-side fail-safe for the masthead identity,
// mirroring dashboard.Identity.resolve semantics so the dossier and dashboard
// chrome stay in lockstep: a blank Instance or Operator falls back to the static
// placeholder copy. It lives here (not in internal/dashboard) so the fallback is
// seam-testable at the dossier HTTP boundary without making internal/dashboard a
// fourth edited file (its resolve is unexported). Realm has no slot on the dossier
// masthead (its title is the realm-subtitle-free "Hub dossier"), so it is ignored.
func resolveIdentity(id dashboard.Identity) (instance, operator string) {
	instance, operator = id.Instance, id.Operator
	if instance == "" {
		instance = instanceFallback
	}
	if operator == "" {
		operator = operatorFallback
	}
	return instance, operator
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
// simply leaves a store-verified hub showing "verified". id is the operator-supplied
// instance identity rendered on the masthead chrome; any empty field falls back to
// the static placeholder copy (resolveIdentity), so a zero-value Identity renders
// exactly today's masthead. It is the SAME dashboard.Identity value the "/" masthead
// receives, so the two mastheads stay byte-identical.
func Handler(st *store.Store, hubID int64, statuses StatusSource, id dashboard.Identity) http.Handler {
	instance, operator := resolveIdentity(id)
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
		status := overlayStatus(summary, statuses)
		// The Exhibit read stays off the hot path: only a frozen hub queries
		// violations. frozen is store-provable and the overlay never downgrades it
		// (it only promotes verified), so status == "frozen" is a safe gate.
		var violations []store.Violation
		if status == "frozen" {
			violations, err = st.ListViolations(r.Context(), summary.HubID)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, buildData(summary, status, violations, instance, operator)); err != nil {
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

// buildData maps one store summary and its already-resolved glossary status into
// the dossier view-model, precomputing the fixed-table badge label so the template
// carries no status logic. The status is always a valid badge.Label key (the store
// subset and the registry both emit glossary keys), so ok is always true here; the
// view still falls back to the status string if badge.Label ever returns ok==false,
// so the page never renders an unlabeled badge. The caller passes the violations
// (empty for a non-frozen hub, which never reads them); buildData folds them into
// the Exhibit rows and sets Frozen so the template renders the panel. instance and
// operator are the already-resolved masthead identity strings (resolveIdentity has
// applied the fail-safe fallback) carried verbatim onto the view-model.
func buildData(s store.HubSummary, status string, violations []store.Violation, instance, operator string) dossierData {
	label, ok := badge.Label(status)
	if !ok {
		label = status
	}
	anchorLbl, anchorDot := anchorLabel(s.Anchor)
	hasHeight := s.Anchor == store.OTSStatusConfirmed && s.AnchorHeight > 0
	return dossierData{
		Domain:          s.Domain,
		Origin:          s.Origin,
		Status:          status,
		Label:           label,
		LastSize:        s.LastSize,
		HasCoverage:     s.Coverage.Set,
		SinceSize:       s.Coverage.Size,
		SinceTime:       coverageTime(s.Coverage),
		CoverageDays:    coverageDays(s.Coverage),
		ObservedTime:    observedTime(s.CheckpointObserved),
		AnchorLabel:     anchorLbl,
		AnchorDot:       anchorDot,
		HasAnchorHeight: hasHeight,
		AnchorHeight:    s.AnchorHeight,
		ShowCaution:     status == "unresolvable" || status == "unverified",
		StatusNote:      statusNote(status),
		Instance:        instance,
		Operator:        operator,
		Frozen:          status == "frozen",
		Violations:      violationRows(violations),
	}
}

// anchorLabel maps a hub's stored OTS status to its §4 display label and the
// presentation-only dot keyword, ported verbatim from internal/dashboard so the two
// surfaces render the same anchor copy: a confirmed root renders "confirmed"; a
// pending root "pending"; any other value (including the empty never-stamped state)
// the honest "not anchored" — never implying a Bitcoin anchor exists. It compares
// against store.OTSStatusConfirmed / store.OTSStatusPending (not hand-typed
// literals) so the status strings never drift from the store's single source of
// truth. The dot is decorative; the label carries the meaning grayscale-safe
// (ADR-0010 inv.4).
func anchorLabel(status string) (label, dot string) {
	switch status {
	case store.OTSStatusConfirmed:
		return "confirmed", "confirmed"
	case store.OTSStatusPending:
		return "pending", "pending"
	default:
		return "not anchored", "none"
	}
}

// observedTime renders the latest checkpoint's observed-at as RFC 3339 UTC, or the
// empty string when zero (no recorded time) — the template then shows the honest
// "observed time unknown" rather than a fabricated instant, the same coverage-honesty
// discipline coverageTime applies (ADR-0001).
func observedTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// coverageDays renders the whole-days-observed string for §2 ("N days observed"),
// derived from the coverage start (ADR-0001). It returns "" when coverage is not set
// or the start time is unknown — the dossier invents no observation count for an
// uncovered hub. A start in the future (clock skew) clamps to "0 days observed".
func coverageDays(c store.CoverageInfo) string {
	if !c.Set || c.Since.IsZero() {
		return ""
	}
	days := int(time.Since(c.Since).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("%d days observed", days)
}

// statusNote returns the soft-caution copy for a non-frozen hub whose live verdict
// is unresolvable or unverified, mapping any "fork" wording to the canonical "split
// view" vocabulary (CLAUDE.md Language) per the binding rule that the avoid-listed
// "fork" never surfaces in §-level status copy. A status with no caution returns "".
func statusNote(status string) string {
	switch status {
	case "unresolvable":
		return "The monitor cannot currently fetch or parse this hub's did:web document, so its signing key is unresolved. The mirrored log is preserved; this is not a split-view finding."
	case "unverified":
		return "A checkpoint signature did not match any key in this hub's own did:web document — an internally-broken hub. The mirrored log is preserved; this is not a split-view finding."
	default:
		return ""
	}
}

// violationRows maps the store violation rows into the dossier render structs,
// formatting each DetectedAt as RFC 3339 (or the empty string when the time is
// unknown — the same idiom coverageTime uses, never a fabricated epoch). Only kind
// and detected-at are surfaced; the raw evidence bytes are out of scope here.
func violationRows(violations []store.Violation) []violationRow {
	rows := make([]violationRow, 0, len(violations))
	for _, v := range violations {
		rows = append(rows, violationRow{
			Kind:       v.Kind,
			DetectedAt: violationTime(v.DetectedAt),
		})
	}
	return rows
}

// violationTime renders a violation's detected-at as RFC 3339 UTC, or the empty
// string when zero (a NULL detected_at). It mirrors coverageTime so evidence
// timestamps follow the same honesty discipline as the coverage window.
func violationTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
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
