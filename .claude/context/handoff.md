## 2026-06-22 — Make the certificate Tier-2 honesty header honest on the no-JS baseline

**Done:** Reworded the `{{if .HasBundle}}` honesty-header sentence in `cert.html` so it no longer
asserts a present-tense browser re-verification on the no-JS baseline — it now states the always-true
offline path unconditionally and makes the browser re-check explicitly conditional on JavaScript,
aligning register with the `#tier2-result` panel default. Extended the no-JS baseline subtest with a
non-vacuous negative+positive assertion pair.

**Files changed:**
- `internal/certificate/cert.html`: line 484 only — replaced "This browser re-verifies the proof below
  for you, and you can download the bundle and re-verify it offline." with "You can download the bundle
  and re-verify it offline; with JavaScript enabled, this browser also re-checks the proof below for
  you." `{{else}}` branch and the `#tier2-result` panel (:501) are byte-unchanged.
- `internal/certificate/handler_test.go`: extended the "No-JS baseline" block inside
  `TestCertificateRendersWasmVerifier` with a negative assertion (`!Contains "This browser re-verifies
  the proof below"`) and a header-specific positive assertion (`Contains "with JavaScript enabled, this
  browser also re-checks the proof below"`, distinct from the panel's "above" wording at :501).

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 28 packages ok);
`gofmt -l .` empty.
- [x] `go test -count=1 -run TestCertificateRendersWasmVerifier ./internal/certificate` → PASS.
- [x] Mutation (manual): reverting ONLY the `cert.html:484` copy back to "This browser re-verifies the
  proof below for you, and you can download the bundle and re-verify it offline." makes
  `TestCertificateRendersWasmVerifier` FAIL (the negative assertion trips). Restored byte-identical;
  `git diff` over the template shows only the :484 sentence changed.
- [x] `#tier2-result` panel text (`cert.html:501`) is byte-unchanged (confirmed via `git diff`).

**Next:** Continue draining code-closable `normal`s. The remaining §5/§6-adjacent ones are
store-touching: (1) the §6 `· at` timestamp needs a store schema column on `RecordRow` (define
carefully); (2) the `/` Checkpoint/Anchor-column `normal` also needs a store/projection change. The
front-of-queue WASM-verifier signature half (no browser did:web signature check) stays design-first / a
STOP-candidate — do a design pass before touching `verifier.html` or the WASM core.

**Notes:**
- Pure copy + test change, no oracle/conformance/proof/signature/Merkle path touched; no WASM-pure leaf
  affected (certificate is server-side only). No `data.*` field, store read, or template variable added —
  string edit only, as scoped.
- The positive assertion deliberately targets the header-specific phrasing ("...also re-checks the proof
  below...") rather than the bare literal "with JavaScript enabled" (which also appears in the unchanged
  `:501` panel), so the positive case is non-vacuous for the HEADER change, not just the branch.
- Open count unchanged minus this drain: this closes the `normal` "Certificate tier-2 honesty header
  overstates 'This browser re-verifies' on the no-JS baseline" once review confirms. DONE still requires
  0 `normal`.
