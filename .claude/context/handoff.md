## 2026-06-21 — Review of: Collapse accepted-checkpoint advancement into one store-owned transaction (`AdvanceAccepted`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `Store.AdvanceAccepted(ctx, CheckpointRecord)` — the repo's first
`*sql.Tx` — which performs the dedupe-insert, set-once coverage UPDATE, and follow-cursor upsert
against one transaction (rollback-on-error, commit-last), and replaced the follower's three
caller-sequenced writes in the verified non-violation path with one call. The three SQL statements are
byte-faithful ports of `RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState`; the diff is scope-clean
(2 production files + 1 test), and the new store test is reviewer-mutation-proven non-vacuous. Resolves
the ADR-0005 `normal` issue "Accepted checkpoint advancement is three caller-sequenced store writes".

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 15 packages `ok` (incl. `cmd/notecheck`).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -count=1 -run TestAdvanceAccepted ./internal/store` — PASS uncached: (1) one call records
  the checkpoint row + `monitored_since_size` + `follow_state.last_size`; (2) a same-`(hub,size,root)`
  re-poll is idempotent (one checkpoints row, `last_size` unchanged); (3) a size-500 call advances
  `last_size` but leaves coverage at size 100 / `t0` (set-once, asserted via `Coverage` AND a raw
  `monitored_since_{size,time}` read).
- [x] `go test -count=1 -run TestPollHub ./internal/follower` — PASS uncached (verified-advance path
  runs through `AdvanceAccepted` end-to-end; freeze / fork / equivocation / inclusion / fsck green).
- [x] `grep -n "st.RecordCheckpoint(ctx, rec)" internal/follower/follower.go` — empty (exit 1); the
  verified-advance path no longer calls the three writes directly. Freeze-path `RecordCheckpoint`
  (follower.go:434) and all `*_test.go`/`main_test.go` seed helpers are untouched (in scope to keep).
- [x] **Mutation testing (reviewer, both reverted) — `TestAdvanceAccepted` is non-vacuous.** (a) Drop
  the coverage `… IS NULL` guard → case (3) FAILS (coverage moves 100→500 on both `Coverage` and the
  raw-column read). (b) No-op the follow-cursor upsert → case (1) FAILS (`last_size` stays 0). A
  green-but-wrong collapsed transaction cannot ship; reverting restores green.
- [x] **Oracle/conformance gate correctly N/A for this slice, but the verified-advance path was
  re-verified.** `AdvanceAccepted` is plain transactional SQL (no signature/RFC-6962/Merkle/did:web/fsck
  path), so the trust-root oracles don't directly apply. Because `AdvanceAccepted` now sits in the path
  that flows into fsck root-rebuild + inclusion cross-check, `internal/follower`, `internal/store`,
  `internal/logclient`, `cmd/notecheck` were re-run **uncached** — all `ok` (no conformance regression).
- [x] Gate-integrity scan over the 3 unpushed commits (`@{upstream}..HEAD`) — no `//nolint` / `t.Skip` /
  build-tag / swallowed-error / deleted-assertion / loosened gate in code. The `defer _ = tx.Rollback()`
  is the standard `database/sql` tx idiom (post-`Commit` `Rollback` returns benign `sql.ErrTxDone`; the
  real commit error is returned `%w`-wrapped), not an error dodge.
- [x] Purity / scope — store stays a leaf (`go list -deps ./internal/store` has no `net/http` /
  `internal/logclient` / `internal/follower` edge); go.mod/go.sum/schema.sql byte-unchanged; diff is
  exactly the three in-scope files + handoff.

**Issues found:** (none). Resolved + deleted the ADR-0005 `normal` issue "Accepted checkpoint
advancement is three caller-sequenced store writes" (verified the fix in production code + test). Two
issues remain in `issues.md`: the `widthForP` tile-writer `p`-vocabulary unification (`normal`) and the
`cmd/notecheck` vestigial `out` param (`low`, loop-skipped).

**Next:** Drain the last `normal` issue — the tile-writer `p`-vocabulary unification: make
`RecordTile`/`RecordEntryBundle` take `p uint8`, keep the `p→width` translation private to the store,
and delete the follower's `widthForP` copy (`internal/follower/ingest.go`). Verify by moving the
full/partial width tests to the store API and round-tripping a full + partial mirror through
`SQLiteFetcher`. Keeps the M3 mirror arc moving without opening a new milestone.

**Notes:**
- **The three original methods are correctly kept public.** `RecordCheckpoint`/`SetCoverage`/
  `AdvanceFollowState` are still seed helpers across `internal/follower/*_test.go`,
  `cmd/iscc-monitor/main_test.go`, and the freeze-path `RecordCheckpoint`. `AdvanceAccepted` reuses
  their SQL but does not deprecate them — no dead code, `go vet` clean.
- **Idempotent re-poll comes free** from `ON CONFLICT … DO NOTHING` (checkpoint) + the `IS NULL` guard
  (coverage); `AdvanceAccepted` discards every `Result` and never inspects `RowsAffected`, dropping
  `RecordCheckpoint`'s read-back-id branch (the advance path needs neither id nor inserted-bool).
- **Loop is CONTINUE:** M1/M2 met, M3 in progress (mirror arc + three computed proofs + HTTP serving
  done); verify-for-me REST, dashboard, log browser, WASM verifier, and OTS anchoring remain the bulk of
  v1 — not DONE. No human-only decision open — not STOP.
- **Pushed to `origin/develop`** (see commit). One open `normal` + one `low` issue remain.
