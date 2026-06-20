# Next Work Package

## Step: Wire the hub_keys cache-hit fast path into `cacheHubKey` (skip the 2nd did.json fetch)

## Goal
Stop the verified `PollHub` path from resolving did.json twice per poll. Recover the signed-note
key id from the raw checkpoint (no fetch), consult `store.LookupHubKey(hubID, keyID)`, and on a cache
hit refresh the cached row in place instead of calling `ResolveVerifierKey` a second time. This is the
cheapest unblocked M1 slice — both pure prerequisites (`KeyIDFromCheckpoint`, `LookupHubKey`) already
exist — and it closes the "fetches did.json twice" wart recorded in `learnings.md` and `state.md`.

## Scope
- **Modify**: `internal/follower/follower.go` (the only production file — change `cacheHubKey`, and
  thread the already-fetched `raw` checkpoint bytes into it from `PollHub`).
- **Modify (test)**: `internal/follower/follower_test.go` (add a test proving the second verified poll
  does NOT re-fetch did.json for the cache write; tests/docs don't count against the 3-file budget).
- **Reference** (read for context, do not modify):
  - `/workspace/iscc-monitor/internal/logclient/checkpointkey.go` — `KeyIDFromCheckpoint(raw) (name, keyID, err)`.
  - `/workspace/iscc-monitor/internal/logclient/keyid.go` — `KeyIDFromVerifier(vkey)` (the resolve-path twin).
  - `/workspace/iscc-monitor/internal/logclient/origin.go` — `Origin(baseURL)` (the exported `<domain>/log` deriver).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `LookupHubKey` / `RecordHubKey` / `HubKey`.
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — confirms the FIRST resolve (inside
    `AcceptCheckpoint`) is unavoidable: it drives the `ValidAt` window check, which the cache can't
    reconstruct. Only the SECOND resolve (in `cacheHubKey`) is the one being skipped.

## Not In Scope
- **Do NOT add a `hub_keys` validity-window schema column** (`valid_from`/`valid_until`). The cached row
  still carries only `revoked_at`, so a window re-check from the cache alone stays impossible — that is
  fine here because `AcceptCheckpoint`'s first resolve already did the `ValidAt` check this poll. A
  schema step is a *separate, later* slice if a fully cache-only fast path is ever wanted.
- **Do NOT touch `AcceptCheckpoint` / `ResolveVerifierKey`** to thread the resolved key out of the first
  resolve. That refactor (eliminating the first redundant work) is a different, larger change; this step
  only removes the *second* resolve via the cache.
- **Do NOT** start the equivocation/merkle trigger, structured logs, `/metrics`, or the real alert
  transport — each is its own later step.
- **Do NOT** change `store` (no new method needed — `LookupHubKey` + `RecordHubKey` suffice). Keep
  `store` a leaf (no `logclient` import) and `schema.sql` byte-identical.

## Implementation Notes
- Thread the raw checkpoint bytes into the cache step. `PollHub` already has `raw` in scope; change
  `cacheHubKey(ctx, st, fetcher, hubID, baseURL, observedAt)` to also take `raw []byte` and pass it at
  the single call site (line ~124).
- New `cacheHubKey` shape (keep it short; the existing resolve path stays as the fallback, unchanged):
  1. `name, keyID, err := logclient.KeyIDFromCheckpoint(raw)`. On error, **fall through to the existing
     resolve path** (a garbled note here is unexpected on a just-verified checkpoint, but the resolve
     path is the safe superset — do not hard-fail the poll on a key-id-recovery miss).
  2. Compute the expected origin via `logclient.Origin(baseURL)` (propagate a real error). Assert
     `name == expectedOrigin`. A mismatch means the cached key id would key on the wrong identity —
     treat it as a **cache miss** and fall through to `ResolveVerifierKey`. This guards the
     **Correctness rule: origin = `<domain>/log`, never the bare domain** — the signed-note `name` must
     equal the hub's origin or the lookup is meaningless. (Live sb0 signs `sb0.iscc.id/log`, which is
     exactly `Origin("https://sb0.iscc.id")`, so the hit path fires for the real fixture.)
  3. `cached, found, err := st.LookupHubKey(ctx, hubID, keyID)`; propagate a real query error with a wrap.
  4. **Cache hit** (`found`): refresh the row in place via `st.RecordHubKey`, reusing the cached fields
     (`PubkeyRaw`, `PubkeyZ`, `Revoked`) and bumping `ResolvedAt: observedAt`. Do NOT call
     `ResolveVerifierKey`. `RecordHubKey`'s guarded UPDATE keeps it a single row (count stays 1).
  5. **Cache miss** (`!found`, or any fall-through above): the current behavior — `ResolveVerifierKey`
     → `KeyIDFromVerifier` → `RecordHubKey` — unchanged, so the first verified poll still populates the
     cache exactly as today.
- Keep the production import set as-is (`{context, fmt, logclient, store, time}`); no new imports.
- Edge case from `learnings.md` (hub_keys cache wiring): `cacheHubKey` runs ONLY on the verified,
  non-violation path; the fork/shrink/unverified tests already assert `hub_keys` count == 0, and the
  fast path is not reached on those paths. Do not move the call site or touch `freeze`.
- Edge case: a `RecordHubKey` failure on the hit path is still a genuine fault — wrap and return it
  (`cache hub key: %w`), never swallow (mirrors the existing resolve-path error handling).
- Test design (seam-based, observable outputs only — never follower internals): use a Fetcher that
  **counts did.json fetches**. Extend the test's `compositeFetcher` (or wrap it) to tally how many times
  a `did.json` URL is fetched. First `PollHub` (cold cache) fetches did.json for BOTH `AcceptCheckpoint`
  and the miss-path cache write. Second `PollHub` (warm cache) fetches did.json ONLY for
  `AcceptCheckpoint` → assert the did.json fetch count grew by exactly 1 between the two polls, not 2.
  Re-assert the standing invariants: `hub_keys` count stays 1, `key_id == 0x40b74463`, 32-byte pubkey.
  The existing `TestPollHubVerifiedAdvances` "refresh in place → still 1 row" assertion must keep passing
  unchanged.

## Verification
- `mise run check` is green (build + vet + test, all packages).
- `gofmt -l internal/follower/follower.go internal/follower/follower_test.go` is empty.
- `go test -run TestPollHub ./internal/follower` passes (all existing follower tests — fork, shrink,
  unverified, verified-advances — plus the new cache-hit-skips-fetch test).
- The new test asserts: across two verified polls of the same hub, the did.json fetch count rises by
  exactly 1 on the SECOND poll (only `AcceptCheckpoint`'s resolve), proving `cacheHubKey` skipped its
  own resolve on the cache hit.
- `hub_keys` row count stays exactly 1 after the second poll with `key_id == 0x40b74463` and a 32-byte
  `pubkey_raw` (the fast path refreshes in place, never duplicates or drops the key).
- `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no schema/dep change).

## Done When
`cacheHubKey` consults `LookupHubKey(hubID, keyID)` — with `keyID` recovered from the raw checkpoint and
guarded by `name == Origin(baseURL)` — and, on a cache hit, refreshes the cached row WITHOUT a second
`ResolveVerifierKey`, proven by the fetch-count test, with all Verification criteria passing and no
`store`/schema/dep changes.
