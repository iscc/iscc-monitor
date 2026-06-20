---
status: accepted
---

# v1 trust guarantee: self-consistency detection + equivocation anchor

A single monitor sees only one audience's view of a hub, so v1 does **not** claim
autonomous split-view (equivocation) detection — that requires monitor gossip,
deferred to M7. Instead v1 guarantees two things: (1) it **autonomously detects**
signature failures and *self-consistency violations* (a hub rewriting history
against this monitor over time, caught via RFC-6962 consistency on successive
observations), and (2) it serves as an independently-signed **comparison anchor**
that lets any client detect equivocation by checking *its own* `(size, root)`
against the monitor's mirrored tree.

## Consequences

- The verification surface (WASM / verify-for-me / proof bundle) MUST accept the
  client's own `(size, root)` and prove it against the monitor's tree via an
  RFC-6962 consistency proof — not merely return the monitor's own roots. This is
  what makes the guarantee "compare," not "trust."
- The aggregator tile mirror (M2) is therefore **load-bearing for the trust
  guarantee**, not just an availability backstop: without mirrored tiles a client
  cannot run the consistency check that detects equivocation.
- Dashboard and docs language must distinguish "self-consistency violation"
  (detected autonomously, triggers freeze/alert) from "split view" (detectable
  only by client/peer comparison until M7).

## Coverage and cold start (added 2026-06-20)

tlog-tiles serves only the *current* checkpoint; preserving the checkpoint history
is the monitor's job. Two facts bound what that history can guarantee:

- **Missing intermediate checkpoints is harmless.** RFC-6962 consistency proofs
  are transitive and composable, so the monitor verifies consistency *between the
  checkpoints it caught* (e.g. 100 → 5 000 → 90 000) from the tiles — gaps are
  bridged. Poll cadence affects observation granularity, not the guarantee. Split
  view detection is likewise unaffected: a client at a missed size compares its
  `(size, root)` against the monitor's *tree* (the root at that size is
  reconstructed from tiles), no checkpoint-at-that-size required.
- **A late-starting monitor cannot retroactively detect pre-coverage
  equivocation.** Starting on an existing log, it backfills all tiles for
  `0..N`, reconstructs and verifies the full tree against the current signed root
  — recovering **all data + full forward guarantees** — but it never had the
  historical checkpoint *signatures*, and the hub won't serve a past checkpoint
  that contradicts its current head. This is spec §13's "cold start"; it is
  inherent and unfixable by any storage choice.

**Therefore:** record **`monitored_since` (size + time) per hub**, and state every
guarantee as holding *from coverage start onward*. The dashboard MUST show the
coverage window and never imply guarantees over the pre-coverage period. The realm
registry / coverage record (spec §13) is the public home for "monitored since."

