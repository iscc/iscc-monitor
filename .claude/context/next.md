# Next Work Package

## Step: `/` realm-index — add the per-hub Anchor projection + render the mockup's Checkpoint & Anchor ledger columns

## Advances
M-UI design-parity "named-region" bar — the `/` realm-index region. The mockup
(`.claude/design/ISCC Monitor - Realm Index.dc.html:65-66`) specifies a six-column ledger
`# | Hub · domain | Coverage since | Checkpoint | Anchor | Status`; the live page renders only five
(`# | Hub · domain | Coverage since | Observed size | Status`). This step closes sub-delta (3) of the
open **`normal`** "`/` realm-index sub-region deltas vs the mockup (… Checkpoint/Anchor columns)"
(issues.md:214) — the largest remaining code-closable slice of that issue. The state's Next Milestone
names this first: "`/` realm-index Checkpoint/Anchor data columns … the last large self-contained,
code-closable, normal-CLOSING slice — prefer it over re-polishing met surfaces" (state.md:166-172).

## Goal
Surface each hub's Bitcoin-anchor state on the realm index honestly. Add a leaf `Anchor` field to
`store.HubSummary`/`ListHubs` (the only genuinely new data; the Checkpoint column re-purposes the
already-present accepted `LastSize`), then render the mockup's `Checkpoint` and `Anchor` columns so the
`/` ledger reaches named-region parity with its mockup grid.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/store/hubs.go` — add `Anchor string` to `HubSummary`; extend the `ListHubs` query with a
    correlated subselect that projects each hub's latest-stamped-root OTS status.
  - `internal/dashboard/handler.go` — add `Checkpoint` + `Anchor` (+ a presentation-only `AnchorDot`)
    to the `row` view-model; populate them in `buildRows`.
  - `internal/dashboard/dashboard.html` — widen the colhead + data-row grids to six columns and render
    the `Checkpoint` and `Anchor` cells (the in-template column comment kept in sync).
- **Reference**:
  - `.claude/context/learnings/dashboard.md` — the `ListHubs` leaf-read rule; the constraint-win note
    that Checkpoint/Anchor were deferred "do NOT add a store read for them" (this step is exactly that
    deferred store read landing); the CSS-Grid `min-width:0` ellipsis trap; the narrowed no-CDN ban.
  - `.claude/context/learnings/store.md` — the OTS-table CRUD seam idioms; the `OTSStatusPending` /
    `OTSStatusConfirmed` literal-drift consts; the "store stays a leaf" invariant; the no-migration
    `normal` this column inherits (Not In Scope below).
  - `internal/store/ots.go` — the `OTSStatusPending`/`OTSStatusConfirmed` consts + the `ots` row shape.
  - `internal/store/hubs.go` + `internal/store/checkpoints.go` — CoverageInfo / HubSummary shape.
  - `internal/dashboard/handler_test.go` — the `TestDashboardRendersEveryHub` seam + its seed helper
    (`UpsertHub` / `AdvanceAccepted` / `AdvanceFollowState`).
  - `.claude/design/ISCC Monitor - Realm Index.dc.html:65-103` — the authoritative six-column grid and
    the `anchorDot`/`anchorShort` derivation (pending → yellow `#ffc300` / "pending"; else green
    `#7aa832` / "confirmed").

## Not In Scope
- **Config-driven instance identity / realm name** (sub-delta 2 of the same `normal`) — needs new env
  config wiring (`internal/config`); a separate later step. Leave the static masthead copy as-is.
- **"Recent declarers checked" hero footer** (sub-delta 4) — needs a recent-lookup history the store
  does not track; out of scope.
- **The on-disk DB-migration mechanism** — the new `Anchor` field is a *read-only projection* off the
  existing `ots` table, so it adds **no new column** and does NOT trip the no-migration `normal`
  (issues.md:40). Do not introduce a migration framework here.
- Touching the OTS write/stamp/upgrade loop, `proofserve`, the dossier, or the certificate. This is a
  read projection + the `/` template only.
- Any signature / RFC-6962 / Merkle / `proof` / `didweb` / fsck path — the oracle gate is N/A (pure
  HTML of a persisted leaf read; keep it that way).

## Implementation Notes
- **Anchor projection = the hub's latest-stamped-root OTS status, store-honest.** In `ListHubs`, add a
  correlated subselect so a never-stamped hub yields SQL NULL → empty `Anchor` (rendered "—" / "not
  anchored", an honest pending-style state, never a guarantee). The newest stamped row is the relevant
  one, e.g.
  `(SELECT o.status FROM ots o WHERE o.hub_id = h.hub_id ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1)`.
  Read it through `sql.NullString` exactly like the other nullable scans; map NULL → `""`. Keep the
  outer `ORDER BY h.hub_id` unchanged. **Store stays a leaf** — this adds only SQL, no new import;
  verify `go list -deps ./internal/store | grep -E 'net/http|internal/'` stays empty.
- **Use the consts, not literals.** When deriving the dot/label in `buildRows`, compare against
  `store.OTSStatusConfirmed` / `store.OTSStatusPending` (the re-exported values, not hand-typed
  "confirmed"/"pending") so the literal-drift trap (store.md) does not reopen here.
- **View-model.** Add `Checkpoint uint64` (= `s.LastSize`, the accepted checkpoint size the mockup's
  `checkpointSize` column shows — a rename of the existing "Observed size" cell, not a new read),
  `Anchor string`, and a presentation-only `AnchorDot` ("pending"/"confirmed"/"none") to `row`. Render
  the anchor as **label + grayscale-safe**: a short text label ("confirmed"/"pending"/"—") is the
  load-bearing signal; any colored dot is decorative only (ADR-0010 invariant 4), exactly as the
  frozen-row tint already is. Do NOT make the dot an external image (no-CDN).
- **Template grid.** Both the `.ledger-colhead` and the `a.ledger-row` `grid-template-columns` must
  change from `34px 1.8fr 1.3fr 1fr 150px` to the mockup's six-column
  `34px 1.8fr 1.2fr 1fr 1.1fr 150px`; add the `Checkpoint` + `Anchor` colhead `<span>`s and the two new
  data cells **between** the Coverage cell and the Status cell, matching the mockup's column order.
  Keep the `.hub-cell { min-width: 0 }` ellipsis fix; the new mono cells do not ellipsize.
- **No-CDN posture unchanged.** Add no external URL; the body still renders scheme-less `Origin`, so
  `TestDashboardLinksTokensNoCDN` keeps passing. Render the dot via an inline `<span>` styled by a DS
  token (or a literal-fallback hue like the frozen-row tint), not an `<img>`.
- **Render into the buffer first** — the existing 500-before-200 discipline is unchanged; you are only
  adding fields to the same `pageData`.
- **Correctness rule (learnings index):** "`store.ListHubs` is a pure read returning plain Go types …
  so store stays a leaf." The Anchor projection must keep that property.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` empty
  outside `cauldron/`).
- `go test -count=1 -run TestDashboard ./internal/dashboard` passes — extend
  `TestDashboardRendersEveryHub`: seed one hub with a `RecordOTS` row whose status is
  `store.OTSStatusConfirmed` and assert the served HTML carries a `Checkpoint` AND an `Anchor` colhead
  and the `confirmed` anchor label; assert a hub with no OTS row renders the honest "—"/not-anchored
  cell.
- `go test -count=1 -run TestListHubs ./internal/store` passes — add or extend a store test asserting
  `HubSummary.Anchor` is `store.OTSStatusConfirmed` for a hub with a confirmed OTS row,
  `store.OTSStatusPending` for a pending-only hub, and `""` for a hub with no OTS row (the never-stamped
  honest empty state).
- Mutation (advance proves non-vacuous, then reverts): dropping the new `Anchor` `<span>` from the
  colhead OR the anchor data cell from `dashboard.html` makes `TestDashboardRendersEveryHub` FAIL on the
  `Anchor` assertion; reverting the `ots` subselect to a literal `""` makes the store `TestListHubs`
  anchor-status case FAIL.
- `go list -deps ./internal/store | grep -E 'net/http|internal/'` is empty (store stays a leaf).

## Done When
`mise run check` is green and the new `TestDashboard` + `TestListHubs` anchor assertions pass with the
`/` ledger rendering the six-column `# | Hub · domain | Coverage since | Checkpoint | Anchor | Status`
grid, each hub's Anchor cell showing its honest store-derived OTS state (confirmed / pending / not
anchored), the store still a leaf.
