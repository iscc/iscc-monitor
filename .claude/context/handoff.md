## 2026-06-21 — Review of: verify-for-me JSON verdict — `GET /<domain>/log/verify?iscc_id=<id>`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a `/verify` route to the existing per-hub `proofserve.Handler` returning a
single self-contained JSON verdict (`VerifyVerdict`): the store-provable hub status, the accepted
checkpoint `(size, root)`, and a REAL RFC-6962 inclusion result recomputed from the local mirror and
Merkle-verified against the accepted root. The diff is scope-clean (2 production `.go` files +
`CLAUDE.md` doc + 1 new test file), composes the M2 proof machinery without re-hitting the hub, and the
inclusion check is reviewer-mutation-proven non-vacuous through the HTTP seam. Closes the nearest M3
Verify criterion (now 2/4).

**Verification:**
- [x] `mise run check` — green, all 15 packages `ok` (build + vet + test).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -count=1 -run TestVerify ./internal/proofserve` — PASS (golden across the 256-leaf
  boundary {0,5,255,256,260,299}; unknown-id 200; missing-id 200; non-vacuous corrupted-root negative;
  frozen hub_status; no-accepted-checkpoint; leaf-hash-matches-tree; non-GET 405).
- [x] `go test -count=1 ./internal/proofserve ./cmd/iscc-monitor` — PASS uncached (route mount + handler).
- [x] `go list -deps ./internal/store | grep -E 'proofserve|net/http'` — empty (store stays a leaf).
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` — empty (byte-unchanged).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM verifier seam untouched).
- [x] Known-id verdict (leaf 5): HTTP 200, `verified==true`, `included==true`, `tree_size==300`,
  `leaf_index==5`, served `root` base64-decodes to `tree.HashAt(size) == tree.Hash()`. Unknown id: 200,
  `verified==false`, non-empty `reason`, never 5xx.
- [x] **Oracle gate APPLIES (RFC-6962 inclusion crypto) — independently re-proven non-vacuous.** I
  reverted-mutated `serveVerify` to ignore the Merkle result (`_ = proof.VerifyInclusion(...); included
  := true`) → `TestVerifyInclusionIsNonVacuous` FAILS all three asserts against the corrupted accepted
  root; reverted → green. A green-but-wrong handler cannot ship.
- [x] Gate-integrity scan over all unpushed commits — no `//nolint`/`t.Skip`/build-tag exclusion/
  swallowed error/deleted assertion. The `_ = encodeJSON(w, v)` in `writeVerdict` is the documented
  package post-200 write-drop idiom (shared with `writeEvidence`/`writeConsistency`/`writeRecord`), not
  a new dodge.
- [x] Scope discipline — exactly 2 production `.go` files (`handler.go`, `main.go`) + `CLAUDE.md` doc +
  1 new `_test.go`. Nothing from `## Not In Scope` was done (no HTML dashboard, no bundle assembler, no
  status column, no metrics threading into proofserve, no CORS/caching on `/verify`).

**Issues found:** (none blocking). One doc-only nit fixed directly: the `serveVerify` comment claimed it
defaults to the first seq "via selectSeq" but the code uses `seqs[0]` directly (correct — verify takes
no `index` param, and the empty-seqs case is guarded before any index access). Reworded the comment; no
behavior change. `included` and `verified` are always equal in this handler, which is correct given the
logic (a separate `Included` field is forward-useful for the later bundle path); not flagged.

**Codex second opinion:** unavailable — the launch command requires
`--dangerously-bypass-approvals-and-sandbox`, which the environment's auto-mode classifier denied (no
output file produced). Recorded as a note per graceful-degradation; not a blocker.

**Next:** The remaining two M3 Verify criteria — the HTML dashboard (`GET /`) and the HTML log browser
(`GET /<domain>/log/`) — are the next sub-steps in this same arc (M3 → 3/4, 4/4). After those, the
authoritative path: a proof-bundle assembler packaging `{checkpoint, inclusion proof, record bytes, hub
key, ots?}` into one downloadable client-verifiable artifact. `serveVerify` already composes every
piece except the resolved hub key (`hub_keys`) and the raw checkpoint bytes (`CheckpointAt` returns
`raw` too, currently discarded with `_`).

**Notes:**
- `hub_status` is the store-provable subset only (`frozen` via `FollowState.Frozen`, else `verified`
  once `LastSize > 0`, ADR-0006). The richer unverified/unresolvable/rotated statuses live in the
  in-memory metrics registry and are deliberately not visible to this route — a documented limitation,
  not a defect, and explicitly out of scope per `next.md`.
- Verify-for-me INVERTS the other proof routes' status mapping: an id-shaped fault is always a 200
  verdict; non-200 is reserved for genuine infra faults only. Recorded this in
  `learnings/http-surface.md` for the future bundle-path author (the stronger client-verifies path has a
  different posture).
- Rotated `learnings/http-surface.md`: it had crossed its ~150-line / ~40-bullet budget (184 lines), so
  I collapsed the fully-landed tilesserve static-mirror detail into a one-line `settled:` summary and
  added the verify-for-me findings — net down to 153 lines / 16 bullets.
- `issues.md` unchanged: the only open entry (notecheck vestigial `out io.Writer`, `low`) is unrelated
  to this slice and skipped by the loop; no new issues filed (no real defects found).
