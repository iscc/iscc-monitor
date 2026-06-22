## 2026-06-22 — Review of: Render the self-hosted ISCC logo in the remaining FOUR SSR mastheads

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Advance `6a442b4` adds the identical one-line `<img class="chrome-logo"
src="/_ds/iscc-logo-black.png" alt="ISCC">` + `.chrome-divider` (inside a `.chrome-brand` flex wrapper)
plus the three `.chrome-brand`/`.chrome-logo`/`.chrome-divider` CSS rules — copied byte-verbatim from
`dashboard.html` — to the four SSR mastheads that still rendered a text-only mark
(`certificate/cert.html`, `proofserve/{browser,record,records}.html`), extending the certificate and
browser handler tests to assert the src. The ISCC logo now renders on ALL SIX server-rendered surfaces,
fully closing the front-of-queue human `critical`. The work is tight and exactly on-scope (4 template
edits + 2 test-loop additions; zero `.go` source change; `go.mod`/`go.sum` untouched), all gates pass,
both new assertions are mutation-proven non-vacuous, and the ADR-0012 visual pass confirms the masthead
matches the mockup chrome with no remaining "no logo" delta.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 25 packages `ok`).
- [x] `gofmt -l .` (excl. `cauldron/`) — clean (zero files listed).
- [x] `go test -run TestCertificate ./internal/certificate` — PASS; the extended `TestCertificateKnownID`
  asserts the served cert HTML contains `src="/_ds/iscc-logo-black.png"`.
- [x] `go test ./internal/proofserve` — PASS; the existing no-CDN body bans (`jsdelivr`/`http://`/`https://`/
  `cdn.`) stay green with the new same-origin `/_ds/` img src; `TestBrowserLinksTokensNoCDN` now also asserts it.
- [x] `grep -L 'iscc-logo-black.png'` over all four templates — prints nothing (all four reference the logo).
- [x] Mutation check (certificate) — dropping the `<img>` line from `cert.html` makes `TestCertificateKnownID`
  FAIL; restoring → PASS; cert.html restored byte-identical.
- [x] Mutation check (proofserve) — dropping the `<img>` line from `browser.html` makes
  `TestBrowserLinksTokensNoCDN` FAIL; restoring → PASS. Both new assertions are non-vacuous.
- [x] Masthead markup + CSS byte-equal to the `dashboard.html` canonical block (reviewer-compared
  lines 40-54 + 328-336); `<img src>` literal byte-equal to `web.LogoPath`.
- [x] Oracle/conformance gate — N/A (pure static-asset rendering; no signature/RFC-6962/Merkle/proof path touched).
- [x] Gate-circumvention scan over unpushed range (`origin/develop..HEAD`, 3 commits) — no
  `//nolint`/`t.Skip`/swallowed-error/build-tag/deleted-assertion patterns in code (two grep hits are in
  handoff prose only); diff is purely additive markup + CSS + two test assertions.
- [x] Scope discipline — 4 template edits + 2 test-loop additions, zero `.go` source change, matching
  `next.md`; nothing from `## Not In Scope` done (serve route, instance-identity block, Checkpoint/Anchor
  columns, the unrelated open `normal` issues all left untouched).

**Issues found:** (none new from this diff.) Resolved + deleted: the front-of-queue human `critical` "Add
the ISCC logo to the remaining FOUR mastheads" — all four served mastheads now carry
`<img src="/_ds/iscc-logo-black.png">`, the certificate + browser handler tests assert it (mutation-proven),
`mise run check` is green, and the visual pass files no remaining "no logo" delta on any SSR surface. The
"`/` realm-index sub-region deltas" issue's sub-item (1) "No logo" was also marked CLOSED (the logo renders
on all six surfaces); that issue's remaining scope is items 2-4 (instance-identity copy, Checkpoint/Anchor
columns, recent-declarers footer). No open `critical` remains.

**Codex second opinion:** Clean — "The change consistently adds the self-hosted logo markup and matching
CSS to the remaining mastheads without altering routing or server behavior. The referenced asset is already
served under the shared /_ds/ path, and the test suite passes." No findings to triage; matches my own review.

**Visual check:** Performed (ADR-0012). `agent-browser` 0.29.0 launches its bundled browser. Built the
binary, ran a live testnet instance (`0.0.0.0:41464`, 30s poll), and screenshotted the live log browser
(`/sb0.iscc.id/log/`), the record-list (`/sb0.iscc.id/log/records`), and the certificate page (cannot-certify
honest state) against the Log-Browser + Certificate `.dc.html` mockups. All three live mastheads now render
the ISCC logo silhouette + 1px divider + "TRUST & TRANSPARENCY MONITOR" / "Evidence of record · ISCC-Hub
network" — matching the mockup's left-side chrome exactly; the prior text-only delta is gone on every touched
surface. (The single-record page was not screenshotted as rich state — the live testnet index is cold-start
empty, so no leaf exists; its masthead is the identical `record.html` block already verified by the
mutation-proven cross-surface CSS/markup equality.) No NEW visual delta filed: the remaining index
sub-region deltas (instance-identity copy, Checkpoint/Anchor columns, recent-declarers footer) are already
tracked in the existing `normal` "/ realm-index sub-region deltas" issue (items 2-4), out of scope for this
critical.

**Next:** The lone `critical` is fully closed — every SSR surface carries the shared chrome (logo + text mark
+ divider), satisfying target.md:148-151. The WASM milestone resumes: wire the tier-2 in-browser verifier
caller into the hub dossier (mirroring the certificate's data-island + `/_ds/` loader pattern), then build the
standalone monitor-agnostic `monitor.iscc.codes` Independent Verification app (takes `?monitor=<url>`,
verifies that instance client-side). The open `normal` issues — certificate §5 digest binding, OTS
`safeStamp` panic/timeout guard, registry `hubDomain` ForceQuery, §4/bundle `host:port` DID, §6 timestamp,
the `safeIndex` test gap, the tier-2 honesty copy, the `/` sub-region deltas — are weighed against the
state→target gap by `define-next`. Optional KISS follow-up (still deferred, not required): factor the now
six near-identical mastheads into a shared `html/template` chrome partial.

**Notes:**
- The `<img src>` literal is byte-equal to `web.LogoPath` in all four templates (templates can't read the Go
  const; the cert + browser test assertions are the regression guards). No Go-side template field — the
  static literal is intentional, matching dashboard/dossier (learnings/web.md logo contract, now updated to
  record all-six-mastheads-covered).
- `internal/web` purity unaffected — no Go change; `contentTypePNG` stays a string literal (no `image/*`
  import), the leaf remains the WASM-green pure-stdlib closure.
- DONE is not reachable: no `critical` remains, but the WASM milestone (tier-2 dossier caller +
  `monitor.iscc.codes` app) is incomplete and 8 `normal` issues are open. Loop CONTINUE.
- Pushing to `origin/develop` on this PASS.
