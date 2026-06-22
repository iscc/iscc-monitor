## 2026-06-22 — Review of: Move the WASM `safeIndex` integer guard into the untagged `verifyadapter` and table-test its reject branches

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance relocated the JS→Go integer guard (`safeIndex`+`maxSafeInteger`) byte-for-byte
out of the build-tagged `cmd/wasm/main.go` into the untagged, WASM-pure `cmd/wasm/verifyadapter` as
exported `SafeIndex`, re-pointed `main.go`'s two call sites, and added a table-driven `TestSafeIndex`
that now runs on every linux `go test` — closing the `normal` issue "the `safeIndex` guard is trapped
in the tagged `main.go` and has NO executable test." Scope is tight (2 production files + 1 test file),
the port is byte-identical to the original logic, all gates are green, and I independently re-ran the
four-branch mutation sweep: each reject branch is caught (including the subtle NaN/Inf branch the
`wantMsg` substring assertion was added to gate). Codex completed clean.

**Verification:**
- [x] `mise run check` (build + vet + test) — GREEN, all 27 packages ok.
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `GOOS=js GOARCH=wasm go build ./cmd/wasm` — exit 0 (the WASM entrypoint still compiles after the move).
- [x] `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` — exit 0 (`verifyadapter` stays WASM-pure;
  the real proof, not a dep-list grep). Direct imports = `encoding/base64`/`fmt`/`proof/verify`/`math`
  only; no `net`/`syscall/js`/`database/sql`.
- [x] `go test -count=1 -run TestSafeIndex ./cmd/wasm/verifyadapter` — PASS (8 subtests).
- [x] `go test -count=1 ./cmd/wasm/verifyadapter` — PASS (existing `TestVerifyJSON` golden parity still
  green; relocation introduced no behavior change).
- [x] Assertions: `SafeIndex(1.9,"index")`→`(0, non-empty)`; `SafeIndex(5,"index")`→`(5,"")`;
  `SafeIndex(float64(1<<53-1),"size")`→`(1<<53-1,"")`; `SafeIndex(float64(1<<53),"size")`→`(0, non-empty)`.
  Verified in the `-v` run.
- [x] **Mutation (reviewer-rerun, the load-bearing check):** reverting each of the four reject branches
  *individually* — NaN/Inf finite-number, fractional `Trunc`, lower-bound `v<0`, upper-bound
  `>maxSafeInteger` — makes `TestSafeIndex` FAIL on the matching subtest. All four caught; file restored
  byte-identical afterward (`git diff` empty). The NaN/Inf non-vacuity (the advance's documented nuance)
  holds because the table pins the `"not a finite number"` message.
- [x] Byte-for-byte port — `git show HEAD~1:cmd/wasm/main.go` `safeIndex` body diffed against the new
  `SafeIndex` body (func name normalized): IDENTICAL. No bound relaxed.
- [x] `safeIndex`/`maxSafeInteger`/`"math"` no longer present in `cmd/wasm/main.go` (grep: NONE).
- [x] Scope: only `cmd/wasm/main.go` + `cmd/wasm/verifyadapter/verify_adapter.go` (2 production) +
  the test file + context. No Not-In-Scope path touched: `internal/proof/verify`, `VerifyJSON`'s
  signature, the golden vector, the `isccVerifyInclusion` arg-count/arg-order, the dossier, the
  Surface-C `readTarget`, the Pages doc note — all untouched (verified in the diff).
- [x] Gate-integrity scan over the 3 unpushed commits (`origin/develop..HEAD`) — no
  `//nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in the `.go` diff. (The
  `//go:build js && wasm` tag on `main.go` is the pre-existing WASM-entrypoint tag, not a gate dodge.)
- [x] Oracle/conformance gate — N/A: no signature/RFC-6962/Merkle/did:web/fsck/proof code touched.
  This is a pure relocation of a marshaling-boundary integer guard; the shared `verify.VerifyInclusion`
  core + its golden vector are unchanged (`TestVerifyJSON` re-confirms parity).

**Issues found:** (none). Deleted the resolved `normal` "the `safeIndex` guard is trapped in the tagged
`main.go` and has NO executable test" — fix verified (moved to untagged `verifyadapter.SafeIndex`,
mutation-proven table test). Also removed a stray root-level `wasm` build artifact I produced running
`go build ./cmd/wasm` (it is not gitignored; tree restored clean — `git status` empty).

**Codex second opinion:** Completed clean (exit 0, ~couple minutes). Verdict: "a pure relocation of the
existing SafeIndex logic into a testable package, with call sites updated and added coverage for the
moved guard. The code builds and existing behavior appears preserved." No `[P1]`–`[P3]` findings.
Agrees with my independent review; nothing to triage.

**Visual check:** n/a — no SSR surface changed. This increment is the WASM marshaling adapter + its
test + the WASM entrypoint's call sites; no template or server-rendered handler was touched.

**Next:** Two WASM fronts remain (both filed `normal`, neither code-closable in one ≤3-file step):
(1) the **cross-origin verifier-scope gap** — `isccVerifyInclusion`/`VerifyJSON` prove only inclusion
math, not checkpoint-signature / did:web-key / id-binding, so a malicious monitor can render a green
`verified` on the cross-origin Surface-C verifier (the most trust-root-meaningful WASM gap; wants a
define-next split into a design pass + implement sub-steps — browser did:web resolution + signature
verify + id-decode); (2) the one-time human repo-Settings step for the Pages custom domain
(`monitor.iscc.codes`) + its deploy-setup doc note. I'd queue the verifier-scope **design** pass — it
is the WASM milestone's actual trust bar — but it is design-first, not a direct build.

**Notes:**
- The guard's `>= 2^53` strictness (rejects `> maxSafeInteger = 2^53-1`) is preserved verbatim from the
  original `main.go`; the table pins `2^53-1` ACCEPTED and `2^53` REJECTED. Well outside any real leaf
  domain — do not relax when the table is next touched.
- Learnings: `learnings/cmd-wasm.md` updated — the "trapped behind the build tag" forward rule is now a
  `settled:` note, plus the durable NaN/Inf non-vacuity trap (a `wantErr`-only table is vacuous on that
  branch; the `wantMsg` message pin is what gates it). Index gist for `cmd/wasm` updated accordingly.
- Issue count after this iteration: 0 critical / 10 normal / 8 low (one `normal` resolved + deleted;
  none added). CI green expected at this commit (`ci.yml` unchanged; the change is inside the existing
  `mise run check` Go gate, which is green).
