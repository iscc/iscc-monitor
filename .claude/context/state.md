<!-- assessed-at: (none — pre-bootstrap, no commits yet) -->

# Project State

## Status: IN_PROGRESS

## Phase: pre-bootstrap (design complete, no code yet)

The repository holds the full design (`.claude/adr`, `.claude/prd`, `.claude/plans`, the `CLAUDE.md`
glossary) and gitignored read-only reference copies under `cauldron/`. There is **no Go module, no
source, and no commits yet**. The first CID iteration must bootstrap the module and begin M1.

## M1 — Read-only Monitor

**Status**: not started

- Missing: everything — `go.mod`, `cmd/iscc-monitor/`, `internal/{config,registry,didweb,logclient,store,index,proof,ots,api,web,metrics,supervisor}`.
- Highest-leverage first step (plan "Correctness rules" #1, the highest-probability bug): the
  `origin()` helper (`origin = <domain>/log`, not the bare domain) and the did→`verifierKey`
  derivation, golden-tested against both live testnet hubs using `.claude/derive_vkey.py` as the
  reference. These are pure functions with no I/O — the cheapest verifiable increment.

## M2 — Aggregator · M3 — Trust API + dashboard · WASM · OTS

**Status**: not started.

## Quality gates

**Status**: harness present (`mise.toml`) but **red until the module is bootstrapped** — `go build`
fails with no `go.mod`. The first `advance` step creates `go.mod`, after which `mise run check`
becomes the live gate.

## Next Milestone

Bootstrap the Go module (`go mod init github.com/iscc/iscc-monitor`) and land M1's first verifiable
unit: `internal/logclient/origin.go` (and/or `internal/didweb/vkey.go`), golden-tested against
`sb0.iscc.id/log` and `sb1.amlet.id/log` via `.claude/derive_vkey.py`.
