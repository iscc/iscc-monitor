## 2026-06-22 — Harden the otsclient upgrade path — panic-recover + per-request timeout

**Done:** Made the production `follower.Upgrader` closure fail-closed against the two
Codex-confirmed hazards on the calendar-upgrade call: a library `panic` (uncomputable/unimplemented
op) is now recovered into a wrapped error and a per-request `context.WithTimeout(upgradeTimeout)`
bounds each calendar GET so one `OTSTick` pass can never hang. Both routed through a new
`safeUpgrade` FFI-boundary helper that mirrors the existing `recoverRead` pattern; closes the open
`normal` otsclient issue before the `stampRoot` calendar-submit step makes `upgrade()` reachable.

**Files changed:**
- `internal/otsclient/client.go`: added `upgradeTimeout = 30 * time.Second` const (best-effort
  transport bound, documented as not a gate); added `safeUpgrade(ctx, upgrade, seq, digest)` —
  derives `context.WithTimeout` with one explicit `defer cancel()` (no defer-in-loop leak) and a
  documented `recover()` that re-surfaces the library panic as a wrapped fail-closed error; the
  `buildUpgrader` loop now calls `safeUpgrade` instead of `upgrade` directly. The existing
  `if err != nil { ... upgrade sequence: %w ... }` wrap is unchanged, so a recovered panic flows
  to `OTSTick` as a back-off, never a crash. `GetBitcoinAttestedSequences` kept verbatim;
  pending-only upgrade preserved.
- `internal/otsclient/client_test.go`: added `TestUpgradePanicRecovered` (panic fake over the
  `merkle1` pending fixture → wrapped error carrying "upgrade sequence"+"panicked", no test-binary
  crash) and `TestUpgradeBoundsContext` (capturing fake asserts the upgrade ctx reports a deadline).
  Added `strings` import.

**Verification:** `mise run check` → GREEN (build + vet + test, all 23 packages incl.
`internal/otsclient`). `gofmt -l .` excl. `cauldron/` → clean. Per-criterion:
- [x] `go test -count=1 ./internal/otsclient` passes incl. the two new tests.
- [x] Mutation (reverted): removing the `safeUpgrade` `recover()` block → `TestUpgradePanicRecovered`
  PANICS/FAILS the binary; restored → green.
- [x] Mutation (reverted): replacing `context.WithTimeout(ctx, upgradeTimeout)` with the bare ctx →
  `TestUpgradeBoundsContext` FAILS ("upgrade ctx carried no deadline"); restored → green.
- [x] `go test -count=1 -run TestUpgradeAlreadyConfirmed ./internal/otsclient` passes — confirmed
  fixtures still classify at their exact heights (`hello-world`→358391, `empty`→129405);
  pending-only-upgrade invariant held, `failUpgrade` never fired.
- [x] Isolation: `GOOS=js GOARCH=wasm go build ./internal/{didweb,index,badge}` OK;
  `go list -deps ./internal/badge | grep -c -e internal/ots -e internal/otsclient` == 0;
  `go list -deps ./internal/otsclient | grep -c '^database/sql$'` == 1 (store.OTSRecord seam,
  unchanged).

**Next:** Wire `otsclient.Stamp` into `follower.stampRoot` (`follower.go:411`) so a stamped pending
row carries real `OTSBytes`/`CalendarURLs` — the deferred follow-up this step de-risked. With both
upgrade-path hazards now closed, that step can safely let `upgrade()` go live (a real proof reaching
an unimplemented op or a stalled calendar now degrades to a back-off, not a crash/hang). After it:
the `.ots` HTTP route (reads `OTSForRoot`) and certificate §5 BITCOIN ANCHOR (both depend on
confirmed rows existing).

**Notes:**
- Scope held: exactly 1 non-test file (`client.go`) + 1 test file, no `## Not In Scope` surface
  touched (`follower.go`, `stampRoot`, `main.go`/`runOTSLoop`, the `.ots` route and certificate §5 all
  unchanged; no `Stamper` seam / `MarkOTSSubmitted` added; `Stamp` body untouched).
- `safeUpgrade` takes `(ctx, upgrade, seq, digest)` (ctx first, Go convention) rather than next.md's
  literal `(upgrade, ctx, ...)` sketch — same seam, idiomatic ordering; the injected `seqUpgrade` fake
  still drives the recover offline.
- The `recover()` is an FFI-boundary guard with a docstring that re-surfaces the panic as a returned
  error (identical posture to `recoverRead`/`recoverParse`) — not a gate-dodge; no `//nolint`/`t.Skip`/
  swallow introduced.
- The other open `normal` issues (`hubDomain` ForceQuery; §4/bundle `host:port` DID; §6 timestamp) are
  on surfaces this diff did not touch — none resolved or made stale.
