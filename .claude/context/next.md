# Next Work Package

## Step: M-API slice 4 — fix the three contract-accuracy defects + pin them with a per-operation golden + ban mermaid

## Advances
Closes the **last open M-API Verify criterion** (target.md, M-API):

> "a hand-authored **OpenAPI 3.1** document lives in-repo and is served byte-verbatim … the body is a
> valid OpenAPI 3.1 document covering the machine surface … `GET /docs` … has **no external CDN host in
> the body and makes no external runtime call**".

Slices 1–3 are met (serve + path-only drift test + `/docs` + Stoplight Elements). Slice 4 is the
remaining contract-**accuracy** gap: the path-only drift test (`TestOpenAPIDrift` /
`TestOpenAPIDocsAgree`) is structurally blind to params / media-types / response-codes, so three filed
defects ride along green. This step also closes the filed `normal` issue **"`/docs` Elements bundle can
fetch Mermaid from unpkg"** via a doc-body ban (the served OpenAPI doc is the bundle's only mermaid
trigger). This is the standing code-closable milestone — 0 `critical` open; oracle gate N/A (touches no
crypto/proof path).

## Goal
Make the served OpenAPI document describe the real handler behavior exactly, and add a non-vacuous guard
so a future one-sided edit cannot silently re-introduce the kind of drift the path-only test misses.
After this, a later `update-state` can close the umbrella M-API criterion.

## Scope
- **Modify**:
  - `internal/openapi/openapi.yaml` — the hand-authored source of truth (the 3 edits below).
  - `internal/openapi/openapi.json` — the byte-distinct twin; **regenerate deterministically from the
    edited YAML** (do not hand-edit field-by-field).
- **Create**:
  - `internal/openapi/contract_test.go` — the per-operation golden + the mermaid ban (a test file, not
    counted against the ≤3 non-test/doc budget).
- **Reference** (read before writing):
  - `.claude/context/learnings/openapi.md` — the drift-test-gates-paths-only pitfall, the "regenerate the
    JSON twin deterministically after any YAML edit" rule, and the "checkpoint is a tilesserve BLOB route"
    note.
  - `internal/proofserve/handler.go` — `serveVerify` (≈ lines 571-630; the comment at 619-620 reads
    "verify-for-me takes no index param, so seqs[0] is the deterministic subject") vs `serveInclusion`
    (≈ line 261, which DOES read `index` via `selectSeq`).
  - `internal/tilesserve/handler.go` — `const contentType = "application/octet-stream"` (line 32);
    `serveCheckpoint` → `writeBlob` sets it (line 170).
  - `internal/healthz/handler.go` — line 51: a non-nil store ping → `503` `{"status":"unavailable"}`.
  - `internal/openapi/openapi_test.go` — existing idioms (`containsFold`, `parsePaths`,
    `TestOpenAPIDocsAgree`, `TestVerifyForMeFlaggedWeaker` — the "assert in BOTH JSON and YAML" pattern).

## Three doc edits (each grounded in the real handler)
1. **Remove the phantom `index` query param from `/{domain}/log/verify`** (`openapi.yaml` ≈ lines
   237-243). `serveVerify` reads ONLY `iscc_id` and always uses `seqs[0]`; it never reads `index`.
   **Do NOT touch `/{domain}/log/inclusion`'s `index` param** — `serveInclusion` legitimately reads it via
   `selectSeq` (≈ proofserve line 261). And **do NOT touch `/{domain}/log/entries`'s `index`** — that one
   is a real, required param. Only `verify`'s `index` is phantom.
2. **Fix `/{domain}/log/checkpoint` `200` media type** `text/plain` → `application/octet-stream`
   (`openapi.yaml` ≈ lines 264-270). `/checkpoint` is a **tilesserve** BLOB route; `writeBlob` sets
   `application/octet-stream`. Match the `checkpoint.ots` / `tile` / `entries` BLOB shape already in the
   doc (`type: string`, `format: binary`).
3. **Add `/healthz`'s `503` response** (`openapi.yaml` ≈ lines 45-51). A non-nil store ping returns `503`
   with `{"status":"unavailable"}` (`healthz/handler.go:51`). Add a `"503"` response describing "the store
   is not ready" with an `application/json` `type: object` body, mirroring the `200`.

## Not In Scope
- **Do NOT add a committed JSON-twin generator / build step.** Regenerate the twin with a throwaway
  in-module `yaml.v3` converter (create → run → delete; `git status` must stay clean), exactly as the
  prior slice did. A committed generator is a separate, larger decision.
- **Do NOT change any handler** (`proofserve`, `tilesserve`, `healthz`). The defect is purely the document
  drifting from correct handlers — fix the doc to match the code, never the reverse.
- **Do NOT hand-patch `internal/web/elements.min.js`** to strip its unpkg-mermaid loader — it is a
  byte-pinned vendored artifact (never hand-edit a pinned bundle). The doc-body ban IS the fix.
- **Do NOT promote the drift test to derive `machineProbes()` from the mux**, and do NOT broaden the
  per-operation golden into a full schema validator — pin only params + `200` media type + the documented
  failure codes for the three operations this step touches.
- The other open `normal`s (DB migration, `iscc_index.seq` multi-hub PK, WASM signature half, dossier
  §1/§3, realm-index Anchor honesty) wait for later steps — unrelated to M-API.

## Implementation Notes
- **Regenerate the twin deterministically (learnings/openapi.md):** after editing the YAML, decode it with
  `yaml.v3` and marshal to JSON with stable key order + 2-space indent so the twin stays a faithful
  full-body render of the YAML (the reviewer checks this via `yaml.Unmarshal` both → `reflect.DeepEqual`).
  `TestOpenAPIDocsAgree` only gates the path SET, so the twin's body fidelity is on you — regenerate, do
  not hand-edit.
- **Per-operation golden (`contract_test.go`):** parse the embedded `YAML` (and `JSON`) into a minimal
  struct walking `paths.<path>.get.{parameters, responses}` (yaml.v3 reads JSON too) and assert,
  table-driven, the ground-truth facts this step pins — at minimum:
  - `/{domain}/log/verify` `get.parameters` contains `iscc_id` + `domain` but **NOT** `index`;
  - `/{domain}/log/inclusion` `get.parameters` **does** contain `index` (the legitimate one — proves the
    test distinguishes the two and is non-vacuous, not a blanket "no index anywhere" check);
  - `/{domain}/log/checkpoint` `responses.200.content` key is `application/octet-stream` (not `text/plain`);
  - `/healthz` `responses` has BOTH `200` and `503`.
  Assert each fact holds in BOTH `JSON` and `YAML` (mirroring `TestVerifyForMeFlaggedWeaker`) so the twin
  cannot silently rot. **Make it non-vacuous:** the table must FAIL if any one of the three edits is
  reverted — add a comment naming which revert each row catches.
- **Mermaid ban (`contract_test.go`):** assert neither `JSON` nor `YAML` contains a fenced
  ```` ```mermaid ```` block (scan the embedded bytes, reusing the `containsFold`-style helper). This is the
  durable guard the review filed: the served OpenAPI doc body is the ONLY trigger for the vendored Elements
  bundle's unpkg-mermaid lazy-load, so banning it in the doc closes the latent no-CDN gap. The ban must FAIL
  if a mermaid code block is added to a description; reverting the guard makes it pass.
- **Correctness rule (learnings.md, always-loaded SSR/honesty discipline):** the contract must not assert
  behavior the handler does not have — a phantom param / wrong media type is the doc-surface analogue of
  rendering an un-run verification. Pin to the handler, which is ground truth.
- **`go.mod`/`go.sum` stay byte-identical** — this step adds no dependency (`yaml.v3` is already present).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test ./internal/openapi` passes — the new `contract_test.go` AND the unchanged `TestOpenAPIDocsAgree`
  / `TestOpenAPIServesJSONVerbatim` / `TestOpenAPIServesYAMLVerbatim` (the twin is still byte-served and
  path-equal to the YAML).
- `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` still passes (the path SET is unchanged — only
  params/media-types/responses moved, which the path-only drift test does not see).
- The per-operation golden is **non-vacuous**: reverting any one of the three edits (re-add the `verify`
  `index` param, restore checkpoint `text/plain`, drop the `healthz` 503) makes a `contract_test.go`
  assertion FAIL.
- The mermaid ban is **non-vacuous**: inserting a ```` ```mermaid ```` fence into any description in
  `openapi.yaml` makes the ban FAIL.
- `git status` shows no stray generator file committed; `go.mod` / `go.sum` byte-identical to HEAD.

## Done When
`mise run check` is green, the served `openapi.{yaml,json}` describe `verify` (no `index`), `checkpoint`
(`application/octet-stream`), and `healthz` (`200` + `503`) exactly as the handlers behave, the
per-operation golden + the mermaid ban both pass and are mutation-non-vacuous, and the JSON twin is a
faithful regenerated render of the edited YAML — meeting the 4th and final M-API Verify criterion.
