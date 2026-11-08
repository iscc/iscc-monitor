---
status: accepted
---

# OpenAPI 3.1 contract + app-hosted interactive API docs (Stoplight Elements)

The monitor already exposes a machine-consumable HTTP surface — the proof routes
(`inclusion`/`consistency`/`entries`), the `verify-for-me` verdict, the downloadable
proof bundle, the checkpoint/`.ots`/tile artifacts, `/healthz`, `/version`, and
`/metrics` — all over stdlib `net/http` with `Access-Control-Allow-Origin: *` on every
public GET (ADR-0013 §6, `corsmw`). But there is **no machine-readable contract** for
that surface and **no interactive documentation**: the only reference is prose in
`CLAUDE.md`, and the response shapes (`VerifyVerdict`, `ConsistencyEvidence`,
`InclusionEvidence`, the bundle) live solely as Go structs in `internal/*`. A
third-party system integrating against an instance has nothing to generate a client
from, no schema to validate responses against, and no in-browser way to explore the
API. This is a gap, not a recorded choice — no prior ADR addresses it.

This fits the project's posture exactly. A **verifiable cache** wants its surface to be
*discoverable and self-describing*, not folklore: the same reason we publish proof
bundles a client verifies without trusting us argues for publishing a contract a client
integrates against without reading our source. And the response types are already
concrete Go structs, so the spec is mostly transcription, not design.

This is the **API-contract** surface. It is distinct from, and must not be conflated
with, the human-facing SSR surfaces (dashboard, dossier, log browser, certificate —
ADR-0010 / M-UI): those are documented by their rendered HTML + `CLAUDE.md`, **not** by
OpenAPI. The OpenAPI document describes only the JSON / artifact / text endpoints a
machine consumes.

## Decision

1. **Ship a hand-authored OpenAPI 3.1 document, in-repo, as the source of truth.** It
   describes the machine-consumable surface only: `GET /healthz`, `GET /version`,
   `GET /metrics` (described as Prometheus text exposition, not JSON-schema'd),
   `GET /<domain>/log/inclusion`, `/consistency`, `/entries`, `/checkpoint`,
   `/checkpoint.ots`, `/tile/...`, `GET /<domain>/log/verify` (the **explicitly weaker,
   non-authoritative** `verify-for-me` path — labelled as such in its `description`),
   and `GET /inclusion/<iscc_id>.bundle`. The HTML SSR surfaces are **out of the
   OpenAPI contract** by design. The schema components are derived from the existing Go
   response structs so the document and the code describe the same bytes.

2. **The app serves the spec itself, CORS-enabled.** `GET /openapi.json` (and
   `GET /openapi.yaml`) return the in-repo document via `go:embed`, byte-verbatim, under
   the same `corsmw` `*` policy as the rest. No build-time network, no codegen toolchain,
   no CGO — the spec is committed text, consistent with `CGO_ENABLED=0` (ADR-0003) and
   reproducible builds.

3. **A drift test keeps the spec honest — this is the load-bearing guarantee, not
   codegen.** A test asserts (a) the served `/openapi.json` bytes equal the embedded
   document, and (b) **every path the document declares is mounted in the real mux, and
   every machine-consumable route the mux mounts is declared** — so adding or renaming a
   JSON route without updating the spec FAILS the gate, and a documented-but-removed
   route FAILS it too. The HTML SSR routes are on an explicit allow-list excluded from
   (b). Without this test an OpenAPI file is just stale prose; with it the contract
   cannot silently diverge from the handler.

4. **Host interactive docs in-app at `GET /docs` using Stoplight Elements, self-hosted
   and byte-pinned.** The page is a server-rendered shell (same no-CDN DS shell as every
   other surface) that mounts the **Stoplight Elements** `<elements-api>` web component
   with `apiDescriptionUrl="/openapi.json"`. Elements ships as **two** assets — the
   web-component JS bundle and its stylesheet — both served from `/_ds/` byte-verbatim
   with the same strong-ETag + no-cache + 304 policy as `verify.wasm`, each SHA-256
   published as a build-pinned constant in `internal/web` alongside `WasmVerifyHash` (the
   SRI / pinned self-hosted-asset discipline, ADR-0003 / ADR-0010). **No external CDN, no
   external runtime call**: Elements' "Try It" console issues requests **browser → this
   instance directly** by default, so the `tryItCorsProxy` attribute MUST be left unset —
   relying on the CORS `*` the surface already sets, never a third-party proxy.

5. **No new heavy dependency or language change.** OpenAPI is committed text; Stoplight
   Elements is two pinned static assets embedded like the fonts and the WASM. ADR-0003's
   static-binary / `CGO_ENABLED=0` stack is unchanged. The work is in-repo and
   CI-verifiable end to end — no human/infra step gates it (it composes with M-Deploy's
   "code-closable while feature milestones are blocked" property).

**Renderer choice — Stoplight Elements (decided).** The brief allowed Stoplight Elements
/ RapiDoc / ReDoc / Scalar; the operator chose Elements for the **cleanest, most legible
default design**. Elements also fits the project's invariants well: Apache-2.0 licence,
native OpenAPI 3.1, a self-hostable web component that pins like `verify.wasm`, and a
built-in "Try It" console that issues requests **directly from the browser by default** —
no external proxy to disable, a cleaner no-external-call story than Scalar's. Its costs
are accepted in exchange for the design: **two** pinned assets instead of one (the
web-component JS *and* its stylesheet), and coarser DS-token theming than Scalar/RapiDoc
(Elements is styled largely by its own shipped CSS). **Alternatives considered:** *ReDoc*
OSS has no "try it" console (that is Redocly's paid tier), so it fails the interactivity
ask; *Scalar* is a single, heavily-themeable bundle but ships a default external request
proxy that must be turned off; *RapiDoc* is a lean single web component with zero external
calls by design — *Scalar* and *RapiDoc* remain the **sanctioned fallbacks** if Elements'
footprint or theming proves undesirable.

## Consequences

- **CID-loop-ownable and fully in-repo verifiable.** The OpenAPI document, the
  `/openapi.json|yaml` + `/docs` routes, the pinned Stoplight Elements assets + their published hashes,
  and the drift test all land and verify under `mise run check` + golden HTTP-seam tests.
  No GHCR/DNS/Pages-style human step is involved.
- **The no-CDN / no-external-runtime-call invariant is preserved**, because "Try It" calls
  the instance directly and every asset is self-hosted and pinned — the same property the
  rest of the site guarantees. A reviewer can assert "no external host in the `/docs` body
  or in any request it makes" exactly as for the other surfaces.
- **The asset-pin surface grows by two.** The Stoplight Elements web-component JS *and* its
  stylesheet join `verify.wasm` under the fragile-pin discipline: each must be
  re-fetched/repinned deliberately, never hand-edited, and its hash is gated. (See the
  standing "verify.wasm pin is fragile" learning.)
- **A standing maintenance obligation, by design.** Any new or changed machine-consumable
  route must update the OpenAPI document or the drift test fails. That is the point — it
  converts "the docs drifted" from a silent rot into a red gate.
- **`CLAUDE.md` stays the human narrative; OpenAPI becomes the machine contract.** They
  are complementary: prose explains *why/when*, the spec defines *shape*. The drift test
  binds the spec to the code; `CLAUDE.md` is updated by hand as today.
- **`verify-for-me` is documented as weaker, in band.** Encoding the tier-1 ("the monitor
  reports") vs tier-2 ("verify the bundle yourself") honesty *inside the spec's
  descriptions* keeps the trust framing with the contract, not just in the UI.
