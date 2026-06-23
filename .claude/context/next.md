# Next Work Package

## Step: Repoint the dossier's "Browse the log →" at the record-list browser (close the no-JS dead-end)

## Advances
The reopened **M-UI** Verify criterion — "Navigation closure" — and the lone `critical` issue
"Hub-dossier 'Browse the log →' lands on a dead-end checkpoint page" (`issues.md`, `[human]` host-machine
finding). target.md M-UI, "Navigation closure":

> the dossier's **"Browse the log →"** action — which must land on the **log-browser record list**
> (`ISCC Monitor - Log Browser.dc.html`), **never** the `/<domain>/log/` checkpoint-summary /
> proof-link landing, which carries no link to the record list and is therefore a no-JS dead end for a
> human browsing the log.

A `critical` issue preempts everything (issues.md priority semantics), and this is the code-closable,
no-new-wiring half: the dossier handler already knows its `Origin`, so the forward repoint is a
self-contained href change. The record-list *dressing* (part 2 of the critical) needs the hub domain +
`dashboard.Identity` threaded through `proofserve.Handler` and is the explicit follow-on (Not In Scope).

## Goal
Make the dossier's "Browse the log →" action link to `/<domain>/log/records` (the paginated record-list
log browser, where every row already links to its single record) instead of `/<domain>/log/` (the
checkpoint-summary page that carries no link to the record list). This restores the forward leg of the
no-JS navigation chain dossier → record list → single record, which is currently a dead end with
JavaScript disabled.

## Scope
- **Modify**: `internal/dossier/dossier.html` — the single href on the "Browse the log →"
  `action-secondary` anchor (line ~573): `href="/{{.Origin}}/"` → `href="/{{.Origin}}/records"`.
- **Modify** (test, not counted against the ≤3 budget): `internal/dossier/handler_test.go` — the existing
  assertion (~line 319) `href="/sb0.iscc.id/log/">Browse the log →` must become
  `href="/sb0.iscc.id/log/records">Browse the log →`; add the no-JS chain assertion (see Verification).
- **Reference**:
  - `.claude/context/learnings/dossier.md` — the "Browse the log →" link is `/{{.Origin}}/` today;
    masthead + overlay are byte-identical 3× ports — do NOT touch them here.
  - `.claude/context/learnings/http-surface.md` — §"HTML record list at `/records`": `serveRecords` is
    mounted at `/<domain>/log/records` via the inner path switch; each row already links
    `record?index=<seq>` to its single record.
  - `.claude/design/ISCC Monitor - Hub Dossier.dc.html:103` — the mockup's "Browse the log →" →
    `ISCC Monitor - Log Browser.dc.html`, NOT the checkpoint page.
  - `internal/proofserve/handler.go:199-205` — the inner `switch r.URL.Path`: `/records` → `serveRecords`
    (confirms the repointed href resolves to a live 200 record list).

## Not In Scope
- **Dressing `records.html` to the Log-Browser mockup** (part 2 of the critical): the
  `← <hub> dossier` breadcrumb, the eyebrow "Log browser" + hub name + "<domain> · N records mirrored"
  head, the instance-identity + `verify ↗ monitor.iscc.codes` masthead chrome, the top+bottom
  "seq X–Y of Z" range pager (replacing the bottom-only "showing N of M"), and dropping the Status badge
  row the mockup omits. These need the hub **domain string** + `dashboard.Identity` threaded from
  `cmd/iscc-monitor` (`hubHandler` → `mirrorHandler` → `buildMux`) through `proofserve.Handler` →
  `serveRecords`, plus every `Handler(...)`/`buildMux(...)` test callsite — a multi-file follow-on step.
- **The back-breadcrumb leg** (record list → dossier; single record → record list): both `records.html`
  and `record.html` lack any `←` back-link, and a correct dossier back-link needs the threaded domain.
  Add it with the dressing step above.
- Do **not** delete or alter the `/<domain>/log/` checkpoint-summary page
  (`internal/proofserve/browser.html`) — it is M3-functional and CLAUDE.md-documented; this step only
  stops the dossier from *sending a human there as the "log browser"*.
- Do **not** touch the dossier masthead / overlay / `§1–§5` (the 3× duplicated identity+overlay code), or
  any `low` locality issue (skipped by the loop).

## Implementation Notes
- The single source change is the `href` on the `action action-secondary` anchor in `dossier.html`
  (currently `href="/{{.Origin}}/"`). `.Origin` is `<domain>/log` (e.g. `sb0.iscc.id/log`), so
  `/{{.Origin}}/records` renders `/sb0.iscc.id/log/records` — the absolute path `serveRecords` is mounted
  at. Keep the visible text " →" and the `action-secondary` class unchanged; only the path suffix changes
  (`/` → `/records`).
- The dossier test (handler_test.go ~lines 316-319) asserts the OLD dead-end href verbatim; update that
  literal to `/sb0.iscc.id/log/records`. This makes the change mutation-proven — a revert to `/{{.Origin}}/`
  fails the assertion.
- For the no-JS chain assertion, exercise the SAME fixture store through BOTH handlers (the dossier
  `Handler` and a `proofserve.Handler` read the same `*store.Store`). Render the dossier for the seeded
  hub, confirm its "Browse the log →" href equals `/<origin>/records`, then request `GET /records` against
  `proofserve.Handler(st, hubID, nil)` and assert `200 text/html` (a real record list, not a 404 dead-end).
  The dossier test's `coveredHub` helper seeds the hub; if seeding records into that store is awkward in
  the dossier test, the equivalent proof is: assert the dossier href in `internal/dossier/handler_test.go`
  AND rely on the existing `internal/proofserve` `TestRecords*` tests (which already prove `GET /records`
  → 200) — together they show the leg is unbroken.
- M-UI hard constraint (always-loaded): the surface stays "complete with JavaScript disabled" — the
  forward link is a plain `<a href>`, no script. The repointed path is same-origin relative-of-root, so no
  external CDN/host enters the body.
- No crypto/proof/Merkle/did:web path is touched — the oracle/conformance gate is **N/A** (a pure SSR href
  + test-literal change). State the N/A in the advance.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestDossier ./internal/dossier` passes, including the updated href assertion.
- Assertion: the rendered dossier's "Browse the log →" href equals `/<origin>/records` (e.g.
  `/sb0.iscc.id/log/records`); reverting `dossier.html` to `href="/{{.Origin}}/"` makes the dossier test
  FAIL (mutation-proven).
- `go test -count=1 -run TestRecords ./internal/proofserve` passes — the repointed path is live
  (`GET /records` → `200 text/html` record list), so the dossier no longer links to a dead end.

## Done When
`mise run check` is green and the dossier's "Browse the log →" href resolves to the live
`/<domain>/log/records` record list (asserted at the HTTP seam, mutation-proven), closing the no-JS
navigation-closure dead-end that the `critical` issue's part (1) names — leaving the record-list mockup
dressing (part 2) as the next step.
