---
title: What the monitor guarantees
description: The ISCC Monitor's trust model, what it detects on its own and the limits of its evidence
icon: lucide/scale
---

# What the monitor guarantees

Nobody has to take the monitor's word for anything. This page shows what that means and where the evidence stops.
{ .iscc-lead }

## A verifiable cache, not a trusted oracle

You can check everything the monitor serves against the hub's signature and Merkle hashing. Fetching data from the
monitor is not the same as trusting it. You can check an ISCC-ID in three ways:

- **Proof bundle**: the hub-signed checkpoint, the record and the inclusion proof. If you verify it and resolve
    the hub's key yourself, the monitor is out of the trust path. This is the authoritative way to check an
    ISCC-ID. See [Verify a proof bundle](../guides/verify-bundle.md).
- **Certificate of Inclusion**: the monitor's readable account of the same evidence. Your browser re-checks its
    inclusion proof, but the monitor renders the rest of the page.
- **verify-for-me** (`GET /<hub>/log/verify`): a verdict you trust the monitor for. It is convenient, but it is
    the weaker path.

## What the monitor detects on its own

The monitor detects a hub that rewrites its history *against the monitor*. This is a self-consistency violation.
It takes one of three forms:

- The log shrinks.
- A different root appears at a size the monitor already saw.
- A checkpoint fails a consistency proof.

The monitor freezes the hub. It keeps the contradicting hub-signed checkpoints as evidence that can never be
regenerated.

## What needs a comparison

In a **split view**, a hub shows one history to the monitor and another to you. Neither observer can see it alone.
To make it detectable, the monitor publishes the log states it observed, so you can compare yours against them.
See [Compare your view of a log](../guides/split-view.md). Monitors do not yet compare notes with each other
(gossip) or cosign checkpoints as witnesses.

## Coverage

Guarantees hold only from the **coverage start**, the moment a monitor started watching a hub. A monitor that
starts late copies the full log and can verify it from then on. It cannot tell whether the hub showed different
histories before anyone watched. Each hub's page and each certificate show the coverage start.

## What the evidence does not say

| Not shown                       | Why                                                                                     |
| ------------------------------- | --------------------------------------------------------------------------------------- |
| Who owns the content            | A certificate shows that an ISCC-ID was declared in a hub's log, at a position and by a time. It does not vouch for the content the ISCC-ID identifies, or for the declarer's claims. |
| That the hub's domain is honest | A hub's identity is its domain, and its key comes from the domain's did:web document. The monitor cannot detect a compromised domain. |
| Anything about redactions       | A hub can stop serving a declaration without writing a record. The monitor keeps the declaration, but the redaction leaves no trace in the log. |

## Why the Bitcoin anchor helps

A hub signature shows what the hub claims, at a time the hub chooses to state. An OpenTimestamps anchor confirmed
in Bitcoin fixes when a log state existed, independent of the hub and the monitor. A record in an anchored log
state was in the log by that block, whatever the hub does later.
