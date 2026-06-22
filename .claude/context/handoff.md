## 2026-06-22 — Write the deployment/operability doc (`deploy/OPERATING.md`)

**Done:** Added the tracked operator-facing `deploy/OPERATING.md` — the deployment/operability
contract for a server instance — sourcing every claim from ADR-0013 + confirmed code facts (uid
65532, bind `:9464`, baked realm `/etc/iscc-monitor/realm.txt`, WAL `-wal`/`-shm` siblings, OTS
calendar host). Doc-only: no Go, Dockerfile, or workflow touched. It answers all three remaining
`critical` iscc-infra ops asks (persistence/volume/uid/migration, `/metrics` exposure, egress +
footprint) so `review` can clear the `critical` count to 0.

**Files changed:**
- `deploy/OPERATING.md` (new): sections — What this is / scope (links CLAUDE.md "Running a local dev
  instance" as the env-var source of truth, names `monitor-test.iscc.io` as first consumer); Image &
  tags (`:develop` floating + `:sha-<short>` immutable, self-contained, uid 65532, `GET /version`
  provenance); State/volume/backup (`/data/monitor.db` on a mounted volume, the `.db`+`-wal`+`-shm`
  backup unit, single-writer, uid-65532-writable, baked realm); Migration policy (interim "recreate
  the volume on a schema change", links the open issue); Reverse-proxy & port (binds `:9464`,
  publishes no host port, own vhost); Route exposure & `/metrics` decision (public-by-design allow
  list, no-secret/no-signing-key, recommend denying `/metrics` at Caddy); Egress (hub `/log` +
  `/.well-known/did.json` + `alice.btc.calendar.opentimestamps.org`, per-realm DNS, labelled
  footprint estimates + DO disk alert); Graceful shutdown (SIGTERM drain + `stop_grace_period`);
  Quick start (Compose fragment + `docker run`, no host port).

**Verification:** `mise run fmt` → clean; `mise run check` → green (all 28 packages `ok`; doc-only,
no Go/test/Dockerfile/workflow touched). All 11 `next.md` Verification greps PASS (file exists;
`monitor-test.iscc.io`; `65532`; `:9464`; `no host port|publishes no`; `-wal` + `backup`; `/metrics`;
`alice.btc.calendar.opentimestamps.org` + `did.json`; `recreate the volume|schema change`). Manual
content cross-check: every "Verify fixed" line of all three open `critical` issues (persistence,
`/metrics` exposure, egress+footprint) is answered — see Notes. CLAUDE.md anchor
`#running-a-local-dev-instance` matches the live `## Running a local dev instance` heading;
`deploy/OPERATING.md` is git-tracked and NOT in `.dockerignore`.

**Next:** The root `README.md` (the last `target.md` "Done When" gate, a `normal` issue) — the human
front door: what iscc-monitor is (verifiable cache, not trusted oracle), the Go 1.26 / `CGO_ENABLED=0`
/ single-binary stack, a build+run snippet against the testnet realm, `mise run check`, and pointers
to `.claude/prd` / `.claude/adr` / the CLAUDE.md glossary — link CLAUDE.md "Running a local dev
instance" rather than duplicating the env table.

**Notes:**
- This doc is the single artifact that closes all THREE remaining open `critical` issues. Mapping
  for `review` to delete them:
  - "Persistence contract for the SQLite DB volume + acknowledge the in-place migration hazard" →
    sections **State, volume & backup** (path + volume + `.db`/`-wal`/`-shm` backup unit +
    confirmation that one file-set captures all durable state + uid 65532) and **Migration policy**
    (interim "recreate the volume on a schema change", links the open `normal` migration issue).
  - "Decide which routes are safe to publish at the public vhost (especially /metrics)" → section
    **Route exposure & the `/metrics` decision** (explicit public allow-list, the no-secret /
    no-signing-key confirmation, the ADR-0013 Decision-6 per-instance operator choice, and the
    recommendation to deny `/metrics` at Caddy and scrape it internally).
  - "Document egress + resource footprint for box sizing" → sections **Egress** (hub `/log` +
    `/.well-known/did.json` + `alice.btc.calendar.opentimestamps.org`, per-realm DNS / no fixed IP
    allow-list) and **Footprint** (labelled ballpark RAM / CPU / mirror-BLOB disk-growth for the
    2-hub testnet realm + the DO disk-usage-alert flag).
- Footprint numbers are deliberately HONEST ballpark ranges labelled "estimates ... not measured
  benchmarks" per `next.md` Not-In-Scope — no invented precision. Refine against live testnet data.
- Strictly doc-only and within scope: no Dockerfile / workflow / Go change, no `/metrics` auth flag,
  no second listener, no `publish.yml` ref-guard fix (those stay their own open issues). The
  `/metrics`-deny and the `publish.yml` `workflow_dispatch` ref-guard remain open backlog items
  (`normal`), untouched here.
- Oracle/conformance gate N/A — no proof/verify/didweb/merkle/signature path touched. No visual
  surface changed (a doc, no template).
