<!-- assessed-at: 5c82580db8abf155ad5a91e0f4f5ad55cac19b94 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — module bootstrapped, two pure trust-root units landed

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24`) and the first two M1 units
exist and are golden-tested: `origin()` in `internal/logclient` and the did:key → C2SP
verifier-key derivation in `internal/didweb`. The bulk of M1 (config, registry, did:web resolution,
the follower with its three-trigger consistency check, SQLite store, coverage, `/metrics`) is still
unwritten. Last `review` verdict (HEAD, `5c82580`) is **PASS** with the gate green.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present:
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; tested by `TestOrigin`
    (incl. trailing-slash, no-scheme, port, whitespace) + `TestOriginErrors`. Uses `net/url`.
  - `internal/didweb/vkey.go` — `b58decode` / `pubkeyFromDID` / `keyID` / `verifierKey`, a stdlib-only
    port of `.claude/derive_vkey.py`; tested by `TestVerifierKey` (asserts literal C2SP vkeys for both
    live hubs: `sb0.iscc.id/log+40b74463+…`, `sb1.amlet.id/log+22b08f3e+…`) + `TestPubkeyFromDIDErrors`.
  - Oracle parity confirmed by `review`: Go vkeys byte-match `derive_vkey.py` for both hubs.
- Missing (the majority of M1):
  - `internal/didweb/resolve.go` — fetch `/.well-known/did.json`, parse `verificationMethod`, honor
    CID 1.0 validity windows / `revoked`; inject HTTP client at the outbound-fetch seam.
  - config, realm registry (domains only), the per-hub follower (Ed25519 signed-note verify +
    fork/shrink/equivocation consistency check + persist + freeze), per-network SQLite store,
    coverage tracking (`monitored_since`), structured logs, `/metrics`, restart survival.
  - `cmd/` is absent — no binary entrypoint yet.
- Missing fixtures: `testdata/live/` does not exist (no sb0/sb1 checkpoints, tiles, or `did.json`).
- Reuse imports not yet wired: no `transparency-dev/*`, `golang.org/x/mod/sumdb/note`,
  `nbd-wtf/opentimestamps`, or `modernc.org/sqlite` in the source tree; `go.mod` is dependency-free.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `verifierKey`
  byte-match — **met**. The synthetic fork/shrink/equivocation → `violations.kind` + `frozen=1` +
  one alert + restart-survival half of M1 is **not started** (no follower/store).

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present; the latest `review` handoff records `mise run check` green (build/vet/test exit 0)
  and `gofmt -l .` empty at HEAD `5c82580`.
- Remote `origin` configured (github.com/iscc/iscc-monitor) but **no `.github/workflows/` — no CI
  configured**; `gh run list --branch develop` returns no runs. Whoever wires CI must avoid checking
  out gitignored `cauldron/` (its module-less reference trees break `go build ./...`) — see learnings.
- Working branch is `develop`; `review` pushed on PASS.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): `internal/didweb/resolve.go` — resolve a
hub's `did:web` document, parse `verificationMethod` to the `z6Mk` did:key, honor CID 1.0 validity
windows / `revoked`, and feed the pubkey into `verifierKey`. Inject the HTTP client at the
outbound-fetch seam and test against a real captured `testdata/live/.../did.json` fixture (not a mock).
After resolution, the follower + SQLite store + consistency-check triggers are the path to completing
M1's Verify criteria. No CI is configured — flag for whoever sets up the GitHub workflow.
