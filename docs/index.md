---
title: ISCC Monitor
description: Independent Trust & Transparency service for the ISCC-Hub network, with verifiable evidence for every ISCC-ID
icon: lucide/house
hide:
  - toc
---

# ISCC Monitor

## Evidence for every ISCC-ID that you can check yourself

ISCC hubs record every ISCC-ID they issue in a public transparency log. The ISCC Monitor watches those logs from
the outside: it follows every hub of a network, checks each hub's signatures and that its log only ever grows,
keeps a complete copy, and publishes evidence about any ISCC-ID that anyone can check without trusting the hub or
the monitor.
{ .iscc-lead }

[Check an ISCC-ID](guides/certificate.md){ .iscc-btn .iscc-btn--primary }
[How it works](concepts/how-it-works.md){ .iscc-btn }

*Open source · Apache-2.0 · maintained by the [ISCC Foundation](https://iscc.io)*

## Find your path

<div class="iscc-cards" markdown>

<div class="iscc-card" markdown>

### Anyone with an ISCC-ID

Open the Certificate of Inclusion for an ISCC-ID and read what it proves: which hub recorded it, where in the
log, and since when Bitcoin attests that it existed.

[Read a certificate →](guides/certificate.md)

</div>

<div class="iscc-card" markdown>

### Developers and auditors

Download a proof bundle and verify it on your own machine, check Bitcoin timestamps, and compare your view of a
hub's log against the monitor's.

[Verify a proof bundle →](guides/verify-bundle.md)

</div>

<div class="iscc-card" markdown>

### Operators

Run your own monitor instance for mainnet or testnet from the container image, under your own name.

[Deploy an instance →](operate/deploy.md)

</div>

</div>

## Public instances

The ISCC Foundation runs one monitor instance per network. Anyone can run another one; every instance checks the
hubs on its own.

| Network | Instance                        | API reference                        |
| ------- | ------------------------------- | ------------------------------------ |
| Mainnet | <https://monitor.iscc.io>       | <https://monitor.iscc.io/docs>       |
| Testnet | <https://monitor-test.iscc.io>  | <https://monitor-test.iscc.io/docs>  |

To look up an ISCC-ID, enter it on the instance's start page, or open `/inclusion/<ISCC-ID>` directly, for
example [`https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB`](https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB).

## What the monitor does

- **Follows** the transparency log of every hub listed in the network's
    [Hub-List](https://github.com/iscc/iscc-hub/tree/main/hubs), every five minutes by default.
- **Verifies** each new checkpoint: the hub's Ed25519 signature under the key in its did:web document, and an
    RFC 6962 consistency proof showing that the log only grew. A hub that rewrites its history is frozen and the
    evidence is kept.
- **Mirrors** every hub's log, so its records stay available and verifiable if the hub goes offline.
- **Anchors** checkpoint roots in Bitcoin with OpenTimestamps.
- **Publishes** a Certificate of Inclusion and a downloadable proof bundle for any ISCC-ID in the network, plus
    the raw log data and proofs over an [HTTP API](reference/http-api.md).

The monitor is a *verifiable cache*, not a trusted oracle: everything it serves comes with the evidence to check
it. [What the monitor guarantees](concepts/trust-model.md) explains where that holds and where it ends.
