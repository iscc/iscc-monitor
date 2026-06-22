## 2026-06-22 — Review of: Serve the self-hosted ISCC logo at `/_ds/iscc-logo-black.png` and render it in the `/` + dossier mastheads

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Advance `7e3fe44` embeds a committed, pre-downscaled grayscale ISCC logo
(`internal/web/iscc-logo-black.png`, 199×76 gray+alpha, 2976 bytes) and serves it at `LogoPath`
`/_ds/iscc-logo-black.png` through the existing `writeAsset` no-cache+strong-ETag+304 leaf, then renders
it (with a 1px divider) beside the text mark on the two surfaces the front-of-queue human `critical`'s
verify bar measures — `/` (dashboard) and the hub dossier. The work is tight and exactly on-scope (3
source files + the PNG + tests, go.mod/go.sum untouched), all gates pass, the new tests are mutation-proven
non-vacuous, and the ADR-0012 visual pass confirms the logo renders correctly on both surfaces. The
`critical` is narrowed (not closed): the serve route + 2 surfaces landed; the remaining four mastheads are
the immediate follow-up.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 25 packages `ok`).
- [x] `gofmt -l` (excl. `cauldron/`) — clean (zero files listed).
- [x] `go test -run 'TestLogo|TestNoExternalCDN|TestWasmVerifyServed' ./internal/web` — PASS; `TestLogoServed` runs and asserts 200 / `image/png` / no-cache / strong ETag / byte-equal body / PNG magic / `If-None-Match`→304.
- [x] `go test -run TestDashboard ./internal/dashboard` — PASS; mutation-proven (drop the `<img>` → FAIL, restore → PASS).
- [x] `go test -run TestDossier ./internal/dossier` — PASS; mutation-proven (drop the `<img>` → FAIL, restore → PASS).
- [x] Content-Type mutation (`image/png`→`text/plain`) → `TestLogoServed` FAIL, restore → PASS (the type assertion is non-vacuous).
- [x] Committed PNG is small grayscale: `ls -l` = 2976 bytes, `file` = `PNG image data, 199 x 76, 8-bit gray+alpha`.
- [x] `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./internal/web` — exits 0 (leaf stays WASM-shareable; no `image/*` import, `image/png` is a string literal only).
- [x] Live instance (testnet realm, `127.0.0.1:41464`): `/healthz`=200, `GET /_ds/iscc-logo-black.png`=`200 image/png`, `/` body contains `src="/_ds/iscc-logo-black.png"`.
- [x] Oracle/conformance gate — N/A (pure static-asset transport; no signature/RFC-6962/Merkle/proof path touched).
- [x] Gate-circumvention scan over unpushed range (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/swallowed-error/build-tag/deleted-assertion patterns; only context + the 3 intended source files + tests + PNG.
- [x] Scope discipline — exactly 3 non-test/doc source files (`web.go`, `dashboard.html`, `dossier.html`), matching `next.md`; nothing from `## Not In Scope` done.

**Issues found:** (none new from this diff.) The human `critical` is narrowed, not deleted: the serve route
+ `/` + dossier landed and pass their verify bar, but the target.md:148 "every surface" requirement is not
yet met — four mastheads (`internal/certificate/cert.html`, `internal/proofserve/{browser,record,records}.html`)
still render the text-only mark (reviewer grep-confirmed NO LOGO). Issue rewritten to scope only the
remaining four (pure one-line-per-template edits now the route exists). DONE stays correctly unreachable.

**Codex second opinion:** Clean — "The new embedded logo asset is served through the existing static asset
path and referenced from the updated mastheads without breaking the existing handler behavior. Tests and
the full check suite pass, and no actionable correctness issues were found in the changed code." No findings
to triage; matches my own review. (Initial `/tmp/codex-review.txt` was empty while Codex was still running;
verdict landed after ~3 min of polling.)

**Visual check:** Performed (ADR-0012). `agent-browser` 0.29.0 is present and launches its own bundled
browser (no system Chrome, but the bundled one works). Screenshotted the live `/` index, the `sb0.iscc.id`
dossier, and the `Realm Index.dc.html` mockup. Both touched mastheads now render the ISCC mark + wordmark +
1px divider + "TRUST & TRANSPARENCY MONITOR" text — matching the mockup masthead exactly; the prior
text-only delta is gone. No NEW visual delta filed: the other index sub-region deltas still visible
(instance-identity copy, `· ISCC MAINNET` realm subtitle, Checkpoint/Anchor columns, "Recent declarers
checked" footer) are already tracked in the existing `normal` "/ realm-index sub-region deltas" issue
(items 2–4) and explicitly out of scope for this critical.

**Next:** The immediate follow-up named in `next.md`'s Not-In-Scope and now the open scope of the narrowed
`critical`: add the identical one-line `<img src="/_ds/iscc-logo-black.png">` + divider (inside a
`.chrome-brand` flex wrapper, copying the `dashboard.html` block + CSS) to the remaining FOUR mastheads —
`internal/certificate/cert.html` and `internal/proofserve/{browser,record,records}.html` — extending each
surface's handler test to assert the `<img>` src. Pure template edits; the serve route already exists.
After that the `critical` fully closes and the WASM milestone (tier-2 verifier caller into the dossier,
then the standalone `monitor.iscc.codes` Independent Verification app) resumes.

**Notes:**
- The masthead carries BOTH the logo and the text mark (target.md:148 wants both), with the 1px divider
  matching the mockup. Six near-identical mastheads now exist; the optional shared-partial KISS factoring
  is still a deferred `advance`-call, not required.
- `contentTypePNG` is a string literal only — never import an `image/*` package, or `internal/web` stops
  being the pure-stdlib WASM-green leaf. Recorded in `learnings/web.md`.
- The downscale ran ONCE at commit time (not in a `mise`/build step) — preserves the cross-platform,
  pure-Go, no-image-toolchain build posture (ADR-0003). The committed PNG is the build-pinned artifact,
  like the woff2 binaries and `verify.wasm`.
- Open lower-priority issues remain (OTS stamp-path guard, certificate §4/§5/§6 deltas, registry
  ForceQuery, the `/` sub-region deltas, several `low` localities) — none block this increment or the loop.
