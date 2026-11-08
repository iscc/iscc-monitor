---
title: Verify a proof bundle
description: Check an ISCC Monitor proof bundle on your own machine, without trusting the monitor
icon: lucide/shield-check
---

# Verify a proof bundle

A proof bundle has all you need to check that an ISCC-ID is in a hub's log. If you verify it yourself, the
result rests on the hub's signature and on Merkle hashing, not on the monitor's word.
{ .iscc-lead }

## 1. Download the bundle

Click "Download proof bundle" on the [certificate](certificate.md), or fetch it:

```bash
curl -OJ https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB.bundle
```

This saves `MEIGKV5N6NTVF4AB.bundle.json`. Its members are in [Proof bundle format](../reference/proof-bundle.md).
The monitor gives a bundle only for an ISCC-ID whose inclusion it verified itself. For any other ID it answers
`{"error": "no proof bundle available for this id"}`.

## 2. Run the verifier

[`verify_bundle.py`](verify_bundle.py) is a short, self-contained Python script. Read it before you rely on it;
that is the point. It declares its one dependency (`cryptography`, for Ed25519) inline, so
[uv](https://docs.astral.sh/uv/) runs it in one step:

```bash
uv run verify_bundle.py --resolve MEIGKV5N6NTVF4AB.bundle.json
```

Without uv, install `cryptography` and use Python 3.10 or newer. A bundle that passes prints:

```text
ok  record names ISCC:MEIGKV5N6NTVF4AB
ok  checkpoint: signed by iscc.id/log key 68c3b89a, record at 0 of 220
ok  anchor: signed by iscc.id/log key 68c3b89a, record at 0 of 1
    check the Bitcoin anchor: ots verify -d 951420d087ae7ff1ac814091012ac68a89736f0f137f74c62dce7503ddaebb80 MEIGKV5N6NTVF4AB.anchor.ots
ok  did:web:iscc.id lists the signing key
```

A failed check prints `FAIL` with the reason and exits with status 1.

## What the script checks

1. The record names the ISCC-ID. The record is the exact bytes the hub logged, JSON with an `iscc_id` field.
1. The hub signed the checkpoint. The checkpoint is a [signed note](https://c2sp.org/signed-note) with the log's
    origin, size and Merkle root. The script checks its Ed25519 signature with the bundle's key.
1. The record is in that log state. The script hashes the record into an RFC 6962 leaf, applies the inclusion
    proof and compares the result with the checkpoint's root.
1. The Bitcoin anchor covers the record, if the bundle has an `anchor` member. The script checks its checkpoint
    and inclusion proof the same way. It writes the OpenTimestamps proof to a `.anchor.ots` file and prints the
    command to [check it](bitcoin-anchor.md).
1. The key is the hub's own, with `--resolve`. The bundle's key comes from the monitor. `--resolve` fetches the
    hub's did:web document, `https://<hub>/.well-known/did.json`, and checks that it lists the key. Without it,
    step 2 shows only that the checkpoint matches the key the monitor gave.

## What a passing bundle proves

A passing bundle proves that the hub signed a log state with this record at this position. With a confirmed
anchor, `ots verify` adds that the record existed by the time of a specific Bitcoin block.

It does not prove that the hub showed everyone the same log. That needs a comparison between independent
observers; see [Compare your view of a log](split-view.md). It says nothing about the content the ISCC-ID
identifies beyond what the record declares.

## The script

```python title="verify_bundle.py"
--8<-- "docs/guides/verify_bundle.py"
```
