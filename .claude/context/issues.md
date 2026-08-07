# Issues

Lightweight backlog `define-next` can prioritize. Append entries; `review` deletes resolved ones.

**Format** — one entry per issue:

```
## <short title>
- **Priority:** critical | normal | low
- **Source:** [human] | [review] | [advance]
- **What / where / how to verify:** <the problem, its location, and the check that proves it fixed>
- **Spec:** <optional — target.md or an ADR section this is rooted in>
```

**Priority semantics:** `critical` preempts everything; `normal` is weighed against the state→target
gap; **`low` is skipped by the loop** (reserved for human-directed work). The `Source` tag records who
filed it and does **not** affect priority.

---

## Mirror-repair residuals after the corrupt-completed-coord fix (the critical itself is CLOSED)
- **Priority:** low
- **Source:** [review] + [codex] (residuals of the now-closed critical; every load-bearing half landed
  and is mutation-proven)
- **What / where / how to verify:** **CLOSED this pass** — `store.RecordTile` rejects a FULL tile that
  is not exactly `TileWidth*32` bytes; `ingestEntryBundles` rejects a FULL bundle that decodes to fewer
  than `TileWidth` records (an empty body decodes to zero leaves without error, so this is the nasty
  case); `MirroredFullEntryBundles` requires the bundle's `iscc_index` projection to exist before
  calling it skippable, so a legacy/half-written database re-folds instead of freezing its index gap;
  `PollHub` re-fetches every coord authoritatively (`ingestTiles(..., force=true)`) BOTH before
  convicting a hub of a self-consistency violation and after a failed root rebuild; and `fsckMirror`
  runs under `fsckTimeout` on its own goroutine so a wedged rebuild cannot stall `Loop.Tick`'s
  sequential realm walk. All five are mutation-proven (reverting each fails its test).
  Residuals, none blocking:
  (a) **The `sha256` column is still never read.** `RecordTile` writes it "for fsck cross-checks" but
      nothing `SELECT`s it anywhere. The length gate covers every width (`width*32`, verified against a
      live 300258-leaf log — full 8192, level-0 `.p/226` 7232, level-1 `.p/148` 4736, level-2 `.p/4`
      128), so the malformed-body class is closed at the door; a right-length WRONG-bytes tile is still
      admitted and is *repaired* (the forced re-walk) rather than *prevented*. Re-checking the stored
      digest for the coords a proof is about to read would close that remainder at admission.
  (b) **A legacy bundle whose projection fold died PART-way through still reads as projected.** The
      skip's completeness check is an EXISTS on the bundle's FIRST leaf seq (a PK seek, chosen so the
      per-poll cost stays one seek per bundle). `RecordProjections` is a per-row loop, not one
      transaction, so a half-written OLD fold is missed. Exact for anything the current write order
      produces (projections land before the BLOB, so a mirrored full bundle has all its rows or none).
  (c) **`tilesserve` serves full-width rows `public, max-age=31536000, immutable`**
      (`tilesserve/handler.go:39`). A row poisoned before the admission gate existed is now repaired
      locally, but any downstream cache that already fetched it keeps the bad bytes for up to a year
      with no invalidation path.
  (d) **The repair walk is all-or-nothing.** `force` re-fetches EVERY enumerated coord for that hub, so
      a large tree pays a full re-download to fix one bad byte. Bounding it to the coords the failed
      proof/rebuild actually read would be far cheaper, but needs the fsck to report which coord failed.
- **Spec:** target.md "Follow-traffic contract" (incl. "a full entry bundle present in the mirror
  implies its `iscc_index` projection was written"); ADR-0005 (tiles are rebuildable, evidence is not);
  ADR-0006 (freezing is reserved for genuine self-consistency violations).

## `deploy/realm-testnet.txt` names hubs that no longer serve — the authoritative Hub-List has different hosts
- **Priority:** normal
- **Source:** [review] (found while verifying tile widths against a live hub)
- **What / where / how to verify:** The baked fallback realm document `deploy/realm-testnet.txt` (and
  the fixture `internal/registry/testdata/realm.txt`) list `sb0.iscc.id` and `sb1.amlet.id`. Neither
  serves today: `sb0.iscc.id` fails the TLS handshake outright (`tlsv1 alert internal error`) and
  `sb1.amlet.id` presents a certificate that does not cover that name. The **authoritative iscc-hub
  Hub-List** (`https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/testnet.yaml`)
  instead names `https://staging.iscc.id` (hub_id 1) and `https://staging.amlet.id` (hub_id 2), and the
  mainnet list names `https://iscc.id` / `https://amlet.id` — all four of which serve a checkpoint
  (verified: `amlet.id` is at tree size 300258, `staging.amlet.id` at 146, both `iscc.id` and
  `staging.iscc.id` at 1). So the image's baked fallback points a fresh deploy at two dead hosts; an
  operator who does not set `ISCC_MONITOR_REALM` to a URL gets a monitor that follows nothing and
  reports every hub unresolvable. CLAUDE.md documents this file as "the fallback when no URL override
  is set", which is exactly the path that breaks. Fix: update the baked document to the current
  staging hosts (or drop the domains-only fallback in favour of the authoritative URL default, which
  also carries the real 12-bit `hub_id` the interim document-order mapping only guesses). Verify fixed:
  the baked realm's hosts each return a `200` on `/log/checkpoint`. NOT urgent for the loop's own
  tests (they use in-process fixtures, not the network) — it is a deploy-correctness issue.
- **Spec:** CLAUDE.md `ISCC_MONITOR_REALM` (the baked `/etc/iscc-monitor/realm.txt` fallback);
  ADR-0009 realm registry advertises domains; ADR-0013 server packaging.

## `tessera/fsck` DEADLOCKS rather than erroring on a corrupt completed hash tile — report upstream
- **Priority:** low
- **Source:** [review] (reproduced locally with a throwaway probe during the mirror-repair pass)
- **What / where / how to verify:** With a corrupt COMPLETED (width-256) hash tile in the mirror,
  `logclient.RunFsck` never returns: `tessera@v1.0.2` blocks forever on a channel send in
  `fsck.(*fsckTree).flushPartialTiles` (`fsck/fsck.go:313`, reached from `Fsck.Check` `:144`), and it
  does **NOT** observe context cancellation (probe: still blocked at 10s with a 5s ctx deadline long
  passed). A corrupt PARTIAL tile returns an error cleanly, which is why
  `TestPollHubFsck/RejectsCorruptedMirror` (which corrupts a partial) never surfaced it. Reproduce: seed
  a 300-leaf `buildVerifiedMirror`, run one clean `PollHub`, flip a byte in the width-256 tile `{0,0}`,
  then call `fsckMirror` — it blocks. **Locally mitigated** by the `fsckTimeout` bound in `fsckMirror`
  (a wedged rebuild is abandoned rather than allowed to stall the realm), and the abandoned goroutine is
  proven not to strand the single-connection store (`TestPollHubBoundsAndRepairsWedgedRootRebuild` runs
  the repair walk and a second rebuild after it). The upstream report is FILED (2026-08-07, by Titusz +
  Claude outside the loop): transparency-dev/tessera#1098, with a self-contained in-memory repro that
  reproduces the deadlock on v1.0.2 AND v1.0.4 (no fsck change between; bare `expectedResources` sends
  in `visit`/`flushPartialTiles` + plain `errgroup.Group` + the lone `Opts{N:1}` worker exiting on the
  mismatch error). The repro's goroutine dump also settled the open question: the abandoned goroutine is
  parked on `chan send` and is NOT reclaimable — it leaks for the process's lifetime, so the local bound
  trades a guaranteed realm-wide stall for a bounded goroutine leak (the right trade, but not free).
  What remains here: track #1098 and drop `fsckTimeout`'s wedge rationale once a fixed release ships
  (the bound itself can stay as belt-and-braces).
- **Spec:** CLAUDE.md "Mirror" (`fsck`-verifiable); `learnings/follower.md` `fsckMirror`.

## `MirroredFullTiles` / `MirroredFullEntryBundles` scan the whole per-hub tree on EVERY poll
- **Priority:** low
- **Source:** [review] (architecture/efficiency, filed alongside the mirror-repair pass)
- **What / where / how to verify:** Both are unbounded per-hub scans materialized into whole-set maps
  once per poll, so while the follow-traffic contract makes the poll's OUTBOUND cost proportional to the
  tree's GROWTH, the LOCAL cost stays proportional to its total SIZE. `width` is the LAST column of
  `PRIMARY KEY (hub_id, level, tile_index, width)`, so `WHERE hub_id = ? AND width = ?` seeks on
  `hub_id` and then scans every row that hub owns — including the stale partial-width rows that
  accumulate forever (see the partial-accumulation issue above) — on the `SetMaxOpenConns(1)` pool every
  other store user shares. The entry-bundle query now also does one `iscc_index` PK seek per candidate
  bundle. At millions of leaves that is hundreds of thousands of rows scanned and map entries allocated
  per poll per hub, plus the walk still enumerates every `TileCoord`/`BundleCoord`. `PollHub` already
  holds `fs.LastSize` one line above the ingest call, so enumerating only the delta (or keeping a
  per-level watermark) would make the local cost growth-proportional too and would need no set read at
  all. Verify fixed: a poll of an unchanged large tree reads O(1) rows, not O(tree). Low — correctness
  is unaffected and the realm is small today.
- **Spec:** target.md "Follow-traffic contract" (outbound bar; this is its local-cost twin);
  ADR-0007 single-writer sizing.

## `learnings/` detail files are over their rotation cap (follower 208 / store 181 vs ~150 lines)
- **Priority:** low
- **Source:** [advance] (observed while appending to both; the overflow predates the append)
- **What / where / how to verify:** `.claude/context/README.md` Hygiene caps each `learnings/<name>.md`
  at "~40 bullets / ~150 lines" and says to **net-reduce** on overflow by collapsing settled notes to a
  one-line `settled:` summary. `learnings/follower.md` (208) and `learnings/store.md` (181) are both
  well over; they were already over (192 / 166) before the follow-traffic append, which collapsed four
  ceremony-heavy bullets but did not close the pre-existing gap. Over-cap detail files cost every cold
  subagent context on a step that touches those packages — the exact thing the index/detail split
  exists to avoid. `review` owns rotation, so the pass belongs to it rather than to an advance that
  merely appends. Fix: collapse the remaining verification-ceremony bullets (the ones recording *that*
  a mutation was reproduced, rather than the forward-looking pitfall) to `settled:` one-liners, keeping
  every DURABLE TRAP and every load-bearing-order rule verbatim. Verify fixed: both files are ≤ ~150
  lines and no bullet tagged `DURABLE TRAP` or `LOAD-BEARING` was removed. Low — hygiene, no gate impact.
- **Spec:** `.claude/context/README.md` Hygiene ("Rotation, not unbounded append"; "Record the
  forward-looking pitfall, not the verification ceremony").

## Partial tile/bundle rows accumulate — the PK includes `width`, so a growing partial inserts instead of overwriting
- **Priority:** low
- **Source:** [advance] (observed while reading the mirror write path; pre-existing, unrelated to the
  change that surfaced it)
- **What / where / how to verify:** `tiles` and `entry_bundles` are keyed
  `PRIMARY KEY (hub_id, level, tile_index, width)` / `(hub_id, bundle_index, width)`
  (`internal/store/schema.sql:85,98`), and `RecordTile`/`RecordEntryBundle` upsert on that key. A
  partial's `width` **is its leaf count**, so as a tree grows through a tile the same coord is written
  at width 224, then 234, then 241… — each a *new row*, not an overwrite. Up to 255 partial rows can
  accumulate per tile index before the coord completes; for entry bundles each such row holds up to
  256 whole records, so on a busy hub polled every 5 minutes this, not the completed data, is the
  dominant mirror-growth term. The ADR-0005 / `store/tiles.go` wording "partials are re-fetched and
  overwritten in place every poll via the composite-PK upsert" is only true at *constant* width and
  should be corrected alongside any fix. Superseded partials are pure garbage: nothing reads them
  (`SQLiteFetcher.readTileAt` asks for one exact width and falls back to the full row), they are not
  evidence (rebuildable, ADR-0005), and the completed full-width row supersedes all of them.
  **Candidate fix:** when writing a coord, delete that coord's rows at a *smaller* width in the same
  statement/transaction (`DELETE … WHERE hub_id=? AND … AND width < ?`), so at most one partial row
  per coord exists and it vanishes when the coord completes. Verify fixed: record a coord at widths
  10, 20, then full, and assert exactly one row remains (the full one); a `SELECT COUNT(*)` probe
  before the fix shows 3. Low — a storage/vacuum concern, not a correctness one, and it interacts with
  the disk-growth-rate measurement issue below (fix them together when live sizing data exists).
- **Spec:** ADR-0005 partial-tile discipline (the "overwritten in place" claim this contradicts);
  ADR-0007 per-network DB sizing; `learnings/store.md` p↔width seam.

## OTS stamping is per-poll (anchors every observed root) — switch to the daily latest-root cadence the PRD/ADR-0004 specify
- **Priority:** normal
- **Source:** [human] (Titusz — production review of `monitor.iscc.io` OTS overgrowth)
- **What / where / how to verify:** The OTS anchor row is created on the **poll path**:
  `PollHub` calls `stampRoot` on every verified, non-frozen advance (`internal/follower/follower.go:256`
  → `stampRoot` at `:410` → `store.RecordOTS`). At the production 5-minute poll cadence this anchors
  **every distinct observed root**, deduped only by `UNIQUE(hub_id, tree_size, root)` — so an
  actively-growing hub accrues one OTS row **and one calendar submission** per observed advance. Confirmed
  on production: `amlet.id` carried **56 anchored roots over ~51 h** (the bulk from +1/+2-entry advances),
  vs `iscc.id` (a static empty log) with 1. This scales with **traffic, not time**, and is unbounded for a
  busy hub (PRD/OPERATING.md anticipate millions of records/day). The PRD already specifies the correct
  behavior — "**OTS = daily per hub**" (`.claude/prd/0001-iscc-monitor-v1.md:193`) — so the per-poll
  stamping is a **drift from spec**, now pinned by the ADR-0004 amendment (2026-06-28).
  **Fix (mechanism is `define-next`'s call):**
  1. **Remove the stamp from the poll path** — `PollHub` no longer calls `stampRoot` (drop `stampRoot` +
     its test; `RecordOTS` stays, now driven by the daily pass).
  2. **Add a daily anchor pass:** once per day, for each **non-frozen** followed hub, read its **latest
     accepted `(size, root)`** (e.g. `ListHubs` → `Frozen`/`LastSize`, then `CheckpointAt(hubID, LastSize)`
     → root) and `RecordOTS` it (deduped). A frozen hub is **not** anchored (PRD freeze rule: "no
     advance/anchor/OTS"). The simplest landing folds this pass into the existing 24 h OTS loop tick
     (`runOTSLoop` / `OTSTick`, `cmd/iscc-monitor/main.go:236` / `internal/follower/otsloop.go`) — run
     "anchor latest non-frozen roots" before the stamp/upgrade sweep; decoupling the anchor cadence from a
     future faster *upgrade* cadence stays possible.
  The background OTS loop (`OTSTick`: calendar-submit the empty-`ots_bytes` sentinel rows, then upgrade
  pending → Bitcoin-confirmed) is **unchanged** — only **where the pending row is created** moves.
  **Backward-compatible — NO schema change, NO DB reset:** the `ots` table is untouched; existing rows
  (incl. every Bitcoin-confirmed anchor — *irreplaceable evidence*) stay valid; the unchanged
  `UNIQUE(hub_id, tree_size, root)` dedupe means the daily pass never conflicts with a prior row, and the
  one in-flight pending row finishes its normal upgrade. Do **not** write a migration or reset production.
  **How to verify fixed (store/follower seam, fixtures — `OTS`-prefixed test names):** (a) a fixture poll
  that advances the tree leaves the `ots` table **empty** (`PollHub` records no OTS row); reverting the
  stampRoot removal makes it FAIL; (b) the daily pass records **exactly one** pending row for a non-frozen
  hub's latest accepted root, **dedupes** a re-run and an unchanged latest root (no second row), and
  **skips a frozen hub** (no row) — each mutation-proven; (c) `mise run check` green, no gate weakened.
- **Spec:** ADR-0004 "Amendment (2026-06-28) — OTS anchoring cadence: daily, latest-root-only";
  `.claude/prd/0001-iscc-monitor-v1.md:193,262` ("OTS = daily per hub"; freeze rule `:157` "no
  advance/anchor/OTS"); target.md "OTS / Bitcoin anchoring" (daily latest-root Verify criteria);
  `learnings/ots.md` / `learnings/follower.md` (the "stamp each distinct observed root" criterion the
  amendment supersedes). Related (do NOT merge): the realm-index per-hub-vs-per-checkpoint **Anchor display
  honesty** `normal` below — daily cadence makes the badge behave far better (the latest-stamped root
  confirms within ~a day instead of chasing a fresh pending root every 5 min), but the display-semantics
  decision is a separate design question.

## Hub-dossier "Browse the log →" lands on a dead-end checkpoint page; the record-list log browser is off-mockup
- **Priority:** critical
- **Source:** [human] (Titusz, host-machine frontend review)
> **HUMAN SIGN-OFF NEEDED (does not halt the loop):** every code-closable half of this `critical` has now
> landed and is reviewer-verified (navigation closure + part-2a chrome/head + single-record back-leg +
> part-2b pager parity). The ONLY remaining gate is the **human M-UI exit sign-off** — a human-only
> decision the loop cannot make. Please traverse `/` → dossier → record list → single record → cert/back
> (JS disabled) and confirm the record-list log browser matches `ISCC Monitor - Log Browser.dc.html`
> closely enough to sign off. Until then the loop CONTINUES on the open `normal` M-API contract-accuracy
> fixes (which are NOT human-blocked); `define-next` should NOT re-attempt this human-blocked `critical`.
- **STATUS (review of the part-2b pager-parity slice):** the no-JS navigation chain is
  **UNBROKEN end-to-end** AND every code-closable parity half has now landed. Remaining is the
  **human M-UI sign-off ONLY** — no code slice is left.
  **Part-2b (record-list pager parity — top+bottom "seq X–Y of Z" + drop the off-mockup Status row) — DONE**
  (`b6eee37`, reviewer-verified this iteration): `records.html` now renders a `.pager-top` (seamed into the
  ledger card) + a `.pager-bottom`, each a three-slot `← newer` / `seq RangeTop – RangeBottom of Total` /
  `older →` row with disabled-`<span>` ends; the off-mockup Status-badge row + its dead CSS are gone (the
  overlaid status reaches only the `.ledger data-status` frozen-tint attribute). Mutation-proven by the
  reviewer (swap RangeTop/RangeBottom → `TestRecordsPagerRangeAndTopBottom` FAILS; drop the top pager →
  FAILS; re-add the Status row → `TestRecordsRendersInMemoryStatus` FAILS), each reverted; an `agent-browser`
  visual pass vs the Log Browser mockup confirms top+bottom pager + range label + no Status row, with only
  the constraint-win residuals (plain-link pager vs mockup buttons, no JS "Jump to sequence" input). Gates green.
  **Part-1 (dossier "Browse the log →" repoint) — DONE** (recovered in `de9ed3c`):
  `internal/dossier/dossier.html:573` now reads `href="/{{.Origin}}/records"` → the record list
  (reviewer-verified this iteration; the prior git-race loss is recovered).
  **Part-2a (record-list chrome + `← <domain> dossier` breadcrumb + "Log browser" head) — DONE** (`7ea7fef`):
  `records.html` carries the shared masthead chrome, absolute-site-root breadcrumb, eyebrow/name/sub-line
  head; mutation-proven.
  **Part-2 single-record back-leg (this slice) — DONE:** `record.html` now carries the chrome identity +
  `verify ↗`, the `← Log browser` breadcrumb (relative `href="records"`), the no-JS older/newer stepper
  (disabled `<span>` at the ends), and the honesty-gated "Prove this record's inclusion →" / "Back to list"
  actions; threaded `domain, instance, operator` into `serveRecord`. Mutation-proven by
  `TestRecordBreadcrumbAndChromeIdentity` + `TestRecordStepperEnds` + `TestRecordProveInclusionHonestyGate`
  (reviewer re-mutated the honesty gate + stepper guards → both FAIL); gates green.
  The full chain `/` → dossier → record list → single record → (cert / back) is now traversable forward
  **and** back with JavaScript disabled (reviewer-traced every href).
  **REMAINING (human M-UI exit sign-off ONLY — NOT a code slice):** every code-closable half has landed and
  is reviewer-verified. Keep this critical OPEN until the human gives the M-UI exit sign-off; the loop has
  no further code work here (residual type-badge-tint / footnote micro-copy are deferred to the human pass,
  not named regions). The `define-next` after this should NOT re-attempt this critical — it is human-blocked.
- **What / where / how to verify:** Two coupled defects break the dossier→log-browser leg of the no-JS
  navigation chain and miss `ISCC Monitor - Log Browser.dc.html` parity.
  **(1) Wrong link target / dead end.** The hub dossier's "Browse the log →" action
  (`internal/dossier/dossier.html:573`) links to `/{{.Origin}}/` → the `/<domain>/log/`
  checkpoint-summary landing (`internal/proofserve/browser.html`), not the record-list log browser the
  mockup's "Browse the log →" points at (`.claude/design/ISCC Monitor - Hub Dossier.dc.html:103` →
  `ISCC Monitor - Log Browser.dc.html`). That landing shows only `(status, accepted size, accepted root)`
  plus a "Proof surface" list of placeholder example links (`entries?index=0`, `inclusion?iscc_id=ISCC:…`)
  and carries **no link to `records`** — so with JavaScript disabled the record-list browser
  (`/<domain>/log/records`) is unreachable by clicking from the dossier (a navigation-closure dead end).
  **(2) Record list off-mockup.** The record-list surface (`/<domain>/log/records`,
  `internal/proofserve/records.html`) does not render `ISCC Monitor - Log Browser.dc.html`'s landmark
  regions: no `← <hub> dossier` breadcrumb; head reads "Records" instead of the eyebrow "Log browser" +
  hub name + "<domain> · N records mirrored"; the document chrome omits the instance-identity block +
  `verify ↗ monitor.iscc.codes` link (the cross-cutting handoff header, target.md M-UI); one bottom-only
  pager labelled "showing N of M" instead of the mockup's top+bottom pagers with a "seq X – Y of Z" range
  disabled at the ends; it carries a Status badge row the mockup's log browser does not; type-badge tint
  and the append-only footnote copy diverge. Implement to follow the mockup as closely as the hard
  constraints allow (no-JS → the "Jump to sequence" input becomes a plain `GET` form or is dropped;
  self-hosted fonts/tokens; grayscale-safe badge), flagging any forced deviation.
  **Candidate fix (not mandated — `define-next` chooses the mechanism):** repoint "Browse the log →" at
  the record list and dress `records.html` to the mockup. The `/<domain>/log/` checkpoint+proof-surface
  page is M3-functional and CLAUDE.md-documented, so keep it (do not delete) — but it must not be the
  "log browser" the dossier sends a human to, and must not be a no-JS dead end.
  **How to verify fixed (HTTP seam, fixture store):** (a) the dossier's "Browse the log →" href resolves
  to the record-list log browser, and the no-JS click path dossier → record list → single record → back
  is unbroken (forward links + `←` breadcrumbs both present, no dead end); (b) the record-list HTML carries
  the `ISCC Monitor - Log Browser.dc.html` landmark regions named in target.md M-UI (breadcrumb;
  "Log browser"/hub/"N records mirrored" head; chrome with instance identity + verify link; top+bottom
  "seq X–Y of Z" pager disabled at the ends; `Seq·Type·ISCC-ID·Logged` rows each linking to its single
  record; append-only footnote); (c) golden/region tests assert each region so removing one FAILS; (d)
  `mise run check` green. A headless `agent-browser` visual pass (ADR-0012) against the mockup files the
  residual visual deltas.
- **Spec:** target.md M-UI "log browser / record list" landmark regions + "Navigation closure" +
  "Document chrome + instance identity"; `.claude/design/ISCC Monitor - Log Browser.dc.html` (authoritative
  for layout/affordances, subordinate to the no-JS / no-CDN / self-host / grayscale-safe constraints);
  `.claude/design/ISCC Monitor - Hub Dossier.dc.html:103` (the mockup's "Browse the log →" target).

## Nil-Stamper + an empty-OTSBytes row falls through to the Upgrader instead of being left untouched
- **Priority:** low
- **Source:** [review] (Codex P3, reviewer-confirmed by probe)
- **What / where / how to verify:** `OTSTick`'s stamp guard is `if stamper != nil && len(r.OTSBytes) == 0`
  (`internal/follower/otsloop.go:144`). When `stamper == nil` AND a pending row has the empty-OTSBytes
  sentinel, the guard is false, so execution falls through to `up(ctx, r)` with empty bytes — the real
  Upgrader (`recoverRead`) parses empty bytes → error → a bogus back-off (`MarkOTSAttempted`,
  Attempts++), rather than leaving the row untouched. This contradicts the `OTSTick` docstring's
  nil-tolerant claim ("A nil Stamper skips stamping, leaving the row empty"). Reviewer-confirmed by a
  throwaway probe: nil Stamper + empty row → `upCalled == true`, `Attempts == 1` after the tick. NOT a
  production hazard — `stampFunc()` always wires a non-nil Stamper, so the live loop never hits this; it
  is a docstring-vs-code contract mismatch on the test-only nil path. Fix when `otsloop.go` is next
  touched: handle `len(r.OTSBytes) == 0` FIRST and `continue` when `stamper == nil` (skip the row), so the
  nil-tolerant contract the docstring states actually holds. Verify fixed: a test with a nil Stamper + an
  empty-OTSBytes row asserts the Upgrader is NOT invoked and the row's Attempts stays 0; reverting the
  guard reorder makes it FAIL. Low — production wires a non-nil Stamper, the suite is green.
- **Spec:** CLAUDE.md "Write evergreen comments that describe the current state" (docstring must match
  behavior); next.md Implementation Note "Prefer nil-tolerant, mirroring the Loop's nil-Logger discipline".

## Single-record label test is vacuous on the kind-label constant value
- **Priority:** low
- **Source:** [review] (mutation-found in the constant-fix review)
- **What / where / how to verify:** The constant-fix advance set `schemaDeclaration` /
  `schemaDeletion` (`internal/proofserve/handler.go:111-112`) to the correct full wire URIs — verified
  byte-equal to the golden `internal/logclient/projection_test.go:19-20` — so the production feature is
  CORRECT. But the guarding test cannot prove it: `record_test.go`'s `schemaForSeq` (lines 40-52)
  returns the `schemaDeclaration`/`schemaDeletion` *constants*, and `recordKind`
  (`handler.go:838-847`) switches on the *same constants*, so reverting BOTH constants to the old short
  forms leaves the entire proofserve record suite GREEN (reviewer mutation-verified: both reverts →
  `go test -run TestRecord ./internal/proofserve` still `ok`). The test is tied to the symbol under
  test, not to ground truth, so it would not catch a future regression of the constant value. Fix when
  `record_test.go` is next touched: make `TestRecordKindLabels` (or a sibling) seed a HARDCODED literal
  URI (`"http://purl.org/iscc/schema/iscc-note-0.8.0.json"` / `…delete…`) — or assert the constants
  equal those literals — so the gate is non-vacuous. Verify fixed: reverting either constant to a short
  form makes a proofserve test FAIL. Low — the production code is already correct; this only hardens the
  regression gate.
- **Spec:** target.md M-UI single-record Verify criterion; CLAUDE.md Testing ("tests covering
  implemented functionality" + use ground-truth data, not fixtures matched to the code).

## `-run TestOTS` does not catch the OTS store tests (`TestMarkOTSAttempted*`, `TestPendingOTS` back-off)
- **Priority:** low
- **Source:** [review] (reviewer-confirmed against next.md Verification line)
- **What / where / how to verify:** `next.md`'s Verification explicitly required naming the new tests so
  `go test -count=1 -run TestOTS ./internal/store ./internal/follower` catches them all ("name the new
  tests `TestOTS…` / `TestMarkOTSAttempted` / `TestPendingOTSBackoff` so this filter catches them all —
  the filter caveat the prior OTS review flagged"). The advance instead named the store tests
  `TestMarkOTSAttempted`, `TestMarkOTSAttemptedAbsent`, `TestMarkOTSAttemptedZeroNextRetryNull` and put
  the back-off assertions inside the existing `TestPendingOTS` — none of which match the `TestOTS` prefix.
  Reviewer-confirmed: `go test -v -run TestOTS ./internal/store` runs ONLY `TestOTSForRootAbsent`; the
  three `TestMarkOTSAttempted*` and the `TestPendingOTS` back-off path are silently skipped by that
  filter. The tests DO exist, are non-vacuous (reviewer reproduced the next_retry-filter + no-op-
  `MarkOTSAttempted` mutations), and run+pass under `mise run check` and the broader
  `-run 'TestOTS|TestMarkOTSAttempted|TestPendingOTS'` union — so this is a developer-convenience /
  spec-literal gap, NOT a coverage hole and NOT a gate weakening. The follower tests (`TestOTSTick*`,
  `TestOTSBackoff`) DO match the prefix. Fix when the OTS store tests are next touched: rename
  `TestMarkOTSAttempted*` → `TestOTSMarkAttempted*` (or add a `TestOTSPendingBackoff` wrapper) so the
  documented `-run TestOTS` shorthand catches the whole suite. Verify fixed: `go test -v -run TestOTS
  ./internal/store` lists every OTS store test. Low — the suite is green and complete under `mise run
  check`; only the shorthand filter under-selects.
- **Spec:** next.md Verification "name the new tests … so this filter catches them all"; CLAUDE.md
  Testing (clean, discoverable test output).

## `cmd/notecheck`'s `run` has a vestigial `out io.Writer` parameter
- **Priority:** low
- **Source:** [review]
- **What / where / how to verify:** `cmd/notecheck/main.go` `run(vkey string, in io.Reader, out
  io.Writer) (string, error)` never writes to `out` — it returns the signer name and `main` prints
  `OK %s` to `os.Stdout` itself. The param matches the literal signature `next.md` specified and is
  harmless (tests pass a throwaway buffer; `go vet` does not flag unused params), but the signature
  is misleading. Fix when `run` is next touched: drop `out`, OR have `run` print `OK %s` to `out` and
  let the test assert on it. Verify fixed: `out` is either gone or written to. Low — skipped by the loop.
- **Spec:** KISS / YAGNI (CLAUDE.md code standards); no spec contract.

## Hub-status overlay precedence is duplicated across dashboard, proofserve, AND dossier (now 3x)
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/dashboard/handler.go:161-190`,
  `internal/proofserve/handler.go:600-635`, AND now `internal/dossier/handler.go:742-787` all implement
  the same five-status `overlayStatus` + `hubStatus` precedence verbatim — every docstring confesses it.
  The third copy landed with the hub dossier (deliberately, per its `next.md` Not-In-Scope), so the
  consolidation pressure is now 3x: a precedence fix is a three-site edit. Still `low` (no progress
  gate), but the move is more valuable now. The `internal/badge` package owns *rendering*
  the five statuses (silhouette + the single-source label table) but not *resolving* them, so the
  ADR-0010 visual-contract precedence (frozen/inactive are durable truths that win; only `verified`
  consults the live verdict, and only to adopt `unresolvable`/`unverified`) lives in two places keyed on
  two input types (`store.HubSummary`, which carries `.Active` → `inactive`, vs `store.FollowState`,
  which does not). A precedence fix in one silently diverges from the other; the log-browser overlay has
  no HTTP-seam test of its own. Deepen by moving resolution into `badge` (already the taxonomy owner) as
  one `Resolve(provable-status, live-verdict) → status` both handlers cross; the inactive case (only
  `HubSummary` has it) is decided before the seam so the resolver stays one function. Verify fixed: the
  overlay precedence exists in exactly one place, both handlers call it, and one test covers the
  five-status taxonomy. Pure locality deepening — contradicts no ADR.
- **Spec:** ADR-0010 five-status `HubStatusBadge` visual contract; CLAUDE.md "Hub status" glossary.

## Mirror write path leaks tile coordinates and the partial-`p` convention into the follower
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/follower/ingest.go:58-83` walks `tiles.TileCoords` /
  `tiles.BundleCoords` and passes each coord's `Partial` (the tlog-tiles p qualifier) straight to
  `store.RecordTile` / `store.RecordEntryBundle`, then projects via `logclient.BundleProjections` →
  `store.RecordProjections`. The "Mirror" (glossary: the complete copy of a hub's tiles + entries) has
  no single owner — coordinate enumeration, the p→width convention, projection, and BLOB writes are
  split between the follower's ingest path and the store's 22-method CRUD surface, so the follower must
  learn the tile layout to drive storage. Deepen by absorbing the walk + p + projection + writes behind
  one deep Mirror seam (e.g. `Sync(hubID, treeSize, fetcher)`); the follower stops referencing tile
  coordinates and the store's per-tile methods go private behind it. Verify fixed: `ingest.go` no longer
  references `tiles.*Coords` or a `Partial` qualifier, and the mirror round-trip is tested through the
  single Mirror interface. NOTE: this is **not** "add a store interface" — there is exactly one SQLite
  adapter (ADR-0005), so that would be a hypothetical seam with one adapter; Mirror still writes to the
  same SQLite store and `store.SQLiteFetcher` stays its read side. Larger move — wants a design/grilling
  pass before building.
- **Spec:** ADR-0005 single SQLite store; CLAUDE.md "Mirror" glossary.

## Add a scaling trip-wire: writer-wait time + per-network DB file size metrics
- **Priority:** low
- **Source:** [human]
- **What / where / how to verify:** The single-file-per-network store (ADR-0007) is right for
  10s–100s of hubs, but two axes can eventually bind: the single writer (`SetMaxOpenConns(1)`,
  `internal/store/sqlite.go`) serializing all hubs' poll-commits, and per-network file size (one
  high-traffic hub at millions/day bloating the shared file). Expose two Prometheus metrics via the
  existing `internal/metrics` registry so the bind is visible *before* it hurts, not discovered under
  load: (1) writer-wait / commit latency — how long a poll-commit waits on or holds the single
  connection (a rising p99 is the writer-contention signal); (2) per-network DB file size in bytes
  (e.g. `os.Stat` on the `.db` file, refreshed per poll cycle). Both feed the "revisit per-hub files
  or rebuildable-bulk tiering" decision recorded in ADR-0007. Verify fixed: `GET /metrics` exposes a
  writer-wait/commit-latency series and a DB-file-size gauge, both labelled per network, with a test
  asserting they appear. Low — skipped by the loop; reserved for when load planning resumes.
- **Spec:** ADR-0007 "Why network-level and not hub-level" (the trip-wire it names); CLAUDE.md
  `GET /metrics` surface.

## proofserve repeats the `os.ErrNotExist`→404 mapping that the sibling tilesserve already centralised
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** The three proof routes in `internal/proofserve/handler.go` —
  `serveInclusion` (198-202), `serveConsistency` (287-291), `serveEntries` (349-353) — each spell out the
  same `errors.Is(err, os.ErrNotExist) → 404 "… not mirrored", else → 500` mapping inline. The sibling
  `internal/tilesserve/handler.go:145-152` already lifts this into one `writeReadError(w, err)` reused by
  all its routes; a change to the not-mirrored mapping is a three-site edit in proofserve. Fix: lift one
  proofserve-local `writeReadError` and call it from the three proof routes. **Exclude `serveVerify`** —
  it deliberately maps a not-yet-mirrored tile/bundle (`os.ErrNotExist`) to a 200 verdict
  (handler.go:399-400), not a 404, so it must NOT share the helper. Verify fixed: the not-mirrored→404
  mapping for the three proof routes lives in one helper and `serveVerify`'s 200 behaviour is unchanged.
  Cosmetic locality only.
- **Spec:** `internal/tilesserve` `writeReadError` pattern; no spec contract.

## `noExternalCDN`'s `stripLineComments` still strips whitespace-prefixed protocol-relative CDN URLs
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed by probe)
- **What / where / how to verify:** The quoted-delimiter over-strip (`src="//cdn..."`, `url("//cdn...")`)
  is now CLOSED — `stripLineComments` (`internal/web/web_test.go:57`) treats `//` as a comment only at
  line-start or when preceded by whitespace, so a `"`-preceded protocol-relative URL survives and trips
  the ban (`TestNoExternalCDNProtocolRelative`, mutation-proven). Codex flags the residual narrower case:
  a `//` preceded by **whitespace** is still stripped, so the (rare, mostly-invalid HTML / valid-but-odd
  CSS) whitespace-before-URL forms `<script src = //cdn.jsdelivr.net/x.js>` and `url( //cdn.example/x.woff2)`
  are truncated before `cdn.`/`jsdelivr` and the ban misses them. Reviewer-confirmed by probe (both forms
  → `cdn.present=false`). This is **not a regression**: the prior `:`-only guard stripped these same forms
  too (reviewer-verified), and NO served asset (tokens.css/fonts.css/byte-verbatim `wasm_exec.js`) uses a
  whitespace-prefixed protocol-relative URL — the hole is latent, same class as before, and strictly
  narrower than what this advance fixed. Does NOT block progress: the increment's stated goal (the
  quoted-delimiter `//cdn.` trips the ban) is fully met, all gates green, and the gate is strictly
  stronger than its prior state. Low because the form is not realistic in a hand-authored asset and the
  loop skips lows; promote only if a real asset needs a whitespace-tolerant URL. Fix when the helper is
  next touched: a tokenizer-grade check (treat `//` as a comment only OUTSIDE a quoted string / `url(...)`
  token), not another preceding-byte blocklist — a per-delimiter list will keep losing edge forms. Verify
  fixed: a test feeds `noExternalCDN` `<script src = //cdn.jsdelivr.net/x.js>` and `url( //cdn.example/x)`
  and asserts the ban FIRES for both; reverting the tokenizer makes them pass (regress).
- **Spec:** target.md M-UI hard CDN-free constraint; `learnings/web.md` `noExternalCDN` bans third-party
  origins; CLAUDE.md "Never weaken a quality gate to pass" (the fix is the root cause, not the gate).

## The WASM verifier never checks the checkpoint signature against the hub's did:web key (the design-blocked signature half; id-binding AND copy-honesty halves now CLOSED in source)
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed against the verify core; affects BOTH tier-2 callers)
- **What / where / how to verify:** UPDATE (advance `4a0c24b`, reviewer-verified): the **copy-honesty
  interim** half is now CLOSED. The verifier page no longer LISTS the un-run "Check the signature against
  the hub's did:web key" step (replaced by "Confirm the record commits the requested ISCC-ID", the
  id-binding the WASM does run) and no longer CLAIMS the browser re-verified a "hub-signed checkpoint
  root" — the `verified` verdict now asserts only the accepted-root + id-binding check and points
  signature trust at server-side certificate §4. Mutation-proven (`TestVerifierDoesNotClaimSignatureCheck`
  FAILs if either the did:web step is re-added or "hub-signed checkpoint root" is restored to the verdict).
  The earlier **id-binding** half was CLOSED in source (advance `22f0420`): `verifyadapter.RecordCommitsID`
  binds the record's committed `iscc_id` to the requested id, and the 6-arg `isccVerifyInclusion` gates
  `verified` on it. What REMAINS open is ONLY the **design-blocked signature half**: `isccVerifyInclusion`
  (`VerifyJSON` → `internal/proof/verify.VerifyInclusion` + `RecordCommitsID`) verifies ONLY the RFC-6962
  inclusion math and the id-binding — it does NOT verify the checkpoint note signature against the hub's
  did:web key. So a malicious/compromised monitor can still return a bundle whose record+proof+root are
  internally consistent under a FORGED (unsigned / wrong-key) checkpoint and — provided the record commits
  the requested id — the browser renders green `verified`, trusting the monitor for the signature. This is
  the SAME verifier-core scope the certificate's same-origin tier-2 ships (`cert.html`), so it is NOT a
  regression and does NOT block progress; but it is more serious cross-origin. NOT currently exploitable on
  the testnet (the fixture monitor is honest). The page is now HONEST about this gap; closing the gap
  itself needs a DESIGN PASS (browser did:web resolution + note-signature verify). Fix when the WASM
  verifier scope is next expanded: extend the verifier (or a sibling export) to verify the checkpoint note
  signature against a did:web key fetched/resolved in the browser, gating `verified` on signature +
  id-binding + inclusion. Verify fixed: a bundle with a valid inclusion proof + matching id but a
  checkpoint signed by a non-did:web key renders `error`/`failed`, NOT `verified`; reverting the added
  signature check makes that test FAIL.
- **Spec:** CLAUDE.md "Verifier app" / "Proof bundle" / "Verifiable cache" (the monitor is NOT in the
  trust path; the client re-verifies signature + Merkle); ADR-0009 did:web is the only key source;
  learnings.md always-loaded "gate a rendered ✓ on a re-VERIFICATION" (a full re-verification includes
  the signature + id binding, not inclusion math alone); `learnings/cmd-wasm.md` `isccVerifyInclusion` scope.

## `publish.yml`'s `docker/login-action@v3` + `docker/build-push-action@v6` still target deprecated Node 20
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed via `gh api .../action.yml?ref=v3|v6`)
- **What / where / how to verify:** The Node-20 action bump (advance `4909dd2`) fixed every `actions/*`
  pin in all three workflows (the `pages.yml`/`ci.yml` deprecation-annotation set — `checkout@v4`→`@v7`,
  `setup-go@v5`→`@v6`, `configure-pages@v5`→`@v6`, `upload-pages-artifact@v3`→`@v5`,
  `deploy-pages@v4`→`@v5` — all CLOSED) but LEFT `docker/login-action@v3` (`publish.yml:49`) and
  `docker/build-push-action@v6` (`publish.yml:61`) on Node 20. The advance/next claimed these are
  "container actions, not in the Node-20 list" — that is FALSE: reviewer-confirmed `gh api
  repos/docker/login-action/contents/action.yml?ref=v3` and `...build-push-action/...?ref=v6` both report
  `runs.using: 'node20'` (they are node20 JavaScript actions). So `publish.yml` is NOT fully off the
  deprecated runtime — a `publish` run still hits the Node-20 deprecation path. NOT a current breakage:
  the job runs green today because GitHub force-runs node20 actions on node24; it becomes a hard failure
  only once GitHub removes the node20 shim (github.blog/changelog/2025-09-19-deprecation-of-node-20). Does
  NOT block progress — same `low` class as the original Node-20 issue, and `publish.yml` only runs on
  push-to-develop / maintainer dispatch. Fix when `publish.yml` is next touched: bump
  `docker/login-action@v3`→`@v4` and `docker/build-push-action@v6`→`@v7` (their current majors, both
  node24); reviewer-confirmed the `@v4`/`@v7` inputs are unchanged — `login-action@v4` keeps
  `registry`/`username`/`password`, `build-push-action@v7` keeps `context`/`file`/`push`/`tags`/`build-args`,
  so the existing usage stays valid. Verify fixed: `! grep -RqE "docker/login-action@v3|docker/build-push-action@v6"
  .github/workflows/` AND a dispatched `publish.yml` run is green with NO Node-20 deprecation annotation.
- **Spec:** ADR-0013 server packaging (GHCR publish); CLAUDE.md "Building the Surface-C verifier site"
  (workflow maintenance); `learnings/ci.md` §publish (docker actions ARE node20).

## `cmd/verifier-site` `generate` writes non-atomically — a mid-run error leaves a partial deploy tree
- **Priority:** low
- **Source:** [review] (Codex P3, reviewer-confirmed against the code)
- **What / where / how to verify:** `cmd/verifier-site/main.go` `generate` writes `index.html` first
  (`main.go:66`) and then render-then-writes each `/_ds/` asset in the `for _, p := range paths` loop
  (`main.go:85-93`). If a LATER step fails — a future `/_ds/` path that 404s (the fail-closed branch),
  or a `writeFile` error (e.g. `_ds` already exists as a *file* under a reused `-out`) — `generate`
  returns an error but `index.html` (and any already-written assets) are ALREADY on disk, leaving a
  partially-updated tree in a reused `dist/`. The generator's stated contract ("fails closed … rather
  than writing a partial site", `main.go` docstring + next.md) is honored at the run level (it errors →
  `os.Exit(1)` → CI/`TestGenerate` catches it, so a broken deploy is NEVER silently published), but NOT
  at the output level: the directory itself is left half-written. NOT a current hazard — `TestGenerate`
  uses a fresh `t.TempDir()`, the happy path materializes the full 14-file tree, and the Pages publish
  workflow (next sub-step) gates on the non-zero exit. Reviewer-confirmed by inspection (write-before-
  later-render ordering). Fix when the generator is next touched: stage into a temp dir and
  `os.Rename` it into place on success, OR buffer every handler response (collect all `(path, body)`
  pairs) before the first `writeFile`, so `outDir` is updated atomically. Verify fixed: force a mid-run
  render error (e.g. inject a 404 path) and assert `outDir` is left unchanged (no stale `index.html`);
  reverting the staging makes it FAIL. Low — skipped by the loop; the run-level fail-closed is intact.
- **Spec:** next.md "fail closed so a broken deploy is caught … not in production"; `main.go` docstring
  ("errors rather than writing a partial site"); learnings.md always-loaded fail-closed discipline;
  `learnings/verifier-site.md` non-atomic-output note.

## Realm-index `/` Anchor column is per-hub (latest-stamped root), not tied to the displayed Checkpoint — a design-honesty question for the M-UI exit
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-triaged — implementation is spec-faithful; the design question is real)
- **What / where / how to verify:** The `/` ledger Anchor cell (`internal/store/hubs.go:50-51` +
  `internal/dashboard/handler.go anchorLabel`) projects the hub's LATEST-STAMPED-root OTS status
  (`SELECT o.status … ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1`), while the same row's Checkpoint
  cell shows `f.last_size` (the accepted tree size). The two are DECOUPLED: nothing ties the chosen OTS
  row's `tree_size`/`root` to the displayed checkpoint. Codex's framing (it could render "confirmed" for
  an older root while the newer checkpoint is shown, "overstating current anchoring") is technically
  accurate but is the EXPECTED steady state, not a defect: OTS is async/best-effort (ADR-0004, never
  blocks the poll), so the displayed checkpoint is almost always AHEAD of the latest Bitcoin-confirmed
  anchor. The column is — by design AND by the mockup (`anchorState` is a free-standing per-hub property,
  `.dc.html:88-103`) — a per-HUB "this hub anchors its roots" indicator, NOT a per-checkpoint
  attestation. The AUTHORITATIVE per-checkpoint claim already exists in **certificate §5**, which binds
  `OTSForRoot(hubID, treeSize, root)` to the §2 accepted root via `ots.ConfirmedFor`. This increment is
  spec-faithful (matches `next.md`'s "latest-stamped-root" projection + the mockup), all gates green,
  mutation-proven — so it does NOT block progress. The open question for the M-UI exit / a design pass:
  should the realm-index Anchor cell (a) stay a per-hub activity indicator (current, mockup-faithful),
  (b) gain a distinct label that makes the "latest confirmed anchor, not this checkpoint" semantics
  explicit, or (c) tie to `o.tree_size = f.last_size` — but (c) is REJECTED without a design pass because
  it would render "not anchored" for virtually every actively-polling hub (the newest checkpoint is rarely
  confirmed yet) and defeat the column. Verify resolved: the design pass records the chosen semantics and,
  if (b), the realm-index Anchor label distinguishes hub-anchoring-activity from a per-checkpoint claim;
  the certificate §5 per-root surface stays the authoritative per-checkpoint attestation.
- **Spec:** CLAUDE.md "Bitcoin anchoring" (Bitcoin-only meaning) + "Coverage" (never imply a guarantee
  the data does not support); ADR-0004 OTS async/best-effort; ADR-0010 Evidence-Ledger honesty;
  `.claude/design/ISCC Monitor - Realm Index.dc.html` per-hub anchorState model; `learnings/dashboard.md`
  per-hub-vs-per-checkpoint Anchor note; `internal/certificate/handler.go` §5 authoritative per-root surface.

## Masthead identity fallback consts are now duplicated across dashboard + dossier + certificate (3x) instead of one shared resolve leaf
- **Priority:** low
- **Source:** [review] (filed alongside the dossier masthead-identity slice `413efe8`; updated when the cert copy landed `3c64097`)
- **What / where / how to verify:** The masthead-identity arc has now copied `instanceFallback` /
  `operatorFallback` + a private `resolveIdentity` into THREE packages: `internal/dashboard/handler.go:114-115`
  (the original `Identity.resolve` owner), `internal/dossier/handler.go:107-108`, and now
  `internal/certificate/handler.go:174-175` (advance `3c64097`). All are LITERALS byte-identical with a "MUST
  stay byte-identical" comment, because neither package can import the other's unexported consts and exporting
  `dashboard.resolve` would push each slice to a 4th prod file (over the ≤3 budget). This is a documented,
  commented, mutation-proven duplication, not a defect — but a future change to the static masthead copy is now
  a THREE-site edit (FOUR once the proofserve mastheads land) that can silently diverge. Fix when the
  masthead-identity arc finishes across all surfaces: lift `Identity` + the fallback consts + a single exported
  `Resolve` into ONE owner (the `internal/dashboard` package already owns the type, or a tiny new shared leaf)
  that dossier/cert/proofserve all import, so the fallback exists once. Verify fixed: the
  `instanceFallback`/`operatorFallback` literals appear in exactly one package and every masthead resolves
  through it; a test asserting dashboard+dossier+cert render the SAME fallback line passes. Low — the consts are
  currently byte-identical and the duplication is commented; this only removes the divergence risk once the arc
  is complete (best folded WITH the proofserve masthead slice, the natural 4th-copy trigger).
- **Spec:** CLAUDE.md DRY ("Reduce code duplication even if refactoring requires extra effort"); next.md
  Implementation Note (per-package helper chosen to stay ≤3 prod files, consolidation deferred).

## Stale `.chrome-identity` CSS comment in dashboard.html still says "static copy in this skeleton"
- **Priority:** low
- **Source:** [review] (observed during the dossier masthead-identity review)
- **What / where / how to verify:** `internal/dashboard/dashboard.html:75-76` carries the comment "The
  instance-identity block: this deployment's domain + operator/realm. It is static copy in this skeleton (a
  config-driven identity is a separate concern)." — inaccurate since the dashboard masthead became
  config-driven in `b30b84e` (the block now renders `{{.Instance}}`/`{{.Operator}}` from
  `dashboard.Identity`). The dossier slice wrote an ACCURATE comment on its ported copy
  (`internal/dossier/dossier.html`) but correctly left the dashboard untouched (it was out of scope and a
  4th prod file). Violates CLAUDE.md "write evergreen comments that describe the current state". Fix when
  `dashboard.html` is next touched: update the comment to match the dossier's accurate wording. Verify
  fixed: the comment no longer says "static copy in this skeleton". Low — cosmetic; the rendered output is
  already correct.
- **Spec:** CLAUDE.md "Write evergreen comments that describe the current state, not historical changes".

## `.dockerignore` secret/sidecar globs are slashless — they only exclude CONTEXT-ROOT files, not nested ones
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed against Docker's `filepath.Match` vs git basename matching)
- **What / where / how to verify:** The `.dockerignore` (advance `a15a9f4`) now lists `.env`, `.env.*`,
  `*.db-wal`, `*.db-shm` (`/workspace/iscc-monitor/.dockerignore:23-36`) — closing the ROOT-level gap — but
  these are SLASHLESS patterns. Docker's `.dockerignore` uses Go `filepath.Match`, where a slashless
  pattern matches ONLY a file directly under the build-context root; `.gitignore`, by contrast, matches the
  basename at ANY depth. Reviewer-confirmed: `git check-ignore` IGNORES `deploy/.env` and
  `data/monitor.db-wal`, but Docker would NOT exclude them — so a nested secret/sidecar (e.g. `deploy/.env`,
  `data/monitor.db-wal`) is still sent to the build stage by `COPY . .`. The intent ("a superset of the
  gitignore's never-commit set") therefore holds only for root-level files. `**/auth.json` already uses the
  correct recursive form. NOT a leak in the shipped artifact (the final stage only `COPY --from=build`s the
  binary, never the context) and NOT a CI issue (a fresh checkout has none of these files) — a latent
  defense-in-depth gap, same class as the now-closed root-level one, strictly narrower. Does NOT block
  progress; all gates green. Fix when `.dockerignore` is next touched: use recursive forms — `**/.env`,
  `**/.env.*`, `**/*.db-wal`, `**/*.db-shm` (mirroring the already-recursive `**/auth.json`) — so the
  exclusion matches the gitignore at any depth. Verify fixed: a throwaway `deploy/.env` /
  `data/monitor.db-wal` in the working tree is NOT in the build context (a test stage `RUN ls` cannot see
  them, or `docker build --progress=plain` shows them excluded).
- **Spec:** repo `.gitignore` "Local secrets / state — never commit"; ADR-0013 server packaging;
  `learnings/ci.md` (`.dockerignore` matching is not `.gitignore` matching).

## `TestNoMermaidInContract` ban is a substring check — misses CommonMark-equivalent fence forms (tilde / whitespace-after-fence)
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed by probe; residual of the now-CLOSED `normal` mermaid-ban ask)
- **What / where / how to verify:** The slice-4 guard `TestNoMermaidInContract`
  (`internal/openapi/contract_test.go:193`) bans a mermaid fence via `containsFold(a.doc, "```mermaid")` —
  a SUBSTRING match for the canonical adjacent-backtick form only. CommonMark also treats a TILDE fence
  (`~~~mermaid`) and a fence with WHITESPACE before the info string (`` ``` mermaid ``) as a mermaid code
  block, and Stoplight Elements' Markdown renderer would lazy-load `unpkg.com/mermaid@9.4.3/...` for those
  too. Reviewer-probed BOTH forms in a description: `TestNoMermaidInContract` stays GREEN (the ban does NOT
  fire) while still being a no-CDN trigger. The PRIMARY ask of the prior `normal` is DELIVERED and CLOSED —
  the guard exists, catches the realistic human-authored ` ```mermaid ` form, and is mutation-proven
  (adding ` ```mermaid ` FAILS it; reverting the guard passes) — so this is a NARROWER residual, same class
  as the `noExternalCDN`-whitespace / `.dockerignore`-slashless lows: latent (the served doc has ZERO
  mermaid of any form today), defense-in-depth, and reachable only if a FUTURE author writes a
  non-canonical CommonMark fence in a description that currently has no diagrams. Does NOT block progress —
  all gates green, every `next.md` Verify criterion met. Fix when the guard is next touched: parse the
  description's fenced-code LANGUAGE TOKEN (`~~~`/```` ``` ```` + optional whitespace + `mermaid`) rather
  than extending a per-form substring blocklist (a substring list keeps losing edge forms). Verify fixed:
  a test feeds `~~~mermaid` and ` ``` mermaid ` into a description and the ban FIRES for both; reverting the
  token-parse makes them pass.
- **Spec:** target.md M-UI hard CDN-free constraint; ADR-0014 §4 ("No external CDN, no external runtime
  call"); `learnings/openapi.md` mermaid-ban-substring nuance; `learnings/web.md` Elements no-CDN nuance;
  CLAUDE.md "Never weaken a quality gate" (a substring guard with a known bypass is the root cause to fix).

---

<!-- The entries below are pre-deployment asks from the iscc-infra ops side, raised
     while preparing a testnet TEST INSTANCE at https://monitor-test.iscc.io on an
     existing DigitalOcean box (Docker Compose + caddy-docker-proxy). They are framed
     as what the deploy needs FROM this repo, not loop-internal defects. Filed 2026-06-22.
     Durable target: these are the work items of milestone **M-Deploy** in target.md
     (ratified in ADR-0013). issues.md is ephemeral; M-Deploy / ADR-0013 are the
     standing spec the loop verifies against — re-derive these if this list is pruned. -->

## Publish a deployable container image to GHCR (Dockerfile + push workflow)
- **Priority:** low
- **Source:** [human] (iscc-infra ops, pre-deploy blocker)
- **STATUS — code-complete (advance `760213b` Dockerfile + `a15a9f4` publish workflow):** the production
  multi-stage `Dockerfile` (static `CGO_ENABLED=0` binary → distroless/static nonroot, non-root uid 65532,
  CA roots, ~28 MB, version-stamped, fail-fast on empty VERSION) AND `.github/workflows/publish.yml`
  (push-to-`develop` + `workflow_dispatch`, `packages: write`, build-push tagging `:develop` + `:sha-<short>`
  with a non-empty `VERSION` build-arg) both exist and are reviewer-verified (Dockerfile via the static-ELF
  build half + the CI `docker` /healthz smoke; publish.yml via YAML-validity + tag/permission/trigger
  inspection — Docker is CI-only on the dev host). What REMAINS is purely iscc-infra repo-settings work,
  explicitly OUT of the loop's scope per `target.md` M-Deploy "Out of the loop's scope": make the GHCR
  package public OR issue infra a `read:packages` token. Demoted to `low` (was `critical`) — the loop has
  delivered everything code-closable; the residual is a one-time human/infra step that does not gate DONE
  here. Kept as a tracking record until the human confirms the package is pullable.
- **What / where / how to verify (original ask):** There was no production Dockerfile (only
  `.devcontainer/Dockerfile`) and no image-publish workflow — `.github/workflows/ci.yml`
  only builds+vets+tests, and `pages.yml` deploys the SEPARATE `.codes` verifier site, not
  the server. iscc-infra deploys via Docker Compose + caddy-docker-proxy and needs a
  *pullable image*, not a source build on the box. Ask: add a multi-stage `Dockerfile` that
  builds the `cmd/iscc-monitor` static binary (Go 1.26, `CGO_ENABLED=0`; it is already
  pure-Go incl. `modernc.org/sqlite`, so a `scratch`/distroless final stage with no libc
  works) running as a NON-root uid, plus a workflow that builds and pushes to
  `ghcr.io/iscc/iscc-monitor` on push to `develop`, tagged BOTH `develop` (floating) and
  `sha-<short>` (immutable, so infra can pin a known-good build and roll back). Make the
  GHCR package public, or hand infra a `read:packages` token. The server image is
  self-contained: it embeds and serves its own `/_ds/` assets incl. `verify.wasm`
  (`internal/web`), so it needs NEITHER the Pages site NOR any CDN at runtime. Fold in a
  build stamp — pass the git SHA via `-ldflags` and surface it (on `/healthz` JSON or a tiny
  `GET /version`) so infra can confirm exactly which build is live. Verify fixed:
  `docker run ghcr.io/iscc/iscc-monitor:develop` with the required env starts and serves
  `/healthz` = 200; `docker image inspect` shows a non-root user and a small (<~30 MB)
  image; the running git SHA is reported by the binary.
- **Spec:** ADR-0003 `CGO_ENABLED=0` static build; CLAUDE.md "single binary configured
  entirely through environment variables".

## `deploy/OPERATING.md` §Footprint gives a QUALITATIVE disk-growth answer, not a concrete per-hub/N-hub rate
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-triaged — bar met as well as in-repo data allows; the missing number needs live testnet measurement)
- **What / where / how to verify:** When the egress+footprint `critical` was pruned (advance `3af0084`),
  Codex flagged that the deleted Verify bar asked for "ballpark RAM / CPU / **disk-growth** for an N-hub
  realm" (the original body emphasized "especially the **disk-growth rate** of the mirror BLOBs per hub
  over time, so infra can size the volume"), but `deploy/OPERATING.md` §Footprint (lines 161-166) answers
  disk-growth only QUALITATIVELY: "proportional to each hub's log activity — a quiet testnet hub adds
  little; a high-traffic hub at millions of records/day would dominate", plus a "set a DigitalOcean
  disk-usage alert + size with headroom" recommendation. RAM ("tens of MB") and CPU ("near-idle, brief
  per-poll bursts") DO carry ballpark numbers; the disk-growth clause is the one answered without a rate.
  Reviewer-confirmed the gap is real (the §Footprint text is qualitative) AND that the missing number is
  **not derivable in-repo**: there are no benchmarks, no on-disk size fixtures, and the rate depends on
  each testnet hub's real-world record volume + actual ISCC-note/tile BLOB sizes — none of which is loop
  ground truth (the §Footprint header itself says the estimates are "to be refined against live data —
  not measured benchmarks"). Fabricating a "~X MB/day" number would assert an un-run measurement, which
  is worse than the honest qualitative answer. So this is **`low`, not a re-block**: the critical's bar is
  met as well as in-repo data allows, the prune stays correct, and a concrete rate is a live-data
  refinement (a human/infra observation), NOT a loop-closeable doc edit. Fix when the testnet instance has
  run long enough to measure: record an OBSERVED per-hub BLOB-growth rate (e.g. MB per N records, or per
  day on each testnet hub) in §Footprint, replacing the qualitative-only disk clause. Verify fixed:
  §Footprint states a measured/estimated disk-growth rate (bytes per record or per day) for the testnet
  hubs, not just "proportional to activity". Low — skipped by the loop until live data exists.
- **Spec:** ADR-0007 mirror growth / per-network DB sizing; ADR-0013 M-Deploy footprint note; CLAUDE.md
  "Coverage" (never imply a guarantee the data does not support — including a fabricated sizing number);
  the (pruned) egress+footprint critical's disk-growth-rate Verify clause.


## `schemaDeclaration`/`schemaDeletion` note-schema URIs are now triplicated (certificate + proofserve + dashboard)
- **Priority:** low
- **Source:** [out-of-loop UI work] (2026-06-23, adding the dashboard's declaration filter)
- **What / where / how to verify:** the full wire URIs `http://purl.org/iscc/schema/iscc-note-0.8.0.json`
  (declaration) / `…iscc-note-delete-0.8.0.json` (deletion) are defined as unexported consts in BOTH
  `internal/certificate/handler.go` and `internal/proofserve/handler.go`, and the dashboard's "Recently
  declared" filter added a THIRD copy of the declaration URI in `internal/dashboard/handler.go`. Three
  copies of a version-bearing URI is a DRY/drift risk: an `iscc-note-0.9.0` bump must touch three files,
  and the store deliberately stays schema-agnostic (ADR-0008) so it is NOT the home. Fix when next
  touching any of the three: hoist a shared, exported constants leaf (e.g. `internal/notes` with
  `SchemaDeclaration`/`SchemaDeletion`) and have all three view packages reference it; keep the store
  schema-agnostic. Verify fixed: exactly one definition of each URI, referenced by cert + proofserve +
  dashboard. Low — skipped by the loop until one of those files is next edited.
- **Spec:** DRY (CLAUDE.md code standards); ADR-0008 (schema interpretation lives in the view layer, not
  the store); no spec contract.

## The out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)`, so a downgrade-from-newer-binary still re-applies the baseline DDL before the reject
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed against `Open` ordering)
- **What / where / how to verify:** `Open` (`internal/store/sqlite.go:142,146`) runs `db.Exec(schemaSQL)`
  BEFORE `applyMigrations`, and the new out-of-range guard (`internal/store/sqlite.go:181-184`, advance
  `ed3206d`) lives INSIDE `applyMigrations` — so the guard rejects a `user_version > len(migrations)` DB
  (one written by a NEWER binary, then opened by this OLDER one) only AFTER the baseline DDL has already
  run. Reviewer-confirmed the residual is strictly bounded: `schemaSQL` is ENTIRELY
  `CREATE TABLE/INDEX IF NOT EXISTS` (grep-verified — NO `DROP`/`ALTER`/`DELETE`/`UPDATE`/`INSERT`), so on
  a downgrade it can only RE-CREATE a table/index a newer schema had dropped or renamed; it can never
  alter or corrupt existing data. NOT reachable today — this is the FIRST migration (`len(migrations)==1`),
  there is no newer binary, and the guard this advance added is strictly STRONGER than the prior no-guard
  state (the migration-layer reject is mutation-proven by `TestMigrationOutOfRangeVersion`). The
  `schemaSQL`-before-guard ordering is PRE-EXISTING (not introduced by this advance — only the guard is
  new), so this is a narrower defense-in-depth gap, same class as the other latent `low`s, not a
  regression. Does NOT block progress; all gates green, every `next.md` Verify met. Fix when `Open` /
  the migration runner is next touched: read `PRAGMA user_version` and apply the `version < 0 || version >
  len(migrations)` reject BEFORE `db.Exec(schemaSQL)` (hoist the guard out of `applyMigrations` into `Open`
  ahead of the schema pass, or split a `checkSchemaVersion` step), so an unsupported on-disk version
  fails closed without the baseline DDL touching the DB at all. Verify fixed: opening a DB whose
  `user_version > len(migrations)` returns the wrapped error AND leaves the schema untouched (a probe
  that drops a baseline table on a future-version DB finds it still dropped after the failed Open);
  reverting the hoist re-creates it.
- **Spec:** correctness rule 6 "fail-closed" discipline; `store.Open` docstring + `deploy/OPERATING.md`
  §Migration-policy ("fail-closed … never leaving a half-migrated database"); CLAUDE.md "fail-closed"
  posture; `learnings/store.md` migration-runner note.

## Composite-PK rebuild dropped `seq`'s standalone ordering path — `RecentRecords`' `ORDER BY i.seq DESC` now sorts instead of walking an index
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed against the schema + `RecentRecords` query)
- **What / where / how to verify:** Re-keying `iscc_index` from `seq INTEGER PRIMARY KEY` (the rowid) to
  the composite `PRIMARY KEY (hub_id, seq)` (advance `ed3206d`) removed the standalone ordering path on
  `seq`: `seq` is now the SECOND column of the composite PK, so there is no index SQLite can walk for the
  realm-wide `ORDER BY i.seq DESC` in `RecentRecords` (`internal/store/iscc_index.go:237`) — on a populated
  monitor that query now scans + sorts `iscc_index` instead of walking the old rowid order in reverse. This
  is a PERFORMANCE observation, NOT a correctness defect: the query returns the right rows in the right
  order (mutation-proven by `TestRecentRecords`); only the access path changed. The cost is negligible at
  current scale — `RecentRecords` is realm-wide with a small `LIMIT n` over a 2-hub testnet — and `next.md`
  explicitly scoped this step to the PK rework + left `RecentRecords`' ordering as-is (Not In Scope: "Do NOT
  re-key `RecentRecords`' cross-hub ordering"). Does NOT block progress; all gates green. Fix when the
  dashboard-recent path or `iscc_index` schema is next touched (and only if a populated monitor shows the
  sort as a hot path): add `CREATE INDEX IF NOT EXISTS iscc_index_by_seq ON iscc_index (seq)` to
  `schema.sql` AND recreate it inside migration 0's rebuild (append to its `stmts`, since a released
  migration's effect must converge with the fresh-DB schema). Verify fixed: `EXPLAIN QUERY PLAN` for the
  `RecentRecords` query uses the `seq` index (no `USE TEMP B-TREE FOR ORDER BY`); the fresh-DB schema and
  the migrated DB both carry the index. Low — skipped by the loop; a scale-time refinement, not a defect.
- **Spec:** ADR-0007 per-network store sizing / scaling trip-wire; ADR-0008 schema-agnostic index;
  CLAUDE.md `GET /` "Recently declared" surface; `learnings/store.md` `RecentRecords` ordering note.

## OpenAPI contract omits `/healthz`'s 503 store-down readiness response (only 200 documented)
- **Priority:** low
- **Source:** [review] (Codex P3, reviewer-confirmed against `healthz.Handler`)
- **What / where / how to verify:** `internal/openapi/openapi.yaml:45-51` documents only a `200` response
  for `GET /healthz`, but `healthz.Handler` (`internal/healthz/handler.go`) returns `503` +
  `{"status":"unavailable"}` (`application/json`) when the store `Ping` fails — the endpoint's PRIMARY
  failure mode and the whole point of a readiness probe. A readiness-check client generated from
  `/openapi.json` therefore treats the store-down case as undocumented. NOT a code regression (healthz is
  correct); the contract is incomplete. Does NOT block progress (gates green); low because a readiness
  client typically checks the status code regardless and the 200 path is documented. Fix when the OpenAPI
  doc is next touched: add a `503` response to the `/healthz` operation (`{status: unavailable}`,
  `application/json`) in both docs and regenerate the JSON twin. Verify fixed: the served `/openapi.json`
  `healthz` operation declares both `200` and `503`. Low — skipped by the loop until the doc is next edited.
- **Spec:** ADR-0014 §1 (document the machine surface's real responses); `internal/healthz/handler.go`
  (the 503 unavailable path); CLAUDE.md `GET /healthz` ("liveness + store readiness").
