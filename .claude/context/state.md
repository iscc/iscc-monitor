<!-- assessed-at: f8bae84cee3a8c5d54ae232c14c233d9d8d272ab -->

# Project State

## Status: IN_PROGRESS

## Phase: §6 store prerequisite landed — note.timestamp now flows through the iscc_index projection; gate green + pushed.
The optional per-record `note.timestamp` (verbatim RFC-3339, both note types) is now carried
end-to-end through the schema-agnostic `iscc_index` projection: the pure fold
(`logclient.Projection.Timestamp`), a nullable `note_timestamp TEXT` column, the store write/read
structs (`ProjectionRecord`/`RecordRow.NoteTimestamp`), and the follower copy site
(`ingest.go:114`). This is the STORE half of the certificate §6 `· at` render — the render itself is
still unbuilt. Latest `review` verdict is **PASS_WITH_NOTES (loop CONTINUE)**; HEAD (`f8bae84`) ==
`origin/develop` (0 ahead), CI **success** at HEAD. DONE not reached: WASM "published" + signature
halves and OTS Bitcoin-confirmed stay open, and 5 `normal` issues stand.

Incremental review against assessed-at `c0d8273`. The `c0d8273..HEAD` diff touched exactly THREE
production files — `internal/logclient/projection.go` (add `Timestamp` field to the fold),
`internal/store/iscc_index.go` (add `NoteTimestamp` write/read + nullable column), and
`internal/follower/ingest.go` (one copy-site line) — plus `internal/store/schema.sql` (one nullable
column), 2 test files, and `.claude/context/*` + learnings docs. All M1/M2/M3/M-UI/WASM/OTS
production source outside that additive projection field is byte-unchanged — those sections carry
forward met/open as before. Verified this assessment: `Projection.Timestamp = env.Note.Timestamp`
(WASM-pure — no `time` import), `note_timestamp` bound via `nullStringOrNil` (absent → SQL NULL),
read back through `RecordAt`/`ListRecords` ("" for NULL). `git status` clean, HEAD pushed.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact (byte-pinned
    `internal/web/verify.wasm` carries the 6-arg id-binding shim; pin `WasmVerifyHash` matches;
    reproducible from `mise run build:wasm` under `go=1.26.4`). **Still OPEN on the milestone Verify**
    ("the verifier artifact … [published at its] published value"): the Pages deploy **does not run** —
    the Pages run at HEAD (27952454368) `failure` at `Configure Pages` (Pages not enabled / not set to
    "GitHub Actions" source); built-but-undeployed, a one-time human repo-Settings step. Also still
    open: **NO dossier tier-2 WASM caller** (no WASM island in `internal/dossier`); and the
    **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies inclusion math + id-binding
    only — no checkpoint-signature / did:web key resolution — a malicious monitor can still render a
    green `verified`; design-first remainder, STOP-candidate).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, digest-bound via `ots.ConfirmedFor`) and BOTH calendar-transport
    guards (`safeUpgrade` + `safeStamp`) are in place. What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~2 milestone-Verify-advancing / ~8 chrome·plumbing·hardening.** Recent arc:
  §4/bundle `host:port` did:web encode → §5 OTS digest-binding (PASS) → certificate Tier-2 no-JS
  honesty copy (PASS) → Surface-C readTarget URL normalization (PASS) → **note.timestamp projection
  (this iteration, PASS_WITH_NOTES — store half of §6).** **DRIFT WATCH (amber):** no milestone Verify
  criterion has closed for ~14 increments. The front-of-queue WASM "published" half is **human-blocked,
  not code-blocked** (a workflow file cannot self-enable Pages), so the loop correctly pivoted off it to
  drain code-closable backlog. The §6 note.timestamp slice was the store prerequisite for the §6 render
  `normal` — a legitimate self-contained, code-closable schema slice (not re-polish), but it
  **narrowed** rather than **closed** the §6 normal (render still pending) AND **net-grew** the normal
  count 4→5 by articulating a pre-existing no-migration hazard. The immediate next pick is now the
  short, fully-unblocked §6 RENDER (thread `RecordAt.NoteTimestamp` → `HistoryRow.At` → `cert.html`),
  which actually CLOSES a normal. After that the remaining work is design-first (WASM signature half) or
  human-blocked (Pages, custom domain) — surface that inflection so define-next does not re-polish met
  surfaces.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `c0d8273..HEAD` diff touched no signature / RFC-6962 /
Merkle / `proof` / `didweb` / `logclient.verify` file (review confirmed by name-only globs; the
logclient edit is a pure JSON-fold field, the store edit a nullable column). All M1 Verify criteria
remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with
freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`,
  `go 1.26.1` language directive in `go.mod`; build toolchain `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. The follower-ingest edit was a single copy-site line
(`NoteTimestamp: p.Timestamp`), additive to the projection; no fsck / fetcher / mirror BLOB path
touched. fsck root-rebuild on every verified non-frozen poll; inclusion cross-check
conformance-tested over the real verified mirror; `inclusion`, `consistency`, `entries` all served
from the local mirror.

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
No M-UI SSR surface render changed this diff — the §6 work landed only the STORE prerequisite
(`note_timestamp` projection), NOT the certificate §6 render. All six certificate clauses (§1–§6) +
both anchor panels + badge + DS shell + `/` index + log browser + hub dossier (with the shared chrome
masthead + `← Realm index` back-link) + frozen Exhibit + record list + single-record page + ISCC-IDv1
decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify;
every SSR masthead carries the shared chrome (self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the
  dossier but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still
  lack the full instance-identity block + back-link chain on every surface); `/` sub-region deltas
  (config-driven instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer)
  filed `normal`; the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes:** certificate §6 rows still omit the per-record `· at` timestamp at the
  RENDER level — but the STORE prerequisite is now LANDED (this iteration), so the §6 normal is narrowed
  to **render-only**: thread `RecordAt`'s `NoteTimestamp` → a `HistoryRow.At` and render `seq N · <at>`
  in `cert.html`. The `/` sub-region parity deltas remain.

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
  `failure` at `Configure Pages` at HEAD, run 27952454368), needs the one-time human repo-Settings step.
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
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`f8bae84`) == `origin/develop` (run
27952454351); the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate
defect). Latest `review` verdict is PASS_WITH_NOTES (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build
  toolchain `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run
  check` green at HEAD (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l .` empty); both the
  fold field and the nullable-column round-trip are mutation-proven (Mutation 1: drop `note_timestamp`
  from the `DO UPDATE SET` → `TestRecordProjectionsIdempotent` fails; Mutation 2: pin
  `Projection.Timestamp` to a constant → `TestBundleProjections` fails on both present and absent
  leaves), and WASM purity is preserved (`GOOS=js GOARCH=wasm go build ./internal/logclient` clean, no
  `time` import). The trust-root oracles are unaffected — name-only diff over the signature / RFC-6962 /
  Merkle / `proof` / `didweb` globs is empty.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`f8bae84`) is
  pushed (0 ahead of upstream); CI run 27952454351 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27952454368) at HEAD
  is **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions"
  source); deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 5 normal, 10 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  Tally moved from "0 critical / 4 normal" last assessment to "0 critical / 5 normal" — the §6 store
  half did NOT close a normal (it narrowed the existing §6 issue to render-only), and a new no-migration
  `normal` was articulated (net 4→5). The 5 normal: certificate §6 `· at` RENDER (now render-only,
  store landed), the `/` sub-region parity deltas, the WASM verifier-scope SIGNATURE-half gap, the Pages
  custom-domain/enablement gap, and the no-on-disk-DB-migration hazard.

## Next Milestone
**CI green and pushed; the §6 store prerequisite just landed. The cheapest fully-unblocked, normal-
CLOSING pick is now the §6 RENDER — do that next; it converts a narrowed normal into a closed one.**
In order:
1. **Certificate §6 `· at` RENDER (closes a normal, fully unblocked).** Thread `RecordAt`'s
   `NoteTimestamp` into a `HistoryRow.At` field in `internal/certificate/handler.go`'s §6 loop and
   render `seq N · <at>` in `cert.html` (`.dc.html:68`), choosing the format/relativize policy for the
   verbatim RFC-3339 string (ADR-0008 leaves the policy to the renderer). The log-browser record-list
   `Logged` column can reuse the same `RecordRow.NoteTimestamp`. This is the immediate self-contained,
   code-closable, normal-CLOSING slice — prefer it over re-polishing met surfaces.
2. **Design-first pass on the signature half of the verifier-scope gap** (the WASM milestone's actual
   trust bar) — browser did:web resolution + checkpoint-note signature verify, gating `verified` on
   signature + id-binding + inclusion. `review` recommends a **design pass before building**; a good
   STOP-candidate if the design is unclear. Until it lands, do NOT loosen `verifier.html`'s "hub-signed
   root" success copy. And/or wire the tier-2 WASM caller into the **hub dossier** (no WASM island today).
3. **A self-contained store-schema `normal`** — the `/` Checkpoint/Anchor data columns (add fields to
   `ListHubs`/`HubSummary` + render) is a store-projection + follower-ingest change, larger but
   code-closable.
4. **Unblock + verify the Pages deploy** (one-time human repo-Settings step: Settings → Pages → source
   "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages; flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the named-region + `←`
back-link parity pass across the remaining SSR surfaces; the no-on-disk-DB-migration mechanism (a
deliberate design step, the project's first migration framework); the M-UI exit visual-pass + human
sign-off (ADR-0012).
