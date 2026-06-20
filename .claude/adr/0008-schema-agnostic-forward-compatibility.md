---
status: accepted
---

# Schema-agnostic verification; interpretation is an extensible projection layer

New log message types are coming (e.g. gateway-URL update, ISCC-ID ownership
transfer). The monitor must handle unknown types gracefully — the severe failure
mode being that an old monitor treats a *new, valid* message type as a problem and
raises false alarms across every up-to-date hub.

The design enforces a two-layer separation:

**1. Verification + evidence path is schema-agnostic.** Checkpoint acceptance,
consistency checking, freeze detection (ADR-0006), mirroring, and inclusion proofs
depend ONLY on raw record bytes + the Merkle tree + checkpoint signatures. They
MUST NOT inspect `note.$schema` or note internals. A new message type flows through
untouched and can never trigger a false violation or break following.

**2. Interpretation is a separate, extensible projection layer.** Only this layer
maps a `note.$schema` to meaning. Adding support for a new type later is purely
additive — teach the projection layer; the verification/evidence path is unchanged.

## Rules

- **Store the raw `note.$schema` per record**, not a closed enum. Future types are
  recorded faithfully the day they appear, even by a monitor that predates them.
- **Depend only on the stable envelope** (`$schema`, `iscc_id`, `note`). `iscc_id`
  is always extractable at the envelope level, so every record — known or not — is
  indexed by its id (one-to-many; see ADR rule on iscc_id).
- **`inclusion?iscc_id=` returns the declaration proof + a generic list of all
  other records referencing the id: `[(seq, $schema)]`.** This subsumes deletions
  and future transfers/gateway-updates without special-casing any type. `?seq=`
  proves a specific record.
- Unknown schemas are indexed + proof-able but never *interpreted* (spec §11.4:
  skip-don't-guess) and never gate verification.

## Scope line

v1 monitor = **evidence + per-id record history**. It mirrors, proves, and lists
transfers/gateway-updates generically the moment they appear. **Resolution
projections** (current owner, current gateway URL) are stateful folds over those
records — Aggregator projections (spec §1.3), **deferred from v1 but enabled** by
the schema-agnostic index, buildable later with no change to verification.

## Property (from ADR Q8)

Because redaction is a local hub policy that writes no log record (spec §13), the
monitor never sees it and always preserves the original declaration. The monitor is
the un-redactable record; it does not honor a hub's local takedowns. This is an
intended feature.
