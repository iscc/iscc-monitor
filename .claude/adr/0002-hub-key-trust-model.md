---
status: superseded in part by ADR-0009
---

> **Superseded in part by ADR-0009:** the key *source* (central Hub-List) and the
> `valid_from` rotation model are replaced by did:web. The report-don't-go-dark
> behavior, the freeze orthogonality, and the status-taxonomy *idea* survive (the
> taxonomy itself is revised in ADR-0009).

# Hub key trust model: authoritative Hub-List key set, size-bounded rotation, report-don't-reject

The Hub-List is the authoritative trust root for *vouching* (ISCC-Log §2.2: a
verifier checks the signature against the Hub's public key from the Hub-List). We
make three decisions about how the monitor relates a hub's *served* signing key to
that trust root.

**1. Per-hub key set, keyed by `valid_from` tree_size.** A hub's Hub-List entry
carries an ordered list of keys, each with `valid_from` = the inclusive `tree_size`
from which it is the expected signer. The expected key for a checkpoint of size `N`
is the one with the largest `valid_from ≤ N`, giving a clean half-open partition
`[fⱼ, fⱼ₊₁)` with exactly one expected key per checkpoint. We use `valid_from`
(not `valid_until`) because activation time is known but retirement time is not, so
rotation is a pure **append** to the list. We bind to `tree_size` (not `seq`, not
wall-clock) because it is the only quantity that is both authenticated inside the
signed checkpoint and monotonic.

**2. Report mismatches, never reject silently; never cosign them.** A checkpoint
whose signature does not verify under its size-selected key is a first-class
finding, not a discard: the monitor keeps **following and mirroring** the hub (the
availability/evidence value is real and clients re-verify independently) but marks
the hub `key-mismatch` and **never cosigns or shows it as `verified`**. This makes
the monitor surface exactly the class of bug it found first (the live sb1
misconfiguration) instead of going dark on it.

**3. Per-hub status taxonomy.** `verified` · `key-mismatch` · `unverified` (no
known key verifies) · `frozen` (self-consistency violation, see ADR-0001) ·
`inactive` (Hub-List `active: false`).

## Consequences

- Requires a **Hub-List schema addition on the iscc-hub side** (the Hub-List is
  defined by the IDP Declaration Profile). Specified in
  `.claude/hublist-schema-proposal.md`. The monitor normalizes the legacy singular
  `pubkey` to a one-element list with `valid_from: 0`, so it works against today's
  `testnet.yaml` unchanged and is rotation-ready before the schema lands.
- Rotation is orthogonal to consistency/split-view: the Merkle chain is continuous
  across a key change, so only signature verification consults the key set. No
  special consistency handling.
- A retired key that later leaks can only forge in its historical `tree_size`
  range, and forging a different root there is itself a self-consistency violation
  the Merkle machinery catches — so size-bounding limits blast radius.
- Granularity: a key_id matching a *known but wrong-for-this-size* key is
  distinguishable from a *wholly unknown* key; both are `key-mismatch` but the
  finding can say which.
- "Cosign only `verified` hubs" composes with ADR-0001's cosign preconditions.
