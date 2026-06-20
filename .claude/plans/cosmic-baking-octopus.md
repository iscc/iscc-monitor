# iscc-monitor — Trust & Transparency service for the ISCC-Hub network

## Context

Each ISCC-Hub publishes a **C2SP tlog-tiles** transparency log (`/log/checkpoint`, `/log/tile/<L>/<K>`,
`/log/tile/entries/<K>`; signed-note Ed25519 checkpoints; RFC-6962 / SHA-256 tree). Today **nothing independently
watches these logs**, which the ISCC-Log spec itself calls out as a weakness (`cauldron/iscc-hub/specs/iscc-log.md` §13:
a hub with no monitor "carries weaker guarantees, because no party is positioned to detect a split view"). The spec
defines explicit **Monitor / Aggregator** roles (§2.3) and anticipates OpenTimestamps anchoring (§12.1) but leaves them
unbuilt.

`iscc-monitor` becomes that independent party: a single, easy-to-deploy service that follows every hub in a realm,
continuously verifies checkpoint signatures + RFC-6962 consistency, mirrors the logs so it can serve **verifiable
inclusion proofs for any ISCC-ID** plus a **log browser**, **Bitcoin-anchors** the roots it has independently verified
via **OpenTimestamps**, and surfaces a **trust dashboard** — all without requiring anyone to trust the monitor itself
(clients verify the proofs locally, in-browser). The premise is already proven: running off-the-shelf conformance tools
against the live testnet during research caught a real key issue on one hub.

**What v1 guarantees (ADR-0001):** the monitor *autonomously* detects signature failures and **self-consistency
violations** (a hub rewriting history against this monitor). It does **not** autonomously detect **split views**
(equivocation) — a single monitor sees only one audience's view; instead it is the **comparison anchor** that lets any
client detect equivocation by checking *its own* `(size, root)` against the monitor's mirrored tree. All guarantees hold
only *from coverage start* (cold start is inherent; see ADR-0001 Coverage).

## Decisions (locked with the user; see `.claude/adr/`)

- **Trust root: did:web (ADR-0009).** A hub's signing key comes from its did:web document
  (`did:web:<domain>` → `/.well-known/did.json`), managed by the domain owner (rotation/revocation via CID 1.0
  `verificationMethod.revoked`). The realm registry advertises **domains/membership only — no keys**. Domain ownership
  *is* the identity; **domain-compromise detection is out of scope**.
- **Stack: Go (ADR-0003).** Real rationale: (1) reuse the battle-tested Go transparency stack rather than owning
  security-critical proof code, (2) one verifier codebase compiling to both native and WASM. Single static binary
  (CGO_ENABLED=0) is a secondary benefit. OpenTimestamps via `nbd-wtf/opentimestamps`.
- **No cosigning in v1 (ADR-0004).** The v1 independent attestation is **OTS-anchoring observed roots** (trustless
  timestamp, no monitor signing key in the trust path). Cosigning has no v1 consumer; it returns at **M7** as the C2SP
  witness/gossip wire format. The monitor therefore needs a TLS identity but publishes **no signing key** in v1.
- **Client verification surface (ADR-0003):** (a) **in-browser** verifier (Go→WASM, in v1, for non-technical users),
  (b) **downloadable proof bundles**, (c) **REST verify-for-me** (thin, non-authoritative). All three share one pure
  verifier package. The verifier MUST accept the client's *own* `(size, root)` and prove it against the monitor's tree
  (compare, not trust). The WASM build MUST be reproducible + hash-pinned (the monitor could otherwise equivocate about
  its own verifier).
- **Storage: one SQLite database per network (ADR-0005, ADR-0007).** Everything — checkpoints, violations, OTS proofs,
  the iscc_index, **and the mirrored tiles + entry bundles (as BLOBs)** — lives in SQLite. No filesystem tile mirror.
  `mainnet.db` and `testnet.db` are separate files (isolation + simpler schema; the `network` column is gone).
- **OpenTimestamps (ADR-0004):** anchor **each hub's checkpoint root daily**, keyed by `(hub, tree_size, root)` UNIQUE
  (so each *distinct* root is anchored once — re-anchoring an unchanged root only yields a weaker, later "existed-before"
  proof). No "notable growth" trigger (YAGNI; re-addable for active hubs later). Staleness ("unchanged for a day") is a
  real signal but is delivered by the **observation history + dashboard lag**, not by OTS. Calendar-attested
  server-side; authoritative verification is the user running `ots verify`.

## Architecture (one process; state in per-network SQLite databases)

1. **Realm registry sync** — fetch the realm membership doc (domains only), derive per-hub
   `{domain, origin = domain+"/log", base_url}`, reconcile add/remove/inactive. Reconciliation changes follow-status
   only; it **never deletes evidence or mirror** (ADR-0007).
2. **did:web key resolution (ADR-0009)** — resolve `did:web:<domain>` → DID doc → Ed25519 key(s) + validity. Port
   `derive_vkey.py`'s did→verifier-key derivation; the *source* is the DID doc, not a YAML key list. Resolution failure
   → status `unresolvable` (keep mirroring). Cache resolved keys; re-resolve on cadence.
3. **Per-hub follower** (`transparency-dev/tessera/client`) — poll `/log/checkpoint`; verify the Ed25519 signed-note
   against the did:web key; then run the **three self-consistency checks (ADR-0006)** against last accepted: size
   monotonic (else **shrink**), same-size-same-root (else **equivocation**), forward `ProofBuilder.ConsistencyProof` +
   `merkle/proof.VerifyConsistency` (else **fork**). On any violation: persist both raws + proof permanently, `frozen=1`,
   alert once, keep serving last consistent state; **keep polling evidence-only** (no advance/anchor) at a backed-off
   cadence; **no auto-unfreeze**. Else pull only the new tile/bundle range, re-derive the root from fetched tiles
   (fsck-equivalent), persist, extend the `iscc_index`, then mark OTS-eligible. Record `monitored_since` on first
   observation (ADR-0001 Coverage).
4. **Storage (ADR-0005)** — `modernc.org/sqlite` (WAL, **single writer goroutine per DB file**). Per poll: fetch tiles
   over the network *outside* the write transaction, then commit `{checkpoint, any violation+freeze, new full
   tiles/bundles, index updates}` in **one atomic transaction**. Tiles/bundles are BLOBs keyed by `(level, index,
   width)`. Irreplaceable evidence (checkpoints, violations, OTS) is what backups protect; tiles + index are rebuildable
   via re-fetch + fsck.
5. **Mirror serving (ADR-0005)** — a `SQLiteFetcher` implements `client.Fetcher` (`ReadCheckpoint`, `ReadTile`,
   `ReadEntryBundle`) over the BLOBs, so `fsck.New(...)` runs against the local store; canonical tlog-tiles paths are an
   HTTP handler over the same BLOBs (Aggregator / availability backstop). A later `export-mirror` command can write the
   static tlog-tiles tree if a CDN-servable copy is ever wanted.
6. **OpenTimestamps (ADR-0004)** — stamp each distinct observed root to calendars **daily** (UNIQUE dedupes unchanged
   roots); background upgrade loop (pending → Bitcoin-confirmed); serve `.ots`. Server-side Bitcoin verification is
   **calendar-attested only**; authoritative verification is the user running `ots verify`. OTS **never blocks** the
   follower loop.
7. **Trust API (REST)** — hubs/status (incl. **coverage window**); latest checkpoint (+ots); `inclusion?iscc_id=…[&seq=]`
   → self-contained bundle `{checkpoint_note, treeSize, leafIndex, inclusionProof[], record_bytes, hub_didweb_key, ots?,
   other_records:[(seq,$schema)]}` (ADR-0008 generic per-id record list); `consistency?hub&from&to`; entry/range fetch;
   `verify-for-me` (wraps the shared verifier); plus the raw tlog-tiles mirror at canonical paths.
8. **Frontend** — server-rendered (`html/template`, `go:embed`) per-hub dashboard (size, lag, last-seen, **coverage
   since**, status [verified/unresolvable/unverified/frozen/inactive], violation/OTS status) + log browser;
   **lazy-loaded WASM** verifies checkpoint+inclusion+consistency fully client-side (reproducible + hash-pinned).
9. **Observability & alerting** — three layers, **no built-in notifier integrations**: (a) `/metrics` (Prometheus) with
   explicit series for every alert-worthy condition (per-hub `status`, `violations_total{kind}`, `unresolvable`, poll/
   availability failures, `lag_seconds`, `last_observed_at`) — the primary path, operators alert via Alertmanager;
   (b) **one generic outbound webhook** (URL + optional HMAC secret) that POSTs a structured JSON event **on state
   *transitions*** (`hub_frozen` with `kind`, `became_unresolvable`, `availability_failure`, and recovery transitions),
   fire-and-forget with backoff, never blocking the follower; (c) structured logs. Three distinct failure classes —
   `violation` (frozen, with kind) · `unresolvable` (did.json) · `availability_failure` (endpoints unreachable) — are
   kept separate. Plus a continuous **inclusion cross-check** job (Verification §4).

## Go module / package layout

Module `github.com/iscc/iscc-monitor`, Go 1.24, `CGO_ENABLED=0`.

```
cmd/iscc-monitor/main.go        # wire DI, supervisor, graceful shutdown
internal/
  config/                       # env+flags+file; network = testnet|mainnet|both; per-network DB paths; OTS calendars
  registry/                     # fetch+parse realm membership (domains only); reconcile add/remove/inactive
  didweb/{resolve.go,vkey.go}   # did:web -> DID doc -> Ed25519 key(s)+validity; origin(); keyID(); verifierKey() (port derive_vkey.py)
  logclient/{origin.go,follower.go,verify.go}   # origin = domain+"/log" (single source of truth); tessera/client wrap; backfill+live; 3-trigger consistency
  store/{sqlite.go,schema.sql,fetcher.go,models.go}   # per-network WAL + single-writer; tiles/bundles as BLOBs; SQLiteFetcher implements client.Fetcher
  index/iscc.go                 # parse iscc_id per record; iscc_id -> []seq (ONE-TO-MANY); store raw note.$schema (schema-agnostic); ISCC-ID codec
  ots/{anchor.go,upgrade.go,serve.go}    # stamp/upgrade/serialize; verify.go optional, default off
  proof/{bundle.go,verify.go}   # verify.go = PURE (no net/os/sqlite): shared by WASM + verify-for-me + tests
  api/{server.go,handlers_trust.go,handlers_verify.go,handlers_mirror.go}
  web/{server.go,templates/,assets/}     # dashboard + browser; embeds wasm_exec.js + monitor.wasm
  metrics/  supervisor/
wasm/verify/main.go             # GOOS=js GOARCH=wasm; exposes verify* to JS (same internal/proof/verify source)
build/                          # reproducible GOOS=js GOARCH=wasm build -> internal/web/assets/monitor.wasm (+ published hash)
testdata/live/                  # captured sb0 checkpoints/tiles/receipts/did.json for offline tests
# cosign/  -> DEFERRED to M7 (gossip/witness); not in v1
```

**Reused libraries (prefer over reimplementing):** `transparency-dev/tessera` (`client`, `api`, `api/layout`, `fsck`),
`transparency-dev/merkle` (`rfc6962`, `proof`), `transparency-dev/formats` (`note`, `log`), `golang.org/x/mod/sumdb/note`,
`github.com/nbd-wtf/opentimestamps`, `modernc.org/sqlite`, `gopkg.in/yaml.v3`, `github.com/mr-tron/base58`,
`github.com/prometheus/client_golang`. Router: stdlib `net/http` 1.22 patterns (add chi only if needed). Defer
`transparency-dev/witness` / `filippo.io/torchwood` and `note.Sign`-based cosigning to the M7 witness/gossip milestone.

**Local reference files to port/oracle against** (vendored read-only copies under `cauldron/`, gitignored; mirrors the
sibling `iscc-hub`/`tessera` repos so these resolve without a specific checkout layout):
- `cauldron/tessera/client/{client.go,fetcher.go}` — clone/verify spine + the `Fetcher` interface our `SQLiteFetcher` mirrors.
- `cauldron/iscc-hub/conformance/runfsck/main.go` — the exact leaf hasher (`rfc6962.DefaultHasher.HashLeaf` over `api.EntryBundle`) + `fsck.New(...)`; the structural oracle (we feed it a `SQLiteFetcher`).
- `cauldron/iscc-hub/conformance/notecheck/main.go` — the checkpoint-signature oracle (can shell out in CI).
- `cauldron/iscc-hub/iscc_hub/checkpoint_note.py` — exact signed-note / keyid wire format to mirror in Go.
- `.claude/derive_vkey.py` — did → origin → verifier-key derivation to port into `internal/didweb/vkey.go`; the two live testnet hubs are golden vectors (source the did from did:web, not the YAML).
- `cauldron/iscc-hub/iscc_hub/log_tree.py` (`inclusion_evidence`, `log_origin`) + `cauldron/iscc-hub/iscc_hub/schema.py` (`Evidence`) — authoritative shape of the `IsccLogInclusionProof` cross-check oracle (the `cauldron/iscc-hub/specs/schemas/iscc-receipt.yaml` is stale; trust `schema.py`).

## SQLite schema (core tables; per-network DB, no `network` column)

`hubs(hub_id PK, domain, origin, base_url, active, status, monitored_since_size, monitored_since_time, first_seen, last_seen)` ·
`hub_keys(hub_id, key_id, pubkey_raw, pubkey_z, revoked_at, resolved_at)` — did:web key cache (source of truth is the DID doc) ·
`checkpoints(id PK, hub_id, tree_size, root, raw, observed_at, consistent, root_rebuilt; UNIQUE(hub_id,tree_size,root))` ·
`violations(id PK, hub_id, kind, detected_at, raw_a, raw_b, proof_json)` — `kind ∈ {fork,shrink,equivocation}` ·
`tiles(hub_id, level, tile_index, width, data BLOB, is_full, sha256, updated_at, PK(hub_id,level,tile_index,width))` ·
`entry_bundles(hub_id, bundle_index, width, data BLOB, is_full, sha256, updated_at, PK(hub_id,bundle_index,width))` ·
`iscc_index(hub_id, seq PK, iscc_id BLOB, iscc_id_str, note_schema, record_sha256; INDEX(iscc_id))` —
**`iscc_id → seq` is one-to-many**; `note_schema` stores the raw `note.$schema` (schema-agnostic; unknown types indexed, never gate verification) ·
`follow_state(hub_id PK, last_size, frozen, last_error)` ·
`ots(id PK, hub_id, tree_size, root, status, ots_bytes, calendar_urls, stamped_at, upgraded_at, btc_height, attempts, next_retry; UNIQUE(hub_id,tree_size,root))`.
*(No `cosigs` table in v1 — deferred to M7.)*

## Correctness rules (the load-bearing gotchas)

1. **Origin = `<domain>/log`**, not the bare domain — used for the signed-note `name` and the verifier key. One
   `origin()` helper, golden-tested against both live hubs. (Highest-probability bug.)
2. **`iscc_id → seq` is one-to-many, and verification is schema-agnostic (ADR-0008).** Index by `seq`; store the raw
   `note.$schema`; lookups return a list. Unknown schemas are indexed + proof-able but never interpreted and never gate
   checkpoint acceptance. `inclusion?iscc_id=` proves the declaration by default and lists all other records
   `(seq,$schema)`; `?seq=` disambiguates.
3. **Partial-tile discipline (ADR-0005).** Mark a tile/bundle BLOB `is_full` (immutable) only when `width == 256`;
   re-fetch partials (`.p/<W>`) every poll and overwrite. Never promote a partial to a full row.
4. **Self-consistency violation freezes, never crashes (ADR-0006).** Three triggers — fork / shrink / equivocation.
   Persist both raws + proof permanently, set `frozen=1`, alert once. Keep polling evidence-only (no advance/anchor) at a
   backed-off cadence; **no auto-unfreeze**; survive restart; other hubs unaffected.
5. **Coverage honesty (ADR-0001).** Record `monitored_since`; state guarantees *from coverage start*. A late start
   backfills tiles for full data + forward guarantees but cannot retroactively detect pre-coverage equivocation. The
   dashboard must show the coverage window.
6. **SQLite single writer per DB (ADR-0005, 0007).** WAL + `busy_timeout`; one goroutine owns all writes for each
   network's file; network fetch happens outside the write transaction.
7. **OTS never blocks** the follower loop; calendars are best-effort with backoff + multi-calendar redundancy.
8. **did:web is the only key source (ADR-0009).** Resolution failure → `unresolvable` (keep mirroring). A signature that
   doesn't match the hub's own did:web key → `unverified` (internally-broken hub). Domain-compromise detection is out of
   scope.

## Milestones (revised per ADR-0004)

- **M1 — Read-only Monitor (working core):** config + realm registry + did:web resolution + `vkey`/`origin`
  (golden-tested) + follower (verify sig + three-trigger consistency, persist, freeze) + per-network SQLite + coverage
  tracking + logs + `/metrics`. Survives restart.
- **M2 — Aggregator:** tiles + entry bundles in SQLite (partial-tile discipline) + `SQLiteFetcher` (fsck-able) +
  `iscc_index` (schema-agnostic) + `inclusion`/`consistency`/`entries` served from the local store via `ProofBuilder`
  (never re-hitting the hub).
- **M3 — Trust API + dashboard:** REST surface + verify-for-me + server-rendered dashboard (status, coverage, lag,
  violations, OTS) & log browser + raw mirror at canonical paths.
- **WASM upgrade:** in-browser verifier (`internal/proof/verify` → WASM), lazy-loaded progressive enhancement on M3;
  reproducible build + published hash + SRI pin (ADR-0003).
- **OTS / Bitcoin anchoring:** stamp observed roots on cadence/growth + upgrade loop + serve `.ots` (the v1 independent
  attestation; calendar-attested). **No cosigning.**
- **M7 (deferred):** multi-monitor gossip **+ cosigning** (C2SP witness cosignatures) + witness endpoint
  (`tlog-witness add-checkpoint`) + cross-monitor split-view comparison. Out of v1.

## Verification (end-to-end, live testnet `sb0.iscc.id/log`, reusing existing oracles)

1. **Origin/did:web golden test first:** `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and our `verifierKey`
   (derived from the hub's did:web key) byte-matches `derive_vkey.py` for both testnet hubs.
2. **Checkpoint parity:** our verdict for every accepted checkpoint matches the `notecheck` binary (shell out in CI).
3. **Structural oracle (live + local):** `tessera/fsck` with `runfsck`'s leaf hasher, fed a **`SQLiteFetcher`** over our
   store *and* the live hub, must reconstruct the same root.
4. **Inclusion cross-check (unique oracle, promoted to a continuous job):** for sampled `iscc_id`s, our computed
   inclusion proof matches the hub's `GET /declaration/{iscc_id}/receipt` → `evidence.IsccLogInclusionProof` and verifies
   to the same root.
5. **Violation drill:** synthetic fork / shrink / equivocation checkpoints → correct `violations` row recorded with
   `kind`, hub frozen, alert fired, other hubs unaffected, evidence survives restart.
6. **WASM/REST parity:** identical vectors through the WASM build and the server-side verifier yield identical verdicts;
   verifier artifact hash matches the published value.
7. **OTS:** a stamped root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client.
8. **Coverage/cold-start:** a monitor started late backfills tiles, reconstructs the full tree + holds forward
   guarantees; `monitored_since` is recorded and the dashboard scopes guarantees to it.

## Config defaults (no further user input needed)

One binary; `network` selects `testnet` / `mainnet` / `both`; **one SQLite file per network** (`mainnet.db`,
`testnet.db`). Keys resolved from each hub's **did:web** document — no central key list. OTS calendars default set. **No
cosigning key in v1** (the monitor needs only a TLS identity). Ships as the static binary plus a container image
(mirroring the hub's release flow). License Apache-2.0. `derive_vkey.py` stays as the reference/golden-vector source for
`internal/didweb/vkey.go` (key resolution from did:web).
