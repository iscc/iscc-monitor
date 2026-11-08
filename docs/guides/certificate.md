---
title: Read a Certificate of Inclusion
description: What each part of the monitor's Certificate of Inclusion for an ISCC-ID shows and proves
icon: lucide/file-badge
---

# Read a Certificate of Inclusion

A Certificate of Inclusion shows, clause by clause, the evidence that an ISCC-ID is in the transparency log of the
hub that issued it.
{ .iscc-lead }

## Open a certificate

Enter the ISCC-ID on the start page of a [monitor instance](../index.md#public-instances), or open
`/inclusion/<ISCC-ID>`:

```text
https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB
```

Any letter case works, with or without the `ISCC:` or `did:iscc:` prefix. You do not need to know the hub. The
ISCC-ID names its network and hub, and the monitor finds the hub in the network's Hub-List.

Use the instance for the ISCC-ID's network: `monitor.iscc.io` for mainnet, `monitor-test.iscc.io` for testnet.

## The header

The header shows the ISCC-ID, the hub whose log has it, and its position in that log. If the certificate proves
inclusion, it has the **ISCC Trust Seal**. The seal repeats the ISCC-ID and the issue date, with the time beneath
it.

The seal and the certificate are the monitor's account of the evidence. They vouch only that the ISCC-ID is in the
log, not for the content it identifies or for who declared it.

## The clauses

§1 Subject
:   The ISCC-ID and the position (*seq*) of the log record that commits it.

§2 Checkpoint
:   The size and Merkle root of the hub's log that the certificate refers to. This is the latest checkpoint the
    monitor accepted for the hub. The hub signed it and the monitor checked it.

§3 Inclusion Proof
:   The sibling hashes that rebuild the §2 root from the record. The monitor rebuilt the root before it rendered
    the page. With JavaScript on, your browser checks the proof again.

§4 Signing Key
:   The hub's Ed25519 key that signed the §2 checkpoint, from the hub's did:web document at
    `https://<hub>/.well-known/did.json`.

§5 Bitcoin Anchor
:   The earliest OpenTimestamps proof, confirmed in a Bitcoin block, of a log state that includes the record. It
    shows that the record existed by the time of that block. The clause links the proof file and the `ots verify`
    command; see [Check a Bitcoin anchor](bitcoin-anchor.md). Until a confirmed anchor covers the record, the
    clause shows the timestamp state of the hub's newest root, usually pending. Bitcoin confirmation takes a few
    hours.

Comparison Anchor
:   The log state the monitor observed from the hub, and since when it has watched the hub. If the hub showed you
    a different history, a comparison with this log state finds it. See
    [Compare your view of a log](split-view.md).

§6 Record History
:   Every record in the log that names this ISCC-ID: the declaration and any later deletion. A deletion is a new
    record. The original declaration stays in the log.

## When the monitor cannot certify

If the ISCC-ID is malformed, belongs to an unknown hub, or is not (yet) in the log the monitor accepted, the page
says so and shows no clauses. A newly declared ISCC-ID appears after the monitor's next poll of its hub, by
default within five minutes.

## Keep the evidence

The certificate is a page from one monitor. The "Download proof bundle" button gives you the same evidence as one
JSON file, the **proof bundle**. It holds the hub-signed checkpoint, the inclusion proof, the record and the hub's
key. If there is a Bitcoin anchor, it holds that evidence too. You can store the bundle, pass it on and check it
without the monitor. See [Verify a proof bundle](verify-bundle.md).
