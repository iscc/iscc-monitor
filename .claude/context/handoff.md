## 2026-06-22 — Review of: Make `verify.wasm` reproducible — add `-buildvcs=false`, rebuild, re-pin `WasmVerifyHash`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance added the missing `-buildvcs=false` flag to the `mise run build:wasm` task,
rebuilt `internal/web/verify.wasm` deterministically (VCS stamp gone), and re-pinned `WasmVerifyHash`
to the emitted `f03b9b89…`. I independently rebuilt across clean / untracked-dirty / tracked-dirty /
`go clean -cache` tree states — all four produce a byte-identical artifact equal to the committed blob
and the const. This closes the open `critical` reproducibility issue; scope held to the exact three
files `next.md` named.

**Verification:**
- [x] `mise run check` green — all 25 packages `ok` (build + vet + test).
- [x] `go test -count=1 -run TestWasmVerifyHashPinned ./internal/web` — PASS against the rebuilt artifact
  (`sha256(verify.wasm) == WasmVerifyHash`).
- [x] `go test -count=1 -run TestWasmVerify ./internal/web` — PASS (serve route / content-type / ETag /
  304 / 405 unchanged).
- [x] `strings internal/web/verify.wasm | grep -c 'vcs\.'` → `0` (prior committed blob had 3:
  `vcs.revision=b2667f86…`, `vcs.time=…`, `vcs.modified=true`; now none).
- [x] Reproducibility across tree states — `mise run build:wasm` on clean, untracked-dirty,
  tracked-dirty, and after `go clean -cache` all emit `f03b9b89…`; `git diff --stat
  internal/web/verify.wasm` is empty after a fresh rebuild; `git show HEAD:internal/web/verify.wasm`
  hashes to `f03b9b89…`.
- [x] `gofmt -l .` (excl. gitignored `cauldron/`) — empty.
- [x] Oracle/trust-root gate — N/A: diff touches only the build flag + hash const + regenerated binary;
  `cmd/wasm` source and `internal/proof/verify` untouched (confirmed via `git diff --name-only`).
  Functional guard still green: `verifyadapter` parity test PASS, `proof/verify` stays pure +
  WASM-green (`GOOS=js GOARCH=wasm go build`, no net/sql/sqlite in closure). The 174-byte size delta
  (2870559→2870385) is exactly the removed VCS build-info section, not a logic change.
- [x] Gate-integrity scan over all unpushed commits — no `nolint`/`t.Skip`/swallowed errors/build-tag
  exclusions/deleted assertions. Tests were ADDED (`TestWasmVerifyServed`, `TestWasmVerifyHashPinned`),
  never weakened.

**Issues found:** (none) — clean, minimal, on-scope. Removed the resolved `critical` reproducibility
entry from `issues.md` after verifying the fix.

**Codex second opinion:** Clean (exit 0). Codex independently ran `go tool buildid` on the artifact and
verdict: "The build task now disables VCS stamping, the pinned hash matches the committed WASM bytes,
and the documented checks pass. I did not find any new actionable correctness issues in the changed
code." No findings to triage; corroborates the reviewer's own measurements.

**Visual check:** n/a — no SSR surface changed. `internal/web` serves only static `/_ds/...` assets; no
template, no rendered-HTML surface (dashboard/dossier/web template/certificate) was touched this
increment.

**Next:** Honor the human sequencing steer in `target.md` (Titusz, 2026-06-22): now that the WASM pin
is reproducible, front-load **M-UI design-parity** — the remaining open `critical` *"`/` realm index is
far below its authoritative mockup"*. Bring `/` to its mockup's named regions: (a) the claim-lookup hero
as a no-JS `GET` form → `/inclusion/…`, (b) every hub row wrapped in an `<a href>` to its dossier
(restore the realm-index→dossier traversal, anchor count ≥ hub count), and (c) the masthead logo +
instance-identity block + `verify ↗ monitor.iscc.codes` link. The dashboard handler golden test must
assert those landmark regions; an ADR-0012 visual pass against the mockup confirms no headline-region
deviation. The WASM `<script>` caller (and the `cmd/wasm/main.go:39-40` `js.Value.Int()` truncation fix
that belongs to it) resumes after the parity pass, safely on top of a reproducible pin.

**Notes:**
- Toolchain: Go 1.26.1 (the `[tools] go = "1.26"` mise pin). The emitted hash + size match the prior
  reviewer's independent measurement and Codex's `go tool buildid` check exactly.
- The grandparent revision `b2667f8` stamped into the prior dirty blob corresponds to the
  `b2667f8 cid(define-next): build + serve the verifier .wasm reproducibly` commit — confirming the old
  artifact was built before this revision and never re-pinned, the exact failure the issue diagnosed.
- Remaining open issues after this sweep: 1 `critical` (M-UI `/` parity, the Next focus), 5 `normal`
  (two certificate latent fail-opens, the OTS stamp guard, the `hubDomain` ForceQuery gap, the WASM
  shim `Int()` truncation, certificate §6 timestamp), and several `low` (loop-skipped). None block this
  increment.
- Pushing `develop` to `origin` (remote configured, upstream `origin/develop`).
