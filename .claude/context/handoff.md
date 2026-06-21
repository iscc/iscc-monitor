## 2026-06-21 — HTML log browser at `GET /<domain>/log/` (mirrored checkpoint size + root)

**Done:** Added a server-rendered HTML log browser at each hub's `GET /<domain>/log/` root that exposes
the monitor's accepted checkpoint `(size, root)` plus relative links into the hub's `entries`/proof
routes. This closes M3's fourth and final Verify criterion (M3 → 4/4). The page reads only persisted
store rows (`FollowState` + `CheckpointAt`) and renders them verbatim — no proof/signature/Merkle path.

**Files changed:**
- `internal/proofserve/browser.html` (new): embedded `html/template` — accepted size + base64-Std root +
  store-provable status, a "No accepted checkpoint yet" coverage-honesty state, and a proof-surface link
  list (`entries`/`inclusion`/`consistency`/`verify`/`checkpoint`, relative URLs resolving under the
  trailing-slash subtree base).
- `internal/proofserve/handler.go`: added `bytes`/`_ "embed"`/`html/template` imports, a parsed-once
  package-scope `browserTmpl`, a `browserData` view-model, `serveBrowser` (FollowState → CheckpointAt →
  base64-Std root, render-into-buffer-then-200), and `case "/": serveBrowser(...)` before `default` in
  the path switch; updated the `Handler` doc comment.
- `cmd/iscc-monitor/main.go`: `hubHandler` now mounts a tiny dispatch `http.HandlerFunc` at `/` that
  sends the bare path `/` to `proofserve.Handler` (renders the browser) and delegates every deeper path
  to `tilesserve.Handler` — an `http.ServeMux` cannot hold both an exact `/` and a subtree `/`, so the
  func is the minimal fix. The four exact proof mounts are unchanged. Updated the doc comment.
- `internal/proofserve/browser_test.go` (new): golden HTTP-seam test over `buildMirror` (size 300) —
  `GET /` → 200 `text/html; charset=utf-8`, body contains the accepted size + base64-Std root + the four
  proof links; `POST /` → 405; followed-but-unpolled hub → 200 "No accepted checkpoint yet".
- `CLAUDE.md`: added the `GET /<domain>/log/` HTML log-browser line to the dev-instance endpoint list.

**Verification:** `mise run check` → green (all 16 packages `ok`; build + vet + test). Per criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/proofserve` PASS uncached (3 new browser tests + all existing
  inclusion/consistency/entries/verify tests).
- [x] `go test -count=1 ./cmd/iscc-monitor` PASS uncached (`TestMirrorRouter` + `TestMirrorInclusionRoute`
  + `TestMirrorEntriesRoute` — the `hubHandler` rewiring keeps existing mirror routing intact).
- [x] HTTP-seam (fixture size 300): `GET /` → 200, `text/html; charset=utf-8`, body contains `300` and
  `base64.StdEncoding(tree.Hash())`; `POST /` → 405; unpolled hub → 200 with the no-coverage state.
- [x] Mutation-proven non-vacuous (both reverted): dropping `{{.Root}}` AND dropping `{{.Size}}` each
  FAIL `TestBrowserExposesAcceptedCheckpoint`; reverted → green.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` empty — store stays a leaf.
- [x] `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty — byte-identical.
- [x] End-to-end dispatch confirmed via a throwaway test through the full `buildMux` (then removed):
  `GET /sb0.iscc.id/log/` → 200 text/html "Log Browser"; `GET /sb0.iscc.id/log/checkpoint` → 200
  octet-stream (still reaches tilesserve).

**Next:** M3 is now 4/4 Verify — the natural next arc is the **proof-bundle assembler**
(`{hub-signed checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`), the authoritative
client-verifies path that removes the monitor from the trust path. That is a different posture than the
verify-for-me/browser surfaces (the client verifies the bundle itself) and lands the in-browser verifier
/ CLI shared artifact. Update `state.md` to reflect M3 closed before defining it.

**Notes:**
- **Oracle/conformance gate is N/A here** (stated per next.md): this is pure HTML rendering of persisted
  store rows — no signature / RFC-6962 / Merkle / did:web / fsck / proof computation. The served
  `(size, root)` are read back verbatim (`FollowState.LastSize` + `CheckpointAt(size)`), never recomputed.
  `go.mod`/`go.sum`/`schema.sql` are byte-identical.
- **No new store method** — `serveBrowser` reuses `FollowState` + `CheckpointAt`, the same composition
  `serveVerify` uses; store stays a leaf (no `ListHubs`-style addition).
- **The page shows no hub domain/origin** — `proofserve.Handler` carries only `hubID`, not the origin,
  and next.md fixed the signature as `serveBrowser(w, r, st, hubID)` with "No new store method needed".
  The page is correctly addressed by its URL and the relative links resolve under it, so the identity is
  the URL itself. If a future step wants the domain in the page heading, it would need the origin threaded
  through `Handler` (a signature change) — out of scope here.
- **Status is the store-provable subset only** (`frozen` else `verified`, via the existing
  `proofserve.hubStatus`) exactly as next.md scoped — the richer in-memory statuses
  (`unverified`/`unresolvable`/`rotated`) are not threaded in.
- **`!found` at the accepted size → 500** mirrors `serveVerify`'s treatment of the same real store
  inconsistency; an unpolled hub (`LastSize == 0`) is a 200 no-coverage page, not a 404.
