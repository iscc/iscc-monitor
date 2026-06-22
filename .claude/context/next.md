# Next Work Package

## Step: Wire the first SSR WASM `<script>` caller — light up the certificate's tier-2 "your browser verified" result against `/_ds/verify.wasm`

## Advances
Target → **WASM verifier upgrade** milestone, Verify criterion:
> "identical vectors yield identical verdicts (WASM vs server)" — and the M-UI two-tier honesty
> requirement "the tier-2 result itself lands in the WASM milestone".

This is the **front of the queue** per `state.md` ("the WASM `<script>` verifier caller is now the
front of the queue") and the `review` handoff ("resume the WASM `<script>` caller and the
`cmd/wasm/main.go:39-40` `js.Value.Int()` truncation fix"). The `.wasm` artifact is already built
reproducibly and served byte-pinned at `/_ds/verify.wasm`; the milestone Verify stays 1/1 open ONLY
because no SSR surface yet calls it. This step closes the "first real caller" sub-step of that
criterion as a **verifiable skeleton** (remaining sub-steps in `## Not In Scope`).

It also closes the open `normal` issue **"WASM shim `js.Value.Int()` truncates a non-integer JS
`index`/`size`"** — the issue text says to fix it "when the FIRST real caller is wired (the natural
place to enforce the JS→Go arg contract)", which is exactly this step.

## Goal
Make the certificate page's tier-2 promise real: a progressive-enhancement `<script>` loads the
served `/_ds/wasm_exec.js` + `/_ds/verify.wasm`, calls `globalThis.isccVerifyInclusion` over the proof
data already embedded in the page (record, root, proof hashes, position, size), and renders an honest
in-browser verdict region — without breaking the no-JS baseline (§3 still renders server-side). This
is the first wiring that proves WASM-vs-server verdict parity end-to-end in a browser, and the natural
place to harden the JS→Go integer arg contract.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc source files):
  1. `internal/certificate/handler.go` — add a `RecordB64 string` field to `certData` and populate it
     from the already-computed `arts.record` (base64-Std) on the §3 success path (the only data the
     WASM caller needs that the template does not yet expose). Set it inside the same `if ok` block that
     sets `arts.record`/`data.ProofHashes` (~`handler.go:814-828`), gated by the §3 re-verification, so
     it is empty on every honest decline (read only under `{{if .HasBundle}}`).
  2. `internal/certificate/cert.html` — add the tier-2 in-browser verifier: a result `<div>` (rendered
     only under `{{if .HasBundle}}`, with no-JS default text the script replaces) plus a
     `<script src="/_ds/wasm_exec.js">` + an inline `<script>` that instantiates `/_ds/verify.wasm` and
     calls `isccVerifyInclusion(record, root, [proofHashes...], position, size)`, then writes the
     verdict into the region. Pass the Go values into JS via a `<script type="application/json">` data
     island read with `JSON.parse` — NOT string-interpolated into executable JS.
  3. `cmd/wasm/main.go` — fold the open `normal` `js.Value.Int()` truncation fix in at this first
     caller: before calling `verifyadapter.VerifyJSON`, validate that `args[3]`/`args[4]` (index/size)
     are integral and within the JS safe-integer range; on a violation return the same
     `{verified:false, error:...}` map the arg-count guard uses. (This file is `//go:build js && wasm`,
     so `go build ./...` skips it on linux — but `mise run build:wasm` rebuilds it; see Verification.)
- **Modify (docs — keep in sync):**
  - `CLAUDE.md` — extend the `GET /inclusion/<iscc_id>` bullet (lines 63-67) to note the tier-2
    in-browser re-verification result now renders for a certifiable id via the
    `/_ds/wasm_exec.js` + `/_ds/verify.wasm` progressive-enhancement loader (no-JS baseline unchanged).
- **Reference**:
  - `.claude/context/learnings/cmd-wasm.md` — the `js.Value.Int()` truncation gotcha + the exact JS
    arg order (`record, root, proofArray, index, size`) + the defensive arg-count-guard pattern to
    mirror. **Read before editing `cmd/wasm/main.go`.**
  - `.claude/context/learnings/certificate.md` — §3 re-verification gate (`HasClause3`/`HasBundle`),
    `arts.record`/`builtProof`, base64-Std cross-surface, and the `html/template` `+`→`&#43;` text-node
    entity-escaping trap (matters for the base64 JSON data island). **Read before editing.**
  - `.claude/context/learnings/web.md` — `noExternalCDN` bans only third-party origins; same-origin
    `/_ds/...` script/wasm refs pass; `wasm_exec.js`/`verify.wasm` are served at `WasmExecPath` /
    `WasmVerifyPath`. **Read before editing `cert.html`.**
  - `cmd/wasm/verifyadapter/verifyadapter.go` (+ its `_test.go`) — the verdict contract the shim must
    preserve and the golden vector that proves WASM-vs-server parity.
  - `internal/certificate/cert.html` (existing honesty/actions region, ~lines 424-432) — where the
    tier-2 result region and scripts attach.
  - `mise.toml` (`[tasks."build:wasm"]`, line 39-42) — the `-buildvcs=false` rebuild command.
  - `internal/web/web.go` (`WasmVerifyHash`, line 78) — the SRI pin to re-set if the rebuilt wasm hash
    moves.

## Not In Scope
- The standalone `monitor.iscc.codes` **Independent Verification** app (Surface C,
  `ISCC Monitor - Independent Verification.dc.html`, monitor-agnostic via `?monitor=<url>`) — a
  separate later sub-step of the WASM milestone.
- The guided **split-view alert** on a `(size, root)` mismatch — a later WASM sub-step; this step only
  renders verified / failed / error for the certificate's own embedded proof.
- Wiring the WASM caller into the **dossier** or any other SSR surface — certificate first; dossier is
  a later sub-step.
- Changing `verifyadapter.VerifyJSON`, `internal/proof/verify`, or any signature / RFC-6962 / Merkle /
  proof-bundle crypto — the adapter and core are already parity-proven; only the `cmd/wasm/main.go`
  arg-marshaling guard changes.
- Carrying the named-region + `←` back-link parity pass to the remaining SSR surfaces — deferred.

## Implementation Notes
- **No-JS baseline is the hard constraint (target.md M-UI "complete with JavaScript disabled").** §1–§6
  and the existing honesty/actions copy MUST stay rendered server-side. The tier-2 result region's
  default (no-JS) content must read honestly ("Re-verify the downloadable bundle yourself"), and the
  script only ENHANCES it — never gate any clause behind a `<script>`. The existing no-CDN body
  assertions must still pass.
- **Pass Go→JS data via a `<script type="application/json">` data island + `JSON.parse`, never by
  interpolating Go values into executable JS.** `html/template` does not contextually escape inside a
  `<script>` the same way it does an attribute, and base64 `+`/`/` plus the `ISCC:`-prefixed id are
  exactly the chars that break naive interpolation (the certificate.md `+`→`&#43;` trap). A JSON island
  holding `{{.RecordB64}}`, `{{.CheckpointRoot}}`, the `{{range .ProofHashes}}` array, `{{.Position}}`,
  `{{.CheckpointSize}}` is the safe channel — `html/template` JSON-context-escapes it. Parse it, then
  call `isccVerifyInclusion`.
- **JS arg order is `(record, root, proofArray, index, size)`** (cmd-wasm.md): record + root are
  base64-Std strings, proofArray is a JS array of base64-Std strings, index = `Position`, size =
  `CheckpointSize`. The returned JS object is `{verified: bool, error: string}` — render: `error != ""`
  → show the error; else `verified` → "✓ your browser re-verified this proof against the accepted
  root"; else → "✗ in-browser verification did not match" (a negative VERDICT, not an error — keep that
  distinction visible; it is what the later split-view alert keys on).
- **Loader shape:** `<script src="/_ds/wasm_exec.js"></script>` then an inline script that does
  `const go = new Go();` and `WebAssembly.instantiateStreaming(fetch("/_ds/verify.wasm"),
  go.importObject)` (ours serves `application/wasm`, so streaming works; add the
  `instantiate(await (await fetch).arrayBuffer())` fallback for robustness), `go.run(inst)`, then call
  the now-registered global. Wrap in `try/catch`; on any load failure leave the no-JS default text
  (graceful degradation). A plain inline script at end-of-body is acceptable for this skeleton.
- **`cmd/wasm/main.go` integer guard (closes the `normal` issue).** Before `index := …`, read
  `fIdx := args[3].Float()`, `fSize := args[4].Float()`; reject when not integral
  (`fIdx != math.Trunc(fIdx)` / same for size) or out of safe range (`< 0` or `> 1<<53`), returning the
  `{verified:false, error:"…"}` map. Then `uint64(fIdx)` / `uint64(fSize)`. Add `import "math"`. Per
  cmd-wasm.md, `js.Value.Int()` is `int(v.Float())` and silently truncates — this is the documented fix
  point.
- **Correctness rule (always-loaded learnings):** "On a self-verifiable surface, gate a rendered
  ✓/Merkle assertion on a re-VERIFICATION, not a status flag." The tier-2 ✓ here is genuinely
  re-verified — it runs `proof.VerifyInclusion` in WASM over the embedded proof — exactly the rule
  honored; the server-side §3 ✓ already gates on `HasClause3` re-verification, so the page carries two
  independent re-verifications that must agree. Do not weaken either.
- **Reproducible build / pin discipline (web.md).** The `cmd/wasm/main.go` change recompiles
  `verify.wasm`, so its SHA-256 WILL change. Run `mise run build:wasm` (it carries `-buildvcs=false`),
  then update `WasmVerifyHash` in `internal/web/web.go` to the new lowercase-hex SHA-256 and commit the
  rebuilt `internal/web/verify.wasm`. `TestWasmVerifyHashPinned` FAILS until the pin matches — that is
  the guard working, not a regression. (The wasm artifact + its pin are one indivisible change moving
  together, not a fourth feature file.)

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` excluding
  `cauldron/` empty).
- `mise run build:wasm` produces `internal/web/verify.wasm` whose SHA-256 equals the updated
  `web.WasmVerifyHash` — `go test -count=1 -run TestWasmVerifyHashPinned ./internal/web` passes.
- `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./cmd/wasm` succeeds with the new integer guard (the WASM
  build gate; `go build ./...` correctly skips the tagged package on linux).
- `go test -count=1 -run TestVerifyJSON ./cmd/wasm/verifyadapter` passes — the WASM-vs-server golden
  parity check still holds (the parity oracle for this milestone).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes, including a NEW test
  (`TestCertificateRendersWasmVerifier` or similar) asserting the served certifiable-id HTML carries
  `/_ds/wasm_exec.js`, `/_ds/verify.wasm`, the JSON data island with the base64 record/root/proof, and
  the tier-2 result region — AND that the non-certifiable / `!HasBundle` body does NOT (no fabricated
  verifier on an uncertifiable id). Mutation: removing the `RecordB64` field or the script block makes
  the new test FAIL.
- `go test -count=1 -run TestCertificateNoCDN ./internal/certificate` (the existing no-CDN test) still
  passes — the same-origin `/_ds/` script + wasm refs do not trip the third-party-CDN ban
  (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`); `monitor.iscc.codes` stays the one permitted external origin.
- Assertion: the no-JS baseline holds — with scripting removed, the served certifiable-id HTML still
  contains the §3 clause markup and the honesty/actions region verbatim (no §1–§6 clause is inside a
  `<script>` / script-gated).

## Done When
`mise run check` is green, `mise run build:wasm` reproduces the re-pinned `verify.wasm`, the new
certificate test proves the tier-2 WASM loader + JSON data island render for a certifiable id (and are
absent for an uncertifiable one) without breaking the no-JS/no-CDN baseline, and the WASM shim rejects
a non-integer `index`/`size` — wiring the first real SSR caller of the in-browser verifier and closing
the `js.Value.Int()` truncation issue.
