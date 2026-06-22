<!-- assessed-at: 3b8db9f3ea5423dfadb32c5291a4d2e80dbe2462 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI tail + OTS milestone. The OTS path now has its first production wiring: PollHub stamps each distinct accepted root as a `pending` OTS row (`stampRoot` -> `store.RecordOTS`). Remaining v1 work: the OTS upgrade loop + `.ots` route (which unblock certificate §5 BITCOIN ANCHOR), the WASM verifier, the anchor panels, and the mandatory M-UI visual-exit gate.

The OTS daily stamp pass is the active increment (HEAD `3b8db9f`, a `cid(advance)` commit). It adds
`stampRoot(ctx, st, hubID, info, observedAt)` to `internal/follower`, called on the verified
non-violation advance path between `fsckMirror` and `recordVerdict`, writing the accepted root as a
`pending` `ots` row via the existing `RecordOTS` seam — a local SQLite insert with no calendar/Bitcoin
I/O (never blocks the follower), idempotent via `UNIQUE(hub, tree_size, root)`. No new dependency, no
HTTP route, no `opentimestamps` code: certificate §5 stays deliberately unrendered. M1/M2/M3 remain
fully met. **HEAD is not yet reviewed** (no `cid(review)` for the stamp pass) and the branch is **3
commits ahead of `origin/develop`** — so the last confirmed-green gate (review PASS + CI success) is at
`381764d`, the prior OTS store-seam commit, not at HEAD.

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
    ANCHOR** (`HasClause5` still hard-false — verified at `internal/certificate/handler.go:286`); the
    separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI exit visual-pass +
    human sign-off** (ADR-0012 agent-browser).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open — second sub-step landed.** The `ots`-table CRUD seam (`internal/store/ots.go`)
    plus now the **daily stamp pass** (`stampRoot` in PollHub) exist and are tested, but the milestone's
    Verify ("a stamped root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard
    `ots` client") needs the **upgrade loop + `.ots` route + the `nbd-wtf/opentimestamps` dependency**,
    none of which exist (no `opentimestamps` import in any source or in `go.mod`/`go.sum` — the one grep
    hit is a literal calendar-URL string in `ots_test.go`). Still 1/1 open: stamping is necessary plumbing,
    not the closed criterion. §5 of the certificate is its downstream consumer.
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 foundational·plumbing.** Recent arc: cert §3
  fail-close -> §6 -> proof-bundle endpoint -> #ZgotmplZ link fix -> ADR-0011 stack bump -> OTS store CRUD
  seam -> **OTS stamp pass**. The last three increments (stack bump, OTS store seam, OTS stamp pass) are
  foundational/plumbing rather than direct Verify-closures, but all are target-mandated prerequisites on
  the only path to the still-open OTS milestone + certificate §5 — not avoidable polish. No drift; the
  remaining tail (OTS upgrade/route -> §5, WASM, anchor panels, visual exit) is the genuine large
  remainder. Watch: a fourth consecutive non-closing increment would start to read as drift if it isn't
  the upgrade loop that actually reaches toward the OTS Verify bar.

## M1 — Read-only Monitor
**Status**: **met** — carried forward, re-verified at the seam touched this iteration. The
`381764d..HEAD` diff touched ONLY `internal/follower/follower.go` (+`_test.go`) and `.claude/*` context;
`go.mod`/`go.sum`/`internal/store/*`/`schema.sql` byte-unchanged. The follower change adds the `stampRoot`
call on the verified-advance path and zero-`ots`-row assertions on the fork/shrink/unverified/frozen-clean
paths — it does not alter the three-trigger consistency logic, the freeze/alert path, or coverage
tracking. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation
golden-tested end-to-end with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **19 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, proofserve, registry, store, tiles, tilesserve, web` (no new package this
  iteration; the prior state.md's "20" was a miscount — `ls -d internal/*/` returns 19). Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure; production `internal/index` leaf stays iscc-lib-free under the
  ADR-0011 carve-out). **Not wired:** `nbd-wtf/opentimestamps` (not in `go.mod`/`go.sum`; the stamp pass
  writes opaque `pending` rows only, no calendar/Bitcoin code).

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
  `handler.go:286`; the stamp pass now writes the `pending` OTS rows §5 will eventually read via
  `OTSForRoot`, but no anchor data path: upgrade loop + `.ots` route + the `opentimestamps` dependency
  all absent); the separate **Bitcoin-anchor vs comparison-anchor** panels; the **mandatory M-UI exit
  visual-pass + human sign-off** (ADR-0012).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — store seam + stamp pass landed; WASM — not started.**
- **OTS:** two sub-steps now exist. (1) `internal/store/ots.go` — typed `ots`-table CRUD
  (`RecordOTS`/`OTSForRoot`/`PendingOTS`/`MarkOTSUpgraded`), reviewed PASS at `381764d`. (2) **The daily
  stamp pass** — `stampRoot` in `internal/follower/follower.go` writes the accepted root as a `pending`
  row on the verified non-violation advance path (after `fsckMirror`, before `recordVerdict`), idempotent
  via `RecordOTS`'s `UNIQUE(hub, tree_size, root)`. Tests (`follower_test.go`): one `ots` row after the
  first verified poll, still one after a second (dedupe), `pending` with empty `OTSBytes`, and zero rows
  on the fork/shrink/unverified/frozen-clean paths. The advance reports `mise run check` green and the
  stamp neutered-to-no-op mutation failing — **but this is the advance's self-report; `review` has not
  yet confirmed it (HEAD is an unreviewed `cid(advance)`).** **Not yet built:** the background upgrade
  loop (`PendingOTS` -> `opentimestamps` calendar -> `MarkOTSUpgraded`, exercising `Attempts`/`NextRetry`),
  the `.ots` HTTP route, certificate §5. The milestone Verify is still 1/1 open.
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **NOT YET CONFIRMED AT HEAD — last confirmed-green at `381764d`, not HEAD.** HEAD (`3b8db9f`)
is an unreviewed `cid(advance)`; the branch is **3 commits ahead of `origin/develop`**, so CI has not run
on HEAD.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. The
  advance self-reports it green (all 21 packages `ok`), but `review` owns gate confirmation and has not
  run on this commit.
- The latest **`review` PASS verdict is `381764d`** (the OTS store seam), NOT HEAD. The stamp pass at
  `3b8db9f` is unreviewed.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **Latest CI run is at `381764d` = `success`** (run
  27922172138); `2bf66a8` also `success`. **No CI run exists for HEAD `3b8db9f`** — it is unpushed.
- **Open issues: 0 `critical`, 3 `normal`, 6 `low`.** The 3 `normal`: (a) Hub-List `hubDomain`
  `ForceQuery` fail-open; (b) §4 AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID
  (2 surfaces); (c) §6 RECORD HISTORY omits the per-record `· at` timestamp. The 6 `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones.** Immediate gate hygiene: the stamp pass at HEAD is
unreviewed and unpushed — `review` must confirm `mise run check` green (the advance self-reported it) and
push so CI runs on HEAD before more feature work stacks on an unverified commit. Then, per the roadmap:

1. **OTS upgrade loop** — read `PendingOTS`, pull in `nbd-wtf/opentimestamps` + calendar HTTP (the first
   real `go.mod`/`go.sum` change for this milestone), mark via `MarkOTSUpgraded` once Bitcoin-confirmed,
   exercising the `Attempts`/`NextRetry` retry policy; runs in its own goroutine off the poll path (OTS
   never blocks the follower). Then the `.ots` HTTP route and certificate **§5 BITCOIN ANCHOR**
   (`HasClause5`, reading `OTSForRoot`) — closing the last open certificate clause + the Bitcoin-anchor
   panel. This is the increment that actually reaches the OTS Verify bar.
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` -> `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
