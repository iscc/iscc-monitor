---
title: Compare your view of a log
description: Detect a split view by comparing the hub log state you saw with the one the monitor observed
icon: lucide/git-compare
---

# Compare your view of a log

A hub could show different, internally consistent histories to different audiences: a *split view*. No single
party can see that from its own view alone. Comparing the log state a hub showed you with the one it showed an
independent monitor reveals it.
{ .iscc-lead }

## What you compare

A hub's log state is a **checkpoint**: the log's size and Merkle root, signed by the hub. You get one from the hub
directly:

```bash
curl https://iscc.id/log/checkpoint
```

and the monitor's from its mirror of the same log:

```bash
curl https://monitor.iscc.io/iscc.id/log/checkpoint
```

Both are signed by the hub, so two checkpoints that cannot belong to one log are proof that the hub equivocated.

## Same size: compare the roots

If both checkpoints have the same size, their roots must be equal. Different roots at the same size, both with a
valid hub signature, are a split view.

## Different sizes: check consistency

A log only grows, so a smaller checkpoint must be a prefix of a larger one. An RFC 6962 **consistency proof**
shows that, using only the two roots and a few hashes.

The monitor computes consistency proofs from your earlier size to the size it accepted:

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

Verify the proof with the root of *your* checkpoint at `firstSize` and the root of the monitor's checkpoint at
`secondSize`, using an RFC 6962 library such as Go's
[`transparency-dev/merkle`](https://github.com/transparency-dev/merkle) (`proof.VerifyConsistency`). A proof that
fails to verify means that your checkpoint and the monitor's cannot both be states of one log.

`from` can be any size up to the monitor's accepted size, whether or not the monitor ever saw a checkpoint at that
size. When your checkpoint is larger than the monitor's, the endpoint returns `400`: the monitor has not seen that
state yet. Wait for its next poll, by default within five minutes.

Any tlog-tiles client can also compute the consistency proof from the hub's own log tiles, without the monitor.

## What the monitor checks on its own

The monitor compares each checkpoint it fetches with the ones it saw before. If a hub's log shrinks, changes its
root at an unchanged size, or fails a consistency proof, the monitor freezes the hub and keeps both checkpoints as
evidence; see [How it works](../concepts/how-it-works.md#hub-status). What it cannot see on its own is a hub
showing a different history to someone else. That is why the comparison above matters, and why more than one
independent monitor helps.

## Coverage

The monitor vouches only for what it observed: from the start of its coverage, shown on each hub's page and on the
certificate's Comparison Anchor, onward. A split view that ended before the monitor started watching is out of its
reach.
