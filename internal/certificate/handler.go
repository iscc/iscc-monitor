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
// The certificate grows clause by clause: §1 Subject, §2 Checkpoint (the accepted
// (size, root) the subject position falls within), §3 Inclusion Proof (the
// RFC-6962 leaf→siblings→root chain recomputed from the hub's mirrored tiles), and
// §4 Signing Key (the did:web-resolved Ed25519 key that signed the accepted
// checkpoint) are real for a certifiable id, alongside the documented honesty
// states. §3 is gated on a fail-closed re-VERIFICATION: the built proof must rebuild
// the accepted checkpoint root (proof.VerifyInclusion against the §2 root) before the
// clause renders its ✓. This makes the rendered ✓ true by construction — the
// certificate asserts inclusion only when the proof it shows actually rebuilds the
// root it shows — and fails closed against ANY tile↔root divergence (a
// frozen-after-fork hub whose mirror holds the contradictory tree's tiles, or a
// request landing in the fork-poll window before the freeze commits). The buildData
// §3 branch documents the mechanism. §4 derives the key id from the §2 accepted
// checkpoint's own raw signature line (logclient.KeyIDFromCheckpoint) and reads the
// cached resolution back from hub_keys (store.LookupHubKey), so the certificate can
// only ever show the key that actually signed what §2 vouches for; a cache miss (key
// not yet resolved) honestly omits §4 rather than fabricate a key (ADR-0009: did:web
// is the only key source). §6 RECORD HISTORY lists the full one-to-many set of
// accepted-tree seqs the hub indexed under the subject id (the declaration and any
// later deletion), each labelled by its verbatim note.$schema kind — a store read
// with no crypto path, so it renders unconditionally for a certifiable id. §5 BITCOIN
// ANCHOR surfaces the OpenTimestamps anchor state of the §2 accepted root from the
// mirrored OTS row (store.OTSForRoot classified via ots.ConfirmedFor, which BINDS the
// proof's committed digest to §2's root): a Bitcoin-confirmed root shows the confirming
// block height, a still-pending (calendar-asserted) root shows the honest "pending"
// state, and an un-anchored root (no OTS row, the empty-bytes sentinel, an unparseable
// proof, or a proof whose digest does not commit to §2's root) omits §5 — never an
// error (ADR-0001 / ADR-0004: OTS never faults a surface). The COMPARISON ANCHOR panel reframes §2's accepted
// (size, root) as the monitor's own independently-observed record of what this hub
// showed THIS monitor — the artifact a client checks its own (size, root) against to
// detect a split view (CLAUDE.md "Comparison anchor") — bounded by the coverage window
// (ADR-0001). It is a SEPARATE, distinctly-labelled element from §5: it does not depend
// on the OTS row (a hub with no §5 still renders it) and carries no "anchoring"/Bitcoin
// copy (target.md: "anchoring" stays Bitcoin-only). target.md mandates this panel the
// certificate mockup omits (design-parity: the constraint wins over the mockup).
//
// The downloadable proof bundle is served at GET /inclusion/{iscc_id}.bundle: a
// self-contained JSON artifact {checkpoint (verbatim signed-note text), inclusion
// proof (IsccLogInclusionProof-shaped), record bytes, hub key} a client verifies on
// its own (the Proof-bundle / Verifiable-cache contract — removing the monitor from
// the trust path). It is offered ONLY when the §3 re-verification succeeded (the same
// fail-closed gate as the §3 ✓): the certificate's "Download proof bundle" action is
// an enabled link to .bundle when HasBundle, the disabled placeholder otherwise, and a
// .bundle request for a non-certifiable id is an honest 200 {error:…}, never a
// fabricated bundle. The OTS member (§5) is omitted until the OTS store seam exists.
// The bundle reuses buildData's §3 crypto path verbatim (the verified proof + record
// bytes it already computed), so it never re-derives Merkle.
//
// Fail-closed / coverage-honesty discipline (ADR-0001): every "cannot certify"
// branch — a malformed id, an id resolving to no listed slot, a resolved domain
// with no followed hub, or an id with no indexed leaf — is a 200 with an honest
// explanation, NEVER a 5xx and NEVER a fabricated proof. A decode error is a
// verdict, not a fault. Only a genuine infra fault (a ListHubs / SeqsForISCCID DB
// error or a template render error) is a 500, and the page is rendered into a
// buffer first so such a fault is a 500 BEFORE any 200 is committed.
//
// The oracle/conformance gate APPLIES from §3 onward: the §3 inclusion proof is the
// real RFC-6962 path recomputed from the mirror (logclient.InclusionProofFromTiles
// over store.SQLiteFetcher), so its test is mutation-proven non-vacuous against
// testonly.Tree.InclusionProof (the independent prover). The earlier §1 subject and
// §2 checkpoint clauses touch no crypto path (a decode + a registry resolve + store
// reads), so the gate was N/A there.
package certificate

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/index"
	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/ots"
	"github.com/iscc/iscc-monitor/internal/proof/verify"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// PathPrefix is the realm-wide subtree this handler is mounted at. http.ServeMux
// subtree matching delivers paths like /inclusion/MAIGHFECJMOPMIAB; the raw id is
// the suffix after this prefix. It is exported so cmd/iscc-monitor mounts the
// handler and derives the path in one place.
const PathPrefix = "/inclusion/"

// didWeb builds a hub's did:web identifier from its domain, percent-encoding the
// port colon (host:port -> did:web:host%3Aport) so the DID denotes the same host
// the signing key was resolved from. did:web reads a bare colon as a path-segment
// boundary, so an unencoded host:port would name a different did.json than
// didweb.DocumentURL fetches; replacing only the FIRST colon leaves a clean,
// no-port domain (e.g. sb0.iscc.id) byte-identical. It mirrors the resolver's
// idiom (internal/didweb uses strings.Replace(host, ":", "%3A", 1)).
func didWeb(domain string) string {
	return "did:web:" + strings.Replace(domain, ":", "%3A", 1)
}

// pageTemplate is the embedded certificate template, parsed once at package init so
// a malformed template fails the build, not a request.
//
//go:embed cert.html
var pageTemplate string

// tmpl is the parsed certificate template. It is html/template (NOT text/template)
// so the id, domain, and position auto-escape. template.Must panics at init if the
// source fails to parse, surfacing a template bug at startup.
var tmpl = template.Must(template.New("certificate").Parse(pageTemplate))

// Record-kind labels for the §6 RECORD HISTORY clause, mapping the verbatim
// note.$schema to a human-readable kind. The schema constants are the FULL wire URIs
// the iscc_index projection stores verbatim (CLAUDE.md's iscc-note-0.8.0 is prose
// shorthand, never the wire value); they mirror internal/proofserve's recordKind
// (the same schema→label mapping the single-record page uses). The mapping is the
// ONLY interpretation §6 performs (ADR-0008: verification is schema-agnostic): an
// unknown or empty schema is the catch-all kindUnknown and is listed verbatim
// alongside its seq, never gating or erroring the clause.
const (
	schemaDeclaration = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"
	schemaDeletion    = "http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"
	kindDeclaration   = "Declaration"
	kindDeletion      = "Deletion"
	kindUnknown       = "Unknown record type"
)

// recordKind maps the verbatim note.$schema to a §6 row label and a deletion flag —
// the only interpretation §6 performs (ADR-0008). It is fail-open by construction:
// the declaration / deletion schemas map to their friendly labels, and anything else
// (including an empty schema) is kindUnknown, so an unknown or empty schema never
// gates or errors the clause; the row is listed regardless. It mirrors
// internal/proofserve.recordKind so the certificate and the single-record page agree.
func recordKind(noteSchema string) (label string, isDeletion bool) {
	switch noteSchema {
	case schemaDeclaration:
		return kindDeclaration, false
	case schemaDeletion:
		return kindDeletion, true
	default:
		return kindUnknown, false
	}
}

// Default masthead identity copy used when an identity field is left empty, so an
// unconfigured deployment renders today's static placeholder rather than a false
// claim. These MUST stay byte-identical to internal/dashboard's instanceFallback /
// operatorFallback consts (and the dossier's copy): the certificate, dossier, and
// dashboard mastheads are required to render the same chrome, and neither package
// can import the other's unexported consts, so the defaults are duplicated here as
// literals.
const (
	instanceFallback = "monitor instance"
	operatorFallback = "independent Trust & Transparency service · ISCC-Hub network"
)

// resolveIdentity applies the certificate-side fail-safe for the masthead identity,
// mirroring dashboard.Identity.resolve semantics so the certificate, dossier, and
// dashboard chrome stay in lockstep: a blank Instance or Operator falls back to the
// static placeholder copy. It lives here (not in internal/dashboard) so the fallback
// is seam-testable at the certificate HTTP boundary without making internal/dashboard
// a fourth edited file (its resolve is unexported). Realm has no slot on the
// certificate masthead (like the dossier's, it carries no realm subtitle), so it is
// ignored.
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

// HistoryRow is one §6 RECORD HISTORY entry: an accepted-tree leaf the hub indexed
// under the subject id, with its human-readable kind Label (from the verbatim
// note.$schema, recordKind), an IsDeletion flag the clause folds into HasDeletion,
// and the row's verbatim note.timestamp At. Seq is the leaf's absolute index, listed
// verbatim and never interpreted (ADR-0008).
type HistoryRow struct {
	// Seq is the leaf's absolute index in the accepted tree (seq < CheckpointSize).
	Seq uint64
	// Label is the row's human-readable kind (Declaration / Deletion / unknown), from
	// recordKind over the row's verbatim note.$schema.
	Label string
	// IsDeletion is true only for a deletion record; the clause ORs it into HasDeletion
	// to decide whether to render the deletion note.
	IsDeletion bool
	// At is the record's verbatim note.timestamp (RFC-3339) from the iscc_index projection, or "" when the
	// note carried none (rendered conditionally). It is displayed verbatim and never parsed (ADR-0008).
	At string
}

// certData is the certificate template view-model. For a certifiable id it
// populates the §1 SUBJECT clause + subject banner (subject id, resolved hub
// domain, subject position), the §2 CHECKPOINT clause (the accepted (size, root)),
// the §3 INCLUSION PROOF clause (the RFC-6962 sibling-hash chain that rebuilds the
// accepted root from the subject leaf), the §4 SIGNING KEY clause (the
// did:web-resolved key that signed the §2 accepted checkpoint), and the §6 RECORD
// HISTORY clause (the full one-to-many list of accepted-tree seqs indexed under the
// subject id, each labelled by its note.$schema kind). §3 is withheld unless the
// built proof is re-verified against the accepted root, so a hub whose mirror
// diverges from its accepted root renders §1+§2 but no §3 (the rebuild gate in
// buildData). §4 is withheld unless the key the accepted checkpoint was signed with
// is found in the hub_keys cache, so a hub whose key is not yet resolved renders
// §1+§2(+§3) but no §4 (the cache-miss decline in buildData). §6 renders
// unconditionally for a certifiable id (a store read, no crypto gate to fail closed
// on — the seqs are the accepted-tree projections §1 already certified against). It
// carries the honest "cannot certify" state with a human-readable Reason; the subject
// id is echoed back even on a not-found so the page names what was looked up. The §5
// HasClause5 flag stays false so the template's gated Bitcoin-anchor placeholder
// renders nothing yet (the OTS store seam does not exist).
type certData struct {
	// IsccID is the subject id as supplied by the caller (echoed verbatim, never
	// interpreted beyond the decode). It is shown even on a not-found.
	IsccID string
	// BundleHref is the path-rooted, ISCC:-prefix-free href the "Download proof
	// bundle" action links to (/inclusion/<bare-id>.bundle). It is built canonically
	// — a leading slash and the stripped ISCC: prefix both remove the scheme
	// ambiguity, so html/template's URL escaper emits it verbatim instead of the
	// #ZgotmplZ sentinel it produces for the raw ISCC:-prefixed form. The .bundle
	// handler decodes the bare form identically. Meaningful only when HasBundle (the
	// template reads it under {{if .HasBundle}}); populated only on the certifiable
	// path so a non-certifiable id leaves it empty.
	BundleHref string
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
	// store.CheckpointAt; meaningful only when HasClause2. It doubles as the §3 root
	// chip — the chain the inclusion proof rebuilds is the same accepted root.
	CheckpointRoot string
	// ProofHashes is the §3 INCLUSION PROOF sibling chain: the RFC-6962 inclusion
	// proof of the subject leaf (Position) against the accepted tree (CheckpointSize),
	// recomputed from the hub's mirrored tiles via
	// logclient.InclusionProofFromTiles. Each hash is base64-Std encoded (matching
	// CheckpointRoot and verify-for-me's writeEvidence so the strings are
	// byte-identical across surfaces). Meaningful only when HasClause3; it may be
	// empty (a single-leaf tree has an empty-but-valid proof), so HasClause3 gates on
	// the proof rebuilding the accepted root, not on len(ProofHashes) > 0. It is
	// never populated unless the built proof re-verifies against the accepted root
	// (the rebuild gate withholds §3 on any tile↔root divergence).
	ProofHashes []string
	// RecordB64 is the subject leaf's raw record bytes (the same arts.record §3
	// computed and the bundle base64-encodes), base64-Std encoded for the tier-2
	// in-browser verifier. The certificate's progressive-enhancement <script> feeds
	// it — together with CheckpointRoot, ProofHashes, Position, and CheckpointSize —
	// into the WASM globalThis.isccVerifyInclusion so the BROWSER re-verifies the same
	// inclusion proof the server's §3 already re-verified (the two-tier honesty: the
	// tier-2 ✓ is a genuine re-VERIFICATION, not a status flag). It is populated only
	// on the §3 success path (inside the HasBundle branch), so it is empty on every
	// honest decline; the template reads it only under {{if .HasBundle}}, exposed
	// through a JSON data island (never string-interpolated into executable JS).
	RecordB64 string

	// SigningKeyDID is the hub's did:web identifier (didWeb(Domain): "did:web:" +
	// the domain with its port colon %3A-encoded), the §4 SIGNING KEY clause subject
	// (ADR-0009: domain ownership is identity). Meaningful only when HasClause4.
	SigningKeyDID string
	// SigningKeyID is the BE-uint32 signed-note keyhash of the key that signed the §2
	// accepted checkpoint, hex-formatted (%08x, matching how the codebase prints key
	// ids). It is derived from the accepted checkpoint's own raw signature line
	// (logclient.KeyIDFromCheckpoint), so it always names the key that actually signed
	// what §2 vouches for. Meaningful only when HasClause4.
	SigningKeyID string
	// SigningKeyMultibase is the cached key's z6Mk… multibase Ed25519 public key
	// (store.HubKey.PubkeyZ). It may be empty (the cache row stored a NULL pubkey_z);
	// the template renders this chip conditionally. Meaningful only when HasClause4.
	SigningKeyMultibase string
	// SigningKeyRevoked is the cached key's revocation instant (RFC-3339), set only
	// when the hub_keys row carries a non-zero revoked_at; empty otherwise (the common
	// case). It surfaces the cached revocation timestamp as-is; this clause does NOT
	// evaluate CID 1.0 validity windows. Meaningful only when HasClause4.
	SigningKeyRevoked string

	// BTCConfirmed reports whether the §5 BITCOIN ANCHOR is Bitcoin-confirmed (the
	// mirrored OpenTimestamps proof carries a Bitcoin attestation) rather than still
	// pending (calendar-asserted, awaiting confirmation). The template renders the
	// confirmed block height when true and the honest "pending" state when false (a
	// not-yet-anchored root is NOT an error). Meaningful only when HasClause5.
	BTCConfirmed bool
	// BTCHeight is the confirming Bitcoin block height of the §5 anchor (ots.ConfirmedFor),
	// rendered only when BTCConfirmed. Meaningful only when HasClause5.
	BTCHeight int64
	// BTCConfirmedAt is the §5 anchor's confirmation instant (RFC-3339), from the
	// mirrored OTS row's upgraded_at; empty when the row carries no upgrade time (the
	// template renders it conditionally, mirroring SigningKeyRevoked's zero-time guard).
	// Meaningful only when HasClause5 && BTCConfirmed.
	BTCConfirmedAt string

	// CoverageSize is the monitor's coverage-start tree size for this hub
	// (HubSummary.Coverage.Size, monitored_since_size), the lower bound of the window
	// the comparison anchor's observation is honest over (ADR-0001: guarantees hold
	// only from coverage start). Meaningful only when HasComparisonAnchor &&
	// HasCoverageWindow.
	CoverageSize uint64
	// CoverageSince is the monitor's coverage-start instant for this hub (RFC-3339,
	// from HubSummary.Coverage.Since); empty when coverage has not started yet
	// (Coverage.Set false), so the template renders the honest "coverage just started"
	// state instead of implying a pre-coverage guarantee (the same zero-guard as
	// SigningKeyRevoked). Meaningful only when HasComparisonAnchor.
	CoverageSince string

	// RecordHistory is the §6 RECORD HISTORY rows: every accepted-tree seq the hub
	// indexed under the subject id (seqs ascending, capped to seq < CheckpointSize),
	// each labelled by its note.$schema kind (recordKind). iscc_id → seq is
	// one-to-many (ADR-0008): a declaration and its later deletion share an id and
	// list as two rows. It always has at least the subject position (seqs[0]) for a
	// certifiable id, so the list is non-empty when HasClause6.
	RecordHistory []HistoryRow
	// HasDeletion is true when any row in RecordHistory is a deletion; the template
	// renders the "a deletion is a new record — the declaration is preserved" note
	// only then. Meaningful only when HasClause6.
	HasDeletion bool

	// HasClause2..6 gate the later clauses (checkpoint, inclusion proof, signing
	// key, Bitcoin anchor, record history). HasClause2 is set when the accepted
	// checkpoint's (size, root) is read for a certifiable id; HasClause3 when the
	// inclusion proof built from the mirror re-verifies against that accepted root
	// (proof.VerifyInclusion succeeds, so the rendered ✓ is true by construction);
	// HasClause4 when the key that signed the accepted checkpoint is found in the
	// hub_keys cache (an honest cache-miss decline leaves it false, never a fabricated
	// key); HasClause5 when the accepted root has a mirrored OpenTimestamps proof with
	// non-empty bytes that ots.ConfirmedFor could classify AND whose committed digest
	// equals that root (confirmed → BTCHeight, pending → the honest "pending" state); an
	// un-anchored root (no OTS row, the empty-bytes sentinel, a proof ots.ConfirmedFor
	// cannot parse, or a proof whose digest does not commit to the root) leaves it false
	// so §5 is omitted, never an error; HasClause6 for every certifiable id (a store read of the
	// accepted-tree record history, no crypto gate to fail closed on).
	HasClause2 bool
	HasClause3 bool
	HasClause4 bool
	HasClause5 bool
	HasClause6 bool

	// HasComparisonAnchor gates the COMPARISON ANCHOR panel — the monitor's
	// independently-observed record of the (size, root) this hub showed THIS monitor,
	// the artifact a client checks its own (size, root) against to detect a split view
	// (CLAUDE.md "Comparison anchor"; NOT a witness, the deferred M7 role). It is the §2
	// accepted (size, root) reframed as the monitor's own observation, bounded by the
	// coverage window — never Bitcoin, never a re-verification, never "anchoring" copy
	// (target.md: "anchoring" stays Bitcoin-only). It is set inside the HasClause2 guard
	// (the anchor is meaningful only when there is an accepted (size, root) to anchor),
	// so a hub WITH no §5 OTS row still renders it (it does NOT depend on the Bitcoin
	// anchor — the two panels are decoupled, distinctly-labelled elements).
	//
	// Mockup deviation (design-parity rule: the constraint wins over the mockup):
	// target.md mandates a comparison-anchor panel the certificate mockup omits, so this
	// panel is rendered beyond the mockup. Flagged here rather than silently dropped.
	HasComparisonAnchor bool
	// HasCoverageWindow reports whether the monitor has a recorded coverage window for
	// this hub (HubSummary.Coverage.Set). When true the panel states the window (since
	// CoverageSize · CoverageSince); when false it renders the honest "coverage just
	// started" state, never implying a pre-coverage guarantee (ADR-0001 coverage
	// honesty). Meaningful only when HasComparisonAnchor.
	HasCoverageWindow bool

	// HasBundle is set to HasClause3 (the §3 re-verification gate): the downloadable
	// proof bundle is offered ONLY when the built inclusion proof actually rebuilt the
	// accepted root, the same fail-closed gate that renders the §3 ✓. The template shows
	// the enabled "Download proof bundle" link when true and the disabled placeholder
	// otherwise, so the page never offers a bundle the monitor cannot assemble.
	HasBundle bool

	// Instance is this deployment's configured instance domain rendered in the
	// masthead identity block (the chrome-instance line), so the certificate chrome
	// is honest per-deployment instead of a static placeholder. It is the resolved
	// dashboard.Identity.Instance (falling back to instanceFallback when unset), set
	// in Handler on the value buildData returns — on EVERY branch (the masthead
	// renders on the certifiable AND the cannot-certify path), so every honest 200
	// carries it.
	Instance string
	// Operator is this deployment's configured operator/realm line rendered in the
	// masthead identity block (the chrome-operator line), the resolved
	// dashboard.Identity.Operator (falling back to operatorFallback when unset). Like
	// Instance it is set in Handler on every branch so the masthead is honest on
	// every honest 200.
	Operator string
}

// bundleArtifacts carries the raw, in-hand artifacts buildData computes on the §3
// crypto path so serveBundle can assemble the downloadable proof bundle without
// re-deriving any Merkle. The fields are meaningful only when the certData's
// HasClause3 is set (the §3 re-verification succeeded); on any honest decline they
// stay zero and serveBundle declines to offer a bundle. The HTML path ignores this
// value entirely — the page renders from certData alone.
type bundleArtifacts struct {
	// raw is the verbatim signed-note checkpoint text (CheckpointAt's raw), the body a
	// client checks the hub signature on. Carried as text (not base64), matching
	// InclusionEvidence.Checkpoint.
	raw []byte
	// record is the subject leaf's raw record bytes (RecordBytesFromBundle), base64-Std
	// encoded by serveBundle.
	record []byte
	// builtProof is the verified RFC-6962 inclusion proof of the subject leaf against
	// the accepted tree — the same [][]byte §3 base64-encoded into ProofHashes. It fed
	// the proof.VerifyInclusion gate, so it rebuilds the accepted root by construction.
	builtProof [][]byte
	// keyID is the BE-uint32 signed-note keyhash of the key that signed the accepted
	// checkpoint (KeyIDFromCheckpoint), set only when §4 found the cached key. zero when
	// the key is not cached (the bundle then omits the key member, never fabricates one).
	keyID uint32
	// key is the cached did:web key the accepted checkpoint was signed with
	// (store.LookupHubKey). hasKey reports whether it was found; when false the bundle
	// omits the key member rather than emit an empty one.
	key    store.HubKey
	hasKey bool
}

// bundleKey is the proof bundle's signing-key member: the BE-uint32 key id (hex), the
// z6Mk… multibase Ed25519 public key, and the cached revocation instant (RFC-3339,
// omitted when not revoked). A client resolves the hub's did:web document and checks
// this is the key that signed the checkpoint (ADR-0009: did:web is the only key
// source); the bundle carries the cached resolution, never an independent claim.
type bundleKey struct {
	ID        string `json:"id"`
	Multibase string `json:"multibase,omitempty"`
	Revoked   string `json:"revoked,omitempty"`
}

// bundleHub is the proof bundle's hub member: the resolved domain and its did:web
// identifier (didWeb(domain): "did:web:" + the domain with its port colon
// %3A-encoded). A client uses the DID to resolve the signing key.
type bundleHub struct {
	Domain string `json:"domain"`
	DID    string `json:"did"`
}

// proofBundle is the self-contained, machine-readable proof bundle served at GET
// /inclusion/{iscc_id}.bundle for a certifiable id whose §3 inclusion proof rebuilt
// the accepted root. It is the Proof-bundle / Verifiable-cache contract artifact: a
// client verifies it on its own — checking the checkpoint signature, re-running the
// RFC-6962 inclusion proof, and hashing the record — removing the monitor from the
// trust path. The bundle is offered ONLY when the §3 re-verification succeeded (the
// same fail-closed gate as the §3 ✓), never on a flag.
//
// The Inclusion member is shaped exactly like logclient.InclusionEvidence (the hub's
// IsccLogInclusionProof VC evidence member), so it feeds straight into
// logclient.VerifyInclusionEvidence — the external-oracle cross-check. The OTS /
// Bitcoin-anchor member (§5) is omitted until the OTS store seam exists; it is never
// fabricated.
type proofBundle struct {
	IsccID     string                      `json:"iscc_id"`
	Hub        bundleHub                   `json:"hub"`
	Checkpoint string                      `json:"checkpoint"`
	Inclusion  logclient.InclusionEvidence `json:"inclusion"`
	Record     string                      `json:"record"`
	Key        *bundleKey                  `json:"key,omitempty"`
}

// bundleSuffix marks a bundle request: GET /inclusion/<id>.bundle returns the
// downloadable proof bundle (JSON), everything else the HTML certificate page. The
// suffix (over a ?format= query) gives the download a clean filename and a distinct
// path while keeping the whole /inclusion/ subtree inside this one handler.
const bundleSuffix = ".bundle"

// Handler returns an http.Handler that serves the realm-wide Certificate of
// Inclusion at the /inclusion/ subtree. It decodes the id from the path suffix,
// resolves the issuing hub via the Hub-List, finds that hub's store row, and looks
// up the id's indexed leaf seqs, then renders the §1 SUBJECT clause + subject
// banner for a certifiable id or an honest 200 "cannot certify" state otherwise.
//
// A trailing .bundle on the id (GET /inclusion/<id>.bundle) instead serves the
// downloadable proof bundle as JSON — the self-contained {checkpoint, inclusion
// proof, record bytes, hub key} artifact a client verifies on its own. The suffix is
// detected and stripped before the id is decoded, so the two surfaces share the same
// decode→resolve→build chain; the bundle is offered only when the §3 re-verification
// succeeded (serveBundle), otherwise an honest 200 "no proof bundle available".
//
// Only GET is served (any other method is 405, mirroring dossier). A bare
// /inclusion/ (empty id) is the honest "no id supplied" 200 state. Every
// cannot-certify branch is a 200 (ADR-0001 fail-closed); only a genuine infra
// fault (a ListHubs / SeqsForISCCID DB error or a template render error) is a 500,
// detected before any 200 is committed (buffer-then-200).
//
// hubList resolves a decoded hub_id slot to the issuing hub's domain. It is the
// registry.HubResolver behavior, not a fixed snapshot, so the binary passes its
// hot-swappable *registry.AtomicHubList and an hourly realm refresh updates the
// mapping under the handler without a restart. st must be non-nil (the binary
// always passes the real store). statuses is the in-memory status overlay accepted
// for forward-compatible wiring; the skeleton does not consult it. A nil hubList or
// nil statuses is tolerated: a nil hubList makes every id resolve to "not in this
// realm".
//
// id is this deployment's configured masthead identity (the SAME dashboard.Identity
// value the / and dossier mastheads render), resolved once via resolveIdentity and
// set on the certData buildData returns so the certificate chrome is honest
// per-deployment; an unconfigured binary (a zero-value Identity) falls back to
// today's static placeholder copy. The masthead renders on EVERY path (certifiable
// AND cannot-certify), so the fields are set on both the HTML and the .bundle branch
// (serveBundle ignores them — the assignment is harmless and kept uniform).
func Handler(hubList registry.HubResolver, st *store.Store, statuses StatusSource, id dashboard.Identity) http.Handler {
	instance, operator := resolveIdentity(id)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rawID := strings.TrimPrefix(r.URL.Path, PathPrefix)
		// No-JS hero-form fallback: an HTML <form method="get"> can only emit a query
		// string (?iscc_id=…), never a path segment, so the dashboard's claim-lookup
		// hero posts to a bare /inclusion/ with ?iscc_id=<id>. When the path id is empty
		// AND a query id is present, use the query value so it flows through the
		// identical decode→resolve→render chain; a bare /inclusion/ with no query stays
		// the honest "no id supplied" 200.
		if rawID == "" {
			if qID := r.URL.Query().Get("iscc_id"); qID != "" {
				rawID = qID
			}
		}
		// Detect+strip the .bundle suffix BEFORE decoding the id (the suffix is not
		// part of the id), so /inclusion/<id> and /inclusion/<id>.bundle share the same
		// decode→resolve→build chain.
		if bareID, ok := strings.CutSuffix(rawID, bundleSuffix); ok {
			data, arts, status := buildData(r, hubList, st, bareID)
			if status != http.StatusOK {
				http.Error(w, "internal server error", status)
				return
			}
			// Set the masthead identity on the value buildData returns so it is uniform
			// across every branch (buildData has certData{} literal early returns that
			// would bypass any field set inside it); serveBundle ignores these fields.
			data.Instance, data.Operator = instance, operator
			serveBundle(w, data, arts)
			return
		}
		data, _, status := buildData(r, hubList, st, rawID)
		if status != http.StatusOK {
			http.Error(w, "internal server error", status)
			return
		}
		// Set the masthead identity on the value buildData returns (NOT inside buildData,
		// whose certData{} literal early-return branches would bypass it) so every honest
		// 200 carries the configured chrome, on the certifiable AND cannot-certify path.
		data.Instance, data.Operator = instance, operator
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

// serveBundle writes the downloadable proof bundle for the certificate built into
// data. It is offered ONLY when data.HasBundle (== HasClause3, the §3
// re-verification gate): the built inclusion proof actually rebuilt the accepted
// root. When the §3 re-verification declined (a tile/bundle gap, ErrLeafOutOfBundle,
// or a proof that did not rebuild the root), the bundle request is an honest 200
// {error:…} "no proof bundle available", never a fabricated bundle and never a 5xx
// for a coverage gap (a genuine DB fault was already a 500 from buildData). The
// bundle's inclusion member is shaped like logclient.InclusionEvidence so a client
// (or VerifyInclusionEvidence) re-verifies it against the mirrored tiles.
//
// It keeps proofserve.writeEvidence's drop-the-write-error-after-200 posture: a
// marshal of a fixed-shape struct of strings/uints/[]string cannot fail for content
// reasons, so a mid-write fault cannot un-send the 200.
func serveBundle(w http.ResponseWriter, data certData, arts bundleArtifacts) {
	w.Header().Set("Content-Type", "application/json")
	if !data.HasBundle {
		// Honest "not available": the §3 re-verification declined (or the id is not
		// certifiable), so there is no verified proof to package. A 200 verdict, never a
		// fabricated bundle, never a 5xx for a coverage gap (ADR-0001 fail-closed).
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"iscc_id": data.IsccID,
			"error":   "no proof bundle available for this id",
		})
		return
	}

	encoded := make([]string, len(arts.builtProof))
	for i, h := range arts.builtProof {
		encoded[i] = base64.StdEncoding.EncodeToString(h)
	}
	bundle := proofBundle{
		IsccID: data.IsccID,
		Hub: bundleHub{
			Domain: data.Domain,
			DID:    didWeb(data.Domain),
		},
		Checkpoint: string(arts.raw),
		Inclusion: logclient.InclusionEvidence{
			Type:           "IsccLogInclusionProof",
			Checkpoint:     string(arts.raw),
			TreeSize:       data.CheckpointSize,
			LeafIndex:      data.Position,
			InclusionProof: encoded,
		},
		Record: base64.StdEncoding.EncodeToString(arts.record),
	}
	if arts.hasKey {
		key := &bundleKey{
			ID:        fmt.Sprintf("%08x", arts.keyID),
			Multibase: arts.key.PubkeyZ,
		}
		if !arts.key.Revoked.IsZero() {
			key.Revoked = arts.key.Revoked.UTC().Format(time.RFC3339)
		}
		bundle.Key = key
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+data.IsccID+`.bundle.json"`)
	w.WriteHeader(http.StatusOK)
	// Drop-the-write-error-after-200 (matching proofserve.writeEvidence): the fixed
	// shape cannot fail to marshal for content reasons, and a mid-write fault cannot
	// un-send the 200.
	_ = json.NewEncoder(w).Encode(bundle)
}

// buildData runs the decode→resolve→store-lookup chain for rawID and returns the
// certificate view-model, the raw artifacts the downloadable proof bundle reuses, and
// the HTTP status to use. The status is http.StatusOK for every certifiable id AND
// every cannot-certify verdict (ADR-0001 fail-closed: a decode/resolve/not-in-log
// miss is a verdict, not a fault), and only http.StatusInternalServerError for a
// genuine store fault (a ListHubs / SeqsForISCCID DB error). The HTML caller renders
// the returned data on a 200 status and writes a plain 500 otherwise; the bundle
// caller reads the artifacts. The artifacts are meaningful only when data.HasBundle
// (== HasClause3) is set — the same §3 re-verification gate as the page ✓, so the
// bundle reuses the §3 crypto path verbatim instead of re-deriving Merkle.
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
//  7. For a certifiable id, recompute the RFC-6962 inclusion proof of the subject
//     leaf (seqs[0]) against the accepted tree (hub.LastSize) from the hub's mirrored
//     tiles (InclusionProofFromTiles over a SQLiteFetcher), then re-verify it against
//     the accepted root (proof.VerifyInclusion); §3 renders ONLY when that
//     re-verification succeeds — see the rebuild-gate paragraph at the §3 branch. A
//     tile/bundle not yet mirrored (os.ErrNotExist, or ErrLeafOutOfBundle) leaves §3
//     unrendered (an honest gap, NOT a 500 — the certificate can decline a clause,
//     unlike verify-for-me which has committed to serving a proof); a proof that does
//     not rebuild the accepted root silently declines §3; any other build/read error
//     is a 500 (buffered before any 200).
//  8. For a certifiable id, derive the key id from the §2 accepted checkpoint's own
//     raw signature line (KeyIDFromCheckpoint) and read the cached did:web key back
//     from hub_keys (LookupHubKey); §4 renders the did:web identifier + key id (+ the
//     cached multibase) ONLY when that key is cached. A KeyIDFromCheckpoint error
//     (malformed sig line) or a cache miss leaves §4 unrendered (an honest "key not
//     yet resolved" decline, never a fabricated key — ADR-0009 did:web is the only key
//     source); only a real LookupHubKey DB fault is a 500 (buffered before any 200).
//  9. For a certifiable id, surface the §5 BITCOIN ANCHOR of the §2 accepted root:
//     read the mirrored OTS row (OTSForRoot keyed on the §2 root bytes) and classify a
//     non-empty proof via ots.ConfirmedFor, which BINDS the proof's committed digest to
//     the §2 root so §5 vouches the anchor only for a proof that provably commits to
//     that root. A confirmed proof renders the block height (+ the upgrade instant), a
//     calendar-only proof the honest "pending" state. No OTS row, the empty-bytes
//     sentinel, a proof ots.ConfirmedFor cannot parse, or a proof whose digest does not
//     commit to the §2 root leaves §5 unrendered (an un-anchored root is NOT an error);
//     only a real OTSForRoot DB fault is a 500 (buffered before any 200).
//     9b. For a certifiable id, render the COMPARISON ANCHOR panel: §2's accepted
//     (size, root) reframed as the monitor's own independently-observed record of what
//     this hub showed THIS monitor (CLAUDE.md "Comparison anchor"), bounded by the
//     coverage window (followedHub's Coverage, no new read). It is a SEPARATE,
//     distinctly-labelled element from §5 — it does not depend on the OTS row — and
//     carries no "anchoring"/Bitcoin copy (target.md invariant). No new error path.
//  10. For a certifiable id, list the §6 RECORD HISTORY: the accepted-tree seqs (seq <
//     LastSize) from the same SeqsForISCCID result, each read via RecordAt for its
//     note.$schema and labelled by recordKind (declaration / deletion / unknown). It
//     renders unconditionally (a store read, no crypto gate); a RecordAt miss is an
//     honest gap (the seq lists with the unknown label), only a real DB fault is a 500.
func buildData(r *http.Request, hubList registry.HubResolver, st *store.Store, rawID string) (certData, bundleArtifacts, int) {
	var arts bundleArtifacts
	if rawID == "" {
		return certData{Reason: "no ISCC-ID supplied"}, arts, http.StatusOK
	}
	data := certData{IsccID: rawID}

	id, err := index.Decode(rawID)
	if err != nil {
		data.Reason = "not a valid ISCC-ID"
		return data, arts, http.StatusOK
	}

	if hubList == nil {
		data.Reason = "not found in this realm"
		return data, arts, http.StatusOK
	}
	domain, ok := hubList.Resolve(id.HubID)
	if !ok {
		data.Reason = "not found in this realm"
		return data, arts, http.StatusOK
	}

	hub, ok, err := followedHub(r, st, domain)
	if err != nil {
		return certData{}, arts, http.StatusInternalServerError
	}
	if !ok {
		data.Domain = domain
		data.Reason = "hub not followed by this monitor"
		return data, arts, http.StatusOK
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
		return certData{}, arts, http.StatusInternalServerError
	}
	if len(seqs) == 0 {
		data.Reason = "not found in log"
		return data, arts, http.StatusOK
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
		return data, arts, http.StatusOK
	}
	// seqs is ascending (SeqsForISCCID ORDER BY seq), so seqs[0] is the earliest
	// indexed candidate — the right one to gate on.
	if seqs[0] >= hub.LastSize {
		data.Reason = "not in accepted tree"
		return data, arts, http.StatusOK
	}

	// iscc_id → seq is one-to-many and schema-agnostic (ADR-0008): the subject
	// position defaults to seqs[0] — the first committed seq is the deterministic
	// default (seqs ascending), matching serveVerify. Nothing about the id is
	// interpreted.
	data.Certifiable = true
	data.Position = seqs[0]
	// Build the proof-bundle download href canonically: path-rooted at the mount
	// (PathPrefix) and ISCC:-prefix-free. cert.html cannot use the raw .IsccID here —
	// for the ISCC:-prefixed request form html/template's URL escaper reads the
	// leading ISCC: as an unknown scheme and emits the #ZgotmplZ sentinel, breaking
	// the headline download link. A leading "/" plus the stripped prefix make this an
	// unambiguous path the escaper passes through verbatim, and the .bundle handler
	// decodes the bare form identically (decode is prefix-agnostic), so it resolves to
	// the same bundle.
	data.BundleHref = PathPrefix + strings.TrimPrefix(rawID, "ISCC:") + bundleSuffix

	// §2 CHECKPOINT: render the accepted (size, root) the cap above keys on. The
	// size is hub.LastSize (already proven > 0 by the cap), so only the root needs a
	// store read. CheckpointAt reads back the accepted root the follow_state does not
	// persist (ADR-0001 coverage honesty: only the accepted-tree checkpoint, never a
	// contradicted one). A DB error is a 500 (buffered before any 200); a found ==
	// false is the rare honest gap — leave HasClause2 false rather than fabricate a
	// root (AdvanceAccepted records the checkpoint at the same tree_size it advances
	// LastSize to, so found is realistically always true on this path). The root is
	// base64-Std encoded to match the log browser and verify-for-me.
	root, raw, found, err := st.CheckpointAt(r.Context(), hub.HubID, hub.LastSize)
	if err != nil {
		return certData{}, arts, http.StatusInternalServerError
	}
	if found {
		data.CheckpointSize = hub.LastSize
		data.CheckpointRoot = base64.StdEncoding.EncodeToString(root)
		data.HasClause2 = true
		// Carry the verbatim signed-note checkpoint text for the proof bundle (the body
		// a client checks the hub signature on). Meaningful only once HasBundle is set.
		arts.raw = raw
	}

	// §3 INCLUSION PROOF: recompute the RFC-6962 inclusion proof of the subject leaf
	// (data.Position == seqs[0]) against the accepted tree (hub.LastSize) from the
	// hub's mirrored tiles, THEN re-verify it rebuilds the accepted root before
	// rendering. This is the first certificate clause on the Merkle path; it reuses
	// the same oracle-gated builder + proof.VerifyInclusion verify-for-me uses
	// (serveInclusion/serveVerify), so the certificate never hand-rolls Merkle math.
	// The cap above already proved hub.LastSize > 0 and seqs[0] < hub.LastSize, so the
	// leaf is in range. A valid proof can be empty (a single-leaf tree), so HasClause3
	// gates on the proof REBUILDING the root, not on the proof length. §3 also renders
	// the accepted root chip (data.CheckpointRoot), so it is meaningful only alongside §2.
	//
	// Rebuild gate (the proof.VerifyInclusion check below): InclusionProofFromTiles is
	// a pure builder — it folds whatever tile bytes the mirror returns and never checks
	// the proof rebuilds the accepted root, so a proof BUILT is not a proof VERIFIED.
	// On a self-verifiable surface the rendered ✓ must be gated on a re-VERIFICATION
	// against the accepted root, not on a status flag read from a separate, racily
	// updated row. The divergence window is the frozen-after-fork case: the follower
	// ingests the contradictory candidate tiles (follower.go:174) BEFORE the freeze
	// check and returns early BEFORE AdvanceAccepted/fsckMirror, so a hub can hold the
	// contradictory tree's tiles in the mirror while CheckpointAt(LastSize) still
	// returns the OLD accepted root — and because the HTTP server runs concurrently
	// with the follower, a request can land in that window before the freeze flag
	// commits. Building §3 from those tiles would render a sibling chain under a
	// `root … ✓` the siblings do not rebuild — a self-contradictory certificate that
	// violates the Proof-bundle / Verifiable-cache contract (a client verifies the
	// artifact itself) and ADR-0006 (freeze preserves evidence, never advances accepted
	// state). Re-verifying the built proof against the accepted root fails closed
	// against ALL of these (steady-state frozen AND the fork-poll race) without reading
	// any status flag. §1/§2 are unaffected (they read the irreplaceable
	// accepted-checkpoint record via CheckpointAt, which a fork cannot corrupt), so
	// only §3 — the mirror-tile read — gets the gate. Fail-closed (ADR-0001): when in
	// doubt about the mirror, decline the clause.
	f := store.SQLiteFetcher{Store: st, HubID: hub.HubID}
	builtProof, err := logclient.InclusionProofFromTiles(r.Context(), f.ReadTile, data.Position, hub.LastSize)
	if err != nil {
		// A tile not yet mirrored is an honest gap, not a fault: leave §3 unrendered
		// (the page still shows §1 + §2) rather than fabricating a proof or 500ing.
		// proofserve maps this to a 404 because it has committed to serving a proof;
		// the clause-by-clause certificate can decline a clause honestly (as §2 does
		// on a missing checkpoint row). Any non-os.ErrNotExist build error is a real
		// fault → 500 (buffered before any 200), matching §2's split.
		if !errors.Is(err, os.ErrNotExist) {
			return certData{}, arts, http.StatusInternalServerError
		}
	} else if data.HasClause2 {
		// Read the subject leaf's raw record bytes from the mirrored entry bundle (the
		// same way serveVerify does), derive its RFC-6962 leaf hash, and verify the built
		// proof rebuilds the accepted root captured by §2. The final bundle of a
		// non-multiple-of-256 tree is a partial, so request its expected p (the
		// SQLiteFetcher does the partial→full fallback).
		bundleIndex := data.Position / tiles.TileWidth
		offset := data.Position % tiles.TileWidth
		p := tiles.PartialTileSize(0, bundleIndex, hub.LastSize)
		bundle, err := f.ReadEntryBundle(r.Context(), bundleIndex, p)
		if err != nil {
			// A bundle not yet mirrored is the same honest gap as a missing tile: leave
			// §3 unrendered. Any other read fault → 500 (buffered before any 200).
			if !errors.Is(err, os.ErrNotExist) {
				return certData{}, arts, http.StatusInternalServerError
			}
		} else {
			record, err := logclient.RecordBytesFromBundle(bundle, offset)
			switch {
			case errors.Is(err, logclient.ErrLeafOutOfBundle):
				// The leaf's bundle is mirrored but does not yet cover it — an honest gap,
				// not a fault. Leave §3 unrendered.
			case err != nil:
				return certData{}, arts, http.StatusInternalServerError
			default:
				// re-verify via proof/verify (the shared pure core). data.Position <
				// hub.LastSize is already guaranteed by the §1 accepted-tree cap, so the
				// precondition branch is unreachable; a non-nil error or a false verdict is
				// a SILENT decline of §3 (the proof did not rebuild the accepted root),
				// never a 500 — the certificate can decline a clause.
				if ok, _ := verify.VerifyInclusion(record, data.Position, hub.LastSize, builtProof, root); ok {
					hashes := make([]string, len(builtProof))
					for i, h := range builtProof {
						hashes[i] = base64.StdEncoding.EncodeToString(h)
					}
					data.ProofHashes = hashes
					data.HasClause3 = true
					// Carry the base64-Std record for the tier-2 in-browser verifier: the
					// certificate's <script> feeds it (with CheckpointRoot, ProofHashes,
					// Position, CheckpointSize) into the WASM isccVerifyInclusion so the
					// browser re-runs the SAME inclusion proof the server just re-verified.
					// Set here, inside the §3 re-verification gate, so it is empty on every
					// honest decline (the template reads it only under {{if .HasBundle}}).
					data.RecordB64 = base64.StdEncoding.EncodeToString(record)
					// The §3 re-verification succeeded, so a verified proof bundle exists.
					// Offer it (HasBundle) and carry the in-hand artifacts (the verified
					// proof + the record bytes) so serveBundle reuses this crypto path
					// verbatim instead of re-deriving Merkle. This is the SINGLE gate for
					// both the page ✓ and the bundle (the load-bearing fail-closed rule).
					data.HasBundle = true
					arts.builtProof = builtProof
					arts.record = record
				}
			}
		}
	}

	// §4 SIGNING KEY: show the did:web-resolved key the §2 accepted checkpoint was
	// signed with. The key id is recovered from the accepted checkpoint's OWN raw
	// signature line (KeyIDFromCheckpoint reads the BE-uint32 keyhash without resolving
	// did.json — it does not verify the signature), so the displayed key is always the
	// one that actually signed what §2 vouches for; it equals KeyIDFromVerifier of the
	// resolved vkey. The clause is meaningful only alongside §2 (it grounds itself in
	// the accepted checkpoint's raw bytes), so it renders inside the HasClause2 guard.
	//
	// Cache-miss decline (ADR-0009: did:web is the only key source): §4 reads the
	// cached resolution back from hub_keys (LookupHubKey, populated by the follower's
	// cacheHubKey on every verified poll) and renders ONLY on a cache hit. A
	// KeyIDFromCheckpoint error (a synthetic/garbled checkpoint with no well-framed sig
	// line) or a cache miss leaves §4 unrendered — an honest "key not yet resolved"
	// decline, NOT a 500 and NOT a fabricated key (the same discipline as §3's honest
	// tile-gap). Only a real LookupHubKey DB fault is a 500 (buffered before any 200).
	if data.HasClause2 {
		if _, keyID, err := logclient.KeyIDFromCheckpoint(raw); err == nil {
			key, found4, err := st.LookupHubKey(r.Context(), hub.HubID, keyID)
			if err != nil {
				return certData{}, arts, http.StatusInternalServerError
			}
			if found4 {
				data.SigningKeyDID = didWeb(data.Domain)
				data.SigningKeyID = fmt.Sprintf("%08x", keyID)
				data.SigningKeyMultibase = key.PubkeyZ
				if !key.Revoked.IsZero() {
					data.SigningKeyRevoked = key.Revoked.UTC().Format(time.RFC3339)
				}
				data.HasClause4 = true
				// Carry the cached key for the proof bundle's key member (the same
				// cache hit §4 renders). When no key is cached the bundle omits the
				// member rather than fabricate one (ADR-0009: did:web is the only key
				// source), so the bundle's key clause mirrors the §4 honest decline.
				arts.keyID = keyID
				arts.key = key
				arts.hasKey = true
			}
		}
	}

	// §5 BITCOIN ANCHOR: surface the OpenTimestamps anchor state of the §2 accepted
	// root from the mirrored OTS row (store.OTSForRoot, keyed on the same
	// (hub_id, tree_size, root) the .ots route and the stamp loop use — root is §2's
	// raw CheckpointAt bytes, NOT the base64 CheckpointRoot string). The clause is
	// meaningful only alongside §2 (it anchors §2's root), so it renders inside the
	// HasClause2 guard. Three honest, fail-closed states (ADR-0001 / ADR-0004: OTS
	// never crashes a surface):
	//   - No OTS row (found == false) OR the empty-OTSBytes sentinel (a root stamped
	//     at observation but not yet calendar-submitted, the load-bearing edge case the
	//     .ots route guards): the root is not yet anchored — leave HasClause5 false so
	//     §5 is OMITTED. An un-anchored root is NOT an error.
	//   - A non-empty proof ots.ConfirmedFor cannot parse (a garbage/malformed blob) OR
	//     whose committed SHA-256 digest does NOT equal §2's accepted root (a mis-stamped
	//     row): a SILENT decline (HasClause5 stays false), NEVER a 500 — the same
	//     discipline as §3's non-nil VerifyInclusion silent decline. The digest binding
	//     (ots.ConfirmedFor, NOT the digest-agnostic ots.Confirmed) upgrades the classify
	//     into a verification that the proof actually anchors THIS root, so §5 cannot
	//     vouch "block N" for a root the proof does not commit to. OTS must never fault
	//     the surface.
	//   - A parseable, digest-bound proof: render §5. A Bitcoin-attested proof shows the
	//     confirming block height (+ the upgrade instant when the row carries one); a
	//     calendar-only proof shows the honest "pending" state (calendar-asserted,
	//     awaiting Bitcoin confirmation), never an error (target.md: a not-yet-anchored
	//     root renders the normal "pending" state). Only a genuine OTSForRoot DB fault is
	//     a 500 (buffered before any 200, like every other clause).
	if data.HasClause2 {
		rec, found, err := st.OTSForRoot(r.Context(), hub.HubID, hub.LastSize, root)
		if err != nil {
			return certData{}, arts, http.StatusInternalServerError
		}
		if found && len(rec.OTSBytes) > 0 {
			if confirmed, height, cerr := ots.ConfirmedFor(rec.OTSBytes, root); cerr == nil {
				data.HasClause5 = true
				data.BTCConfirmed = confirmed
				if confirmed {
					data.BTCHeight = height
					if !rec.UpgradedAt.IsZero() {
						data.BTCConfirmedAt = rec.UpgradedAt.UTC().Format(time.RFC3339)
					}
				}
			}
		}

		// COMPARISON ANCHOR: the monitor's independently-observed record of the
		// (size, root) this hub showed THIS monitor — the artifact a client checks its
		// own (size, root) against to detect a split view (CLAUDE.md "Comparison
		// anchor"). It is §2's accepted (size, root) reframed as the monitor's own
		// observation (no new read, no re-encode — it reuses data.CheckpointSize /
		// data.CheckpointRoot), PLUS the coverage window that bounds the claim (ADR-0001:
		// guarantees hold only from coverage start). It is a SEPARATE, distinctly-labelled
		// element from the §5 Bitcoin anchor: it does NOT depend on the OTS row, so a hub
		// with no §5 still renders it (target.md: the two anchor panels are decoupled and
		// "anchoring" copy stays Bitcoin-only). No new error path — the data is already in
		// hand, so this panel cannot 500 on its own; it renders inside the HasClause2
		// guard and stays absent (like §2) when there is no accepted checkpoint. The
		// coverage window rides out of followedHub's HubSummary (no second store
		// round-trip): when Coverage.Set is true the panel states the window, when false
		// it renders the honest "coverage just started" state (the zero-time guard mirrors
		// SigningKeyRevoked / §5's UpgradedAt).
		data.HasComparisonAnchor = true
		if hub.Coverage.Set {
			data.HasCoverageWindow = true
			data.CoverageSize = hub.Coverage.Size
			if !hub.Coverage.Since.IsZero() {
				data.CoverageSince = hub.Coverage.Since.UTC().Format(time.RFC3339)
			}
		}
	}

	// §6 RECORD HISTORY: list the full one-to-many set of accepted-tree seqs the hub
	// indexed under the subject id — the declaration plus any later deletion (iscc_id →
	// seq is one-to-many, ADR-0008). seqs is already in hand from SeqsForISCCID
	// (ascending), so each row needs one RecordAt for its verbatim note.$schema, mapped
	// to a kind label by recordKind. Two honesty disciplines apply:
	//   - Cap to the accepted tree (ADR-0001 coverage honesty): list only rows with seq
	//     < hub.LastSize, mirroring §1's accepted-tree cap and every sibling record
	//     route, so a record indexed ABOVE the accepted checkpoint (an unaccepted
	//     projection left by a frozen/failed poll) is never implied to be vouched for.
	//     The history always has at least seqs[0] (the cap above proved seqs[0] <
	//     LastSize), so the list is non-empty for a certifiable id.
	//   - A RecordAt MISS (found == false, a projection gap) is an honest gap, NOT a
	//     500: list the seq with the empty/unknown-schema label rather than dropping it
	//     or erroring the page. Only a real RecordAt DB fault is a 500 (buffered before
	//     any 200, like every other clause). The schema is never gated on (an unknown
	//     schema lists verbatim with the unknown label) — §6 interprets nothing beyond
	//     the kind label.
	// §6 renders unconditionally for a certifiable id: unlike §3/§4 there is no crypto
	// or cache gate to fail closed on — the seqs are the accepted-tree projections §1
	// already certified against.
	var history []HistoryRow
	deletion := false
	for _, seq := range seqs {
		if seq >= hub.LastSize {
			continue
		}
		row, found, err := st.RecordAt(r.Context(), hub.HubID, seq)
		if err != nil {
			return certData{}, arts, http.StatusInternalServerError
		}
		schema := ""
		at := ""
		if found {
			schema = row.NoteSchema
			at = row.NoteTimestamp
		}
		label, isDeletion := recordKind(schema)
		if isDeletion {
			deletion = true
		}
		history = append(history, HistoryRow{Seq: seq, Label: label, IsDeletion: isDeletion, At: at})
	}
	data.RecordHistory = history
	data.HasDeletion = deletion
	data.HasClause6 = true

	return data, arts, http.StatusOK
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
