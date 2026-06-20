# Next Work Package

## Step: Single-poll follower — wire fetch → accept → record → advance for one hub

## Goal
Land the first real *caller* of the M1 verification + storage seams: a pure-wiring
`PollHub` that, for one hub, fetches its checkpoint, runs the four-way `AcceptCheckpoint`
verdict, persists the observed checkpoint, and advances the follow cursor **only** when
verified. This closes the "nothing yet consumes FetchCheckpoint/AcceptCheckpoint/the store
CRUD" gap with an end-to-end, fixture-driven slice — the backbone every later follower
concern (consistency check, freeze, coverage, metrics) hangs off.

## Goal-fit (state → target gap)
`state.md` and the handoff `**Next:**` both point at "the follower poll loop", but that
bundle (fetch + verdict→`CheckpointRecord` mapping + persistence + the `hub_keys` did:web
cache + sb1 fixture refresh + the three-trigger consistency check + freeze/alert + coverage
+ logs + `/metrics` + the goroutine/loop wrapper) spans far more than 3 files and many
distinct behaviors. The pure chain (`FetchCheckpoint`, `AcceptCheckpoint`) and the store CRUD
(`UpsertHub`/`RecordCheckpoint`/`AdvanceFollowState`) all exist and are tested, but **nothing
calls them together**. This step takes the smallest coherent slice that proves the wiring:
one observation, one hub, no loop. The consistency check, freeze, `hub_keys` cache, and the
loop each follow as their own steps.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/follower/follower.go` — the new `follower` package +
    `PollHub` (the one new non-test source file).
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` — fixture-driven seam test
    (test file, not counted against the 3-file budget).
- **Modify**: (none — purely additive)
- **Reference** (read for context; do not import the store into logclient or vice-versa):
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — `AcceptCheckpoint`, `Status`,
    `CheckpointInfo`, `Fetcher`. Note `CheckpointInfo` is the **zero value on every
    non-verified verdict**, and the err-before-status contract (lines 84-104).
  - `/workspace/iscc-monitor/internal/logclient/checkpoint.go` — `FetchCheckpoint` signature
    and the `os.ErrNotExist`-on-404 contract.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `UpsertHub`,
    `RecordCheckpoint`, `AdvanceFollowState`, `FollowState`, `CheckpointRecord` (note `Root`
    is `[]byte`, `Status` is a string, `ObservedAt time.Time` is injected).
  - `/workspace/iscc-monitor/internal/logclient/checkpoint_test.go` — the established
    httptest-fetch-then-verify-against-sb0 pattern (`TestFetchCheckpointOverHTTP`) and the
    `fakeFetcher`/`readCheckpoint`/`readFixture` helpers to mirror.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — the `t.TempDir()` store
    test style and the public-method assertion patterns (`FollowState`).

## Not In Scope
- **No poll loop, goroutine, ticker, or single-writer goroutine wrapper.** `PollHub` does
  exactly one observation per call and returns; the loop/scheduler is a later step.
- **No consistency check** (fork/shrink/equivocation), no `transparency-dev/merkle`, no
  tiles — leave `checkpoints.consistent`/`root_rebuilt` NULL.
- **No freeze/alert path**, and do not read or honor `FollowState.Frozen` yet.
- **No `hub_keys` did:web cache write**, and therefore **no sb1 `did.json` /
  `derive_vkey.py` refresh** — that is its own step. IMPORTANT: the sb1 did:web fixture key
  derives to keyhash `22b08f3e`, which does **not** match the sb1 *checkpoint* signer
  `069d0f14`, so sb1 currently resolves to `StatusUnverified` end-to-end. Drive the verified
  golden path with **sb0 only**; do not use sb1 as a verified vector here.
- **No coverage (`monitored_since`), structured logs, `/metrics`, config, realm registry, or
  `cmd/` binary.**
- Do **not** add a status column to `checkpoints` or touch `schema.sql` — `Status` rides on
  `CheckpointRecord` only.

## Implementation Notes
- New package `internal/follower` (NOT inside `store`, which must stay a leaf per learnings,
  nor inside `logclient`): it is the composition layer importing both `internal/logclient`
  and `internal/store`. The dependency direction is follower → {logclient, store}, never the
  reverse, so `net/http` never enters the store closure.
- Signature (clock injected — learnings: never `time.Now()` in this layer; `hubID` +
  `baseURL` are pre-resolved by the caller to keep the seam small):
  `func PollHub(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, observedAt time.Time) (logclient.Status, error)`.
- Sequence inside `PollHub`:
  1. `raw, err := logclient.FetchCheckpoint(ctx, fetcher, baseURL)` — on error, return it
     wrapped (`%w`) and persist nothing. A fetch fault is transport, not a four-way verdict.
  2. `status, info, err := logclient.AcceptCheckpoint(ctx, fetcher, baseURL, raw, observedAt)`
     — **check `err` first**: a verified-but-garbled body returns non-nil err alongside
     `StatusUnverified`'s zero (accept.go lines 92-100). On that error, return it wrapped and
     persist nothing.
  3. Persist only on `StatusVerified` (the non-verified verdicts carry the zero
     `CheckpointInfo`, so there is no trustworthy `(size, root)` to record). Build a
     `store.CheckpointRecord{HubID: hubID, Status: status.String(), TreeSize: info.TreeSize,
     Root: info.Root[:], Raw: raw, ObservedAt: observedAt}` (note `info.Root [32]byte` →
     `[]byte` via `info.Root[:]`) and call `st.RecordCheckpoint(ctx, rec)` (idempotent on
     re-observe). Document this "record-only-on-verified" choice in the file docstring so a
     later step can revisit recording non-verified observations.
  4. Call `st.AdvanceFollowState(ctx, hubID, info.TreeSize)` **only** on `StatusVerified`.
  5. Return `(status, nil)` for any of the four verdicts (non-verified is a *verdict*, not a
     Go error); reserve the returned error for transport/garbled-body faults.
- Test fetcher: `AcceptCheckpoint` resolves both the checkpoint *and* the did.json through
  the same `Fetcher`, and the key is origin-bound — so an httptest live host cannot serve the
  sb0-origin did.json (learnings: "httptest can prove fetch but NOT verify against a
  live-host URL"). Mirror `TestFetchCheckpointOverHTTP`: use a small in-test composite
  `Fetcher` that routes on the URL — return the sb0 `did.json` bytes when
  `strings.HasSuffix(url, "did.json")`, else the sb0 checkpoint bytes — and pass
  `baseURL="https://sb0.iscc.id"` with `observedAt` inside sb0's validity window. One
  fetcher, full fetch→verify→persist chain, no live network.
- Relevant Correctness rules (learnings.md): "verified is the only outcome that advances
  accepted state"; "did:web is the only key source → unresolvable keeps mirroring, no
  advance"; "Origin = `<domain>/log`" (reused via `FetchCheckpoint`/`AcceptCheckpoint`, never
  re-derived here); clock injection.
- `store.Store.db` is unexported, so from `internal/follower` assert via the store's public
  methods (`FollowState`); do not reach into store internals. A `checkpoints` row-count check
  belongs in the store's own in-package test, not here — for this step, asserting
  `FollowState(...).LastSize` is the load-bearing observable.
- Style: file-level docstring stating purpose; short pure functions; wrap errors with `%w`;
  no `t.Skip` / `//nolint` / swallowed errors / build tags.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...` all exit 0).
- `gofmt -l .` lists nothing.
- `go test -count=1 -run TestPollHub ./internal/follower` passes.
- Verified path: `PollHub` with the composite fetcher (sb0 checkpoint + sb0 did.json),
  `baseURL="https://sb0.iscc.id"`, and an `observedAt` inside sb0's validity window returns
  `StatusVerified` and afterward `st.FollowState(ctx, hubID).LastSize == 10183`.
- Non-advancing path: `PollHub` where the did.json serves a mismatching key (e.g. the sb1
  multibase `z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk`) returns `StatusUnverified`
  and advances nothing — `st.FollowState(ctx, hubID).LastSize == 0` afterward.
- `go list -deps ./internal/store` shows **no** `github.com/iscc/iscc-monitor/internal`
  dependency (store stays a leaf; the follower depends on store, never the reverse).

## Done When
`PollHub` exists in the new `internal/follower` package, drives one observation end-to-end
against the sb0 fixtures (verified → recorded + cursor advanced; non-verified → never
advanced), keeps store a leaf, and every Verification criterion passes with `mise run check`
green.
