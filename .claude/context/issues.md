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

## Adopt the iscc-lib Go codec + bump the toolchain to Go 1.26 (ADR-0011)
- **Priority:** normal
- **Source:** [human]
- **What / where / how to verify:** Land ADR-0011's adoption — reuse `iscc/iscc-lib` for ISCC
  en/decoding instead of owning a second hand-rolled codec. Sequence it **before** more M-UI feature
  work (it is a foundational stack change). Steps:
  1. **Bump the locked toolchain Go 1.24 → 1.26** across `mise.toml` (`go = "1.26"`), `go.mod` (`go
     1.26.x`, plus a `toolchain` line only if the pinned patch needs it), `.github/workflows/ci.yml`,
     and `.devcontainer/` — `iscc-lib`'s `packages/go/go.mod` requires `go 1.26.1`.
  2. **Add `github.com/iscc/iscc-lib/packages/go` (pinned v0.5.0)** to `go.mod` / `go.sum`.
  3. **Make the dependency real + encode the migration trigger** with ONE tripwire/parity test in
     `internal/index` (a `_test.go` / sibling test package — NOT `iscc.go`) that imports iscc-lib and
     asserts `IsccDecode` currently **rejects** a known ISCC-IDv1 (`"MAIGHFECJMOPMIAB"`, Version 1)
     with a Version error. When [iscc/iscc-lib#43](https://github.com/iscc/iscc-lib/issues/43) lands,
     that assertion flips RED and signals: migrate `internal/index.Decode` to iscc-lib and delete the
     port. Production `internal/index/iscc.go` must stay a **pure stdlib-only leaf** (no iscc-lib
     import) so `GOOS=js GOARCH=wasm go build ./internal/index` still succeeds — only the test imports
     iscc-lib.
  - **Verify fixed:** `go.mod` requires Go 1.26 and `iscc-lib/packages/go` v0.5.0; the tripwire test
    is present and green (asserts today's rejection); `mise run check` is green under a Go 1.26
    toolchain; `GOOS=js GOARCH=wasm go build ./internal/index` still succeeds.
  - **Toolchain caveat:** local dev at filing time is Go 1.24; this increment must run where mise can
    provision Go 1.26. Do **not** flip `go.mod`'s `go` directive without the 1.26 toolchain present —
    it reds the gate for the whole module.
- **Spec:** ADR-0011; target.md "Stack (locked — ADR-0003, ADR-0011)"; iscc/iscc-lib#43.

## Hub-List `hubDomain` accepts a scheme'd path-bearing URL (fail-open against the claimed contract)
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed)
- **What / where / how to verify:** `internal/registry/registry.go` `hubDomain` (lines 161-173) only
  rejects a url with an empty `Host`, so a scheme'd path-bearing url like `https://sb0.iscc.id/log`
  parses cleanly and `Resolve` silently returns `sb0.iscc.id`, dropping `/log`. This contradicts
  (a) `next.md`'s Implementation Notes ("Fail closed... a `url` that is empty/**path-bearing** — so a
  path is never silently coerced into a domain"), (b) the function's own docstring ("fails closed on a
  url that... carries no host (e.g. a bare path)"), and (c) the advance handoff's claim
  ("fail-closed on... empty-or-path-bearing url"). The guarding test `TestParseHubListErrors`
  case `"path-bearing url"` (hublist_test.go) uses the **scheme-less** `sb0.iscc.id/log`, which is
  rejected via the *no-host* branch, NOT a path branch — so it gives false confidence and a genuine
  scheme'd-path url slips through (reviewer-confirmed: `hubDomain("https://sb0.iscc.id/log")` returns
  `("sb0.iscc.id", nil)`). Not currently exploitable (live wiring deferred; the `testnet.yaml` fixture
  uses clean `https://host` urls), but it is a trust-root-adjacent resolver (a wrong domain proves the
  wrong leaf, per the index learnings). Fix: in `hubDomain`, after the host check, reject a non-empty
  `u.Path` (and likely `RawQuery`/`Fragment`) with a wrapped "path-bearing"/"not a bare host base url"
  error; then replace/augment the test's `"path-bearing url"` case with a **scheme'd** path url
  (`https://sb0.iscc.id/log`) so the gate is non-vacuous. Verify fixed: `ParseHubList` with a
  `https://host/path` url returns a non-nil error + nil list, and reverting the new `u.Path` check
  makes that test FAIL.
- **Spec:** next.md "Fail closed, like Parse" Implementation Note; ADR-0010 Hub-List schema; CLAUDE.md
  "did:web key resolution" / registry-rejects-URL-shapes precedent (`Parse` rejects path separators).

## Hub-List entry missing `hub_id` silently becomes slot 0 (YAML zero-value fail-open)
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed)
- **What / where / how to verify:** `internal/registry/registry.go` `Hub.HubID uint16` (line 86) with
  `yaml:"hub_id"`: a Hub-List entry that omits or misspells `hub_id` unmarshals the YAML zero value, so
  the entry silently becomes slot **0** and passes the range/duplicate checks (reviewer-confirmed: a
  `hubs:` entry with only `url`/`active` parses with `HubID == 0` and `Resolve(0)` returns its domain).
  In a real Hub-List where slot 0 is otherwise absent, a typo'd `hub_id` would map that hub to slot 0
  and mis-resolve hub-0 ISCC-IDs to the wrong hub while leaving the intended slot unresolved — a
  fail-open in a trust-root-adjacent resolver. Not required by `next.md` (it named only invalid YAML /
  over-range / duplicate / empty-or-path url), so this is hardening beyond the original scope, but it is
  the same fail-open class as the path-bearing gap above. Fix when `ParseHubList` is next touched:
  decode with presence tracking — either decode each entry to a `yaml.Node` (or a struct with
  `HubID *uint16` / a custom `UnmarshalYAML`) and reject an absent `hub_id` with a wrapped error. Verify
  fixed: a `hubs:` entry omitting `hub_id` returns a non-nil error + nil list; the valid fixtures still
  parse.
- **Spec:** next.md "Fail closed, like Parse" Implementation Note (extends it); ADR-0010 Hub-List
  schema (`hub_id` is a required field).

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

