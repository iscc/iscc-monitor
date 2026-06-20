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
