# Handoff

## 2026-06-20 — Wire CheckShrink + CheckFork into follower.PollHub (freeze + alert-once)

**Done:** Composed the two landed pure verdicts (`CheckShrink`/`CheckFork`) into M1's ADR-0006 freeze
behavior inside `follower.PollHub`: on a `StatusVerified` observation that contradicts the prior
accepted checkpoint for the same hub (strict size decrease = shrink, equal size + different root =
fork), `PollHub` now records the violation, persists the contradictory checkpoint as evidence,
freezes the hub, and fires an injected alert exactly once on the not-frozen → frozen transition —
without advancing the cursor. Added the one supporting store read method (`CheckpointAt`) the fork
check + violation evidence need.

**Files changed:**
- `internal/store/checkpoints.go`: added `CheckpointAt(ctx, hubID, treeSize) (root, raw []byte, found
  bool, err error)` — `SELECT root, raw FROM checkpoints WHERE hub_id=? AND tree_size=? LIMIT 1`;
  absent row → `found=false`, nil error (mirrors `FollowState`). Returns `[]byte`, never a logclient
  type, so `store` stays a leaf.
- `internal/follower/follower.go`: added `AlertFunc` func seam; `PollHub` signature gained an `alert
  AlertFunc` param; on `StatusVerified` it now reads `FollowState`, runs `checkConsistency` (shrink
  first, then fork — guarded by the `prevSize>0`/`prevFound` rules), and on a true verdict calls
  `freeze` (RecordViolation + RecordCheckpoint-as-evidence + Freeze + alert-once on `!wasFrozen`)
  instead of advancing. A violation returns `(StatusVerified, nil)` — freezes, never crashes. Two
  helper funcs (`checkConsistency`, `freeze`) keep `PollHub` short; package/func docs updated so only
  equivocation/poll-loop/key-cache remain "later steps".
- `internal/follower/follower_test.go`: updated the two existing `PollHub` calls for the new `alert`
  param (added a `Frozen==false` assertion to the clean-advance test); added `TestPollHubFork` and
  `TestPollHubShrink` driving the freeze paths through the outbound-fetch seam against the real sb0
  fixture, with a seeded prior checkpoint at the contradicting size/root. Assertions read the
  `violations`/`follow_state` rows over an independent read-only `sqlite` connection (no follower
  internals).
- `internal/store/checkpoints_test.go`: added `TestCheckpointAt` (round-trip + absent-size +
  absent-hub, all nil-error).

**Verification:** `mise run check` → green (`go build ./...`, `go vet ./...`, `go test ./...` all
`ok`: didweb/follower/logclient/store); `gofmt -l .` empty; `git status --short go.mod go.sum` empty
(no dep added).
- [x] `go test -count=1 -run TestPollHub ./internal/follower` — PASS (existing verified-advances +
  unverified-no-advance still green after the signature change).
- [x] `TestPollHubFork` — second verified observation at the same `tree_size` (10183) with a different
  root → `violations.kind == "fork"`, `frozen == 1`, cursor stays at 10183 (no advance), alert fired
  exactly once.
- [x] Re-detection: a third poll of the already-frozen hub records a 2nd `violations` row but alert
  count stays 1 (exactly-one-alert across re-detection).
- [x] Other hubs unaffected: a 2nd registered hub polled with the clean verified checkpoint advances
  to 10183 and stays `frozen == 0`.
- [x] Restart: after reopening the store from the same path, `FollowState(...).Frozen == true` and
  both violation rows survive.
- [x] `TestPollHubShrink` — verified observation at 10183 < prior accepted 20000 → `violations.kind
  == "shrink"`, `frozen == 1`, cursor stays at 20000, exactly one alert.
- [x] `TestCheckpointAt` — round-trips `(root, raw)`; absent `(hubID, treeSize)` → `found == false`,
  nil error.

**Next:** The merkle-backed **equivocation** trigger (RFC-6962 consistency-proof failure across
*growing* sizes) is the last M1 self-consistency trigger — its own slice, since it needs
`transparency-dev/merkle` + tile fixtures and trips the conformance/oracle gate (`fsck` root-rebuild,
inclusion cross-check vs the hub's `IsccLogInclusionProof`). It plugs into the same `checkConsistency`
seam (a third branch returning `ViolationEquivocation` + a real `ProofJSON`). Alternatively, the
**poll-loop / single-writer goroutine wrapper** (the backed-off evidence-only re-poll cadence of a
frozen hub) is the other unblocked next slice.

**Notes:**
- **Alert seam is a `func(hubID int64, kind string)` field/param** (`AlertFunc`), not an interface
  (YAGNI per `next.md`). It only signals the not-frozen→frozen transition; real delivery
  (email/webhook/log) and frozen-hub re-poll cadence stay out of scope.
- **`rootBytes` is package-private to `logclient`**, so the follower copies the stored `[]byte` root
  into a plain `[32]byte` (`copy(prevRoot[:], prevRootBytes)`) before calling `CheckFork`. `[32]byte`
  is the concrete type of both `CheckpointInfo.Root` and `CheckFork`'s `[rootBytes]byte` params, so
  this compiles cleanly without exposing the constant.
- **Test assertions read rows over an independent `sql.Open("sqlite", path)` connection** rather than
  store internals or test-only exported store helpers. `export_test.go` would not have been visible
  across the package-boundary (follower test is package `follower`, store test is package `store`), so
  a separate read-only connection on the same WAL file is the clean observable-output seam. This adds
  the `modernc.org/sqlite` blank import to the follower *test* only — `go.mod`/`go.sum` unchanged
  (already a dep), and the production follower package's import graph is untouched (still
  follower → {logclient, store}, no `net/http`/`sqlite` in the follower's own imports).
- **Fork test non-vacuousness:** the seeded prior root (`"fork-seed-root-distinct-padding32"`) is
  pinned-distinct from the real sb0 fixture root (base64 `uir3z5T1…`, asserted via the
  `sb0FixtureRootB64` const guard), and the test asserts kind `"fork"` (not `"shrink"`) at equal size
  10183 — so the fork branch genuinely fired, not the size-only shrink path.
- **`fs.LastSize == 0` fresh-store guard:** `checkConsistency` early-returns on `prevSize == 0` and
  both `CheckShrink`/`CheckFork` carry the `prev>0` guard, so a never-advanced hub (and the clean
  first-observation tests) never trip a violation.
- **Conformance/oracle gate: N/A for this slice** (composes pure size/root verdicts + store CRUD;
  no signature, RFC-6962 proof, didweb, or merkle code touched). The didweb + logclient golden suites
  still pass (no regression). The equivocation slice will trip that gate.
