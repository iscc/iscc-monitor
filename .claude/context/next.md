# Next Work Package

## Step: Stop the verifier app from claiming an un-run did:web signature check

## Advances
Closes the **code-closable (copy-honesty) half** of the open `normal` issue *"The WASM verifier never
checks the checkpoint signature against the hub's did:web key … the copy overstates this: `verifier.html:451`
lists 'Check the signature against the hub's did:web key' as a step the verifier WILL run, and
`verifier.html:637` reports it '… re-verified this inclusion proof against the **hub-signed** checkpoint
root' — but neither the signature nor a did:web resolution runs."* The issue itself authorizes this interim
narrowing: *"until then, **narrow the success copy** + drop the unrun did:web step from the record block so
the page does not claim a signature/key check it skips."*

This serves the **target.md WASM verifier** milestone honesty bar and the always-loaded learnings rule:
*"On a self-verifiable surface, gate a rendered `✓` … on a re-VERIFICATION, not a status flag. A built proof
is not a verified proof"* — and, equally, a page must not list a verification step (signature/did:web) it
never runs. The full in-browser signature + did:web resolution (the **other**, design-blocked half) stays
out of scope.

**Why this, not the handoff's `Next:`** The handoff/`state.md`/`issues.md` steer to "M-API contract-accuracy
doc fixes (phantom `index` param on `/verify`; checkpoint media type)". Those are **already done and
committed** (commit `53ee328`, an ancestor of HEAD) and gated by per-op tests
(`TestContractVerifyHasNoIndexParam`, `TestContractCheckpointMediaType`, `TestContractHealthzHas503`,
`TestNoMermaidInContract`). The served `/openapi.json` already shows `verify` with no `index` param and
`checkpoint` `200` as `application/octet-stream`; `go test ./internal/openapi` is green. That tracking is
**stale, not open work** — see Not In Scope for the prune note.

## Goal
Make the verifier app (`monitor.iscc.codes`, Surface C) state truthfully that its in-browser WASM check
re-verifies **RFC-6962 inclusion + the id-binding against the committed/accepted root** — and NOT a
checkpoint signature against the hub's did:web key, which `isccVerifyInclusion` does not run. Removes a
green-but-overstated claim a skeptical client would otherwise trust.

## Scope
- **Modify**: `internal/verifier/verifier.html` — the verification-step list (the `vstep` at line 451
  asserting "Check the signature against the hub's did:web key") and the tier-2 verdict copy (the
  `setVerdict("verified", …)` / `setVerdict("failed", …)` strings at lines 637 / 639, plus the matching
  vstep label at line 447 — all currently say "hub-signed checkpoint root").
- **Modify**: `internal/verifier/handler_test.go` — update `TestVerifierRendersNamedRegions` (lines 101–102,
  which assert the OLD step strings) and add a mutation-provable honesty test (see Verification).
- **Reference**:
  - `internal/verifier/verifier.html:438-463` (the four `vstep` labels + the verdict region) and
    `:625-640` (the `isccVerifyInclusion` call + its own comment confirming it gates on inclusion math + the
    id-binding only — no signature / no did:web).
  - `internal/certificate/cert.html:587` — the already-honest sibling tier-2 copy to match:
    `"✓ Your browser re-verified this inclusion proof against the accepted root."` (note: NOT "hub-signed").
  - `.claude/context/learnings/verifier.md` — the package-local rules (pure stdlib leaf; no-CDN body ban
    golden-tested; the illustrative-mismatch honesty already settled; the three distinct render states).
  - `.claude/context/issues.md` — the issue *"The WASM verifier never checks the checkpoint signature
    against the hub's did:web key"* (this step closes its copy-honesty half only).

## Not In Scope
- **Do NOT implement in-browser did:web resolution or checkpoint-signature verification** — that is the
  design-blocked other half of the same `normal` (needs a design pass; the issue says so). This step removes
  the false claim only; it does not add the missing check.
- **Do NOT touch `internal/certificate/cert.html`** — its tier-2 copy ("accepted root"; §4 is server-rendered
  evidence) is already honest; it is the parity reference, not an edit target.
- **Do NOT re-edit the OpenAPI contract** (`openapi.yaml` / `openapi.json`) — the `verify`-`index` and
  `checkpoint` media-type fixes are already committed (`53ee328`) and gated; re-doing them is churn. The
  stale `normal`/`low` entries describing them (and the resolved `/` hero-footer entry, all 4 sub-items
  closed) are for a future `update-state`/`review` to **prune**, not for this step to fix.
- Do NOT change the WASM artifact, `cmd/wasm`, or `internal/proof/verify` — the verifier core is unchanged;
  only the page's prose about what it does changes.
- Do NOT re-attempt the human-blocked `critical` (dossier→log-browser navigation) in any form.

## Implementation Notes
- The WASM checker (`globalThis.isccVerifyInclusion`, `verifier.html:631`, comment at `:625-630`) verifies
  ONLY (a) the RFC-6962 inclusion math (`record`+`proof`+`size`→`root`) and (b) the id-binding (the record
  commits `target.id`). It does NOT fetch a did:web document or verify the checkpoint note signature. So
  every page string that implies a signature / key check is the un-run claim to fix.
- **The fourth `vstep` (lines 449-452)** is the false step. Replace its label so it describes what the
  browser actually does — bind the record's committed ISCC-ID to the requested id (the id-binding half),
  e.g. "Confirm the record commits the requested ISCC-ID". Do NOT simply delete the row and leave the list
  implying a full verification; the page must not imply the signature is checked here. If a phrase is wanted
  to locate signature trust honestly, point it at the SERVER / certificate §4 (which IS server-verified),
  not at this in-browser run.
- **The "hub-signed checkpoint root" phrasing** at `:447`, `:637`, `:639` overstates: the WASM rebuilds the
  committed root from the inclusion proof; it does not validate the root was hub-signed. Match the
  certificate's already-honest wording — "the accepted root" / "the committed root" — so the verdict claims
  only the inclusion + id-binding it actually ran.
- **Read each remaining "hub-signed" string in context before changing it.** The `#mismatch` body copy
  (`:500`) and the split-view lede refer to the EVIDENCE the user holds (their own signed checkpoint), not
  to a check the WASM ran — those may keep "hub-signed". Only change strings that assert the *in-browser run*
  performed a signature check.
- Keep this a pure static-template + golden-test edit. Oracle/conformance gate is **N/A** — no signature,
  RFC-6962, Merkle, did:web, fsck, or proof code is touched (`internal/proof/verify` and `cmd/wasm`
  untouched; the page only stops claiming a check). The verifier is a pure stdlib leaf
  (`learnings/verifier.md`): the template parses at init, so a malformed edit fails the build, not a request.
- The no-CDN body ban is golden-tested (`TestVerifierNoExternalCDN`) — do not introduce any `http://` /
  `https://` / `cdn.` / `jsdelivr` literal; the bare text "did:web" is fine (no scheme).
- Relevant learnings rule (always-loaded): a rendered `✓` a reader trusts must reflect an actual
  re-verification — and, by the same honesty, a listed verification STEP must be one the code runs. This is
  the recurring "Verdict-UI honesty" pattern (auto-memory): advance keeps shipping UI that asserts an un-run
  verification, review catches it — fix it at the source here.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestVerifier ./internal/verifier` passes (the updated
  `TestVerifierRendersNamedRegions` no longer asserts the dropped "Check the signature against the hub's
  did:web key" step string).
- A NEW mutation-provable honesty test (e.g. `TestVerifierDoesNotClaimSignatureCheck`) asserts the served
  body: (1) does **not** contain "Check the signature against the hub's did:web key" (the un-run step is
  gone); (2) the `setVerdict("verified", …)` copy does **not** claim the browser re-verified a "hub-signed
  checkpoint root"; (3) the `verified` verdict copy asserts only an inclusion / accepted-/committed-root
  re-verification. Reverting the template change (re-add the did:web step / restore "hub-signed checkpoint
  root" to the verdict) makes this test FAIL.
- Assertion: `grep -c "Check the signature against the hub's did:web key" internal/verifier/verifier.html`
  returns `0`.
- Assertion: the `setVerdict("verified", …)` line in `internal/verifier/verifier.html` no longer contains
  the substring `hub-signed` (it claims only the inclusion / accepted-root check the WASM ran).

## Done When
`mise run check` is green and the new honesty test (mutation-proven: reverting the template change FAILs it)
plus the updated `TestVerifierRendersNamedRegions` confirm the verifier app no longer lists or claims an
in-browser did:web signature check it does not run.
