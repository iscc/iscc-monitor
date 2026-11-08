---
title: Proof bundle format
description: The JSON members of an ISCC Monitor proof bundle and how they fit together
icon: lucide/package-check
---

# Proof bundle format

A proof bundle is one JSON file with the evidence that an ISCC-ID is in its hub's log. It is the authoritative way
to check an ISCC-ID. A client verifies it on its own, without trusting the monitor.
{ .iscc-lead }

Get it from `GET /inclusion/<iscc_id>.bundle`, or with the "Download proof bundle" button on a certificate.
[Verify a proof bundle](../guides/verify-bundle.md) shows how to check one. Example:
[`MEIGKV5N6NTVF4AB.bundle.json`](MEIGKV5N6NTVF4AB.bundle.json).

## Members

`iscc_id`
:   The ISCC-ID, in its canonical `ISCC:` form.

`hub`
:   The hub whose log has the record: its `domain` and its did:web identifier `did`. The hub publishes its key at
    `https://<domain>/.well-known/did.json`.

`checkpoint`
:   The latest checkpoint the monitor accepted for the hub, as the hub signed it. It is a
    [signed note](https://c2sp.org/signed-note): the log's origin, size and base64 Merkle root, then a blank line
    and the signature line.

`inclusion`
:   The inclusion proof of the record against `checkpoint`, in the hub's `IsccLogInclusionProof` shape:

    | Member           | Content                                                  |
    | ---------------- | -------------------------------------------------------- |
    | `type`           | `"IsccLogInclusionProof"`                                |
    | `checkpoint`     | The same signed checkpoint                               |
    | `treeSize`       | The checkpoint's size                                    |
    | `leafIndex`      | The record's position in the log, counted from 0         |
    | `inclusionProof` | The RFC 6962 sibling hashes, base64, from the leaf upward |

`record`
:   The record exactly as the hub logged it, base64-encoded. Its RFC 6962 leaf hash is the SHA-256 of a zero byte
    followed by these bytes.

`key`
:   The key that signed `checkpoint`. `id` is its 4-byte signed-note key hash in hex. `multibase` is the Ed25519
    public key as the hub's did:web document lists it (`publicKeyMultibase`).

`anchor` (optional)
:   Present when a confirmed Bitcoin anchor covers the record. It has the evidence for the earliest such anchor.

    | Member       | Content                                                             |
    | ------------ | ------------------------------------------------------------------- |
    | `checkpoint` | The hub-signed checkpoint at the anchored size                      |
    | `inclusion`  | The record's inclusion proof against that checkpoint                 |
    | `key`        | The key that signed that checkpoint                                 |
    | `ots`        | The OpenTimestamps proof of that checkpoint's root, base64           |

## How the parts fit

1. Hash `record` to a leaf. `inclusion.inclusionProof` rebuilds the root of a tree of `treeSize` from that leaf at
    `leafIndex`.
1. The rebuilt root must equal the root in `checkpoint`.
1. `checkpoint` must have a valid signature by `key`.
1. `key` must be in the hub's did:web document. Check this yourself, because the bundle's copy comes from the
    monitor.
1. For `anchor`, repeat steps 1 to 4 with its `checkpoint`, `inclusion` and `key`. Then
    `ots verify -d <root in hex>` checks its `ots` proof against Bitcoin.

## No bundle

If the monitor cannot certify an ISCC-ID, the route returns `200` with:

```json
{"error": "no proof bundle available for this id", "iscc_id": "..."}
```

The monitor offers a bundle only after it has verified the inclusion proof against its accepted root. It never
returns a partial bundle.
