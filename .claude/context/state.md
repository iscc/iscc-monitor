<!-- assessed-at: 655d00442d19167a8bea95dc6d5e2e76edbe72f4 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI tail + OTS milestone. The OTS path keeps assembling its parts but has not yet reached
its Verify bar. The newest piece is the **OpenTimestamps adapter** (`internal/ots.Confirmed`) — the
first real `nbd-wtf/opentimestamps` dependency and the pure crypto-classify seam that answers "is this
serialized `.ots` root Bitcoin-confirmed?" offline. Still absent: the real `Upgrader` (calendar
submit/upgrade transport), the live `Run` goroutine in `main.go` (no production caller for `OTSTick` or
`ots.Confirmed`), the `.ots` route, and certificate §5 — so no root has actually reached Bitcoin.
Remaining v1 work: those OTS payload pieces (which unblock §5 BITCOIN ANCHOR), the WASM verifier, the
anchor panels, and the mandatory M-UI visual-exit gate.

The OpenTimestamps adapter is the latest landed increment (HEAD `655d004`, a `cid(review)`
PASS_WITH_NOTES commit). The `66fd02a..HEAD` diff touched ONLY: new `internal/ots/` (`ots.go` +
`ots_test.go` + 4 `.ots` testdata fixtures), `go.mod`/`go.sum` (the OTS lib + btcsuite/decred/x-crypto
indirects), and `.claude/*` context. No M1/M2/M3/M-UI production source changed. M1/M2/M3 remain fully
met. **HEAD is in sync with `origin/develop`** (0 ahead / 0 behind). **CI at HEAD is `in_progress`**
(run 27924765333) — last confirmed-green is the prior commit `66fd02a` (run 27924202260 = success); the
HEAD run had not finished at assessment time.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): partially met.** Met: five-status `HubStatusBadge`
    (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/`
    realm-index grid; `/<domain>/log/` browser; hub dossier (`GET /<domain>`, `internal/dossier`);
    frozen Exhibit; paginated record list; single-record page; ISCC-IDv1 decoder
    (`internal/index.Decode`); `(realm, hub_id) -> domain` Hub-List resolver (`internal/registry`);
    certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION PROOF + §4 SIGNING KEY + §6 RECORD
    HISTORY**; the proof-bundle endpoint (`.bundle`) + download link. **Open:** certificate **§5 BITCOIN
    ANCHOR** (`HasClause5` still hard-false — re-verified at `internal/certificate/handler.go:286`); the
    separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI exit visual-pass +
    human sign-off** (ADR-0012 agent-browser).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open — fourth sub-step landed.** Four pieces now exist and are tested: (1) the
    `ots`-table CRUD seam (`internal/store/ots.go`); (2) the daily stamp pass (`stampRoot` in PollHub);
    (3) the upgrade-loop control core (`OTSTick` over the injected `Upgrader` seam + `backoff`); (4) the
    **OpenTimestamps adapter** (`internal/ots.Confirmed`) — first real `nbd-wtf/opentimestamps`
    dependency, the offline crypto-classify (`confirmed?` + height) over bundled `.ots` example vectors,
    which **is** the `ots verify` oracle gate. The milestone Verify ("a stamped root upgrades to
    Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client") still needs the
    **real `Upgrader`** (the calendar-HTTP submit/upgrade transport — `internal/ots` does no network
    I/O), the live `Run` goroutine wired into `main.go` (`OTSTick` and `ots.Confirmed` have **zero
    production callers** — re-verified none in `cmd/`), the `.ots` route, and §5. No root has actually
    been stamped to Bitcoin. Still 1/1 open.
- **Last ~10 iterations: ~4 milestone-Verify-advancing / ~6 foundational·plumbing.** Recent arc: cert §6
  -> proof-bundle endpoint -> #ZgotmplZ link fix -> ADR-0011 stack bump -> OTS store CRUD seam -> OTS
  stamp pass -> OTS upgrade-loop core -> **OTS adapter**. The last **five** increments (stack bump, OTS
  store seam, stamp pass, upgrade-loop core, adapter) are foundational/plumbing rather than direct
  Verify-closures. **The drift watch-line the prior state.md sharpened — "a fifth plumbing step that
  still does not call `OTSTick` from `main.go` with a real calendar client would read as drift" — is now
  exactly at its edge.** Mitigant: the adapter is correctly *not* a fifth pure-plumbing step — it is the
  first real `go.mod` dependency add and where the `ots verify` oracle gate first applies (the
  bundled-fixture golden test). But the line is now hard: the **next** increment must be the real
  `Upgrader` + `main.go` wiring that finally makes a stamped root reach Bitcoin and closes the OTS
  Verify criterion. Another seam-only step that does not call a real calendar from `main.go` would be
  drift.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`66fd02a..HEAD` diff touched ONLY `internal/ots/*` (new), `go.mod`/`go.sum`, and `.claude/*` context;
all M1 sources (`follower`, `store/sqlite.go`, `schema.sql`, the consistency-trigger code) are
byte-unchanged. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation
golden-tested end-to-end with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **20 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, proofserve, registry, store, tiles, tilesserve, web` (one new this
  iteration: `internal/ots`). Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure). **Now wired:** `github.com/nbd-wtf/opentimestamps` v0.4.0 —
  in `go.mod`/`go.sum` and imported by `internal/ots/ots.go` (no other importer; zero production callers
  of `internal/ots`). The library is parse/classify only; no calendar transport is wired.

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry `no-cache` +
strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — carried forward (no M-UI source in the diff).** §1 SUBJECT, §2 CHECKPOINT,
§3 INCLUSION PROOF, §4 SIGNING KEY, §6 RECORD HISTORY all sound. Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, ISCC-IDv1
decoder, Hub-List resolver, proof-bundle endpoint + download link all built and verified.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` hard-false at
  `handler.go:286`, re-verified; §5 will read OTS rows via `OTSForRoot` + classify via `ots.Confirmed`,
  but no real `Upgrader`/wiring exists yet, so no confirmed anchor data exists to render); the separate
  **Bitcoin-anchor vs comparison-anchor** panels; the **mandatory M-UI exit visual-pass + human
  sign-off** (ADR-0012).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — four sub-steps landed (store seam + stamp pass + upgrade-loop core + adapter); WASM
— not started.**
- **OTS:** (1) `internal/store/ots.go` — typed `ots`-table CRUD
  (`RecordOTS`/`OTSForRoot`/`PendingOTS`/`MarkOTSUpgraded`/`MarkOTSAttempted`). (2) The daily stamp pass
  (`stampRoot` in `internal/follower/follower.go`) writes the accepted root as a `pending` row, idempotent
  via `UNIQUE(hub, tree_size, root)`. (3) The upgrade-loop control core — `OTSTick(ctx, st, up, now,
  logger)` in `internal/follower/otsloop.go` over the injected `Upgrader func` seam + capped-exponential
  `backoff`. (4) **The OpenTimestamps adapter** — `internal/ots.Confirmed(otsBytes) (bool, int64, error)`
  wraps `nbd-wtf/opentimestamps` v0.4.0: parses serialized `.ots` bytes and classifies confirmed +
  height, **fail-closed** (parse error / library panic recovered into a wrapped error; no-attestation =
  pending, not error). Pinned in tests to the library's bundled `.ots` example vectors (real Bitcoin
  attestations = external ground truth). `review` confirmed PASS_WITH_NOTES at HEAD; non-vacuous
  (mutations: always-`(false,0,nil)` and drop-height both reproduced + reverted); WASM-purity held (no
  package imports `internal/ots`); the `ots verify` oracle gate is this golden test.
  **Not yet built (the actual Verify-closer):** the **real `Upgrader`** (the calendar-HTTP
  submit/`UpgradeSequence` transport — `internal/ots` does NO network I/O), the live `Run` goroutine in
  `cmd/iscc-monitor/main.go` (`OTSTick` AND `ots.Confirmed` have **zero production callers** today —
  re-verified not referenced in `cmd/`), the `.ots` HTTP route (re-verified absent), and certificate §5.
  The milestone Verify is still 1/1 open: no root has actually been anchored to Bitcoin.
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **GREEN at the prior commit; HEAD run in flight.** HEAD (`655d004`) is itself the
`cid(review)` PASS_WITH_NOTES commit; the branch is in sync with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (all 22 packages `ok` incl. `internal/ots`; `gofmt -l .`
  clean; `go mod verify`/`go mod tidy -diff` clean) at HEAD.
- The latest **`review` PASS_WITH_NOTES verdict is at HEAD `655d004`** (OpenTimestamps adapter).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `655d004` = `in_progress`** (run
  27924765333) at assessment time; last confirmed `success` is the prior commit `66fd02a` (run
  27924202260). No CI failure observed.
- **Open issues: 0 `critical`, 4 `normal`, 6 `low`.** The 4 `normal`: (a) NEW this iteration —
  `ots.Confirmed` wraps an oversized `uint64` Bitcoin height to a negative `int64` while reporting
  confirmed (`internal/ots/ots.go:67`, Codex P2 reviewer-confirmed; not exploitable — zero importers; fix
  before the `Upgrader` persists `BTCHeight`); (b) Hub-List `hubDomain` `ForceQuery` fail-open; (c) §4
  AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID (2 surfaces); (d) §6 RECORD
  HISTORY omits the per-record `· at` timestamp. The 6 `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate green at prior commit, HEAD CI in flight — no
CI hygiene needed unless the HEAD run fails.** Per the roadmap, the immediate next goal is the OTS
payload that actually reaches the Verify bar — and the drift watch-line now requires it:

1. **Real `Upgrader` + live wiring** — make `stampRoot` submit to a calendar (`opentimestamps.Stamp`,
   persisting the initial sequence bytes into `OTSRecord.OTSBytes`/`CalendarURLs`); implement
   `follower.Upgrader` as a closure that calls `opentimestamps.UpgradeSequence` then `ots.Confirmed`,
   returning `UpgradeResult{Confirmed, OTSBytes, BTCHeight}`; wire the live `Run` goroutine into
   `cmd/iscc-monitor/main.go` (own ticker, `defer Stop()`, log-and-continue, OFF the poll path — keep
   "OTS never blocks the follower"). This is what gives `OTSTick`/`ots.Confirmed` their first production
   caller. **Fold in the `> math.MaxInt64` height fail-closed guard** (the new `normal` issue) when this
   step touches the adapter path, since it is what first persists `BTCHeight`. Then the `.ots` HTTP
   route (reads `OTSForRoot`) and certificate **§5 BITCOIN ANCHOR** (`HasClause5`, reading `OTSForRoot`
   + classifying via `ots.Confirmed`) — closing the last open certificate clause + the Bitcoin-anchor
   panel + the OTS milestone Verify. This is the increment that finally closes a Verify criterion after
   five plumbing steps; another seam-only step would read as drift.
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` -> `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
