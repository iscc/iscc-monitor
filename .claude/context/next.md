# Next Work Package

## Step: Thread config-driven instance identity into the hub-dossier masthead

## Advances
target.md **M-UI — Evidence Ledger frontend**, the design-parity "Document chrome + instance identity"
cross-cutting requirement (held on EVERY surface):

> **Document chrome + instance identity.** Every surface carries the shared handoff header: the ISCC
> logo + "Trust & Transparency Monitor" mark, the **instance-identity** block (this instance's domain +
> operator + realm), and the **`verify ↗ monitor.iscc.codes`** tier-2 link — legible instance identity
> (handoff invariant 9) and the tier-1/tier-2 split present on the page itself.

The `/` realm-index masthead already renders config-driven identity (`b30b84e`). This step continues the
SAME identity arc onto the next SSR surface — the **hub dossier** — exactly as the review handoff
`**Next:**` directs: "Continue the same identity arc to the OTHER five SSR mastheads with the SAME
`dashboard.Identity` value, one ≤3-file sub-step each … `internal/dossier` (`dossier.html:358`) … first".
It also chips at the open `normal` issue (the dossier masthead still renders the static `monitor
instance` placeholder while the rest of the network chrome is honest per-deployment).

## Goal
Make the hub-dossier masthead render this deployment's configured instance identity (instance domain +
operator/realm line) — the same `dashboard.Identity` value already built in `main.go` for the `/`
dashboard — instead of the hard-coded `monitor instance` placeholder, so the dossier chrome is honest
per-deployment and stays in lockstep with the `/` masthead it was ported from.

## Scope
- **Create**: (none)
- **Modify** (3 production files):
  - `internal/dossier/handler.go` — add a `dashboard.Identity` argument to `Handler(st, hubID,
    statuses)` (→ `Handler(st, hubID, statuses, id)`); carry the resolved `Instance`/`Operator` strings
    on `dossierData`; apply the SAME fail-safe defaults the dashboard uses so an unconfigured binary (or a
    nil/zero `Identity`) renders today's exact static masthead copy.
  - `internal/dossier/dossier.html` — replace the static `<span class="chrome-instance">monitor
    instance</span>` (line 358) with the two-line `chrome-identity` block ported VERBATIM from
    `dashboard.html` (`<div class="chrome-identity"><div class="chrome-instance">{{.Instance}}</div><div
    class="chrome-operator">{{.Operator}}</div></div>`), porting the `.chrome-identity`/`.chrome-operator`
    CSS too so the dossier and dashboard mastheads stay byte-identical (the byte-identical-chrome rule).
  - `cmd/iscc-monitor/main.go` — forward the already-constructed `identity()` value into the per-hub
    `dossier.Handler(...)` mount (`main.go:367`, currently `dossier.Handler(st, r.HubID, m)`). The value
    is the SAME `dashboard.Identity` already threaded to `dashboard.Handler`; thread it through to the
    dossier mount (via `serveMetrics`/`buildMux`, which already receive `id`, or build it once and pass
    it — pick the smaller diff, do NOT broaden unrelated handler signatures).
- **Test files (not counted against the ≤3 budget):** add `TestDossierRendersInstanceIdentity` mirroring
  `TestDashboardRendersInstanceIdentity` (populated path + zero-value fallback path), and update any
  existing dossier test that calls `Handler(st, hubID, statuses)` for the new 4th arg.
- **Reference**:
  - `.claude/context/learnings/dashboard.md` — the **config-driven masthead identity** bullet (the
    `dashboard.Identity` value, the `Identity.resolve()` fail-safe that lives INSIDE the package and is
    what makes the fallback seam-testable, the `ISCC_MONITOR_REALM_NAME`-not-`ISCC_MONITOR_REALM` key
    rule, and the explicit "**dossier masthead chrome is a VERBATIM port of cert.html — keep the two
    byte-identical**" rule); also the no-CDN-ban-narrowed-to-third-party-hosts bullet.
  - `internal/dashboard/handler.go:84-130, 175-181` — the `Identity` struct, `resolve()` fail-safe +
    fallback consts, and how the dashboard threads `Instance`/`Operator` onto its view-model (the port
    source).
  - `internal/dashboard/dashboard.html:73-90, 360-368` — the `.chrome-identity`/`.chrome-instance`/
    `.chrome-operator` CSS + the masthead `chrome-identity` block to port verbatim.
  - `internal/dossier/handler.go:63-74, 114-155, 177-179` — `dossierData`, `Handler`, and `buildData`,
    the existing wiring to extend.
  - `internal/dossier/dossier.html:66-90, 355-372` — the dossier chrome region this step edits.

## Not In Scope
- **Do NOT thread identity into `internal/certificate` / `cert.html` this step.** Cert is the lockstep
  twin (cert.html:391 carries the same static placeholder) and is the very NEXT sub-step, but adding it
  here would touch 5 production files. Flag in the handoff that certificate is the next surface and the two
  mastheads must end up byte-identical.
- **Do NOT move the three identity env keys into `internal/config` this step.** The handoff schedules the
  config-leaf move for the SECOND surface in this arc; this dossier step is the FIRST. Keep `identity()`
  reading `os.Getenv`/`os.LookupEnv` inline in `main.go` exactly as today (the open config-move `normal`
  stays filed; do not also touch CLAUDE.md's env table yet — that lands with the config move).
- Do NOT touch the proofserve surfaces (`browser.html`, `records.html`, `record.html`) — later sub-steps.
- Do NOT touch `internal/verifier` (its `.codes` chrome is correctly the verifier-app identity, EXCLUDED).
- Do NOT change the dossier's `← Realm index` back-link, the badge partial, the Exhibit, any store read,
  or the `monitor.iscc.codes` tier-2 `chrome-verify` link.
- Do NOT invent a dossier "Realm register · <realm>" subtitle — the dossier's title is "Hub dossier",
  which has no realm-subtitle slot; thread only `Instance`/`Operator` here, leave `Realm` to surfaces that
  have a realm subtitle.

## Implementation Notes
- **Reuse `dashboard.Identity`; do not redefine it.** Import `internal/dashboard` in the dossier handler
  and accept a `dashboard.Identity` argument — the dashboard already owns the type, its fail-safe
  defaults, and the fallback consts, so this avoids a second copy of the struct/defaults.
- **Keep the fail-safe seam-testable while staying ≤3 prod files.** `dashboard.resolve()` is unexported.
  Exporting it (`resolve`→`Resolve`, updating dashboard's own call site) would make `internal/dashboard/
  handler.go` a FOURTH edited prod file — over budget. So apply the fallback on the dossier side instead:
  add a tiny private dossier helper (e.g. `resolveIdentity(dashboard.Identity) (instance, operator
  string)`) that mirrors `resolve()` semantics — blank `Instance`/`Operator` → the dashboard's static
  fallback strings (reference them as literals copied from `dashboard.instanceFallback`/`operatorFallback`,
  with a one-line comment that they MUST match the dashboard's, since both can't import the other's
  unexported consts). This keeps the fallback testable at the dossier HTTP seam (a zero-value `Identity{}`
  must render today's static copy) without touching `internal/dashboard`. Document the chosen variant in
  the advance commit. (If exporting `Resolve` turns out to keep the dossier diff smaller AND the dashboard
  edit is genuinely trivial — a pure rename — the advance MAY take that route and note it pushed to a 4th
  prod file with justification; default is the dossier-local helper.)
- **Byte-identical chrome rule (dashboard.md):** port the `.chrome-identity` CSS + the two-line
  `chrome-instance`/`chrome-operator` block from `dashboard.html` EXACTLY (same class names, same nesting
  inside `chrome-actions` BEFORE `chrome-verify`). When certificate is wired next, all three mastheads
  must match.
- **No-CDN ban (web.md / dashboard.md):** the identity strings are scheme-less text — do not introduce any
  `http(s)://`/`cdn.`/`jsdelivr` token. The existing `https://monitor.iscc.codes/` tier-2 link is the only
  allowed external URL and must stay (the dossier ban is the narrowed third-party-host form, NOT a blanket
  `https://` ban; keep `monitor.iscc.codes` positively asserted).
- **html/template auto-escaping holds** — `Instance`/`Operator` are plain strings rendered as text nodes;
  do NOT use `template.HTML`. The operator line's `·` (U+00B7) is a literal UTF-8 middot in the default.
- **Render-into-buffer + 500-before-200** stays exactly as today; you are only adding fields to the
  template context, not changing the error path or any store read.
- **Oracle gate is N/A** (dashboard.md): pure HTML render of masthead strings — no signature / RFC-6962 /
  Merkle / did:web / fsck / proof / store path; `go.mod`/`go.sum`/`schema.sql` stay byte-identical. Do not
  touch any trust-root glob.
- **Correctness rule from the learnings index:** no crypto rule applies (pure render); the load-bearing
  rules are dashboard.md's "fail-safe lives where it is seam-testable" + "keep dossier/cert mastheads
  byte-identical".

## Verification
- `mise run check` is green (build + vet + `go test ./...`, all packages ok; `gofmt -l .` empty outside
  `cauldron/`).
- `go test -count=1 -run TestDossier ./internal/dossier` passes (existing dossier tests under the new
  4-arg `Handler` signature + the new identity test).
- New `TestDossierRendersInstanceIdentity`: a populated `Identity{Instance:"monitor.example.test",
  Operator:"operated by Example Org · example net"}` renders BOTH literals in the dossier masthead AND the
  static placeholder `monitor instance` is ABSENT; a zero-value `Identity{}` renders the fallback `monitor
  instance` + today's generic operator line.
- Mutation check (advance runs + reverts, leaves tree byte-clean): deleting the `{{.Instance}}` binding
  (or the `.chrome-instance` text node) from `dossier.html` makes `TestDossierRendersInstanceIdentity`
  FAIL; restore byte-clean (`git diff` clean, HEAD unchanged).
- The dossier no-CDN test still passes (no new `cdn.`/`jsdelivr`/`http(s)://`; `monitor.iscc.codes` still
  positively present).
- `go list -deps ./internal/store | grep -E 'net/http|internal/dossier|internal/dashboard'` is empty
  (store stays a leaf; no store change this slice).

## Done When
`mise run check` is green and `go test -count=1 -run TestDossier ./internal/dossier` passes with a
mutation-proven `TestDossierRendersInstanceIdentity`, so the served hub-dossier masthead renders the
configured `dashboard.Identity` instance + operator strings exactly as the `/` masthead does (falling
back to today's static copy when unset), with certificate and the remaining surfaces left for the next
sub-steps.
