## 2026-06-21 — Conditional GET (strong ETag + If-None-Match → 304) on the tlog-tiles mirror

**Done:** Added a strong content ETag (quoted hex of `sha256.Sum256(data)`, no `W/` prefix) to every
tlog-tiles 200 inside `writeBlob`, plus an `If-None-Match` short-circuit: a request whose value is `*`
or that exact tag returns `304 Not Modified` with an empty body and the validating headers echoed.
`writeBlob` now takes `r *http.Request`; all three callers (`serveCheckpoint`/`serveTile`/`serveEntries`)
updated. Docstrings flipped from "conditional GET … out of scope" to describing the now-present behavior.

**Files changed:**
- `internal/tilesserve/handler.go`: imports `crypto/sha256` + `fmt`; `writeBlob` signature now
  `writeBlob(w, r, data, immutable)` — sets `Content-Type`, `Cache-Control`, then `ETag` BEFORE the
  `If-None-Match` branch (so a 304 still carries `ETag`/`Cache-Control` per RFC 7232 §4.1), then writes
  the body on the 200 path; package doc + `Handler` doc updated.
- `internal/tilesserve/handler_test.go`: added `getWithIfNoneMatch` helper and `TestHandlerConditionalGet`
  (5 subtests: strong-quoted-ETag on a 200; exact-tag → 304/empty-body/echoed-ETag; `*` wildcard → 304;
  non-matching tag → 200 with full body; two distinct resources → distinct ETags). Added `strings` import.

**Verification:** `mise run check` → green (build + vet + test; all 15 packages `ok`). Per criterion:
- [x] `mise run check` green.
- [x] `gofmt -l internal/tilesserve/handler.go` (and the test) print nothing (ran `mise run fmt` first).
- [x] `go test -count=1 -run TestHandler ./internal/tilesserve` — all 9 prior `TestHandlerServesSeededBytes`
  subtests + all 5 new `TestHandlerConditionalGet` subtests PASS (verbose, uncached). Cache-Control and
  byte-equality subtests did not regress.
- [x] New: full-tile 200 ETag is non-empty, quoted, and does NOT start with `W/` (strong).
- [x] New (load-bearing): re-GET with `If-None-Match: <exact ETag>` → 304, empty body, same ETag echoed.
- [x] New: `If-None-Match: "deadbeef"` (non-matching) → 200 with the full seeded body.
- [x] New: full tile vs entry bundle (distinct seeded bytes) → different ETags (content-derived, not constant).
- [x] Also added `*` wildcard → 304 subtest (spec'd in Implementation Notes; covers the always-match case).
- [x] `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency/schema change).
- [x] ETag set only in `writeBlob`; the 400/404/405/500 `http.Error` paths never call `writeBlob` and are
  byte-untouched (grep-confirmed).

**Next:** Cache-Control / conditional-GET is now complete on the `tilesserve` mirror. The companion gap is
the size-dependent **proof** surfaces in `proofserve` (`/inclusion` / `/consistency` / `/entries`) — these
have no cache policy yet, are tied to `LastSize`, and were explicitly left out of this slice; they'd need a
derived/weak validator. Alternatively, begin draining a `normal` issue — the ADR-0006 split-view /
frozen-advance cluster (`TestPollHubFork` re-detection via `PollHub`, frozen-hub evidence-only
short-circuit, `CheckConsistency` collapse) is the highest-value next cluster — or start the proof-bundle
JSON + verify-for-me arc.

**Notes:**
- Oracle/conformance gate correctly **N/A**: this is pure HTTP header wiring on opaque BLOBs — no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path touched. `derive_vkey.py` / `notecheck` /
  `fsck` golden vectors untouched and unaffected.
- No new dependency: `crypto/sha256` + `fmt` are stdlib and WASM-safe; `go.mod` / `go.sum` /
  `schema.sql` byte-identical. The package stays a clean `tilesserve → store` leaf (no new `net`/`sqlite`
  edge); WASM purity rides on `internal/didweb` (untouched).
- No header collision with CORS: `corsmw.Handler` sets only `Access-Control-*` (re-read from source);
  `ETag` is set alongside `Content-Type`/`Cache-Control` before the first write, all before the status is
  frozen.
- Single-token match (exact tag or `*`) is intentional and sufficient — a client echoes back the exact
  ETag the server sent; no brittle comma-separated `If-None-Match` list parser, as `next.md` directed.
- The `_, _ = w.Write(data)` on the 200 path is the pre-existing documented post-status-write idiom
  (matches `metricshttp`/`healthz`), not introduced here and not a gate dodge. No `//nolint`/`t.Skip`/
  swallowed-error/build-tag added.
- Scope clean: only `internal/tilesserve/handler.go` (1 production file) + its `_test.go` changed.
  Nothing from `## Not In Scope` touched (no `proofserve`/`corsmw`/`Last-Modified`/`http.ServeContent`/
  Range handling/issue-draining).
