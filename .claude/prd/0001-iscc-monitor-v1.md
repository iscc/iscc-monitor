# PRD: iscc-monitor v1 — Trust & Transparency service for the ISCC-Hub network

> Status: ready-for-agent (to be published to the issue tracker once the repo exists; apply the `ready-for-agent` label).
> Synthesized from the grilling/domain-modeling session. Authoritative decisions live in `.claude/adr/0001–0009`,
> vocabulary in `CONTEXT.md`, the build plan in `.claude/plans/cosmic-baking-octopus.md`, and the cross-repo asks in
> `.claude/iscc-hub-side-changes.md`. Where this PRD and an ADR disagree, the ADR wins.

## Problem Statement

ISCC-Hubs each publish a tlog-tiles transparency log, but **nobody independently watches them**. A declarer who logged
an ISCC-ID has no way to prove inclusion without trusting the hub's continued good behavior or even its continued
existence; an auditor or litigator has no independent evidence; and the network has no party positioned to notice if a
hub rewrites its history or shows different histories to different audiences. The ISCC-Log spec itself flags this as a
weakness (§13 "cold start") and defines Monitor/Aggregator roles, but leaves them unbuilt. Today the only way to check a
hub is to run conformance tools by hand — which, when done once during research, already caught a real key issue on a
live testnet hub.

Crucially, the people who most need to trust the network — declarers, auditors, businesses, legal staff — are **not
command-line users**. They need to *see* the trust situation and verify a claim in a browser, without installing tools
and without having to trust the watcher either.

## Solution

A single, easy-to-deploy service — a **monitor instance** — that follows every hub in a realm and continuously:

- resolves each hub's signing key from its **did:web** document, verifies every checkpoint signature, and verifies
  **RFC-6962 consistency** between successive observations, detecting **self-consistency violations** (fork / shrink /
  equivocation) and **freezing** a misbehaving hub while preserving permanent evidence and continuing to serve;
- **mirrors** every hub's tiles and entry bundles so it is an availability backstop and can serve **verifiable inclusion
  and consistency proofs for any ISCC-ID** from its own store;
- **Bitcoin-anchors** the roots it has verified via **OpenTimestamps**, giving a trustless timestamp and anti-rewrite
  guarantee;
- surfaces a **trust dashboard** and **log browser**, and — for the people who won't touch a CLI — a **verifier app**
  (`monitor.iscc.codes`) whose Go→WASM core re-verifies everything **in the user's own browser**, so nobody has to trust
  the monitor.

The monitor is a **verifiable cache, not a trusted oracle**: fetching data from it is never the same as trusting its
verdict. For inclusion, clients verify the *hub's* signature plus Merkle math; for equivocation, clients compare their
own `(size, root)` against the monitor's mirrored tree; for time, anyone runs `ots verify`. All guarantees hold **from
coverage start** (`monitored_since`), and the dashboard says so.

## User Stories

1. As a **declarer**, I want a verifiable inclusion proof for my ISCC-ID, so that I can prove my declaration is committed
   in the hub's log without trusting the hub.
2. As a **declarer**, I want to fetch a self-contained proof bundle, so that I can keep durable evidence of inclusion
   even if the hub later goes offline or prunes data.
3. As a **declarer whose ISCC-ID was later deleted**, I want the inclusion response to also tell me a deletion record
   exists (and at which seq), so that I see the full per-id record history, not just the declaration.
4. As an **auditor/litigator**, I want to verify a proof bundle entirely myself (CLI or library), so that the evidence
   stands without trusting either the hub or the monitor.
5. As an **auditor**, I want to attest once that the verifier app faithfully implements the algorithm, so that every
   non-technical user can rely on in-browser verification.
6. As a **non-technical business user**, I want to open a web page and see each hub's trust status at a glance, so that I
   can judge the network's health without tools.
7. As a **non-technical business user**, I want my own browser to re-verify a checkpoint/inclusion claim (not just read a
   badge the server rendered), so that I don't have to trust the monitor's display.
8. As a **skeptical client**, I want to check my own `(size, root)` against the monitor's mirrored tree, so that I can
   detect if the hub showed me a different history than it showed the monitor (a split view).
9. As a **skeptical client**, I want the verifier app to be served from an independent origin (`monitor.iscc.codes`), so
   that a malicious monitor instance cannot serve me a tampered verifier.
10. As a **client of any monitor**, I want one verifier app that works against any monitor instance via `?monitor=<url>`,
    so that I audit one checker and use it across the whole federation.
11. As an **ISCC Foundation operator**, I want to run a monitor instance at `monitor.iscc.id`, so that the federation has
    at least one independent watcher (avoiding §13 cold start).
12. As a **third party**, I want to run my own monitor instance on my own domain, so that the network does not depend on
    a single watcher.
13. As a **monitor operator**, I want a single static binary plus a container image, so that deployment is trivial and
    matches the hub's release flow.
14. As a **monitor operator**, I want one binary that can follow `testnet`, `mainnet`, or `both`, so that I can run one
    process while keeping networks isolated.
15. As a **monitor operator**, I want each network's state in its own SQLite file, so that testnet churn or a testnet bug
    cannot corrupt or stall the irreplaceable mainnet evidence.
16. As a **monitor operator**, I want the monitor to survive restart with no lost evidence, so that detected violations
    and observed checkpoints are durable.
17. As a **monitor operator**, I want a self-consistency violation to freeze only the offending hub while all other hubs
    keep being served, so that one bad hub never takes down coverage of the rest.
18. As a **monitor operator**, I want a frozen hub to keep being polled in evidence-only mode, so that I keep collecting
    proof of ongoing misbehavior without advancing or anchoring bad state.
19. As a **monitor operator**, I want freezes to require manual review to clear, so that a misbehaving hub cannot silently
    "heal" and erase the alert.
20. As a **monitor operator**, I want alerts on state transitions (frozen, became-unresolvable, availability failure, and
    recoveries), so that I'm notified once per change rather than spammed every poll.
21. As a **monitor operator**, I want alerting via Prometheus `/metrics` and a generic HMAC-signed webhook, so that I can
    wire notifications into my own stack without vendor lock-in.
22. As a **monitor operator**, I want structured logs of every observation and decision, so that I can audit the
    monitor's own behavior.
23. As a **monitor operator**, I want OpenTimestamps anchoring to never block the follower loop, so that a slow or down
    calendar never delays verification.
24. As a **hub operator**, I want the monitor to read my signing key from my did:web document, so that I manage and
    rotate my own key without anyone editing a central list.
25. As a **hub operator**, I want to rotate my checkpoint signing key by editing my own did.json (CID 1.0 `revoked`), so
    that rotation needs no coordination with the monitor or a central registry.
26. As a **hub operator**, I want a key rotation to not flip my hub's status to broken, so that legitimate rotation is
    seamless.
27. As a **hub operator**, I want my hub's status surfaced honestly (verified / unresolvable / unverified / frozen /
    inactive), so that I can spot a misconfiguration (e.g. signing with a key my did.json doesn't list).
28. As a **hub operator who goes offline**, I want the monitor to keep serving my mirrored log, so that my declarers'
    proofs remain verifiable even without me.
29. As a **declarer/auditor**, I want a trustless timestamp that a root existed before a given Bitcoin block, so that I
    can prove *when* something was logged without trusting any clock.
30. As a **declarer/auditor**, I want to download the `.ots` and run `ots verify` myself, so that the Bitcoin timestamp
    is authoritative, not calendar-asserted.
31. As a **client**, I want to browse a hub's log records in the dashboard, so that I can inspect what was declared.
32. As a **client**, I want each hub's coverage window (`monitored_since`) shown, so that I don't over-trust guarantees
    for periods before the monitor was watching.
33. As a **monitor**, I want to verify any checkpoint signature against any key the hub's did.json lists (current or
    revoked), so that historical checkpoints stay verifiable across rotations (consistent with the domain-ownership
    axiom).
34. As a **monitor**, I want to keep mirroring and serving a hub even when it is `unresolvable` or `unverified`, so that
    I report the problem instead of going dark on it.
35. As a **monitor**, I want to index *every* record by its ISCC-ID and its raw `note.$schema`, including types I don't
    recognize, so that inclusion proofs work for any record and future message types never break me.
36. As the **ISCC Foundation**, I want a new message type (e.g. gateway-url update, ownership transfer) to be mirrored,
    proven, and listed by existing monitors the day it appears, so that I can evolve the log without forcing a monitor
    upgrade first.
37. As a **monitor operator**, I want to run an integrity self-check (`fsck`) over my own stored tiles, so that I can
    confirm my mirror reconstructs each accepted root.
38. As a **CI pipeline**, I want the monitor's verdict for every accepted checkpoint to match the `notecheck` oracle and
    its mirror to pass `fsck` with the `runfsck` leaf hasher, so that conformance is continuously proven.
39. As a **monitor**, I want a continuous inclusion cross-check job comparing my computed proof against the hub's own
    receipt (`evidence.IsccLogInclusionProof`), so that two independent derivations agree.
40. As a **developer integrating the monitor**, I want a `verify-for-me` REST endpoint, so that I can get a quick
    (explicitly non-authoritative) verdict without embedding the verifier — knowing the authoritative path is verifying a
    bundle myself.
41. As a **client/tool on another origin**, I want the monitor's read API and mirror to be CORS-enabled, so that the
    browser verifier app and third-party tools can fetch and re-verify the data.
42. As a **monitor operator**, I want the raw tlog-tiles mirror served at canonical paths, so that off-the-shelf
    transparency tooling can treat my instance as an aggregator/backstop.

## Implementation Decisions

**Architecture (ADR-0005, ADR-0007).** One process; all state in **per-network SQLite databases** (`mainnet.db`,
`testnet.db`), WAL, one single-writer goroutine per file. Everything — observed checkpoints, violations, OTS proofs, the
ISCC-ID index, **and the mirrored hash tiles + entry bundles as BLOBs** — lives in SQLite; there is **no filesystem tile
mirror**. The `network` column is dropped (separate files make it redundant). Per poll: fetch over the network *outside*
the write transaction, then commit `{checkpoint, any violation+freeze, new full tiles/bundles, index updates}` in **one
atomic transaction**.

**Trust root: did:web (ADR-0009).** The hub's signing key is resolved from `did:web:<domain>` →
`https://<domain>/.well-known/did.json`; the realm registry advertises **domains/membership only — no keys**. Rotation
lives in the DID document (CID 1.0 `verificationMethod` + `revoked`). The signed-note `name` remains the origin
(`<domain>/log`); the did→verifier-key derivation is ported from `derive_vkey.py`. Domain ownership *is* the identity;
**domain-compromise detection is out of scope**, so a checkpoint is accepted if it verifies against *any* Ed25519 key the
DID document lists (current or revoked). The key source is a **pluggable resolver** so a future on-chain registry could
be added without touching verification.

**Hub status taxonomy (ADR-0002 revised by ADR-0009):** `verified` · `unresolvable` (can't fetch/parse did.json) ·
`unverified` (signature matches no listed key = internally-broken hub) · `frozen` (self-consistency violation) ·
`inactive` (removed/paused in the registry). Hubs are **followed and mirrored regardless of status**; only `verified`
checkpoints advance accepted state. Reconciliation changes follow-status only and **never deletes evidence or mirror**.

**Follower & freeze (ADR-0006).** Per poll, in order: parse + verify signature → **size monotonic?** (else `shrink`) →
**same-size ⇒ same-root?** (else `equivocation`) → **forward RFC-6962 consistency** vs last accepted (else `fork`) → only
then accept, mirror, index, mark OTS-eligible. On any trigger: persist both contradictory raw checkpoints + proof
permanently to `violations(kind ∈ {fork,shrink,equivocation})`, set `frozen=1`, **alert once**, keep serving last
consistent state. A frozen hub keeps being polled **evidence-only** at a backed-off cadence (no advance/anchor/OTS);
**no auto-unfreeze**.

**Coverage (ADR-0001).** Record `monitored_since` (size + time) per hub on first observation. A late-starting monitor
backfills all tiles to reconstruct + verify the full tree (full data + forward guarantees) but **cannot** retroactively
detect pre-coverage equivocation; all guarantees are stated *from coverage start*. Missing intermediate checkpoints is
harmless — RFC-6962 consistency is transitive, so the monitor verifies consistency between the checkpoints it caught.

**Schema-agnostic verification (ADR-0008).** The verification/evidence path depends only on raw record bytes + the tree +
checkpoint signatures, never on `note.$schema`. The index stores the raw `note.$schema` per record. Interpretation
(declaration vs deletion vs future types) is a separate, additive projection layer; **resolution projections (current
owner/gateway) are out of v1 scope** but enabled by the schema-agnostic index.

**Mirror & fsck (ADR-0005).** A `SQLiteFetcher` implements tessera's existing `client.Fetcher`
(`ReadCheckpoint`/`ReadTile(l,i,p)`/`ReadEntryBundle(i,p)`) over the BLOBs, so `fsck.New(...)` and `ProofBuilder` run
against the local store unchanged, and the canonical tlog-tiles paths are an HTTP handler over the same BLOBs. **Partial
tiles** (`width<256`) are re-fetched each poll and overwritten; a tile/bundle is marked `is_full` (immutable) only at
`width==256`.

**Client verification (ADR-0003).** A single **pure** `proof/verify` package (no net/os/sqlite) is shared by the
server-side verifier, the CLI/library, and the **WASM** build. The verify surface MUST accept the client's *own*
`(size, root)` and prove it against the monitor's tree (compare, not trust). The in-browser verifier is **in v1** for
non-technical users, hosted at **`monitor.iscc.codes`** (GitHub Pages from the repo) — monitor-agnostic via
`?monitor=<url>`, reproducibly built, hash-published, SRI-pinned, served from an origin no monitor controls. It is a
**progressive enhancement** on the M3 server-rendered dashboard.

**No cosigning in v1 (ADR-0004).** The v1 independent attestation is **OTS-anchoring observed roots** (trustless
timestamp + anti-rewrite; no monitor signing key in the trust path). Cosigning + gossip + witness endpoint are deferred
to M7. **OTS = daily per hub**, keyed by `(hub, tree_size, root)` UNIQUE (each distinct root anchored once); calendar-
attested server-side, authoritative via the user's `ots verify`; never blocks the follower.

**API contracts.** REST over stdlib `net/http`, all public GET endpoints **CORS-enabled** (`Access-Control-Allow-Origin:
*`): hubs/status (incl. coverage); latest checkpoint (+ots); `consistency?hub&from&to`; entry/range fetch;
`verify-for-me` (wraps the shared verifier, explicitly non-authoritative); the raw tlog-tiles mirror at canonical paths;
and `inclusion?iscc_id=…[&seq=]` returning a self-contained proof bundle:

```
{ checkpoint_note, treeSize, leafIndex, inclusionProof[], record_bytes,
  hub_didweb_key, ots?, other_records: [ (seq, $schema), ... ] }   // generic per-id record list (ADR-0008)
```

**Core schema (per-network DB; encodes decisions):**

```
hubs(hub_id PK, domain, origin, base_url, active, status, monitored_since_size, monitored_since_time, first_seen, last_seen)
hub_keys(hub_id, key_id, pubkey_raw, pubkey_z, revoked_at, resolved_at)        -- did:web key cache; DID doc is source of truth
checkpoints(id PK, hub_id, tree_size, root, raw, observed_at, consistent, root_rebuilt; UNIQUE(hub_id,tree_size,root))
violations(id PK, hub_id, kind, detected_at, raw_a, raw_b, proof_json)         -- kind ∈ {fork,shrink,equivocation}
tiles(hub_id, level, tile_index, width, data BLOB, is_full, sha256, updated_at, PK(hub_id,level,tile_index,width))
entry_bundles(hub_id, bundle_index, width, data BLOB, is_full, sha256, updated_at, PK(hub_id,bundle_index,width))
iscc_index(hub_id, seq PK, iscc_id BLOB, iscc_id_str, note_schema, record_sha256; INDEX(iscc_id))   -- iscc_id→seq one-to-many
follow_state(hub_id PK, last_size, frozen, last_error)
ots(id PK, hub_id, tree_size, root, status, ots_bytes, calendar_urls, stamped_at, upgraded_at, btc_height, attempts, next_retry; UNIQUE(hub_id,tree_size,root))
```

**Modules (by role, not path):** realm `registry`; `didweb` resolver + vkey/origin derivation; `logclient` follower +
verify; `store` (per-network SQLite + `SQLiteFetcher`); ISCC-ID `index`; pure `proof` (bundle + verify); `ots`; `api`;
`web` (dashboard + browser); `wasm/verify`; `metrics`; `supervisor`. The `cosign` module is **not** built in v1.

**Reuse over reimplement:** `transparency-dev/tessera` (`client`, `api`, `api/layout`, `fsck`),
`transparency-dev/merkle` (`rfc6962`, `proof`), `transparency-dev/formats`, `golang.org/x/mod/sumdb/note`,
`nbd-wtf/opentimestamps`, `modernc.org/sqlite`. Go 1.24, `CGO_ENABLED=0`.

## Testing Decisions

**What makes a good test here:** drive the system through its *external* boundary and assert on *observable outputs* —
never on follower internals. The monitor's only contact with the world is **outbound HTTP fetches**; its outputs are the
SQLite state, `/metrics`, structured logs, the REST/proof-bundle responses, and the verifier-app verdicts. Tests feed
fixtures in at the fetch boundary and check those outputs.

**Seams (fewest, highest; existing preferred):**

1. **One primary integration seam — the outbound-fetch boundary.** For hub logs, reuse tessera's **existing
   `client.Fetcher`** interface; for the realm registry and did:web docs, inject the `*http.Client` (custom
   `RoundTripper`) at the highest point. Drive the whole follower → store → metrics/API pipeline against fixtures and
   assert on: accepted `checkpoints`, `violations` rows + `frozen`, `monitored_since`, `/metrics` series, log events,
   served bundles, and the mirror. Restart-survival = point a fresh instance at the same DB file.
2. **Pure-function units (inherent seams, no I/O):** `proof/verify` and `didweb` (`origin()`, `verifierKey()`) — table-
   driven golden-vector tests, with the two live testnet hubs (`sb0.iscc.id/log`, `sb1.amlet.id/log`) as golden vectors
   (`derive_vkey.py` is the reference).
3. **External oracle gates (CI):** `notecheck` for signature-verdict parity; `fsck` via the `SQLiteFetcher` for
   structural root-rebuild; the continuous inclusion cross-check vs the hub's `IsccLogInclusionProof`.

**Modules tested:** the follower/store pipeline end-to-end through seam 1; `proof/verify`, `didweb`, `index` codec and
the ISCC-ID one-to-many behavior through unit seams; the API/bundle shape through seam 1's HTTP outputs; WASM↔server
verifier parity (identical vectors → identical verdicts, plus verifier-artifact hash matches the published value).

**Specific behaviors that must have tests:** the three violation drills (synthetic fork / shrink / equivocation →
correct `violations.kind`, freeze, one alert, other hubs unaffected, evidence survives restart); `unresolvable` and
`unverified` status (keep mirroring, no advance); did:web rotation (key change does not break status); coverage/cold
start (late start backfills + records `monitored_since`, guarantees scoped); schema-agnostic indexing (an unknown
`note.$schema` is indexed and proof-able and does not gate verification); partial-tile discipline (partials re-fetched,
only `width==256` marked full); OTS daily + UNIQUE dedupe of unchanged roots.

**Prior art:** the iscc-hub conformance tooling (`conformance/runfsck`, `conformance/notecheck`) is the model for the
oracle gates and ships a reusable leaf hasher; `tessera/client/client_test.go` and `fetcher.go` show fetcher-driven
tests and the interface our `SQLiteFetcher` mirrors; `checkpoint_note.py` is the byte-exact wire-format reference;
`log_tree.py` + `schema.py` are the inclusion-proof oracle shape.

## Out of Scope

- **Cosigning, multi-monitor gossip, witness endpoint** (`tlog-witness add-checkpoint`), cross-monitor split-view
  comparison — all deferred to **M7** (ADR-0004).
- **Resolution projections** (current owner, current gateway URL) — deferred; enabled but not built (ADR-0008).
- **Domain-compromise detection** — out of scope by axiom (ADR-0009); the monitor does not try to distinguish legitimate
  rotation from a domain takeover.
- **On-chain / ownerless registry** for membership or key commitments — a future resolver, not v1 (ADR-0009).
- **HA multi-signer hubs** — not supported (the spec mandates one writer per log).
- **Server-side authoritative Bitcoin verification** — calendar-attested only; authoritative verification is the user
  running `ots verify` (ADR-0004).
- **Honoring a hub's local redaction** — the monitor preserves the original declaration by design (it never sees
  redactions; ADR-0008).
- **Creating the GitHub repo / publishing this PRD to a tracker** — handled separately; this doc is the local artifact.

## Further Notes

**Cross-repo dependencies (iscc-hub side; see `.claude/iscc-hub-side-changes.md`).** v1 depends on three edits the
iscc-hub/spec maintainers must land — and mainnet has **not launched**, so this is the window before lock-in: (1)
ISCC-Log §2.2 amendment to source keys from did:web; (2) realm registry becomes domains-only (drop `pubkey`, keep
`hub_id` — it is the 12-bit hub field inside every ISCC-ID); (3) make the §13 coverage registry concrete (publish
`monitored_since` per monitor/hub). The superseded `valid_from` Hub-List proposal is obsolete (rotation is did:web now).

**Verifier integrity.** The monitor can equivocate about its *own* verifier; that is why the verifier app lives on an
independent origin (`monitor.iscc.codes`), is reproducibly built, and has its hash published independently. Same-origin
SRI alone is insufficient.

**Known live data point.** `sb1.amlet.id` signs with a key its did.json matches, so it verifies as `verified` under the
did:web model. `sb0.iscc.id` verifies end-to-end. Both are golden vectors and offline fixtures (`testdata/live/`).

**Suggested milestone order (the headline-value path):** M1 read-only Monitor → M2 Aggregator (mirror + index +
proofs) → M3 Trust API + dashboard + log browser → WASM verifier upgrade → OTS / Bitcoin anchoring → (M7 deferred:
gossip + cosigning + witness). M1 is independently shippable and would already have caught the sb1 issue.
