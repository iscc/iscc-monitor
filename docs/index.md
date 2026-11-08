---
title: ISCC Monitor
description: Independent Trust & Transparency service for the ISCC-Hub network, with verifiable evidence for every ISCC-ID
icon: lucide/house
hide:
  - toc
---

# ISCC Monitor

## Evidence for every ISCC-ID that you can check yourself

ISCC hubs record every ISCC-ID they issue in a public transparency log. The ISCC Monitor follows every hub's log
from the outside. It checks the signatures, checks that each log only grows, and keeps a complete copy. For any
ISCC-ID, it publishes evidence that you can check without trusting the hub or the monitor.
{ .iscc-lead }

[Check an ISCC-ID](guides/certificate.md){ .iscc-btn .iscc-btn--primary }
[How it works](concepts/how-it-works.md){ .iscc-btn }

*Open source · Apache-2.0 · maintained by the [ISCC Foundation](https://iscc.io)*

## Find your path

<div class="iscc-cards" markdown>

<div class="iscc-card" markdown>

### Anyone with an ISCC-ID

Open the Certificate of Inclusion for an ISCC-ID. It shows which hub recorded the ID, where in the log, and since
when Bitcoin attests that it existed.

[Read a certificate →](guides/certificate.md)

</div>

<div class="iscc-card" markdown>

### Developers and auditors

Download a proof bundle and verify it on your own machine. Check Bitcoin timestamps. Compare your view of a hub's
log against the monitor's.

[Verify a proof bundle →](guides/verify-bundle.md)

</div>

<div class="iscc-card" markdown>

### Operators

Run your own monitor instance for mainnet or testnet from the container image, under your own name.

[Deploy an instance →](operate/deploy.md)

</div>

</div>

## Public instances

The ISCC Foundation runs one monitor instance per network. Anyone can run another one, and every instance checks
the hubs on its own.

| Network | Instance                        | API reference                        |
| ------- | ------------------------------- | ------------------------------------ |
| Mainnet | <https://monitor.iscc.io>       | <https://monitor.iscc.io/docs>       |
| Testnet | <https://monitor-test.iscc.io>  | <https://monitor-test.iscc.io/docs>  |

To look up an ISCC-ID, enter it on the instance's start page or open `/inclusion/<ISCC-ID>`, for example
[`https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB`](https://monitor.iscc.io/inclusion/ISCC:MEIGKV5N6NTVF4AB).

## What the monitor does

- Follows the transparency log of every hub in the network's
    [Hub-List](https://github.com/iscc/iscc-hub/tree/main/hubs), every five minutes by default.
- Verifies each new checkpoint. It checks the hub's Ed25519 signature under the key in its did:web document, and
    an RFC 6962 consistency proof that the log only grew. It freezes a hub that rewrites its history and keeps the
    evidence.
- Mirrors every hub's log, so the records stay available and verifiable if the hub goes offline.
- Anchors checkpoint roots in Bitcoin with OpenTimestamps.
- Publishes a Certificate of Inclusion and a proof bundle for any ISCC-ID in the network. Its
    [HTTP API](reference/http-api.md) serves the raw log data and proofs.

The monitor is a *verifiable cache*, not a trusted oracle. Everything it serves comes with the evidence to check
it. [What the monitor guarantees](concepts/trust-model.md) shows where that holds and where it ends.
