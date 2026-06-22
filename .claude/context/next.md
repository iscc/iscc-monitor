# Next Work Package

## Step: Serve the mirrored OTS proof at `GET /<domain>/log/checkpoint.ots`

## Advances
The **OTS / Bitcoin anchoring** milestone Verify criterion (target.md §"OTS / Bitcoin anchoring"):

> stamp each distinct observed root daily … + background upgrade loop (pending → Bitcoin-confirmed) +
> **serve `.ots`**; never blocks the follower. **Verify:** a stamped root upgrades to Bitcoin-confirmed
> and the served `.ots` verifies with the standard `ots` client.

This step closes the **observable HTTP-surface half** of that criterion — the first served, parseable
`.ots` file keyed on the monitor's accepted `(size, root)`. The stamp→upgrade transit is already wired in
code (state.md); the criterion is still 1/1 open *only* because no `.ots` route exists, so the served
bytes cannot be handed to the standard `ots` client. state.md is explicit: "The single best next step is
now the `.ots` route" and "A step that adds another internal OTS seam instead of an observable HTTP
surface is drift." This is the observable surface, not another seam.

## Goal
Add a `GET /checkpoint.ots` route to `proofserve.Handler` that resolves the hub's accepted `(size, root)`,
reads the mirrored OTS proof via `store.OTSForRoot`, and writes `OTSBytes` verbatim so a client can fetch
it and run the standard `ots` toolchain against it. This is the canonical anchoring artifact every later
OTS surface (certificate §5, the dossier §4 Bitcoin-anchor panel) links to.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/proofserve/ots_test.go` (test, not counted) — HTTP-seam
  golden tests for the new route.
- **Modify** (≤3 non-test/doc source files):
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — add a `serveOTS` func + a
    `case "/checkpoint.ots":` to the `Handler` dispatch switch. (1 of ≤3)
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — add `mux.Handle("/checkpoint.ots", proofs)` to
    `hubHandler` so the exact mount beats the `/` subtree (mirroring the existing `/record`, `/inclusion`,
    … mounts). (2 of ≤3)
  - `/workspace/iscc-monitor/CLAUDE.md` — document the new `GET /<domain>/log/checkpoint.ots` endpoint in
    the "Running a local dev instance" route list (doc, not counted).
- **Reference** (read before writing):
  - `/workspace/iscc-monitor/.claude/context/learnings/otsclient.md` — read before touching ANY OTS code:
    `internal/ots`/`internal/otsclient` are NOT WASM-pure and pull `net/http`/`opentimestamps`; keep them
    out of any closure that must stay a leaf.
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the proofserve dispatch, the
    `CheckpointAt found==false` → 500 contract, the `serveVerify` inverted-status posture, the
    buffer-then-200 discipline, and the Mux-mount trap (exact route vs the `/` subtree).
  - `/workspace/iscc-monitor/internal/store/ots.go` — `OTSForRoot(ctx, hubID, treeSize, root) (OTSRecord,
    found bool, err error)`; an un-anchored root is a plain miss `(OTSRecord{}, false, nil)`, NOT an error.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` `serveVerify` (≈ lines 485-510) — the exact
    `FollowState` → `size := fs.LastSize` → `CheckpointAt(ctx, hubID, size)` → `root` resolution to copy.
  - `/workspace/iscc-monitor/internal/tilesserve/handler.go` `writeBlob` (lines 169-191) — the
    `application/octet-stream` + strong content-ETag + `If-None-Match` → 304 pattern to mirror.

## Not In Scope
- **Certificate §5 BITCOIN ANCHOR** (`HasClause5` in `internal/certificate/handler.go`) — a separate later
  sub-step that reads `OTSForRoot` + classifies via `ots.Confirmed`; do NOT touch the certificate handler.
- **The dossier / certificate Bitcoin-anchor vs comparison-anchor panels** — a later M-UI sub-step.
- **The `safeStamp` guard / nil-Stamper fix** (the open `normal`+`low` OTS issues) — this route is a pure
  store-read serve and does NOT touch the stamp path (`internal/otsclient`/`internal/follower`); fold
  `safeStamp` in when that path is next edited, not here.
- **Importing `internal/ots` or `internal/otsclient` into production proofserve** — serve `OTSBytes` as
  opaque bytes; do NOT parse/classify them in the route (keeps proofserve off the anchoring + non-WASM
  closure).
- A real Bitcoin-confirmed `.ots` (depends on a live calendar + chain confirmation; offline-unprovable —
  the criterion's "upgrades to Bitcoin-confirmed" half stays open after this step).

## Implementation Notes
- **Resolution order (copy `serveVerify`):** `fs, err := st.FollowState(ctx, hubID)` → on err 500;
  `size := fs.LastSize`; if `size == 0` → **404** "no accepted checkpoint" (coverage honesty — there is no
  root to anchor yet); `root, _, found, err := st.CheckpointAt(ctx, hubID, size)` → on err 500, on `!found`
  → 500 (a genuine store inconsistency at the accepted size, exactly as `serveVerify` treats it).
- **OTS lookup:** `rec, found, err := st.OTSForRoot(ctx, hubID, size, root)` → on err 500; on `!found` →
  **404** "root not yet anchored" (the honest pending state, NOT a 5xx — `OTSForRoot` already returns
  `(…, false, nil)` for a miss). Also treat `found && len(rec.OTSBytes) == 0` as 404: a row with the
  empty-OTSBytes sentinel (stamped-but-not-yet-calendar-submitted) has no servable proof yet — serving
  zero bytes would hand the client an unparseable `.ots`. **This empty-sentinel guard is the load-bearing
  edge case.**
- **Serve verbatim:** write `rec.OTSBytes` as `Content-Type: application/octet-stream`. Mirror
  `tilesserve.writeBlob`'s strong content-ETag (`fmt.Sprintf("\"%x\"", sha256.Sum256(data))`) +
  `If-None-Match` (`*` or exact) → 304 pattern; `Cache-Control: no-cache` (the served proof is overwritten
  in place on the pending→confirmed upgrade, so it is revalidating, never `immutable`). Do NOT export a
  helper from tilesserve — proofserve is a separate package; inline a tiny proofserve-local block to stay
  in budget. Method-not-GET → 405 is already handled by the `Handler` wrapper.
- **Stay opaque (learnings/otsclient.md):** do NOT import `internal/ots` or `internal/otsclient` into
  production proofserve — they pull `net/http`/`opentimestamps` and are NOT WASM-pure. `OTSBytes` is an
  opaque `[]byte` from the store leaf; serve it without parsing.
- **Mux mount (learnings/http-surface.md):** `/checkpoint.ots` MUST be an exact `mux.Handle` in
  `hubHandler` (like `/record`, `/inclusion`), else the `/` subtree dispatch sends it to `tilesserve`,
  which would 404 the unknown path. Note `tilesserve` serves the raw `/checkpoint` BLOB (the signed note);
  `.ots` is a DIFFERENT artifact (the timestamp proof) served by proofserve from the `ots` table — not a
  tilesserve mirror BLOB.
- **Correctness rule (learnings.md, always-loaded):** "OTS never blocks/crashes the follower" — this route
  is a pure read off the HTTP path, so it cannot block the follower; just keep it fail-closed
  (200-or-honest-status, never panic), like the sibling proof routes.
- **Oracle gate:** N/A — opaque-byte serve of an already-stored proof; no signature/RFC-6962/Merkle/
  did:web/proof code touched. The `ots verify` oracle still applies to the unchanged `internal/ots`
  classifier + its `testdata/*.ots` fixtures.

## Verification
- `mise run check` is green (build + vet + test, all packages; `gofmt -l .` excl. `cauldron/` empty).
- `go test -count=1 -run TestOTS ./internal/proofserve` passes (name the new tests `TestOTS…` so this
  filter catches them all — avoid the prior `-run TestOTS` under-selection issue).
- The test seeds a hub with an accepted checkpoint (`RecordCheckpoint` + `AdvanceAccepted`, or the
  fixture-store helper the other proofserve tests use) AND an `ots` row whose `OTSBytes` is a real fixture
  (read `internal/ots/testdata/hello-world.txt.ots` — note the `.txt.ots` suffix — and store those bytes
  via `RecordOTS`), then asserts `GET /checkpoint.ots` returns `200`, `Content-Type:
  application/octet-stream`, and a body **byte-equal** to the stored `OTSBytes`.
- Parse-validity assertion (the "verifies with the standard `ots` client" half, offline form): a test
  parses the served body with `opentimestamps.ReadFromFile(body)` and asserts no error + a non-nil
  `*opentimestamps.File`, proving the served bytes are a real `.ots`, not opaque garbage. Keep the
  `opentimestamps` import to a `_test.go` only (do NOT pull it into production proofserve); if a test-only
  import still trips the dep-closure check, put the parse assertion in an `internal/ots` test instead and
  keep proofserve's test asserting byte-equality.
- `GET /checkpoint.ots` for a hub with `LastSize == 0` → 404; for an accepted root with no `ots` row → 404;
  for a row with the empty-OTSBytes sentinel → 404 "not yet anchored" (never 5xx). All asserted in the test.
- `go list -deps ./internal/proofserve | grep -E 'iscc-monitor/internal/ots($|/)|iscc-monitor/internal/otsclient'`
  is empty (production proofserve stays off the anchoring/non-WASM closure).

## Done When
`mise run check` is green, `GET /<domain>/log/checkpoint.ots` returns the stored OTS proof bytes verbatim
(byte-equal, parseable as a valid `.ots`) for an anchored accepted root and an honest 404 for an
un-anchored / unpolled one, and production proofserve's dep closure still excludes `internal/ots` +
`internal/otsclient`.
