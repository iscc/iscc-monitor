---
title: Deploy an instance
description: Run an ISCC Monitor instance from the container image behind a reverse proxy
icon: lucide/rocket
---

# Deploy an instance

The monitor is one static binary in a small container image, configured with environment variables. One instance
follows one network, mainnet or testnet, and keeps all its state in one SQLite file.
{ .iscc-lead }

## Requirements

- A container runtime, such as Docker with Compose.
- A database volume that can grow with the logs of the busiest hub in the network.
- A reverse proxy that terminates TLS, such as Caddy.
- A subdomain for the instance alone. The monitor serves its pages, assets and hub routes from the root of its
    host, so it cannot run under a sub-path.
- Outbound HTTPS. See [Egress](#egress).

## Quick start

1. Create `compose.yaml`:

    ```yaml title="compose.yaml"
    services:
      iscc-monitor:
        image: ghcr.io/iscc/iscc-monitor:v0.1.0
        environment:
          ISCC_MONITOR_DB: /data/monitor.db
          ISCC_MONITOR_REALM: https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/mainnet.yaml
          ISCC_MONITOR_INSTANCE: monitor.example.com
        volumes:
          - monitor-data:/data
        stop_grace_period: 60s
        # No ports: the reverse proxy reaches the monitor on port 9464 over the internal network.

    volumes:
      monitor-data:
        name: monitor-data
    ```

    For testnet, use `.../hubs/testnet.yaml`. [Configuration](configuration.md) lists every setting.

1. Before the first start, give the volume to uid 65532. Then start the instance:

    ```bash
    docker run --rm -v monitor-data:/data alpine chown -R 65532:65532 /data
    docker compose up -d
    ```

    The image runs as the unprivileged uid 65532, but a new Docker volume belongs to root. Without this step, the
    monitor cannot create its database. `name: monitor-data` makes Compose mount this volume, not one prefixed
    with the project name. A bind-mounted host directory owned by uid 65532 also works.

1. Check that the instance is up. The first poll finishes within seconds.

    ```bash
    curl https://monitor.example.com/healthz
    curl https://monitor.example.com/version
    ```

1. Open the start page. It lists every hub of the network with its status.

## Image and tags

The image is `ghcr.io/iscc/iscc-monitor`.

| Tag            | Use                                                                                               |
| -------------- | ------------------------------------------------------------------------------------------------- |
| `:vX.Y.Z`      | A release, described in the [changelog](../reference/changelog.md). Pin production instances to it |
| `:latest`      | The newest release. It moves, so do not pin to it                                                 |
| `:develop`     | The latest build of the `develop` branch, for continuous testnet deployments                      |
| `:sha-<short>` | One exact build of a release or of `develop`. It never moves. Use it or `:vX.Y.Z` to roll back     |

A released version is never re-tagged. `GET /version` shows the running build. A release image shows the release
and its commit, such as `{"version":"v0.1.0+ac65974"}`. A `develop` image shows the full commit.

The image is based on distroless `static-debian12:nonroot`. It has the CA certificates for outbound HTTPS and
loads nothing from a CDN at runtime. Its license terms are in `/usr/share/doc/iscc-monitor/`.

## Reverse proxy

The monitor serves plain HTTP on `:9464` and sets no connection timeouts, so it must not face the internet
directly. Do not publish the port on the host, because Docker's published ports bypass the host firewall. Let the
reverse proxy reach the container over the internal network and terminate TLS.

All routes share one listener, so the proxy must keep them apart. Everything is public by design except
`/metrics`, which is your decision; see [Monitoring](monitoring.md#metrics). No route carries a secret, because
the monitor holds no signing key.

With Caddy:

```text title="Caddyfile"
monitor.example.com {
    respond /metrics 403
    reverse_proxy iscc-monitor:9464
}
```

## Egress

The monitor sends outbound HTTPS requests to:

- Every hub in the Hub-List: its log at `https://<hub>/log/...` and its did:web document at
    `https://<hub>/.well-known/did.json`.
- The Hub-List, once an hour, when `ISCC_MONITOR_REALM` is a URL.
- The OpenTimestamps calendar `https://alice.btc.calendar.opentimestamps.org`, at startup and then every
    `ISCC_MONITOR_OTS_INTERVAL` (default one hour).

Hubs join and leave the network, so there is no fixed address list. An egress allow-list must permit HTTPS to the
hub domains and the calendar.

## Resources

Estimates, not benchmarks:

- Memory: tens of megabytes. The peak is the tiles in flight during a poll.
- CPU: close to idle between polls, with short bursts for signature checks and Merkle root rebuilds.
- Disk: the value to watch. The database holds the full log of every hub and grows with their activity. Set a
    disk-usage alert on the volume and size it with headroom.

## Shutdown

On `SIGTERM`, which `docker stop` sends, the monitor finishes the current poll, closes the database and exits
cleanly. Set `stop_grace_period` long enough for this, as in the quick start. Otherwise the runtime kills the
monitor mid-poll.

## Next steps

- [Put your name on the instance](branding.md)
- [Back up the database and upgrade safely](data.md)
- [Watch it with Prometheus](monitoring.md)
