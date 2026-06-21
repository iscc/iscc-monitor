## 2026-06-21 — Review of: Serve computed consistency proofs over HTTP from the local mirror

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `GET /consistency?from=<n>` to the per-hub proof surface (a path switch in
`proofserve.Handler` → `serveConsistency`, the `ConsistencyEvidence` response, `writeConsistency`) and
landed the deterministic `CheckpointAt … ORDER BY rowid` fix with its doc. The RFC-6962 consistency
oracle gate is genuinely non-vacuous — I independently reproduced two reverting mutations (corrupt the
served proof → fails byte-equal + `VerifyConsistency`; reverse the `rowid` order → fork-determinism test
fails). Scope is clean (exactly 3 production files, nothing from `## Not In Scope`), the store stays a
leaf, and the full suite + notecheck oracle are green uncached.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` pass; all 13 packages `ok` uncached
  (incl. `notecheck`, `follower`, `proofserve`, `store`).
- [x] `gofmt -l .` empty.
- [x] `go test -run TestConsistency -count=1 ./internal/proofserve` PASS — 8 named cases ran (served
  proof verifies for `from ∈ {1,200,255,256,299}` byte-equal to `tree.ConsistencyProof` + accepts under
  `proof.VerifyConsistency` + wrong-prior-root rejected; degenerate `from ∈ {0, LastSize}` → 200 empty;
  missing/non-numeric → 400; `from>LastSize` → 400; `LastSize==0` → 404; unknown `from` → 404; non-GET → 405).
- [x] `go test -run TestCheckpointAt -count=1 ./internal/store` PASS — `TestCheckpointAt` +
  `TestCheckpointAtDeterministicOnFork` (lowest-rowid prior accepted row returned on a fork).
- [x] `go test -run TestMirror -count=1 ./cmd/iscc-monitor` PASS — `TestMirrorRouter` +
  `TestMirrorInclusionRoute` (existing mirror/inclusion routing intact alongside the new `/consistency` mount).
- [x] `git diff --quiet HEAD~1 -- internal/store/schema.sql go.mod go.sum` exit 0 (no schema/dep change).
- [x] `go list -deps ./internal/store | grep -E 'proofserve|net/http'` empty (store stays a leaf).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` OK (WASM verifier seam untouched).
- [x] **Oracle / trust-root gate:** reviewer reproduced two reverting mutations — (1) corrupt
  `encoded[0]` in `writeConsistency` → `TestConsistencyServedProofVerifies` FAILS on both the byte-equal
  assert and `VerifyConsistency`; (2) `ORDER BY rowid` → `DESC` → `TestCheckpointAtDeterministicOnFork`
  FAILS. Both reverted, tree clean. External `cmd/notecheck` run locally: accepts the real sb0 checkpoint
  (`OK sb0.iscc.id/log`) and rejects a corrupted sig line. CI `notecheck` signature-parity job present in
  `.github/workflows/ci.yml` (gates on push to develop). `notecheck`/`derive_vkey.py` correctly N/A to
  this slice's payload — the consistency response carries no checkpoint and re-parses no signature.
- [x] Gate-integrity scan over `@{upstream}..HEAD`: no `//nolint`/`t.Skip`/swallowed-err/build-tag/
  deleted-test/loosened-gate additions.

**Issues found:** One follow-up (not a blocker, downgraded the existing `CheckpointAt` issue rather than
closing it): the store-level root cause is now fixed, but `internal/follower/follower_test.go:276-282`
`TestPollHubFork` still drives fork re-detection through `freeze` directly, with a now-**stale** comment
claiming `CheckpointAt`'s "unordered LIMIT 1 … non-deterministic". Rewiring it to re-detect via a second
`PollHub` and deleting that comment is a follower-test slice (out of scope for the store-query-only change
here). Retargeted in `issues.md`.

**Next:** The `entries` proof/read endpoint — the third remaining M2 proof surface (serve the raw entry
bundle / record bytes for a leaf range from the mirror), reusing this `/consistency` + `/inclusion`
routing pattern in `hubHandler`'s switch. Alternatively, the small `TestPollHubFork` re-detection cleanup
(drive re-detection through a second `PollHub` now that `CheckpointAt` is deterministic; remove the stale
comment) — a quick win that closes the retargeted issue.

**Notes:**
- `parseUint` (shared with `serveInclusion`, pre-existing, unchanged) has no overflow guard — a
  pathologically long digit string silently wraps `n`. Not introduced here and harmless for valid input
  (a wrapped `from` still hits the `from > size` 400 or a `CheckpointAt` 404). Not worth a fix or an issue.
- Response shape is intentionally a local `ConsistencyEvidence` (`{type, firstSize, secondSize,
  consistencyProof:[base64-Std]}`), NOT a hub-served `IsccLogConsistencyProof` (iscc-log §10.2: the
  consistency proof is verifier-computed). base64-Std matches the inclusion encoding and iscc_hub convention.
- M7 (gossip/cosigner) remains out of scope and never blocks DONE. No open `critical` issue; the
  remaining `normal` issues (frozen-hubs-still-advance, `AcceptCheckpoint` context reuse, tile-writer `p`
  vocabulary, deep `AdvanceAccepted`, `CheckConsistency` collapse, this `TestPollHubFork` cleanup) are
  for `define-next` to weigh against the state→target gap.
