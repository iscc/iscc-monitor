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

## ISCC-IDv1 decoder accepts a nonzero Length nibble (fail-closed gap on the trust root)
- **Priority:** normal
- **Source:** [review] (Codex-found P2, reviewer-confirmed against the hub schema + golden vectors)
- **What / where / how to verify:** `internal/index/iscc.go` `Decode` (the realm-wide certificate's
  trust-root decoder) validates the header's MainType (`raw[0]>>4 == 6`) and Version (`raw[1]>>4 == 1`)
  nibbles but never checks the **Length nibble** (`raw[1] & 0xF`). For a canonical 64-bit ISCC-IDv1 the
  Length nibble is **0** (both real golden vectors `MAIGHFECJMOPMIAB` / `MEIGHFECJMOPMIAC` have byte1 =
  0x10). A header like `MAIQAAAAAAAAAAAA` (byte1 = 0x11, Length nibble 1) is NOT a valid v1 ID — it
  describes a different/truncated body — yet `Decode` accepts it and mis-reads bytes 2:10 as the 64-bit
  body, so a malformed id is routed to a hub instead of rejected. This violates the explicit
  fail-closed contract (`next.md`: "a header that is not an ISCC-IDv1 ... must return a descriptive
  error") and the file's own docstring, which lists Length as a header nibble. Fix: guard
  `raw[1] & 0xF == 0` (reject otherwise) before reading the body, and add a golden malformed case for
  `MAIQAAAAAAAAAAAA`. Verify fixed: `Decode("MAIQAAAAAAAAAAAA")` returns a non-nil error; both existing
  golden vectors still decode (their Length nibble is 0); a mutation flipping the guard makes a test
  FAIL. Confirmed harmless to every valid id (all real v1 ids have Length 0), so it is `normal`, not
  `critical` — it does not corrupt any legitimate decode and does not block the resolver sub-step, but
  it must land before this decoder is consumed on the certificate trust path.
- **Spec:** ADR-0010:83-88 (ISCC-IDv1 header layout); `next.md` fail-closed contract;
  iscc-hub `schema.py:65` (64-bit body = 52-bit ts + 12-bit hub).

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

