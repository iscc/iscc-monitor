# Next Work Package

## Step: Dossier §1 stops claiming "Key resolved from did:web" on the `unresolvable` path

## Advances
target.md **M-UI** Verify bar — the SSR-honesty requirement that the served no-JS HTML must not assert
something the data does not support ("a pre-coverage state never renders as a guarantee (ADR-0001)"; the
always-loaded learnings rule that a rendered assertion is gated on real data, never overstating a verdict).
It also closes the open `normal` issue **"Dossier §1 unconditionally says 'Key resolved from did:web:…' even
on the `unresolvable` overlay path"** (issues.md).

**Why this, not the M-API fixes the handoff named:** those are STALE — commit `53ee328` (already an
ancestor of HEAD `0673ff0`) already removed the phantom `/{domain}/log/verify` `index` param, fixed
`/{domain}/log/checkpoint`'s `200` media type to `application/octet-stream`, added the `/healthz` 503
response, and landed the mermaid no-CDN ban. `go test ./internal/openapi` is green; the M-API
contract-fidelity criterion is closed in code. The three matching issue entries are resolved-pending-prune,
NOT open work.

**Why this, not the `critical`:** the lone `critical` (dossier→log-browser navigation + record-list parity)
is fully code-closed and human-blocked on the M-UI exit sign-off — it must NOT be re-attempted in code.
Of the remaining open `normal`s, the WASM cross-origin signature half and the realm-index per-checkpoint
Anchor honesty are both explicitly design-first / design-blocked. This §1 honesty fix is the one open
`normal` that is code-closable WITHOUT a design pass: it does NOT rewrite the mockup-specified copy for the
normal case — it only suppresses a self-contradiction on the failed-resolution path.

## Goal
On the `unresolvable` overlay path the dossier currently renders §1 Identity as "Key resolved from
did:web:<domain>" while the soft caution on the SAME page says the signing key "is unresolved" — a direct
self-contradiction that overstates a key resolution that failed. Make §1 render neutral source wording when
the live verdict is `unresolvable`, keeping the mockup's "Key resolved from did:web:<domain>" phrasing for
every other status.

## Scope
- **Modify**:
  - `internal/dossier/handler.go` — add one bool field to `dossierData` (e.g. `KeyUnresolved`) and set it
    in `buildData` when `status == "unresolvable"`; carries the §1-honesty gate (1 of the ≤3 prod files).
  - `internal/dossier/dossier.html` — gate the §1 `section-value` text on that field: render neutral source
    wording (e.g. `Key source: <span class="mono">did:web:{{.Domain}}</span>`) when `KeyUnresolved`, else
    keep today's `Key resolved from <span class="mono">did:web:{{.Domain}}</span>` (`dossier.html:533`).
- **Reference**:
  - `.claude/context/learnings/dossier.md` — §1-static-derived note (lines 34-40): §1 is "WHERE the key
    comes from", the mockup specifies the static phrasing, the fix is neutral "Key source:" wording or
    gating "resolved" off `unresolvable`; do NOT silently rewrite the mockup copy for the resolved case.
  - `.claude/context/learnings/dashboard.md` — the shared overlay/masthead mechanics the dossier ports.
  - `internal/dossier/handler.go:308` (`ShowCaution = status == "unresolvable" || status == "unverified"`)
    and `handler.go:370-371` (the `unresolvable` caution copy that §1 contradicts).
  - `.claude/design/ISCC Monitor - Hub Dossier.dc.html` §1 (~line 88) — confirm the resolved-case phrasing
    stays mockup-faithful.

## Not In Scope
- Do NOT touch the `unverified` path's §1 wording. `unverified` means the signature matched no listed
  did:web key — the key WAS resolved (the doc fetched/parsed); only the signature failed. So "resolved"
  stays honest for `unverified`; gate the new wording on `unresolvable` ONLY.
- Do NOT re-attempt the human-blocked `critical` (dossier→log-browser navigation / record-list parity) in
  any form — no chrome, breadcrumb, or pager edits.
- Do NOT consolidate the 3× masthead/overlay duplication or the §2/§3/§4 honesty plumbing (tracked `low`s).
- Do NOT prune the now-stale M-API issue entries here — that is an `update-state` / `review` bookkeeping job.
- Do NOT change the §1 phrasing for `verified` / `frozen` / `inactive` — they keep the mockup copy.

## Implementation Notes
- The overlaid status the dossier renders is already computed in `buildData` (the `status` string passed in,
  resolved via `overlayStatus`/`hubStatus`). Add `KeyUnresolved bool` to `dossierData` and set it
  `status == "unresolvable"` right where `ShowCaution` is set (`handler.go:308`) — one source of truth, no
  new store read (§1 stays static-derived; this is a pure view-model flag off the already-resolved status).
- In `dossier.html:533`, gate the `section-value` text with `{{if .KeyUnresolved}}Key source: …{{else}}Key
  resolved from …{{end}}`, keeping `<span class="mono">did:web:{{.Domain}}</span>` in BOTH arms so the
  domain still shows (the source is always honest; only the verb "resolved" is gated off the failed path).
- Honesty rule (always-loaded learnings + target.md M-UI): a served no-JS surface must not assert a verdict
  the data does not support. On `unresolvable` the monitor cannot fetch/parse the did:web doc, so it has NOT
  resolved a key — §1 must not say "resolved". This mirrors the §2/§3/§4 honesty gating already in place
  (`learnings/dossier.md`: every §1–§4 value is honesty-gated against real data).
- Keep the change behavior-neutral for every non-`unresolvable` status: the resolved-case copy is unchanged,
  so the mockup-parity and existing region tests (`TestDossierRendersCoveredHub`, the chrome tests) stay green.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...`; `gofmt -l .` empty).
- `go test -count=1 -run TestDossier ./internal/dossier` passes (existing dossier suite unaffected).
- NEW test (add to `internal/dossier/handler_test.go`, e.g. `TestDossierUnresolvedKeyWordingHonesty`): drive
  a store-verified hub with `fakeStatusSource{id: "unresolvable"}` (the `TestDossierRendersInMemoryStatus`
  seam, handler_test.go:919), GET `/<domain>`, and assert the body does NOT contain `Key resolved from`
  AND DOES contain the neutral `Key source:` wording (and still shows `did:web:<domain>`).
- Mutation check (state it; review re-runs it): reverting the §1 gate (so the template always renders "Key
  resolved from") makes that new test FAIL; a sibling assertion that a `fakeStatusSource{id: "verified"}`
  (or nil source) hub STILL renders `Key resolved from did:web:<domain>` guards against over-gating.

## Done When
`mise run check` is green and the new `unresolvable`-path test asserts §1 no longer says "Key resolved from"
while a verified-path assertion confirms the mockup's "Key resolved from did:web:<domain>" copy is unchanged
for every other status.
