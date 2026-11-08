---
title: How it works
description: How the ISCC Monitor follows, verifies, mirrors and anchors the transparency logs of ISCC hubs
icon: lucide/workflow
---

# How it works

An ISCC hub issues ISCC-IDs and records each declaration in a public, append-only transparency log. The monitor
watches those logs from the outside. It runs one loop per network: follow, verify, mirror, anchor, publish.
{ .iscc-lead }

## Roles

| Role             | What it does                                                                                       |
| ---------------- | -------------------------------------------------------------------------------------------------- |
| Hub              | Issues ISCC-IDs and logs every declaration in a [tlog-tiles](https://c2sp.org/tlog-tiles) transparency log, a Merkle tree served as static files. It signs its log state with an Ed25519 key from its did:web document. |
| Hub-List         | The authoritative list of a network's hubs, in the [iscc-hub repository](https://github.com/iscc/iscc-hub/tree/main/hubs). It gives each hub's domain and 12-bit `hub_id`. Every ISCC-ID the hub issues carries this number. |
| Monitor instance | Follows every hub on the Hub-List, checks their logs and publishes what it saw. Anyone can run one, and each checks the hubs on its own. |
| Client           | A person or program that looks up ISCC-IDs on a monitor and verifies the evidence.                |

## The loop

### 1. Follow

Every five minutes by default, the monitor fetches each hub's latest checkpoint: the log's size and Merkle root,
signed by the hub. The monitor re-reads the Hub-List every hour, so a changed hub entry takes
effect without a restart. The monitor follows a newly listed hub after the next restart.

### 2. Verify

The monitor accepts a new checkpoint only if:

1. Its signature verifies under the hub's key from `https://<hub>/.well-known/did.json`.
2. An RFC 6962 consistency proof shows that it extends the last accepted checkpoint. No record was changed or
    removed.
3. The monitor can rebuild its Merkle root from the fetched log tiles.

### 3. Mirror

The monitor stores every hub's tiles and entry bundles in one SQLite file per network and serves them at the hub's
own paths. The mirror keeps a log available and verifiable if the hub goes offline. Full tiles never change, so
the monitor fetches each one once. Its traffic grows with the log's growth, not with its size.

### 4. Anchor

Every hour, and at startup, the monitor submits newly accepted roots to an
[OpenTimestamps](https://opentimestamps.org) calendar and collects the Bitcoin confirmations of earlier ones. A
confirmed timestamp shows that a log state existed by a given Bitcoin block.

### 5. Publish

For any ISCC-ID in the network, the monitor shows a Certificate of Inclusion and offers a proof bundle. Its
[HTTP API](../reference/http-api.md) serves the mirrored log, inclusion and consistency proofs, and a summary
verdict.

## Hub status

Each hub has one of five statuses, shown on the instance's pages and in its metrics:

| Status           | Meaning                                                                                              |
| ---------------- | ---------------------------------------------------------------------------------------------------- |
| **verified**     | The latest checkpoint is signed with the hub's did:web key, consistent with earlier ones and rebuilt |
| **unresolvable** | The monitor cannot fetch or parse the hub's did:web document                                         |
| **unverified**   | The checkpoint's signature does not match the hub's own did:web key                                  |
| **frozen**       | The hub's log contradicted itself. The monitor stopped accepting its checkpoints and kept the evidence |
| **inactive**     | The hub is removed or paused in the Hub-List                                                         |

The monitor follows and mirrors every hub, whatever its status. Only a *verified* hub's checkpoints advance the
accepted state.

The monitor freezes a hub when its log shrinks, shows a different root at a size already seen, or fails a
consistency proof. Each is a self-consistency violation: a rewritten history. The monitor keeps both
contradicting hub-signed checkpoints as permanent evidence. It logs a warning and keeps polling the hub at a slower
pace. It does not lift a freeze automatically.

## Records and ISCC-IDs

A log record is a **declaration**, which commits an ISCC-ID, or a **deletion**, a later record that marks a
declaration as deleted. A deletion never removes the declaration from the log, so the monitor always keeps it. A
hub can also *redact* a declaration from its own responses. A redaction writes no record, and the monitor never
sees it.

Verification does not interpret record contents. The monitor reads them only to index ISCC-IDs and to label records
on its pages.
