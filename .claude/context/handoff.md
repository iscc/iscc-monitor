## 2026-06-22 — Review of: Make the certificate Tier-2 honesty header honest on the no-JS baseline

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance reworded a single sentence of the `{{if .HasBundle}}` honesty header in
`cert.html:484` so the no-JS baseline no longer asserts a present-tense "This browser re-verifies the
proof below" — the always-true offline path is now unconditional and the browser re-check is explicitly
conditional on JavaScript, matching the register of the unchanged `#tier2-result` panel (`:501`). The
no-JS subtest gained a non-vacuous negative+positive assertion pair. Scope is exemplary: one template
sentence + one test block, nothing from `## Not In Scope` touched, no trust-root / SSR-output-layout /
dependency change; the mutation reproduced independently.

**Verification:**
- [x] `mise run check` — green (build + vet + `go test ./...`, all 28 packages ok).
- [x] `gofmt -l .` — empty (no formatting failure).
- [x] `go test -count=1 -run TestCertificateRendersWasmVerifier ./internal/certificate` — PASS.
- [x] Mutation (independent) — reverting ONLY the `:484` sentence back to "This browser re-verifies the
      proof below for you, and you can download the bundle and re-verify it offline." makes
      `TestCertificateRendersWasmVerifier` FAIL (the negative assertion trips). Restored byte-identical;
      `git diff` over the template clean.
- [x] `#tier2-result` panel (`cert.html:501`) byte-unchanged — confirmed via `git diff HEAD~1..HEAD`
      (the panel line does not appear in the diff). Header now says "below", panel says "above" — the
      two regions are distinct and both honest; neither asserts a verdict ran.
- [x] Old overstatement fully removed — `grep "This browser re-verifies the proof below"
      internal/certificate/cert.html` → 0.
- [x] Oracle/conformance gate — N/A. No signature / RFC-6962 / Merkle / `proof` / `didweb` / `logclient`
      / fork-shrink-equivocation file touched (`git diff --name-only` over the trust-root globs → empty);
      this is template prose + a string assertion. The §3 re-verification gate that sets `HasBundle` is
      untouched (no `data.*` field, store read, or template variable added).
- [x] Gate-circumvention scan over the unpushed range (`@{upstream}..HEAD`, 3 commits) — no `//nolint`,
      `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion in added Go/template lines.
      The lone `//go:build` match is prose inside `state.md` (describing the existing WASM tag), not a
      new exclusion in source. Only Go/template files changed are `cert.html` + `handler_test.go`.

**Issues found:** (none new). Resolved + deleted the `normal` "Certificate tier-2 honesty header
overstates 'This browser re-verifies' on the no-JS baseline" after independently mutation-verifying the
close (the old copy is gone, both regions honest, reverting the copy FAILs the guarding test).

**Codex second opinion:** Clean — "The change is limited to copy in the certificate template and a
targeted regression assertion. I did not identify any introduced correctness, security, performance, or
maintainability issues." Corroborates the reviewer's verification; no findings to triage.

**Visual check:** skipped — Chrome/Chromium not in PATH (agent-browser is installed but has no browser
to drive in this headless container). Best-effort per ADR-0012 graceful degradation; not a NEEDS_WORK.
The change is a single sentence of honesty prose with no layout/region change — substituted a textual
render check: the header (:484) and panel (:501) phrasings coexist, are distinct ("below" vs "above"),
and the old overstatement is fully absent. The named-region/affordance bar is unaffected (every clause
marker + the Tier 1/Tier 2/Download-bundle affordances still render before the loader, per the
unchanged no-JS subtest). Visual fidelity's hard gate remains the human M-UI exit sign-off.

**Next:** Continue draining code-closable `normal`s. The remaining certificate-adjacent ones are
store-touching and larger: (1) the §6 `· at` per-record timestamp needs a store schema column on
`RecordRow` + a follower-ingest write (define carefully); (2) the `/` Checkpoint/Anchor data columns +
config-driven instance identity also need a store/projection + config change. The front-of-queue
WASM-verifier signature half (no browser did:web checkpoint-signature check, `normal`) stays
design-first / a STOP-candidate — do a design pass before touching `verifier.html` or the WASM core.

**Notes:**
- 4 commits ahead of `origin/develop` after this review commit (update-state + define-next + advance +
  review). Pushing on PASS. The known `Pages` workflow failure on develop is the human-blocked
  custom-domain repo-settings step (a documented `normal`), not a code regression.
- Learnings: `certificate.md` gained a no-JS two-tier-honesty copy rule (header OFFERS paths only; the
  `#tier2-result` panel is the SOLE asserter of a verdict; "below" vs "above" keeps the header assertion
  non-vacuous) — package-local, subsumed by the indexed "gate a rendered ✓ on a re-VERIFICATION" rule,
  so NOT promoted. Net change +5 lines: collapsed the settled §5 digest-binding sub-note to one line to
  honor the rotation budget; file at 162 lines / 15 bullets (marginally over the ~150 soft cap, within
  the ~40 bullet budget — the new note is load-bearing copy discipline).
- Open count after this close: 0 critical / 5 normal / 10 low. DONE still requires 0 normal.
