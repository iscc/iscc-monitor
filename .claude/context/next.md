# Next Work Package

## Step: Wire the `hub_keys` did:web key cache write into `PollHub`

## Goal
Make the follower actually populate the `hub_keys` did:web key cache: on a verified, non-violation
observation, `PollHub` records the resolved hub signing key via the already-built `store.RecordHubKey`.
This connects the dangling `RecordHubKey` seam (built last step, never called) and closes one of the two
named M1 wiring gaps, advancing M1 toward complete.

## Scope
- **Create**: `internal/logclient/keyid.go` — a tiny pure helper
  `KeyIDFromVerifier(vkey string) (uint32, error)` that parses the middle `+<keyid:08x>+` field out of a
  `<name>+<keyid>+<base64>` verifier-key string (the string `ResolveVerifierKey` returns). Pure
  (`{fmt,strings,strconv}` only), golden-testable.
- **Modify**: `internal/follower/follower.go` — on the verified, non-violation path (alongside the
  `SetCoverage`/`AdvanceFollowState` writes), call `logclient.ResolveVerifierKey(ctx, fetcher, baseURL)`
  once to obtain `(vkey, didKey)`, derive `key_id` via `KeyIDFromVerifier(vkey)`, and call
  `st.RecordHubKey(ctx, store.HubKey{...})`.
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` (line 71 `HubKey` struct; `RecordHubKey`
    at line 324; `nullStringOrNil` at line 365) — the target struct/method and its field semantics.
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` (`ResolveVerifierKey`, line 93) — returns
    `(string vkey, didweb.DIDKey, error)`; the vkey is `<name>+<keyid:08x>+<base64>`.
  - `/workspace/iscc-monitor/internal/didweb/vkey.go` (`VerifierKey`, line 93) — the exact
    `"%s+%08x+%s"` format `KeyIDFromVerifier` must invert; the base64 tail is `base64.StdEncoding`,
    whose alphabet INCLUDES `+` and `/`.
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` (lines 54–59 `DIDKey` fields:
    `PublicKey`, `Multibase`, `Revoked`) — the source of the cache columns.
  - `/workspace/iscc-monitor/internal/follower/follower.go` (lines 100–121) — the verified non-violation
    branch where `SetCoverage`/`AdvanceFollowState` already live; the new write joins it.
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` (lines 1–145) — `compositeFetcher`,
    `sb0VerifiedFetcher`, `sb0ObservedAt`, `countRows` are the offline harness to assert the new row.

## Not In Scope
- **No key *reader*.** A `HubKey` lookup (e.g. `store.LookupHubKey`) is a separate later slice; this step
  only writes the cache. Do not add a reader or make verification consult the cache.
- **No stale-fixture refresh.** Do NOT touch `internal/logclient/testdata/sb1.amlet.id_did.json`,
  `.claude/derive_vkey.py`, or any `22b08f3e`→`069d0f14` sb1 vector. Refreshing the trust-root fixture
  re-arms the oracle gate and is its own step; sb0 is the verified-path fixture used here and stays valid.
- **No merkle / equivocation trigger** and **no `transparency-dev/merkle` dependency** — that is the other
  M1 slice, deferred.
- **No refactor of `AcceptCheckpoint`'s signature** to surface the key it already resolves internally.
  Calling `ResolveVerifierKey` a second time on the verified path is acceptable for v1 (same offline
  `Fetcher` seam, YAGNI); a caching fetcher is a later optimization, not this step.
- **No schema change, no `/metrics`, no structured logs, no alert transport.**

## Implementation Notes
- **Recover `key_id` from the vkey string, do not re-derive it.** `didweb.keyID` is package-private and
  `VerifierKey` formats the id as `%08x` between the two `+`. CRITICAL: the base64 tail is
  `base64.StdEncoding` whose alphabet contains `+` and `/`, so a plain `strings.Split(vkey, "+")` over-
  splits (verified: sb0's vkey is `sb0.iscc.id/log+40b74463+AaV+ivnly67...` — the tail itself has a `+`).
  Use `strings.SplitN(vkey, "+", 3)`, require `len == 3`, then `strconv.ParseUint(parts[1], 16, 32)` and
  return the `uint32`. Error (named, `%q` the input) on `len != 3` or a non-hex middle field. This mirrors
  the learning that the vkey middle `+<hex>+` field IS the signed-note keyhash (`learnings.md`,
  "Signed-note keyhash is independently checkable from the raw sig line").
- **Map `DIDKey` → `store.HubKey` exactly per the handoff:** `PublicKey`→`PubkeyRaw`,
  `Multibase`→`PubkeyZ`, `Revoked`→`Revoked`, derived id→`KeyID`, injected `observedAt`→`ResolvedAt`.
  `RecordHubKey` already maps empty `PubkeyZ`→NULL and zero `Revoked`→NULL (`nullStringOrNil`/`unixOrNil`),
  so pass the zero values through untouched.
- **Placement is load-bearing (ADR-0001/0006 + learnings).** The `RecordHubKey` call goes ONLY on the
  verified, non-violation path — the same branch as `SetCoverage`/`AdvanceFollowState`, never inside
  `freeze` and never on a non-verified verdict. A contradictory or unverified observation must not write a
  cache row (mirrors the coverage placement: "a contradictory observation never starts coverage").
- **`store` stays a leaf; the follower owns the mapping.** Build the `store.HubKey` in `follower.go`; do
  not let `store` import `logclient`/`didweb`. The follower already imports both `logclient` and `store`,
  so the dependency direction is unchanged.
- **Oracle gate is N/A this step** (`learnings.md`: plain `hub_keys` CRUD path, no
  proof/verify/didweb-derivation/merkle/fsck math changes; `go.mod`/`go.sum`/`schema.sql` stay
  byte-identical). `KeyIDFromVerifier` is a string parse, not a crypto derivation — but it MUST round-trip
  the live vkeys (golden vectors below) so it cannot silently diverge from `VerifierKey`.
- **Error handling:** a `ResolveVerifierKey` failure on a path that *already* reached `StatusVerified`
  is unexpected (the key just resolved inside `AcceptCheckpoint`); return it as a wrapped fault
  (`fmt.Errorf("follower.PollHub: hub %d: cache hub key: %w", hubID, err)`) consistent with the other
  `PollHub` store-write error wraps, so the fault surfaces rather than being swallowed. Do not `t.Skip`,
  swallow, or `//nolint` it.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestKeyIDFromVerifier ./internal/logclient` passes with golden vectors:
  `KeyIDFromVerifier("sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5") ==
  0x40b74463` (note the `+` inside the base64 tail) and
  `KeyIDFromVerifier("sb1.amlet.id/log+069d0f14+AW9UGZSxDvYFeewtbNU74zEMv12ChQPcuE4veN80nNtb") ==
  0x069d0f14`; a malformed input (e.g. `"no-plus-fields"` or `"a+zzzz+b"`) returns a non-nil error.
- `go test -run TestPollHub ./internal/follower` passes — add/extend a test asserting that after a
  verified `PollHub` over `sb0VerifiedFetcher`, `countRows(t, dbPath, "hub_keys") == 1`, the row's
  `key_id` equals sb0's `0x40b74463` (sb0's signed-note keyhash), and `pubkey_raw` reads back as 32 bytes.
- `go test -run "TestPollHubFork|TestPollHubShrink|TestPollHubUnverifiedDoesNotAdvance"
  ./internal/follower` still passes AND the freeze/unverified cases assert
  `countRows(t, dbPath, "hub_keys") == 0` (no cache write on a contradictory or unverified observation).
- `go list -deps ./internal/store | grep '^github.com/iscc/iscc-monitor'` shows only the self line
  (store stays a leaf; no `logclient`/`didweb` leaked by the wiring).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dependency change).

## Done When
`PollHub` writes exactly one `hub_keys` row (with the correct `key_id` and 32-byte `pubkey_raw`) on a
verified non-violation observation and zero rows on freeze/unverified paths, `KeyIDFromVerifier`
round-trips both live vkeys, and all Verification checks pass with `mise run check` green.
