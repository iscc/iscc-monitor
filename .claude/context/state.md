<!-- assessed-at: ee2ff9760506aa3c37d4598522c61230a6f8ed99 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — pure did:web resolution chain complete (origin, vkey, parser, URL mapping); no follower/store/binary yet

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24`, dependency-free) and the
pure half of M1's trust-root path is now complete and golden-tested: `origin()` in
`internal/logclient`, the did:key → C2SP verifier-key derivation, the `did.json` document parser, and
`DocumentURL()` mapping `did:web:<msid>` → its `did.json` HTTPS URL. The bulk of M1 (HTTP did:web
fetch + status mapping, config, realm registry, the follower with its three-trigger consistency
check, SQLite store, coverage, structured logs, `/metrics`, restart survival) is still unwritten, and
there is no binary entrypoint. Last `review` verdict (HEAD, `ee2ff97`) is **PASS** with the gate
recorded green; branch `develop` is in sync with `origin/develop` (pushed on PASS).

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; only `internal/didweb/` changed since last assessment):
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; tested by `TestOrigin`
    (trailing-slash, no-scheme, port, whitespace) + `TestOriginErrors`. Uses `net/url`. Unchanged.
  - `internal/didweb/vkey.go` — `b58decode` / `pubkeyFromDID` / `keyID` / `VerifierKey` (exported), a
    stdlib-only port of `.claude/derive_vkey.py`; tested by `TestVerifierKey` (literal C2SP vkeys for
    both live hubs: `sb0.iscc.id/log+40b74463+…`, `sb1.amlet.id/log+22b08f3e+…`) +
    `TestPubkeyFromDIDErrors`.
  - `internal/didweb/resolve.go` — pure (`encoding/json`/`fmt`/`time` only) `ParseDIDDocument([]byte)
    (DIDKey, error)` + `assertionKey` + `parseTime`. Decodes a hub `did.json`, resolves the
    `assertionMethod`-referenced verification method (string `#fragment` ref OR inline object),
    extracts `publicKeyMultibase` via `pubkeyFromDID`, and surfaces (does NOT enforce) CID 1.0
    validity timestamps (`ValidFrom`/`ValidUntil`/`Revoked`). Tested by `TestParseDIDDocument`
    (sb0/sb1 golden), `TestParseDIDDocumentErrors`, `TestParseDIDDocumentInlineAssertion`. Fixtures at
    `internal/didweb/testdata/sb0.iscc.id_did.json` and `…/sb1.amlet.id_did.json`.
  - `internal/didweb/url.go` (NEW since last assessment) — pure (`fmt`/`net/url`/`strings`; no
    `net`/`net/http`/`sql`) `DocumentURL(did string) (string, error)`: maps a `did:web:<msid>` to its
    `did.json` HTTPS URL per the W3C did:web method spec (host[:port] percent-decoded, path segments
    joined; `/.well-known/did.json` when no path). Tested by `TestDocumentURL` (incl. sb0/sb1 goldens
    and `example.com%3A3000:user:alice` → `https://example.com:3000/user/alice/did.json`) +
    `TestDocumentURLErrors` (missing prefix, empty msid/host, invalid percent-encoding).
  - Exported follower-facing surface is exactly `DocumentURL` + `ParseDIDDocument` + `VerifierKey` +
    `DIDKey`; `pubkeyFromDID`/`keyID`/`b58decode` stay private.
  - Oracle parity re-confirmed by `review` against `derive_vkey.py` for the resolve→vkey chain (the
    export renames moved no derived bytes).
- Missing (the majority of M1 — all stateful/networked work):
  - did:web HTTP fetch wired at the outbound-fetch seam: inject a `Fetcher`/`*http.Client`, call
    `DocumentURL` → fetch → `ParseDIDDocument` → `VerifierKey`, and map outcomes to hub status
    (`unresolvable` on fetch/parse fail, `unverified` on signature mismatch). The pure string→URL and
    parse halves exist; nothing fetches or consumes them yet. This crosses into `net`/`sql` and must
    live OUTSIDE `internal/didweb` to keep that package pure/WASM-shareable.
  - config, realm registry (domains only), the per-hub follower (Ed25519 signed-note verify +
    fork/shrink/equivocation consistency check + persist + freeze + alert-once + backed-off polling),
    per-network SQLite store, coverage tracking (`monitored_since`), structured logs, `/metrics`,
    restart survival. The `hub_keys` cache + now-vs-window validity enforcement (consuming
    `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`) belongs to the store/follower step.
  - `cmd/` is absent — no binary entrypoint yet.
- Fixtures: did:web golden fixtures exist as flat files under `internal/didweb/testdata/`. There is
  **no** `testdata/live/` tree (no sb0/sb1 checkpoints, tiles, or entry bundles for follower/aggregator
  work).
- Reuse imports not yet wired: no `transparency-dev/*`, `golang.org/x/mod/sumdb/note`,
  `nbd-wtf/opentimestamps`, or `modernc.org/sqlite` in the source tree (only a doc comment in
  `vkey.go` mentions them); `go.mod` has no `require` block.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
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
  (`go build`/`vet`/`test` exit 0, `gofmt -l .` empty, `GOOS=js GOARCH=wasm go build ./internal/didweb`
  succeeds) at HEAD `ee2ff97`.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop`. **No `.github/workflows/` — no CI configured.** Whoever wires CI must avoid
  `go build ./...` over the gitignored `cauldron/` (its module-less reference trees break a fresh
  build) — see learnings.
- The `notecheck` external-oracle CI job does not exist yet; correct at this stage (no
  end-to-end signature-verification code in tree). The trust-root oracle gate is `derive_vkey.py`
  parity, which `review` re-ran and confirmed.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): wire the did:web HTTP fetch at the
follower's outbound-fetch seam — inject a `Fetcher`/`*http.Client`, call `DocumentURL(did)` → fetch →
`ParseDIDDocument(bytes)` → `VerifierKey(origin, key.PublicKey)`, and map outcomes to hub status
(`unresolvable` on fetch/parse failure, `unverified` on signature mismatch). This crosses into
`net`/`sql`, so it must live outside `internal/didweb`. From there the follower + per-network SQLite
store + the three consistency-check triggers + the `hub_keys` validity-window enforcement + coverage +
restart-survival are the path to completing M1's Verify criteria. No CI is configured — flag for
whoever sets up the GitHub workflow.
