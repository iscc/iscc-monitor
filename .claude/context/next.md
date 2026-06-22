# Next Work Package

## Step: Add the dossier's tier-2 `verify ↗ monitor.iscc.codes` chrome link, instance-identity block, and `← Realm index` back-link

## Advances
WASM milestone Verify — the milestone calls for the Evidence-Ledger **tier-2** affordance
("verify in your browser") on the certificate **and the dossier**:
> "elevates the Evidence Ledger's **tier-2** result … on the M-UI certificate/dossier, plus the
> standalone Independent Verification verifier app … at `monitor.iscc.codes`".

It also closes part of the M-UI cross-cutting Verify bar (target.md "Design parity (named-region)"):
> "**Document chrome + instance identity.** Every surface carries … the **instance-identity** block …
> and the **`verify ↗ monitor.iscc.codes`** tier-2 link"; and
> "**Navigation closure.** … forward links *and* `←` breadcrumb back-links both present, so … no
> surface is a dead end."

The hub dossier (`internal/dossier`) is the front-of-queue open WASM/M-UI surface flagged by the
`review` handoff `**Next:**` ("the **dossier tier-2 caller** — no caller in `internal/dossier` today")
and by state.md. The dossier mockup's tier-2 affordance is the chrome `verify ↗ monitor.iscc.codes`
link (`.dc.html:37`), NOT a baked-in WASM proof island — the dossier has no single ISCC-ID subject to
re-verify, unlike the certificate. So the correct dossier tier-2 caller is the cross-surface link to
Surface C, the instance-identity chrome, and the `← Realm index` back-link the dossier currently lacks.

## Goal
Bring the hub dossier's masthead and navigation to the same shared-chrome baseline the certificate
already ships: the tier-2 link out to the monitor-agnostic verifier app, the instance-identity label,
and a back-link up to the realm index — making `/` → dossier → log browser fully traversable with
JavaScript disabled and giving the dossier its mandated tier-1/tier-2 split on the page itself.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/dossier/dossier.html` — add `.chrome-actions` / `.chrome-instance` / `.chrome-verify`
    CSS (port verbatim from `internal/certificate/cert.html:66-88`), the `<div class="chrome-actions">`
    block with the `monitor instance` label + `verify ↗ monitor.iscc.codes` link inside the existing
    `<header class="chrome">`, and a `.backlink-row` / `.backlink` `← Realm index` link as the first
    child of `<main class="page">` (port from `cert.html:96-104` + `:390-399`).
  - `internal/dossier/handler_test.go` *(test — does not count toward the ≤3 non-test/doc budget)* —
    extend `TestDossierRendersCoveredHub` (or add a focused `TestDossierChromeTierTwoAndBackLink`) to
    assert `← Realm index`, `monitor.iscc.codes`, and `monitor instance` appear; **and narrow the
    existing no-CDN ban (line ~234) from banning all `https://` to banning only third-party CDN hosts**
    (`jsdelivr`, `cdn.`, `unpkg`, `googleapis`) plus bare `http://` — mirroring
    `certificate/handler_test.go:201`, since `https://monitor.iscc.codes/` is the one intentional
    external https origin.
- **Reference**:
  - `internal/certificate/cert.html` (lines 27-104 CSS, 380-399 markup) — the exact chrome + back-link
    pattern to port; `internal/certificate/handler_test.go:166-204` — the assertion + narrowed-ban
    style to mirror.
  - `.claude/design/ISCC Monitor - Hub Dossier.dc.html` (lines 36-37 chrome instance + verify link,
    line 43 `← Realm index`) — the authoritative mockup regions.
  - `.claude/context/learnings/certificate.md` — chrome/back-link/tier-2 mechanics on the sibling surface.
  - `.claude/context/learnings/verifier.md` — Surface C is `monitor.iscc.codes` (the `.codes` verifier
    app, not an instance); the link target is the static verifier site root.
  - `.claude/context/learnings/web.md` — the `noExternalCDN` discipline (bans third-party origins only;
    `monitor.iscc.codes` is the one allowed external https origin, like the certificate).

## Not In Scope
- **No WASM proof island / `verify.wasm` loader on the dossier.** The dossier has no single ISCC-ID
  subject, so there is no specific proof to re-verify in-browser; its tier-2 affordance is the link to
  Surface C, not a baked-in re-verification. (Embedding a proof island here would be wrong.)
- **No config-driven instance identity.** Keep the static `monitor instance` label the certificate
  already uses — making the domain/operator/realm env-configurable is the separate, filed `normal`
  issue ("`/` realm-index sub-region deltas … static instance identity"). Match the existing baseline.
- **No "Prove an ISCC-ID in this hub →" action.** Its honest no-id target (the realm-index claim-lookup
  hero) is debatable and belongs to a later parity step; this step stays on the unambiguous chrome +
  back-link requirements. The existing "Browse the log →" surface link stays as-is.
- **No Surface-C / GitHub-Pages deploy, no client-side `.HasTarget` rework, no WASM verifier-scope
  signature/id expansion** — those are the next, larger WASM sub-steps (filed `normal`).
- **No `overlayStatus` / `hubStatus` consolidation** (the 3x-duplication `low` issue) — out of scope.

## Implementation Notes
- This is a **template-only** production change (one non-test/doc file: `dossier.html`). `handler.go`'s
  `dossierData` needs NO new fields — the verify link, instance label, and back-link are static literals
  (the certificate hard-codes `href="https://monitor.iscc.codes/"` and `href="/"`), so do not thread a
  `VerifierURL`/`BackLink` field. Keep `html/template` literals, matching the dossier's existing
  `/_ds/...` literal convention.
- Port the chrome CSS and markup **verbatim** from `cert.html` so the two mastheads stay visually
  identical (DRY across surfaces; reviewer can diff). Use the same DS tokens already in `dossier.html`
  (`var(--text-link)`, `var(--border-default)`, `var(--radius-xs)`, `var(--space-*)`); add NO new token.
  Confirm every `var(--*)` you introduce already resolves in `web/tokens.css` (the chrome CSS reuses
  tokens the dossier already references, so no new token is needed).
- The `← Realm index` link target is `/` (the dashboard realm index, under which the dossier mounts),
  matching `cert.html:398`. The `verify ↗ monitor.iscc.codes` href is `https://monitor.iscc.codes/`
  (the static `.codes` verifier app root, per `learnings/verifier.md` — the `.codes` ≠ `.id`
  distinction; this is the verifier app, never an instance).
- **EDGE CASE (must handle or the suite goes red):** the dossier no-CDN ban
  (`handler_test.go:~234`) currently bans every `https://` substring; adding
  `https://monitor.iscc.codes/` will trip it. Narrow that ban to the third-party CDN host list
  (`jsdelivr`, `cdn.`, `unpkg`, `googleapis`) plus bare `http://`, exactly as
  `certificate/handler_test.go:201` does — this narrowing is the correct fix (the certificate already
  established `monitor.iscc.codes` as the one intentional external origin), NOT a gate weakening.
- Relevant learnings rule (`learnings/web.md`): `noExternalCDN` bans third-party origins only; the
  monitor's own verifier app at `monitor.iscc.codes` is allowed. The DS shell links stay same-origin
  `/_ds/...`.
- No oracle/conformance gate applies: this is a pure HTML render of one persisted store row plus static
  chrome — it touches no signature, RFC-6962, Merkle, did:web, fsck, or proof path (per the dossier
  package docstring). Preserve the existing GET-only 405 gate and buffer-then-200 render discipline.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/dossier` passes (all existing dossier tests plus the chrome/back-link
  assertions).
- The served dossier HTML for a covered hub contains all three literals: `← Realm index`,
  `monitor.iscc.codes`, and `monitor instance` (asserted by the extended/new test).
- The served dossier HTML still contains NO third-party CDN reference (`jsdelivr`, `cdn.`, `unpkg`,
  `googleapis`, `http://`) — the narrowed ban still fires for those and passes for
  `https://monitor.iscc.codes/`.
- The DS shell links remain same-origin: the body still contains `href="/_ds/tokens.css"` and
  `href="/_ds/fonts.css"`.

## Done When
`mise run check` and `go test -count=1 ./internal/dossier` are green, and the served dossier carries the
`verify ↗ monitor.iscc.codes` tier-2 link, the `monitor instance` identity label, and the
`← Realm index` back-link — with no third-party CDN URL in the body.
