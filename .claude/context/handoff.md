## 2026-06-22 — Add the dossier's tier-2 `verify ↗ monitor.iscc.codes` chrome link, instance-identity block, and `← Realm index` back-link

**Done:** Ported the shared-chrome masthead actions (`.chrome-actions` / `.chrome-instance` /
`.chrome-verify`) and the `.backlink-row` / `.backlink` `← Realm index` link from `certificate/cert.html`
into `internal/dossier/dossier.html`, so the dossier now carries the static `monitor instance`
identity label, the tier-2 link out to the monitor-agnostic verifier app (`https://monitor.iscc.codes/`),
and a back-link up to the realm index at `/` — making `/` → dossier → log browser fully no-JS
traversable. Template-only production change (no `dossierData` fields added; all three are static
literals, matching the certificate). Narrowed the dossier's no-CDN body ban to third-party CDN hosts +
bare `http://` (mirroring `certificate/handler_test.go:201`) so the one intentional external https
origin passes.

**Files changed:**
- `internal/dossier/dossier.html`: added the chrome `.chrome-actions`/`.chrome-instance`/`.chrome-verify`
  CSS (verbatim from `cert.html:66-88`) and the `.backlink-row`/`.backlink` CSS (from `cert.html:96-104`);
  added the `<div class="chrome-actions">` block (instance label + verify link) inside `<header class="chrome">`,
  and the `<div class="backlink-row">` `← Realm index` link as the first child of `<main class="page">`.
  Updated the `.chrome` comment to note the tier-2 affordance. No new DS token introduced (all `var(--*)`
  already resolved in the dossier).
- `internal/dossier/handler_test.go` *(test)*: narrowed the `TestDossierRendersCoveredHub` no-CDN ban from
  `{"jsdelivr","http://","https://","cdn."}` to `{"jsdelivr","cdn.","unpkg","googleapis","http://"}` (the
  certificate baseline); added `TestDossierChromeTierTwoAndBackLink` asserting `← Realm index` + `href="/"`,
  `monitor instance`, `monitor.iscc.codes` + `href="https://monitor.iscc.codes/"`, and the same-origin
  `/_ds/` DS-shell links.

**Verification:** `mise run check` → all packages `ok` (`go build`/`go vet`/`go test ./...`); `gofmt -l .`
(excl. `cauldron/`) empty. `go test -count=1 ./internal/dossier` → all 9 tests PASS.
Per-criterion:
- [x] `mise run check` green.
- [x] `go test -count=1 ./internal/dossier` green (all prior + new chrome/back-link test).
- [x] Served dossier contains all three literals: `← Realm index`, `monitor.iscc.codes`, `monitor instance`.
- [x] No third-party CDN reference (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`); narrowed ban passes
  for `https://monitor.iscc.codes/`.
- [x] DS shell stays same-origin (`href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`).
- [x] Mutation-proven: replacing the `← Realm index` link text in the template makes
  `TestDossierChromeTierTwoAndBackLink` FAIL; restored → green.

**Next:** The remaining open WASM/M-UI sub-steps from the prior `review` handoff are still open and larger:
(1) the **GitHub-Pages / `monitor.iscc.codes` deploy workflow** for Surface C, which MUST reconcile the
static-deployment gating Codex-P1 (read `?monitor=&id=` client-side, not via server-side `.HasTarget`);
(2) the **WASM verifier-scope signature/id-binding gap** (the verifier core proves inclusion only — no
checkpoint-signature / did:web-key / id-binding check), shared with the certificate's tier-2; (3) the
`safeIndex` WASM test-gap (move it into `verifyadapter`). A small follow-on parity item: the dossier's
honest "Prove an ISCC-ID in this hub →" action (deliberately Not-In-Scope here pending its no-id target
design).

**Notes:**
- This step touches NO signature/RFC-6962/Merkle/did:web/fsck/proof path (pure HTML render of one persisted
  store row + static chrome), so the oracle/conformance gate is N/A — same posture the dossier package
  docstring records.
- The narrowed no-CDN ban is the certificate-established correct fix, not a gate weakening: `monitor.iscc.codes`
  is the one intentional external https origin (per `learnings/web.md` + `learnings/verifier.md`'s `.codes` ≠
  `.id` distinction), so banning every `https://` substring would now be wrong. Bare `http://` and the
  third-party CDN hosts are still banned.
- Instance identity is the static `monitor instance` label (matching the certificate baseline). Making it
  config-driven (domain/operator/realm from env) is the separate filed `normal` issue and is Not-In-Scope.
- No WASM proof island on the dossier — it has no single ISCC-ID subject, so its tier-2 affordance is
  correctly the cross-surface link to Surface C, not a baked-in re-verification (per next.md Not-In-Scope).
- The certificate vs dossier mastheads are now byte-identical in the chrome CSS + actions/back-link markup
  (verbatim port; reviewer can diff cert.html:66-104/380-399 against the dossier).
