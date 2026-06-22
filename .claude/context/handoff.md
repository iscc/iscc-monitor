## 2026-06-22 — Render config-driven instance identity on the `/` realm-index masthead (skeleton)

**Done:** Replaced the dashboard masthead's hard-coded `monitor instance` / static service copy and the
bare "Realm register" ledger subtitle with three operator-supplied values (instance domain, operator/realm
line, realm name) that flow from environment → the live binary → the rendered `/` page. A new
`dashboard.Identity` view value carries them; `dashboard.Handler` gained an `Identity` parameter and applies
fail-safe defaults INSIDE the handler so an unconfigured binary renders exactly today's masthead.

**Files changed:**
- `internal/dashboard/handler.go`: added exported `Identity struct { Instance, Operator, Realm string }`
  with an `Identity.resolve()` that centralizes the empty-field fallbacks (`instanceFallback` =
  "monitor instance", `operatorFallback` = the generic service line, empty Realm left empty); changed
  `Handler(st, statuses)` → `Handler(st, statuses, id Identity)`; carried `Instance`/`Operator`/`Realm` on
  `pageData`.
- `internal/dashboard/dashboard.html`: templated the three identity text nodes —
  `{{.Instance}}` in `.chrome-instance`, `{{.Operator}}` in `.chrome-operator`, and
  `Realm register{{if .Realm}} · {{.Realm}}{{end}}` in `.ledger-title`. Masthead structure, classes, logo
  `<img>`, and the `verify ↗ monitor.iscc.codes` link byte-unchanged.
- `cmd/iscc-monitor/main.go`: added an `identity()` helper reading three optional env vars via `os.Getenv`,
  threaded `dashboard.Identity` through `serveMetrics` → `buildMux` → the `/` mount. The buildMux/serveMetrics
  signatures gained the `id` param.
- `internal/dashboard/handler_test.go` (test): updated the 6 existing `Handler(...)` call sites to pass
  `Identity{}` (fallback path) and added mutation-proven `TestDashboardRendersInstanceIdentity`.
- `cmd/iscc-monitor/main_test.go` (test): updated the 8 `buildMux(...)` call sites to pass
  `dashboard.Identity{}` and added the `dashboard` import.

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 28 packages ok).
- [x] `go test -count=1 -run TestDashboard ./internal/dashboard` passes (all existing tests under the new
  3-arg signature + new identity test).
- [x] New `TestDashboardRendersInstanceIdentity`: populated `Identity` renders all three literals AND
  `Realm register · example net` AND does NOT show the static placeholder; zero-value `Identity{}` renders
  `monitor instance` + `independent Trust &amp; Transparency service` + the exact bare
  `<h2 class="ledger-title">Realm register</h2>` with NO trailing `·`.
- [x] Mutation A (template): replacing `{{.Instance}}` with the literal `monitor instance` → test FAILS;
  restored byte-clean.
- [x] Mutation B (threading): handler ignoring `id.Instance` (uses `instanceFallback`) → test FAILS;
  restored byte-clean. Proves the test pins BOTH the template binding AND the wired value (non-vacuous).
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard'` empty (store stays a leaf).
- [x] `gofmt -l .` empty outside `cauldron/`.
- [x] `TestDashboardLinksTokensNoCDN` + `TestDashboardRendersEveryHub` still pass (no new `http(s)://` /
  `cdn.` / `jsdelivr` / `<table>`; `monitor.iscc.codes` still positively asserted present).

**Next:** Continue the same arc to the OTHER five SSR mastheads with the SAME `dashboard.Identity` value
(reused), one ≤3-file sub-step each: `internal/dossier` (`dossier.html:358`), `internal/certificate`
(`cert.html:391`), and the proofserve surfaces (`browser.html`, `records.html`, `record.html`).
`internal/verifier` stays EXCLUDED (its chrome is the `.codes` verifier-app identity). When the second
surface lands, move the env parsing into `internal/config`'s `optional(get, key, fallback)` leaf so all six
surfaces draw from one validated source (deferred here only for the ≤3-file budget). The dossier+cert
mastheads are byte-identical ports (learnings/dashboard.md) — keep them in lockstep when templating them.

**Notes:**
- **HUMAN REVIEW REQUESTED — env-var name deviation from `next.md`.** `next.md` Scope named the realm-name
  env var `ISCC_MONITOR_REALM`, but that key is ALREADY taken by `internal/config` as the REQUIRED
  realm-membership-document **filesystem path** (documented in CLAUDE.md and `internal/config/config.go:37`).
  Reading `os.Getenv("ISCC_MONITOR_REALM")` for the masthead would render the document path (e.g.
  `internal/registry/testdata/realm.txt`) into the ledger subtitle in every production deployment — a real
  regression, not the mockup's human realm name. I used a distinct NEW key `ISCC_MONITOR_REALM_NAME` for the
  realm display name instead. `ISCC_MONITOR_INSTANCE` and `ISCC_MONITOR_OPERATOR` are genuinely new, as
  specified. This is the smallest correct fix; flagging because it diverges from the literal env-var name in
  the work package. The follow-on config-leaf sub-step should adopt `ISCC_MONITOR_REALM_NAME` (or pick the
  final name) when these keys move into `internal/config`.
- Defaults are centralized in `Identity.resolve()` in the handler (not main.go), so the HTTP-seam test pins
  the fallback deterministically regardless of env — exactly as `next.md` requested. main.go passes raw
  `os.Getenv` values (empty when unset); the handler owns the fallback.
- Docs: I did NOT update CLAUDE.md's env-var list — `next.md` lists no doc files in scope, and CLAUDE.md's
  env table belongs with the config-leaf integration (deferred). Review may want to add
  `ISCC_MONITOR_INSTANCE` / `ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME` to the "Running a local dev
  instance" section then.
- Oracle/trust-path gate N/A: pure HTML render of persisted rows + masthead strings; no signature / RFC-6962
  / Merkle / did:web / fsck / proof path; go.mod/go.sum/schema byte-identical.
- Production file count: 3 (`handler.go`, `dashboard.html`, `main.go`) — within the ≤3 budget; the two
  `_test.go` files are not counted.
