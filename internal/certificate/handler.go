// Package certificate serves the realm-wide Certificate of Inclusion at GET
// /inclusion/{iscc_id}. The certificate is keyed on the self-describing ISCC-IDv1
// alone (ADR-0010): the handler decodes the id (realm + 12-bit hub_id), resolves
// the issuing hub's domain via the Hub-List (internal/registry), finds that hub's
// store row, and looks up the id's indexed leaf seqs (ADR-0008 one-to-many) under
// the canonical ISCC:-prefixed key the log writes (projection.go). It then renders
// the Evidence-Ledger certificate page whose §1 SUBJECT clause and subject banner
// are real — the subject id, the resolved hub domain, and the subject position
// (seqs[0]).
//
// The §1 inclusion claim is gated on the accepted tree (ADR-0001 coverage
// honesty): an id is only certified when its earliest indexed seq falls below the
// hub's accepted checkpoint size (seqs[0] < LastSize), mirroring every sibling
// record route. A leaf indexed above the accepted checkpoint (a frozen/failed poll
// left an unaccepted projection) or a hub with no accepted checkpoint yet renders
// an honest cannot-certify state, never an affirmative claim.
//
// The certificate grows clause by clause: §1 Subject and §2 Checkpoint (the
// accepted (size, root) the subject position falls within) are real for a
// certifiable id, alongside the documented honesty states. Clauses §3-§6
// (inclusion proof, signing key, Bitcoin anchor, record history) and the
// downloadable proof bundle are gated placeholders that render nothing yet — later
// sub-steps grow the template without rework. The Download-proof-bundle action
// renders as a disabled placeholder.
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
	"encoding/base64"
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

// certData is the certificate template view-model. For a certifiable id it
// populates the §1 SUBJECT clause + subject banner (subject id, resolved hub
// domain, subject position) and the §2 CHECKPOINT clause (the accepted (size,
// root)). It carries the honest "cannot certify" state with a human-readable
// Reason; the subject id is echoed back even on a not-found so the page names what
// was looked up. The §3-§6 HasClauseX flags are all false so the template's gated
// clause placeholders render nothing yet.
type certData struct {
	// IsccID is the subject id as supplied by the caller (echoed verbatim, never
	// interpreted beyond the decode). It is shown even on a not-found.
	IsccID string
	// Certifiable is true only when the id decoded, resolved to a followed hub, had
	// at least one indexed leaf under the canonical ISCC:-prefixed key, AND that
	// earliest leaf falls within the hub's accepted checkpoint (seqs[0] < LastSize,
	// the accepted-tree cap, ADR-0001). It is the state the subject banner and §1
	// clause render against. When false the page renders the honest not-found state.
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

	// CheckpointSize is the hub's accepted checkpoint tree size (hub.LastSize, the
	// accepted tree the subject position falls within). Meaningful only when
	// HasClause2 — the §2 CHECKPOINT clause renders it.
	CheckpointSize uint64
	// CheckpointRoot is the accepted checkpoint's RFC-6962 tree head at
	// CheckpointSize, base64-Std encoded (matching the log browser and verify-for-me
	// so the root string is byte-identical across surfaces). Read back via
	// store.CheckpointAt; meaningful only when HasClause2.
	CheckpointRoot string

	// HasClause2..6 gate the later clauses (checkpoint, inclusion proof, signing
	// key, Bitcoin anchor, record history). HasClause2 is set when the accepted
	// checkpoint's (size, root) is read for a certifiable id; the rest are false in
	// this skeleton so their gated placeholders render nothing; later sub-steps set
	// them.
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
//  4. Look up the id's indexed seqs under the canonical ISCC:-prefixed key — none →
//     "not found in log".
//  5. Gate the affirmative claim on the accepted tree (ADR-0001 coverage honesty):
//     no accepted checkpoint yet (LastSize == 0) → "no accepted checkpoint yet";
//     seqs[0] >= LastSize (indexed but above the accepted checkpoint) → "not in
//     accepted tree"; only seqs[0] < LastSize certifies, with subject position
//     seqs[0] (ascending, the deterministic default, ADR-0008).
//  6. For a certifiable id, read the accepted checkpoint's root back via
//     CheckpointAt(hub.LastSize) and populate the §2 CHECKPOINT clause with the
//     accepted (size, root). A DB error here is a 500; an absent row leaves §2
//     unrendered (no fabricated checkpoint).
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

	hub, ok, err := followedHub(r, st, domain)
	if err != nil {
		return certData{}, http.StatusInternalServerError
	}
	if !ok {
		data.Domain = domain
		data.Reason = "hub not followed by this monitor"
		return data, http.StatusOK
	}
	data.Domain = domain

	// Canonicalize the lookup id to the stored form: logclient writes iscc_id
	// VERBATIM and ISCC:-prefixed (projection.go:31-32), so a PATH route must query
	// the prefixed form. TrimPrefix accepts either /inclusion/MAIG… or
	// /inclusion/ISCC:MAIG… and never double-prefixes; index.iscPrefix is unexported,
	// so the literal "ISCC:" is used here (matching how cert.html carries literal
	// /_ds/ paths). rawID is still echoed as data.IsccID for display.
	lookupID := "ISCC:" + strings.TrimPrefix(rawID, "ISCC:")
	seqs, err := st.SeqsForISCCID(r.Context(), hub.HubID, lookupID)
	if err != nil {
		return certData{}, http.StatusInternalServerError
	}
	if len(seqs) == 0 {
		data.Reason = "not found in log"
		return data, http.StatusOK
	}

	// Accepted-tree cap (Correctness rule: coverage honesty, ADR-0001). PollHub
	// writes iscc_index projections BEFORE the consistency/freeze checks and
	// AdvanceAccepted, so iscc_index can hold projections ABOVE the accepted
	// LastSize (the documented http-surface trap). Gate the affirmative inclusion
	// claim on the accepted checkpoint, mirroring every sibling record route
	// (serveInclusion/serveEntries/serveRecord cap at leafIndex/seq >= size). A
	// frozen hub's LastSize is its last ACCEPTED size (freeze stops advance,
	// ADR-0006), so the same cap correctly caps a frozen hub at its accepted window.
	if hub.LastSize == 0 {
		data.Reason = "no accepted checkpoint yet"
		return data, http.StatusOK
	}
	// seqs is ascending (SeqsForISCCID ORDER BY seq), so seqs[0] is the earliest
	// indexed candidate — the right one to gate on.
	if seqs[0] >= hub.LastSize {
		data.Reason = "not in accepted tree"
		return data, http.StatusOK
	}

	// iscc_id → seq is one-to-many and schema-agnostic (ADR-0008): the subject
	// position defaults to seqs[0] — the first committed seq is the deterministic
	// default (seqs ascending), matching serveVerify. Nothing about the id is
	// interpreted.
	data.Certifiable = true
	data.Position = seqs[0]

	// §2 CHECKPOINT: render the accepted (size, root) the cap above keys on. The
	// size is hub.LastSize (already proven > 0 by the cap), so only the root needs a
	// store read. CheckpointAt reads back the accepted root the follow_state does not
	// persist (ADR-0001 coverage honesty: only the accepted-tree checkpoint, never a
	// contradicted one). A DB error is a 500 (buffered before any 200); a found ==
	// false is the rare honest gap — leave HasClause2 false rather than fabricate a
	// root (AdvanceAccepted records the checkpoint at the same tree_size it advances
	// LastSize to, so found is realistically always true on this path). The root is
	// base64-Std encoded to match the log browser and verify-for-me.
	root, _, found, err := st.CheckpointAt(r.Context(), hub.HubID, hub.LastSize)
	if err != nil {
		return certData{}, http.StatusInternalServerError
	}
	if found {
		data.CheckpointSize = hub.LastSize
		data.CheckpointRoot = base64.StdEncoding.EncodeToString(root)
		data.HasClause2 = true
	}
	return data, http.StatusOK
}

// followedHub maps a resolved hub domain to the monitor's store hub summary,
// reporting ok=false when no followed hub matches that domain (a
// resolved-but-not-followed hub, the honest "hub not followed" verdict). It reads
// every hub summary (the same read the dossier uses) and matches on Domain. The
// matched HubSummary carries both the HubID and the accepted LastSize the
// accepted-tree cap reads, so buildData gates the inclusion claim with no second
// store round-trip. A store read error returns a non-nil error the caller maps to
// a 500.
func followedHub(r *http.Request, st *store.Store, domain string) (store.HubSummary, bool, error) {
	summaries, err := st.ListHubs(r.Context())
	if err != nil {
		return store.HubSummary{}, false, err
	}
	for _, s := range summaries {
		if s.Domain == domain {
			return s, true, nil
		}
	}
	return store.HubSummary{}, false, nil
}
