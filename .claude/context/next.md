# Next Work Package

## Step: Embed the DS v2 token CSS as a shared `internal/web` leaf and link it from `GET /` (no CDN URL in body)

## Advances
M-UI — Evidence Ledger frontend, the shared-shell prerequisite of its Verify criterion:

> "every SSR surface (`/` realm index, hub dossier, log-browser record list, single record, certificate)
> returns `200 text/html`, embeds the DS tokens + self-hosted fonts with **no external CDN URL in the
> body**, and is **complete with JavaScript disabled** … `mise run check` green; any new store
> status-derivation stays a leaf read (store keeps no `net/http`/web dependency)."

This is the **skeleton-first** first slice of that criterion and the top item of the state's
convergence-driven order ("DS v2 tokens + self-hosted fonts embedded — the shared shell every remaining
surface needs"). Stand up the embedded DS-token surface every remaining M-UI screen will link, and
prove `/` carries the tokens with **no CDN URL in its body**. The self-hosted **fonts** half (the woff2
binaries) is split into its own later sub-step (see Not In Scope) — the woff2 files don't exist in the
repo and can't be fetched at build time, so it cannot ride in this ≤3-file step.

## Goal
Create a tiny stdlib-only static-asset package (`internal/web`) that `go:embed`s the ISCC Design
System v2 **token CSS** (a single concatenated, CDN-free stylesheet) and serves it at a stable path;
wire `cmd/iscc-monitor`'s mux to mount it and have the `/` dashboard `<head>` link it. This gives every
later M-UI surface (dossier, record list, certificate) one shared, no-JS, no-CDN style shell to link.

## Scope
- **Create**: `internal/web/web.go` — package doc + `//go:embed tokens.css` `var TokensCSS []byte`,
  a `const TokensPath = "/_ds/tokens.css"`, and `Handler() http.Handler` that serves `TokensCSS` as
  `text/css; charset=utf-8` (GET-only → 405 otherwise; `Cache-Control: public, max-age=31536000,
  immutable` is fine since the bytes are build-pinned). Keep it a pure leaf (`bytes`/`embed`/`net/http`
  only; no `internal/store`, no `internal/metrics`).
- **Create**: `internal/web/tokens.css` — the concatenated DS token CSS, **CDN-free**. Port the four
  pure-token files verbatim in this order: `colors.css`, `typography.css`, `spacing.css`, `base.css`
  from the `_ds/.../tokens/` bundle. **Do NOT include `fonts.css`** (it is the jsDelivr `@font-face`
  block — that is the fonts sub-step). In `base.css`'s `.iscc-grain` rule, the `url("../assets/grain.png")`
  reference points at an asset this step does not serve — neutralize it (drop the `.iscc-grain`
  `background-image` line) so the served CSS has **zero external or dangling URLs**. The
  `--font-sans`/`--font-mono` stacks already list `Arial`/`Consolas` fallbacks, so the page renders
  correctly with no webfont loaded.
- **Modify**: `internal/dashboard/dashboard.html` — add `<link rel="stylesheet" href="/_ds/tokens.css">`
  in `<head>` (literal path matching `web.TokensPath`; the template has no access to Go consts, so the
  literal must stay in sync — note it). No inline `style=`, no CDN URL.
- **Modify**: `cmd/iscc-monitor/main.go` — in `buildMux`, mount `web.Handler()` at the exact path
  `web.TokensPath` (`mux.Handle(web.TokensPath, web.Handler())`) next to `/metrics` and `/healthz`.
  Most-specific match keeps it from shadowing `/` or the per-hub subtrees; update the `buildMux` doc
  comment (lines ~141-164) to mention the new exact route.
- **Modify (doc, not counted against the 3-file limit)**: `CLAUDE.md` — add `GET /_ds/tokens.css` to the
  HTTP-surface list under the `GET /` bullet.
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/design/_ds/iscc-design-system-v2-50a54aa2-10e0-420b-8f94-b11168b55f5e/tokens/colors.css`,
    `.../typography.css`, `.../spacing.css`, `.../base.css` (the verbatim token sources to concatenate)
  - `/workspace/iscc-monitor/.claude/design/_ds/iscc-design-system-v2-50a54aa2-10e0-420b-8f94-b11168b55f5e/tokens/fonts.css`
    (the CDN `@font-face` block — confirm what is being excluded and why)
  - `/workspace/iscc-monitor/.claude/design/ISCC Monitor - Realm Index.dc.html` (head/link structure;
    note its `<link>`s are the design source, but we serve ONE concatenated `tokens.css`)
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` (the `/`-mount + exact-path-guard +
    buffer-first render rules — do NOT regress them)
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` (tilesserve `Cache-Control`/ETag
    conventions, the single `corsmw.Handler(mux)` wrap)
  - `/workspace/iscc-monitor/internal/badge/badge.go` (the `go:embed` + stdlib-leaf idiom to mirror)
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` `buildMux` (lines ~158-164, the exact-path mount
    pattern next to `/metrics`/`/healthz`)

## Not In Scope
- **Self-hosted fonts (woff2 binaries).** Do NOT add `fonts.css`, `@font-face`, or any
  `cdn.jsdelivr.net` URL. Obtaining + `go:embed`ing the Readex Pro / JetBrains Mono woff2 files and
  serving them under `/_ds/fonts/...` (replacing the CDN block with self-hosted `src: url(...)`) is the
  **next** sub-step of this criterion — it needs the binaries fetched and committed, which is its own
  ≤3-file change. The Arial/Consolas fallback stacks already make `/` render correctly meanwhile.
- **Restyling the dashboard markup.** Only add the `<link>`; do not rewrite the `<table>` into the
  Evidence-Ledger realm-register grid, add header/footer chrome, or apply token classes to rows. That
  is the realm-index redress sub-step.
- **Wiring tokens into the log browser / future dossier / record / certificate pages.** Each links the
  same `/_ds/tokens.css` later, surface by surface.
- **The grain texture asset** (`assets/grain.png`) and any image serving — out of scope; the grain rule
  is neutralized, not served.
- No `internal/store`/`internal/metrics` import in `internal/web`; no new dep, no schema change.

## Implementation Notes
- **Mirror the `internal/badge` leaf idiom**: package doc explaining purpose, `//go:embed` a sibling
  file into a package var, serve once. Here it is simpler — raw CSS bytes, not a template — so
  `var TokensCSS []byte` is enough; no `html/template` needed.
- **Handler shape** mirrors `internal/healthz`/`internal/metricshttp`: a `http.HandlerFunc` that
  rejects non-GET with 405, sets `Content-Type: text/css; charset=utf-8`, and writes the embedded
  bytes. The CORS wildcard already rides the outer `corsmw.Handler(mux)` wrap (per
  `learnings/http-surface.md`), so do NOT set CORS headers here. A strong content `ETag` +
  `If-None-Match`→304 is optional polish (tilesserve has the pattern) — `Cache-Control: public,
  max-age=31536000, immutable` is sufficient and simplest since the bytes are build-pinned; do not
  over-build (KISS/YAGNI).
- **CDN-free is the load-bearing invariant** (target M-UI Verify: "no external CDN URL in the body").
  The only file in the token bundle that carries CDN URLs is `fonts.css`; excluding it is what keeps
  `tokens.css` and the `/` body clean. Grep the concatenated `tokens.css` for `jsdelivr`/`http://`/
  `https://` and `url(` before finalizing — the only `url(` is the grain `background-image`, which must
  be neutralized.
- **Exact-path mount, not subtree.** Mount at the exact `/_ds/tokens.css` (no trailing slash), so
  `http.ServeMux` most-specific match keeps it isolated; the dashboard's own `r.URL.Path != "/"` guard
  is unaffected (per `learnings/dashboard.md`: keep BOTH the `/` mount AND the in-handler path guard).
  Do NOT mount a `/_ds/` subtree yet — a single exact file needs no subtree, and a subtree would invite
  the fonts sub-step to sneak in here.
- **`html/template` auto-escaping**: the new `<link href="/_ds/tokens.css">` is a static literal in the
  template with no dynamic data — no escaping concern. Keep the dashboard's existing buffer-first render
  (500-before-200) and exact-path/method guards in `Handler` untouched.
- **Oracle/conformance gate is N/A** (per learnings): pure static-asset transport + a static `<link>`
  in HTML; no signature / RFC-6962 / Merkle / did:web / fsck / proof path. `go.mod`/`go.sum`/`schema.sql`
  stay byte-identical (verify in review).
- **Leaf purity** (learnings index rule "store keeps no web dependency"): `go list -deps ./internal/web`
  must contain no `internal/store`/`internal/metrics`/`internal/logclient`; `internal/web` does not
  touch store at all.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run 'TestTokens|TestDashboard' ./internal/web ./internal/dashboard` passes (new
  `internal/web` test for the CSS handler + the existing dashboard golden still passing with the added
  `<link>`).
- New `internal/web` test asserts: `GET /_ds/tokens.css` (through `web.Handler()` on an
  `httptest.ResponseRecorder`) returns `200` with a `Content-Type` starting `text/css` and a non-empty
  body containing a known token (e.g. `--iscc-blue`), and a non-GET returns `405`.
- CDN-free assertion (mechanical, in the `internal/web` test): the served `tokens.css` body contains
  **no** substring `jsdelivr` and **no** `http` — i.e.
  `!bytes.Contains(body, []byte("jsdelivr")) && !bytes.Contains(body, []byte("http"))`.
- Dashboard-body assertion (mechanical): the `GET /` HTML body contains `href="/_ds/tokens.css"` and
  contains **no** `jsdelivr`/`http://`/`https://`/`cdn.` substring (extend the existing dashboard
  HTTP-seam test or add `TestDashboardLinksTokensNoCDN`).
- `go list -deps ./internal/web | grep -E 'internal/(store|metrics|logclient)'` is empty (leaf purity).
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (no dep/schema change).

## Done When
`mise run check` is green and the new tests prove `/_ds/tokens.css` serves the CDN-free DS token CSS
and `GET /`'s HTML body links it with no external CDN URL — the shared embedded-token shell every later
M-UI surface links, with self-hosted fonts deferred to the next sub-step.
