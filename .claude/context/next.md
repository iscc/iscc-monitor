# Next Work Package

## Step: Make the certificate Tier-2 honesty header honest on the no-JS baseline

## Advances
This closes a `normal` issue, not a milestone Verify criterion directly — but it is the
handoff-named **highest-priority code-closable** target, and it serves the M-UI Verify bar it
sits inside:

> target.md M-UI certificate: "the tier-1 ('the monitor reports') vs tier-2 ('verify in your
> browser') affordance is present and visually distinct" and (Document chrome) the two-tier
> honesty split must hold on the **no-JS baseline** — every SSR surface is "complete with
> JavaScript disabled".

Issue (issues.md "Certificate tier-2 honesty header overstates 'This browser re-verifies' on the
no-JS baseline", `normal`, [review]): `cert.html:484` — the `{{if .HasBundle}}` honesty header
asserts, unconditionally and present-tense, "This browser re-verifies the proof below for you,"
even though the tier-2 verdict is progressive enhancement that does NOT run with JavaScript
disabled. The verdict panel below (`cert.html:501`) is correctly conditional, so the two regions
disagree on the no-JS baseline. This preempts the front-of-queue WASM-signature half, which
state.md + the review handoff both flag as **design-first / a STOP-candidate** (browser did:web
resolution is non-trivial) — so the WASM milestone's remaining code work is deliberately deferred
to a design pass, leaving this copy fix as the correct code-closable increment. DONE still
requires 0 `normal`; this drains one.

## Goal
Reword the `HasBundle` honesty header so it describes only the always-true bundle/offline path
plus a *conditional* ("with JavaScript enabled") browser re-check — never a present-tense claim
that a verdict ran — so the static no-JS copy stops promising a verification that may not happen.
The script's `#tier2-result` panel stays the sole asserter of an actual browser re-verification.

## Scope
- **Modify**: `internal/certificate/cert.html` (the one honesty-header line, `:484`)
- **Modify**: `internal/certificate/handler_test.go` (extend the existing no-JS baseline subtest
  in `TestCertificateRendersWasmVerifier` to assert the header copy is consistent with the
  conditional verdict panel)
- **Reference**: `.claude/context/learnings/certificate.md` (two-tier-honesty copy rules; the §3
  re-verification gate that sets `HasBundle`/`RecordB64`; the `html/template` entity-escape note
  on base64 chips)
- **Reference**: `internal/certificate/cert.html:482-508` (the honesty `<div>`, the actions row,
  and the conditional `#tier2-result` / `#tier2-text` panel whose phrasing to align with)
- **Reference**: `internal/certificate/handler_test.go:834-859` (the existing "No-JS baseline"
  block inside `TestCertificateRendersWasmVerifier` to extend)

## Not In Scope
- The WASM verifier **signature half** (no checkpoint-note / did:web signature check in the
  browser) — design-first / STOP-candidate per the handoff; do NOT touch `verifier.html` or the
  WASM core, and do NOT loosen any "hub-signed root" success copy.
- The `#tier2-result` panel text (`cert.html:501`) — it is already honest and conditional; leave
  it byte-unchanged so it remains the single source of the active-verdict claim.
- The `{{else}}` branch ("…land in a later release") — it renders only when there is no bundle and
  is accurate (confirmed a FALSE POSITIVE in the issue); leave it unchanged.
- The §6 `· at` timestamp `normal` and the `/` Checkpoint/Anchor-column `normal` — both need a
  store schema / projection change; out of scope for this copy-only step.
- Any new store read, handler-data field, or template variable — this is a string edit only; do
  NOT add a `data.*` field.

## Implementation Notes
- The fix is **copy-only** inside the existing `{{if .HasBundle}}` branch of the honesty span
  (`cert.html:484`). Replace the present-tense "This browser re-verifies the proof below for you,
  and you can download the bundle and re-verify it offline." with copy that (a) states the
  always-true offline path unconditionally and (b) makes the browser re-check explicitly
  conditional on JavaScript — e.g. "You can download the bundle and re-verify it offline; with
  JavaScript enabled, this browser also re-checks the proof below." Match the phrasing register of
  the panel default at `:501` ("with JavaScript enabled, this browser re-checks…") so the two
  regions agree.
- **Honesty rule (learnings/certificate.md, two-tier-honesty):** on a Tier-1 self-verifiable
  surface honesty copy is load-bearing — the static SSR copy must never assert a verdict that did
  not run. Let the `#tier2-result` script be the sole asserter; the header only *offers* paths.
- Do NOT change `HasBundle` semantics or which markers render — only the words inside the span.
  The `Tier 1`/`Tier 2`/`Download proof bundle` markers the no-JS baseline subtest already checks
  for must still render unchanged (they are outside the edited sentence, so the existing loop at
  handler_test.go:846-859 keeps passing).
- **Test (extend the certifiable no-JS branch in `TestCertificateRendersWasmVerifier`,
  handler_test.go ~:834-859):** add a positive+negative pair so the assertion is non-vacuous —
  assert the served body does NOT contain the present-tense claim
  (`!strings.Contains(body, "This browser re-verifies the proof below")`) AND DOES contain the new
  conditional phrasing (the literal "with JavaScript enabled" — already used by the panel default,
  so prefer the same words). Use `rec.Body.String()` directly; plain prose needs no
  `html.UnescapeString` (only base64 `+`/`/` chips do, per the learnings note) — so pick a literal
  with no `+`/`/` to avoid entity-escape surprises.
- No oracle/conformance path is touched (no signature / RFC-6962 / Merkle / proof code) — this is
  template prose plus a string assertion. No WASM-pure leaf is affected.

## Verification
- `mise run check` is green (build + vet + `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCertificateRendersWasmVerifier ./internal/certificate` passes.
- Mutation (manual, record in handoff): reverting only the `cert.html:484` copy back to "This
  browser re-verifies the proof below for you…" makes `TestCertificateRendersWasmVerifier` FAIL —
  the new assertion is non-vacuous.
- `git diff` over the template shows only the `:484` sentence changed; the `#tier2-result` panel
  text (`cert.html:501`) is byte-unchanged.

## Done When
`mise run check` is green and `TestCertificateRendersWasmVerifier` passes with a non-vacuous
assertion proving the no-JS `HasBundle` header no longer claims an active browser re-verification,
while the conditional `#tier2-result` panel remains the sole asserter of a browser verdict.
