## 2026-06-23 — Review of: M-API slice 1 — embed + serve OpenAPI 3.1 contract at `/openapi.json` + `/openapi.yaml` with a route↔spec drift test

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance added a clean `internal/openapi` leaf (hand-authored OpenAPI 3.1 YAML +
byte-distinct JSON twin, `go:embed`-served byte-verbatim under the CORS `*` wrap with no-cache+ETag+304),
mounted both exact routes in `buildMux`, reserved their mount names, and gated the lot with a non-vacuous
route↔spec drift test. All `next.md` Verify criteria are met and `mise run check` is green. The contract is
accurate on PATHS, but Codex surfaced three contract-vs-handler mismatches (a phantom `verify` query param,
a wrong `checkpoint` media type, a missing `healthz` 503) that the drift test cannot catch because it gates
paths, not params/media-types/responses — all reviewer-confirmed and filed as issues, none blocking.

**Verification:**
- [x] `mise run check` (build + vet + test, 28 pkgs) — GREEN.
- [x] `gofmt -l .` (outside `cauldron/`) — empty; `main.go`'s function-call map keys are correctly left
  unaligned by gofmt.
- [x] `go test -run TestOpenAPI ./internal/openapi` — PASS (JSON/YAML verbatim+content-type, 405, 404,
  304-echo-ETag, JSON⇔YAML path-set identical, valid 3.1 skeleton, verify-for-me weaker golden).
- [x] `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` — PASS; reviewer mutation-confirmed NON-VACUOUS in
  BOTH directions: (1) dropping `/version` from the YAML → set-equality FAILS; (2) declaring a `ghost` path
  the mux doesn't mount → "has no probe" FAILS. Restored → green.
- [x] `go test -run TestOpenAPIServedVerbatimWithCORS ./cmd/iscc-monitor` — PASS; reviewer re-probed the
  real mux: `/openapi.json` is byte-equal to `openapi.JSON` and carries `Access-Control-Allow-Origin: *`.
- [x] `/openapi.json` parses as `openapi: 3.1.0`, 13 non-empty machine paths.
- [x] `grep -c "openapi" CLAUDE.md` = 1 (>0); the two endpoints documented in the dev-instance list.
- [x] Leaf purity — `go list -deps internal/openapi` shows only itself in the `internal/*` closure (no
  store/metrics/logclient). `go.mod`/`go.sum` byte-identical to HEAD~1.
- [x] Scope discipline — exactly 2 non-test/doc prod files (`main.go`, `openapi.go`); nothing in `## Not In
  Scope` (no `/docs`, no Stoplight asset, no route/handler change). Clean.
- [x] Twin fidelity (reviewer probe, beyond what the gate asserts) — decoded both embedded artifacts via
  yaml.v3 and `reflect.DeepEqual`: the JSON twin is a faithful FULL-BODY render of the YAML, not just the
  path set. (Note: nothing GATES full-body agreement — only the path set — so a future one-sided edit can
  rot it; captured in `learnings/openapi.md`.)
- [x] Gate-circumvention scan over unpushed commits — no `nolint`/`t.Skip`/swallowed-err/build-tag dodge,
  no deleted assertions or tests.

**Issues found:** Three contract-accuracy defects (all Codex-raised, all reviewer-confirmed, none blocking;
filed in issues.md):
- `normal` — `/{domain}/log/verify` declares a phantom `index` query param the handler NEVER reads
  (`serveVerify` always uses `seqs[0]`; its own comment says so). A generated client sends `index` and
  silently gets a verdict for a different leaf. Most client-misleading of the three.
- `normal` — `/{domain}/log/checkpoint` advertises `text/plain`, but tilesserve's `writeBlob` serves
  `application/octet-stream` (reviewer-probed the live mux: `200`, `octet-stream`).
- `low` — `/healthz` documents only `200`; the handler returns `503 {status:unavailable}` on store-down
  (the readiness probe's primary failure mode), which the contract omits.
The umbrella M-API issue was updated to record slices 1+2 as LANDED with slice 3 (`/docs` + Stoplight
Elements) remaining.

**Codex second opinion:** Ran clean (exit 0), three findings, all triaged CONFIRMED REAL against the
handler code/live mux and filed as issues above — none refuted, none blocking. P2 verify-`index`: confirmed
`serveVerify` never reads `index`. P2 checkpoint media type: confirmed `tilesserve.writeBlob` always sets
`octet-stream` (const, line 32) + probed the real mux. P3 healthz 503: confirmed `healthz.Handler` returns
503 on `Ping` failure. Codex correctly found the one class of bug the PATH-only drift test is structurally
blind to — exactly the residual the advance's own handoff Notes flagged.

**Visual check:** n/a — no SSR surface changed. The OpenAPI doc is served as raw JSON/YAML bytes (no
template, no DS shell); the diff touches no `internal/dashboard|dossier|web|certificate` or `.html`.

**Next:** M-API slice 3 — `GET /docs` + the self-hosted, byte-pinned Stoplight Elements assets (the
`<elements-api>` JS + CSS under `/_ds/` with published `internal/web` hash constants next to
`WasmVerifyHash`, same strong-ETag+no-cache+304, `apiDescriptionUrl="/openapi.json"`, no `tryItCorsProxy`),
closing the last two M-API Verify criteria. **Strongly consider folding the two `normal` contract-accuracy
fixes into that slice** (remove the `verify` `index` param + fix the `checkpoint` media type, regenerate the
JSON twin) since it is the natural next doc-touch — and add a golden that pins each documented operation's
params + `200` media type against the handler's real behavior, since the path-only drift test cannot.

**Notes:**
- The drift test's structural blind spot (paths only; `machineProbes()` is hand-maintained, not
  mux-derived) is the load-bearing thing the next OpenAPI editor must know — recorded in the new
  `learnings/openapi.md` detail file (index pointer added). The fix is NOT to weaken the drift test but to
  hand-check params/media-types/responses on every doc edit and consider a per-operation golden.
- Regenerate the JSON twin from the YAML deterministically after ANY YAML edit — `TestOpenAPIDocsAgree`
  gates only the path set, so a one-sided body edit silently rots the twin in everything but paths.
- Pushed to `origin/develop`.
