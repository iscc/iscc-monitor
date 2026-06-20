# Target — iscc-monitor v1

> Authoritative specs: `.claude/prd/0001-iscc-monitor-v1.md`, `.claude/plans/cosmic-baking-octopus.md`,
> `.claude/adr/0001`–`0009`, glossary in `CLAUDE.md`. Where this file and an ADR/PRD disagree, the
> ADR/PRD wins. This file is the *fixed target* the CID loop advances toward — the desired end-state
> plus the bar every increment is verified against.

## Stack (locked — ADR-0003)

Go 1.24, `CGO_ENABLED=0`, module `github.com/iscc/iscc-monitor`. **Reuse, do not reimplement** the
transparency stack: `transparency-dev/tessera` (`client`, `api`, `api/layout`, `fsck`),
`transparency-dev/merkle` (`rfc6962`, `proof`), `transparency-dev/formats`,
`golang.org/x/mod/sumdb/note`, `nbd-wtf/opentimestamps`, `modernc.org/sqlite`. Read-only vendored
reference copies live under `cauldron/` (gitignored) — port/oracle against them, never import them.

## Quality bar (the gate — non-negotiable)

`mise run check` is green: `go build ./...`, `go vet ./...`, `go test ./...` all pass and `gofmt -l .`
is empty. Testing is **integration-first at the outbound-fetch seam** (PRD "Testing Decisions"):
drive follower → store → API against fixtures and assert on *observable outputs* (SQLite state,
`/metrics`, REST / proof-bundle responses, verifier verdicts) — never on follower internals. Pure
units (`proof/verify`, `didweb`) get table-driven golden-vector tests, with the two live testnet
hubs `sb0.iscc.id/log` and `sb1.amlet.id/log` as golden vectors (`.claude/derive_vkey.py` is the
reference). **Never weaken a gate to pass** — no `//nolint`, `t.Skip`, swallowed errors, or build-tag
exclusions to dodge checks. Fix the root cause.

**Oracle / conformance gate (crypto path — PRD "External oracle gates").** Any change to signature
verification, RFC-6962 / Merkle consistency, or proof code must pass the monitor's **conformance
tests** before a PASS verdict — they run as ordinary `go test` / `mise run check` once the package
exists and encode: golden-vector parity (`derive_vkey.py` vectors), `fsck` root-rebuild over the
`SQLiteFetcher`, and the inclusion cross-check against the hub's own `IsccLogInclusionProof`. The
**fully-independent** external oracle `notecheck` (a separate binary fed checkpoint text) is built and
**shelled out in CI** — not `go run` from gitignored `cauldron/` (those are module-less `main.go`
files with external deps: no local compile path). Independence honesty: `notecheck` and the
hub-computed `IsccLogInclusionProof` are the true external oracles; the ported `runfsck` hasher /
`fsck`-over-`SQLiteFetcher` is an in-process structural **self-check** (it shares the monitor's code, so
it can share its bugs). The LLM reviewer is **not** ground truth for crypto correctness — a
green-but-wrong verify (e.g. one accepting a malformed proof) must not ship on `mise run check` + an
LLM PASS alone. A conformance/oracle regression is a gate failure.

**Coverage (ratified decision — no percentage gate).** There is **deliberately no coverage-percentage
gate**. This overrides the general "every project needs tests / 100 % coverage" expectation (and
iscc-lib's 100 % gate) in favour of *this* project's PRD "Testing Decisions": correctness is proven by
the seam-based integration tests + golden vectors + the external oracles above — not by a line-coverage
number, which would force brittle tests on wiring / `main` and tempt the internal-detail assertions the
PRD forbids. Revisit only if the testing strategy itself changes.

## Milestones (advance in ADR-0004 order)

### M1 — Read-only Monitor  `[not started]`

config + realm registry (domains only) + did:web resolution + `vkey`/`origin` (golden-tested) +
per-hub follower (verify Ed25519 signed-note + the three-trigger consistency check
fork/shrink/equivocation, persist, freeze) + per-network SQLite + coverage tracking
(`monitored_since`) + structured logs + `/metrics`. Survives restart.
**Verify:** `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"`; `verifierKey` byte-matches
`derive_vkey.py` for both hubs; synthetic fork/shrink/equivocation each → correct `violations.kind` +
`frozen=1` + exactly one alert + other hubs unaffected + evidence survives restart.

### M2 — Aggregator  `[not started]`

tiles + entry bundles as SQLite BLOBs (partial-tile discipline: `is_full` only at `width==256`) +
`SQLiteFetcher` implementing `client.Fetcher` so `fsck.New(...)` runs on the local store +
`iscc_index` (schema-agnostic; `iscc_id → seq` one-to-many; stores raw `note.$schema`) +
`inclusion`/`consistency`/`entries` served from the local store via `ProofBuilder` (never re-hitting
the hub). **Verify:** `fsck` rebuilds each accepted root from the `SQLiteFetcher`; computed inclusion
proof matches the hub's `evidence.IsccLogInclusionProof` for sampled `iscc_id`s.

### M3 — Trust API + dashboard  `[not started]`

REST surface (CORS on every public GET) + `verify-for-me` + server-rendered dashboard (status,
coverage, lag, violations, OTS) + log browser + raw tlog-tiles mirror at canonical paths.

### WASM verifier upgrade  `[not started]`

`internal/proof/verify` → `GOOS=js GOARCH=wasm`, lazy-loaded progressive enhancement on the M3
dashboard; reproducible build + published hash + SRI pin (ADR-0003). **Verify:** identical vectors
yield identical verdicts (WASM vs server); the verifier artifact hash matches the published value.

### OTS / Bitcoin anchoring  `[not started]`

stamp each distinct observed root daily (`UNIQUE(hub, tree_size, root)`) + background upgrade loop
(pending → Bitcoin-confirmed) + serve `.ots`; **never blocks the follower**. **Verify:** a stamped
root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client.

### M7 — DEFERRED (out of v1)

multi-monitor gossip + cosigning (C2SP witness cosignatures) + witness endpoint
(`tlog-witness add-checkpoint`) + cross-monitor split-view comparison. Out of scope; never blocks DONE.

## Done When

Every v1 milestone (M1 → OTS) meets its **Verify** criteria with `mise run check` green and no open
`critical` or `normal` issue in `issues.md`. M7 is explicitly out of scope.
