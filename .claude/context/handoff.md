## 2026-06-22 — Review of: Render config-driven instance identity on the `/` realm-index masthead (skeleton)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** A clean, tightly-scoped M-UI skeleton slice that makes the `/` masthead honest
per-deployment: three operator-supplied strings (`dashboard.Identity{Instance, Operator, Realm}`) flow
env → binary → page, with fail-safe fallback to today's static copy applied INSIDE the handler
(`Identity.resolve()`), so an unconfigured binary renders exactly today's masthead. Scope is exactly 3
production files (handler + template + main) + 2 test files, nothing from Not-In-Scope touched, all gates
green, the new test is mutation-proven non-vacuous on BOTH the template binding and the wiring, Codex is
clean, and the visual pass confirms the masthead now matches the Realm-Index mockup. The one note: the
advance used `ISCC_MONITOR_REALM_NAME` instead of next.md's literal `ISCC_MONITOR_REALM` — a justified
naming correction (the latter is already the required realm-document PATH key), NOT a human-blocking design
deviation (see below). Verdict is PASS_WITH_NOTES because the env keys still need to land in the config
leaf and CLAUDE.md's env docs lack them — both deferred and tracked, neither blocks progress.

**Verification:**
- [x] `mise run check` (build + vet + `go test ./...`) — green, all 28 packages ok.
- [x] `go test -count=1 -run TestDashboard ./internal/dashboard` — PASS (all existing tests under the new
  3-arg `Handler(st, statuses, Identity)` signature + the new identity test; verbose-confirmed the new test
  actually runs).
- [x] New `TestDashboardRendersInstanceIdentity` — populated `Identity` renders all three literals +
  `Realm register · example net` AND the static placeholder is ABSENT; zero-value `Identity{}` renders the
  fallback `monitor instance` + `independent Trust &amp; Transparency service` + the bare
  `<h2 class="ledger-title">Realm register</h2>` with no trailing `·`.
- [x] Mutation A (reviewer-run, restored byte-clean): `{{.Instance}}`→literal `monitor instance` in the
  template → test FAILS (`body missing identity literal "monitor.example.test"` + `body still shows the
  static placeholder`); tree restored byte-clean, HEAD unchanged.
- [x] Mutation B (reviewer-run, restored byte-clean): `resolve()` forced to overwrite `Instance` with the
  fallback (ignoring the supplied value) → test FAILS the same way; tree restored byte-clean. Proves the
  test pins BOTH the template binding AND the wired value (non-vacuous).
- [x] `gofmt -l .` — empty outside `cauldron/`.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard'` — empty (store stays a leaf;
  no store change this slice).
- [x] `TestDashboardLinksTokensNoCDN` + `TestDashboardRendersEveryHub` — still pass (no new
  `http(s)://`/`cdn.`/`jsdelivr`/`<table>`; the identity strings are scheme-less text, `monitor.iscc.codes`
  still positively asserted present).
- [x] Gate-circumvention scan over all 3 unpushed commits — no `nolint`/`t.Skip`/build-tag/swallowed-error
  in added Go lines; the only matches are prose in handoff/context text.
- [x] Oracle/trust-path gate — correctly N/A: pure HTML render of persisted rows + masthead strings; no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path; go.mod/go.sum/schema byte-identical.
- [x] Scope discipline — 3 prod files (`handler.go`, `dashboard.html`, `main.go`) within the ≤3 budget;
  Not-In-Scope honored (other five SSR mastheads, `internal/config`, logo/hero/badge/crypto all untouched).

**Issues found:**
- (filed `normal`) Instance-identity env keys are read inline in `main.go`, not validated via
  `internal/config`, and CLAUDE.md's env-var list lacks the three new keys. Both are deferred-by-design
  (next.md Not-In-Scope, ≤3-file budget) and tracked for the follow-on config-leaf sub-step — they do NOT
  block progress.
- (issue #214 sub-2 CLOSED for `/`) The static-instance-identity sub-region delta on `/` is now resolved;
  retitled #214 to its only remaining sub-item ("recent declarers checked" hero footer, needs store
  history). The SAME `dashboard.Identity` value still needs threading into the other five SSR mastheads —
  that is the follow-on arc, not a defect of this slice.

**Env-var deviation (advance flagged HUMAN REVIEW — reviewer DISMISSES the escalation):** The advance
prepended a "HUMAN REVIEW REQUESTED" over using `ISCC_MONITOR_REALM_NAME` instead of next.md's literal
`ISCC_MONITOR_REALM`. I verified the root cause: `internal/config/config.go:37` already binds
`ISCC_MONITOR_REALM` as the REQUIRED realm-document filesystem PATH (CLAUDE.md documents it too); reading
that key for the masthead would render the document path (e.g. `internal/registry/testdata/realm.txt`) into
the ledger subtitle — a real regression. Picking a distinct, non-colliding optional key is a routine,
correct implementation fix, not a design deviation, public-API break, or wrong target a human must own:
the next.md *design* (config-driven masthead identity, env → binary → page) is delivered exactly; only the
literal env-var *string* changed, for a concrete correctness reason, and the keys are brand-new (no
backward-compat surface) referenced only inside `main.go`. So this is CONTINUE, not STOP. The naming
decision is recorded in `learnings/dashboard.md` + the new config-move issue for the follow-on sub-step to
ratify in `internal/config`.

**Codex second opinion:** Clean. Codex: "The identity threading and template changes are consistent,
existing call sites were updated, and the test suite passes. I did not identify any introduced correctness
issues." No findings to triage; matches my own assessment.

**Visual check:** Performed (agent-browser 0.29.0). Built the binary, launched against the testnet realm
with the three identity env vars set to the mockup values, and screenshotted `GET /` vs the Realm-Index
mockup (`.claude/design/ISCC Monitor - Realm Index.dc.html`). The live masthead renders the configured
identity exactly — `monitor.iscc.id` (chrome-instance), `instance operated by ISCC Foundation · ISCC
mainnet` (chrome-operator), `REALM REGISTER · ISCC MAINNET` (ledger-title, uppercased by CSS) — matching
the mockup's named instance-identity region. Also curl-confirmed the env → binary → page wiring end-to-end
on the real binary. Remaining deltas are all pre-existing tracked items (the "recent declarers checked"
hero footer #214 sub-4; the mockup's illustrative 7-hub count vs the testnet's 2). No NEW delta from this
slice; nothing filed.

**Next:** Continue the same identity arc to the OTHER five SSR mastheads with the SAME `dashboard.Identity`
value, one ≤3-file sub-step each, AND on the SECOND surface move the env parsing into the `internal/config`
`optional(get, key, fallback)` leaf (ratifying the realm-name key name there) so all six surfaces draw from
one validated source — that closes the new config-move issue. Candidate order: `internal/dossier`
(`dossier.html:358`) and `internal/certificate` (`cert.html:391`) first — their mastheads are byte-identical
ports, keep them in lockstep — then the proofserve surfaces (`browser.html`, `records.html`, `record.html`).
`internal/verifier` stays EXCLUDED (its `.codes` chrome is correctly the verifier-app identity). Also
remember to add the three keys to CLAUDE.md's "Running a local dev instance" env table when the config move
lands.

**Notes:**
- The fail-safe default lives in `Identity.resolve()` INSIDE the dashboard package (not main.go), which is
  what makes the fallback golden-testable at the HTTP seam regardless of env. `main.go` passes raw
  `os.Getenv` values; next.md suggested `os.LookupEnv` but `os.Getenv` is equivalent here (a present-but-empty
  var collapses to the same fallback as an unset one). Not a defect.
- Open `normal` issues unchanged by this slice: DB migration (#40); WASM verifier signature half; Pages
  custom-domain binding; realm-index Anchor design-honesty; the new config-move issue; #214 sub-4 (recent
  declarers footer). None preempt the masthead-identity arc and none are touched here.
- `learnings/dashboard.md` net-reduced this iteration (collapsed the settled `inactive`-unreachable and
  frozen-dossier-Exhibit bullets to one-line summaries) to stay at ~150 lines after adding the
  instance-identity + env-key-naming bullets.
