## 2026-06-23 — Tie the dossier §3 observed-time to the accepted `last_size` checkpoint row (frozen-hub honesty fix)

**Done:** Changed the §3 `observed_at` correlated subselect in `store.ListHubs` so it reads the
`observed_at` of the checkpoint row whose `tree_size = f.last_size` (the accepted size §3 renders),
instead of the newest-by-`tree_size` row. A frozen hub no longer pairs its accepted size with a
rejected (higher-tree-size) checkpoint's timestamp. Renderer is unchanged — the fix is store-side.

**Files changed:**
- `internal/store/hubs.go`: subselect now `(SELECT c.observed_at FROM checkpoints c WHERE c.hub_id =
  h.hub_id AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1)`; `id DESC LIMIT 1` keeps a
  re-observed same-size row deterministic. Rewrote the `CheckpointObserved` doc comment and the
  in-query comment block to describe the "tied to the accepted size" semantics (evergreen).
- `internal/store/hubs_test.go`: added `TestListHubsFrozenObservedTracksAcceptedSize` — seeds an
  accepted checkpoint (`TreeSize 100 @ tAccepted`) via `AdvanceAccepted`, then a rejected higher-size
  row (`TreeSize 200 @ tRejected`) via `RecordCheckpoint` + `Freeze` (no `last_size` advance, mirroring
  `follower.freeze`), and asserts `CheckpointObserved == tAccepted` (plus `Frozen` true, `LastSize` 100).

**Verification:** `mise run check` → green (build + vet + test across all 30 packages; `gofmt -l .`
empty). Per-criterion:
- [x] `go test -count=1 -run TestListHubs ./internal/store` passes — existing
  `TestListHubsCheckpointAndAnchorHeight` (verified-hub size 42 = last_size 42; bare-hub zero) unchanged
  and green; `TestListHubsAnchorStatus` unchanged; the new frozen-hub test passes.
- [x] New frozen test mutation-proven load-bearing: reverting the subselect to
  `ORDER BY c.tree_size DESC, c.id DESC LIMIT 1` (dropping `AND c.tree_size = f.last_size`) makes it
  FAIL — it reports `tRejected` (`2023-11-15 00:59:59`) instead of `tAccepted` (`2023-11-14 22:13:20`);
  `TestListHubsCheckpointAndAnchorHeight` still passed under the mutation (verified-path no-regression).
  Restore confirmed clean (`go.mod`/`go.sum` byte-unchanged, subselect back in place).
- [x] Store leaf purity intact: `go list -deps ./internal/store | grep '^net/http$'` empty.
- [x] No schema/migration/`user_version` change (query-shape change inside the existing statement only).

**Oracle/conformance gate:** N/A — touches no signature / RFC-6962 / Merkle / did:web / proof path; it
is a pure `checkpoints`/`follow_state` read-shape change. (Confirmed by inspection; `next.md` calls this
N/A.)

**Next:** This closes the §3 size/time-decouple `normal`. The two remaining dossier/realm-index honesty
`normal`s are both DESIGN-BLOCKED (the §1 "Key resolved from did:web:…" unconditional-wording-vs-
`unresolvable` nit and the realm-index `/` per-hub-vs-per-checkpoint Anchor honesty), per their issues +
`learnings/dossier.md`/`learnings/store.md` — neither is code-closable without a design pass, so do not
pull them as an advance. `define-next` should pull from the remaining open `normal` backlog after the
4 stale M-API entries are pruned (bookkeeping, not advance work — see Notes); the two `low`
hardening follow-ups (guard-hoist ahead of the baseline DDL; `seq` ordering index) remain the natural
fold-ins WHEN `Open`/the migration runner or the dashboard-recent path is next touched.

**Notes:**
- The §3 subselect references `f.last_size` from the already-joined `follow_state` LEFT JOIN (alias `f`).
  NULL-safety holds three ways: a never-polled hub (`f.last_size` NULL → `c.tree_size = NULL` matches no
  row → SQL NULL → `observedAt.Valid == false` → zero `time.Time` → renderer "observed time unknown"),
  a hub whose accepted size has no recorded checkpoint row, and a checkpoint row with a NULL
  `observed_at`. The existing bare-hub assertion (`CheckpointObserved.IsZero()`) still holds and pins
  this. Verified-hub path is unchanged (accepted size == the accepted row's `tree_size`, same row picked).
- The 4 stale M-API `normal` issue entries (phantom `verify` `index` param, `/checkpoint` media-type,
  `/healthz` 200/503) are ALREADY satisfied in the served `internal/openapi/openapi.{yaml,json}` per the
  prior define-next read — they await a prune by `update-state`/`review`, not an advance code change.
- Pre-existing `.claude/context/target.md` working-tree mod (from steer `d2f259e`) is still uncommitted
  and is NOT mine to commit (advance commits only implementation/test files + handoff.md); left for
  `update-state`/`steer`. My commit excludes it.
