# Operating an iscc-monitor instance

This is the deployment and operability contract for a **server instance** of the
monitor — a running deployment of the data API plus its server-rendered dashboard
at its own domain. It is distinct from the standalone in-browser **verifier app**
(`monitor.iscc.codes`, a separate GitHub-Pages static artifact): a server image
embeds and serves its own `/_ds/` assets (including `verify.wasm`), so it needs
neither the Pages site nor any CDN at runtime.

The first consumer is **iscc-infra**, which runs ISCC services as Docker Compose
stacks behind a label-driven `caddy-docker-proxy`. The first deployment is a
testnet test instance at **`monitor-test.iscc.io`** on an existing DigitalOcean box.

The monitor is configured entirely through environment variables. CLAUDE.md's
[Running a local dev instance](../CLAUDE.md#running-a-local-dev-instance) section
is the **authoritative env-var table** (`ISCC_MONITOR_DB`, `ISCC_MONITOR_REALM`,
the `NORMAL`/`FROZEN`/`ADDR` cadence + bind, and the optional masthead identity
keys). This doc references those keys but does not duplicate the table, so the two
cannot drift.

## Image & tags

The image is `ghcr.io/iscc/iscc-monitor`, published on every push to `develop` by
`.github/workflows/publish.yml`, with two tags:

- **`:develop`** — floating, always the latest `develop` build. Track this for
  continuous testnet deploys.
- **`:sha-<short>`** — immutable, keyed on the build's git SHA. **Pin and roll back
  with this tag** — a `:develop` re-pull can move under you, a `:sha-<short>` cannot.

The image is self-contained: it embeds `/_ds/` (including `verify.wasm`) and carries
the CA root bundle for outbound HTTPS, so there is no CDN dependency at runtime. It
is built on the distroless `static-debian12:nonroot` base and runs as the
**unprivileged uid 65532**.

Confirm exactly which build is live with **`GET /version`** — it returns
`{"version":"<git-sha>"}` stamped at build time. Do not guess the running build from
the floating `:develop` tag; ask the binary.

## State, volume & backup

`ISCC_MONITOR_DB` points at a single SQLite file that holds the **entire durable
state** of one network (ADR-0007), including the glossary's *irreplaceable evidence*
— observed checkpoints, split-view pairs, and OTS proofs that can never be
regenerated.

- **Put the file on a mounted volume.** For example, mount a named Docker volume at
  `/data` and set `ISCC_MONITOR_DB=/data/monitor.db`. State must outlive the
  container (restart-survival = point a fresh container at the same file).
- **The backup unit is that one file PLUS its `-wal` and `-shm` siblings.** The
  store runs in WAL mode (`PRAGMA journal_mode=WAL`), so committed-but-not-yet-
  checkpointed data can live in `monitor.db-wal`. Back up `monitor.db`,
  `monitor.db-wal`, and `monitor.db-shm` together as one consistent set — that
  captures all durable state and all irreplaceable evidence. Backing up the `.db`
  alone while the instance is running can miss the tail of the WAL.
- **The volume directory must be writable by uid 65532** (the non-root image user).
  A volume owned by `root:root` with no group/other write bit will make `store.Open`
  fail to create or write the database. Set the volume's ownership/permissions so
  uid 65532 can read and write it.

There is a single writer per database file (ADR-0005/0007): one network file, one
writing goroutine, WAL plus a `busy_timeout`. Do not point two running instances at
the same `.db` file.

The image bakes the canonical testnet realm document at
**`/etc/iscc-monitor/realm.txt`** (from `deploy/realm-testnet.txt`) and **sets
`ISCC_MONITOR_REALM` to that path via `ENV`**, so a fresh container has a valid
`ISCC_MONITOR_REALM` out of the box — you need not pass it. A deploy may instead mount
its own realm document and override `ISCC_MONITOR_REALM` by passing its own var.

## Migration policy (interim)

`store.Open` applies the schema as `CREATE TABLE IF NOT EXISTS` only — there is **no
on-disk migration mechanism yet** (it is a tracked open item; see the "No on-disk DB
migration story" issue). A column added to an existing table in a later image
**silently never reaches a pre-existing database**: the first `:develop` bump that
adds a column over a populated volume would fail the new code path with `no such
column`.

**Interim policy: recreate the volume on a schema change.** This is acceptable for a
throwaway testnet instance, where the data is reconstructable by re-fetch + fsck.
Operate the testnet instance with this as the explicit assumption until the migration
mechanism lands; revisit before any deployment that must preserve a populated
production database across an image bump.

## Reverse-proxy & port contract

The container **binds `:9464`** on the internal Docker network and **publishes no
host port**. Only the reverse proxy — caddy-docker-proxy in iscc-infra — terminates
TLS and publishes 80/443. This is an iscc-infra invariant: Docker's published ports
bypass the host firewall, so the monitor must never publish `:9464` to the host;
reachability is mediated entirely by Caddy on the internal network.

The monitor needs its **own vhost / subdomain**, because it serves root-absolute
`/_ds/` asset paths and root-mounted hub routes (`/<domain>/log/…`). It cannot be
hung off a sub-path of a shared host. `monitor-test.iscc.io` satisfies this.

## Route exposure & the `/metrics` decision

All routes share the **one `:9464` mux** (`serveMetrics` builds a single
`http.Server` over a single mux), so routes can be separated only **at the proxy,
never by port**.

**Public by design** (the monitor is a *verifiable cache*, not a trusted oracle —
its data is meant to be fetched and re-verified, and it serves CORS `*`):

- `GET /` — realm-index dashboard
- `GET /<domain>` — per-hub dossier
- `GET /<domain>/log/…` — the mirror, computed proofs, entries, verify-for-me
- `GET /inclusion/<iscc_id>` — certificate of inclusion (+ `.bundle`)
- `GET /_ds/…` — embedded DS assets (`tokens.css`, fonts, `wasm_exec.js`, `verify.wasm`)
- `GET /healthz` — liveness + store readiness
- `GET /version` — build provenance

**No route carries a secret.** The monitor holds **no signing key** in v1 (cosigning
is the M7 milestone, deferred), so there is no credential anywhere in the surface to
leak, and no route needs authentication.

**`/metrics`** is the one route to decide on. `GET /metrics` exposes operational
internals — poll-failure counts, self-consistency violation counts, per-hub status —
in Prometheus format. There is no secret in it, but it is operator telemetry, not
public-cache evidence. Per ADR-0013 Decision 6, **whether `/metrics` is public or
proxy-denied is the operator's per-instance choice**, recorded with the instance, not
a code gate (there is no second listener and no `/metrics` auth flag — exposure is a
proxy concern).

- **Recommendation for the public testnet vhost:** **deny `/metrics` at Caddy** and
  scrape it only on the internal Docker network. iscc-infra enforces this in the Caddy
  labels (a path-deny rule on the public vhost). Everything else in the list above is
  served publicly.

## Egress

The monitor makes outbound **HTTPS** connections to:

- **Each realm hub's `/log` tiles** — the tlog-tiles transparency log it follows and
  mirrors, at every hub domain in the realm document.
- **Each realm hub's `/.well-known/did.json`** — did:web key resolution (ADR-0009);
  the hub's signing key comes from its did:web document, not the realm registry.
- **The OpenTimestamps calendar `https://alice.btc.calendar.opentimestamps.org`**
  (ADR-0004) — for Bitcoin-anchor timestamping of checkpoint roots.

There is **no fixed IP allow-list**: hub egress is per-realm DNS and changes as the
realm membership changes. DigitalOcean droplets default-allow outbound egress, so this
usually needs no firewall change — the point is that it is a **conscious requirement**.
If an egress allow-list is in force, it must permit HTTPS to arbitrary realm hub
domains plus the OTS calendar host above.

### Footprint (estimates)

The following are **rough estimates** for the **2-hub testnet realm**
(`sb0.iscc.id` + `sb1.amlet.id`), to be refined against live data — they are not
measured benchmarks:

- **Resident memory:** order of tens of MB for the process itself; the dominant
  variable is in-flight tile/bundle buffers during a poll. Budget a small instance and
  watch it; this fits comfortably alongside other small services on a shared box.
- **CPU:** near-idle between polls; brief bursts during each poll cycle (signature
  verification + Merkle root rebuild). The poll cadence is `ISCC_MONITOR_NORMAL`
  (default 5m), so steady-state CPU is low.
- **Disk growth (the variable to watch):** the durable cost is the **mirror BLOBs**
  (a hub's hash tiles + entry bundles) plus the index, stored in the SQLite file.
  Growth is proportional to each hub's log activity — a quiet testnet hub adds little;
  a high-traffic hub at millions of records/day would dominate. **Set a DigitalOcean
  disk-usage alert** on the volume so growth is visible before it bites, and size the
  volume with headroom for the realm's busiest hub.

## Graceful shutdown

Containers and orchestrators stop a process with **SIGTERM** (not SIGINT alone).
On SIGTERM the monitor cancels its run context, drains the in-flight poll, runs
`store.Close()`, and exits 0 (ADR-0013 Decision 7; this is the `shutdown_test.go`-
proven trap). A `docker stop` therefore finishes cleanly instead of being cut off
mid-poll.

Set a Compose **`stop_grace_period`** long enough to cover one clean poll plus the
store flush, so the orchestrator does not escalate to SIGKILL before the drain
completes.

## Quick start

A Compose fragment mirroring the CLAUDE.md dev snippet, but for the GHCR image, with a
mounted volume and the required env. See CLAUDE.md's
[Running a local dev instance](../CLAUDE.md#running-a-local-dev-instance) for the full
env-var table.

```yaml
services:
  iscc-monitor:
    image: ghcr.io/iscc/iscc-monitor:develop   # pin :sha-<short> for a fixed build
    environment:
      ISCC_MONITOR_DB: /data/monitor.db        # on the mounted volume below
      # ISCC_MONITOR_REALM is baked into the image (-> /etc/iscc-monitor/realm.txt);
      # set it here only to use a mounted realm document instead.
      ISCC_MONITOR_NORMAL: 5m                   # clean-hub poll interval
      ISCC_MONITOR_ADDR: ":9464"               # internal bind; do NOT publish to host
    volumes:
      - monitor-data:/data                      # the volume dir must be writable by uid 65532 (see note below)
    stop_grace_period: 60s                      # cover a clean poll + store flush on SIGTERM
    # No `ports:` — caddy-docker-proxy terminates TLS and publishes 80/443.
    # Add the caddy-docker-proxy labels for the monitor-test.iscc.io vhost here,
    # including a path-deny rule for /metrics (scrape it on the internal network).

volumes:
  monitor-data:
    name: monitor-data                          # pin the literal engine-volume name (no project prefix), so the chown prep below targets the SAME volume Compose mounts
```

**Volume ownership (do this first).** A fresh named Docker volume is created
`root:root`, but the image runs as uid 65532, so a bare `-v monitor-data:/data`
makes `store.Open` fail permission-denied on the first write. The explicit
`name: monitor-data` in the Compose `volumes:` block above is what makes this
literal-name prep line up with the volume `docker compose up` mounts — without it
Compose would mount a project-prefixed `<project>_monitor-data` the `chown` never
touched. Either pre-`chown` the volume once before the first run, e.g.

```sh
docker run --rm -v monitor-data:/data alpine chown -R 65532:65532 /data
```

or bind-mount a host directory you have already made 65532-writable
(`mkdir -p ./monitor-data && sudo chown 65532:65532 ./monitor-data`, then
`-v ./monitor-data:/data`). This is the same uid-65532 requirement stated in
[State, volume & backup](#state-volume--backup) above.

The equivalent bare `docker run` (no host port published — `ISCC_MONITOR_REALM` comes
from the image's baked `ENV`, so it is not passed) is:

```sh
docker run --rm \
  -e ISCC_MONITOR_DB=/data/monitor.db \
  -e ISCC_MONITOR_NORMAL=5m \
  -e ISCC_MONITOR_ADDR=:9464 \
  -v monitor-data:/data \
  ghcr.io/iscc/iscc-monitor:develop
```

After the first poll (a few seconds) the HTTP surface is live. Confirm with
`GET /healthz` (200) and `GET /version` (the live build's git SHA).
