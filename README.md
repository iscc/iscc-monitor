# iscc-monitor

An independent **Trust & Transparency** service for the ISCC-Hub network. It follows every hub's
[tlog-tiles](https://c2sp.org/tlog-tiles) transparency log, verifies Ed25519 signed-note signatures
and RFC-6962 consistency, mirrors the logs as SQLite BLOBs, and publishes verifiable evidence — so no
party has to trust the monitor itself.

The monitor is a **verifiable cache**, not a trusted oracle: clients re-verify whatever it serves
against the hub's own did:web signature and the Merkle math, removing the monitor from the trust path.
It autonomously detects a **self-consistency violation** (a hub rewriting history against this
monitor — successive checkpoints that fail RFC-6962 consistency), which freezes the hub and raises an
alert; a **split view** (a hub showing different histories to different audiences) is detectable only
by *comparison* against the monitor as a reference anchor.

## Stack

- **Go 1.26, `CGO_ENABLED=0`** — a single static binary, `cmd/iscc-monitor`, configured entirely
  through environment variables.
- Pure-Go SQLite (`modernc.org/sqlite`); one SQLite file per network holds the whole durable state.
- Reuses the transparency-dev tlog-tiles stack and the [`iscc-lib`](https://github.com/iscc/iscc-lib)
  Go codec for ISO 24138 en/decoding. See [`.claude/adr/0011-iscc-lib-codec-dependency.md`](.claude/adr/0011-iscc-lib-codec-dependency.md)
  and [`.claude/adr/0003-client-verification-and-in-browser-verifier.md`](.claude/adr/0003-client-verification-and-in-browser-verifier.md)
  for the reuse posture.

## Build & run

Run a local instance against the 2-hub testnet realm with a short poll cadence:

```sh
go build -o /tmp/iscc-monitor ./cmd/iscc-monitor
ISCC_MONITOR_DB=/tmp/monitor-dev.db \
ISCC_MONITOR_REALM=internal/registry/testdata/realm.txt \
ISCC_MONITOR_NORMAL=30s \
ISCC_MONITOR_ADDR=0.0.0.0:41464 \
/tmp/iscc-monitor
```

After the first poll (a few seconds) the HTTP surface is live: `GET /` (server-rendered dashboard),
`GET /healthz` (liveness), `GET /metrics` (Prometheus), plus the per-hub log, certificate, and
verify-for-me routes.

Two variables are **required** and have no default — `ISCC_MONITOR_DB` (the SQLite file path; there is
no safe default, it must point at your own storage) and `ISCC_MONITOR_REALM` (the realm-membership
document). The poll intervals default (`ISCC_MONITOR_NORMAL=5m`, `ISCC_MONITOR_FROZEN=1h`) and the
masthead identity keys are optional. For the full variable list and every endpoint, see the
[Running a local dev instance](CLAUDE.md#running-a-local-dev-instance) section of `CLAUDE.md` — the
single authoritative env-var table.

## Quality gate

Quality gates run via [mise](https://mise.jdx.dev/):

- `mise run check` — the full gate (`go build ./... && go vet ./... && go test ./...`); it must stay
  green.
- `mise run fmt` — format all Go source in place (or check with `gofmt -l .`).

## Deployment

A container image is published to GHCR (`ghcr.io/iscc/iscc-monitor`). See
[`deploy/OPERATING.md`](deploy/OPERATING.md) for the operator contract — image tags, the SQLite volume
and backup unit, the non-root uid, the reverse-proxy / port invariant, egress, and the `/metrics`
exposure decision.

## Specs & pointers

The specifications are the source of truth:

- [`.claude/prd/`](.claude/prd/) — the product requirements (v1 scope and guarantees).
- [`.claude/adr/`](.claude/adr/) — the architecture decision records (ADR-0001 … ADR-0013).
- [`CLAUDE.md`](CLAUDE.md) — **agent-facing** project instructions, conventions, the authoritative
  env-var / endpoint reference, and the canonical glossary (hub status, coverage, comparison anchor,
  Bitcoin anchoring, proof bundle, and the rest).

`CLAUDE.md` is the agent-facing contributor reference and
[`.claude/context/README.md`](.claude/context/README.md) is the CID-loop-internal context pack; this
`README.md` is the human-facing front door for the repository.
