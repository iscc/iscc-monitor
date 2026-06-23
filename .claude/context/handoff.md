## 2026-06-23 — Review of: Finish the dossier §3 honesty fix — `ORDER BY c.id ASC` so a same-size fork reads the accepted row's time

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance is a one-token SQL change (`ORDER BY c.id DESC` → `ORDER BY c.id ASC`) in the §3
`observed_at` correlated subselect of `store.ListHubs`, plus a mutation-pinned same-size-fork regression
test and accurate doc-comment updates. Independently verified correct: `AdvanceAccepted` inserts the
accepted checkpoint first (lowest `id`), `RecordCheckpoint` inserts the contradictory fork row later
(higher `id`), and `ON CONFLICT(hub_id,tree_size,root) DO NOTHING` means two rows at `last_size` can only
be a fork — so `id ASC LIMIT 1` deterministically selects the accepted row (matching `CheckpointAt`'s
`ORDER BY rowid`). This closes the §3 size/time honesty `normal` end-to-end. All gates green, scope clean,
store stays a leaf, Codex concurs.

**Verification:**
- [x] `mise run check` — green (build + vet + test across all 30 packages).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -count=1 -run TestListHubs ./internal/store` — PASS: new
  `TestListHubsFrozenObservedTracksAcceptedSameSizeFork` + existing
  `TestListHubsFrozenObservedTracksAcceptedSize`, `TestListHubsCheckpointAndAnchorHeight`,
  `TestListHubsAnchorStatus` all green (re-ran verbosely).
- [x] Mutation check (reviewer re-ran) — reverting the §3 subselect `ORDER BY c.id ASC` → `ORDER BY c.id
  DESC` makes `…SameSizeFork` FAIL (reports fork `2023-11-15 00:59:59` vs accepted `2023-11-14 22:13:20`)
  while the higher-size `…AcceptedSize` test STAYS GREEN under the mutation; restoring `ASC` passes both.
  Restore left `hubs.go` byte-clean. Test is non-vacuous and load-bearing.
- [x] Store leaf purity — `go list -deps ./internal/store | grep '^net/http$'` empty.
- [x] No schema/migration/`user_version`/column change — confirmed; `go.mod`/`go.sum` byte-unchanged in
  the advance commit (`git diff HEAD~1..HEAD --stat -- go.mod go.sum` empty).
- [x] Scope discipline — only the two `next.md`-scoped files touched (`hubs.go` + `hubs_test.go` + the
  handoff); no renderer (`internal/dossier`), `ListCheckpoints`, or `Anchor`/`AnchorHeight` subselect.
- [x] Gate-circumvention scan over the 3 unpushed commits — clean (no `nolint`/`t.Skip`/build-tag/swallowed-
  err; no removed assertions — the only deleted test-file lines are the docstring rewrite on the existing
  higher-size test).
- [x] `id` monotonicity — `checkpoints.id INTEGER PRIMARY KEY` is the SQLite rowid alias, so insertion
  order == `id` order (confirmed against `schema.sql`); the accepted-first / fork-later ordering holds.
- [x] Oracle/conformance gate — N/A (no signature/RFC-6962/Merkle/did:web/proof path; pure `checkpoints`
  read-order change). Confirmed by inspection.

**Issues found:** (none). Deleted the resolved §3 `normal` ("Dossier §3 observed-time still tracks the
REJECTED row for a same-size FORK violation") from `issues.md` after verifying the fix + mutation.

**Codex second opinion:** Clean. Verdict: "The change correctly switches the same-size checkpoint tie-break
to the earliest row and adds a focused regression test; I did not identify any introduced correctness
issues." No findings to triage — matches my independent review.

**Visual check:** n/a — no SSR surface changed. The diff is store-side only (`internal/store/hubs.go` + its
test); the §3 renderer (`internal/dossier`) is untouched (renderer was already correct; the query was
picking the wrong row).

**Next:** This was the sole code-closable convergence move. Every remaining open `normal` is design- or
human-blocked: the §1 "Key resolved from did:web:…" unconditional wording on the `unresolvable` path, the
realm-index `/` per-hub-vs-per-checkpoint Anchor honesty, and the WASM cross-origin signature half all need
a design pass before any code advance. The proofserve-trio masthead-identity threading is the natural next
non-blocked arc IF a design decision authorizes it (and it would also unlock the 3x masthead-fallback-const
consolidation `low`). Otherwise the loop is at the "only design/human-blocked work remains" point — see
MEMORY "Loop stalls on human-blocked DONE"; consider whether a STOP for human design input is warranted
next iteration rather than spinning on cosmetic chrome.

**Notes:**
- The §3 size/time honesty `normal` is now CLOSED end-to-end (both the equivocation/higher-size case from
  `820a831` and the same-size fork case from `244d450`), both mutation-pinned. The `store.md` TRAP bullet
  that predicted this exact fix (`id ASC`) was collapsed to a `settled:` line; `dossier.md`'s §3 bullet
  updated to "settled / CLOSED".
- Learnings rotation: `store.md` net-reduced 179 → 166 lines (collapsed three landed `settled` bullets —
  the §3 TRAP, the `RecentRecords`+composite-PK pair, and the migration out-of-range guard — into one-line
  summaries; git history keeps the detail). Still ~16 over the ~150 target but materially reduced; the
  remainder is all durable forward-looking traps (modernc pin, per-connection FK/WAL pragmas,
  append-never-edit migration discipline, schema-verbatim rule).
- Stale bookkeeping pending an `update-state` prune (NOT advance work): the 4 M-API `normal` entries
  (phantom `verify` `index` param, `/checkpoint` media-type, `/healthz` 200/503) and the now-fully-resolved
  realm-index hero entry (all four sub-items CLOSED).
- Pre-existing `.claude/context/target.md` working-tree mod (steer `d2f259e`) is still uncommitted; not the
  review's to commit. Left for steer/update-state.
