## 2026-06-23 — Review of: Dossier §1 stops claiming "Key resolved from did:web" on the `unresolvable` path

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Tightly-scoped, correct SSR honesty fix: on the `unresolvable` overlay path the dossier §1
Identity no longer asserts "Key resolved from did:web:<domain>" (which contradicted the same page's
"signing key is unresolved" caution), rendering neutral "Key source:" wording instead via a new pure
view-model flag `KeyUnresolved` set at the same site as `ShowCaution`. Exactly 3 files (2 prod + 1 test),
all in next.md scope; `mise run check` green, gofmt clean, every next.md Verification line met, both
mutation checks reviewer-reproduced. This code-closes the open `normal` dossier §1-honesty issue.

**Verification:**
- [x] `mise run check` green (build + vet + test across all 30 packages) — confirmed.
- [x] `gofmt -l .` empty — confirmed.
- [x] `go test -count=1 -run TestDossier ./internal/dossier` — PASS (existing suite unaffected).
- [x] NEW `TestDossierUnresolvedKeyWordingHonesty` — PASS; asserts the `unresolvable` body drops "Key
  resolved from", carries "Key source:", and still shows `did:web:sb0.iscc.id`, plus a verified-sibling
  (nil source) case asserting the mockup's `Key resolved from <span class="mono">did:web:…</span>` copy
  is unchanged and NOT over-gated.
- [x] Mutation 1 (revert the §1 template gate → always "Key resolved from") — reviewer-reproduced: the
  unresolvable assert FAILS; reverted → PASS.
- [x] Mutation 2 (over-gate: `KeyUnresolved: true` in `buildData`) — reviewer-reproduced: the
  verified-sibling assert FAILS; reverted → PASS. Gate guards both directions, non-vacuous.
- [x] Mockup-faithfulness — `.dc.html:88` §1 copy is "Key resolved from did:web:{{ hub.domain }}";
  the `{{else}}` arm preserves it verbatim for every non-`unresolvable` status.
- [x] Scope — diff touches exactly the 3 next.md-scoped files (handler.go, dossier.html, handler_test.go)
  + handoff; nothing in `## Not In Scope` (the `unverified` §1 wording is correctly left as "resolved",
  since `unverified` DID resolve a key — the caution copy at handler.go:383 confirms).
- [x] Quality-gate integrity — full unpushed diff (3 commits, `@{upstream}..HEAD`) scanned: no `nolint`,
  `t.Skip`, swallowed errors, build-tag exclusions, deleted assertions, or loosened gates.
- Oracle/conformance gate: **N/A** — pure SSR template + view-model flag off the already-resolved status
  string; no signature, RFC-6962, Merkle, did:web resolution, fsck, or proof path touched; no new store
  read; `go.mod`/`go.sum` untouched.

**Issues found:** (none new). Resolved + pruned the open `normal` "Dossier §1 unconditionally says 'Key
resolved from did:web:…' even on the `unresolvable` overlay path" — its verify-fixed criterion is exactly
what the new test (mutation-reproduced) asserts.

**Codex second opinion:** unavailable — my `codex review --commit HEAD` invocation was denied by the
auto-mode sandbox classifier (autonomous-agent / full-access policy). The stale `/tmp/codex-review.txt`
present (timestamp 17:02) describes the PRIOR iteration's record-list domain/identity threading, NOT this
dossier §1 change, so I did not treat it as a verdict on this commit. Graceful degradation: a missing
second opinion is a note, not NEEDS_WORK; my own review + the two reproduced mutations cover the change.

**Visual check:** skipped — the change is a pure text-content swap ("Key resolved from" → "Key source:")
inside the existing `.section-value` div, visible only on the `unresolvable` overlay path; it adds/moves
no named region, affordance, layout, or chrome, so an `agent-browser` screenshot pass would file no
genuine visual delta (its only effect is one word's wording on a failed-resolution path). `agent-browser`
IS available; the skip is on low value, not unavailability. Visual fidelity's hard gate remains the human
M-UI exit sign-off (target.md), unaffected by this micro-copy.

**Next:** The dossier §1 honesty `normal` is now code-closed. Remaining non-blocked code-closable work is
the **M-API contract-accuracy** doc-fixes (open `normal`s, NOT human-blocked, oracle-N/A): remove the
phantom `index` query param from `/{domain}/log/verify` in `openapi.yaml` + the JSON twin, and fix the
`/{domain}/log/checkpoint` `200` media type from `text/plain` → `application/octet-stream` (the live mux
serves octet-stream). Steer `define-next` there as one or two small slices. The remaining other `normal`s
(WASM cross-origin signature half; realm-index per-checkpoint Anchor honesty) are design-first/blocked,
and the lone `critical` is human-blocked on the M-UI exit sign-off — do NOT re-attempt either in code.

**Notes:**
- `learnings/dossier.md`: the §1 bullet is rewritten as `settled (advance 988d48d)` and marked CLOSED;
  the finding stays package-local (not promoted to the index) — it is dossier-specific copy, not a
  cross-cutting rule. dossier.md 71 lines / 8 bullets; index 109 lines — both under budget.
- The §1-honesty gate is set at the SAME site as `ShowCaution` (one source of truth off the resolved
  `status`), so the §1 verb and the caution copy can never drift out of sync — a clean, minimal seam.
- The masthead-identity / overlay duplication across dashboard+dossier+certificate (3×) is unchanged by
  this slice and remains the tracked `low` consolidation; this change correctly did not touch it.
