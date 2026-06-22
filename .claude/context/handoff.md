## 2026-06-22 — Review of: WASM verifier entrypoint — `cmd/wasm` exporting `verify.VerifyInclusion` to JS

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance lays the `GOOS=js GOARCH=wasm` WASM verifier entrypoint exactly as the milestone
arc requires: a tagged `cmd/wasm/main.go` `syscall/js` shim exposing `globalThis.isccVerifyInclusion`,
plus a pure, linux-testable `cmd/wasm/verifyadapter` package that base64-Std-decodes the proof-bundle
fields into the shared `internal/proof/verify` core and folds the three-way verdict into `(verified,
errMsg)`. All gates are green, the adapter is WASM-pure and its golden-vector parity test is
mutation-proven non-vacuous. One Codex P2 (a real but currently-unreachable JS-number truncation gotcha
in the untagged shim) is confirmed and filed as a `normal` issue for the sub-step that wires the first
caller; it does not block this increment.

**Verification:**
- [x] `mise run check` (build + vet + test) — green; all 24 packages `ok`, including
  `cmd/wasm/verifyadapter`.
- [x] `gofmt -l .` (excl. `cauldron/`) — empty.
- [x] `GOOS=js GOARCH=wasm go build -o /tmp/iscc-verify.wasm ./cmd/wasm` — exit 0 (2.9 MB artifact, not
  committed, per Not-In-Scope).
- [x] `go vet ./cmd/wasm` on linux — exit 1 "build constraints exclude all Go files". This is the
  EXPECTED response to a platform-empty package (it is NOT a `syscall/js` compile error); next.md's exact
  wording ("does NOT error on the syscall/js import") assumed the adapter lived in `package main`, which
  it no longer does (see deviation below). The gate uses `go vet ./...` (wildcard) which skips it; check
  is green.
- [x] `go test -count=1 -run TestVerifyJSON ./cmd/wasm/verifyadapter` — PASS, all 5 cases (positive;
  wrong-record + tampered-root → `verified=false, errMsg==""`; malformed base64 + `index>=size` →
  `errMsg!=""`). Run via the subpackage path, not `./cmd/wasm`.
- [x] Golden-vector WASM-vs-server parity: `VerifyJSON("bGVhZi0x", goldenRoot, goldenProof, 1, 4)` →
  `verified=true` — the SAME verdict the core test pins for the SAME 4-leaf vector; literals copied
  verbatim from `internal/proof/verify/verify_test.go`.
- [x] Mutation-proven non-vacuous: forcing `verified=true` fails `tampered_root`/`wrong_record`;
  collapsing the error channel fails `index>=size`; reverted byte-identical, green.
- [x] Adapter WASM-pure: imports only `encoding/base64`, `fmt`, `internal/proof/verify`;
  `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` exit 0.
- [x] `go mod tidy -diff` clean; `go.mod`/`go.sum` untouched by the commit (`syscall/js` is stdlib).
- [x] Gate-integrity scan over unpushed commits — no `//nolint`, `t.Skip`, build-tag exclusions (beyond
  the legitimate `js && wasm` tag), swallowed errors, or deleted assertions.
- [x] Oracle gate — N/A as a *modification*: the diff does not touch `internal/proof/verify`, signature/
  Merkle/consistency logic, `internal/didweb`, or fork/shrink/equivocation; it only *consumes* the
  verify core. The relevant oracle (the pinned 4-leaf golden vector, identical on both sides) is green
  inside `mise run check`.

**Issues found:** One filed (Codex-sourced, see below). No reviewer-independent defects beyond it.

**Codex second opinion:** One [P2] finding, reviewer-CONFIRMED and filed as a `normal` issue.
- [P2] `cmd/wasm/main.go:39-40` — `js.Value.Int()` truncates a non-integer JS `index`/`size` (it is
  `int(v.Float())`, so `1.9 → 1`), so malformed bundle metadata could read as `verified` against the
  truncated leaf. Independently confirmed (`int(1.9)==1`). Real, but (a) lives only in the untagged glue
  shim — the tested, parity-proven `verifyadapter.VerifyJSON` takes `uint64` and is correct; (b) there is
  no caller yet (the wiring is explicitly Not-In-Scope), and the real callers emit server-computed
  integer indexes. So it does not block this increment's goal. Filed for the sub-step that wires the
  first caller (where the JS→Go arg contract belongs) rather than fixed in review (it adds a behavioral
  validation contract, beyond a minor review fix).

**Visual check:** n/a — no SSR surface changed (the diff is entirely under `cmd/wasm/`; no
`internal/dashboard`/`dossier`/`web`/`certificate` or template touched).

**Design deviation (reviewer-VERIFIED, accepted):** advance moved the pure adapter out of `package main`
(next.md's literal layout) into a non-main subpackage `cmd/wasm/verifyadapter`. Independently reproduced
in a scratch module AND this repo: an untagged file in `package main` whose only `func main()` is in a
`js && wasm`-tagged file makes `go build ./...` FAIL on linux with `runtime.main_main·f: function main is
undeclared in the main package`. next.md's scoping claim that `go build ./...` "silently skips" such a
package is incorrect — only a package with *no* Go files for the platform is skipped. The subpackage fix
is the idiomatic, minimal, in-scope correction (3 new files, all under `cmd/wasm/`); all other next.md
intent (untagged+linux-tested marshaling, exported verify-to-JS, same golden vector, three-way verdict
mapping) is preserved. Captured in the new `learnings/cmd-wasm.md` so the next WASM step does not retrip
it.

**Next:** The WASM side is now callable. The natural next sub-step is the **tier-2 progressive
enhancement** wiring it into the certificate/dossier (embed `wasm_exec.js` from
`$(go env GOROOT)/lib/wasm/wasm_exec.js`, a `<script>` that loads the `.wasm` and calls
`isccVerifyInclusion` with the base64-Std fields the surface already emits) — and that is the right place
to land the `js.Value.Int()` integer/safe-integer validation the Codex P2 issue tracks, since it wires
the first real caller. After that: the standalone `monitor.iscc.codes` Independent Verification app
(Surface C) and the reproducible-build / published-hash / SRI pin + `mise run build:wasm` task. A
`VerifyConsistency` sibling export still waits for a caller and needs its own arg-order wrapper
(proof-verify learning).

**Notes:**
- Scope discipline clean: 3 new files (1 non-test `verify_adapter.go`, 1 tagged shim, 1 test), all under
  `cmd/wasm/`. Nothing in `## Not In Scope` was touched — no HTML/DS/`internal/web`, no committed `.wasm`,
  no `mise` WASM task, no `VerifyConsistency`, no fold-in of the standing `normal`/`low` hardening defects.
- `learnings.md` index gained one pointer row (`cmd/wasm`); new `learnings/cmd-wasm.md` detail file
  created. No promotion to the always-loaded section — these are package-local mechanics, not
  cross-cutting. No detail file exceeded the rotation budget.
- Standing OTS / registry / certificate `normal`/`low` issues remain open and untouched (correctly out of
  scope for this WASM step).
