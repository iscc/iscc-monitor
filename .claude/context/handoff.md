## 2026-06-22 — Serve the mirrored OTS proof at `GET /<domain>/log/checkpoint.ots`

**Done:** Added a `GET /checkpoint.ots` route to `proofserve.Handler` that resolves the hub's accepted
`(size, root)` (the `serveVerify` `FollowState`→`CheckpointAt` flow), reads the mirrored OTS proof via
`store.OTSForRoot`, and serves `OTSBytes` verbatim as `application/octet-stream` with a strong content
ETag + `If-None-Match`→304 + `Cache-Control: no-cache`. Un-anchored roots (no row OR the empty-OTSBytes
sentinel) and `LastSize==0` are honest 404s; the proof bytes are served opaquely so production proofserve
stays off the non-WASM `internal/ots`/`internal/otsclient` closure. This closes the observable
HTTP-surface half of the OTS milestone Verify criterion — a client can now fetch the `.ots` and run the
standard `ots` toolchain against it.

**Files changed:**
- `internal/proofserve/handler.go`: added `serveOTS` + `writeOTS` (the conditional-GET block mirroring
  `tilesserve.writeBlob`, inlined since proofserve is a separate package) and a `case "/checkpoint.ots":`
  to the dispatch switch; added `crypto/sha256` + `fmt` imports; doc-comment updates.
- `cmd/iscc-monitor/main.go`: added `mux.Handle("/checkpoint.ots", proofs)` to `hubHandler` (exact mount
  beats the `/` subtree, the Mux-mount trap) + a doc note distinguishing `.ots` (timestamp proof, from the
  `ots` table) from the raw `/checkpoint` signed-note BLOB tilesserve serves.
- `internal/proofserve/ots_test.go` (new, test): HTTP-seam golden tests — verbatim byte-equal serve +
  real `.ots` parse-validity (`opentimestamps.ReadFromFile`, test-only import), conditional-GET 304, and
  the three honest-404 cases (`LastSize==0`, no row, empty sentinel) + non-GET 405.
- `cmd/iscc-monitor/main_test.go` (test): `TestMirrorOTSRoute` — binary-level routing proof (exact mount
  beats `/` subtree, byte-equal proof; raw `/checkpoint` still reaches the static mirror).
- `CLAUDE.md` (doc): documented `GET /<domain>/log/checkpoint.ots` in the route list.

**Verification:** `mise run check` → green (build + vet + test, all 23 packages). `gofmt -l .` (excl
`cauldron/`) → clean. go.mod/go.sum byte-identical.
- `go test -count=1 -run TestOTS ./internal/proofserve` → passes; all 6 new tests match the `TestOTS`
  prefix (verified with `-v`).
- 200 + `application/octet-stream` + body byte-equal to stored `OTSBytes` + parses as valid `.ots` → PASS
  (`TestOTSServesStoredProofVerbatim`).
- `LastSize==0` → 404, accepted root with no `ots` row → 404, empty-OTSBytes sentinel → 404 (never 5xx) →
  PASS (`TestOTSNoAcceptedCheckpoint`/`TestOTSRootNotAnchored`/`TestOTSEmptySentinelNotAnchored`).
- `go list -deps ./internal/proofserve | grep -E 'internal/ots($|/)|internal/otsclient'` → empty (PASS).
- Mutation-proven (all reverted byte-identical): dropping `len(rec.OTSBytes)==0` → sentinel test FAILS;
  serving a truncated proof → verbatim test FAILS; removing the `mux.Handle("/checkpoint.ots", …)` line →
  `TestMirrorOTSRoute` FAILS (404 fall-through).

**Next:** The OTS milestone Verify criterion now has its observable `.ots` surface. Remaining closers:
(1) **Certificate §5 BITCOIN ANCHOR** (`HasClause5` in `internal/certificate/handler.go`) — reads
`OTSForRoot` + classifies via `ots.Confirmed`, linking to this `.ots` route; (2) the open `normal`+`low`
OTS issues (`safeStamp` guard + nil-Stamper guard-order fix in `internal/otsclient`/`internal/follower`)
before the stamp path is exercised against a real calendar. The "upgrades to Bitcoin-confirmed" half of
the criterion stays open offline (needs a live calendar + chain confirmation).

**Notes:**
- Scope: 2 source files touched (handler.go + main.go), within the ≤3 budget. Tests (ots_test.go,
  main_test.go) and CLAUDE.md not counted.
- Added `TestMirrorOTSRoute` to `cmd/iscc-monitor/main_test.go` (a test file, not counted) following the
  established per-mount precedent (`TestMirrorEntriesRoute`/`TestMirrorInclusionRoute`): the Mux-mount
  trap is the load-bearing integration risk for this step, and it is mutation-proven non-vacuous.
- The `opentimestamps` parse assertion is in `ots_test.go` (test-only import). next.md's dep-closure
  check is the non-test `go list -deps` form, which is unaffected by `_test.go` imports — verified empty.
  No need to relocate the parse assertion to an `internal/ots` test.
- `writeOTS` uses `Cache-Control: no-cache` (revalidating, not `immutable`): the served proof is
  overwritten in place on the pending→confirmed upgrade, exactly as next.md specifies.
- Oracle gate correctly N/A: opaque-byte serve of an already-stored proof; no signature/RFC-6962/Merkle/
  did:web/proof code touched. The `ots verify` oracle still applies to the unchanged `internal/ots`
  classifier + its `testdata/*.ots` fixtures.
- This route is a pure HTTP-path store read; it does not touch the stamp path
  (`internal/otsclient`/`internal/follower`), so the open `safeStamp`/nil-Stamper issues are untouched
  (correctly out of scope per next.md).
