# Next Work Package

## Step: Dress the record-list log browser with the shared chrome + `← dossier` breadcrumb + the "Log browser" head (record-list parity, part 2a)

## Advances
The reopened M-UI Verify criterion + the open `critical` ("Hub-dossier 'Browse the log →' lands on a
dead-end checkpoint page; the record-list log browser is off-mockup"). Specifically it closes the
**document-chrome / instance-identity** and **navigation-back-link** + **head** landmark regions of
target.md M-UI's "log browser / record list" bar, quoting target.md:

> **log browser / record list** — `ISCC Monitor - Log Browser.dc.html`: … it carries the shared
> document chrome with instance identity + the `verify ↗` link. Its named regions: `← <hub> dossier`
> back-link; the head (eyebrow "Log browser", hub name, "<domain> · N records mirrored"); …

This is part (2) of the `critical` the steer reopened — the named-region dressing follow-on to the
already-implemented (uncommitted) forward-repoint half (part 1). It preempts all other milestone work
because the `critical` outranks everything (issues.md priority semantics) and the rest of the backlog
is design/human-blocked `normal`s + skipped `low`s. (Part 1 — the uncommitted `dossier.html` href +
test — is `review`'s to gate; it touches no file this step touches.)

## Goal
Thread the hub domain + the operator-supplied `dashboard.Identity` through `proofserve.Handler` →
`serveRecords`, and dress `records.html` to render the Log-Browser mockup's shared masthead chrome
(instance identity + `verify ↗ monitor.iscc.codes` link), the `← <hub> dossier` breadcrumb, and the
"Log browser" / hub name / "<domain> · N records mirrored" head — so the record list stops being an
off-mockup orphan and becomes the navigable, instance-identified surface the dossier sends a human to.

## Scope
- **Create**: (none)
- **Modify**:
  - `cmd/iscc-monitor/main.go` — pass the hub `Domain` + the `dashboard.Identity` into
    `proofserve.Handler` (via `hubHandler`, which already has both in hand in `mirrorHandler`).
  - `internal/proofserve/handler.go` — extend the `Handler` factory signature with `domain string,
    id dashboard.Identity`; resolve the masthead identity once at construction (mirror dossier's
    `resolveIdentity` + `instanceFallback`/`operatorFallback` literals); add `Domain`, `Instance`,
    `Operator` fields to `recordsData` and populate them in `serveRecords`.
  - `internal/proofserve/records.html` — add the chrome's right-side instance-identity block +
    `verify ↗ monitor.iscc.codes` link; add the `← {{.Domain}} dossier` breadcrumb above the head;
    replace the generic "Records" `<h1>` with the eyebrow "Log browser" + `{{.Domain}}` head + the
    "<domain> · N records mirrored" sub-line (count = `{{.Total}}`). No external/CDN URL in the body.
- **Reference**:
  - `.claude/design/ISCC Monitor - Log Browser.dc.html` (authoritative for layout/affordances; lines
    30-56 = chrome + breadcrumb + head; subordinate to no-JS / no-CDN / self-host / grayscale-safe).
  - `internal/dossier/handler.go:148-211` — the proven `resolveIdentity` + `instanceFallback`/
    `operatorFallback` + `Handler(st, hubID, statuses, id dashboard.Identity)` pattern to mirror.
  - `internal/dossier/dossier.html` — the already-dressed chrome masthead (instance identity +
    `verify ↗` link) to port the markup from (same DS shell, same `/_ds/iscc-logo-black.png`).
  - `.claude/context/learnings/http-surface.md` — the `/records` (`serveRecords`) settled notes:
    buffer-then-200, unquoted `[data-status=…]`/`[data-kind=…]` CSS, store-leaf purity, no-CDN body ban.
  - `.claude/context/learnings/dossier.md` — masthead/overlay-port discipline ("edit any masthead →
    mirror all three SSR HTML files"; the now-3× `resolveIdentity` duplication is a tracked `low`).
  - `cmd/iscc-monitor/main.go:378-433` (`mirrorHandler` / `hubHandler`) — where `r.Domain` + `id` are
    already in hand and `proofserve.Handler` is constructed (`proofs := proofserve.Handler(...)`).

## Not In Scope
- **The pager rework is a SEPARATE follow-on slice (part 2b).** Do NOT touch the pager this step:
  leave the existing bottom-only "showing N of M" pager as-is. The mockup's top+bottom pagers with the
  "seq X–Y of Z" range (disabled at the ends) is its own ≤3-file step once the chrome/head land.
- **The single-record page chrome.** `record.html` / `serveRecord` keep their current chrome this step;
  threading identity into the single-record surface (and its `← Log browser` breadcrumb) is a later slice.
- **Consolidating the now-4×-duplicated `resolveIdentity` / `instanceFallback`** into one shared leaf —
  that is the tracked `low` ("Masthead identity fallback consts … 3x"); copy the literals as dossier
  does (with the "MUST stay byte-identical" comment) and let the consolidation `low` cover it.
- **Re-dressing `browser.html`** (the `/<domain>/log/` checkpoint-summary page) — it stays the
  M3-functional proof-surface landing; this step touches only the `/records` record list.
- **Touching the uncommitted part-(1) advance** (`dossier.html` href + `dossier handler_test.go`) — it
  is `review`'s to gate and does not overlap this step's files.

## Implementation Notes
- **Signature ripple (the main mechanical cost):** adding `domain, id` to `Handler` breaks every
  positional `Handler(...)` call in all 8 proofserve `*_test.go` files (~30 sites: `browser_test.go`,
  `consistency_test.go`, `entries_test.go`, `handler_test.go`, `ots_test.go`, `record_test.go`,
  `records_test.go`, `verify_test.go`). Update them all to pass a test domain (e.g. `"sb0.iscc.id"`)
  and a zero-value `dashboard.Identity{}` (which `resolveIdentity` maps to the static fallback copy, so
  every existing assertion that does not check identity stays green). This is mechanical; tests are
  excluded from the ≤3-file non-test budget.
- **Mirror dossier exactly.** Add the `internal/dashboard` import to `proofserve` for the `Identity`
  type only — it is a plain data struct, so it pulls no `net/http`/`database/sql` into the closure
  (confirm with the dep-list check in Verification). Resolve identity ONCE at `Handler` construction
  (outside the request closure), like dossier, not per request.
- **Hub name = the domain (constraint-win).** The mockup head shows a human "hub name" (`hub.name`,
  "ISCC Foundation Hub") above the domain, but the store has no display name — `HubSummary` carries
  only `Domain`/`Origin`, and the dossier itself renders the bare domain as both the `<h1>` and the
  sub-line. So use `{{.Domain}}` as the head name and `← {{.Domain}} dossier` as the breadcrumb; do
  NOT fabricate a prettified name. Flag this as a constraint-win deviation in a code comment (the
  mockup's display name has no store source — same deviation the dossier already made).
- **Breadcrumb target is an ABSOLUTE site-root path, not relative.** The record rows use relative
  `record?index=` because they share the `/<origin>/` (`/<domain>/log/`) subtree, but the dossier is
  mounted at the SITE ROOT `/<domain>` (e.g. `/sb0.iscc.id`), OUTSIDE the `/log/` subtree. So the
  breadcrumb must be `href="/{{.Domain}}"` (leading slash), never a relative `../` walk.
- **No-CDN body ban.** Keep the chrome markup token-driven (`var(--*)`), same DS shell as
  `dossier.html`; the `verify ↗` link href is the literal `https://monitor.iscc.codes` — the `.codes`
  verifier app is same-federation, not a third-party asset CDN, and the existing dossier/browser bodies
  already carry this exact link past their no-CDN asserts, so the ban tolerates it. Sanity-check against
  the existing `TestBrowserLinksTokensNoCDN` / `TestRecordsLinksTokensNoCDN` pattern.
- **Honesty rule (learnings.md):** assert nothing the data does not support. "N records mirrored" is
  `{{.Total}}` (the already-honest accepted-tree-capped total `serveRecords` computes); render "0
  records mirrored" honestly for an empty index (the empty-state block still wins below). No new store
  read is needed — `Total` already exists on `recordsData`.
- **Store stays a leaf.** Template/view-model + wiring only; no `store` change, no new query, no
  schema/migration. `go.mod`/`go.sum` stay byte-identical.
- **Oracle/conformance gate is N/A** — pure SSR chrome + view-model threading; no signature, RFC-6962,
  Merkle, did:web, or proof path is touched. State the N/A in the advance.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 -run 'TestRecords|TestBrowser|TestRecord|TestInclusion|TestConsistency|TestEntries|TestOTS|TestVerify' ./internal/proofserve` passes (all 8 test files compile + pass under the new `Handler` signature).
- A new/extended record-list test asserts the served `/records` body contains the breadcrumb
  `href="/sb0.iscc.id"` (the `← <hub> dossier` link), the eyebrow text `Log browser`, the head domain
  `sb0.iscc.id`, and the `records mirrored` head copy — and is mutation-proven (removing the breadcrumb
  OR the "Log browser" eyebrow from `records.html` FAILS it).
- A new/extended test asserts the served `/records` body contains the instance-identity masthead with
  the `monitor.iscc.codes` verify link (proving the shared chrome with instance identity is present)
  AND still contains no third-party CDN host in the body (the existing no-CDN assertion stays green).
- `go list -deps ./internal/proofserve | grep -E '^(internal/metrics|database/sql)$'` is empty
  (proofserve did not pull a forbidden dependency by importing `dashboard.Identity`).

## Done When
`proofserve.Handler` threads the hub domain + `dashboard.Identity`, the served `/records` page renders
the Log-Browser mockup's shared chrome (instance identity + `verify ↗`), the `← <hub> dossier`
breadcrumb, and the "Log browser" / hub / "<domain> · N records mirrored" head — each region pinned by
a mutation-proven HTTP-seam test — with `mise run check` green and the pager rework left for part 2b.
