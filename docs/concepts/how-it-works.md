---
title: How it works
description: How the ISCC Monitor follows, verifies, mirrors and anchors the transparency logs of ISCC hubs
icon: lucide/workflow
---

# How it works

An ISCC hub issues ISCC-IDs and records each declaration in a public, append-only transparency log. The monitor is
an outside observer of those logs. It runs one loop per network: fetch, verify, mirror, anchor, publish.
{ .iscc-lead }

## Roles

- **Hubs** issue ISCC-IDs and log every declaration in a [tlog-tiles](https://c2sp.org/tlog-tiles) transparency
    log, a Merkle tree served as static files. Each hub signs its log state with an Ed25519 key published in
    its did:web document.
- **The Hub-List** is the authoritative list of a network's hubs, maintained in the
    [iscc-hub repository](https://github.com/iscc/iscc-hub/tree/main/hubs): each hub's domain and its 12-bit
    `hub_id`, the number every ISCC-ID it issues carries.
- **A monitor instance** follows every hub on the Hub-List, checks their logs and publishes what it saw. Anyone
    can run one; each checks the hubs on its own.
- **Clients**, people and programs, look up ISCC-IDs on a monitor and verify the evidence it hands out.

## The loop

### 1. Follow

Every five minutes by default, the monitor fetches each hub's latest checkpoint: the log's size and Merkle root,
signed by the hub. It re-reads the Hub-List every hour, so a changed hub entry takes effect without a restart; a
hub newly added to the list is followed after the next restart.

### 2. Verify

A new checkpoint is accepted only if:

- its signature verifies under the hub's key, resolved from `https://<hub>/.well-known/did.json`;
- an RFC 6962 consistency proof shows that it extends the checkpoint accepted before, so no record was changed or
    removed;
- the monitor can rebuild its Merkle root from the log tiles it fetched.

### 3. Mirror

The monitor stores every hub's tiles and entry bundles in one SQLite file per network and serves them at the same
paths the hub uses. Its copy keeps a log available and verifiable if the hub goes offline. Full tiles never change,
so each is fetched once; the monitor's traffic grows with the log's growth, not with its size.

### 4. Anchor

Every hour, and at startup, the monitor submits newly accepted roots to an
[OpenTimestamps](https://opentimestamps.org) calendar and collects the Bitcoin confirmations of earlier ones. A
confirmed timestamp shows that a log state existed by a given Bitcoin block.

### 5. Publish

For any ISCC-ID in the network, the monitor renders a Certificate of Inclusion and offers a proof bundle. It also
serves the mirrored log, inclusion and consistency proofs, and a summary verdict over an
[HTTP API](../reference/http-api.md).

## Hub status

Each hub has one of five statuses, shown on the instance's pages and in its metrics:

| Status           | Meaning                                                                                              |
| ---------------- | ---------------------------------------------------------------------------------------------------- |
| **verified**     | The latest checkpoint is signed with the hub's did:web key, consistent with earlier ones and rebuilt |
| **unresolvable** | The monitor cannot fetch or parse the hub's did:web document                                         |
| **unverified**   | The checkpoint's signature does not match the hub's own did:web key                                  |
| **frozen**       | The hub's log contradicted itself: the monitor stopped accepting its checkpoints and kept the evidence |
| **inactive**     | The hub is removed or paused in the Hub-List                                                         |

Every hub is followed and mirrored regardless of its status, but only checkpoints from a *verified* hub advance
what the monitor accepts.

A hub is **frozen** when its log shrinks, shows a different root at a size already seen, or fails a consistency
proof. Those are the forms a rewritten history takes. The monitor keeps both contradicting checkpoints, signed by
the hub, as permanent evidence, logs a warning and keeps polling the hub at a slower pace. A freeze is not lifted
automatically.

## Records and ISCC-IDs

A log record is a **declaration**, which commits an ISCC-ID, or a **deletion**, a later record that marks a
declaration as deleted. A deletion never removes the declaration from the log, so the monitor always keeps it. A
hub may also *redact* a declaration from its own responses; that writes no record and the monitor never sees it.

The monitor's verification does not interpret record contents. It reads them only to index ISCC-IDs and to label
records on its pages.
