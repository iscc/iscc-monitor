<!-- assessed-at: e4152539fd67667c51959260877b220c7817ca15 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — trust-root chain + Ed25519 signed-note checkpoint verify complete (pure); no follower/store/binary yet

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`; one dep:
`golang.org/x/mod v0.33.0` for `sumdb/note`). M1's *verification primitives* are now complete and
golden-tested: the did:web trust-root chain (`internal/didweb` pure parse/derive + the networked
`internal/logclient/didresolve.go` `ResolveVerifierKey` over an injectable `Fetcher`) plus, new this
iteration, `internal/logclient/verify.go` `VerifyCheckpoint(vkey, raw)` which verifies a hub-signed
C2SP checkpoint via `sumdb/note` and maps a non-matching signature to `ErrUnverified`. What remains
of M1 is all the *stateful* work: the per-hub follower (three-trigger consistency check + freeze +
alert), the per-network SQLite store (`hub_keys` cache + validity-window enforcement), coverage,
structured logs, `/metrics`, restart survival — and there is still no binary entrypoint.
Last `review` verdict (HEAD `e415253`) is **PASS** with the gate recorded green; branch `develop` in
sync with `origin/develop`; no open issues.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; only `internal/logclient/verify.go` + its test + two
  `testdata/live/` checkpoint fixtures changed since the last assessment):
  - `internal/logclient/verify.go` (NEW since last assessment) — pure `VerifyCheckpoint(vkey, raw)
    -> (origin, treeSize, root, err)`: `note.NewVerifier` + `note.Open(VerifierList)`, the oracle's
    exact `len(Sigs)==0 || len(UnverifiedSigs)!=0` reject, then `parseCheckpointBody` (private)
    parsing the verified 3-line body — origin / leading-zero-rejecting decimal size (`"0"` ok,
    `"01"` rejected) / std-base64 32-byte root — and asserting signer-name == origin. Exported
    sentinel `var ErrUnverified` distinct from `didresolve.go`'s `ErrUnresolvable`; malformed-but-
    signed bodies stay plain parse errors (separable). Imports only `sumdb/note` + stdlib (WASM-safe).
    Tested by `TestVerifyCheckpoint` (sb0/sb1 positive, real fixtures), `TestVerifyCheckpointUnverified`
    (wrong-key / stale-rotated-key / tampered-root → `errors.Is ErrUnverified`), `TestParseCheckpointBody`
    (valid / zero-size / leading-zero / non-numeric / bad-base64 / short-root / too-few-lines).
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` (NEW) — real hub-signed C2SP
    checkpoints (sb0 size 10183, sb1 size 61). First population of `testdata/live/`. sb1's sig keyhash
    is the CURRENT rotated key (`069d0f14`), independently confirmed by `review` from the raw sig line.
  - `internal/logclient/didresolve.go` — networked did:web resolver: `Fetcher` 1-method seam,
    `NewHTTPFetcher`, `ErrUnresolvable`, `ResolveVerifierKey(ctx, Fetcher, baseURL)`. Tested by
    `TestResolveVerifierKey` (sb0/sb1), `…Unresolvable`, `…OverHTTP` (httptest TLS), `TestHTTPFetcherNotFound`.
    Carried forward unchanged.
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; `TestOrigin` + `TestOriginErrors`.
    Carried forward unchanged.
  - `internal/didweb/{url,resolve,vkey}.go` — pure `DocumentURL`, `ParseDIDDocument` (polymorphic
    `assertionMethod`; parses-but-does-not-enforce CID 1.0 `ValidFrom`/`ValidUntil`/`Revoked`),
    `VerifierKey`/`DIDKey` (stdlib-only port of `derive_vkey.py`, byte-exact for both hubs). Golden-tested
    (`TestDocumentURL[Errors]`, `TestParseDIDDocument[/Errors/InlineAssertion]`, `TestVerifierKey`,
    `TestPubkeyFromDIDErrors`). Carried forward unchanged.
  - Gate-dodge scan clean (no `//nolint` / `t.Skip` / swallowed err / build-tag exclusion in `internal`).
- Missing (the stateful majority of M1 — nothing consumes the verified key/checkpoint yet):
  - **Per-hub follower**: wire `ResolveVerifierKey → VerifyCheckpoint` into a checkpoint-acceptance
    path that maps `ErrUnresolvable → unresolvable` and `ErrUnverified → unverified`, runs the
    three-trigger RFC-6962 consistency check (fork/shrink/equivocation via `transparency-dev/merkle`),
    persists both raw checkpoints + proof, sets `frozen=1`, alerts once, polls evidence-only at a
    backed-off cadence, no auto-unfreeze, other hubs unaffected.
  - **Per-network SQLite store** (`modernc.org/sqlite`): `hub_keys` cache + now-vs-window validity
    enforcement (consuming the currently parsed-but-unenforced `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`),
    `violations`, coverage (`monitored_since`), restart survival.
  - config + realm registry (domains only), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: did:web golden fixtures under `internal/{didweb,logclient}/testdata/`; `testdata/live/`
  now holds the two checkpoints. Still **no tiles or entry bundles** in `testdata/live/` (needed for
  the consistency check + M2 aggregator). Known stale-fixture drift (filed in handoff, not yet acted
  on): `derive_vkey.py` `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's
  PRE-rotation key (`22b08f3e`); refresh to `069d0f14` lands with the `hub_keys`/validity step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (in `verify.go`). Not yet wired:
  `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`, `modernc.org/sqlite`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (now also exercised end-to-end: real checkpoints verify under
  the resolved key). The synthetic fork/shrink/equivocation → `violations.kind` + `frozen=1` +
  exactly-one-alert + other-hubs-unaffected + restart-survival half of M1 is **not started** (no
  follower or store).

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, requires `x/mod v0.33.0`); `mise run check` runnable
  (`go build ./... && go vet ./... && go test ./...`). Latest `review` handoff records the gate green
  at HEAD `e415253` (build/vet/test exit 0, `gofmt -l .` empty, `GOOS=js GOARCH=wasm go build
  ./internal/didweb` succeeds, `verify.go` WASM-safe). Trust-root oracle re-confirmed: `derive_vkey.py`
  sb0 byte-exact AND the live checkpoint fixtures actually verify under `note.Open` (the strongest proof).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop`. **No `.github/workflows/` — no CI configured.** When CI is wired it must avoid
  `go build ./...` over the gitignored `cauldron/` reference trees (module-less, break a fresh build)
  and shell out the future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): the **SQLite `hub_keys` cache + follower
checkpoint-acceptance path** — wire `ResolveVerifierKey → VerifyCheckpoint`, mapping
`ErrUnresolvable → unresolvable` and `ErrUnverified → unverified` to hub status, and enforce CID 1.0
validity windows (consuming `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`; refresh the stale sb1
fixture + `derive_vkey.py` to `069d0f14` here). The three-trigger RFC-6962 consistency check (two
checkpoints + `transparency-dev/merkle`, needs tiles in `testdata/live/`) is the step after; then
freeze/alert + per-network store + coverage + restart-survival complete M1's Verify criteria. No CI
is configured — flag for whoever sets up the GitHub workflow.
