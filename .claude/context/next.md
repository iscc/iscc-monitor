# Next Work Package

## Step: Surface-C skeleton — render the standalone Independent Verification page (`internal/verifier`)

## Advances
WASM verifier upgrade milestone (`target.md` §"WASM verifier upgrade"):
> "plus the standalone **Independent Verification** verifier app (`ISCC Monitor - Independent
> Verification.dc.html`, Surface C) at `monitor.iscc.codes` (monitor-agnostic via `?monitor=<url>`) …
> **Verify:** … a `(size, root)` mismatch renders the guided split-view alert, not a dead error."

This is the larger of the two open WASM sub-steps named by `state.md`'s "Next Milestone" / the
`review` handoff (the other being a dossier tier-2 caller). Surface C currently does **not exist**
(grep-confirmed: no `cmd/verifier*` / `internal/verifier*`). Per the protocol's skeleton-first rule
for a big feature, this step lays the *verifiable skeleton* — the standalone page (named regions +
the guided split-view alert copy + the `.codes ↔ .id` independence statement + a reference to the
audited `verify.wasm`) with a golden HTTP-seam test — and defers the live `?monitor=<url>` fetch +
WASM verdict wiring and the GitHub Pages deploy to later sub-steps (listed under `## Not In Scope`).

## Goal
Create the standalone Independent Verification surface (Surface C) as a server-rendered HTML page in a
new `internal/verifier` leaf, so the named regions of the `Independent Verification.dc.html` mockup —
including the **guided split-view mismatch alert** ("Mismatch — possible split view … do not discard
either") and the `monitor.iscc.codes` ↔ `monitor.iscc.id` independence statement — are present and
golden-tested. This is the first half of the open WASM Surface-C Verify criterion; it makes the app
real and testable before the live WASM `?monitor=` fetch is wired.

## Scope
- **Create**: `internal/verifier/handler.go` (the `package verifier` leaf: an `//go:embed`-ed
  `verifier.html`, parsed once with `template.Must` at init, plus an exported `http.Handler` —
  GET-only, 405 otherwise, buffer-then-200 — that renders the page; mirror the `internal/dossier`
  handler shape).
- **Create**: `internal/verifier/verifier.html` (the SSR template; port the named regions from the
  mockup, subordinate to the no-JS / no-CDN / self-hosted constraints — see Implementation Notes).
- **Create**: `internal/verifier/handler_test.go` (the golden HTTP-seam test — not counted against
  the 3-file budget).
- **Reference**:
  - `.claude/design/ISCC Monitor - Independent Verification.dc.html` — the authoritative mockup
    (named regions, the verification-record block, the split-view input, the two alert states).
  - `.claude/design/ISCC Monitor - Developer Handoff.dc.html` lines 64 / 112 / 185–189 / 282 — the
    Surface-C contract: `monitor.iscc.codes`, "Static · Go→WASM", `?monitor={url}&id={id}`, "the
    `.codes ↔ .id` distinction stays visible".
  - `internal/dossier/handler.go` — the SSR leaf shape to mirror (embed + `template.Must` init +
    GET/405 + buffer-then-200 + Content-Type).
  - `internal/dashboard/dashboard.html` lines 28–95 + 328–342 — the shared document-chrome markup +
    CSS (logo / divider / mark / instance-identity / `chrome-verify` link) to reuse byte-for-byte.
  - `internal/web/web.go` lines 54–93 — `Prefix`/`WasmExecPath`/`WasmVerifyPath`/`WasmVerifyHash`
    (the audited-artifact pin the page references and the test asserts).
  - `internal/certificate/cert.html` lines 499–567 + `internal/certificate/handler_test.go` lines
    715–861 — the tier-2 loader markup + its golden-assertion test, as the *pattern* for the later
    live-wiring sub-step (do NOT port the full loader now; see Not In Scope).
  - `.claude/context/learnings/cmd-wasm.md` — the WASM caller mechanics (data-island JSON-escaping,
    the three distinct render states `error`/`failed`/`verified`, the `isccVerifyInclusion` global)
    that the *deferred* live-wiring sub-step will follow.

## Not In Scope
- **No live `?monitor=<url>` fetch or WASM verdict execution this step.** Do NOT embed/run
  `isccVerifyInclusion`, do NOT instantiate `verify.wasm`, do NOT parse a query string. The page is a
  static SSR skeleton; the alert states render as static markup (matching the no-JS baseline), and the
  live WASM wiring is the next Surface-C sub-step.
- **No GitHub Pages deploy / `monitor.iscc.codes` workflow.** Adding a `.github/workflows/pages.yml`
  or copying `verify.wasm`/`wasm_exec.js` into a published static bundle is a separate later sub-step.
- **No `cmd/iscc-monitor` mount change.** Surface C lives on a *different origin* (a static site), not
  the instance binary; do NOT register this handler in `buildMux`. The exported handler exists for the
  golden test (and a future local-preview/deploy harness), not for instance serving.
- **No dossier tier-2 caller** (the sibling open WASM sub-step) — a separate step.
- **Do not touch** any open `normal` issue (certificate §5 digest binding, OTS `safeStamp`, `hubDomain`
  ForceQuery, §4/bundle `host:port` DID, §6 timestamp, the `safeIndex` test gap, the tier-2 honesty
  copy, the `/` sub-region deltas) — none is on this step's path.

## Implementation Notes
- **Mirror the `internal/dossier` leaf exactly**: `//go:embed verifier.html` → `template.Must(... )` at
  package init (a malformed template fails the build, not a request); the exported `Handler()
  http.Handler` is GET-only (`http.MethodGet` else 405), renders into a `bytes.Buffer`, then writes
  `Content-Type: text/html; charset=utf-8` + 200 + `buf.WriteTo(w)` (post-200 write-drop). This page
  has no per-request data, so `Handler()` takes no args and the view-model can be a small static struct
  (or none) — keep it the simplest thing that renders the template.
- **Self-hosted + no-CDN, like every other surface.** The mockup links `_ds/...tokens/*.css` at a
  hashed jsDelivr-style path and runs `_ds_bundle.js` — the **constraint wins** (ADR-0010 design-parity:
  "where a mockup conflicts … the constraint wins and the deviation is flagged"). Link the shared
  shell at the same literals the other surfaces use — `/_ds/tokens.css` (and `/_ds/fonts.css` if the
  page needs it) — and reference the WASM artifacts at `/_ds/wasm_exec.js` + `/_ds/verify.wasm`
  (the future static deploy republishes these same audited bytes at its own origin; keep the literal
  paths in sync with `web.WasmExecPath`/`web.WasmVerifyPath`). Put **no** third-party origin in the
  body — the `noExternalCDN`-class bans (jsdelivr / `http://` / `https://` / `cdn.`) the sibling
  surfaces enforce apply here too; assert it in the test.
- **Port the named regions** from the mockup (region checklist, never a pixel diff):
  - the shared **document chrome** (reuse `dashboard.html`'s `.chrome*` block + the three logo CSS
    rules byte-for-byte: `<img class="chrome-logo" src="/_ds/iscc-logo-black.png" alt="ISCC">` +
    divider + "Trust &amp; Transparency Monitor" mark), with the instance-identity block and the
    `verify ↗ monitor.iscc.codes` link;
  - the `← Certificate` back-link (navigation closure — the mockup's breadcrumb);
  - the eyebrow **"Independent verification"** + the "Re-run the proof yourself." head;
  - the **independence statement** naming both origins — the result is produced *here* and
    `monitor.iscc.id` (the instance) is *not in the trust path*; the `.codes`/`.id` distinction must
    be visible (Handoff invariant 5);
  - the **verification-record block** (the 5 step labels from the mockup's `vSteps` —
    "Fetch proof bundle…", "Recompute the leaf hash…", "Walk the Merkle inclusion path", "Rebuild the
    root and match the hub-signed checkpoint", "Check the signature against the hub's did:web key") as
    static no-JS list rows;
  - the **split-view input** region (Tree size + Root hash + a "Compare" affordance) — as a plain
    `GET` form / static inputs, the no-JS equivalent of the mockup's JS buttons;
  - the **guided split-view alert** copy — "Mismatch — possible split view" + the guidance
    ("…hub may have shown you a different history. Keep your signed checkpoint and the bundle … do not
    discard either.") — rendered as static markup. This is the load-bearing Verify fragment: the
    mismatch path is a *guided alert*, never a dead error.
- **Relevant Correctness rule (learnings.md, always-loaded):** "On a self-verifiable surface, gate a
  rendered ✓/Merkle assertion on a re-VERIFICATION, not a status flag." The skeleton renders **no**
  live ✓ verdict yet (the WASM run is deferred) — so be careful the static copy does **not** assert in
  the present tense that "this browser verified …" the way the certificate's no-JS honesty bug does
  (the open `normal` issue). The skeleton's verification-record block describes the *steps the verifier
  will run* and offers the "Run verification in my browser" affordance; it must not claim a verdict
  that has not run. The actual ✓-on-re-verify gate lands with the deferred live-wiring sub-step.
- **Keep the leaf pure** (like `internal/web`/`internal/dossier`): stdlib + `internal/web` only (for
  the path consts if referenced); no `internal/store`, no `internal/metrics`, no network. The
  oracle/conformance gate is **N/A** (pure static HTML render; no signature/RFC-6962/Merkle/proof path
  touched) — state that in the package docstring, as the sibling leaves do.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test ./internal/verifier` passes.
- The golden test asserts the served body (`Handler()` over an `httptest` request) is `200
  text/html; charset=utf-8` and **contains** each named region marker:
  `"Independent verification"`, the independence statement naming both `monitor.iscc.codes` and
  `monitor.iscc.id`, all five verification-record step labels, `"Mismatch — possible split view"`,
  the "do not discard either" guidance fragment, `src="/_ds/iscc-logo-black.png"`, and
  `"/_ds/verify.wasm"`.
- The test bans external CDNs in the body (no `jsdelivr` / `http://` / `https://` / `cdn.`), matching
  the sibling no-CDN seam tests.
- The test asserts a non-GET request returns 405.
- At least one assertion is mutation-proven by the `advance` step (e.g. dropping the
  `"Mismatch — possible split view"` alert block makes the test FAIL).

## Done When
`internal/verifier` renders the standalone Independent Verification page with the chrome, the
independence statement, the verification-record steps, and the guided split-view mismatch alert, the
golden HTTP-seam test asserts every named region (mutation-proven non-vacuous), the body is no-CDN and
references `/_ds/verify.wasm`, and `mise run check` is green.
