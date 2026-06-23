## 2026-06-23 — M-API slice 4 — fix the three contract-accuracy defects + pin them with a per-operation golden + ban mermaid

**Done:** Edited `openapi.yaml` to match the real handlers exactly (removed the phantom `verify`
`index` param, changed `/{domain}/log/checkpoint` `200` media type `text/plain`→`application/octet-stream`,
added `/healthz`'s `503`), deterministically regenerated the byte-distinct JSON twin from the edited
YAML, and added `contract_test.go` — a per-operation golden (params + `200` media type + response codes
for the three touched ops, asserted in BOTH YAML and JSON) plus a `mermaid`-fence ban over both doc
bodies. This closes the last open M-API contract-accuracy criterion.

**Files changed:**
- `internal/openapi/openapi.yaml`: 3 edits, each grounded in the handler — `/verify` `index` param
  removed (`serveVerify` reads only `iscc_id`, always `seqs[0]`); `/{domain}/log/checkpoint` `200` →
  `application/octet-stream` + `format: binary` (tilesserve `writeBlob` BLOB shape); `/healthz` `503`
  `{store not ready}` `application/json type: object` added.
- `internal/openapi/openapi.json`: regenerated full-body render of the edited YAML (throwaway in-module
  `yaml.v3`→`encoding/json` converter, `SetEscapeHTML(false)` + `SetIndent("","  ")`; created → run →
  deleted, `git status` clean). The `/inclusion` `index`, `/entries` `index`, and all other ops are
  untouched.
- `internal/openapi/contract_test.go` (new test file): table-driven golden over both artifacts —
  `/verify` has iscc_id+domain but NOT index; `/inclusion` DOES have index (non-vacuity anchor);
  `/checkpoint` `200` is octet-stream not text/plain; `/healthz` has 200+503; plus the `mermaid` ban.

**Verification:** `mise run check` → GREEN (exit 0; `go build`/`go vet`/`go test` all 30 pkgs ok; no
FAIL/error/panic). Per-criterion:
- [x] `go test ./internal/openapi` passes (new golden + unchanged `TestOpenAPIDocsAgree` /
  `TestOpenAPIServesJSONVerbatim` / `TestOpenAPIServesYAMLVerbatim` — twin still byte-served + path-equal).
- [x] `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` passes (path SET unchanged; only
  params/media-types/responses moved, invisible to the path-only drift test).
- [x] `gofmt -l .` empty after `mise run fmt`.
- [x] **Golden is mutation-non-vacuous** — each reverted edit fails its row, confirmed by 5 mutation
  runs: re-add `verify` index → `TestContractVerifyHasNoIndexParam` FAIL; restore checkpoint
  `text/plain` → `TestContractCheckpointMediaType` FAIL; drop `healthz` 503 → `TestContractHealthzHas503`
  FAIL; add `\`\`\`mermaid` fence → `TestNoMermaidInContract` FAIL; drop `inclusion` index →
  `TestContractInclusionHasIndexParam` FAIL. Artifacts restored after each; full suite green.
- [x] `go.mod` / `go.sum` byte-identical to HEAD (no dependency added; `yaml.v3` already present).
- [x] No stray generator committed; the JSON twin reproduces byte-for-byte from the YAML (verified the
  converter regenerates HEAD's twin from HEAD's YAML byte-identically before applying the edits).

**Next:** The 4th and final M-API Verify criterion is now met (`verify` has no `index`, `checkpoint` is
`application/octet-stream`, `healthz` has 200+503, all pinned + mermaid banned). A later `update-state`
can close the umbrella M-API OpenAPI criterion. The remaining open `normal`s are unrelated to M-API: the
M-UI dossier §1/§3 design-parity rework (`critical`s filed by steer `d2f259e`) + masthead-identity
threading into the other 5 surfaces are the most likely next priorities; behind them sit the data-model
issues (DB migration story, `iscc_index.seq` multi-hub global-PK collision) and the WASM signature half +
realm-index Anchor honesty.

**Notes:**
- **JSON twin regeneration is verifiably faithful.** Before applying any edit I restored HEAD's YAML and
  ran the converter — it reproduced HEAD's committed `openapi.json` byte-for-byte, proving the encoder
  settings (alphabetical key sort, 2-space indent, HTML-escaping OFF so `≤`/`—` stay literal, trailing
  newline from `Encode`) exactly match how the prior slice generated the twin. The reviewer can re-confirm
  twin fidelity via `yaml.Unmarshal` both → `reflect.DeepEqual` (the learnings/openapi.md check).
- **The mermaid ban reuses the existing `containsFold` helper** from `openapi_test.go` (same package,
  same file-set), scanning the embedded `YAML`/`JSON` bytes — no new helper, no new import beyond
  `yaml.v3` (already in `openapi_test.go`).
- **Pre-existing `target.md` modification** remains in the working tree from the earlier `cid(steer)`
  commit (`d2f259e`) — NOT mine to touch per the context-file rule; left for the next `update-state`/`steer`.
- **No handler changed** (proofserve/tilesserve/healthz untouched); the defect was purely the doc
  drifting from correct handlers, fixed doc→code as instructed. Oracle/conformance gate is N/A — this
  touches no signature/RFC-6962/Merkle/did:web/fsck/proof path (serves committed bytes + reconciles
  route strings).
