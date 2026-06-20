---
status: accepted
---

# Trust root = did:web (domains-only registry); domain-compromise out of scope

Supersedes the key-source parts of ADR-0002 and the `valid_from` Hub-List proposal.

The hub's signing key is obtained from the hub's **did:web** document
(`did:web:<domain>` → `https://<domain>/.well-known/did.json`), not from a central
key registry. The realm registry advertises **domains/membership only** — no keys.
Key management (current key, rotation, revocation) is the **domain owner's**
responsibility, expressed natively in the DID document (CID 1.0
`verificationMethod` + `revoked` validity windows). This holds for **both testnet
and mainnet**.

**Axiom:** domain ownership *is* the hub's cryptographic identity. **Detecting
domain compromise is explicitly out of scope** — if an attacker controls the
domain (DNS/TLS), they control the identity, by definition.

## Conformance

tlog-tiles conformant: the signed-note format only requires the verifier to
*possess* the Ed25519 key; it is silent on the key's source. The checkpoint
signed-note `name` stays the origin (`<domain>/log`); did:web supplies the key.
`derive_vkey.py`'s did→verifier-key derivation is unchanged — only the *source*
moves from YAML to `/.well-known/did.json`. Requires an **ISCC-Log §2.2
amendment** ("key from the Hub-List" → "key from the hub's did:web document"),
which is an iscc-hub-side edit, not a tlog-tiles deviation.

## Consequences

- **Status taxonomy simplifies** (revises ADR-0002): no registry key to mismatch
  against, so `key-mismatch` collapses to `unresolvable` (can't fetch/parse
  `did.json`) and `unverified` (signature doesn't match the hub's *own* did:web
  key = an internally-broken hub). `verified` / `unresolvable` / `unverified` /
  `frozen` / `inactive`.
- **sb1 verifies as valid** under this model (its did.json matches its signing
  key); the prior "misconfiguration" verdict only existed relative to a central
  registered baseline, which is now gone by design.
- **`valid_from` machinery is superseded** by did:web `revoked` validity windows;
  `.claude/hublist-schema-proposal.md` is marked superseded.
- **Resolver abstraction retained:** the monitor consumes `origin → {key,
  validity}` from a pluggable resolver (did:web for v1). A future on-chain
  key-commitment resolver could be added without touching the verification core —
  but is **not** pursued, per the domain-ownership axiom above.
- **Retained from ADR-0002:** report-don't-go-dark behavior (surface
  `unresolvable`/`unverified` as findings, keep mirroring), and the orthogonality
  of key resolution from the consistency/freeze logic (ADR-0006).
