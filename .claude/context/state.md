<!-- assessed-at: 34e189e7e6684e0db3868d59d5ee0b10b8fd1b9b -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — certificate clause-by-clause; the proof-bundle download-link critical is CLOSED, gate green at HEAD. Remaining: §5 BITCOIN ANCHOR (OTS-blocked), the anchor panels, WASM verifier, OTS anchoring, and the M-UI visual-exit gate.

The certificate of inclusion (`internal/certificate`, `GET /inclusion/{iscc_id}`) now has five of its
six numbered clauses sound (§1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY, §6 RECORD
HISTORY) and a working downloadable proof bundle (endpoint + a now-fixed download link for both id
forms). §5 BITCOIN ANCHOR remains blocked on the absent OTS store seam. M1/M2/M3 remain fully met. The
proof-bundle cycle plus the `#ZgotmplZ` link fix are pushed and **CI is green at HEAD (`34e189e`)**.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): partially met.** Met: five-status `HubStatusBadge`
    (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/`
    realm-index grid; `/<domain>/log/` browser; hub dossier (`GET /<domain>`, `internal/dossier`);
    frozen Exhibit; paginated record list (`GET /<domain>/log/records`); single-record page
    (`GET /<domain>/log/record?index=<seq>`); ISCC-IDv1 decoder (`internal/index.Decode`);
    `(realm, hub_id) → domain` Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT +
    §2 CHECKPOINT + §3 INCLUSION PROOF + §4 SIGNING KEY + §6 RECORD HISTORY** (all sound); the
    proof-bundle **endpoint** (`.bundle`, gated on §3 re-verification, oracle-cross-checked) **and its
    download LINK** (canonical path-rooted `BundleHref`, works for bare AND `ISCC:`-prefixed forms —
    critical CLOSED at `34e189e`). **Open:** certificate **§5 BITCOIN ANCHOR** (`HasClause5`
    deliberately false until the OTS seam exists); the separate **Bitcoin-anchor vs comparison-anchor**
    panels; and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012 agent-browser).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/source). §5 of
    the certificate is its downstream consumer, so OTS gates the last certificate clause too.
- **Last ~10 iterations: ~7 milestone-Verify-advancing / ~3 refactor·polish·gate.** The arc builds the
  certificate clause-by-clause (badge → DS tokens → webfonts → `/` grid → log browser → dossier →
  frozen Exhibit → record list → single record → ISCC-IDv1 decoder → Hub-List resolver → cert §1 → §2
  → §3 → §4 → §6 → proof-bundle endpoint → #ZgotmplZ link fix). No drift — the loop converges on real
  Verify criteria; the gate correctly caught a green-but-broken headline affordance (bundle endpoint
  passed Go tests but the page link was dead for the canonical id form) and held it at NEEDS_WORK, then
  the next cycle closed it cleanly. The remaining work (§5 / OTS / WASM / visual exit gate) is the
  genuinely large, partly-blocked tail, not avoidable polish.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `89641a4..HEAD` diff touched ONLY `internal/certificate`
(handler.go + cert.html + bundle_test.go) and `.claude/*` context/learnings — **no M1 source touched.**
All M1 Verify criteria remain satisfied: `origin`/`vkey` golden; all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival;
coverage tracked; structured logs; `/metrics` served over HTTP.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **20 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- All three triggers WIRED + golden-tested inside `logclient.CheckConsistency`; `AcceptCheckpoint`
  4-way verdict threads `VerifiedContext` into the hub-key cache upsert + `fsckMirror`. `cmd/notecheck`
  is the fully-independent signature-parity oracle, shelled out in CI. `store/*.go` uses
  `modernc.org/sqlite` with ADR-0005/0007 single-writer discipline.
- **Missing (M1 connective tissue, off the Verify bar):** real alert transport (`alertFunc` is a WARN
  `slog` emit); warm-path second `did.json` resolve.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`+`proof.VerifyInclusion`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (Hub-List parser).
  **Not wired:** `nbd-wtf/opentimestamps`, `github.com/iscc/iscc-lib/packages/go` (ADR-0011, not yet
  adopted — see Quality gates).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched in the diff. Both Verify criteria exercised:
fsck root-rebuild WIRED on every verified non-frozen poll; inclusion cross-check conformance-tested over
the real verified mirror. All three computed proofs — `inclusion`, `consistency`, `entries` — served
from the local mirror, never re-hitting the hub.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry `no-cache` +
strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — proof-bundle ENDPOINT + download LINK both landed and verified; §5 +
anchor panels + visual-exit gate remain.** §1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING
KEY, and §6 RECORD HISTORY are all sound. Badge, DS shell, `/` index, `/<domain>/log/` browser, hub
dossier, frozen Exhibit, record list, single-record page, the ISCC-IDv1 decoder, and the Hub-List
resolver are all built and verified.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` browser; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /<domain>/log/record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode`
  (`GOOS=js GOARCH=wasm`-buildable); the fail-closed `(realm, hub_id) → domain` Hub-List resolver.
- **§1 SUBJECT (verified):** `internal/certificate.Handler` decodes the id → `hubList.Resolve` →
  `followedHub`/`ListHubs` → `SeqsForISCCID`, accepted-tree-capped, canonicalized `"ISCC:" + TrimPrefix`.
- **§2 CHECKPOINT (verified):** for a certifiable id `buildData` reads the accepted root via
  `CheckpointAt(LastSize)`, sets `HasClause2` + size/root (base64-Std); rendered only when `HasClause2`.
- **§3 INCLUSION PROOF (verified):** builds the RFC-6962 proof of `data.Position` against `hub.LastSize`
  over a `SQLiteFetcher`, gates `HasClause3` on `proof.VerifyInclusion(...) == nil` against the §2
  accepted root. Fails closed; mutation-proven.
- **§4 SIGNING KEY (verified):** derives the key id from the §2 checkpoint's own raw signature line,
  looks it up via `store.LookupHubKey`, fails closed (no §4, no 500) on a malformed sig or cache miss.
- **§6 RECORD HISTORY (verified):** the full accepted-tree-capped one-to-many seq list (declaration +
  any later deletion), each labelled by verbatim `note.$schema`, rendered unconditionally for a
  certifiable id.
- **Proof-bundle ENDPOINT (verified):** `serveBundle` writes a self-contained
  `{checkpoint, inclusion, record, key}` JSON, dispatched on the `.bundle` suffix inside `Handler`,
  gated on the SAME §3 re-verification (`HasBundle == HasClause3`). Declines honestly (200, no
  attachment) on a tile gap or contradictory tree; `InclusionEvidence` re-verifies via the external
  oracle. Endpoint mutation-proven.
- **Proof-bundle download LINK (verified — critical CLOSED at `34e189e`):** `cert.html:397` now links
  `href="{{.BundleHref}}"`, a canonical path-rooted, `ISCC:`-prefix-free field built in `buildData`
  (`PathPrefix + strings.TrimPrefix(rawID,"ISCC:") + bundleSuffix`). Resolves to a working
  `/inclusion/<bare-id>.bundle` for BOTH the bare and the `ISCC:`-prefixed request forms — no more
  `#ZgotmplZ`. `TestCertificateProofBundleLinkRendered` now loops over both id forms and asserts the
  `#ZgotmplZ` sentinel is absent; review (`34e189e`) confirms it is mutation-proven non-vacuous and
  Codex-clean. This satisfies the M-UI "offers a downloadable proof bundle … the certificate page links
  it via an enabled download action" criterion.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` deliberately false;
  BLOCKED on a non-existent OTS/anchor store seam); the separate **Bitcoin-anchor vs comparison-anchor**
  panels; and the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together when handler.go is
    next touched (`%3A`-encode the port).
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in any source file). The
`internal/badge`, `internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves
are WASM-shareable primitives the verifier app will reuse, but the verifier itself does not exist. OTS
is the upstream blocker for certificate §5.

## Quality gates
**Status**: **GREEN at HEAD.** The proof-bundle cycle + the `#ZgotmplZ` link fix are pushed; HEAD
(`34e189e`) is in sync with `origin/develop` (0 ahead, 0 behind). The latest `review` verdict
(`34e189e`) is **PASS / CONTINUE** with `mise run check` green (all 21 packages `ok`), the oracle gate
N/A for a pure rendering fix, and Codex clean.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. **Latest
  CI run on HEAD (`34e189e`) concluded `success`.**
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" mandates **Go 1.26** + the
  **`iscc-lib/packages/go` v0.5.0** codec, but `go.mod` is still `go 1.24.0` with no iscc-lib require.
  Deliberate, sequenced, not-yet-started increment — target and code disagree on the stack until it lands.
- **Open issues: 0 `critical`, 4 `normal`, 6 `low`.** The proof-bundle download-link critical is CLOSED.
  The 4 `normal`: (a) [human] adopt iscc-lib codec + bump to Go 1.26 (ADR-0011, foundational);
  (b) [review/Codex] Hub-List `hubDomain` `ForceQuery` fail-open; (c) [review/Codex] §4 AND bundle
  `did:web:` + raw domain mis-render a `host:port` hub's DID (2 surfaces); (d) [review] §6 RECORD HISTORY
  omits the per-record `· at` timestamp. The 6 `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone with the critical now closed.** The next
`define-next`/`advance` should pick from:

1. **OTS / Bitcoin anchoring** — build the anchor store seam + stamp/upgrade loop, which then
   **unblocks certificate §5 BITCOIN ANCHOR** and the Bitcoin-anchor panel. §5 cannot be honestly
   rendered until this exists. Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle
   sites) when handler.go is next touched.
2. **WASM verifier** remains a v1 milestone (1/1 Verify open) — `internal/proof/verify` →
   `GOOS=js GOARCH=wasm` plus the `monitor.iscc.codes` Independent Verification app.
3. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the
   `internal/index` tripwire/parity test) — flagged foundational; confirm mise can provision Go 1.26
   before flipping `go.mod`'s `go` directive.
4. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.
