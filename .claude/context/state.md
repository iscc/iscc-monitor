<!-- assessed-at: aaaea50a56a12c3afdd519a546ec81d16691b967 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone underway — first SSR `<script>` verifier caller LANDED on the certificate (tier-2 "your browser verified"); a human-filed `critical` (ISCC logo masthead) is now the front of the queue

The certificate page (`GET /inclusion/<iscc_id>`) is the first real SSR caller of the in-browser WASM
verifier: under `{{if .HasBundle}}` it embeds the proof in a `<script type="application/json">` data
island and a progressive-enhancement loader pulls `/_ds/wasm_exec.js` + `/_ds/verify.wasm` to render an
honest tier-2 verdict — review verified it live end-to-end (WASM ran headlessly, produced the correct
`verified` verdict matching server §3). The latest **`review` verdict is PASS** (loop CONTINUE), CI is
**green at current HEAD `aaaea50`**, and the branch is in sync with `origin/develop`. M1/M2/M3 stay met;
the WASM and OTS milestone Verify criteria remain open, and a **human-filed `critical` ISCC-logo masthead
issue** plus 8 `normal` issues are open — so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: behavioral Verify met; `/` headline-region parity met; one open `critical` (logo) + the
    M-UI exit visual-pass + human sign-off still pending.** Met: five-status `HubStatusBadge`; DS v2
    shared shell (CDN-free, self-hosted fonts); `/` realm-index grid + claim-lookup hero + per-row
    dossier links + instance-identity masthead; `/<domain>/log/` browser; hub dossier; frozen Exhibit;
    paginated record list; single-record page; ISCC-IDv1 decoder; Hub-List resolver; certificate §1–§6
    + proof-bundle endpoint + COMPARISON ANCHOR panel. **Still open:** the **ISCC logo masthead**
    (human-filed `critical` — asset exists at `.claude/design/assets/iscc-logo-black.png`, NOT yet
    embedded/served at `/_ds/iscc-logo-black.png`; every SSR masthead still renders a text-only mark);
    the named-region + `←` back-link parity pass not yet carried to dossier / log browser / single
    record / certificate; `/` sub-region deltas (config-driven instance identity, Checkpoint/Anchor
    columns, recent-declarers footer) filed `normal`; the **mandatory M-UI exit visual-pass + human
    sign-off** (ADR-0012) not executed.
  - **WASM verifier: 1/1 open (now in active progress).** `.wasm` artifact built reproducibly, served
    byte-pinned at `/_ds/verify.wasm` (`WasmVerifyHash = 7d57ab1b…`, `TestWasmVerifyHashPinned`). The
    **first SSR `<script>` caller now exists** (certificate tier-2, live-verified). Still open: NO caller
    on the **dossier** yet; **no standalone `monitor.iscc.codes` Independent Verification app** (the
    `monitor.iscc.codes` reference in `cert.html` is only a tier-2 link, not Surface C); **no guided
    split-view alert** on a `(size, root)` mismatch. Verify not met.
  - **OTS anchoring: 1/1 open (untouched this iteration).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 foundational·plumbing·hardening.** Recent
  arc: `internal/proof/verify` core → `cmd/wasm` entrypoint → `/_ds/wasm_exec.js` loader → build+serve
  `verify.wasm` → reproducible-build fix → `/` realm-index named-region parity → **first SSR WASM
  `<script>` caller on the certificate + `cmd/wasm` index/size `safeIndex` guard**. **DRIFT WATCH
  (clear):** increments are genuinely closing milestone Verify criteria, not polish — this iteration
  opened the WASM milestone with a real, live-verified in-browser caller. Loop is repointed at the
  human-filed `critical` logo, then the dossier caller + standalone verifier app.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `25c12af..HEAD` diff (confined to
`cmd/wasm/main.go`, `internal/certificate/{cert.html,handler.go,handler_test.go}`, `internal/web/{web.go,
verify.wasm}`, and context/docs). All M1 Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart survival;
structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}`; 22 internal packages — `badge,
  certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, web`.
  Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`. `transparency-dev/merkle` backs `internal/proof/verify`, which
  `cmd/wasm` reuses (`syscall/js` is stdlib).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core);
`GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally complete; `/` headline-region parity met; ONE open `critical` (ISCC logo
masthead) blocks the design-chrome bar, and the M-UI exit visual-pass + human sign-off is pending.** All
six certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index (hero + per-row
dossier links + instance-identity masthead) + log browser + hub dossier + frozen Exhibit + record list +
single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the
behavioral HTTP-seam Verify.
- **Open `critical` (human-filed):** the masthead renders a **text-only `.chrome-mark`** on every SSR
  surface; the grayscale logo asset exists in-repo (`.claude/design/assets/iscc-logo-black.png`) but is
  NOT embedded/served. Fix: copy into `internal/web/`, `go:embed`, serve at `/_ds/iscc-logo-black.png`
  (mirroring the `wasm_exec.js`/woff2 idiom), reference from all six masthead templates. Verified absent:
  no `iscc-logo` reference in `internal/web/web.go` or the masthead templates.
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass has not been
  carried to the remaining SSR surfaces (dossier / log browser / single record / certificate lack the
  masthead instance-identity block + full back-link chain); `/` sub-region deltas (config-driven instance
  identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest to §2's
  root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a `host:port` DID);
  `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN but advancing: first SSR `<script>` caller landed on the certificate
(live-verified); no dossier caller / no standalone `monitor.iscc.codes` app / no split-view alert yet.
OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the pure
  `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`. `cert.html`
  now embeds a `<script id="tier2-data" type="application/json">` proof island, loads `/_ds/wasm_exec.js`
  + `/_ds/verify.wasm`, and replaces `id="tier2-result"` with the WASM verdict — only when §3 passed (the
  `if ok` block carries `RecordB64`, so the browser can never re-verify a proof the server declined).
  `cmd/wasm/main.go:52-83`'s `safeIndex`/`maxSafeInteger` guard closes the `js.Value.Int()` truncation in
  production. **Still 1/1 open on the milestone Verify:** no dossier caller, no Surface-C Independent
  Verification app, no in-browser identical-verdict parity surface, no guided split-view alert. **Carried
  `normal` defect (NOT fixed):** `safeIndex` is a pure `float64→(uint64,string)` fn trapped in the tagged
  `main.go`, so its NaN/fractional/negative/range branches have NO executable test — move it into the
  untagged `verifyadapter` and table-test it.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the calendar
  transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer not yet
  built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation. Still
  1/1 open. **Open `normal` defect (NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS, and CI is green at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (25 packages `ok`; `gofmt -l .` excl. `cauldron/` clean);
  `mise run build:wasm` reproduces `verify.wasm` byte-identical to `WasmVerifyHash`.
- **Latest `review` verdict: PASS (loop CONTINUE)** at HEAD `aaaea50`. The certificate tier-2 WASM caller
  was independently verified live end-to-end (mutation-proven, Codex second opinion triaged: one minor
  confirmed → filed `normal`, one refuted).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`,
  in sync with `origin/develop`. **Latest CI run is `success` at current HEAD `aaaea50`** (run
  27934463243).
- **Open issues: 1 `critical`, 8 `normal`, 9 `low`.** DONE requires 0 critical AND 0 normal, so the loop
  stays CONTINUE. The `critical` is the human-filed ISCC-logo masthead. The 8 normal span: certificate §5
  digest binding, OTS `safeStamp` guard, `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode,
  certificate §6 timestamp, the `safeIndex` test gap, the certificate tier-2 no-JS honesty-copy
  overstatement, and the `/` sub-region parity deltas.

## Next Milestone
**Land the human-filed `critical` ISCC-logo masthead — it preempts everything.** Copy
`.claude/design/assets/iscc-logo-black.png` into `internal/web/`, downscale it at the embed step,
`go:embed` it, serve at `/_ds/iscc-logo-black.png` (`image/png`, sibling-`/_ds/` ETag/304 policy), and
reference it from all six masthead templates next to the existing text mark; assert the `<img>` on `/`
and one other surface; keep `mise run check` green and re-run the ADR-0012 visual pass vs the Realm-Index
mockup.

After the logo lands, continue the WASM milestone: wire the same tier-2 caller into the **dossier**, then
the standalone `monitor.iscc.codes` **Independent Verification app** (Surface C — in-browser
identical-verdict parity + the guided split-view alert keyed on the `data-state="failed"` verdict);
move `safeIndex` into `verifyadapter` and table-test it at that touch. Subsequent: carry the named-region
+ `←` back-link parity pass across the deferred SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are next
edited.
