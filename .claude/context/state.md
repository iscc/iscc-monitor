<!-- assessed-at: 427a09ed5d8f6bf3e91ebae80104eab029fe3ca5 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — certificate clause-by-clause; §6 RECORD HISTORY landed.
The certificate of inclusion (`internal/certificate`, `GET /inclusion/{iscc_id}`) now has five of its
six numbered clauses sound and verified: §1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF (fail-closed
`proof.VerifyInclusion`, critical CLOSED), §4 SIGNING KEY, and **§6 RECORD HISTORY — the last review
(`427a09e`) PASSed it WITH_NOTES**: the per-id one-to-many seq list (declaration + any later deletion)
is read from `SeqsForISCCID`/`RecordAt`, accepted-tree-capped, labelled by verbatim `note.$schema`,
and renders unconditionally for a certifiable id (`HasClause6 = true`). §5 BITCOIN ANCHOR is the only
remaining clause and is BLOCKED on the OTS/anchor store seam (does not exist). M1/M2/M3 remain fully
met. The downloadable proof bundle, the separate anchor panels, the M-UI visual-exit gate, the WASM
verifier, and OTS anchoring remain.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): 1 still open** — certificate **§5** (Bitcoin anchor, blocked on
    the OTS store seam), the **downloadable proof-bundle assembler**, the separate Bitcoin-anchor vs
    comparison-anchor panels, and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
    Met: five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell (`/_ds/` tokens +
    self-hosted webfonts, CDN-free); `/` realm-index grid; `/<domain>/log/` log browser; hub dossier
    (`GET /<domain>`, `internal/dossier`); frozen Exhibit; paginated record list
    (`GET /<domain>/log/records`); single-record page (`GET /<domain>/log/record?index=<seq>`);
    ISCC-IDv1 decoder (`internal/index.Decode`); `(realm, hub_id) → domain` Hub-List resolver
    (`internal/registry`); certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION PROOF + §4 SIGNING
    KEY + §6 RECORD HISTORY** (all sound, verified). **Open:** certificate **§5** (`HasClause5` never
    set — line 175/259 comments confirm it deliberately stays `false` until the OTS seam exists); the
    downloadable proof-bundle assembler; the separate anchor panels; the M-UI exit gate.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/source). §5 of
    the certificate is its downstream consumer, so OTS gates the last certificate clause too.
- **Last ~10 iterations: ~8 milestone-Verify-advancing / ~2 refactor·polish·tooling.** The arc is
  building the certificate clause-by-clause (badge → DS tokens → webfonts → `/` grid → log browser →
  dossier → frozen Exhibit → record list → single record → ISCC-IDv1 decoder → Hub-List resolver →
  cert §1 → §2 → §3 → §4 → §6). Each clause verified + mutation-proven; §3 took three cycles to close
  a real trust-root defect the automated gate could not catch (green-but-wrong proof render). No drift
  — the loop is converging on real Verify criteria, and the gate (review + Codex) correctly holds
  clauses at NEEDS_WORK until the crypto is verified. The remaining work (§5 / proof bundle / OTS /
  WASM) is the genuinely large, blocked tail, not avoidable polish.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `3ef0dd7..HEAD` diff touched ONLY `internal/certificate`
(handler.go + cert.html + handler_test.go) and `.claude/*` context/docs — **no M1 source touched.** All
M1 Verify criteria remain satisfied: `origin`/`vkey` golden; all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival;
coverage tracked; structured logs; `/metrics` served over HTTP.
- **Test totals at HEAD**: **319 `func Test`** across **59** `_test.go` files (up from 317 — the two new
  §6 tests). Package count **19 internal + 2 cmd = 21**.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **19 internal packages** —
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
**Status**: **in progress — §6 RECORD HISTORY landed (PASS_WITH_NOTES); §5 + proof bundle + visual-exit
gate remain.** §1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY, and §6 RECORD HISTORY are
all sound and verified. Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen
Exhibit, record list, single-record page, the ISCC-IDv1 decoder, and the Hub-List resolver are all
built and verified.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` browser; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /<domain>/log/record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode`
  (`GOOS=js GOARCH=wasm`-buildable); the fail-closed `(realm, hub_id) → domain` Hub-List resolver.
- **§1 SUBJECT (verified):** `internal/certificate.Handler` decodes the id → `hubList.Resolve` →
  `followedHub`/`ListHubs` → `SeqsForISCCID`, with the accepted-tree cap (`seqs[0] < LastSize`) and the
  canonicalized `"ISCC:" + TrimPrefix` lookup key.
- **§2 CHECKPOINT (verified):** for a certifiable id, `buildData` reads the accepted root via
  `CheckpointAt(LastSize)`, sets `HasClause2` + `CheckpointSize`/`CheckpointRoot` (base64-Std);
  `cert.html` renders §2 only when `HasClause2`. Mutation-proven non-vacuous.
- **§3 INCLUSION PROOF (verified — critical CLOSED at `96f6ed9`):** `handler.go` builds the RFC-6962
  proof of `data.Position` against `hub.LastSize` via `logclient.InclusionProofFromTiles` over a
  `SQLiteFetcher`, reads the subject leaf's raw bytes from the mirrored entry bundle, and gates
  `HasClause3 = true` on `proof.VerifyInclusion(...) == nil` against the §2 accepted root. Fails closed
  against ANY tile↔root divergence. Mutation-proven by `TestCertificateInclusionProofContradictory`.
- **§4 SIGNING KEY (verified — PASS_WITH_NOTES at `3ef0dd7`):** inside the `HasClause2` guard,
  `handler.go` derives the key id from the §2 checkpoint's OWN raw signature line
  (`logclient.KeyIDFromCheckpoint`, grounded in the `0x40b74463` oracle pin), looks it up via
  `store.LookupHubKey(hubID, keyID)`, and sets `SigningKeyDID`/`SigningKeyID`/`SigningKeyMultibase`
  (+ conditional `SigningKeyRevoked`) with `HasClause4 = true` only on a cache HIT. Fails closed (no §4,
  no 500, no fabricated key) on a malformed sig line or a cache miss. Mutation-proven by
  `TestCertificateSigningKeyUncached`; oracle gate green.
- **§6 RECORD HISTORY (verified — PASS_WITH_NOTES at `427a09e`):** renders the full one-to-many list of
  accepted-tree seqs the hub indexed under the subject id (declaration + any later deletion), each
  labelled by its verbatim `note.$schema`, with the deletion note shown when any row is a deletion. A
  pure store read (`seqs` from `SeqsForISCCID`, one `RecordAt` per row), accepted-tree-capped
  (`seq >= LastSize` dropped, matching §1), rendering unconditionally for a certifiable id
  (`HasClause6 = true`, handler.go:613). Mutation-proven by `TestCertificateRecordHistory` +
  `TestCertificateRecordHistoryDeclarationOnly` (both new this cycle); no crypto path touched, oracle
  gate run only to prove no regression.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` never set — the
  handler comments confirm it deliberately stays `false`; BLOCKED on a non-existent OTS/anchor store
  seam — there is no `anchor`/`ots` store method); the **downloadable proof-bundle assembler**
  `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` (re-engages the
  oracle/conformance gate — shares the §3 build+verify crypto path + the §4 key read-path); the separate
  **Bitcoin-anchor vs comparison-anchor** panels; and the **mandatory M-UI exit visual-pass + human
  sign-off** (ADR-0012 agent-browser visual verification).
- **Residual notes (filed `normal`, NOT fixed):**
  - `internal/certificate/handler.go` builds `"did:web:" + data.Domain`, mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm; key id still correct. Fold in when handler.go
    (§5) is next touched.
  - `internal/registry` `hubDomain` (registry.go:188) does not check `u.ForceQuery`, so a bare trailing
    `?` slips the guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in source). The `internal/badge`,
`internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves are WASM-shareable
primitives the verifier app will reuse, but the verifier itself does not exist. OTS is the upstream
blocker for certificate §5.

## Quality gates
**Status**: **green at HEAD; no open `critical`.** The §6 advance passed `review` (`427a09e`,
PASS_WITH_NOTES / CONTINUE) and is pushed; CI is **green at `427a09e`** (`gh run list`: latest
concluded run `success`, headSha == HEAD). Tree is clean; HEAD == `origin/develop` == `427a09e`
(nothing unpushed).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`;
  latest run on `develop` concluded `success` at HEAD.
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" mandates **Go 1.26** + the
  **`iscc-lib/packages/go` v0.5.0** codec, but `go.mod` is still `go 1.24.0` with no iscc-lib require.
  Deliberate, sequenced, not-yet-started increment — target and code disagree on the stack until it lands.
- **Open issues: 0 `critical`, 4 `normal`, 5 `low`.** The 4 `normal`: (a) [human] adopt iscc-lib codec +
  bump to Go 1.26 (ADR-0011, foundational); (b) [review/Codex] Hub-List `hubDomain` `ForceQuery`
  fail-open; (c) [review/Codex] §4 `did:web:` + raw domain mis-renders a `host:port` hub's DID; (d)
  [review] §6 RECORD HISTORY omits the per-record `· at` timestamp the mockup shows. The 5 `low` are
  loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone. §5 is blocked on OTS, so the unblocked next step is the
proof bundle.** The next `define-next`/`advance` SHOULD:

1. **Build the downloadable proof-bundle assembler** `{checkpoint, inclusion/consistency proof, record
   bytes, hub key, ots?}` — the largest remaining unblocked M-UI criterion. It re-engages the
   oracle/conformance gate (reuses the §3 build+verify crypto path + the §4 key read-path) and does not
   depend on the missing OTS seam (`ots?` is optional). Fold in the deferred §4 `host:port` DID-encoding
   fix when handler.go is next touched.
2. **OTS / Bitcoin anchoring** — build the anchor store seam + stamp/upgrade loop, which then **unblocks
   certificate §5** and the Bitcoin-anchor panel. §5 cannot be honestly rendered until this exists.
3. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the
   `internal/index` tripwire/parity test) — flagged foundational; confirm mise can provision Go 1.26
   before flipping `go.mod`'s `go` directive.
4. **WASM verifier** remains a v1 milestone (1/1 Verify open).
5. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.
