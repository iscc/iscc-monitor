## 2026-06-22 — Review of: Close the CDN-free gate hole — narrow `stripLineComments` so protocol-relative `//cdn.` URLs trip `noExternalCDN`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance narrowed the load-bearing `noExternalCDN` test helper's `stripLineComments`
predicate (`internal/web/web_test.go:57`) from the over-broad `:`-only guard to a comment-context-only
rule — `//` is a comment ONLY at line-start or when preceded by whitespace — and added a focused,
mutation-proven regression test (`TestNoExternalCDNProtocolRelative`). The quoted-delimiter over-strip
named in the closed issue (`src="//cdn..."`, `url("//cdn...")`) is genuinely fixed: those forms now
survive the strip and trip the ban, while the byte-verbatim `wasm_exec.js` comment URL stays suppressed.
Scope is exactly the one test-only file the next.md named; all gates green; the gate is strictly stronger
than before. Codex raised one residual narrower latent case (whitespace-prefixed protocol-relative URLs),
which I confirmed but filed as `low` — it is pre-existing (the old guard stripped it too), not a
regression, not the form this step targeted, and triggered by no served asset.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test`, all 25 packages `ok`.
- [x] `gofmt -l .` empty (clean).
- [x] `go test -count=1 -run TestWasmExec ./internal/web` → PASS (byte-verbatim `wasm_exec.js` still
  passes `noExternalCDN`; its `// ` comment URL stays suppressed).
- [x] `go test -count=1 -run TestNoExternalCDN ./internal/web` → PASS (the new regression test).
- [x] Assertion: `stripLineComments(<script src="//cdn.jsdelivr.net/x.js">)` retains `cdn.` + `jsdelivr`
  (host survives the strip, re-arming the ban). Confirmed by the test.
- [x] Assertion: `stripLineComments("\t// (… https://github.com/golang/go/issues/28975)")` drops
  `https://` (genuine comment URL still stripped — the test uses the exact `wasm_exec.js:288` bytes).
- [x] Mutation check: reverted the predicate to the `:`-only guard → `TestNoExternalCDNProtocolRelative`
  FAILS on all three protocol-relative assertions (cdn./jsdelivr/url() host stripped); restored
  (byte-identical to HEAD, gofmt-clean). The gate hole is genuinely closed and the test is non-vacuous.
- [x] Strict-improvement check: under the OLD `:`-only guard the quoted form was ALSO stripped, so this
  change introduces ZERO new hole and closes the quoted-delimiter one — it tightens, never loosens.
- [x] Gate-circumvention scan over all 7 unpushed commits (`git diff origin/develop..HEAD -- '*.go'`):
  no `//nolint`, `t.Skip`/`SkipNow`, build-tag exclusions, swallowed errors, or deleted assertions
  (the only source change is the predicate narrowing + the added test).
- Oracle/conformance gate: N/A — test-helper change; no signature/RFC-6962/Merkle/did:web/fsck/proof code.

**Issues found:** ONE residual (filed `low`, non-blocking) — Codex's whitespace-prefixed case below.
Deleted the resolved `normal` issue ("`stripLineComments` over-strips protocol-relative CDN URLs") after
verifying its prescribed fix landed and its verify (ban fires for `src="//cdn..."`; reverting regresses)
is satisfied by `TestNoExternalCDNProtocolRelative`.

**Codex second opinion:** One [P2] — "Do not strip whitespace-prefixed CDN URLs" (`web_test.go:57`): the
predicate still treats a `//` preceded by space/tab as a comment, so `<script src = //cdn.jsdelivr.net/x.js>`
and CSS `url( //cdn.example/x.woff2)` are truncated before `cdn.`/`jsdelivr`, so the ban misses them.
**CONFIRMED real** by reviewer probe (both forms → `cdn.present=false`) — but triaged as `low`, NOT
blocking: (1) it is **pre-existing**, not a regression — the prior `:`-only guard stripped these exact
forms too (reviewer-verified); (2) it is strictly NARROWER than and orthogonal to the quoted-delimiter
form this step's goal targeted (which is now fully closed); (3) NO served asset (tokens.css / fonts.css /
byte-verbatim `wasm_exec.js`) uses a whitespace-prefixed protocol-relative URL, so the hole is latent;
(4) the `<script src = //…>` form is invalid HTML for `src`, so only the rare CSS `url( //… )` form is
genuinely loadable. Filed as a `low` issue with a tokenizer-grade fix prescribed (the per-delimiter
blocklist keeps losing edge forms — treat `//` as a comment only outside a quoted-string / `url(...)`
token). No other findings; Codex agreed the quoted-case fix is correct.

**Visual check:** n/a — no SSR surface changed. This is a test-only change to `internal/web/web_test.go`;
no `internal/dashboard|dossier|web|certificate` template or rendered HTML was touched.

**Next:** The CDN-free gate hole (the quoted-delimiter form) is closed and the WASM-loader sub-step is
re-greened to PASS-able. Resume the WASM tier-2 progression: build + serve the verifier `.wasm` (a
`mise run build:wasm` task + ADR-0003 reproducible-build / published-hash / SRI pin), then the `cert.html`
`<script>` loader + record-bytes emission — the natural first real caller where the open `normal`
`js.Value.Int()` truncation hardening belongs. The residual whitespace-prefixed CDN strip (`low`) and the
5 other standing `normal` issues each wait for a step that edits their own lines.

**Notes:**
- Scope clean: exactly one test-only file touched (`internal/web/web_test.go`) plus the context files.
  Nothing from `## Not In Scope` was done — `web.go`, the byte-verbatim `wasm_exec.js`, and the ban list
  are all untouched (the *strip* was tightened, not the *ban*); no `.wasm`, no `build:wasm`, no
  `cert.html` `<script>`, no other open issues.
- Good fidelity touch: the regression test's comment-line fixture is the EXACT byte string from
  `wasm_exec.js:288` (the real Go-issue-tracker comment), so it pins ground-truth bytes, not a synthetic
  approximation.
- 6 `normal` + 9 `low` issues remain open (all latent / non-blocking, each waiting for a step that edits
  its own lines), and no `critical` is open — so the Loop is CONTINUE, not DONE (DONE needs no open
  normal/critical). The §3 certificate fail-closed critical was closed earlier (commit 96f6ed9).
- Push: verdict is PASS_WITH_NOTES and `origin/develop` is the upstream; pushing the working branch.
