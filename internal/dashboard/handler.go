// Package dashboard serves the monitor's human-facing root page: a server-rendered
// HTML list of every followed hub with its status and coverage window (ADR-0001).
// It is a thin leaf in the shape of internal/healthz and internal/metricshttp — a
// package-local handler whose only internal dependencies are internal/store (read
// once per request via ListHubs), internal/badge (the status partial), and
// internal/chrome (the shared masthead and footer). The dashboard owns the page body;
// cmd/iscc-monitor only wires the handler at "/".
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
// (bytes / embed / html/template / net/http / internal/store / internal/badge /
// internal/chrome).
//
// The oracle/conformance gate is N/A: this is pure HTML rendering of persisted
// store rows plus an in-memory status overlay, touching no signature, RFC-6962,
// Merkle, did:web, fsck, or proof path.
package dashboard

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/iscc/iscc-monitor/internal/badge"
	"github.com/iscc/iscc-monitor/internal/chrome"
	"github.com/iscc/iscc-monitor/internal/store"
)

// pageTemplate is the embedded dashboard template, parsed once at package init so
// a malformed template fails the build, not a request.
//
//go:embed dashboard.html
var pageTemplate string

// tmpl is the parsed dashboard template with the HubStatusBadge partial and the
// shared chrome partials associated into the same set, so the page invokes
// {{template "hubStatusBadge" .}} over each row (which exposes .Status and .Label)
// and the siteHeader / siteFooter partials over .Chrome. template.Must panics at init
// if any source fails to parse, surfacing a template bug at startup.
var tmpl = func() *template.Template {
	t := template.Must(template.New("dashboard").Parse(pageTemplate))
	t = template.Must(t.Parse(badge.Source))
	return template.Must(t.Parse(chrome.Source))
}()

// row is one hub's view-model for the template: the precomputed glossary status
// string, its fixed-table badge label, plus the raw summary fields the page
// renders. It is built from a store.HubSummary so the template stays free of
// status logic. The hubStatusBadge partial reads .Label directly (it does not
// re-derive the label from .Status), so the row carries a precomputed Label from
// the badge package's single source of truth. RowNo is the 1-based, zero-padded
// "#" the mockup's ledger renders; it is presentation only (not a store value).
//
// Checkpoint is the accepted checkpoint size the mockup's "Checkpoint" column
// shows (== LastSize, a relabel of the former "Observed size" cell, not a new
// store read). Anchor, AnchorNotes and AnchorDot are the Anchor cell from anchorCell:
// the label, the lines beneath it, and the presentation-only dot keyword ("confirmed" /
// "pending" / "none") the template keys the decorative dot color on — never the sole
// status signal (the label carries it grayscale-safe, ADR-0010 invariant 4).
type row struct {
	RowNo       string
	Domain      string
	Origin      string
	Status      string
	Label       string
	LastSize    uint64
	HasCoverage bool
	SinceSize   uint64
	SinceTime   string
	Checkpoint  uint64
	Anchor      string
	AnchorNotes []string
	AnchorDot   string
}

// recentDecl is one entry of the hero's "Recently declared" row: a recently
// indexed declaration the user can click straight through to its Certificate of
// Inclusion, so the page ships a working example without typing. IsccID is the
// verbatim ISCC:-prefixed id (the link text); Body is that id without the "ISCC:"
// prefix, used for the /inclusion/<body> path so no colon ever enters the URL path
// (the certificate handler re-adds the prefix, accepting either form).
type recentDecl struct {
	IsccID string
	Body   string
}

// pageData is the whole template context: the rendered hub rows, the count of
// followed hubs (HubCount == len(Hubs)) the ledger heading reports, the
// recently-declared ids for the hero shortcut row, the shared masthead and footer
// (Chrome), and the realm name in the "Realm register · <realm>" subtitle. An empty
// Realm renders the bare "Realm register" subtitle with no trailing separator.
type pageData struct {
	Hubs     []row
	HubCount int
	Recent   []recentDecl
	Chrome   chrome.View
	Realm    string
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
// hub showing "verified". site is the operator-configured identity and co-branding the
// shared masthead and footer render; a zero-value Site renders the static fallbacks.
func Handler(st *store.Store, statuses StatusSource, site chrome.Site) http.Handler {
	view := site.View()
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
		rows := buildRows(summaries, statuses)
		recent, err := buildRecent(r.Context(), st)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		var buf bytes.Buffer
		data := pageData{
			Hubs:     rows,
			HubCount: len(rows),
			Recent:   recent,
			Chrome:   view,
			Realm:    site.Realm,
		}
		if err := tmpl.Execute(&buf, data); err != nil {
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
	for i, s := range summaries {
		status := overlayStatus(s, statuses)
		label, ok := badge.Label(status)
		if !ok {
			label = status
		}
		anchor, anchorNotes, anchorDot := anchorCell(s)
		rows = append(rows, row{
			RowNo:       fmt.Sprintf("%02d", i+1),
			Domain:      s.Domain,
			Origin:      s.Origin,
			Status:      status,
			Label:       label,
			LastSize:    s.LastSize,
			HasCoverage: s.Coverage.Set,
			SinceSize:   s.Coverage.Size,
			SinceTime:   coverageTime(s.Coverage),
			Checkpoint:  s.LastSize,
			Anchor:      anchor,
			AnchorNotes: anchorNotes,
			AnchorDot:   anchorDot,
		})
	}
	return rows
}

// schemaDeclaration is the verbatim note.$schema URI of a declaration record (the
// default subject of an inclusion proof, CLAUDE.md "Declaration"). The dashboard
// interprets it to pick the hero's "recently declared" ids; the store stays
// schema-agnostic (ADR-0008), so this interpretation lives here in the view layer
// (the "projection" pattern). It MUST equal the value the indexer stores verbatim —
// the same literal internal/certificate and internal/proofserve already carry; a
// shared constant for all three is a documented future cleanup.
const schemaDeclaration = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"

// recentFetch over-fetches recent records from the store so filtering to
// declarations (skipping past deletions / duplicates) can still reach recentDisplay;
// recentDisplay caps how many ids the hero row renders — one working example is
// enough to click straight through to a Certificate of Inclusion.
const (
	recentFetch   = 60
	recentDisplay = 1
)

// buildRecent fetches the most recent realm-wide records and folds them into the
// hero's "recently declared" shortcut list: the newest declarations (note.$schema ==
// schemaDeclaration), de-duplicated by id and capped at recentDisplay, each carrying
// both its verbatim ISCC:-prefixed id (link text) and its prefix-stripped body (the
// /inclusion/<body> path). It interprets note_schema here in the view layer because
// the store is schema-agnostic (ADR-0008). A record with an empty id is skipped (it
// cannot link). An empty index yields an empty slice, so the template renders no row
// — never an empty "Recently declared:" label. The store read error is surfaced to
// the caller, which fails the page like any other store error.
func buildRecent(ctx context.Context, st *store.Store) ([]recentDecl, error) {
	recs, err := st.RecentRecords(ctx, recentFetch)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, recentDisplay)
	out := make([]recentDecl, 0, recentDisplay)
	for _, rec := range recs {
		if rec.NoteSchema != schemaDeclaration || rec.IsccID == "" || seen[rec.IsccID] {
			continue
		}
		seen[rec.IsccID] = true
		out = append(out, recentDecl{
			IsccID: rec.IsccID,
			Body:   strings.TrimPrefix(rec.IsccID, "ISCC:"),
		})
		if len(out) == recentDisplay {
			break
		}
	}
	return out, nil
}

// anchorCell returns the register's Anchor cell for a hub: its label, the lines beneath
// it and the presentation-only dot keyword. A hub with a confirmed anchor shows the
// block and the size that anchor covers, so comparing the size with the Checkpoint
// column shows how many entries are not yet anchored. "pending" is reserved for a hub
// with no confirmed anchor at all. A confirmed row with no recorded height falls back
// to the word "confirmed" and never reads as block 0. The date is when this monitor
// noticed the confirmation, which is why the cell shows no time of day. Size and date
// are separate lines, like the Coverage since cell, because the column is too narrow to
// hold both without breaking the date.
func anchorCell(s store.HubSummary) (label string, notes []string, dot string) {
	switch s.AnchorState() {
	case store.OTSStatusConfirmed:
		label = "confirmed"
		if s.AnchorHeight > 0 {
			label = fmt.Sprintf("block %d", s.AnchorHeight)
		}
		notes = []string{fmt.Sprintf("size %d", s.AnchorSize)}
		if !s.AnchorSeenAt.IsZero() {
			notes = append(notes, "at "+s.AnchorSeenAt.UTC().Format(time.DateOnly))
		}
		return label, notes, "confirmed"
	case store.OTSStatusPending:
		return "pending", []string{"no confirmed anchor yet"}, "pending"
	default:
		return "not anchored", nil, "none"
	}
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
