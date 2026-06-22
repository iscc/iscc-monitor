## 2026-06-22 — Guard the live OTS `Stamp` calendar call with a `safeStamp` (timeout + panic-recover), symmetric with `safeUpgrade`

**Done:** Routed the live `otsclient.Stamp` calendar submit through a new `safeStamp` wrapper that
derives a per-request `context.WithTimeout(ctx, stampTimeout=30s)` and a `recover()`-to-error guard,
mirroring `safeUpgrade` verbatim, so a malformed or stalled calendar response now surfaces as a wrapped
best-effort error the loop backs off on — never a process crash or a hung OTS goroutine. Introduced the
injectable `stampFn` seam + `buildStamper(stampFn)` split (parallel to `seqUpgrade`/`buildUpgrader`),
keeping `Stamp`'s exported signature unchanged so `cmd/iscc-monitor` is untouched.

**Files changed:**
- `internal/otsclient/client.go`: added `const stampTimeout = 30 * time.Second` (best-effort transport
  bound, same wording class as `upgradeTimeout`); added the `stampFn` seam type
  (`func(ctx, calendarURL, digest [32]byte) (opentimestamps.Sequence, error)`, the signature of
  `opentimestamps.Stamp`); split `Stamp` into a thin delegator over `buildStamper(opentimestamps.Stamp)`
  holding the File-assembly/serialize; added `safeStamp` wrapping *only* the `stampFn` call with the
  timeout + recover guard. Preserved the existing `"otsclient.Stamp: %q: %w"` error wrap on the
  non-panic transport-fault path (the recover path uses `"otsclient.Stamp: %q panicked: %v"`).
- `internal/otsclient/client_test.go`: added `TestStampPanicRecovered` (inject a panicking `stampFn`,
  assert wrapped non-nil error + nil bytes, no crash) and `TestStampBoundsContext` (inject a
  ctx-capturing `stampFn`, assert the captured ctx reports a deadline). `TestStampSerializesRoundTrip`
  left intact (the File-assembly seam is unchanged).

**Verification:** `mise run check` → green (all 27 packages build + vet + test; `gofmt -l .` empty).
- `mise exec -- go test -count=1 -run 'TestStamp|TestUpgrade' ./internal/otsclient` → `ok` (forced fresh) — **PASS**
- `TestStampPanicRecovered` passes; **mutation-proven**: deleting the `recover()` in `safeStamp` makes
  the test panic the binary (FAIL with goroutine dump) — **PASS**
- `TestStampBoundsContext` passes; **mutation-proven**: reverting `safeStamp` to the bare `ctx`
  (drop `context.WithTimeout`) makes it FAIL ("stamp ctx carried no deadline") — **PASS**
- `git diff --name-only` shows only `internal/otsclient/client.go` + `internal/otsclient/client_test.go`
  (`cmd/iscc-monitor/main.go` unchanged) — **PASS**
- `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge` succeeds; those
  leaves' dep lists contain `internal/otsclient` 0 times (purity guard holds) — **PASS**

**Next:** The two `normal` OTS guard issues are now both resolved (upgrade via `safeUpgrade`, stamp via
`safeStamp`) — the asymmetry the `otsclient.md` durable rule flagged is closed. With the production-path
defects cleared, the loop can return to the front-of-queue milestone arc: the WASM-verifier **signature
half** (browser did:web key resolution + checkpoint-note signature verify). That one is still flagged
**design-first / STOP-candidate** (browser-side did:web resolution is non-trivial) — do a DESIGN PASS
before building, and do NOT loosen `verifier.html`'s "hub-signed root" success copy until it lands. The
other open milestone sub-steps remain human-blocked (Pages repo-Settings enablement) or
offline-unprovable (OTS Bitcoin-confirmed half needs a live calendar).

**Notes:**
- Mirrored `safeUpgrade` exactly: `safeStamp` wraps *only* the `stampFn` call (not the
  serialize/File-assembly body), same as `safeUpgrade` wraps only `upgrade(...)`. The recover is a
  fail-closed FFI boundary returning an `err` (ADR-0004: OTS never crashes the follower) — not a
  gate-dodge; no `//nolint`, `t.Skip`, or swallow.
- No new imports: `context` and `time` were already imported.
- The `otsclient.md` learnings detail file still carries the open `normal` "Stamp is UNGUARDED" bullet
  (lines 45-53); review should update/collapse it to a `settled:` entry now that `safeStamp` landed,
  matching the `safeUpgrade` `settled:` bullet's shape — the durable rule "any new live OTS calendar
  call MUST go through such an FFI-boundary guard" still holds. (Learnings files are review-owned, so I
  did not edit it.)
