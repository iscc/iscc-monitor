<!-- assessed-at: 7a7a3fac90a3368e6287b126cbeedfb2d687e3d1 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config + `cmd/` binary + coverage + `hub_keys` cache (write + read) + **now the reader→follower wiring is live** (verified poll consults the cache and skips the 2nd did.json fetch); equivocation trigger + structured logs + `/metrics` + real alert transport still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, and the full `hub_keys`
did:web key cache have landed — and as of this slice the cache is finally **consulted on the verified
path**: `cacheHubKeyFast` recovers `(name, keyID)` from the raw checkpoint via the pure
`KeyIDFromCheckpoint`, guards `name == Origin(baseURL)`, calls `LookupHubKey`, and on a hit refreshes
the row in place WITHOUT a second `ResolveVerifierKey`/did.json fetch (cold poll still resolves to
populate the row). What remains to close M1: the merkle-backed **equivocation** trigger (the third
self-consistency trigger), structured logs, `/metrics`, and a real alert transport (still a stderr
placeholder).

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache write **and read**
**and now reader→follower wiring**; equivocation trigger + logs/metrics/alert transport still missing)

- Verified present (incremental re-check from prior assessment at `daef4f1`; the diff
  `daef4f1..HEAD` touches **only** `internal/follower/follower.go` + `follower_test.go` plus context
  files — confirmed `go.mod`/`go.sum` + `schema.sql` byte-identical, fixtures unchanged, no new reuse
  imports. All non-follower sections re-confirmed unchanged and carried forward):
  - **NEW — reader→follower wiring landed** (`internal/follower/follower.go`): the prior assessment's
    "cheapest open slice" is closed. `cacheHubKey` now splits into:
    - `cacheHubKeyFast(ctx, st, hubID, baseURL, raw, observedAt) (hit bool, err)` — the fetch-free
      path: `logclient.KeyIDFromCheckpoint(raw) → (name, keyID)`; on a recovery error it falls through
      (returns `hit=false, nil` — a garbled note on a just-verified checkpoint is unexpected but the
      resolve path is the safe superset). It then derives `expectedOrigin = logclient.Origin(baseURL)`
      and **guards `name == expectedOrigin`** (a mismatch is treated as a cache miss, so the key id is
      only trusted when keyed on the hub's own identity). On `LookupHubKey` hit it refreshes the row in
      place via `RecordHubKey` (reusing cached `PubkeyRaw/PubkeyZ/Revoked`, bumping `ResolvedAt`) and
      returns `hit=true` — **no second `ResolveVerifierKey`/did.json fetch**.
    - `cacheHubKeyResolve(...)` — the unchanged cold-cache fallback: `ResolveVerifierKey` →
      `KeyIDFromVerifier` → `RecordHubKey`. The first verified poll always takes this (cache cold),
      populating the row so later polls hit the fast path.
    Import block unchanged `{context, fmt, logclient, store, time}` — store stays a leaf (no
    net/http in its closure). **+1 `func Test`** (`TestPollHubCacheHitSkipsDidFetch`): the review
    handoff records cold poll = 2 did.json fetches, warm poll = +1 (only `AcceptCheckpoint`'s fetch),
    proving `cacheHubKey` skipped its own resolve on the hit; `hub_keys` stays exactly 1 row
    (`key_id == 0x40b74463`, 32-byte `pubkey_raw`). A broken guard/lookup would fall through to +2 and
    fail the test (non-vacuous).
  - **Schema caveat carried forward** (handoff): the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check **from the cache alone** is
    still not possible — a `hub_keys` schema step would have to precede a fully window-honoring fast
    path. The current fast path refreshes/reuses the key but does not re-check the validity window from
    cache; this is a known, documented limitation, not a green-but-wrong path.
  - **`KeyIDFromCheckpoint`** (`internal/logclient/checkpointkey.go`) — pure raw-checkpoint key-id
    reader: `note.Open(raw, note.VerifierList())` (empty list) + `errors.As` on `*UnverifiedNoteError`,
    reading `ue.Note.UnverifiedSigs[0].{Name,Hash}`; malformed input wrapped, never panicked. Twin of
    `keyid.go`'s `KeyIDFromVerifier`. 3 tests (golden `name=="sb0.iscc.id/log"`, `keyID==0x40b74463`;
    cross-check `==KeyIDFromVerifier(sb0VKey)`; garbled-input non-nil error).
  - **`hub_keys` key reader** (`internal/store/checkpoints.go`): `LookupHubKey(ctx, hubID, keyID
    uint32) (HubKey, bool, error)` — column-by-column inverse of `RecordHubKey`; "absent row → zero +
    found=false + nil err" convention; `uint32` reconstructed from the lookup arg (no truncation/sign
    risk off the signed `int64` column). 4 tests.
  - **`hub_keys` cache write wired into `PollHub`** (`cacheHubKeyResolve`): on the verified,
    non-violation path it resolves + recovers key id + upserts `store.HubKey`; a verified poll writes
    exactly 1 row, a second stays at 1, fork/shrink/unverified write 0.
  - **coverage tracking** (`internal/store/checkpoints.go` + `follower.go`): set-once
    `monitored_since_{size,time}` (ADR-0001), wired between `RecordCheckpoint` and `AdvanceFollowState`,
    outside `freeze`. Store + follower assertions.
  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): wired M1 entrypoint; `alert(hubID,
    kind)` is a documented stderr placeholder (confirmed: `fmt.Fprintln(os.Stderr, ...)`). 1 `func Test`.
  - **exported `logclient.Origin`** + golden `TestOrigin`; **config loader leaf** (4 tests, `Frozen >=
    Normal` cross-check); **realm-registry parser** (3 tests, domains-only, fails closed on URLs);
    **poll-loop cadence** (`loop.go` + `loop_test.go`, single-writer, pure `due`, frozen hubs re-poll
    at `Frozen`); **freeze + alert-once** (`checkConsistency` shrink-then-fork; `freeze` →
    `RecordViolation` + evidence `RecordCheckpoint` + `Freeze` + one-shot `AlertFunc`, gated on
    `!wasFrozen` — re-detection records again but never re-alerts).
  - `internal/store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`/
    `LookupHubKey`; `CheckpointAt` recovers the prior accepted root via `LIMIT 1` (rowid order).
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind`;
    `transparency-dev/merkle` is **prose-only** (a comment in `consistency.go`; not in go.mod, not
    imported — re-confirmed).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (incl. `hub_keys`).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey}.go` —
    transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`, pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`/`Origin()`, pure
    `KeyIDFromVerifier` (vkey string) + pure `KeyIDFromCheckpoint` (raw bytes).
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 23 (`logclient`) + 31 (`store`) + 8 (`follower`: 5 `follower_test.go`
    + 3 `loop_test.go`) + 3 (`registry`) + 4 (`config`) + 1 (`cmd/iscc-monitor`) = **79 `func Test`**
    (follower +1 for `TestPollHubCacheHitSkipsDidFetch`).

- Missing (the remaining M1 connective tissue):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check,
    golden-vector parity, `notecheck` in CI). More than one verifiable slice — needs decomposing. This
    is now the largest open M1 gap.
  - **Structured logs** — no `slog` anywhere in `internal`/`cmd` (confirmed). Two stderr placeholders
    remain: `alert` in `cmd/iscc-monitor/main.go` and the per-tick swallowed error in `loop.go`
    (commented "surfaces via the store/metrics later").
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server in `internal`/`cmd`
    (the one `metric` grep hit is a comment word in `loop.go`).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder; production delivery
    (email/webhook) is unbuilt.

- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed for
  the equivocation trigger + M2). **Known stale-fixture drift, still not acted on:** the
  `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py`
  still carry sb1's PRE-rotation key (`z6MkiNW…` / `22b08f3e`); the live sb1 signer of the captured
  checkpoint is `069d0f14`. `verify_test.go` documents both keys and tests the stale-key mismatch, so the
  drift is *captured in prose/tests* (not green-but-wrong) — but the did.json fixtures themselves remain
  stale. The refresh remains its own trust-root step (re-arms the oracle gate).

- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`,
  `logclient/checkpointkey.go`), `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed
  zero non-comment import statements + zero go.mod entries): `transparency-dev/*`
  (merkle/tessera/formats), `nbd-wtf/opentimestamps`.

- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart, through the poll loop). Coverage (`monitored_since`) — **tracked** and
  asserted set-once. The **third trigger, equivocation, is not met** (absent), so the M1 Verify line is
  not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "Wire the hub_keys cache-hit fast
  path into `cacheHubKey`", verdict **PASS / CONTINUE**) records the gate green at HEAD `7a7a3fa`:
  `mise run check` green (build + vet + test, all 7 packages ok, re-run uncached), `gofmt -l .` empty,
  `git diff --quiet HEAD~1..HEAD -- go.mod go.sum schema.sql` exit 0 (confirmed here:
  `daef4f1..HEAD` touches only `internal/follower/*` + context files). Oracle/conformance gate
  correctly **N/A** this step (no `verify`/`proof`/`merkle`/`didweb`/`fsck` path; the fast path reads an
  already-decoded keyhash and reuses cached bytes).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop`; tree clean at HEAD `7a7a3fa`. **No `.github/workflows/` — no CI configured.** When
  CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell
  out the future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. The reader→follower wiring slice is **done**; remaining candidate slices, in rough order:
1. **Merkle-backed equivocation trigger** (RFC-6962 consistency-proof failure across growing sizes): a
   third branch in the `checkConsistency` seam returning `ViolationEquivocation` + real `ProofJSON`,
   compared against the prior *accepted* root (`CheckpointAt` `LIMIT 1` rowid-order). Needs
   `transparency-dev/merkle` (new dep) + tile fixtures and **trips the conformance/oracle gate**
   (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check, golden-vector parity, `notecheck`
   in CI). Needs decomposing into multiple slices. This is now the critical-path M1 gap.
2. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`
   HUBS) — its own trust-root step that re-arms the oracle gate.
3. **Optional `hub_keys` schema step** for `valid_from`/`valid_until` if a fully CID-1.0
   validity-window-honoring fast path is wanted (the current fast path reuses/refreshes the key but
   can't re-check the window from cache alone).
4. The lighter remaining M1 gaps — structured logs (`slog`, replacing the two stderr placeholders),
   `/metrics`, real alert transport — then complete M1's Verify criteria.
No CI is configured: flag for whoever sets up the workflow (it becomes load-bearing the moment the
merkle path or a fixture refresh lands).
