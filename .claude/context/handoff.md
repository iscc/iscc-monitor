## 2026-06-22 — OTS daily stamp pass: record each distinct accepted root through `RecordOTS` in PollHub

**Done:** Wired the first production caller of the `store.RecordOTS` seam into `PollHub`: on the
verified, non-violation advance path (after `fsckMirror`, before the final `recordVerdict`) a tiny
`stampRoot` helper writes the newly-accepted checkpoint root as a `pending` `ots` row. The write is a
local SQLite insert with no calendar/Bitcoin I/O (never blocks the follower), and `RecordOTS`'s
`UNIQUE(hub, tree_size, root)` dedupe makes a re-poll of the same root a silent no-op.

**Files changed:**
- `internal/follower/follower.go`: added the `stampRoot(ctx, st, hubID, info, observedAt) error`
  helper (modeled on `fsckMirror`) building `store.OTSRecord{HubID, TreeSize, Root: info.Root[:],
  Status: store.OTSStatusPending, StampedAt: observedAt}`; called it on the verified non-violation
  path between `fsckMirror` and `recordVerdict`, wrapping a non-nil error as `follower.PollHub: hub %d:
  %w` (the `stamp root: %w` sub-wrap matches the sibling call sites). Updated the package + frozen-clean
  short-circuit doc comments to name the stamp.
- `internal/follower/follower_test.go`: extended `TestPollHubVerifiedAdvances` to assert exactly one
  `ots` row after the first poll, still one after a second verified poll (dedupe), and that the row is
  `pending` with empty `OTSBytes` via `s.OTSForRoot(ctx, hubID, m.size, m.tree.Hash())`. Added
  zero-`ots`-row assertions to the fork, shrink, unverified, and frozen-clean (`preOTS`==`postOTS`)
  paths.

**Verification:** `mise run check` → green (all 21 packages `ok`). Per-criterion:
- [x] `mise run check` green (`go build`/`go vet`/`go test`); `gofmt -l .` (excl. `cauldron/`) clean.
- [x] `go test -count=1 -run TestPollHub ./internal/follower` passes (verified-advance, fork, shrink,
  unverified, frozen-clean, equivocation all green).
- [x] Verified poll over `buildVerifiedMirror` writes exactly one `ots` row; still one after a second
  verified poll at the same root (dedupe).
- [x] Recorded row is `pending`: `OTSForRoot` returns `found==true`, `Status==OTSStatusPending`, empty
  `OTSBytes`.
- [x] Non-verified, fork, shrink, frozen-clean paths write zero `ots` rows.
- [x] `git diff --stat`: only `internal/follower/follower.go` + `follower_test.go`; `go.mod`/`go.sum`/
  `internal/store/*` byte-unchanged (`git diff --name-only -- internal/store go.mod go.sum` empty).
- [x] Mutation-proven non-vacuous (reverted, tree restored): neutering the `stampRoot` call to a no-op
  FAILS `TestPollHubVerifiedAdvances` (`ots rows = 0, want 1` + `OTSForRoot found = false`).
- [x] Store stays a leaf (`go list -deps internal/store | grep '^net/http'` empty); follower production
  imports unchanged (`store`/`time` already present, no new import).

**Next:** The background **upgrade loop** — read `store.PendingOTS`, stamp via the OpenTimestamps
calendar HTTP, and flip rows to confirmed via `MarkOTSUpgraded` once Bitcoin-confirmed. This is the
first step to pull in `nbd-wtf/opentimestamps` + calendar HTTP (a real `go.mod`/`go.sum` change) and
where the `Attempts`/`NextRetry` retry-policy columns finally get exercised. It runs in its own
goroutine off the poll path (OTS never blocks the follower). After that: the `.ots` HTTP route and
certificate §5 BITCOIN ANCHOR (`HasClause5`), both reading `OTSForRoot`.

**Notes:**
- Oracle/conformance gate correctly N/A for this slice: it records an opaque `pending` row over an
  already-fsck-verified accepted root — no signature/RFC-6962/Merkle/did:web/proof path added or
  changed. The verified-advance fsck/inclusion conformance tests re-ran (under `mise run check`) and
  stayed green.
- `stampRoot` discards `RecordOTS`'s `(id, inserted)` return (the dedupe is silent by design) and keeps
  only the error — not a swallowed error, the `id`/`inserted` are genuinely unused on the stamp path.
- A stamp fault surfaces to the caller (NOT a freeze); accepted state is already committed by
  `AdvanceAccepted`, so the next poll re-stamps via the idempotent dedupe — same error-vs-violation
  discipline as `ingestTiles`/`fsckMirror`.
- The `next.md` `-run TestPollHub` filter caveat the prior review flagged for the OTS area does not
  apply here: every new/extended test is named `TestPollHub*`, so the filter catches them all.
- Open `normal` issues (ForceQuery fail-open, `host:port` DID, §6 timestamp) untouched per Not-In-Scope.
