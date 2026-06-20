# Handoff

## 2026-06-20 — Wire the `hub_keys` did:web key cache write into `PollHub`

**Done:** Connected the dangling `store.RecordHubKey` seam: on the verified, non-violation `PollHub`
path (alongside `SetCoverage`/`AdvanceFollowState`) the follower now resolves the hub's did:web key,
recovers its key id from the verifier-key string, and upserts the `hub_keys` cache row. Added a tiny
pure `logclient.KeyIDFromVerifier` helper that inverts `didweb.VerifierKey`'s `<name>+<keyid:08x>+
<base64>` format, golden-tested against both live vkeys.

**Files changed:**
- `internal/logclient/keyid.go` (new): `KeyIDFromVerifier(vkey string) (uint32, error)` — `SplitN(vkey,
  "+", 3)`, require 3 fields, `strconv.ParseUint(parts[1], 16, 32)`. Pure (`{fmt,strconv,strings}`).
  The `SplitN` n=3 is load-bearing: sb0's base64 tail itself contains a `+` (`AaV+ivnly67…`), so a plain
  `Split` would over-split and read the wrong middle field.
- `internal/logclient/keyid_test.go` (new): golden vectors (sb0 `0x40b74463` incl. the `+`-in-tail case,
  sb1 `0x069d0f14`) + 5 malformed-input error cases.
- `internal/follower/follower.go`: new `cacheHubKey` helper called only after `AdvanceFollowState` on the
  verified non-violation path. Re-runs `ResolveVerifierKey`, derives `key_id` via `KeyIDFromVerifier`,
  maps `DIDKey`→`store.HubKey` (`PublicKey`→`PubkeyRaw`, `Multibase`→`PubkeyZ`, `Revoked`→`Revoked`,
  injected `observedAt`→`ResolvedAt`), calls `st.RecordHubKey`. A failure here is wrapped as a fault
  (`follower.PollHub: hub %d: cache hub key: %w`) and surfaced, never swallowed. Doc comment updated.
- `internal/follower/follower_test.go`: `openTemp` now returns `(*store.Store, path)`; added `readHubKey`
  inspector helper; `TestPollHubVerifiedAdvances` asserts `hub_keys==1`, `key_id==0x40b74463`,
  `len(pubkey_raw)==32`, and that a second poll refreshes in place (still 1 row); fork/shrink/unverified
  tests assert `hub_keys==0` (no cache write on contradictory/unverified observations).

**Verification:** `mise run check` → green (build / vet / test all ok, 7 packages). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -run TestKeyIDFromVerifier ./internal/logclient` PASS — both live vkeys round-trip
  (incl. sb0's `+`-bearing tail); malformed inputs error.
- [x] `go test -run TestPollHub ./internal/follower` PASS — verified poll writes exactly one `hub_keys`
  row, `key_id==0x40b74463`, `pubkey_raw` reads back as 32 bytes; second poll stays at 1 (refresh).
- [x] `go test -run "TestPollHubFork|TestPollHubShrink|TestPollHubUnverifiedDoesNotAdvance"
  ./internal/follower` PASS — freeze/unverified cases assert `hub_keys==0`.
- [x] `go list -deps ./internal/store | grep '^github.com/iscc/iscc-monitor'` → only the self line
  (store stays a leaf; no `logclient`/`didweb` leaked). No `net/http` in the store closure.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` → exit 0 (no schema/dep change).
- [x] Follower production imports unchanged: `{context fmt logclient store time}`; `keyid.go` imports
  `{fmt strconv strings}`.

**Next:** The key *reader* slice — a `store.LookupHubKey` (or similar) so verification/serving can consult
the cache. After that, the headline remaining M1 gap is the merkle-backed **equivocation** trigger
(needs `transparency-dev/merkle` + tile fixtures + a conformance/oracle package + a CI `notecheck` job),
which is more than one verifiable slice. The sb1 fixture refresh (`22b08f3e`→`069d0f14`) + `derive_vkey.py`
HUBS update remains its own trust-root step (re-arms the oracle gate); it was explicitly out of scope here
and was not touched.

**Notes:**
- Oracle/conformance gate correctly **N/A** this step: `KeyIDFromVerifier` is a string parse, not a crypto
  derivation; no proof/verify/didweb-derivation/merkle/fsck math changed; `go.mod`/`go.sum`/`schema.sql`
  byte-identical. The golden vectors still pin it to `VerifierKey`'s output so it cannot silently diverge.
- Per `next.md`'s allowance, `cacheHubKey` re-runs `ResolveVerifierKey` (a second call on the verified
  path) rather than threading the already-resolved key out of `AcceptCheckpoint` — same offline `Fetcher`
  seam, YAGNI. A caching fetcher or an `AcceptCheckpoint` signature change to surface the key is a later
  optimization, not this step (it was listed under Not In Scope).
- `openTemp`'s signature changed to return the path (test-only helper); both existing call sites updated.
  No production API changed.
