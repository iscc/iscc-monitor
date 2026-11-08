---
title: Data, backups and upgrades
description: Where an ISCC Monitor instance keeps its state, how to back it up and how to upgrade safely
icon: lucide/database
---

# Data, backups and upgrades

One SQLite file holds an instance's entire state for its network. Part of it is evidence that can never be
regenerated, so treat the file with care.
{ .iscc-lead }

## What is in the database

- **The mirror** of every hub's log: tiles and entry bundles. It is the bulk of the file, and it could be fetched
    again from the hubs.
- **Irreplaceable evidence:** the checkpoints the monitor observed, the contradicting checkpoints of any frozen
    hub, and the OpenTimestamps proofs. If a hub goes offline or rewrites its log, nothing can recreate these.

Keep the file on a volume that outlives the container. A new container pointed at the same file continues where
the old one stopped.

## Backups

The database runs in SQLite's WAL mode, so recent changes can sit in two files next to it. **Back up three files
together as one set:** `monitor.db`, `monitor.db-wal` and `monitor.db-shm`. Copying `monitor.db` alone while the
instance runs can miss the latest changes.

The simplest consistent backup is a copy taken while the instance is stopped. For a backup of a running instance,
use SQLite's own backup, for example:

```bash
sqlite3 /data/monitor.db ".backup '/backup/monitor-$(date -u +%Y%m%dT%H%M%SZ).db'"
```

The image itself contains no shell or `sqlite3`; run the command from another container that mounts the same
volume.

## Upgrades

1. **Snapshot the database** (all three files) and keep the snapshot until the new version has proven itself.
1. Change the image tag to the new `:vX.Y.Z` and restart.
1. Check `GET /version` and the start page.

A new version migrates the database on startup, automatically and in one transaction per step: a failed step
leaves the database as it was and the monitor does not start. **Never recreate the volume to get a clean schema**;
that destroys the evidence.

## Rollbacks

Between versions with the same database schema, rolling back is a tag change. A version refuses a database that a
newer version has migrated, though, so rolling back across a migration means restoring the snapshot from before
the upgrade. Whatever the monitor observed since then is lost.

## Rules

- One running instance per database file. The monitor is the file's only writer.
- The volume must be writable by uid 65532, the image's user; see [Deploy](deploy.md#quick-start).
