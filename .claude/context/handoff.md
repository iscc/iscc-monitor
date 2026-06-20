# Handoff

## 2026-06-20 — did:web HTTP resolver at the outbound-fetch seam (base URL → verifier key, `unresolvable` on failure)

**Done:** Added `internal/logclient/didresolve.go`: a 1-method `Fetcher` seam, an `httpFetcher`
adapter (constructed via `NewHTTPFetcher`) mirroring Tessera's fetch shape, an exported
`ErrUnresolvable` sentinel, and `ResolveVerifierKey(ctx, fetcher, baseURL)` that wires
`origin` → `did:web:<host>` → `didweb.DocumentURL` → `fetcher.Fetch` → `didweb.ParseDIDDocument` →
`didweb.VerifierKey`, collapsing every fetch/parse/derive failure to `ErrUnresolvable`. Purely
additive; `internal/didweb` is untouched and still builds for WASM.

**Files changed:**
- `internal/logclient/didresolve.go` (new): the networked did:web resolver + `Fetcher` seam.
- `internal/logclient/didresolve_test.go` (new): table-driven golden + error tests via a fake
  `Fetcher`, plus a real-HTTP path over `httptest.NewTLSServer`.
- `internal/logclient/testdata/{sb0.iscc.id_did.json,sb1.amlet.id_did.json}` (new): copies of the
  didweb fixtures so the resolver test stays offline.

**Verification:** `mise run check` → green (build + vet + test, exit 0). `gofmt -l .` prints nothing.
`GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds. `go list -deps ./internal/didweb` shows no
`net/http`/`database/sql`. Per criterion:
- [x] sb0 fixture → `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` (full
  string asserted) and fetched URL `https://sb0.iscc.id/.well-known/did.json`.
- [x] sb1 fixture → `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`.
- [x] fetch error / `os.ErrNotExist` (404) / malformed JSON / no-assertionMethod / empty base URL each
  satisfy `errors.Is(err, ErrUnresolvable)`.
- [x] Real-HTTP path over `httptest.NewTLSServer` resolves through `httpFetcher`; a 404 server yields
  `os.ErrNotExist` from `Fetch` and `ErrUnresolvable` from `ResolveVerifierKey`.

**Next:** Wire signature verification — verify a hub-signed checkpoint (Ed25519 signed-note) against
the verifier key this resolver returns, mapping a signature that matches no listed key to status
`unverified` (vs `unresolvable` here). That step needs an actual checkpoint and a `note.Verifier`/
signed-note parser, and pairs with the `hub_keys` SQLite cache + CID 1.0 validity-window enforcement
(consuming `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`, currently parsed but unenforced).

**Notes:**
- **did:web port encoding (design decision, worth a look):** for a host with a port, the resolver
  percent-encodes the colon (`strings.Replace(host, ":", "%3A", 1)`) before forming
  `did:web:<host%3Aport>`, per W3C did:web §3.2, so `DocumentURL` round-trips it back to `host:port`
  rather than splitting the port off as a path segment. Live hubs (sb0/sb1) have no port, so this only
  affects local/test hosts — but it is required to make the `httptest` server (random port) resolve
  correctly, and matches the existing `did:web:example.com%3A3000` golden in `internal/didweb/url_test.go`.
- **did:web is HTTPS-only:** `DocumentURL` always emits `https://`, so the real-HTTP test uses
  `httptest.NewTLSServer` + `srv.Client()` (trusts the test cert). A plain-HTTP hub cannot be resolved
  by design — correct per spec.
- **Deferred body close discards its error** (`defer func() { _ = resp.Body.Close() }()`). This is the
  idiomatic Go pattern, not a gate dodge; Tessera logs it via klog, which I deliberately did not pull
  in (stdlib `net/http` only, per scope). No swallowed error affects the returned result.
- **Validity-window fields still parsed-but-unenforced** (carried over from prior handoff): `DIDKey`'s
  `ValidFrom`/`ValidUntil`/`Revoked` are returned but not checked here — enforcement belongs to the
  store/follower step (explicitly out of scope).
- Branch `develop`; committing implementation + tests + this handoff only.
