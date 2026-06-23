## 2026-06-23 — Finish the dossier §3 honesty fix — `ORDER BY c.id ASC` so a same-size fork reads the accepted row's time

**Done:** Changed the §3 `observed_at` correlated subselect in `store.ListHubs` from `ORDER BY c.id
DESC` → `ORDER BY c.id ASC`, so on a frozen hub with a same-size fork (two rows at
`tree_size = last_size`) §3 reads the EARLIEST (accepted) row's time, not the later rejected fork
row's. Added a mutation-pinned same-size-fork regression test. This closes the remaining §3 size/time
honesty `normal` end-to-end (the equivocation/higher-size half was closed in `820a831`).

**Files changed:**
- `internal/store/hubs.go`: §3 subselect `ORDER BY c.id DESC` → `ORDER BY c.id ASC` (one token); updated
  the `HubSummary.CheckpointObserved` field doc and the `ListHubs` doc comment to state the `id ASC`
  tiebreak rationale (the accepted row at a size is the earliest `id`; a re-observed accepted checkpoint
  is deduped by `ON CONFLICT(hub_id,tree_size,root)`, so two rows at `last_size` means a fork whose
  rejected row has a later `id` — the same "earliest row is the accepted one" selection
  `store.CheckpointAt` makes with `ORDER BY rowid`).
- `internal/store/hubs_test.go`: added `TestListHubsFrozenObservedTracksAcceptedSameSizeFork`
  (accepted root @ tAccepted, fork root [different bytes] @ tForkRejected > tAccepted, Freeze →
  asserts `CheckpointObserved == tAccepted`); clarified the existing
  `TestListHubsFrozenObservedTracksAcceptedSize` docstring to note it covers the higher-size case where
  the `tree_size = last_size` clause already excludes the rejected row (ordering irrelevant there).

**Verification:** `mise run check` → GREEN (build + vet + test across all 30 packages; `gofmt -l .`
empty).
- [x] `go test -count=1 -run TestListHubs ./internal/store` — PASS: new
  `TestListHubsFrozenObservedTracksAcceptedSameSizeFork` + existing
  `TestListHubsFrozenObservedTracksAcceptedSize`, `TestListHubsCheckpointAndAnchorHeight`,
  `TestListHubsAnchorStatus` all green.
- [x] Mutation check — reverting `internal/store/hubs.go`'s §3 subselect `ORDER BY c.id ASC` → `ORDER BY
  c.id DESC` makes the new same-size-fork test FAIL (reports tForkRejected `2023-11-15 00:59:59` vs
  tAccepted `2023-11-14 22:13:20`) while the higher-size test stays GREEN; restoring `ASC` makes both
  pass. Restore left `hubs.go` byte-clean.
- [x] Store leaf purity intact — `go list -deps ./internal/store | grep '^net/http$'` empty.
- [x] No schema/migration/`user_version`/column change — query-shape change inside the existing
  `ListHubs` statement only. `go.mod`/`go.sum` byte-unchanged.
- [x] Scope discipline — only the two `next.md`-scoped files touched (`hubs.go` + `hubs_test.go`); no
  renderer (`internal/dossier`), `ListCheckpoints`, or `Anchor`/`AnchorHeight` subselect touched.
- [x] Assertion holds: frozen hub with accepted `tree_size = last_size = 100 @ tAccepted` + same-size
  fork `tree_size = 100` (different root) `@ tForkRejected` → `ListHubs[hub].CheckpointObserved ==
  tAccepted`.
- Oracle/conformance gate — N/A (no signature/RFC-6962/Merkle/did:web/proof path; pure `checkpoints`
  read-order change).

**Next:** The two remaining dossier/realm-index honesty `normal`s are DESIGN-BLOCKED (§1 "Key resolved
from did:web:…" unconditional wording on the `unresolvable` path; realm-index `/`
per-hub-vs-per-checkpoint Anchor) — do not pull either as a code advance without a design pass. The
WASM cross-origin signature half and the proofserve-trio masthead identity are likewise design/
human-blocked. The 4 stale M-API `normal` entries (phantom `verify` `index` param, `/checkpoint`
media-type, `/healthz` 200/503) and the resolved realm-index hero entry remain pending a prune by
`update-state` — bookkeeping, not advance work. With this `normal` closed, the loop may be near the
point where only design/human-blocked work remains (see MEMORY: "Loop stalls on human-blocked DONE").

**Notes:**
- The §3 honesty `normal` this increment opened against itself in `820a831` is now CLOSED end-to-end:
  both the higher-size/equivocation case (closed in `820a831`) and the same-size fork case (this
  increment) tie §3's observed-time to the accepted row. Both are mutation-pinned.
- The fork root literal `"forked-checkpoint-root-diff-pad32"` (33 bytes) is deliberately distinct from
  the accepted `"accepted-checkpoint-root-pad-32!!"` (33 bytes) so `ON CONFLICT(hub_id,tree_size,root)`
  does NOT dedupe and a genuine second fork row exists — without distinct roots the test would be
  vacuous.
- Pre-existing `.claude/context/target.md` working-tree mod (steer `d2f259e`) is still uncommitted; not
  this advance's to commit (advance commits only implementation + test + handoff). Left for
  steer/update-state.
