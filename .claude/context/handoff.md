## 2026-06-22 — Render the self-hosted ISCC logo in the remaining FOUR SSR mastheads

**Done:** Added the identical one-line `<img class="chrome-logo" src="/_ds/iscc-logo-black.png" alt="ISCC">` + `.chrome-divider` (inside a `.chrome-brand` flex wrapper) plus the three `.chrome-brand`/`.chrome-logo`/`.chrome-divider` CSS rules — copied verbatim from `dashboard.html` — to the four SSR mastheads that still rendered a text-only mark. The ISCC logo now renders on EVERY server-rendered surface, closing the front-of-queue human `critical`. No `.go` source changed; the `web.LogoPath` serve route already exists.

**Files changed:**
- `internal/certificate/cert.html`: wrapped the masthead `<div>` in `.chrome-brand`, prepended the `<img>` + `<span class="chrome-divider">`, added the 3 CSS rules after the `.chrome` block (left `.chrome-actions` + sub-copy verbatim).
- `internal/proofserve/browser.html`: same masthead + CSS edit.
- `internal/proofserve/record.html`: same masthead + CSS edit.
- `internal/proofserve/records.html`: same masthead + CSS edit.
- `internal/certificate/handler_test.go`: extended `TestCertificateKnownID`'s string-contains loop with the MANDATORY `src="/_ds/iscc-logo-black.png"` assertion (byte-equal to `web.LogoPath`).
- `internal/proofserve/browser_test.go`: added the same one-line assertion to `TestBrowserLinksTokensNoCDN`'s DS-shell-marker loop (symmetry).

**Verification:** `mise run check` → green (`go build ./...`, `go vet ./...`, `go test ./...`; all 25 packages `ok`). Per-criterion:
- [x] `gofmt -l .` (excl. `cauldron/`) — clean.
- [x] `go test -run TestCertificate ./internal/certificate` — PASS; the extended test asserts the served cert HTML contains `src="/_ds/iscc-logo-black.png"`.
- [x] `go test ./internal/proofserve` — PASS; the existing no-CDN body bans (`jsdelivr`/`http://`/`https://`/`cdn.`) stay green with the new same-origin `/_ds/` img src.
- [x] `grep -L 'iscc-logo-black.png'` over all four templates — prints nothing (all four reference the logo).
- [x] Mutation check — removing the `<img>` line from `cert.html` makes `TestCertificateKnownID` FAIL (`body missing "src=\"/_ds/iscc-logo-black.png\""`); restoring it PASSes. Assertion is non-vacuous.

**Next:** The `critical` is now fully closed — every SSR surface carries the shared chrome (logo + text mark + divider), satisfying target.md:148-151. The WASM milestone resumes: the tier-2 verifier caller into the dossier, then the standalone `monitor.iscc.codes` Independent Verification app. Optional KISS follow-up (still deferred, not required): the now six near-identical mastheads could be factored into a shared `html/template` chrome partial — a separate `advance`-call.

**Notes:**
- The `<img src>` literal is byte-equal to `web.LogoPath` in all four templates (templates can't read the Go const; the two test assertions are the regression guards). No Go-side template field introduced — the static literal is intentional, matching dashboard/dossier (learnings/web.md logo contract).
- Each file's existing `.chrome-sub` copy was left verbatim ("Evidence of record · ISCC-Hub network" in all four — same as dashboard); cert's `.chrome-actions` block untouched.
- Oracle/conformance gate is N/A: pure static-asset rendering, no signature/RFC-6962/Merkle/proof path touched.
- Scope discipline: 4 template edits (no `.go` source change) + 2 test-loop extensions. Nothing from `## Not In Scope` done — `web.go`/serve route, the instance-identity block / back-link parity / Checkpoint-Anchor columns, and the unrelated open `normal` issues (cert §5 digest binding, OTS `safeStamp`, registry ForceQuery, §4/§6, `safeIndex` test gap) were all left untouched.
- A visual (ADR-0012) pass over the four newly-touched surfaces was NOT run by `advance` (no live render performed here); `review` should screenshot a certificate, the log browser, a record, and a record list to confirm the masthead matches the `.dc.html` mockups.
