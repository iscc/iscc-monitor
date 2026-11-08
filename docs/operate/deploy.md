---
title: Deploy an instance
description: Run an ISCC Monitor instance from the container image behind a reverse proxy
icon: lucide/rocket
---

# Deploy an instance

The monitor is one static binary in a small container image, configured through environment variables. One
instance follows one network, mainnet or testnet, and keeps its whole state in one SQLite file.
{ .iscc-lead }

## Requirements

- A container runtime, such as Docker with Compose.
- A volume for the database; plan for it to grow with the logs of the network's busiest hub.
- A reverse proxy that terminates TLS, such as Caddy, and a **subdomain of its own** for the instance. The monitor
    serves its pages, assets and hub routes from the root of its host, so it cannot run under a sub-path.
- Outbound HTTPS, see [Egress](#egress).

## Quick start

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

For testnet, point `ISCC_MONITOR_REALM` at `.../hubs/testnet.yaml`. [Configuration](configuration.md) lists every
setting.

**Prepare the volume before the first start.** The image runs as the unprivileged user with uid 65532, and a new
Docker volume belongs to root, so the monitor could not create its database. Hand the volume to uid 65532 once:

```bash
docker run --rm -v monitor-data:/data alpine chown -R 65532:65532 /data
docker compose up -d
```

The explicit `name: monitor-data` makes Compose mount the same volume this command prepared, instead of one
prefixed with the project name. A bind-mounted host directory works too, once it is owned by uid 65532.

The first poll finishes within seconds. Check that the instance is up:

```bash
curl https://monitor.example.com/healthz
curl https://monitor.example.com/version
```

Then open the instance's start page: it lists every hub of the network with its status.

## Image and tags

The image is `ghcr.io/iscc/iscc-monitor`:

| Tag             | Use                                                                                  |
| --------------- | ------------------------------------------------------------------------------------ |
| `:vX.Y.Z`       | A release, described in the [changelog](../reference/changelog.md). Pin production instances to it |
| `:latest`       | The newest release; it moves, so do not pin to it                                    |
| `:develop`      | The latest build of the `develop` branch, for continuous testnet deployments         |
| `:sha-<short>`  | One exact build of a release or of `develop`; never moves. Use it or `:vX.Y.Z` to roll back |

A released version is never re-tagged. `GET /version` reports the build that is actually running: the release and
its commit on a release image, such as `{"version":"v0.1.0+ac65974"}`, or the full commit on a `develop` image.

The image is built on distroless `static-debian12:nonroot`, carries the CA certificates for outbound HTTPS and
loads nothing from a CDN at runtime. Its license terms are in `/usr/share/doc/iscc-monitor/`.

## Reverse proxy

The monitor listens on `:9464` and serves plain HTTP. Do not publish that port on the host: Docker's published
ports bypass the host firewall. Let the reverse proxy reach the container over the internal network and terminate
TLS. The HTTP server sets no connection timeouts of its own, so it must not face the internet directly.

All routes share one listener, so separating them is the proxy's job. Everything is public by design except
`/metrics`, which you decide on; see [Monitoring](monitoring.md#metrics). No route carries a secret: the monitor
holds no signing key.

With Caddy, the site block is short:

```text title="Caddyfile"
monitor.example.com {
    respond /metrics 403
    reverse_proxy iscc-monitor:9464
}
```

## Egress

The monitor makes outbound HTTPS requests to:

- every hub in the Hub-List: its log at `https://<hub>/log/...` and its did:web document at
    `https://<hub>/.well-known/did.json`;
- the Hub-List itself, when `ISCC_MONITOR_REALM` is a URL, once an hour;
- the OpenTimestamps calendar `https://alice.btc.calendar.opentimestamps.org`, at startup and then every
    `ISCC_MONITOR_OTS_INTERVAL` (default one hour).

Hubs join and leave the network, so there is no fixed list of addresses. An egress allow-list must permit HTTPS to
the hub domains and the calendar.

## Resources

- **Memory:** tens of megabytes; the peak is the tiles in flight during a poll.
- **CPU:** close to idle between polls, with short bursts for signature checks and Merkle root rebuilds.
- **Disk:** the variable to watch. The database holds every hub's full log and grows with the hubs' activity. Set
    a disk-usage alert on the volume and size it with headroom.

These are estimates, not benchmarks.

## Shutdown

On `SIGTERM`, which `docker stop` sends, the monitor finishes the poll in progress, closes the database and exits
cleanly. Set `stop_grace_period` long enough for that, as in the quick start, so the runtime does not kill it
mid-poll.

## Next steps

- [Put your name on the instance](branding.md)
- [Back up the database and upgrade safely](data.md)
- [Watch it with Prometheus](monitoring.md)
