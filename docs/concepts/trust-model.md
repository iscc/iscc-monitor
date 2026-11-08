---
title: What the monitor guarantees
description: The ISCC Monitor's trust model, what it detects on its own and the limits of its evidence
icon: lucide/scale
---

# What the monitor guarantees

The monitor is built so that nobody has to take its word for anything. This page explains what that means in
practice, and where its evidence ends.
{ .iscc-lead }

## A verifiable cache, not a trusted oracle

Everything the monitor serves can be checked against the hub's own signature and Merkle hashing. Fetching data from
the monitor is not the same as trusting it:

- A **proof bundle** holds the hub-signed checkpoint, the record and the inclusion proof. A client that verifies
    it, and resolves the hub's key itself, has removed the monitor from the trust path. This is the authoritative
    way to check an ISCC-ID; see [Verify a proof bundle](../guides/verify-bundle.md).
- The **Certificate of Inclusion** is the monitor's readable account of the same evidence. Your browser re-checks
    its inclusion proof, but the page as a whole is still something the monitor renders.
- The **verify-for-me** endpoint, `GET /<hub>/log/verify`, returns a verdict you trust the monitor for. It is
    convenient and explicitly the weaker path.

## What the monitor detects on its own

A hub that rewrites its history *against the monitor*: a log that shrinks, a different root at a size already
seen, or a checkpoint that fails a consistency proof. The monitor freezes the hub and keeps the contradicting,
hub-signed checkpoints as evidence that can never be regenerated.

## What needs a comparison

A **split view**, a hub showing one history to the monitor and another to you, is invisible to each observer
alone. The monitor makes it detectable: it publishes the log states it observed, so you can compare yours against
them; see [Compare your view of a log](../guides/split-view.md). Monitors do not yet compare notes with each other
(gossip), and they do not cosign checkpoints as witnesses.

## Coverage

Guarantees hold only from the moment a monitor started watching a hub, its **coverage start**, onward. A monitor
that starts late copies the full log and can verify it from then on, but it cannot tell whether the hub showed
different histories before anyone watched. Each hub's page and each certificate show the coverage start.

## What the evidence does not say

- **Not who owns the content.** A certificate shows that an ISCC-ID was declared in a hub's log, at a position and
    by a time. It does not vouch for the content the ISCC-ID identifies, or for the declarer's claims.
- **Not that the hub's domain is honest.** A hub's identity is its domain: the key comes from the did:web
    document the domain serves. A compromised domain is outside what the monitor can detect.
- **Not about redactions.** A hub can stop serving a declaration without writing a record. The monitor keeps the
    original declaration, but a redaction itself leaves no trace in the log.

## Why the Bitcoin anchor helps

A hub signature shows what the hub claims, at a time the hub chooses to state. An OpenTimestamps anchor confirmed
in Bitcoin fixes when a log state existed, independently of the hub and the monitor. A record in an anchored log
state cannot have been added after that block, whatever the hub does later.
