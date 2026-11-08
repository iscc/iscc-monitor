---
title: Compare your view of a log
description: Detect a split view by comparing the hub log state you saw with the one the monitor observed
icon: lucide/git-compare
---

# Compare your view of a log

A hub can show different, internally consistent histories to different audiences: a *split view*. No one can see
it from their own view alone. You find it by comparing the log state a hub showed you with the one it showed an
independent monitor.
{ .iscc-lead }

## What you compare

A hub's log state is a **checkpoint**: the log's size and Merkle root, signed by the hub. Get one from the hub:

```bash
curl https://iscc.id/log/checkpoint
```

Get the monitor's from its mirror of the same log:

```bash
curl https://monitor.iscc.io/iscc.id/log/checkpoint
```

The hub signs both. Two checkpoints that cannot belong to one log prove a split view.

- Same size: the roots must be equal. Different roots, both with a valid hub signature, are a split view.
- Different sizes: check consistency, as below.

## Check consistency

A log only grows, so a smaller checkpoint must be a prefix of a larger one. An RFC 6962 **consistency proof**
shows this with the two roots and a few hashes.

The monitor computes a consistency proof from your earlier size to the size it accepted:

```bash
curl "https://monitor.iscc.io/iscc.id/log/consistency?from=1"
```

```json
{
  "type": "IsccLogConsistencyProof",
  "firstSize": 1,
  "secondSize": 220,
  "consistencyProof": ["6EXyB1geohg7S5JLntu1k2itax1HegKkRjZQ1gxUsAg=", "..."]
}
```

Verify it with the root of *your* checkpoint at `firstSize` and the root of the monitor's at `secondSize`. Use an
RFC 6962 library such as Go's [`transparency-dev/merkle`](https://github.com/transparency-dev/merkle)
(`proof.VerifyConsistency`). If the proof fails, the two checkpoints cannot both be states of one log.

`from` can be any size up to the monitor's accepted size, even one the monitor never saw a checkpoint for. If your
checkpoint is larger than the monitor's, the endpoint returns `400`: the monitor has not seen that state yet. Wait
for its next poll, by default within five minutes.

Any tlog-tiles client can also compute the consistency proof from the hub's own log tiles, without the monitor.

## What the monitor checks on its own

The monitor compares each checkpoint it fetches with the earlier ones. If a hub's log shrinks, changes its root at
the same size, or fails a consistency proof, the monitor freezes the hub and keeps both checkpoints as evidence.
See [How it works](../concepts/how-it-works.md#hub-status).

The monitor cannot see on its own that a hub shows a different history to someone else. That is why your
comparison matters, and why more than one independent monitor helps.

## Coverage

The monitor vouches only for what it observed from its coverage start onward. Each hub's page and the
certificate's Comparison Anchor show the coverage start. A split view that ended before then is out of the
monitor's reach.
