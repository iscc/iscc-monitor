<!-- assessed-at: 809f17488d026c206980610c74764924384a49e3 -->

# Project State

## Status: IN_PROGRESS

## Phase: Honesty/polish streak while the lone `critical` waits on human M-UI sign-off; the M-API contract-accuracy `normal`s the prior state pointed at as "next code work" are already CLOSED in code but unpruned.
This window (`46e690a..809f174`, 4 commits) made a single copy-honesty edit: the Surface-C verifier
page (`monitor.iscc.codes`) no longer LISTS the un-run "Check the signature against the hub's did:web
key" step and its tier-2 `verified` verdict no longer claims a "hub-signed checkpoint root" — it now
asserts only the RFC-6962 inclusion + id-binding the WASM actually runs. All feature milestones
(M1–M3, M-UI code-halves, M-Deploy, M-API) carry forward unchanged. **Correction to the prior state:**
the two M-API contract-accuracy `normal`s it flagged as the immediate code-closable work are
**already fixed in the served spec** (commit `53ee328`) — see Convergence.

This window: update-state `914cb01` → define-next `e4179ee` (stop the un-run did:web claim) → advance
`4a0c24b` (drop the step + verdict copy) → review `809f174` (PASS / CONTINUE; both mutations
reproduced, visual-pass clean, Codex sandbox-denied). Code touched: `internal/verifier/{verifier.html,
handler_test.go}` (2 files + context). Branch `develop`, HEAD `809f174` level with `origin/develop`;
working tree clean; CI + Pages + Publish all `success` on HEAD.

## Convergence
- **Remaining Verify criteria (per unmet milestone):**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-Deploy: 0 open. M-API: 0 open (4/4 MET — see below).**
  - **M-UI: 1 Verify criterion still REOPENED** (log-browser record-list named-region parity) — `critical`,
    **FULLY code-closed**; only the human M-UI exit sign-off remains. No M-UI named-region code work is
    left on the critical. Other open M-UI items are all `normal`/design- or human-blocked (realm-index
    per-hub-vs-per-checkpoint Anchor honesty).
  - **WASM verifier: 0 of the target.md Verify criteria open** (identical-verdict / hash-match / mismatch-alert
    all MET). The open WASM item is the cross-origin **signature half** (no checkpoint-signature / did:web
    check) — a CLAUDE.md verifiable-cache honesty `normal`, **design-blocked**, NOT a target.md WASM Verify
    criterion. Its copy-honesty interim half CLOSED this window.
  - **OTS: 1/1 Verify open** — only a real Bitcoin confirmation remains (offline-unprovable).
- **Stale/unpruned issues (must be pruned by review):** the prior state.md + `next.md` steered the loop
  toward "M-API contract-accuracy doc fixes" as the genuine next code work. **That work is already done.**
  The served `openapi.json`/`.yaml` (verified at the seam): `/{domain}/log/verify` params = `Domain` +
  `iscc_id` only (**no phantom `index`**); `/{domain}/log/checkpoint` `200` content = `application/octet-stream`
  (**not `text/plain`**). Fixed by commit `53ee328` ("fix 3 OpenAPI contract-accuracy defects"). But
  issues.md still carries both as open `normal`s (lines 638, 662, with stale line-number bodies) plus the
  M-API umbrella (line 699) — all three are RESOLVED-BUT-UNPRUNED. Net: there are effectively **no
  autonomous, code-closable `normal`s left** that the loop has not already addressed.
- **Last ~10 iterations: ~3 milestone-Verify-closing / ~7 honesty/polish/contract-accuracy.** Recent
  closes: M-API slice 4 contract-accuracy (`53ee328`), dossier §1 unresolvable honesty (`988d48d`),
  verifier copy-honesty (`4a0c24b`, this window). **DRIFT WATCH:** with the critical human-blocked and the
  M-API normals already closed, the loop is at risk of spinning on cosmetic honesty/chrome polish
  (auto-memory: `loop-stalls-on-human-blocked-done`). The remaining genuinely-open `normal`s (WASM
  signature half, realm-index Anchor honesty) are BOTH design-blocked. The honest next move is to **prune
  the resolved issues and surface that the loop is out of autonomous code work pending human/design input.**

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 path (config, registry, follower, didweb, metrics, proof)
touched this window.
- **Packages**: `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal — `badge, certificate,
  config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index, logclient, metrics,
  metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / `iscc_index` path touched this
window. `SQLiteFetcher` / `ProofBuilder` read side unchanged; the M2 mirror/fsck contract holds.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. No proofserve / dashboard functional
contract touched (CORS, verify-for-me JSON, `/` index, `/<domain>/log/`).
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — the `critical` log-browser gap is FULLY code-closed; only the human M-UI
exit sign-off remains.** Not touched this window. The full no-JS chain (`/` → dossier → record list →
single record → cert/back) is traversable forward AND back end-to-end; the record-list browser matches
the Log-Browser mockup's named regions.
- **`critical` (CLOSED in code, committed across `de9ed3c`/`7ea7fef`/`14d6bc9`/`b6eee37`):** forward
  repoint, record-list chrome/breadcrumb/head, single-record back-leg, top+bottom pager parity — all
  reviewer-verified, mutation-proven. **Only remaining gate: human M-UI exit sign-off** (ADR-0012). The
  `define-next` after this should NOT re-attempt this critical — it is human-blocked.
- **Open `normal` (design-blocked):** realm-index per-hub-vs-per-checkpoint Anchor honesty (the per-hub
  "latest-stamped-root" Anchor cell vs the displayed checkpoint — a design question for the M-UI exit;
  certificate §5 is the authoritative per-checkpoint surface).
- **Prunable:** the `/` realm-index hero-footer `normal` (all 4 sub-items CLOSED).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live, byte-pinned); all target.md WASM Verify criteria
MET; the cross-origin signature half is a design-blocked honesty `normal`. OTS — observable halves +
both transport guards landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM (touched this window — copy-honesty only):** `verifier.html` no longer lists the un-run did:web
  step and the `verified` verdict no longer claims a "hub-signed checkpoint root"; mutation-proven by
  `TestVerifierDoesNotClaimSignatureCheck` (grep-confirmed: 0 occurrences of the did:web step string; no
  `hub-signed` substring in `verifier.html`). The id-binding half was already CLOSED in source
  (`22f0420`). **Still open (design-blocked `normal`):** the cross-origin SIGNATURE-half gap — no
  checkpoint-signature / did:web verification in the browser; a forged-but-internally-consistent bundle
  with a matching id still renders `verified`. Closing it needs a design pass (browser did:web resolution
  + note-signature verify), NOT another code attempt.
- **OTS:** `.ots` route, §4 anchor clause, store layer, stamp/upgrade loop, offline classifier, both
  calendar-transport guards wired. 1/1 Verify open (a root reaching Bitcoin-confirmed needs a live
  calendar + real BTC confirmation). Carried `low` defect: nil-Stamper + empty-OTSBytes row falls through
  to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** Carried forward — no M-Deploy source touched. SIGTERM
trap, version-stamped binary + `/version`, production `Dockerfile` + CI `/healthz` smoke, GHCR
`publish.yml` (`:develop` + `:sha-<short>`), canonical `deploy/realm-testnet.txt`, `deploy/OPERATING.md`,
root `README.md`, on-disk migration mechanism + its first real entry (composite-PK) all verified
previously.
- **Carried `low` traps (latent):** out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)`;
  composite-PK rebuild dropped `seq`'s standalone ordering path; `docker/login-action@v3` +
  `build-push-action@v6` still Node-20; `.dockerignore` slashless globs; §Footprint qualitative
  disk-growth; `cmd/verifier-site` non-atomic write; `schemaDeclaration/Deletion` URI triplication;
  masthead-fallback consts 3x; dossier overlay 3x.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET — and the contract-accuracy doc fixes are ALSO already landed** (no M-API
source touched this window; verified at the seam this assessment).
- **Slice 1 (serve):** OpenAPI 3.1 served byte-verbatim at `GET /openapi.json` + `/openapi.yaml`.
- **Slice 2 (drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts path↔mux alignment.
- **Slice 3 (`/docs` + Stoplight Elements):** `/docs` mounts `<elements-api>` against same-origin
  byte-pinned `/_ds/elements.min.{js,css}`; no CDN body.
- **Slice 4 (contract accuracy — VERIFIED CLOSED, `53ee328`):** the served `openapi.json`/`.yaml`
  `/{domain}/log/verify` operation declares NO `index` param (only `Domain` + `iscc_id`), and
  `/{domain}/log/checkpoint` `200` content type is `application/octet-stream`. The two `normal`
  contract-accuracy issues (issues.md:638, 662) and the umbrella (issues.md:699) are **RESOLVED but
  UNPRUNED** — `review` should delete them. Remaining residuals are `low` only (the omitted `/healthz`
  503; the Elements-mermaid-from-unpkg substring ban).

## Quality gates
**Status**: **GREEN on HEAD (`809f174`): CI + Pages + Publish all `success`.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable
  (NOT run by this role). `gofmt -l .` reported empty by this assessment. The code change this window is a
  pure SSR-template + golden-test edit (no crypto/Merkle/did:web/proof path) — oracle/conformance gate N/A.
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On HEAD `809f174` all three
  `success` (`gh run list --branch develop`).
- **Latest `review` verdict: PASS / CONTINUE** (commit `809f174`, recorded in handoff.md) — `mise run
  check` green across 30 packages, `gofmt` empty, `TestVerifierDoesNotClaimSignatureCheck` mutation-proven
  both ways, live visual pass clean. Codex second opinion unavailable (sandbox-denied — treated as a note,
  not NEEDS_WORK).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical (human-blocked), ~4 normal (most resolved-but-unpruned or design-blocked),
  ~22 low.** The 1 critical (dossier→log-browser navigation + record-list parity) is FULLY code-closed —
  open only for the human M-UI exit sign-off. Of the normals: the 2 M-API contract-accuracy + the M-API
  umbrella + the `/` realm-index hero-footer are RESOLVED-BUT-UNPRUNED; the genuinely-open ones (WASM
  signature half, realm-index Anchor honesty) are BOTH design-blocked. DONE requires 0 critical AND 0
  normal — so DONE stays blocked, but the only blockers are human-sign-off / design / unpruned bookkeeping.

## Next Milestone
**The loop is out of autonomous, code-closable work.** The lone `critical` is human-blocked, the M-API
contract-accuracy `normal`s are already fixed, and the remaining open `normal`s are design-blocked. Steer
`define-next`/`review` accordingly:
1. **Prune the resolved-but-unpruned issues** (`review`'s job): the two M-API contract-accuracy `normal`s
   (issues.md:638, 662 — verified fixed in the served spec), the M-API umbrella (issues.md:699), and the
   `/` realm-index hero-footer `normal` (all 4 sub-items closed). This is bookkeeping, not feature work.
2. **Request the human M-UI exit visual-pass + sign-off** (ADR-0012) — the only remaining gate on the lone
   `critical`; every code-closable half has landed and is reviewer-verified.
3. **Surface the design-blocked `normal`s for a human/design pass — do NOT code-attempt them blind:** the
   WASM cross-origin signature half (browser did:web resolution + note-signature verify) and the
   realm-index per-hub-vs-per-checkpoint Anchor honesty semantics. The OTS Bitcoin-confirmed half remains
   offline-unprovable.
4. **Drift guard:** with no autonomous code work left, the loop must not spin on cosmetic chrome/honesty
   polish (auto-memory `loop-stalls-on-human-blocked-done`). Prefer pruning + flagging the human/design
   gate over manufacturing a `low` refactor.
