# Handoff

## 2026-06-20 — Store-side freeze + violations seam (RecordViolation, Freeze)

**Done:** Added the typed `internal/store` freeze-path persistence seam: a `Violation` struct, a
plain-INSERT `RecordViolation(ctx, Violation) (int64, error)` (persists `kind` + both raw
contradictory checkpoints + proof JSON, no dedupe — re-detection is itself evidence), and an
upsert-based `Freeze(ctx, hubID int64) error` that sets/keeps `frozen=1` (no auto-unfreeze,
ADR-0006). One source file changed, no callers wired (out of scope), all gates green.

**Files changed:**
- `internal/store/checkpoints.go`: added `Violation` struct + `RecordViolation` (plain INSERT into
  `violations`, returns new id via `LastInsertId`, `DetectedAt` mapped through `unixOrNil`,
  `ProofJSON` stored as a plain string) and `Freeze` (`INSERT INTO follow_state (hub_id, frozen)
  VALUES (?, 1) ON CONFLICT(hub_id) DO UPDATE SET frozen = 1`). No new imports — store stays a leaf.
- `internal/store/checkpoints_test.go`: added 7 tests — `TestRecordViolation` (round-trips
  `kind`/`raw_a`/`raw_b`/`proof_json`/`detected_at`, id > 0), `TestRecordViolationNoDedupe`
  (re-detection → distinct row, empty proof = empty string not NULL), `TestRecordViolationZeroDetectedAtNull`,
  `TestFreezeNoPriorRow`, `TestFreezeNoAutoUnfreeze` (Freeze→Advance keeps frozen=1, last_size=99),
  `TestFreezeOtherHubsUnaffected`, `TestFreezeRestartSurvival` (violation + freeze survive reopen).

**Verification:** `mise run check` → green (`go build ./... && go vet ./... && go test ./...` exit 0,
go1.24); `gofmt -l .` empty.
- `go test -count=1 -run 'TestRecordViolation|TestFreeze' ./internal/store` → PASS (all 7 subtests).
- RecordViolation round-trip (kind + both raw blobs + proof JSON + id > 0) → pass.
- Freeze no-prior-row (`FollowState.Frozen == true` after Freeze on a hub with no row) → pass.
- No-auto-unfreeze (`Freeze` then `AdvanceFollowState(…,99)` → `Frozen == true` AND `LastSize == 99`) → pass.
- Other-hubs-unaffected (freeze hub A → hub B `Frozen == false`) → pass.
- `go list -deps ./internal/store` → only `github.com/iscc/iscc-monitor/internal/store` (zero internal
  deps; store stays a leaf, no net/http in the closure).

**Next:** The three-trigger RFC-6962 consistency check (fork/shrink/equivocation) is now the natural
follow-on — it drives this seam. That step needs the `transparency-dev/merkle` dep (`go get`) plus
tiles fixtures, so it is heavier; the alert ("exactly one alert") mechanism and wiring
`RecordViolation`/`Freeze` into `follower.PollHub` are separate later slices. The independent
`hub_keys` did:web cache write (which must also refresh the stale `sb1.amlet.id_did.json` fixture and
`derive_vkey.py` HUBS to signer `069d0f14`) and the poll-loop / single-writer goroutine wrapper remain
available as parallel ≤3-file steps.

**Notes:**
- `Freeze` is now the only writer of `frozen`; `AdvanceFollowState` still omits it from its conflict
  update, so the two upserts compose correctly (advance-after-freeze keeps the freeze). Both target the
  same `follow_state` row via `ON CONFLICT(hub_id)`.
- `RecordViolation` is a plain INSERT by design — `violations` has no UNIQUE constraint, so repeated
  detection records distinct rows (verified by `TestRecordViolationNoDedupe`). The future consistency
  check is responsible for *deciding* when to record, not for dedupe here.
- `proof_json` is a TEXT column; an empty `ProofJSON` is stored as the empty string (not coerced to
  NULL), matching the work-package instruction and asserted in `TestRecordViolationNoDedupe`.
- No signature/consistency/proof code touched (pure persistence seam) → conformance/oracle gate N/A
  this step; `derive_vkey.py` vectors and WASM purity untouched.
- gofmt aligned the `var (…)` declaration block in `TestRecordViolation` (cosmetic, expected).
