---
title: Data, backups and upgrades
description: Where an ISCC Monitor instance keeps its state, how to back it up and how to upgrade safely
icon: lucide/database
---

# Data, backups and upgrades

One SQLite file holds all the state of an instance for its network. Part of it is evidence that nothing can
regenerate, so handle the file with care.
{ .iscc-lead }

## What is in the database

- **The mirror** of every hub's log: tiles and entry bundles. It is the bulk of the file. The monitor could fetch
    it again from the hubs.
- **Irreplaceable evidence**: the checkpoints the monitor observed, the contradicting checkpoints of any frozen
    hub, and the OpenTimestamps proofs. If a hub goes offline or rewrites its log, nothing can recreate them.

Keep the file on a volume that outlives the container. A new container on the same file continues where the old
one stopped.

## Backups

The database runs in SQLite's WAL mode, so recent changes can sit in two files next to it. Back up all three files
as one set: `monitor.db`, `monitor.db-wal` and `monitor.db-shm`. A copy of `monitor.db` alone, taken while the
instance runs, can miss the latest changes.

The simplest consistent backup is a copy taken while the instance is stopped. For a running instance, use
SQLite's own backup, for example:

```bash
sqlite3 /data/monitor.db ".backup '/backup/monitor-$(date -u +%Y%m%dT%H%M%SZ).db'"
```

The image has no shell and no `sqlite3`, so run the command from another container that mounts the same volume.

## Upgrades

1. Snapshot the database (all three files). Keep the snapshot until the new version has proven itself.
1. Change the image tag to the new `:vX.Y.Z` and restart.
1. Check `GET /version` and the start page.

On startup, the new version migrates the database automatically, one transaction per step. If a step fails, the
database stays as it was and the monitor does not start.

!!! warning
    Never recreate the volume to get a clean schema. That destroys the evidence.

## Rollbacks

Between versions with the same database schema, a rollback is a tag change. A version refuses a database that a
newer version migrated, so a rollback across a migration means restoring the snapshot from before the upgrade.
You lose everything the monitor observed since then.

## Rules

- One running instance per database file. The monitor is the only writer of the file.
- uid 65532, the user of the image, must be able to write to the volume. See [Deploy](deploy.md#quick-start).
