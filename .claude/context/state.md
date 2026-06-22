<!-- assessed-at: e0cc5a35f3294bf5bf221c1556c4a3fa3435ed07 -->

# Project State

## Status: IN_PROGRESS

## Phase: Certificate §6 `· at` render landed — the per-record verbatim note.timestamp now renders; gate green + pushed.
The §6 store prerequisite (last assessment) is now followed by its render: `HistoryRow.At` carries the
verbatim `RecordAt.NoteTimestamp` (captured under the existing `if found` gate) and `cert.html` renders
`{{.Label}} · seq {{.Seq}}{{if .At}} · {{.At}}{{end}}` conditionally. Latest `review` verdict is **PASS
(loop CONTINUE)**, mutation-proven non-vacuous + visual-pass confirmed + Codex clean; the §6 `normal`
is closed (net normal 5→4). HEAD (`e0cc5a3`) == `origin/develop` (0 ahead/0 behind), CI **success** at
HEAD. DONE not reached: WASM "published" + signature halves and OTS Bitcoin-confirmed stay open, and 4
`normal` issues stand.

Incremental review against assessed-at `f8bae84`. The `f8bae84..HEAD` diff touched exactly ONE
production Go file (`internal/certificate/handler.go` — add `At` field + capture it) plus one template
(`internal/certificate/cert.html` — conditional render) and one test (`handler_test.go`); the rest is
`.claude/context/*` + learnings docs. All M1/M2/M3/M-UI/WASM/OTS production source outside that additive
render is byte-unchanged — those sections carry forward met/open as before. Verified this assessment:
`HistoryRow.At = row.NoteTimestamp` is set only inside `if found`, the §6 marker was STRENGTHENED
(`seq %d` → `seq %d · %s`) not weakened, and the deletion (no-timestamp) row renders no trailing `· `.
`git status` clean, HEAD pushed.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact (byte-pinned
    `internal/web/verify.wasm` carries the 6-arg id-binding shim; pin `WasmVerifyHash` matches;
    reproducible from `mise run build:wasm` under `go=1.26.4`). **Still OPEN on the milestone Verify**
    ("the verifier artifact … [published at its] published value"): the Pages deploy **does not run** —
    the latest Pages run at HEAD (27953433943) is `failure` at `Configure Pages` (Pages not enabled /
    not set to "GitHub Actions" source), built-but-undeployed, a one-time human repo-Settings step. Also
    still open: **NO dossier tier-2 WASM caller** (no WASM island in `internal/dossier`); and the
    **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies inclusion math + id-binding
    only — no checkpoint-signature / did:web key resolution — a malicious monitor can still render a
    green `verified`; design-first remainder, STOP-candidate).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, digest-bound via `ots.ConfirmedFor`) and BOTH calendar-transport
    guards (`safeUpgrade` + `safeStamp`) are in place. What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~2 milestone-Verify-advancing / ~8 chrome·plumbing·hardening.** Recent arc:
  §5 OTS digest-binding (PASS) → certificate Tier-2 no-JS honesty copy (PASS) → Surface-C readTarget
  URL normalization (PASS) → §6 note.timestamp store layer (PASS_WITH_NOTES) → **§6 `· at` RENDER (this
  iteration, PASS — closes the §6 normal).** **DRIFT WATCH (amber, easing):** no *milestone-level*
  Verify criterion has closed in ~15 increments (the last were the WASM id-binding slices `c032548`/
  `cf94d5b` ~10 iters back), but this iteration legitimately **CLOSED a `normal`** (the §6 render) and
  the prior two iterations were its self-contained store+render slices — a real backlog drain, not
  re-polish. The front-of-queue WASM "published" half stays **human-blocked, not code-blocked** (a
  workflow file cannot self-enable Pages), so the loop correctly pivots off it. The remaining
  code-closable `normal`s are the `/` sub-region store-projection deltas; after that the work is
  design-first (WASM signature half) or human-blocked (Pages, custom domain). Surface that inflection so
  define-next does not re-polish met surfaces — prefer the `/` Checkpoint/Anchor projection (the last
  large code-closable `normal`) over chrome touch-ups.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `f8bae84..HEAD` diff touched no signature / RFC-6962 /
Merkle / `proof` / `didweb` / `logclient.verify` file (the only production edit is the certificate
view-model + template). All M1 Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart survival;
structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`,
  `go 1.26.1` language directive in `go.mod`; build toolchain `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward; no fsck / fetcher / mirror BLOB / follower-ingest path touched
this diff. fsck root-rebuild on every verified non-frozen poll; inclusion cross-check conformance-tested
over the real verified mirror; `inclusion`, `consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 handler source touched. CORS on
every public GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared
`verify.VerifyInclusion` core); `GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All
golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is pending.**
This diff landed the certificate §6 `· at` RENDER (the per-record verbatim `note.timestamp` now shows as
`{{.Label}} · seq {{.Seq}} · {{.At}}`, conditional on a non-empty timestamp), closing the §6 `normal`.
All six certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index + log browser +
hub dossier (with the shared chrome masthead + `← Realm index` back-link) + frozen Exhibit + record list
+ single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the
behavioral HTTP-seam Verify; every SSR masthead carries the shared chrome (self-hosted logo + text mark
+ divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the
  dossier but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still
  lack the full instance-identity block + back-link chain on every surface); `/` sub-region deltas
  (config-driven instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer)
  filed `normal`; the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes:** the §6 `· at` render `normal` is now CLOSED. The remaining M-UI `normal`
  is the `/` realm-index sub-region parity deltas (instance identity + Checkpoint/Anchor columns). The
  §6 humanization (`2026-02-14 18:40 UTC` vs verbatim RFC-3339) is intentional ADR-0008-deferred visual
  polish, not a filed delta.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN (published half human-blocked, signature half design-blocked),
reproducibility critical CLOSED, HEAD is PASS. OTS — §5 render digest-bound + both calendar-transport
guards landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither the WASM core
nor OTS source was touched by this diff; both sections carry forward.
- **WASM (carried):** the id-binding half of the verifier-scope trust gap is closed IN SOURCE and
  ARTIFACT, and the artifact is reproducible from its documented command: the committed
  `internal/web/verify.wasm` carries the 6-arg/id-binding shim (`verifyadapter.RecordCommitsID`), its
  SHA-256 matches `WasmVerifyHash` (`internal/web/web.go`), and `mise run build:wasm` under `go = "1.26.4"`
  deterministically re-emits those bytes (`TestWasmVerifyHashPinned` green). `cmd/wasm/main.go`
  (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; `verifyadapter` exposes `SafeIndex`,
  `VerifyJSON`, `RecordCommitsID`; `cert.html` embeds the tier-2 island; `internal/verifier` is a SINGLE
  STATIC cross-origin artifact; `cmd/verifier-site` is the reproducible build command;
  `.github/workflows/pages.yml` is the PUBLISH workflow; `verifier.Handler` is NOT mounted in
  `cmd/iscc-monitor`. **Still 1/1 OPEN on the milestone Verify** — the deploy is not live (Pages
  `failure` at `Configure Pages` at HEAD, run 27953433943), needs the one-time human repo-Settings step.
  **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a
  malicious cross-origin monitor can render green `verified`; success copy overstates a key check that
  never runs) — design-first remainder / STOP-candidate; the Pages custom-domain / Actions-source
  enablement gap. **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause (digest-bound via
  `ots.ConfirmedFor`), the store layer (`internal/store/ots.go`), the off-path stamp/upgrade loop
  (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots.{Confirmed,
  ConfirmedFor}` sharing one private `classify`), and the calendar transport (`internal/otsclient`, with
  BOTH `safeUpgrade` + `safeStamp` guards) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer
  not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation.
  Still 1/1 open. **Carried `low` defect:** nil-Stamper + empty-OTSBytes row falls through to the
  Upgrader instead of being left untouched (`otsloop.go:144`; docstring-vs-code mismatch on the
  test-only nil path; production always wires a non-nil Stamper).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`e0cc5a3`) == `origin/develop` (run
27953433969); the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate
defect). Latest `review` verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build
  toolchain `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run
  check` green at HEAD (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l .` empty); the §6
  render is mutation-proven (dropping `{{if .At}}…{{end}}` from `cert.html` makes
  `TestCertificateRecordHistory` FAIL on the exact `Declaration · seq 24815 · 2026-02-14T18:40:00Z`
  marker while `TestCertificateRecordHistoryDeclarationOnly` still PASSES). The trust-root oracles are
  unaffected — name-only diff over the signature / RFC-6962 / Merkle / `proof` / `didweb` globs is empty
  (the only production edit is a store-read into an `html/template` text node).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`e0cc5a3`) is
  pushed (0 ahead / 0 behind upstream); CI run 27953433969 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27953433943) at HEAD
  is **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions"
  source); deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 4 normal, 10 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  Tally moved from "0 critical / 5 normal" last assessment to "0 critical / 4 normal" — the §6 `· at`
  render CLOSED its `normal` (the store half had only narrowed it). The 4 normal: the `/` sub-region
  parity deltas, the WASM verifier-scope SIGNATURE-half gap, the Pages custom-domain/enablement gap, and
  the no-on-disk-DB-migration hazard.

## Next Milestone
**CI green and pushed; the §6 render `normal` is closed. The largest remaining code-closable `normal` is
the `/` realm-index store-projection deltas — do that next; the rest of the backlog is design-first or
human-blocked.** In order:
1. **`/` realm-index Checkpoint/Anchor data columns + config-driven instance identity (closes a
   `normal`, code-closable).** Add the per-hub checkpoint-size and OTS-anchor-state fields to
   `ListHubs`/`HubSummary` (a store-projection change, leaf read — no `net/http` dep in store) and render
   the mockup's `Checkpoint` + `Anchor` columns; make the instance-identity masthead env-configurable
   (domain / operator / realm name). This is the last large self-contained, code-closable, normal-CLOSING
   slice — prefer it over re-polishing met surfaces. (A cheap sibling: thread the same
   `RecordRow.NoteTimestamp` into the log-browser record-list `Logged` column in `internal/proofserve` —
   reuses the landed store field, no new store read.)
2. **Design-first pass on the signature half of the verifier-scope gap** (the WASM milestone's actual
   trust bar) — browser did:web resolution + checkpoint-note signature verify, gating `verified` on
   signature + id-binding + inclusion. `review` recommends a **design pass before building**; a good
   STOP-candidate if the design is unclear. Until it lands, do NOT loosen `verifier.html`'s "hub-signed
   root" success copy. And/or wire the tier-2 WASM caller into the **hub dossier** (no WASM island today).
3. **Unblock + verify the Pages deploy** (one-time human repo-Settings step: Settings → Pages → source
   "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages; flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the named-region + `←`
back-link parity pass across the remaining SSR surfaces; the no-on-disk-DB-migration mechanism (a
deliberate design step, the project's first migration framework); the M-UI exit visual-pass + human
sign-off (ADR-0012).
