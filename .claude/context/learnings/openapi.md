# Learnings — `internal/openapi` (the hand-authored OpenAPI 3.1 contract)

The machine-surface contract: `openapi.yaml` (hand-authored source of truth) + a byte-distinct
`openapi.json` twin, both `go:embed`-ed and served byte-verbatim at `/openapi.json` + `/openapi.yaml`
(ADR-0014). Pure stdlib + `gopkg.in/yaml.v3` leaf (no `internal/store|metrics|logclient`); `Handler`
mirrors `internal/web`'s `writeAsset` cache shape (no-cache + strong content ETag + 304); CORS rides the
outer `corsmw` wrap (the handler sets none). The route↔spec drift test lives in `cmd/iscc-monitor` (that
is where doc-paths meet the real mux).

## Pitfalls a future implementer hits

- **The drift test gates PATHS only — never params, media types, response codes, or schemas.** It
  reconciles the doc's declared path SET against a HAND-MAINTAINED `machineProbes()` ground-truth map, so
  it catches a renamed/added/removed PATH in either direction (reviewer mutation-confirmed) but is BLIND to
  a wrong query param, a wrong `Content-Type`, or a missing response code. Three such defects shipped in
  slice 1 and passed every gate (filed in issues.md): the `verify` op's phantom `index` param (handler uses
  `seqs[0]`, never reads `index`), `/{domain}/log/checkpoint` advertised `text/plain` (tilesserve serves
  `application/octet-stream`), and `/healthz` missing its 503. When you edit the doc, hand-check each
  operation's params + `200` media type + failure codes against the actual handler — the gate will not.
- **`machineProbes()` is hand-maintained ground truth, NOT derived from the mux.** A genuinely new machine
  route mounted in `buildMux`/`hubHandler` that nobody adds to BOTH the doc and `machineProbes()` passes
  set-equality silently (both miss it). The SSR side (`ssrExclusions()`) probes-and-asserts-mounted, giving
  partial coverage. If you add a machine route, you MUST add it to the doc AND `machineProbes()`.
- **Regenerate the JSON twin from the YAML deterministically after ANY YAML edit.** `TestOpenAPIDocsAgree`
  gates only the path SET, so a one-sided body edit (a param, a description) is NOT caught — the twin
  silently diverges in everything but paths. Reviewer-verified the committed twin is currently a faithful
  FULL-BODY render (decode both via yaml.v3 → `reflect.DeepEqual`), but nothing GATES that, so an edit to
  one artifact alone will rot it. The advance generated the JSON via a throwaway in-module `yaml.v3`
  converter (created, run, deleted — not committed; `git status` confirmed no leak).
- **`/{domain}/log/checkpoint` is a tilesserve route, not proofserve.** Only `checkpoint.ots` is an exact
  proofserve mount in `hubHandler`; plain `checkpoint` falls through the `/` dispatch to tilesserve →
  `writeBlob` → `application/octet-stream`. Document its media type as the BLOB type, not text.
- **`records`/`record` are SSR (HTML) even though proofserve serves them — they belong on the exclusion
  list, NOT in the contract.** proofserve mixes machine (`inclusion`/`consistency`/`entries`/`verify`) and
  HTML (`records`/`record`) routes; "served by proofserve" is not the machine/SSR boundary — the rendered
  shape is.

settled: leaf-purity (only itself in the internal closure), CORS-rides-outer-wrap, no-cache+ETag+304
parity with `internal/web`, and the byte-verbatim serve are all gate-green and mutation-confirmed this
slice (git history holds the proof).
