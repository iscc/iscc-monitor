## 2026-06-22 — Review of: Serve the mirrored OTS proof at `GET /<domain>/log/checkpoint.ots`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance adds a `GET /checkpoint.ots` route to `proofserve.Handler` that resolves the
hub's accepted `(size, root)` via the `serveVerify` flow, reads the mirrored OTS proof through
`store.OTSForRoot`, and serves `OTSBytes` verbatim as `application/octet-stream` with a strong content
ETag + `If-None-Match`→304 + `Cache-Control: no-cache`. It is correct, scope-disciplined (2 source
files), mutation-proven non-vacuous, and dep-clean (production proofserve stays off the non-WASM
`internal/ots`/`internal/otsclient` closure). This closes the observable HTTP-surface half of the OTS
milestone Verify criterion.

**Verification:**
- [x] `mise run check` green (build + vet + test, all 23 packages) — confirmed
- [x] `gofmt -l .` (excl `cauldron/`) clean — confirmed (incl. my doc-fix edit)
- [x] `go test -count=1 -run TestOTS ./internal/proofserve` — all 6 new tests match the prefix and PASS
  (`TestOTSServesStoredProofVerbatim`, `TestOTSConditionalGET`, `TestOTSNoAcceptedCheckpoint`,
  `TestOTSRootNotAnchored`, `TestOTSEmptySentinelNotAnchored`, `TestOTSNonGET`)
- [x] `TestMirrorOTSRoute` (binary-level Mux-mount proof) PASS — both subtests
- [x] 200 + `application/octet-stream` + body byte-equal to stored `OTSBytes` + parses as valid `.ots`
  (`opentimestamps.ReadFromFile`, test-only import) — PASS
- [x] `LastSize==0`→404, no `ots` row→404, empty-OTSBytes sentinel→404 (never 5xx) — PASS
- [x] `go list -deps ./internal/proofserve | grep -E 'internal/ots($|/)|internal/otsclient'` empty — PASS
- [x] go.mod/go.sum byte-identical (not in diff) — confirmed
- [x] Oracle gate: correctly N/A — opaque-byte serve of an already-stored proof; no
  signature/RFC-6962/Merkle/did:web/proof code touched
- [x] Mutation-proven (4 mutations, all reverted byte-identical, working tree clean):
  - removed `mux.Handle("/checkpoint.ots", …)` → `TestMirrorOTSRoute` FAILS (404 fall-through)
  - dropped the `len(rec.OTSBytes)==0` sentinel guard → `TestOTSEmptySentinelNotAnchored` FAILS
  - truncated the served proof → `TestOTSServesStoredProofVerbatim` FAILS
  - un-anchored 404 → 500 → `TestOTSRootNotAnchored`/`TestOTSEmptySentinelNotAnchored` FAIL
- [x] Quality-gate integrity: no `//nolint`/`t.Skip`/build-tag exclusion/deleted assertion in the
  unpushed range; the only swallowed error is the documented post-200 `_, _ = w.Write(data)` write-drop
  (matches `writeRecord`/`tilesserve.writeBlob`)
- [x] Scope: 2 source files (handler.go + main.go) within ≤3 budget; nothing in Not-In-Scope touched
  (certificate / otsclient / follower / registry all untouched)

**Issues found:** (none) — one minor doc-fix applied directly (step 9): the shared `Handler` docstring
said "caching, and conditional GET are intentionally out of scope for this slice", but `serveOTS`/
`writeOTS` deliberately add conditional-GET + caching for the `.ots` opaque BLOB. Updated the docstring
to record the `/checkpoint.ots` exception (behavior-neutral). Re-built + re-tested green.

**Codex second opinion:** Clean. Verdict: "The new /checkpoint.ots route is mounted correctly, resolves
the accepted root, serves stored OTS bytes with appropriate headers, and the added tests cover routing,
success, conditional GET, and expected 404/405 cases. No blocking correctness issues were found." No
findings to triage; matches my independent assessment.

**Visual check:** n/a — no SSR surface changed. The diff touches only `internal/proofserve/handler.go`
(opaque `application/octet-stream` serve, no template) + `cmd/iscc-monitor/main.go` (mount wiring) +
docs; no `internal/{dashboard,dossier,web,certificate}` or `.html`/`.css` file touched.

**Next:** Remaining OTS milestone closers, in priority order:
1. **Certificate §5 BITCOIN ANCHOR** (`HasClause5` in `internal/certificate/handler.go`) — reads
   `OTSForRoot` + classifies via `ots.Confirmed`, linking to this new `.ots` route. This is the next
   observable surface for the anchoring milestone now that the `.ots` bytes are servable.
2. **`safeStamp` guard + nil-Stamper guard-order fix** (open `normal`+`low` OTS issues) — fold in when
   the stamp path (`internal/otsclient`/`internal/follower`) is next edited, BEFORE it runs against a
   live calendar (panic→crash + stall→hang on a live calendar response).
The "upgrades to Bitcoin-confirmed" half of the criterion stays open offline (needs a live calendar +
chain confirmation; offline-unprovable).

**Notes:**
- `serveOTS` correctly takes `st` directly and ignores the `f` StatusSource overlay — the proof routes
  don't need the live verdict, matching the other proof routes.
- `writeOTS` uses `Cache-Control: no-cache` (revalidating, NOT `immutable`): the served proof is
  overwritten in place on the pending→confirmed upgrade, so a client must revalidate. Correct per next.md.
- The `opentimestamps` parse assertion lives in `ots_test.go` (test-only import) — the production
  dep-closure check (`go list -deps`, non-test form) is unaffected and verified empty.
- Open issues unchanged this iteration (all out of scope here): the two OTS stamp-path guards, the
  `hubDomain` ForceQuery fail-open, the §4/bundle `did:web:host:port` mis-render, the §6 timestamp gap,
  and four `low` items. None resolved, none newly stale.
- Learnings: added a 2-bullet `.ots` route section to `learnings/http-surface.md` (the empty-OTSBytes
  sentinel is the load-bearing edge case; `no-cache` not `immutable`; exact Mux mount); net-reduced the
  CORS + `/records` + `/record` settled blocks to stay within the rotation budget (162 lines, 21 bullets).
