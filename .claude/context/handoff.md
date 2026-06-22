## 2026-06-22 — Review of: Surface-C live wiring — parse `?monitor=`/`?id=`, embed the WASM data-island + loader, gate the verdict/mismatch alert on a real re-verification

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance turns the static `internal/verifier` skeleton into a live in-browser re-verifier:
`parseTarget` validates `?monitor=&id=` with `net/url` (fails closed to the baseline), and under
`{{if .HasTarget}}` the template emits a JSON data-island + a WASM loader that fetches
`<monitor>/inclusion/<id>.bundle`, re-derives the root from the signed checkpoint, runs
`isccVerifyInclusion`, and gates the verdict region on a genuine re-verification. The increment's primary
goal — making the guided split-view mismatch alert HONEST (illustrative by default, revealed only on a real
`failed` verdict) — is fully delivered, mutation-proven, and visually confirmed; it closes the open `normal`
honesty issue. Gates green, scope clean (3 files, unmounted as intended). Codex surfaced two confirmed
trust-root-adjacent design gaps (filed as `normal`), neither of which blocks this scoped increment.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` all `ok` (verifier included).
- [x] `go test -count=1 -v ./internal/verifier` — all 8 tests PASS (3 prior + 3 new + 2 prior baseline).
- [x] `gofmt -l .` (excl. `cauldron/`) — empty.
- [x] No-target baseline asserts NO un-run verdict — PASS (`TestVerifierNoTargetBaselineIsHonest`:
  present-tense "Your (size, root) does not match" absent; "not yet run" / "no verdict is claimed" /
  "Illustrative — what a real mismatch shows" present; no data-island/loader).
- [x] Configured target wires the data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader + the
  three `data-state`s, target reflected only inside the island — PASS
  (`TestVerifierConfiguredTargetWiresLiveVerification`).
- [x] Fail-closed validation (wrong scheme / empty host / fragment / missing id / missing monitor → baseline)
  — PASS (`TestVerifierRejectsMalformedTarget`); reviewer re-probed `net/url` for the `%23frag`/`#frag`/
  empty-host/`ftp://` cases (all caught).
- [x] No-CDN body ban still passes (runs the no-target baseline; user `https://` target lives only in the
  data-island) — PASS.
- [x] Mutation check reproduced: reverting the illustrative `.mismatch-body` to the unconditional
  present-tense "Your (size, root) does not match" makes `TestVerifierNoTargetBaselineIsHonest` FAIL;
  restored → PASS.
- [x] Field-name mapping verified against the bundle/evidence contract: JS reads
  `inclusion.inclusionProof`/`leafIndex`/`treeSize` + `bundle.checkpoint`, and re-derives the root as
  `lines[2]` of the checkpoint body — byte-matches `proofBundle` + `logclient.InclusionEvidence` JSON tags
  and `parseCheckpointBody`.
- [x] Gate-integrity scan over the unpushed diff (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/build-tag/
  swallowed-error/deleted-assertion. Oracle/conformance gate N/A (pure HTML render; the live verdict runs
  in the browser, not `go test`).

**Issues found:**
- (resolved this iteration) The open `normal` "Surface-C verifier renders the split-view mismatch alert
  unconditionally with a present-tense (un-run) verdict" — verified fixed (illustrative-by-default + gated
  on a real `failed` verdict), mutation-proven, visually confirmed; deleted from `issues.md`.
- (new, `normal`, from Codex P1 #1) Surface-C live wiring is gated on SERVER-side `.HasTarget`, but the
  documented deployment is a STATIC GitHub-Pages artifact — `{{if .HasTarget}}` freezes at generation time,
  so `?monitor=&id=` is unreachable in production. The next deploy/wiring step must read the target
  client-side (or serve dynamically). Filed.
- (new, `normal`, from Codex P1 #2) The WASM verifier proves only inclusion math — it never verifies the
  checkpoint signature against the hub's did:web key, nor binds the record to the requested id, so a
  malicious monitor can produce a green `verified` (the monitor stays in the trust path on a surface whose
  whole promise is the opposite). Same verifier-core scope the certificate already ships; the copy
  (verifier.html:449 "Check the signature against the hub's did:web key", :601 "hub-signed") overstates it.
  Filed.

**Codex second opinion:** Completed (~6 min); 2 P1 findings, BOTH reviewer-CONFIRMED and filed as `normal`
(neither blocks this scoped increment — its Verify is met through `Handler()` and the deploy step is
Not-In-Scope):
- P1 #1 "Do not gate the live verifier on server-side query state" → CONFIRMED against the documented
  static GitHub-Pages deployment (CLAUDE.md / next.md / learnings); filed.
- P1 #2 "Verify the fetched bundle before rendering success (signature + id binding unchecked)" → CONFIRMED
  against `verifyadapter.VerifyJSON`/`proof/verify` (inclusion-only); same scope as the shipped certificate
  tier-2, more acute cross-origin; filed.

**Visual check:** Done (ADR-0012, `agent-browser` 0.29.0). Built a throwaway in-module harness (mount
`verifier.Handler()` + `web.Handler()` on :43911, since the leaf is intentionally unmounted), screenshotted
the no-target baseline, a configured-target render, and the `.dc.html` mockup; removed the harness (no
stray files). The honesty fix renders exactly as intended: baseline shows the mismatch alert
illustratively (dashed + muted + "Illustrative — what a real mismatch shows … On a mismatch you would
see:"); the configured target flips the run-label to "verifying in your browser…" and the verdict region to
"Re-verifying this inclusion proof…". No new verifier-specific visual delta worth filing — the masthead
`.codes` chrome divergence and the missing config-driven instance identity are intentional / already-tracked
in the open `/` issue. CAVEAT: the live `verified`/`failed` WASM transition could not be exercised end-to-end
(the testnet `monitor.iscc.id` is not reachable from the sandboxed harness and there is no local verifier
bundle fixture); it is golden-tested as markup and is structurally identical to the certificate tier-2 that
was previously verified live.

**Next:** The sibling open WASM sub-step — the **dossier tier-2 caller** (no caller in `internal/dossier`
today). When the **GitHub Pages / `monitor.iscc.codes` deploy workflow** lands (the final Surface-C piece),
it MUST reconcile the static-deployment gating (new Codex-P1 issue) — read the target client-side, not via
server-side `.HasTarget`. Also still open: the WASM `safeIndex` test-gap (pair with a `cmd/wasm` touch) and
the verifier-scope signature/id-binding gap (pair with a WASM-verifier-scope expansion).

**Notes:**
- The new static-deployment finding (Codex P1 #1) is the most important latent gap: the feature passes every
  test through the `Handler()` HTTP seam but that is NOT how GitHub Pages serves the page. Do not let the
  green seam tests imply the live verifier works in production until the client-side gating lands.
- The verifier-scope gap (Codex P1 #2) is shared with the certificate's tier-2 — fixing it is a WASM-core
  scope expansion (add signature + did:web-key + id-binding verification), larger than a wiring slice; until
  then the success/step copy overstates what runs.
- Scope was clean: 3 files, `verifier.Handler()` deliberately NOT mounted in `buildMux` (per next.md
  Not-In-Scope + `learnings/verifier.md`), no `internal/web` import, no `go.mod`/`go.sum` change.
- Learnings updated: `learnings/verifier.md` (honesty gap marked settled + the cross-origin live-wiring
  mechanics: target-only island, JS root re-derivation, three-state distinction, parseTarget-as-guard);
  `learnings/cmd-wasm.md` (the `isccVerifyInclusion` scope gap). No index promotion — the always-loaded
  "re-verify, don't render a flag" rule already covers this; index stays at 97 lines.
