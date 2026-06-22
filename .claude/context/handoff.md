## 2026-06-22 — Render certificate timestamps in UTC (`.UTC().Format`) to green the gate on non-UTC hosts

**Done:** Normalized all four `.Format(time.RFC3339)` call sites in `internal/certificate/handler.go`
to `.UTC().Format(time.RFC3339)` so every rendered cert timestamp is locale-independent `…Z`, matching
the rest of the federation. Added one host-independent regression test pinning the two test-covered
chips (§5 confirmation + COMPARISON ANCHOR coverage-since) to their UTC form. `mise run check` is now
green under any host TZ — clearing the gate the queued M-Deploy critical steps verify against.

**Files changed:**
- `internal/certificate/handler.go`: 4 sites `.Format(time.RFC3339)` → `.UTC().Format(time.RFC3339)`
  — `:653` bundle key `Revoked`, `:962` §4 `SigningKeyRevoked`, `:1013` §5 `BTCConfirmedAt`,
  `:1040` Comparison-Anchor `CoverageSince`. Format unchanged (still RFC-3339); only the zone is fixed.
- `internal/certificate/handler_test.go`: added `TestCertificateRendersTimestampsInUTC` (+ a small
  `renderCertBody` helper). It forces `time.Local` to a fixed UTC+1 zone (restored via `t.Cleanup`) so
  the store's `time.Unix` round-trip yields a `+01:00` offset on EVERY host, then asserts the §5 and
  coverage chips render `…Z` and no `+01:00` leaks. The two pinnable chips live on different fixtures
  (§5 needs the tiled-OTS fixture; coverage needs the set-once time seeded before `AdvanceAccepted`,
  i.e. `fixtureStoreCovered`), so each renders from its own store under the one `time.Local` swap.

**Verification:** `mise run check` → GREEN both under host TZ (all 27 packages `ok`) AND under
`TZ=America/New_York` (the gate that was RED on parent `HEAD`). Per-criterion:
- [x] `mise run check` green on a non-UTC host (`TZ=America/New_York mise run check` passes; was 2
  failures on parent).
- [x] `TZ=America/New_York go test -count=1 ./internal/certificate` passes; `TZ=UTC` passes; also
  `TZ=Asia/Kolkata` (half-hour offset) passes — no regression on UTC hosts.
- [x] Mutation / host-independence: reverting `.UTC()` at `:1013` OR `:1040` makes the new test FAIL
  under `TZ=UTC` (proving it is non-vacuous on UTC hosts, unlike the two pre-existing TZ-sensitive
  tests). Restored byte-clean (all 4 `.UTC()` present).
- [x] `gofmt -l internal/certificate/handler.go internal/certificate/handler_test.go` empty;
  `go vet ./internal/certificate` clean.
- [x] Oracle gate **N/A** — pure timestamp-rendering change; no signature / RFC-6962 / Merkle / did:web
  / proof / `go.mod` / `go.sum` / `schema.sql` path touched (`git diff --stat` = the 2 cert files only).

**Next:** The gate is now green on any host, so the queued M-Deploy `critical` work can be verified
cleanly. Strongest candidate: **trap SIGTERM in `run()`** (`signal.NotifyContext(..., os.Interrupt,
syscall.SIGTERM)` in `cmd/iscc-monitor/main.go:133`) — a 1-file change closing a `critical` ops issue,
needed before the container/GHCR step so `docker stop` drains cleanly. The Dockerfile + GHCR publish
workflow (with the `-ldflags` git-SHA build stamp on `/healthz`/`GET /version`) is the larger
`critical` follow-on. The `deploy/realm-testnet.txt` canonical realm doc is a cheap independent slice.

**Notes:**
- Two of the four fixed sites (`:653` / `:962`, both `key.Revoked`) have NO fixture exercising the
  §4/bundle revoked path today, so reverting their `.UTC()` fails no test (confirmed in the mutation
  sweep). This is exactly what `next.md` directed: fix all four for uniform locale-independence even
  though only two are test-covered. Not new debt — the revoked-key render path was already untested
  before this slice; a future §4-revoked fixture would naturally pin them.
- The store reads coverage/anchor instants back via `time.Unix` (`internal/store/hubs.go:84`,
  `ots.go:146/206`), which always re-wraps in `time.Local`. That is why the seeded `FixedZone`
  location does NOT survive a store round-trip — the `next.md` "seed a `FixedZone` time" sketch can't be
  realized through the public store seam. The faithful host-independent realization is to control
  `time.Local` for the duration of the test (a scoped, restored swap; the cert suite has zero
  `t.Parallel`, and `time.Local` is per-process so cross-package `go test` parallelism is unaffected).
  This is the minimal deviation from the literal Implementation-Note sketch, in service of its stated
  goal ("fails before the fix on EVERY host"). No `t.Setenv("TZ")` / process-env mutation used.
- `.claude/context/issues.md` is modified in the working tree by another role (the M-Deploy ops asks) —
  left uncommitted, not mine to touch. Only the 2 source files + this handoff are committed.
- Out of scope, untouched (per `next.md` Not-In-Scope): SIGTERM, Dockerfile/GHCR, version stamp,
  `deploy/` realm doc, README, the cert §6 mockup-humanize format step, the other SSR surfaces'
  timestamps (already UTC).
