# Next Work Package

## Step: HTML log browser at `GET /<domain>/log/` (mirrored checkpoint size + root)

## Advances
M3 — Trust API + dashboard, the **fourth and final** unmet M3 Verify criterion (target.md §M3):

> `GET /<domain>/log/` (log browser) returns `200 text/html` exposing the mirrored checkpoint
> `(size, root)` with links into `entries`/proofs.

Closing it takes M3 from 3/4 → 4/4 Verify. This is the same server-rendered HTML arc the `GET /`
dashboard just landed (review PASS at `c940248`); the `review` handoff `**Next:**` names exactly this
step and notes it must mount under the per-hub subtree (reached via `hubHandler`), NOT in
`internal/dashboard`. No `critical`/`normal` issue is open (the one open issue is `low`, loop-skipped),
so milestone work proceeds.

## Goal
Serve a minimal server-rendered HTML page at each hub's `GET /<domain>/log/` root showing the
monitor's accepted checkpoint `(size, root)` for that hub plus relative links into its `entries` and
proof routes — so a human can browse the mirror and a client can discover the proof surface. This
closes M3.

## Scope
- **Create**: `internal/proofserve/browser.html` — the embedded log-browser template.
- **Modify** (≤3 non-test/doc files):
  1. `internal/proofserve/handler.go` — add `case "/": serveBrowser(...)` to the `Handler` path switch
     (before `default`); add `serveBrowser` (reads `FollowState` + `CheckpointAt`, renders the embedded
     template into a `bytes.Buffer`, then writes `200 text/html`); add the `bytes` + `_ "embed"` +
     `html/template` imports and a parsed-once package-scope `var browserTmpl`.
  2. `cmd/iscc-monitor/main.go` — in `hubHandler`, make the bare-`/` request reach `proofserve` instead
     of falling through to `tilesserve`'s 404. See Implementation Notes for the exact mux mechanics
     (an `http.ServeMux` cannot hold both an exact `/` and a subtree `/`).
- **Modify (doc)**: `CLAUDE.md` — the "Running a local dev instance" endpoint list: add the
  `GET /<domain>/log/` HTML log-browser line (currently undocumented).
- **Create (test)**: `internal/proofserve/browser_test.go` — golden HTTP-seam test over the
  `buildMirror` fixture store.
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — per-hub subtree mounting +
    `StripPrefix` strip discipline (strip `"/"+Origin`, leave the leading slash); proofserve's
    path-switch + 405/404 mapping; the most-specific-match rule for the inner hub mux.
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` — the render-into-buffer-then-copy
    HTML idiom, `html/template` (NOT `text/template`) for auto-escaping, the post-200 write-drop
    convention, and the coverage-honesty render rule.
  - `/workspace/iscc-monitor/internal/dashboard/handler.go` + `/workspace/iscc-monitor/internal/dashboard/dashboard.html`
    — the working `//go:embed` + `template.Must` + buffer-render + `text/html; charset=utf-8` pattern
    to mirror.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — `serveVerify`/`hubStatus` already read
    `FollowState` + `CheckpointAt(size)` and base64-Std encode the root; reuse that exact composition.
  - `/workspace/iscc-monitor/internal/proofserve/handler_test.go` `buildMirror` — the 300-leaf fixture
    that records + advances an accepted checkpoint; the browser test should drive `proofserve.Handler`
    over it.

## Not In Scope
- The proof-bundle assembler (`{checkpoint, inclusion proof, record bytes, hub key, ots?}`) — the next
  arc after M3 closes, a different (client-verifies) posture.
- CSS / JS / WASM progressive enhancement — the WASM verifier is a separate milestone; this page is
  plain server-rendered HTML like the dashboard.
- ETag / `Cache-Control` / conditional-GET on the browser page — not a Verify criterion (the static
  mirror has them; this HTML page does not need them for M3).
- Threading the richer in-memory statuses (`unverified`/`unresolvable`/`rotated`) — keep the
  store-provable subset (`frozen` else `verified`) exactly as `proofserve.hubStatus` already does.
- Refreshing the stale `sb1.amlet.id` did.json fixture or adding a registry-deactivation writer — both
  off the Verify bar.

## Implementation Notes
- **Mux mechanics (the load-bearing detail).** A request to `GET /<domain>/log/` is stripped by
  `mirrorHandler` (strip `"/"+Origin`, leaving the leading slash) so the inner `hubHandler` mux sees
  path `/`. Today `hubHandler` mounts `tilesserve.Handler` at `/` (a subtree), and `tilesserve`
  `TrimPrefix`es the leading slash → empty path → its `default` branch → **404**. So
  `GET /<domain>/log/` currently 404s; the step is to make the *bare* `/` render the browser while
  every other path (`/checkpoint`, `/tile/...`) still reaches tilesserve. In Go's `http.ServeMux` you
  cannot register both an exact `/` and a subtree `/` (the subtree pattern `/` *is* the bare-`/`
  match). Minimal fix: in `hubHandler`, replace the `mux.Handle("/", tilesserve…)` line so the `/`
  slot is a tiny `http.HandlerFunc` that does `if r.URL.Path == "/" { proofs.ServeHTTP(w, r); return }`
  then delegates to `tilesserve.Handler(...)`. `proofs` (the `proofserve.Handler`) then sees path `/`
  and its new `case "/"` renders the browser. Keep the four exact proof mounts (`/inclusion`,
  `/consistency`, `/entries`, `/verify`) unchanged — they still win by most-specific match. (The
  405 method-gate at the top of `proofserve.Handler` then also covers a non-GET to the bare `/`.)
- **Handler body.** Add `case "/": serveBrowser(w, r, st, hubID)` before `default`. `serveBrowser`
  reads `fs, err := st.FollowState(ctx, hubID)`; a DB error → 500. If `fs.LastSize == 0`, render the
  page in a "no accepted checkpoint yet" state (a **200**, not a 404 — the browser page exists for a
  followed-but-unpolled hub, mirroring the dashboard's "no coverage yet" honesty). Otherwise
  `root, _, found, err := st.CheckpointAt(ctx, hubID, fs.LastSize)`; base64-Std encode `root` exactly
  as `serveVerify` does (`base64.StdEncoding.EncodeToString(root)`). A DB read error → 500; a `!found`
  at the accepted size is the same real store inconsistency `serveVerify` treats as 500.
- **Template.** Mirror `dashboard.html` / `dashboard/handler.go`: `//go:embed browser.html` →
  `template.Must(template.New("browser").Parse(...))` at package scope; render the view-model into a
  `bytes.Buffer`; on template error → 500 *before* any 200; then
  `w.Header().Set("Content-Type", "text/html; charset=utf-8")`, `WriteHeader(200)`, `buf.WriteTo(w)`
  (post-200 write-drop). Use `html/template`, NOT `text/template`, so the origin/root strings
  auto-escape. The page must show the accepted `size` and the base64 `root`, the hub status (`frozen`
  else `verified`), and include relative links into the proof surface — e.g. `entries?index=0`,
  `inclusion?iscc_id=…`, `consistency?from=0`, `verify?iscc_id=…` (relative URLs resolve under the
  hub's `/<domain>/log/` base because the page is served from the trailing-slash subtree root).
- **No new store method needed.** `FollowState` (`internal/store/checkpoints.go:174`) and `CheckpointAt`
  (`:157`) already return everything; do not add a `ListHubs`-style method. Keep store a leaf —
  `proofserve` already depends on `store`, never the reverse.
- **Correctness rule (learnings index): coverage honesty (ADR-0001).** Do not present pre-coverage
  state as a guarantee; an unpolled hub renders an explicit "no accepted checkpoint yet", never a
  fabricated `(0, "")`.
- **Oracle/conformance gate is N/A here** — this is pure HTML rendering of persisted store rows (no
  signature / RFC-6962 / Merkle / did:web / fsck / proof computation); the served `(size, root)` are
  read back verbatim, never recomputed. State this in the review handoff. `go.mod`/`go.sum`/`schema.sql`
  must stay byte-identical (the browser reads existing columns only).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` empty).
- `go test -count=1 ./internal/proofserve` passes (the new `browser_test.go` plus all existing
  inclusion/consistency/entries/verify tests).
- `go test -count=1 ./cmd/iscc-monitor` passes uncached (the `hubHandler` mux rewiring keeps existing
  mirror routing intact).
- HTTP-seam assertion (in `browser_test.go`, on the `buildMirror` fixture, size 300): `GET /` → `200`,
  `Content-Type: text/html; charset=utf-8`, and the body contains the accepted size (`300`) and the
  base64-Std encoding of `tree.Hash()` (the accepted root). `POST /` → `405`.
- Mutation check (run by `advance`/`review`, reverted after): dropping the `{{.Root}}` (or size) cell
  from `browser.html` FAILS the golden body assert — proving the page non-vacuous.
- `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` stays empty (store remains
  a leaf; no new reverse dependency introduced).

## Done When
`mise run check` is green and `GET /<domain>/log/` returns `200 text/html` exposing the mirrored
checkpoint `(size, root)` with links into `entries`/proofs, golden-tested at the HTTP seam — closing
M3's fourth and final Verify criterion (M3 → 4/4).
