## 2026-06-22 — Review of: Bring `/` realm index to its mockup's named regions — claim-lookup hero, per-row dossier links, instance-identity masthead

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance brought `GET /` to its authoritative mockup's three headline landmark regions — a
no-JS claim-lookup hero (`<form method="get" action="/inclusion/">` → `iscc_id` query), every hub row
wrapped in an `<a href="/{{.Domain}}">` dossier link (navigation closure), and the masthead
instance-identity block + `verify ↗ monitor.iscc.codes` tier-2 link — plus a symmetric query-param
fallback on the certificate handler so the no-JS form resolves. Scope held to exactly the 3 named source
files + 2 test files; both new tests are mutation-proven non-vacuous; all gates green; the visual pass
confirms faithful named-region parity. This closes the lone open `critical`.

**Verification:**
- [x] `mise run check` — green (all 25 packages `ok`: build + vet + test).
- [x] `go test -count=1 -run TestDashboard ./internal/dashboard` — PASS. Mutation-confirmed non-vacuous:
  reverting the row `<a>` wrapper → `dossier link count = 0`; removing the hero `<section>` → `<form`
  missing. Both fail the new `TestDashboardRendersHeroAndNavigation`.
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — PASS. Mutation-confirmed:
  removing the `?iscc_id=` fallback makes the query request render "no ISCC-ID supplied" → test FAILS.
- [x] Updated no-CDN test still bans real CDN hosts (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`) AND now
  positively asserts `monitor.iscc.codes` is present — the certificate's already-merged precedent, not a
  weakening (the third-party-origin rule still holds; the one permitted external origin is the intentional
  tier-2 verify link).
- [x] `gofmt -l .` (excl. gitignored `cauldron/`) — empty.
- [x] Every new `var(--*)` token resolves in `internal/web/tokens.css` (checked all 39; only
  `--status-error-bg` is the documented pre-existing literal-fallback exception).
- [x] Scope — exactly 3 non-test/doc source files (`dashboard.html`, `dashboard/handler.go`,
  `certificate/handler.go`), as `next.md` named. No Not-In-Scope path touched (no `internal/dossier`,
  `internal/proofserve`, `cmd/wasm`, `internal/config`; no store read/column added).
- [x] Gate-integrity scan over all 3 unpushed commits — no `nolint`/`t.Skip`/swallowed errors/build-tag
  exclusions/deleted assertions/removed tests. Only 2 test functions ADDED.
- [x] Oracle/conformance gate — N/A: pure SSR of persisted rows + a query-param intake on the certificate
  handler that flows through the IDENTICAL existing decode→resolve→render chain. No
  signature/RFC-6962/Merkle/did:web/fsck/proof code touched. `<a>`-wrapping-a-grid-row HTML5 validity
  holds because the `hubStatusBadge` partial has no nested interactive element (verified).

**Issues found:** No new blocking defects. Filed one `normal` `[review]` issue capturing the four deferred
`/` sub-region deltas the parity step left as constraint-wins (no logo asset, static instance identity +
realm name, absent Checkpoint/Anchor data columns, omitted "recent declarers" footer) — the named
sub-steps to finish full `/` design-parity at the M-UI exit. Deleted the resolved `critical` (its (a)/(b)/(c)
Verify criteria are all met).

**Codex second opinion:** Clean (exit 0). Codex independently reviewed the dashboard additions + the
certificate query fallback and the full suite: "The dashboard additions and certificate query fallback are
consistent with the intended routing and existing handler behavior, and the full test suite passes. I did
not find any introduced correctness issues that warrant an actionable finding." No findings to triage;
corroborates the reviewer's own measurements.

**Visual check:** Performed (ADR-0012, `agent-browser` headless). Built + launched the monitor against a
seeded rich-state DB (a verified hub with coverage + observed size 1428, a frozen hub) on `127.0.0.1:43464`;
screenshotted live `/` and the `ISCC Monitor - Realm Index.dc.html` mockup. The three headline landmark
regions render faithfully: masthead (mark + instance-identity block + `verify ↗ monitor.iscc.codes`), the
claim-lookup hero (ISO-24138 eyebrow + identical title/lede + input + "Find evidence →"), and the realm
register (`#`/Hub·domain/Coverage since/Observed size/Status + "N hubs followed & mirrored" + Verified
badge). Remaining deltas are all the constraint-wins/deferred sub-regions (logo, static identity/realm,
Checkpoint/Anchor columns, recent-declarers footer) — filed as the one `normal` issue above, none blocks
this surface's primary function or navigation closure.

**Next:** Carry the same named-region + `←` back-link parity pass to the remaining SSR surfaces
(`internal/dossier`, then `internal/proofserve` log browser / single record), which lack the masthead
instance-identity block and the full back-link chain — the explicit follow-on sub-steps named in this
step's Not-In-Scope. After that, resume the WASM `<script>` caller and the `cmd/wasm/main.go:39-40`
`js.Value.Int()` truncation fix (open `normal` issue).

**Notes:**
- The query-param fallback and the hero form are a LOCKSTEP pair: a `method=get` form can only emit a
  query string, so the certificate handler must accept `?iscc_id=`. Keep them in sync if either moves;
  recorded in `learnings/dashboard.md`.
- The seed program used for the visual pass was a throwaway in `cmd/seedvis_tmp/` (removed immediately
  after; `git status` clean — it never entered a commit). No standalone seed binary exists in-repo; a
  future visual pass must re-seed via the store API (the dashboard `fixtureStore` shape).
- Open issues after this sweep: 0 `critical`, 6 `normal` (two certificate latent fail-opens, the OTS
  stamp guard, the `hubDomain` ForceQuery gap, the WASM shim `Int()` truncation, certificate §6 timestamp,
  plus this new `/` sub-region parity entry), several `low` (loop-skipped). None blocks the next increment.
- Pushing `develop` to `origin` (remote configured, upstream `origin/develop`).
