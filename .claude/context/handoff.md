## 2026-06-22 — Review of: Normalize Surface-C `readTarget` to return the parsed `u.href`, not the raw monitor string

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance changes one production line — `readTarget` (`internal/verifier/verifier.html:556`)
now returns the WHATWG-normalized `{ monitor: u.href, id: id }` instead of the raw `{ monitor: monitor, … }`
query string — plus an evergreen comment update and one mutation-provable markup test. An opaque-scheme form
(`https:example.com`) now flows downstream as `https://example.com/`, closing the only pure-code-closable
`normal`. Scope is exemplary: 2 files (1 production + 1 test), nothing from `## Not In Scope` touched, no
trust-root / dependency / SSR-layout change; reject branches and the loader fetch flow are byte-unchanged.

**Verification:**
- [x] `mise run check` — green (build + vet + `go test ./...`, all 28 packages ok).
- [x] `gofmt -l .` — empty (no formatting failure).
- [x] `go test -count=1 -run TestVerifier ./internal/verifier` — PASS (whole suite, incl. the unchanged
      `TestVerifierNoServerSideTarget`, `TestVerifierNoExternalCDN`, `TestVerifierStaticBodyAlwaysCarriesLoader`).
- [x] Mutation (independent) — reverting the production return to `{ monitor: monitor, id: id }` makes
      `TestVerifierReadTargetReturnsNormalizedURL` FAIL; restored byte-identical (`git diff` over the
      template clean), the test re-passes. The assertion is non-vacuous on both positive and negative checks.
- [x] No-CDN ban green and genuinely safe — `TestVerifierNoExternalCDN` passes; I probed the *rendered*
      body via an in-package throwaway test: it contains NEITHER `https://` NOR the new comment text
      (`html/template` strips the `//`-prefixed JS line comment at render, so the `https://example.com/`
      inside the docstring never reaches the served HTML). No static `http(s)://` literal was introduced
      (the only diff match is inside the stripped comment).
- [x] Oracle/conformance gate — N/A. `git diff --name-only HEAD~1..HEAD` over the trust-root globs
      (`internal/proof/`, `logclient/verify`, `didweb`, fork/shrink/equivocation/consistency) → empty.
      This is a JS-source normalization fix on a pure static HTML render (markup-golden by design; no
      go-test JS-execution gate exists).
- [x] Gate-circumvention scan over the unpushed range (`@{upstream}..HEAD`, 3 commits) — no `//nolint`,
      `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion in added lines.

**Issues found:** (none new). Resolved + deleted the `normal` "Surface-C `readTarget` accepts opaque-scheme
monitor forms" after independently mutation-verifying the close: the rendered body now carries
`return { monitor: u.href, id: id };` (not the raw string), and reverting it FAILs the guarding test.

**Codex second opinion:** Clean — "The change is narrowly scoped to returning the parsed URL from readTarget
and adds a regression assertion. I did not identify any introduced correctness, security, performance, or
maintainability issues." Notably, Codex's transcript shows it probed the URL-userinfo edge cases
(`https:example.com%40evil.com`) and still cleared the change. I independently reproduced those in node:
the userinfo-confusion case (`https://example.com@evil.com/x` → `u.host="evil.com"`) is INHERENT to
`new URL()` and IDENTICAL under the old raw-string path — not introduced by `u.href`, and harmless (wrong
host → honest `error` / a bundle that fails WASM re-verification, since `readTarget` is a usability guard,
not a trust boundary). No findings to triage.

**Visual check:** n/a — no SSR surface changed. The edit is a JS-source return value computed at runtime
from `location.search`, never displayed in the no-JS body; the named-region/affordance markup is
byte-unchanged (the unchanged `TestVerifierNoTargetBaselineIsHonest` still renders the same baseline).

**Next:** The remaining open `normal`s are no longer pure-code-closable in one package:
(1) the WASM-verifier **signature half** (no browser did:web checkpoint-signature check) is the
front-of-queue design-first / STOP-candidate — do a design pass before touching `verifier.html` or the WASM
core. (2) the certificate §6 per-record `· at` timestamp needs a store schema column on the `iscc_index`
projection + a follower-ingest write. (3) the `/` Checkpoint/Anchor data columns + config-driven instance
identity need a store/projection + config change. The WASM "published" half (Pages custom-domain) stays
human-blocked. Suggest define-next picks the §6 timestamp or the `/` projection (both store-scoped,
self-contained) over the signature half (which wants a STOP/design pass first).

**Notes:**
- Open count after this close: 0 critical / 4 normal / 10 low. DONE still requires 0 normal.
- Learnings: `verifier.md` collapsed the opaque-URL `normal` bullet into a `settled:` summary (net change,
  no growth — file ~92 lines / 14 bullets, within budget) and added the userinfo-confusion note (inherent to
  `new URL`, not introduced). The index gist row was updated to drop "filed `normal`". Package-local; nothing
  promoted to the always-loaded index (the rule it serves — "usability guard, not a trust boundary" — is
  already captured by the indexed "client re-verifies; the monitor is not in the trust path").
- 4 commits ahead of `origin/develop` after this review commit (update-state + define-next + advance +
  review). Pushing on PASS. The known `Pages` workflow failure on develop is the human-blocked custom-domain
  repo-settings step (a documented `normal`), not a code regression.
