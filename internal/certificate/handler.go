// Package certificate serves the realm-wide Certificate of Inclusion at GET
// /inclusion/{iscc_id}. The certificate is keyed on the self-describing ISCC-IDv1
// alone (ADR-0010): the handler decodes the id (realm + 12-bit hub_id), resolves
// the issuing hub's domain via the Hub-List (internal/registry), finds that hub's
// store row, and looks up the id's indexed leaf seqs (ADR-0008 one-to-many). It
// then renders the Evidence-Ledger certificate page whose §1 SUBJECT clause and
// subject banner are real — the subject id, the resolved hub domain, and the
// subject position (seqs[0]).
//
// This is the verifiable skeleton of the certificate: §1 Subject plus the
// documented honesty states. Clauses §2-§6 (checkpoint, inclusion proof, signing
// key, Bitcoin anchor, record history) and the downloadable proof bundle are gated
// placeholders that render nothing yet — later sub-steps grow the template without
// rework. The Download-proof-bundle action renders as a disabled placeholder.
//
// Fail-closed / coverage-honesty discipline (ADR-0001): every "cannot certify"
// branch — a malformed id, an id resolving to no listed slot, a resolved domain
// with no followed hub, or an id with no indexed leaf — is a 200 with an honest
// explanation, NEVER a 5xx and NEVER a fabricated proof. A decode error is a
// verdict, not a fault. Only a genuine infra fault (a ListHubs / SeqsForISCCID DB
// error or a template render error) is a 500, and the page is rendered into a
// buffer first so such a fault is a 500 BEFORE any 200 is committed.
//
// The oracle/conformance gate is N/A for this skeleton: it is a pure HTML render
// of a decode + a registry resolve + a store SeqsForISCCID lookup, touching no
// signature, RFC-6962, Merkle, did:web, fsck, or proof path. The gate APPLIES to
// the later proof-bundle sub-step (the served bundle's inclusion proof must be
// mutation-proven non-vacuous), not here.
package certificate

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/iscc/iscc-monitor/internal/index"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
)

// PathPrefix is the realm-wide subtree this handler is mounted at. http.ServeMux
// subtree matching delivers paths like /inclusion/MAIGHFECJMOPMIAB; the raw id is
// the suffix after this prefix. It is exported so cmd/iscc-monitor mounts the
// handler and derives the path in one place.
const PathPrefix = "/inclusion/"

// pageTemplate is the embedded certificate template, parsed once at package init so
// a malformed template fails the build, not a request.
//
//go:embed cert.html
var pageTemplate string

// tmpl is the parsed certificate template. It is html/template (NOT text/template)
// so the id, domain, and position auto-escape. template.Must panics at init if the
// source fails to parse, surfacing a template bug at startup.
var tmpl = template.Must(template.New("certificate").Parse(pageTemplate))

// StatusSource reports a hub's current in-memory glossary status by hub_id. It is
// the read seam later clauses use to overlay the live poll verdict (the richer
// unresolvable / unverified states the store cannot prove) onto the store-provable
// subset. ok is false when no live status is recorded for the hub. *metrics.Registry
// satisfies it structurally via its Status method; the certificate takes the
// interface, not the concrete package, so it never imports internal/metrics
// (mirroring dossier.StatusSource). The skeleton accepts it for forward-compatible
// wiring; the §1 subject clause does not yet consult it.
type StatusSource interface {
	Status(hubID int64) (string, bool)
}

// certData is the certificate template view-model. The skeleton populates only the
// §1 SUBJECT clause and the subject banner: whether the id could be certified, the
// subject id, the resolved hub domain, and the subject position. NotFound carries
// the honest "cannot certify" state with a human-readable Reason; the subject id is
// echoed back even on a not-found so the page names what was looked up. The §2-§6
// HasClauseX flags are all false in the skeleton so the template's gated clause
// placeholders render nothing yet.
type certData struct {
	// IsccID is the subject id as supplied by the caller (echoed verbatim, never
	// interpreted beyond the decode). It is shown even on a not-found.
	IsccID string
	// Certifiable is true only when the id decoded, resolved to a followed hub, and
	// had at least one indexed leaf — the state the subject banner and §1 clause
	// render against. When false the page renders the honest not-found state.
	Certifiable bool
	// Domain is the resolved issuing-hub domain (e.g. sb1.amlet.id), shown in the
	// subject banner and §1 clause. Empty until the id resolves to a followed hub.
	Domain string
	// Position is the subject leaf seq (seqs[0]; ascending, the deterministic
	// default matching serveVerify, ADR-0008). Meaningful only when Certifiable.
	Position uint64
	// Reason is the human-readable explanation rendered in the not-found state
	// (e.g. "not a valid ISCC-ID", "not found in log"). Empty when Certifiable.
	Reason string

	// HasClause2..6 gate the later clauses (checkpoint, inclusion proof, signing
	// key, Bitcoin anchor, record history). All false in this skeleton so the
	// gated placeholders render nothing; later sub-steps set them.
	HasClause2 bool
	HasClause3 bool
	HasClause4 bool
	HasClause5 bool
	HasClause6 bool
}

// Handler returns an http.Handler that serves the realm-wide Certificate of
// Inclusion at the /inclusion/ subtree. It decodes the id from the path suffix,
// resolves the issuing hub via the Hub-List, finds that hub's store row, and looks
// up the id's indexed leaf seqs, then renders the §1 SUBJECT clause + subject
// banner for a certifiable id or an honest 200 "cannot certify" state otherwise.
//
// Only GET is served (any other method is 405, mirroring dossier). A bare
// /inclusion/ (empty id) is the honest "no id supplied" 200 state. Every
// cannot-certify branch is a 200 (ADR-0001 fail-closed); only a genuine infra
// fault (a ListHubs / SeqsForISCCID DB error or a template render error) is a 500,
// detected before any 200 is committed (buffer-then-200).
//
// hubList resolves a decoded hub_id slot to the issuing hub's domain. st must be
// non-nil (the binary always passes the real store). statuses is the in-memory
// status overlay accepted for forward-compatible wiring; the skeleton does not
// consult it. A nil hubList or nil statuses is tolerated: a nil hubList makes every
// id resolve to "not in this realm".
func Handler(hubList *registry.HubList, st *store.Store, statuses StatusSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rawID := strings.TrimPrefix(r.URL.Path, PathPrefix)
		data, status := buildData(r, hubList, st, rawID)
		if status != http.StatusOK {
			http.Error(w, "internal server error", status)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// Post-200 write-drop: the status is already committed, so a copy error can
		// only signal a broken client connection, which a second status cannot fix
		// (matching dossier / proofserve).
		_, _ = buf.WriteTo(w)
	})
}

// buildData runs the decode→resolve→store-lookup chain for rawID and returns the
// certificate view-model plus the HTTP status to use. The status is http.StatusOK
// for every certifiable id AND every cannot-certify verdict (ADR-0001 fail-closed:
// a decode/resolve/not-in-log miss is a verdict, not a fault), and only
// http.StatusInternalServerError for a genuine store fault (a ListHubs /
// SeqsForISCCID DB error). The caller renders the returned data on a 200 status
// and writes a plain 500 otherwise.
//
// The chain, each step's miss being an honest 200 verdict:
//  1. Decode the id — a decode error → "not a valid ISCC-ID".
//  2. Resolve the hub_id slot via the Hub-List — an unknown slot → "not in this realm".
//  3. Find the resolved domain's followed-hub row — none → "hub not followed by this monitor".
//  4. Look up the id's indexed seqs — none → "not found in log"; else the subject
//     position is seqs[0] (ascending, the deterministic default, ADR-0008).
func buildData(r *http.Request, hubList *registry.HubList, st *store.Store, rawID string) (certData, int) {
	if rawID == "" {
		return certData{Reason: "no ISCC-ID supplied"}, http.StatusOK
	}
	data := certData{IsccID: rawID}

	id, err := index.Decode(rawID)
	if err != nil {
		data.Reason = "not a valid ISCC-ID"
		return data, http.StatusOK
	}

	if hubList == nil {
		data.Reason = "not found in this realm"
		return data, http.StatusOK
	}
	domain, ok := hubList.Resolve(id.HubID)
	if !ok {
		data.Reason = "not found in this realm"
		return data, http.StatusOK
	}

	hubID, ok, err := followedHub(r, st, domain)
	if err != nil {
		return certData{}, http.StatusInternalServerError
	}
	if !ok {
		data.Domain = domain
		data.Reason = "hub not followed by this monitor"
		return data, http.StatusOK
	}

	seqs, err := st.SeqsForISCCID(r.Context(), hubID, rawID)
	if err != nil {
		return certData{}, http.StatusInternalServerError
	}
	if len(seqs) == 0 {
		data.Domain = domain
		data.Reason = "not found in log"
		return data, http.StatusOK
	}

	// iscc_id → seq is one-to-many and schema-agnostic (ADR-0008): the subject
	// position defaults to seqs[0] — the first committed seq is the deterministic
	// default (seqs ascending), matching serveVerify. Nothing about the id is
	// interpreted.
	data.Certifiable = true
	data.Domain = domain
	data.Position = seqs[0]
	return data, http.StatusOK
}

// followedHub maps a resolved hub domain to the monitor's store hub_id, reporting
// ok=false when no followed hub matches that domain (a resolved-but-not-followed
// hub, the honest "hub not followed" verdict). It reads every hub summary (the same
// read the dossier uses) and matches on Domain. A store read error returns a
// non-nil error the caller maps to a 500.
func followedHub(r *http.Request, st *store.Store, domain string) (int64, bool, error) {
	summaries, err := st.ListHubs(r.Context())
	if err != nil {
		return 0, false, err
	}
	for _, s := range summaries {
		if s.Domain == domain {
			return s.HubID, true, nil
		}
	}
	return 0, false, nil
}
