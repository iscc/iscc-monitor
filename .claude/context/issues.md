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

## No on-disk DB migration story — a column added to an existing table never reaches a pre-existing database
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed against `store.Open`; codebase-wide pre-existing gap)
- **What / where / how to verify:** `store.Open` (`internal/store/sqlite.go:72`) applies the embedded
  `schema.sql` as one `db.Exec(schemaSQL)` whose every statement is `CREATE TABLE IF NOT EXISTS` (9
  tables, ZERO `ALTER TABLE`, no `PRAGMA user_version`, no migration framework — reviewer grep-confirmed).
  So a column added to an EXISTING table (here `iscc_index.note_timestamp`, but this applies to EVERY
  column ever added: `note_schema`, `record_sha256`, the OTS columns, the freeze columns, …) is a silent
  no-op on a database created before that commit. An upgraded node opening such a DB would then fail the
  new INSERT/SELECT paths with `no such column: note_timestamp`. This is **not a regression of the
  timestamp slice** — it is a pre-existing, codebase-wide property: every prior column landed the same
  way, and `next.md` Not-In-Scope explicitly chose it (dev DBs are ephemeral; no schema-versioning
  framework). Does NOT block this increment (fresh DBs — the only deployed kind so far — get the column;
  all gates green). It becomes a real operational hazard the first time the monitor needs an in-place
  upgrade over a populated production DB. Fix when a deliberate migration step is scheduled: introduce a
  `PRAGMA user_version`-gated (or `ALTER TABLE ADD COLUMN`-idempotent) migration mechanism applied on
  `Open` AFTER the `CREATE TABLE IF NOT EXISTS` pass, covering all post-bootstrap columns; this is a
  design decision (it is the project's FIRST migration mechanism), not a field-slice side effect. Verify
  fixed: opening a DB seeded with the pre-`note_timestamp` `iscc_index` DDL then running an ingest +
  `RecordAt` succeeds (column auto-added), with a test that seeds the old schema and asserts no
  `no such column` error.
- **Spec:** ADR-0007 one-file-per-network store; CLAUDE.md "Irreplaceable evidence" (a populated prod DB
  that cannot be upgraded in place is a backup/continuity risk); `next.md` Not-In-Scope migration note.

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

## `/` realm-index: only the "recent declarers checked" hero footer remains vs the mockup (logo, instance-identity, Checkpoint/Anchor all CLOSED)
- **Priority:** normal
- **Source:** [review] (visual pass vs the Realm-Index mockup, after the named-region parity landed)
- **What / where / how to verify:** The three HEADLINE landmark regions (claim-lookup hero, per-row
  dossier link, instance-identity masthead) now render and the lone `critical` is closed — but the
  ADR-0012 visual pass against `.claude/design/ISCC Monitor - Realm Index.dc.html` shows four remaining
  sub-region deltas the parity step deferred as constraint-wins (all flagged in that handoff):
  (1) **No logo** — **CLOSED (reviewer-confirmed `6a442b4`).** The self-hosted logo now renders on ALL SIX
  SSR mastheads (`/`, dossier, certificate, the three proofserve surfaces); the ADR-0012 visual pass shows no
  remaining "no logo" delta on any surface. No further action — kept here only as a resolved sub-item record.
  (2) **Static instance identity + realm name** — **CLOSED for `/` (reviewer-confirmed `b30b84e`).** The
  `/` masthead now renders three operator-supplied strings (`dashboard.Identity{Instance, Operator, Realm}`)
  flowing env → binary → page via `ISCC_MONITOR_INSTANCE` / `ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME`
  (the last distinct from config's required realm-document PATH `ISCC_MONITOR_REALM`), with handler-side
  fail-safe fallback to today's static copy when unset. Visual pass confirms the live binary renders
  `monitor.iscc.id` / "instance operated by ISCC Foundation · ISCC mainnet" / "REALM REGISTER · ISCC MAINNET"
  matching the mockup; mutation-proven non-vacuous (template binding + wiring). The SAME `dashboard.Identity`
  value still needs threading into the OTHER five SSR mastheads (dossier, certificate, the three proofserve
  surfaces) — tracked separately as the follow-on arc; `internal/verifier` stays excluded (`.codes` chrome).
  Kept here only as a resolved sub-item record for the `/` surface.
  (3) **Checkpoint-size + Bitcoin-anchor data columns** — **CLOSED (reviewer-confirmed `b74931f`).** The
  `/` ledger now renders the mockup's six columns `# | Hub · domain | Coverage since | Checkpoint | Anchor
  | Status`; `store.HubSummary` gained a read-only `Anchor` projection (latest-stamped-root OTS status via
  a correlated subselect) and the Checkpoint cell re-purposes the accepted `LastSize`. Visual pass vs the
  mockup confirms column order + naming; mutation-proven non-vacuous; store stays a leaf. The per-hub
  (vs per-checkpoint) honesty design question Codex raised is its own `normal` below — kept here only as a
  resolved sub-item record.
  (4) **"Recent declarers checked" hero footer** — **CLOSED (direct human design tweak, out-of-loop,
  2026-06-23).** Shipped with a twist that sidesteps the missing lookup-history: instead of "recent
  declarers *checked*" (which needs a lookup log the store does not track), the hero now renders a
  **"Recently declared:"** row — the newest *declarations* the monitor has indexed (which the store DOES
  have: `store.RecentRecords`, realm-wide, accepted-tree-bounded, schema-agnostic), filtered to
  declarations + de-duped + capped in the dashboard view layer, each a click-through link to its
  Certificate of Inclusion (`/inclusion/<id-body>`). Empty index → the row is omitted (honest empty
  state). Tests: `store.TestRecentRecords`/`TestRecentRecordsEmpty`,
  `dashboard.TestDashboardRecentlyDeclared`/`TestDashboardNoRecentRowWhenIndexEmpty`. The hero copy was
  also changed in the same tweak ("Prove a specific ISCC declaration is in the log." → "ISCC-ID
  Verification") and the input placeholder made paler. **All four sub-items of this issue are now CLOSED**
  — the next `update-state` may prune this entry.
- **Spec:** target.md M-UI design-parity "named-region" bar (the `/` realm-index region) + "Document chrome
  + instance identity"; ADR-0010 Evidence-Ledger handoff; ADR-0012 visual-pass.

## The WASM verifier never checks the checkpoint signature against the hub's did:web key (the signature half of the verifier-scope trust gap; id-binding half now CLOSED in source)
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed against the verify core; affects BOTH tier-2 callers)
- **What / where / how to verify:** UPDATE: the **id-binding half** of this gap is now CLOSED IN SOURCE
  (advance `22f0420`): `verifyadapter.RecordCommitsID` binds the record's committed `iscc_id` to the
  requested id, and the 6-arg `isccVerifyInclusion` shim gates `verified` on it (the cross-origin
  `verifier.html:628` passes `target.id`). What REMAINS open is the **signature half**:
  `isccVerifyInclusion` (`VerifyJSON` → `internal/proof/verify.VerifyInclusion`) still verifies ONLY
  that `record`+`proof`+`size`→`root` (RFC-6962 inclusion) and the id-binding — it does NOT verify the
  checkpoint note signature against the hub's did:web key. So a malicious/compromised monitor can still
  return a bundle whose record+proof+root are internally consistent under a FORGED (unsigned / wrong-key)
  checkpoint and — provided the record commits the requested id — the browser renders the green
  `verified` state, trusting the monitor for the signature. The copy overstates this: `verifier.html:449`
  lists "Check the signature against the hub's did:web key" as a step the verifier WILL run, and
  `verifier.html:631` reports "✓ … re-verified this inclusion proof against the **hub-signed** checkpoint
  root" — but neither the signature nor a did:web resolution runs. This is the SAME verifier-core scope
  the certificate's same-origin tier-2 ships (`cert.html:565`), so it is NOT a regression and does NOT
  block progress; but it is more serious cross-origin. NOT currently exploitable on the testnet (the
  fixture monitor is honest). It needs a DESIGN PASS (browser did:web resolution + note-signature verify)
  — review flagged it as the design-first remainder. Fix when the WASM verifier scope is next expanded:
  extend the verifier (or a sibling export) to verify the checkpoint note signature against a did:web key
  fetched/resolved in the browser, gating `verified` on signature + id-binding + inclusion; until then,
  narrow the success copy + drop the unrun did:web step from the record block so the page does not claim a
  signature/key check it skips. Verify fixed: a bundle with a valid inclusion proof + matching id but a
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


## `iscc_index.seq` is a single global PRIMARY KEY but ingest writes per-hub absolute leaf indices — multi-hub PK collision
- **Priority:** normal
- **Source:** [out-of-loop UI work] (surfaced while adding `store.RecentRecords` for the dashboard "Recently declared" row, 2026-06-23)
- **What / where / how to verify:** `schema.sql` declares `iscc_index(seq INTEGER PRIMARY KEY, …)` — a
  single global rowid — yet `follower/ingest.go` folds projections at the per-hub ABSOLUTE leaf index
  (`logclient.BundleProjections(raw, bundleIndex*tiles.TileWidth)`), which restarts at 0 for each hub. So
  two followed hubs whose logs share a leaf index (every realm with ≥2 active hubs: both have seq 0, 1, …)
  COLLIDE on the PK, and `RecordProjections`' `ON CONFLICT(seq) DO UPDATE SET hub_id = excluded.hub_id, …`
  silently OVERWRITES the earlier hub's row with the later hub's. The index therefore cannot faithfully
  hold both hubs' low leaves; per-hub readers (`ListRecords`/`SeqsForISCCID`/`RecordAt`) still filter by
  `hub_id` so they only ever see the surviving (last-written) rows, and the realm-wide `RecentRecords`
  inherits the same clobbered set. Likely fix: composite PK `(hub_id, seq)` (and update the BLOB index +
  the `ON CONFLICT` target accordingly), with a migration story (see the existing no-migration issue).
  Verify fixed: two hubs can both index seq 0 without one clobbering the other (a store test seeding
  `{hubA,0}` and `{hubB,0}` then reading both back per hub). NOT introduced by the dashboard work — that
  feature only reads what is there; this is a pre-existing data-model bug it surfaced. Normal: it is a
  multi-hub correctness limitation, not a current crash, and the testnet realm is small.
- **Spec:** ADR-0008 schema-agnostic record index; ADR-0007 one-file-per-network; CLAUDE.md "Mirror" /
  "Projection". Pairs with the existing "No on-disk DB migration story" issue (a PK change needs one).

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

## Dossier §3 latest-checkpoint size and observed-time can come from different rows on a FROZEN hub
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against `follower.freeze`)
- **What / where / how to verify:** The §3 observed-time subselect added in `internal/store/hubs.go:69-70`
  picks the newest `checkpoints` row by `tree_size DESC, id DESC`, but §3 renders its SIZE from `.LastSize`
  (= accepted `follow_state.last_size`). For a verified hub these agree. For a FROZEN hub they can diverge:
  `follower.freeze` (`internal/follower/follower.go:475`) calls `RecordCheckpoint` for the contradictory
  (often HIGHER-tree-size) checkpoint WITHOUT advancing `last_size` (reviewer-confirmed: `RecordCheckpoint`
  never writes `last_size`; only `AdvanceAccepted`/`AdvanceFollowState` do). So a frozen hub whose rejected
  checkpoint is larger renders the accepted size (`§3 N entries`) paired with the REJECTED checkpoint's
  `observed_at` — an accepted size with a rejected timestamp. Confined to the frozen edge state, where the
  loud non-dismissable Exhibit already renders "do not trust new state from this hub" ABOVE §3, and a
  `shrink` violation cannot trigger it (the rejected size is smaller, so `tree_size DESC` still picks the
  accepted row). Does NOT block this increment — all gates green, the verified/pending/confirmed primary
  paths are correct, and increment 2 reworks §3 + the Exhibit. Fix when §3 is next touched (likely
  increment 2): select `observed_at` for the row whose `tree_size = f.last_size` (or render §3 size + time
  from ONE checkpoint row), so the two halves always describe the same checkpoint. Verify fixed: a frozen
  fixture whose rejected checkpoint has a higher tree_size renders §3 size + observed-time from the SAME
  (accepted) checkpoint; reverting makes the time track the rejected row.
- **Spec:** CLAUDE.md "Coverage" / "Self-consistency violation" (never imply a guarantee the data does not
  support); ADR-0001 coverage honesty; learnings.md always-loaded SSR-honesty rule; `learnings/dossier.md`
  §3 size/time-decouple note.

## Dossier §1 unconditionally says "Key resolved from did:web:…" even on the `unresolvable` overlay path
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed)
- **What / where / how to verify:** `internal/dossier/dossier.html:490` renders §1 Identity as "Key resolved
  from did:web:{{.Domain}}" UNCONDITIONALLY (§1 is static-derived, no network/store read — per next.md's
  Implementation Note + the mockup line 88, which use this phrasing because §1 is about WHERE the key comes
  from = domain ownership, not a per-request verdict). But when the live overlay reports `unresolvable`, the
  soft caution copy on the SAME page says "the monitor cannot currently fetch or parse this hub's did:web
  document, so its signing key is unresolved" — so §1 asserts a successful key resolution exactly on the
  failed-resolution path. A self-contradiction confined to the `unresolvable` state; it overstates §1 but
  does not falsely claim verification of any proof/signature. Does NOT block this increment — the advance
  followed next.md + the mockup literally, all gates green. This is design-rooted (the mockup specifies the
  static "resolved" phrasing), so do NOT silently rewrite the mockup-specified copy without a design pass.
  Fix when §1 is next touched / a design pass runs: use neutral source wording ("Key source: did:web:<domain>")
  OR gate the word "resolved" off the `unresolvable` status (e.g. render "Key source unresolved" in §1 when
  `ShowCaution` for `unresolvable`). Verify fixed: a fixture hub with a live `unresolvable` verdict does NOT
  render "Key resolved from" in §1; reverting makes it reappear.
- **Spec:** CLAUDE.md "Hub status" (unresolvable = can't resolve the key) + "did:web key resolution";
  ADR-0010 Evidence-Ledger honesty; learnings.md always-loaded SSR-honesty rule; `.claude/design/ISCC
  Monitor - Hub Dossier.dc.html` §1 static phrasing; `learnings/dossier.md` §1 note.

## OpenAPI contract advertises a phantom `index` query param on `/{domain}/log/verify` the handler never reads
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against `serveVerify`)
- **What / where / how to verify:** `internal/openapi/openapi.yaml:237-243` (+ the JSON twin) declares an
  optional `index` query parameter on `GET /{domain}/log/verify` described as "An explicit committed leaf
  sequence to verify". But `serveVerify` (`internal/proofserve/handler.go`) NEVER reads `index` — its own
  comment is explicit: "verify-for-me takes no index param, so seqs[0] is the deterministic subject" — it
  always verifies `seqs[0]` (the lowest committed sequence). For an ISCC-ID with multiple committed leaves,
  a client generated from this contract sends `?index=<n>` and silently receives a verdict for a DIFFERENT
  leaf (`seqs[0]`), with no error. This is the most client-misleading of the three contract-accuracy gaps:
  the param looks supported and the request succeeds, so the divergence is invisible. The drift test gates
  PATHS, not params, so it cannot catch this (the path is correct; only the operation's param list is
  wrong). NOT a code regression — `serveVerify` is correct; the CONTRACT overstates it. Does NOT block
  progress (gates green; the increment met every `next.md` Verify criterion). Fix when the OpenAPI doc is
  next touched (likely M-API slice 3, the `/docs` slice): EITHER remove the `index` parameter from the
  `/{domain}/log/verify` operation in both `openapi.yaml` and `openapi.json` (regenerate the JSON twin from
  the YAML), OR — only with a design decision — implement `index` selection in `serveVerify` so the
  contract becomes true. Verify fixed: the `verify` operation in the served `/openapi.json` declares no
  `index` parameter (or `serveVerify` reads it and verifies that leaf); add a golden asserting the verify
  operation's param set matches the handler's actual query params.
- **Spec:** ADR-0014 §1 (the contract describes the real machine surface, accurately); CLAUDE.md
  "verify-for-me" (the monitor reports a verdict — for `seqs[0]`, the deterministic subject); next.md
  Implementation Note (the contract must describe the documented response shapes faithfully).

## OpenAPI contract advertises `text/plain` for `/{domain}/log/checkpoint` but the handler serves `application/octet-stream`
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed at the live mux seam)
- **What / where / how to verify:** `internal/openapi/openapi.yaml:265-270` (+ the JSON twin) advertises the
  `200` response of `GET /{domain}/log/checkpoint` as `text/plain`. But that route is served by tilesserve
  (the `/`-fallthrough in `hubHandler`, since `checkpoint` is NOT an exact proofserve mount), whose
  `writeBlob` (`internal/tilesserve/handler.go:170`) unconditionally sets `Content-Type:
  application/octet-stream` (the package `contentType` const, line 32). Reviewer-confirmed by probing the
  real `buildMux` mux: `GET /sb0.iscc.id/log/checkpoint` → `200`, `Content-Type: application/octet-stream`.
  So an OpenAPI validator / generated client expects `text/plain` and gets `octet-stream` for this route —
  a media-type mismatch (the `.ots` route at `:289` already correctly says `application/octet-stream`; only
  the plain `/checkpoint` is wrong). NOT a code regression (tilesserve is correct — the checkpoint is an
  opaque signed-note BLOB); the CONTRACT is inaccurate. Does NOT block progress (gates green). Fix when the
  OpenAPI doc is next touched: change the `/{domain}/log/checkpoint` `200` content key from `text/plain` to
  `application/octet-stream` in both docs and regenerate the JSON twin. Verify fixed: the served
  `/openapi.json` `checkpoint` operation's `200` content type equals what the live mux sends
  (`application/octet-stream`); a probe comparing the two matches.
- **Spec:** ADR-0014 §1 (accurate machine-surface contract); CLAUDE.md `GET /<domain>/log/checkpoint`
  ("served verbatim as `application/octet-stream`" is how the sibling `.ots` is described); next.md
  "describe the documented response shapes faithfully".

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

## No machine-readable API contract (OpenAPI) and no interactive API docs hosted by the app
- **Priority:** normal
- **Source:** [human]
- **STATUS — slices 1+2 LANDED (advance `e2de5e6`, reviewer-verified):** the in-repo OpenAPI 3.1 document
  (`internal/openapi/openapi.yaml` + byte-distinct JSON twin) is served byte-verbatim at `GET /openapi.json`
  + `GET /openapi.yaml` under the CORS `*` wrap with the `no-cache`+strong-ETag+304 policy, covering the 13
  machine paths, excluding the HTML SSR surfaces, with `verify-for-me` flagged weaker in-band — and a
  NON-VACUOUS route↔spec drift test (`cmd/iscc-monitor/openapi_drift_test.go`, reviewer mutation-confirmed
  both directions). What REMAINS to close this issue is **slice 3: `GET /docs` + the self-hosted, byte-pinned
  Stoplight Elements assets** (the `<elements-api>` JS+CSS under `/_ds/` with published `internal/web` hash
  constants, `apiDescriptionUrl="/openapi.json"`, no `tryItCorsProxy`). Three contract-accuracy defects the
  drift test cannot catch (it gates PATHS, not params/media-types/responses) are filed as their own entries
  above (the phantom `verify` `index` param + the `checkpoint` media type are `normal`; the `healthz` 503 is
  `low`) — fold those fixes into slice 3 or a doc-touch.
- **What / where / how to verify:** The monitor exposes a machine-consumable HTTP surface
  (`/healthz`, `/version`, `/metrics`, the per-hub `inclusion`/`consistency`/`entries`/`checkpoint`/
  `checkpoint.ots`/`tile` routes, `verify-for-me` at `/<domain>/log/verify`, and the
  `/inclusion/<iscc_id>.bundle` proof bundle — all CORS `*`, wired in `cmd/iscc-monitor/main.go`
  `buildMux`), but there is **no OpenAPI document and no interactive docs**. The only reference is
  prose in `CLAUDE.md`; the response shapes exist solely as Go structs (`VerifyVerdict`,
  `ConsistencyEvidence`, `InclusionEvidence`, the bundle) in `internal/proofserve` / `internal/logclient`
  / `internal/certificate`. A third-party integrator has nothing to generate a client from or validate
  against, and no in-browser way to explore the API. Implement per ADR-0014: (1) a hand-authored
  **OpenAPI 3.1** document in-repo covering the machine-consumable surface ONLY (HTML SSR surfaces
  excluded), with `verify-for-me` flagged in its `description` as the weaker non-authoritative path; (2)
  serve it byte-verbatim via `go:embed` at `GET /openapi.json` + `GET /openapi.yaml` under `corsmw`; (3)
  host **Stoplight Elements** interactive docs at `GET /docs` — the `<elements-api>` web component (its JS
  bundle **and** stylesheet) **self-hosted under `/_ds/`**, each byte-pinned with a published
  `internal/web` hash next to `WasmVerifyHash` (same strong-ETag + no-cache + 304 policy as `verify.wasm`),
  with `apiDescriptionUrl="/openapi.json"` and **no** `tryItCorsProxy` set so "try it" goes
  browser→this-instance directly on the existing CORS `*` — NO external CDN / external runtime
  call. Verify fixed (all at the HTTP seam against fixtures + golden): `GET /openapi.json` returns `200`
  + a valid OpenAPI 3.1 body byte-equal to the embedded doc and carries `Access-Control-Allow-Origin: *`;
  a **drift test** asserts every path the doc declares is mounted in the real mux AND every
  machine-consumable route the mux mounts is declared (HTML SSR routes on an explicit exclusion list) —
  adding/renaming a JSON route without updating the doc, or documenting a removed route, FAILS it; `GET
  /docs` returns `200 text/html` with **no external host in the body or in any request it makes** and
  loads the pinned `/_ds/` Stoplight Elements assets (JS + CSS) whose served bytes hashes match the
  published constants; `mise run check` green and no gate weakened. Order-independent (like M-Deploy) —
  no feature milestone gates it, and the touched routes add no new crypto/proof path so the oracle gate
  is N/A.
- **Spec:** ADR-0014 (the authoritative decision); target.md M-API + "Done When"; CLAUDE.md endpoint
  reference (the prose this makes machine-readable); the "verify.wasm pin is fragile" learning (the
  Stoplight Elements assets join the same pin discipline).
