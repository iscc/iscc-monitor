## 2026-06-22 — Review of: Gate Surface-C's live verification CLIENT-side so the static GitHub-Pages artifact works

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance moves the `?monitor=&id=` target resolution out of the Go handler and into the
always-emitted browser loader (`new URLSearchParams(location.search)` + a faithful JS port of
`parseTarget`'s validation), so `internal/verifier` is now a single static artifact rendered identically
for every request — which is what makes the documented GitHub-Pages deployment actually functional.
The diff is tight (2 non-test/doc files + 1 test + handoff), all gates are green, both claimed mutations
reproduce, and the change closes the filed `normal` "Surface-C gated on SERVER-side `.HasTarget`" issue
exactly as that issue prescribed. One genuine but non-blocking behavioral delta surfaced (Codex P2,
reviewer-confirmed): the JS URL guard is slightly more permissive than the Go original on opaque-scheme
inputs — filed `normal`, not a trust defect.

**Verification:**
- [x] `mise run check` (build + vet + test) — GREEN, all 27 packages ok.
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `GOOS=js GOARCH=wasm go build ./cmd/wasm` (WASM build gate) — OK (note: emits a stray `wasm`
  binary into cwd; not gitignored — I removed it; tooling-convenience gap, not introduced here).
- [x] `go test -count=1 -run TestVerifier ./internal/verifier` — all 9 PASS.
- [x] Static body ALWAYS carries the loader (`/_ds/wasm_exec.js`, `/_ds/verify.wasm`,
  `isccVerifyInclusion`, `URLSearchParams`, `location.search`) on a plain no-query GET — PASS.
- [x] No server-emitted data-island (`id="verify-target"` absent; no `?monitor=`/`?id=` reflected;
  query-bearing render byte-identical to baseline) — PASS.
- [x] No-JS baseline asserts no un-run verdict (displayed body keeps "not yet run" / "no verdict is
  claimed" / "Illustrative — what a real mismatch shows"; present-tense copy lives only in the JS) — PASS.
- [x] Three distinct verdict states (`error`/`failed`/`verified`); no-CDN body ban (`http://`/`https://`/
  `jsdelivr`/`cdn.` absent — the JS `"http:"`/`"https:"` protocol literals do NOT trip it); 405 non-GET — PASS.
- [x] Scope: only `handler.go` + `verifier.html` (2 non-test/doc) + the test + handoff; no Not-In-Scope
  path touched (`cmd/wasm`/`internal/proof/verify`/`.github/workflows`/`cmd/iscc-monitor` untouched);
  `verifier.Handler` still NOT mounted in `buildMux` — PASS.
- [x] `net/url`/`parseTarget`/`pageData`/`HasTarget` fully removed (remaining grep hits are prose/comments) — PASS.
- [x] Mutations reproduced by reviewer: (A) reverting the run-label to "verifying…" in static markup FAILS
  `TestVerifierNoTargetBaselineIsHonest`; (B) replacing the `URLSearchParams(location.search)` read FAILS
  `TestVerifierStaticBodyAlwaysCarriesLoader`; both restored → green. Gates are non-vacuous.
- [x] Gate-integrity scan over the 3 unpushed commits (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion in the production diff (grep hits were handoff/learnings prose).
- [x] Oracle/conformance gate — N/A (pure static HTML; no signature/RFC-6962/Merkle/did:web/fsck/proof path).

**Issues found:**
- (Codex P2, confirmed → filed `normal`) `readTarget` (`verifier.html:550-553`) accepts opaque-scheme
  monitor forms (`https:example.com`) that the Go `parseTarget` rejected, returning the raw string rather
  than the normalized `u.href`. Not a trust/security defect — the browser resolves the raw fetch to the
  SAME host the parser reported (no wrong-host/SSRF), WASM re-verifies the bundle regardless, and an
  unreachable host yields the documented honest `error` render. Filed for a later advance; does not block.
- Resolved + deleted from `issues.md`: "Surface-C live wiring is gated on SERVER-side `.HasTarget`" — this
  advance is its exact prescribed fix (read the target client-side via `URLSearchParams`/`location.search`,
  always emit the loader); reviewer-verified the closed-issue Verify criterion is met.

**Codex second opinion:** Codex ran (slow — ~6 min) and produced ONE finding, [P2] "Fail closed for
malformed monitor URLs" at `verifier.html:550-553`. Triage: CONFIRMED the underlying behavioral fact
(reproduced in node: `new URL("https:example.com")` → `protocol=https:`, `host=example.com`, so the guard
passes and returns the raw string; cross-checked Go `net/url.Parse` → `Host=""`, so the original
`parseTarget` rejected it — the JS port is strictly more permissive). But Codex's "fetch the WRONG URL"
framing is REFUTED by its own probe log: the browser resolves the raw fetch to exactly the validated host
(`https:example.com` → `https://example.com/inclusion/…`), and the loader re-verifies the bundle in WASM
(the monitor is never in the trust path), so the worst case is an honest `error` render — never a false
verdict. Filed as `normal` (not a trust defect, not exploitable, not a regression in any harmful sense),
fix = return `u.href` not the raw `monitor`.

**Visual check:** SSR surface (`internal/verifier`) screenshotted via agent-browser 0.29.0 (Chrome
available; built an in-repo preview harness mounting `verifier.Handler` + `web.Handler`, removed after).
The no-target baseline renders all named regions correctly (masthead + logo, eyebrow, "Re-run the proof
yourself." head, the `.codes` independence statement, the Verification Record block "not yet run", the
five step rows). Deltas vs `.claude/design/ISCC Monitor - Independent Verification.dc.html`: the chrome
identity (`monitor.iscc.codes` vs the mockup's `monitor.iscc.id`) and the absence of a live "Verified"
panel on the no-JS baseline are BOTH documented-correct divergences (`.codes ↔ .id` invariant; post-run
states are loader-driven) — no new visual delta filed. (The mockup is a Vue template that renders raw
`{{ }}` bindings statically, so only region-level parity is comparable.)

**Next:** Surface-C's deploy sub-step is now unblocked — generate the static `index.html` (render
`verifier.Handler` once to a file) + the `/_ds/` assets and publish to GitHub Pages at `monitor.iscc.codes`
via a `.github/workflows/*`. When that lands, fold in the new `normal` `readTarget` normalization fix
(return `u.href`) and the still-open WASM-verifier-scope honesty issue (the core proves inclusion math only
— no checkpoint-signature / did:web-key / id-binding check; the step copy still lists a did:web step the
core does not run).

**Notes:**
- The advance handoff flagged a "design reconciliation" tension (next.md's raw-whole-body ban on the
  present-tense `does not match` copy is unsatisfiable once the loader is always emitted). Reviewer agrees
  with the resolution: scoping the honesty assertion to the DISPLAYED (script-stripped) body via
  `displayedBody` is the correct honesty boundary — what a no-JS reader SEES, not what sits in a
  never-displayed `<script>`. Mutation A proves it non-vacuous. Not a deviation worth STOP.
- The `readTarget` opaque-URL permissiveness is the residual usability-guard imperfection; it is by design
  NOT a trust boundary (WASM re-verification is), so it cannot produce a false `verified`. Watch it only if
  a deploy harness adds a JS-level test for the guard.
- 0 critical / 10 normal open issues after this iteration (one closed, one added). CI green expected at this
  commit (same gate set as `mise run check`, no `cauldron/` in tree).
