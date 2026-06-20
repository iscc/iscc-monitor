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
