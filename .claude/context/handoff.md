## 2026-06-22 — Thread config-driven instance identity into the hub-dossier masthead

**Done:** The hub-dossier masthead (`GET /<domain>`) now renders this deployment's configured
`dashboard.Identity` instance + operator strings — the SAME value already threaded to the `/`
dashboard — instead of the hard-coded `monitor instance` placeholder, falling back to today's exact
static copy when unset. The dossier and dashboard masthead chrome blocks are now byte-identical.

**Files changed:**
- `internal/dossier/handler.go`: imported `internal/dashboard`; added a `dashboard.Identity` 4th arg to
  `Handler(st, hubID, statuses, id)`; carry resolved `Instance`/`Operator` on `dossierData`; added a
  private `resolveIdentity(dashboard.Identity) (instance, operator string)` fail-safe helper (the
  default-side variant next.md recommended) plus dossier-local `instanceFallback`/`operatorFallback`
  consts duplicated as literals (with a comment that they MUST match the dashboard's — neither package
  can import the other's unexported consts). `buildData` gained `instance, operator` params.
- `internal/dossier/dossier.html`: replaced the static `<span class="chrome-instance">monitor
  instance</span>` (masthead) with the two-line `<div class="chrome-identity"><div
  class="chrome-instance">{{.Instance}}</div><div class="chrome-operator">{{.Operator}}</div></div>`
  block ported verbatim from `dashboard.html`; ported the `.chrome-identity`/`.chrome-operator` CSS so
  the two mastheads are byte-identical.
- `cmd/iscc-monitor/main.go`: threaded the already-constructed `identity()` value into the per-hub
  dossier mount — added the `id dashboard.Identity` param to `mirrorHandler` (which `buildMux` already
  holds `id`) and forwarded it to `dossier.Handler(st, r.HubID, m, id)`. Smaller diff than broadening
  `serveMetrics`; no unrelated handler signature widened. `buildMux`'s own signature is unchanged, so
  the `cmd` tests (which call `buildMux`, not `mirrorHandler`) needed no edits.
- `internal/dossier/handler_test.go` (test, not counted): updated the 8 existing `Handler(...)` call
  sites for the new 4th arg (zero-value `dashboard.Identity{}`), added the `internal/dashboard` import,
  and added `TestDossierRendersInstanceIdentity` (populated path: exact Instance/Operator literals
  present, `monitor instance` placeholder absent; zero-value path: fallback `monitor instance` +
  generic operator line). Realm is intentionally NOT asserted (the dossier has no realm-subtitle slot).

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 27 packages ok).
- `go test -count=1 -run TestDossier ./internal/dossier` → PASS (all existing tests under the new 4-arg
  signature + the new identity test; verbose-confirmed the new test runs).
- `TestDossierRendersInstanceIdentity` non-vacuous, mutation-proven on BOTH halves (reviewer can
  re-run): (a) `{{.Instance}}`→literal `monitor instance` in `dossier.html` → test FAILS (`body missing
  identity literal "monitor.example.test"`), restored byte-clean; (b) `resolveIdentity` forced to drop
  the supplied value (`instance, operator = "", ""`) → test FAILS the same way, restored byte-clean.
  So the test pins BOTH the template binding AND the wired value.
- `gofmt -l .` outside `cauldron/` → empty.
- `go list -deps ./internal/store | grep -E 'net/http|internal/dossier|internal/dashboard'` → empty
  (store stays a leaf; no store change this slice).
- `go.mod`/`go.sum`/`schema.sql` byte-identical (untouched). No new `http(s)://`/`cdn.`/`jsdelivr`
  token; `monitor.iscc.codes` still positively present (`TestDossierRendersCoveredHub` +
  `TestDossierChromeTierTwoAndBackLink` pass). Masthead chrome blocks diffed byte-identical
  dossier-vs-dashboard.
- Scope: exactly 3 production files + 1 test file; nothing from Not-In-Scope touched.

**Next:** Continue the SAME identity arc to `internal/certificate` (`cert.html:391` carries the same
static `monitor instance` placeholder — the lockstep twin). The three mastheads (dashboard / dossier /
certificate) must all end up byte-identical; after this dossier slice, dossier and dashboard already
match, so the cert slice must port the EXACT same `chrome-identity` block + CSS and thread the same
`dashboard.Identity`. On the certificate (the SECOND surface of this arc) ALSO move the three env keys
(`ISCC_MONITOR_INSTANCE`/`ISCC_MONITOR_OPERATOR`/`ISCC_MONITOR_REALM_NAME`) into the `internal/config`
`optional(get, key, fallback)` leaf (ratifying the realm-name key name there) and add them to CLAUDE.md's
env table — that closes the open config-move `normal` issue. After cert: the proofserve surfaces
(`browser.html`, `records.html`, `record.html`). `internal/verifier` stays EXCLUDED (`.codes` chrome).

**Notes:**
- **Fail-safe-helper variant chosen (the next.md default):** I applied the fallback on the dossier side
  via a private `resolveIdentity` + literal consts copied from `dashboard.instanceFallback`/
  `operatorFallback`, rather than exporting `dashboard.resolve` (which would have made
  `internal/dashboard/handler.go` a 4th edited prod file, over budget). The duplicated consts MUST stay
  in sync with `internal/dashboard`'s; this is a documented, commented duplication — review may want to
  file a `low` to consolidate when the masthead-identity arc finishes across all surfaces (a shared
  identity-resolve leaf), but it is NOT a blocker.
- **The unconfigured dossier masthead now renders a SECOND line** (the operator fallback `independent
  Trust & Transparency service · ISCC-Hub network`) it did not show before — this is the
  byte-identical-chrome rule landing: the dossier previously had only the single static `monitor
  instance` line, and now matches the dashboard's two-line block. This is intended per next.md; the
  existing `TestDossierChromeTierTwoAndBackLink` (`monitor instance` present) still passes.
- **Stale comment NOT fixed (out of scope):** `internal/dashboard/dashboard.html`'s `.chrome-identity`
  CSS comment still says "It is static copy in this skeleton (a config-driven identity is a separate
  concern)" — inaccurate since the dashboard became config-driven in `b30b84e`. I wrote an accurate
  comment on the dossier's ported copy but did NOT touch the dashboard (would be a 4th prod file and
  unrelated to this slice). Flagging for a future tidy.
- Oracle/trust-path gate correctly N/A (pure HTML render of masthead strings; no signature / RFC-6962 /
  Merkle / did:web / fsck / proof / store-write path). No `nolint`/`t.Skip`/build-tag/swallowed-error
  introduced.
