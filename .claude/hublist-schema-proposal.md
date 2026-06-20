# Hub-List schema proposal: graceful signing-key rotation

> **SUPERSEDED (2026-06-20) by ADR-0009.** Key management moved to did:web: the
> domain owner manages keys (and rotation/revocation via CID 1.0
> `verificationMethod.revoked`) in `/.well-known/did.json`. The registry advertises
> domains/membership only, no keys, so this `valid_from` schema is no longer
> needed. Kept for history.

**Audience:** iscc-hub / IDP Declaration Profile maintainers (the Hub-List is
defined there; `iscc-monitor` only consumes it).
**Status:** superseded by ADR-0009.
**Goal:** let a hub rotate its checkpoint signing key without ever losing its
`verified` status on monitors — and without anyone having to predict, in advance,
*when* a key will be retired.

## Problem

Today each hub entry carries a single `pubkey`:

```yaml
version: 1
network: testnet
hubs:
  - hub_id: 0
    pubkey: z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ
    url: https://sb0.iscc.id
    active: true
```

With a single key there is an unavoidable window during rotation where the hub
signs with the new key but the Hub-List still names the old one (or vice-versa). A
spec-conforming monitor flags every checkpoint in that window as a key mismatch —
training operators to ignore the very alert that also fires for a real attack.

## Solution: a per-hub ordered key list keyed by `valid_from`

Add an optional `keys` list to a hub entry. Each entry is a public key plus the
`valid_from` **tree_size** at which it becomes the expected signer.

```yaml
version: 1
network: testnet
hubs:
  - hub_id: 0
    url: https://sb0.iscc.id
    active: true
    keys:
      - pubkey: z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ
        valid_from: 0          # genesis key
      - pubkey: z6Mk...NEW...   # the rotated-in key
        valid_from: 250000      # takes over at tree_size 250000
```

### Field semantics

- **`pubkey`** — the hub's Ed25519 public key as a `z6Mk…` did:key multibase
  string, identical in form to today's singular `pubkey`.
- **`valid_from`** — an integer **tree_size** (the value on line 2 of the signed
  checkpoint body), **inclusive**. This key is the expected signer for every
  checkpoint whose `tree_size` is `≥ valid_from` and `<` the next key's
  `valid_from`. NOTE: this is a `tree_size`, **not** a record `seq` (they differ by
  one: a tree of size `N` commits records `0..N-1`).

### Validation rules (MUST)

1. A hub entry has **either** `pubkey` (single key, never rotated) **or** `keys`
   (one or more). Not both.
2. `keys` is ordered by ascending `valid_from`; `valid_from` values are **strictly
   increasing** (no duplicates).
3. The first key's `valid_from` is `0` (the genesis key signs from the start).
4. `pubkey: X` is exactly equivalent to `keys: [{pubkey: X, valid_from: 0}]`. A
   consumer normalizes the singular form to that.

### How a verifier/monitor uses it

For a checkpoint of size `N`:
1. Select the expected key = the one with the **largest `valid_from ≤ N`**.
2. Confirm the checkpoint signature line's 4-byte key id equals
   `SHA-256(origin || 0x0A || 0x01 || pubkey)[:4]` for that key, then verify the
   Ed25519 signature over the checkpoint body.
3. If it verifies → the checkpoint is attributable to the hub. If the signature
   matches a *different* listed key than the size-selected one, or no listed key →
   the monitor records a `key-mismatch` finding (it does not silently drop the
   checkpoint).

## Rotation procedure (operator runbook)

1. Generate the new keypair `K2`.
2. Pick a `valid_from` boundary **comfortably ahead** of the current tree_size
   (enough lead time for all monitors to re-fetch the Hub-List — e.g. current size
   + a margin sized to your append rate × the monitor poll interval).
3. **Append** `K2` to the hub's `keys` with that `valid_from`, and publish the
   Hub-List update. Do **not** edit or remove `K1`.
4. When the log reaches the boundary tree_size, switch the hub to sign with `K2`.
   Checkpoints `< boundary` remain `K1`-signed; `≥ boundary` are `K2`-signed.
5. `K1` stays in the list permanently so its historical checkpoints remain
   verifiable.

Because the boundary is in the future and authenticated by the checkpoint's own
`tree_size`, there is no clock dependence and no ambiguous overlap window.

## Backward compatibility

- Hubs that have never rotated keep their single `pubkey` unchanged.
- No `version` bump is required: `keys` is purely additive and opt-in.
- A future HA-signing need (two signers valid in the same range) can be met by
  later adding an optional `valid_until` to *widen* a key's range — backward
  compatible, and explicitly out of scope now (the spec mandates one writer per
  log).
