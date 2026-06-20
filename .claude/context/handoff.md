# Handoff

## 2026-06-20 — Single-poll follower: wire fetch → accept → record → advance for one hub

**Done:** Added the new `internal/follower` package with `PollHub`, the first real caller that
composes the M1 verification chain (`FetchCheckpoint` → `AcceptCheckpoint`) with the store CRUD
(`RecordCheckpoint` → `AdvanceFollowState`) for one hub, one observation. It persists and advances
the follow cursor **only** on `StatusVerified`; the other three verdicts are returned to the caller
without touching the store. Purely additive — no existing source file changed.

**Files changed:**
- `internal/follower/follower.go` (new): `PollHub(ctx, *store.Store, logclient.Fetcher, hubID,
  baseURL, observedAt)` — fetch (transport error → wrapped, persist nothing); accept (checks `err`
  before status, so a verified-but-garbled body's non-nil err short-circuits before persistence);
  on `StatusVerified` build `store.CheckpointRecord` (`info.Root[:]` → `[]byte`, injected
  `observedAt`), `RecordCheckpoint`, then `AdvanceFollowState`. File docstring documents the
  record-only-on-verified choice and the follower → {logclient, store} direction.
- `internal/follower/follower_test.go` (new): fixture-driven seam tests via an in-test
  `compositeFetcher` that routes on the URL (sb0 did.json for `*did.json`, sb0 checkpoint
  otherwise). Verified path asserts `FollowState.LastSize == 10183`; non-advancing path advertises
  a mismatching key and asserts `LastSize == 0`.

**Verification:** `mise run check` → green (build + vet + test all exit 0, go1.24).
- [x] `gofmt -l .` → empty.
- [x] `go test -count=1 -run TestPollHub ./internal/follower` → PASS (both subtests).
- [x] Verified path: composite fetcher (sb0 checkpoint + sb0 did.json), `baseURL=
  "https://sb0.iscc.id"`, `observedAt` 2026-06-20 (in sb0's unconstrained window) →
  `StatusVerified`, `FollowState(ctx, hubID).LastSize == 10183`.
- [x] Non-advancing path: did.json serves the mismatching sb1 multibase
  `z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk` → `StatusUnverified`, `LastSize == 0`.
- [x] Store-leaf: `go list -deps ./internal/store` shows only `internal/store` itself, no other
  iscc-monitor internal dep; `internal/follower` depends on `logclient` + `store` (+ transitive
  `didweb`), confirming the direction follower → {logclient, store}, never the reverse.
- [x] Gate integrity: no `//nolint`, `t.Skip`, build-tag exclusions, deleted tests, or swallowed
  errors in the new files.

**Conformance/oracle gate:** N/A this step (correctly). The diff is composition-only wiring — it
touches no signature verification, RFC-6962/Merkle, proof code, `internal/didweb`, or split-view
logic. The pure verify chain (`AcceptCheckpoint`) is reused unchanged; `derive_vkey.py` vectors and
WASM purity are untouched (no pure package modified).

**Next:** The poll loop / single-writer goroutine wrapper that calls `PollHub` on a cadence and owns
all writes per network DB (ADR-0005/0007), OR the `hub_keys` did:web cache write — which also needs
the stale sb1 `did.json` fixture + `derive_vkey.py` `HUBS` refreshed to the current sb1 key
`069d0f14` (the committed `sb1.amlet.id_did.json` is still the pre-rotation `22b08f3e`). The
consistency check (fork/shrink/equivocation over `transparency-dev/merkle`) and the freeze/alert path
are the other independent follower steps; each is its own ≤3-file slice. When the consistency/freeze
or `hub_keys`/oracle code lands, the `derive_vkey.py` parity + conformance gates re-apply.

**Notes:**
- **`PollHub` returns `StatusUnverified` (the zero Status) as the placeholder verdict on a
  transport/garbled-body fault**, paired with a non-nil error. Callers must check `err` first
  (mirroring the `AcceptCheckpoint` contract) — the Status is meaningless when err != nil. This is
  the cleanest single return shape; an alternative would be a separate "fault" sentinel, but the
  err-before-status discipline is already the established convention here.
- **`sb1Multibase` value differs across packages but both are valid negatives.** I used
  `z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk` exactly as `next.md` line 114 prescribes;
  `accept_test.go`'s `sb1Multibase` is a different multibase (`z6MkiNWE7CmYP2bSeYZ3KvHRVoKsAvfY…`).
  Both are real Ed25519 keys that do not sign the sb0 checkpoint, so both force `StatusUnverified`;
  no conflict, just a per-test constant.
- **Record-only-on-verified is a deliberate, documented choice** (file docstring), not a shortcut:
  non-verified verdicts carry a zero `CheckpointInfo` with no trustworthy `(size, root)`. A later
  step may want to record non-verified observations as evidence of an internally-broken hub — the
  docstring flags this for revisiting.
- Per `next.md` scope, left `checkpoints.consistent`/`root_rebuilt` NULL (no consistency check), did
  not read/honor `FollowState.Frozen`, wrote no `hub_keys` cache, and touched no schema/coverage/
  logs/metrics/`cmd`.
