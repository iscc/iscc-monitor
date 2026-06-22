## 2026-06-22 — Review of: Move the three instance-identity env keys into the `internal/config` leaf

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance moved `ISCC_MONITOR_{INSTANCE,OPERATOR,REALM_NAME}` out of `main.go`'s inline
`os.Getenv`-based `identity()` into typed `Config` fields read via the existing no-validation
`optional(get, key, "")` helper, with `identity(cfg)` rebuilding `dashboard.Identity` from them and
CLAUDE.md documenting all three keys (incl. the realm-name-vs-path distinction). The increment is exactly
what `next.md` asked: scope-disciplined (2 prod files + 1 doc + 1 test), import-pure (`internal/config`
stays `{fmt time}`-only, WASM-green), mutation-proven non-vacuous, and its own gates are clean. The ONE
caveat (hence PASS_WITH_NOTES not PASS): `mise run check` is RED, but solely from two PRE-EXISTING,
TZ-dependent `internal/certificate` test failures that touch zero code in this diff and that I proved
fail identically on the parent commit — root-caused and filed as a new `normal` issue.

**Verification:**
- [x] `go test -count=1 -run TestLoad ./internal/config` — pass (golden round-trip incl. 3 identity
  fields, absent→`""` default, both partial-set table cases).
- [x] `go test -count=1 ./cmd/iscc-monitor` — pass (`identity(cfg)` signature compiles + wires at the
  `serveMetrics` goroutine launch).
- [x] Import-purity assertion — `go list -f '{{.Imports}}' .../internal/config` = exactly `[fmt time]`
  (no `os`/`dashboard`/`net`); `GOOS=js GOARCH=wasm go build ./internal/config` builds.
- [x] Mutation check (reviewer re-ran on HEAD, restored byte-clean) — forcing `instance := "MUTANT"`
  in `Load` FAILS both `TestLoadGolden` AND `TestLoadDefaults`; `config.go` restored byte-identical.
- [x] `gofmt -l .` empty outside `cauldron/`; `go vet ./internal/config ./cmd/iscc-monitor` clean.
- [x] Scope discipline — diff = `config.go` + `main.go` (2 prod) + `CLAUDE.md` (doc) + `config_test.go`
  (test); every `next.md` Not-In-Scope item correctly left undone (no proofserve threading, no fallback-
  const consolidation, no `Identity.Realm` rename, no validation added, no `dashboard` import in config).
- [x] Quality-gate integrity scan over all unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed-
  error/deleted-assertion. The lone grep hit is a prose line in the prior handoff describing this scan.
- [x] Oracle gate N/A — pure startup-value parsing, no signature/Merkle/did:web/proof path touched;
  `go.mod`/`go.sum`/`schema.sql` not in the diff (byte-identical).
- [ ] `mise run check` overall — RED, but ONLY from two pre-existing, unrelated `internal/certificate`
  failures (see Issues). The config increment's own packages (`internal/config`, `cmd/iscc-monitor`) and
  all 23 other packages are `ok`.

**Issues found:**
- **Certificate timestamps rendered in LOCAL time, not UTC** (filed `normal`). `mise run check` fails on
  any non-UTC host: `TestCertificateComparisonAnchor` and `TestCertificateBitcoinAnchorConfirmed` expect
  `…Z` (UTC) chips but the handler renders the same instants in the server's local zone
  (`2026-01-05T10:00:00+01:00` / `2026-02-14T19:40:00+01:00` on this CET box). PROVEN pre-existing
  (identical 2 failures on parent `d7e1fdc` in a throwaway worktree) and PROVEN environmental (CI/UTC is
  green, hence the prior `7a32458` "27 ok"). The advance's "stale-fixed-timestamps vs wall-clock" guess
  was the wrong mechanism — the instants are correct, only the rendered zone is local. Fix:
  `.UTC().Format(time.RFC3339)` on the coverage-since + §5 confirmation chips. Untouched by this diff.
- Resolved + deleted: **"Instance-identity env keys read inline in main.go…"** — this increment closes
  it (keys parsed through `internal/config`, realm-name key ratified, CLAUDE.md documents all three).

**Codex second opinion:** unavailable — the `codex review` launch was denied by the Claude Code auto-mode
classifier ("Create Unsafe Agents": it refused to spawn an agent with `sandbox_mode=danger-full-access`
+ `approval_policy=never` under a generic iteration request). No second opinion this iteration; graceful
degradation per protocol. The increment is low-risk (config-leaf value parsing, no trust-root path), so
the missing second skeptic is a low concern here.

**Visual check:** n/a — no SSR surface changed. The diff touches only `internal/config` value parsing and
`main.go` wiring; no template, no SSR handler render path (`dashboard`/`dossier`/`web`/`certificate`)
is modified.

**Next:** The proofserve-trio masthead slice — thread `dashboard.Identity` into `browser.html`,
`records.html`, `record.html` (the remaining 3 of 6 SSR mastheads), and fold the now-3x-duplicated
`instanceFallback`/`operatorFallback` consts + a single exported `Resolve` into one shared leaf (closes
the `low` duplication issue, its natural 4th-copy trigger). `internal/verifier` stays EXCLUDED (`.codes`
chrome). ALTERNATIVELY, the certificate-UTC fix is a small, well-scoped, gate-greening candidate (it is
the only thing keeping `mise run check` red locally) — a strong candidate for the very next slice since a
green local check matters for the M-Deploy work now queued in target.md/ADR-0013.

**Notes:**
- The cert TZ failure means `mise run check` is RED *locally* but GREEN in CI (UTC). The config increment
  is genuinely complete and correct; I withheld a clean PASS only because the protocol forbids PASS while
  any check is red, and chose PASS_WITH_NOTES because the redness is provably pre-existing, unrelated, and
  environment-specific. The next define-next should weigh fixing the cert TZ bug first so subsequent
  reviews aren't masked by a persistent red.
- Uncommitted `.claude/` files NOT mine (other roles): `target.md`, `issues.md` (the M-Deploy ops asks +
  ADR-0013), and untracked `.claude/adr/0013-server-packaging-and-deployment.md`. I committed only my
  review artifacts (handoff, learnings/config.md, issues.md edits).
- Two stale prunable git worktrees exist (`/tmp/iscc-monitor-536e429`, `/workspace/iscc-monitor-qa`) from
  prior runs — left alone (not this iteration's; force-removing others' could disrupt parallel work). My
  own parent-commit worktree was cleanly removed.
- `internal/config` purity nuance confirmed: direct imports are exactly `{fmt time}`; `os` appears only
  transitively via `fmt` (stdlib, unavoidable) — the load-bearing rule (no `net`/`net/http`/`sql`/
  `dashboard`; WASM-shareable) holds, verified by the `GOOS=js GOARCH=wasm` build.
