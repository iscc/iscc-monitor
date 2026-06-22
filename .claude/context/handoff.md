## 2026-06-22 — Review of: Static-site generator for the Surface-C verifier deploy (`cmd/verifier-site`)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance adds `cmd/verifier-site`, a thin-main build-time generator that renders the
complete Surface-C static tree (`index.html` from `verifier.Handler` + every `/_ds/` asset from
`web.Handler`, all over httptest) into an output dir — the reproducible build command the Pages deploy
will invoke. The diff is tight (1 production file + 1 test + the one allowed `CLAUDE.md` doc edit +
handoff), all gates are green, the generator copies the byte-pinned `verify.wasm` (hash verified equal to
`web.WasmVerifyHash`), and the fail-closed contract is non-vacuous (reviewer mutation-proven). One
confirmed-but-non-blocking robustness finding (Codex P3): the output is non-atomic, so a mid-run error
leaves a partial tree — filed `low`, does not weaken any gate or block the increment's stated goal.

**Verification:**
- [x] `mise run check` (build + vet + test) — GREEN, all 27 packages ok (incl. `cmd/verifier-site`).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -count=1 -run TestGenerate ./cmd/verifier-site` — PASS.
- [x] `go run ./cmd/verifier-site -out <tmp>` exits 0 and writes the full 14-file tree (index.html + 5
  named `/_ds/` assets + 8 woff2) — reviewer re-ran; `find` confirms the exact tree.
- [x] Generated `_ds/verify.wasm` SHA-256 = `7d57ab1b…f22d2c` == `web.WasmVerifyHash` — reviewer
  `sha256sum`-verified (copy, not rebuild).
- [x] No-CDN re-assertion on the generated `index.html` (`jsdelivr`/`http://`/`https://`/`cdn.` absent) —
  reviewer `grep`-verified on the live output (0 hits).
- [x] Dep closure: only `internal/verifier` + `internal/web`; no `database/sql`/store/logclient/follower;
  `net/http` only via `net/http/httptest` — reviewer `go list -deps`-verified.
- [x] Loader markers present in `index.html` (`/_ds/wasm_exec.js`, `/_ds/verify.wasm`,
  `isccVerifyInclusion`, `URLSearchParams`) — asserted by the test and re-checked on the generated file.
- [x] Fonts enumerated from `fonts.css`, not hardcoded — 8 `"/_ds/fonts/…woff2"` markers, all legit
  `src:` lines, matching the 8 embedded woff2 (mirrors `TestFontsCSSReferencesEmbeddedSubsets`).
- [x] Fail-closed non-vacuous (reviewer mutation): renaming a handler `switch` CASE so `GET
  /_ds/tokens.css` 404s → `generate` aborts → `TestGenerate` FAILS with `…= 404, want 200`; restored →
  green. (Note: renaming the `web.*` CONST does NOT fail — generator + handler path move together; the
  handler-404 probe is the right one, recorded in `learnings/verifier-site.md`.)
- [x] Scope: only `cmd/verifier-site/{main.go,main_test.go}` (new) + `CLAUDE.md` (the one allowed doc) +
  handoff — 1 non-test/doc production file (≤3). No Not-In-Scope path touched (`cmd/wasm`,
  `internal/proof/verify`, `.github/workflows`, `cmd/iscc-monitor`, `internal/verifier/*`, `verify.wasm`
  all untouched); `verifier.Handler` still NOT mounted in `buildMux` — reviewer-confirmed.
- [x] Gate-integrity scan over the 3 unpushed commits (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion in the production diff; working tree clean (no stray
  `wasm`/`verifier-site` binary, no `dist/` left behind).
- [x] Oracle/conformance gate — N/A (pure static HTML/asset assembly from already-golden-tested embedded
  bytes; no signature/RFC-6962/Merkle/did:web/fsck/proof path).

**Issues found:**
- (Codex P3, confirmed → filed `low`) `cmd/verifier-site` `generate` writes non-atomically: `index.html`
  is written before the asset loop, so a later 404/write error returns an error AFTER `index.html` (and
  earlier assets) are on disk, leaving a partial tree in a reused `dist/`. The run-level fail-closed is
  intact (it errors → `os.Exit(1)` → CI/test catches a broken deploy); only the output dir is left
  half-written. Not a current hazard (`TestGenerate` uses `t.TempDir()`; happy path is complete; the
  publish workflow gates on exit code). Fix = stage to a temp dir + rename, or buffer all responses
  before the first write. Does NOT block.

**Codex second opinion:** Codex ran (slow — ~11 min, completed exit 0) and produced ONE finding, [P3]
"Stage files before updating the output tree" at `cmd/verifier-site/main.go:66`. Triage: CONFIRMED real
against the code — the write-before-later-render ordering does leave a partial tree on an error path. But
Codex's framing ("violates the intended fail-closed behavior") is PARTIALLY refuted: fail-closed holds at
the RUN level (the error is surfaced and `os.Exit(1)` fires, so a broken deploy is never silently
published); the gap is only that the OUTPUT directory isn't updated atomically. Filed as `low` (matches
Codex's own P3 tier) — a real robustness refinement, not a blocker, not a gate weakening, and the loop
skips lows.

**Visual check:** n/a — no SSR surface changed. `cmd/verifier-site` is a build-time generator that only
READS the existing `verifier.Handler`/`web.Handler` output; no template or SSR handler was touched, and
the generated `index.html` is byte-identical to `verifier.Handler`'s already-reviewed output.

**Next:** Build the GitHub-Pages publish workflow (`.github/workflows/*`) that runs `go run
./cmd/verifier-site -out <dir>` and deploys the tree to `monitor.iscc.codes` — this generator is its build
command. When that lands, consider folding in the new `low` non-atomic-output fix (stage + rename), plus
the two still-open Surface-C `normal`s (the `readTarget` `u.href` normalization, and the WASM-verifier
honesty gap: the core proves inclusion math only — no checkpoint-signature/did:web-key/id-binding check,
yet the step copy lists a did:web step).

**Notes:**
- The generator deliberately leaves `dist/` out of `.gitignore` (the test uses `t.TempDir()`; the
  deploy-output ignore belongs with the publish workflow). The working tree stayed clean across both my
  `go run` and Codex's `GOOS=js` build (Codex's own stray `verifier-site` binary was its artifact and it
  cleaned it up; nothing from our diff dirties the tree).
- `verify.wasm` reproducibility rides `web.WasmVerifyHash` (the `-buildvcs=false` build) — the generator
  copies the byte-pinned blob (hash equality proven), never rebuilds, so the published artifact hash is
  reproducible from a clean checkout.
- New learnings detail file `learnings/verifier-site.md` created (+ index pointer row) recording the
  one-source-of-truth httptest idiom, the font enumeration, the const-rename-vs-handler-404 fail-closed
  probe nuance, and the non-atomic-output `low`.
- Issue count after this iteration: 0 critical / 10 normal / N low (one `low` added; none resolved — no
  open issue touched `cmd/verifier-site`). CI green expected at this commit (same gate set as `mise run
  check`, no `cauldron/` in tree).
