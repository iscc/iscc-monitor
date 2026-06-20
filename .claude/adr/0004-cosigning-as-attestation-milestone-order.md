---
status: accepted
---

> **Update (2026-06-20):** decision strengthened — cosigning is **dropped from v1
> entirely**, not merely fused-and-deferred. The v1 independent attestation is
> **OTS-anchoring observed roots alone** (trustless timestamp, no monitor key in
> the trust path); cosigning returns at **M7** as the C2SP witness/gossip wire
> format, where it has an actual consumer. This also removes the monitor's-own-key
> / Monitor-List problem from v1 (the monitor needs a TLS identity but publishes
> no signing key until M7). Rationale below stands; see the revised milestone list.

# Cosigning is independent-attestation infrastructure; dropped from v1 (OTS is the v1 attestation)

A self-published monitor cosignature has **no v1 consumer on its own**. The
business-user dashboard consumes badges or in-browser verification, not a monitor
cosig; and "what the monitor saw" is already published as the mirrored
**hub-signed** checkpoints (unforgeable by the monitor), which are sufficient as
the split-view comparison anchor (ADR-0001). What a cosig adds over that is narrow:
monitor non-repudiation, a one-signature "an independent party vouched" artifact,
and the C2SP witness-cosignature format — whose real consumer is **M7 gossip**,
which is out of v1.

The one place a cosig earns real v1 value is **paired with OTS**: cosign +
Bitcoin-anchor = "an independent party observed this exact root, with trustless
proof it existed before block B" — a hard-to-repudiate, independently-timestamped
witness record.

**Decision.** Keep cosigning, but treat it as part of an **Independent Attestation**
capability fused with OTS, and sequence it **after the audience-facing product is
live**. Revised milestone order:

1. **M1** — read-only Monitor (verify sig + consistency, persist, freeze, metrics)
2. **M2** — mirror + iscc_index (serves inclusion/consistency from the local mirror)
3. **M3** — Trust API + server-rendered dashboard + log browser
4. **WASM upgrade** — in-browser verifier as progressive enhancement on M3 (ADR-0003)
5. **OTS / Bitcoin anchoring** — anchor observed roots (the v1 independent
   attestation; trustless timestamp + anti-rewrite). **No cosigning.**
6. **M7 (deferred)** — multi-monitor gossip **+ cosigning** (C2SP witness
   cosignatures) + witness endpoint. Cosigning lives here because gossip is its
   only real consumer.

The original plan placed bare cosigning at M4, ahead of OTS and the dashboard —
delivering a weak-trust artifact with no consumer yet. This reorders to deliver the
headline value (a verifiable trust dashboard for business users) first, makes OTS
the v1 attestation, and parks cosigning with the gossip protocol that consumes it.
