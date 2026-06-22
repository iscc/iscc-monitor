---
status: accepted
---

# Server packaging & deployment: a container image published to GHCR

PRD story 13 already scopes "a single static binary **plus a container image**, so
that deployment is trivial and matches the hub's release flow." The static binary
exists (`cmd/iscc-monitor`, `CGO_ENABLED=0`, ADR-0003), but the **container image
and its publish path were never specified**, so no milestone ever built them. This
ADR fixes the packaging + run contract so the CID loop has a concrete, verifiable
target (`target.md` → M-Deploy).

The first consumer is **iscc-infra**, which deploys ISCC services as Docker Compose
stacks behind a label-driven `caddy-docker-proxy` (no PaaS, no static Caddyfile). It
needs a *pullable image*, not an on-box source build. The first deployment is a
**testnet test instance at `monitor-test.iscc.io`** on an existing DigitalOcean box.

This is the **server instance** packaging. It is distinct from, and must not be
conflated with, the standalone in-browser **verifier app** (`monitor.iscc.codes`,
GitHub Pages, ADR-0003) — that is a separate static artifact with its own publish
path. A server image embeds and serves its own `/_ds/` assets (incl. `verify.wasm`,
`internal/web`), so it needs neither the Pages site nor any CDN at runtime.

## Decision

1. **Artifact = an OCI image on GHCR.** Publish `ghcr.io/iscc/iscc-monitor`. A
   multi-stage `Dockerfile` builds the pure-Go `cmd/iscc-monitor` (`CGO_ENABLED=0`;
   SQLite via `modernc.org/sqlite`, no libc) into a minimal final stage (`scratch`
   or distroless), running as a **non-root** user, carrying CA roots (egress is
   HTTPS) and the baked default realm document.

2. **Build from `develop`, pre-release-tags (interim).** Until v1 cuts release tags,
   the publish workflow runs on push to `develop` and pushes two tags: `develop`
   (floating = latest develop) and `sha-<short>` (immutable = pin + rollback). When
   v1 tagging lands, add `vX.Y.Z` + `latest` (the "matches the hub's release flow"
   half of story 13); the `develop`/`sha` channel stays for the loop's continuous
   deploys.

3. **12-factor config, no secrets.** All config is env (`internal/config`):
   `ISCC_MONITOR_DB`, `ISCC_MONITOR_REALM`, `ISCC_MONITOR_NORMAL`/`FROZEN`/`ADDR`,
   and the optional masthead identity `ISCC_MONITOR_INSTANCE`/`OPERATOR`/`REALM_NAME`.
   The monitor holds **no secret** in v1 (no signing key — cosigning is M7 /
   ADR-0004-deferred), so there is no secret-management requirement; this is a
   deliberate property to preserve, not an omission.

4. **State = one SQLite file per network = the backup unit.** `ISCC_MONITOR_DB`
   points at a file on a mounted volume; that single file (+ its `-wal`/`-shm`
   siblings) is the entire durable state and the unit of backup (ADR-0007;
   restart-survival per story 16 = point a fresh container at the same file). The
   volume directory must be writable by the non-root image uid. **In-place schema
   migration is out of v1 scope** (the open "no on-disk DB migration story" item):
   the interim policy is **recreate the volume on a schema change**, which is
   acceptable for the testnet instance and MUST be documented, never silently
   assumed — the first `:develop` bump that adds a column over a populated volume
   would otherwise fail with `no such column`.

5. **Runs behind a TLS-terminating reverse proxy.** The container binds `:9464` on
   the internal Docker network and **publishes no host port** — only the reverse
   proxy (caddy-docker-proxy in iscc-infra) terminates TLS and publishes 80/443
   (iscc-infra invariant; Docker bypasses host firewalls). The monitor needs its own
   vhost/subdomain (it uses root-absolute `/_ds/` asset paths + root-mounted hub
   routes), which `monitor-test.iscc.io` satisfies.

6. **Single listener, conscious route exposure.** `/metrics`, `/healthz`, the
   dashboard, and the per-hub mirror all share the one `:9464` mux (`serveMetrics`),
   so routes can be separated only at the proxy, never by port. The dashboard /
   dossier / mirror / proof / `/healthz` surfaces are public **by design**
   (verifiable cache, ADR-0003; CORS `*`). Whether `/metrics` is public or
   proxy-denied is the **operator's** per-instance decision (recorded with the
   instance), not a code gate.

7. **Graceful shutdown on SIGTERM.** Containers and orchestrators stop via
   **SIGTERM**; the process must cancel its run context, drain the in-flight poll,
   and close the store on SIGTERM (not SIGINT alone). Operators set a Compose
   `stop_grace_period` covering a clean poll + store flush.

8. **Outbound egress is HTTPS to arbitrary realm domains.** The monitor egresses to
   each hub's `/log` tiles + `/.well-known/did.json` (did:web, ADR-0009) and to the
   OTS calendar (`alice.btc.calendar.opentimestamps.org`, ADR-0004). A deployment
   must allow that egress (DO droplets default-allow it; the point is it is a
   conscious requirement, and there is no fixed allow-list of hub IPs — it is
   per-realm DNS).

9. **Build provenance.** Stamp the image/binary with the git SHA (`-ldflags`) and
   surface it (on `/healthz` JSON or a tiny `GET /version`) so an operator can
   confirm exactly which build is live without guessing from a floating tag.

## Consequences

- **The CID loop owns and can verify in-repo / CI** (the M-Deploy bar): the
  `Dockerfile`, the GHCR publish workflow, the SIGTERM fix + test, the canonical
  realm document + its CLAUDE.md env docs, the version stamp + endpoint, and the
  deployment/operability docs (volume/backup, egress, exposure policy, grace
  period). None of these need a feature milestone to land — M-Deploy is
  **independently advanceable** and a good source of code-closable work while the
  feature milestones are design- or human-blocked.
- **Human / infra-only steps are NOT loop-gating** (mirroring how Pages-enable is
  human-blocked in the WASM milestone): making the GHCR package public (or issuing a
  `read:packages` token), DNS for `monitor-test.iscc.io`, the Caddy labels / Compose
  stack, box selection, and the per-instance `/metrics` exposure choice all live in
  **iscc-infra**, not this repo, and do not gate DONE here.
- **No new runtime dependency or language change.** ADR-0003's
  static-binary / `CGO_ENABLED=0` stack is unchanged; this ADR only adds the
  packaging + run contract around it.
