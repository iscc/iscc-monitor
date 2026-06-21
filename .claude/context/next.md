# Next Work Package

## Step: Conditional GET (strong ETag + If-None-Match → 304) on the tlog-tiles mirror

## Goal
Add HTTP conditional-GET support to the raw tlog-tiles mirror handler so caches and verifiers can
revalidate with a 304 instead of re-downloading a BLOB. This is the deferred follow-up the last
`review` handoff named (`**Next:**`) and it completes the cross-cutting cache plumbing on the mirror
(Cache-Control landed last slice; conditional GET is its companion) before the M3 dashboard /
verify-for-me arc.

## Scope
- **Modify**: `internal/tilesserve/handler.go` (the only production file; 1 of ≤3)
- **Modify (test)**: `internal/tilesserve/handler_test.go`
- **Reference**:
  - `/workspace/iscc-monitor/internal/tilesserve/handler.go` — current `writeBlob` / `serve*`
    structure; the `immutable bool` discriminant is already threaded through every served BLOB and the
    docstring currently says conditional GET is out of scope (flip that).
  - `/workspace/iscc-monitor/internal/corsmw/corsmw.go` — confirms CORS sets only `Access-Control-*`
    headers, so adding an `ETag` here cannot collide with the single CORS wrap.
  - `/workspace/iscc-monitor/internal/tilesserve/handler_test.go` — reuse `newServer` (seeds one full
    tile / partial tile / bundle / checkpoint over `httptest`) and the `getWithHeader` helper that
    already returns the response header.

## Not In Scope
- Cache-Control or conditional GET on the size-dependent **proof** surfaces
  (`/inclusion` / `/consistency` / `/entries` in `proofserve`) — those are tied to `LastSize` and are a
  separate later slice. Touch only `tilesserve`.
- `Last-Modified` / `If-Modified-Since` time-based validators. A strong content `ETag` fully covers the
  mirror's needs (BLOBs are content-addressed or overwritten-in-place); a second validator is YAGNI here.
- HTTP range requests / `Accept-Ranges`. Do **not** switch to `http.ServeContent` — it would re-sniff
  Content-Type and add Range handling, fighting the explicit `Content-Type` / `Cache-Control` already set.
- Draining any `normal` issue (frozen-advance, `TestPollHubFork`, `AcceptCheckpoint` context reuse,
  tile-writer `p` vocab, `AdvanceAccepted`, `CheckConsistency` collapse) — keep this slice scope-clean to
  finish the coherent M3 HTTP-plumbing arc first.

## Implementation Notes
- **Compute a strong ETag from the BLOB bytes already in memory** inside `writeBlob`:
  `etag := fmt.Sprintf("\"%x\"", sha256.Sum256(data))` — quoted hex per RFC 7232; a *strong* validator
  has **no** `W/` prefix. Set it with `w.Header().Set("ETag", etag)` alongside the existing
  `Content-Type` / `Cache-Control` sets, BEFORE the conditional check and BEFORE the first `w.Write`
  (the 200/304 status freezes the header map on first write).
- **`If-None-Match` short-circuit**: read `inm := r.Header.Get("If-None-Match")`. If `inm == "*"` (the
  wildcard, which always matches an existing resource) OR `inm == etag`, call
  `w.WriteHeader(http.StatusNotModified)` and `return` WITHOUT writing the body. Keep the match to
  exact-token-or-`*`; do not hand-roll a full comma-separated list parser (a client echoes back the exact
  ETag the server sent — single-token match is sufficient and avoids a brittle parser).
- **`writeBlob` needs the request** to read `If-None-Match`. Change its signature to
  `writeBlob(w http.ResponseWriter, r *http.Request, data []byte, immutable bool)` and update the three
  callers (`serveCheckpoint`, `serveTile`, `serveEntries`) — each already has `r` in scope.
- **Header order** (RFC 7232 §4.1: a 304 still carries the validating headers): set `Content-Type`,
  `Cache-Control`, and `ETag` first; then the `If-None-Match` branch; then `w.Write(data)` on the 200
  path. Setting the headers before the branch makes the 304 carry the right `ETag` / `Cache-Control` for
  free. The `http.Error` 400/404/405/500 paths are untouched (no ETag needed).
- **Leaf / WASM posture**: add `crypto/sha256` and `fmt` to the import block — both stdlib, WASM-safe, no
  `net`/`sqlite` edge beyond what is already imported; the package stays a clean tilesserve→store leaf.
- **Docstring sync**: the package doc and the `Handler` doc currently state "conditional GET (ETag /
  If-None-Match) is intentionally out of scope for this slice." Flip that to describe the now-present
  behavior: a strong content ETag on every 200 and an `If-None-Match` (exact tag or `*`) → 304. This is a
  doc-in-code edit in the same file, so no separate doc file changes.
- **Correctness rule (learnings.md):** this is **pure HTTP header wiring on opaque BLOBs** — the
  oracle/conformance gate (`notecheck` / `derive_vkey.py` / `fsck`) is **N/A** (no signature / RFC-6962 /
  Merkle / did:web / fsck / proof path). Do not weaken any gate (`//nolint` / `t.Skip` / swallowed
  errors). The two cache vocabularies (path-API full = `width 0` vs store-column full = `256`) do not
  re-enter here — this slice only consumes the already-correct `immutable bool` and the raw BLOB bytes.
- **No new dependency**: `crypto/sha256` + `fmt` are stdlib, so `go.mod` / `go.sum` / `schema.sql` must
  stay byte-unchanged.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l internal/tilesserve/handler.go` prints nothing.
- `go test -count=1 -run TestHandler ./internal/tilesserve` passes — the existing byte-equality and
  Cache-Control subtests stay green (ETag/304 must not regress the 200 body or the Cache-Control header).
- New subtest: a GET to the full-tile path returns a non-empty, quoted, **strong** `ETag` on its 200
  (assert the header is non-empty and does NOT start with `W/`).
- New subtest (load-bearing): re-issuing the same GET with `If-None-Match: <that exact ETag>` returns
  **304 Not Modified** with an **empty body** and the same `ETag` echoed.
- New subtest: a GET with `If-None-Match: "deadbeef"` (a non-matching tag) returns **200** with the full
  seeded body bytes (a mismatch must NOT suppress the body).
- New subtest: two distinct resources (full tile vs entry bundle, distinct seeded bytes) produce
  **different** ETags — proving the tag is content-derived, not a constant.
- `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency/schema change).

## Done When
`advance` has added a strong content ETag to every tlog-tiles 200 and an `If-None-Match` (exact tag or
`*`) → 304 short-circuit in `internal/tilesserve/handler.go`, the docstring reflects it, and all new
subtests above plus every prior `TestHandler` subtest are green under `mise run check`, with scope limited
to that one production file (+ its test) and `go.mod` / `go.sum` / `schema.sql` byte-unchanged.
