<!-- assessed-at: 44cf9e9fc9e36fcfd7d81b1619d44bf2d7d1da89 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — full did:web resolution chain (pure parse/derive + networked fetch seam) complete; no follower/store/binary yet

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24`, dependency-free — no
`require` block) and the entire did:web trust-root resolution chain now exists and is golden-tested:
the pure half in `internal/didweb` (`origin` lives in `logclient`, `DocumentURL`, `ParseDIDDocument`,
`VerifierKey`) plus the networked half in `internal/logclient/didresolve.go`
(`ResolveVerifierKey(ctx, Fetcher, baseURL)` over an injectable `Fetcher` seam, collapsing every
failure to `ErrUnresolvable`). The bulk of M1 (Ed25519 signed-note checkpoint verify, the
three-trigger consistency check, freeze/alert, per-network SQLite store, coverage, `hub_keys` cache +
validity-window enforcement, structured logs, `/metrics`, restart survival) is still unwritten, and
there is no binary entrypoint. Last `review` verdict (HEAD, `44cf9e9`) is **PASS** with the gate
recorded green; branch `develop` is in sync with `origin/develop`.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; only `internal/logclient/` changed since last assessment — the
  new `didresolve.go` + its test + the two did.json fixtures):
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; tested by `TestOrigin`
    (trailing-slash, no-scheme, port, whitespace) + `TestOriginErrors`. Stdlib `net/url` only. Unchanged.
  - `internal/didweb/vkey.go` — `b58decode` / `pubkeyFromDID` / `keyID` / `VerifierKey` (exported), a
    stdlib-only port of `.claude/derive_vkey.py`; tested by `TestVerifierKey` (literal C2SP vkeys for
    both live hubs: `sb0.iscc.id/log+40b74463+…`, `sb1.amlet.id/log+22b08f3e+…`) +
    `TestPubkeyFromDIDErrors`. Unchanged.
  - `internal/didweb/resolve.go` — pure `ParseDIDDocument([]byte) (DIDKey, error)` resolving the
    `assertionMethod` (string `#fragment` ref OR inline object), extracting `publicKeyMultibase`, and
    surfacing (NOT enforcing) CID 1.0 validity timestamps (`ValidFrom`/`ValidUntil`/`Revoked`). Tested
    by `TestParseDIDDocument` (sb0/sb1 golden), `…Errors`, `…InlineAssertion`. Unchanged.
  - `internal/didweb/url.go` — pure `DocumentURL(did) (string, error)` per W3C did:web. Tested by
    `TestDocumentURL` + `TestDocumentURLErrors`. Unchanged.
  - `internal/logclient/didresolve.go` (NEW since last assessment) — the networked did:web resolver:
    `Fetcher` 1-method seam, unexported `httpFetcher` via `NewHTTPFetcher(*http.Client)` (404→
    `os.ErrNotExist`, idiomatic body close), exported sentinel `ErrUnresolvable`, and
    `ResolveVerifierKey(ctx, Fetcher, baseURL)` wiring `origin → did:web:<host%3Aport> → DocumentURL →
    Fetch → ParseDIDDocument → VerifierKey`, collapsing every fetch/parse/derive failure to
    `ErrUnresolvable`. Imports `net/http` — correctly placed in `logclient`, NOT `didweb`, to keep the
    WASM build pure. Tested by `TestResolveVerifierKey` (sb0/sb1 golden via fake fetcher),
    `TestResolveVerifierKeyUnresolvable`, `TestResolveVerifierKeyOverHTTP` (`httptest` TLS server, asserts
    `<addr>/log` prefix), `TestHTTPFetcherNotFound`. Fixtures `internal/logclient/testdata/{sb0,sb1}…` are
    byte-identical to the `internal/didweb` source fixtures (review confirmed no drift).
  - Exported `internal/didweb` follower-facing surface stays `DocumentURL` + `ParseDIDDocument` +
    `VerifierKey` + `DIDKey`; `logclient` exports `Fetcher` + `NewHTTPFetcher` + `ResolveVerifierKey` +
    `ErrUnresolvable`. `pubkeyFromDID`/`keyID`/`b58decode`/`httpFetcher`/`origin` stay private.
  - Oracle parity re-confirmed by `review` against `derive_vkey.py` (both golden vectors byte-exact).
- Missing (the majority of M1 — all signed-note/stateful work):
  - **Ed25519 signed-note checkpoint verification**: parse a hub-signed checkpoint and verify it against
    the key `ResolveVerifierKey` returns; a signature matching no listed key → status `unverified`
    (distinct from `unresolvable`). Needs a `note.Verifier`/signed-note parser and a real checkpoint
    fixture. Nothing consumes the resolved key yet.
  - config, realm registry (domains only), the per-hub follower (the three-trigger
    fork/shrink/equivocation consistency check + persist + freeze + alert-once + backed-off polling),
    per-network SQLite store, coverage tracking (`monitored_since`), structured logs, `/metrics`,
    restart survival. The `hub_keys` cache + now-vs-window validity enforcement (consuming
    `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`, currently parsed-but-unenforced) belongs to the
    store/follower step.
  - `cmd/` is absent — no binary entrypoint yet.
- Fixtures: did:web golden fixtures exist as flat files under `internal/didweb/testdata/` and
  `internal/logclient/testdata/`. There is **no** `testdata/live/` tree (no sb0/sb1 checkpoints, tiles,
  or entry bundles for signed-note/follower/aggregator work) — a prerequisite for the next unit.
- Reuse imports not yet wired: no `transparency-dev/*`, `golang.org/x/mod/sumdb/note`,
  `nbd-wtf/opentimestamps`, or `modernc.org/sqlite` in the source tree (only a doc comment in
  `vkey.go` mentions them); `go.mod` has no `require` block.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (and now also exercised end-to-end through the networked
  resolver). The synthetic fork/shrink/equivocation → `violations.kind` + `frozen=1` +
  exactly-one-alert + other-hubs-unaffected + restart-survival half of M1 is **not started** (no
  signed-note verify, follower, or store).

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
  succeeds, `internal/didweb` dep closure free of `net`/`net/http`/`database/sql`) at HEAD `44cf9e9`.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop`. **No `.github/workflows/` — no CI configured.** Whoever wires CI must avoid
  `go build ./...` over the gitignored `cauldron/` (its module-less reference trees break a fresh
  build) — see learnings.
- The `notecheck` external-oracle CI job does not exist yet; correct at this stage (no end-to-end
  signature-verification code in tree). The trust-root oracle gate is `derive_vkey.py` parity, which
  `review` re-ran and confirmed.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): wire **Ed25519 signed-note checkpoint
verification** — parse a hub-signed checkpoint with a `note.Verifier`/signed-note parser, verify it
against the key `ResolveVerifierKey` returns, and map a signature matching no listed key to status
`unverified` (distinct from `unresolvable`). This needs a real sb0/sb1 checkpoint captured into
`testdata/live/` (prerequisite) and pairs with the `hub_keys` SQLite cache + CID 1.0 validity-window
enforcement (consuming the currently parsed-but-unenforced `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`).
From there the follower + per-network SQLite store + the three consistency-check triggers + coverage +
restart-survival complete M1's Verify criteria. No CI is configured — flag for whoever sets up the
GitHub workflow.
