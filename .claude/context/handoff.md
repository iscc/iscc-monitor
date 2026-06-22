## 2026-06-22 — Review of: Guard the live OTS `Stamp` calendar call with a `safeStamp` (timeout + panic-recover), symmetric with `safeUpgrade`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance routed the live `otsclient.Stamp` calendar submit through a new `safeStamp`
wrapper that derives a per-request `context.WithTimeout(ctx, stampTimeout=30s)` and a `recover()`-to-error
guard, mirroring `safeUpgrade` verbatim via the new `stampFn` seam + `buildStamper` split. `Stamp`'s
exported signature is unchanged so `cmd/iscc-monitor` is untouched. This closes the last open OTS guard
`normal` — the stamp path is now symmetric with the already-hardened upgrade path, exactly as `next.md`
specified. Scope is precisely the two files asked for (1 production + 1 test), nothing from `## Not In
Scope` was touched.

**Verification:**
- [x] `mise run check` → green (all 27 packages build + vet + test) — **PASS**.
- [x] `mise exec -- go test -count=1 -run 'TestStamp|TestUpgrade' ./internal/otsclient` → `ok` (forced fresh) — **PASS**.
- [x] `TestStampPanicRecovered` mutation-proven: removing the `recover()` in `safeStamp` makes the test PANIC the binary (goroutine dump, FAIL at `client_test.go:214`); restored byte-identical — **PASS**.
- [x] `TestStampBoundsContext` mutation-proven: dropping `context.WithTimeout` makes it FAIL ("stamp ctx carried no deadline"); restored byte-identical — **PASS**.
- [x] `git diff HEAD~1..HEAD --name-only` shows only `internal/otsclient/client.go` + `client_test.go` (+ context handoff); `cmd/iscc-monitor/main.go` untouched — **PASS**.
- [x] Production-path reality check — `main.go:236` (`stampFunc`) calls `otsclient.Stamp` inside `runOTSLoop`'s live goroutine (`main.go:151`), so the guard is genuinely on a live path, not a dormant helper — **PASS**.
- [x] `gofmt -l .` → empty — **PASS**.
- [x] WASM purity guard — `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge` succeeds; `go list -deps` shows `internal/otsclient` dep count 0 for all of didweb/index/badge/proof-verify — **PASS**.
- [x] Conformance/oracle gate — N/A: the diff touches no signature/Merkle/proof/didweb/fork-shrink-equivocation code, only the OTS transport-adapter FFI boundary. `internal/proof/verify` stays import-pure. CI `notecheck` config unchanged (0 files under `.github/` touched).
- [x] Gate-circumvention scan over all unpushed commits (`origin/develop..HEAD`) — no `//nolint`, `t.Skip`, build-tag exclusions, or deleted tests/assertions; the lone `//nolint` token is inside a docstring stating the guard is "never a //nolint or swallow" (the opposite of a dodge). Code diff is purely additive (+62 src, +53 test) — **PASS**.

**Issues found:** (none new). Resolved + deleted the `normal` "production OTS stamp path has neither a panic-recover nor a per-request timeout" issue after mutation-verifying both guards are live.

**Codex second opinion:** Clean — "The new Stamp wrapper preserves the exported behavior while adding a bounded context and panic recovery around the calendar call. Tests pass, and I found no introduced correctness issues." Independently corroborates the reviewer's verification; no findings to triage.

**Visual check:** n/a — no SSR surface changed. The diff is an OTS transport-adapter guard + offline tests; no template, render path, or chrome touched.

**Next:** The two OTS guard `normal`s are now both closed (upgrade + stamp symmetric). Resume the front-of-queue **WASM-verifier signature half** (state.md "Next Milestone"): browser did:web-key resolution + checkpoint-note signature verify — today only inclusion + id-binding run, so a cross-origin `verified` still trusts the monitor for the signature (filed `normal`). This is flagged **design-first / STOP-candidate** (browser-side did:web resolution is non-trivial) — do a DESIGN PASS before building, and do NOT loosen `verifier.html`'s "hub-signed root" success copy until the check lands. The other open milestone sub-steps remain human-blocked (Pages repo-Settings custom-domain enablement, `normal`) or offline-unprovable (OTS Bitcoin-confirmed half needs a live calendar + real BTC confirmation).

**Notes:**
- The implementation is a faithful, verbatim mirror of `safeUpgrade`: `safeStamp` wraps *only* the `stampFn` call (not the serialize/File-assembly body), uses the same one-`defer cancel()`-per-call shape (no defer-in-loop, since `Stamp` is single-shot), and the error wraps are preserved (`"otsclient.Stamp: %q: %w"` on transport fault, `"... panicked: %v"` on recover). No new imports (`context`/`time` already present).
- CI is green at the parent (`8496579`, the prior review commit); the current advance (`6440e24`) is unpushed and additive (isolated guard + offline tests, all green under local `mise run check`), so CI will stay green once pushed. The `Pages` workflow failure on develop is the known human-blocked custom-domain enablement step (filed `normal`), not a code regression.
- Learnings: collapsed the `otsclient.md` "Stamp is UNGUARDED" bullet into a `settled:` entry matching the `safeUpgrade` shape, preserving the durable rule "any new live OTS calendar call MUST go through such an FFI-boundary guard." File is 6 bullets / well under the rotation budget.
