## 2026-06-22 — Move the three instance-identity env keys into the `internal/config` leaf

**Done:** Moved the three optional masthead-identity env keys (`ISCC_MONITOR_INSTANCE` /
`ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME`) out of `main.go`'s inline `os.Getenv`-based
`identity()` and into `internal/config` as typed `Config.Instance`/`Operator`/`RealmName` fields read
through the existing no-validation `optional(get, key, "")` helper; `main.go`'s `identity` is now
`identity(cfg config.Config) dashboard.Identity`, and CLAUDE.md documents all three keys with the
realm-name-vs-path distinction. `internal/config` stays a pure leaf (still imports exactly `{fmt time}`).

**Files changed:**
- `internal/config/config.go`: added `keyInstance`/`keyOperator`/`keyRealmName` const block (with the
  realm-name-vs-path rationale ported from main.go), three `string` fields on `Config`, three
  `optional(get, key, "")` reads in `Load`; updated the package doc + `Config` struct doc to list the
  three optional keys.
- `cmd/iscc-monitor/main.go`: deleted the inline identity const block + `os.Getenv`-based `identity()`
  body; `identity(cfg config.Config)` now builds `dashboard.Identity` from `cfg.Instance/Operator/
  RealmName`; its single caller at the `serveMetrics` goroutine launch passes `cfg`. `os` stays imported
  (still used for `os.ReadFile`/`os.LookupEnv`/`os.Exit`/`os.Stderr`).
- `internal/config/config_test.go`: `TestLoadGolden` now sets + asserts all three identity keys
  (full round-trip); `TestLoadDefaults` asserts the three fields are `""` when absent; added two
  partial-set `TestLoad` table cases (instance-only, realm-name-only) so per-field independence is
  non-vacuous; updated the test-file doc comment.
- `CLAUDE.md`: added the three keys to the "Running a local dev instance" env-var bullet list, each
  `(optional)`, with the `ISCC_MONITOR_REALM_NAME` = human NAME vs `ISCC_MONITOR_REALM` = realm-document
  PATH distinction.

**Verification:** `mise run check` — build + vet clean; `gofmt -l .` empty outside `cauldron/`; 25/27
packages `ok`. Per-criterion:
- [x] `go test -count=1 -run TestLoad ./internal/config` — pass (golden incl. three identity fields,
  absent→`""` default, both partial-set cases).
- [x] `go test -count=1 ./cmd/iscc-monitor` — pass (`identity(cfg)` signature compiles + wires at the
  `serveMetrics` launch).
- [x] Assertion (keys set → fields equal; keys absent → `""`) covered by `TestLoadGolden` /
  `TestLoadDefaults` and proven non-vacuous by the mutation below.
- [x] Import-purity assertion: `go list -f '{{.Imports}}' .../internal/config` = exactly `[fmt time]`
  (no `os`/`dashboard`/`net`); `GOOS=js GOARCH=wasm go build ./internal/config` still builds.
- [x] Mutation check (run + reverted byte-clean): forcing `instance := "MUTANT"` in `Load` makes
  `TestLoadGolden` AND `TestLoadDefaults` FAIL; restored, `git diff --stat` shows only the intended
  additive change (47 ins / 1 del), no mutation residue.
- [x] Oracle gate **N/A** — pure startup-value parsing, no signature/Merkle/did:web/proof path.
  `go.mod`/`go.sum`/`schema.sql` byte-identical (not in the diff).

**Next:** The proofserve-trio masthead slice (`browser.html`, `records.html`, `record.html`) — thread
`dashboard.Identity` into those three SSR mastheads, and (the natural trigger) fold the now-3x-duplicated
`instanceFallback`/`operatorFallback` consts + a single exported `Resolve` into one shared leaf, closing
the `low` issue. `internal/verifier` stays EXCLUDED (its `.codes` chrome is the verifier-app identity).

**Notes:**
- **PRE-EXISTING, UNRELATED FAILURE (not introduced by this increment):** `internal/certificate` fails
  two tests — `TestCertificateComparisonAnchor` (handler_test.go:1478, "body missing comparison-anchor
  marker `2026-01-05T09:00:00Z`") and `TestCertificateBitcoinAnchorConfirmed` (handler_test.go:1685,
  "body missing §5 confirmation time `2026-02-14T18:40:00Z`"). I confirmed these FAIL on a clean
  `git stash` of my changes, and the last cert review handoff (commit 7a32458) recorded "all 27 packages
  ok" — so this is a regression that surfaced since, NOT from the config move (which touches no cert
  code). Both rendered bodies now show a tier-2 `data-state="failed"` panel instead of the expected
  RFC-3339 markers. The fixtures use fixed past timestamps (coverSince 2026-01-05, confirm 2026-02-14)
  vs today 2026-06-22 — strong signal of a wall-clock-dependent rendering branch (or a recently-landed
  cert change that broke these) that has gone stale. Out of scope here; flagging for `review` to triage
  / file. This is why `mise run check` is RED overall despite every criterion of THIS package passing.
- `internal/config` purity nuance still holds: `go list -deps` shows `os` transitively (via `fmt`), but
  the load-bearing rule (no direct `net`/`net/http`/`database/sql`/`dashboard`; WASM-shareable) is
  verified by the exact direct-import set `{fmt time}` and the `GOOS=js GOARCH=wasm` build.
- Did NOT touch the `low` duplication (`instanceFallback`/`operatorFallback` × 3 packages) — that lands
  WITH the proofserve slice, per next.md's Not-In-Scope. No handler fail-safe defaulting was changed;
  config carries empty strings when keys are unset, handlers keep applying their static placeholder.
- Untracked `.claude/adr/0013-server-packaging-and-deployment.md` and modified
  `.claude/context/{issues,target}.md` are NOT mine (other roles) and are excluded from my commit.
