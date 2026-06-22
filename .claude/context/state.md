<!-- assessed-at: 381764d2c884f06ef0b7250360d1bd6b8bada3c5 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI tail + OTS milestone. The OTS store CRUD seam (`internal/store/ots.go`) landed and was reviewed PASS, the first concrete step toward Bitcoin anchoring. Remaining v1 work: the OTS stamp pass + upgrade loop + `.ots` route (which then unblocks certificate §5 BITCOIN ANCHOR), the WASM verifier, the anchor panels, and the mandatory M-UI visual-exit gate.

The OTS store seam is the active increment: `internal/store/ots.go` adds typed CRUD
(`RecordOTS`/`OTSForRoot`/`PendingOTS`/`MarkOTSUpgraded`) over the already-present `ots` schema table,
porting the checkpoint family's exact idioms. It is store-only — not yet wired into the follower, no
HTTP route, no `opentimestamps` dependency — so certificate §5 stays deliberately unrendered. M1/M2/M3
remain fully met; the ADR-0011 Go 1.26 + iscc-lib v0.5.0 stack increment is closed and CI-confirmed.

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
    download LINK** (canonical path-rooted `BundleHref`, both id forms). **Open:** certificate **§5
    BITCOIN ANCHOR** (`HasClause5` deliberately false until the OTS path is wired end-to-end); the
    separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI exit visual-pass +
    human sign-off** (ADR-0012 agent-browser).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file).
  - **OTS anchoring: 1/1 open — first sub-step landed.** The `ots`-table CRUD store seam exists and is
    reviewed PASS, but the milestone's Verify ("a stamped root upgrades to Bitcoin-confirmed and the
    served `.ots` verifies with the standard `ots` client") needs the stamp pass + upgrade loop + `.ots`
    route + the `nbd-wtf/opentimestamps` dependency, none of which exist yet. Still 1/1 open; the seam is
    necessary plumbing, not a closed criterion. §5 of the certificate is its downstream consumer.
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 refactor·foundational·plumbing.** Recent
  arc: cert §3 fail-close → §6 → proof-bundle endpoint → #ZgotmplZ link fix → ADR-0011 stack bump →
  **OTS store CRUD seam**. The last two increments (stack bump, OTS store seam) are foundational/plumbing
  rather than direct Verify-closures, but both are target-mandated prerequisites (the stack gap was a
  filed `normal` issue; the OTS table is the only path to §5 + the OTS milestone), not avoidable polish.
  No drift — the remaining tail (OTS stamp/upgrade/route → §5, WASM, anchor panels, visual exit) is the
  genuinely large, partly-blocked remainder.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `2bf66a8..HEAD` diff touched ONLY `internal/store/ots.go` +
`internal/store/ots_test.go` and `.claude/*` context — **no M1 source touched** (`schema.sql`, `go.mod`,
`go.sum` byte-unchanged, verified empty diff). All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden; all three triggers (fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once
+ restart survival; coverage tracked; structured logs; `/metrics` over HTTP.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **20 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, proofserve, registry, store, tiles, tilesserve, web` (unchanged — no new package
  this iteration). Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`Consistency`+`VerifyInclusion`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (Hub-List parser),
  `github.com/iscc/iscc-lib/packages/go v0.5.0` (test-only in the build closure; production
  `internal/index` leaf stays iscc-lib-free under the ADR-0011 carve-out + tripwire test). **Not
  wired:** `nbd-wtf/opentimestamps` (the OTS store seam stores opaque `ots_bytes` BLOBs only — no
  calendar/Bitcoin code yet).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. Both Verify criteria remain exercised:
fsck root-rebuild on every verified non-frozen poll; inclusion cross-check conformance-tested over the
real verified mirror. All three computed proofs — `inclusion`, `consistency`, `entries` — served from
the local mirror, never re-hitting the hub.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry `no-cache` +
strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — carried forward (no M-UI source in the diff).** §1 SUBJECT, §2 CHECKPOINT,
§3 INCLUSION PROOF, §4 SIGNING KEY, and §6 RECORD HISTORY are all sound. Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, the ISCC-IDv1
decoder, the Hub-List resolver, and the proof-bundle endpoint + download link are all built and verified.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` deliberately false —
  now has its store backing via the OTS seam but no live anchor data path: stamp pass + upgrade loop +
  `.ots` route + the `opentimestamps` dependency are all absent); the separate **Bitcoin-anchor vs
  comparison-anchor** panels; and the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — first sub-step landed (store CRUD seam, reviewed PASS); WASM — not started.**
- **OTS:** `internal/store/ots.go` adds the typed `ots`-table CRUD seam —
  `RecordOTS` (DO-NOTHING dedupe on `(hub, tree_size, root)`), `OTSForRoot` (ErrNoRows→miss read),
  `PendingOTS` (oldest-first pending list), `MarkOTSUpgraded` (pending → confirmed) — over the
  pre-existing `ots` schema table, with 7 tests (`internal/store/ots_test.go`), all mutation-proven
  per the review verdict. The store stays a `net/http`-free leaf (verified: empty `net/http` in
  `go list -deps ./internal/store`). The `Attempts`/`NextRetry` columns are persisted + round-tripped
  but no method increments them yet (correctly deferred to the upgrade loop's retry policy). **Not yet
  built:** the daily stamp pass (write through `RecordOTS` for each distinct accepted root, never
  blocking the follower), the background upgrade loop (`PendingOTS` → `nbd-wtf/opentimestamps` calendar
  → `MarkOTSUpgraded`), the `.ots` HTTP route, and certificate §5. The milestone Verify (stamped root
  upgrades to Bitcoin-confirmed + served `.ots` verifies with the standard `ots` client) is still 1/1
  open.
- **WASM:** not started. No `internal/proof` package; no WASM build target (`syscall/js` not in any
  source file). The `internal/badge`, `internal/web`, `internal/metrics`, `internal/index`, and
  `internal/registry` leaves are WASM-shareable primitives the verifier app will reuse, but the verifier
  itself does not exist.

## Quality gates
**Status**: **GREEN at HEAD — review PASS + CI confirmed success.** HEAD (`381764d`) is the OTS-seam
review commit, in sync with `origin/develop` (0 ahead, 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` + the iscc-lib v0.5.0 require);
  `mise run check` runnable.
- The latest `review` verdict (`381764d`) is **PASS / CONTINUE**: `mise run check` green (all 21
  packages `ok`), 3 mutations proven non-vacuous, store leaf preserved, oracle gate correctly N/A
  (opaque-BLOB round-trip + plain CRUD, no crypto path), Codex clean.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **Latest CI run on HEAD (`381764d`) concluded
  `success`** (run 27922172138) — this also confirms the Go 1.26 toolchain provisioning + the iscc-lib
  transitive deps build clean under `CGO_ENABLED=0`, resolving the prior iteration's in-progress
  uncertainty. The prior run at `2bf66a8` also concluded `success`.
- **Open issues: 0 `critical`, 3 `normal`, 6 `low`.** The 3 `normal`: (a) Hub-List `hubDomain`
  `ForceQuery` fail-open; (b) §4 AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID
  (2 surfaces); (c) §6 RECORD HISTORY omits the per-record `· at` timestamp. The 6 `low` are
  loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. The OTS store seam is in place — the next
increment builds on it.** Per the review's stated roadmap, pick from:

1. **OTS daily stamp pass** — write through `RecordOTS` for each distinct accepted root without blocking
   the follower poll ("OTS never blocks the follower"). This is the immediate next sub-step and needs no
   new external dependency.
2. **OTS upgrade loop** — read `PendingOTS`, pull in `nbd-wtf/opentimestamps` + calendar HTTP, mark via
   `MarkOTSUpgraded` once Bitcoin-confirmed, exercising the `Attempts`/`NextRetry` retry policy. Then the
   `.ots` HTTP route and certificate **§5 BITCOIN ANCHOR** (`HasClause5`, reading `OTSForRoot`) — which
   unblocks the last open certificate clause and the Bitcoin-anchor panel.
3. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` → `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
4. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched. CI is confirmed green, so feature work proceeds.
