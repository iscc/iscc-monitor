# Handoff

## 2026-06-20 — Review of: `cmd/iscc-monitor` binary — wire config → registry → store → poll loop

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance stands up `cmd/iscc-monitor` as the wired
`config.Load → os.ReadFile+registry.Parse → store.Open/UpsertHub → follower.Loop.Run` entrypoint, with
all registry→store wiring extracted into a testable `registerHubs` helper, plus a one-line exported
`logclient.Origin` wrapper that delegates to the private `origin` (no second deriver). Gates are green,
scope is tight (2 non-test/doc files, both exactly what `next.md` asked), and the highest-probability
bug — origin must be `<domain>/log`, never the bare domain — is avoided and proven by an offline smoke
run persisting `sb0.iscc.id/log` / `sb1.amlet.id/log`.

**Verification:**
- [x] `mise run check` — green: `go build`/`go vet`/`go test ./...` all `ok` (7 packages).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go build -o /tmp/iscc-monitor ./cmd/iscc-monitor` — exit 0.
- [x] `go test -run TestOrigin ./internal/logclient` — PASS (golden `Origin`/`origin` vectors hold).
- [x] `go test -run TestRegisterHubs ./cmd/iscc-monitor` — PASS (2 targets, `HubID>0`, `BaseURL ==
  "https://"+domain`, idempotent second call returns identical `HubID`s).
- [x] `git diff HEAD~1..HEAD -- go.mod go.sum` — empty (no dependency added).
- [x] Missing-env failure surfaces cleanly: `env -u ISCC_MONITOR_DB -u ISCC_MONITOR_REALM
  /tmp/iscc-monitor` → exit 1, stderr `iscc-monitor: config: required key "ISCC_MONITOR_DB" is missing`
  (no panic).
- [x] Whitespace-only realm path fails at `os.ReadFile` with the path named (`read realm document "   ":
  open …`), exit 1 — the config presence-only gap is now surfaced, not silent.
- [x] Offline end-to-end smoke (`NORMAL=10m`, no tick): registered both hubs, clean SIGINT → exit 0,
  persisted `hubs` rows = `(sb0.iscc.id, sb0.iscc.id/log, https://sb0.iscc.id)` and
  `(sb1.amlet.id, sb1.amlet.id/log, https://sb1.amlet.id)` — origin is `<domain>/log`, never bare.
- [x] Gate-integrity scan over all unpushed commits (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion in code (the only `nolint`/`Skip` strings are in prior
  context-doc prose). Go changes are exactly the 4 expected files; private `origin` body byte-identical.
- [x] Oracle/conformance gate — correctly **N/A**: the diff touches no
  proof/verify/didweb/merkle/consistency/signature/fsck/notecheck path (`Origin` only re-exports the
  already-golden `origin`); go.mod/go.sum byte-identical.

**Issues found:** (none)

**Next:** The merkle-backed **equivocation** trigger — the heavy slice deferred from this and prior
steps, and the first slice to trip the oracle/conformance gate (needs `transparency-dev/merkle` + tile
fixtures, `fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs the hub's
`IsccLogInclusionProof`, `notecheck` parity in CI). Alternatively, the lighter remaining M1 gaps this
wiring surfaces: coverage (`monitored_since`), structured logging to replace the two stderr placeholders
(`alert` + the documented swallowed `Run`/per-tick error), `/metrics`, and the `hub_keys` did:web cache
write (with the stale sb1 fixture refresh `22b08f3e`→`069d0f14`).

**Notes:**
- Signatures match exactly: `UpsertHub(ctx, domain, origin, baseURL)` called with `(e.Domain, org,
  e.BaseURL)`; `Loop{Store,Fetcher,Targets,Normal,Frozen,Alert}` all fields present; `alert(int64,
  string)` matches `AlertFunc`; `NewHTTPFetcher(nil)` valid (nil → http.DefaultClient).
- `Loop.lastPoll` is lazily inited inside `Tick`, so the binary building `&follower.Loop{…}` without
  the unexported field is panic-safe — verified in `loop.go` (`if l.lastPoll == nil`).
- `Run` returns `ctx.Err()` unwrapped, so the binary's `err != context.Canceled` (`==`, not
  `errors.Is`) is correct for a `signal.NotifyContext(os.Interrupt)` cancel → exit 0 (smoke-confirmed).
  A style-only `errors.Is` would be more defensive but is not a bug today; not flagged.
- The stderr `alert` and the documented per-tick swallowed error are placeholders explicitly in
  `next.md`'s Not-In-Scope (real alert transport + structured logging are later steps), with justifying
  inline comments — not gate-dodging.
- `Loop.Run` remains correctly untested (blocking ticker `select`); the smoke run exercises the real
  `Run` path manually but is not a committed test (would require sleeping/network). All branching lives
  in the covered `registerHubs` + the follower's injected-`now` `Tick`/`due()`.
