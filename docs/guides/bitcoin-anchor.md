---
title: Check a Bitcoin anchor
description: Verify the OpenTimestamps proof that ties a hub's log state to a Bitcoin block
icon: lucide/bitcoin
---

# Check a Bitcoin anchor

The monitor timestamps the Merkle root of each log state it accepts with [OpenTimestamps](https://opentimestamps.org).
Once a Bitcoin block confirms the timestamp, anyone can check that the log state, and every record in it, existed
by the time of that block.
{ .iscc-lead }

## What you need

- The **OpenTimestamps proof** (`.ots`) for the log state that covers the record.
- The **root** it timestamps, in hex.
- The `ots` command from [opentimestamps-client](https://github.com/opentimestamps/opentimestamps-client):
    `pip install opentimestamps-client`, or run it without installing as
    `uvx --from opentimestamps-client ots ...`. On Windows the client fails to load its OpenSSL library. Use WSL
    or a Linux container.
- For the full check, a **Bitcoin node** that `ots` can reach: a local Bitcoin Core, or one you give with
    `--bitcoin-node <URL>`. If you have none, see [Without a Bitcoin node](#without-a-bitcoin-node).

## From a certificate

§5 of the [certificate](certificate.md) shows the block, the log size the anchor covers, a link to the
`checkpoint.ots` file and the exact command. Download the file and run the command, for example:

```bash
ots verify -d 951420d087ae7ff1ac814091012ac68a89736f0f137f74c62dce7503ddaebb80 checkpoint.ots
```

`-d` gives the root as the digest to check. If the proof is for a different root, `ots` stops with
"Digest provided does not match digest in timestamp". If not, it gets the attesting block from the Bitcoin node
and reports that the root existed by that block's time.

## Without a Bitcoin node

Without a node, `ots verify` checks the digest, then stops with "Could not connect to Bitcoin node". Finish the
check by hand:

1. Run `ots info`. It prints the block the proof commits to and the merkle root that block must have:

    ```bash
    ots info checkpoint.ots
    ```

    ```text
    ...
    verify BitcoinBlockHeaderAttestation(956107)
    # Bitcoin block merkle root a697ce8abe505838208a26b6454eb7af9f5f3c06af05ffac1083335b4c238ca4
    ```

1. Get the block hash from a block explorer, for example `https://mempool.space/api/block-height/956107`.
1. Read the `merkle_root` from `https://mempool.space/api/block/<hash>`.
1. Compare the two merkle roots. If they match, the root existed by the block's time.

This check trusts the explorer to report the real block header. A Bitcoin node of your own removes that trust.

## From a proof bundle

A bundle with an `anchor` member has the same proof in base64. It also has the hub-signed checkpoint at the
anchored size and the record's inclusion proof against it. The [verification script](verify-bundle.md) checks
this evidence, writes the proof to `<id>.anchor.ots` and prints the `ots verify` command with the root.

## From the API

- `GET /<hub>/log/checkpoint.ots` returns the proof for the hub's latest accepted root.
- `GET /<hub>/log/checkpoint.ots?size=N` returns the proof for the root the monitor accepted at size `N`.

For `ots verify`, decode the checkpoint's root from base64 to hex.

A *pending* proof is in an OpenTimestamps calendar but not yet in Bitcoin. The monitor asks the calendar for an
upgrade, by default every hour, and less often for a root that stays pending. It serves the confirmed proof once
it has it.

## Why the anchor matters

The hub's signature shows what the hub claims. The Bitcoin anchor adds a time that depends on neither the hub nor
the monitor. A record in a log state that block *N* confirmed cannot have been added later, even if the hub
rewrote its log afterwards.
