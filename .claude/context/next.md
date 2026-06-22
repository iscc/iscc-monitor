# Next Work Package

## Step: Bring `/` realm index to its mockup's named regions — claim-lookup hero, per-row dossier links, instance-identity masthead

## Advances
Closes the lone open **`critical`** (`issues.md` — *"`/` realm index is far below its authoritative
mockup — no claim-lookup hero, rows not linked, masthead chrome absent"*, human-escalated 2026-06-22),
which the handoff `**Next:**` and `target.md`'s human steer both name as the front-loaded blocker. It
directly advances the **M-UI** Verify bar's design-parity named-region requirement for the `/` surface:

> **`/` realm index** — `ISCC Monitor - Realm Index.dc.html`: the **claim-lookup hero** foregrounded
> *above* the register … every row a link to that hub's dossier … the cross-cutting **Document chrome +
> instance identity** (the ISCC logo + instance-identity block + `verify ↗ monitor.iscc.codes`) … and
> **Navigation closure** — the `/` → hub dossier chain fully traversable with JavaScript disabled.

DONE requires 0 `critical`; this is the only one open, so closing it is the highest-priority work.

## Goal
Render the three headline landmark regions the mockup demands on `/` so the realm index resembles its
design and is no longer a no-JS navigation dead-end: a claim-lookup hero (no-JS `GET` form →
`/inclusion/…`), every hub row wrapped in an `<a href>` to its dossier, and the masthead instance-identity
block + `verify ↗ monitor.iscc.codes` tier-2 link.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc files):
  1. `internal/dashboard/dashboard.html` — add the hero `<section>` (eyebrow + plain-language explainer +
     no-JS `GET` form), wrap each `{{range .Hubs}}` row in `<a href="/{{.Domain}}">`, add the masthead
     instance-identity + `verify ↗` block, and the "N hubs followed & mirrored" count. Add page-scoped
     `<style>` rules for the new regions (DS `var(--*)` tokens only).
  2. `internal/dashboard/handler.go` — add the view-model fields the new regions need: `HubCount`
     (`len(rows)`) on `pageData`, and a per-row `RowNo` (1-based, zero-padded to match the mockup `#`
     column) if the `#` column is rendered. The dossier href is `/{{.Domain}}` from the existing `.Domain`
     field — no new store read.
  3. `internal/certificate/handler.go` — when the path id is empty (bare `/inclusion/`), fall back to the
     `?iscc_id=` query param so the dashboard's no-JS `GET` hero form (which can only emit a query string,
     not a path segment) resolves to a real certificate lookup. One small change inside `Handler` before
     `buildData` is called; the whole decode→resolve→render chain is reused unchanged.
- **Reference**:
  - `.claude/design/ISCC Monitor - Realm Index.dc.html` — the authoritative mockup; port the hero copy,
    the masthead instance-identity block, the `#`/Checkpoint/Anchor column structure, and the per-row
    `<a>` wrapper. Constraint wins where the mockup conflicts (self-host, no JS, no CDN), flag the deviation.
  - `internal/certificate/cert.html` (lines 327-341) + `internal/certificate/handler.go` (line 464,
    `rawID := strings.TrimPrefix(r.URL.Path, PathPrefix)`) — the established `.chrome` masthead +
    `verify ↗` pattern to mirror, and the exact id-intake point to add the query fallback.
  - `internal/certificate/handler_test.go` (lines 192-203) — the established no-CDN test posture that bans
    only third-party CDN hosts (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`) while permitting the one
    intentional `https://monitor.iscc.codes/` tier-2 link; the dashboard's no-CDN test must adopt the same
    narrowed ban (see Implementation Notes).
  - `.claude/context/learnings/dashboard.md` — the `min-width:0` grid-ellipsis trap, the `<style>`-stays-
    local rule, the `frozen`-row tint with literal fallback, and the badge-partial render contract.

## Not In Scope
- The same named-region + `←` back-link parity pass across **dossier / log browser / single record**
  (certificate already has its masthead + back-link). Those are the explicit follow-on sub-steps once `/`
  lands — do not touch `internal/dossier` or `internal/proofserve` here.
- The Checkpoint-size and Bitcoin-anchor **data** columns as live values. The mockup shows them, but the
  store summary (`HubSummary`) carries no per-hub checkpoint-size-vs-observed split or OTS anchor state for
  the index, and surfacing them is a store-projection change larger than this step. Render a column's
  structure only if it is honest with existing fields (e.g. `LastSize` as "observed size"); otherwise
  defer the anchor column to a later step and flag the deviation. Do NOT add a store read or column for it.
- The WASM `<script>` caller and the `cmd/wasm/main.go:39-40` `js.Value.Int()` truncation fix — they
  resume after this parity pass (state.md "Subsequent").
- The mandatory ADR-0012 **M-UI exit** visual-pass + human sign-off; `review` runs a per-surface visual
  pass on this changed surface, but the full exit gate is a separate later milestone-close.
- Making the instance identity (domain / operator / realm) configurable via env. Render it as static copy
  matching the mockup for now (the certificate's `monitor instance` placeholder is the precedent); a
  config-driven instance identity is a separate concern.

## Implementation Notes
- **No-JS hero form mechanics (the load-bearing subtlety).** An HTML `<form method="get">` can only emit a
  query string (`?iscc_id=…`), never a path segment, so a form alone cannot produce `/inclusion/<id>`. The
  certificate `Handler` today reads the id ONLY from the path
  (`strings.TrimPrefix(r.URL.Path, PathPrefix)`, handler.go:464). Add a fallback: when that trimmed id is
  empty AND `r.URL.Query().Get("iscc_id")` is non-empty, use the query value as `rawID`. Keep it ahead of
  the `.bundle` CutSuffix so a query id flows through the identical decode→resolve→render chain and a bare
  `/inclusion/` with no id (no query) stays the honest "no id supplied" 200. The hero form is then
  `<form method="get" action="/inclusion/"><input name="iscc_id" …><button>Find evidence →</button></form>`.
- **Per-row dossier link = Navigation closure.** Wrap each row in `<a href="/{{.Domain}}">` (the dossier is
  mounted at the exact bare-domain path `/`+Domain per main.go:50). `html/template` auto-escapes `.Domain`
  in the `href` attribute context. The Verify criterion is `anchor count ≥ hub count`; the masthead
  `verify ↗` and the hero `<button>` add more controls, so the row-link assertion should count
  `href="/sb0.iscc.id"`-style dossier links specifically, not bare `<a `.
- **Masthead + no-CDN tension (do not break the gate the wrong way).** The mockup's
  `verify ↗ monitor.iscc.codes` link points at `https://monitor.iscc.codes/`. The dashboard's current
  `TestDashboardLinksTokensNoCDN` bans `https://` outright (handler_test.go:208), which would reject that
  link. The certificate already resolved this exact tension by narrowing its ban to third-party CDN hosts
  only (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`) while permitting the single intentional
  `https://monitor.iscc.codes/` origin. Mirror that: update the dashboard no-CDN test's banned list to the
  certificate's narrowed set so the masthead link is allowed but real CDNs stay banned. This is the
  established precedent, not a gate weakening — the load-bearing rule (no third-party CDN origin) holds, and
  `learnings/web.md`'s `noExternalCDN` likewise bans third-party origins only, not same-origin/self links.
- **Keep `<style>` page-local (dashboard.md rule).** All new layout rules go in `dashboard.html`'s
  page-scoped `<style>`, expressed through `var(--*)` DS tokens, never in the shared `tokens.css`, so they
  cannot regress another surface. Any new ellipsizing grid cell needs `min-width:0` on the grid item (the
  dashboard.md grid-ellipsis trap).
- **Store stays a leaf.** Add no new store method or read; `RowNo`/`HubCount` derive from the existing
  `summaries` slice in `buildRows`/`Handler`. The dashboard's closure must remain
  `bytes embed html/template net/http internal/store internal/badge` (dashboard.md).
- **Correctness rule (learnings.md, always-loaded):** Origin = `<domain>/log`; this step renders the bare
  `<domain>` as the dossier href and `<domain>/log` (`.Origin`) as the sub-label — keep them distinct
  exactly as today. No proof/crypto/signature path is touched, so the oracle/conformance gate is N/A for
  this increment (pure SSR of persisted rows + a query-param intake on the certificate handler that does
  NOT alter the decode/verify chain).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestDashboard ./internal/dashboard` passes, including a new/updated assertion that
  the served `/` body contains: (a) a hero region with a `<form` whose `action` resolves to `/inclusion/`
  and an `<input name="iscc_id"`, (b) a dossier link per hub — the count of
  `href="/sb0.iscc.id"`+`href="/sb1.amlet.id"`-style links is ≥ the hub count (anchor count ≥ hub count),
  and (c) the masthead carries the instance-identity block + the text `monitor.iscc.codes`.
- `go test -count=1 -run TestCertificate ./internal/certificate` passes, with a new case asserting
  `GET /inclusion/?iscc_id=<golden-id>` certifies the same subject as `GET /inclusion/<golden-id>` (the
  query fallback reuses the decode→resolve chain), and bare `GET /inclusion/` (no id, no query) still
  renders the honest "no id supplied" 200.
- The updated dashboard no-CDN test still bans real CDN hosts (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`) and
  the served body contains none of them, while permitting the `https://monitor.iscc.codes/` link.
- Manual seam check (optional, mirrors the issue's curl): the served `/` HTML has a dossier `<a href>` per
  hub (no longer 0), a hero `GET` form, and the `verify ↗ monitor.iscc.codes` masthead link.

## Done When
`mise run check` is green and the dashboard + certificate tests above pass — proving `/` renders the
claim-lookup hero (no-JS `GET` form → `/inclusion/…`), a dossier link per hub row, and the
instance-identity masthead with the `verify ↗ monitor.iscc.codes` link, closing the lone open `critical`.
