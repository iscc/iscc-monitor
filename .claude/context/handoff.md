# Handoff

## 2026-06-21 — Wire `CheckEquivocation` into `follower.checkConsistency` as the third self-consistency trigger

**Done:** Closed the last open M1 self-consistency trigger end-to-end. `checkConsistency` now evaluates
shrink → fork → **equivocation**: on the growing-pair case (`info.TreeSize > prevSize`, prior accepted
checkpoint on record) it builds the RFC-6962 consistency proof from the LOCAL mirror via
`ConsistencyProofFromTiles(ctx, store.SQLiteFetcher{Store, HubID}.ReadTile, prevSize, info.TreeSize)`,
feeds the hashes to `CheckEquivocation(prevSize, prevRoot, info.TreeSize, info.Root, proof)`, and freezes
the hub (kind `"equivocation"`) on a true verdict. A proof-build error (most often a missing tile —
production does not mirror tiles yet, that is M2) is treated as "cannot evaluate this poll" and skips the
branch — never a false-positive freeze, never an aborted poll (ADR-0006 "freeze, never crash").

**Files changed:**
- `internal/follower/follower.go`: extended `checkConsistency`'s `switch` with the equivocation branch
  (proof sourced from the mirror, compared against the prior accepted root, missing-tile error swallowed
  narrowly + documented); updated the package + `PollHub` + freeze doc comments to name the third trigger.
  Production imports unchanged (`{context, fmt, time}` + internal `logclient`/`store`).
- `internal/follower/equivocation_test.go` (new, test-only): ports `buildTree`/`nodeHash`/tile-synthesis
  from `proofbuilder_test.go`, records a real ~300-leaf `testonly.Tree`'s level-0 hash tiles into the
  store (full tile index 0 width 256, partial index 1 width 44), then drives both the freeze case
  (wrong root → `checkConsistency` + `freeze`) and the non-vacuous happy path (correct root → no freeze),
  plus a missing-tiles-does-not-freeze case.

**Verification:** `mise run check` → green, all 8 packages; `gofmt -l .` empty.
- [x] `go test -run TestPollHubEquivocation ./internal/follower` — PASS (freeze + happy path + restart).
- [x] `go test -run 'TestPollHubFork|TestPollHubShrink|TestPollHubVerifiedAdvances' ./internal/follower`
      — PASS (other two triggers + clean advance unregressed).
- [x] Equivocation freeze asserts: `violations.kind == "equivocation"`, `follow_state.frozen == 1`,
      cursor did NOT advance (stays at prevSize 5), exactly one alert, re-detection records a 2nd
      violation without re-alerting, evidence survives a store reopen.
- [x] Non-vacuous happy path: a correct-root growing observation over the SAME mirrored tiles does NOT
      freeze (0 violations on the clean store).
- [x] Missing-tiles case: no tiles mirrored → proof-build error → branch skipped → no violation, nil
      error (the ADR-0006 false-positive guard).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — succeeds (pure proof builder untouched).
- [x] go.mod/go.sum byte-identical (test reuses already-required merkle/testonly/tessera-api deps).
- [x] Production follower imports = `{context, fmt, time}` + internal; `go list -deps ./internal/store`
      has no `net/http` (store stays a leaf).

**Mutation evidence (non-vacuousness proven both directions, then reverted byte-clean):**
- Forced `eq` always false → the freeze case FAILED (`want (true, "equivocation")`).
- Forced `eq` always true → the happy path FAILED (`want clean`). So a "never freezes" or "always
  freezes" wiring cannot ship green.

**Next:** The `fsck` root-rebuild conformance slice — `fsck.New(...).Check(...)` over the
`SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof`. It is explicitly the
*next* conformance step (Not In Scope here): it needs the heavy `fsck`/otel/klog dep, which cannot land
in a leaf package, and it wants real on-disk tile fixtures under `testdata/live/` (this step synthesizes
tiles in-test). That slice is where the mirror path first faces the truly-independent trust-root oracle.
Also still pending (separate later slices, untouched here): mirroring tiles inside `PollHub`'s production
path (M2 — only then does this equivocation branch become load-bearing against real hub data), the sb1
stale-key fixture refresh (`22b08f3e`→`069d0f14`), and CI/`notecheck` + `go mod tidy` go.sum divergence
wiring.

**Notes:**
- **The equivocation branch is wired but DORMANT in production until M2 mirrors tiles.** Today
  `PollHub` never writes tiles, so the live path always hits the missing-tile → skip case (no freeze).
  This is the deliberate conservative choice (`next.md` Implementation Notes): freezing on a missing
  tile would be a false positive. The branch is fully proven correct against synthesized mirrored tiles
  in-test; it becomes load-bearing the instant the M2 tile-ingestion writer lands. Reviewer should treat
  the in-test tile seeding (`seedMirrorTiles` via `RecordTile`) as the stand-in for that future writer.
- **Error-vs-violation discipline is the load-bearing subtlety.** `CheckEquivocation` turns a
  non-verifying proof into `(true, nil)` (the proof failing IS the evidence), but
  `ConsistencyProofFromTiles` returns a genuine Go error on a tile fault. The branch swallows ONLY the
  proof-build error (documented inline); a genuine `st` query failure still surfaces via `CheckpointAt`
  above, and `CheckEquivocation`'s own (currently unreachable-by-type) error is propagated, not swallowed.
- **"Compare against the prior accepted root, not the contradicting evidence" is honored.** `prevRoot`/
  `prevSize` come from `CheckpointAt(prevSize)` (the prior accepted checkpoint, `LIMIT 1` = lower-rowid
  prior root after a freeze records the contradiction at the same size); `info.Root`/`info.TreeSize` are
  the new observation. The freeze evidence row is never the proof/root source.
- **Test seam decision:** the equivocation case calls the unexported `checkConsistency` + `freeze`
  directly (package-internal) rather than driving `PollHub` end-to-end, because `PollHub` needs a
  genuinely `StatusVerified` signed checkpoint whose signature encodes the synthesized tree's root — no
  such fixture exists, and forging one is out of scope. Assertions remain on observable store outputs
  only (`violations`/`follow_state`/alert count via the shared `countRows`/`assertViolation` inspectors),
  never on follower internals, per the PRD seam rule. This mirrors how the fork/shrink tests would not
  meaningfully differ driven through `PollHub` vs the helpers.
- Exactly 2 files touched (1 production + 1 test), within the ≤3 scope. No out-of-scope changes.
