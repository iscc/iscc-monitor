## 2026-06-23 — Review of: Stop the verifier app from claiming an un-run did:web signature check

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance does exactly what `next.md` asked: the Surface-C verifier page
(`monitor.iscc.codes`) no longer lists "Check the signature against the hub's did:web key" as a step it
runs, and its tier-2 `verified` verdict + in-progress copy no longer claim the browser re-verified a
"hub-signed checkpoint root" — both now assert only the RFC-6962 inclusion + id-binding the WASM
actually runs, pointing signature trust at server-side certificate §4. Pure static-template + golden-test
edit (2 files, ≤3 budget); no verifier core / WASM / proof code touched. I independently re-verified the
WASM scope, mutation-proved the new honesty test both ways, ran every gate green, and did a live visual
pass — quality is solid.

**Verification:**
- [x] `mise run check` green — build + vet + test across all 30 packages, all `ok`.
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -count=1 -run TestVerifier ./internal/verifier` — ok (updated `TestVerifierRendersNamedRegions`).
- [x] NEW `TestVerifierDoesNotClaimSignatureCheck` passes; **I re-mutated both halves myself** — (A) re-add
  the did:web step → `TestVerifierDoesNotClaimSignatureCheck` + `TestVerifierRendersNamedRegions` FAIL;
  (B) restore "hub-signed checkpoint root" to the `verified` verdict → `TestVerifierDoesNotClaimSignatureCheck`
  FAILs at handler_test.go:153. Reverted → green. Test is non-vacuous on both halves.
- [x] `grep -c "Check the signature against the hub's did:web key" internal/verifier/verifier.html` → 0.
- [x] `setVerdict("verified", …)` line carries no `hub-signed` substring (grep + test).
- [x] No-CDN ban holds — `TestVerifierNoExternalCDN` green; added lines contain no `http://`/`https://`/
  `cdn.`/`jsdelivr` literal (the bare text "did:web" with no scheme is allowed and present in a negation).
- [x] WASM-scope claim independently confirmed: `isccVerifyInclusion` → `verifyadapter.VerifyJSON`
  (`verify.VerifyInclusion`) + `RecordCommitsID` checks ONLY inclusion math + id-binding; no signature /
  did:web path exists in `cmd/wasm` or `internal/proof/verify`. The new copy is truthful.
- [x] Quality-gate-integrity scan over unpushed commits — no `//nolint`, `t.Skip`, build-tag exclusion,
  swallowed error, or deleted assertion. The diff only swaps prose + ADDS a mutation-proven test.
- [x] The two surviving "hub-signed" strings (verifier.html:500 `#mismatch-body`, :579 comment) are the
  user's-own-evidence / descriptive-text ones `next.md` explicitly said to leave — verified in context.

**Issues found:** (none) — no new issue filed. Updated the open `normal` verifier-signature issue to
record that its copy-honesty interim half is now CLOSED, narrowing it to the design-blocked
in-browser-signature-verification remainder only.

**Codex second opinion:** unavailable — the `codex review --commit HEAD -c sandbox_mode="danger-full-access"
-c approval_policy="never"` launch was denied by the Claude Code auto-mode classifier ("creates an
autonomous agent loop with no approval gates"), so the truncating `>` redirect never ran. The
`/tmp/codex-review.txt` on disk is STALE (it describes the prior iteration's record-list "threads the
domain and identity" work, not this verifier copy change) — treated as unavailable, NOT as a clean
verdict. No second opinion this cycle; graceful degradation applied, loop not blocked.

**Visual check:** PASS — `agent-browser` is available (bundles its own Chromium; no system Chrome). Built a
throwaway in-module harness serving `verifier.Handler` + `web.Handler()`, screenshotted the live page
headlessly, and `Read` both the masthead and the scrolled step-list region. The page renders cleanly: the
`monitor.iscc.codes / independent verifier app` chrome + `← Certificate` breadcrumb intact; the
VERIFICATION RECORD list now shows "Rebuild the root and match the committed checkpoint root" + "Confirm
the record commits the requested ISCC-ID" (the false did:web step is gone); the illustrative split-view
panel correctly KEEPS "hub-signed root" in its muted/dashed user's-own-evidence context. No layout
breakage, no visual delta introduced — this was a copy-only change to existing regions, no new issue to
file. Harness removed; tree clean.

**Next:** This closes the copy-honesty half of the verifier-signature `normal`; the remaining half (actual
in-browser did:web resolution + checkpoint-signature verification) is DESIGN-BLOCKED and should not be
re-attempted in code without a design pass. With the lone `critical` human-blocked (M-UI exit sign-off
only) and no other open `normal` that is code-closeable without design input, the next `define-next` is
running low on autonomous code work. Candidate small autonomous items if needed: prune the stale/CLOSED
sub-items in the `/` realm-index `normal` (all four sub-items closed) and in the dossier/record-list
critical's narrative (every code half landed); or pick up one of the `low` locality-deepening refactors
(masthead-identity / overlay-precedence / note-schema-URI consolidation) IF a future slice naturally
touches those files — but lows are loop-skipped, so prefer pruning. Watch for the loop spinning on
cosmetic chrome (auto-memory: loop-stalls-on-human-blocked-done).

**Notes:**
- Oracle/conformance gate is **N/A** this iteration — no signature, RFC-6962, Merkle, did:web, fsck, or
  proof code touched; `internal/proof/verify` + `cmd/wasm` unchanged; the template parses at init so a
  malformed edit fails the build, not a request.
- Scope discipline clean: exactly 2 code files (1 prod template + 1 test), nothing from `## Not In Scope`
  touched (cert.html, openapi, the WASM artifact, `cmd/wasm`, `proof/verify`, the human-blocked critical
  all untouched).
- Codex tooling note for future cycles: the `danger-full-access`/`never` codex invocation is being denied
  by the auto-mode classifier in this environment — the second opinion has been unavailable for several
  cycles. Not a loop blocker (graceful degradation), but the human may want to add a Bash allow-rule for
  `codex review` if the second opinion is wanted.
- Learnings: added one settled bullet to `learnings/verifier.md` (step-list honesty CLOSED; the rule binds
  the STEP LIST, not just the rendered ✓). Detail file is ~104 lines, under the rotation budget.
