# Next Work Package

## Step: M-API slice 1 — embed + serve the OpenAPI 3.1 contract at `/openapi.json` + `/openapi.yaml`, with a route↔spec drift test

## Advances
The new **M-API** milestone (ADR-0014), `target.md` §"M-API — OpenAPI contract + hosted interactive
API docs", which is **0/4 Verify met** (state.md: "No `/openapi.json` / `/openapi.yaml`, no `/docs`, no
OpenAPI document … This is the immediate standing code-closable milestone"). This step closes the first
two of its four Verify criteria:

> - a hand-authored **OpenAPI 3.1** document lives in-repo and is served byte-verbatim (`go:embed`) at
>   `GET /openapi.json` **and** `GET /openapi.yaml`, each returning `200` with `Access-Control-Allow-Origin:
>   *`; the body is a valid OpenAPI 3.1 document covering the machine surface … and **excludes** the HTML
>   SSR surfaces; the `verify-for-me` path's `description` flags it as the weaker, non-authoritative path …
> - a **drift test** asserts every path the document declares is mounted in the real mux **and** every
>   machine-consumable route the mux mounts is declared (HTML SSR routes on an explicit exclusion list), so
>   adding/renaming a JSON route without updating the document — or documenting a removed route — **FAILS**
>   the gate (reverting the spec/route alignment makes it FAIL)

M-API is **order-independent** with no human/infra/oracle dependency — the standing code-closable work now
that the lone `critical` is closed (state.md: "0 `critical` issues remain"). The handoff `**Next:**` names
exactly this slice ("the order-independent M-API slice (ADR-0014, OpenAPI + Stoplight Elements)").

## Goal
Make the monitor's machine-consumable HTTP surface discoverable and self-describing: ship the in-repo
OpenAPI 3.1 document, serve it byte-verbatim at `/openapi.json` + `/openapi.yaml` under the existing CORS
`*` wrap, and gate it with a drift test so the spec can never silently diverge from the real mux. This is
the verifiable skeleton M-API's later `/docs` slice (Stoplight Elements) renders.

## Scope
- **Create**:
  - `internal/openapi/openapi.yaml` — the hand-authored OpenAPI 3.1 source-of-truth document.
  - `internal/openapi/openapi.json` — the SAME document as JSON (a byte-distinct artifact, semantically
    identical). Author the YAML first; the two MUST describe the same paths (a test asserts their declared
    path-set is equal). Both committed and embedded.
  - `internal/openapi/openapi.go` — `package openapi`: `//go:embed openapi.json` + `//go:embed openapi.yaml`
    into two `[]byte` vars; exported `JSON []byte` / `YAML []byte`; an exported `DeclaredPaths() []string`
    (parse the `paths` key from the embedded YAML once via `gopkg.in/yaml.v3`) for the drift test to
    consume; and `Handler() http.Handler` serving `GET /openapi.json` (`application/json`) +
    `GET /openapi.yaml` (`application/yaml`) byte-verbatim, GET-only (405 else), 404 on any other path —
    mirror `internal/web`'s `writeAsset` cache shape (`no-cache` + strong content ETag + `If-None-Match`
    → 304). Keep this a pure stdlib + `gopkg.in/yaml.v3` leaf (yaml.v3 is already a dep).
- **Modify**:
  - `cmd/iscc-monitor/main.go` — in `buildMux`, mount `mux.Handle("/openapi.json", openapi.Handler())` and
    `mux.Handle("/openapi.yaml", openapi.Handler())` (exact paths, like `/metrics` / `/version`); add the
    two names to `reservedMountNames` so a realm domain literally named `openapi.json`/`openapi.yaml`
    cannot collide (defense-in-depth, matching the existing `metrics`/`healthz`/`version` pattern). Extend
    `buildMux`'s docstring to list the new routes. (1 non-test file.)
  - `CLAUDE.md` — add the two new endpoints (`GET /openapi.json`, `GET /openapi.yaml`) to the endpoint
    list in the "Running a local dev instance" section, noting they describe the machine surface only
    (HTML SSR surfaces excluded) and that `verify-for-me` is flagged weaker in-band. (Doc sync — required
    by the rules when usage docs change; doc files don't count against the ≤3 budget.)
- **Reference**:
  - `.claude/adr/0014-openapi-contract-and-hosted-api-docs.md` — the authoritative decision (Decision §1–§3
    define exactly which routes the doc covers, the byte-verbatim serve, and the drift-test guarantee).
  - `target.md` §"M-API" Verify block — the exact route list the doc must cover and exclude.
  - `cmd/iscc-monitor/main.go` `buildMux` (lines 288–297) + `hubHandler` (lines 418–437) — the GROUND TRUTH
    route set the drift test reconciles against (the machine routes vs the SSR exclusion list).
  - `internal/web/web.go` `writeAsset` (lines 228–253) — the exact `no-cache` + strong-ETag + 304 cache
    shape to mirror in `openapi.Handler`.
  - `.claude/context/learnings/web.md` — the `go:embed` leaf idiom + the no-CDN/asset-pin discipline this
    package joins; `.claude/context/learnings/http-surface.md` §CORS — `corsmw` already wraps the whole mux
    once, so the new handler sets NO CORS headers of its own (the `*` rides the outer wrap).

## Not In Scope
- **`GET /docs` + the Stoplight Elements assets** — that is M-API slice 3 (the `<elements-api>` web
  component JS + CSS, self-hosted + byte-pinned under `/_ds/` with published `internal/web` hash
  constants). Do NOT add any Stoplight asset, hash constant, or `/docs` route this step. It waits for a
  later iteration so this stays ≤3 files and the asset-pin work lands as its own coherent slice.
- **Per-response JSON-schema components** beyond what ADR-0014 requires — describe the paths + the
  documented response shapes faithfully, but do not over-engineer exhaustive `$ref` component graphs for
  every Go struct; the drift test is on PATHS, not schemas.
- **Any change to the actual machine routes or their handlers** — this step only DOCUMENTS the existing
  surface; it adds no new endpoint behaviour and touches no crypto/proof/store path.
- **A codegen / OpenAPI-validation toolchain dependency** — the doc is hand-authored committed text
  (ADR-0014 §2/§5: "no codegen toolchain, no CGO"). Do not add a heavy OpenAPI library.

## Implementation Notes
- **Exactly which paths the document declares (the machine surface — ADR-0014 §1 + target.md):** the
  flat exact routes `/healthz`, `/version`, `/metrics`, `/openapi.json`, `/openapi.yaml` themselves; the
  per-hub log routes under the `/{domain}/log/` prefix — `/{domain}/log/inclusion`, `/{domain}/log/consistency`,
  `/{domain}/log/entries`, `/{domain}/log/verify`, `/{domain}/log/checkpoint`, `/{domain}/log/checkpoint.ots`,
  `/{domain}/log/tile/{tile}`; and the realm-wide proof bundle `/inclusion/{iscc_id}.bundle`. Use OpenAPI
  path templating (`{domain}`, `{iscc_id}`, `{tile}`) — the drift test matches these templates against the
  mux's per-hub subtree, NOT against a concrete domain.
- **Excluded (HTML SSR — on the drift test's explicit exclusion list, NEVER in the document):** `/` (realm
  index), `/{domain}` (dossier), `/{domain}/log/` (browser root), `/{domain}/log/records`, `/{domain}/log/record`,
  `/inclusion/{iscc_id}` (the HTML certificate — note: distinct from the `.bundle` machine artifact, which
  IS documented), and the entire `/_ds/` subtree (tokens.css, fonts, wasm_exec.js, verify.wasm, logo).
  ADR-0014 is explicit: "The HTML SSR surfaces are out of the OpenAPI contract by design."
- **`verify-for-me` honesty (ADR-0014 §1 + Consequences):** the `/{domain}/log/verify` operation's
  `description` MUST flag it as the explicitly weaker, non-authoritative tier-1 path ("the monitor
  reports") vs the client-verified proof bundle (tier-2). Pin that wording with a golden assertion (the
  served body contains the weaker-path phrasing) so the honesty framing can't be silently dropped.
  (`learnings.md` SSR-honesty rule generalizes: never let a surface assert more trust than it earns — here,
  encode the tier-1 framing in the contract.)
- **`/metrics` is Prometheus text, not JSON-schema'd (ADR-0014 §1):** document it as `text/plain`
  Prometheus exposition; do not invent a JSON response schema for it.
- **The drift test is the load-bearing guarantee (ADR-0014 §3), and it must be NON-VACUOUS.** Build it so:
  (a) the served `/openapi.json` bytes equal `openapi.JSON` (byte-verbatim serve); (b) construct the REAL
  mux via the same wiring `buildMux` uses (a fixture store + a one-hub route, e.g. `sb0.iscc.id`), then for
  EACH path the document declares, probe the mux and assert the route is mounted (a documented route that
  is NOT mounted FAILS); and (c) enumerate the machine-consumable routes the mux mounts and assert each is
  declared in the document (a mounted-but-undocumented machine route FAILS), with the SSR routes on an
  explicit exclusion list. Demonstrate non-vacuity in the test's design: removing a documented path from
  the YAML, OR mounting an undocumented machine route, must make the test FAIL. Map the mux's per-hub
  concrete origin (`sb0.iscc.id/log/inclusion`) to the doc's `{domain}` template when reconciling.
- **`openapi.json` vs `openapi.yaml` must agree:** add a test asserting both embedded documents declare
  the IDENTICAL set of paths (parse both with `yaml.v3` — JSON is valid YAML, so `yaml.Unmarshal` reads
  both — and compare the `paths` key sets), so the two committed artifacts can't drift from each other.
- **Mount placement:** `/openapi.json` + `/openapi.yaml` are exact-path mounts on the outer `mux` in
  `buildMux` (next to `/metrics`, `/healthz`, `/version`) — `http.ServeMux` most-specific match keeps them
  from being shadowed by `/` or any subtree, exactly as the existing exact routes are. They ride the outer
  `corsmw.Handler(mux)` wrap, so they answer cross-origin GETs with `Access-Control-Allow-Origin: *` and a
  204 OPTIONS preflight for free — set NO CORS header in `openapi.Handler`
  (`learnings/http-surface.md` §CORS).
- **Oracle/conformance gate is N/A** (ADR-0014 Consequences; state.md): this adds no signature / RFC-6962 /
  Merkle / did:web / fsck / proof path — it serves committed bytes and reconciles route strings. Note that
  explicitly in the package docstring, matching `internal/web`'s "oracle gate N/A" note.
- Keep `internal/openapi` a pure leaf: stdlib + `gopkg.in/yaml.v3` only, no `internal/store`/`metrics`/
  `logclient` import (the parser reads embedded bytes, not the live mux — the drift test, in
  `cmd/iscc-monitor`'s test, is where doc-paths meet the real mux).

## Verification
- `mise run check` is green (`go build ./...` + `go vet ./...` + `go test ./...`) and `gofmt -l .` is empty.
- `go test -run TestOpenAPI ./internal/openapi` passes — covers: `/openapi.json` → `200`,
  `Content-Type: application/json`, body byte-equal to `openapi.JSON`; `/openapi.yaml` → `200`,
  `application/yaml`, body byte-equal to `openapi.YAML`; a non-GET → `405`; an unknown path → `404`;
  `If-None-Match` echoing the served ETag → `304`; and the JSON/YAML declared-path sets are identical.
- `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` passes — every path the document declares is mounted
  in the real `buildMux` mux, and every machine route the mux mounts is declared (SSR routes excluded);
  the test demonstrates non-vacuity (dropping a documented path or mounting an undocumented machine route
  would FAIL).
- The served `/openapi.json` body parses as a valid OpenAPI 3.1 document (`openapi: 3.1.x`, a non-empty
  `paths` object) and carries `Access-Control-Allow-Origin: *` when fetched through the full `buildMux`
  (corsmw-wrapped) mux — asserted at the HTTP seam.
- `grep -c "openapi" CLAUDE.md` is `> 0` — the two new endpoints are documented in the dev-instance
  endpoint list.

## Done When
`mise run check` is green and the OpenAPI 3.1 document is served byte-verbatim at `/openapi.json` +
`/openapi.yaml` (CORS `*`, 304 on revalidate) covering only the machine surface with `verify-for-me`
flagged weaker, guarded by a non-vacuous drift test that fails the moment the document and the real mux's
machine routes diverge.
