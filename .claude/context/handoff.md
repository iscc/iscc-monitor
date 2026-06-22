## 2026-06-22 — Review of: Add the dossier's tier-2 `verify ↗ monitor.iscc.codes` chrome link, instance-identity block, and `← Realm index` back-link

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance ports the certificate's shared-chrome masthead (`.chrome-actions` /
`.chrome-instance` / `.chrome-verify`) and the `.backlink-row` / `← Realm index` link verbatim into
`internal/dossier/dossier.html`, giving the dossier the static `monitor instance` identity label, the
tier-2 link out to the `.codes` verifier app, and a back-link to the realm index — making `/` → dossier
→ log browser fully no-JS traversable. It is a template-only production change (no new `dossierData`
fields; three static literals) plus the test narrowing the no-CDN body ban to third-party CDN hosts so
`https://monitor.iscc.codes/` (the one intentional external origin) passes. Gates green, scope clean
(1 production + 1 test file), mutation-proven, visual pass and Codex both clean.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` all `ok` (dossier included).
- [x] `gofmt -l .` (excl. `cauldron/`) — empty.
- [x] `go test -count=1 -v ./internal/dossier` — all 14 tests PASS (incl. new `TestDossierChromeTierTwoAndBackLink`).
- [x] Served dossier contains all three literals: `← Realm index`, `monitor.iscc.codes`, `monitor instance` — confirmed by test + live harness probe.
- [x] No third-party CDN reference; narrowed ban passes for `https://monitor.iscc.codes/` — independently probed: the ONLY absolute URL in the served body is `https://monitor.iscc.codes/`, no bare `http://`. The narrowing is the certificate-established correct fix, NOT a gate weakening (a blanket `https://` ban would wrongly reject the legitimately-allowed verifier-app link).
- [x] DS shell stays same-origin (`href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`).
- [x] Verbatim-port claim verified: the dossier chrome/back-link CSS + markup are byte-identical to `cert.html:66-104` / `:390-399` (only the `.chrome-verify` comment is adapted to explain the dossier has no single subject — correct, matches next.md).
- [x] Mutation reproduced: changing the `← Realm index` copy in the template makes `TestDossierChromeTierTwoAndBackLink` FAIL; restored → green.
- [x] Gate-integrity scan over the unpushed diff (`@{upstream}..HEAD`, 4 unpushed commits) — no `//nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in the production diff (the grep hits were all handoff/learnings prose).
- [x] `go.mod`/`go.sum`/schema byte-identical (unchanged). Oracle/conformance gate N/A — pure HTML render of one persisted store row + static chrome; touches no signature/RFC-6962/Merkle/did:web/fsck/proof path.

**Issues found:** (none) — no defect from me or Codex. This iteration resolves no existing `issues.md` entry (additive chrome only, touching no filed-issue code path); no new issue filed.

**Codex second opinion:** Completed (exit 0, ~1 min); CLEAN, no findings: "The change is limited to
adding the dossier chrome/back-link markup and corresponding tests. The new external verifier link is
intentional and the test adjustment matches the established certificate behavior; the full test suite
passes." Agrees with my own review — no triage action needed.

**Visual check:** Done (ADR-0012, `agent-browser` 0.29.0). Built a throwaway in-module harness (mount
`dossier.Handler` over a seeded covered-hub store + `web.Handler` on :43922, since the live testnet
cold-start index is empty), screenshotted the live dossier and the `.dc.html` mockup, removed the
harness (no stray files; tree clean). The three new chrome regions match the mockup's named regions:
top-right `monitor instance` identity label, the `verify ↗ monitor.iscc.codes` tier-2 chip (visually
identical bordered chip), and the `← Realm index` back-link (same position + blue link styling). The
mockup renders a frozen hub (Exhibit) vs the live verified hub — a fixture-state difference, not a
layout delta. No new visual delta worth filing — the only deviations (config-driven instance identity;
mockup Checkpoint/Anchor columns) are already-tracked open `normal` issues (the latter belongs to the
`/` realm-index issue, not the dossier).

**Next:** The remaining open WASM/M-UI sub-steps are larger and still open: (1) the GitHub-Pages /
`monitor.iscc.codes` Surface-C deploy workflow, which MUST reconcile the static-deployment gating
Codex-P1 (read `?monitor=&id=` client-side, not server-side `.HasTarget`); (2) the WASM verifier-scope
signature/id-binding gap (verifier core proves inclusion only — no checkpoint-signature / did:web-key /
id-binding check), shared with the certificate's tier-2; (3) the `safeIndex` WASM test-gap (move it into
`verifyadapter`). A smaller parity follow-on: the dossier's honest "Prove an ISCC-ID in this hub →"
action (deliberately out of scope here pending its no-id target design).

**Notes:**
- Scope was exemplary: 1 production file (`dossier.html`), 1 test file, no `internal/web` import, no
  `handler.go` change, no `go.mod`/`go.sum` change. Well within the ≤3 non-test/doc budget.
- The two mastheads (`cert.html` + `dossier.html`) are now byte-identical in the chrome CSS +
  actions/back-link markup. Learnings updated: added one bullet to `learnings/dashboard.md` (the
  dossier-shares-certificate-masthead rule + the "mirror any masthead edit in both files" reminder);
  no index promotion (the always-loaded rules already cover this; index unchanged at 97 lines,
  dashboard.md now ~123 lines, under the rotation budget).
- No DONE: the WASM milestone is still open (Surface-C deploy, verifier-scope sig/id, safeIndex test)
  and there are open `normal` issues; M1→OTS milestones are not all independently re-confirmed this
  iteration. CONTINUE is correct.
