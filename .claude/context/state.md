<!-- assessed-at: 2bf66a8bb056ffbb307be3563189b6ecfe8f3e72 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — stack now on the locked Go 1.26 + iscc-lib v0.5.0 (ADR-0011 closed, TARGET/CODE gap eliminated). Remaining M-UI tail: §5 BITCOIN ANCHOR (OTS-blocked), the anchor panels, the WASM verifier, OTS anchoring, and the mandatory M-UI visual-exit gate.

The ADR-0011 stack increment landed and was reviewed PASS: `go.mod` is now `go 1.26.1` with a pinned
`github.com/iscc/iscc-lib/packages/go v0.5.0` require, mirrored across `mise.toml` (`go = "1.26"`),
`ci.yml` (`go-version: "1.26"`), and the devcontainer Dockerfile — closing the long-standing
TARGET/CODE stack mismatch. The certificate of inclusion still has five of six clauses sound (§1, §2,
§3, §4, §6) plus a working downloadable proof bundle; §5 BITCOIN ANCHOR remains blocked on the absent
OTS store seam. M1/M2/M3 remain fully met.

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
    download LINK** (canonical path-rooted `BundleHref`, both id forms — critical closed at `34e189e`).
    **Open:** certificate **§5 BITCOIN ANCHOR** (`HasClause5` deliberately false until the OTS seam
    exists); the separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI exit
    visual-pass + human sign-off** (ADR-0012 agent-browser).
  - **WASM verifier: 1/1 open** (re-verified not started — no `internal/proof` package, no `syscall/js`
    in any source file).
  - **OTS anchoring: 1/1 open** (re-verified not started — `opentimestamps` absent from `go.mod`). §5 of
    the certificate is its downstream consumer, so OTS gates the last certificate clause too.
- **Last ~10 iterations: ~7 milestone-Verify-advancing / ~3 refactor·foundational·gate.** The recent arc
  built the certificate clause-by-clause (cert §3 fail-close → §6 → proof-bundle endpoint → #ZgotmplZ
  link fix → ADR-0011 stack bump). The latest increment (ADR-0011) is foundational config, not feature
  drift: it closes a target-mandated stack gap that was filed as a `normal` issue and is a prerequisite
  for the iscc-lib codec reuse the target locks. No drift — the remaining work (§5 / OTS / WASM / visual
  exit gate) is the genuinely large, partly-blocked tail, not avoidable polish.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `34e189e..HEAD` diff touched ONLY build config (`go.mod`,
`go.sum`, `mise.toml`, `.devcontainer/Dockerfile`, `.github/workflows/ci.yml`), one new test file
(`internal/index/iscclib_tripwire_test.go`), and `.claude/*` context — **no M1 source touched.** All M1
Verify criteria remain satisfied: `origin`/`vkey` golden; all three triggers (fork/shrink/equivocation)
golden-tested end-to-end with freeze + alert-once + restart survival; coverage tracked; structured logs;
`/metrics` over HTTP.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **20 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, **now `go 1.26.1`** (no explicit `toolchain` line; CI `setup-go`
  resolves the latest 1.26 patch).
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`Consistency`+`VerifyInclusion`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (Hub-List parser). **Newly wired:**
  `github.com/iscc/iscc-lib/packages/go v0.5.0` — present as a live require (`go.sum` has 2 iscc-lib
  entries) but **test-only** in the build closure: the production decoder leaf `internal/index/iscc.go`
  does NOT import it (verified empty grep), held by the carve-out (ADR-0011) + the tripwire test. **Not
  wired:** `nbd-wtf/opentimestamps`.

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
**Status**: **in progress — carried forward (no M-UI source in the diff); proof-bundle endpoint +
download link both landed and verified; §5 + anchor panels + visual-exit gate remain.** §1 SUBJECT,
§2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY, and §6 RECORD HISTORY are all sound. Badge, DS shell,
`/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, the
ISCC-IDv1 decoder, and the Hub-List resolver are all built and verified.
- **§3 INCLUSION PROOF (verified):** builds the RFC-6962 proof of `data.Position` against `hub.LastSize`
  over a `SQLiteFetcher`, gates `HasClause3` on `proof.VerifyInclusion(...) == nil` against the §2
  accepted root. Fails closed; mutation-proven.
- **Proof-bundle ENDPOINT + download LINK (verified, critical closed at `34e189e`):** `serveBundle`
  writes a self-contained `{checkpoint, inclusion, record, key}` JSON gated on the same §3
  re-verification; `cert.html` links a canonical path-rooted `BundleHref` working for both the bare and
  the `ISCC:`-prefixed request forms. Both mutation-proven.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` deliberately false;
  BLOCKED on a non-existent OTS/anchor store seam); the separate **Bitcoin-anchor vs comparison-anchor**
  panels; and the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `opentimestamps` not in `go.mod`/`go.sum` or source; no
`internal/proof` package; no WASM build target (`syscall/js` not in any source file). The
`internal/badge`, `internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves
are WASM-shareable primitives the verifier app will reuse (the production `internal/index` leaf is
confirmed WASM-buildable + iscc-lib-free), but the verifier itself does not exist. OTS is the upstream
blocker for certificate §5.

## Quality gates
**Status**: **GREEN at HEAD per the review verdict; CI run for HEAD still in progress.** HEAD
(`2bf66a8`) is the ADR-0011 review commit, in sync with `origin/develop` (0 ahead, 0 behind). The
latest `review` verdict (`2bf66a8`) is **PASS / CONTINUE** with `mise run check` green under Go 1.26.1
(all 21 packages `ok`), the oracle gate N/A (dependency + tripwire test, no crypto path touched), and
Codex clean.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.26.1`** + the iscc-lib v0.5.0
  require); `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR, now pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **Latest CI run on HEAD (`2bf66a8`) is
  `in_progress`** (first run exercising the Go 1.26 toolchain provisioning + the three new transitive
  deps under `CGO_ENABLED=0`); the prior run at `34e189e` concluded `success`. Watch this run conclude
  before treating CI as confirmed-green at the new toolchain.
- **TARGET/CODE GAP: CLOSED.** `target.md` "Stack (locked)" mandated Go 1.26 + iscc-lib v0.5.0; the
  code now matches on all four config surfaces. The previously-open foundational `normal` issue was
  resolved and deleted by review.
- **Open issues: 0 `critical`, 3 `normal`, 6 `low`.** The 3 `normal`: (a) Hub-List `hubDomain`
  `ForceQuery` fail-open; (b) §4 AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID
  (2 surfaces); (c) §6 RECORD HISTORY omits the per-record `· at` timestamp. The 6 `low` are
  loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone; the ADR-0011 stack prerequisite is now satisfied.** The
next `define-next`/`advance` should pick from:

1. **OTS / Bitcoin anchoring** — build the anchor store seam + stamp/upgrade loop, which then
   **unblocks certificate §5 BITCOIN ANCHOR** and the Bitcoin-anchor panel. §5 cannot be honestly
   rendered until this exists. Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle
   sites) when handler.go is next touched.
2. **WASM verifier** remains a v1 milestone (1/1 Verify open) — `internal/proof/verify` →
   `GOOS=js GOARCH=wasm` plus the `monitor.iscc.codes` Independent Verification app. The stack bump just
   landed makes the iscc-lib codec available where generic ISO-24138 codec work is needed.
3. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.

If the in-progress CI run at `2bf66a8` fails on Go 1.26 toolchain provisioning, fixing CI preempts all
feature work.
