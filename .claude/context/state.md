<!-- assessed-at: 244519e9b0435e1c6299b0db1b17844a2797a19c -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — trust-root primitives landed (origin, vkey, pure did:web parser); no follower/store/binary yet

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24`, dependency-free) and three
pure M1 units exist and are golden-tested: `origin()` in `internal/logclient`, the did:key → C2SP
verifier-key derivation, and now a pure `did.json` document parser in `internal/didweb`. The bulk of
M1 (HTTP did:web fetch + status mapping, config, realm registry, the follower with its three-trigger
consistency check, SQLite store, coverage, structured logs, `/metrics`, restart survival) is still
unwritten, and there is no binary entrypoint. Last `review` verdict (HEAD, `244519e`) is **PASS**
with the gate recorded green.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (re-checked this assessment):
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; tested by `TestOrigin`
    (trailing-slash, no-scheme, port, whitespace) + `TestOriginErrors`. Uses `net/url`.
  - `internal/didweb/vkey.go` — `b58decode` / `pubkeyFromDID` / `keyID` / `verifierKey`, a stdlib-only
    port of `.claude/derive_vkey.py`; tested by `TestVerifierKey` (literal C2SP vkeys for both live
    hubs: `sb0.iscc.id/log+40b74463+…`, `sb1.amlet.id/log+22b08f3e+…`) + `TestPubkeyFromDIDErrors`.
  - `internal/didweb/resolve.go` (NEW since last assessment) — pure (`encoding/json`/`fmt`/`time`
    only; no `net`/`sql`) `parseDIDDocument([]byte) (DIDKey, error)` + `assertionKey` + `parseTime`.
    Decodes a hub `did.json`, resolves the `assertionMethod`-referenced verification method (string
    `#fragment` ref OR inline object), extracts `publicKeyMultibase` via `pubkeyFromDID`, and surfaces
    (does NOT enforce) CID 1.0 validity timestamps. Tested by `TestParseDIDDocument` (sb0/sb1 golden),
    `TestParseDIDDocumentErrors` (bad JSON, empty `verificationMethod`, missing multibase, no
    `assertionMethod`, bad multibase), `TestParseDIDDocumentInlineAssertion`. Fixtures captured at
    `internal/didweb/testdata/sb0.iscc.id_did.json` and `…/sb1.amlet.id_did.json`.
  - Oracle parity re-confirmed by `review` against `derive_vkey.py` for the resolve→vkey chain.
- Missing (the majority of M1):
  - did:web HTTP fetch + `did:web:<domain>` → `https://<domain>/.well-known/did.json` mapping wired at
    the outbound-fetch seam; status mapping (`unresolvable` on fetch/parse fail, `unverified` on
    signature mismatch). The pure parser exists; nothing fetches or consumes it yet.
  - config, realm registry (domains only), the per-hub follower (Ed25519 signed-note verify +
    fork/shrink/equivocation consistency check + persist + freeze + alert-once + backed-off polling),
    per-network SQLite store, coverage tracking (`monitored_since`), structured logs, `/metrics`,
    restart survival.
  - `cmd/` is absent — no binary entrypoint yet.
- Fixtures: did:web golden fixtures exist as flat files under `internal/didweb/testdata/`. There is
  **no** `testdata/live/` tree (no sb0/sb1 checkpoints, tiles, or entry bundles for follower/aggregator
  work).
- Reuse imports not yet wired: no `transparency-dev/*`, `golang.org/x/mod/sumdb/note`,
  `nbd-wtf/opentimestamps`, or `modernc.org/sqlite` in the source tree (only a doc comment mentions
  them); `go.mod` has no `require` block.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `verifierKey`
  byte-match for both hubs — **met**. The synthetic fork/shrink/equivocation → `violations.kind` +
  `frozen=1` + exactly-one-alert + other-hubs-unaffected + restart-survival half of M1 is **not
  started** (no follower/store).

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present and `mise run check` runnable; latest `review` handoff records the gate green
  (`go build`/`vet`/`test` exit 0, `gofmt -l .` empty) at HEAD `244519e`.
- Remote `origin` configured (github.com/iscc/iscc-monitor) but **no `.github/workflows/` — no CI
  configured**. Whoever wires CI must avoid `go build ./...` over the gitignored `cauldron/` (its
  module-less reference trees break a fresh build) — see learnings.
- The `notecheck` external-oracle CI job does not exist yet; correct at this stage (no
  signature-verification code in tree). The trust-root oracle gate is `derive_vkey.py` parity, which
  `review` re-ran and confirmed.
- Working branch is `develop`; `review` pushes on PASS.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): wire the did:web HTTP fetch at the
outbound-fetch seam — map `did:web:<domain>` → `https://<domain>/.well-known/did.json`, inject a
`Fetcher`/`*http.Client`, fetch + call `parseDIDDocument`, and map outcomes to hub status
(`unresolvable` on fetch/parse failure, `unverified` on signature mismatch). Export the minimal
`didweb` surface (`parseDIDDocument`/`DIDKey`/`pubkeyFromDID`/`verifierKey`) the follower needs. From
there the follower + per-network SQLite store + the three consistency-check triggers + coverage +
restart-survival are the path to completing M1's Verify criteria. No CI is configured — flag for
whoever sets up the GitHub workflow.
