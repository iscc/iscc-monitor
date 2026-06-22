## 2026-06-22 — Review of: Surface-C skeleton — render the standalone Independent Verification page (`internal/verifier`)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** Advance `c579589` adds a new pure-stdlib `internal/verifier` leaf — a static, no-JS,
no-CDN, self-hosted SSR skeleton (Surface C) rendering the named regions of the Independent
Verification mockup, with a golden HTTP-seam test. The work is tightly on-scope (3 new files: handler +
template + test, zero existing-file source change, `go.mod`/`go.sum` untouched, deliberately unmounted),
all gates pass, the mutation proof holds non-vacuous, Codex is clean, and the visual pass confirms strong
mockup parity. One `normal` honesty gap was found (the mismatch alert asserts an un-run verdict) and
filed; it does not block progress (the live-wiring sub-step is its natural fix point).

**Verification:**
- [x] `mise run check` — green (build + vet + test; all 26 packages `ok`, `internal/verifier` included).
- [x] `gofmt -l .` (excl. `cauldron/`) — clean (zero files listed).
- [x] `go test ./internal/verifier` — PASS (all 5 tests: named-regions, independence-statement,
  guided-mismatch-alert, no-external-CDN, non-GET-405).
- [x] 200 `text/html; charset=utf-8`; "Independent verification" eyebrow; independence statement names
  both `monitor.iscc.codes` + `monitor.iscc.id` + "not in the trust path"; all five verification-record
  step labels; "Mismatch — possible split view" + "do not discard either"; `src="/_ds/iscc-logo-black.png"`;
  `/_ds/verify.wasm` + `/_ds/wasm_exec.js` referenced — all present (grep-confirmed in rendered body).
- [x] No-CDN body ban — `jsdelivr`/`http://`/`https://`/`cdn.` all 0 hits in `verifier.html`; the lone
  `rgba(...)` is an inline CSS fallback, not a CDN.
- [x] All `var(--*)` DS tokens referenced by the template resolve in `web/tokens.css` (0 missing).
- [x] The five `/_ds/...` template literals cross-checked byte-equal against the `web.*` consts
  (`TokensPath`/`FontsCSSPath`/`WasmVerifyPath`/`WasmExecPath`/`LogoPath`) — all in sync.
- [x] Mutation-proof reproduced: replacing the `mismatch-title` "Mismatch — possible split view" copy
  makes `TestVerifierGuidedMismatchAlert` FAIL; restored byte-identical → PASS.
- [x] Gate-circumvention scan over the 3 unpushed commits (`@{upstream}..HEAD`) — no
  `//nolint`/`t.Skip`/`SkipNow`/swallowed-error/build-tag/deleted-assertion patterns; diff is purely
  additive (new package + template + test).
- [x] Scope discipline — 3 new files (handler + template + test), zero existing-file source change,
  handler deliberately NOT mounted in `buildMux` per Not-In-Scope; no `?monitor=` parse, no WASM run, no
  Pages workflow, no dossier tier-2 caller, no open `normal` issue touched.
- [x] Purity — leaf imports stdlib only (`bytes`/`embed`/`html/template`/`net/http`); does not import
  `internal/store`/`internal/metrics`/network; oracle/conformance gate N/A (pure static HTML render).

**Issues found:** One `normal`, filed (does not block): the Surface-C split-view mismatch alert renders
UNCONDITIONALLY in the present tense ("Your (size, root) does NOT match the monitor's mirrored tree …")
even though the skeleton runs no comparison — the page asserts a negative verdict it never computed. The
verification-record block above it IS honest ("not yet run" / "until it runs, no verdict is claimed"), so
the two regions disagree. This is the inverse of the open certificate no-JS honesty bug (over-claims a
negative instead of a positive verdict) and falls under the always-loaded "gate a rendered verdict on a
re-VERIFICATION, not a static render" rule. The step's Verify criteria are still met (the guided alert
exists, is golden-tested, is never a dead error); the live `?monitor=` wiring that makes the alert
conditional is the next sub-step, its natural fix point.

**Codex second opinion:** Clean — "The new verifier skeleton is isolated, compiles, and matches the
stated scope without breaking existing behavior. I found no actionable correctness issues in the HEAD
diff." Log confirms Codex genuinely read the template (not an error masquerading as a clean verdict). No
findings to triage; matches my own review. (My one honesty finding is independent of Codex — it surfaced
in the visual + rendered-body pass, not the diff alone.)

**Visual check:** Performed (ADR-0012). `agent-browser` 0.29.0; built a throwaway preview harness mounting
`verifier.Handler()` + `web.Handler()` (the page is deliberately unmounted in the binary), served on
`:41465`, and screenshotted the live page (top + scrolled bottom) against the Independent Verification
`.dc.html` mockup. Strong parity: chrome (logo + 1px divider + "TRUST & TRANSPARENCY MONITOR" mark),
`← Certificate` breadcrumb, eyebrow, "Re-run the proof yourself." head, independence statement,
five-step verification-record block, split-view (size, root) form, and the guided mismatch alert all
render and match the mockup layout. One INTENTIONAL/correct chrome divergence: the mockup's identity reads
`monitor.iscc.id / instance operated by ISCC Foundation`, the live page reads `monitor.iscc.codes /
independent verifier app · audits any monitor instance` — Surface C IS the verifier app, not an instance,
so the live page is more correct (the `.codes ↔ .id` distinction, Handoff invariant 5). The one visual
delta that became the filed issue: the live skeleton shows the mismatch alert as always-visible static
markup (the mockup gates it behind a `Simulate split view` JS button) — acceptable for a no-JS skeleton,
but the present-tense copy asserts an un-run verdict (the honesty issue above). Preview harness +
screenshots cleaned up; tree clean.

**Next:** The deferred Surface-C live-wiring sub-step (the larger remaining WASM Verify criterion): parse
`?monitor=<url>` (+ `?id=`), embed the `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader + JSON data-island
(mirror the certificate tier-2 pattern, `cert.html` + `cmd-wasm.md`), fetch the target instance's proof
bundle client-side, run `isccVerifyInclusion`, and gate the rendered ✓ / mismatch-alert on the genuine
re-VERIFICATION — which also closes the honesty issue filed this iteration (the alert becomes conditional
on a real verdict). Keep the three render states distinct (`error` broken-input vs `failed`
negative-verdict/split-view vs `verified`). The sibling open WASM sub-step (the dossier tier-2 caller) is
still open. After both, the GitHub Pages / `monitor.iscc.codes` deploy workflow is the final Surface-C
piece. `define-next` weighs these plus the open `normal` issues against the state→target gap.

**Notes:**
- DONE is not reachable: the WASM milestone (Surface-C live wiring + dossier tier-2 caller +
  `monitor.iscc.codes` deploy) is incomplete and multiple `normal` issues are open. No `critical` open.
  Loop CONTINUE.
- New learnings detail file `learnings/verifier.md` created + pointer row added to the index (records the
  unmounted/different-origin posture, the `/_ds/` literal-sync convention, the correct `.codes` chrome
  divergence from the mockup, the no-CDN ban, and the honesty gap).
- The `/_ds/` paths are template literals synced to `web.*` consts by comment, not by import — same as
  dashboard/dossier. The golden test pins the literals, so a future `web.*` const rename is caught only
  if the test literal is updated too. Watch this when the WASM paths change.
- Pushing to `origin/develop` on this PASS_WITH_NOTES.
