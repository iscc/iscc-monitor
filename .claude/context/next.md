# Next Work Package

## Step: Render config-driven instance identity on the `/` realm-index masthead (skeleton)

## Advances
target.md **M-UI — Evidence Ledger frontend**, the design-parity "Document chrome + instance identity"
cross-cutting requirement and the `/` realm-index named-region:

> **Document chrome + instance identity.** Every surface carries the shared handoff header: the ISCC logo
> + "Trust & Transparency Monitor" mark, the **instance-identity** block (this instance's domain +
> operator + realm), and the **`verify ↗ monitor.iscc.codes`** tier-2 link — legible instance identity
> (handoff invariant 9) …

and closes sub-item (2) of the open `normal` issue **"`/` realm-index sub-region deltas vs the mockup …
Static instance identity + realm name"** (#214 sub-2) for the `/` surface specifically. The handoff
`**Next:**` and state.md "Next Milestone" #1 both name this as the strongest next candidate: it is the
last cheap **code-only** slice (the remaining backlog is design-first or human-blocked), it unblocks
honest per-deployment chrome, and the identity type it introduces is reused by the five other SSR
mastheads in follow-on sub-steps.

This is a **skeleton-first** slice: the full criterion ("Every surface carries … the instance-identity
block") spans 7 templates + their handler constructors + the binary wiring — far more than one ≤3-file
step. This step lays the verifiable skeleton on the single most-referenced surface (`/`) and lists the
remaining surfaces under `## Not In Scope` so later iterations continue the same arc.

## Goal
Replace the dashboard masthead's hard-coded `monitor instance` / "independent Trust & Transparency
service" placeholder copy with three operator-supplied values (instance domain, operator/realm line,
realm name) flowing from environment → the live binary → the rendered `/` page, and render the realm
name in the ledger subtitle ("Realm register · <realm>"), so the served `/` masthead is honest
per-deployment instead of generic.

## Scope
- **Create**: (none)
- **Modify** (3 production files):
  - `internal/dashboard/handler.go` — add an exported `Identity struct { Instance, Operator, Realm string }`
    view value; change `Handler(st *store.Store, statuses StatusSource)` →
    `Handler(st *store.Store, statuses StatusSource, id Identity)`; carry the three resolved strings on
    `pageData` (e.g. `Instance`, `Operator`, `Realm`). Apply zero-value fallbacks INSIDE the handler (a
    blank field falls back to the current static copy / a generic realm label) so an unconfigured binary
    renders exactly today's masthead — no behavioral regression, fail-safe defaults.
  - `internal/dashboard/dashboard.html` — render `{{.Instance}}` in `.chrome-instance`, `{{.Operator}}`
    in `.chrome-operator`, and the realm name in the `.ledger-title` ("Realm register · {{.Realm}}" — the
    mockup's "Realm register · ISCC mainnet"). Keep the masthead structure, classes, logo `<img>`, and
    the `verify ↗ monitor.iscc.codes` tier-2 link byte-unchanged; only the text nodes become templated.
  - `cmd/iscc-monitor/main.go` — at the dashboard mount (line 271), build an `Identity` from three NEW
    optional env vars read via `os.LookupEnv` with mockup-faithful defaults
    (`ISCC_MONITOR_INSTANCE`, `ISCC_MONITOR_OPERATOR`, `ISCC_MONITOR_REALM`) and pass it to
    `dashboard.Handler(st, m, id)`. Use a tiny local `envOr(key, fallback string) string` helper (or
    inline `os.LookupEnv`) — do NOT add these keys to `internal/config` this step (see Not In Scope).
- **Reference**:
  - `.claude/context/learnings/dashboard.md` — the masthead/no-CDN bullets: the no-CDN ban is NARROWED
    to third-party hosts (so `monitor.iscc.codes` must still pass and is positively asserted); the page
    is a CSS-grid `<ul>` not a `<table>`; render into a `bytes.Buffer` first (500-before-200);
    `html/template` auto-escapes; oracle gate N/A (pure HTML of persisted rows + masthead strings).
  - `.claude/context/learnings/config.md` — why env-parsing belongs in the `internal/config` leaf (the
    `optional(get, key, fallback)` helper pattern); the reason it is DEFERRED here is the ≤3-file budget,
    not a design disagreement — the follow-on sub-step moves these keys into config.
  - `internal/dashboard/handler.go` lines 67-88 (`row` + `pageData`), 99-138 (`Handler`), 147-172
    (`buildRows`) — the existing wiring to extend.
  - `internal/dashboard/dashboard.html` lines 354-389 (the `<header class="chrome">` masthead +
    `.chrome-identity` block + the `.ledger-head`/`.ledger-title` "Realm register" heading).
  - `internal/dashboard/handler_test.go` lines 24-29 (`fakeStatusSource`), 89-115
    (`TestDashboardRendersEveryHub` fixture pattern), 259-308 (`TestDashboardRendersHeroAndNavigation` —
    the masthead-region assertion that currently checks the literal `"monitor instance"`; this test MUST
    be updated to the new call signature + new copy).
  - `.claude/design/ISCC Monitor - Realm Index.dc.html` line 37 (the authoritative identity copy:
    `monitor.iscc.id` / `instance operated by ISCC Foundation · ISCC mainnet`) and line 61 (the ledger
    subtitle `Realm register · ISCC mainnet`).
  - `cmd/iscc-monitor/main.go` lines 113-116 (`config.Load`), 269-277 (`buildMux` where `dashboard.Handler`
    is mounted) — note `buildMux` does NOT currently receive any identity; the cleanest seam is to build
    the `Identity` in `run()` (where `os.LookupEnv` already lives) and thread it through `serveMetrics` →
    `buildMux` → the mount, OR build it directly at the mount. Pick the smaller diff; do NOT broaden
    other handler signatures.

## Not In Scope
- Do NOT thread instance identity into the OTHER five SSR mastheads this step — `internal/dossier`
  (`dossier.html:358`), `internal/certificate` (`cert.html:391`), and the proofserve surfaces
  (`browser.html`, `records.html`, `record.html`). They keep their static `monitor instance` copy for now;
  wiring each is its own ≤3-file follow-on sub-step (same `Identity` value, reused). `internal/verifier`
  is EXCLUDED entirely — its chrome is correctly the `.codes` verifier-app identity, not an instance
  (see `learnings/verifier.md`), and must stay static.
- Do NOT add the three env keys to `internal/config` this step (it would push the diff to 4 production
  files). Reading them inline in `main.go` with defaults is the skeleton; the follow-on sub-step that
  threads identity to the remaining surfaces SHOULD move parsing into `internal/config` (the
  `optional(get, key, fallback)` leaf) so all six surfaces draw from one validated source.
- Do NOT touch the "recent declarers checked" hero footer (#214 sub-4 — needs a store lookup history that
  does not exist), the Checkpoint/Anchor columns (already closed), the per-hub-vs-per-checkpoint Anchor
  design question (issue:326), the logo, the hero form, the badge partial, pagination, or any proof/crypto
  path. This is a masthead-text render + one env-wiring slice only (oracle gate N/A).
- Do NOT change the masthead's logo `<img>`, the tier-2 `verify ↗ monitor.iscc.codes` link, or any CSS —
  only the three identity text nodes + the ledger subtitle text become templated.

## Implementation Notes
- **Fail-safe defaults, mockup-faithful.** The mockup copy is the default so an unconfigured dev binary
  still renders a sensible masthead. Suggested defaults (apply in the handler when the field is empty, so
  the fallback is centralized and golden-testable independent of `main.go`):
  `Instance` → today's `"monitor instance"` (keep the existing placeholder so an unset deploy is honest,
  NOT a false `monitor.iscc.id` claim); `Operator` → today's
  `"independent Trust & Transparency service · ISCC-Hub network"`; `Realm` → `""` → render the bare
  "Realm register" subtitle (no `· <realm>` suffix) when empty. main.go's defaults can be the same
  strings, OR main.go can pass the mockup values for the live testnet — but the HANDLER's empty-field
  fallback is what the HTTP-seam test pins, so behavior is deterministic regardless of env.
- **Conditional realm suffix.** Render the ledger title as `Realm register{{if .Realm}} · {{.Realm}}{{end}}`
  so an empty realm renders exactly today's "Realm register" (no trailing separator) — a coverage-honesty
  /no-empty-affordance discipline, and it keeps the existing `TestDashboardRendersHeroAndNavigation`
  "Realm register"-adjacent assertions stable.
- **Signature change is contained.** `dashboard.Handler` has exactly ONE production call site
  (`main.go:271`) and SIX test call sites (`handler_test.go` lines 94, 201, 231, 262, 313, 322). Update
  all of them: the existing tests pass a zero-value `Identity{}` (proving the fallback path); the new
  test passes a populated `Identity`. Tests are not counted against the ≤3-file budget.
- **html/template auto-escaping holds.** `Instance`/`Operator`/`Realm` are plain strings rendered as text
  nodes — `html/template` escapes them, so an operator who sets `&`/`<` cannot break the page. Do NOT use
  `template.HTML`. The mockup's operator line contains `·` (U+00B7) — render it as a literal UTF-8
  middot in the default string (the file is UTF-8), matching the existing `&middot`-free copy.
- **No-CDN ban unaffected.** The identity strings are scheme-less text; the only `https://` in the body
  stays the masthead's `monitor.iscc.codes` tier-2 link, which the narrowed ban positively allows.
  `TestDashboardLinksTokensNoCDN` must still pass — do not let a default value introduce an `http(s)://`
  / `cdn.` / `jsdelivr` substring.
- **Render-into-buffer + 500-before-200** stays exactly as today; you are only adding fields to the
  template context, not changing the error path.
- **Mutation-proof the render.** The new test must FAIL if the identity text node is dropped from
  `dashboard.html` AND if the handler stops threading the value — assert the EXACT operator-supplied
  strings appear in the body (not the static default), driven from a populated `Identity{}` in the test,
  so reverting the template binding makes it fail.

## Verification
- `mise run check` is green (build + vet + `go test ./...` all pass; `gofmt -l .` empty outside
  `cauldron/`).
- `go test -count=1 -run TestDashboard ./internal/dashboard` passes (all existing dashboard tests under
  the new `Handler(st, statuses, Identity{})` signature + the new identity-render test).
- New test (e.g. `TestDashboardRendersInstanceIdentity`): `Handler(st, nil, Identity{Instance:
  "monitor.example.test", Operator: "operated by Example Org · example net", Realm: "example net"})`
  renders a `/` body containing all three literals AND `Realm register · example net`; a zero-value
  `Identity{}` renders the fallback `monitor instance` + the bare `Realm register` (no trailing `·`).
- Mutation check (run + revert, leave tree byte-clean): deleting the `{{.Instance}}` binding (or the
  `.chrome-instance` text node) from `dashboard.html` makes the new test FAIL; restore byte-clean
  (`git diff` clean, HEAD unchanged).
- `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard'` is empty (store stays a leaf
  — no store change).
- The body adds no `http://` / `https://` (other than the existing `monitor.iscc.codes`) / `cdn.` /
  `jsdelivr` and no `<table>` (`TestDashboardLinksTokensNoCDN` + `TestDashboardRendersEveryHub` still
  pass).

## Done When
`mise run check` is green and `go test -count=1 -run TestDashboard ./internal/dashboard` passes with a
mutation-proven `TestDashboardRendersInstanceIdentity`, so the served `/` masthead renders the
operator-supplied instance / operator / realm identity (from `ISCC_MONITOR_INSTANCE` /
`ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM`) and falls back to today's static copy when unset.
