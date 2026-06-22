<!-- assessed-at: 39d395f1370907fcd192a56836a0a061fcf9cc96 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI logo-masthead `critical` half-landed — the self-hosted ISCC logo now serves at `/_ds/iscc-logo-black.png` and renders on `/` + the hub dossier; the same one-line `<img>` is still missing from the remaining FOUR mastheads (certificate + the three log-browser surfaces)

The front-of-queue human `critical` (ISCC logo masthead) is partially resolved: advance `7e3fe44`
embedded a pre-downscaled grayscale PNG (`internal/web/iscc-logo-black.png`, 199×76 gray+alpha, 2976 B),
serves it via the existing no-cache+strong-ETag+304 static-asset leaf, and renders it beside the text
mark on `/` (dashboard) and the hub dossier. The latest **`review` verdict is PASS** (loop CONTINUE), CI
is **green at current HEAD `39d395f`**, and the branch is in sync with `origin/develop`. M1/M2/M3 stay
met; the WASM and OTS Verify criteria remain open; the logo `critical` is **narrowed but still open**
(four mastheads remain) and 8 `normal` issues are open — so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: behavioral Verify met; `/` headline-region parity met; ONE open `critical` (logo masthead)
    narrowed to 4 surfaces; the M-UI exit visual-pass + human sign-off still pending.** Met: five-status
    `HubStatusBadge`; DS v2 shared shell (CDN-free, self-hosted fonts); `/` realm-index grid + claim-lookup
    hero + per-row dossier links + instance-identity masthead **+ ISCC logo**; `/<domain>/log/` browser;
    hub dossier **+ ISCC logo**; frozen Exhibit; paginated record list; single-record page; ISCC-IDv1
    decoder; Hub-List resolver; certificate §1–§6 + proof-bundle endpoint + COMPARISON ANCHOR panel.
    **Still open:** the **ISCC logo masthead on the remaining FOUR surfaces** — `internal/certificate/cert.html`,
    `internal/proofserve/{browser,record,records}.html` still render the text-only `.chrome-mark` (grep-confirmed:
    no `iscc-logo-black.png` reference); the serve route exists, so each is now a pure one-line `<img>`
    template edit + a handler-test assertion. Also open: the named-region + `←` back-link parity pass not
    yet carried to dossier / log browser / single record / certificate; `/` sub-region deltas (config-driven
    instance identity, Checkpoint/Anchor columns, recent-declarers footer) filed `normal`; the **mandatory
    M-UI exit visual-pass + human sign-off** (ADR-0012) not executed.
  - **WASM verifier: 1/1 open (carried unchanged).** `.wasm` artifact built reproducibly, served byte-pinned
    at `/_ds/verify.wasm` (`WasmVerifyHash`, `TestWasmVerifyHashPinned`); the **first SSR `<script>` caller
    exists** (certificate tier-2, live-verified). Still open: NO caller on the **dossier**; **no standalone
    `monitor.iscc.codes` Independent Verification app** (Surface C); **no guided split-view alert** on a
    `(size, root)` mismatch. Verify not met.
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots` serve route
    + certificate §5 anchor render). What remains: a root that actually transits to **Bitcoin-confirmed** —
    offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 foundational·plumbing·hardening.** Recent arc:
  `cmd/wasm` entrypoint → `/_ds/wasm_exec.js` loader → build+serve `verify.wasm` → reproducible-build fix →
  `/` realm-index named-region parity → first SSR WASM `<script>` caller on the certificate → **serve the
  self-hosted ISCC logo + render on `/` and the dossier**. **DRIFT WATCH (clear):** increments are genuinely
  closing milestone Verify criteria, not polish — the loop is correctly working a human-filed `critical`
  before resuming WASM. Repointed at: the four remaining logo mastheads, then the dossier WASM caller + the
  standalone verifier app.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `aaaea50..HEAD` diff (confined to
`internal/web/{web.go,web_test.go,iscc-logo-black.png}`, `internal/dashboard/{dashboard.html,handler_test.go}`,
`internal/dossier/{dossier.html,handler_test.go}`, and context/docs). All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze +
alert-once + restart survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}`; 22 internal packages — `badge, certificate,
  config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp,
  ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
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
**Status**: **behaviorally complete; `/` headline-region parity met; the open `critical` (ISCC logo
masthead) is NARROWED — serve route + `/` + dossier landed, FOUR mastheads remain — and the M-UI exit
visual-pass + human sign-off is pending.** All six certificate clauses (§1–§6) + both anchor panels + badge
+ DS shell + `/` index (hero + per-row dossier links + instance-identity masthead + logo) + log browser +
hub dossier (+ logo) + frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder + Hub-List
resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify.
- **Logo masthead (this iteration):** `internal/web/web.go` embeds `iscc-logo-black.png` (`//go:embed`),
  defines `const LogoPath = "/_ds/iscc-logo-black.png"`, and serves it (`contentTypePNG = "image/png"`, a
  string literal so the WASM-green leaf never imports `image/*`) through the sibling `writeAsset`
  no-cache+strong-ETag+304 leaf — verified live + `TestLogoServed` (200 / `image/png` / byte-equal / 304),
  mutation-proven. Rendered as `<img class="chrome-logo" src="/_ds/iscc-logo-black.png">` + a 1px divider
  on `dashboard.html:330` and `dossier.html:315` (asserted by `TestDashboard…`/`TestDossier…`,
  mutation-proven).
- **Open `critical` (human-filed, NARROWED):** the masthead still renders a **text-only `.chrome-mark`**
  with NO logo on the other four SSR surfaces — `internal/certificate/cert.html`,
  `internal/proofserve/{browser,record,records}.html` (grep-confirmed: zero `iscc-logo-black.png`
  references). Fix is now a pure one-line-per-template `<img>` + divider edit (the serve route exists),
  plus a handler-test asserting the `<img>` on at least the certificate surface.
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass has not been carried
  to the remaining SSR surfaces (dossier / log browser / single record / certificate lack the full
  masthead instance-identity block + back-link chain); `/` sub-region deltas (config-driven instance
  identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the mandatory
  **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest to §2's
  root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a `host:port` DID);
  `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN: first SSR `<script>` caller landed on the certificate (live-verified);
no dossier caller / no standalone `monitor.iscc.codes` app / no split-view alert yet. OTS — both
observable HTTP halves landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither
milestone's source was touched this iteration.
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the pure
  `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`. `cert.html`
  embeds a `<script id="tier2-data" type="application/json">` proof island, loads `/_ds/wasm_exec.js` +
  `/_ds/verify.wasm`, and replaces `id="tier2-result"` with the WASM verdict — only when §3 passed.
  **Still 1/1 open on the milestone Verify:** no dossier caller, no Surface-C Independent Verification app,
  no in-browser identical-verdict parity surface, no guided split-view alert. **Carried `normal` defect
  (NOT fixed):** `safeIndex` is a pure `float64→(uint64,string)` fn trapped in the tagged `main.go`, so its
  NaN/fractional/negative/range branches have NO executable test — move it into the untagged
  `verifyadapter` and table-test it.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the calendar
  transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer not yet
  built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation. Still
  1/1 open. **Open `normal` defect (NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the symmetric
  guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS, and CI is green at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (25 packages `ok`; `gofmt -l .` excl. `cauldron/` clean);
  `mise run build:wasm` reproduces `verify.wasm` byte-identical to `WasmVerifyHash`.
- **Latest `review` verdict: PASS (loop CONTINUE)** at HEAD `39d395f`. The logo serve route + the two
  rendered mastheads were independently verified live + mutation-proven, ADR-0012 visual pass confirmed the
  logo renders on both surfaces, and Codex second opinion was clean.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`,
  in sync with `origin/develop`. **Latest CI run is `success` at current HEAD `39d395f`** (run
  27935154701).
- **Open issues: 1 `critical`, 8 `normal`, 10 `low`.** DONE requires 0 critical AND 0 normal, so the loop
  stays CONTINUE. The `critical` is the human-filed ISCC-logo masthead (narrowed to the remaining 4
  surfaces). The 8 normal span: certificate §5 digest binding, OTS `safeStamp` guard, `hubDomain`
  ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the `safeIndex` test gap, the
  certificate tier-2 no-JS honesty-copy overstatement, and the `/` sub-region parity deltas.

## Next Milestone
**Finish the human-filed `critical` ISCC-logo masthead — it preempts everything.** The serve route + `/` +
dossier already landed; add the identical one-line
`<img class="chrome-logo" src="/_ds/iscc-logo-black.png" alt="ISCC">` + `.chrome-divider` (inside the
`.chrome-brand` flex wrapper, copying the `dashboard.html` block + CSS) to the remaining FOUR mastheads —
`internal/certificate/cert.html` and `internal/proofserve/{browser,record,records}.html` — and extend each
surface's handler test to assert the `<img>` src (at least the certificate surface). Keep `mise run check`
green and re-run the ADR-0012 visual pass; that closes the `critical`.

After the logo fully lands, continue the WASM milestone: wire the same tier-2 caller into the **dossier**,
then the standalone `monitor.iscc.codes` **Independent Verification app** (Surface C — in-browser
identical-verdict parity + the guided split-view alert keyed on the `data-state="failed"` verdict); move
`safeIndex` into `verifyadapter` and table-test it at that touch. Subsequent: carry the named-region + `←`
back-link parity pass across the deferred SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in `safeStamp`,
the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are next edited.
