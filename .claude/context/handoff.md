## 2026-06-21 — Review of: Conditional GET (strong ETag + If-None-Match → 304) on the tlog-tiles mirror

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a strong content ETag (`fmt.Sprintf("\"%x\"", sha256.Sum256(data))`, no
`W/` prefix) to every tlog-tiles 200 inside `writeBlob`, plus an `If-None-Match` short-circuit (`inm ==
"*" || inm == etag` → 304 with empty body). `writeBlob` now takes `r *http.Request`; the three callers
were updated. The change is exactly what `next.md` asked — one production file + its test, scope-clean,
all gates green, trust root untouched, oracle gate correctly N/A.

**Verification:**
- [x] `mise run check` — green (build + vet + test; all 15 packages `ok`).
- [x] `gofmt -l .` — empty (whole tree, not just the changed files).
- [x] `go test -count=1 -run TestHandler ./internal/tilesserve` — all 9 prior
  `TestHandlerServesSeededBytes` subtests + all 5 new `TestHandlerConditionalGet` subtests PASS
  (uncached, verbose). No regression to the byte-equality / Cache-Control subtests.
- [x] Full-tile 200 ETag is non-empty, quoted, strong (no `W/`) — subtest passes; reviewer
  re-derived `sha256(0x11*8192)` in Python (`"a44d83e2…"`) → matches Go's `%x` over `[32]byte`.
- [x] Re-GET with the exact ETag → 304, empty body, ETag echoed — load-bearing subtest passes.
- [x] `If-None-Match: "deadbeef"` (non-matching) → 200 with the full seeded body — passes.
- [x] Full tile vs entry bundle → distinct (content-derived) ETags — passes.
- [x] `*` wildcard → 304 — passes.
- [x] 304 also carries `Cache-Control` (RFC 7232 §4.1) — reviewer wrote a throwaway test (removed):
  the full-tile 304 echoes both the ETag and `public, max-age=31536000, immutable`.
- [x] `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 — byte-unchanged.
- [x] Gate integrity: scanned all 3 unpushed commits (`origin/develop..HEAD`); no
  `//nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion. The lone `_, _ = w.Write(data)` is
  pre-existing (not in the added-line diff), the documented post-status-write idiom (matches
  `metricshttp`/`healthz`), not a dodge.
- [x] Trust root unaffected: `internal/logclient` + `internal/didweb` conformance tests pass uncached;
  `derive_vkey.py` reproduces both golden vectors (`40b74463` / `22b08f3e`); didweb WASM leaf builds
  green. Oracle gate correctly N/A (no signature/RFC-6962/Merkle/did:web/fsck/proof path).
- [x] Scope: only `internal/tilesserve/handler.go` (1 prod file) + its `_test.go` + the handoff doc.
  Nothing from `## Not In Scope` touched (no `proofserve`/`corsmw`/`Last-Modified`/`http.ServeContent`/
  Range handling/issue-draining).

**Issues found:** (none) — clean, well-scoped, well-tested slice. No open `issues.md` entry was touched
(this slice deliberately drained none), so none could be resolved.

**Next:** The cache plumbing on the `tilesserve` mirror is now complete (Cache-Control + conditional
GET). The two strongest candidates for the next slice: (1) **drain a `normal` issue** — the ADR-0006
split-view / frozen-advance cluster is the highest-value gap (frozen-hub evidence-only short-circuit
that stops `AdvanceFollowState` advancing accepted state when `fs.Frozen`; `TestPollHubFork`
re-detection via a second `PollHub` instead of a direct `freeze`; collapsing the self-consistency
decision into a pure `logclient.CheckConsistency`). (2) Begin the **proof-bundle JSON + verify-for-me**
arc, or cache policy on `proofserve`'s size-dependent surfaces (`/inclusion`/`/consistency`/`/entries`,
tied to `LastSize`, needing a derived/weak validator). The frozen-advance fix is touching the
trust-relevant freeze path, so it is both higher value and a good time to re-arm the consistency goldens.

**Notes:**
- The whole M3 HTTP-plumbing arc (CORS → Cache-Control → conditional GET) is now coherent and done;
  shifting to the ADR-0006 evidence cluster means the next reviewer should expect the oracle/conformance
  gate to **re-arm** (freeze logic touches RFC-6962 consistency proofs).
- Two cache vocabularies remain a live trap for any future `tilesserve` touch: path-API full = `width 0`
  (what the immutable predicate keys on) vs. store-column full = `256`. This slice consumed only the
  already-correct `immutable bool` + the raw bytes, so it did not re-enter the trap.
- Pushed `develop` to origin on PASS (loop runs on `develop`; never `main`).
