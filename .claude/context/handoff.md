## 2026-06-22 — Review of: Adopt iscc-lib Go codec + bump the locked toolchain to Go 1.26 (ADR-0011)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance lands ADR-0011 exactly as `next.md` scoped: Go 1.24 → 1.26 across all four
config surfaces, a pinned `github.com/iscc/iscc-lib/packages/go v0.5.0` require, and one load-bearing
tripwire test in `internal/index` that asserts iscc-lib **currently rejects** the golden ISCC-IDv1
`"MAIGHFECJMOPMIAB"` with `"invalid Version"` — the executable migration trigger for iscc/iscc-lib#43.
The production decoder leaf (`iscc.go`) is untouched and stays iscc-lib-free + WASM-buildable. Tight,
in-scope, gate-green; every claim independently re-verified, mutation-proven non-vacuous, and Codex agrees.

**Verification:**
- [x] `mise run check` → green under Go 1.26.1 (build + vet + test, all 21 packages `ok`).
- [x] `go test -count=1 -run TestISCCLib ./internal/index` → PASS (asserts today's Version-1 rejection).
- [x] `go test -count=1 ./internal/index` → PASS (no decoder golden/mutation regression).
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` → OK (production leaf stays WASM-pure).
- [x] `go.mod` has `go 1.26.1` + the iscc-lib require; `grep -c iscc-lib go.sum` = 2; the three
  transitive deps (`zeebo/blake3`, `golang.org/x/text`, `klauspost/cpuid`) all present.
- [x] `mise.toml` shows `go = "1.26"`; `ci.yml` `setup-go` shows `go-version: "1.26"`.
- [x] `grep -rn "iscc-lib" internal/index/iscc.go` → EMPTY (production leaf does not import iscc-lib).
- [x] Dependency boundary re-proven: `go list -test -deps ./internal/index` shows iscc-lib (test
  closure); `go list -deps ./internal/index` and `go list -deps ./...` show it ABSENT from every
  non-test build closure — require is live, WASM artifact stays clean.
- [x] `go mod verify` → all modules verified; `go.sum` is **byte-identical** after a fresh `go mod tidy`
  (no missing/extra entries).
- [x] `gofmt -l .` empty (excluding gitignored `cauldron/`).
- [x] Tripwire non-vacuous (two reviewer mutations): flipping `err == nil`→`err != nil` FAILS and prints
  the exact migration-trigger message; swapping the asserted `"invalid Version"` substring FAILS and
  reveals the real error is the literal `"iscc: invalid Version: 1"`. Both restored; tree clean.
- [x] iscc-lib API confirmed against modcache source: `IsccDecode` strips `ISCC:`, base32-decodes, then
  `decodeHeader` rejects `versionVal > 0` at `codec.go:268` — the tripwire is grounded in ground truth.
- [x] Scope discipline: 5 config files + 1 new test file; the three application-source surfaces
  (`go.mod`/`go.sum`/`mise.toml`) are within the ≤3 budget; `ci.yml`/`Dockerfile` are CI/docs config;
  `iscc.go` untouched per the ADR carve-out. Cumulative unpushed change set is only this work + loop
  context files — no surprise feature code.
- [x] Oracle/conformance gate: N/A — adds a dependency + a tripwire test; touches no signature /
  RFC-6962 / Merkle / proof code. No gate weakened.
- [x] Gate-circumvention scan over unpushed commits: no `//nolint`, `t.Skip`, build-tag exclusion, or
  swallowed error in added code (the only `nolint`/`t.Skip` matches are inside handoff prose).

**Issues found:** (none new). Resolved + deleted the open `normal` issue
("Adopt the iscc-lib Go codec + bump the toolchain to Go 1.26 (ADR-0011)") — fully verified and
mutation-proven, closing the TARGET/CODE stack gap.

**Codex second opinion:** Clean — "The Go toolchain bump, test-only iscc-lib dependency, and tripwire
test are consistent with the stated migration plan. Build, vet, tests, and the WASM build for
internal/index pass." No findings to triage. (Codex ran ~3.5 min, exit 0; verdict from
`/tmp/codex-review.txt`.) Corroborates my own verification.

**Visual check:** n/a — no SSR surface changed. The diff touches only `go.mod`/`go.sum`/`mise.toml`/
`ci.yml`/`Dockerfile` + one `internal/index` test file; no template, dashboard, dossier, web, or
certificate code.

**Next:** ADR-0011 is closed and the interim `internal/index.Decode` port stays by the carve-out
(tripwire now watches iscc/iscc-lib#43). The unblocked M-UI tail resumes: the OTS store seam
(unblocking certificate §5 BITCOIN ANCHOR + the `ots` proof-bundle member) or the WASM in-browser
re-verifier (`monitor.iscc.codes`). Still-open `normal` issues queued: the `host:port` `did:web:`
mis-render on §4 + `serveBundle`'s `bundle.Hub.DID` (fix both DID sites together, `%3A`-encode the
port); the §6 `· at` per-record timestamp (needs an `iscc_index` schema column); the Hub-List
`ForceQuery` fail-open guard.

**Notes:**
- The context snapshot at review start showed a stale HEAD (`5fb2454`); the true advance commit is
  `ac9fbfa`. Reconciled: HEAD is the advance commit and the cumulative unpushed set is the ADR-0011
  work plus the loop's context files only.
- CI's `actions/setup-go@v5` with `go-version: "1.26"` resolves the latest 1.26 patch (>= the 1.26.1
  floor iscc-lib's `go.mod` declares), so CI satisfies the patch-pin without an explicit `toolchain`
  line; the fresh-checkout module cache will download the three pure-Go transitive deps cleanly under
  `CGO_ENABLED=0`. Watch the first CI run at the new HEAD for the 1.26 toolchain provisioning.
- `learnings/index.md` gained one durable package-local bullet (the ADR-0011 tripwire location + the
  literal `"iscc: invalid Version: 1"` ground truth + the WASM/`go list -deps` purity guard); file
  stays within budget (~46 lines).
