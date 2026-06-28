---
status: accepted
---

# Cosigning is independent-attestation infrastructure; dropped from v1 (OTS is the v1 attestation)

A self-published monitor cosignature has **no v1 consumer on its own**. The
business-user dashboard consumes badges or in-browser verification, not a monitor
cosig; and "what the monitor saw" is already published as the mirrored
**hub-signed** checkpoints (unforgeable by the monitor), which are sufficient as
the split-view comparison anchor (ADR-0001). What a cosig adds over that is narrow:
monitor non-repudiation, a one-signature "an independent party vouched" artifact,
and the C2SP witness-cosignature format — whose real consumer is **M7 gossip**,
which is out of v1.

**Decision.** Drop cosigning from v1 entirely. The v1 independent attestation is
**OTS-anchoring observed roots alone** (trustless timestamp, no monitor signing key
in the trust path). Cosigning returns at **M7** as the C2SP witness/gossip wire
format, where it has an actual consumer. The monitor therefore needs a TLS identity
but publishes **no signing key** in v1. Milestone order:

1. **M1** — read-only Monitor (verify sig + consistency, persist, freeze, metrics)
2. **M2** — mirror + iscc_index (serves inclusion/consistency from the local mirror)
3. **M3** — Trust API + server-rendered dashboard + log browser
4. **WASM upgrade** — in-browser verifier as progressive enhancement on M3 (ADR-0003)
5. **OTS / Bitcoin anchoring** — anchor observed roots (the v1 independent
   attestation; trustless timestamp + anti-rewrite). **No cosigning.**
6. **M7 (deferred)** — multi-monitor gossip **+ cosigning** (C2SP witness
   cosignatures) + witness endpoint. Cosigning lives here because gossip is its
   only real consumer.

This sequencing delivers the headline value (a verifiable trust dashboard for
business users) first, makes OTS the v1 attestation, and parks cosigning with the
gossip protocol that consumes it.

## Amendment (2026-06-28) — OTS anchoring cadence: daily, latest-root-only

**Context.** This ADR set "OTS-anchoring observed roots" as the v1 attestation; the PRD
refined it to "**OTS = daily per hub**, keyed by `(hub, tree_size, root)` UNIQUE." The
implementation **drifted** from that cadence: it stamps on the **poll path**
(`PollHub → stampRoot`, one `RecordOTS` per accepted-checkpoint advance), so at the
production 5-minute poll cadence it anchors **every observed root**, not one per day. A
single actively-growing hub accumulates one OTS row — and one calendar submission — per
observed advance (production `amlet.id`: 56 anchored roots over ~51 h, the bulk from
+1/+2-entry advances). This scales with **traffic, not time**, and is unbounded for a busy
hub (the PRD/OPERATING.md anticipate a hub at millions of records/day).

**Decision.** Anchor on a **daily cadence, latest-root-only**: once per day, for each
**non-frozen** followed hub, anchor that hub's **current latest accepted `(size, root)`**
(deduped on the existing `UNIQUE(hub_id, tree_size, root)`). Intermediate roots observed
between daily passes are **not** individually anchored. A frozen hub is not anchored (this
PRD's freeze rule is "no advance/anchor/OTS"). Net effect: **≤ 1 new anchor per non-frozen
hub per day, and 0 for an idle hub** (the dedupe no-ops when the latest root has not moved).

**Why this is sound (not merely cheaper).** The log is an append-only RFC-6962 Merkle tree
and the monitor verifies consistency between successive checkpoints. Anchoring the
**latest** root therefore transitively timestamps every earlier entry: `OTS(root@N)` plus a
consistency proof `M→N` proves everything ≤ N existed by that Bitcoin block. So daily
latest-root anchoring is **evidence-equivalent** to per-root anchoring for the
"existed-before-block-H" guarantee (user stories 29–30) — at daily granularity. It does
**not** weaken autonomous detection: split-view / rewrite detection is the 5-minute poll +
RFC-6962 self-consistency check against the monitor's own stored checkpoints, which is
unchanged. OTS is purely the trustless Bitcoin backstop, for which daily precision suffices.

**Trade-off.** An entry added just after a daily anchor is not Bitcoin-pinned until the next
day's anchor (≤ ~24 h coarser *Bitcoin* timestamp). Acceptable: the hub's own signed
checkpoint and the monitor's `observed_at` already give ~5-minute timestamps; OTS is the
daily trustless floor, not the precision instrument.

**Backward compatibility — no database reset.** Policy change **only, no schema change**
(`ots` table unchanged). Existing OTS rows — including every Bitcoin-confirmed anchor (the
glossary's *irreplaceable evidence*) — stay valid; the unchanged
`UNIQUE(hub_id, tree_size, root)` dedupe means the daily pass never conflicts with a
pre-existing row, and any in-flight pending row finishes its normal upgrade lifecycle.
**Production databases are NOT reset** — a reset would destroy confirmed Bitcoin anchors for
zero benefit. The change is purely forward-looking: stop creating a row per observed root;
start creating ≤ 1 per non-frozen hub per day.

**Implementation shape** (mechanism is `define-next`'s call): remove the stamp from the poll
path (`PollHub` no longer calls `stampRoot`) and add a daily pass that reads each non-frozen
hub's latest accepted `(size, root)` and `RecordOTS`-es it (deduped). The existing
background OTS loop (`OTSTick`: calendar-submit the empty-sentinel rows, then upgrade pending
→ Bitcoin-confirmed) is unchanged — only **where the pending row is created** moves. The
simplest landing folds the daily pass into the existing 24 h OTS loop tick; decoupling the
anchor cadence (daily) from a future faster *upgrade* cadence stays possible.

**Unchanged.** OTS remains best-effort and **never blocks the follower** (the core of this
ADR). The `.ots` route, certificate §5, and the `Confirmed`/`ConfirmedFor` classifiers are
untouched. M7 cosigning stays deferred.
