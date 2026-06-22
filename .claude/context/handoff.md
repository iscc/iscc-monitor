## 2026-06-22 — Serve the self-hosted ISCC logo at `/_ds/iscc-logo-black.png` and render it in the `/` + dossier mastheads

**Done:** Added a build-pinned grayscale ISCC logo as a `go:embed`-ed `/_ds/` asset served through the
existing `writeAsset` leaf, and rendered it as an `<img>` beside the text `.chrome-mark` in the realm-index
(`/`) and hub-dossier mastheads (with a 1px divider, matching the mockup). Closes the front-of-queue
human-filed `critical` (ISCC logo masthead chrome). The serve route is now in place for the remaining four
mastheads to reuse with a one-line `<img>` edit.

**Files changed:**
- `internal/web/iscc-logo-black.png`: committed, pre-downscaled grayscale PNG (199×76, gray+alpha, 2976 bytes
  ≈3 KB) — `convert .claude/design/assets/iscc-logo-black.png -resize x76 -strip …` run ONCE at this step
  (not at build time; no image toolchain on the build path, preserving the pure-Go CGO_ENABLED=0 posture).
- `internal/web/web.go`: added `LogoPath` const, `contentTypePNG` const, `//go:embed iscc-logo-black.png`
  → `var logoPNG`, and a `case LogoPath: writeAsset(w, r, logoPNG, contentTypePNG)` in `Handler`'s path
  switch. Updated the package + `Handler` doc comments. No new imports (`image/png` is a string literal
  only), so the leaf stays pure stdlib and WASM-green.
- `internal/dashboard/dashboard.html`: wrapped the masthead in `.chrome-brand` (flex), added the
  `<img class="chrome-logo" src="/_ds/iscc-logo-black.png" alt="ISCC">` + a `.chrome-divider` span before
  the existing text block; added `.chrome-brand`/`.chrome-logo`/`.chrome-divider` CSS (height:19px,
  `--space-3` gap, 1px×24px divider).
- `internal/dossier/dossier.html`: identical masthead change + CSS.
- `internal/web/web_test.go`: added `TestLogoServed` — `GET /_ds/iscc-logo-black.png` → 200,
  `Content-Type: image/png`, body byte-equal to embedded bytes, PNG magic, strong ETag, `If-None-Match`→304.
- `internal/dashboard/handler_test.go`: extended `TestDashboardRendersHeroAndNavigation` to assert
  `src="/_ds/iscc-logo-black.png"` in the masthead.
- `internal/dossier/handler_test.go`: extended `TestDossierRendersCoveredHub` DS-shell block to assert the
  same `<img>` src.

**Verification:** `mise run check` → green (all 25 packages `ok`); `gofmt -l .` excl. `cauldron/` clean.
Per criterion:
- [x] `go test -run 'TestLogo|TestNoExternalCDN|TestWasmVerifyServed' ./internal/web` PASS (incl. new `TestLogoServed`).
- [x] `go test -run TestDashboard ./internal/dashboard` PASS; mutation-proven non-vacuous (dropped the `<img>` → FAIL, restored → PASS).
- [x] `go test -run TestDossier ./internal/dossier` PASS.
- [x] Committed PNG is small grayscale PNG: `ls -l` shows 2976 bytes (not 113 KB), `file` reports `PNG image data, 199 x 76, 8-bit gray+alpha`.
- [x] `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./internal/web` exits 0 (leaf stays WASM-shareable).
- [x] No-CDN ban satisfied: same-origin `/_ds/iscc-logo-black.png` has no `http(s)://`/`cdn.`/`jsdelivr` substring; `TestDashboardLinksTokensNoCDN` + the dossier no-CDN body scan stay green.

**Next:** The immediate follow-up named in `next.md` Not-In-Scope: add the identical one-line
`<img src="/_ds/iscc-logo-black.png" …>` (+ divider) to the remaining FOUR mastheads —
`internal/certificate/cert.html` and `internal/proofserve/{browser,record,records}.html` — now that the
serve route exists; pure template edits, each beside its own `.chrome-mark`. After that, continue the WASM
milestone (wire the tier-2 verifier caller into the dossier, then the standalone `monitor.iscc.codes`
Independent Verification app). Optional KISS factoring of a shared chrome partial (the six near-identical
mastheads) is deferred per the issue.

**Notes:**
- The masthead now carries BOTH the logo and the text mark (target.md:148 wants both), with a 1px divider
  between them per the mockup (`.dc.html:30-34`).
- The `image/png` content type is a string const only — no `image/*` package imported — so the leaf's
  pure-stdlib closure (`crypto/sha256`/`embed`/`fmt`/`io/fs`/`net/http`/`strings`) is unchanged and the WASM
  build still passes. Confirmed via `GOOS=js GOARCH=wasm go build ./internal/web`.
- Oracle/conformance gate: N/A — pure static-asset transport, no signature/RFC-6962/Merkle/proof path
  touched (matches `internal/web`'s docstring and the `next.md` correctness rule).
- Visual check (ADR-0012) is a `review` responsibility: the `/` masthead is the surface the critical's
  verify bar measures; the served HTML now carries the `<img>` (asserted in-test), but a live screenshot vs
  the `Realm Index.dc.html` mockup is the reviewer's call.
