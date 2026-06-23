## 2026-06-23 — Review of: M-API slice 4 — fix the three contract-accuracy defects + pin them with a per-operation golden + ban mermaid

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance made the served OpenAPI doc match the real handlers exactly — removed the phantom
`verify` `index` param, fixed `/{domain}/log/checkpoint`'s `200` media type `text/plain`→`application/octet-stream`,
and added `/healthz`'s `503` — regenerated a faithful JSON twin, and added a per-operation golden
(`contract_test.go`) plus a mermaid-fence ban. The work is clean, exactly in-scope (1 doc + 1 regenerated
twin + 1 new test file; ≤3 non-test/doc budget honored — only the YAML counts), and every gate is green.
PASS_WITH_NOTES (not PASS) only because Codex surfaced a real, narrow, latent bypass in the new mermaid
guard (substring-only — misses tilde/whitespace CommonMark fences), confirmed by reviewer probe and filed
`low`. The slice's own four criteria are fully met and mutation-proven.

**Verification:**
- [x] `mise run check` (build + vet + test, 30 pkgs) — GREEN.
- [x] `gofmt -l .` — empty (clean).
- [x] `go test ./internal/openapi` — PASS (5 new contract tests + unchanged `TestOpenAPIDocsAgree` /
  `TestOpenAPIServesJSONVerbatim` / `TestOpenAPIServesYAMLVerbatim` — twin still byte-served + path-equal).
- [x] `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` — PASS (path SET unchanged; the params/media-types/
  responses moved are invisible to the path-only drift test, as expected).
- [x] **All three doc edits confirmed against ground-truth handlers:** `serveVerify`
  (`proofserve/handler.go:583`) reads only `iscc_id` + always `seqs[0]`, never `index` — phantom param
  correctly removed; `serveCheckpoint`→`writeBlob` sets `Content-Type: application/octet-stream`
  (`tilesserve/handler.go:32,170`) — media type correctly fixed; `healthz.Handler` returns `503` +
  `{"status":"unavailable"}` `application/json` (`healthz/handler.go:50-55`) — `503` correctly added.
  `serveInclusion` legitimately reads `index` via `selectSeq` (`proofserve/handler.go:261`) — correctly
  left intact (the non-vacuity anchor).
- [x] **JSON twin is a faithful render of the YAML** — reviewer decoded both via yaml.v3 →
  `reflect.DeepEqual` == OK (the body fidelity `TestOpenAPIDocsAgree` does NOT gate).
- [x] **Golden is mutation-non-vacuous — independently re-verified all 5 reverts:** re-add `verify` index →
  `TestContractVerifyHasNoIndexParam` FAIL; restore checkpoint `text/plain` → `TestContractCheckpointMediaType`
  FAIL; drop `healthz` 503 → `TestContractHealthzHas503` FAIL; drop `inclusion` index →
  `TestContractInclusionHasIndexParam` FAIL; add a ` ```mermaid ` fence → `TestNoMermaidInContract` FAIL.
  Each reverted; full suite green; openapi files pristine at HEAD afterward.
- [x] `go.mod` / `go.sum` byte-identical to HEAD; `git status` shows no stray generator committed (only the
  pre-existing `target.md` steer-artifact mod, which the advance correctly did NOT touch).
- [x] Gate-circumvention scan over the 3 unpushed commits — no `nolint`/`t.Skip`/swallowed-err/build-tag
  dodge, no deleted tests/assertions; the source diff is purely additive.

**Issues found:** One new `low` (Codex-confirmed by reviewer probe): the new `TestNoMermaidInContract` ban is
a substring check (`containsFold "```mermaid"`), so it catches only the canonical adjacent-backtick fence —
NOT CommonMark-equivalent forms that still trigger Elements' unpkg-mermaid load (tilde `~~~mermaid`, or
whitespace-after-fence ` ``` mermaid `). Reviewer-probed both: the ban stays green. Latent (the doc has zero
mermaid today), narrower than what the slice delivered, same class as the existing `noExternalCDN`-whitespace
/ `.dockerignore`-slashless lows. Filed `low`. The prior `normal` "`/docs` Elements bundle can fetch Mermaid
from unpkg" issue is RESOLVED and deleted — its requested durable guard now exists and is mutation-proven for
the realistic form; this `low` is the residual.

**Codex second opinion:** Returned exactly one finding, `[P2]` — "Broaden the Mermaid fence guard"
(`contract_test.go:193`): the substring ban misses `~~~mermaid` and whitespace-after-fence forms.
**CONFIRMED** by reviewer probe (both forms pass the ban green while still being a no-CDN trigger). Triaged as
a real but `low` residual (not a blocker): the slice's stated guard is delivered + non-vacuous for the
canonical form, the doc has no mermaid of any form, and exploitation needs a future author to write a
non-canonical CommonMark fence. Filed `low`; not applied here (review is read-only — a confirmed defect
becomes an issue for a later advance). No other findings. (Codex ran ~9 min, grinding the diff + go.mod.)

**Visual check:** n/a — no SSR surface changed. This slice touches only `internal/openapi/openapi.{yaml,json}`
(the machine contract, served byte-verbatim) and a new test file; no server-rendered HTML template was edited.

**Next:** The 4th and final M-API contract-accuracy criterion is now met — a later `update-state` can close
the umbrella M-API OpenAPI criterion. The most likely next priorities are the `critical` M-UI dossier §1/§3
design-parity rework filed by steer `d2f259e` (the dossier §3 size/time decouple + §1 unconditional
"resolved" `normal`s pair with it), then masthead-identity threading into the remaining SSR surfaces, then the
standing data-model `normal`s (DB migration story, `iscc_index.seq` multi-hub global-PK collision) and the
WASM signature half + realm-index Anchor honesty.

**Notes:**
- The pre-existing `.claude/context/target.md` working-tree modification (from steer `d2f259e`) is still
  uncommitted — NOT review's to commit (must not modify target.md); left for the next `update-state`/`steer`.
- Oracle/conformance gate is N/A — this slice touches no signature/RFC-6962/Merkle/did:web/fsck/proof path;
  it reconciles route strings + serves committed doc bytes.
- The `containsFold` helper the mermaid ban reuses is in `openapi_test.go` (same package, test-only); no new
  helper, no new non-test import — verified.
