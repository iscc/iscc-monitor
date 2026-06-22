## 2026-06-22 — Review of: Thread config-driven instance identity into the hub-dossier masthead

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** A clean, tightly-scoped M-UI slice that makes the hub-dossier masthead honest
per-deployment: it threads the SAME `dashboard.Identity` value already on `/` into
`dossier.Handler(st, hubID, statuses, id)`, renders `{{.Instance}}`/`{{.Operator}}` through a verbatim
port of the dashboard's `chrome-identity` block, and applies a dossier-local `resolveIdentity` fail-safe
so an unconfigured binary renders today's static copy. Scope is exactly 3 production files + 1 test file,
nothing from Not-In-Scope touched, all gates green, the new test is reviewer-mutation-proven non-vacuous
on BOTH the template binding AND the wiring, Codex is clean, and the visual pass confirms the live
masthead matches the dossier mockup's instance-identity region. PASS_WITH_NOTES because two deferred
follow-ups remain tracked (the config-leaf env-key move + the fallback-const duplication), neither
blocking progress.

**Verification:**
- [x] `mise run check` (build + vet + `go test ./...`) — green, all 27 packages ok.
- [x] `go test -count=1 -v -run TestDossier ./internal/dossier` — PASS; the new
  `TestDossierRendersInstanceIdentity` verbose-confirmed to actually run (9 dossier tests all pass).
- [x] Mutation A (reviewer-run, restored byte-clean): `{{.Instance}}`→literal `monitor instance` in
  `dossier.html` → test FAILS ("body still shows the static placeholder despite a configured Instance");
  tree restored byte-clean, HEAD unchanged.
- [x] Mutation B (reviewer-run, restored byte-clean): `resolveIdentity`'s `instance, operator = id.Instance,
  id.Operator` forced to `"", ""` → test FAILS ("body missing identity literal monitor.example.test" +
  the operator literal); tree restored byte-clean. Proves the test pins BOTH the template binding AND the
  wired value (non-vacuous on both halves).
- [x] `gofmt -l .` outside `cauldron/` — empty.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dossier|internal/dashboard'` — empty
  (store stays a leaf; no store change this slice).
- [x] Fallback consts verified byte-identical: dossier `instanceFallback`/`operatorFallback`
  (`handler.go:97-103`) match dashboard `handler.go:113-116` exactly ("monitor instance" /
  "independent Trust & Transparency service · ISCC-Hub network").
- [x] Masthead chrome verified byte-identical: the `.chrome-identity`/`.chrome-instance`/`.chrome-operator`
  CSS rule bodies + the `chrome-actions` HTML block diff byte-identical dossier-vs-dashboard (only the
  explanatory CSS comment differs — the dossier got an accurate one, the dashboard kept its stale one).
- [x] `go.mod`/`go.sum`/`schema.sql` byte-identical (not in the diff); no new `http(s)://`/`cdn.`/`jsdelivr`
  token; `monitor.iscc.codes` still the only external URL and positively present.
- [x] `cmd/iscc-monitor` tests pass under the new `mirrorHandler(..., id)` signature (`buildMux`'s own
  signature unchanged, so cmd tests needed no edits).
- [x] Gate-circumvention scan over all 3 unpushed commits — no `nolint`/`t.Skip`/build-tag/swallowed-error
  in added Go lines; the only match is prose in the handoff text.
- [x] Oracle/trust-path gate — correctly N/A: pure HTML render of masthead strings + one persisted store
  row; no signature / RFC-6962 / Merkle / did:web / fsck / proof / store-write path.
- [x] Scope discipline — exactly 3 prod files (`dossier/handler.go`, `dossier.html`, `cmd/.../main.go`)
  within the ≤3 budget; Not-In-Scope honored (no `internal/certificate`, no `internal/config` env-key move,
  no proofserve, no `internal/verifier`, no back-link/badge/store/tier-2-link change).

**Issues found:**
- (filed `low`) Masthead identity fallback consts duplicated across dashboard + dossier (cert next) — a
  documented, byte-identical, commented duplication chosen to stay ≤3 prod files; consolidate into one
  shared resolve leaf when the masthead arc finishes across all surfaces. Does NOT block.
- (filed `low`) Stale `.chrome-identity` CSS comment in `dashboard.html:75-76` still says "static copy in
  this skeleton" — inaccurate since `b30b84e` made the dashboard config-driven. The advance correctly left
  it out of scope (4th prod file); tidy when `dashboard.html` is next touched.
- The config-move `normal` issue (identity env keys read inline in `main.go`, CLAUDE.md env docs lack them)
  correctly STAYS OPEN — verified the keys are still `os.Getenv` in `main.go:293-307` and `internal/config`
  has no identity keys; the move was deferred to the cert (second-surface) slice by design.

**Codex second opinion:** Clean. Codex: "The identity value is consistently threaded from the mux into the
dossier handler, rendered through html/template with fallbacks matching the dashboard, and existing call
sites/tests were updated. The full Go test suite passes and I did not identify any introduced correctness
issues." No findings to triage; matches my own assessment.

**Visual check:** Performed (agent-browser 0.29.0). Built the binary, launched against the testnet realm
with the three identity env vars set to the mockup values (`monitor.iscc.id` / "instance operated by ISCC
Foundation · ISCC mainnet" / "ISCC mainnet"), and screenshotted `GET /sb0.iscc.id` vs the dossier mockup
(`.claude/design/ISCC Monitor - Hub Dossier.dc.html`). The live masthead instance-identity region matches
the mockup exactly: `monitor.iscc.id` (chrome-instance) over `instance operated by ISCC Foundation · ISCC
mainnet` (chrome-operator), right-aligned before the `verify ↗ monitor.iscc.codes` tier-2 link, same
logo + "TRUST & TRANSPARENCY MONITOR" + "Evidence of record · ISCC-Hub network" left block. Also
curl-confirmed the env → binary → page wiring end-to-end on the real binary. Body deltas (the mockup is a
frozen-Exhibit state with `{{ hub.name }}` placeholders + a "Compiled by … checkpointTime" sub-line; the
live shows a verified, no-coverage hub with the DOMAIN/ORIGIN/STATUS/COVERAGE table) are pre-existing
state/layout differences outside this masthead-only slice. No NEW visual delta from this increment;
nothing filed.

**Next:** Continue the SAME identity arc to `internal/certificate` (`cert.html:391` carries the same static
`monitor instance` placeholder — the lockstep twin). Port the EXACT `chrome-identity` block + CSS and
thread the same `dashboard.Identity`, so all three mastheads (dashboard / dossier / certificate) end
byte-identical. On this SECOND surface ALSO move the three env keys
(`ISCC_MONITOR_INSTANCE`/`ISCC_MONITOR_OPERATOR`/`ISCC_MONITOR_REALM_NAME`) into the `internal/config`
`optional(get, key, fallback)` leaf (ratifying the realm-name key name) and add them to CLAUDE.md's env
table — that closes the config-move `normal` issue. Folding `Identity` + the fallback consts + a single
exported `Resolve` into one shared leaf at that point would also close the new dup `low`. After cert: the
proofserve surfaces (`browser.html`, `records.html`, `record.html`). `internal/verifier` stays EXCLUDED
(`.codes` chrome is the verifier-app identity).

**Notes:**
- The dossier REUSES `dashboard.Identity` (imports `internal/dashboard`) rather than redefining it — the
  right call, but it means the fallback consts are now duplicated (cert will be the third copy). The
  advance commented both const blocks "MUST stay byte-identical"; I verified they currently are. Filed the
  consolidation as a `low`.
- The unconfigured dossier masthead now renders a SECOND line (the operator fallback) it did not show
  before — intended per next.md (byte-identical-chrome rule); `TestDossierChromeTierTwoAndBackLink`
  (`monitor instance` present) still passes.
- `learnings/dashboard.md` net-reduced this iteration (collapsed three settled bullets to one-line
  summaries) to absorb the two new dossier-identity findings and land at 150 lines (was 155).
- Open `normal` issues unchanged by this slice: DB migration (#40); WASM verifier signature half; Pages
  custom-domain binding; realm-index Anchor design-honesty; config-move; `/` recent-declarers footer. None
  preempt the masthead-identity arc; none are touched here.
