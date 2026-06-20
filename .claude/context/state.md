<!-- assessed-at: c59d3809415cc4f591a48c1a1bd3426d49d4d728 -->

# Project State

## Status: IN_PROGRESS

## Phase: pre-bootstrap (design complete, scaffolding committed, no Go code yet)

Three commits exist, but they are all design/scaffolding — specs (`.claude/prd`, `.claude/plans`,
`.claude/adr`), the `CLAUDE.md` glossary, `mise.toml`, and a devcontainer. There is **no Go module,
no `cmd/`, no `internal/`, and not a single line of Go**. The next CID iteration must bootstrap the
module and begin M1.

## M1 — Read-only Monitor
**Status**: not started
- Verified present: `mise.toml` with the `check`/`build`/`vet`/`test`/`fmt` tasks; the reference
  oracle `.claude/derive_vkey.py`; full specs/ADRs.
- Missing: everything Go — `go.mod` (module `github.com/iscc/iscc-monitor`), `cmd/iscc-monitor/`,
  and `internal/{config,registry,didweb,logclient,store,index,proof,ots,api,web,metrics,supervisor}`.
- Missing fixtures: `testdata/live/` is absent (no sb0/sb1 checkpoints, tiles, or `did.json`).
- None of the reuse imports (`transparency-dev/*`, `golang.org/x/mod/sumdb/note`,
  `nbd-wtf/opentimestamps`, `modernc.org/sqlite`) are wired in — there is no code to import them.
- Highest-leverage first unit (plan "Correctness rules" #1): the `origin()` helper
  (`origin = <domain>/log`) and the did→`verifierKey` derivation — pure, no-I/O functions
  golden-tested against `sb0.iscc.id/log` and `sb1.amlet.id/log` via `.claude/derive_vkey.py`.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: red (not yet runnable)
- No `go.mod` → `go build ./...` fails, so `mise run check` cannot pass yet. The harness
  (`mise.toml`) is present and correct; it goes live once `advance` runs `go mod init`.
- Remote `origin` is configured (github.com/iscc/iscc-monitor) but there is **no `.github/workflows/`
  — no CI configured**. No `review` PASS verdict on record (first iteration).

## Next Milestone
Bootstrap the Go module (`go mod init github.com/iscc/iscc-monitor`) and land M1's first verifiable
unit: `internal/logclient/origin.go` and/or `internal/didweb/vkey.go`, golden-tested against
`sb0.iscc.id/log` and `sb1.amlet.id/log` using `.claude/derive_vkey.py` as the oracle.
