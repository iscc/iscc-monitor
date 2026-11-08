---
status: accepted
---

# One SQLite file per network; reconciliation never deletes evidence

**Per-network database files.** Each network gets its own SQLite file
(`mainnet.db`, `testnet.db`). One binary still serves `testnet | mainnet | both`;
`both` opens both files with independent single-writer goroutines.

Row-level namespacing (a `network` column) does **not** isolate the failure modes
that matter — disk-full, file corruption, and single-writer serialization are all
*file-level*. Mainnet is the irreplaceable evidentiary log; testnet is noisy and
experimental (where the sb1 misconfiguration surfaced). Separate files keep testnet
churn/bugs/bloat away from mainnet evidence and make per-network backup cadence
trivial. Bonus: the `network` column is dropped from every table — the schema gets
*simpler*. Cost: cross-network queries read two files instead of one `WHERE` —
trivial and rare.

**Reconciliation changes follow-status only; it never deletes evidence or the
mirror.**

- `active: false` → pause polling; keep serving the mirror + all evidence; resume
  from persisted `follow_state` if reactivated.
- Removed from the Hub-List → status `removed`; stop following, **retain
  everything**, keep serving as an availability backstop. A hub vanishing from the
  list is exactly when an independent mirror matters most.
- Re-appears → resume from persisted `follow_state` + existing mirror.

The active follow set is the only thing reconciliation adds to or removes from;
the irreplaceable evidence (ADR-0001, ADR-0005) is forever.

**Why network-level and not hub-level (considered, deferred).** The same
file-level-failure-isolation argument that splits networks would, taken one step
further, split *each hub* into its own file. We stop at the network for now: the
realm is 10s–100s of hubs, so per-hub file lifecycle (create-on-join, never-delete
per the retention rule above) and the fan-out of every cross-hub read buy nothing
the `hub_id` namespacing inside a network doesn't already give us. Building it now
would be over-engineering.

The decision is deferred, not foreclosed, because the design keeps the migration
clean *by construction* — and that is the property to preserve, not the single
file:

- **Writes are hub-scoped.** The store's only transaction (`AdvanceAccepted`)
  touches exactly one `hub_id`; every data-access method is keyed by `hubID`.
  There is no cross-hub transaction, so splitting hubs across files preserves
  every atomicity guarantee ADR-0005 relies on. *This is the load-bearing
  invariant: never introduce a write or transaction that spans hubs.*
- **The `both`-mode multi-file/multi-writer pattern is the precedent.** Per-hub
  files extend an existing shape (independent single-writer goroutines per file),
  not a new one.
- **BLOB keys are storage-agnostic** (`(hub_id, level, index, width)`), and reads
  already go through the `client.Fetcher` seam — so the high-volume axis (a single
  hub at millions of records/day) has a *second*, independent clean migration:
  tier the **rebuildable** bulk (tiles, entry bundles, `iscc_index`) onto a
  different backend while the small **irreplaceable** evidence stays in SQLite.
  Per-hub files *isolate* a hot hub; tiering the rebuildable bulk *subdivides* it.
- **Rebuildability lowers migration risk.** A future split need not copy the bulk
  at all — stand up the new backend, hand-copy only the irreplaceable evidence
  tables, and let the follower backfill tiles/index by re-fetch + fsck.

A trip-wire on writer-wait time and per-network file size (tracked in
`.claude/context/issues.md`) gives the early signal to revisit this before the
single writer or file size actually binds.
