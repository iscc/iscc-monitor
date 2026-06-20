# Next Work Package

## Step: did:web HTTP resolver at the outbound-fetch seam (base URL → verifier key, `unresolvable` on failure)

## Goal
Turn the pure did:web chain (`DocumentURL` → `ParseDIDDocument` → `VerifierKey`) into a working
resolver that, given a hub base URL, fetches the hub's `did.json` over an injected `Fetcher` and
returns the hub's signed-note verifier key. This is the first networked unit of M1 and the gate that
lets the follower (next steps) obtain the key it verifies checkpoints against; failures collapse to a
single `unresolvable` sentinel the follower maps to hub status.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` — the did:web resolver: a small
    `Fetcher` interface (`Fetch(ctx, url) ([]byte, error)`), an `httpFetcher` adapter over
    `*http.Client`, an exported `ErrUnresolvable` sentinel, and `ResolveVerifierKey(ctx, fetcher,
    baseURL) (key string, didKey didweb.DIDKey, err error)` that wires
    `origin` + `did:web:<host>` + `didweb.DocumentURL` → `fetcher.Fetch` → `didweb.ParseDIDDocument`
    → `didweb.VerifierKey`. (1 non-test source file.)
  - `/workspace/iscc-monitor/internal/logclient/didresolve_test.go` — table-driven tests driving the
    resolver against the sb0/sb1 fixtures through an in-test fake `Fetcher` and through
    `net/http/httptest` (test file, not counted toward the 3-file limit).
- **Modify** (≤3 non-test/doc files): none — this step is purely additive.
- **Reference** (read, do not import):
  - `/workspace/iscc-monitor/cauldron/tessera/client/fetcher.go` — the `HTTPFetcher.fetch` shape to
    mirror: `http.NewRequestWithContext`, map `404` → `os.ErrNotExist`, non-200 → error, `io.ReadAll`,
    close body. Port the *shape*, do not import it.
  - `/workspace/iscc-monitor/internal/didweb/url.go` (`DocumentURL`),
    `/workspace/iscc-monitor/internal/didweb/resolve.go` (`ParseDIDDocument`, `DIDKey`),
    `/workspace/iscc-monitor/internal/didweb/vkey.go` (`VerifierKey`) — the three exported entrypoints
    to call in sequence.
  - `/workspace/iscc-monitor/internal/logclient/origin.go` — `origin(baseURL)` is the signed-note
    name passed to `VerifierKey`; reuse it (do not re-derive).
  - `/workspace/iscc-monitor/internal/didweb/testdata/sb0.iscc.id_did.json` and
    `/workspace/iscc-monitor/internal/didweb/testdata/sb1.amlet.id_did.json` — copy/symlink into a new
    `internal/logclient/testdata/` (test fixtures, not source) so the resolver test stays offline.
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` §Architecture step 2 (did:web key
    resolution; `unresolvable` on failure, keep mirroring) and ADR-0009.

## Not In Scope
- **No signature verification and no `unverified` status.** Verifying an Ed25519 signed-note against
  the resolved key is the follower's job and needs an actual checkpoint — it is a *later* step. This
  step stops at producing the verifier key + `DIDKey`.
- No `hub_keys` SQLite cache, no validity-window (`ValidFrom`/`ValidUntil`/`Revoked`) enforcement, no
  re-resolution cadence — those belong to the store/follower step.
- No follower loop, no per-network SQLite, no `cmd/` entrypoint, no `/metrics`, no realm registry.
- Do not move or alter `internal/didweb` (it must stay WASM-pure); do not change the derived
  verifier-key bytes or the fixtures' contents.
- Do not add `transparency-dev/*` or any external module dependency — stdlib `net/http` only.

## Implementation Notes
- **Placement (load-bearing): the resolver lives in `internal/logclient`, NOT `internal/didweb`.**
  This file imports `net/http`/`net/url`, which would break the WASM build of `internal/didweb`
  (learnings: `proof/verify` and `didweb` must stay import-clean; verify with
  `GOOS=js GOARCH=wasm go build ./internal/didweb`). `logclient` is the non-WASM follower package per
  the plan's layout (`internal/logclient/{origin.go,follower.go,verify.go}`).
- **The did:web identifier comes from the host, not the base path.** Derive it as
  `"did:web:" + <host>` where `<host>` is the host[:port] of `baseURL` (the same host `origin` uses,
  minus the `/log` suffix). For the live hubs `https://sb0.iscc.id` → `did:web:sb0.iscc.id` →
  `DocumentURL` → `https://sb0.iscc.id/.well-known/did.json`. Reuse `net/url` parsing already proven
  in `origin.go` (default the scheme in for a bare host); keep one host-extraction helper, do not
  duplicate `origin`'s logic by hand.
- **`Fetcher` is a 1-method interface for the seam (PRD "Testing Decisions": test at the
  outbound-fetch seam against fixtures).** `Fetch(ctx context.Context, url string) ([]byte, error)`.
  Provide `httpFetcher{c *http.Client}` (nil client → `http.DefaultClient`) mirroring
  `cauldron/tessera/client/fetcher.go`'s `fetch`: `http.NewRequestWithContext(ctx, GET, url, nil)`,
  on `404` return `os.ErrNotExist` (wrapped), non-200 → error, `defer Body.Close()`, `io.ReadAll`.
  Tests inject a fake `Fetcher` returning fixture bytes (and `httptest.NewServer` for one real-HTTP
  path) — never hit the live network.
- **Error mapping (ADR-0009 / learnings "did:web is the only key source"):** any failure to fetch OR
  parse OR derive (bad URL, fetch error, non-200, invalid JSON, missing assertionMethod, bad key)
  wraps `ErrUnresolvable` via `fmt.Errorf("...: %w", ErrUnresolvable)` so the follower maps it to
  status `unresolvable` and keeps mirroring — without inspecting resolver internals. Keep
  `ErrUnresolvable` exported; keep the `Fetcher` interface and `ResolveVerifierKey` exported; keep the
  `httpFetcher` adapter unexported with an exported constructor if one is needed (`NewHTTPFetcher`),
  YAGNI otherwise.
- **Reuse, don't re-derive.** Call `origin(baseURL)` for the signed-note name and `didweb.VerifierKey`
  for the key string — the byte-exact trust-root oracle path stays single-sourced (learnings: verifier
  key is the trust-root gate). Do not re-implement key derivation in `logclient`.
- **Context-first signature:** `ResolveVerifierKey(ctx, ...)` and `Fetch(ctx, ...)` take a
  `context.Context` first arg (cancellation/timeout for the follower).
- Short, pure-ish functions with evergreen docstrings; file starts with a one-line purpose docstring.
- Do NOT use `t.Skip`, `//nolint`, build tags, or swallow errors to pass the gate (target quality
  bar; learnings "Never weaken a gate").

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0).
- `gofmt -l /workspace/iscc-monitor` prints nothing.
- `go test -run TestResolveVerifierKey ./internal/logclient` passes.
- Against the sb0 fixture, `ResolveVerifierKey` returns the golden verifier key
  `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` for base URL
  `https://sb0.iscc.id` (assert the full string in the test).
- Against the sb1 fixture, it returns `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`
  for `https://sb1.amlet.id` (assert the full string).
- A `Fetcher` that returns an error, a 404 (`os.ErrNotExist`), or malformed JSON each makes
  `ResolveVerifierKey` return an error satisfying `errors.Is(err, ErrUnresolvable)` (assert with
  `errors.Is`).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` still succeeds — `internal/didweb` is untouched and
  the new `net/http` import lives only in `internal/logclient`.

## Done When
`ResolveVerifierKey` fetches each live-hub fixture through an injected `Fetcher` and returns the exact
golden verifier-key string, every fetch/parse/derive failure surfaces as `errors.Is(err,
ErrUnresolvable)`, `internal/didweb` still builds for WASM, and `mise run check` is green.
