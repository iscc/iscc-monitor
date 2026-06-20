# Handoff

## 2026-06-20 — Review of: internal/config pure config loader leaf

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/config` — a pure, dependency-free `Load(get func(key string)
(string, bool)) (Config, error)` that turns a flat injected key/value lookup into a validated, typed
`Config{DBPath, RealmPath string; Normal, Frozen time.Duration}`, with documented defaults
(Normal=5m, Frozen=1h), `time.ParseDuration` parsing, and the load-bearing `Frozen >= Normal`
cross-check (ADR-0006). Scope is exactly the two scoped files (1 production + 1 test) in a new leaf
package; nothing else touched. Independently re-verified: gates green, 100% statement coverage,
imports clean (`{fmt time}`), go.mod/go.sum byte-identical, no gate-dodging.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` all `ok` (config + the five
  existing packages).
- [x] `gofmt -l internal/config` and `gofmt -l .` (whole tree) — both print nothing.
- [x] `go test -count=1 ./internal/config` — PASS (uncached, 0.021s); `-cover` reports 100.0%.
- [x] `go test -count=1 -run TestLoad ./internal/config` — PASS.
- [x] `go list -deps ./internal/config | grep -E '^(net|net/http)$'` — empty; own `.Imports` =
  `[fmt time]` (pure leaf; the only `os` mentions in source are docstring prose about the binary's
  future `os.LookupEnv` closure, not imports).
- [x] `git status --short go.mod go.sum` — empty; `git diff HEAD~1..HEAD -- go.mod go.sum` empty
  (byte-identical; no dependency added).
- [x] Table/golden assertions non-vacuous: `TestLoadGolden` round-trips every field of a full input;
  `TestLoadDefaults` proves minimal `{DB, realm}` loads with `Normal=5m`/`Frozen=1h` and asserts the
  defaults satisfy `Frozen >= Normal`; `TestLoad` covers single-override + `Frozen == Normal`
  accepted; `TestLoadErrors` covers missing/empty DB+realm, unparseable Normal/Frozen, non-positive
  Normal/negative Frozen, and `Frozen < Normal` — each asserting a non-nil error naming the offending
  key AND the zero `Config` returned alongside it.
- [x] Cross-check against consumer: `Loop.Normal`/`Loop.Frozen` (`follower/loop.go`) are
  `time.Duration` fields config feeds directly, and `Frozen >= Normal` matches `due()`'s back-off
  (a frozen hub polls on the longer `Frozen` interval, ADR-0006).
- [x] Oracle/conformance gate correctly N/A — diff touches no proof/verify/didweb/merkle/consistency/
  signature/fsck/notecheck path; pure no-crypto, no-network value parser.
- [x] Gate-integrity scan over all unpushed commits (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion.

**Issues found:** (none)

**Next:** `cmd/iscc-monitor/main.go` is the now-unblocked consumer slice and config's only consumer:
build the `os.LookupEnv`-backed `get` closure → `config.Load` → `os.ReadFile(cfg.RealmPath)` →
`registry.Parse` → `store.Open(cfg.DBPath)` → per `Entry` `store.UpsertHub(domain, origin, baseURL)`
→ `[]follower.HubTarget` → `follower.Loop{Normal: cfg.Normal, Frozen: cfg.Frozen, …}` →
`Loop.Run(ctx)`. That step must make the still-open **origin-export decision**: `UpsertHub` needs
`<domain>/log` and `logclient.origin` is private — export `logclient.Origin` OR carry origin on
`registry.Entry`, never a second deriver (learnings: "One `origin()` helper, golden-tested"). The
blocking `Run` loop is untestable, so it is correctly the binary step, not a leaf. Equally valid (and
heavier) alternative: the merkle-backed **equivocation** trigger — first slice to trip the oracle
gate (needs `transparency-dev/merkle` + tile fixtures + `fsck`/`notecheck` parity).

**Notes:**
- `required` does NOT trim whitespace, so a whitespace-only path (`" "`) passes config and would fail
  later at `os.ReadFile`/`store.Open`. Acceptable for a leaf that validates presence (the binary owns
  I/O), but the binary should surface that downstream fs error clearly. Not a blocker.
- Key names `ISCC_MONITOR_{DB,REALM,NORMAL,FROZEN}` are package constants the binary references
  symbolically — not yet load-bearing against any external contract, renamable cheaply at wiring time
  if a flag-based surface is preferred.
- M1 remains in progress (state.md): equivocation trigger, `cmd/` binary, coverage
  (`monitored_since`), structured logs, `/metrics`, real alert transport still missing — Loop is
  CONTINUE, not DONE. No open critical/normal issue; no human-only decision pending.
- Branch is `develop` (never `main`); pushing on PASS.
