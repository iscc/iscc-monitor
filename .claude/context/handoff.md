## 2026-08-07 — Advance + Review: Follow-traffic contract (mirror-as-cache) + mirror repair path

**Role:** advance + review, **human-directed, outside the normal loop cadence** (filed from a
production egress report by the ISCC Hub operator; GitHub issue #4). **Verdict: PASS** — reviewed by
Titusz + Claude working outside the loop; **COMMITTED to `develop` and pushed** in the commit that
carries this handoff. The next loop iteration starts from a clean tree.

**What landed (two passes, one increment):**

*Pass 1 — fetch only what can have changed (the issue-#4 fix proper):*
- `internal/store/tiles.go` — `TileKey`, `MirroredFullTiles`, `MirroredFullEntryBundles`: hub-scoped
  set reads filtered on `widthForP(0)`, the same p→width authority the write side and `SQLiteFetcher`
  use. Two queries per poll, never per-coord (the `SetMaxOpenConns(1)` pool).
- `internal/follower/ingest.go` — both walks skip a coord when `c.Partial == 0 && alreadyFull`;
  partials (`.p/<W>`), the checkpoint and `did.json` are always fetched fresh. `ingestEntryBundles`
  writes the `iscc_index` projection BEFORE `RecordEntryBundle` so "full bundle mirrored ⟹ projection
  written" holds. Steady-state poll: ~2350 requests → ~6 on a 300k-entry log.
- `target.md` gained the **Follow-traffic contract** standing bar (binds every outbound path);
  `CLAUDE.md`'s Mirror glossary entry records the fetch-cache role.

*Pass 2 — consequences of making the mirror authoritative (the trust-model flip):*
- **Admission gates:** `store.RecordTile` rejects a full tile that is not exactly `TileWidth*32`
  bytes; `ingestEntryBundles` rejects a full bundle decoding to fewer than `TileWidth` records.
- **Repair path:** `ingestTiles(..., force=true)` re-walks a hub authoritatively; `PollHub` runs it
  BOTH before convicting a hub of a self-consistency violation and after a failed root rebuild —
  never convict on cached bytes (freeze is irreversible in v1). `MirroredFullEntryBundles` requires
  the projection to exist, so a legacy half-written DB re-folds instead of freezing its index gap.
- **`fsckTimeout` (2m) bound in `fsckMirror`:** tessera's fsck DEADLOCKS (not errors) on a corrupt
  completed tile — reported upstream as **transparency-dev/tessera#1098** (repro confirms v1.0.2 AND
  v1.0.4 affected; the abandoned goroutine leaks but does not strand the store). Details in
  `issues.md`.
- **Fixture divergence fixed in 4 test builders** (`certificate/handler_test.go`,
  `proofserve/handler_test.go`, `proofserve/verify_test.go`, `follower/fsck_test.go`): they
  synthesized partial tiles by a "subtree starts below size" bound instead of the width the path
  advertises — verified against a live 300258-leaf hub.

**Scope note for the record:** `next.md`'s Not-In-Scope said "No mirror-repair path"; the human
directed folding it in after the tessera deadlock made the missing repair path a live hazard rather
than a filed `normal`. The residuals (unread `sha256` column, all-or-nothing repair walk, CDN-cached
poisoned rows, per-poll set-scan cost) are filed as `low` in `issues.md` — the critical itself is
CLOSED.

**Verification (recap — full detail in `issues.md` + `learnings/follower.md` + `learnings/store.md`):**
- `mise run check` green (30/30 packages), `gofmt -l .` clean, at the committed tree.
- Mutation-proven: the 4 pass-1 mutations (both skips, the `Partial == 0` guard, the
  projection/record order) and the 5 pass-2 load-bearing halves (length gate, short-bundle gate,
  projection-EXISTS requirement, force re-ingest before conviction, fsckTimeout non-stranding) each
  FAIL their test when reverted.
- Equivocation reasoning re-derived independently (review): the mirror-mix argument holds —
  `ingestTiles` still precedes `checkConsistency`, a mixed tree cannot reconstruct both roots, an
  honest hub's mix is byte-identical, and the force re-ingest before conviction makes a false freeze
  strictly harder than before.

**Next:** normal loop cadence resumes. Open code-closable normals: the OTS per-poll→daily cadence,
and the dead baked realm-fallback hosts (`deploy/realm-testnet.txt`). Deploying this commit to
`monitor.iscc.io` is what actually stops the production egress — an ops step for the human.
