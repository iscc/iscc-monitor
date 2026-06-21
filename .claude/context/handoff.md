## 2026-06-21 — Review of: HTML log browser at `GET /<domain>/log/` (mirrored checkpoint size + root)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a server-rendered HTML log browser at each hub's `GET /<domain>/log/` root
exposing the monitor's accepted checkpoint `(size, root)` plus relative links into the hub's
`entries`/proof routes. The diff is scope-clean (2 non-test source files + new template + new test +
doc), reads only persisted store rows (`FollowState` + `CheckpointAt`, root base64-Std verbatim — no
crypto), keeps `internal/store` a leaf, and is golden + mutation-proven at the HTTP seam. This closes
M3's fourth and final Verify criterion (M3 → 4/4).

**Verification:**
- [x] `mise run check` green — all 16 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (clean).
- [x] `go test -count=1 ./internal/proofserve` PASS uncached (3 new browser tests + all existing
  inclusion/consistency/entries/verify tests). The 3 named tests pass verbosely.
- [x] `go test -count=1 ./cmd/iscc-monitor` PASS uncached (`TestMirrorRouter` + `TestMirrorInclusionRoute`
  + `TestMirrorEntriesRoute` — the `hubHandler` mux rewiring keeps existing mirror routing intact).
- [x] HTTP-seam (fixture size 300): `GET /` → 200, `text/html; charset=utf-8`; body contains `300` and
  `base64.StdEncoding(tree.Hash())` + the four proof links; `POST /` → 405; unpolled hub → 200 with
  "No accepted checkpoint yet". (Reviewer re-ran the 3 tests independently.)
- [x] Mutation-proven non-vacuous (reviewer's OWN run, both reverted): dropping `{{.Root}}` AND
  replacing `{{.Size}}` with a constant each FAIL `TestBrowserExposesAcceptedCheckpoint`; reverted →
  green and tree byte-clean.
- [x] End-to-end dispatch (reviewer's OWN throwaway through the full `buildMux`, then removed):
  `GET /<domain>/log/` → 200 text/html "Log Browser"; `/<domain>/log/checkpoint` → 200 octet-stream
  (still reaches tilesserve); `POST /<domain>/log/` → 405 (shared method-gate); `/<domain>/log` (no
  slash) → 301 to trailing slash (correct subtree behavior).
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` empty — store a leaf.
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty — byte-identical.
- [x] Gate-integrity scan over all unpushed commits (`@{upstream}..HEAD`) — no `//nolint` / `t.Skip` /
  build-tag exclusion / deleted assertion. The two new `_ =` are legit: `_ = st.Close()` in a test
  cleanup and `_, _ = buf.WriteTo(w)` (the documented post-200 write-drop, used package-wide).
- [x] Oracle/conformance gate correctly N/A — pure HTML rendering of persisted store rows; the served
  `(size, root)` are read back verbatim (`FollowState.LastSize` + `CheckpointAt`), never recomputed; no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path.

**Issues found:** (none) — no defect. No issues.md change (the lone open `low` notecheck item is
unrelated and still valid). One non-blocking observation noted below.

**Codex second opinion:** Codex (gpt-5.5, xhigh reasoning) finished after ~5 min and explored the diff
extensively (ran its own mux probe + checkpoint-verify reads). Final verdict: "The change correctly adds
the per-hub HTML log browser while preserving existing proof and static mirror routing. Tests and vet
pass, and no actionable regressions were found." No findings to triage — clean second opinion, matches
my independent assessment.

**Next:** M3 is 4/4 Verify — `update-state` should mark M3 closed. The natural next arc is the
**proof-bundle assembler** (`{hub-signed checkpoint, inclusion/consistency proof, record bytes, hub key,
ots?}`), the authoritative client-verifies path that removes the monitor from the trust path — a
different posture than the verify-for-me/browser surfaces, and the shared artifact for the in-browser
verifier + CLI. The oracle gate (`notecheck`/Merkle) WILL apply there (a bundle re-asserts the hub
signature + proofs), unlike this pure-render slice.

**Notes:**
- **Non-blocking observation (not a defect, not filed):** for a followed-but-unpolled hub (`LastSize ==
  0`, not frozen) the no-checkpoint page reads "Status: verified" because `proofserve.hubStatus` only
  distinguishes `frozen` vs `verified` (the store-provable subset). This is identical to existing
  `serveVerify`/`hubStatus` behavior and is softened by the explicit "No accepted checkpoint yet —
  guarantees hold only from coverage start" sentence (ADR-0001 coverage honesty preserved). A future
  richer-status thread-through (the in-memory `unverified`/`unresolvable`/`rotated`) would refine it; out
  of scope here. Worth a glance if a later slice threads `metrics.Registry` into proofserve.
- **The page shows no hub domain/origin** — `proofserve.Handler` carries only `hubID`. next.md fixed the
  signature as `serveBrowser(w, r, st, hubID)`; the page is addressed by its URL and relative links
  resolve under it. Threading the origin into a page heading would be a `Handler` signature change — a
  later step if wanted.
- **Learnings:** collapsed two now-settled+stable seams (`/inclusion`, `/consistency` routing+crypto
  ceremony) to `settled:` one-liners preserving only their durable traps, and added one log-browser
  bullet (`/`-mount-dispatch-func + store-read render posture) to `learnings/http-surface.md`. Net file
  ~158 lines (slightly over the soft ~150 after a full new section; net-reduced via the two collapses).
  No promotion to the cross-cutting index (all package-local mux/render mechanics).
