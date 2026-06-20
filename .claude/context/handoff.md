# Handoff

## 2026-06-20 — Review of: Store-side freeze + violations seam (RecordViolation, Freeze)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance added the leaf-only freeze-path persistence seam exactly as `next.md` asked:
a `Violation` struct, a plain-INSERT `RecordViolation(ctx, Violation) (int64, error)`, and an
upsert-based `Freeze(ctx, hubID int64) error`. One non-test/doc source file changed
(`internal/store/checkpoints.go`), one test file, no callers wired, `go.mod` untouched. All gates
green; the no-auto-unfreeze and other-hubs-unaffected properties are pinned with non-vacuous tests.

**Verification:**
- [x] `mise run check` green — `go build ./... && go vet ./... && go test ./...` exit 0 (go1.24).
- [x] `gofmt -l .` empty — no formatting failures.
- [x] `go test -count=1 -run 'TestRecordViolation|TestFreeze' ./internal/store` — PASS (all 7 subtests:
  RecordViolation, NoDedupe, ZeroDetectedAtNull, FreezeNoPriorRow, NoAutoUnfreeze, OtherHubsUnaffected,
  RestartSurvival).
- [x] RecordViolation round-trip — raw `SELECT kind, raw_a, raw_b, proof_json, detected_at` matches the
  inserted `equivocation` kind, both raw blobs, proof JSON, and `id > 0`. Verified.
- [x] Freeze no-prior-row — `FollowState.Frozen == true` after `Freeze` on a hub with no row. Verified.
- [x] No-auto-unfreeze — `Freeze` then `AdvanceFollowState(…,99)` → `Frozen == true` AND `LastSize == 99`
  (advance moved the cursor but did not clear the freeze). Non-vacuous; verified.
- [x] Other-hubs-unaffected — freeze hub A → hub B `Frozen == false`. Verified.
- [x] `go list -deps ./internal/store` — zero internal iscc-monitor deps (only the self-line); store
  stays a leaf. `net/http` absent from the closure (the bare `net`/`net/url`/`net/netip` come from
  `modernc.org/sqlite`, not iscc-monitor code).
- [x] Quality-gate integrity — scanned all 3 unpushed commits: no `//nolint`, `t.Skip`, build tags,
  swallowed errors, or deleted assertions in code. (grep hits are markdown prose noting their absence.)
- [x] Scope discipline — diff touches only the two store files + handoff; nothing from `## Not In Scope`
  (no merkle dep / `go get`, no caller wiring, no `Unfreeze`, no `hub_keys` write).
- [n/a] Conformance/oracle gate — no signature / RFC-6962 / proof / didweb / fork-shrink-equivocation
  code touched (pure persistence seam), so `notecheck` / `derive_vkey.py` / `fsck` parity is N/A.

**Issues found:** (none)

**Next:** The three-trigger RFC-6962 consistency check (fork/shrink/equivocation) is the natural
follow-on — it now has a tested seam (`RecordViolation` + `Freeze`) to drive. That step is heavier: it
needs the `transparency-dev/merkle` dep (`go get`, with the v1.24-compatible pin caveat) plus tiles
fixtures, so consider scoping it as detection-only over fixtures first, then a separate slice for
wiring into `follower.PollHub` and the "exactly one alert" mechanism. The independent `hub_keys`
did:web cache write (also refreshing the stale `sb1.amlet.id_did.json` fixture + `derive_vkey.py` HUBS
to signer `069d0f14`) and the poll-loop / single-writer goroutine wrapper remain available as parallel
≤3-file steps.

**Notes:**
- `Freeze` is now the sole writer of `frozen`; `AdvanceFollowState` omits it from its `DO UPDATE`, so
  the two `ON CONFLICT(hub_id)` upserts compose correctly (advance-after-freeze keeps the freeze). The
  next step must not introduce any other writer of `frozen` except a deliberate human-authorized
  unfreeze (none planned for v1 — ADR-0006).
- `RecordViolation` is `LastInsertId`-only by design (no `ON CONFLICT`, so no `RowsAffected` dedupe
  dance like `RecordCheckpoint`): re-detection records distinct rows, which is the intended evidence
  behavior. The future consistency check owns *when* to record, not dedupe.
- `proof_json` is TEXT; an empty `ProofJSON` is stored as the empty string (asserted), not coerced to
  NULL — only `detected_at` goes through `unixOrNil` (zero → NULL).
- Branch is `develop` with upstream `origin/develop`; pushing on PASS.
