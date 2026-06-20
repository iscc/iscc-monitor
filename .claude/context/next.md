# Next Work Package

## Step: Networked checkpoint fetch over the Fetcher seam (`FetchCheckpoint`)

## Goal
Add the one missing networked-fetch primitive the follower needs before it can poll a hub:
`FetchCheckpoint(ctx, fetcher, baseURL)` returns the raw signed-checkpoint bytes from the canonical
`https://<domain>/log/checkpoint` URL via the existing 1-method `Fetcher`. It produces exactly the
`raw []byte` that `AcceptCheckpoint` already takes — wiring the fetch closes the last gap between
"verify bytes I'm handed" and "go get the bytes", without yet touching persistence or the poll loop.

## Goal-fit (state → target gap)
`state.md` lists the follower poll loop as the immediate M1 unit, but the handoff `**Next:**` bundles
fetch + verdict→`CheckpointRecord` mapping + `RecordCheckpoint`/`AdvanceFollowState` persistence +
the `hub_keys` did:web cache + the sb1 fixture refresh + the single-writer goroutine wrapper into one
item — far more than 3 files and many distinct behaviors. The pure verification chain
(`ResolveVerifierKey` → `VerifyCheckpoint` → `DIDKey.ValidAt` → `AcceptCheckpoint`) and the store CRUD
both exist, but **nothing yet fetches a checkpoint over the wire** — `AcceptCheckpoint` is handed
`raw` by its tests. This step takes the smallest coherent prerequisite slice: the transport-only
checkpoint fetch, parallel to the existing `ResolveVerifierKey`, reusing the same `Fetcher` seam. It
is fully testable offline and end-to-end against the captured `sb0` fixture. The verdict→store mapping
and the poll loop are the very next step and consume this.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/logclient/checkpoint.go` — the `FetchCheckpoint` function +
    file/function docstrings.
  - `/workspace/iscc-monitor/internal/logclient/checkpoint_test.go` — offline fake-`Fetcher` tests +
    one `httptest` round-trip (test file, not counted toward the 3-file budget).
- **Modify**: (none — purely additive; no existing source file changes)
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/origin.go` — reuse `origin(baseURL)` to derive
    `<domain>/log`; do NOT add a second host/path parser.
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` — the `Fetcher` interface,
    `httpFetcher`, `NewHTTPFetcher`, and the `os.ErrNotExist`-on-404 contract this fetch relies on
    (already implemented; reuse as-is — do not re-implement an HTTP client).
  - `/workspace/iscc-monitor/internal/logclient/didresolve_test.go` — the
    `fakeFetcher{data,err,gotURL}` pattern and the `httptest.NewTLSServer` + `srv.Client()` real-HTTP
    pattern to copy.
  - `/workspace/iscc-monitor/internal/logclient/verify_test.go` — the `readCheckpoint(t, name)` helper
    (loads `../../testdata/live/<name>`); reuse it, do NOT re-declare it (same `logclient` package).
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — `AcceptCheckpoint` signature and
    `CheckpointInfo.TreeSize`, for the end-to-end round-trip assertion.
  - `/workspace/iscc-monitor/cauldron/tessera/client/fetcher.go` (`ReadCheckpoint` →
    `layout.CheckpointPath`, returns raw bytes; parsing is the caller's job) and
    `/workspace/iscc-monitor/cauldron/iscc-hub/specs/iscc-log.md` §9 (`GET /log/checkpoint`) — confirm
    the canonical path is the `checkpoint` resource under the `/log` root, i.e.
    `https://<domain>/log/checkpoint`.

## Not In Scope
- The follower poll loop itself: mapping `CheckpointInfo`/`Status` into a `CheckpointRecord`, calling
  `RecordCheckpoint` / `AdvanceFollowState`, the `err`-before-status handling, backoff, or any
  `internal/store` interaction. That is the next step and consumes `FetchCheckpoint`.
- The `hub_keys` did:web cache write and the stale-`sb1` `did.json` / `derive_vkey.py` `HUBS` fixture
  refresh (`22b08f3e` → `069d0f14`). Defer to the follower-persistence step that populates `hub_keys`.
- The three-trigger RFC-6962 consistency check, freeze/alert, coverage (`monitored_since`), structured
  logs, `/metrics`, config/realm registry, and any `cmd/` binary.
- Tile / entry-bundle fetching (`ReadTile`/`ReadEntryBundle` analogues) — that is M2.
- Any change to `origin.go`, `didresolve.go`, `verify.go`, or `accept.go` — this step is additive only.

## Implementation Notes
- Signature: `func FetchCheckpoint(ctx context.Context, fetcher Fetcher, baseURL string) ([]byte, error)`.
  Keep it a free function in package `logclient`, parallel to `ResolveVerifierKey`.
- Derive the URL by reuse: `name, err := origin(baseURL)` (yields scheme-less `<domain>/log`), then the
  checkpoint URL is `"https://" + name + "/checkpoint"` (e.g. `https://sb0.iscc.id/log/checkpoint`).
  tlog/did:web are always HTTPS, and `origin()` already strips any caller scheme — build the URL by
  concatenation/`fmt.Sprintf`, NOT a second `net/url` parse, and never from the bare domain.
- On `origin()` error, wrap and return (`fmt.Errorf("fetch checkpoint: %w", err)`). Do NOT invent a new
  sentinel and do NOT wrap in `ErrUnresolvable` — that sentinel is owned by did:web resolution; a
  checkpoint-fetch failure is a plain transport error the follower will classify separately. Wrap the
  `Fetcher`'s error with `%w` so a 404's `errors.Is(err, os.ErrNotExist)` still holds through the
  wrapper (the follower needs to tell "no checkpoint served" from other transport faults).
- Return the raw bytes verbatim — no trimming, no parsing, no `note.Open`. `VerifyCheckpoint` /
  `AcceptCheckpoint` own the signed-note framing; this function is transport only (mirror tessera's
  `ReadCheckpoint`, which returns raw bytes and leaves `ParseCheckpoint` to the caller).
- Imports stay minimal: `context`, `fmt`, and the package-private `origin`/`Fetcher` only. Do NOT add
  `net/http` (the concrete `httpFetcher` is injected by the caller), `database/sql`, or `os` (the
  `os.ErrNotExist` contract is the injected Fetcher's, surfaced through the returned error).
- Relevant Correctness rule (learnings.md, seeded): **Origin = `<domain>/log`, never the bare domain**
  — the checkpoint resource hangs off `/log`, so reuse the single `origin()` helper. Constructing the
  URL from the bare domain (dropping `/log`) is the exact highest-probability bug this rule guards
  against; the golden URL assertion below catches it.
- Match file conventions: leading file-purpose docstring, evergreen per-function docstring, no
  `t.Skip` / `//nolint` / swallowed errors.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` exit 0; `gofmt -l .`
  empty).
- `go test -run TestFetchCheckpoint ./internal/logclient` passes.
- Offline URL golden (fake `Fetcher`): `FetchCheckpoint(ctx, &fakeFetcher{data: want},
  "https://sb0.iscc.id")` records `gotURL == "https://sb0.iscc.id/log/checkpoint"` and returns the
  canned bytes unchanged (`bytes.Equal(got, want)`).
- Bare-host input still hits `/log/checkpoint`: `FetchCheckpoint(ctx, fake, "sb0.iscc.id")` (no scheme)
  also yields `gotURL == "https://sb0.iscc.id/log/checkpoint"`.
- 404 contract: a `fakeFetcher{err: os.ErrNotExist}` makes `FetchCheckpoint` return a non-nil error
  with `errors.Is(err, os.ErrNotExist) == true`.
- Empty base URL: `FetchCheckpoint(ctx, fake, "")` returns a non-nil error (propagated from `origin()`).
- End-to-end over real HTTP (`httptest.NewTLSServer` serving `readCheckpoint(t,
  "sb0.iscc.id_checkpoint")` at `/log/checkpoint` and the sb0 did.json at `/.well-known/did.json`):
  bytes fetched by `FetchCheckpoint` fed straight into `AcceptCheckpoint(ctx, NewHTTPFetcher(srv.Client()),
  srv.URL, raw, <fixed observedAt>)` yield `StatusVerified` with `info.TreeSize == 10183` (matching the
  captured sb0 fixture), proving fetch → verify composes.

## Done When
`FetchCheckpoint` exists in `internal/logclient`, derives the canonical `https://<domain>/log/checkpoint`
URL via the shared `origin()` helper, returns raw bytes verbatim (propagating `os.ErrNotExist` on 404),
adds no `net/http`/`os`/`database/sql` import to the function, and every Verification criterion passes
with `mise run check` green.
