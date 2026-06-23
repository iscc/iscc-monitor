## 2026-06-23 — Review of: Tie the dossier §3 observed-time to the accepted `last_size` checkpoint row (frozen-hub honesty fix)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance re-ties the §3 `observed_at` subselect in `store.ListHubs` to the accepted size
(`AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1`), with a mutation-proven frozen-hub test. This
genuinely CLOSES the higher-tree-size / equivocation decouple (a rejected LARGER checkpoint no longer
supplies §3's time) and regresses nothing — all gates green, scope clean, store stays a leaf. But Codex's
P2, reviewer-confirmed by reproduction, shows the increment does NOT close the SAME-SIZE FORK case: a fork
records a contradictory row at `tree_size == last_size`, so `id DESC` still picks the rejected row. The §3
honesty `normal` is therefore PARTIALLY closed; the issue stays open, narrowed to the fork remainder.

**Verification:**
- [x] `mise run check` — green (build + vet + test across all 30 packages).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -count=1 -run TestListHubs ./internal/store` — PASS; new `TestListHubsFrozenObservedTracksAcceptedSize`
  + existing `TestListHubsCheckpointAndAnchorHeight` (verified size 42 = last_size 42; bare-hub zero) +
  `TestListHubsAnchorStatus` all green.
- [x] New frozen test mutation-proven load-bearing (reviewer re-ran): reverting the subselect to
  `ORDER BY c.tree_size DESC, c.id DESC LIMIT 1` makes it FAIL (reports tRejected 2023-11-15 00:59:59 vs
  tAccepted 2023-11-14 22:13:20); the existing verified-hub test stays GREEN under the mutation. Restore
  left `hubs.go` byte-clean and `go.mod`/`go.sum` unchanged.
- [x] Store leaf purity intact — `go list -deps ./internal/store | grep '^net/http$'` empty.
- [x] No schema/migration/`user_version` change — query-shape change inside the existing statement only.
- [x] Scope discipline — only the two `next.md`-scoped files touched (`hubs.go` + `hubs_test.go`); no SSR
  template changed. The dirty `target.md` is a pre-existing steer mod, correctly excluded by advance.
- [x] Gate-circumvention scan over unpushed commits — clean (no `nolint`/`t.Skip`/build-tag/swallowed-err).
- [x] Oracle/conformance gate — N/A (no signature/RFC-6962/Merkle/did:web/proof path; pure `checkpoints`
  read-shape change). Confirmed by inspection.
- [ ] Closes the §3 size/time-decouple `normal` — PARTIAL: equivocation/higher-size case closed; SAME-SIZE
  FORK case still pairs the accepted size with the rejected fork checkpoint's time (see Issues + Codex).

**Issues found:** One (Codex-originated, reviewer-confirmed by reproduction) — the §3 fork remainder. The
existing §3 `normal` in `issues.md` is REWRITTEN to record the partial fix: the equivocation/higher-size
case is closed, the same-size fork case remains, durable fix is `ORDER BY c.id ASC` (accepted row at a size
is the earliest — `store.CheckpointAt` precedent via `ORDER BY rowid`) or key by accepted root. Not deleted.

**Codex second opinion:** One P2 finding at `internal/store/hubs.go:77` — "same-size fork violations still
select the rejected checkpoint's timestamp." **CONFIRMED by reviewer reproduction** (throwaway probe:
accepted size 100 @ tAccepted, fork at size 100 different root @ tForkRejected, Freeze → `CheckpointObserved
== tForkRejected`, not tAccepted). Grounded in the violation taxonomy (`logclient/checkconsistency.go:35-37`:
fork = `next == prev`) and `follower.freeze` (records the contradictory checkpoint at `info.TreeSize ==
last_size`). The new subselect's `tree_size = f.last_size` matches both rows; `id DESC LIMIT 1` picks the
later (rejected) one. Real, in-scope, and the same defect class the increment targeted — kept the §3 `normal`
open and narrowed it to the fork case rather than deleting it. Not a regression (the pre-fix query was also
wrong for forks) and the increment's stated higher-size goal IS met + mutation-proven, so PASS_WITH_NOTES
not NEEDS_WORK.

**Visual check:** n/a — no SSR surface changed. The diff is store-side only (`internal/store/hubs.go` + its
test); the §3 renderer (`internal/dossier`) is untouched.

**Next:** Finish the §3 honesty fix — change the `ListHubs` §3 subselect `ORDER BY c.id DESC` → `ORDER BY
c.id ASC` (the accepted checkpoint at a given size is the EARLIEST `id`; `store.CheckpointAt` already uses
`ORDER BY rowid LIMIT 1` for exactly this "the accepted row at this size" selection). Add a same-size-fork
regression test (accepted root @ t1, forked root @ t2>t1, no `last_size` advance, Freeze) asserting §3 time
== t1; keep the higher-size `TestListHubsFrozenObservedTracksAcceptedSize` green (distinct sizes → only one
row matches, ordering irrelevant). One-file + test, ≤3 budget, oracle N/A. This is the natural immediate
follow-up since it completes the very `normal` this increment opened against itself.

**Notes:**
- The advance's handoff justified `id DESC LIMIT 1` as "keeps a re-observed same-size row deterministic" —
  but that rationale is exactly backwards for honesty: a re-observed ACCEPTED checkpoint (same root) is
  deduped by `RecordCheckpoint`'s `ON CONFLICT(hub_id,tree_size,root) DO NOTHING`, so the ONLY way two rows
  share `tree_size = last_size` is a fork (different root), and the accepted row is always the EARLIER `id`.
  `id ASC` is the correct tiebreak. Recorded in both `learnings/store.md` and `learnings/dossier.md`.
- The two remaining dossier/realm-index honesty `normal`s (§1 "Key resolved from did:web:…"
  unconditional-wording on the `unresolvable` path, realm-index `/` per-hub-vs-per-checkpoint Anchor) are
  still DESIGN-BLOCKED — do not pull either as a code advance.
- The 4 stale M-API `normal` entries (phantom `verify` `index` param, `/checkpoint` media-type, `/healthz`
  200/503) remain pending a prune by `update-state` per the prior handoff — bookkeeping, not advance work.
- Pre-existing `.claude/context/target.md` working-tree mod (steer `d2f259e`) is still uncommitted; not the
  review's to commit (review commits learnings/handoff/issues + any minor fixes only). Left for steer/update-state.
