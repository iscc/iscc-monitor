---
title: Verify a proof bundle
description: Check an ISCC Monitor proof bundle on your own machine, without trusting the monitor
icon: lucide/shield-check
---

# Verify a proof bundle

A proof bundle carries everything needed to check that an ISCC-ID is in a hub's log. Verifying it yourself takes
the monitor out of the trust path: the result rests on the hub's signature and on Merkle hashing, not on the
monitor's word.
{ .iscc-lead }

## 1. Download the bundle

Click "Download proof bundle" on the [certificate](certificate.md), or fetch it:

```bash
curl -OJ https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB.bundle
```

This saves `MEIGKV5N6NTVF4AB.bundle.json`. [Proof bundle format](../reference/proof-bundle.md) describes its
members. The monitor offers a bundle only for an ISCC-ID whose inclusion it has verified itself; for any other it
answers `{"error": "no proof bundle available for this id"}`.

## 2. Run the verifier

[`verify_bundle.py`](verify_bundle.py) is a short, self-contained Python script that checks a bundle. Read it
before you rely on it; that is the point. With [uv](https://docs.astral.sh/uv/) it runs in one step, because it
declares its one dependency (`cryptography`, for Ed25519) inline:

```bash
uv run verify_bundle.py --resolve MEIGKV5N6NTVF4AB.bundle.json
```

Without uv, install `cryptography` and run it with Python 3.10 or newer. A bundle that passes prints:

```text
ok  record names ISCC:MEIGKV5N6NTVF4AB
ok  checkpoint: signed by iscc.id/log key 68c3b89a, record at 0 of 220
ok  anchor: signed by iscc.id/log key 68c3b89a, record at 0 of 1
    check the Bitcoin anchor: ots verify -d 951420d087ae7ff1ac814091012ac68a89736f0f137f74c62dce7503ddaebb80 MEIGKV5N6NTVF4AB.anchor.ots
ok  did:web:iscc.id lists the signing key
```

Any failed check prints `FAIL` with the reason and exits with status 1.

## What the script checks

1. **The record names the ISCC-ID.** The record is the exact bytes the hub logged; it is JSON with an `iscc_id`
    field.
1. **The hub signed the checkpoint.** The checkpoint is a [signed note](https://c2sp.org/signed-note) with the
    log's origin, its size and its Merkle root. The script checks the Ed25519 signature with the bundle's key.
1. **The record is in that log state.** It hashes the record into an RFC 6962 leaf, applies the inclusion proof
    and compares the result with the checkpoint's root.
1. **The Bitcoin anchor covers the record.** When the bundle has an `anchor` member, the script checks its
    checkpoint and inclusion proof the same way, then writes the OpenTimestamps proof to a `.anchor.ots` file
    and prints the command to [check it](bitcoin-anchor.md).
1. **The key is the hub's own** (with `--resolve`). The key in the bundle comes from the monitor. `--resolve`
    fetches the hub's did:web document, `https://<hub>/.well-known/did.json`, and checks that it lists that key.
    Without it, the signature check only shows that the checkpoint matches the key the monitor supplied.

## What a passing bundle proves, and what it does not

A passing bundle proves that the hub signed a log state that contains this record at this position. With a
confirmed anchor, `ots verify` adds that the record existed by the time of a specific Bitcoin block.

It does not prove that the hub showed everyone the same log. That needs a comparison between independent
observers; see [Compare your view of a log](split-view.md). And it says nothing about the content the ISCC-ID
identifies beyond what the record declares.

## The script

```python title="verify_bundle.py"
--8<-- "docs/guides/verify_bundle.py"
```
