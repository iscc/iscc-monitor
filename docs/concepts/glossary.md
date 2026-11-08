---
title: Glossary
description: Terms used by the ISCC Monitor and its documentation
icon: lucide/book-a
---

# Glossary

The terms the monitor's pages, API and documentation use, and what they mean here.
{ .iscc-lead }

Bitcoin anchoring
:   Timestamping a checkpoint root in the Bitcoin chain with OpenTimestamps. "Anchoring" refers only to this, never
    to the comparison anchor.

Checkpoint
:   A hub's signed statement of its log state: the log's origin, its size and its Merkle root, in the
    [signed-note](https://c2sp.org/signed-note) format.

Comparison anchor
:   The monitor's role as an independent record of what each hub showed it, so a client can compare its own log
    state against it and detect a split view.

Consistency proof
:   The hashes that show a smaller log state is a prefix of a larger one, so nothing was changed or removed
    (RFC 6962).

Coverage
:   The window from `monitored_since` (the size and time at which the monitor started watching a hub) to now. All
    guarantees hold only within it.

Declaration
:   A log record that commits an ISCC-ID. The default subject of an inclusion proof.

Deletion
:   A later log record that carries an existing ISCC-ID and marks its declaration as deleted. It never removes the
    declaration from the log.

did:web
:   The method by which a hub publishes its signing key: `did:web:<domain>` resolves to
    `https://<domain>/.well-known/did.json`. The domain's owner manages and rotates the key there.

Hub
:   A service that issues ISCC-IDs and records them in its transparency log.

Hub-List
:   The authoritative list of a network's hubs, with each hub's domain and 12-bit `hub_id`, published in the
    [iscc-hub repository](https://github.com/iscc/iscc-hub/tree/main/hubs).

Hub status
:   The monitor's verdict on a hub: verified, unresolvable, unverified, frozen or inactive. See
    [Hub status](how-it-works.md#hub-status).

Inclusion proof
:   The sibling hashes that rebuild a log's Merkle root from one record, showing that the record is in that log
    state (RFC 6962).

Irreplaceable evidence
:   What the monitor can never regenerate: the checkpoints it observed, contradicting checkpoint pairs and
    OpenTimestamps proofs. Backups exist to protect it. The mirror, by contrast, can be fetched again.

Mirror
:   The monitor's complete copy of a hub's log tiles and entry bundles, served at the hub's own paths.

Monitor instance
:   One running deployment of the monitor at its own domain, such as `monitor.iscc.io`. Anyone can run one.

Proof bundle
:   A JSON file with the hub-signed checkpoint, the inclusion proof, the record bytes and the hub's key, plus the
    Bitcoin anchor evidence when there is one, that a client verifies on its own. See
    [Proof bundle format](../reference/proof-bundle.md).

Redaction
:   A hub's local choice to stop serving a declaration. It writes no log record, so the monitor never sees it and
    keeps the declaration.

Self-consistency violation
:   Successive checkpoints from one hub that cannot belong to one log: the hub rewrote its history against this
    monitor. The monitor detects this on its own and freezes the hub.

Split view
:   A hub showing different, internally consistent histories to different audiences. Detectable by comparing
    views, for example your own against a monitor's.

Verifiable cache
:   What the monitor is, as opposed to a trusted oracle: clients re-verify whatever it serves against the hub's
    signature and Merkle hashing.

verify-for-me
:   The API endpoint whose verdict you trust the monitor for. The weaker path compared with verifying a proof
    bundle yourself.
