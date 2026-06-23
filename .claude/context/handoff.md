## 2026-06-23 — Dress the record-list log browser with the shared chrome + `← dossier` breadcrumb + the "Log browser" head (record-list parity, part 2a)

**Done:** Threaded the hub `Domain` + the operator-supplied `dashboard.Identity` through
`proofserve.Handler` → `hubHandler` → `serveRecords`, resolving the masthead identity once at
`Handler` construction (mirroring dossier's `resolveIdentity` + fallback consts). Dressed
`records.html` with the Log-Browser mockup's shared chrome (instance-identity block + `verify ↗
monitor.iscc.codes` link), the absolute-site-root `← {{.Domain}} dossier` breadcrumb, and the
"Log browser" eyebrow / `{{.Domain}}` head / "<domain> · {{.Total}} records mirrored" sub-line.
The pager rework is left for part 2b (untouched), as scoped.

**Files changed:**
- `internal/proofserve/handler.go`: added `internal/dashboard` import + `instanceFallback`/
  `operatorFallback` consts + `resolveIdentity`; extended `Handler` signature to
  `Handler(st, hubID, domain string, statuses, id dashboard.Identity)` (resolves identity once);
  added `Domain`/`Instance`/`Operator` to `recordsData` + `serveRecords` signature/population.
- `internal/proofserve/records.html`: chrome `.chrome-actions`/`.chrome-identity`/`.chrome-verify`
  CSS + markup, `.backlink-row`/`.log-head` CSS, the breadcrumb, and the Log-browser head;
  replaced the generic "Records" `<h1>` with the eyebrow/name/sub-line head.
- `cmd/iscc-monitor/main.go`: `hubHandler` gained `domain string` + `id dashboard.Identity`,
  passed into `proofserve.Handler`; `mirrorHandler` now passes `r.Domain` + `id`.
- 8 proofserve `*_test.go` files: mechanically updated all 62 `Handler(...)` call sites for the
  new signature (test domain `"sb0.iscc.id"` + zero-value `dashboard.Identity{}`) and added the
  `dashboard` import. `records_test.go`: refined `TestRecordsLinksTokensNoCDN`'s no-CDN ban to the
  dossier/browser pattern (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`, tolerating the
  same-federation `https://monitor.iscc.codes` verify link) + added two new mutation-proven tests
  (`TestRecordsHeadAndBreadcrumb`, `TestRecordsChromeInstanceIdentity`).

**Verification:** `mise run check` → green (build + vet + test across all 30 packages; `gofmt -l .`
empty). Per-criterion:
- [x] targeted `go test -run 'TestRecords|TestBrowser|TestRecord|TestInclusion|TestConsistency|
  TestEntries|TestOTS|TestVerify' ./internal/proofserve` — PASS (all 8 test files compile + pass).
- [x] record-list head/breadcrumb test asserts `href="/sb0.iscc.id"`, `← sb0.iscc.id dossier`,
  `Log browser`, head domain, `records mirrored` (+ honest `· 300 records mirrored` total) —
  mutation-proven: removing the breadcrumb FAILS, removing the eyebrow FAILS.
- [x] chrome instance-identity test asserts the `class="chrome-identity"` block + populated
  Instance/Operator + `href="https://monitor.iscc.codes/"`; the no-CDN assert stays green —
  mutation-proven: removing the chrome-identity block FAILS.
- [x] `internal/metrics` NOT in the proofserve dep closure (the real forbidden dep) — confirmed.
- [x] `go.mod`/`go.sum` byte-identical (no new module dep from the `dashboard.Identity` import).

**Next:** Part 2b — the pager rework: replace the bottom-only "showing N of M" pager with the
mockup's top+bottom pagers carrying the "seq X–Y of Z" range (disabled at the ends). Its own
≤3-file slice (`records.html` + `recordsData`/`serveRecords` view-model + test). After that, the
single-record-page chrome + `← Log browser` breadcrumb (`record.html`/`serveRecord`) is the next
parity slice.

**Notes:**
- **CONCURRENT-LOOP GIT RACE (MEMORY "Concurrent loop git race").** Mid-implementation a concurrent
  CID iteration committed `7adada3 cid(update-state)` + `8a9210a cid(define-next)` and ran
  `git reset` (reflog: two `reset: moving to HEAD`), wiping my entire uncommitted working tree —
  including the uncommitted **part-1** advance (`internal/dossier/dossier.html` href +
  `TestDossierBrowseLogLandsOnLiveRecordList`). **dossier.html is now back to the dead-end
  `href="/{{.Origin}}/"`** and the dossier no longer imports proofserve — i.e. **part-1 is LOST and
  must be re-done.** I re-applied all my part-2a work from scratch against the reset tree. Because
  part-1 was reverted, this slice no longer touches any dossier file (the dossier→proofserve test
  call I had patched no longer exists). Flagging so `review`/`define-next` reinstates part-1.
- **next.md dep-list grep is over-broad.** `go list -deps ./internal/proofserve | grep -E
  '^(internal/metrics|database/sql)$'` is NOT empty — it returns `database/sql`, but that is a
  pre-existing transitive dep through `internal/store` (proofserve has always been an HTTP handler
  over the SQLite store; confirmed `database/sql` present at HEAD before this change). The
  load-bearing check (the `dashboard.Identity` import pulled no NEW forbidden dep, and
  `internal/metrics` stays out of the closure) PASSES. The only new internal dep is
  `internal/dashboard` itself, for the `Identity` plain struct.
- **Constraint-win deviation (same as dossier):** the mockup head shows a prettified `hub.name`
  ("ISCC Foundation Hub") above the domain, but the store has no display name (`HubSummary` carries
  only `Domain`/`Origin`). The head name + breadcrumb use the bare `{{.Domain}}`, NOT a fabricated
  name — flagged in a code comment in `records.html`. Not a HUMAN REVIEW item; it is the identical
  deviation the dossier head already shipped.
- **Tracked `low` unchanged:** the masthead identity fallback consts (`instanceFallback`/
  `operatorFallback`) + `resolveIdentity` are now duplicated a 4th time (dashboard, dossier,
  certificate?, proofserve). Copied the literals with the "MUST stay byte-identical" comment as
  dossier does; the consolidation `low` covers it, out of scope here.
- **Oracle/conformance gate: N/A** — pure SSR chrome + view-model threading; no signature,
  RFC-6962, Merkle, did:web, or proof path is touched. Store stays a leaf (no query/schema change).
