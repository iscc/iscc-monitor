<!-- area: cmd/verifier-site (Surface-C static-site generator, the monitor.iscc.codes deploy build command) -->
<!-- indexed-as: verifier-site.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `cmd/verifier-site` — Surface-C static-site generator

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Generator shape (build-time, NOT a request handler)

- **One source of truth: drive the REAL handlers over httptest, never re-embed.** `index.html` is
  `verifier.Handler()`'s GET `/` body; each `/_ds/` asset is `web.Handler()`'s GET at its URL path,
  written at `assetPath(out, urlPath)` so the on-disk tree matches what the page fetches (GitHub Pages
  serves files at their path). This guarantees the deployed bytes equal what the golden tests already
  gate — do not add a second template/asset source.
- **Two internal deps ONLY (`internal/verifier`, `internal/web`); pure-stdlib otherwise.** Reviewer-
  confirmed `go list -deps ./cmd/verifier-site` has no `database/sql`/`internal/store`/`logclient`/
  `follower`; `net/http` rides in solely via `net/http/httptest` (allowed). It copies already-built
  embedded bytes — never a network fetch, a DB, or a `go build -GOOS=js`.
- **Fonts are enumerated from `fonts.css`, NOT hardcoded.** `fontPaths` substring-scans for the marker
  `"` + `web.Prefix` + `fonts/` (the `"`-preceded `src: url("/_ds/fonts/…woff2")` form, mirroring
  `web_test.go`'s `TestFontsCSSReferencesEmbeddedSubsets`). Exactly 8 markers today, all legit `src:`
  lines. A future font add/remove in `fonts.css` flows through; if the embed is missing, the GET 404s
  and `generate` fails closed (desirable). It is a substring scanner, not a CSS parser — a marker in a
  comment would mint a spurious GET; acceptable for a hand-maintained, golden-tested file.
- **The generated `verify.wasm`'s SHA-256 == `web.WasmVerifyHash` (reviewer re-measured `7d57ab1b…`).**
  Proves the generator COPIES the byte-pinned SRI artifact, never rebuilds it. The published deploy hash
  therefore inherits the `-buildvcs=false` reproducibility (web.md). `TestGenerate` asserts this.

## Fail-closed nuance (the right vs the wrong mutation probe)

- **The fail-closed gate fires on a HANDLER-served-path 404, NOT on a `web.*` const rename.** Renaming
  `web.TokensPath` moves BOTH the generator's request path AND the handler's `switch` case together, so
  the GET still returns 200 — `TestGenerate` stays GREEN (reviewer-verified; same vacuity class the
  advance handoff flagged for a bad `verifier.Handler` URL). The correct mutation is to make the handler
  404 the expected path (rename only the `case` body): then `generate` aborts with
  `GET /_ds/tokens.css = 404, want 200` and the test FAILS (reviewer mutation-proven, restored → green).
  Use the handler-404 probe, not the const rename, to test this generator's fail-closed contract.
- **Output is NON-ATOMIC — a partial tree survives on the error path (open `low`, Codex P3).**
  `generate` writes `index.html` first (`main.go:66`) then render-then-writes each asset in a loop, so a
  later 404 or write error returns an error AFTER `index.html`/earlier assets are on disk, leaving a
  partially-updated tree in a reused `dist/`. NOT a current hazard: the run still errors → `os.Exit(1)` →
  CI/test catches it (a broken deploy is never silently published), and `TestGenerate` uses a fresh
  `t.TempDir()`. Fix when next touched: stage into a temp dir + rename, or buffer all responses before
  the first write, so the output dir is updated atomically. Verify fixed: a forced mid-run render error
  leaves `outDir` unchanged (no stale `index.html`); reverting makes it FAIL.
