# Issues

Lightweight backlog `define-next` can prioritize. Append entries; `review` deletes resolved ones.

**Format** — one entry per issue:

```
## <short title>
- **Priority:** critical | normal | low
- **Source:** [human] | [review] | [advance]
- **What / where / how to verify:** <the problem, its location, and the check that proves it fixed>
- **Spec:** <optional — target.md or an ADR section this is rooted in>
```

**Priority semantics:** `critical` preempts everything; `normal` is weighed against the state→target
gap; **`low` is skipped by the loop** (reserved for human-directed work). The `Source` tag records who
filed it and does **not** affect priority.

---

## `/_ds/tokens.css` uses `immutable` Cache-Control on a stable, overwrite-in-place URL
- **Priority:** normal
- **Source:** [review] (Codex [P2], reviewer-confirmed against the project's own convention)
- **What / where / how to verify:** `internal/web/web.go:42` sets `cacheControl = "public, max-age=31536000, immutable"`
  on `/_ds/tokens.css`. The dashboard links the **stable, non-content-addressed** literal path, so the
  bytes are overwritten in place on every redeploy (no content hash in the URL). `immutable` lets a
  browser/proxy pin the response for up to a year with NO revalidation, so after a redeploy that changes
  `tokens.css` a client can get the new HTML but a stale stylesheet — token/style fixes silently fail to
  take effect. This contradicts the codebase's own established discipline: `internal/tilesserve/handler.go`
  uses `cacheImmutable` ONLY for content-addressed FULL tiles (the URL changes when bytes change) and
  `cacheRevalidate = "no-cache"` for resources "overwritten in place at a stable URL" (its own comment,
  lines 40-46, warns such a resource "must NOT carry the immutable directive or a client would pin a
  soon-overwritten" version). `next.md` explicitly sanctioned `immutable` ("bytes are build-pinned"), but
  build-pinned is not content-addressed — a redeploy reuses the same URL. Cosmetic only (stale design
  tokens; no correctness/security/trust-root impact), so it does not block progress. Fix in a later
  M-UI slice: switch `/_ds/tokens.css` to the tilesserve `no-cache` + strong-ETag + `If-None-Match`→304
  pattern (the `writeBlob` shape already exists), OR serve the stylesheet at a build-fingerprinted path
  (`/_ds/tokens.<hash>.css`) and link that. Verify fixed: the response either revalidates (`no-cache`
  with a content ETag and a 304 on `If-None-Match`) or the linked path carries a content fingerprint.
- **Spec:** ADR-0005 cache discipline as embodied in `internal/tilesserve` (`cacheRevalidate` for
  overwrite-in-place resources); M-UI shared-shell criterion.

## `cmd/notecheck`'s `run` has a vestigial `out io.Writer` parameter
- **Priority:** low
- **Source:** [review]
- **What / where / how to verify:** `cmd/notecheck/main.go` `run(vkey string, in io.Reader, out
  io.Writer) (string, error)` never writes to `out` — it returns the signer name and `main` prints
  `OK %s` to `os.Stdout` itself. The param matches the literal signature `next.md` specified and is
  harmless (tests pass a throwaway buffer; `go vet` does not flag unused params), but the signature
  is misleading. Fix when `run` is next touched: drop `out`, OR have `run` print `OK %s` to `out` and
  let the test assert on it. Verify fixed: `out` is either gone or written to. Low — skipped by the loop.
- **Spec:** KISS / YAGNI (CLAUDE.md code standards); no spec contract.
