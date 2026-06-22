# Next Work Package

## Step: Thread config-driven instance identity into the certificate masthead

## Advances
target.md **M-UI — Evidence Ledger frontend**, the design-parity "Document chrome + instance identity"
cross-cutting requirement (held on EVERY surface):

> **Document chrome + instance identity.** Every surface carries the shared handoff header: the ISCC
> logo + "Trust & Transparency Monitor" mark, the **instance-identity** block (this instance's domain +
> operator + realm), and the **`verify ↗ monitor.iscc.codes`** tier-2 link — legible instance identity
> (handoff invariant 9) and the tier-1/tier-2 split present on the page itself.

The `/` realm-index (`b30b84e`) and the hub-dossier (`413efe8`) mastheads already render config-driven
identity. This step continues the SAME identity arc onto the THIRD SSR masthead — the **certificate of
inclusion** (`/inclusion/{iscc_id}`) — exactly as the latest `review` handoff `**Next:**` directs:
"Continue the SAME identity arc to `internal/certificate` (`cert.html:391` carries the same static
`monitor instance` placeholder — the lockstep twin)." It also chips at the open `normal` issue (the
certificate masthead still renders the static `monitor instance` placeholder while `/` and the dossier
are honest per-deployment).

## Goal
Make the certificate masthead render this deployment's configured instance identity (instance domain +
operator/realm line) — the SAME `dashboard.Identity` value already built in `main.go` for `/` and the
dossier — instead of the hard-coded `monitor instance` placeholder, so the certificate chrome is honest
per-deployment and ends byte-identical to the `/` and dossier mastheads it is ported from.

## Scope
- **Create**: (none)
- **Modify** (3 production files):
  - `internal/certificate/handler.go` — add a `dashboard.Identity` argument to `Handler(hubList, st,
    statuses)` (→ `Handler(hubList, st, statuses, id)`); add `Instance`/`Operator` string fields to
    `certData`; apply the SAME fail-safe defaults the dashboard/dossier use so an unconfigured binary
    (or a nil/zero `Identity`) renders today's exact static masthead copy. The masthead renders on EVERY
    path (certifiable AND cannot-certify), and `buildData` has `certData{}` literal early returns (e.g.
    handler.go:667, 689) that would bypass any field set inside `buildData` — so set the identity fields
    in `Handler` on the `certData` value `buildData` RETURNS, on BOTH the HTML path (before
    `tmpl.Execute`) and the `.bundle` path (before `serveBundle`), uniformly covering every branch
    without editing `buildData`'s internals. Resolve `id` once in `Handler` via the ported
    `resolveIdentity` (`instance, operator := resolveIdentity(id)`).
  - `internal/certificate/cert.html` — replace the static `<span class="chrome-instance">monitor
    instance</span>` (line 391) with the two-line `chrome-identity` block ported VERBATIM from
    `dashboard.html`/`dossier.html` (`<div class="chrome-identity"><div
    class="chrome-instance">{{.Instance}}</div><div class="chrome-operator">{{.Operator}}</div></div>`),
    and add the `.chrome-identity`/`.chrome-operator` CSS rules (cert.html already has `.chrome-actions`,
    `.chrome-instance`, `.chrome-verify`) so all three mastheads' CSS rule bodies stay byte-identical
    (the byte-identical-chrome rule).
  - `cmd/iscc-monitor/main.go` — forward the already-constructed `id` into the certificate mount: change
    `certificate.Handler(hubList, st, m)` (`main.go:276`, inside `buildMux`) to
    `certificate.Handler(hubList, st, m, id)`. `buildMux` already receives `id`; no other signature
    broadens.
- **Test files (not counted against the ≤3 budget):** add `TestCertificateRendersInstanceIdentity`
  mirroring `TestDossierRendersInstanceIdentity` (populated path + zero-value fallback path), and update
  any existing certificate test that calls `Handler(hubList, st, statuses)` for the new 4th arg.
- **Reference**:
  - `.claude/context/learnings/certificate.md` — the handler shell pattern (`html/template`,
    `template.Must` at init, buffer-then-200, post-200 write-drop) and the masthead-lockstep note
    ("`cert.html` is the still-pending lockstep twin").
  - `.claude/context/learnings/dashboard.md` — the **config-driven masthead identity** bullets (the
    `dashboard.Identity` value, the `Identity.resolve()` fail-safe that lives INSIDE the package and is
    what makes the fallback seam-testable, the `ISCC_MONITOR_REALM_NAME`-not-`ISCC_MONITOR_REALM` key
    rule, and the explicit "**dossier/dashboard/cert mastheads are byte-identical VERBATIM ports — edit
    all together**" rule); also the no-CDN-ban-narrowed-to-third-party-hosts bullet.
  - `internal/dossier/handler.go:100-127` — the `instanceFallback`/`operatorFallback` const block +
    `resolveIdentity` helper to port VERBATIM (the proven shape; copy its "MUST stay byte-identical"
    comment).
  - `internal/dashboard/dashboard.html:75-99` and `internal/dossier/dossier.html` — the
    `.chrome-identity`/`.chrome-instance`/`.chrome-operator` CSS + the masthead `chrome-identity` block
    to mirror byte-for-byte.
  - `internal/certificate/handler.go:215-242, 487-534, 664-693` — `certData`, `Handler`, and `buildData`
    (note the multiple `certData{}` literal early returns), the existing wiring to extend.
  - `internal/certificate/cert.html:66-99, 363-394` — the certificate chrome region this step edits.

## Not In Scope
- **Do NOT move the three identity env keys into `internal/config` this step.** The combined "cert
  masthead + config-leaf move" the handoff sketched is 4 production files (`config.go` added on top of
  the three above) plus CLAUDE.md docs — over the ≤3-file budget. Keep `cmd/iscc-monitor/main.go`'s
  inline `identity()` reading `os.Getenv` exactly as today. The config-leaf move (ratifying
  `ISCC_MONITOR_REALM_NAME` in `internal/config`'s `optional` leaf + adding the three keys to CLAUDE.md's
  env table) is the dedicated NEXT sub-step — a focused `config.go` + `main.go` + CLAUDE.md change — and
  THAT step closes the config-move `normal` issue. The `normal` stays filed; do not touch CLAUDE.md yet.
- **Consolidating the duplicated `instanceFallback`/`operatorFallback` consts** into one shared
  `Resolve` leaf — that is the tracked `low`, to fold once the arc reaches all surfaces; this step
  copies them a THIRD time (deliberately, with the "MUST stay byte-identical" comment), matching the
  dossier slice's proven pattern. Exporting `dashboard.resolve` would make `internal/dashboard` a 4th
  prod file — over budget.
- Do NOT touch the proofserve surfaces (`browser.html`, `records.html`, `record.html`) — later sub-steps.
- Do NOT touch `internal/verifier` (its `.codes` chrome is correctly the verifier-app identity, EXCLUDED).
- Do NOT change any §1–§6 clause, the proof bundle, the §3 re-verification gate, the badge, the
  `← Realm index` back-link, any store read, or the `monitor.iscc.codes` tier-2 `chrome-verify` link.
- Do NOT invent a certificate "Realm register · <realm>" subtitle — the certificate, like the dossier,
  has no realm-subtitle slot; thread only `Instance`/`Operator`, ignore `id.Realm`.

## Implementation Notes
- **Port, do not invent.** This is the lockstep twin of the dossier slice (`413efe8`). Copy
  `resolveIdentity` and the two const literals from `internal/dossier/handler.go:100-127` VERBATIM
  (keep the "MUST stay byte-identical to dashboard's consts" comment — neither package can import the
  other's unexported consts). The certificate masthead, like the dossier's, has NO Realm subtitle slot —
  thread only `Instance`/`Operator`, ignore `id.Realm`.
- **Set the fields in `Handler`, not in `buildData`.** `buildData` returns fresh `certData{}` literals
  on several early-return branches (no-id, decode-fail, infra-fault), so assigning inside `buildData`
  would miss them. Resolve `id` once at the top of `Handler` (`instance, operator := resolveIdentity(id)`)
  and set `data.Instance = instance; data.Operator = operator` on the value `buildData` returns — on
  BOTH the HTML branch (before `tmpl.Execute`) and the `.bundle` branch (before `serveBundle`) — so every
  honest 200 carries the masthead. (`serveBundle` ignores the identity fields; setting them is harmless
  and keeps the assignment uniform.) Mirror the dossier's choice of resolving in the handler so the
  fallback is seam-testable regardless of env.
- **Byte-identical chrome rule (dashboard.md):** port the `.chrome-identity` CSS
  (`text-align: right; line-height: var(--leading-snug);`) and `.chrome-operator` CSS
  (`font-size: var(--text-xs); color: var(--text-muted); font-weight: var(--weight-light);`) EXACTLY as
  in dashboard.html:77-90, and the two-line block nested inside `chrome-actions` BEFORE `chrome-verify`.
  After this slice all three mastheads (dashboard / dossier / certificate) must be byte-identical.
- **No-CDN ban (web.md / dashboard.md):** the identity strings are scheme-less text — introduce no
  `http(s)://`/`cdn.`/`jsdelivr`/`unpkg`/`googleapis` token. The existing `https://monitor.iscc.codes/`
  tier-2 link is the only allowed external URL and must stay positively present (the cert ban is the
  narrowed third-party-host form, NOT a blanket `https://` ban).
- **html/template auto-escaping holds** — `Instance`/`Operator` are plain strings rendered as text
  nodes; do NOT use `template.HTML`. The operator fallback line's `·` (U+00B7) is a literal UTF-8 middot.
- **Render-into-buffer + 500-before-200** stays exactly as today; you are only adding fields to the
  template context, not changing the error path, the §3 gate, or any store read.
- **Oracle gate is N/A** (certificate.md / dashboard.md): pure HTML render of masthead strings — no
  signature / RFC-6962 / Merkle / did:web / fsck / proof / store path is touched; `go.mod`/`go.sum`/
  `schema.sql` stay byte-identical. State this in the verdict.
- **Correctness rule from the learnings index:** no crypto rule applies (pure render); the load-bearing
  rules are dashboard.md's "fail-safe lives where it is seam-testable" + "keep dashboard/dossier/cert
  mastheads byte-identical".

## Verification
- `mise run check` is green (build + vet + `go test ./...`, all packages ok; `gofmt -l .` empty outside
  `cauldron/`).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes (existing certificate tests
  under the new 4-arg `Handler` signature + the new identity test).
- New `TestCertificateRendersInstanceIdentity`: a populated `Identity{Instance:"monitor.example.test",
  Operator:"operated by Example Org · example net"}` renders BOTH literals in the certificate masthead
  AND the static placeholder `monitor instance` is ABSENT (exercise it on a cannot-certify id so the
  masthead is tested on the honest-200 path); a zero-value `Identity{}` renders the fallback `monitor
  instance` + the generic operator fallback `independent Trust & Transparency service · ISCC-Hub
  network`.
- Mutation check (advance runs + reverts, leaves tree byte-clean): deleting the `{{.Instance}}` binding
  (or the `.chrome-instance` text node) from `cert.html` makes `TestCertificateRendersInstanceIdentity`
  FAIL; forcing `resolveIdentity` to drop the supplied value makes it FAIL; restore byte-clean
  (`git diff` clean, HEAD unchanged).
- `go test -count=1 ./cmd/iscc-monitor` passes (the new `certificate.Handler(hubList, st, m, id)`
  signature compiles; `buildMux`'s own signature is unchanged so cmd tests need no edits).
- The certificate body still contains `monitor.iscc.codes` and contains no new `cdn.`/`jsdelivr`/
  `unpkg`/`googleapis`/`http://` token (no-CDN ban intact).

## Done When
`mise run check` is green and `go test -count=1 -run TestCertificate ./internal/certificate` passes with
a mutation-proven `TestCertificateRendersInstanceIdentity`, so the served certificate masthead renders
the configured `dashboard.Identity` instance + operator strings exactly as the `/` and dossier mastheads
do (falling back to today's static copy when unset), with the config-leaf env move + CLAUDE.md docs and
the remaining proofserve surfaces left for the next sub-steps.
