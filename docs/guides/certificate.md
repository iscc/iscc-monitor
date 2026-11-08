---
title: Read a Certificate of Inclusion
description: What each part of the monitor's Certificate of Inclusion for an ISCC-ID shows and proves
icon: lucide/file-badge
---

# Read a Certificate of Inclusion

A Certificate of Inclusion states that an ISCC-ID is recorded in the transparency log of the hub that issued it,
and lays out the evidence for it, clause by clause.
{ .iscc-lead }

## Open a certificate

Enter the ISCC-ID on the start page of a [monitor instance](../index.md#public-instances), or open
`/inclusion/<ISCC-ID>` on the instance:

```text
https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB
```

The ISCC-ID can be written in any letter case, with or without the `ISCC:` or `did:iscc:` prefix. You do not need
to know the hub: an ISCC-ID names its network and the hub that issued it, and the monitor finds that hub in the
network's Hub-List.

Use the instance for the ISCC-ID's network: a mainnet ISCC-ID on `monitor.iscc.io`, a testnet ISCC-ID on
`monitor-test.iscc.io`.

## The header

The header names the ISCC-ID, the hub whose log records it and its position in that log. A certificate that proves
inclusion carries the **ISCC Trust Seal**, which repeats the ISCC-ID and the date the page was issued; the time
appears beneath it.

The seal and the certificate are the monitor's account of the evidence. They vouch that the ISCC-ID is in the log,
not for the content the ISCC-ID identifies or for who declared it.

## The clauses

§1 Subject
:   The ISCC-ID and the position (*seq*) of the log record that commits it.

§2 Checkpoint
:   The hub's log state the certificate refers to: its size and Merkle root. This is the latest checkpoint the
    monitor accepted for the hub, signed by the hub and checked by the monitor.

§3 Inclusion Proof
:   The sibling hashes that rebuild the §2 root from the record. The monitor rebuilt the root before rendering
    the page, and with JavaScript enabled your browser checks the proof again.

§4 Signing Key
:   The hub's Ed25519 key that signed the §2 checkpoint, as published in the hub's did:web document at
    `https://<hub>/.well-known/did.json`.

§5 Bitcoin Anchor
:   The earliest OpenTimestamps proof, confirmed in a Bitcoin block, of a log state that includes the record. It
    shows that the record existed by the time of that block. The clause links the proof file and the
    `ots verify` command to check it; see [Check a Bitcoin anchor](bitcoin-anchor.md). Until a confirmed anchor
    covers the record, the clause shows the timestamp state of the hub's newest root, usually pending. A Bitcoin
    confirmation takes a few hours.

Comparison Anchor
:   The log state the monitor observed from the hub, and since when it has watched it. If a hub showed you a
    different history, comparing your own log state against this one reveals it; see
    [Compare your view of a log](split-view.md).

§6 Record History
:   Every record in the log that names this ISCC-ID: the declaration and any later deletion. A deletion is a new
    record; the original declaration stays in the log.

## When the monitor cannot certify

For an ISCC-ID that is malformed, belongs to an unknown hub, or is not (yet) in the log the monitor has accepted,
the page says so instead of showing clauses. A newly declared ISCC-ID appears after the monitor's next poll of
its hub, within five minutes by default.

## Keep the evidence

The certificate is a page rendered by one monitor. The **proof bundle** behind the "Download proof bundle"
button holds the same evidence as a JSON file: the hub-signed checkpoint, the inclusion proof, the record and the
hub's key, plus the Bitcoin anchor evidence when there is one. You can store it, pass it on and check it without
the monitor; see [Verify a proof bundle](verify-bundle.md).
