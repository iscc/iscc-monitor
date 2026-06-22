# Next Work Package

## Step: Write the deployment/operability doc (`deploy/OPERATING.md`)

## Advances
The M-Deploy Verify item (target.md, M-Deploy block):

> a tracked **deployment/operability doc** states, for an operator: the SQLite **volume path +
> single-file backup unit + non-root uid**, the **interim "recreate the volume on a schema change"
> migration policy** (until the on-disk migration mechanism lands — its own open item), the **egress
> endpoints** (hub `/log`, did:web `/.well-known/did.json`, the OTS calendar), the **reverse-proxy
> contract** (binds `:9464`, publishes no host port), and the **`/metrics` exposure** decision

It is the last code/doc-closable M-Deploy Verify criterion before the root README, and it simultaneously
closes ALL THREE remaining open `critical` issues (the iscc-infra ops asks), which preempt the README:
- "Persistence contract for the SQLite DB volume + acknowledge the in-place migration hazard"
- "Decide which routes are safe to publish at the public vhost (especially /metrics)"
- "Document egress + resource footprint for box sizing"

DONE requires 0 `critical`; this doc is the single artifact that clears all three. Review's handoff
`**Next:**` recommends exactly this ("Suggest the operability doc next — it closes 3 `critical`s at once").

## Goal
Add one tracked operator-facing document so a real instance (the testnet `monitor-test.iscc.io`) can be
deployed correctly — volume/backup contract, non-root uid, egress allow-list, reverse-proxy/port
contract, the `/metrics` exposure decision, and the interim "recreate the volume on a schema change"
migration policy.

## Scope
- **Create**: `deploy/OPERATING.md` (the deployment/operability doc — doc, not counted against the
  ≤3 non-test/doc budget)
- **Modify**: none (no Go source, no Dockerfile, no workflow, no behavior change)
- **Reference**:
  - `.claude/adr/0013-server-packaging-and-deployment.md` — the authoritative packaging/run contract;
    each Decision maps to a section here (1 image, 4 state/backup, 5 reverse-proxy, 6 route exposure,
    7 SIGTERM, 8 egress, 9 provenance).
  - `Dockerfile` — confirms the baked realm path `/etc/iscc-monitor/realm.txt`, the distroless
    `nonroot` base = **uid 65532**, and `EXPOSE 9464` / no host-publish.
  - `.github/workflows/publish.yml` + the `docker` `/healthz` smoke in `.github/workflows/ci.yml` —
    confirm the published tags (`:develop` floating + `:sha-<short>` immutable) and the boots-clean smoke.
  - `internal/otsclient/client.go:40` — the OTS calendar egress endpoint
    (`https://alice.btc.calendar.opentimestamps.org`, `DefaultCalendarURL`).
  - `internal/store/sqlite.go:39-40` — confirms `journal_mode=WAL` (so the backup unit is the `.db`
    PLUS its `-wal`/`-shm` siblings) + single-writer discipline.
  - The three open `critical` issues in `.claude/context/issues.md` (persistence, exposure, egress) —
    this doc must answer every "Verify fixed" line in all three so `review` can delete them.
  - `CLAUDE.md` "Running a local dev instance" — the authoritative env-var table; LINK it, do not
    duplicate the full table (keeps the two from drifting).
  - `.claude/context/learnings/registry.md` — the canonical-realm-doc / `.dockerignore` invariant
    (the deploy file is baked, must stay out of `.dockerignore`).

## Not In Scope
- The root `README.md` — a separate `normal` issue and its own next step (do it AFTER this doc clears
  the three `critical`s; the README closes no `critical`).
- Implementing an actual DB migration mechanism — this doc only DOCUMENTS the interim "recreate the
  volume on a schema change" policy; the migration mechanism is its own open `normal` issue.
- Any Go source / Dockerfile / workflow change — doc-only. Do NOT add a `/metrics` auth flag or a
  second listener; ADR-0013 Decision 6 makes `/metrics` exposure a proxy-level operator choice, not a
  code gate. Do not "fix" the `publish.yml` `workflow_dispatch` ref-guard here (no workflow file is
  touched this step).
- Duplicating the full env-var table from CLAUDE.md — link it as the source of truth.
- Resource/footprint numbers presented as measured benchmarks — give honest ballpark ranges for the
  2-hub testnet realm and label them estimates to refine against live data, not invented precision.

## Implementation Notes
Structure the doc around what an operator must do/know, sourcing every claim from ADR-0013 + the
confirmed code facts (do not invent values):

1. **What this is / scope.** One paragraph: the packaged server instance (distinct from the `.codes`
   verifier app), deployed by iscc-infra behind caddy-docker-proxy; the testnet first consumer is
   `monitor-test.iscc.io`. Link CLAUDE.md "Running a local dev instance" as the env-var source of truth.

2. **Image & tags.** `ghcr.io/iscc/iscc-monitor`, published on push to `develop` (`publish.yml`),
   tagged `:develop` (floating) + `:sha-<short>` (immutable — pin/rollback with the SHA). The image is
   self-contained (embeds `/_ds/` incl. `verify.wasm`); no CDN at runtime. Distroless `nonroot` base,
   **uid 65532**. Confirm the live build via `GET /version` (ADR-0013 Decision 9 — git SHA, default
   `dev` when unstamped).

3. **State, volume & backup (closes the persistence `critical`).** `ISCC_MONITOR_DB` points at a file
   on a **mounted volume** (e.g. `/data/monitor.db` on a named Docker volume mounted at `/data`). The
   single SQLite file PLUS its **`-wal` and `-shm` siblings** (WAL is on — `sqlite.go:39`) are the
   entire durable state and the unit of backup — backing up that one set captures all *irreplaceable
   evidence* (ADR-0007). The volume directory must be **writable by uid 65532** (the non-root image
   user). State the baked default realm `/etc/iscc-monitor/realm.txt` and that a deploy may mount its
   own and repoint `ISCC_MONITOR_REALM`.

4. **Migration policy (closes the migration half of the persistence `critical`).** `store.Open` is
   `CREATE TABLE IF NOT EXISTS` only — no on-disk migration mechanism yet (open `normal`). Interim
   policy, explicit: **recreate the volume on a schema change.** A `:develop` bump that adds a column
   over a populated volume would otherwise fail with `no such column`. Acceptable for a throwaway
   testnet; revisit when the migration mechanism lands. Link the open issue as the known constraint.

5. **Reverse-proxy & port contract (part of the exposure `critical`).** The container binds `:9464` on
   the internal Docker network and **publishes NO host port** — only caddy-docker-proxy terminates TLS
   and publishes 80/443 (iscc-infra invariant: Docker bypasses host firewalls). The monitor needs its
   own vhost/subdomain (root-absolute `/_ds/` + root-mounted hub routes), satisfied by
   `monitor-test.iscc.io`.

6. **Route exposure / `/metrics` decision (closes the exposure `critical`).** All routes share one
   `:9464` mux, so they separate only at the proxy, never by port. Give explicit allow/deny guidance:
   the dashboard / dossier / mirror / proof / `/healthz` surfaces are **public by design** (verifiable
   cache, CORS `*`); `/metrics` exposes operational internals (poll failures, violation counts, per-hub
   status). State the v1 fact that the monitor holds **no secret / no signing key** (cosigning is M7,
   deferred), so nothing leaks a credential. Then record the decision per ADR-0013 Decision 6: whether
   `/metrics` is public or proxy-denied (scraped only on the internal network) is the **operator's
   per-instance choice** — recommend denying `/metrics` at Caddy for the public testnet vhost and
   scraping it internally, and note infra enforces it in the Caddy labels.

7. **Egress (closes the egress `critical`).** Outbound HTTPS to: each realm hub's `/log` tiles AND its
   `/.well-known/did.json` (did:web key resolution, ADR-0009) — **per-realm DNS, no fixed IP allow-list**;
   and the OTS calendar `https://alice.btc.calendar.opentimestamps.org` (ADR-0004,
   `otsclient/client.go:40`). DO droplets default-allow egress — the point is it is a conscious
   requirement. Then a short **footprint** subsection: honest ballpark RAM / CPU / mirror-BLOB
   disk-growth-per-hub for the 2-hub testnet realm, labelled as estimates to refine against live data
   (do not invent measured numbers); flag setting a DO disk-usage alert.

8. **Graceful shutdown.** SIGTERM (not SIGINT alone) cancels the run context, drains the in-flight poll,
   runs `store.Close()`, exits 0 (ADR-0013 Decision 7, the `shutdown_test.go`-proven trap). Operators set
   a Compose `stop_grace_period` covering a clean poll + store flush.

9. **Quick-start snippet.** A copy-pasteable `docker run` (or Compose fragment) with a mounted volume +
   the required env, mirroring the CLAUDE.md dev snippet but for the GHCR image; link CLAUDE.md for the
   full env table.

Keep it evergreen and operator-facing (no CID-loop/agent jargon). Match the confirmed code facts exactly
— uid **65532**, port **:9464**, baked realm **/etc/iscc-monitor/realm.txt**, calendar
**alice.btc.calendar.opentimestamps.org**, WAL siblings **-wal/-shm**. No new spec claims beyond ADR-0013.

## Verification
- `mise run check` is green (doc-only change — no Go/test/Dockerfile touched; gate must stay green).
- `test -f deploy/OPERATING.md` exits 0 (the tracked doc exists at a fixed path).
- `grep -q 'monitor-test.iscc.io' deploy/OPERATING.md` (names the first-consumer instance).
- `grep -Eq '65532' deploy/OPERATING.md` (states the non-root uid the volume must be writable by).
- `grep -Eq ':9464' deploy/OPERATING.md` AND `grep -Eqi 'no host port|publishes no' deploy/OPERATING.md`
  (the bind + no-host-publish reverse-proxy contract).
- `grep -Eqi -- '-wal' deploy/OPERATING.md && grep -Eqi 'backup' deploy/OPERATING.md` (the single-file +
  WAL-siblings backup unit).
- `grep -q '/metrics' deploy/OPERATING.md` (the public/deny exposure decision is recorded).
- `grep -q 'alice.btc.calendar.opentimestamps.org' deploy/OPERATING.md` AND
  `grep -q 'did.json' deploy/OPERATING.md` (the egress endpoints).
- `grep -Eqi 'recreate the volume|schema change' deploy/OPERATING.md` (the interim migration policy).
- Manual content check (rare non-runnable item): the doc answers every "Verify fixed" line of the three
  open `critical` issues, so `review` can delete all three.

## Done When
`deploy/OPERATING.md` exists, every grep above passes, `mise run check` is green, and the doc satisfies
the M-Deploy operability-doc Verify item — answering all three remaining `critical` issues (persistence
+ volume + uid + migration policy; the `/metrics` exposure decision; egress endpoints + footprint;
reverse-proxy `:9464` contract) so `review` clears the `critical` count to 0.
