---
title: Check a Bitcoin anchor
description: Verify the OpenTimestamps proof that ties a hub's log state to a Bitcoin block
icon: lucide/bitcoin
---

# Check a Bitcoin anchor

The monitor timestamps the Merkle root of each log state it accepts with [OpenTimestamps](https://opentimestamps.org).
Once a timestamp is confirmed in a Bitcoin block, anyone can check that the log state, and every record in it,
existed by the time of that block.
{ .iscc-lead }

## What you need

- The **OpenTimestamps proof** (`.ots`) for the log state that covers the record.
- The **root** it timestamps, in hex.
- The `ots` command from [opentimestamps-client](https://github.com/opentimestamps/opentimestamps-client):
    `pip install opentimestamps-client`, or run it without installing as
    `uvx --from opentimestamps-client ots ...`. On Windows the client fails to load its OpenSSL library; run it in
    WSL or a Linux container.
- For the full check, a **Bitcoin node** that `ots` can reach: a local Bitcoin Core, or one given with
    `--bitcoin-node <URL>`. Without one, see [Without a Bitcoin node](#without-a-bitcoin-node).

## From a certificate

§5 of the [certificate](certificate.md) shows the block, the log size the anchor covers, a link to the
`checkpoint.ots` file and the exact command. Download the file and run the command shown, for example:

```bash
ots verify -d 951420d087ae7ff1ac814091012ac68a89736f0f137f74c62dce7503ddaebb80 checkpoint.ots
```

`-d` passes the root as the digest to check. If the proof is for a different root, `ots` stops with
"Digest provided does not match digest in timestamp". Otherwise it asks the Bitcoin node for the attesting block
and reports that the root existed by that block's time.

## Without a Bitcoin node

Without a node, `ots verify` stops with "Could not connect to Bitcoin node", after it has checked the digest.
You can still finish the check by hand. `ots info` prints the block the proof commits to and the merkle root that
block must have:

```bash
ots info checkpoint.ots
```

```text
...
verify BitcoinBlockHeaderAttestation(956107)
# Bitcoin block merkle root a697ce8abe505838208a26b6454eb7af9f5f3c06af05ffac1083335b4c238ca4
```

Compare that merkle root with the header of the same block in a block explorer, for example
`https://mempool.space/api/block-height/956107` for the block hash and then `https://mempool.space/api/block/<hash>`
for its `merkle_root`. If they match, the root existed by the block's time. This check relies on the explorer
reporting the real block header; a Bitcoin node of your own removes that reliance.

## From a proof bundle

A bundle with an `anchor` member carries the same proof in base64, together with the hub-signed checkpoint at the
anchored size and the record's inclusion proof against it. The [verification script](verify-bundle.md) checks
that evidence, writes the proof to `<id>.anchor.ots` and prints the `ots verify` command with the root.

## From the API

`GET /<hub>/log/checkpoint.ots` returns the proof for the hub's latest accepted root, and
`GET /<hub>/log/checkpoint.ots?size=N` the proof for the root the monitor accepted at size `N`. The root to pass to
`ots verify` is the checkpoint's root decoded from base64 to hex.

A proof that is not yet confirmed is *pending*: it is committed to an OpenTimestamps calendar but not yet to
Bitcoin. The monitor asks the calendar for an upgrade every hour and serves the confirmed proof once it has it.

## Why the anchor matters

The hub's signature shows what the hub claims. The Bitcoin anchor adds a time that does not depend on the hub or
the monitor: a record in a log state that was confirmed in block *N* cannot have been added later, even by a hub
that rewrote its log afterwards.
