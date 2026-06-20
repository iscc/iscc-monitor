---
status: accepted
---

# Self-consistency violations: three triggers, evidence-only polling, manual unfreeze

The monitor freezes a hub on a **self-consistency violation** (ADR-0001's
autonomous-detection class). There are three distinct triggers, each checked every
poll — the plan named only the first:

1. **Fork/rewrite** — RFC-6962 consistency proof from an accepted root to a later
   same-or-larger root fails (`ErrInconsistency`).
2. **Shrink/rollback** — a new checkpoint's `tree_size` is *smaller* than one
   already accepted (violates the spec's monotonic-size rule; not an
   `ErrInconsistency`, so it needs its own check).
3. **Equivocation** — two checkpoints at the *identical* `tree_size` with different
   roots. Not caught by a forward consistency proof; this row also *is* portable
   split-view evidence (two contradictory hub-signed checkpoints).

On any trigger: persist the contradictory raw checkpoints + proof permanently,
set `frozen=1`, alert **once** on the transition (idempotent).

**Frozen hubs keep being polled — in evidence-only mode**, at a backed-off cadence:
record raw checkpoints as further evidence, but never advance accepted state, never
cosign, never anchor. Going silent on the most interesting hub would defeat the
monitor's purpose.

**No auto-unfreeze.** A violation is permanent evidence; auto-recovery would clear
the signal. Unfreezing is a manual operator decision after review; the evidence
rows are permanent regardless. (Genuine consistency failures are not transient
noise — stale/old checkpoints are *consistent*, not inconsistent.)

## Consequences

- **Rename `split_views` → `violations`** with a `kind` column
  (`fork | shrink | equivocation`). Per CONTEXT.md, "split view" is reserved for
  equivocation detectable only by comparison; this table holds violations the
  monitor detects *autonomously*. Only the `equivocation` rows double as
  standalone split-view evidence.
- The follower's per-poll check is: parse+verify signature (ADR-0002) → size
  monotonic? → same-size-same-root? → forward consistency? → only then accept,
  mirror, index, and mark cosign/anchor eligible.
