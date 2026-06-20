---
status: accepted
---

# Single SQLite store; no separate filesystem tile mirror

All monitor state — observed checkpoints, split-view evidence, cosigs, OTS proofs,
the `iscc_index`, **and the mirrored hash tiles + entry bundles** — lives in one
SQLite database (WAL, single writer). The plan's separate filesystem tlog-tiles
mirror is dropped.

## Why

- **Atomicity eliminates the divergence/crash-tear problem.** Each poll commits
  `{checkpoint, any split-view + freeze, new tiles/bundles, index updates}` in one
  transaction — all-or-nothing. Two stores can no longer drift apart.
- **One backup.** Irreplaceable evidence and tiles in a single file.
- **No cross-platform path foot-guns.** The tlog-tiles filesystem layout
  (thousands-grouped indices, `.p/<W>` partial suffix, path-length/separator
  issues on Windows) is bug-prone; keying BLOBs by `(level, index, width)` avoids
  it entirely.

## We lose nothing essential (verified)

- **`fsck`:** `fsck.New` takes a `client.Fetcher` (3 methods: `ReadCheckpoint`,
  `ReadTile(l,i,p)`, `ReadEntryBundle(i,p)`), and tessera already ships
  `HTTPFetcher` + `FileFetcher`. A `SQLiteFetcher` is a trivial third
  implementation reading BLOBs — same code paths as `runfsck`.
- **Aggregator / availability backstop:** canonical tlog-tiles paths become an
  HTTP handler over BLOBs; the stored bytes are identical to what the hub served.
- **Proofs:** `ProofBuilder` reads through the same fetcher.

## The one real trade-off (accepted)

A filesystem mirror is itself a static artifact a dumb host can serve even if the
monitor binary is dead; SQLite serving needs the running service. Judged a
nice-to-have for v1 (if the process is down, the backstop is down regardless), and
re-addable later as an `export-mirror` command that walks SQLite into the static
tlog-tiles tree.

## Consequences

- Per-poll pattern: fetch tiles over the network **outside** the write
  transaction; commit the batch **inside** the single writer. WAL so serving/WASM
  reads don't block the writer; checkpoint the WAL periodically.
- Backup protects the **irreplaceable** evidence (checkpoints, split_views,
  cosigs, ots). Tiles + `iscc_index` are **rebuildable** (re-fetch + fsck +
  re-walk), so they may be excluded from backups if size ever demands it.
- A `verify` self-check runs `fsck` via the `SQLiteFetcher` to confirm stored
  tiles rebuild each accepted root.
- Supersedes the filesystem-mirror parts of the plan (`store/blobstore.go`, the
  canonical filesystem layout) and the three-tier *storage* split proposed during
  grilling — the irreplaceable-vs-rebuildable distinction now governs only
  backup/rebuild policy.
