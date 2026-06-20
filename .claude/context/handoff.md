# Handoff

## 2026-06-20 — Networked checkpoint fetch over the Fetcher seam (`FetchCheckpoint`)

**Done:** Added the transport-only `FetchCheckpoint(ctx, fetcher, baseURL) ([]byte, error)` to
`internal/logclient`, parallel to `ResolveVerifierKey`: it derives the canonical
`https://<domain>/log/checkpoint` URL by reusing the shared `origin()` helper (never the bare domain)
and returns the hub's signed-checkpoint body verbatim through the injected `Fetcher`, propagating a
404's `os.ErrNotExist` through the `%w` wrapper. Purely additive — no existing source file changed.

**Files changed:**
- `internal/logclient/checkpoint.go` (new): `FetchCheckpoint` free function + file/function docstrings.
  Imports only `context` + `fmt`; URL built by concatenation (`"https://" + name + "/checkpoint"`), no
  second `net/url` parse, no `net/http`/`os`/`database/sql`.
- `internal/logclient/checkpoint_test.go` (new): fake-`Fetcher` URL-golden + pass-through tests, 404
  contract, empty-URL error, and one real-HTTP `httptest` round-trip composing fetch → verify.

**Verification:** `mise run check` → green (`go build ./...`, `go vet ./...`, `go test ./...` all exit
0; `gofmt -l .` empty). `go test -run TestFetchCheckpoint ./internal/logclient` → PASS.
- [x] Offline URL golden: `FetchCheckpoint(ctx, fake, "https://sb0.iscc.id")` records
  `gotURL == "https://sb0.iscc.id/log/checkpoint"` and returns canned bytes unchanged (`bytes.Equal`).
- [x] Bare-host input `"sb0.iscc.id"` (no scheme) also yields `gotURL == "https://sb0.iscc.id/log/checkpoint"`.
- [x] 404 contract: `fakeFetcher{err: os.ErrNotExist}` → non-nil err with `errors.Is(err, os.ErrNotExist)`.
- [x] Empty base URL: `FetchCheckpoint(ctx, fake, "")` → non-nil err (propagated from `origin()`).
- [x] End-to-end over real HTTP: fetch sb0 checkpoint at `/log/checkpoint` via `httptest.NewTLSServer`
  + `NewHTTPFetcher(srv.Client())`, fed into `AcceptCheckpoint` → `StatusVerified`, `info.TreeSize == 10183`.

**Conformance/oracle gate:** N/A this step. The diff adds a transport-only fetch primitive; it touches
no signature verification, RFC-6962/Merkle, proof code (`internal/proof` still does not exist, pre-M2),
or split-view logic. The existing pure verify chain (`ResolveVerifierKey`/`VerifyCheckpoint`/`ValidAt`/
`AcceptCheckpoint`) is reused **unchanged** — the `derive_vkey.py` parity + `notecheck` oracle gates
have nothing to regress here and re-apply at the next step (which refreshes the sb1 fixture).

**Next:** The follower poll loop — the real caller. It calls `FetchCheckpoint` then `AcceptCheckpoint`
(**check `err` before the status**; a verified-but-garbled body returns non-nil err alongside
`StatusUnverified`'s zero), maps `Status.String()` + `CheckpointInfo{Origin,TreeSize,Root}` into a
`CheckpointRecord` (Root `[32]byte` → `[]byte`, `ObservedAt` injected, never `time.Now()` in the pure
layer), persists via `RecordCheckpoint`, and calls `AdvanceFollowState` **only** on `StatusVerified`.
It also writes the `hub_keys` did:web cache and refreshes the stale sb1 `did.json` fixture +
`derive_vkey.py` `HUBS` to the current key `069d0f14`. The single-writer goroutine wrapper is the
follower's concern. That step touches the trust root indirectly and refreshes a golden fixture, so the
`derive_vkey.py` parity + `notecheck` oracle gates re-apply there.

**Notes:**
- **Deviation from the literal end-to-end criterion (Verification line 98-102).** `next.md` asked the
  `httptest` round-trip to call `AcceptCheckpoint(ctx, NewHTTPFetcher(srv.Client()), srv.URL, raw, ...)`
  and get `StatusVerified`. That is **not achievable**: `AcceptCheckpoint` re-derives the verifier-key
  origin from its `baseURL`, and `VerifierKey`/keyhash are origin-dependent (`SHA-256(name||0x0A||...)`).
  A live `httptest` host is `127.0.0.1:<random-port>`, so passing `srv.URL` derives origin
  `127.0.0.1:<port>/log` — which cannot match the fixture's `sb0.iscc.id/log` signature → `ErrUnverified`,
  never `StatusVerified`. The existing `TestAcceptCheckpoint` already side-steps this by passing
  `baseURL = "https://sb0.iscc.id"` with a **fake** Fetcher for the did.json (so origin is correct AND
  no live net). I kept the spirit faithfully: the **fetch** runs over real HTTP via
  `FetchCheckpoint(srv.URL)` (proving the transport + URL derivation), then those exact fetched bytes are
  verified through `AcceptCheckpoint(didFetcher, "https://sb0.iscc.id", raw, fixedObservedAt)` →
  `StatusVerified`, `TreeSize == 10183`. This proves fetch → verify composes end-to-end while respecting
  the verify chain's origin binding. Not a code-behavior change, only a test-wiring choice; flagging for
  `review` since it diverges from the criterion's exact call shape.
- `observedAt` in the round-trip is fixed (`2026-06-20`) inside sb0's CID 1.0 validity window so the
  in-window key yields `StatusVerified` rather than `StatusRotated` — deterministic, no `time.Now()`.
- Reused the package-shared helpers `fakeFetcher` (didresolve_test.go), `readCheckpoint`
  (verify_test.go, module-root `testdata/live/`), and `readFixture` (didresolve_test.go, package
  `testdata/`); none re-declared.
