# iscc-hub-side changes the monitor depends on (did:web trust root)

**Audience:** iscc-hub / IDP Declaration Profile / ISCC-Log spec maintainers.
**Status:** proposed by iscc-monitor (see ADR-0009). Replaces the superseded
`hublist-schema-proposal.md`.
**Premise (axiom):** domain ownership *is* a hub's cryptographic identity. Keys are
managed by the domain owner via did:web; **detecting domain compromise is out of
scope.** `iscc-monitor` only *consumes* these; the changes below live in iscc-hub.

There are three changes, in priority order.

## Change 1 — ISCC-Log §2.2: key source becomes did:web

Today §2.2 says a Verifier "verifies the checkpoint signature against the Hub's
public key **from the Hub-List**." Amend to source the key from the hub's **did:web
document** instead:

> A conforming Verifier resolves the Hub's `did:web:<domain>` document
> (`https://<domain>/.well-known/did.json`), selects the Ed25519
> `verificationMethod`, and verifies the checkpoint signed-note signature against
> it. The signed-note `name` remains the checkpoint **origin** (`<domain>/log`).

Nothing about tlog-tiles or the signed-note wire format changes — only where the
key comes from. The `key_id` / verifier-key derivation is unchanged
(`derive_vkey.py`): `key_id = SHA-256(origin || 0x0A || 0x01 || pubkey)[:4]`,
verifier key `= origin + key_id + base64(0x01 || pubkey)`, with `origin =
<domain>/log`.

### did:web document requirements (per hub)

- Publish `https://<domain>/.well-known/did.json` for `did:web:<domain>`.
- Include an Ed25519 `verificationMethod` whose public key **is the key the hub
  signs checkpoints with** (`publicKeyMultibase` `z6Mk…`, i.e. the existing
  did:key form).
- **Rotation/revocation** uses CID 1.0 semantics: add the new
  `verificationMethod`; mark the old one `revoked` (timestamp). The hub starts
  signing with the new key around the rotation.
- **Historical verification note (consequence of the axiom):** the checkpoint body
  carries **no timestamp**, so a verifier cannot precisely map an old checkpoint to
  a `revoked` instant. Because domain-compromise is out of scope, the monitor
  therefore accepts a checkpoint signed by **any Ed25519 key the DID document lists
  (current or `revoked`)**; `revoked` is informational (it records *when* a
  rotation happened), not a hard gate. Append-only correctness comes from the
  consistency chain, independent of which listed key signed. (Revisit only if
  compromise-handling is ever brought in scope.)

## Change 2 — Realm registry: domains/membership only, no keys

The registry stops carrying keys. Migration is a one-field drop (`pubkey`):

```yaml
# before (current testnet.yaml)
version: 1
network: testnet
hubs:
  - hub_id: 0
    pubkey: z6Mk...        # <-- remove; key now comes from did:web
    url: https://sb0.iscc.id
    active: true

# after
version: 2
network: testnet
hubs:
  - hub_id: 0
    url: https://sb0.iscc.id
    active: true
```

- **Keep `hub_id`.** It is the stable 12-bit hub identifier embedded in every
  ISCC-ID (52-bit timestamp + 12-bit hub), so it is load-bearing, not cosmetic.
  The registry binds `hub_id ↔ domain`; did:web binds `domain ↔ key`.
- `active: false` = paused; absence = removed. The monitor retains all evidence and
  keeps serving a removed/paused hub's mirror regardless (ADR-0007).
- The registry MAY later move from a static YAML to an ownerless on-chain registry
  (the original vision) without affecting the monitor's verification core — the
  monitor consumes membership through a pluggable resolver.

## Change 3 — §13 coverage registry (monitor-coverage)

§13 already recommends the Foundation "maintain a public monitor-coverage
registry." Make it concrete, because v1 guarantees are **coverage-bounded**
(ADR-0001): a monitor that starts late cannot retroactively detect pre-coverage
equivocation.

- For each `(monitor, hub)` pair, publish **`monitored_since` (tree_size + time)**.
- This lets clients and operators see the window over which a hub has independent
  coverage, and ("§13 cold start") not treat a just-launched hub as carrying the
  same guarantees as a long-watched one.
- This is primarily a Foundation + monitor concern; the hub side only needs to
  reference it from the coverage/trust documentation.

## Not needed anymore

The `valid_from` Hub-List key-rotation schema (`hublist-schema-proposal.md`) is
**obsolete** — rotation now lives entirely in the hub's did:web document.
