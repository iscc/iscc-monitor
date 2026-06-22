# Next Work Package

## Step: Serve the self-hosted ISCC logo at `/_ds/iscc-logo-black.png` and render it in the `/` + dossier mastheads

## Advances
Preempts the milestone gap by closing the front-of-queue **human-filed `critical`**:
*"Add the ISCC logo to the nav-bar masthead chrome (self-hosted asset, replacing the text-only mark)"*
(`issues.md`). A `critical` preempts everything (protocol step 3), and the `review` handoff `**Next:**`
names exactly this. It is rooted in **target.md:148** ("Document chrome + instance identity — Every
surface carries the shared handoff header: the ISCC logo + 'Trust & Transparency Monitor' mark"), an
M-UI design-parity / Document-chrome requirement. DONE is unreachable while any `critical` is open, so
this must land before the WASM milestone continues.

## Goal
Serve the grayscale ISCC logo as a build-pinned, CDN-free `/_ds/` asset (mirroring the existing
`wasm_exec.js` / woff2 embed idiom) and render it as a plain `<img>` next to the existing text
`.chrome-mark` on the two surfaces the critical's verify bar asserts (`/` and the hub dossier). This
removes the text-only-mark delta from the index masthead — the surface measured by the ADR-0012 visual
pass — and lays the serve infrastructure the remaining four mastheads reuse with a one-line edit.

## Scope
- **Create**: `internal/web/iscc-logo-black.png` — a *committed, pre-downscaled* grayscale PNG generated
  ONCE by advance (not at build time; see Implementation Notes), embedded via `go:embed`.
- **Modify** (≤3 non-test/doc source files):
  1. `internal/web/web.go` — add `LogoPath` const, `contentTypePNG` const, `//go:embed iscc-logo-black.png`
     var, and a `case LogoPath:` in `Handler`'s path switch.
  2. `internal/dashboard/dashboard.html` — add `<img src="/_ds/iscc-logo-black.png" …>` beside the
     `.chrome-mark` (line ~315).
  3. `internal/dossier/dossier.html` — same `<img>` beside the `.chrome-mark` (line ~300).
- **Tests (not counted in the 3)**: add `TestLogoServed` to `internal/web/web_test.go`; extend
  `internal/dashboard/handler_test.go` and `internal/dossier/handler_test.go` to assert the `<img>` in
  the served body.
- **Reference**:
  - `.claude/context/learnings/web.md` — the `/_ds/` subtree mount, `writeAsset` no-cache+strong-ETag+304
    leaf, and the `wasm_exec.js`/woff2 embed pattern to copy verbatim.
  - `internal/web/web.go:166-185` — `Handler`'s path-switch + `writeAsset` (the exact idiom to extend).
  - `.claude/design/ISCC Monitor - Realm Index.dc.html:30-34` — the mockup masthead: `<img src=
    "assets/iscc-logo-black.png" alt="ISCC" style="height:19px;width:auto;display:block">` then a
    `1px×24px` divider span then the text mark — the layout to reproduce.
  - `.claude/design/assets/iscc-logo-black.png` — the source asset (5000×1906 gray+alpha, 113 KB) to
    downscale and copy in.
  - `issues.md` — the critical's full "Verify fixed" criteria.

## Not In Scope
- The remaining FOUR mastheads (`internal/certificate/cert.html`, `internal/proofserve/{browser,record,
  records}.html`) get the identical one-line `<img>` insertion in the **immediate follow-up step** — the
  serve route this step lands makes that a pure-template edit. Listing them here keeps the arc coherent
  rather than ballooning this step past the 3-file bar.
- Extracting a shared chrome partial (the issue's optional KISS factoring) — defer; six near-identical
  one-liners is the smaller, lower-risk move now.
- Any white/dark logo variant — only the black variant exists and the masthead is light (issue scope).
- Request-time resizing — forbidden; the served bytes are the committed pre-downscaled asset.
- The `/` `normal` sub-region deltas (instance-identity copy, Checkpoint/Anchor columns, recent-declarers
  footer) — separate `normal` issue, not this critical.

## Implementation Notes
- **Downscale at the embed step, NOT at build time.** The mockup renders the logo at `height:19px`. The
  source is 5000×1906 / 113 KB — wildly oversized. Advance must run a ONE-TIME downscale to a slim,
  height-appropriate (≤2× retina, e.g. ~76px tall) grayscale PNG and COMMIT that file as
  `internal/web/iscc-logo-black.png`. Use the available `convert` (ImageMagick), e.g.
  `convert ".claude/design/assets/iscc-logo-black.png" -resize x76 -strip "internal/web/iscc-logo-black.png"`
  (verify the result is a few KB, preserves the alpha channel so it sits on the `#fbf9f4` chrome, and
  stays grayscale). **Do NOT add a `mise`/build-step that resizes** — that would require ImageMagick on
  every CI/dev machine and break the cross-platform, reproducible-build posture (ADR-0003,
  `CGO_ENABLED=0`, pure-Go). The committed PNG is the build-pinned artifact, exactly like the committed
  woff2 binaries and `verify.wasm`.
- **Serve via the existing leaf, do not invent a new policy.** Add to `internal/web/web.go`:
  `const LogoPath = "/_ds/iscc-logo-black.png"`, `const contentTypePNG = "image/png"`,
  `//go:embed iscc-logo-black.png` → `var logoPNG []byte`, and a `case LogoPath:
  writeAsset(w, r, logoPNG, contentTypePNG)` in `Handler`. `writeAsset` already gives the
  no-cache + strong-content-ETag + `If-None-Match`→304 policy the sibling `/_ds/` assets use — reuse it,
  set NO extra headers (the issue's "sibling-`/_ds/` ETag/304 policy"). CORS rides the outer `corsmw`
  wrap; this handler sets none.
- **Template insertion.** Beside the existing `<div class="chrome-mark">Trust &amp; Transparency
  Monitor</div>`, add `<img src="/_ds/iscc-logo-black.png" alt="ISCC" style="height:19px;width:auto">`
  (and, matching the mockup, an optional `1px` divider span) so the served HTML carries **both** the
  logo and the text mark (target.md:148 wants both). The src is a literal string in the template — like
  `TokensPath`, templates cannot read the Go const, so the literal must match `LogoPath` exactly.
- **Purity / WASM-green is preserved.** `embed` is already imported and `image/png` is only a string
  literal (no `image/*` package import), so `internal/web` stays the pure stdlib leaf
  (`bytes`/`embed`/`fmt`/`io/fs`/`net/http`/`strings`/`crypto/sha256`) and `GOOS=js GOARCH=wasm go build
  ./internal/web` stays OK (learnings/web.md "Pure stdlib leaf, WASM-green").
- **No-CDN ban is satisfied.** A same-origin `/_ds/iscc-logo-black.png` is a relative path with no
  `http://`/`https://`/`cdn.`/`jsdelivr` substring, so it passes `noExternalCDN` and the dashboard body
  ban (`TestDashboardLinksTokensNoCDN`) — same class as the existing `/_ds/wasm_exec.js` ref.
- **Correctness rule (learnings.md):** none of the crypto/freeze/re-verify rules apply — this is pure
  static-asset transport; the oracle/conformance gate is **N/A** (no signature/RFC-6962/Merkle/proof
  path touched), exactly as `internal/web`'s docstring states.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` excl.
  `cauldron/` empty).
- `go test -run 'TestLogo|TestNoExternalCDN|TestWasmVerifyServed' ./internal/web` passes — including a
  new `TestLogoServed` asserting `GET /_ds/iscc-logo-black.png` → `200`, `Content-Type: image/png`,
  body byte-equal to the embedded bytes, a strong ETag, and an `If-None-Match` of that ETag → `304`.
- `go test -run TestDashboard ./internal/dashboard` passes — the `/` body assertion now requires
  `src="/_ds/iscc-logo-black.png"` present in the masthead (mutation: drop the `<img>` → FAIL).
- `go test -run TestDossier ./internal/dossier` passes — same `<img>` assertion on the dossier body.
- The committed `internal/web/iscc-logo-black.png` is a grayscale PNG of a few KB (not 113 KB):
  `ls -l internal/web/iscc-logo-black.png` shows a small file and `file` reports `PNG image data`.
- `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./internal/web` exits 0 (the leaf stays WASM-shareable).

## Done When
`mise run check` is green, `GET /_ds/iscc-logo-black.png` returns a small build-pinned `image/png` with
the sibling `/_ds/` ETag/304 policy, and the served `/` and hub-dossier HTML both carry
`<img src="/_ds/iscc-logo-black.png">` beside the existing text mark — all asserted by the tests above.
